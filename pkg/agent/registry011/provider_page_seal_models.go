package registry011

// Check only the one operation-index delta plus fixed boundary summaries.
// The old two models and their exact prefix were privately verified before
// either proposal; no old map is copied, imported, or traversed here.
func providerPageSealModelsEqual(o *providerPageSealOwner, cost providerFrontierRetention) bool {
	if !o.selectedLocked() || o.change == nil || o.change.delta == nil || o.boundary == nil || !o.boundary.used {
		return false
	}
	g := o.guardian
	s, p, r, old := g.store, g.indexes, g.store.frontierReplay.indexes, g.prefix
	if !g.selectedLocked() || old.owner != g || p != old.coordinator || r != old.retained || p == r ||
		p.ledger != old.coordinatorLedger || r.ledger != old.replay || !p.valid() || !r.valid() ||
		p.heads != r.heads || p.counts != r.counts || p.schemaBytes != r.schemaBytes ||
		p.heads[0] != old.heads[0] || p.heads[2] != old.heads[2] || p.counts[0] != old.counts[0] || p.counts[2] != old.counts[2] ||
		p.counts[1] != old.counts[1]+1 || len(p.tickets) != old.mapSizes[0] || len(r.tickets) != old.mapSizes[0] ||
		len(p.operations) != old.mapSizes[1]+1 || len(r.operations) != old.mapSizes[1]+1 ||
		len(p.segments) != old.mapSizes[2] || len(r.segments) != old.mapSizes[2] ||
		len(s.objects) != old.mapSizes[5]+5 || len(s.bindings) != old.mapSizes[6] || s.stored != old.stored {
		return false
	}
	a, b := p.ledger, r.ledger
	if a == b || a.current == nil || b.current == nil || a.current == b.current || a.lastTicket == nil || b.lastTicket == nil ||
		a.lastTicket == b.lastTicket || a.lastTicket.digest != old.ticket || b.lastTicket.digest != old.ticket ||
		a.lastConsumption == nil || b.lastConsumption == nil || a.lastConsumption == b.lastConsumption ||
		a.lastConsumption.digest != old.consumed || b.lastConsumption.digest != old.consumed ||
		len(a.tickets) != old.mapSizes[3] || len(b.tickets) != old.mapSizes[3] || len(a.segments) != old.mapSizes[4] || len(b.segments) != old.mapSizes[4] ||
		providerContinuingBudgetData(a.current) != providerContinuingBudgetData(b.current) ||
		!providerReplayTextEqual(a.current.raw, b.current.raw, g.meter) {
		return false
	}
	updated := providerContinuingBudgetData(a.current)
	updated.privateMeta = old.budget.privateMeta
	closing, closingOK := providerUpperSum(uint64(len(o.boundary.pageObject.raw)), uint64(len(o.change.delta.entry.raw)))
	private, privateOK := providerUpperSum(old.budget.privateMeta, o.claim.metadata, cost.metadata, o.change.delta.metadata, closing)
	if updated != old.budget || !closingOK || !privateOK || a.current.privateMetadata != private {
		return false
	}
	left, exists := p.operations[o.binding.seal]
	right, other := r.operations[o.binding.seal]
	if !exists || !other || left != right || left.kind != "seal" || left.operation != o.binding.seal || left.sequence != s.sequence ||
		!bytesEqualProviderValueObserved(left.object(), right.object(), g.meter) {
		return false
	}
	c, d := a.continuation, b.continuation
	page, pageOK := providerContinuingPageData(c)
	otherPage, otherOK := providerContinuingPageData(d)
	if !pageOK || !otherOK || page != otherPage || c == d || !c.valid(a) || !d.valid(b) || len(c.records) != 0 ||
		c.page != old.page.page+1 || c.sealedRecords != old.page.sealedRecords+uint64(old.page.records) ||
		c.lastPage != o.binding.page || c.lastSeal != o.binding.seal || c.lastBudget != old.budget.digest ||
		c.sealedPayload != a.current.payload || c.sealedMetadata != a.current.metadata || c.sealedExtra != c.extraMetadata ||
		c.sealedAuxiliary != c.auxiliaryMetadata || c.sealedExternal != c.auxiliaryPayload {
		return false
	}
	guardian := s.frontierReplay.body.guardian
	metadata, metadataOK := providerUpperSum(g.genesisMetadata, g.auxiliaryMetadata)
	return metadataOK && guardian != nil && !guardian.failed && guardian.terminal && guardian.consumed &&
		guardian.ticket == old.ticket && guardian.head == g.pin && guardian.metadata == metadata &&
		g.meter.actual[0] == 0 && g.meter.actual[1] == 0 && !g.meter.failed && !g.meter.closed
}
