package registry011

// This live bridge preserves the original first-operation handoff. It can be
// created only while that operation still owns its measured R transaction,
// after original S readback and the complete independent model comparison.
// It is not a group acknowledgement, a new budget or a dispatch permission.
// Future group writers must retain and independently replay their own T/C/S.
type providerHistoryProgramStart struct {
	owner      *providerHistoryProgramStart
	parent     *providerHistoryProgramOwner
	first      *providerFirstOperationOwner
	main       providerFrontierHighwater
	admission  string
	ticket     string
	budget     string
	enrollment string
	objects    string
	exchange   string
}

func (*providerHistoryProgramStart) UnmarshalJSON([]byte) error { return errProviderIntegration }

// The original pre-H0 parent records the exact F/first-P owner before I. A
// completed result or a reconstructed first owner cannot initialize it later.
func (p *providerHistoryProgramOwner) bindFirstOwnerLocked(o *providerFirstOperationOwner) bool {
	if !p.matchesLocked() || !p.startupClaimed || p.firstOwner != nil || p.firstIntegration != nil || p.firstIndexes != nil || p.start != nil ||
		o == nil || o.owner != o || o.store != p.store || o.store.firstOperation != o || o.config != p.config || o.origin != p.original ||
		o.state != providerFirstOperationPrepared || o.clock != p.clock || o.first == nil || o.first != p.store.firstAdmissionOwner || o.first.operation != o || o.first.state != providerFirstAdmissionStarted ||
		o.first.meter == nil || o.first.meter != p.bodyMeter || o.first.meter.owner != o.first.meter || o.first.meter.closed || o.first.meter.failed || o.first.meter.firstAdmission != o.first || p.store.activeMeter.Load() != o.first.meter ||
		o.integration != nil || o.indexes != nil || o.store.sequence != 1 || o.store.frontierFiles == nil || o.first.mainPin != o.store.frontierFiles.pin {
		return false
	}
	p.firstOwner, p.firstMain = o, o.first.mainPin
	p.firstAdmission, p.firstMeter = o.first, o.first.meter
	return true
}

// F has just independently retained I/T and compared the two original
// models. Preserve pointers to their exact live coordinator, never its maps.
func (p *providerHistoryProgramOwner) bindFirstIntegrationLocked(o *providerFirstOperationOwner) bool {
	if !p.matchesLocked() || o == nil || p.firstOwner != o || p.firstIntegration != nil || p.firstIndexes != nil || p.start != nil ||
		o.owner != o || o.store != p.store || o.config != p.config || o.origin != p.original || o.state != providerFirstOperationPrepared ||
		o.clock != p.clock || o.first != p.firstAdmission || o.first.operation != o || o.first.state != providerFirstAdmissionStarted || o.first.mainPin != p.firstMain ||
		p.firstMeter.closed || p.firstMeter.failed || p.firstMeter.firstAdmission != o.first ||
		o.integration == nil || o.integration.owner != o.integration || o.integration.store != p.store || o.integration.failed ||
		o.indexes == nil || o.integration.ledger == nil || o.integration.ledger != o.indexes.ledger || o.integration.intent == nil || o.integration.firstTicket == nil ||
		o.store.activeMeter.Load() != o.first.meter || o.store.sequence != 2 || o.store.last != nil {
		return false
	}
	r := o.store.frontierReplay
	if r == nil || r.failed || r.body == nil || r.body.intent == nil || r.body.ledger == nil || r.indexes == nil ||
		r.indexes == o.indexes || r.body.ledger == o.integration.ledger || r.body.intent == o.integration.intent ||
		r.body.intent.digest != o.integration.intent.digest || r.current.enrollment != o.store.enrollment ||
		r.body.ledger.current == nil || o.integration.ledger.current == nil || r.body.ledger.current.digest != o.integration.ledger.current.digest {
		return false
	}
	p.firstIntegration, p.firstIndexes = o.integration, o.indexes
	return true
}

