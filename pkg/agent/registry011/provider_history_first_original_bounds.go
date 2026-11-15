package registry011

import "strconv"

// This is an explicit sublimit of the original future hold. It is not a
// whole-program sufficient budget, a new hold, or a production certificate.
// The unfinished whole-source compiler must still justify both independent
// passes, every later replay, memory, failure and final-page work before H0.
type providerHistoryFirstPReadBounds struct {
	objects, raw uint64
	work         candidateWorkVector
}

// One original firstPReading interval, including its optional ordered-read
// recorder. The same row can therefore constrain both ordinary initialization
// and recorded dispatch replay. It excludes the surrounding main-control,
// model/ledger roots, transcript sealing and native CPU/backing/RAM work.
// Constituents are included here, not additional disjoint whole-program rows.
func providerHistoryFirstPReadUpper(l providerHistoryLegacyControlLimits, capture ProviderCaptureLimits) (providerHistoryFirstPReadBounds, bool) {
	var empty providerHistoryFirstPReadBounds
	legacy, legacyOK := providerHistoryLegacyControlUpper(l)
	interpretation, interpretationOK := providerHistoryFirstPInterpretationUpper(l.objectBytes, capture)
	if !legacyOK || !interpretationOK {
		return empty, false
	}
	cursor := providerHistoryCursorInitializationUpper()
	var a providerHistoryReadArithmetic
	var work candidateWorkVector
	for c := range work {
		work[c] = a.sum(legacy.firstLoad[c], a.product(12, legacy.firstLookup[c]),
			a.product(3, legacy.firstUnique[c]), legacy.firstCustody[c], interpretation.total[c], cursor.cursor[c])
	}
	// Four inventories, at most min(firstObjects,4*entries) objects and five
	// C/S control files. Before EACH read, p.raw+maximum must fit firstRaw;
	// this funds every prospective read at its actual prefix, not just a sum
	// of successful lengths. Two identity calls add sixteen read(0) events.
	files := a.sum(min(l.firstObjects, a.product(4, l.entries)), 9)
	reads := a.sum(l.firstRaw, files, 16)
	// store.read records inside this interval when replayReads is active:
	// SHA(r+1), two owned-name copies2*(n+1), descriptor129; n<=255.
	// Thus recording is at most firstRaw+642*files, independently of reads.
	// The recorder constructor and rolling transcript roots run OUTSIDE.
	recording := a.sum(l.firstRaw, a.product(642, files))
	work[10] = a.sum(work[10], reads, recording)
	out := providerHistoryFirstPReadBounds{objects: l.firstObjects, raw: l.firstRaw, work: work}
	if a.failed || !out.valid() {
		return empty, false
	}
	return out, true
}

// The existing constructor owns two independent first readers before any T.
// Every later original T window keeps its own uniform prefix count, including
// post-S advancement in that SAME window. These are decoded/logical DATA rows
// for firstPReading only, not the surrounding ledger/control/cursor roots or
// a complete stage, future hold, native/RAM qualification or authority.
type providerHistoryFirstPReadCompositionCost struct {
	first                        providerHistoryFirstPReadBounds
	initialization, stage, whole candidateWorkVector
}

func providerHistoryCollectionFirstReadCompositionUpper(groups, requests uint64, l providerHistoryLegacyControlLimits, capture ProviderCaptureLimits) (providerHistoryFirstPReadCompositionCost, bool) {
	var empty providerHistoryFirstPReadCompositionCost
	calls, callsOK := providerHistoryCollectionUniformRootCallsBound(groups, requests)
	first, firstOK := providerHistoryFirstPReadUpper(l, capture)
	if !callsOK || !firstOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	out := providerHistoryFirstPReadCompositionCost{first: first}
	for coordinate, units := range first.work {
		out.initialization[coordinate] = a.product(2, units)
		out.stage[coordinate] = a.product(calls.initialModels, units)
		out.whole[coordinate] = a.sum(out.initialization[coordinate], a.product(groups, out.stage[coordinate]))
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

func (b providerHistoryFirstPReadBounds) valid() bool {
	if b.objects == 0 || b.objects > 1<<20 || b.raw == 0 || b.raw > 64<<20 || b.work[2] == 0 || b.work[10] == 0 {
		return false
	}
	for i, n := range b.work {
		if i != 2 && i != 10 && n != 0 {
			return false
		}
	}
	return true
}

func (b providerHistoryFirstPReadBounds) object() map[string]any {
	return map[string]any{"maximum_objects": strconv.FormatUint(b.objects, 10),
		"maximum_raw_bytes": strconv.FormatUint(b.raw, 10), "work": candidateLedgerWorkObject(b.work)}
}

// No new window/clock is installed. Scalar checks run under the same original
// meter mutex and constrain, rather than reset, its cumulative observations.
func (o *providerHistoryGroupOwner) firstPWorkFits(m *providerWorkMeter, incoming candidateWorkVector) bool {
	if !o.firstPReading {
		return true
	}
	if o.parent == nil || o.parent.bounds.collection == nil || !o.parent.collectionMatchesLocked() {
		return false
	}
	b := o.parent.collectionInput.firstRead
	if !b.valid() {
		return false
	}
	for i, n := range incoming {
		if m.actual[i] < o.firstPStart[i] || !candidateCapacity(0, m.actual[i]-o.firstPStart[i], n, b.work[i]) {
			return false
		}
	}
	return true
}

// The common initial-model commitment binds both independent readers to the
// same actual selected first P. Legacy profiles retain their exact old bytes.
func providerHistoryInitialRootObserved(b providerHistoryProgramBounds, profile string, pin providerFrontierHighwater, exchange, budget, inventory string, selected *FinalizedHeader, meter *providerWorkMeter) string {
	v := map[string]any{"format": "sage-provider-history-initial-model-v1", "program": b.program, "profile": profile, "cost_revision": b.costs,
		"initial_frontier": pin.frontier, "first_exchange": exchange, "budget": budget, "inventory": inventory}
	if b.collection != nil {
		if selected == nil || !b.collection.selectedObserved(*selected, meter) {
			return ""
		}
		v["collection_manifest"] = b.collection.digest
		v["selected_first_finalized"] = map[string]string{"hash": selected.Hash, "parentHash": selected.ParentHash,
			"number": strconv.FormatUint(selected.Number, 10), "timestamp": strconv.FormatUint(selected.Timestamp, 10)}
	}
	raw := providerReplayCanonical(v, candidateLedgerMetadataLimit, meter)
	if len(raw) == 0 {
		return ""
	}
	return providerReplayDigest("sage-provider-history-initial-model-v1", raw, meter)
}
