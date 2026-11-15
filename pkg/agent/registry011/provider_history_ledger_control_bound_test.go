package registry011

import (
	"context"
	"testing"
)

func TestProviderHistoryLedgerControlCostsMatchIndependentRows(t *testing.T) {
	for _, tc := range []struct {
		members, ticket2, ticket10, consumption10, ledger2, ledger10 uint64
	}{
		{1, 16384, 4832689716, 4899831661, 32768, 9799638737},
		{2, 32768, 5303008015, 5370149960, 65536, 10740275335},
		{32, 524288, 19412556985, 19479698930, 1048576, 38959373275},
	} {
		got, ok := providerHistoryLedgerControlUpper(tc.members)
		var request, class, ticket, consumption, ledger candidateWorkVector
		request[2], request[10] = 8192, 403117431
		class[2], class[10] = 8192, 67117058
		ticket[2], ticket[10] = tc.ticket2, tc.ticket10
		consumption[2], consumption[10] = tc.ticket2, tc.consumption10
		ledger[2], ledger[10] = tc.ledger2, tc.ledger10
		if !ok || got != (providerHistoryLedgerControlCost{request, class, ticket, consumption, ledger}) {
			t.Fatal("request/recursive record/ledger rows differ from independent constants", tc, got, ok)
		}
	}
}

func TestProviderHistoryLedgerControlCostsFundEveryClosedRequestClass(t *testing.T) {
	upper, ok := providerHistoryLedgerControlUpper(1)
	if !ok {
		t.Fatal("prospective original request quote")
	}
	h := admissionTestDigest("a")
	for _, tc := range []struct {
		method, class string
		params        any
	}{
		{"eth_chainId", "scalar", []any{}},
		{"web3_clientVersion", "scalar", []any{}},
		{"eth_getCode", "code", []any{SepoliaRegistryAddress, "0x1"}},
		{"eth_call", "state", []any{map[string]string{"to": SepoliaRegistryAddress, "data": "0x"}, "0x1"}},
		{"debug_getRawBlock", "block", []any{h}},
		{"eth_getBlockReceipts", "receipts", []any{h}},
		{"debug_getRawReceipts", "receipts", []any{h}},
		{"eth_getBlockByHash", "header", []any{h, false}},
		{"eth_getBlockByNumber", "block", []any{"finalized", true}},
		{"debug_traceTransaction", "gas", []any{h, map[string]any{"tracer": "stateGasTracer", "reexec": 0}}},
		{"debug_traceBlockByHash", "execution", []any{h, map[string]string{"tracer": "callTracer"}}},
	} {
		r := historyGroupRequest(t, tc.method, tc.params)
		before := *r
		meter := providerHistoryContentUnitMeter(upper.requestCopy)
		copy, err := providerHistoryGroupRequestCopyObserved(r, meter)
		if err != nil || copy != before || *r != before || meter.failed {
			t.Fatal("closed request copy changed original bytes or exceeded its envelope", tc.method, err, meter.actual)
		}
		for coordinate, actual := range meter.actual {
			if actual > upper.requestCopy[coordinate] {
				t.Fatal("request copy escaped its coordinate", tc.method, coordinate, actual)
			}
		}
		classMeter := providerHistoryContentUnitMeter(upper.requestClass)
		class, err := providerHistoryGroupRequestClassObserved(r, classMeter)
		if err != nil || class != tc.class || classMeter.failed || *r != before {
			t.Fatal("class reconstruction lost original request or distinction", tc.method, class, err)
		}
		for coordinate, actual := range classMeter.actual {
			if actual > upper.requestClass[coordinate] {
				t.Fatal("class decode escaped its coordinate", tc.method, coordinate, actual)
			}
		}
	}
}

