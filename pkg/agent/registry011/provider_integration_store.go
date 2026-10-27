package registry011

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

// This private implementation is a cooperating-writer store, not an operating
// selection API. Both preprovisioned roots and the independently selected
// anchor/high-water must come from trusted Agent configuration. No directory,
// Source ID, candidate JSON or fixture can create that selection.
type providerCustodyConfig struct {
	custodyDirectory, evidenceDirectory            string
	anchor, expectedTip, expectedSetupTip, profile string
	identity                                       ProviderCandidateIdentity
	policy                                         *ProviderPolicyCandidate
	capture                                        ProviderCaptureLimits
	bounds                                         providerIntegrationBounds
	attempt                                        int
}

type providerCustodyStore struct {
	mu                           sync.Mutex
	owner                        *providerCustodyStore
	custody, evidence            *retentionDisk
	config                       providerCustodyConfig
	clock                        *providerOwnedClock
	tip, enrollment, objectsRoot string
	sequence, stored             uint64
	retainedRaw                  uint64
	objects                      map[string]providerIntegrationObjectRef
	bindings                     map[string]string
	operations                   map[string]bool
	last                         *providerCustodyAcknowledgment
	setupOrigin                  ClockSample
	setupTip                     string
	setupCommits                 map[string]string
	setups                       []providerSetupReference
	replay                       *providerCustodyReplay
	frontierFiles                *providerFrontierFiles
	frontierReplay               *providerFrontierReplay
	activeMeter                  atomic.Pointer[providerWorkMeter]
	newReferences                []providerIntegrationObjectRef
	requiredReferences           map[string]bool
	failed, closed               bool
}

// An ACK is private, nonserializable and belongs to one live store. A recovered
// ACK is metadata only; the executor never uses it to recreate a work permit.
type providerCustodyAcknowledgment struct {
	owner                                   *providerCustodyStore
	previous, commit, enrollment, operation string
	objectsRoot                             string
	fence, sequence                         uint64
	recovered                               bool
}

func (*providerCustodyStore) UnmarshalJSON([]byte) error { return errProviderIntegration }
func (*providerCustodyAcknowledgment) UnmarshalJSON([]byte) error {
	return errProviderIntegration
}

func providerCustodyName(kind string, index uint64) string {
	return fmt.Sprintf("%s-%020d", kind, index)
}

func providerObjectName(kind, digest string) string { return "object-" + kind + "-" + digest[2:] }

// Opening consumes one finite, immutable setup slot before complete-prefix
// validation. Its full setup envelope remains charged after any later refusal.
// Slots and the original anchor are selected configuration, never auto-reset.
func openProviderCustody(config providerCustodyConfig, clock *providerOwnedClock) (*providerCustodyStore, error) {
	// The old whole-prefix mode must not acquire a private V2 lifetime merely
	// because its bounds parser understands the separate candidate profile.
	if config.bounds.continuation != nil {
		return nil, ErrConfiguration
	}
	id := config.identity
	if config.policy == nil || config.policy.raw == "" || config.policy.policy != id.Policy || config.policy.instance != id.Instance || config.policy.provider != id.Provider || !config.capture.valid() || !config.bounds.valid() || config.attempt < 0 || config.attempt > config.bounds.maximumSetupRecoveries ||
		!lowerHash(config.anchor) || !lowerHash(config.expectedTip) || !lowerHash(config.expectedSetupTip) || config.profile != providerIntegrationProfile(config.bounds, config.capture) ||
		!lowerHash(id.Namespace) || !lowerHash(id.Instance) || !lowerHash(id.Policy) || !lowerHash(id.SourceID) || id.Generation != 0 || clock == nil ||
		config.custodyDirectory == "" || config.evidenceDirectory == "" {
		return nil, ErrConfiguration
	}
	custody, err := openRetentionDisk(config.custodyDirectory)
	if err != nil {
		return nil, errProviderIntegration
	}
	evidence, err := openRetentionDisk(config.evidenceDirectory)
	if err != nil {
		_ = custody.close()
		return nil, errProviderIntegration
	}
	s := &providerCustodyStore{custody: custody, evidence: evidence, config: config, clock: clock, tip: config.anchor, objects: make(map[string]providerIntegrationObjectRef), operations: make(map[string]bool)}
	s.owner = s
	accepted := false
	defer func() {
		if !accepted {
			_ = evidence.close()
			_ = custody.close()
		}
	}()
	if os.SameFile(custody.directoryInfo, evidence.directoryInfo) || os.SameFile(custody.lockInfo, evidence.lockInfo) {
		return nil, errProviderIntegration
	}
	anchor, err := custody.read("ANCHOR", int64(config.bounds.maximumMetadataPerCommit))
	localAnchor, localErr := evidence.read("ANCHOR", int64(config.bounds.maximumMetadataPerCommit))
	if err != nil || localErr != nil || !bytes.Equal(anchor, localAnchor) || providerCandidateHash("sage-provider-custody-anchor-v1", anchor) != config.anchor {
		return nil, errProviderIntegration
	}
	a := contentObject(contentJSON(anchor), "format", "namespace", "instance_key", "profile", "initial_objects_root")
	empty, err := newProviderIntegrationInventory("", nil, config.bounds.maximumEntries, min(config.bounds.maximumStoredBytes, 64<<20))
	if err != nil || a == nil || a["format"] != "sage-provider-custody-anchor-v1" || a["namespace"] != id.Namespace || a["instance_key"] != id.Instance || a["profile"] != config.profile || a["initial_objects_root"] != empty.digest {
		return nil, errProviderIntegration
	}
	s.objectsRoot = empty.digest
	sample, err := clock.sample()
	if err != nil {
		return nil, errProviderIntegration
	}
	if s.writeSetup(sample, uint64(2*len(anchor))) != nil {
		return nil, errProviderIntegration
	}
	// No E0, intent or request can be constructed from this handle until the
	// independently retained complete prefix and finite setup slots agree.
	if s.loadPrefix(sample) != nil || s.tip != config.expectedTip {
		return nil, errProviderIntegration
	}
	end, err := clock.sample()
	if err != nil || end.Epoch != s.setupOrigin.Epoch || end.MonotonicMS < s.setupOrigin.MonotonicMS || uint64(end.MonotonicMS-s.setupOrigin.MonotonicMS) > config.bounds.setupMaximumDurationMS {
		return nil, errProviderIntegration
	}
	accepted = true
	return s, nil
}

