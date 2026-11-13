package registry011

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func historySemanticWindowShapeTypes(t *testing.T) uint64 {
	t.Helper()
	mp := uint64(reflect.TypeFor[map[string]any]().Size())
	mm := uint64(reflect.TypeFor[map[[32]byte]bool]().Size())
	text := uint64(reflect.TypeFor[string]().Size())
	value := uint64(reflect.TypeFor[any]().Size())
	reflected := uint64(reflect.TypeFor[reflect.Value]().Size())
	field := uint64(reflect.TypeFor[reflect.StructField]().Size())
	ak := uint64(reflect.TypeFor[replayABIKey]().Size())
	as := uint64(reflect.TypeFor[replayABIService]().Size())
	key := uint64(reflect.TypeFor[replayKey]().Size())
	service := uint64(reflect.TypeFor[replayService]().Size())
	if max(mp, mm, text, value, reflected, field, ak, as, key, service) > 4096 {
		t.Fatal("changed source shapes require new portable polynomials")
	}
	return 1097*mp + mm + 4338*text + 5500*value + 1888*reflected + 944*field + 640*ak + 80*as + 256*key + 32*service
}

func TestProviderHistorySemanticWindowAtomsMatchIndependentRows(t *testing.T) {
	types := historySemanticWindowShapeTypes(t)
	argument := uint64(reflect.TypeFor[abi.Argument]().Size())
	method := uint64(reflect.TypeFor[abi.Method]().Size())
	claim := uint64(reflect.TypeFor[replayClaimID]().Size())
	state := uint64(reflect.TypeFor[replayClaimState]().Size())
	if max(argument, method, claim, state) > 4096 {
		t.Fatal("changed original ABI targets")
	}
	for _, row := range []struct{ logs, evidence, raw, commandDecoded, commandActual, commandReserved, abiDecoded, abiLogical, abiMethod, abiArgument uint64 }{
		{1, 1, 3142, 82419, 4193674979, 2478296168164, 12241, 663404, 8, 2192},
		{1, 1024, 9280, 94695, 12391151981, 2486493645166, 13264, 305110250, 8, 2192},
		{2, 4096, 30848, 213966, 41193284314, 2515295777499, 28576, 1219709396, 16, 4384},
		{256, 1048576, 7094272, 33679104, 9473951362304, 11948053855489, 4182016, 312151960064, 2048, 561152},
		{65536, 67108864, 608174080, 6205931520, 812066536226816, 814540638720001, 869269504, 19995705344000, 524288, 143654912},
	} {
		command, commandOK := providerHistoryCommandWindowUpper(row.logs, row.evidence)
		binary, binaryOK := providerHistoryABIWindowUpper(row.logs, row.evidence)
		wantABI := providerHistoryABIWork{
			fixed:    candidateWorkVector{2: 10015, 10: 56514744},
			selector: candidateWorkVector{10: 8*method + 80}, framing: candidateWorkVector{2: 12208, 10: 2176*argument + 364678},
			changed: candidateWorkVector{2: 32, 10: 16*argument + 1038}, mutation: candidateWorkVector{2: 12240, 10: 2192*argument + 8*method + 365796},
			commit:          candidateWorkVector{2: 40, 10: method + 2*argument + 2*claim + state + 3189},
			event:           candidateWorkVector{2: 12240, 10: 2192*argument + 8*method + 365796},
			canonicalWindow: candidateWorkVector{2: row.evidence, 10: 297602*row.evidence + 6*row.logs},
			window:          candidateWorkVector{2: row.abiDecoded, 10: row.abiLogical + row.abiMethod*method + row.abiArgument*argument},
			total:           candidateWorkVector{2: row.abiDecoded + 10015, 10: row.abiLogical + 56514744 + row.abiMethod*method + row.abiArgument*argument},
			initializations: 1, eventSlots: row.logs,
		}
		if !binaryOK || binary != wantABI || !commandOK || command.windowOpens != 1 || command.commandSlots != row.logs || command.rawPool != row.raw || command.linear != (candidateWorkVector{2: 76135, 10: types + 75218728}) || command.actualWindow != (candidateWorkVector{2: row.commandDecoded, 10: row.logs*types + row.commandActual}) || command.window != (candidateWorkVector{2: row.commandDecoded, 10: row.logs*types + row.commandReserved}) || command.total != command.window || command.headroom != (candidateWorkVector{10: 2474102493185}) {
			t.Fatal("independent ONEwindow atoms or legacy incoming changed", row, binary, command, binaryOK, commandOK)
		}
	}
}

