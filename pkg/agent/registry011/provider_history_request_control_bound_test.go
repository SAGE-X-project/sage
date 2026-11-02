package registry011

import "testing"

func historyRequestControlMethods(t *testing.T) []*providerIntegrationRequest {
	t.Helper()
	h := admissionTestDigest("a")
	var requests []*providerIntegrationRequest
	for _, row := range []struct {
		method string
		params any
	}{
		{"eth_chainId", []any{}},
		{"web3_clientVersion", []any{}},
		{"eth_getCode", []any{SepoliaRegistryAddress, "0x1"}},
		{"eth_call", []any{map[string]string{"to": SepoliaRegistryAddress, "data": "0x"}, "0x1"}},
		{"debug_getRawBlock", []any{h}},
		{"eth_getBlockReceipts", []any{h}},
		{"debug_getRawReceipts", []any{h}},
		{"eth_getBlockByHash", []any{h, false}},
		{"eth_getBlockByNumber", []any{"finalized", true}},
		{"debug_traceTransaction", []any{h, map[string]any{"tracer": "stateGasTracer", "reexec": 0}}},
		{"debug_traceBlockByHash", []any{h, map[string]string{"tracer": "callTracer"}}},
	} {
		requests = append(requests, historyGroupRequest(t, row.method, row.params))
	}
	return requests
}

func TestProviderHistoryRequestControlMatchesIndependentRows(t *testing.T) {
	for _, row := range []struct{ members, tableDecoded, tableLogical, packDecoded, packLogical uint64 }{
		{1, 24576, 4832449951, 114692, 14098297187},
		{2, 49152, 5369801498, 163844, 15307235555},
		{32, 786432, 21490347908, 1638404, 51575386595},
	} {
		got, ok := providerHistoryRequestControlUpper(row.members, 1<<20)
		want := providerHistoryRequestControlCost{
			candidateWorkVector{2: 24576, 10: 671586687}, candidateWorkVector{10: 4295098404},
			candidateWorkVector{2: 24576, 10: 537351547}, candidateWorkVector{2: row.tableDecoded, 10: row.tableLogical},
			candidateWorkVector{10: 2812518}, candidateWorkVector{2: row.packDecoded, 10: row.packLogical},
		}
		if !ok || got != want {
			t.Fatal("independent original/table/complete-pack cost", row, got)
		}
	}
	for _, row := range []struct{ groups, requests, decoded, logical uint64 }{
		{1, 1, 27754536, 1195153845351},
		{1, 32, 55181352, 1836451077572},
		{2, 2, 116523144, 4902919858264},
		{2, 3, 118784136, 4955577846209},
		{2, 33, 186613896, 6535317484559},
		{3, 96, 476578080, 16020490917624},
		{32, 32, 31151124864, 1284500558161984},
	} {
		got, ok := providerHistoryCollectionRepeatedRequestControlUpper(row.groups, row.requests, 1<<20)
		if !ok || got != (candidateWorkVector{2: row.decoded, 10: row.logical}) {
			t.Fatal("independent expanded request/table/outside-record composition", row, got)
		}
	}
}

func TestProviderHistoryRequestControlFundsEveryClosedOriginal(t *testing.T) {
	upper, ok := providerHistoryRequestControlUpper(1, 1<<20)
	if !ok {
		t.Fatal("finite original request quote")
	}
	for _, original := range historyRequestControlMethods(t) {
		before := *original
		meter := providerHistoryContentUnitMeter(upper.original)
		got, err := providerHistoryRequestOriginalObserved(providerReplayBytes(original.wire, meter), meter)
		if err != nil || got == nil || *got != before || *original != before || meter.failed || meter.actual[2] == 0 {
			t.Fatal("closed original HTTP reconstruction lost bytes or exceeded quote", original.method, err)
		}
		for coordinate, actual := range meter.actual {
			if actual > upper.original[coordinate] {
				t.Fatal("original request escaped coordinate", original.method, coordinate)
			}
		}
	}
}

