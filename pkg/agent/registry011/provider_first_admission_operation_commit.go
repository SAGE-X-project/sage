package registry011

// The whole caller owns store.mu. Independent original retention precedes
// the separate provisional model charge and every private candidate receipt.
func (o *providerFirstOperationOwner) commitLocked(kind, operation string, change *CandidateAdmissionLedgerChange, extra []providerIntegrationObjectInput, clock CandidateClockDiagnostic) error {
	if !o.selectedLocked() || o.integration == nil || o.indexes == nil || o.change != nil || kind == "consume" && o.state != providerFirstOperationConsumption || kind == "settle" && o.state != providerFirstOperationSettlement || kind != "consume" && kind != "settle" {
		return errProviderIntegration
	}
	s, c := o.store, o.integration
	meter := s.activeMeter.Load()
	p, err := providerFirstOperationChangeData(s, c.root, c.ledger, o.indexes, s.frontierReplay.current, kind, operation, change, extra, clock, meter)
	if err != nil {
		return errProviderIntegration
	}
	o.change = p
	cost, err := s.frontierReplay.retainOperationOwnedLocked(s, p.inventory, p.fresh, p.intent, p.commit, p.frontier, p.after, nil, nil, o)
	if err != nil || cost.objects != uint64(len(p.fresh)) || s.frontierFiles.pin != p.after || s.enrollment != p.root.digest || s.stored != o.stored || s.last != nil || o.indexes.applyRetained(p.delta, cost) != nil || !providerFirstOperationModelsEqual(s, o.indexes, s.frontierReplay.indexes, meter) {
		return errProviderIntegration
	}
	c.root, o.change = p.root, nil
	return nil
}

func (o *providerFirstOperationOwner) consumeLocked() (*providerIntegrationPermit, error) {
	if !o.selectedLocked() || o.state != providerFirstOperationPrepared || o.integration == nil || o.permit != nil || o.consumption || o.settlement || o.store.sequence != 2 || o.store.activeMeter.Load() != o.execution {
		return nil, errProviderIntegration
	}
	c, meter := o.integration, o.execution
	clock, err := c.currentClock()
	if err != nil {
		return nil, errProviderIntegration
	}
	consumption, err := providerFirstOperationConsumptionData(c.firstTicket, providerClockDiagnostic(clock), meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	change, err := c.ledger.AppendCandidateConsumption(consumption)
	if err != nil {
		return nil, errProviderIntegration
	}
	o.state = providerFirstOperationConsumption
	if o.commitLocked("consume", consumption.digest, change, []providerIntegrationObjectInput{{"consumption", providerReplayPhaseBytes(consumption.raw, meter)}}, providerClockDiagnostic(clock)) != nil || !o.consumption || o.store.sequence != 3 {
		return nil, errProviderIntegration
	}
	// This private receipt is meaningful only to this live candidate owner.
	// No generic store ACK is created and store.last remains nil.
	permit := &providerIntegrationPermit{owner: c, ticket: c.firstTicket, root: c.root, request: o.request, clock: clock, meter: meter}
	o.permit, o.state = permit, providerFirstOperationConsumed
	return permit, nil
}

func (o *providerFirstOperationOwner) burnLocked(permit *providerIntegrationPermit) bool {
	if !o.selectedLocked() || o.state != providerFirstOperationConsumed || permit == nil || permit != o.permit || permit.owner != o.integration || permit.used || permit.ack != nil || permit.ticket != o.integration.firstTicket || permit.request != o.request || permit.root != o.integration.root || permit.root.digest != o.store.enrollment || permit.meter != o.execution || o.store.activeMeter.Load() != o.execution || !o.consumption || o.settlement || o.store.sequence != 3 || o.store.last != nil {
		return false
	}
	// Burn before sampling, hashing, dialing or sending; even expiry cannot
	// recover this candidate receipt or create a second network invocation.
	permit.used, o.state = true, providerFirstOperationUsed
	clock, err := o.integration.currentClock()
	if err != nil || !candidateLedgerClockAdvance(providerClockDiagnostic(permit.clock), providerClockDiagnostic(clock)) || candidateLedgerExpired(providerClockDiagnostic(clock), permit.ticket.deadline) {
		return false
	}
	meter := o.execution
	if !providerReplayTextEqual(o.request.head, providerFirstOperationRequestHead, meter) || !providerReplayTextEqual(o.request.body, providerFirstOperationRequestBody, meter) || !providerReplayTextEqual(o.request.wire, providerFirstOperationRequestWire, meter) {
		return false
	}
	input, ok := providerIntegrationObjectDigestObserved("request", providerReplayPhaseBytes(o.request.wire, meter), meter)
	params := providerReplayDigest("sage-provider-params-v1", providerReplayPhaseBytes(o.request.params, meter), meter)
	if !ok || input != permit.ticket.input || params != permit.ticket.params || permit.ticket.method != o.request.method {
		return false
	}
	var began candidateWorkVector
	began[0], began[8] = 1, 1
	if !meter.add(began) {
		return false
	}
	permit.clock = clock
	return true
}

// R was independently held before I. A failed E is closed permanently, never
// reopened or accepted as actual. Switching observer does not alter T0 or P.
func (o *providerFirstOperationOwner) beginRemainingLocked(permit *providerIntegrationPermit) bool {
	if !o.selectedLocked() || o.state != providerFirstOperationUsed || permit == nil || permit != o.permit || !permit.used || permit.meter != o.execution || o.store.activeMeter.Load() != o.execution || o.remaining != nil || !o.consumption || o.settlement || o.store.sequence != 3 {
		return false
	}
	o.execution.mu.Lock()
	o.execution.closed = true
	o.execution.mu.Unlock()
	meter := newProviderWorkMeter(o.integration, providerOperationReservation{work: o.plan.reservation.work, execution: o.plan.reservation.remaining})
	meter.ticket, meter.authority, meter.maximumClocks, meter.firstOperation = permit.ticket.digest, true, 5, o
	o.remaining, o.state = meter, providerFirstOperationSettlement
	o.attachLocked(meter)
	return o.fits(meter)
}
