package registry011

// Successful T/C/member/S allocation and the explicitly instrumented physical
// writer primitives only. This is numeric DATA, not a whole-program hold,
// filesystem qualification, resident-memory bound or production derivation.
type providerHistoryCollectionWriteCost struct {
	storage                               providerHistoryStorageCharge
	writes, syncs, renames, identityReads uint64
	work                                  candidateWorkVector
}

// objectBytes and wireBytes are original pre-H0 maxima, not later observed
// smaller payloads. Each pack can use objectBytes; small metadata is bounded by
// the original fixed 8KiB codec limit and each TLS original by 64KiB.
func providerHistoryCollectionSuccessfulWriteUpper(groups, requests, objectBytes, wireBytes, overhead uint64) (providerHistoryCollectionWriteCost, bool) {
	var empty providerHistoryCollectionWriteCost
	maximumRequests, ok := providerUpperProduct(groups, providerHistoryGroupMembers)
	if !ok || groups == 0 || groups > (1<<20)/3 || requests < groups || requests > maximumRequests ||
		objectBytes == 0 || objectBytes > providerHistoryOriginalPackBytes || wireBytes == 0 || wireBytes > 16<<20 || overhead == 0 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	// Two packs (T requests and S results), three inner small objects, and
	// seven outer metadata allocations per T/C/S. Response/TLS spools are
	// additional originals even though S also retains their result pack.
	payload := a.sum(a.product(a.product(2, groups), objectBytes), a.product(requests, wireBytes))
	metadata := a.sum(a.product(a.product(24, groups), candidateLedgerMetadataLimit), a.product(requests, 64<<10))
	raw := a.sum(payload, metadata)
	objects := a.sum(a.product(5, groups), a.product(2, requests))
	// T9/C8/S9 allocations and two spools per member. HEAD replacement
	// remains charged; the old allocation is never refunded after rename.
	files := a.sum(a.product(26, groups), a.product(2, requests))
	physical := a.sum(raw, a.product(files, overhead))
	renames := a.product(3, groups)
	syncs := a.sum(a.product(2, files), renames)
	// Each write wrapper: two four-event identities plus OpenFile and the
	// post-write close observation. Surrounding owner/catalog reads are not
	// counted here. Each rename also has its separate directory sync.
	identityReads := a.product(10, files)
	units := a.sum(raw, files, syncs, renames, identityReads)
	if a.failed {
		return empty, false
	}
	var work candidateWorkVector
	work[9], work[10] = raw, units
	return providerHistoryCollectionWriteCost{
		storage: providerHistoryStorageCharge{physical: physical, raw: raw, payload: payload, metadata: metadata, objects: objects, files: files},
		writes:  files, syncs: syncs, renames: renames, identityReads: identityReads, work: work,
	}, true
}
