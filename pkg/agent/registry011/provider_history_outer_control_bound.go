package registry011

// Original outer-control primitives only. Inputs are prospective numeric DATA,
// not a qualified store/profile, source or complete reservation. Parsers do
// not include caller input copies; frame parsing DOES include actual V4 copies
// of its three reconstructed strings, even if an inert legacy meter omits them.
type providerHistoryOuterControlCost struct {
	profile, highwaterBytes, highwaterParsed, predecessor candidateWorkVector
	storedObject, inventoryBuilt, inventoryParsed         candidateWorkVector
	frameBuilt, frameParsed                               candidateWorkVector
}

func providerHistoryOuterControlUpper(metadata, objectBytes, objects uint64) (providerHistoryOuterControlCost, bool) {
	var empty providerHistoryOuterControlCost
	if metadata == 0 || metadata > 1<<20 || objectBytes == 0 || objectBytes > providerHistoryOriginalPackBytes || objects == 0 || objects > 8 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	q := func(n uint64) uint64 { return a.sum(a.product(n, n), 1) }
	const h uint64 = candidateLedgerMetadataLimit
	i := min(metadata, h)
	waterHash := a.sum(h, uint64(len("sage-provider-custody-highwater-v1")), 2)
	var out providerHistoryOuterControlCost
	// All six closed profile domains have the same length. The actual profile
	// encode preflight uses selected metadata, not the smaller highwater cap.
	// Preflight checks remaining work; it is not itself charged actual work.
	out.profile[10] = a.sum(q(metadata), metadata, uint64(len("sage-provider-store-profile-v6")), 2)
	out.highwaterBytes[10] = a.sum(out.profile[10], q(h))
	out.highwaterParsed[2] = h
	out.highwaterParsed[10] = a.sum(out.profile[10], a.product(2, q(h)), a.product(2, h), 1)
	out.predecessor[10] = a.sum(a.product(3, out.profile[10]), q(h), waterHash)
	// Inventory rebuilds every supplied object independently. Its aggregate
	// raw guard is64MiB, even though up to8 individual objects are allowed.
	domain := uint64(max(len(providerHistoryOriginalPackDomain), len(providerHistoryGroupTicketDomain),
		len(providerHistoryGroupConsumptionDomain), len(providerHistoryFailureDomain), len(providerHistoryGroupSettlementDomain)))
	out.storedObject[10] = a.sum(a.product(2, objectBytes), domain, 3)
	raw := min(a.product(objects, objectBytes), uint64(64<<20))
	out.inventoryBuilt[10] = a.sum(a.product(2, raw), a.product(objects, a.sum(domain, 3, 133)),
		q(i), i, 1, i, uint64(len(providerHistoryInventoryDomain)), 2)
	out.inventoryParsed = out.inventoryBuilt
	out.inventoryParsed[10] = a.sum(out.inventoryParsed[10], 133, i, 1, a.product(2, i), 1)
	// Initial/before validation+encoding and predecessor validation rebuild
	// seven profiles in total. Two initial highwater encodes, three frame
	// encodes and predecessor's highwater encode are six separate Q(h) rows.
	frameHashes := a.sum(a.product(3, h), uint64(len(providerHistoryCustodyIntentDomain)),
		uint64(len(providerHistoryCustodyCommitDomain)), uint64(len(providerHistoryCustodyFrontierDomain)), 6)
	out.frameBuilt[10] = a.sum(a.product(7, out.profile[10]), a.product(6, q(h)),
		a.product(3, waterHash), frameHashes, a.product(3, a.sum(h, 1)))
	out.frameParsed = out.frameBuilt
	out.frameParsed[2] = h // Only intent is decoded; commit/frontier are rebuilt.
	out.frameParsed[10] = a.sum(out.frameParsed[10], q(h), a.product(3, a.sum(h, 1)),
		a.product(3, a.sum(a.product(2, h), 1)))
	if a.failed {
		return empty, false
	}
	return out, true
}

// Current/prior successful T/C/S control subgraph; initial first-P/catalog,
// remaining caller/model/readback comparisons and failure/final are separate.
type providerHistoryOuterControlCalls struct {
	frameBuilds, frameParses, inventoryBuilds, inventoryParses uint64
	packObjects, recordObjects, packReads, recordReads         uint64
	highwaterBytes, highwaterParses                            uint64
}

func providerHistoryCollectionOuterControlCallsBound(groups, requests uint64) (providerHistoryOuterControlCalls, bool) {
	var empty providerHistoryOuterControlCalls
	if _, ok := providerHistoryCollectionRootCallsBound(groups, requests); !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	j := a.product(groups, groups-1) / 2
	builds := a.product(3, groups)
	// T10/C7/S2 outer readers, plus one authenticated current-C anchor per
	// seal and two per member. SAME-C does not reparse INNER T/C records.
	frames := a.sum(a.product(20, groups), a.product(42, j), a.product(2, requests))
	inventories := a.sum(a.product(19, groups), a.product(42, j))
	packs := a.sum(a.product(14, groups), a.product(28, j))
	records := a.sum(a.product(22, groups), a.product(42, j))
	packReads := a.sum(a.product(12, groups), a.product(28, j))
	recordReads := a.sum(a.product(19, groups), a.product(42, j))
	// Three writer after-pins and seven current C-reader before-pins. Every
	// archived checkpoint parses a pin; each current-C anchor parses two.
	waterBytes := a.product(10, groups)
	waterParses := a.sum(a.product(42, j), a.product(2, groups), a.product(4, requests))
	if a.failed {
		return empty, false
	}
	return providerHistoryOuterControlCalls{builds, frames, builds, inventories, packs, records,
		packReads, recordReads, waterBytes, waterParses}, true
}

// Includes outer stored-object reconstruction and reader text/digest checks.
// Inventory rows already rebuild their own objects: do not add those inner
// object calls again. All normal T/C/S inventories have at most two objects.
// Other raw/model/getter/profile callers, I/O, native CPU/RAM and complete
// all-coordinate/failure/final/factory costs remain mandatory separate rows.
func providerHistoryCollectionRepeatedOuterControlUpper(groups, requests, metadata, objectBytes uint64) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	calls, callsOK := providerHistoryCollectionOuterControlCallsBound(groups, requests)
	one, oneOK := providerHistoryOuterControlUpper(metadata, objectBytes, 2)
	recordBytes := min(objectBytes, uint64(candidateLedgerMetadataLimit))
	record, recordOK := providerHistoryOuterControlUpper(metadata, recordBytes, 2)
	if !callsOK || !oneOK || !recordOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var work candidateWorkVector
	for c := range work {
		work[c] = a.sum(a.product(calls.frameBuilds, one.frameBuilt[c]), a.product(calls.frameParses, one.frameParsed[c]),
			a.product(calls.inventoryBuilds, one.inventoryBuilt[c]), a.product(calls.inventoryParses, one.inventoryParsed[c]),
			a.product(calls.packObjects, one.storedObject[c]), a.product(calls.recordObjects, record.storedObject[c]),
			a.product(calls.highwaterBytes, one.highwaterBytes[c]), a.product(calls.highwaterParses, one.highwaterParsed[c]))
	}
	// Each independently read object is copied to its stored text and its
	// reconstructed digest is compared to the retained expected reference.
	work[10] = a.sum(work[10], a.product(calls.packReads, a.sum(objectBytes, 1, 133)),
		a.product(calls.recordReads, a.sum(recordBytes, 1, 133)))
	if a.failed {
		return empty, false
	}
	return work, true
}