func TestProviderHistoryRequestControlFundsTablesAndCompleteOriginalPacks(t *testing.T) {
	// These observers have no real store/Source. Legacy phase-copy omission is
	// not V4 evidence; the quote still includes the actual V4 copy in full.
	bounds, capture := historyGroupBounds()
	methods := historyRequestControlMethods(t)
	for _, count := range []int{1, providerHistoryGroupMembers} {
		requests := make([]*providerIntegrationRequest, count)
		before := make([]providerIntegrationRequest, count)
		inputs := make([]providerHistoryOriginalInput, count)
		for i := range requests {
			requests[i] = methods[i%len(methods)]
			before[i] = *requests[i]
			inputs[i] = providerHistoryOriginalInput{"request", requests[i].wire}
		}
		upper, ok := providerHistoryRequestControlUpper(uint64(count), 1<<20)
		if !ok {
			t.Fatal("finite table/pack quote")
		}
		built := providerHistoryContentUnitMeter(upper.tableBuilt)
		table, err := newProviderHistoryGroupTableObserved(admissionTestDigest("b"), "content", 1, requests, bounds, capture, built)
		if err != nil || table == nil || built.failed {
			t.Fatal("normal selected table construction", count, err)
		}
		pack, err := newProviderHistoryOriginalPackObserved(1, table.digest, "requests", 0, count, inputs, 1<<20, nil)
		if err != nil {
			t.Fatal("normal retained opaque request-pack precursor", err)
		}
		beforePack, beforeTable := *pack, *table
		parsed := providerHistoryContentUnitMeter(upper.packParsed)
		gotPack, gotTable, err := parseProviderHistoryRequestPackObserved(providerReplayBytes(pack.raw, parsed), pack.digest, 1,
			table.program, table.purpose, table.digest, count, bounds, capture, 1<<20, parsed)
		if err != nil || gotPack == nil || gotTable == nil || *gotPack != beforePack || *gotTable != beforeTable || parsed.failed {
			t.Fatal("full original request/table parser unfunded", count, err)
		}
		equal := providerHistoryContentUnitMeter(upper.tableEqual)
		if !providerHistoryCollectionTableEqualObserved(table, gotTable, equal) || equal.failed || *pack != beforePack || *table != beforeTable {
			t.Fatal("independent full table equality changed originals", count)
		}
		for i, r := range requests {
			if *r != before[i] || table.members[i].request != before[i] {
				t.Fatal("table retained modified caller request", count, i)
			}
		}
		for _, row := range []struct {
			meter *providerWorkMeter
			upper candidateWorkVector
		}{{built, upper.tableBuilt}, {parsed, upper.packParsed}, {equal, upper.tableEqual}} {
			for coordinate, actual := range row.meter.actual {
				if actual > row.upper[coordinate] {
					t.Fatal("request/table/pack work escaped its disjoint row", count, coordinate)
				}
			}
		}
	}
}

