package registry011

import "strings"

// Private local admission after four independently completed RPCs. These
// digests are lookup hints, never imported permits or an outstanding parent.
func newProviderContinuingPinCandidate(s *providerCustodyStore, agent string, digests [4]string) (*providerContinuingOwner, error) {
	if s == nil || !rpcAgent(agent) {
		return nil, errProviderIntegration
	}
	for i, digest := range digests {
		if !lowerHash(digest) {
			return nil, errProviderIntegration
		}
		for j := 0; j < i; j++ {
			if digest == digests[j] {
				return nil, errProviderIntegration
			}
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.guardian
	if !s.valid() || g == nil || g.integration == nil || g.integration.ledger == nil || g.integration.ledger.lastTicket == nil || !lowerHash(g.integration.bootstrap) || g.integration.ledger.providerStart != g.integration.bootstrap || g.integration.pin != "" || !lowerHash(g.integration.firstExchange) || s.firstLimits == nil || s.last != nil || s.config.bounds.methods == nil || s.config.bounds.tls == nil {
		return nil, errProviderIntegration
	}
	if g.startClaimLocked("local-admission") != nil {
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
	if !g.continuingPrefixMatchesLocked() {
		return nil, errProviderIntegration
	}
	c, m := o.integration, g.meter
	selected, err := providerContinuingPinSelected(s, c, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	raw := providerPinInputObserved(c.bootstrap, agent, m)
	input, err := newProviderIntegrationObjectObserved("pin-input", raw, 4096, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	command := &providerContinuingPinCommand{parent: o, input: input, selected: selected}
	for i, digest := range digests {
		if !m.event(providerCopyEvent, uint64(len(digest))) {
			return nil, errProviderIntegration
		}
		command.exchanges[i] = strings.Clone(digest)
	}
	command.requests, err = providerPinRequestsObserved(selected, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	for i, request := range command.requests {
		command.captures[i], err = s.config.bounds.methods.captureObserved(request, s.config.capture, m)
		if err != nil {
			return nil, errProviderIntegration
		}
	}
	o.pin = command
	plan, ok := providerContinuingPinPlan(g, command)
	if !ok || !candidateCapacity(0, s.stored, plan.physical, s.config.bounds.maximumStoredBytes) || !candidateCapacity(0, s.stored, plan.physical, s.firstLimits.live) || !candidateCapacity(0, s.stored, plan.physical, s.firstLimits.retained) {
		return nil, errProviderIntegration
	}
	clock := s.clock
	clock.mu.Lock()
	reads, exact := providerUpperSum(11, g.bounds.claim.clocks)
	clockOK := exact && clock.owner == clock && !clock.failed && candidateCapacity(0, clock.read, reads, clock.maximum)
	clock.mu.Unlock()
	if !clockOK {
		return nil, errProviderIntegration
	}
	s.stored += plan.physical
	o.plan, o.stored = plan, s.stored
	x, err := newProviderScanCodec(s.config, m)
	if err != nil {
		return nil, errProviderIntegration
	}
	target, err := x.guardianTicketTarget("local-admission", c.intent.digest, "local-work", input.digest, "", "")
	if err != nil || g.retainClaimLocked(target) != nil {
		return nil, errProviderIntegration
	}
	prior := s.frontierReplay.body.guardian
	if prior == nil || prior.failed || !prior.terminal || !prior.consumed || prior.ticket != c.ledger.lastTicket.digest {
		return nil, errProviderIntegration
	}
	o.priorGuardian = prior.head
	data, err := providerGuardianClaimOriginals(s, m, prior.head, o.before, c.intent.digest, target, "local-admission")
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
	ticket, err := providerContinuingTicketData(c.policy, c.intent, c.ledger.lastTicket, segment, "local-work", input.digest, "", "", plan.reservation, providerClockDiagnostic(now), m)
	if err != nil {
		return nil, errProviderIntegration
	}
	binding, err := providerGuardianBindingForTicket(x, data, ticket)
	if err != nil {
		return nil, errProviderIntegration
	}
	o.binding, err = x.parseGuardianAdmissionBinding(providerReplayPhaseBytes(binding.raw, m))
	if err != nil {
		return nil, errProviderIntegration
	}
	o.ticket = ticket
	change, err := c.ledger.AppendCandidateTicket(ticket)
	if err != nil || o.commitLocked("admit", ticket.digest, change, []providerIntegrationObjectInput{{"pin-input", providerReplayPhaseBytes(input.raw, m)}, {"guardian-admission-binding", providerReplayPhaseBytes(binding.raw, m)}, {"ticket", providerReplayPhaseBytes(ticket.raw, m)}}, providerClockDiagnostic(now)) != nil {
		return nil, errProviderIntegration
	}
	g.phase = providerGuardianAdmission
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	e := newProviderWorkMeter(c, plan.reservation)
	e.ticket, e.maximumClocks, e.continuing = ticket.digest, 7, o
	o.execution, o.state = e, providerContinuingPrepared
	o.attachLocked(e)
	if !o.fits(e) {
		return nil, errProviderIntegration
	}
	success = true
	return o, nil
}

func providerContinuingPinSelected(s *providerCustodyStore, c *providerIntegration, meter *providerWorkMeter) (string, error) {
	if s == nil || c == nil || meter == nil || s.activeMeter.Load() != meter || !lowerHash(c.bootstrap) {
		return "", errProviderIntegration
	}
	raw, err := s.object("bootstrap", c.bootstrap)
	m, valid := providerReplayJSON(raw, meter).(map[string]any)
	if err != nil || !valid || m["format"] != "sage-provider-start-v2" || m["admission_digest"] != c.intent.digest || m["selected_finality_exchange"] != c.firstExchange || !providerCandidateDigest(m["selected_block_hash"]) {
		return "", errProviderIntegration
	}
	selected := contentString(m["selected_block_hash"])
	if !meter.event(providerCopyEvent, uint64(len(selected))) {
		return "", errProviderIntegration
	}
	return strings.Clone(selected), nil
}
