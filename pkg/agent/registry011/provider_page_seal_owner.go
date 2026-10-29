package registry011

// Separate metadata-only live role. The exhausted page cannot contain a
// synthetic T/C/S operation to authorize its own closure. This role spends
// only the next already-held G claim; it never mints an E/R or work permit.
type providerPageSealOwner struct {
	owner    *providerPageSealOwner
	guardian *providerGuardianOwner
	before   providerFrontierHighwater
	stored   uint64
	boundary *providerLedgerBoundary
	claim    providerGuardianClaimData
	binding  providerGuardianSealBinding
	change   *providerFirstOperationChange
	plan     providerPageSealPlan
	physical uint64
	retained bool
	closed   bool
}

func (*providerPageSealOwner) UnmarshalJSON([]byte) error { return errProviderIntegration }

type providerPageSealData struct {
	page, seal, enrollment string
	main                   providerFrontierHighwater
	guardian               providerGuardianHeadData
	physical               uint64
}

func (*providerPageSealData) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (o *providerPageSealOwner) selectedLocked() bool {
	if o == nil || o.owner != o || o.closed || o.guardian == nil {
		return false
	}
	g := o.guardian
	s := g.store
	return g.selectedLocked() && g.sealing == o && g.continuing == nil && g.active && !g.ready &&
		g.claim != nil && g.claim.purpose == "ledger-page-seal" && g.fits(g.meter) &&
		s.stored == o.stored && s.last == nil && s.frontierFiles != nil && s.frontierReplay != nil &&
		g.integration != nil && g.integration.valid() && g.integration.ledger != nil &&
		g.indexes != nil && g.indexes.ledger == g.integration.ledger && o.physical <= o.plan.physical
}

func (o *providerPageSealOwner) attachLocked() {
	g := o.guardian
	r := g.store.frontierReplay
	r.meter, r.body.meter, r.indexes.meter, r.body.ledger.meter = g.meter, g.meter, g.meter, g.meter
	g.indexes.meter, g.integration.ledger.meter = g.meter, g.meter
	g.store.activeMeter.Store(g.meter)
}

func (o *providerPageSealOwner) failLocked() {
	if o == nil || o.owner != o || o.guardian == nil || o.guardian.sealing != o || o.closed {
		return
	}
	o.guardian.failLocked()
	o.closed = true
}

func (o *providerPageSealOwner) writingLocked(s *providerCustodyStore) bool {
	if !o.selectedLocked() || o.retained || o.physical != 0 || o.guardian.store != s || o.change == nil ||
		o.boundary == nil || o.boundary.owner != o.guardian.integration.ledger || !o.boundary.valid(o.boundary.owner) ||
		o.change.delta == nil || o.change.delta.owner != o.guardian.indexes || o.change.delta.kind != "seal" ||
		o.change.delta.entry.digest != o.binding.seal || o.change.root == nil || o.change.inventory == nil {
		return false
	}
	p := o.change
	return o.guardian.phase == providerGuardianRecorded && o.guardian.pin == o.claim.head &&
		s.frontierFiles.pin == o.before && p.after.ordinal == o.before.ordinal+1 && p.after.sequence == s.sequence+1 &&
		p.inventory.previous == s.objectsRoot && p.root.digest != s.enrollment && len(p.fresh) == 5 &&
		p.fresh[0].kind == "ledger-page" && p.fresh[0].digest == o.binding.page &&
		p.fresh[1].kind == "ledger-seal" && p.fresh[1].digest == o.binding.seal &&
		p.fresh[2].kind == "guardian-seal-binding" && p.fresh[3].kind == "enrollment" && p.fresh[4].kind == "index-mutation"
}
