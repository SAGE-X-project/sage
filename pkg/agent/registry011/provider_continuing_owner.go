package registry011

const (
	providerContinuingAdmission = iota + 1
	providerContinuingPrepared
	providerContinuingConsumption
	providerContinuingConsumed
	providerContinuingUsed
	providerContinuingSettlement
	providerContinuingDone
	providerContinuingFailed
)

// This concrete role is not restored from a ticket, G record or snapshot.
// Only the still-live G may construct it, after consuming the next original
// metadata allowance. Its new P is held before T; its E/R are distinct from
// the closed first operation and every earlier continuing operation.
type providerContinuingOwner struct {
	owner                *providerContinuingOwner
	guardian             *providerGuardianOwner
	store                *providerCustodyStore
	config               providerCustodyConfig
	policy               ProviderPolicyCandidate
	clock                *providerOwnedClock
	original             ClockSample
	integration          *providerIntegration
	indexes              *providerLedgerIndexes
	plan                 providerContinuingPlan
	before               providerFrontierHighwater
	priorGuardian        providerGuardianHeadData
	claim                providerGuardianClaimData
	binding              providerGuardianAdmissionBinding
	ticket               *CandidateAdmissionTicket
	previousTicket       *CandidateAdmissionTicket
	request              *providerIntegrationRequest
	capture              ProviderCaptureLimits
	change               *providerFirstOperationChange
	permit               *providerIntegrationPermit
	execution, remaining *providerWorkMeter
	stored, physical     uint64
	metadataPhysical     uint64
	metadataRaw          uint64
	clocks               uint64
	state                int
	consumption          bool
	settlement           bool
}

func (*providerContinuingOwner) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (o *providerContinuingOwner) selectedLocked() bool {
	if o == nil || o.owner != o || o.store == nil || o.guardian == nil || o.clock == nil || o.state < providerContinuingAdmission || o.state >= providerContinuingDone {
		return false
	}
	g, s := o.guardian, o.store
	return g.selectedLocked() && g.continuing == o && g.active && !g.ready && g.admitted &&
		g.store == s && s.config == o.config && o.config == g.config && providerContinuingProfileMatches(g) && o.config.policy != nil &&
		*o.config.policy == o.policy && o.policy == g.policy && s.clock == o.clock && g.clock == o.clock &&
		s.setupOrigin == o.original && g.original == o.original && s.stored == o.stored && s.last == nil &&
		g.integration == o.integration && g.indexes == o.indexes && o.integration != nil && o.integration.valid() &&
		o.indexes != nil && o.integration.ledger == o.indexes.ledger && o.clocks <= 11 && o.physical <= o.plan.physical
}

func (o *providerContinuingOwner) fits(m *providerWorkMeter) bool {
	if !o.selectedLocked() || o.state < providerContinuingPrepared || o.ticket == nil || o.guardian.phase != providerGuardianAdmission ||
		o.guardian.meter == nil || !o.guardian.meter.closed || o.guardian.meter.failed ||
		m == nil || m.integration != o.integration || m.continuing != o || m.scan != nil || m.guardian != nil ||
		m.firstAdmission != nil || m.firstOperation != nil || m.ticket != o.ticket.digest || o.store.activeMeter.Load() != m {
		return false
	}
	if m.authority {
		return m == o.remaining && o.state == providerContinuingSettlement && m.upper == o.plan.reservation.remaining && m.maximumClocks == 5
	}
	return m == o.execution && o.state >= providerContinuingPrepared && o.state <= providerContinuingUsed && m.upper == o.plan.reservation.execution && m.maximumClocks == 7
}

func (o *providerContinuingOwner) attachLocked(m *providerWorkMeter) {
	r := o.store.frontierReplay
	r.meter, r.body.meter, r.indexes.meter, r.body.ledger.meter = m, m, m, m
	o.indexes.meter, o.integration.ledger.meter = m, m
	o.store.activeMeter.Store(m)
}

