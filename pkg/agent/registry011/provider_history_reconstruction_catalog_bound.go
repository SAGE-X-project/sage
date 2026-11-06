package registry011

// Both actual catalog observations and their closure, not the expected-name
// builder, file payload readers or model/semantic reconstruction. The slot
// allowance is the existing declared catalog-slot premise, not native RAM.
// The enclosing source compiler must bind both inputs to the same selection.
type providerHistoryReconstructionCatalogCost struct {
	observations, comparison, closingRoots candidateWorkVector
	normal, incoming, envelope             candidateWorkVector
	slotBytes                              uint64
}

// The entries limit is shared by BOTH roots in ONE observation. Treating it
// as a per-root limit would price another graph. Both independent before/after
// observations are included here; they are not the collection execution
// catalog multiplicity and do not acquire network, write or clock permission.
func providerHistoryReconstructionCatalogUpper(entries, filenameEntryOverhead uint64) (providerHistoryReconstructionCatalogCost, bool) {
	var empty providerHistoryReconstructionCatalogCost
	if !providerHistoryCatalogEntriesSupported(entries) || filenameEntryOverhead == 0 || filenameEntryOverhead > 1<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryReconstructionCatalogCost
	pairs := a.product(entries, entries-1) / 2
	// One allocation copy(N): N+1. Per entry: ReadDir1, read(name)256,
	// Lstat1, clone(name)256 and copy(slot)2, plus511 per prior name.
	// Each root has two four-event identities, open/two Stats/close and
	// its own terminal EOF read: thirteen units, including empty roots.
	one := a.sum(a.product(517, entries), 27, a.product(511, pairs))
	out.observations[10] = a.product(2, one)
	// Both root metadata comparisons, at most N*N name comparisons, then
	// one matched-entry metadata comparison per original entry.
	out.comparison[10] = a.sum(2, a.product(511, a.product(entries, entries)), entries)
	// Two original final roots: identity4, Stat1 and metadata comparison1.
	out.closingRoots[10] = 12
	out.normal[10] = a.sum(out.observations[10], out.comparison[10], out.closingRoots[10])
	// readRootLocked preflights len(entries)*511+3*256+8 BEFORE every
	// ReadDir, including terminal EOF and the first over-capacity refusal.
	// Retain ONE maximum outstanding guard, not a sum over every iteration
	// or two copies for the two observations. Actual work stays in normal.
	out.incoming[10] = a.sum(a.product(511, entries), 776)
	out.envelope[10] = a.sum(out.normal[10], out.incoming[10])
	out.slotBytes = a.product(a.product(4, entries), a.sum(filenameEntryOverhead, 255))
	if a.failed {
		return empty, false
	}
	return out, true
}

// Logical namespace subgraph only: observed catalogs, both independently
// reconstructed expected catalogs and their comparisons. Original final HEAD
// payload/copies, models and member/tail comparisons remain in other source
// rows. No result binds itself to an execution profile or qualifies native
// CPU/backing/peak/lifetime RAM, physical filesystem costs or elapsed time.
type providerHistoryReconstructionNamespaceCost struct {
	catalog                                          providerHistoryReconstructionCatalogCost
	expectedNames, expectedReceipts, expectedMatches candidateWorkVector
	normal, incoming, envelope                       candidateWorkVector
}

func providerHistoryReconstructionNamespaceUpper(entries, filenameEntryOverhead, metadata, chunks uint64) (providerHistoryReconstructionNamespaceCost, bool) {
	var empty providerHistoryReconstructionNamespaceCost
	catalog, ok := providerHistoryReconstructionCatalogUpper(entries, filenameEntryOverhead)
	if !ok || metadata == 0 || metadata > 1<<20 || chunks == 0 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	out := providerHistoryReconstructionNamespaceCost{catalog: catalog}
	pairs := a.product(entries, entries-1) / 2
	// Per valid expected name, at most one255-byte formatting copy and
	// one510-byte equality (member spool role):256+511. add() separately
	// clones255 bytes and copies its slot (256+2), after prior comparisons.
	// The allocation is copy(N), and both independent passes rebuild it.
	oneNames := a.sum(a.product(1026, entries), 1, a.product(511, pairs))
	out.expectedNames[10] = a.product(2, oneNames)
	// addInitialNames rehashes each actual command and chunk independently
	// BEFORE building their two root names. Each receipt has one phase-byte
	// copy(D+1) and one domain-bearing hash(D+domain+2). The per-name row
	// above separately covers formatting, even when both roots share a name.
	domain := uint64(max(len(providerScanCommandV2), len(providerScanChunkV2)))
	perReceipt := a.sum(a.product(2, metadata), domain, 3)
	out.expectedReceipts[10] = a.product(4, a.product(chunks, perReceipt))
	out.expectedMatches[10] = a.product(2, a.product(511, a.product(entries, entries)))
	out.normal[10] = a.sum(catalog.normal[10], out.expectedNames[10], out.expectedReceipts[10], out.expectedMatches[10])
	// An attempted (N+1)th expected name may already have formatted before
	// add() refuses its capacity. Its <=767 units are not a valid extra slot.
	// Keep MAX outstanding room across these source alternatives; do not add
	// it per name or reopen a failed inspection for an after-the-fact charge.
	out.incoming[10] = max(catalog.incoming[10], 767)
	out.envelope[10] = a.sum(out.normal[10], out.incoming[10])
	if a.failed {
		return empty, false
	}
	return out, true
}
