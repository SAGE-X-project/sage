package registry011

// Prospective limits for original legacy closure/control work. They do not
// select a profile, authenticate an inventory or qualify a complete hold.
// The four immutable inventory pages cap original objects at4*entries even
// when the selected first-read object allowance is larger.
type providerHistoryLegacyControlLimits struct {
	metadata, objectBytes, firstObjects, firstRaw, entries, attempts, chunks uint64
}

type providerHistoryLegacyControlCost struct {
	object, inventoryParsed, emptyInventory           candidateWorkVector
	firstLoad, firstLookup, firstUnique, modelControl candidateWorkVector
	firstCustody, catalog                             candidateWorkVector
}

// Explicit decoded/logical work only. File contents/metadata are separately
// counted; no native CPU, RAM, timer, authority or eleven-coordinate claim is
// made here. Nonempty inventory construction/sorting is not called by these
// readers: only the original parser and the empty-anchor builder are priced.
func providerHistoryLegacyControlUpper(l providerHistoryLegacyControlLimits) (providerHistoryLegacyControlCost, bool) {
	var empty providerHistoryLegacyControlCost
	if l.metadata == 0 || l.metadata > 1<<20 || l.objectBytes == 0 || l.objectBytes > 32<<20 ||
		l.firstObjects == 0 || l.firstObjects > 1<<20 || l.firstRaw == 0 || l.firstRaw > 64<<20 ||
		l.entries == 0 || l.entries > 256 || l.attempts == 0 || l.attempts > 64 || l.chunks == 0 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	q := func(n uint64) uint64 { return a.sum(a.product(n, n), 1) }
	const h, page uint64 = candidateLedgerMetadataLimit, 64 << 10
	const kind, digest uint64 = 26, 66 // Actual closed legacy kinds and lowerHash.
	domain := uint64(len("sage-provider-guardian-admission-binding-v1"))
	objects := min(l.firstObjects, a.product(4, l.entries))
	raw := min(l.firstRaw, a.product(objects, l.objectBytes))
	key := kind + 1 + digest
	keyPair := a.sum(a.product(4, key), 3) // two copies and their comparison
	adjacent := l.entries - 1
	invDomain := uint64(len("sage-provider-custody-objects-v2"))
	var out providerHistoryLegacyControlCost
	// All objects reserve the domain-bearing codec, including plain request/
	// params/response/TLS objects whose actual constructors skip that codec.
	out.object[2] = l.objectBytes
	out.object[10] = a.sum(q(l.objectBytes), a.product(2, l.objectBytes), domain, 3)
	out.inventoryParsed[2] = page
	out.inventoryParsed[10] = a.sum(q(page), a.product(2, page), invDomain, 3, a.product(adjacent, keyPair))
	out.emptyInventory[10] = a.sum(q(page), a.product(2, page), invDomain, 3)
	// Four raw inventories are decoded once to obtain previous and once by
	// their parser. No sorting or inner-object reconstruction is inside parse.
	// Aggregate object codec squares satisfy sum(n*n)<=P*sum(n)<=P*raw.
	out.firstLoad[2] = a.sum(8*page, raw)
	pairs := a.product(objects, objects-1) / 2
	pairCompare := 2*kind + 2*digest + 1
	objectNames := uint64(len("object-")) + kind + 1 + 64 + 1
	inventoryName := uint64(len("inventory-")) + 64 + 1
	out.firstLoad[10] = a.sum(a.product(4, a.sum(out.inventoryParsed[10], q(page), 133, inventoryName)),
		a.product(l.objectBytes, raw), objects, a.product(2, raw), a.product(objects, a.sum(domain, 3, 133, 2, objectNames)),
		a.product(pairs, pairCompare), out.emptyInventory[10], 133)
	// Each successful selected-object lookup scans all original references,
	// copies its exact bytes, hashes them independently and checks the digest.
	out.firstLookup[10] = a.sum(a.product(objects, pairCompare), a.product(2, l.objectBytes), domain, 3, 133)
	out.firstUnique[10] = a.product(objects, 2*kind+1)
	// initialModelAtLocked: two pin copies, HEAD/immutable comparisons, original
	// frontier/commit hashes, frontier decode + canonical rebuild + comparison.
	// Model/cursor roots and the selected first-P semantic replay are separate.
	f := min(l.metadata, page)
	frontierDomain := uint64(len("sage-provider-custody-frontier-v1"))
	commitDomain := uint64(len("sage-provider-custody-commit-v2"))
	intentDomain := uint64(len("sage-provider-custody-intent-v2"))
	out.modelControl[2] = f
	out.modelControl[10] = a.sum(2*(h+1), 2*(2*h+1), f, frontierDomain, 2, l.metadata, commitDomain, 2,
		q(f), q(page), 2*f+1)
	// First-P C/S custody: prior frontier, then TWO intent decodes per row
	// (the direct check and commit reconstruction). Commit's canonical encoder
	// retains its original1MiB preflight even for a smaller selected metadata.
	out.firstCustody[2] = a.sum(f, a.product(4, l.metadata))
	out.firstCustody[10] = a.sum(q(f), f, frontierDomain, 2, 133, a.product(4, q(l.metadata)),
		a.product(2, q(1<<20)), a.product(2, a.sum(l.metadata, commitDomain, 2, 133, l.metadata, intentDomain, 2, 2*l.metadata+1)),
		uint64(len("frontier"))+22, 2*(uint64(len("intent"))+22), 2*(uint64(len("commit"))+22))
	// Three scan-codec constructors: readPrefix, its final readHighwater,
	// and the catalog's independently rebuilt binding. Conservatively include
	// optional guardian's two encodes, one decode and canonical comparison.
	profile := a.sum(q(l.metadata), l.metadata, uint64(len("sage-provider-store-profile-v6")), 2)
	equal := 2*h + 1
	setup := a.sum(profile, a.product(2, q(l.metadata)), q(h), equal)
	hash := h + uint64(len(providerScanHighwaterDomain)) + 2
	// Fixed encodes8/hashes5/comparisons6. Each prior attempt has6 encodes,
	// two decodes, four hashes, five comparisons and one policy byte copy.
	// readExact names: four fixed highwater names and five per-attempt names.
	fixedNames := 4 * (uint64(len("scan-highwater")) + 22)
	attemptNames := uint64(len("scan-intent")) + 22 + uint64(len("scan-claim")) + 22 + uint64(len("scan-commit")) + 22 + 2*(uint64(len("scan-highwater"))+22)
	fixed := a.sum(a.product(3, setup), a.product(8, q(l.metadata)), a.product(5, hash), a.product(6, equal), fixedNames)
	attempt := a.sum(a.product(6, q(l.metadata)), a.product(2, q(h)), a.product(4, hash), a.product(5, equal), h, 1, attemptNames)
	// Both chunk/command originals are independently decoded/re-encoded in
	// both roots. Expected command adds two encodes and a fifth decode. The
	// extra name calls in the catalog are distinct from reader name calls.
	receiptName := uint64(len("scan-")) + uint64(len("command")) + uint64(len("-v2-")) + 64 + 1
	chunk := a.sum(a.product(6, q(l.metadata)), a.product(5, q(h)), a.product(4, hash), a.product(7, equal), 4*receiptName)
	// The catalog reads references, not object payloads. Its two inventory
	// name calls/page and one ref-key/object-name pair/object are independent
	// of the full first-P load above. General catalog namespace names/map CPU
	// and enumeration remain separate; only actual calls listed here are funded.
	legacyCatalog := a.sum(a.product(4, a.sum(out.inventoryParsed[10], q(page), 133, 2*inventoryName)),
		a.product(objects, a.sum(key, 1, objectNames)), out.emptyInventory[10], 133)
	out.catalog[2] = a.sum(8*page, 3*h, a.product(2*h, l.attempts), a.product(5*h, l.chunks))
	out.catalog[10] = a.sum(fixed, a.product(l.attempts, attempt), a.product(l.chunks, chunk), legacyCatalog)
	if a.failed {
		return empty, false
	}
	return out, true
}

// The successful first-P path has twelve selected-object lookups and three
// unique-kind scans. Other selected policy/admission/T/C/S/HTTP/base64/header
// parsing, request construction, initial-root encoding and cursor construction
// are NOT supplied by this component. Initial models are14G; catalogs retain
// their independently derived16R+16G-2 multiplicity, not one per collection.
func providerHistoryCollectionRepeatedLegacyControlUpper(groups, requests uint64, limits providerHistoryLegacyControlLimits) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	calls, callsOK := providerHistoryCollectionRootCallsBound(groups, requests)
	catalog, catalogOK := providerHistoryCollectionCatalogReadBound(groups, requests, limits.attempts, limits.chunks)
	one, oneOK := providerHistoryLegacyControlUpper(limits)
	if !callsOK || !catalogOK || !oneOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out candidateWorkVector
	for c := range out {
		initial := a.sum(one.modelControl[c], one.firstLoad[c], one.firstCustody[c],
			a.product(12, one.firstLookup[c]), a.product(3, one.firstUnique[c]))
		out[c] = a.sum(a.product(calls.initialModels, initial), a.product(catalog.calls, one.catalog[c]))
	}
	if a.failed {
		return empty, false
	}
	return out, true
}