func (o *providerContinuingOwner) sampleLocked() (ClockSample, error) {
	if o == nil || o.store == nil || !o.fits(o.store.activeMeter.Load()) {
		return ClockSample{}, errProviderIntegration
	}
	now, err := o.store.samplePrimitive()
	if err != nil || !candidateLedgerClockAdvance(providerClockDiagnostic(o.original), providerClockDiagnostic(now)) ||
		candidateLedgerExpired(providerClockDiagnostic(now), o.ticket.deadline) ||
		uint64(now.MonotonicMS-o.original.MonotonicMS) > o.guardian.bounds.lifetime.duration {
		o.failLocked()
		return ClockSample{}, errProviderIntegration
	}
	return now, nil
}

func (o *providerContinuingOwner) failLocked() {
	if o == nil || o.owner != o || o.guardian == nil || o.guardian.continuing != o || o.state == providerContinuingDone {
		return
	}
	o.state = providerContinuingFailed
	o.guardian.failLocked()
	for _, m := range []*providerWorkMeter{o.execution, o.remaining} {
		if m != nil {
			m.mu.Lock()
			m.failed, m.closed = true, true
			m.mu.Unlock()
		}
	}
}

func (o *providerContinuingOwner) writingLocked(s *providerCustodyStore) bool {
	if !o.selectedLocked() || o.store != s || o.change == nil || o.change.delta == nil || o.ticket == nil ||
		o.change.after.ordinal != s.frontierFiles.pin.ordinal+1 || o.change.after.sequence != s.sequence+1 ||
		o.change.inventory == nil || o.change.inventory.previous != s.objectsRoot || o.change.root == nil ||
		o.change.root.digest == s.enrollment || o.change.delta.owner != o.indexes || o.change.delta.ledger != o.integration.ledger {
		return false
	}
	switch o.state {
	case providerContinuingAdmission:
		return !o.consumption && !o.settlement && s.frontierFiles.pin == o.before &&
			o.guardian.phase == providerGuardianRecorded && s.activeMeter.Load() == o.guardian.meter &&
			len(o.change.fresh) <= 10 && o.change.delta.kind == "admit"
	case providerContinuingConsumption:
		return !o.consumption && !o.settlement && s.sequence == o.before.sequence+1 &&
			o.fits(o.execution) && len(o.change.fresh) == 6 && o.change.delta.kind == "consume"
	case providerContinuingSettlement:
		return o.consumption && !o.settlement && s.sequence == o.before.sequence+2 &&
			o.fits(o.remaining) && (len(o.change.fresh) == 7 || len(o.change.fresh) == 8 || len(o.change.fresh) == 11) && o.change.delta.kind == "settle"
	}
	return false
}

// The concrete writer calls this once before its first byte. T uses the G
// physical allowance already held before H0. C/S consume only this new P.
// No error or unknown outcome decreases either original hold.
func (o *providerContinuingOwner) consumeStorageLocked(physical uint64) bool {
	if o == nil || !o.writingLocked(o.store) || physical == 0 {
		return false
	}
	if o.state == providerContinuingAdmission {
		return o.consumeGuardianStorageLocked(physical)
	}
	if !candidateCapacity(0, o.physical, physical, o.plan.physical) {
		return false
	}
	upper := o.plan.consumption
	if o.state == providerContinuingSettlement {
		upper = o.plan.settlement
	}
	if physical > upper {
		return false
	}
	o.physical += physical
	if o.state == providerContinuingConsumption {
		o.consumption = true
	} else {
		o.settlement = true
	}
	return true
}

// The original first owner retains value snapshots of the pre-I profile.
// Pointer identity alone cannot establish unchanged method/TLS/scan limits.
// These are fixed-size scalar comparisons, not replay or a renewed owner.
func providerContinuingProfileMatches(g *providerGuardianOwner) bool {
	if g == nil || g.first == nil || g.store == nil || g.store.firstLimits == nil {
		return false
	}
	b, first := g.config.bounds, g.first
	return b.continuation != nil && b.methods != nil && b.tls != nil && b.scan != nil && b.firstAdmission != nil &&
		*b.continuation == first.continuation && *b.methods == first.methods && *b.tls == first.tls &&
		*b.scan == first.scan && *b.firstAdmission == first.firstBounds && *g.store.firstLimits == first.limits
}