func TestProviderHistorySemanticCommandShapeKeepsOnlyClosedAtoms(t *testing.T) {
	mp := uint64(reflect.TypeFor[map[string]any]().Size())
	mm := uint64(reflect.TypeFor[map[[32]byte]bool]().Size())
	s := uint64(reflect.TypeFor[string]().Size())
	v := uint64(reflect.TypeFor[any]().Size())
	r := uint64(reflect.TypeFor[reflect.Value]().Size())
	f := uint64(reflect.TypeFor[reflect.StructField]().Size())
	ak := uint64(reflect.TypeFor[replayABIKey]().Size())
	as := uint64(reflect.TypeFor[replayABIService]().Size())
	key := uint64(reflect.TypeFor[replayKey]().Size())
	service := uint64(reflect.TypeFor[replayService]().Size())
	want := providerHistoryCommandWork{
		tree:       candidateWorkVector{2: 2560, 10: 274*mp + 1083*s + 1371*v + 247818},
		tuple:      candidateWorkVector{10: 1888*r + 944*f + 384*ak + 48*as + 162402},
		generation: candidateWorkVector{10: 274*mp + 1083*s + 1371*v + 2001596},
		consumer:   candidateWorkVector{2: 68455, 10: mm + 10*v + 256*key + 256*ak + 32*service + 32*as + 72310923},
		claim:      candidateWorkVector{10: mp + 6*s + 6*v + 353},
		linear:     candidateWorkVector{2: 76135, 10: historySemanticWindowShapeTypes(t) + 75218728},
	}
	if got, ok := providerHistoryCommandShapeUpper(); !ok || got != want {
		t.Fatal("schema atom extraction gained window counts or omitted original parts", got, ok)
	}
}

func TestProviderHistorySemanticCommandRefusalKeepsProducerBeforeConsumerLimit(t *testing.T) {
	// Integer-only maximal graph descriptor. No corresponding oversized graph,
	// payload, peer exchange, resource workload or actual Source is constructed.
	const nodes, strings, numbers uint64 = 2311, 94431, 2560
	if 6*(nodes+strings+numbers)+64 != 595876 || 595876 <= 262144 {
		t.Fatal("Number text omitted before the later consumer guard")
	}
	types := historySemanticWindowShapeTypes(t)
	got, ok := providerHistoryCommandRefusalUpper()
	want := providerHistoryControlPair{
		normal:   candidateWorkVector{2: 1267887, 10: types + 1797563950709},
		envelope: candidateWorkVector{2: 1267887, 10: types + 4271666443894},
	}
	if !ok || got != want || got.envelope[10]-got.normal[10] != 2474102493185 {
		t.Fatal("failed command prefix used the later successful-output ceiling", got, want, ok)
	}
}

func TestProviderHistorySemanticWindowWrappersPreserveOriginalWholeCounts(t *testing.T) {
	for _, row := range []struct{ groups, requests, windows, opens, pairs uint64 }{
		{1, 1, 1, 15, 17}, {2, 3, 2, 58, 64}, {1, 32, 3, 45, 109}, {3, 33, 3, 129, 195},
	} {
		const logs, evidence uint64 = 2, 4096
		limits := providerHistoryCollectionSemanticLimits{Groups: row.groups, Requests: row.requests, Blocks: row.windows, Windows: row.windows}
		command, ok := providerHistoryCommandWindowUpper(logs, evidence)
		binary, binaryOK := providerHistoryABIWindowUpper(logs, evidence)
		if !ok || !binaryOK {
			t.Fatal("numeric atom prerequisite")
		}
		wantCommand := command
		wantCommand.windowOpens = row.opens
		wantCommand.commandSlots = row.opens * logs
		wantABI := binary
		wantABI.initializations = row.pairs
		wantABI.eventSlots = row.opens * logs
		for c := range command.total {
			wantCommand.total[c] = row.opens * command.window[c]
			wantABI.total[c] = row.pairs*binary.fixed[c] + row.opens*binary.window[c]
		}
		gotCommand, commandOK := providerHistoryCommandUpper(limits, logs, evidence)
		gotABI, abiOK := providerHistoryABIUpper(limits, logs, evidence)
		if !commandOK || !abiOK || gotCommand != wantCommand || gotABI != wantABI {
			t.Fatal("original whole-program wrapper changed", row, gotCommand, gotABI)
		}
	}
	// These original valid scalar domains overflow only the repeated logical
	// totals. No corresponding windows, records or original workload is built.
	if got, ok := providerHistoryCommandUpper(providerHistoryCollectionSemanticLimits{Groups: 10000, Requests: 10000, Blocks: 10000, Windows: 10000}, 1, 64<<20); ok || got != (providerHistoryCommandWork{}) {
		t.Fatal("old command whole-zero overflow changed", got, ok)
	}
	if got, ok := providerHistoryABIUpper(providerHistoryCollectionSemanticLimits{Groups: 100000, Requests: 100000, Blocks: 100000, Windows: 100000}, 1, 64<<20); ok || got != (providerHistoryABIWork{}) {
		t.Fatal("old ABI whole-zero overflow changed", got, ok)
	}
}

