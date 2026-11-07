package registry011

import (
	"bytes"
	"encoding/json"
	"math/big"
	"reflect"
	"testing"
)

func historyCommandBoundWithin(t *testing.T, actual, upper candidateWorkVector) {
	t.Helper()
	for i, n := range actual {
		if n > upper[i] {
			t.Fatal("command stage exceeds its independently derived source bound", i, actual, upper)
		}
	}
}

func TestProviderHistoryCommandUpperMatchesIndependentWindowPolynomials(t *testing.T) {
	mp := uint64(reflect.TypeFor[map[string]any]().Size())
	mm := uint64(reflect.TypeFor[map[[32]byte]bool]().Size())
	s := uint64(reflect.TypeFor[string]().Size())
	v := uint64(reflect.TypeFor[any]().Size())
	r := uint64(reflect.TypeFor[reflect.Value]().Size())
	f := uint64(reflect.TypeFor[reflect.StructField]().Size())
	ak := uint64(reflect.TypeFor[replayABIKey]().Size())
	as := uint64(reflect.TypeFor[replayABIService]().Size())
	k := uint64(reflect.TypeFor[replayKey]().Size())
	ss := uint64(reflect.TypeFor[replayService]().Size())
	if max(mp, mm, s, v, r, f, ak, as, k, ss) > 4096 {
		t.Fatal("changed source type shapes require rederiving the literal oracle")
	}
	// Independent sum of three native trees, tuple conversion, generation,
	// strict consumer and claim preconditions. No pooled codec/headroom here.
	types := 1097*mp + mm + 4338*s + 5500*v + 1888*r + 944*f + 640*ak + 80*as + 256*k + 32*ss
	for _, tc := range []struct {
		g, requests, windows, logs, evidence uint64
		opens, slots, raw, poolD, poolC      uint64
		windowD, actualC, reservedC          uint64
		totalD, totalC                       uint64
	}{
		{1, 1, 1, 1, 1024, 15, 15, 9280, 18560, 12315933253, 94695,
			12391151981, 2486493645166, 1420425, 37297404677490},
		{2, 3, 2, 2, 4096, 58, 116, 30848, 61696, 41042846858, 213966,
			41193284314, 2515295777499, 12410028, 145887155094942},
		{1, 32, 3, 65536, 67108864, 45, 2949120, 608174080, 1216348160, 807137001668608, 6205931520,
			812066536226816, 814540638720001, 279266918400, 36654328742400045},
	} {
		q, ok := providerHistoryCommandUpper(providerHistoryCollectionSemanticLimits{
			Groups: tc.g, Requests: tc.requests, Blocks: tc.windows, Windows: tc.windows,
		}, tc.logs, tc.evidence)
		if !ok || q.windowOpens != tc.opens || q.commandSlots != tc.slots || q.rawPool != tc.raw ||
			q.linear != (candidateWorkVector{2: 76135, 10: types + 75218728}) ||
			q.pool != (candidateWorkVector{2: tc.poolD, 10: tc.poolC}) ||
			q.headroom != (candidateWorkVector{10: 2474102493185}) ||
			q.actualWindow != (candidateWorkVector{2: tc.windowD, 10: tc.logs*types + tc.actualC}) ||
			q.window != (candidateWorkVector{2: tc.windowD, 10: tc.logs*types + tc.reservedC}) ||
			q.total != (candidateWorkVector{2: tc.totalD, 10: tc.slots*types + tc.totalC}) {
			t.Fatal("source-derived whole window/count/type polynomial changed", q, ok)
		}
	}
}

func TestProviderHistoryCommandUpperCoversAllOrdinaryDecodedCommandsAndClaim(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, step := range replayTestSteps(t) {
		c, accepted := replayParseCommand(replayTestRegistry, replayTestAgent, step.History.Command, contract)
		if !accepted {
			t.Fatal("ordinary strict command precursor")
		}
		want, accepted := replayDecodeCommand(replayTestRegistry, c.actor, c.calldata, contract)
		if !accepted || !reflect.DeepEqual(want, c) {
			t.Fatal("ordinary framed/canonical binary precursor")
		}
		method := contract.Methods[replayMethods[c.op]]
		values, err := method.Inputs.Unpack(c.calldata[4:])
		if err != nil {
			t.Fatal(err)
		}
		q, ok := providerHistoryCommandUpper(providerHistoryCollectionSemanticLimits{
			Groups: 1, Requests: 1, Blocks: 1, Windows: 1,
		}, 1, uint64(len(c.calldata)))
		if !ok {
			t.Fatal("finite DATA quote; not an original full replay profile")
		}
		m := providerHistoryContentUnitMeter(q.window)
		got, ok := replayDecodedCommandObserved(replayTestRegistry, c.actor, method.Name, values, c.calldata, contract, m)
		if !ok || !reflect.DeepEqual(got, c) || m.failed {
			t.Fatal("same original observer through generated and strict command", c.op, m.actual)
		}
		if c.op == "create" {
			digest, ok := replayClaimObserved(replayTestRegistry, got, m)
			if !ok || digest != replayClaim(replayTestRegistry, c) {
				t.Fatal("same observer through actual claim object/domain/hash")
			}
		}
		historyCommandBoundWithin(t, m.actual, q.actualWindow)
		if m.actual[10] >= q.window[10] || q.window[10]-q.actualWindow[10] != 2474102493185 {
			t.Fatal("unconsumed encoding headroom recorded as actual")
		}
		seen[c.op] = true
	}
	if len(seen) != 8 {
		t.Fatal("all eight ordinary methods required")
	}
}

