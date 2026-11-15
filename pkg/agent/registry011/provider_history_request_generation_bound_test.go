package registry011

import (
	"fmt"
	"strings"
	"testing"
)

// Complete, internally consistent inert grammar DATA only. All selections are
// fixed before constructing the cursor/fold. None of these states authenticates
// an original S, live Source, custody acknowledgement or request permission.
func historyRequestGenerationCursor(t *testing.T, phase string) (*providerHistoryCollectionCursor, *providerMethodBounds) {
	t.Helper()
	if phase == "effects" {
		c, input, methods := historyCollectionContentInputFixture(t, 2, historyCollectionStreamingCountInput(t, 33), SepoliaAmsterdamRootExecutionGasProfile)
		c = historyCollectionContentStep(t, c, input, methods) // open
		c = historyCollectionContentStep(t, c, input, methods) // complete content
		if c.phase != "effects" || len(c.effects) != 33 || c.commitmentObserved(providerReplayUnitMeter()) == "" {
			t.Fatal("normal complete33-effect DATA precursor")
		}
		return c, methods
	}
	in := historyCollectionInput(t)
	in.effects, in.versions, in.presentStops = 64, 64, 0
	selected := historyFirstFinalizedHeader()
	selected.Timestamp = sepoliaAmsterdamTime + 100
	if phase == "content-before" {
		selected.Timestamp = sepoliaAmsterdamTime - 1
		in.encoding = "debug-raw-rlp-v1"
	}
	if phase == "content-raw-state" {
		in.encoding, in.gas = "debug-raw-rlp-v1", SepoliaAmsterdamGasProfile
	}
	c, err := newProviderHistoryCollectionCursorObserved(admissionTestDigest("a"), historyCollectionManifest(t, in), selected, providerReplayUnitMeter())
	if err != nil || c == nil {
		t.Fatal("normal preselected grammar DATA", err)
	}
	// A consistent typed spine, not returned RPC or runtime chain evidence.
	headers := make([]FinalizedHeader, int(in.blocks))
	headers[0] = in.journal.Deployment
	for i := 1; i < len(headers)-1; i++ {
		headers[i] = FinalizedHeader{fmt.Sprintf("0x%064x", i+1), headers[i-1].Hash,
			in.journal.Deployment.Number + uint64(i), selected.Timestamp - uint64(len(headers)-1-i)}
	}
	headers[len(headers)-2].Hash = selected.ParentHash
	headers[len(headers)-1] = selected
	c.spine = make([]FinalizedHeader, len(headers))
	for i := range headers {
		c.spine[i] = headers[len(headers)-1-i]
	}
	c.phase = phase
	if phase != "open" && phase != "ancestry" {
		c.mode, c.low, c.high, c.next = "content", selected.Number-3, selected.Number, selected.Number
	}
	switch phase {
	case "content-before", "content-json-root", "content-raw-state":
		c.phase, c.low, c.high, c.next = "content", in.journal.Deployment.Number, in.journal.Deployment.Number+3, in.journal.Deployment.Number
	case "history":
		c.historyNext, c.stopVersion = 1, 33
	}
	if !c.valid() || c.commitmentObserved(providerReplayUnitMeter()) == "" {
		t.Fatal("inert grammar fixture is internally inconsistent", phase)
	}
	bounds, _ := providerFrontierBoundsFixture()
	return c, bounds.methods
}

func TestProviderHistoryRequestGenerationMatchesIndependentRows(t *testing.T) {
	for _, row := range []struct{ members, effects, decoded, logical uint64 }{
		{1, 0, 8192, 403145135}, {1, 1, 8192, 403145401},
		{2, 33, 16384, 806299048}, {32, 163840, 262144, 12944225760},
	} {
		got, ok := providerHistoryRequestGenerationUpper(row.members, row.effects)
		if !ok || got != (candidateWorkVector{2: row.decoded, 10: row.logical}) {
			t.Fatal("independent generation/scanned-pending-effect quote", row, got)
		}
	}
	for _, row := range []struct{ groups, requests, effects, decoded, logical uint64 }{
		{1, 1, 0, 131072, 6450322160},
		{1, 32, 33, 4194304, 206410449568},
		{2, 3, 33, 851968, 41927620720},
		{2, 33, 163840, 11665408, 576693558640},
		{3, 96, 163840, 34603008, 1708637800320},
	} {
		got, ok := providerHistoryCollectionRepeatedRequestGenerationUpper(row.groups, row.requests, row.effects)
		if !ok || got != (candidateWorkVector{2: row.decoded, 10: row.logical}) {
			t.Fatal("independent nextTable-only generation multiplicities", row, got)
		}
	}
}

