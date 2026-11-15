package registry011

// Both fresh initial inventory walks, not a producer cache or first-read role.
// The original completed initializer bounds the SAME four anchored pages and
// aggregate referenced payload by firstRead. File/name work is separate; no
// physical RAM, native clock, derivation or execution authority is returned.
type providerHistoryReconstructionInventoryCost struct {
	objects, raw, duplicatePairs uint64
	parsers, payload, control    candidateWorkVector
	normal, incoming, envelope   candidateWorkVector
}

func providerHistoryReconstructionInventoryUpper(l providerHistoryLegacyControlLimits) (providerHistoryReconstructionInventoryCost, bool) {
	var empty providerHistoryReconstructionInventoryCost
	one, ok := providerHistoryLegacyControlUpper(l)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryReconstructionInventoryCost
	out.objects = min(l.firstObjects, a.product(4, l.entries))
	out.raw = min(l.firstRaw, a.product(out.objects, l.objectBytes))
	out.duplicatePairs = a.product(out.objects, out.objects-1) / 2
	const page, kind, digest uint64 = 64 << 10, 26, 66
	key := kind + 1 + digest
	domain := uint64(len("sage-provider-guardian-admission-binding-v1"))
	inventoryDomain := uint64(len(providerHistoryRecoveryInventoryDomain))
	q := a.sum(a.product(page, page), 1)
	// The parser includes its own decode/hash/string and adjacent reference
	// keys. Independent outer decode, digest and canonical rebuild are below.
	for coordinate := range out.parsers {
		out.parsers[coordinate] = a.product(8, one.inventoryParsed[coordinate])
	}
	// For the same original payloads, SUM(n*n)<=P*SUM(n)<=P*raw.
	// Keep one codec invocation per object, including domainless objects that
	// actually skip that child. Hash and retained string each consume n+1.
	out.payload[2] = a.product(2, out.raw)
	out.payload[10] = a.product(2, a.sum(a.product(l.objectBytes, out.raw), a.product(2, out.raw), a.product(out.objects, domain+4)))
	// Each pass allocates its seen slots, then four object arrays. Each page
	// independently hashes, decodes, parses and canonically encodes. Each
	// object makes its own key and scans only that pass's prior keys, checks
	// its digest and pays both seen/frame copy events. Exactly one binding
	// value has one extra copy. Closing builds a fresh empty root and compares
	// it; it does not invent another empty-root file or reuse the other pass.
	pageOuter := a.sum(a.product(2, q), a.product(3, page), inventoryDomain, 136)
	objects := a.product(out.objects, a.sum(key, 1, 133, 4))
	duplicates := a.product(out.duplicatePairs, 2*key+1)
	out.control[2] = a.product(8, page)
	out.control[10] = a.product(2, a.sum(out.objects, 1, out.objects, 8, a.product(4, pageOuter), objects, duplicates, 2, one.emptyInventory[10], 133))
	for coordinate := range out.normal {
		out.normal[coordinate] = a.sum(out.parsers[coordinate], out.payload[coordinate], out.control[coordinate])
	}
	// ONE still-outstanding original codec/encode/copy guard. Nothing renews
	// the shared recovery meter when pass one completes or an input refuses.
	out.incoming[2] = max(page, l.objectBytes)
	out.incoming[10] = max(q, a.sum(a.product(l.objectBytes, l.objectBytes), 1), a.sum(out.objects, 2), 2*key+1)
	for coordinate := range out.normal {
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed || !providerHistoryReadWorkDomain(out.envelope) {
		return empty, false
	}
	return out, true
}
