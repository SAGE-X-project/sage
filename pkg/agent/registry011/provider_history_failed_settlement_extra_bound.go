package registry011

// Additions to the already separate successful-S structural components.
// Failure may retain a third object alongside prefix results and settlement.
// Existing successful-prefix/frame work is not charged again here. Catalog,
// namespace, recording/typed context, native/RAM and full failure composition
// still need their original source-derived bounds before any factory exists.
type providerHistoryFailedSettlementExtraCost struct {
	body                            providerHistoryFailureBodyCost
	outer, reads, writes, directRaw candidateWorkVector
	storage                         providerHistoryStorageCharge
	extra                           providerHistoryStageWorkCost
}

// Conditional BYTE/allocation DATA only. The selected caps must describe the
// same original source before H0. Keeping the successful S upper plus these
// additions also covers an absent partial-results pack without any refund.
// This quote neither admits a stage nor qualifies a filesystem or observer.
func providerHistoryFailedSettlementExtraUpper(requestBytes, wireBytes, maximum, metadata, overhead uint64, certificates providerTLSBounds) (providerHistoryFailedSettlementExtraCost, bool) {
	var empty providerHistoryFailedSettlementExtraCost
	if overhead == 0 {
		return empty, false
	}
	body, bodyOK := providerHistoryFailureBodyUpper(requestBytes, wireBytes, maximum, certificates)
	base, baseOK := providerHistoryOuterControlUpper(metadata, maximum, 2)
	failed, failedOK := providerHistoryOuterControlUpper(metadata, maximum, 3)
	object, objectOK := providerHistoryOuterControlUpper(metadata, body.minimumObjectBytes, 1)
	if !bodyOK || !baseOK || !failedOK || !objectOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	out := providerHistoryFailedSettlementExtraCost{body: body}
	f := body.minimumObjectBytes
	for coordinate := range out.outer {
		// Only THIS S's one inventory construction and two readbacks change
		// arity. Each child already rebuilds all its own inventory objects.
		built := a.difference(failed.inventoryBuilt[coordinate], base.inventoryBuilt[coordinate])
		parsed := a.difference(failed.inventoryParsed[coordinate], base.inventoryParsed[coordinate])
		out.outer[coordinate] = a.sum(a.product(3, object.storedObject[coordinate]), built, a.product(2, parsed))
	}
	// The three external failure-object constructors are distinct from each
	// inventory's internal rebuild. Two readers copy the newly read bytes and
	// compare their digests; the writer also compares the body's own digest.
	// Body-parser caller byte copies stay inside body.parsed, not here again.
	out.outer[10] = a.sum(out.outer[10], a.product(2, a.sum(f, 1, 133)), 133)
	out.reads[10] = a.product(2, a.sum(f, 1))
	out.writes[9], out.writes[10] = f, a.sum(f, 13)
	// The actual writer caller copies the new raw object once. Failed S
	// also clones its stage string (at most seven bytes) for the checkpoint.
	// These two copy events are outside the file/body/outer primitives.
	out.directRaw[10] = a.sum(f, 1, 8)
	// One new file has write+two syncs+two identities/open/close. The existing
	// S rename/directory sync stays in the old row. Failure layout preserves
	// metadata/payload separately, including the rejected wire sentinel.
	payload := a.sum(wireBytes, 1)
	meta := a.sum(uint64(len(providerHistoryFailureMagic)), 4, providerHistoryFailureHeaderBytes)
	out.storage = providerHistoryStorageCharge{physical: a.sum(f, overhead), raw: f, payload: payload, metadata: meta, objects: 1, files: 1}
	for coordinate := range out.extra.work {
		out.extra.work[coordinate] = a.sum(body.failedClosure[coordinate], out.outer[coordinate], out.reads[coordinate], out.writes[coordinate], out.directRaw[coordinate])
	}
	if a.failed {
		return empty, false
	}
	return out, true
}