func TestProviderHistoryRequestGenerationFundsPinnedRegistryCalldata(t *testing.T) {
	for _, agent := range []string{"a", strings.Repeat("A", 64)} {
		for _, method := range []string{"record", "activeGrants", "historyItem"} {
			version := uint64(0)
			if method == "historyItem" {
				version = ^uint64(0)
			}
			m := providerHistoryContentUnitMeter(candidateWorkVector{10: 27420})
			params, err := providerHistoryRegistryParamsObserved(SepoliaRegistryAddress, agent, method, version, admissionTestDigest("b"), m)
			if err != nil || len(params) != 2 || m.failed || m.actual[2] != 0 || m.actual[10] > 27420 {
				t.Fatal("closed selector/ABI/hex explicit events exceeded quote", method, err)
			}
			r := historyGroupRequest(t, "eth_call", params)
			historyCheckPlannedRegistryCall(t, *r, method, agent, version, admissionTestDigest("b"))
		}
	}
}

func TestProviderHistoryRequestGenerationComposesWithTablesInEveryPhase(t *testing.T) {
	for _, row := range []struct {
		phase   string
		members int
	}{
		{"open", 18}, {"ancestry", 4}, {"content-before", 4},
		{"content-json-root", 5}, {"content-raw-state", 5}, {"effects", 32},
		{"state", 1}, {"grants", 1}, {"history", 32}, {"close", 7},
	} {
		c, methods := historyRequestGenerationCursor(t, row.phase)
		before, original := c.commitmentObserved(providerReplayUnitMeter()), c.semantic.observer.actual
		generation, generationOK := providerHistoryRequestGenerationUpper(32, 64)
		tableCost, tableOK := providerHistoryRequestControlUpper(32, 1<<20)
		if !generationOK || !tableOK || before == "" {
			t.Fatal("complete generation/table DATA precursor", row.phase)
		}
		var work candidateWorkVector
		for i := range work {
			var ok bool
			work[i], ok = providerUpperSum(generation[i], tableCost.tableBuilt[i])
			if !ok {
				t.Fatal("finite disjoint generation/table sum")
			}
		}
		m := providerHistoryContentUnitMeter(work)
		table, err := c.nextTableObserved(c.completed+1, *methods, c.selection.input.capture, m)
		if err != nil || table == nil || table.count != row.members || m.failed ||
			c.commitmentObserved(providerReplayUnitMeter()) != before || c.semantic.observer.actual != original {
			t.Fatal("normal closed grammar/table creation changed original DATA", row.phase, err)
		}
		for i, actual := range m.actual {
			if actual > work[i] {
				t.Fatal("normal grammar escaped its disjoint quoted components", row.phase, i)
			}
		}
	}
}

