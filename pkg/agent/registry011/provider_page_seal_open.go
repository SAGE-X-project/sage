package registry011

// One closed metadata-only transition between completed operations. It is
// not callable with a restored owner or a producer-supplied boundary/target.
func sealProviderContinuingPageCandidate(s *providerCustodyStore) (*providerPageSealData, error) {
	if s == nil {
		return nil, errProviderIntegration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.guardian
	if !s.valid() || g == nil || g.sealing != nil || g.continuing != nil || g.integration == nil ||
		g.integration.ledger == nil || !lowerHash(g.integration.bootstrap) ||
		g.integration.ledger.providerStart != g.integration.bootstrap || s.frontierReplay == nil ||
		s.frontierReplay.body == nil || s.frontierReplay.body.start != g.integration.bootstrap {
		return nil, errProviderIntegration
	}
	if g.startClaimLocked("ledger-page-seal") != nil {
		return nil, errProviderIntegration
	}
	o := &providerPageSealOwner{guardian: g, before: g.main, stored: s.stored}
	o.owner, g.sealing = o, o
	success := false
	defer func() {
		if !success {
			o.failLocked()
		}
	}()
	o.attachLocked()
	if !g.continuingPrefixMatchesLocked() {
		return nil, errProviderIntegration
	}
	plan, ok := providerPageSealProspective(g)
	if !ok {
		return nil, errProviderIntegration
	}
	o.plan = plan
	c, m := g.integration, g.meter
	value, ok := providerGuardianSealBoundaryData(c.ledger, g.indexes, o.before)
	x, err := newProviderScanCodec(s.config, m)
	if !ok || err != nil {
		return nil, errProviderIntegration
	}
	target, err := x.guardianSealTarget(value)
	if err != nil || g.retainClaimLocked(target) != nil {
		return nil, errProviderIntegration
	}
	prior := s.frontierReplay.body.guardian
	if prior == nil || prior.failed || !prior.terminal || !prior.consumed || prior.ticket != c.ledger.lastTicket.digest {
		return nil, errProviderIntegration
	}
	data, err := providerGuardianClaimOriginals(s, m, prior.head, o.before, c.intent.digest, target, "ledger-page-seal")
	if err != nil || data.head != g.pin || data.metadata != g.claimRaw || c.ledger.addProviderRetention(c.ledger.current, 0, data.metadata) != nil {
		return nil, errProviderIntegration
	}
	o.claim = *data
	boundary, err := c.ledger.prepareProviderBoundary(false)
	if err != nil {
		return nil, errProviderIntegration
	}
	o.boundary = boundary
	page := boundary.pageObject
	seal, err := boundary.sealObject(g.indexes.heads)
	value.page, value.seal = page.digest, seal.digest
	binding, bindingErr := providerGuardianSealBindingForBoundary(x, data, value)
	if err != nil || bindingErr != nil {
		return nil, errProviderIntegration
	}
	o.binding, err = x.parseGuardianSealBinding(providerReplayPhaseBytes(binding.raw, m))
	now, clockErr := g.sampleLocked()
	if err != nil || clockErr != nil {
		return nil, errProviderIntegration
	}
	change, err := providerPageSealChangeData(o, page, seal, binding, providerClockDiagnostic(now))
	if err != nil {
		return nil, errProviderIntegration
	}
	o.change = change
	cost, err := s.frontierReplay.retainSealingOwnedLocked(s, change.inventory, change.fresh, change.intent, change.commit, change.frontier, change.after, o)
	if err != nil || cost.objects != 5 || s.stored != o.stored || s.last != nil || s.frontierFiles.pin != change.after ||
		g.indexes.applyRetained(change.delta, cost) != nil ||
		c.ledger.applyProviderBoundary(boundary, providerReplayPhaseBytes(page.raw, m), providerReplayPhaseBytes(seal.raw, m), change.delta.previous) != nil ||
		!providerPageSealModelsEqual(o, cost) {
		return nil, errProviderIntegration
	}
	c.root = change.root
	prefix, ok := providerContinuingPrefixData(g)
	if !ok {
		return nil, errProviderIntegration
	}
	g.prefix, o.retained = prefix, true
	if _, err := g.sampleLocked(); err != nil || !g.continuingPrefixMatchesLocked() {
		return nil, errProviderIntegration
	}
	result := &providerPageSealData{page: page.digest, seal: seal.digest, enrollment: c.root.digest, main: s.frontierFiles.pin, guardian: g.pin, physical: o.physical}
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	g.main, g.ready, g.active = s.frontierFiles.pin, true, false
	g.claim, g.meter, g.sealing = nil, nil, nil
	o.closed = true
	s.activeMeter.Store(nil)
	success = true
	return result, nil
}
