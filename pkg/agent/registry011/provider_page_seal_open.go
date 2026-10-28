package registry011

// One closed metadata-only transition between completed operations. It is
// not callable with a restored owner or a producer-supplied boundary/target.
func sealProviderContinuingPageCandidate(s *providerCustodyStore) (*providerPageSealData, error) {
	if s == nil {
		return nil, errProviderIntegration
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := newProviderPageSealCandidateLocked(s)
	if err != nil {
		return nil, errProviderIntegration
	}
	defer o.failLocked() // Successful release has already closed this one-use role.
	if _, err := o.retainLocked(); err != nil {
		return nil, errProviderIntegration
	}
	return o.releaseLocked()
}

// The same lock spans all three closed phases. Only this constructor consumes
// G and creates the one-use role; no callback or restored owner is accepted.
func newProviderPageSealCandidateLocked(s *providerCustodyStore) (*providerPageSealOwner, error) {
	if s == nil {
		return nil, errProviderIntegration
	}
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
	success = true
	return o, nil
}

// Retain both independently checked models before returning any result.
// The old private prefix remains selected during the affected-data check.
func (o *providerPageSealOwner) retainLocked() (providerFrontierRetention, error) {
	var empty providerFrontierRetention
	if !o.selectedLocked() || o.change == nil || o.retained || o.boundary == nil || len(o.change.fresh) != 5 {
		o.failLocked()
		return empty, errProviderIntegration
	}
	success := false
	defer func() {
		if !success {
			o.failLocked()
		}
	}()
	g, change := o.guardian, o.change
	s, c, m := g.store, g.integration, g.meter
	page, seal := change.fresh[0], change.fresh[1]
	cost, err := s.frontierReplay.retainSealingOwnedLocked(s, change.inventory, change.fresh, change.intent, change.commit, change.frontier, change.after, o)
	if err != nil || cost.objects != 5 || s.stored != o.stored || s.last != nil || s.frontierFiles.pin != change.after ||
		g.indexes.applyRetained(change.delta, cost) != nil ||
		c.ledger.applyProviderBoundary(o.boundary, providerReplayPhaseBytes(page.raw, m), providerReplayPhaseBytes(seal.raw, m), change.delta.previous) != nil ||
		!providerPageSealModelsEqual(o, cost) {
		return empty, errProviderIntegration
	}
	c.root = change.root
	o.retained, success = true, true
	return cost, nil
}

func (o *providerPageSealOwner) releaseLocked() (*providerPageSealData, error) {
	if !o.selectedLocked() || !o.retained || o.change == nil {
		o.failLocked()
		return nil, errProviderIntegration
	}
	success := false
	defer func() {
		if !success {
			o.failLocked()
		}
	}()
	g := o.guardian
	s, c, m := g.store, g.integration, g.meter
	prefix, ok := providerContinuingPrefixData(g)
	if !ok {
		return nil, errProviderIntegration
	}
	g.prefix = prefix
	if _, err := g.sampleLocked(); err != nil || !g.continuingPrefixMatchesLocked() {
		return nil, errProviderIntegration
	}
	result := &providerPageSealData{page: o.binding.page, seal: o.binding.seal, enrollment: c.root.digest, main: s.frontierFiles.pin, guardian: g.pin, physical: o.physical}
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