func TestProviderHistoryRequestGenerationScansAllPendingEffectsForPartialTail(t *testing.T) {
	c, input, methods := historyCollectionContentInputFixture(t, 2, historyCollectionStreamingCountInput(t, 33), SepoliaAmsterdamRootExecutionGasProfile)
	c = historyCollectionContentStep(t, c, input, methods) // open
	c = historyCollectionContentStep(t, c, input, methods) // complete content
	c = historyCollectionContentStep(t, c, input, methods) // original first32 results
	before := c.commitmentObserved(providerReplayUnitMeter())
	if before == "" || c.phase != "effects" || len(c.effects) != 33 || c.effectNext != 32 || c.traces != 32 || c.pendingApplied != 1 || c.semantic.blocks != 1 {
		t.Fatal("valid typed one-member tail with33 original pending effects")
	}
	valid := providerHistoryContentUnitMeter(candidateWorkVector{10: 8778})
	if !c.effectFactsObserved(valid) || valid.failed || valid.actual != (candidateWorkVector{10: 8778}) {
		t.Fatal("pending effect scan must cover all33, not just the selected last member", valid.actual)
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{10: 8777})
	if c.effectFactsObserved(short) || !short.failed || short.actual != (candidateWorkVector{10: 8645}) ||
		short.event(providerCompareEvent, 0) || c.commitmentObserved(providerReplayUnitMeter()) != before {
		t.Fatal("last pending block-hash comparison lost successful prefix or changed DATA", short.actual)
	}
	generation, a := providerHistoryRequestGenerationUpper(1, 33)
	tableCost, b := providerHistoryRequestControlUpper(1, 1<<20)
	if !a || !b {
		t.Fatal("finite complete one-member tail quote")
	}
	var work candidateWorkVector
	for i := range work {
		var ok bool
		work[i], ok = providerUpperSum(generation[i], tableCost.tableBuilt[i])
		if !ok {
			t.Fatal("tail component sum")
		}
	}
	m := providerHistoryContentUnitMeter(work)
	table, err := c.nextTableObserved(c.completed+1, *methods, c.selection.input.capture, m)
	if err != nil || table == nil || table.count != 1 || m.failed || !strings.Contains(table.members[0].request.params, `"onlyTopCall":false`) ||
		!strings.Contains(table.members[0].request.params, `"withLog":true`) || c.commitmentObserved(providerReplayUnitMeter()) != before {
		t.Fatal("complete last effect request lost its full-trace selection", err)
	}
}

func TestProviderHistoryRequestGenerationKeepsGenuineConstructorFailurePrefix(t *testing.T) {
	c, methods := historyRequestGenerationCursor(t, "open")
	before := c.commitmentObserved(providerReplayUnitMeter())
	full, fullErr := c.nextTableObserved(1, *methods, c.selection.input.capture, providerReplayUnitMeter())
	if fullErr != nil || full == nil || full.count != 18 || c.commitmentObserved(providerReplayUnitMeter()) != before {
		t.Fatal("normal complete constructor refusal precursor", fullErr)
	}
	// The normal full quote is also checked by the every-phase root.
	// One less than the original body preflight refuses AFTER parameter work;
	// it does not enter table construction or grant any partially built request.
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 1 << 20, 10: 16384 * 16384})
	table, err := c.nextTableObserved(1, *methods, c.selection.input.capture, m)
	if err == nil || table != nil || !m.failed || m.actual[10] == 0 || m.actual[2] != 0 ||
		m.event(providerCopyEvent, 0) || before == "" || c.commitmentObserved(providerReplayUnitMeter()) != before {
		t.Fatal("body preflight shortage erased successful prefix or changed cursor", err, m.actual)
	}
	valid := providerHistoryContentUnitMeter(candidateWorkVector{10: 84})
	if got := providerHistoryNumberTagObserved(^uint64(0), valid); got != "0xffffffffffffffff" || valid.failed || valid.actual[10] != 84 {
		t.Fatal("maximum uint64 tag precursor", got, valid.actual)
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{10: 66})
	if !short.event(providerCopyEvent, 1) || providerHistoryNumberTagObserved(0, short) != "" || !short.failed || short.actual[10] != 2 ||
		short.event(providerEncodeEvent, 0) {
		t.Fatal("tag preflight shortage reset an earlier successful copy", short.actual)
	}
}

func TestProviderHistoryRequestGenerationRefusesInvalidAndOverflowingQuotes(t *testing.T) {
	for _, row := range [][2]uint64{{0, 0}, {33, 0}, {1, 163841}, {^uint64(0), ^uint64(0)}} {
		if got, ok := providerHistoryRequestGenerationUpper(row[0], row[1]); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid grammar member/effect limits returned a partial quote", row, got)
		}
	}
	for _, row := range [][3]uint64{{0, 0, 0}, {1, 0, 0}, {1, 33, 0}, {2, 1, 0}, {349526, 349526, 0}, {1, 1, 163841}, {349525, 11184800, 163840}} {
		if got, ok := providerHistoryCollectionRepeatedRequestGenerationUpper(row[0], row[1], row[2]); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid or late-logical-overflow grammar returned a partial quote", row, got)
		}
	}
}

