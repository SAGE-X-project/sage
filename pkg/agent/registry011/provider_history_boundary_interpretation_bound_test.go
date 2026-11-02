package registry011

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

func historyBoundaryRPCMeter(upper candidateWorkVector) *providerWorkMeter {
	return newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{work: upper, execution: upper})
}

func historyBoundaryRPCWithin(t *testing.T, m *providerWorkMeter, upper candidateWorkVector) {
	t.Helper()
	if m.failed || m.clocks != 0 {
		t.Fatal("inert original parser exceeded its quoted CONTROL row")
	}
	for coordinate, actual := range m.actual {
		if actual > upper[coordinate] {
			t.Fatal("original parser exceeded selected raw-width quote", coordinate, actual, upper)
		}
	}
}

func historyBoundaryABIData(t *testing.T, hasBool bool, values ...string) string {
	t.Helper()
	fields := len(values)
	if hasBool {
		fields++
	}
	head := make([]byte, 0, fields*32)
	if hasBool {
		head = append(head, rpcWord(1)...)
	}
	tail := []byte{}
	for _, value := range values {
		head = append(head, rpcWord(uint64(fields*32+len(tail)))...)
		tail = append(tail, rpcWord(uint64(len(value)))...)
		tail = append(tail, value...)
		tail = append(tail, make([]byte, (32-len(value)%32)%32)...)
	}
	data := append(head, tail...)
	raw, err := json.Marshal("0x" + hex.EncodeToString(data))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestProviderHistoryBoundaryHeaderAndDataCostsMatchIndependentRows(t *testing.T) {
	for _, row := range []struct {
		body             uint64
		normal, envelope candidateWorkVector
	}{
		{1, candidateWorkVector{2: 35, 10: 1316}, candidateWorkVector{2: 35, 10: 1316}},
		{32, candidateWorkVector{2: 128, 10: 4478}, candidateWorkVector{2: 128, 10: 4986}},
		{256, candidateWorkVector{2: 800, 10: 198686}, candidateWorkVector{2: 1024, 10: 264154}},
		{4096, candidateWorkVector{2: 12320, 10: 50345246}, candidateWorkVector{2: 16384, 10: 67130074}},
		{8 << 20, candidateWorkVector{2: 25165856, 10: 211106257700126}, candidateWorkVector{2: 33554432, 10: 281475018654426}},
	} {
		got, ok := providerHistoryBoundaryHeaderControlUpper(row.body)
		if !ok || got != (providerHistoryControlPair{row.normal, row.envelope}) {
			t.Fatal("independent original header codec/comparison/copy sums", row.body, got)
		}
	}
	for _, row := range []struct{ body, decoded, logical uint64 }{
		{1, 2, 8}, {256, 512, 131588}, {4096, 8192, 33562628}, {8192, 16384, 134234116},
	} {
		got, ok := providerHistoryBoundaryDataControlUpper(row.body)
		want := candidateWorkVector{2: row.decoded, 10: row.logical}
		if !ok || got != (providerHistoryControlPair{want, want}) {
			t.Fatal("independent original data JSON/hex/comparison/copy sums", row, got)
		}
	}
}

func TestProviderHistoryBoundaryABICostsKeepFullWordsAndRawEnvelopes(t *testing.T) {
	for _, row := range []struct {
		body             uint64
		flag             bool
		limits           []uint64
		normal, envelope candidateWorkVector
	}{
		{1024, false, []uint64{128}, candidateWorkVector{2: 1472, 10: 1199688}, candidateWorkVector{2: 2112, 10: 2101446}},
		{4096, true, []uint64{65536}, candidateWorkVector{2: 8256, 10: 33566853}, candidateWorkVector{2: 8256, 10: 33566853}},
		{65536, false, []uint64{262144}, candidateWorkVector{2: 131136, 10: 8590100550}, candidateWorkVector{2: 131136, 10: 8590100550}},
		{131072, false, []uint64{262144, 262144}, candidateWorkVector{2: 262272, 10: 34360070280}, candidateWorkVector{2: 262272, 10: 34360070280}},
		{8 << 20, false, []uint64{262144, 262144}, candidateWorkVector{2: 9437568, 10: 71468802707850}, candidateWorkVector{2: 16777344, 10: 140737505661064}},
	} {
		got, ok := providerHistoryBoundaryABIControlUpper(row.body, row.flag, row.limits...)
		if !ok || got != (providerHistoryControlPair{row.normal, row.envelope}) {
			t.Fatal("independent full codec32/padding/value-copy row", row, got)
		}
	}
}

func TestProviderHistoryBoundaryOriginalParsersAcceptOrdinaryRawVariants(t *testing.T) {
	want := FinalizedHeader{admissionTestDigest("a"), admissionTestDigest("b"), 1, 2}
	base := `{"timestamp":"0x2","hash":"` + want.Hash + `","number":"0x1","parentHash":"` + want.ParentHash + `"}`
	header, headerOK := providerHistoryBoundaryHeaderControlUpper(1024)
	if !headerOK {
		t.Fatal("selected original header allowance")
	}
	for _, raw := range []string{base, " \n" + base + "\t ", `{"ordinary":{"label":"demo"},` + base[1:], strings.Replace(base, `"hash":"0x`, `"hash":"\u0030x`, 1)} {
		m := historyBoundaryRPCMeter(header.envelope)
		got, err := providerHistoryRPCHeaderObserved(providerReplayBytes(raw, m), RPCResponseLimits{1024, 16}, m)
		if err != nil || got != want {
			t.Fatal("ordinary field order, whitespace, unknown field and valid escape", err)
		}
		historyBoundaryRPCWithin(t, m, header.normal)
	}
	for _, row := range []struct {
		flag   bool
		values []string
		limits []uint64
	}{
		{false, []string{"demo"}, []uint64{128}},
		{true, []string{"record"}, []uint64{65536}},
		{false, []string{"grants"}, []uint64{262144}},
		{false, []string{"entry", "command"}, []uint64{262144, 262144}},
	} {
		base := historyBoundaryABIData(t, row.flag, row.values...)
		pair, ok := providerHistoryBoundaryABIControlUpper(4096, row.flag, row.limits...)
		if !ok {
			t.Fatal("selected original ABI allowance")
		}
		for _, raw := range []string{base, " \n" + base + "\t ", strings.Replace(base, `"0x`, `"\u0030x`, 1)} {
			m := historyBoundaryRPCMeter(pair.envelope)
			limits := make([]int, len(row.limits))
			for i, limit := range row.limits {
				limits[i] = int(limit)
			}
			values, flag, err := rpcABIObserved(providerReplayBytes(raw, m), row.flag, m, limits...)
			if err != nil || flag != row.flag || len(values) != len(row.values) {
				t.Fatal("ordinary original ABI DATA", err)
			}
			for i, value := range values {
				if string(value) != row.values[i] {
					t.Fatal("original ABI value changed")
				}
			}
			historyBoundaryRPCWithin(t, m, pair.normal)
		}
	}
	data, ok := providerHistoryBoundaryDataControlUpper(256)
	if !ok {
		t.Fatal("selected original data allowance")
	}
	for _, raw := range []string{`"0x64656d6f"`, " \n\"0x64656d6f\"\t "} {
		m := historyBoundaryRPCMeter(data.envelope)
		value, err := rpcDataObserved(providerReplayBytes(raw, m), m)
		if err != nil || string(value) != "demo" {
			t.Fatal("ordinary original data", err)
		}
		historyBoundaryRPCWithin(t, m, data.normal)
	}
}

func historyBoundaryRPCMethods() providerMethodBounds {
	return providerMethodBounds{
		scalar: providerMethodLimit{256, 1024}, header: providerMethodLimit{4096, 8192},
		code: providerMethodLimit{8192, 16384}, state: providerMethodLimit{1 << 20, 2 << 20},
		block: providerMethodLimit{4096, 8192}, receipts: providerMethodLimit{4096, 8192},
		gas: providerMethodLimit{4096, 8192}, execution: providerMethodLimit{4096, 8192},
	}
}

func TestProviderHistoryBoundaryInterpretationMatchesIndependentPhaseRows(t *testing.T) {
	got, ok := providerHistoryBoundaryInterpretationUpper(historyBoundaryRPCMethods(), 1000)
	if !ok {
		t.Fatal("selected class/body and inclusive block limits")
	}
	for _, row := range []struct {
		got              providerHistoryControlPair
		normal, envelope candidateWorkVector
	}{
		{got.opening, candidateWorkVector{2: 1287168, 10: 1101358747945}, candidateWorkVector{2: 2343488, 10: 2200904846359}},
		{got.closing, candidateWorkVector{2: 1098816, 10: 1099714340124}, candidateWorkVector{2: 2163264, 10: 2199294008194}},
		{got.ancestry, candidateWorkVector{2: 394240, 10: 1611053129}, candidateWorkVector{2: 524288, 10: 2148167625}},
		{got.state, candidateWorkVector{2: 1179904, 10: 1116743113031}, candidateWorkVector{2: 2097216, 10: 2199025420421}},
		{got.grants, candidateWorkVector{2: 1573056, 10: 1374525606088}, candidateWorkVector{2: 2097216, 10: 2199025616966}},
		{got.history, candidateWorkVector{2: 67112960, 10: 70368828199168}, candidateWorkVector{2: 67112960, 10: 70368828199168}},
	} {
		if row.got != (providerHistoryControlPair{row.normal, row.envelope}) {
			t.Fatal("independent original phase event sums", row.got, row.normal, row.envelope)
		}
	}
}

func TestProviderHistoryBoundaryInterpretationFundsOriginalIdentityAndAncestry(t *testing.T) {
	// Existing recorded public responses are inert DATA, not an RPC call or
	// an authenticated current origin. Every measured call is the original
	// boundary function, excluding its separately priced table/copy/root.
	for _, phase := range []string{"open", "ancestry", "close"} {
		t.Run(phase, func(t *testing.T) {
			c, headers, methods := historyCollectionBoundaryFixture(t, 3, 2)
			if phase != "open" {
				c = historyCollectionBoundaryStep(t, c, headers, methods)
			}
			if phase == "close" {
				c = historyCollectionBoundaryStep(t, c, headers, methods)
			}
			if c.phase != phase {
				t.Fatal("original grammar precursor", phase, c.phase)
			}
			cost, ok := providerHistoryBoundaryInterpretationUpper(*methods, 3)
			if !ok {
				t.Fatal("original selected method maxima")
			}
			table := historyCollectionBoundaryPlan(t, c, methods)
			results := historyCollectionBoundaryResults(t, c, headers, table)
			before := append([]providerHistoryMemberResult(nil), results...)
			pair := cost.opening
			if phase == "ancestry" {
				pair = cost.ancestry
			} else if phase == "close" {
				pair = cost.closing
			}
			m := historyBoundaryRPCMeter(pair.envelope)
			var valid bool
			if phase == "ancestry" {
				valid = c.ancestryObserved(results, m)
			} else {
				valid = c.windowIdentityObserved(results, phase == "close", m)
			}
			if !valid {
				t.Fatal("normal original boundary interpretation", phase)
			}
			historyBoundaryRPCWithin(t, m, pair.normal)
			for i, result := range results {
				if result != before[i] {
					t.Fatal("original result bytes changed")
				}
			}
		})
	}
}

func TestProviderHistoryBoundaryRefusalRetainsOriginalPrefix(t *testing.T) {
	pair, ok := providerHistoryBoundaryHeaderControlUpper(256)
	if !ok {
		t.Fatal("finite selected original header quote")
	}
	// An ordinary missing-field response refuses within E, without inventing
	// a network responder, damaged private object or attack reproduction.
	m := historyBoundaryRPCMeter(pair.envelope)
	if _, err := providerHistoryRPCHeaderObserved(providerReplayBytes("{}", m), RPCResponseLimits{256, 16}, m); err == nil {
		t.Fatal("missing ordinary header fields were accepted")
	}
	historyBoundaryRPCWithin(t, m, pair.envelope)
	short := pair.envelope
	short[2] = 0
	m = historyBoundaryRPCMeter(short)
	if _, err := providerHistoryRPCHeaderObserved(providerReplayBytes("{}", m), RPCResponseLimits{256, 16}, m); err == nil || !m.failed || m.actual != (candidateWorkVector{10: 9}) {
		t.Fatal("copy and two UTF8/Unicode comparisons must survive first codec refusal")
	}
	prefix, upper := m.actual, m.upper
	if m.event(providerCopyEvent, 0) || m.actual != prefix || m.upper != upper {
		t.Fatal("failed original meter revived or renewed")
	}
	abi, abiOK := providerHistoryBoundaryABIControlUpper(256, false, 128)
	data, dataOK := providerHistoryBoundaryDataControlUpper(256)
	if !abiOK || !dataOK {
		t.Fatal("finite original result parser quotes")
	}
	m = historyBoundaryRPCMeter(abi.envelope)
	if _, _, err := rpcABIObserved(providerReplayBytes(`"0x00"`, m), false, m, 128); err == nil {
		t.Fatal("incomplete ordinary ABI tuple was accepted")
	}
	historyBoundaryRPCWithin(t, m, abi.envelope)
	m = historyBoundaryRPCMeter(data.envelope)
	if _, err := rpcDataObserved(providerReplayBytes(`"plain text"`, m), m); err == nil {
		t.Fatal("ordinary non-data string was accepted")
	}
	historyBoundaryRPCWithin(t, m, data.envelope)
}

func TestProviderHistoryBoundaryInterpretationRejectsInvalidLimitsAndOverflow(t *testing.T) {
	for _, maximum := range []uint64{0, (8 << 20) + 1, ^uint64(0)} {
		if got, ok := providerHistoryBoundaryHeaderControlUpper(maximum); ok || got != (providerHistoryControlPair{}) {
			t.Fatal("invalid header quote leaked a partial result")
		}
		if got, ok := providerHistoryBoundaryDataControlUpper(maximum); ok || got != (providerHistoryControlPair{}) {
			t.Fatal("invalid data quote leaked a partial result")
		}
		if got, ok := providerHistoryBoundaryABIControlUpper(maximum, false, 128); ok || got != (providerHistoryControlPair{}) {
			t.Fatal("invalid ABI quote leaked a partial result")
		}
	}
	for _, row := range []struct {
		flag   bool
		limits []uint64
	}{{false, nil}, {false, []uint64{1, 2, 3}}, {true, []uint64{1, 2}}, {false, []uint64{262145}}, {false, []uint64{^uint64(0)}}} {
		if got, ok := providerHistoryBoundaryABIControlUpper(4096, row.flag, row.limits...); ok || got != (providerHistoryControlPair{}) {
			t.Fatal("unsupported original Registry ABI shape leaked a partial result")
		}
	}
	for _, blocks := range []uint64{0, (1 << 20) + 1, ^uint64(0)} {
		if got, ok := providerHistoryBoundaryInterpretationUpper(historyBoundaryRPCMethods(), blocks); ok || got != (providerHistoryBoundaryInterpretationCost{}) {
			t.Fatal("invalid source interval leaked phase rows")
		}
	}
	methods := historyBoundaryRPCMethods()
	methods.header.body = 0
	if got, ok := providerHistoryBoundaryInterpretationUpper(methods, 1); ok || got != (providerHistoryBoundaryInterpretationCost{}) {
		t.Fatal("invalid selected method profile leaked phase rows")
	}
	pair, ok := providerHistoryBoundaryHeaderControlUpper(8 << 20)
	if !ok {
		t.Fatal("finite maximum source allowance")
	}
	for _, calls := range []uint64{0, ^uint64(0)} {
		if got, priced := pair.pointwiseUpper(calls); priced || got != (candidateWorkVector{}) {
			t.Fatal("invalid/overflowing repeated envelope leaked partial work")
		}
	}
}

func historyBoundaryRoleManifest(t *testing.T, blocks, window, content, effects, versions, present uint64) *providerHistoryCollectionManifest {
	t.Helper()
	// Explicit inert manifest inputs before any H0. No running source, quota,
	// native window or operating configuration is changed by these DATA rows.
	in := historyCollectionInput(t)
	in.blocks, in.maximumSelected, in.native = blocks, sepoliaDeploymentBlock+blocks-1, blocks
	in.windowBlocks, in.contentBlocks = window, content
	in.effects, in.versions, in.presentStops = effects, versions, present
	in.journal.Observation.MaxAncestryHeaders = 1 << 20
	in.journal.Replay.MaxBlocks, in.journal.Replay.MaxTransactions = 4096, 16384
	in.journal.Replay.History.MaxItems, in.journal.MaxBatches = 4096, 1<<20
	return historyCollectionManifest(t, in)
}

func TestProviderHistoryBoundaryPhaseCapsMatchOriginalGrammarRows(t *testing.T) {
	for _, row := range []struct {
		blocks, window, content, effects, versions, present uint64
		want                                                providerHistoryBoundaryPhaseCaps
	}{
		{1, 1, 1, 0, 0, 0, providerHistoryBoundaryPhaseCaps{1, 1, 0, 1, 0, 1, 0, 0, 0, 0, 4}},
		{13, 4, 2, 4, 4, 4, providerHistoryBoundaryPhaseCaps{7, 7, 3, 7, 4, 4, 4, 4, 12, 4, 40}},
		{71, 64, 10, 64, 64, 1, providerHistoryBoundaryPhaseCaps{4, 4, 4, 8, 9, 2, 1, 3, 70, 64, 35}},
		{38420, 4096, 10, 4096, 4096, 10, providerHistoryBoundaryPhaseCaps{20, 20, 1210, 3846, 3973, 10, 10, 137, 38419, 4096, 9226}},
	} {
		manifest := historyBoundaryRoleManifest(t, row.blocks, row.window, row.content, row.effects, row.versions, row.present)
		beforeInput, beforeRaw, beforeDigest, beforeCounts := manifest.input, manifest.raw, manifest.digest, manifest.counts
		got, ok := providerHistoryCollectionBoundaryPhaseCapsBound(manifest)
		if !ok || got != row.want || got.groups != manifest.counts.groups || !manifest.validSnapshot() ||
			manifest.input != beforeInput || manifest.raw != beforeRaw || manifest.digest != beforeDigest || manifest.counts != beforeCounts {
			t.Fatal("independent original group/shared-item caps or immutable manifest changed", got, row.want)
		}
	}
}

func TestProviderHistoryBoundaryTiersRetainSharedItemsAcrossSplitWindows(t *testing.T) {
	manifest := historyBoundaryRoleManifest(t, 71, 64, 10, 64, 64, 1)
	store, _ := providerFrontierBoundsFixture()
	model, ok := providerHistoryCollectionBoundaryRoleModelUpper(manifest, *store.methods)
	if !ok || !model.valid() || model.groups != 35 || model.delta != (candidateWorkVector{2: 523264, 10: 8590964608}) {
		t.Fatal("original finite source model and complete-phase sibling envelope")
	}
	// Ancestry70 is32+32+6, despite four possible split groups. History64
	// is32+32, despite three possible split groups. Empty marginal groups
	// retain the original ancestry slot-copy overhead and all group positions.
	wantCounts := [11]uint64{4, 4, 2, 1, 1, 2, 1, 2, 0, 1, 17}
	for i, tier := range model.tiers {
		if tier.count != wantCounts[i] {
			t.Fatal("split windows invented additional raw header/history items", i, tier.count, wantCounts[i])
		}
	}
	if model.tiers[3].work != (candidateWorkVector{2: 295104, 10: 4832141850}) ||
		model.tiers[4].work != (candidateWorkVector{10: 72}) ||
		model.tiers[7].work != (candidateWorkVector{2: 4198400, 10: 274883285248}) ||
		model.tiers[9].work != (candidateWorkVector{}) || model.tiers[10].work != (candidateWorkVector{}) {
		t.Fatal("independent partial/shared-item tier prices")
	}
	total, _, totalOK := model.rankedUpper(model.groups, false)
	if !totalOK || total != (candidateWorkVector{2: 20773504, 10: 1132303814678}) {
		t.Fatal("normal interpretation journey invented32 items per group", total)
	}
}

func TestProviderHistoryBoundaryRankedPrefixMatchesIndependentAllocation(t *testing.T) {
	// This finite arithmetic subject is not an original manifest, live source
	// or permission. Enumerate allocations independently of the tier selector.
	model := providerHistoryBoundaryRoleModel{groups: 10}
	model.tiers[0] = providerHistoryBoundaryNormalTier{2, candidateWorkVector{2: 20, 10: 1}}
	model.tiers[1] = providerHistoryBoundaryNormalTier{2, candidateWorkVector{2: 3, 10: 50}}
	model.tiers[2] = providerHistoryBoundaryNormalTier{2, candidateWorkVector{2: 9, 10: 4}}
	model.tiers[10].count = 4
	before := model
	var previousSum candidateWorkVector
	for count := uint64(0); count <= model.groups; count++ {
		var want candidateWorkVector
		for x := uint64(0); x <= 2; x++ {
			for y := uint64(0); y <= 2; y++ {
				for z := uint64(0); z <= 2; z++ {
					if x+y+z <= count {
						want[2] = max(want[2], 20*x+3*y+9*z)
						want[10] = max(want[10], x+50*y+4*z)
					}
				}
			}
		}
		normal, prefixes, ok := model.rankedUpper(count, true)
		if !ok || normal != want || prefixes != previousSum || model != before {
			t.Fatal("independent relaxed allocation/weighted prefix or immutable model", count, normal, want, prefixes, previousSum)
		}
		for coordinate := range previousSum {
			previousSum[coordinate] += want[coordinate]
		}
	}
}

func TestProviderHistoryBoundaryRoleAllowancesMatchIndependentRows(t *testing.T) {
	manifest := historyBoundaryRoleManifest(t, 71, 64, 10, 64, 64, 1)
	store, _ := providerFrontierBoundsFixture()
	model, ok := providerHistoryCollectionBoundaryRoleModelUpper(manifest, *store.methods)
	if !ok {
		t.Fatal("original selected model")
	}
	before := model
	for _, row := range []struct {
		prior                         uint64
		advancing                     bool
		before, success, failed, ownA candidateWorkVector
	}{
		{0, true, candidateWorkVector{}, candidateWorkVector{2: 13118464, 10: 833240820352}, candidateWorkVector{}, candidateWorkVector{2: 8920064, 10: 558357535104}},
		{1, true, candidateWorkVector{2: 38308864, 10: 2482540531840}, candidateWorkVector{2: 25713664, 10: 1657890676096}, candidateWorkVector{2: 13118464, 10: 833240820352}, candidateWorkVector{2: 17316864, 10: 1108124105600}},
		{7, true, candidateWorkVector{2: 157591552, 10: 9421286917420}, candidateWorkVector{2: 57601024, 10: 3223470550876}, candidateWorkVector{2: 52879360, 10: 3146156282212}, candidateWorkVector{2: 38575104, 10: 2151844022120}},
		{34, false, candidateWorkVector{2: 187484800, 10: 10199325296710}, candidateWorkVector{2: 62843776, 10: 3405502408642}, candidateWorkVector{2: 62843776, 10: 3405502408642}, candidateWorkVector{}},
	} {
		got, priced := model.roleUpper(row.prior, row.advancing)
		want := providerHistoryBoundaryRoleCost{before: row.before, successClosure: row.success, failedClosure: row.failed, closure: row.success, advance: row.ownA}
		if !priced || got != want || model != before {
			t.Fatal("independent9/3/3/2 roles, separate Cs/Cf and own old-T advance", row.prior, got, want)
		}
		if row.advancing {
			final, finalOK := model.roleUpper(row.prior, false)
			got.advance = candidateWorkVector{}
			if !finalOK || final != got {
				t.Fatal("an actual earlier final stage must remove only its own A")
			}
		}
	}
}

func TestProviderHistoryBoundaryFutureSumMatchesOriginalRoleTopology(t *testing.T) {
	store, _ := providerFrontierBoundsFixture()
	for _, row := range []struct {
		blocks, window, content, effects, versions, present uint64
		want                                                candidateWorkVector
	}{
		{1, 1, 1, 0, 0, 0, candidateWorkVector{2: 95907840, 10: 5738866279720}},
		{13, 4, 2, 4, 4, 4, candidateWorkVector{2: 8190773568, 10: 477881741930739}},
		{71, 64, 10, 64, 64, 1, candidateWorkVector{2: 9026750336, 10: 502149187897860}},
		{38420, 4096, 10, 4096, 4096, 10, candidateWorkVector{2: 298838437306720, 10: 8520045711850852781}},
	} {
		manifest := historyBoundaryRoleManifest(t, row.blocks, row.window, row.content, row.effects, row.versions, row.present)
		model, ok := providerHistoryCollectionBoundaryRoleModelUpper(manifest, *store.methods)
		if !ok {
			t.Fatal("finite source capacities")
		}
		got, priced := model.futureUpper()
		if !priced || got != row.want {
			t.Fatal("independent expanded-source future sum", row.blocks, got, row.want)
		}
		if model.groups <= 40 {
			var sum candidateWorkVector
			for prior := uint64(0); prior < model.groups; prior++ {
				stage, stageOK := model.roleUpper(prior, prior+1 < model.groups)
				if !stageOK {
					t.Fatal("each source T role envelope must remain finite")
				}
				for coordinate := range sum {
					sum[coordinate] += stage.before[coordinate] + stage.closure[coordinate] + stage.advance[coordinate]
				}
			}
			if sum != got {
				t.Fatal("closed future differs from irrevocable sum of complete original T role allowances", sum, got)
			}
		}
	}
}

func TestProviderHistoryBoundaryPhaseCapsCoverOriginalSplitAncestry(t *testing.T) {
	for _, row := range []struct{ blocks, width uint64 }{{1, 1}, {13, 4}, {71, 64}} {
		c, headers, methods := historyCollectionBoundaryFixture(t, row.blocks, row.width)
		manifest := historyCollectionManifest(t, c.selection.input)
		caps, capsOK := providerHistoryCollectionBoundaryPhaseCapsBound(manifest)
		model, modelOK := providerHistoryCollectionBoundaryRoleModelUpper(manifest, *methods)
		if !capsOK || !modelOK {
			t.Fatal("original fixture source capacities")
		}
		var opened, closed, groups, items uint64
		for c.mode == "ancestry" {
			table := historyCollectionBoundaryPlan(t, c, methods)
			switch c.phase {
			case "open":
				opened++
			case "close":
				closed++
			case "ancestry":
				groups++
				items += uint64(table.count)
			default:
				t.Fatal("unexpected original ancestry grammar")
			}
			c = historyCollectionBoundaryStep(t, c, headers, methods)
		}
		if items != row.blocks-1 || items != caps.ancestryItems || groups > caps.ancestry || opened > caps.opening || closed > caps.closing {
			t.Fatal("actual original split windows exceeded shared count caps", row, opened, closed, groups, items, caps)
		}
		if _, _, ok := model.rankedUpper(opened+closed+groups, false); !ok {
			t.Fatal("bounded original ancestry prefix was omitted")
		}
	}
}

func TestProviderHistoryBoundaryRoleModelRefusesInvalidSourceAndLateOverflow(t *testing.T) {
	store, _ := providerFrontierBoundsFixture()
	if got, ok := providerHistoryCollectionBoundaryPhaseCapsBound(nil); ok || got != (providerHistoryBoundaryPhaseCaps{}) {
		t.Fatal("nil selected manifest produced source counts")
	}
	if got, ok := providerHistoryCollectionBoundaryRoleModelUpper(nil, *store.methods); ok || got != (providerHistoryBoundaryRoleModel{}) {
		t.Fatal("nil selected manifest produced a role model")
	}
	manifest := historyBoundaryRoleManifest(t, 13, 4, 2, 4, 4, 4)
	manifest.input.blocks++
	if got, ok := providerHistoryCollectionBoundaryPhaseCapsBound(manifest); ok || got != (providerHistoryBoundaryPhaseCaps{}) {
		t.Fatal("changed unsealed source input supplied counts")
	}
	manifest.input.blocks--
	methods := *store.methods
	methods.state.body = manifest.input.capture.BodyBytes + 1
	methods.state.wire = manifest.input.capture.WireBytes
	if got, ok := providerHistoryCollectionBoundaryRoleModelUpper(manifest, methods); ok || got != (providerHistoryBoundaryRoleModel{}) {
		t.Fatal("unselected wider method class supplied a model")
	}
	tooMany := historyBoundaryRoleManifest(t, 100000, 1, 1, 0, 0, 0)
	if got, ok := providerHistoryCollectionBoundaryPhaseCapsBound(tooMany); ok || got != (providerHistoryBoundaryPhaseCaps{}) {
		t.Fatal("source group count exceeded the original stage domain")
	}
	model, ok := providerHistoryCollectionBoundaryRoleModelUpper(manifest, *store.methods)
	if !ok {
		t.Fatal("unchanged selected source must still be usable")
	}
	for _, prior := range []uint64{model.groups, ^uint64(0)} {
		if got, priced := model.roleUpper(prior, false); priced || got != (providerHistoryBoundaryRoleCost{}) {
			t.Fatal("invalid original ordinal leaked partial role work")
		}
	}
	if got, priced := model.roleUpper(model.groups-1, true); priced || got != (providerHistoryBoundaryRoleCost{}) {
		t.Fatal("final original stage acquired an own advance")
	}
	if first, second, priced := model.rankedUpper(model.groups+1, true); priced || first != (candidateWorkVector{}) || second != (candidateWorkVector{}) {
		t.Fatal("invalid prefix leaked partial work")
	}
	bad := providerHistoryBoundaryRoleModel{groups: 1}
	bad.tiers[0].count, bad.tiers[1].count = ^uint64(0), 2
	if first, second, priced := bad.rankedUpper(1, false); priced || first != (candidateWorkVector{}) || second != (candidateWorkVector{}) {
		t.Fatal("overflowing tier counts supplied a prefix")
	}
	late := providerHistoryBoundaryRoleModel{groups: 3}
	late.tiers[0] = providerHistoryBoundaryNormalTier{3, candidateWorkVector{2: 1, 10: ^uint64(0) / 2}}
	if normal, weighted, finite := late.rankedUpper(1, false); !finite || normal != (candidateWorkVector{2: 1, 10: ^uint64(0) / 2}) || weighted != (candidateWorkVector{}) {
		t.Fatal("unused weighted future must not reject a finite source prefix")
	}
	if first, second, priced := late.rankedUpper(3, false); priced || first != (candidateWorkVector{}) || second != (candidateWorkVector{}) {
		t.Fatal("late logical product retained earlier decoded partial work")
	}
	if got, priced := late.roleUpper(0, false); priced || got != (providerHistoryBoundaryRoleCost{}) {
		t.Fatal("late role product retained partial decoded work")
	}
	if got, priced := late.futureUpper(); priced || got != (candidateWorkVector{}) {
		t.Fatal("late full future overflow retained partial work")
	}
	finalOnly := late
	finalOnly.tiers[0].work[10] = ^uint64(0) / 4
	if normal, prefixes, finite := finalOnly.rankedUpper(3, true); !finite || normal[10] != 3*(^uint64(0)/4) || prefixes[10] != normal[10] {
		t.Fatal("finite journey and weighted prefixes must precede the final sum")
	}
	if got, priced := finalOnly.futureUpper(); priced || got != (candidateWorkVector{}) {
		t.Fatal("final future multiplier overflow retained earlier decoded work")
	}
	one := providerHistoryBoundaryRoleModel{groups: 1, delta: candidateWorkVector{10: ^uint64(0)}}
	one.tiers[0] = providerHistoryBoundaryNormalTier{1, candidateWorkVector{2: 1, 10: 1}}
	if got, priced := one.roleUpper(0, false); priced || got != (providerHistoryBoundaryRoleCost{}) {
		t.Fatal("incoming phase-envelope overflow retained partial roles")
	}
	if got, priced := one.futureUpper(); priced || got != (candidateWorkVector{}) {
		t.Fatal("incoming phase-envelope overflow retained a partial future")
	}
}
