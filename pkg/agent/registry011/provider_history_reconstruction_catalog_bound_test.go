package registry011

import "testing"

// Called by the existing owned runtime's successful original C0. It compares
// observations only and never changes the original observer, upper or quota.
// This finite runtime check does not establish native CPU or RAM qualification.
func historyAssertObservedCatalogSourceUpper(t *testing.T, recovery *providerHistoryRecoveryOwner, before candidateWorkVector) {
	t.Helper()
	cost, ok := providerHistoryReconstructionCatalogUpper(recovery.parent.bounds.catalogEntries, recovery.config.bounds.scan.filenameEntryOverhead)
	if !ok || cost.slotBytes > recovery.parent.bounds.resident {
		t.Fatal("original catalog selection cannot fund its declared slot premise")
	}
	for coordinate, actual := range recovery.meter.actual {
		if actual < before[coordinate] || actual-before[coordinate] > cost.observations[coordinate]/2 {
			t.Fatal("original C0 observations exceed their independently compiled source row", coordinate, actual, before[coordinate])
		}
	}
}

func TestProviderHistoryReconstructionCatalogUpperCoversEachOriginalPrimitiveAndEOFGuard(t *testing.T) {
	for _, entries := range []uint64{1, 2, 3, 16, 32} {
		cost, ok := providerHistoryReconstructionCatalogUpper(entries, 64)
		if !ok {
			t.Fatal("finite catalog inputs refused", entries)
		}
		meter := providerReplayUnitMeter()
		meter.upper = cost.envelope
		add := func(kind providerWorkEvent, bytes uint64) {
			t.Helper()
			if !meter.event(kind, bytes) {
				t.Fatal("handwritten original primitive exceeded its catalog envelope", entries, kind, bytes)
			}
		}
		// Handwritten maximum-length primitive sequence only. No filesystem,
		// recovery owner or real catalog is supplied by this arithmetic subject.
		for range 2 {
			add(providerCopyEvent, entries)
			var prior uint64
			for root := range 2 {
				for range 4 {
					add(providerReadEvent, 0)
				}
				var open candidateWorkVector
				open[10] = 4
				if !meter.preflightWork(open) {
					t.Fatal("open/stat/close preflight lost its outstanding room")
				}
				add(providerReadEvent, 0) // open
				add(providerReadEvent, 0) // before Stat
				count := entries / 2
				if root == 1 {
					count = entries - count
				}
				for i := uint64(0); i <= count; i++ {
					var guard candidateWorkVector
					guard[10] = 511*prior + 3*256 + 8
					if !meter.preflightWork(guard) {
						t.Fatal("next-entry or terminal EOF preflight lost its outstanding room", entries, root, i)
					}
					add(providerReadEvent, 0) // ReadDir, also on terminal EOF
					if i == count {
						break
					}
					add(providerReadEvent, 255)
					for j := uint64(0); j < prior; j++ {
						add(providerCompareEvent, 510)
					}
					add(providerReadEvent, 0) // Lstat
					add(providerCopyEvent, 255)
					add(providerCopyEvent, 1)
					prior++
				}
				add(providerReadEvent, 0) // after Stat
				add(providerReadEvent, 0) // close
				for range 4 {
					add(providerReadEvent, 0)
				}
			}
		}
		if meter.actual != cost.observations {
			t.Fatal("catalog observation row differs from independent primitive sequence", entries, meter.actual, cost.observations)
		}
		for range 2 {
			add(providerCompareEvent, 0)
		}
		// N*N is a conservative comparison upper. It does not assert that
		// every matching permutation actually performs that many comparisons.
		for i := uint64(0); i < entries; i++ {
			for j := uint64(0); j < entries; j++ {
				add(providerCompareEvent, 510)
			}
			add(providerCompareEvent, 0)
		}
		for range 2 {
			for range 5 { // identity4 and final Stat1
				add(providerReadEvent, 0)
			}
			add(providerCompareEvent, 0)
		}
		if meter.actual != cost.normal || meter.failed || cost.envelope[10]-meter.actual[10] != cost.incoming[10] || cost.slotBytes != 4*entries*(64+255) {
			t.Fatal("catalog closure omitted primitives, summed every guard or changed the slot premise", entries)
		}
	}
}

