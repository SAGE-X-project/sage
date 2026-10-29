package registry011

// Retain originals with the independent custodian, apply the coordinator's
// separate typed delta, then compare only the known changed entries. The
// same-live prefix was privately established before any provisional change.
func (o *providerContinuingOwner) commitLocked(kind, operation string, change *CandidateAdmissionLedgerChange, extra []providerIntegrationObjectInput, clock CandidateClockDiagnostic) error {
	if !o.selectedLocked() || o.change != nil || kind == "admit" && o.state != providerContinuingAdmission ||
		kind == "consume" && o.state != providerContinuingConsumption || kind == "settle" && o.state != providerContinuingSettlement ||
		kind != "admit" && kind != "consume" && kind != "settle" {
		return errProviderIntegration
	}
	s, c := o.store, o.integration
	m := s.activeMeter.Load()
	p, err := providerContinuingChangeData(s, c.root, c.ledger, o.indexes, s.frontierReplay.current, kind, operation, change, extra, clock, m)
	if err != nil {
		return errProviderIntegration
	}
	o.change = p
	cost, err := s.frontierReplay.retainContinuingOwnedLocked(s, p.inventory, p.fresh, p.intent, p.commit, p.frontier, p.after, o)
	if err != nil || cost.objects != uint64(len(p.fresh)) || s.frontierFiles.pin != p.after || s.enrollment != p.root.digest ||
		s.stored != o.stored || s.last != nil || o.indexes.applyRetained(p.delta, cost) != nil ||
		!providerContinuingDeltaEqual(s, o.indexes, s.frontierReplay.indexes, p.delta, o.ticket, m) {
		return errProviderIntegration
	}
	c.root, o.change = p.root, nil
	prefix, ok := providerContinuingPrefixData(o.guardian)
	if !ok {
		return errProviderIntegration
	}
	o.guardian.prefix = prefix
	return nil
}

// T consumes the remaining part of the original G claim. Its five auxiliary
// files were already counted once. This common writer supplies the exact raw
// and physical cost it independently derived before any main write.
func (o *providerContinuingOwner) consumeGuardianStorageLocked(physical uint64) bool {
	if o == nil || o.state != providerContinuingAdmission || o.metadataPhysical != 0 || !o.writingLocked(o.store) {
		return false
	}
	g, p, s := o.guardian, o.change, o.store
	raw, a := providerUpperSum(uint64(len(p.inventory.raw)), uint64(len(p.intent)), uint64(len(p.intent)), uint64(len(p.commit)), uint64(len(p.frontier)))
	for _, object := range p.fresh {
		var ok bool
		raw, ok = providerUpperSum(raw, uint64(len(object.raw)))
		if !ok {
			return false
		}
	}
	head, err := p.after.bytesObserved(s.config, s.activeMeter.Load())
	headRaw, b := providerUpperProduct(uint64(len(head)), 2)
	raw, c := providerUpperSum(raw, headRaw)
	claimPhysical, d := providerUpperSum(g.claimPhysical, physical)
	claimRaw, e := providerUpperSum(g.claimRaw, raw)
	allPhysical, f := providerUpperSum(g.physical, physical)
	allRaw, h := providerUpperSum(g.raw, raw)
	allMetadata, i := providerUpperSum(g.metadata, raw)
	storeRaw, j := providerUpperSum(s.retainedRaw, raw)
	if !a || err != nil || !b || !c || !d || !e || !f || !h || !i || !j ||
		claimPhysical > g.bounds.claim.retained || claimPhysical > o.plan.metadataPhysical || claimRaw > o.plan.metadataRaw ||
		claimRaw > g.bounds.claim.raw || claimRaw > g.bounds.claim.metadata || allPhysical > g.bounds.lifetime.retained ||
		allRaw > g.bounds.lifetime.raw || allMetadata > g.bounds.lifetime.metadata || storeRaw > s.stored || storeRaw > s.config.bounds.continuation.maximumMetadataBytes {
		return false
	}
	o.metadataPhysical, o.metadataRaw = physical, raw
	g.physical, g.raw, g.metadata = allPhysical, allRaw, allMetadata
	// guardian replay.metadata is auxiliary G-file metadata only. The main T
	// raw bytes are charged separately by each typed main retention path.
	return true
}