func TestProviderHistoryPlannedNextTableControlMatchesIndependentRows(t *testing.T) {
	for _, row := range []struct {
		members, effects                           uint64
		normalDecoded, normalLogical               uint64
		envelopeDecoded, envelopeLogical, equality uint64
	}{
		{1, 0, 2000, 2646675, 32768, 5235595086, 140832},
		{2, 33, 4000, 5620297, 65536, 6176100546, 141672},
		{32, 163840, 64000, 302190279, 1048576, 34434573668, 166872},
	} {
		got, ok := providerHistoryPlannedNextTableControlUpper(row.members, row.effects, 1<<20)
		want := providerHistoryPlannedNextTableControlCost{
			generated: providerHistoryControlPair{
				candidateWorkVector{2: row.normalDecoded, 10: row.normalLogical},
				candidateWorkVector{2: row.envelopeDecoded, 10: row.envelopeLogical}},
			equality: providerHistoryControlPair{candidateWorkVector{10: row.equality}, candidateWorkVector{10: 2812518}},
		}
		if !ok || got != want {
			t.Fatal("independent generated-request/final-table/residual-equality components", row, got)
		}
		for _, pair := range []providerHistoryControlPair{got.generated, got.equality} {
			for coordinate, normal := range pair.normal {
				if normal > pair.envelope[coordinate] {
					t.Fatal("successful normal cannot exceed its unchanged unfinished envelope", coordinate)
				}
			}
		}
	}
}

func TestProviderHistoryPlannedNextTableNormalFundsEveryOriginalPhase(t *testing.T) {
	for _, phase := range []string{"open", "ancestry", "content-before", "content-json-root", "content-raw-state", "effects", "state", "grants", "history", "close"} {
		c, methods := historyRequestGenerationCursor(t, phase)
		before := c.commitmentObserved(providerReplayUnitMeter())
		cost, ok := providerHistoryPlannedNextTableControlUpper(32, 64, 1<<20)
		if !ok || before == "" {
			t.Fatal("complete planned next-table DATA precursor", phase)
		}
		// All generic incoming preflights stay funded by the original envelope.
		// Only the completed real constructor's entered work is compared to N.
		m := providerHistoryContentUnitMeter(cost.generated.envelope)
		table, err := c.nextTableObserved(c.completed+1, *methods, c.selection.input.capture, m)
		if err != nil || table == nil || m.failed || c.commitmentObserved(providerReplayUnitMeter()) != before {
			t.Fatal("original generated-table route refused or changed its cursor", phase, err)
		}
		for coordinate, actual := range m.actual {
			if actual > cost.generated.normal[coordinate] {
				t.Fatal("completed original phase exceeded planned entered-work component", phase, coordinate, actual)
			}
		}
		m = providerHistoryContentUnitMeter(cost.equality.envelope)
		if !providerHistoryCollectionTableEqualObserved(table, table, m) || m.failed {
			t.Fatal("original whole fixed-table equality", phase)
		}
		for coordinate, actual := range m.actual {
			if actual > cost.equality.normal[coordinate] {
				t.Fatal("completed generated-table equality exceeded its component", phase, coordinate, actual)
			}
		}
	}
}

func TestProviderHistoryRoleRequestControlCallsMatchIndependentCuts(t *testing.T) {
	for _, row := range []struct {
		prior                uint64
		before, good, bad, a [4]uint64
	}{
		{0, [4]uint64{10, 1, 10, 0}, [4]uint64{6, 3, 6, 6}, [4]uint64{3, 0, 6, 6}, [4]uint64{4, 2, 4, 4}},
		{1, [4]uint64{28, 10, 28, 18}, [4]uint64{12, 6, 12, 12}, [4]uint64{9, 3, 12, 12}, [4]uint64{8, 4, 8, 8}},
		{7, [4]uint64{136, 64, 136, 126}, [4]uint64{48, 24, 48, 48}, [4]uint64{45, 21, 48, 48}, [4]uint64{32, 16, 32, 32}},
		{9225, [4]uint64{166060, 83026, 166060, 166050}, [4]uint64{55356, 27678, 55356, 55356}, [4]uint64{55353, 27675, 55356, 55356}, [4]uint64{36904, 18452, 36904, 36904}},
	} {
		toCalls := func(v [4]uint64) providerHistoryRoleRequestControlCalls {
			return providerHistoryRoleRequestControlCalls{v[0], v[1], v[2], v[3]}
		}
		want := providerHistoryRoleRequestControlCounts{toCalls(row.before), toCalls(row.good), toCalls(row.bad), toCalls(row.a)}
		got, ok := providerHistoryCollectionRoleRequestControlCallsBound(row.prior, true)
		if !ok || got != want {
			t.Fatal("independent B/C-success/C-failed/old-T advance cuts", row.prior, got)
		}
		want.advance = providerHistoryRoleRequestControlCalls{}
		got, ok = providerHistoryCollectionRoleRequestControlCallsBound(row.prior, false)
		if !ok || got != want {
			t.Fatal("final or failed S cannot create own advance work", row.prior, got)
		}
	}
}

