package registry011

import "testing"

func TestProviderHistoryFailedSettlementExtraMatchesIndependentSourceLiterals(t *testing.T) {
	for _, row := range []struct {
		request, wire, maximum, metadata, overhead uint64
		certificates                               providerTLSBounds
		outer, io, directRaw, logical              uint64
		storage                                    providerHistoryStorageCharge
	}{
		{756, 65536, 262144, 8192, 4096, providerTLSBounds{1, 1, 8192}, 2229654, 245910, 81974, 1349115998,
			providerHistoryStorageCharge{86061, 81965, 65537, 16428, 1, 1}},
		{16384, 16777216, 33554432, 1048576, 4096, providerTLSBounds{8, 16, 65536}, 134350230, 50380950, 16793654, 2165059185,
			providerHistoryStorageCharge{16797741, 16793645, 16777217, 16428, 1, 1}},
		{1, 2, 65536, 1, 1, providerTLSBounds{1, 1, 1}, 525734, 49308, 16440, 1343785729,
			providerHistoryStorageCharge{16432, 16431, 3, 16428, 1, 1}},
	} {
		got, ok := providerHistoryFailedSettlementExtraUpper(row.request, row.wire, row.maximum, row.metadata, row.overhead, row.certificates)
		if !ok || got.outer != (candidateWorkVector{10: row.outer}) || got.reads[10]+got.writes[10] != row.io ||
			got.directRaw != (candidateWorkVector{10: row.directRaw}) || got.storage != row.storage || got.extra != (providerHistoryStageWorkCost{work: candidateWorkVector{2: 32776, 9: row.storage.raw, 10: row.logical}}) {
			t.Fatal("independent failed-S body/outer/I/O/allocation literal changed", row, got, ok)
		}
	}
}

func TestProviderHistoryFailedSettlementExtraRetainsInventoryAggregateCap(t *testing.T) {
	certificates := providerTLSBounds{1, 1, 8192}
	// At P=32MiB the existing two-object inventory already reserves its
	// whole64MiB aggregate. A third object adds its distinct rebuild/hash
	// events, but cannot raise that original aggregate or renew a read scope.
	got, ok := providerHistoryFailedSettlementExtraUpper(756, 65536, 32<<20, 8192, 4096, certificates)
	if !ok || got.outer[10] != 656790 || got.storage.objects != 1 || got.storage.files != 1 {
		t.Fatal("third inventory entry duplicated an aggregate or lost its object events")
	}
	if got.reads != (candidateWorkVector{10: 163932}) || got.writes != (candidateWorkVector{9: 81965, 10: 81978}) || got.directRaw != (candidateWorkVector{10: 81974}) || got.extra.clocks != 0 {
		t.Fatal("extra failure file duplicated the existing S rename/clock or body-parser caller copy")
	}
}

func TestProviderHistoryFailedSettlementExtraRejectsPartialAndLateAllocationOverflow(t *testing.T) {
	certificates := providerTLSBounds{1, 1, 8192}
	for _, row := range []struct {
		wire, maximum, metadata, overhead uint64
		certificates                      providerTLSBounds
	}{
		{65536, 262144, 8192, 0, certificates}, {65536, 262144, 0, 4096, certificates},
		{65536, 262144, 1048577, 4096, certificates}, {65536, 81964, 8192, 4096, certificates},
		{756, 262144, 8192, 4096, certificates}, {16777217, 33554432, 8192, 4096, certificates},
		{65536, 262144, 8192, 4096, providerTLSBounds{}},
		{65536, 262144, 8192, ^uint64(0), certificates},
		{65536, 262144, 8192, ^uint64(0) - 81964, certificates},
	} {
		if got, ok := providerHistoryFailedSettlementExtraUpper(756, row.wire, row.maximum, row.metadata, row.overhead, row.certificates); ok || got != (providerHistoryFailedSettlementExtraCost{}) {
			t.Fatal("failed child or late physical overflow exposed partial failure additions", row, got, ok)
		}
	}
	// Scalar arithmetic endpoint only, never host qualification/allocation.
	if got, ok := providerHistoryFailedSettlementExtraUpper(756, 65536, 262144, 8192, ^uint64(0)-81965, certificates); !ok || got.storage.physical != ^uint64(0) {
		t.Fatal("checked per-file physical boundary changed")
	}
}

func TestProviderHistoryFailedSettlementExtraPreservesFiniteFailureLayout(t *testing.T) {
	for _, stage := range []string{"dial", "write", "read", "framing"} {
		ticket, consumption, data, roots, certificates := historyFailureDataFixture(t, stage)
		member := ticket.table.members[0]
		original, err := newProviderHistoryFailureOriginalObserved(ticket, consumption, 0, data, roots, certificates, providerHistoryOriginalPackBytes, nil)
		if err != nil || original == nil {
			t.Fatal("inert original failure DATA", stage, err)
		}
		got, ok := providerHistoryFailedSettlementExtraUpper(uint64(len(member.request.wire)), uint64(member.capture.WireBytes), providerHistoryOriginalPackBytes, 8192, 4096, certificates)
		metadata, payload, layoutOK := providerHistoryFailureLayout(original.raw)
		if !ok || !layoutOK || metadata > got.storage.metadata || payload > got.storage.payload ||
			uint64(len(original.raw)) > got.storage.raw || metadata+payload != uint64(len(original.raw)) ||
			payload != uint64(data.requestWritten+data.responseRead) || got.storage.raw != got.storage.metadata+got.storage.payload {
			t.Fatal("failed-S addition lost exact layout classes or rejected extra-byte funding", stage)
		}
		// Two actual direct-raw invocations, distinct from file I/O, body
		// parser input and inventory/stored-object reconstruction.
		meter := historyGroupCodecMeter()
		copy := providerReplayBytes(original.raw, meter)
		stageCopy := providerHistoryRecoveryCloneObserved(stage, meter)
		want := uint64(len(original.raw) + len(stage) + 2)
		if len(copy) != len(original.raw) || stageCopy != stage || meter.failed || meter.clocks != 0 ||
			meter.actual != (candidateWorkVector{10: want}) || want > got.directRaw[10] {
			t.Fatal("writer input and failed-stage copies escaped explicit original byte costs", stage)
		}
	}
	// No live owner/capture/ACK, network or filesystem effect is performed.
}