// Called immediately after the original measured model equality in R. The
// caller still holds store.mu until R closes and the same first owner reaches
// DONE. Only scalar identities are preserved here: no repeated unmetered
// replay, encoding, hashing, clock read, storage write or renewed allowance.
func (p *providerHistoryProgramOwner) handoffFirstLocked(o *providerFirstOperationOwner) bool {
	if !p.matchesLocked() || p.start != nil || o == nil || p.firstOwner != o || p.firstIntegration != o.integration || p.firstIndexes != o.indexes ||
		o.owner != o || o.store != p.store || o.clock != p.clock || o.state != providerFirstOperationSettlement || o.first != p.firstAdmission || o.first.operation != o ||
		o.first.state != providerFirstAdmissionRetained || o.first.mainPin != p.firstMain || !o.consumption || !o.settlement ||
		!p.firstMeter.closed || p.firstMeter.failed || p.firstMeter.firstAdmission != o.first ||
		o.execution == nil || !o.execution.closed || !o.execution.claimed || o.execution.failed ||
		o.remaining == nil || o.remaining.closed || o.remaining.failed || o.store.activeMeter.Load() != o.remaining ||
		o.integration == nil || o.integration.failed || o.integration.firstExchange != "" || o.integration.lastExchange != "" ||
		o.integration.intent == nil || o.integration.firstTicket == nil || o.integration.root == nil || o.integration.ledger == nil ||
		o.integration.ledger.current == nil || o.integration.root.digest != o.store.enrollment ||
		o.store.sequence != 4 || o.store.last != nil || o.store.frontierFiles == nil || o.store.frontierFiles.pending != nil {
		return false
	}
	s, c := o.store, o.integration
	r := s.frontierReplay
	if r == nil || r.failed || r.body == nil || r.indexes == nil || r.indexes == o.indexes ||
		r.body.ledger == nil || r.body.ledger == c.ledger || r.body.ledger.current == nil ||
		r.body.firstExchange == "" || r.current.sequence != 4 || r.current.commit != s.tip ||
		r.current.enrollment != s.enrollment || r.current.objects != s.objectsRoot ||
		r.current.budget != c.ledger.current.digest || r.body.ledger.current.digest != c.ledger.current.digest ||
		s.frontierFiles.pin.sequence != 4 || s.frontierFiles.pin.commit != s.tip || s.frontierFiles.pin.setup != s.setupTip {
		return false
	}
	start := &providerHistoryProgramStart{parent: p, first: o, main: s.frontierFiles.pin, admission: c.intent.digest,
		ticket: c.firstTicket.digest, budget: c.ledger.current.digest, enrollment: s.enrollment, objects: s.objectsRoot, exchange: r.body.firstExchange}
	start.owner = start
	p.start = start
	return true
}

// A group constructor must begin from this exact live bridge. This scalar
// check is necessary, never sufficient for group admission: it grants no I/O
// or execution role, returns no Source, and cannot replace original group ACKs.
func (p *providerHistoryProgramOwner) startReadyLocked() bool {
	if !p.matchesLocked() || p.start == nil || p.start.owner != p.start || p.start.parent != p || p.start.first != p.firstOwner ||
		p.firstOwner == nil || p.firstOwner.owner != p.firstOwner || p.firstOwner.state != providerFirstOperationDone ||
		p.firstOwner.config != p.config || p.firstOwner.origin != p.original || p.firstOwner.store != p.store || p.firstOwner.clock != p.clock ||
		p.firstAdmission.origin != p.original || p.firstMeter.owner != p.firstMeter || !p.firstMeter.closed || p.firstMeter.failed || p.firstMeter.firstAdmission != p.firstAdmission ||
		p.firstIndexes == nil || p.firstOwner.integration != p.firstIntegration || p.firstOwner.indexes != p.firstIndexes ||
		p.firstOwner.first == nil || p.firstOwner.first.store != p.store || p.firstOwner.first.operation != p.firstOwner || p.firstOwner.first.state != providerFirstAdmissionRetained || p.firstOwner.first.mainPin != p.firstMain ||
		p.firstOwner.remaining == nil || p.firstOwner.remaining.owner != p.firstOwner.remaining || p.firstOwner.remaining.firstOperation != p.firstOwner || !p.firstOwner.remaining.closed || p.firstOwner.remaining.failed ||
		p.firstOwner.execution == nil || p.firstOwner.execution.owner != p.firstOwner.execution || p.firstOwner.execution.firstOperation != p.firstOwner || !p.firstOwner.execution.closed || !p.firstOwner.execution.claimed || p.firstOwner.execution.failed ||
		p.store.activeMeter.Load() != nil || p.store.sequence != 4 || p.store.last != nil || p.store.frontierFiles == nil || p.store.frontierFiles.pending != nil ||
		p.store.frontierFiles.pin != p.start.main || p.store.tip != p.start.main.commit || p.store.setupTip != p.start.main.setup || p.store.enrollment != p.start.enrollment || p.store.objectsRoot != p.start.objects ||
		p.firstIntegration == nil || p.firstIntegration.owner != p.firstIntegration || p.firstIntegration.store != p.store || p.firstIntegration.clock != p.clock || p.firstIntegration.failed || p.firstIntegration.intent == nil || p.firstIntegration.firstTicket == nil ||
		p.firstIntegration.root == nil || p.firstIntegration.ledger == nil || p.firstIntegration.ledger.current == nil {
		return false
	}
	c, r, start := p.firstIntegration, p.store.frontierReplay, p.start
	return c.intent.digest == start.admission && c.firstTicket.digest == start.ticket && c.root.digest == start.enrollment &&
		c.ledger.current.digest == start.budget && c.firstExchange == start.exchange && c.lastExchange == start.exchange &&
		r != nil && !r.failed && r.body != nil && r.body.ledger != nil && r.body.ledger != c.ledger && r.body.ledger.current != nil &&
		r.indexes != nil && r.indexes != p.firstIndexes && r.body.firstExchange == start.exchange && r.current.sequence == 4 && r.current.admission == start.admission && r.current.budget == start.budget &&
		r.body.ledger.current.digest == start.budget && r.current.commit == start.main.commit && r.current.objects == start.objects && r.current.enrollment == start.enrollment
}