func TestProviderHistoryLedgerControlCostsFundIndependentRecordAndLedgerValidation(t *testing.T) {
	// These legacy typed DATA fixtures have no cursor/Source, network or
	// original execution permission. They isolate the quoted control work.
	for _, count := range []int{1, 32} {
		upper, ok := providerHistoryLedgerControlUpper(uint64(count))
		if !ok {
			t.Fatal("finite record/ledger precursor")
		}
		maximum := historyLedgerMaximum()
		maximum.work[0] = uint64(2 * count)
		l := historyLedgerFixture(t, maximum)
		checkLedger := func() {
			before := *l
			meter := providerHistoryContentUnitMeter(upper.ledger)
			if !l.validObserved(meter) || meter.failed || *l != before {
				t.Fatal("read-only ledger validation changed DATA or exceeded control quote", count, l.phase, meter.actual)
			}
			for coordinate, actual := range meter.actual {
				if actual > upper.ledger[coordinate] {
					t.Fatal("ledger control escaped a coordinate", count, l.phase, coordinate, actual)
				}
			}
		}
		checkLedger()
		ticket := historyRecordTicket(t, count, 1, l.head)
		beforeTicket := *ticket
		meter := providerHistoryContentUnitMeter(upper.ticket)
		if !providerHistoryGroupTicketValidObserved(ticket, meter) || meter.failed || *ticket != beforeTicket {
			t.Fatal("original-shaped ticket reconstruction exceeded its quote", count, meter.actual)
		}
		for coordinate, actual := range meter.actual {
			if actual > upper.ticket[coordinate] {
				t.Fatal("ticket validation escaped coordinate", count, coordinate)
			}
		}
		if l.admit(ticket) != nil {
			t.Fatal("normal typed admission precursor")
		}
		checkLedger()
		consumption, err := newProviderHistoryGroupConsumption(ticket, l.head, admissionTestDigest("1"))
		if err != nil {
			t.Fatal("normal typed consumption precursor", err)
		}
		beforeConsumption := *consumption
		meter = providerHistoryContentUnitMeter(upper.consumption)
		if !providerHistoryGroupConsumptionValidObserved(consumption, ticket, meter) || meter.failed || *ticket != beforeTicket || *consumption != beforeConsumption {
			t.Fatal("nested consumption/ticket validation changed originals or exceeded quote", count, meter.actual)
		}
		for coordinate, actual := range meter.actual {
			if actual > upper.consumption[coordinate] {
				t.Fatal("consumption validation escaped coordinate", count, coordinate)
			}
		}
		if l.consume(consumption) != nil {
			t.Fatal("normal typed consumed ledger precursor")
		}
		checkLedger()
		results := make([]string, count)
		for i := range results {
			results[i] = admissionTestDigest("2")
		}
		settlement, err := newProviderHistoryGroupSettlement(ticket, consumption, l.head, admissionTestDigest("3"), "complete", "", results)
		if err != nil || l.settle(settlement) != nil {
			t.Fatal("normal typed settled ledger precursor", err)
		}
		checkLedger()
	}
}

func TestProviderHistoryLedgerControlCostsComposeWithOriginalShapedCursor(t *testing.T) {
	// Synthetic canonical DATA only: actual cursor/ledger routines are used,
	// but this fixture cannot authenticate a real deployment or live Source.
	c, headers, methods := historyCollectionBoundaryFixture(t, 13, 4)
	l := historyCollectionLedger(t, c)
	control, ok := providerHistoryLedgerControlUpper(32)
	if !ok {
		t.Fatal("complete control precursor")
	}
	check := func() {
		cursor, bounded := providerHistoryCollectionCursorUpper(historyCollectionCursorCostLimits(t, l.collection))
		if !bounded {
			t.Fatal("independently closed original cursor precursor")
		}
		var work candidateWorkVector
		for coordinate := range work {
			var added bool
			work[coordinate], added = providerUpperSum(cursor.commitment[coordinate], control.ledger[coordinate])
			if !added {
				t.Fatal("normal cursor/control component overflow")
			}
		}
		original := l.collection.semantic.observer
		beforeWork, beforeLedger := original.actual, *l
		meter := providerHistoryContentUnitMeter(work)
		if !l.validObserved(meter) || meter.failed || *l != beforeLedger || original.actual != beforeWork || l.collection.semantic.observer != original {
			t.Fatal("read-only cursor/control composition changed execution or DATA", l.phase, meter.actual)
		}
		for coordinate, actual := range meter.actual {
			if actual > work[coordinate] {
				t.Fatal("connected validation escaped its independent components", l.phase, coordinate, actual)
			}
		}
	}
	check()
	settlement, results := historyCollectionPendingSettlement(t, l, headers, methods)
	check()
	if err := l.settleOriginalsObserved(context.Background(), settlement, results, methods, l.collection.semantic.observer); err != nil {
		t.Fatal("normal original-shaped successful boundary precursor", err)
	}
	check()
}

