package registry011

// This constructor is initial-only and candidate-private. The original F must
// still be PREPARED; a previous retained E1 DATA cannot create the new owner.
// All capacities, the separate P hold and both future envelopes are selected
// before the first F read or I/T0. No endpoint/transport callback is accepted.
func newProviderFirstOperationCandidate(s *providerCustodyStore) (*providerFirstOperationOwner, error) {
	if s == nil {
		return nil, errProviderIntegration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.firstAdmissionOwner
	if !s.valid() || f == nil || f.store != s || f.state != providerFirstAdmissionPrepared || f.operation != nil || s.firstOperation != nil || s.firstLimits == nil || s.frontierFiles == nil || s.sequence != 1 || f.mainPin != s.frontierFiles.pin || s.last != nil {
		return nil, errProviderIntegration
	}
	if f.startLocked() != nil {
		return nil, errProviderIntegration
	}
	var owner *providerFirstOperationOwner
	success := false
	defer func() {
		if !success {
			if owner != nil {
				owner.failLocked()
			} else {
				f.abortLocked()
			}
		}
	}()
	plan, ok := providerFirstOperationPlanUpper(s.config.bounds, s.config.capture, f.scanPin.ordinal, f.records, s.retainedRaw, f.readerOutput)
	held, heldOK := providerScanUpperSum(providerScanOpenUpper(s.config), s.scanControl.held)
	limits := *s.firstLimits
	if !ok || !heldOK || !candidateCapacity(0, s.stored, plan.storage.physical, s.config.bounds.maximumStoredBytes) || !candidateCapacity(0, s.stored, plan.storage.physical, limits.live) || !candidateCapacity(0, s.stored, plan.storage.physical, limits.retained) {
		return nil, errProviderIntegration
	}
	for coordinate, n := range plan.reservation.work {
		if n > limits.operation[coordinate] || n > limits.segment[coordinate] || !candidateCapacity(held.work[coordinate], 0, n, limits.work[coordinate]) {
			return nil, errProviderIntegration
		}
	}
	// P is distinct from the already held O/B/F. All uncertainty retains both.
	s.stored += plan.storage.physical
	owner = &providerFirstOperationOwner{store: s, first: f, plan: plan, config: s.config, policy: *s.config.policy, continuation: *s.config.bounds.continuation, methods: *s.config.bounds.methods, tls: *s.config.bounds.tls, scan: *s.config.bounds.scan, firstBounds: *s.config.bounds.firstAdmission, limits: limits, clock: s.clock, origin: s.setupOrigin, stored: s.stored, state: providerFirstOperationPrepared}
	owner.owner = owner
	s.firstOperation, f.operation = owner, owner
	if !f.planLocked() {
		return nil, errProviderIntegration
	}
	// Only this observed request construction moves before I. Its original
	// proposal event graph stays funded by F, with explicit extra comparisons.
	request, err := newProviderIntegrationRequestObserved("eth_getBlockByNumber", []any{"finalized", false}, f.meter)
	if err != nil || !providerReplayTextEqual(request.head, providerFirstOperationRequestHead, f.meter) || !providerReplayTextEqual(request.body, providerFirstOperationRequestBody, f.meter) || !providerReplayTextEqual(request.wire, providerFirstOperationRequestWire, f.meter) {
		return nil, errProviderIntegration
	}
	owner.request = request
	input, err := f.replayEnrollmentLocked()
	if err != nil {
		return nil, errProviderIntegration
	}
	clock, err := f.sampleLocked()
	if err != nil {
		return nil, errProviderIntegration
	}
	p, err := providerFirstAdmissionProposalWithRequestData(f, input, plan.reservation, providerClockDiagnostic(clock), request)
	if err != nil || p.delta == nil {
		return nil, errProviderIntegration
	}
	custodian, err := f.replayCustodianLocked()
	if err != nil || custodian.replay == input.replay || custodian.replay.indexes == p.indexes || custodian.replay.body.ledger != nil {
		return nil, errProviderIntegration
	}
	if _, err := f.sampleLocked(); err != nil || !f.adoptCustodianLocked(custodian) {
		return nil, errProviderIntegration
	}
	f.plan.used = true
	r := s.frontierReplay
	cost, err := r.retainOwnedLocked(s, p.inventory, p.fresh, p.claim, p.commit, p.frontier, p.after, nil, f)
	if err != nil || cost.objects != 14 || !f.physicalUsed || cost.stored != f.physical || s.stored != owner.stored || s.scanControl.held != f.plan.held || s.sequence != 2 || s.frontierFiles.pin != p.after || s.last != nil || r.body.intent == nil || r.body.anchor == nil || r.current.enrollment != p.root.digest || r.body.intent.anchor != r.body.anchor.digest || p.indexes.applyRetained(p.delta, cost) != nil || !providerFirstOperationModelsEqual(s, p.indexes, r.indexes, f.meter) {
		return nil, errProviderIntegration
	}
	if _, err := f.sampleLocked(); err != nil || f.meter.failed || f.meter.closed || !owner.selectedLocked() {
		return nil, errProviderIntegration
	}
	c := &providerIntegration{store: s, policy: s.config.policy, context: r.body.context, clock: s.clock, primitive: providerPrimitiveBounds{capture: plan.capture, maximumCreatedObjects: 32}, anchor: r.body.anchor, root: p.root, intent: p.intent, ledger: p.ledger, first: request, firstTicket: p.ticket}
	c.owner = c
	owner.integration, owner.indexes = c, p.indexes
	meter := newProviderWorkMeter(c, plan.reservation)
	meter.ticket, meter.maximumClocks, meter.firstOperation = p.ticket.digest, 7, owner
	owner.execution = meter
	// Close F permanently before publishing any new active observer. The
	// replacement is atomic under store.mu; none of r's maps are imported.
	f.state = providerFirstAdmissionRetained
	f.meter.mu.Lock()
	f.meter.closed = true
	f.meter.mu.Unlock()
	owner.attachLocked(meter)
	if !owner.fits(meter) || !c.valid() {
		return nil, errProviderIntegration
	}
	success = true
	return owner, nil
}

// The caller holds store.mu, and has already checked both independently
// reconstructed models. Only private observer pointers are transferred.
func (o *providerFirstOperationOwner) attachLocked(meter *providerWorkMeter) {
	r := o.store.frontierReplay
	r.meter, r.body.meter, r.indexes.meter, r.body.ledger.meter = meter, meter, meter, meter
	o.indexes.meter, o.integration.ledger.meter = meter, meter
	o.store.activeMeter.Store(meter)
}