func TestProviderHistoryRoleRequestControlConservesWholeOriginalGrammar(t *testing.T) {
	for groups := uint64(1); groups <= 64; groups++ {
		var total providerHistoryRoleRequestControlCalls
		var atoms uint64
		for prior := uint64(0); prior < groups; prior++ {
			calls, ok := providerHistoryCollectionRoleRequestControlCallsBound(prior, prior+1 < groups)
			if !ok {
				t.Fatal("finite original source grammar")
			}
			for _, role := range []providerHistoryRoleRequestControlCalls{calls.before, calls.successClosure, calls.advance} {
				total.nextTables += role.nextTables
				total.tableEquals += role.tableEquals
				total.consumptions += role.consumptions
				total.settlements += role.settlements
			}
			atoms += 8 + 9*prior + 3 + 3*prior
			if prior+1 < groups {
				atoms += 2 + 2*prior
			}
			if calls.successClosure.nextTables-calls.failedClosure.nextTables != 3 ||
				calls.successClosure.tableEquals-calls.failedClosure.tableEquals != 3 ||
				calls.successClosure.consumptions != calls.failedClosure.consumptions ||
				calls.successClosure.settlements != calls.failedClosure.settlements {
				t.Fatal("known failed S omits exactly its three current boundaries")
			}
		}
		for _, members := range []uint64{1, 32} {
			old, ok := providerHistoryCollectionRequestControlCallsBound(groups, groups*members)
			if !ok || total.nextTables != old.tableBuilds-old.packParses ||
				total.tableEquals+atoms != old.tableEquals || 2*atoms != old.ticketOps ||
				total.consumptions != old.consumptionOps || total.settlements != old.settlementOps {
				t.Fatal("disjoint new components plus existing admission atoms must conserve old successful grammar", groups, members, total)
			}
		}
	}
}

func TestProviderHistoryRoleRequestControlPartsMatchIndependentPrices(t *testing.T) {
	for _, row := range []struct {
		prior                uint64
		before, good, bad, a [2]uint64
	}{
		{0, [2]uint64{2436864, 58660679226}, [2]uint64{3421440, 98095074205}, [2]uint64{3229440, 97185357106}, [2]uint64{3101440, 96007623895}},
		{1, [2]uint64{6301440, 115890566647}, [2]uint64{4381440, 104357425135}, [2]uint64{4189440, 103450353682}, [2]uint64{3741440, 100182524515}},
		{7, [2]uint64{23581440, 228612883387}, [2]uint64{10141440, 141931530715}, [2]uint64{9949440, 141024459262}, [2]uint64{7581440, 125231928235}},
		{9225, [2]uint64{26571421440, 173407665501607}, [2]uint64{8859421440, 57868282403455}, [2]uint64{8859229440, 57867375332002}, [2]uint64{5907101440, 38609465843395}},
	} {
		vector := func(v [2]uint64) candidateWorkVector { return candidateWorkVector{2: v[0], 10: v[1]} }
		calls, a := providerHistoryCollectionRoleRequestControlCallsBound(row.prior, true)
		next, b := providerHistoryPlannedNextTableControlUpper(32, 163840, 1<<20)
		record, c := providerHistoryPlannedRecordControlUpper(32)
		if !a || !b || !c {
			t.Fatal("independent-price precursor")
		}
		good, d := calls.successClosure.upper(next, record)
		bad, e := calls.failedClosure.upper(next, record)
		if !d || !e || good != vector(row.good) || bad != vector(row.bad) {
			t.Fatal("success and failed closures need separate whole components", row.prior, good, bad)
		}
		got, ok := providerHistoryCollectionRoleRequestControlPartsUpper(row.prior, 163840, 1<<20, true)
		want := providerHistoryRoleRequestControlParts{before: vector(row.before), advance: vector(row.a)}
		for coordinate := range want.closure {
			want.closure[coordinate] = max(good[coordinate], bad[coordinate])
		}
		if !ok || got != want || got.initial != (candidateWorkVector{}) || got.preparation != (candidateWorkVector{}) || got.member != (candidateWorkVector{}) {
			t.Fatal("role prices must preserve every envelope and source-absent I/P/M", row.prior, got)
		}
		want.advance = candidateWorkVector{}
		got, ok = providerHistoryCollectionRoleRequestControlPartsUpper(row.prior, 163840, 1<<20, false)
		if !ok || got != want {
			t.Fatal("final closure must not acquire a fresh advance envelope", row.prior, got)
		}
	}
}

