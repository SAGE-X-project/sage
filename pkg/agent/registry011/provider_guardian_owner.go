package registry011

const (
	providerGuardianMetadata = iota + 1
	providerGuardianRecorded
	providerGuardianAdmission
)

// This separate live role is bound and fully held before H0. Neither DONE
// first-operation DATA nor a decoded guardian record constructs this role.
// Candidate-private only: no constructor selects an operating principal.
type providerGuardianOwner struct {
	owner                                        *providerGuardianOwner
	store                                        *providerCustodyStore
	config                                       providerCustodyConfig
	policy                                       ProviderPolicyCandidate
	bounds                                       providerGuardianBounds
	clock                                        *providerOwnedClock
	original                                     ClockSample
	origin, h0, genesis                          string
	pin, before                                  providerGuardianHeadData
	pinRaw, beforeRaw                            string
	main                                         providerFrontierHighwater
	first                                        *providerFirstOperationOwner
	continuing                                   *providerContinuingOwner
	sealing                                      *providerPageSealOwner
	prefix                                       providerContinuingPrefix
	claimPhysical, claimRaw, auxiliaryMetadata   uint64
	integration                                  *providerIntegration
	indexes                                      *providerLedgerIndexes
	meter                                        *providerWorkMeter
	claim                                        *providerGuardianAttemptData
	ordinal                                      uint64
	phase                                        int
	genesisMetadata, metadata, raw, physical     uint64
	initialized, admitted, ready, active, failed bool
}

func (*providerGuardianOwner) UnmarshalJSON([]byte) error { return errProviderIntegration }

func providerGuardianInitialStored(b providerIntegrationBounds) (uint64, bool) {
	if b.history != nil {
		held, ok := providerHistoryWholeHold(b)
		return held.retained, ok && held.retained <= b.maximumStoredBytes
	}
	if b.guardian == nil {
		return b.setupRetained, true
	}
	retained, ok := providerUpperSum(b.setupRetained, b.guardian.lifetime.retained)
	return retained, ok && retained <= b.maximumStoredBytes
}

func providerGuardianStoredMatches(b providerIntegrationBounds, stored uint64) bool {
	want, ok := providerGuardianInitialStored(b)
	return ok && stored == want
}

// All checks here are scalar. Policy parsing and profile encoding have already
// been funded by O before this call, but no root/clock has been observed yet.
func newProviderGuardianBeforeOpenCandidate(s *providerCustodyStore) (*providerGuardianOwner, error) {
	if s == nil || s.owner != s || s.guardian != nil || s.custody != nil || s.evidence != nil || s.setupOrigin != (ClockSample{}) || s.config.bounds.guardian == nil || s.config.policy == nil || s.firstLimits == nil || s.clock == nil || s.clock.policy != s.config.identity.Policy {
		return nil, errProviderIntegration
	}
	b := s.config.bounds
	stored, ok := providerGuardianInitialStored(b)
	if !ok || s.stored != stored || stored > s.firstLimits.live || stored > s.firstLimits.retained || !providerGuardianInitialFits(s) {
		return nil, errProviderIntegration
	}
	for i, n := range b.guardian.lifetime.work {
		if !candidateCapacity(b.setupWork[i], 0, n, s.firstLimits.work[i]) {
			return nil, errProviderIntegration
		}
	}
	g := &providerGuardianOwner{store: s, config: s.config, policy: *s.config.policy, bounds: *b.guardian, clock: s.clock}
	g.owner = g
	return g, nil
}

func (g *providerGuardianOwner) selectedLocked() bool {
	return g != nil && g.owner == g && !g.failed && g.store != nil && g.store.guardian == g &&
		g.store.valid() && g.store.config == g.config && g.config.policy != nil && *g.config.policy == g.policy &&
		g.config.bounds.guardian != nil && *g.config.bounds.guardian == g.bounds && g.store.clock == g.clock &&
		g.clock != nil && g.original != (ClockSample{}) && g.store.setupOrigin == g.original && g.ordinal <= g.bounds.attempts
}

