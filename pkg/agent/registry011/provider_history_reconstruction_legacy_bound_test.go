package registry011

import "testing"

func TestProviderHistoryReconstructionLegacyKeepsBothOriginalControlPrefixes(t *testing.T) {
	// Literal event-ledger totals, independently enumerated from the original
	// setup/highwater/four-record graph. These are not executed read results.
	for _, row := range []struct {
		limits             providerHistoryLegacyControlLimits
		recoveries         uint64
		counts             [2]uint64
		profiles           uint64
		highwaters, setups [2]uint64
		anchor             uint64
		main               [2]uint64
		lookups, control   uint64
		normal, incoming   [2]uint64
	}{
		{providerHistoryLegacyControlLimits{4096, 4096, 4, 65536, 1, 1, 1}, 0, [2]uint64{34, 2},
			570565730, [2]uint64{40960, 1375961414}, [2]uint64{16384, 235005792}, 8623792538,
			[2]uint64{786432, 8867240150768}, 67774, 3026, [2]uint64{843776, 8878045547042}, [2]uint64{65536, 1099511627777}},
		{historyLegacyControlLimits(), 3, [2]uint64{58, 14},
			249111906170, [2]uint64{131072, 3087385142}, [2]uint64{1835008, 181335241422}, 17180787098,
			[2]uint64{4718592, 9209504969528}, 14846734, 3056, [2]uint64{6684672, 9660235139150}, [2]uint64{65536, 1099511627777}},
		{providerHistoryLegacyControlLimits{1 << 20, 32 << 20, 1 << 20, 64 << 20, 256, 64, 1024}, 64, [2]uint64{546, 258},
			600333921306210, [2]uint64{1130496, 27652232518}, [2]uint64{541065216, 851041346364896}, 2207623938458,
			[2]uint64{36175872, 61779999044408}, 472428558, 3666, [2]uint64{578371584, 1515391015318714}, [2]uint64{1048576, 1099511627777}},
	} {
		before := row.limits
		got, ok := providerHistoryReconstructionLegacyUpper(row.limits, row.recoveries)
		if !ok || row.limits != before || [2]uint64{got.profileCalls, got.setupPairs} != row.counts ||
			got.profiles != (candidateWorkVector{10: row.profiles}) ||
			got.highwaters != (candidateWorkVector{2: row.highwaters[0], 10: row.highwaters[1]}) ||
			got.setups != (candidateWorkVector{2: row.setups[0], 10: row.setups[1]}) ||
			got.anchor != (candidateWorkVector{10: row.anchor}) ||
			got.main != (candidateWorkVector{2: row.main[0], 10: row.main[1]}) ||
			got.lookups != (candidateWorkVector{10: row.lookups}) || got.control != (candidateWorkVector{10: row.control}) ||
			got.normal != (candidateWorkVector{2: row.normal[0], 10: row.normal[1]}) ||
			got.incoming != (candidateWorkVector{2: row.incoming[0], 10: row.incoming[1]}) ||
			got.envelope != (candidateWorkVector{2: row.normal[0] + row.incoming[0], 10: row.normal[1] + row.incoming[1]}) {
			t.Fatal("original controls lost fresh codecs, profile rebuilds, nested parsers or guards", row, got, ok)
		}
	}
}

func TestProviderHistoryReconstructionLegacyRefusesIncompleteAndOverflowedLimits(t *testing.T) {
	for _, mode := range []string{"metadata-zero", "metadata-large", "object-zero", "object-large", "first-objects-zero", "first-objects-large", "first-raw-zero", "first-raw-large", "entries-zero", "entries-large", "attempts-zero", "attempts-large", "chunks-zero", "chunks-overflow", "recoveries-large", "recoveries-overflow"} {
		t.Run(mode, func(t *testing.T) {
			limits, recoveries := historyLegacyControlLimits(), uint64(3)
			switch mode {
			case "metadata-zero":
				limits.metadata = 0
			case "metadata-large":
				limits.metadata = 1<<20 + 1
			case "object-zero":
				limits.objectBytes = 0
			case "object-large":
				limits.objectBytes = 32<<20 + 1
			case "first-objects-zero":
				limits.firstObjects = 0
			case "first-objects-large":
				limits.firstObjects = 1<<20 + 1
			case "first-raw-zero":
				limits.firstRaw = 0
			case "first-raw-large":
				limits.firstRaw = 64<<20 + 1
			case "entries-zero":
				limits.entries = 0
			case "entries-large":
				limits.entries = 257
			case "attempts-zero":
				limits.attempts = 0
			case "attempts-large":
				limits.attempts = 65
			case "chunks-zero":
				limits.chunks = 0
			case "chunks-overflow":
				limits.chunks = ^uint64(0)
			case "recoveries-large":
				recoveries = 65
			case "recoveries-overflow":
				recoveries = ^uint64(0)
			}
			if got, ok := providerHistoryReconstructionLegacyUpper(limits, recoveries); ok || got != (providerHistoryReconstructionLegacyCost{}) {
				t.Fatal("invalid original selection returned a partial control row", mode, got, ok)
			}
		})
	}
}

func TestProviderHistoryReconstructionLegacyKeepsOneOutstandingGuardForBothPasses(t *testing.T) {
	limits := historyLegacyControlLimits()
	zero, a := providerHistoryReconstructionLegacyUpper(limits, 0)
	one, b := providerHistoryReconstructionLegacyUpper(limits, 1)
	maximum, c := providerHistoryReconstructionLegacyUpper(limits, 64)
	if !a || !b || !c || zero.incoming != one.incoming || one.incoming != maximum.incoming ||
		zero.incoming != (candidateWorkVector{2: 65536, 10: 1099511627777}) {
		t.Fatal("a fresh pass or setup recovery renewed the unchanged outstanding guard")
	}
	for coordinate := range maximum.normal {
		if maximum.normal[coordinate] != zero.normal[coordinate]+64*(one.normal[coordinate]-zero.normal[coordinate]) ||
			maximum.envelope[coordinate] != maximum.normal[coordinate]+maximum.incoming[coordinate] {
			t.Fatal("retained setup prefixes did not accumulate separately from the single refusal guard", coordinate)
		}
	}
}
