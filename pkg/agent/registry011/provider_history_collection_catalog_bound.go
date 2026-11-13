package registry011

// Successful-path catalog content reads only. No directory metadata, codecs,
// native work, RAM, failure/recovery/final work or whole-program authority.
// This complements (does not replace) the separately scoped prefix/spool rows.
type providerHistoryCollectionCatalogBound struct {
	perCatalog, calls, contentReads uint64
}

// A and C are the original pre-H0 scan maximumAttempts/maximumChunks, never a
// later measured smaller prefix. C remains checked numeric DATA here; its full
// store/scan/profile qualification is still the missing compiler's obligation.
// Passing arbitrary positive values cannot construct a derivation or owner.
func providerHistoryCollectionCatalogReadBound(groups, requests, attempts, chunks uint64) (providerHistoryCollectionCatalogBound, bool) {
	var empty providerHistoryCollectionCatalogBound
	maximumRequests, ok := providerUpperProduct(groups, providerHistoryGroupMembers)
	if !ok || groups == 0 || groups > (1<<20)/3 || requests < groups || requests > maximumRequests || attempts == 0 || attempts > 64 || chunks == 0 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	// One initial catalog: scan readPrefix6+5A, two-root chunk/command4C,
	// and four original legacy inventories. Names/objects add no extra file
	// contents here; their interpretation/metadata are separate cost rows.
	perCatalog := a.sum(10, a.product(5, attempts), a.product(4, chunks))
	// For one successful group: T1/C1/dispatch2 + member(16m-2) + S12
	// =16m+14. Each nonfinal successful advance adds two catalog derivations.
	// Thus the complete successful invocation count is16R+16G-2. This is
	// monotone for fewer realized optional groups/requests, unlike spool squares.
	calls := a.difference(a.sum(a.product(16, requests), a.product(16, groups)), 2)
	content := a.product(perCatalog, calls)
	if a.failed {
		return empty, false
	}
	return providerHistoryCollectionCatalogBound{perCatalog, calls, content}, true
}

// Complete successful collection content-read and owner sample counts, from
// group initialization through final S. This is NOT a complete execution cost:
// bytes, directory/identity metadata, timers/native time/CPU, RAM, all eleven
// work coordinates, failures/recovery and final C0/Dp/C1 remain separate.
type providerHistoryCollectionSuccessfulReadCost struct {
	prefix                                             providerHistoryCollectionReadBound
	catalog                                            providerHistoryCollectionCatalogBound
	remainingContents, contentReads, ownerClockSamples uint64
}

func providerHistoryCollectionSuccessfulReadUpper(groups, requests, firstObjects, attempts, chunks uint64) (providerHistoryCollectionSuccessfulReadCost, bool) {
	var empty providerHistoryCollectionSuccessfulReadCost
	prefix, p := providerHistoryCollectionPrefixReadBound(groups, requests, firstObjects)
	catalog, c := providerHistoryCollectionCatalogReadBound(groups, requests, attempts, chunks)
	if !p || !c {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	// Per successful group, noncatalog current control suffixes: T23,
	// C53, dispatch45, S76. S also rereads all response/TLS spools three
	// times, adding6m. These are disjoint from H/member transcript rows.
	// The last advance guard rereads main HEAD after both healthy passes.
	remaining := a.sum(a.product(197, groups), a.product(6, requests), a.difference(groups, 1))
	// planning is already a SUBSET of healthyPrefixes; do not add it again.
	content := a.sum(prefix.initialModels, prefix.healthyPrefixes, prefix.memberReplay, prefix.memberPreparation,
		prefix.memberRetention, prefix.advance, catalog.contentReads, remaining)
	// Two initialization samples, eight T/C/dispatch/S samples per group,
	// two per member and two per nonfinal advance =10G+2R. This excludes
	// original startup first P and all uncounted native timer/transport calls.
	clocks := a.sum(a.product(10, groups), a.product(2, requests))
	if a.failed {
		return empty, false
	}
	return providerHistoryCollectionSuccessfulReadCost{prefix, catalog, remaining, content, clocks}, true
}
