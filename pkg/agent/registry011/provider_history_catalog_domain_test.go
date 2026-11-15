package registry011

import "testing"

func TestProviderHistoryInitialCatalogSeparatesNamesPendingAndUnrefundedAllocations(t *testing.T) {
	for _, tc := range []struct {
		name                                                  string
		attempts, highwater, setups, commits, objects, chunks uint64
		names, pending, files                                 uint64
	}{
		{"fresh-exchange", 1, 4, 1, 4, 38, 2, 88, 91, 94},
		{"retained-with-setup-and-scan-history", 2, 10, 3, 7, 45, 4, 135, 138, 149},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := providerHistoryInitialCatalogUpper(tc.attempts, tc.highwater, tc.setups, tc.commits, tc.objects, tc.chunks)
			if !ok || got.names != tc.names || got.pendingNames != tc.pending || got.files != tc.files {
				t.Fatal("names, unfinished HEAD names and all original allocations were conflated", got, ok)
			}
			if got.files <= got.pendingNames {
				t.Fatal("replacing a HEAD refunded a preceding file allocation")
			}
		})
	}
}

func TestProviderHistoryInitialCatalogRefusesUnsupportedOriginalCounts(t *testing.T) {
	for _, tc := range [][6]uint64{
		{65, 4, 1, 4, 38, 2}, {1, (2 << 20) + 65, 1, 4, 38, 2},
		{1, 4, 66, 4, 38, 2}, {1, 4, 1, (2 << 20) + 1, 38, 2},
		{1, 4, 1, 4, (16 << 20) + 1, 2}, {1, 4, 1, 4, 38, (2 << 20) + 66},
		{^uint64(0), 4, 1, 4, 38, 2},
	} {
		if got, ok := providerHistoryInitialCatalogUpper(tc[0], tc[1], tc[2], tc[3], tc[4], tc[5]); ok || got != (providerHistoryInitialCatalogCost{}) {
			t.Fatal("unsupported original count published a partial catalog requirement", tc, got, ok)
		}
	}
}

func TestProviderHistoryCatalogSupportKeepsOtherInputDomainsSeparate(t *testing.T) {
	for _, n := range []uint64{1, 1 << 20, (1 << 20) + 1, providerHistoryCatalogEntriesMaximum} {
		if !providerHistoryCatalogEntriesSupported(n) {
			t.Fatal("structurally supported catalog count refused", n)
		}
	}
	for _, n := range []uint64{0, providerHistoryCatalogEntriesMaximum + 1, ^uint64(0)} {
		if providerHistoryCatalogEntriesSupported(n) {
			t.Fatal("unsupported catalog count accepted", n)
		}
	}
	if _, ok := providerHistoryReconstructionCatalogUpper((1<<20)+1, 64); !ok {
		t.Fatal("the catalog count still uses the unrelated old one-million ceiling")
	}
	if _, ok := providerHistoryReconstructionCatalogUpper(1, (1<<20)+1); ok {
		t.Fatal("extending catalog support changed the filename-entry overhead limit")
	}
}

func TestProviderHistoryCatalogSourceRetainsTheChosenCapacityAndCompleteFuture(t *testing.T) {
	config := historySelectedSourceConfig(t)
	config.bounds.history.catalogEntries = (1 << 20) + 1
	config.bounds.history.catalogNameBytes = 267387135
	before := *config.bounds.history
	got, ok := providerHistoryCompileSelectedSource(config, historySelectedSourceClock(config))
	if !ok {
		t.Fatal("same selected capacity failed in a catalog source child")
	}
	r := got.catalog
	if r.initial != (providerHistoryInitialCatalogCost{names: 88, pendingNames: 91, files: 94}) ||
		r.futureFiles != got.collection.storage.future.files || r.requiredEntries != 94+r.futureFiles ||
		r.requiredNameBytes != 255*r.requiredEntries || r.selectedEntries != 1048577 || r.selectedNameBytes != 267387135 ||
		r.requiredObjects != 38+got.collection.storage.future.objects || r.requiredCommits != 4+3*got.collection.groups {
		t.Fatal("fresh baseline, whole admitted future or selected capacity changed", r)
	}
	if got.collection.storage.stage.files > providerHistoryCatalogFreshStageFiles ||
		providerHistoryCatalogEntriesMaximum < providerHistoryCatalogFreshCeiling ||
		providerHistoryCatalogEntriesMaximum < providerHistoryCatalogGeneralCeiling {
		t.Fatal("common support omitted an original fresh or retained-prefix generation domain")
	}
	if *config.bounds.history != before || config.bounds.history.derivation != nil || config.bounds.valid() {
		t.Fatal("catalog calculation changed the profile or granted execution")
	}
	// A detached inadequate selection is still useful diagnostic DATA. It
	// must neither disappear nor be replaced by a larger selected profile.
	small, smallOK := providerHistoryFreshCatalogUpper(got.collection.storage, 1)
	if !smallOK || small.selectedEntries != 1 || small.requiredEntries != r.requiredEntries || small.within(config.bounds) {
		t.Fatal("an inadequate diagnostic was zeroed, resized or made executable", small, smallOK)
	}
}