func TestProviderHistoryRequestControlTopologyKeepsMemberReplaySeparate(t *testing.T) {
	for _, row := range []struct {
		groups, requests uint64
		want             providerHistoryRequestControlCalls
	}{
		{1, 1, providerHistoryRequestControlCalls{1, 10, 10, 26, 26, 15, 22, 16, 6, 22, 3}},
		{1, 32, providerHistoryRequestControlCalls{1, 10, 320, 26, 832, 15, 22, 16, 6, 22, 96}},
		{2, 3, providerHistoryRequestControlCalls{2, 34, 58, 94, 162, 58, 72, 60, 40, 86, 37}},
		{2, 33, providerHistoryRequestControlCalls{2, 34, 778, 94, 2202, 58, 72, 60, 40, 86, 547}},
	} {
		got, ok := providerHistoryCollectionRequestControlCallsBound(row.groups, row.requests)
		if !ok || got != row.want {
			t.Fatal("original outside-record/table schedule", row, got)
		}
	}
	// Independently enumerate normal current stages and each fresh archived
	// prefix. Member equivalents do not enter these parser/transition counts.
	for _, sizes := range [][]uint64{{1}, {32}, {1, 1}, {2, 1}, {32, 1}, {1, 32}, {32, 32, 32}} {
		var got providerHistoryRequestControlCalls
		var requests uint64
		for g, count := range sizes {
			requests += count
			got.packBuilds++
			got.tableBuilds += 2 // Both planning readers.
			got.tableMembers += 2 * count
			got.tableEquals++
			// T: 1 constructor/2 parsers/3 admits. C: 1 constructor,
			// 2 parsers/3 consumes, with3 more original T parses/admits.
			// Dispatch replays T/C twice. S replays T/C three times,
			// then constructs1, parses2 and reconstructs3 at settlement.
			for _, phase := range []struct{ tparse, cparse, sparse, admit, consume, settle uint64 }{
				{2, 0, 0, 3, 0, 0}, {3, 2, 0, 3, 3, 0}, {2, 2, 0, 2, 2, 0}, {3, 3, 2, 3, 3, 3},
			} {
				got.packParses += phase.tparse
				got.originals += phase.tparse * count
				got.tableBuilds += phase.tparse + phase.admit + phase.settle
				got.tableMembers += (phase.tparse + phase.admit + phase.settle) * count
				got.tableEquals += phase.admit + phase.settle
				got.ticketOps += phase.tparse + phase.admit
				got.consumptionOps += phase.cparse + phase.consume
				got.settlementOps += phase.sparse + phase.settle
				got.transitionHashes += phase.admit + phase.consume + phase.settle
				got.resultHashes += phase.settle * count
			}
			got.ticketOps++
			got.consumptionOps++
			got.settlementOps++
			for prior := range g {
				// Twelve normal full-prefix views and two advance views.
				for range 14 {
					got.packParses++
					got.originals += sizes[prior]
					got.tableBuilds += 3
					got.tableMembers += 3 * sizes[prior]
					got.tableEquals += 2
					got.ticketOps += 2
					got.consumptionOps += 2
					got.settlementOps += 2
					got.transitionHashes += 3
					got.resultHashes += sizes[prior]
				}
			}
		}
		upper, ok := providerHistoryCollectionRequestControlCallsBound(uint64(len(sizes)), requests)
		if !ok || got.packBuilds != upper.packBuilds || got.packParses != upper.packParses || got.tableBuilds != upper.tableBuilds ||
			got.tableEquals != upper.tableEquals || got.ticketOps != upper.ticketOps || got.consumptionOps != upper.consumptionOps ||
			got.settlementOps != upper.settlementOps || got.transitionHashes != upper.transitionHashes ||
			got.originals > upper.originals || got.tableMembers > upper.tableMembers || got.resultHashes > upper.resultHashes {
			t.Fatal("enumerated normal original schedule escaped upper", sizes, got, upper)
		}
	}
}

func TestProviderHistoryRequestControlKeepsGenuineDecodeShortagePrefixes(t *testing.T) {
	upper, ok := providerHistoryRequestControlUpper(2, 1<<20)
	if !ok {
		t.Fatal("finite original/pack refusal precursor")
	}
	r := historyGroupSimple(t)
	before := *r
	valid := providerHistoryContentUnitMeter(upper.original)
	if got, err := providerHistoryRequestOriginalObserved(providerReplayBytes(r.wire, valid), valid); err != nil || got == nil || *got != before || valid.failed || valid.actual[2] <= uint64(len(r.body)) {
		t.Fatal("normal body/params decode precursor", err)
	}
	short := upper.original
	short[2] = valid.actual[2] - 1
	m := providerHistoryContentUnitMeter(short)
	if got, err := providerHistoryRequestOriginalObserved(providerReplayBytes(r.wire, m), m); err == nil || got != nil || !m.failed || m.actual[2] != uint64(len(r.body)) || *r != before {
		t.Fatal("params decode shortage lost successful body prefix or leaked request", err, m.actual)
	}
	if m.event(providerCompareEvent, 0) {
		t.Fatal("failed original parser revived")
	}
	bounds, capture := historyGroupBounds()
	table, err := newProviderHistoryGroupTable(admissionTestDigest("b"), "content", 1, []*providerIntegrationRequest{r, r}, bounds, capture)
	if err != nil {
		t.Fatal("normal request table precursor", err)
	}
	pack, err := newProviderHistoryOriginalPackObserved(1, table.digest, "requests", 0, 2,
		[]providerHistoryOriginalInput{{"request", r.wire}, {"request", r.wire}}, 1<<20, nil)
	if err != nil {
		t.Fatal("normal request pack precursor", err)
	}
	beforePack, beforeTable := *pack, *table
	parse := func(meter *providerWorkMeter) (*providerHistoryOriginalPack, *providerHistoryGroupTable, error) {
		return parseProviderHistoryRequestPackObserved(providerReplayBytes(pack.raw, meter), pack.digest, 1,
			table.program, table.purpose, table.digest, 2, bounds, capture, 1<<20, meter)
	}
	full := providerHistoryContentUnitMeter(upper.packParsed)
	if p, checked, err := parse(full); err != nil || p == nil || checked == nil || full.failed || full.actual[2] <= valid.actual[2] {
		t.Fatal("normal complete request-pack precursor", err)
	}
	short = upper.packParsed
	short[2] = full.actual[2] - 1
	m = providerHistoryContentUnitMeter(short)
	if p, checked, err := parse(m); err == nil || p != nil || checked != nil || !m.failed || m.actual[2] <= valid.actual[2] || m.actual[2] >= full.actual[2] ||
		*pack != beforePack || *table != beforeTable || *r != before {
		t.Fatal("late table decode shortage returned partial originals or changed precursor", err, m.actual)
	}
	if m.event(providerCopyEvent, 0) {
		t.Fatal("failed complete-pack parser revived")
	}
}