func TestProviderHistorySemanticWindowAtomsRefuseInvalidScalarDomains(t *testing.T) {
	for _, row := range [][2]uint64{{0, 1}, {1, 0}, {65537, 1}, {1, (64 << 20) + 1}, {^uint64(0), 1}, {1, ^uint64(0)}} {
		if got, ok := providerHistoryCommandWindowUpper(row[0], row[1]); ok || got != (providerHistoryCommandWork{}) {
			t.Fatal("invalid single command window leaked atoms", row, got, ok)
		}
		if got, ok := providerHistoryABIWindowUpper(row[0], row[1]); ok || got != (providerHistoryABIWork{}) {
			t.Fatal("invalid single ABI window leaked atoms", row, got, ok)
		}
	}
	// Original role domains are still independently required by the wrappers.
	bad := providerHistoryCollectionSemanticLimits{Groups: 0, Requests: 1, Blocks: 1, Windows: 1}
	if got, ok := providerHistoryCommandUpper(bad, 1, 1024); ok || got != (providerHistoryCommandWork{}) {
		t.Fatal("numeric atoms bypassed original command role counts", got, ok)
	}
	if got, ok := providerHistoryABIUpper(bad, 1, 1024); ok || got != (providerHistoryABIWork{}) {
		t.Fatal("numeric atoms bypassed original ABI role counts", got, ok)
	}
}

func TestProviderHistorySemanticWindowAtomsCoverOrdinaryOriginalDecoder(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	command, ok := providerHistoryCommandWindowUpper(1, 64<<20)
	binary, binaryOK := providerHistoryABIWindowUpper(1, 64<<20)
	if !ok || !binaryOK {
		t.Fatal("ordinary fixed-source numeric parts")
	}
	upper := command.actualWindow
	for c := range upper {
		upper[c] += binary.window[c]
	}
	seen := map[string]bool{}
	for _, step := range replayTestSteps(t) {
		want, accepted := replayParseCommand(replayTestRegistry, replayTestAgent, step.History.Command, contract)
		if !accepted {
			t.Fatal("ordinary canonical command precursor")
		}
		m := providerHistorySemanticUnitMeter()
		got, accepted := replayDecodeCommandObserved(replayTestRegistry, want.actor, want.calldata, contract, m)
		if !accepted || !reflect.DeepEqual(got, want) || m.failed || m.clocks != 0 {
			t.Fatal("same original observer through actual selector, ABI and command", want.op, m.actual)
		}
		if got.op == "create" {
			digest, hashed := replayClaimObserved(replayTestRegistry, got, m)
			if !hashed || digest != replayClaim(replayTestRegistry, want) {
				t.Fatal("original create claim precursor", m.actual)
			}
		}
		historyEventQueueWithin(t, m.actual, upper)
		seen[want.op] = true
	}
	if len(seen) != 8 {
		t.Fatal("all eight adopted methods must retain their original path")
	}
	// These are separate private DATA invocations, not an original complete
	// semantic window, source factory or event/command count witness.
}

func TestProviderHistorySemanticCommandRefusalCoversBenignInvalidService(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	var calldata []byte
	actor := ""
	for _, step := range replayTestSteps(t) {
		c, ok := replayParseCommand(replayTestRegistry, replayTestAgent, step.History.Command, contract)
		if !ok {
			t.Fatal("ordinary update-services precursor")
		}
		if c.op != "update-services" {
			continue
		}
		values, err := contract.Methods["updateServices"].Inputs.Unpack(c.calldata[4:])
		if err != nil {
			t.Fatal(err)
		}
		var services []replayABIService
		if !replayConvert(values[2], &services) || len(services) == 0 {
			t.Fatal("ordinary typed service tuple precursor")
		}
		services[0].URI = "not-a-uri" // Short ordinary schema error, no live endpoint.
		calldata, err = contract.Pack("updateServices", values[0], values[1], services)
		if err != nil {
			t.Fatal(err)
		}
		actor = c.actor
		break
	}
	if len(calldata) == 0 || actor == "" {
		t.Fatal("ordinary update-services fixture missing")
	}
	refusal, ok := providerHistoryCommandRefusalUpper()
	binary, binaryOK := providerHistoryABIWindowUpper(1, 64<<20)
	if !ok || !binaryOK {
		t.Fatal("numeric first-refusal components")
	}
	upper := refusal.normal
	for c := range upper {
		upper[c] += binary.window[c]
	}
	m := providerHistorySemanticUnitMeter()
	got, accepted := replayDecodeCommandObserved(replayTestRegistry, actor, calldata, contract, m)
	if accepted || !reflect.DeepEqual(got, replayCommand{}) || m.failed || m.clocks != 0 || m.actual[2] == 0 || m.actual[10] == 0 {
		t.Fatal("ordinary benign service error escaped or lost entered command work", m.actual, got)
	}
	historyEventQueueWithin(t, m.actual, upper)
	// First refusal ends this invocation; no retry, meter edit or source/owner
	// construction, network action or oversized/raw attack payload is supplied.
}