func TestProviderHistoryCommandUpperSeparatesTupleAndGeneratedArgumentStages(t *testing.T) {
	q, ok := providerHistoryCommandUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 1, 64<<20)
	contract, _, err := replayABIs()
	if !ok || err != nil {
		t.Fatal("finite quote and fixed ABI prerequisite")
	}
	var upper candidateWorkVector
	for i := range upper {
		upper[i] = q.tuple[i] + q.generation[i]
	}
	for _, step := range replayTestSteps(t) {
		c, accepted := replayParseCommand(replayTestRegistry, replayTestAgent, step.History.Command, contract)
		if !accepted {
			t.Fatal("ordinary command precursor")
		}
		values, err := contract.Methods[replayMethods[c.op]].Inputs.Unpack(c.calldata[4:])
		if err != nil {
			t.Fatal(err)
		}
		m := providerHistoryContentUnitMeter(upper)
		args, ok := replayGeneratedArgumentsObserved(c.op, values, m)
		if !ok || !reflect.DeepEqual(args, c.args) || m.failed {
			t.Fatal("tuple conversion/generation escaped its disjoint stage bound", c.op)
		}
		historyCommandBoundWithin(t, m.actual, upper)
	}
}

func TestProviderHistoryCommandUpperNativeTreeCountsKeysAndRefusesLastStore(t *testing.T) {
	q, ok := providerHistoryCommandUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 1, 1024)
	if !ok {
		t.Fatal("finite native tree envelope")
	}
	value := map[string]any{"a": []any{"x", json.Number("7")}}
	mp := uint64(reflect.TypeFor[map[string]any]().Size())
	s := uint64(reflect.TypeFor[string]().Size())
	v := uint64(reflect.TypeFor[any]().Size())
	want := candidateWorkVector{2: 1, 10: mp + s + 5*v + 20}
	limit := replayCommandJSONLimit{nodes: 262144, bytes: 262144}
	m := providerHistoryContentUnitMeter(want)
	got, ok := replayCommandNativeObserved(value, 0, &limit, m)
	if !ok || m.actual != want || 262144-limit.nodes != 5 || 262144-limit.bytes != 3 {
		t.Fatal("map key must be a visited/string node; actual owned tree metric", m.actual, limit)
	}
	historyCommandBoundWithin(t, want, q.tree)
	got.(map[string]any)["a"].([]any)[0] = "changed clone"
	if value["a"].([]any)[0] != "x" {
		t.Fatal("tree clone aliases its original")
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{2: 1, 10: want[10] - 1})
	limit = replayCommandJSONLimit{nodes: 262144, bytes: 262144}
	got, ok = replayCommandNativeObserved(value, 0, &limit, short)
	prefix := candidateWorkVector{2: 1, 10: want[10] - (s + v + 1)}
	if ok || got != nil || !short.failed || short.actual != prefix {
		t.Fatal("late map store must preserve the entire original child prefix", short.actual, prefix)
	}
	before := short.actual
	if got, ok := replayCommandNativeObserved(value, 0, &limit, short); ok || got != nil || short.actual != before {
		t.Fatal("failed original tree observer resumed or refunded")
	}
}

func TestProviderHistoryCommandUpperChecksActualJSONWorkAndSharedInputRelation(t *testing.T) {
	q, ok := providerHistoryCommandUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 1, 64<<20)
	contract, _, err := replayABIs()
	if !ok || err != nil {
		t.Fatal("finite quote and fixed ABI prerequisite")
	}
	for _, step := range replayTestSteps(t) {
		c, accepted := replayParseCommand(replayTestRegistry, replayTestAgent, step.History.Command, contract)
		if !accepted {
			t.Fatal("ordinary strict command precursor")
		}
		d := json.NewDecoder(bytes.NewReader(c.raw))
		d.UseNumber()
		var value any
		if d.Decode(&value) != nil {
			t.Fatal("independent ordinary JSON input")
		}
		limit := replayCommandJSONLimit{nodes: 262144, bytes: 262144}
		tree := providerHistoryContentUnitMeter(q.tree)
		if _, ok := replayCommandNativeObserved(value, 0, &limit, tree); !ok {
			t.Fatal("ordinary native tree envelope")
		}
		n := uint64(len(c.raw))
		if uint64((262144-limit.nodes)+(262144-limit.bytes)) > uint64(len(c.calldata))+512 ||
			n > 6*uint64(len(c.calldata))+3136 {
			t.Fatal("normal command graph/raw bytes exceed original calldata relation")
		}
		m := providerHistoryExecutionUnitMeter()
		got := replayCommandJSONObserved(c.raw, m)
		want := tree.actual
		want[2] += 2 * n
		want[10] += 3*n*n + 4*n + 6
		if !reflect.DeepEqual(got, value) || m.failed || m.actual != want {
			t.Fatal("two actual decodes, one actual encode and byte checks; not preflight bytes", c.op, m.actual, want)
		}
	}
}