func TestProviderHistoryRequestControlRefusesInvalidAndOverflowingQuotes(t *testing.T) {
	for _, row := range [][2]uint64{{0, 1 << 20}, {33, 1 << 20}, {1, 0}, {1, 43}, {1, (32 << 20) + 1}, {^uint64(0), ^uint64(0)}} {
		if got, ok := providerHistoryRequestControlUpper(row[0], row[1]); ok || got != (providerHistoryRequestControlCost{}) {
			t.Fatal("invalid request/table quote returned partial cost", row, got)
		}
	}
	for _, row := range [][2]uint64{{0, 0}, {1, 0}, {1, 33}, {2, 1}, {349526, 349526}, {^uint64(0), ^uint64(0)}} {
		if got, ok := providerHistoryCollectionRequestControlCallsBound(row[0], row[1]); ok || got != (providerHistoryRequestControlCalls{}) {
			t.Fatal("invalid topology returned partial counts", row, got)
		}
		if got, ok := providerHistoryCollectionRepeatedRequestControlUpper(row[0], row[1], 1<<20); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid topology returned aggregate cost", row, got)
		}
	}
	if got, ok := providerHistoryCollectionRepeatedRequestControlUpper(1, 1, 43); ok || got != (candidateWorkVector{}) {
		t.Fatal("invalid pack maximum returned aggregate cost")
	}
	// The decoded row is finite (6333054385579462200), but logical work
	// exceeds uint64. It must not return that earlier partial decoded row.
	if got, ok := providerHistoryCollectionRepeatedRequestControlUpper(349525, 11184800, 1<<20); ok || got != (candidateWorkVector{}) {
		t.Fatal("late logical overflow returned partial aggregate", got)
	}
}

func TestProviderHistoryRoleBindingCountsAndPricesMatchIndependentRows(t *testing.T) {
	for _, row := range []struct {
		prior, past, members uint64
		calls                [4][3]uint64
		logical              [4]uint64
	}{
		{0, 0, 1, [4][3]uint64{{7, 13, 0}, {3, 9, 3}, {3, 9, 0}, {2, 6, 2}}, [4]uint64{2660, 1995, 1596, 1330}},
		{0, 0, 32, [4][3]uint64{{7, 13, 0}, {3, 9, 96}, {3, 9, 0}, {2, 6, 64}}, [4]uint64{2660, 14364, 1596, 9576}},
		{1, 32, 32, [4][3]uint64{{16, 40, 288}, {6, 18, 192}, {6, 18, 96}, {4, 12, 128}}, [4]uint64{45752, 28728, 15960, 19152}},
		{7, 224, 32, [4][3]uint64{{70, 202, 2016}, {24, 72, 768}, {24, 72, 672}, {16, 48, 512}}, [4]uint64{304304, 114912, 102144, 76608}},
		{9225, 295200, 32, [4][3]uint64{{83032, 249088, 2656800}, {27678, 83034, 885696}, {27678, 83034, 885600}, {18452, 55356, 590464}}, [4]uint64{397526360, 132522264, 132509496, 88348176}},
	} {
		var roles [4]providerHistoryRoleBindingCalls
		for i, r := range row.calls {
			roles[i] = providerHistoryRoleBindingCalls{r[0], r[1], r[2]}
			work, ok := roles[i].upper()
			if !ok || work != (candidateWorkVector{10: row.logical[i]}) {
				t.Fatal("independent fixed-width source comparison price", row.prior, i, work)
			}
		}
		wantCounts := providerHistoryRoleBindingCounts{roles[0], roles[1], roles[2], roles[3]}
		got, ok := providerHistoryCollectionRoleBindingCallsBound(row.prior, row.past, row.members, true)
		if !ok || got != wantCounts {
			t.Fatal("independent request-table/predecessor/result source cuts", row, got)
		}
		parts, ok := providerHistoryCollectionRoleBindingPartsUpper(row.prior, row.members, true)
		want := providerHistoryRoleBindingParts{before: candidateWorkVector{10: row.logical[0]}, closure: candidateWorkVector{10: max(row.logical[1], row.logical[2])}, advance: candidateWorkVector{10: row.logical[3]}}
		if !ok || parts != want {
			t.Fatal("uniform source-bound comparison parts changed other coordinates", row, parts)
		}
		want.advance = candidateWorkVector{}
		parts, ok = providerHistoryCollectionRoleBindingPartsUpper(row.prior, row.members, false)
		if !ok || parts != want {
			t.Fatal("final stage acquired an extra comparison scope", row, parts)
		}
	}
}