func TestProviderHistoryCatalogCapacityIncludesInitialObjectsAndCommits(t *testing.T) {
	// This is detached capacity DATA. No owner, derivation, native reservation
	// or live store is constructed by this arithmetic boundary scenario.
	storage := providerHistoryUniformStorageCost{groups: 1, future: providerHistoryStorageCharge{objects: 70, files: 91}}
	r, ok := providerHistoryFreshCatalogUpper(storage, 200)
	if !ok || r.requiredEntries != 185 || r.requiredObjects != 108 || r.requiredCommits != 7 {
		t.Fatal("initial and future capacities did not join", r, ok)
	}
	store := providerIntegrationBounds{maximumObjects: 108, maximumCommits: 7,
		history: &providerHistoryProgramBounds{catalogEntries: 200, catalogNameBytes: 51000}}
	if !r.within(store) {
		t.Fatal("exact detached capacity boundary refused")
	}
	for _, name := range []string{"missing-history", "missing-requirement", "empty-initial", "zero-required", "unsupported-selection", "zero-objects", "zero-commits", "objects", "commits", "selected-count", "small-capacity", "names"} {
		t.Run(name, func(t *testing.T) {
			b, requirement := store, r
			h := *store.history
			b.history = &h
			switch name {
			case "missing-history":
				b.history = nil
			case "missing-requirement":
				requirement = providerHistoryCatalogRequirement{}
			case "empty-initial":
				requirement.initial = providerHistoryInitialCatalogCost{}
			case "zero-required":
				requirement.requiredEntries = 0
			case "unsupported-selection":
				requirement.selectedEntries = providerHistoryCatalogEntriesMaximum + 1
			case "zero-objects":
				b.maximumObjects = 0
			case "zero-commits":
				b.maximumCommits = 0
			case "objects":
				b.maximumObjects--
			case "commits":
				b.maximumCommits--
			case "selected-count":
				h.catalogEntries++
			case "small-capacity":
				smaller, smallerOK := providerHistoryFreshCatalogUpper(storage, 184)
				if !smallerOK {
					t.Fatal("inadequate diagnostic disappeared")
				}
				requirement = smaller
				h.catalogEntries, h.catalogNameBytes = 184, 46920
			case "names":
				h.catalogNameBytes--
			}
			if requirement.within(b) {
				t.Fatal("inadequate or mismatched selected capacity accepted", name)
			}
		})
	}
}

func TestProviderHistoryFreshCatalogRefusesInvalidOrOverflowingQuotesEntirely(t *testing.T) {
	for _, name := range []string{"zero-selection", "wide-selection", "zero-groups", "wide-groups", "zero-files", "zero-objects", "file-overflow", "object-overflow"} {
		t.Run(name, func(t *testing.T) {
			storage := providerHistoryUniformStorageCost{groups: 1, future: providerHistoryStorageCharge{objects: 70, files: 91}}
			selected := uint64(200)
			switch name {
			case "zero-selection":
				selected = 0
			case "wide-selection":
				selected = providerHistoryCatalogEntriesMaximum + 1
			case "zero-groups":
				storage.groups = 0
			case "wide-groups":
				storage.groups = (1<<20)/3 + 1
			case "zero-files":
				storage.future.files = 0
			case "zero-objects":
				storage.future.objects = 0
			case "file-overflow":
				storage.future.files = ^uint64(0)
			case "object-overflow":
				storage.future.objects = ^uint64(0)
			}
			if got, ok := providerHistoryFreshCatalogUpper(storage, selected); ok || got != (providerHistoryCatalogRequirement{}) {
				t.Fatal("invalid quote leaked an earlier catalog requirement", name, got, ok)
			}
		})
	}
}