func TestProviderHistoryRoleRequestControlRejectsInvalidAndLatePartialQuotes(t *testing.T) {
	for _, row := range [][3]uint64{{0, 0, 1 << 20}, {33, 0, 1 << 20}, {1, 163841, 1 << 20}, {1, 0, 0}, {1, 0, ^uint64(0)}} {
		if got, ok := providerHistoryPlannedNextTableControlUpper(row[0], row[1], row[2]); ok || got != (providerHistoryPlannedNextTableControlCost{}) {
			t.Fatal("invalid original member/effect/pack bounds returned partial work", row)
		}
	}
	for _, row := range []struct {
		prior, effects, maximum uint64
		advancing               bool
	}{{349525, 0, 1 << 20, false}, {349524, 0, 1 << 20, true}, {^uint64(0), 0, 1 << 20, false}, {0, 163841, 1 << 20, false}, {0, 0, 0, false}} {
		if got, ok := providerHistoryCollectionRoleRequestControlPartsUpper(row.prior, row.effects, row.maximum, row.advancing); ok || got != (providerHistoryRoleRequestControlParts{}) {
			t.Fatal("invalid selected role profile retained earlier components", row)
		}
	}
	if _, ok := providerHistoryCollectionRoleRequestControlPartsUpper(349524, 0, 1<<20, false); !ok {
		t.Fatal("last legal original group without own advancement")
	}
	for _, row := range []struct {
		pair  providerHistoryControlPair
		calls uint64
	}{
		{providerHistoryControlPair{candidateWorkVector{10: 1}, candidateWorkVector{10: 1}}, 0},
		{providerHistoryControlPair{candidateWorkVector{10: 2}, candidateWorkVector{10: 1}}, 1},
		{providerHistoryControlPair{candidateWorkVector{10: 2}, candidateWorkVector{2: 7, 10: 3}}, ^uint64(0)},
	} {
		if got, ok := row.pair.pointwiseUpper(row.calls); ok || got != (candidateWorkVector{}) {
			t.Fatal("empty/reversed/late-overflow pair retained a partial vector", row, got)
		}
	}
	unit := providerHistoryControlPair{candidateWorkVector{10: 1}, candidateWorkVector{10: 1}}
	if got, ok := unit.pointwiseUpper(^uint64(0)); !ok || got != (candidateWorkVector{10: ^uint64(0)}) {
		t.Fatal("finite uint64 endpoint must not be rejected", got)
	}
	next, a := providerHistoryPlannedNextTableControlUpper(32, 0, 1<<20)
	record, b := providerHistoryPlannedRecordControlUpper(32)
	if !a || !b {
		t.Fatal("late refusal precursor")
	}
	// Generation/equality/C finish first; the late S component then overflows.
	// No earlier decoded count or component may survive that complete failure.
	record.normal.settlement[10], record.envelope.settlement[10] = ^uint64(0), ^uint64(0)
	if got, ok := (providerHistoryRoleRequestControlCalls{1, 1, 1, 2}).upper(next, record); ok || got != (candidateWorkVector{}) {
		t.Fatal("late record overflow leaked prior generated/table/C work", got)
	}
}
