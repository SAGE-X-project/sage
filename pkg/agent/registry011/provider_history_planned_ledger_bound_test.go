package registry011

import (
	"strings"
	"testing"
	"time"
)

func TestProviderHistoryPlannedLedgerPartsMatchIndependentCompleteRows(t *testing.T) {
	for _, tc := range []struct{ members, ticket, consumption, ledger, table uint64 }{
		{1, 4218215, 4475573, 13849262, 605}, {2, 6034723, 6292081, 17482278, 1025},
		{3, 8204031, 8461389, 21820894, 1445}, {32, 224581963, 224839321, 454576758, 13625},
	} {
		old, a := providerHistoryLedgerControlUpper(tc.members)
		got, b := providerHistoryPlannedLedgerControlUpper(tc.members)
		want := providerHistoryPlannedLedgerControlCost{
			normal: providerHistoryLedgerControlCost{
				requestCopy: candidateWorkVector{2: 500, 10: 875348}, requestClass: candidateWorkVector{2: 500, 10: 250502},
				ticket: candidateWorkVector{2: 1000 * tc.members, 10: tc.ticket}, consumption: candidateWorkVector{2: 1000 * tc.members, 10: tc.consumption},
				ledger: candidateWorkVector{2: 2000 * tc.members, 10: tc.ledger},
			}, envelope: old, tableBytes: tc.table,
		}
		if !a || !b || got != want {
			t.Fatal("planned request/recursive record parts differ from independent event rows", tc, got, want)
		}
	}
	for _, n := range []uint64{0, 33, ^uint64(0)} {
		if got, ok := providerHistoryPlannedLedgerControlUpper(n); ok || got != (providerHistoryPlannedLedgerControlCost{}) {
			t.Fatal("invalid planned member count returned a partial row", n, got, ok)
		}
	}
}

func TestProviderHistoryPlannedLedgerPartsFundOriginalRequestTemplates(t *testing.T) {
	parts, ok := providerHistoryPlannedLedgerControlUpper(1)
	if !ok {
		t.Fatal("planned one-request parts")
	}
	check := func(method, class string, params any) {
		t.Helper()
		request := historyGroupRequest(t, method, params)
		before := *request
		if len(request.params) > 500 || len(request.body) > 578 || len(request.head) > 178 || len(request.wire) > 756 {
			t.Fatal("original fixed request template escaped its byte extent", method)
		}
		meter := providerHistoryContentUnitMeter(parts.envelope.requestCopy)
		copy, err := providerHistoryGroupRequestCopyObserved(request, meter)
		if err != nil || copy != before || *request != before || meter.failed {
			t.Fatal("original request reconstruction changed its bytes", method, err)
		}
		classMeter := providerHistoryContentUnitMeter(parts.envelope.requestClass)
		got, err := providerHistoryGroupRequestClassObserved(request, classMeter)
		if err != nil || got != class || classMeter.failed || *request != before {
			t.Fatal("original request classification changed", method, got, err)
		}
		for c, units := range meter.actual {
			if units > parts.normal.requestCopy[c] || classMeter.actual[c] > parts.normal.requestClass[c] {
				t.Fatal("original planned template exceeded entered-only row", method, c, units)
			}
		}
	}
	hash := admissionTestDigest("a")
	block := map[string]any{"blockHash": hash, "requireCanonical": true}
	check("eth_chainId", "scalar", []any{})
	check("web3_clientVersion", "scalar", []any{})
	for _, tag := range []string{"0x0", "finalized", "0xffffffffffffffff"} {
		check("eth_getBlockByNumber", "header", []any{tag, false})
	}
	check("eth_getBlockByHash", "block", []any{hash, true})
	check("eth_getCode", "code", []any{SepoliaRegistryAddress, block})
	check("eth_call", "state", []any{map[string]string{"to": SepoliaRegistryAddress, "data": "0x04ef898d"}, block})
	check("debug_getRawBlock", "block", []any{hash})
	check("debug_getRawReceipts", "receipts", []any{hash})
	check("eth_getBlockReceipts", "receipts", []any{hash})
	for _, n := range []int{1, 32, 33, 64} {
		for _, method := range []string{"record", "activeGrants", "historyItem"} {
			version := uint64(0)
			if method == "historyItem" {
				version = ^uint64(0)
			}
			params, err := providerHistoryRegistryParamsObserved(SepoliaRegistryAddress, strings.Repeat("a", n), method, version, hash, providerReplayUnitMeter())
			if err != nil {
				t.Fatal("original finite calldata precursor", method, n, err)
			}
			check("eth_call", "state", params)
		}
	}
	for _, timeout := range []time.Duration{5*time.Minute - 1, time.Microsecond, 1500 * time.Nanosecond} {
		text := timeout.String()
		if len(text) > 32 || timeout < time.Millisecond && (!strings.Contains(text, "µs") || !strings.ContainsRune(text, '\u00b5')) {
			t.Fatal("original duration byte/UTF-8 precursor", text)
		}
		for _, config := range []map[string]bool{{"onlyTopCall": false, "withLog": true}, {"onlyTopCall": true, "withLog": false, "includePrecompiles": true}} {
			options := map[string]any{"tracer": "callTracer", "timeout": text, "reexec": 0, "tracerConfig": config}
			check("debug_traceTransaction", "execution", []any{hash, options})
			check("debug_traceBlockByHash", "execution", []any{hash, options})
		}
		check("debug_traceBlockByHash", "gas", []any{hash, map[string]any{"tracer": "stateGasTracer", "timeout": text, "reexec": 0}})
	}
}