func providerCustodySetup(config providerCustodyConfig, clock ClockSample) []byte {
	id, b := config.identity, config.bounds
	raw := providerIntegrationJSON(map[string]any{"format": "sage-provider-authority-setup-v1", "namespace": id.Namespace, "instance_key": id.Instance, "policy": id.Policy, "source_id": id.SourceID, "generation": strconv.FormatUint(id.Generation, 10), "anchor": config.anchor, "expected_tip": config.expectedTip, "previous_setup_commit": config.expectedSetupTip, "operation": providerCandidateHash("sage-provider-setup-operation-v1", []byte(config.anchor+id.Instance+id.Policy+id.SourceID)), "profile": config.profile, "attempt": strconv.Itoa(config.attempt), "clock": providerClockObject(clock), "work": candidateLedgerWorkObject(b.setupWork), "live_upper": strconv.FormatUint(b.setupLive, 10), "retained_upper": strconv.FormatUint(b.setupRetained, 10), "maximum_clock_reads": strconv.FormatUint(b.setupMaximumClockReads, 10), "maximum_duration_ms": strconv.FormatUint(b.setupMaximumDurationMS, 10)})
	return raw
}

func providerClockObject(clock ClockSample) map[string]string {
	return map[string]string{"id": clock.ClockID, "epoch": clock.Epoch, "wall_ms": strconv.FormatInt(clock.WallMS, 10), "monotonic_ms": strconv.FormatInt(clock.MonotonicMS, 10), "uncertainty_ms": strconv.FormatInt(clock.UncertaintyMS, 10)}
}

func (s *providerCustodyStore) valid() bool {
	return s != nil && s.owner == s && !s.failed && !s.closed && s.custody != nil && s.evidence != nil && s.clock != nil && s.config.bounds.valid() && s.objects != nil && s.operations != nil
}

func (s *providerCustodyStore) close() error {
	if s == nil {
		return errProviderIntegration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.owner != s || s.closed {
		return errProviderIntegration
	}
	s.closed = true
	return errors.Join(s.evidence.close(), s.custody.close())
}

// Read one bounded directory entry at a time: an unexpected or excessive tail
// refuses without allocating a slice proportional to an untrusted directory.
func providerDirectoryNames(disk *retentionDisk, maximum int) (map[string]bool, error) {
	if maximum <= 0 || disk.checkIdentity() != nil {
		return nil, errProviderIntegration
	}
	f, err := disk.root.Open(".")
	if err != nil {
		return nil, errProviderIntegration
	}
	defer func() { _ = f.Close() }()
	names := make(map[string]bool)
	for len(names) <= maximum {
		items, err := f.ReadDir(1)
		if err == io.EOF && len(items) == 0 {
			return names, nil
		}
		if err != nil || len(items) != 1 || !items[0].Type().IsRegular() || names[items[0].Name()] {
			return nil, errProviderIntegration
		}
		names[items[0].Name()] = true
	}
	return nil, errProviderIntegration
}
