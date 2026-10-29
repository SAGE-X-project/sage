package registry011

import "time"

// This live reservation is installed before the FIRST owned sample/root read.
// It cannot dispatch, return a protected result or select an operating Source.
// Retained E0/initial binding, real group writers and their independent ACKs
// remain mandatory before the future execution role can be constructed.
type providerHistoryProgramOwner struct {
	owner                       *providerHistoryProgramOwner
	store                       *providerCustodyStore
	config                      providerCustodyConfig
	policy                      ProviderPolicyCandidate
	bounds                      providerHistoryProgramBounds
	scan                        providerScanBounds
	continuation                providerLedgerBounds
	methods                     providerMethodBounds
	tls                         providerTLSBounds
	first                       providerFirstAdmissionBounds
	limits                      providerFirstAdmissionLimits
	clock                       *providerOwnedClock
	clockID, clockEpoch         string
	clockOrigin                 time.Time
	clockWindow                 time.Duration
	clockUncertainty            int64
	clockMaximum                uint64
	original                    ClockSample
	held                        providerScanUpper
	preI                        providerScanUpper
	inspection                  providerScanUpper
	initialRead, clockCeiling   uint64
	startupClaimed, failed      bool
	openMeter                   *providerWorkMeter
	openControl                 *providerScanControl
	inspectionControl           *providerScanControl
	preparationMeter, bodyMeter *providerWorkMeter
	preparation, body           providerScanUpper
	inspectionPhase             uint8
	clockClaim                  *providerHistoryClockClaim
}

func (*providerHistoryProgramOwner) UnmarshalJSON([]byte) error { return errProviderIntegration }

func newProviderHistoryProgramBeforeOpenCandidate(s *providerCustodyStore) (*providerHistoryProgramOwner, error) {
	if s == nil || s.owner != s || s.history != nil || s.guardian != nil || s.config.bounds.history == nil ||
		s.config.bounds.guardian != nil || s.custody != nil || s.evidence != nil || s.setupOrigin != (ClockSample{}) ||
		s.firstLimits == nil || s.clock == nil || s.config.policy == nil || !s.config.bounds.valid() {
		return nil, errProviderIntegration
	}
	held, ok := providerHistoryWholeHold(s.config.bounds)
	limits, c := *s.firstLimits, s.clock
	m := s.activeMeter.Load()
	if m == nil || m.owner != m || m.scan == nil || m.scan.owner != m.scan || m.scan.store != s || m.integration != nil || m.scanBody != nil || m.firstAdmission != nil || m.firstOperation != nil || m.upper != s.config.bounds.setupWork || m.maximumClocks != s.config.bounds.setupMaximumClockReads {
		return nil, errProviderIntegration
	}
	if !ok || held.retained != s.stored || held.retained > limits.live || held.retained > limits.retained {
		return nil, errProviderIntegration
	}
	for i, n := range held.work {
		if n > limits.work[i] {
			return nil, errProviderIntegration
		}
	}
	// This scalar check neither reads nor constructs a clock sample. The
	// original backend becomes exclusive to this store for its entire hold.
	c.mu.Lock()
	defer c.mu.Unlock()
	ceiling, sumOK := providerUpperSum(c.read, held.clocks)
	if c.owner != c || c.failed || c.history != nil || c.legacyClaims != 0 || c.policy != s.config.identity.Policy || !sumOK || ceiling > c.maximum {
		return nil, errProviderIntegration
	}
	p := &providerHistoryProgramOwner{store: s, config: s.config, policy: *s.config.policy, bounds: *s.config.bounds.history, scan: *s.config.bounds.scan, continuation: *s.config.bounds.continuation, methods: *s.config.bounds.methods, tls: *s.config.bounds.tls, first: *s.config.bounds.firstAdmission, limits: limits, clock: c, clockID: c.id, clockEpoch: c.epoch, held: held, preI: providerHistoryPreIUpper(s.config.bounds), initialRead: c.read, clockCeiling: ceiling}
	p.owner = p
	p.openMeter, p.openControl = m, m.scan
	p.clockOrigin, p.clockWindow, p.clockUncertainty, p.clockMaximum = c.origin, c.readWindow, c.uncertainty, c.maximum
	s.history, c.history = p, p
	return p, nil
}

