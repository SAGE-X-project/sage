package registry011

import "testing"

func TestProviderHistoryReconstructionInventoryKeepsIndependentOriginalPageAndPayloadRows(t *testing.T) {
	for _, row := range []struct {
		limits                    providerHistoryLegacyControlLimits
		objects, raw, pairs       uint64
		parsers, payload, control [2]uint64
		normal, incoming          [2]uint64
	}{
		{providerHistoryLegacyControlLimits{4096, 4096, 4, 65536, 1, 1, 1}, 4, 16384, 6,
			[2]uint64{524288, 34360787232}, [2]uint64{32768, 134283640}, [2]uint64{524288, 77311252164},
			[2]uint64{1081344, 111806323036}, [2]uint64{65536, 4294967297}},
		{historyLegacyControlLimits(), 64, 67108864, 2016,
			[2]uint64{524288, 34361552232}, [2]uint64{134217728, 140737756796800}, [2]uint64{524288, 77312031864},
			[2]uint64{135266304, 140849430380896}, [2]uint64{1048576, 1099511627777}},
		{providerHistoryLegacyControlLimits{1 << 20, 32 << 20, 1 << 20, 64 << 20, 256, 64, 1024}, 1024, 67108864, 523776,
			[2]uint64{524288, 34361552232}, [2]uint64{134217728, 4503599895902208}, [2]uint64{524288, 77507617464},
			[2]uint64{135266304, 4503711765071904}, [2]uint64{33554432, 1125899906842625}},
	} {
		before := row.limits
		got, ok := providerHistoryReconstructionInventoryUpper(row.limits)
		if !ok || row.limits != before || got.objects != row.objects || got.raw != row.raw || got.duplicatePairs != row.pairs ||
			got.parsers != (candidateWorkVector{2: row.parsers[0], 10: row.parsers[1]}) ||
			got.payload != (candidateWorkVector{2: row.payload[0], 10: row.payload[1]}) ||
			got.control != (candidateWorkVector{2: row.control[0], 10: row.control[1]}) ||
			got.normal != (candidateWorkVector{2: row.normal[0], 10: row.normal[1]}) ||
			got.incoming != (candidateWorkVector{2: row.incoming[0], 10: row.incoming[1]}) ||
			got.envelope != (candidateWorkVector{2: row.normal[0] + row.incoming[0], 10: row.normal[1] + row.incoming[1]}) {
			t.Fatal("fresh inventories lost independent raw decoding, canonical encoding, payload or slot work", row, got, ok)
		}
	}
}

func TestProviderHistoryReconstructionInventoryRefusesUnselectedOriginalLimits(t *testing.T) {
	for _, mode := range []string{"metadata", "object", "first-objects", "first-raw", "entries-zero", "entries-large", "catalog-overflow"} {
		t.Run(mode, func(t *testing.T) {
			limits := historyLegacyControlLimits()
			switch mode {
			case "metadata":
				limits.metadata = 0
			case "object":
				limits.objectBytes = 0
			case "first-objects":
				limits.firstObjects = 0
			case "first-raw":
				limits.firstRaw = 0
			case "entries-zero":
				limits.entries = 0
			case "entries-large":
				limits.entries = 257
			case "catalog-overflow":
				limits.chunks = ^uint64(0)
			}
			if got, ok := providerHistoryReconstructionInventoryUpper(limits); ok || got != (providerHistoryReconstructionInventoryCost{}) {
				t.Fatal("unselected first-read bounds returned a partial original quote", mode, got, ok)
			}
		})
	}
}
