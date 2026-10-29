package registry011

// No policy/host/Source is selected here. This private, initial-continuation
// constructor can only consume the selected live G after its first completed
// request. It has one closed local command, not a caller-supplied callback.
func newProviderContinuingBootstrapCandidate(s *providerCustodyStore) (*providerContinuingOwner, error) {
	if s == nil {
		return nil, errProviderIntegration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.guardian
	if !s.valid() || g == nil || g.integration == nil || g.integration.ledger == nil || g.integration.ledger.lastTicket == nil ||
		g.integration.ledger.lastTicket.sequence != 0 || g.integration.bootstrap != "" || g.integration.ledger.providerStart != "" ||
		g.integration.first == nil || !lowerHash(g.integration.firstExchange) || s.firstLimits == nil || s.last != nil {
		return nil, errProviderIntegration
	}
	if g.startClaimLocked("bootstrap-admission") != nil {
		return nil, errProviderIntegration
	}
	var o *providerContinuingOwner
	success := false
	defer func() {
		if !success {
			if o != nil {
				o.failLocked()
			} else {
				g.failLocked()
			}
		}
	}()
	o = &providerContinuingOwner{guardian: g, store: s, config: g.config, policy: g.policy, clock: g.clock, original: g.original, integration: g.integration, indexes: g.indexes, before: g.main, stored: s.stored, state: providerContinuingAdmission}
	o.owner, g.continuing = o, o
	o.attachLocked(g.meter)
	// Only the exact two prior caches established by first R may advance.
	// No decoded G record or comparison of producer snapshots creates this.
	if !g.continuingPrefixMatchesLocked() {
		return nil, errProviderIntegration
	}
	plan, ok := providerContinuingBootstrapPlan(g)
	if !ok || !candidateCapacity(0, s.stored, plan.physical, s.config.bounds.maximumStoredBytes) ||
		!candidateCapacity(0, s.stored, plan.physical, s.firstLimits.live) || !candidateCapacity(0, s.stored, plan.physical, s.firstLimits.retained) {
		return nil, errProviderIntegration
	}
	clock := s.clock
	clock.mu.Lock()
	clockReads, clockSum := providerUpperSum(11, g.bounds.claim.clocks)
	clockOK := clockSum && clock.owner == clock && !clock.failed && candidateCapacity(0, clock.read, clockReads, clock.maximum)
	clock.mu.Unlock()
	if !clockOK {
		return nil, errProviderIntegration
	}
	// G was already held before H0. Only the new protected C/S P is added.
	s.stored += plan.physical
	o.plan, o.stored = plan, s.stored
	x, err := newProviderScanCodec(s.config, g.meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	c := o.integration
	target, err := x.guardianTicketTarget("bootstrap-admission", c.intent.digest, "local-work", c.firstExchange, "", "")
	if err != nil || g.retainClaimLocked(target) != nil {
		return nil, errProviderIntegration
	}
	prior := s.frontierReplay.body.guardian
	if prior == nil || prior.failed || !prior.terminal || prior.ticket != c.ledger.lastTicket.digest {
		return nil, errProviderIntegration
	}
	o.priorGuardian = prior.head
	data, err := providerGuardianClaimOriginals(s, g.meter, prior.head, o.before, c.intent.digest, target, "bootstrap-admission")
	if err != nil || data.head != g.pin || data.metadata != g.claimRaw {
		return nil, errProviderIntegration
	}
	o.claim = *data
	if c.ledger.addProviderRetention(c.ledger.current, 0, data.metadata) != nil {
		return nil, errProviderIntegration
	}
	segment, segmentOK := providerContinuingSegment(c.ledger, plan.reservation)
	if segmentOK && segment == c.ledger.lastTicket.segment {
		plan.reservation.execution[8], plan.reservation.work[8] = 0, 0
	}
	o.plan, o.previousTicket = plan, c.ledger.lastTicket
	now, sampleErr := g.sampleLocked()
	if !segmentOK || sampleErr != nil {
		return nil, errProviderIntegration
	}
	ticket, err := providerContinuingTicketData(c.policy, c.intent, c.ledger.lastTicket, segment, "local-work", c.firstExchange, "", "", plan.reservation, providerClockDiagnostic(now), g.meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	binding, err := providerGuardianBindingForTicket(x, data, ticket)
	if err != nil {
		return nil, errProviderIntegration
	}
	o.binding, err = x.parseGuardianAdmissionBinding(providerReplayPhaseBytes(binding.raw, g.meter))
	if err != nil {
		return nil, errProviderIntegration
	}
	o.ticket = ticket
	change, err := c.ledger.AppendCandidateTicket(ticket)
	if err != nil || o.commitLocked("admit", ticket.digest, change, []providerIntegrationObjectInput{{"guardian-admission-binding", providerReplayPhaseBytes(binding.raw, g.meter)}, {"ticket", providerReplayPhaseBytes(ticket.raw, g.meter)}}, providerClockDiagnostic(now)) != nil {
		return nil, errProviderIntegration
	}
	// The T receipt stays composite and private. It cannot be handed to the
	// old generic acquisition path or used to restore another operation.
	g.phase = providerGuardianAdmission
	g.meter.mu.Lock()
	g.meter.closed = true
	g.meter.mu.Unlock()
	m := newProviderWorkMeter(c, plan.reservation)
	m.ticket, m.maximumClocks, m.continuing = ticket.digest, 7, o
	o.execution, o.state = m, providerContinuingPrepared
	o.attachLocked(m)
	if !o.fits(m) {
		return nil, errProviderIntegration
	}
	success = true
	return o, nil
}

// Move at most to the next segment when the same lifetime permits it. This
// never constructs a new ledger, admission, epoch or uncharged namespace.
func providerContinuingSegment(ledger *CandidateAdmissionLedger, reservation providerOperationReservation) (uint64, bool) {
	if ledger == nil || !ledger.valid() || ledger.continuation == nil || ledger.lastTicket == nil || ledger.lastTicket.segment == ^uint64(0) {
		return 0, false
	}
	n := ledger.lastTicket.segment
	work := reservation.work
	work[8] = 0 // Staying on the same segment does not create a new segment.
	if candidateFits(ledger.segments[n], work, ledger.limits[1]) {
		return n, true
	}
	if uint64(len(ledger.segments)) >= ledger.continuation.bounds.maximumRecords || !candidateFits(candidateBudgetTotals{}, reservation.work, ledger.limits[1]) {
		return 0, false
	}
	return n + 1, true
}