func TestProviderHistoryLedgerControlBoundsRefuseShortAndInvalidBudgets(t *testing.T) {
	upper, ok := providerHistoryLedgerControlUpper(2)
	if !ok {
		t.Fatal("finite request/ledger refusal precursor")
	}
	r := historyGroupSimple(t)
	beforeRequest := *r
	shortWork := upper.requestCopy
	shortWork[10] = 16384 * 16384 // one less than the original body encode preflight
	short := providerHistoryContentUnitMeter(shortWork)
	if copy, err := providerHistoryGroupRequestCopyObserved(r, short); err == nil || copy != (providerIntegrationRequest{}) || !short.failed || *r != beforeRequest || short.actual[2] == 0 {
		t.Fatal("insufficient original body preflight returned a copy or changed input", err, short.actual)
	}
	if short.event(providerCopyEvent, 0) {
		t.Fatal("failed request meter revived")
	}
	l := historyLedgerFixture(t, historyLedgerMaximum())
	ticket := historyRecordTicket(t, 2, 1, l.head)
	if l.admit(ticket) != nil {
		t.Fatal("normal admitted ledger for recursive quota test")
	}
	consumption, err := newProviderHistoryGroupConsumption(ticket, l.head, admissionTestDigest("1"))
	if err != nil || l.consume(consumption) != nil {
		t.Fatal("normal consumed ledger for recursive quota test", err)
	}
	measured := providerHistoryContentUnitMeter(upper.ledger)
	if !l.validObserved(measured) || measured.actual[2] == 0 {
		t.Fatal("normal observed recursive ledger precursor", measured.actual)
	}
	before := *l
	shortWork = upper.ledger
	shortWork[2] = measured.actual[2] - 1
	short = providerHistoryContentUnitMeter(shortWork)
	if l.validObserved(short) || !short.failed || *l != before || short.event(providerCopyEvent, 0) {
		t.Fatal("short recursive decode budget changed DATA or revived", short.actual)
	}
	for _, members := range []uint64{0, 33, ^uint64(0)} {
		if got, valid := providerHistoryLedgerControlUpper(members); valid || got != (providerHistoryLedgerControlCost{}) {
			t.Fatal("invalid cardinality became partial control funding", members, got, valid)
		}
	}
	full, ok := providerHistoryLedgerControlUpper(32)
	if !ok {
		t.Fatal("whole-call composition precursor")
	}
	for _, tc := range []struct{ groups, requests, roots uint64 }{{1, 1, 114}, {2, 33, 640}} {
		var want candidateWorkVector
		for coordinate, units := range full.ledger {
			want[coordinate] = tc.roots * units
		}
		if got, valid := providerHistoryCollectionRepeatedLedgerControlUpper(tc.groups, tc.requests); !valid || got != want {
			t.Fatal("repeated original ledger control omitted a call", tc, got, want, valid)
		}
	}
	for _, tc := range []struct{ groups, requests uint64 }{{0, 1}, {2, 1}, {1, 33}, {349525, 11184800}} {
		if got, valid := providerHistoryCollectionRepeatedLedgerControlUpper(tc.groups, tc.requests); valid || got != (candidateWorkVector{}) {
			t.Fatal("invalid or overflowing whole ledger quote exposed a partial vector", tc, got, valid)
		}
	}
}

