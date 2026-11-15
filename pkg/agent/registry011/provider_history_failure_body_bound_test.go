package registry011

import "testing"

func TestProviderHistoryFailureBodyMatchesIndependentByteLiterals(t *testing.T) {
	for _, row := range []struct {
		request, wire, maximum, minimum uint64
		certificates                    providerTLSBounds
		built, parsed, closure          uint64
	}{
		{756, 65536, 262144, 81965, providerTLSBounds{1, 1, 8192}, 269194596, 538681932, 1346558460},
		{16384, 16777216, 33554432, 16793645, providerTLSBounds{8, 16, 65536}, 386032117, 788751117, 1963534351},
		{1, 2, 65536, 16431, providerTLSBounds{1, 1, 1}, 268599493, 537297377, 1343194247},
	} {
		got, ok := providerHistoryFailureBodyUpper(row.request, row.wire, row.maximum, row.certificates)
		if !ok || got.minimumObjectBytes != row.minimum ||
			got.built != (candidateWorkVector{10: row.built}) ||
			got.parsed != (candidateWorkVector{2: 16388, 10: row.parsed}) ||
			got.failedClosure != (candidateWorkVector{2: 32776, 10: row.closure}) {
			t.Fatal("independent failure byte-event literal changed", row, got, ok)
		}
	}
}

// Inert small DATA only. These existing fixtures make no network attempt,
// install no owner and obtain no settlement ACK or operating source witness.
func TestProviderHistoryFailureBodyCoversOriginalFiniteDataAndTwoReadbacks(t *testing.T) {
	for _, stage := range []string{"dial", "write", "read", "framing"} {
		ticket, consumption, data, roots, certificates := historyFailureDataFixture(t, stage)
		member := ticket.table.members[0]
		got, ok := providerHistoryFailureBodyUpper(uint64(len(member.request.wire)), uint64(member.capture.WireBytes), providerHistoryOriginalPackBytes, certificates)
		record, recordOK := providerHistoryRecordControlUpper(providerHistoryGroupMembers)
		if !ok || !recordOK {
			t.Fatal("ordinary original failure DATA domain")
		}
		meter := historyGroupCodecMeter()
		original, err := newProviderHistoryFailureOriginalObserved(ticket, consumption, 0, data, roots, certificates, providerHistoryOriginalPackBytes, meter)
		if err != nil || original == nil || meter.failed {
			t.Fatal("finite original failure constructor", stage, err)
		}
		for coordinate, actual := range meter.actual {
			if actual > got.built[coordinate]+record.context[coordinate] {
				t.Fatal("failure builder exceeded body plus its one existing context", stage, coordinate)
			}
		}
		for range 2 {
			parsed, err := parseProviderHistoryFailureOriginalObserved(providerReplayBytes(original.raw, meter), original.digest, ticket, consumption, 0, roots, certificates, providerHistoryOriginalPackBytes, meter)
			if err != nil || parsed == original || parsed == nil || *parsed != *original || meter.failed {
				t.Fatal("independent finite failure DATA readback", stage, err)
			}
		}
		for coordinate, actual := range meter.actual {
			if actual > got.failedClosure[coordinate]+5*record.context[coordinate] {
				t.Fatal("one failed body closure lost its five existing context validations", stage, coordinate)
			}
		}
		if meter.clocks != 0 || meter.actual[0] != 0 || meter.actual[1] != 0 || meter.actual[9] != 0 ||
			data.failureStage != stage || !data.knownFailure || data.uncertain {
			t.Fatal("failure DATA acquired effect/clock work or changed its original prefix", stage)
		}
	}
}

func TestProviderHistoryFailureBodyPreservesPredialSentinelAndWholeRefusal(t *testing.T) {
	certificates := providerTLSBounds{1, 1, 8192}
	// 40 magic bytes + four length bytes +16384 header +65536 wire +1.
	if got, ok := providerHistoryFailureBodyUpper(756, 65536, 81965, certificates); !ok || got.minimumObjectBytes != 81965 {
		t.Fatal("exact pre-dial fixed-header/sentinel boundary")
	}
	for _, row := range []struct {
		request, wire, maximum uint64
		certificates           providerTLSBounds
	}{
		{0, 65536, 262144, certificates}, {16385, 65536, 262144, certificates},
		{756, 756, 262144, certificates}, {756, 0, 262144, certificates},
		{756, 16777217, 33554432, certificates}, {756, 65536, 0, certificates},
		{756, 65536, 33554433, certificates}, {756, 65536, 81964, certificates},
		{756, 65536, 262144, providerTLSBounds{}}, {756, 65536, 262144, providerTLSBounds{9, 1, 1}},
		{756, 65536, 262144, providerTLSBounds{1, 17, 1}}, {756, 65536, 262144, providerTLSBounds{1, 1, 65537}},
		{^uint64(0), 65536, 262144, certificates}, {756, ^uint64(0), 262144, certificates},
		{756, 65536, ^uint64(0), certificates},
	} {
		if got, ok := providerHistoryFailureBodyUpper(row.request, row.wire, row.maximum, row.certificates); ok || got != (providerHistoryFailureBodyCost{}) {
			t.Fatal("invalid original failure cap exposed a partial quote", row, got, ok)
		}
	}
}

func TestProviderHistoryFailureBodyKeepsExistingContextAndClosureOwnership(t *testing.T) {
	for _, row := range []struct{ ordinal, packs, contexts uint64 }{{0, 0, 5}, {31, 3, 8}} {
		calls, ok := providerHistoryCollectionRoleResultControlCallsBound(0, 0, 32, row.ordinal, false)
		if !ok || calls.failedClosure.contexts != row.contexts || calls.failedClosure.contexts-row.packs != 5 {
			t.Fatal("writer plus two rebuilt failure parsers need five contexts; prefix-pack contexts are separate")
		}
	}
	got, ok := providerHistoryFailureBodyUpper(756, 65536, 262144, providerTLSBounds{1, 1, 8192})
	if !ok || got.failedClosure[10] != 1346558460 || got.failedClosure[0] != 0 || got.failedClosure[1] != 0 || got.failedClosure[9] != 0 {
		t.Fatal("one terminal failure body gained a request, wire or writer allowance")
	}
	// The actual failed S is entered inside the failed member invocation but
	// explicitly preflights the SAME original closure. This pure body is priced
	// once there; it is not multiplied by32, not folded into every M and does
	// not remove the existing extra P or refund the full admitted T upper.
}
