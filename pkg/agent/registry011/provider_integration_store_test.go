package registry011

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// A normal, isolated owned filesystem fixture only. Its anchors/profile are
// explicitly unqualified test subjects, never selected operating custody.
func providerStoreFixture(t *testing.T) (*providerCustodyStore, *ProviderPolicyCandidate, *ProviderCandidateContext, providerPrimitiveBounds) {
	return providerStoreFixtureWithCapture(t, ProviderCaptureLimits{Timeout: 2 * time.Second, HeaderBytes: 8 << 10, BodyBytes: 16 << 10, WireBytes: 64 << 10, JSONDepth: 64})
}

func providerStoreFixtureWithCapture(t *testing.T, capture ProviderCaptureLimits) (*providerCustodyStore, *ProviderPolicyCandidate, *ProviderCandidateContext, providerPrimitiveBounds) {
	return providerStoreFixtureWithLedger(t, capture, CandidateLedgerBounds{4096, 16 << 20, 64 << 20})
}

// Choose an isolated test profile before its anchor, setup and original clock.
func providerStoreFixtureWithLedger(t *testing.T, capture ProviderCaptureLimits, ledger CandidateLedgerBounds) (*providerCustodyStore, *ProviderPolicyCandidate, *ProviderCandidateContext, providerPrimitiveBounds) {
	t.Helper()
	object := providerPolicyTestObject()
	for _, level := range []string{"operation", "segment", "bootstrap"} {
		m := object["limits"].(map[string]any)[level].(map[string]any)
		for field, value := range map[string]uint64{"requests": 65536, "wire_bytes": 256 << 20, "decoded_bytes": 256 << 20, "blocks": 4096, "transactions": 16384, "frames": 65536, "logs": 65536, "proofs": 4096, "segments": 4096, "disk_bytes_written": 1 << 40, "work_units": 1 << 60, "deadline_ms": 300000} {
			m[field] = strconv.FormatUint(value, 10)
		}
	}
	limits := object["limits"].(map[string]any)
	limits["live_disk_bytes"], limits["retained_bytes"] = "1099511627776", "1099511627776"
	clockObject := object["clock"].(map[string]any)
	clockObject["max_uncertainty_ms"], clockObject["max_read_window_ms"] = "1000", "1000"
	policy, err := ParseProviderPolicyCandidate(providerPolicyTestBytes(t, object), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := policy.BindCandidate(admissionTestDigest("2"), admissionTestDigest("3"), "0")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := candidate.Identity()
	bounds := providerIntegrationBounds{ledger: ledger, maximumCommits: 256, maximumObjects: 4096, maximumEntries: 64, maximumObjectBytes: 64 << 10, maximumStoredBytes: 64 << 20, maximumMetadataPerCommit: 16 << 10, fileOverheadBytes: 4096, maximumSetupRecoveries: 3, setupWork: candidateWorkVector{0, 0, 1 << 20, 0, 0, 0, 0, 0, 0, 1 << 20, 1 << 60}, setupLive: 2 << 20, setupRetained: 1 << 20, setupMaximumClockReads: 65536, setupMaximumDurationMS: 300000}
	profile := providerIntegrationProfile(bounds, capture)
	empty, err := newProviderIntegrationInventory("", nil, 64, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	anchor := providerIntegrationJSON(map[string]string{"format": "sage-provider-custody-anchor-v1", "namespace": id.Namespace, "instance_key": id.Instance, "profile": profile, "initial_objects_root": empty.digest})
	root := t.TempDir()
	custody, evidence := filepath.Join(root, "custody"), filepath.Join(root, "evidence")
	for _, directory := range []string{custody, evidence} {
		if os.Mkdir(directory, 0700) != nil || os.WriteFile(filepath.Join(directory, "AUTHORITY.LOCK"), nil, 0600) != nil || os.WriteFile(filepath.Join(directory, "ANCHOR"), anchor, 0600) != nil {
			t.Fatal("fixture provision")
		}
	}
	clock, err := newProviderOwnedClock(policy, 65536, 10, 10)
	if err != nil {
		t.Fatal(err)
	}
	digest := providerCandidateHash("sage-provider-custody-anchor-v1", anchor)
	config := providerCustodyConfig{custodyDirectory: custody, evidenceDirectory: evidence, anchor: digest, expectedTip: digest, expectedSetupTip: digest, profile: profile, identity: id, policy: policy, capture: capture, bounds: bounds}
	store, err := openProviderCustody(config, clock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if !store.closed {
			_ = store.close()
		}
	})
	return store, policy, candidate, providerPrimitiveBounds{capture: capture, blocks: 4096, transactions: 16384, frames: 65536, logs: 65536, proofs: 4096, maximumCreatedObjects: 32}
}

func TestProviderIntegrationSetupAndInitialAdmissionHaveSeparateAcknowledgments(t *testing.T) {
	store, policy, candidate, primitive := providerStoreFixture(t)
	if store.sequence != 0 || store.enrollment != "" || store.setupTip == store.config.anchor || len(store.setupCommits) != 1 {
		t.Fatal("setup invented enrollment")
	}
	coordinator, err := newProviderIntegration(store, policy, candidate, primitive)
	if err != nil {
		t.Fatal(err)
	}
	if coordinator.bootstrap != "" || coordinator.firstExchange != "" || coordinator.root.status != "enrolling" || coordinator.firstTicket == nil || coordinator.ledger.lastConsumption != nil {
		t.Fatal("admission manufactured completed work")
	}
	if coordinator.firstTicket.method != "eth_getBlockByNumber" || !coordinator.first.firstFinalized() {
		t.Fatal("first request not bound")
	}
	if store.sequence != 2 || store.replay == nil || store.replay.intent.digest != coordinator.intent.digest || store.replay.ledger.current.digest != coordinator.ledger.current.digest {
		t.Fatal("ACK preceded typed replay")
	}
	client, err := coordinator.readClient()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if client.acquisition == nil || client.acquisition.integration != coordinator {
		t.Fatal("missing central gate")
	}
	if _, _, _, err := newSourceAcquisition(context.Background(), client, SourceReadLimits{}); err == nil {
		t.Fatal("nested acquisition stripped gate")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.PinAgentReads(ctx, SepoliaRegistryAddress, "owned-agent"); err == nil {
		t.Fatal("cancelled request reached dispatch")
	}
	if _, err := client.PinAgentReads(context.Background(), SepoliaRegistryAddress, "owned-agent"); err == nil {
		t.Fatal("old initialization probed before the bound first finalized read")
	}
	if coordinator.ledger.lastConsumption != nil {
		t.Fatal("refused request consumed first operation")
	}
}

func TestProviderIntegrationConsumptionIsDurableAndOneUse(t *testing.T) {
	store, policy, candidate, primitive := providerStoreFixture(t)
	c, err := newProviderIntegration(store, policy, candidate, primitive)
	if err != nil {
		t.Fatal(err)
	}
	permit, err := c.consume(c.firstTicket, c.first)
	if err != nil {
		t.Fatal(err)
	}
	if store.sequence != 3 || c.ledger.lastConsumption == nil || store.replay.ledger.lastConsumption.digest != c.ledger.lastConsumption.digest {
		t.Fatal("consumption not independently reconstructed")
	}
	if !c.use(permit) || c.use(permit) {
		t.Fatal("one-use dispatch transition failed")
	}
	if _, err := c.consume(c.firstTicket, c.first); err == nil {
		t.Fatal("ticket consumed again")
	}
	if permit.meter.actual[0] != 1 || permit.meter.actual[1] != 0 {
		t.Fatal("no-network unit invented received bytes")
	}
	// Do not dial anything: this unit leaves the full consumed reservation
	// outstanding, precisely the crash-before-dispatch outcome under test.
	current, _ := c.ledger.Current()
	diagnostic, _ := current.Diagnostic()
	if diagnostic.Outstanding != c.firstTicket.work.diagnostic() || diagnostic.Settlements != 0 {
		t.Fatal("consumption refunded unperformed work")
	}
}

func TestProviderIntegrationReopenReturnsMetadataAndCannotReenroll(t *testing.T) {
	store, policy, candidate, primitive := providerStoreFixture(t)
	c, err := newProviderIntegration(store, policy, candidate, primitive)
	if err != nil {
		t.Fatal(err)
	}
	permit, err := c.consume(c.firstTicket, c.first)
	if err != nil {
		t.Fatal(err)
	}
	config := store.config
	config.attempt = 1
	config.expectedTip = store.tip
	config.expectedSetupTip = store.setupTip
	if store.close() != nil {
		t.Fatal("close")
	}
	reopened, err := openProviderCustody(config, store.clock)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	if reopened.last == nil || !reopened.last.recovered || reopened.tip != config.expectedTip || reopened.enrollment != c.root.digest {
		t.Fatal("recovery changed retained roots")
	}
	if _, err := newProviderIntegration(reopened, policy, candidate, primitive); err == nil {
		t.Fatal("restart created fresh admission")
	}
	if c.use(permit) {
		t.Fatal("old owner regained dispatch")
	}
	if reopened.replay.ledger.current.totals.bootstrap.outstanding != c.firstTicket.work {
		t.Fatal("restart lost outstanding work")
	}
}

func TestProviderIntegrationIncompleteSetupIsNeverAbsence(t *testing.T) {
	store, _, _, _ := providerStoreFixture(t)
	config := store.config
	config.attempt = 1
	config.expectedSetupTip = store.setupTip
	if store.close() != nil {
		t.Fatal("close")
	}
	// Finite unit crash-state construction in this test's own temporary roots.
	// There is no host-bypass, exploit or reusable attack producer here.
	if os.Remove(filepath.Join(config.custodyDirectory, providerCustodyName("setup-commit", 0))) != nil {
		t.Fatal("fixture state")
	}
	if value, err := openProviderCustody(config, store.clock); err == nil || value != nil {
		t.Fatal("partial setup became authenticated absence")
	}
	if _, err := os.Stat(filepath.Join(config.custodyDirectory, providerCustodyName("setup-intent", 0))); err != nil {
		t.Fatal("failed setup charge was removed")
	}
}

func TestProviderIntegrationSameSelectedStoreCannotOpenTwice(t *testing.T) {
	store, _, _, _ := providerStoreFixture(t)
	if other, err := openProviderCustody(store.config, store.clock); err == nil || other != nil {
		t.Fatal("cooperating lock not exclusive")
	}
	if !store.valid() || store.custody.checkIdentity() != nil || store.evidence.checkIdentity() != nil {
		t.Fatal("failed opener changed owner")
	}
}

func TestProviderIntegrationSetupRecoveryBeforeEnrollmentUsesLatestClaim(t *testing.T) {
	store, policy, candidate, primitive := providerStoreFixture(t)
	config := store.config
	config.attempt = 1
	config.expectedSetupTip = store.setupTip
	original := store.setupOrigin
	if store.close() != nil {
		t.Fatal("close")
	}
	reopened, err := openProviderCustody(config, store.clock)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	selected, found := reopened.setupForBoundary(config.anchor)
	if !found || selected.ordinal != 1 || selected.commit != reopened.setupTip || reopened.setupOrigin != original {
		t.Fatal("setup recovery changed origin or chose an old claim")
	}
	coordinator, err := newProviderIntegration(reopened, policy, candidate, primitive)
	if err != nil || coordinator.firstTicket == nil {
		t.Fatal("completed metadata recovery could not create its first admission", err)
	}
	if _, err := reopened.object("setup-commit", selected.commit); err != nil {
		t.Fatal("enrollment did not retain its selected setup")
	}
	if _, retained := reopened.objects["setup-commit:"+store.setupTip]; retained {
		t.Fatal("enrollment substituted an old setup")
	}
}