func TestProviderHistoryReconstructionCatalogUpperRetainsEOFRoomAndSeparateDomains(t *testing.T) {
	for _, tc := range []struct {
		entries, normal, incoming, envelope, slots uint64
	}{
		{1, 1614, 1287, 2901, 1276},
		{2, 5204, 1798, 7002, 2552},
	} {
		got, ok := providerHistoryReconstructionCatalogUpper(tc.entries, 64)
		if !ok || got.normal != (candidateWorkVector{10: tc.normal}) || got.incoming != (candidateWorkVector{10: tc.incoming}) ||
			got.envelope != (candidateWorkVector{10: tc.envelope}) || got.slotBytes != tc.slots {
			t.Fatal("independent small-domain literals or read-only coordinates changed", tc.entries, ok, got)
		}
	}
	maximum, ok := providerHistoryReconstructionCatalogUpper(1<<20, 1<<20)
	if !ok || maximum.normal[10] == 0 || maximum.incoming[10] != 511*(1<<20)+776 || maximum.slotBytes != 4*(1<<20)*((1<<20)+255) ||
		!providerHistoryReadWorkDomain(maximum.envelope) {
		t.Fatal("maximum declared domain overflowed or acquired a forbidden role", ok)
	}
}

func TestProviderHistoryReconstructionCatalogUpperRefusesUnselectedLimitsWithoutPartialRows(t *testing.T) {
	for _, pair := range [][2]uint64{{0, 1}, {1, 0}, {1<<20 + 1, 64}, {1, 1<<20 + 1}, {^uint64(0), 1}, {1, ^uint64(0)}} {
		if got, ok := providerHistoryReconstructionCatalogUpper(pair[0], pair[1]); ok || got != (providerHistoryReconstructionCatalogCost{}) {
			t.Fatal("unselected catalog input leaked a partial cost row", pair, ok)
		}
	}
}

func TestProviderHistoryReconstructionNamespaceUpperKeepsBothExpectedCatalogsAndReceiptHashes(t *testing.T) {
	for _, tc := range []struct {
		entries, expectedNames, receipts, matches, normal, incoming, envelope uint64
	}{
		{1, 2054, 1280, 1022, 5970, 1287, 7257},
		{2, 5128, 1280, 4088, 15700, 1798, 17498},
	} {
		got, ok := providerHistoryReconstructionNamespaceUpper(tc.entries, 64, 64, 2)
		if !ok || got.expectedNames != (candidateWorkVector{10: tc.expectedNames}) || got.expectedReceipts != (candidateWorkVector{10: tc.receipts}) ||
			got.expectedMatches != (candidateWorkVector{10: tc.matches}) || got.normal != (candidateWorkVector{10: tc.normal}) ||
			got.incoming != (candidateWorkVector{10: tc.incoming}) || got.envelope != (candidateWorkVector{10: tc.envelope}) {
			t.Fatal("namespace source rows differ from independent literal primitive counts", tc.entries, ok, got)
		}
	}
	base, ok := providerHistoryReconstructionNamespaceUpper(16, 64, 64, 2)
	more, moreOK := providerHistoryReconstructionNamespaceUpper(16, 64, 64, 3)
	if !ok || !moreOK || more.catalog != base.catalog || more.expectedNames != base.expectedNames || more.expectedMatches != base.expectedMatches ||
		more.expectedReceipts[10]-base.expectedReceipts[10] != 640 || more.normal[10]-base.normal[10] != 640 || more.incoming != base.incoming {
		t.Fatal("another receipt changed independent catalog roles, reused a digest or doubled headroom")
	}
}

func TestProviderHistoryReconstructionNamespaceUpperRefusesLateOverflowWithoutCatalogPrefix(t *testing.T) {
	for _, input := range [][4]uint64{{0, 64, 64, 2}, {1, 0, 64, 2}, {1, 64, 0, 2}, {1, 64, 1<<20 + 1, 2}, {1, 64, 64, 0}, {1, 64, 64, ^uint64(0)}} {
		if got, ok := providerHistoryReconstructionNamespaceUpper(input[0], input[1], input[2], input[3]); ok || got != (providerHistoryReconstructionNamespaceCost{}) {
			t.Fatal("invalid or overflowing namespace quote published an earlier catalog", input, ok)
		}
	}
	if got, ok := providerHistoryReconstructionCatalogUpper(1, 64); !ok || got == (providerHistoryReconstructionCatalogCost{}) {
		t.Fatal("late-overflow case did not have a positive observed-catalog precursor")
	}
}
