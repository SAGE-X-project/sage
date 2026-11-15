package registry011

// Successful owner identities and paired directory enumerations only, from
// group initialization through final S. Writer-wrapper identities are already
// in the separate write component and must not be charged a second time here.
// These are numeric DATA rows, not native filesystem/CPU/RAM qualification.
type providerHistoryCollectionMetadataCost struct {
	directoryPairs, identityCalls, directoryReadCalls uint64
	observedUpper, preflightPadding                   uint64
	work                                              candidateWorkVector
}

func providerHistoryCollectionSuccessfulMetadataUpper(groups, requests, entries, nameBytes uint64) (providerHistoryCollectionMetadataCost, bool) {
	var empty providerHistoryCollectionMetadataCost
	maximumRequests, ok := providerUpperProduct(groups, providerHistoryGroupMembers)
	if !ok || groups == 0 || groups > (1<<20)/3 || requests < groups || requests > maximumRequests ||
		!providerHistoryCatalogEntriesSupported(entries) || nameBytes == 0 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	// T4/C4/dispatch2/member(16m-2)/S11 directory pairs per group,
	// plus two on every nonfinal advance. Catalog derivations are different:
	// their stored-map checks and the discarded S preparation have other counts.
	pairs := a.difference(a.sum(a.product(16, requests), a.product(21, groups)), 2)
	// Initialization6; each group's T13/C10/dispatch6/S12; each member9;
	// each nonfinal advance6. Includes first-P identities in actual full H
	// passes, but not the member's raw-only transcript refresh.
	identities := a.sum(a.product(47, groups), a.product(9, requests))
	// Each owner identity emits eight read(0) units. A paired directory scan
	// emits eight identity units, read+compare per name and two EOF units.
	directoryObserved := a.sum(a.product(2, nameBytes), a.product(2, entries), 10)
	observed := a.sum(a.product(pairs, directoryObserved), a.product(8, identities))
	// ReadDir(1) preflights read(255), even when it returns EOF and records
	// only read(0). Retain 255 extra per pair so either root's last preflight
	// fits without borrowing from another phase or shrinking the original hold.
	padding := a.product(255, pairs)
	units := a.sum(observed, padding)
	readCalls := a.product(pairs, a.sum(entries, 2))
	if a.failed {
		return empty, false
	}
	var work candidateWorkVector
	work[10] = units
	return providerHistoryCollectionMetadataCost{pairs, identities, readCalls, observed, padding, work}, true
}