func TestProviderHistoryPlannedLedgerPartsPreserveOriginalRecursiveValidation(t *testing.T) {
	// Finite closed request/record DATA only. The ledger has no cursor or
	// Source and grants no original-reader/dispatch authority to these parts.
	for _, count := range []int{1, 32} {
		parts, ok := providerHistoryPlannedLedgerControlUpper(uint64(count))
		if !ok {
			t.Fatal("prospective record precursor")
		}
		maximum := historyLedgerMaximum()
		maximum.work[0] = uint64(2 * count)
		ledger := historyLedgerFixture(t, maximum)
		checkLedger := func() {
			t.Helper()
			before := *ledger
			meter := providerHistoryContentUnitMeter(parts.envelope.ledger)
			if !ledger.validObserved(meter) || meter.failed || *ledger != before {
				t.Fatal("original ledger validation changed DATA", count, ledger.phase)
			}
			for c, units := range meter.actual {
				if units > parts.normal.ledger[c] {
					t.Fatal("original ledger control escaped planned entered work", count, ledger.phase, c, units)
				}
			}
		}
		checkLedger()
		ticket := historyRecordTicket(t, count, 1, ledger.head)
		beforeTicket := *ticket
		meter := providerHistoryContentUnitMeter(parts.envelope.ticket)
		if !providerHistoryGroupTicketValidObserved(ticket, meter) || meter.failed || *ticket != beforeTicket || uint64(len(ticket.table.raw)) > parts.tableBytes || len(ticket.raw) > 1647 {
			t.Fatal("original ticket/table exceeded fixed-shape parts", count)
		}
		for c, units := range meter.actual {
			if units > parts.normal.ticket[c] {
				t.Fatal("ticket's complete recursive work escaped normal row", count, c, units)
			}
		}
		if ledger.admit(ticket) != nil {
			t.Fatal("original admission precursor")
		}
		checkLedger()
		consumption, err := newProviderHistoryGroupConsumption(ticket, ledger.head, admissionTestDigest("1"))
		if err != nil {
			t.Fatal("original consumption precursor", err)
		}
		beforeConsumption := *consumption
		meter = providerHistoryContentUnitMeter(parts.envelope.consumption)
		if !providerHistoryGroupConsumptionValidObserved(consumption, ticket, meter) || meter.failed || *ticket != beforeTicket || *consumption != beforeConsumption || len(consumption.raw) > 505 {
			t.Fatal("original consumption changed recursive DATA", count)
		}
		for c, units := range meter.actual {
			if units > parts.normal.consumption[c] {
				t.Fatal("consumption's ticket work escaped normal row", count, c, units)
			}
		}
		if ledger.consume(consumption) != nil {
			t.Fatal("original consumed-ledger precursor")
		}
		checkLedger()
		results := make([]string, count)
		for i := range results {
			results[i] = admissionTestDigest("2")
		}
		settlement, err := newProviderHistoryGroupSettlement(ticket, consumption, ledger.head, admissionTestDigest("3"), "complete", "", results)
		if err != nil || ledger.settle(settlement) != nil {
			t.Fatal("original settled-ledger precursor", err)
		}
		checkLedger()
	}
}

func TestProviderHistoryPlannedLedgerPartsKeepWideGuardsAndStickyRefusals(t *testing.T) {
	parts, ok := providerHistoryPlannedLedgerControlUpper(1)
	if !ok {
		t.Fatal("prospective planned request parts")
	}
	request := historyGroupSimple(t)
	before := *request
	short := providerHistoryContentUnitMeter(parts.normal.requestCopy)
	if _, err := providerHistoryGroupRequestCopyObserved(request, short); err == nil || !short.failed || short.actual[10] == 0 || *request != before {
		t.Fatal("entered-only DATA funded the original wide encoder guard")
	}
	prefix := short.actual
	if _, err := providerHistoryGroupRequestCopyObserved(request, short); err == nil || !short.failed || short.actual != prefix || *request != before {
		t.Fatal("refused observer renewed work or changed request DATA")
	}
	complete := providerHistoryContentUnitMeter(parts.envelope.requestCopy)
	if copy, err := providerHistoryGroupRequestCopyObserved(request, complete); err != nil || copy != before || complete.failed {
		t.Fatal("original complete one-call envelope changed")
	}
	// A bounded generic request is deliberately OUTSIDE the closed collection
	// grammar. No endpoint/transport is invoked. Its wider input must retain
	// the old behavior and must never be certified by the planned normal row.
	wide := historyGroupRequest(t, "eth_getCode", []any{strings.Repeat("a", 1024), "0x1"})
	wideBefore := *wide
	meter := providerHistoryContentUnitMeter(parts.envelope.requestCopy)
	if copy, err := providerHistoryGroupRequestCopyObserved(wide, meter); err != nil || copy != wideBefore || *wide != wideBefore || meter.failed ||
		meter.actual[2] <= parts.normal.requestCopy[2] || meter.actual[10] <= parts.normal.requestCopy[10] {
		t.Fatal("generic request was narrowed to the planned DATA grammar", meter.actual)
	}
}
