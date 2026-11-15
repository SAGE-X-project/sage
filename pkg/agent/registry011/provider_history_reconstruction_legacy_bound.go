package registry011

// Two fresh initial control prefixes, before the separate semantic model.
// These are decoded/logical DATA rows only. File/name/catalog I/O, inventory
// walks, scan roots, native resources and execution authority are separate.
type providerHistoryReconstructionLegacyCost struct {
	profiles, highwaters, setups, anchor candidateWorkVector
	main, lookups, control               candidateWorkVector
	profileCalls, setupPairs             uint64
	normal, incoming, envelope           candidateWorkVector
}

func providerHistoryReconstructionLegacyUpper(l providerHistoryLegacyControlLimits, setupRecoveries uint64) (providerHistoryReconstructionLegacyCost, bool) {
	var empty providerHistoryReconstructionLegacyCost
	one, ok := providerHistoryLegacyControlUpper(l)
	if !ok || setupRecoveries > 64 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	q := func(n uint64) uint64 { return a.sum(a.product(n, n), 1) }
	const h, page, commitGuard uint64 = candidateLedgerMetadataLimit, 64 << 10, 1 << 20
	d, s := l.metadata, setupRecoveries
	highwater, frontier, enrollment := min(d, h), min(d, page), min(l.objectBytes, page)
	objects := min(l.firstObjects, a.product(4, l.entries))
	hash := func(n uint64, domain string) uint64 { return a.sum(n, uint64(len(domain)), 2) }
	var out providerHistoryReconstructionLegacyCost
	// Each parse rebuilds its profile once; every predecessor check does so
	// THREE times: both valid calls and the prior highwater's bytes call.
	out.profileCalls = a.product(2, a.sum(17, a.product(4, s)))
	profile := a.sum(q(d), hash(d, "sage-provider-store-profile-v6"))
	out.profiles[10] = a.product(out.profileCalls, profile)
	parses, predecessors := a.sum(5, s), a.sum(4, s)
	out.highwaters[2] = a.product(2, a.product(parses, highwater))
	out.highwaters[10] = a.product(2, a.sum(a.product(parses, a.sum(q(highwater), q(h), 2*h+1)),
		a.product(predecessors, a.sum(q(h), hash(h, "sage-provider-custody-highwater-v1")))))
	// Original setup plus S recovered setups, and S independent highwater
	// checks. Each pair decodes the claim twice, rebuilds its diagnostic once
	// and reconstructs its commit once. Neither claim copy is cached.
	pairs, originals := a.sum(1, a.product(2, s)), a.sum(1, s)
	out.setupPairs = a.product(2, pairs)
	out.setups[2] = a.product(2, a.product(pairs, a.product(2, d)))
	pair := a.sum(a.product(3, q(d)), q(h), 2*265,
		hash(264, "sage-provider-setup-operation-v1"), hash(d, "sage-provider-authority-setup-v1"))
	originalHashes := a.sum(hash(d, "sage-provider-authority-setup-commit-v1"), hash(d, "sage-provider-authority-setup-v1"))
	out.setups[10] = a.product(2, a.sum(a.product(pairs, pair), a.product(originals, originalHashes),
		a.product(s, hash(d, "sage-provider-authority-setup-commit-v1")), a.product(a.sum(3, a.product(5, s)), a.sum(a.product(2, max(d, h)), 1))))
	// This fresh empty anchor is additional to each inventory walk's close.
	out.anchor[10] = a.product(2, a.sum(one.emptyInventory[10], q(d), a.product(2, 2*d+1), hash(d, "sage-provider-custody-anchor-v1"), 133))
	// Four main records per pass. Each has FOUR intent decodes, TWO frontier
	// decodes, TWO enrollment decodes, and another inventory parser inside
	// the independently rebuilt frontier intent. Preserve every native guard.
	out.main[2] = a.product(8, a.sum(a.product(4, d), a.product(2, frontier), a.product(2, enrollment), one.inventoryParsed[2]))
	codecs := a.sum(a.product(4, q(d)), a.product(2, q(frontier)), a.product(2, q(enrollment)), one.inventoryParsed[10])
	encodes := a.sum(q(commitGuard), q(page), a.product(2, q(h)), a.product(2, q(d)))
	hashes := a.sum(hash(d, "sage-provider-custody-intent-v2"), hash(d, "sage-provider-custody-commit-v2"),
		hash(frontier, "sage-provider-custody-frontier-v1"), a.product(2, hash(enrollment, "sage-provider-enrollment-v2")))
	copies := a.sum(a.product(3, enrollment+1), page+1, a.product(3, d+1))
	comparisons := a.sum(a.product(3, 2*d+1), commitGuard, d, 1, 2*h+1, 2*page+1)
	out.main[10] = a.product(8, a.sum(codecs, encodes, hashes, copies, comparisons))
	// Four enrollment and three ticket/consumption/settlement lookups in each
	// pass. They scan only that pass's original frames, then copy at most P.
	out.lookups[10] = a.product(14, a.sum(a.product(objects, 2*26+2*66+2), l.objectBytes, 1))
	// One prefix allocation, setup/record appends, final four digests, and the
	// first enrollment's independently joined/copied/hashed operation.
	out.control[10] = a.product(2, a.sum(a.product(3, s), 9, a.product(2, originals), 8, 4*133, 2*265,
		hash(264, "sage-provider-enroll-operation-v1"), 133))
	for coordinate := range out.normal {
		out.normal[coordinate] = a.sum(out.profiles[coordinate], out.highwaters[coordinate], out.setups[coordinate], out.anchor[coordinate],
			out.main[coordinate], out.lookups[coordinate], out.control[coordinate])
	}
	// ONE still-outstanding guard across both passes. The unchanged commit
	// encoder's 1MiB guard dominates smaller codec/copy/compare events here.
	out.incoming[2] = max(d, page)
	out.incoming[10] = q(commitGuard)
	for coordinate := range out.envelope {
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed || !providerHistoryReadWorkDomain(out.envelope) {
		return empty, false
	}
	return out, true
}
