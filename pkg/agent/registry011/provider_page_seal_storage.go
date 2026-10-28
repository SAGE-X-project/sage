package registry011

// G was held before H0. Consume only this claim's exact metadata allocation
// once; main metadata does not enter the G auxiliary-original counter twice.
func (o *providerPageSealOwner) consumeStorageLocked(physical uint64) bool {
	if o == nil || o.guardian == nil || physical == 0 || !o.writingLocked(o.guardian.store) {
		return false
	}
	g, p := o.guardian, o.change
	s, m := g.store, g.meter
	raw, ok := providerUpperSum(uint64(len(p.inventory.raw)), uint64(2*len(p.intent)+len(p.commit)+len(p.frontier)))
	for _, object := range p.fresh {
		var valid bool
		raw, valid = providerUpperSum(raw, uint64(len(object.raw)))
		if !valid {
			return false
		}
	}
	head, err := p.after.bytesObserved(s.config, m)
	raw, a := providerUpperSum(raw, 2*uint64(len(head)))
	claimPhysical, b := providerUpperSum(g.claimPhysical, physical)
	claimRaw, c := providerUpperSum(g.claimRaw, raw)
	allPhysical, d := providerUpperSum(g.physical, physical)
	allRaw, e := providerUpperSum(g.raw, raw)
	allMetadata, f := providerUpperSum(g.metadata, raw)
	storeRaw, h := providerUpperSum(s.retainedRaw, raw)
	if !ok || err != nil || !a || !b || !c || !d || !e || !f || !h ||
		claimPhysical > o.plan.physical || claimRaw > o.plan.raw || claimPhysical > g.bounds.claim.retained ||
		claimRaw > g.bounds.claim.raw || claimRaw > g.bounds.claim.metadata || allPhysical > g.bounds.lifetime.retained ||
		allRaw > g.bounds.lifetime.raw || allMetadata > g.bounds.lifetime.metadata || storeRaw > s.stored ||
		storeRaw > s.config.bounds.continuation.maximumMetadataBytes {
		return false
	}
	o.physical = physical
	g.physical, g.raw, g.metadata = allPhysical, allRaw, allMetadata
	return true
}