func (g *providerGuardianOwner) fits(m *providerWorkMeter) bool {
	if !g.selectedLocked() || !g.initialized || !g.admitted || !g.active || g.ready || g.claim == nil ||
		g.phase < providerGuardianMetadata || g.phase > providerGuardianAdmission ||
		g.ordinal != g.before.ordinal+1 || g.ordinal != g.claim.ordinal || g.claim.clock != providerClockDiagnostic(g.original) {
		return false
	}
	head := g.pin == g.before
	if g.phase >= providerGuardianRecorded {
		head = g.pin.ordinal == g.ordinal && g.pin.previous == g.claim.previous && g.pin.admission == g.claim.admission && g.pin.lastStart == g.claim.start
	}
	return head && g.meter == m && m != nil &&
		m.guardian == g && m.integration == nil && m.scan == nil && m.firstAdmission == nil && m.firstOperation == nil &&
		m.upper == g.bounds.claim.work && m.maximumClocks == g.bounds.claim.clocks && g.store.activeMeter.Load() == m
}

// Claim work was irreversibly debited before this first sample. The start is
// a per-claim sublimit; it never replaces the original admission clock/epoch.
func (g *providerGuardianOwner) sampleLocked() (ClockSample, error) {
	if g == nil || !g.fits(g.meter) {
		return ClockSample{}, errProviderIntegration
	}
	now, err := g.store.samplePrimitive()
	if err != nil || !candidateLedgerClockAdvance(providerClockDiagnostic(g.original), providerClockDiagnostic(now)) || candidateLedgerExpired(providerClockDiagnostic(now), g.claim.deadline) || uint64(now.MonotonicMS-g.original.MonotonicMS) > g.bounds.lifetime.duration {
		g.failLocked()
		return ClockSample{}, errProviderIntegration
	}
	if g.claim.start == (CandidateClockDiagnostic{}) {
		g.claim.start = providerClockDiagnostic(now)
	} else if !candidateLedgerClockAdvance(g.claim.start, providerClockDiagnostic(now)) || uint64(now.MonotonicMS-g.claim.start.MonotonicMS) > g.bounds.claim.duration {
		g.failLocked()
		return ClockSample{}, errProviderIntegration
	}
	return now, nil
}

func (g *providerGuardianOwner) failLocked() {
	if g == nil || g.owner != g || g.store == nil || g.store.guardian != g {
		return
	}
	g.failed, g.store.failed = true, true
	if g.integration != nil {
		g.integration.failed = true
	}
	if r := g.store.frontierReplay; r != nil {
		r.failed = true
	}
	if f := g.store.frontierFiles; f != nil {
		f.failed = true
	}
	if g.meter != nil {
		g.meter.mu.Lock()
		g.meter.failed, g.meter.closed = true, true
		g.meter.mu.Unlock()
	}
	g.store.activeMeter.Store(nil)
}

// Origin construction belongs to O, and precedes E0. The independently typed
// E0 replay still has to read and verify this exact immutable object itself.
func (g *providerGuardianOwner) originBeforeEnrollmentLocked(m *providerWorkMeter) (providerIntegrationObject, error) {
	if !g.selectedLocked() || g.initialized || g.origin != "" || g.store.sequence != 0 || g.store.bootstrapOwner == nil || !g.store.bootstrapOwner.writing(g.store) || g.store.bootstrapOwner.meter != m || g.store.activeMeter.Load() != m {
		return providerIntegrationObject{}, errProviderIntegration
	}
	s := g.store
	x, err := newProviderScanCodec(s.config, m)
	if err != nil {
		return providerIntegrationObject{}, errProviderIntegration
	}
	h0 := x.hash("sage-provider-custody-highwater-v1", providerReplayPhaseBytes(s.bootstrapOwner.before, m))
	raw, err := x.guardianOrigin(h0, s.setupTip, providerClockDiagnostic(s.setupOrigin))
	object, objectErr := newProviderIntegrationObjectObserved("guardian-origin", raw, s.config.bounds.maximumMetadataPerCommit, m)
	if err != nil || objectErr != nil {
		return providerIntegrationObject{}, errProviderIntegration
	}
	g.h0, g.origin = h0, object.digest
	return object, nil
}