// All fields here are scalars/value snapshots. A changed profile, capacity,
// copied owner or reassigned clock cannot preserve the original reservation.
func (p *providerHistoryProgramOwner) matchesLocked() bool {
	if p == nil || p.owner != p || p.failed || p.store == nil || p.store.history != p || p.store.closed || p.store.failed ||
		p.store.config != p.config || p.config.bounds.history == nil || *p.config.bounds.history != p.bounds ||
		p.config.bounds.scan == nil || *p.config.bounds.scan != p.scan || p.store.guardian != nil ||
		p.config.bounds.continuation == nil || *p.config.bounds.continuation != p.continuation ||
		p.config.bounds.methods == nil || *p.config.bounds.methods != p.methods || p.config.bounds.tls == nil || *p.config.bounds.tls != p.tls ||
		p.config.bounds.firstAdmission == nil || *p.config.bounds.firstAdmission != p.first ||
		p.config.policy == nil || *p.config.policy != p.policy || !p.bounds.valid(p.config.bounds) ||
		p.store.firstLimits == nil || *p.store.firstLimits != p.limits || p.store.clock != p.clock || p.clock == nil ||
		p.clock.history != p || p.store.stored != p.held.retained {
		return false
	}
	c := p.clock
	c.mu.Lock()
	clockOK := c.owner == c && !c.failed && !c.historyRevoked && c.history == p && p.clockSelectionMatches(c) && c.read >= p.initialRead && c.read <= p.clockCeiling
	c.mu.Unlock()
	return clockOK && (p.original == (ClockSample{}) && p.store.setupOrigin == (ClockSample{}) ||
		p.original != (ClockSample{}) && p.original == p.store.setupOrigin)
}

func (p *providerHistoryProgramOwner) bindOriginalLocked(sample ClockSample) bool {
	if !p.matchesLocked() || p.original != (ClockSample{}) || p.store.setupOrigin != (ClockSample{}) || p.store.custody != nil || p.store.evidence != nil {
		return false
	}
	c := p.clock
	c.mu.Lock()
	valid := c.owner == c && !c.failed && c.history == p && c.read == p.initialRead+1 && c.last == sample &&
		sample.ClockID == c.id && sample.Epoch == c.epoch && p.clockSelectionMatches(c)
	if valid {
		p.original = sample
		p.store.setupOrigin = sample
	}
	c.mu.Unlock()
	if !valid {
		p.failLocked()
		return false
	}
	return true
}

// Caller holds the backend mutex. None of the selected backend's epoch,
// uncertainty, native origin/window or finite lifetime can be swapped later.
func (p *providerHistoryProgramOwner) clockSelectionMatches(c *providerOwnedClock) bool {
	return c.id == p.clockID && c.epoch == p.clockEpoch && c.policy == p.policy.policy && c.origin == p.clockOrigin && c.readWindow == p.clockWindow && c.uncertainty == p.clockUncertainty && c.maximum == p.clockMaximum
}

// Match inspection subclaims to the one pre-H0 maximum. The pre-I bound
// already includes O; only P+B increments are accumulated here. Nothing grows
// s.stored or releases unused capacity after a smaller actual inspection.
func (p *providerHistoryProgramOwner) claimInspectionLocked(before, preparation, body providerScanUpper) bool {
	if !p.matchesLocked() || p.original == (ClockSample{}) || p.startupClaimed || before != p.inspection {
		return false
	}
	next, ok := providerScanUpperSum(p.inspection, preparation, body)
	total, totalOK := providerScanUpperSum(providerScanOpenUpper(p.config), next)
	if !ok || !totalOK || total.retained > p.preI.retained || total.clocks > p.preI.clocks {
		return false
	}
	for i, n := range total.work {
		if n > p.preI.work[i] {
			return false
		}
	}
	p.inspection = next // Once consumed, failure never restores this prefix.
	p.preparation, p.body = preparation, body
	p.inspectionControl = p.store.scanControl
	p.preparationMeter, p.bodyMeter, p.inspectionPhase = nil, nil, 0
	return true
}

func (p *providerHistoryProgramOwner) claimStartupLocked(plan providerFirstOperationPlan) bool {
	if !p.matchesLocked() || p.original == (ClockSample{}) || p.startupClaimed || plan.storage.physical == 0 ||
		plan.storage.physical > p.bounds.startup.retained || p.bounds.startup.clocks < 11 {
		return false
	}
	for i, n := range plan.reservation.work {
		if n > p.bounds.startup.work[i] {
			return false
		}
	}
	p.startupClaimed = true
	return true
}

// Failure never clears the exclusive backend binding or refunds the lifetime
// hold. Revocation shares the clock mutex with sampling, so a rejected or
// closed owner cannot race a direct sample into a fresh reservation.
func (p *providerHistoryProgramOwner) failLocked() {
	if p == nil || p.owner != p || p.clock == nil {
		return
	}
	p.failed = true
	p.clock.mu.Lock()
	if p.clock.history == p {
		p.clock.historyRevoked = true
	}
	p.clock.mu.Unlock()
}
