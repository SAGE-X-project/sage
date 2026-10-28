package registry011

import "reflect"

type providerPageSealPlan struct {
	work                  candidateWorkVector
	raw, physical, memory uint64
}

func (*providerPageSealPlan) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Prospective metadata only. Larger legal page sizes remain supported by
// the data ledger but require a sufficiently qualified original G profile.
// No amount returned here is an actual measurement or a new reservation.
func providerPageSealProspective(g *providerGuardianOwner) (providerPageSealPlan, bool) {
	var empty providerPageSealPlan
	if g == nil || !g.selectedLocked() || g.integration == nil || g.integration.ledger == nil || g.first == nil ||
		g.store.config.bounds.continuation == nil || g.config.bounds.scan == nil {
		return empty, false
	}
	b := g.config.bounds
	page, ok := providerLedgerPageUpper(b.continuation.pageRecords)
	if !ok || page > b.maximumObjectBytes || providerLedgerSealLimit > b.maximumObjectBytes {
		return empty, false
	}
	// Five fresh main objects: page, seal, binding, root and operation index.
	// Seven main envelopes/HEAD allocations plus five G-original allocations.
	raw, a := providerUpperSum(page, providerLedgerSealLimit, 3*candidateLedgerMetadataLimit, 64<<10,
		4*b.maximumMetadataPerCommit, 2*candidateLedgerMetadataLimit, 5*candidateLedgerMetadataLimit)
	files, c := providerUpperProduct(17, b.fileOverheadBytes)
	physical, d := providerUpperSum(raw, files)
	n := max(b.maximumMetadataPerCommit, uint64(64<<10), page, providerLedgerSealLimit)
	q, e := providerUpperProduct(n, n)
	q, f := providerUpperSum(q, 1)
	l, h := providerUpperSum(n, n, 101)
	// Source rows: the larger 10-object C physical/typed graph (552/3106),
	// G originals and binding (128/1024), and closed page/boundary/codec
	// rebuilds (128/1024). Each full page codec is bounded by N, including
	// all records; the actual per-record loop remains within the page cap.
	decoded, j := providerUpperProduct(n, 552+128+128)
	q, k := providerUpperProduct(q, 552+128+128)
	l, m := providerUpperProduct(l, 3106+1024+1024)
	units, p := providerUpperSum(q, l)
	work := candidateWorkVector{0, 0, decoded, 0, 0, 0, 0, 0, 0, raw, units}
	if !a || !c || !d || !e || !f || !h || !j || !k || !m || !p ||
		raw > g.bounds.claim.raw || raw > g.bounds.claim.metadata || physical > g.bounds.claim.retained ||
		physical > g.bounds.claim.live || !providerContinuingWorkFits(work, g.bounds.claim.work) {
		return empty, false
	}
	memory, memoryOK := providerPageSealMemoryUpper(g, raw, n)
	if !memoryOK {
		return empty, false
	}
	return providerPageSealPlan{work: work, raw: raw, physical: physical, memory: memory}, true
}

func providerPageSealMemoryUpper(g *providerGuardianOwner, raw, maximum uint64) (uint64, bool) {
	if g == nil || g.store == nil || g.integration == nil || g.integration.ledger == nil || g.config.bounds.continuation == nil || g.config.bounds.scan == nil {
		return 0, false
	}
	b, s := g.config.bounds, g.store
	ledger := g.integration.ledger
	old, oldOK := providerContinuingBootstrapPlan(g)
	if !oldOK {
		return 0, false
	}
	retained, a := providerUpperSum(s.retainedRaw, raw)
	newRaw, c := providerUpperProduct(raw, 5)
	metadata, d := providerUpperProduct(b.maximumMetadataPerCommit, 16)
	scratch, e := providerUpperSum(newRaw, metadata, maximum)
	operations, f := providerUpperSum(s.sequence, 1)
	models, h := b.continuation.residentUpper(uint64(len(ledger.tickets)), operations, uint64(len(ledger.segments)), uint64(len(s.objects))+5, retained, scratch)
	// The third required-reference map, finite inventories/seen maps, both
	// boundary page maps and all closed codec maps remain independently paid.
	entries, j := providerUpperSum(uint64(len(s.objects)), 5, 3*5, 2*b.continuation.pageRecords, 256)
	collections, k := providerUpperProduct(entries, max(b.continuation.objectMemoryOverhead, b.scan.filenameEntryOverhead))
	var native uint64
	for _, row := range []struct{ bytes, count uint64 }{
		{uint64(reflect.TypeFor[providerPageSealOwner]().Size()), 1},
		{uint64(reflect.TypeFor[providerWorkMeter]().Size()), 1},
		{uint64(reflect.TypeFor[providerLedgerBoundary]().Size()), 2},
		{uint64(reflect.TypeFor[providerFirstOperationChange]().Size()), 1},
		{uint64(reflect.TypeFor[providerIndexDelta]().Size()), 2},
		{uint64(reflect.TypeFor[providerGuardianSealBinding]().Size()), 2},
		{uint64(reflect.TypeFor[providerGuardianClaimData]().Size()), 2},
		{uint64(reflect.TypeFor[providerPageSealData]().Size()), 1},
	} {
		part, ok := providerUpperProduct(row.bytes, row.count)
		var added bool
		native, added = providerUpperSum(native, part)
		if !ok || !added {
			return 0, false
		}
	}
	// Keep the old local continuation's entire prospective peak, including
	// its earlier first-operation peak. No immediate garbage collection is
	// assumed when the metadata-only owner becomes live.
	total, l := providerUpperSum(models, collections, native, old.memory)
	return total, a && c && d && e && f && h && j && k && l && total <= b.scan.maximumResidentBytes && total <= g.bounds.claim.resident
}