func TestProviderHistoryRoleBindingCountsConserveVariableOriginalGrammar(t *testing.T) {
	for groups := uint64(1); groups <= 64; groups++ {
		for pattern := uint64(0); pattern < 3; pattern++ {
			var requests, weighted uint64
			var total providerHistoryRoleBindingCalls
			for prior := uint64(0); prior < groups; prior++ {
				members := uint64(1)
				if pattern == 1 {
					members = 32
				} else if pattern == 2 {
					members = 1 + (19*prior+7*groups)%32
				}
				calls, ok := providerHistoryCollectionRoleBindingCallsBound(prior, requests, members, prior+1 < groups)
				if !ok {
					t.Fatal("complete finite original grammar precursor")
				}
				for _, role := range []providerHistoryRoleBindingCalls{calls.before, calls.successClosure, calls.advance} {
					total.packTables += role.packTables
					total.predecessors += role.predecessors
					total.results += role.results
				}
				if calls.successClosure.packTables != calls.failedClosure.packTables || calls.successClosure.predecessors != calls.failedClosure.predecessors || calls.successClosure.results-calls.failedClosure.results != 3*members {
					t.Fatal("failed S omits only its three current successful result-binding loops")
				}
				requests += members
				weighted += (groups - 1 - prior) * members
			}
			tri := groups * (groups - 1) / 2
			if total != (providerHistoryRoleBindingCalls{10*groups + 14*tri, 22*groups + 42*tri, 3*requests + 14*weighted}) {
				t.Fatal("independent three-site whole-grammar identities", groups, pattern, total)
			}
			old, ok := providerHistoryCollectionRequestControlCallsBound(groups, requests)
			if !ok || old.packParses != total.packTables || old.transitionHashes != total.predecessors || old.resultHashes < total.results {
				t.Fatal("original prospective binding counts underfunded actual source topology", groups, total)
			}
		}
	}
}

func TestProviderHistoryFixedHashComparisonKeepsOriginalRefusalPrefix(t *testing.T) {
	a, b := admissionTestDigest("a"), admissionTestDigest("b")
	if !lowerHash(a) || !lowerHash(b) || len(a) != 66 || len(b) != 66 {
		t.Fatal("two original fixed-width hash DATA precursors")
	}
	for _, row := range []struct {
		right string
		equal bool
	}{{a, true}, {b, false}} {
		m := providerHistoryContentUnitMeter(candidateWorkVector{10: 133})
		if got := providerReplayTextEqual(a, row.right, m); got != row.equal || m.failed || m.actual != (candidateWorkVector{10: 133}) {
			t.Fatal("equal and unequal fixed hashes each enter one complete original compare", got, m.actual)
		}
	}
	m := providerHistoryContentUnitMeter(candidateWorkVector{10: 132})
	if providerReplayTextEqual(a, a, m) || !m.failed || m.actual != (candidateWorkVector{}) || m.event(providerCopyEvent, 0) {
		t.Fatal("one-short original compare must refuse before entering its event")
	}
	m = providerHistoryContentUnitMeter(candidateWorkVector{10: 134})
	if !m.event(providerCopyEvent, 1) || providerReplayTextEqual(a, b, m) || !m.failed || m.actual != (candidateWorkVector{10: 2}) || m.event(providerCopyEvent, 0) {
		t.Fatal("refused comparison erased or renewed its already-entered original prefix", m.actual)
	}
}

