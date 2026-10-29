package registry011

// Both original heads, the independently reconstructed G claim, exact T and
// its same-inventory binding are checked under the new execution observer.
// A producer digest or the generic store ACK cannot stand in for this check.
func (o *providerContinuingOwner) compositeLocked() bool {
	if !o.selectedLocked() || o.ticket == nil || !o.fits(o.execution) || !o.guardian.continuingPrefixMatchesLocked() ||
		o.metadataPhysical == 0 || o.change != nil || o.store.frontierFiles == nil || o.store.frontierFiles.pending != nil {
		return false
	}
	s, g, m := o.store, o.guardian, o.execution
	r := s.frontierReplay.body.guardian
	if r == nil || r.failed || r.terminal || r.head != g.pin || r.ticket != o.ticket.digest || r.consumed != o.consumption ||
		r.metadata != g.genesisMetadata+g.auxiliaryMetadata || g.claim == nil || *g.claim != o.claim.attempt {
		return false
	}
	x, err := newProviderScanCodec(s.config, m)
	if err != nil || !s.frontierFiles.readExact("HEAD", providerReplayPhaseBytes(s.frontierFiles.pinRaw, m)) ||
		!s.frontierFiles.readExact(s.name("highwater", s.frontierFiles.pin.ordinal), providerReplayPhaseBytes(s.frontierFiles.pinRaw, m)) {
		return false
	}
	data, err := providerGuardianClaimOriginals(s, m, o.priorGuardian, o.before, o.integration.intent.digest, o.binding.target, o.binding.purpose)
	if err != nil || *data != o.claim {
		return false
	}
	head, err := x.guardianHead(data.head)
	current, a := s.read(s.custody, "GUARDIAN-HEAD", candidateLedgerMetadataLimit)
	if err != nil || a != nil || !x.equal(head, current) {
		return false
	}
	expected, err := providerGuardianBindingForTicket(x, data, o.ticket)
	raw, a := s.object("guardian-admission-binding", expected.digest)
	ticket, b := s.object("ticket", o.ticket.digest)
	if err != nil || a != nil || b != nil || !providerReplayPhaseRawTextEqual(raw, expected.raw, m) ||
		!providerReplayPhaseRawTextEqual(ticket, o.ticket.raw, m) {
		return false
	}
	binding, err := x.parseGuardianAdmissionBinding(raw)
	return err == nil && binding == o.binding && !m.failed
}

func (o *providerContinuingOwner) consumeLocked() (*providerIntegrationPermit, error) {
	if !o.selectedLocked() || o.state != providerContinuingPrepared || o.permit != nil || o.consumption || o.settlement ||
		o.store.sequence != o.before.sequence+1 || !o.compositeLocked() {
		return nil, errProviderIntegration
	}
	c, m := o.integration, o.execution
	clock, err := c.currentClock()
	if err != nil {
		return nil, errProviderIntegration
	}
	consumption, err := providerContinuingConsumptionData(o.ticket, c.ledger.lastConsumption, providerClockDiagnostic(clock), m)
	if err != nil {
		return nil, errProviderIntegration
	}
	change, err := c.ledger.AppendCandidateConsumption(consumption)
	if err != nil {
		return nil, errProviderIntegration
	}
	o.state = providerContinuingConsumption
	if o.commitLocked("consume", consumption.digest, change, []providerIntegrationObjectInput{{"consumption", providerReplayPhaseBytes(consumption.raw, m)}}, providerClockDiagnostic(clock)) != nil ||
		!o.consumption || o.store.sequence != o.before.sequence+2 {
		return nil, errProviderIntegration
	}
	permit := &providerIntegrationPermit{owner: c, ticket: o.ticket, root: c.root, request: o.request, clock: clock, meter: m}
	o.permit, o.state = permit, providerContinuingConsumed
	return permit, nil
}

func (o *providerContinuingOwner) burnLocked(permit *providerIntegrationPermit) bool {
	if !o.selectedLocked() || o.state != providerContinuingConsumed || permit == nil || permit != o.permit || permit.owner != o.integration ||
		permit.used || permit.ack != nil || permit.ticket != o.ticket || permit.request != o.request || permit.root != o.integration.root ||
		permit.root.digest != o.store.enrollment || permit.meter != o.execution || !o.consumption || o.settlement ||
		o.store.sequence != o.before.sequence+2 || o.store.last != nil {
		return false
	}
	// This irreversible private transition precedes every clock, hash, read
	// and possible transport action below; no refusal can reuse this permit.
	permit.used, o.state = true, providerContinuingUsed
	if !o.compositeLocked() {
		return false
	}
	clock, err := o.integration.currentClock()
	if err != nil || !candidateLedgerClockAdvance(providerClockDiagnostic(permit.clock), providerClockDiagnostic(clock)) || candidateLedgerExpired(providerClockDiagnostic(clock), o.ticket.deadline) {
		return false
	}
	var work candidateWorkVector
	if o.previousTicket == nil || o.ticket.previous != o.previousTicket.digest || o.ticket.segment < o.previousTicket.segment || o.ticket.segment > o.previousTicket.segment+1 {
		return false
	}
	work[8] = o.ticket.segment - o.previousTicket.segment
	switch o.ticket.kind {
	case "local-work":
		if o.request != nil || o.capture != (ProviderCaptureLimits{}) {
			return false
		}
	case "rpc-capture":
		if o.request == nil || o.config.bounds.methods == nil || o.request.inputDigest != o.ticket.input || o.request.method != o.ticket.method || o.request.paramsDigest != o.ticket.params {
			return false
		}
		capture, err := o.config.bounds.methods.captureObserved(o.request, o.config.capture, o.execution)
		if err != nil || capture != o.capture {
			return false
		}
		work[0] = 1
	default:
		return false
	}
	if !o.execution.add(work) {
		return false
	}
	permit.clock = clock
	return true
}

func (o *providerContinuingOwner) beginRemainingLocked(permit *providerIntegrationPermit) bool {
	if !o.selectedLocked() || o.state != providerContinuingUsed || permit == nil || permit != o.permit || !permit.used ||
		permit.meter != o.execution || o.store.activeMeter.Load() != o.execution || o.remaining != nil || !o.consumption ||
		o.settlement || o.store.sequence != o.before.sequence+2 {
		return false
	}
	o.execution.mu.Lock()
	o.execution.closed = true
	o.execution.mu.Unlock()
	m := newProviderWorkMeter(o.integration, providerOperationReservation{work: o.plan.reservation.work, execution: o.plan.reservation.remaining})
	m.ticket, m.authority, m.maximumClocks, m.continuing = o.ticket.digest, true, 5, o
	o.remaining, o.state = m, providerContinuingSettlement
	o.attachLocked(m)
	return o.fits(m)
}
