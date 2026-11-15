package registry011

// These are support ceilings derived from the existing object, main-commit,
// setup and scan domains. They are not an operating profile or a reservation.
// Keep the general retained-prefix domain separate from the fresh route.
const (
	providerHistoryCatalogObjectCeiling  = 16 << 20
	providerHistoryCatalogCommitCeiling  = 2 << 20
	providerHistoryCatalogSetupCeiling   = 64
	providerHistoryCatalogAttemptCeiling = 64
	providerHistoryCatalogChunkCeiling   = providerHistoryCatalogCommitCeiling + providerHistoryCatalogSetupCeiling + 1
	providerHistoryCatalogGeneralCeiling = 13 + providerHistoryCatalogObjectCeiling +
		7*providerHistoryCatalogCommitCeiling + 5*providerHistoryCatalogSetupCeiling +
		7*providerHistoryCatalogAttemptCeiling + 4*providerHistoryCatalogChunkCeiling

	// E0, I/T, C and the actual exchange S retain these distinct objects.
	providerHistoryCatalogFreshObjects = 7 + 14 + 6 + 11
	// Both roots' original baseline is included exactly once. Replacing a
	// HEAD or SCAN-HEAD does not refund its preceding allocation.
	providerHistoryCatalogFreshFiles      = 48 + providerHistoryCatalogFreshObjects + 4*2
	providerHistoryCatalogFreshStageFiles = 26 + 2*providerHistoryGroupMembers + 1
	providerHistoryCatalogFreshCeiling    = providerHistoryCatalogFreshFiles +
		((1<<20)/3)*providerHistoryCatalogFreshStageFiles
	providerHistoryCatalogEntriesMaximum = max(providerHistoryCatalogGeneralCeiling, providerHistoryCatalogFreshCeiling)
)

// The same finite arithmetic domain is used by source, runtime and readers.
// Neither a supported count nor a cost quote authenticates its namespace or
// makes the requested capacity available on the selected host.
func providerHistoryCatalogEntriesSupported(entries uint64) bool {
	return entries > 0 && entries <= providerHistoryCatalogEntriesMaximum && entries <= uint64(^uint(0)>>1)
}

type providerHistoryInitialCatalogCost struct {
	names, pendingNames, files uint64
}

// General original prefix: scan ordinal, main highwater, setup count, main
// sequence, fresh objects and command/chunk pairs. Do not substitute fresh
// route constants for an arbitrary retained prefix. Both roots share ONE sum.
func providerHistoryInitialCatalogUpper(attempts, highwater, setups, commits, objects, chunks uint64) (providerHistoryInitialCatalogCost, bool) {
	var empty providerHistoryInitialCatalogCost
	if attempts > providerHistoryCatalogAttemptCeiling || highwater > providerHistoryCatalogCommitCeiling+providerHistoryCatalogSetupCeiling ||
		setups > providerHistoryCatalogSetupCeiling+1 || commits > providerHistoryCatalogCommitCeiling ||
		objects > providerHistoryCatalogObjectCeiling || chunks > providerHistoryCatalogChunkCeiling {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	names := a.sum(10, a.product(5, attempts), highwater, a.product(3, setups), a.product(5, commits), objects, a.product(4, chunks))
	files := a.sum(10, a.product(7, attempts), a.product(2, highwater), a.product(3, setups), a.product(5, commits), objects, a.product(4, chunks))
	pending := a.sum(names, 3)
	if a.failed {
		return empty, false
	}
	return providerHistoryInitialCatalogCost{names: names, pendingNames: pending, files: files}, true
}

// Detached requirements for the fresh source compiled below. A diagnostic may
// retain selectedEntries < requiredEntries: reporting an inadequate selection
// is not execution permission and must not silently enlarge the profile.
type providerHistoryCatalogRequirement struct {
	initial                                         providerHistoryInitialCatalogCost
	futureFiles, requiredEntries, requiredNameBytes uint64
	requiredObjects, requiredCommits                uint64
	selectedEntries, selectedNameBytes              uint64
}

func providerHistoryFreshCatalogUpper(storage providerHistoryUniformStorageCost, selected uint64) (providerHistoryCatalogRequirement, bool) {
	var empty providerHistoryCatalogRequirement
	if !providerHistoryCatalogEntriesSupported(selected) || storage.groups == 0 || storage.groups > (1<<20)/3 ||
		storage.future.files == 0 || storage.future.objects == 0 {
		return empty, false
	}
	// This compiler's fresh opener/one initial claim/two one-record chunks
	// finish the first exchange at main sequence and highwater four. It is
	// not a formula for a generic resumed store.
	initial, ok := providerHistoryInitialCatalogUpper(1, 4, 1, 4, providerHistoryCatalogFreshObjects, 2)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	required := a.sum(max(initial.files, initial.pendingNames), storage.future.files)
	nameBytes := a.product(required, 255)
	selectedNames := a.product(selected, 255)
	objects := a.sum(providerHistoryCatalogFreshObjects, storage.future.objects)
	commits := a.sum(4, a.product(3, storage.groups))
	if a.failed {
		return empty, false
	}
	return providerHistoryCatalogRequirement{
		initial: initial, futureFiles: storage.future.files, requiredEntries: required, requiredNameBytes: nameBytes,
		requiredObjects: objects, requiredCommits: commits, selectedEntries: selected, selectedNameBytes: selectedNames,
	}, true
}

// A capacity predicate only. The missing complete production issuer must also
// bind this source selection, all other cost rows, native resources and the
// final serialized profile before it can issue the existing private witness.
func (r providerHistoryCatalogRequirement) within(store providerIntegrationBounds) bool {
	if store.history == nil || r.initial == (providerHistoryInitialCatalogCost{}) || r.requiredEntries == 0 ||
		!providerHistoryCatalogEntriesSupported(r.selectedEntries) || store.maximumObjects <= 0 || store.maximumCommits <= 0 {
		return false
	}
	h := store.history
	return h.catalogEntries == r.selectedEntries && h.catalogEntries >= r.requiredEntries &&
		h.catalogNameBytes >= r.selectedNameBytes && r.requiredObjects <= uint64(store.maximumObjects) &&
		r.requiredCommits <= uint64(store.maximumCommits)
}