func TestProviderHistoryRoleControlPartsMatchIndependentRowsAndClockMetric(t *testing.T) {
	metric, ok := providerEventVector(providerClockEvent, 0)
	if !ok || metric != (candidateWorkVector{10: 1}) {
		t.Fatal("original clock logical metric changed; source prices need a new derivation")
	}
	for _, tc := range []struct {
		prior, before2, before10, closure2, closure10, advance2, advance10 uint64
	}{
		{0, 8967648, 164556544762, 6281868, 146886167933, 5104712, 138806324988},
		{1, 14378052, 200454413869, 8085336, 158852124302, 6307024, 146783629234},
		{7, 46840476, 415841628511, 18906144, 230647862516, 13520896, 194647454710},
	} {
		got, accepted := providerHistoryCollectionRoleControlPartsUpper(tc.prior, 1<<20, true)
		want := providerHistoryRoleControlParts{
			initial:     providerHistoryStageWorkCost{work: candidateWorkVector{2: 1112576, 10: 39413950035}, clocks: 2},
			before:      providerHistoryStageWorkCost{work: candidateWorkVector{2: tc.before2, 10: tc.before10}, clocks: 6},
			preparation: providerHistoryStageWorkCost{work: candidateWorkVector{2: 1624576, 10: 43050564098}, clocks: 1},
			member:      providerHistoryStageWorkCost{work: candidateWorkVector{2: 1624576, 10: 43050564099}, clocks: 2},
			closure:     providerHistoryStageWorkCost{work: candidateWorkVector{2: tc.closure2, 10: tc.closure10}, clocks: 2},
			advance:     providerHistoryStageWorkCost{work: candidateWorkVector{2: tc.advance2, 10: tc.advance10}, clocks: 2},
		}
		if !accepted || got != want {
			t.Fatal("source role control/owned-clock rows differ from independent integer literals", tc, got, want, accepted)
		}
		for coordinate := range got.preparation.work {
			if got.preparation.work[coordinate] > got.member.work[coordinate] {
				t.Fatal("entered preparation control exceeds full member control", coordinate)
			}
		}
		if got.preparation.clocks > got.member.clocks {
			t.Fatal("preparation clock count exceeds full member")
		}
		// CONTROL DATA itself has no RPC/wire/write row or complete dispatch.
		for _, part := range []providerHistoryStageWorkCost{got.initial, got.before, got.preparation, got.member, got.closure, got.advance} {
			for _, coordinate := range []int{0, 1, 3, 4, 5, 6, 7, 8, 9} {
				if part.work[coordinate] != 0 {
					t.Fatal("control subgraph invented another resource domain", coordinate, part)
				}
			}
		}
		if (providerHistoryDispatchBounds{member: got.member.work, closure: got.closure.work, memberClocks: 2, closureClocks: 2, tailMS: 1}).valid() {
			t.Fatal("control-only DATA opened a complete dispatch row")
		}
	}
}

func TestProviderHistoryRoleControlKeepsAbsentAdmissionComponentAndFinalAdvanceSeparate(t *testing.T) {
	preparation, ok := providerHistoryRoleControlPartUpper(10, 0, 0, 1)
	if !ok || preparation != (providerHistoryStageWorkCost{work: candidateWorkVector{2: 1624576, 10: 43050564098}, clocks: 1}) {
		t.Fatal("source-proved absent admission component acquired a false count-zero failure")
	}
	for _, prior := range []uint64{0, 1, 7, (1<<20)/3 - 1} {
		last, ok := providerHistoryCollectionRoleControlPartsUpper(prior, 1<<20, false)
		if !ok || last.advance != (providerHistoryStageWorkCost{}) {
			t.Fatal("final-stage DATA borrowed unused advance", prior, last, ok)
		}
		if prior == (1<<20)/3-1 {
			continue
		}
		more, ok := providerHistoryCollectionRoleControlPartsUpper(prior, 1<<20, true)
		more.advance = providerHistoryStageWorkCost{}
		if !ok || more != last {
			t.Fatal("own advance repriced another original role", prior)
		}
	}
	// The collection wrapper always has a nonempty B admission component;
	// its original maximum cannot be bypassed by initial/P/M's zero atoms.
	for _, maximum := range []uint64{0, ^uint64(0)} {
		if got, ok := providerHistoryCollectionRoleControlPartsUpper(0, maximum, false); ok || got != (providerHistoryRoleControlParts{}) {
			t.Fatal("invalid original maximum left partial initial data", maximum, got, ok)
		}
	}
}

func TestProviderHistoryRoleControlRejectsInvalidScopeAndWholeZeroOverflow(t *testing.T) {
	for _, item := range []struct {
		prior     uint64
		advancing bool
	}{
		{(1 << 20) / 3, false}, {^uint64(0), true}, {(1<<20)/3 - 1, true},
	} {
		if got, ok := providerHistoryCollectionRoleControlPartsUpper(item.prior, 1<<20, item.advancing); ok || got != (providerHistoryRoleControlParts{}) {
			t.Fatal("impossible original-stage selection accepted", item, got, ok)
		}
	}
	for _, item := range []struct{ ledger, admission, maximum, clocks uint64 }{
		{0, 0, 0, 0},
		{^uint64(0), 0, 0, 0},
		{1, ^uint64(0), 1 << 20, 0},
		{1, 1, 0, 0},
		{40580042227, 1, 1 << 20, 0},
		{1, 0, 0, ^uint64(0)},
	} {
		if got, ok := providerHistoryRoleControlPartUpper(item.ledger, item.admission, item.maximum, item.clocks); ok || got != (providerHistoryStageWorkCost{}) {
			t.Fatal("invalid source counts or overflow retained partial DATA", item, got, ok)
		}
	}
	// Pure scalar boundary; this is far outside every live clock profile.
	const encodedLedger uint64 = 38959373275
	clocks := ^uint64(0) - encodedLedger
	got, ok := providerHistoryRoleControlPartUpper(1, 0, 0, clocks)
	if !ok || got.work != (candidateWorkVector{2: 1048576, 10: ^uint64(0)}) || got.clocks != clocks {
		t.Fatal("exact checked scalar MAX rejected", got, ok)
	}
	if got, ok := providerHistoryRoleControlPartUpper(1, 0, 0, clocks+1); ok || got != (providerHistoryStageWorkCost{}) {
		t.Fatal("late logical clock-event overflow did not return whole zero", got, ok)
	}
}