func TestProviderHistoryBindingWidthsComeFromOriginalConstructors(t *testing.T) {
	ticket, consumption, inputs, roots, tlsBounds := historyResultDataFixture(t, 1)
	beforeT, beforeC := *ticket, *consumption
	result, err := parseProviderHistoryMemberOriginalsObserved(inputs[0].raw, inputs[1].raw, ticket, consumption, 0, roots, tlsBounds, nil)
	if err != nil {
		t.Fatal("original inert HTTP/TLS DATA parser precursor", err)
	}
	previous, inventory := admissionTestDigest("5"), admissionTestDigest("6")
	settlement, err := newProviderHistoryGroupSettlementObserved(ticket, consumption, previous, inventory, "complete", "", []string{result.digest}, nil)
	if err != nil || settlement == nil {
		t.Fatal("original typed successful S constructor precursor", err)
	}
	parsed, err := parseProviderHistoryGroupSettlementObserved([]byte(settlement.raw), ticket, consumption, previous, inventory, "complete", "", []string{result.digest}, nil)
	if err != nil || parsed == nil || *parsed != *settlement || *ticket != beforeT || *consumption != beforeC {
		t.Fatal("S parser must return the independently rebuilt expected original", err)
	}
	for _, hash := range []string{ticket.previous, consumption.previous, ticket.table.digest, ticket.digest, consumption.digest, result.digest, parsed.previous, parsed.results[0]} {
		if !lowerHash(hash) || len(hash) != 66 {
			t.Fatal("source width must precede the later ledger/boundary comparison")
		}
	}
	// The future factory must bind THESE earlier producer fences. This test
	// neither invokes a damaged private S nor treats a later guard as earlier.
	if s, err := newProviderHistoryGroupSettlementObserved(ticket, consumption, "invalid", inventory, "complete", "", []string{result.digest}, nil); err == nil || s != nil {
		t.Fatal("original S constructor accepted an unbounded previous identity")
	}
	if s, err := parseProviderHistoryGroupSettlementObserved([]byte(settlement.raw), ticket, consumption, "invalid", inventory, "complete", "", []string{result.digest}, nil); err == nil || s != nil {
		t.Fatal("raw S parser adopted its own previous instead of the original expected value")
	}
}

func TestProviderHistoryRoleBindingRejectsInvalidAndOverflowingQuotes(t *testing.T) {
	for _, row := range [][3]uint64{{349525, 349525, 1}, {^uint64(0), ^uint64(0), 1}, {0, 1, 1}, {1, 0, 1}, {1, 33, 1}, {0, 0, 0}, {0, 0, 33}} {
		if got, ok := providerHistoryCollectionRoleBindingCallsBound(row[0], row[1], row[2], false); ok || got != (providerHistoryRoleBindingCounts{}) {
			t.Fatal("invalid original stage/past/member domain returned partial bindings", row)
		}
	}
	for _, row := range [][2]uint64{{^uint64(0), 32}, {0, 0}, {0, 33}} {
		if got, ok := providerHistoryCollectionRoleBindingPartsUpper(row[0], row[1], false); ok || got != (providerHistoryRoleBindingParts{}) {
			t.Fatal("invalid selected original role retained earlier comparison work", row)
		}
	}
	if got, ok := providerHistoryCollectionRoleBindingPartsUpper(349524, 32, true); ok || got != (providerHistoryRoleBindingParts{}) {
		t.Fatal("out-of-domain own advance returned a partial binding scope")
	}
	if _, ok := providerHistoryCollectionRoleBindingPartsUpper(349524, 32, false); !ok {
		t.Fatal("last legal original final stage")
	}
	for _, row := range []providerHistoryRoleBindingCalls{{^uint64(0), 1, 0}, {^uint64(0)/133 + 1, 0, 0}} {
		if got, ok := row.upper(); ok || got != (candidateWorkVector{}) {
			t.Fatal("count-sum or final fixed-event product overflow returned partial work", row, got)
		}
	}
	maximum := ^uint64(0) / 133
	if got, ok := (providerHistoryRoleBindingCalls{maximum, 0, 0}).upper(); !ok || got != (candidateWorkVector{10: maximum * 133}) {
		t.Fatal("finite fixed-event product endpoint was incorrectly refused", got)
	}
	if got, ok := (providerHistoryRoleBindingCalls{}).upper(); !ok || got != (candidateWorkVector{}) {
		t.Fatal("source-proven absence acquired comparison work")
	}
}