func TestProviderHistoryCommandUpperLateConsumerShortagesKeepOriginalPrefix(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	c := replayTestCommand(t, "activate", replayTestWallet, 1, map[string]any{})
	method := contract.Methods["activate"]
	values, err := method.Inputs.Unpack(c.calldata[4:])
	q, ok := providerHistoryCommandUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 1, uint64(len(c.calldata)))
	if err != nil || !ok {
		t.Fatal("ordinary decoded command and finite bound")
	}
	n := uint64(len(c.raw))
	for _, tc := range []struct{ decoded, prefix uint64 }{{2*n - 1, n}, {2*n + 39, 2 * n}} {
		upper := q.window
		upper[2] = tc.decoded
		m := providerHistoryContentUnitMeter(upper)
		got, ok := replayDecodedCommandObserved(replayTestRegistry, c.actor, method.Name, values, c.calldata, contract, m)
		if ok || !reflect.DeepEqual(got, replayCommand{}) || !m.failed || m.actual[2] != tc.prefix || m.actual[10] == 0 {
			t.Fatal("late consumer decoder/address codec shortage lost original generator prefix", tc, m.actual)
		}
		before := m.actual
		if got, ok := replayDecodedCommandObserved(replayTestRegistry, c.actor, method.Name, values, c.calldata, contract, m); ok || !reflect.DeepEqual(got, replayCommand{}) || m.actual != before {
			t.Fatal("failed consumer observer resumed or renewed its allowance")
		}
	}
}

func TestProviderHistoryCommandUpperInvalidDomainsReturnWholeZero(t *testing.T) {
	for _, mode := range []string{"groups-zero", "groups-wide", "requests-zero", "requests-small", "requests-wide", "blocks-zero", "blocks-wide", "windows-zero", "windows-wide", "versions-wide", "stops-wide", "logs-zero", "logs-wide", "evidence-zero", "evidence-wide"} {
		l := providerHistoryCollectionSemanticLimits{Groups: 2, Requests: 2, Blocks: 1, Windows: 1}
		logs, evidence := uint64(1), uint64(1024)
		switch mode {
		case "groups-zero":
			l.Groups = 0
		case "groups-wide":
			l.Groups = 349526
		case "requests-zero":
			l.Requests = 0
		case "requests-small":
			l.Requests = 1
		case "requests-wide":
			l.Requests = 65
		case "blocks-zero":
			l.Blocks = 0
		case "blocks-wide":
			l.Blocks = 1<<20 + 1
		case "windows-zero":
			l.Windows = 0
		case "windows-wide":
			l.Windows = 2
		case "versions-wide":
			l.Versions = 1<<20 + 1
		case "stops-wide":
			l.PresentStops = 2
		case "logs-zero":
			logs = 0
		case "logs-wide":
			logs = 65537
		case "evidence-zero":
			evidence = 0
		case "evidence-wide":
			evidence = 64<<20 + 1
		}
		if got, ok := providerHistoryCommandUpper(l, logs, evidence); ok || got != (providerHistoryCommandWork{}) {
			t.Fatal("invalid command source limits returned a partial quote", mode, got, ok)
		}
	}
}

func TestProviderHistoryCommandUpperLateLogicalOverflowReturnsWholeZero(t *testing.T) {
	// Valid scalar DATA, not a feasible source/factory profile. Decoded
	// total is finite; even the logical constant alone exceeds uint64.
	const decoded uint64 = 1127552338887750000
	logical, ok := new(big.Int).SetString("756332011593979693260000", 10)
	if !ok || logical.Cmp(new(big.Int).SetUint64(^uint64(0))) <= 0 || decoded == 0 {
		t.Fatal("independent finite-decoded/overflow-logical prerequisite")
	}
	got, ok := providerHistoryCommandUpper(providerHistoryCollectionSemanticLimits{Groups: 10000, Requests: 10000, Blocks: 10000, Windows: 10000}, 1, 64<<20)
	if ok || got != (providerHistoryCommandWork{}) {
		t.Fatal("late coordinate overflow leaked finite counts or partial vectors", got, ok)
	}
}
