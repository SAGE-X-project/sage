package registry011

// Candidate-private read-only request after the retained BootstrapStart.
// Its canonical request, method class and complete new holds are fixed before T.
// It accepts no endpoint, peer, transport or callback and selects no Source.
func newProviderContinuingRPCCandidate(s *providerCustodyStore, method string, params any) (*providerContinuingOwner, error) {
	if s == nil {
		return nil, errProviderIntegration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.guardian
	if !s.valid() || g == nil || g.integration == nil || g.integration.ledger == nil || g.integration.ledger.lastTicket == nil ||
		!lowerHash(g.integration.bootstrap) || g.integration.ledger.providerStart != g.integration.bootstrap ||
		!lowerHash(g.integration.firstExchange) || s.firstLimits == nil || s.last != nil ||
		s.config.bounds.methods == nil || s.config.bounds.tls == nil {
		return nil, errProviderIntegration
	}
	if g.startClaimLocked("rpc-admission") != nil {
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
	request, err := newProviderIntegrationRequestObserved(method, params, g.meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	capture, err := s.config.bounds.methods.captureObserved(request, s.config.capture, g.meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	o.request, o.capture = request, capture
	plan, ok := providerContinuingRPCPlan(g, request, capture)
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
	target, err := x.guardianTicketTarget("rpc-admission", c.intent.digest, "rpc-capture", request.inputDigest, request.method, request.paramsDigest)
	if err != nil || g.retainClaimLocked(target) != nil {
		return nil, errProviderIntegration
	}
	prior := s.frontierReplay.body.guardian
	if prior == nil || prior.failed || !prior.terminal || !prior.consumed || prior.ticket != c.ledger.lastTicket.digest {
		return nil, errProviderIntegration
	}
	o.priorGuardian = prior.head
	data, err := providerGuardianClaimOriginals(s, g.meter, prior.head, o.before, c.intent.digest, target, "rpc-admission")
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
	ticket, err := providerContinuingTicketData(c.policy, c.intent, c.ledger.lastTicket, segment, "rpc-capture", request.inputDigest, request.method, request.paramsDigest, plan.reservation, providerClockDiagnostic(now), g.meter)
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
	if err != nil || o.commitLocked("admit", ticket.digest, change, []providerIntegrationObjectInput{{"request", providerReplayPhaseBytes(request.wire, g.meter)}, {"params", providerReplayPhaseBytes(request.params, g.meter)}, {"guardian-admission-binding", providerReplayPhaseBytes(binding.raw, g.meter)}, {"ticket", providerReplayPhaseBytes(ticket.raw, g.meter)}}, providerClockDiagnostic(now)) != nil {
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