func TestProviderHistoryRoleControlComposesEveryOriginalStageClockWithoutGlobalRefund(t *testing.T) {
	for groups := 1; groups <= 16; groups++ {
		var fullClockHold, requests uint64
		for stage := 0; stage < groups; stage++ {
			count := 1 + (19*stage+7)%providerHistoryGroupMembers
			advancing := stage < groups-1
			parts, ok := providerHistoryCollectionRoleControlPartsUpper(uint64(stage), 1<<20, advancing)
			if !ok {
				t.Fatal("finite source control components rejected")
			}
			if stage == 0 {
				fullClockHold = parts.initial.clocks
			}
			// Shape-only scalar UNIT additions for the OTHER member domains.
			// They are not complete source prices or a real owner/profile.
			rows := providerHistoryDispatchBounds{member: parts.member.work, closure: parts.closure.work,
				memberClocks: parts.member.clocks, closureClocks: parts.closure.clocks, tailMS: 1}
			rows.member[0], rows.member[1], rows.member[9], rows.closure[9] = 1, 101, 1, 1
			if advancing {
				rows.advance, rows.advanceClocks, rows.advanceTailMS = parts.advance.work, parts.advance.clocks, 1
			}
			window, ok := providerHistoryCollectionStageWorkUpper(parts.before, parts.preparation, rows, count, advancing)
			want := uint64(9 + 2*count)
			if advancing {
				want += 2
			}
			if !ok || window.clocks != want || window.work[0] != uint64(count) || window.work[1] != uint64(count)*101 {
				t.Fatal("source clock cut or exact original request/wire changed", stage, count, advancing, window, ok)
			}
			fullClockHold += window.clocks
			requests += uint64(count)
		}
		// Each stage retains its own already-entered P; one global P is short
		// by groups-1. All counters here are scalar DATA, not clock samples.
		actualCallBound := uint64(10*groups) + 2*requests
		if fullClockHold != uint64(11*groups)+2*requests || fullClockHold-actualCallBound != uint64(groups) {
			t.Fatal("full original stage clock sums replaced by global headroom", groups, fullClockHold, actualCallBound)
		}
		if groups > 1 && fullClockHold <= actualCallBound+1 {
			t.Fatal("multiple original preparation overlaps funded by only one global clock")
		}
	}
}

func TestProviderHistoryRoleControlRespectsWiderOriginalIncomingClockRows(t *testing.T) {
	parts, ok := providerHistoryCollectionRoleControlPartsUpper(0, 1<<20, true)
	if !ok {
		t.Fatal("source control row unavailable")
	}
	// First bind exact source counts in scalar DATA before either assembly.
	// No live window, meter or hold is changed by these independent examples.
	rows := providerHistoryDispatchBounds{member: parts.member.work, closure: parts.closure.work,
		memberClocks: parts.member.clocks, closureClocks: parts.closure.clocks, tailMS: 1,
		advance: parts.advance.work, advanceClocks: parts.advance.clocks, advanceTailMS: 1}
	rows.member[0], rows.member[1], rows.member[9], rows.closure[9] = 1, 101, 1, 1
	exact, ok := providerHistoryCollectionStageWorkUpper(parts.before, parts.preparation, rows, 3, true)
	if !ok || exact.clocks != 17 {
		t.Fatal("exact source count differs from independent6+1+3*2+2+2", exact, ok)
	}
	// Existing original DATA may select wider incoming allowances. The same
	// assembler must preserve them instead of replacing them with source2/2.
	rows.memberClocks, rows.closureClocks = 4, 3
	held, ok := providerHistoryCollectionStageWorkUpper(parts.before, parts.preparation, rows, 3, true)
	if !ok || held.clocks != 24 || held.clocks <= exact.clocks || rows.memberClocks != 4 || rows.closureClocks != 3 {
		t.Fatal("source call counts replaced wider original incoming allowances", held, exact, rows, ok)
	}
}
