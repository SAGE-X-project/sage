package registry011

import "testing"

func TestProviderHistoryPointwiseLedgerControlMatchesIndependentRows(t *testing.T) {
	for _, tc := range []struct{ calls, decoded, logical uint64 }{
		{1, 1048576, 38959373275}, {2, 1112576, 39413950033},
		{3, 1176576, 39868526791}, {20, 2264576, 47596331677},
		{102, 7512576, 84871625833}, {1000, 64984576, 493081554517},
	} {
		got, ok := providerHistoryCollectionPointwiseLedgerControlUpper(tc.calls)
		want := candidateWorkVector{2: tc.decoded, 10: tc.logical}
		if !ok || got != want {
			t.Fatal("pointwise ledger component differs from independent complete row", tc.calls, got, want)
		}
	}
	// Keep the existing repeated-envelope function and its original generic
	// child intact. Neither numeric quote is a source or reservation witness.
	calls, a := providerHistoryCollectionRootCallsBound(2, 3)
	old, b := providerHistoryCollectionRepeatedLedgerControlUpper(2, 3)
	pointwise, c := providerHistoryCollectionPointwiseLedgerControlUpper(calls.ledgerRoots)
	if !a || !b || !c || old != (candidateWorkVector{2: calls.ledgerRoots * 1048576, 10: calls.ledgerRoots * 38959373275}) ||
		pointwise[2] >= old[2] || pointwise[10] >= old[10] {
		t.Fatal("original wide repeated quote was changed or conflated with actual component", old, pointwise)
	}
}

func TestProviderHistoryPointwiseLedgerControlFundsOriginalRecursiveData(t *testing.T) {
	const repetitions = 3
	upper, ok := providerHistoryCollectionPointwiseLedgerControlUpper(repetitions)
	parts, partsOK := providerHistoryPlannedLedgerControlUpper(providerHistoryGroupMembers)
	if !ok || !partsOK {
		t.Fatal("finite original control precursor")
	}
	// Pure owned ledger DATA with no collection, original reader or I/O.
	// Each check creates a separate inert meter with its upper selected first;
	// no existing meter or entered allowance is changed or renewed.
	for _, count := range []int{1, 32} {
		maximum := historyLedgerMaximum()
		maximum.work[0] = uint64(2 * count)
		ledger := historyLedgerFixture(t, maximum)
		check := func() {
			t.Helper()
			before := *ledger
			meter := providerHistoryContentUnitMeter(upper)
			for range repetitions {
				if !ledger.validObserved(meter) || meter.failed || *ledger != before {
					t.Fatal("original repeated validation refused or changed ledger DATA", count, ledger.phase)
				}
			}
			if meter.actual[10] == 0 || meter.upper != upper {
				t.Fatal("original validation was not observed under its immutable allowance")
			}
			for coordinate, units := range meter.actual {
				if units > repetitions*parts.normal.ledger[coordinate] {
					t.Fatal("original repeated validation escaped planned normal work", ledger.phase, coordinate, units)
				}
			}
		}
		check()
		ticket := historyRecordTicket(t, count, 1, ledger.head)
		if ledger.admit(ticket) != nil {
			t.Fatal("original admitted DATA precursor")
		}
		check()
		consumption, err := newProviderHistoryGroupConsumption(ticket, ledger.head, admissionTestDigest("1"))
		if err != nil || ledger.consume(consumption) != nil {
			t.Fatal("original consumed DATA precursor", err)
		}
		check()
		results := make([]string, count)
		for i := range results {
			results[i] = admissionTestDigest("2")
		}
		settlement, err := newProviderHistoryGroupSettlement(ticket, consumption, ledger.head, admissionTestDigest("3"), "complete", "", results)
		if err != nil || ledger.settle(settlement) != nil {
			t.Fatal("original settled DATA precursor", err)
		}
		check()
	}
}

func TestProviderHistoryPointwiseLedgerControlKeepsOriginalCounterRefusal(t *testing.T) {
	const calls = 3
	upper, ok := providerHistoryCollectionPointwiseLedgerControlUpper(calls)
	parts, partsOK := providerHistoryPlannedLedgerControlUpper(providerHistoryGroupMembers)
	if !ok || !partsOK {
		t.Fatal("finite counter precursor")
	}
	// Complete envelopes below are symbolic cost DATA charged with the real
	// inert meter operations; no ledger failure, native work or I/O is run.
	for stop := range calls {
		meter := providerHistoryContentUnitMeter(upper)
		for index := 0; index <= stop; index++ {
			before := meter.actual
			if !meter.preflightWork(parts.envelope.ledger) || meter.actual != before {
				t.Fatal("original incoming preflight changed entered work", stop, index)
			}
			work := parts.normal.ledger
			if index == stop {
				work = parts.envelope.ledger
			}
			if !meter.add(work) || meter.failed {
				t.Fatal("whole incoming closure did not fit its original prefix", stop, index)
			}
		}
	}
	shortUpper := upper
	shortUpper[10]-- // A separate subject, chosen before any observation.
	short := providerHistoryContentUnitMeter(shortUpper)
	for range calls - 1 {
		if !short.preflightWork(parts.envelope.ledger) || !short.add(parts.normal.ledger) {
			t.Fatal("short subject failed before the last incoming boundary")
		}
	}
	prefix := short.actual
	if prefix != (candidateWorkVector{2: 128000, 10: 909153516}) ||
		short.preflightWork(parts.envelope.ledger) || !short.failed || short.actual != prefix || short.upper != shortUpper ||
		short.add(candidateWorkVector{}) || short.actual != prefix {
		t.Fatal("last incoming shortage revived or partially charged the original counter", short.actual)
	}
}

func TestProviderHistoryPointwiseLedgerControlClearsLateOverflow(t *testing.T) {
	// Independent literal boundary. Its earlier decoded coordinate is finite
	// and positive; the next call overflows only the later logical coordinate.
	const last uint64 = 40580042227
	got, ok := providerHistoryCollectionPointwiseLedgerControlUpper(last)
	if !ok || got[2] == 0 || got[10] <= ^uint64(0)-454576758 ||
		got[2] != (last-1)*64000+1048576 || got[10] != (last-1)*454576758+38959373275 {
		t.Fatal("independent largest finite ledger quote boundary", got, ok)
	}
	for _, calls := range []uint64{0, last + 1, ^uint64(0)} {
		if work, ok := providerHistoryCollectionPointwiseLedgerControlUpper(calls); ok || work != (candidateWorkVector{}) {
			t.Fatal("invalid or overflowing ledger quote returned partial work", calls, work, ok)
		}
	}
}
