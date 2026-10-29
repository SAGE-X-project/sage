package registry011

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math"
	"sync"
	"time"
)

var errProviderOwnedClock = errors.New("registry011: owned integration clock unavailable")

// providerOwnedClock is a private process-lifetime clock backend. Constructing
// it does not qualify a profile or select a policy. The eventual trusted Agent
// selection must independently qualify the clock artifact and uncertainty.
// Candidate JSON never supplies a sample or an operating callback here.
type providerOwnedClock struct {
	mu             sync.Mutex
	owner          *providerOwnedClock
	origin         time.Time
	id, epoch      string
	policy         string
	uncertainty    int64
	readWindow     time.Duration
	maximum        uint64
	read           uint64
	last           ClockSample
	failed         bool
	legacyClaims   uint64
	history        *providerHistoryProgramOwner
	historyRevoked bool
}

// The finite setup envelope must cover construction and every permitted read.
// This factory is private and is not a Source/selection/activation constructor.
func newProviderOwnedClock(policy *ProviderPolicyCandidate, maximum uint64, baseUncertaintyMS, readWindowMS int64) (*providerOwnedClock, error) {
	if policy == nil || policy.raw == "" || maximum == 0 || maximum > 1<<20 || baseUncertaintyMS <= 0 || readWindowMS <= 0 || readWindowMS > math.MaxInt64/int64(time.Millisecond) || baseUncertaintyMS > math.MaxInt64-readWindowMS {
		return nil, ErrConfiguration
	}
	p := contentJSON([]byte(policy.raw)).(map[string]any)["clock"].(map[string]any)
	maximumUncertainty, _ := providerCandidateUint(p["max_uncertainty_ms"], true)
	maximumWindow, _ := providerCandidateUint(p["max_read_window_ms"], true)
	if uint64(baseUncertaintyMS+readWindowMS) > maximumUncertainty || uint64(readWindowMS) > maximumWindow {
		return nil, ErrConfiguration
	}
	// This random epoch label is not a key, secret or authority. A new process
	// clock cannot resume an earlier epoch's admission or dispatch permission.
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, errProviderOwnedClock
	}
	c := &providerOwnedClock{origin: time.Now(), id: contentString(p["id"]), policy: policy.policy, epoch: "0x" + hex.EncodeToString(nonce[:]), uncertainty: baseUncertaintyMS + readWindowMS, readWindow: time.Duration(readWindowMS) * time.Millisecond, maximum: maximum}
	c.owner = c
	return c, nil
}

func (*providerOwnedClock) UnmarshalJSON([]byte) error { return errProviderOwnedClock }

func (c *providerOwnedClock) sample() (ClockSample, error) {
	return c.sampleForHistory(nil, nil)
}

// Legacy stores record their use before opening either root. A future whole
// program cannot reserve a backend already handed to one of those stores,
// even if that store has not taken its first sample or its open later fails.
func (c *providerOwnedClock) claimLegacy(policy string) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.owner != c || c.failed || c.history != nil || c.policy != policy || c.maximum == 0 || c.read >= c.maximum || c.legacyClaims >= 1<<20 {
		return false
	}
	c.legacyClaims++
	return true
}

// The original backend admits only the one pre-H0 parent after reservation.
// No diagnostic, second store, direct legacy call or copied parent can spend
// its samples. Refusal occurs before the counter or native clock is touched.
func (c *providerOwnedClock) sampleForHistory(parent *providerHistoryProgramOwner, claim *providerHistoryClockClaim) (ClockSample, error) {
	if c == nil {
		return ClockSample{}, errProviderOwnedClock
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.history != parent || parent == nil && claim != nil || parent != nil && (c.historyRevoked || parent.owner != parent || parent.clock != c || parent.store == nil || parent.store.history != parent || !parent.clockSelectionMatches(c) || c.read >= parent.clockCeiling || parent.original == (ClockSample{}) && c.read != parent.initialRead || claim == nil || claim.owner != claim || claim.parent != parent || parent.clockClaim != claim || claim.used || claim.expectedRead != c.read || claim.meter == nil || parent.store.activeMeter.Load() != claim.meter) {
		return ClockSample{}, errProviderOwnedClock
	}
	if claim != nil {
		claim.used = true // Burn even when the following native read fails.
	}
	if c.owner != c || c.failed || c.maximum == 0 || c.read >= c.maximum || c.origin.IsZero() || !lowerHash(c.epoch) || c.id == "" || c.uncertainty <= 0 || c.readWindow <= 0 {
		return ClockSample{}, errProviderOwnedClock
	}
	// A refused or delayed read still consumes its finite call allowance.
	c.read++
	before := time.Now()
	now := time.Now()
	after := time.Now()
	if after.Sub(before) < 0 || after.Sub(before) > c.readWindow || now.Before(c.origin) {
		c.failed = true
		return ClockSample{}, errProviderOwnedClock
	}
	sample := ClockSample{ClockID: c.id, Epoch: c.epoch, WallMS: now.UnixMilli(), MonotonicMS: now.Sub(c.origin).Milliseconds(), UncertaintyMS: c.uncertainty}
	if sample.WallMS < sample.UncertaintyMS || sample.MonotonicMS < 0 || c.read > 1 && !candidateLedgerClockAdvance(providerClockDiagnostic(c.last), providerClockDiagnostic(sample)) {
		c.failed = true
		return ClockSample{}, errProviderOwnedClock
	}
	if parent != nil && parent.original != (ClockSample{}) && (sample.MonotonicMS < parent.original.MonotonicMS || uint64(sample.MonotonicMS-parent.original.MonotonicMS) > parent.bounds.durationMS) {
		c.failed, c.historyRevoked = true, true
		return ClockSample{}, errProviderOwnedClock
	}
	c.last = sample
	return sample, nil
}

// Only this direction exists: owned samples may be copied into retained
// candidate metadata. No JSON/candidate diagnostic can reconstruct this clock.
func providerClockDiagnostic(sample ClockSample) CandidateClockDiagnostic {
	return CandidateClockDiagnostic(sample)
}
