package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

func historyABIActualDelta(t *testing.T, m *providerWorkMeter, before candidateWorkVector) candidateWorkVector {
	t.Helper()
	var delta candidateWorkVector
	for i, value := range m.actual {
		if value < before[i] {
			t.Fatal("ABI calibration refunded its existing observer prefix")
		}
		delta[i] = value - before[i]
	}
	return delta
}

func historyABIWithin(t *testing.T, actual, upper candidateWorkVector) {
	t.Helper()
	for i, value := range actual {
		if value > upper[i] {
			t.Fatal("ABI observed component exceeds its source row", i, actual, upper)
		}
	}
}

func TestProviderHistoryABIUpperMatchesIndependentWholeRows(t *testing.T) {
	a := uint64(reflect.TypeFor[abi.Argument]().Size())
	m := uint64(reflect.TypeFor[abi.Method]().Size())
	i := uint64(reflect.TypeFor[replayClaimID]().Size())
	s := uint64(reflect.TypeFor[replayClaimState]().Size())
	if max(a, m, i, s) > 4096 {
		t.Fatal("changed fixed source shapes require rederiving the literal oracle")
	}
	for _, tc := range []struct {
		g, r, w, logs, evidence, pairs, slots uint64
		windowD, windowA, windowM, windowC    uint64
		totalD, totalA, totalM, totalC        uint64
	}{
		{1, 1, 1, 1, 1024, 17, 15, 13264, 2192, 8, 305110250, 369215, 32880, 120, 5537404398},
		{2, 3, 2, 2, 4096, 64, 116, 28576, 4384, 16, 1219709396, 2298368, 254272, 928, 74360088584},
		{1, 32, 3, 65536, 67108864, 109, 2949120, 869269504, 143654912, 524288, 19995705344000,
			39118219315, 6464471040, 23592960, 899812900587096},
	} {
		got, ok := providerHistoryABIUpper(providerHistoryCollectionSemanticLimits{
			Groups: tc.g, Requests: tc.r, Blocks: tc.w, Windows: tc.w,
		}, tc.logs, tc.evidence)
		want := providerHistoryABIWork{
			fixed:           candidateWorkVector{2: 10015, 10: 56514744},
			selector:        candidateWorkVector{10: 8*m + 80},
			framing:         candidateWorkVector{2: 12208, 10: 2176*a + 364678},
			changed:         candidateWorkVector{2: 32, 10: 16*a + 1038},
			mutation:        candidateWorkVector{2: 12240, 10: 2192*a + 8*m + 365796},
			commit:          candidateWorkVector{2: 40, 10: m + 2*a + 2*i + s + 3189},
			event:           candidateWorkVector{2: 12240, 10: 2192*a + 8*m + 365796},
			canonicalWindow: candidateWorkVector{2: tc.evidence, 10: 297602*tc.evidence + 6*tc.logs},
			window:          candidateWorkVector{2: tc.windowD, 10: tc.windowA*a + tc.windowM*m + tc.windowC},
			total:           candidateWorkVector{2: tc.totalD, 10: tc.totalA*a + tc.totalM*m + tc.totalC},
			initializations: tc.pairs,
			eventSlots:      tc.slots,
		}
		if !ok || got != want {
			t.Fatal("independent ABI byte/type-size/count/component literals", got, want, ok)
		}
	}
}

func TestProviderHistoryABIUpperPinsItsClosedSourceShapes(t *testing.T) {
	for _, tc := range []struct{ raw, digest string }{
		{replayContractABI, "0f31f995b0169b27c2ce4797b967a754b4c0b2b0a733e4c21da1b54089380f5f"},
		{replayOutcomesABI, "7d2c1fd84fcc83e3718512c36089b42779fe999f295d4185c0e229b8cf2038c8"},
	} {
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(tc.raw))); got != tc.digest {
			t.Fatal("fixed ABI source bytes changed; rederive all literal schema costs", got)
		}
	}
	type field struct {
		Name, Type string
		Components []field
		Indexed    bool
	}
	var definitions []struct {
		Name, Type string
		Inputs     []field
	}
	if err := json.Unmarshal([]byte(replayContractABI), &definitions); err != nil {
		t.Fatal("independent fixed ABI JSON schema", err)
	}
	var shape func([]field) string
	shape = func(fields []field) string {
		parts := make([]string, len(fields))
		for i, f := range fields {
			parts[i] = f.Name + ":" + f.Type
			if len(f.Components) > 0 {
				parts[i] += "(" + shape(f.Components) + ")"
			}
		}
		return strings.Join(parts, ",")
	}
	key := "name:string,alg:string,key:bytes,signer:string,proof:bytes,hasExpiry:bool,expires:uint64"
	service := "name:string,serviceType:string,uri:string"
	want := map[string]string{
		"revealClaim":       "agentId:string,controller:address,keys:tuple[](" + key + "),services:tuple[](" + service + "),salt:bytes32",
		"activate":          "agentId:string,expectedVersion:uint64",
		"deactivate":        "agentId:string,expectedVersion:uint64",
		"addKey":            "agentId:string,expectedVersion:uint64,entry:tuple(" + key + ")",
		"revokeKey":         "agentId:string,expectedVersion:uint64,name:string",
		"updateServices":    "agentId:string,expectedVersion:uint64,services:tuple[](" + service + ")",
		"authorizeOperator": "agentId:string,expectedVersion:uint64,operator:address,scope:string",
		"revokeOperator":    "agentId:string,expectedVersion:uint64,operator:address,scope:string",
	}
	seen := 0
	for _, d := range definitions {
		if expected, adopted := want[d.Name]; adopted {
			if d.Type != "function" || shape(d.Inputs) != expected {
				t.Fatal("fixed framing type graph changed", d.Name, shape(d.Inputs), expected)
			}
			seen++
		}
	}
	if seen != 8 {
		t.Fatal("all eight fixed mutation type graphs required")
	}
	for name, maximum := range map[string]int{"agentId": 64, "serviceType": 64, "name": 32, "alg": 24,
		"key": 65, "proof": 65, "signer": 189, "uri": 2048, "scope": 15, "did": 132, "operation": 18} {
		if replayABIFieldMaximum(name) != maximum {
			t.Fatal("source field ceiling changed", name)
		}
	}
}

func TestProviderHistoryABIUpperCoversAllOrdinaryMutationStages(t *testing.T) {
	quote, ok := providerHistoryABIUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 1, 64<<20)
	if !ok {
		t.Fatal("finite ABI component quote")
	}
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal("ordinary fixed ABI precursor", err)
	}
	positions := map[string]uint64{"revealClaim": 1, "activate": 2, "deactivate": 3, "addKey": 4,
		"revokeKey": 5, "updateServices": 6, "authorizeOperator": 7, "revokeOperator": 8}
	seen := map[string]bool{}
	for _, step := range replayTestSteps(t) {
		c, ok := replayParseCommand(replayTestRegistry, replayTestAgent, step.History.Command, contract)
		if !ok {
			t.Fatal("ordinary complete command precursor")
		}
		m := providerHistoryExecutionUnitMeter()
		method, ok := replayABIMethodObserved(contract, c.calldata, m)
		if !ok || method == nil || positions[method.Name] == 0 ||
			m.actual != (candidateWorkVector{10: positions[method.Name] * (uint64(reflect.TypeFor[abi.Method]().Size()) + 10)}) {
			t.Fatal("actual selector order/copies/ID comparisons", m.actual)
		}
		historyABIWithin(t, m.actual, quote.selector)
		start := m.actual
		if !replayCallFramingObserved(*method, c.calldata, m) {
			t.Fatal("ordinary original calldata framing")
		}
		historyABIWithin(t, historyABIActualDelta(t, m, start), quote.framing)
		start = m.actual
		values, err := replayABICanonicalObserved(method.Inputs, c.calldata[4:], m)
		n := uint64(len(c.calldata) - 4)
		if err != nil || n > 148800 || historyABIActualDelta(t, m, start) != (candidateWorkVector{2: n, 10: 2*n*n + 2*n + 3}) {
			t.Fatal("actual canonical codec/encode/equality byte arithmetic", err, m.actual)
		}
		packed, err := method.Inputs.Pack(values...)
		if err != nil || !bytes.Equal(packed, c.calldata[4:]) || m.failed || m.closed {
			t.Fatal("native independent re-pack changed the original binary bytes", err)
		}
		seen[method.Name] = true
	}
	if len(seen) != 8 {
		t.Fatal("all eight successful mutation stages required")
	}
}

func TestProviderHistoryABIUpperCoversOrdinaryChangedAndCommitBranches(t *testing.T) {
	quote, ok := providerHistoryABIUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 1, 64<<20)
	if !ok {
		t.Fatal("finite alternative branch quote")
	}
	in := replayTestInput(t)
	t.Run("changed-fields", func(t *testing.T) {
		_, outcomes, err := replayABIs()
		if err != nil {
			t.Fatal(err)
		}
		frame := &in.Blocks[1].Executions[0].Root.Calls[0]
		m := providerHistoryExecutionUnitMeter()
		fields, ok := replayABINonIndexedObserved(outcomes.Events["RecordChanged"].Inputs, m)
		if !ok || len(fields) != 8 || m.actual != (candidateWorkVector{10: 16*uint64(reflect.TypeFor[abi.Argument]().Size()) + 9}) {
			t.Fatal("actual owned eight-field projection", m.actual)
		}
		if n, ok := replayABIFieldsObserved(frame.Logs[0].Data, fields, m); !ok || n != len(frame.Logs[0].Data) || n > 512 {
			t.Fatal("ordinary bounded RecordChanged body")
		}
		historyABIWithin(t, m.actual, quote.changed)
		historyABIWithin(t, quote.mutation, quote.event)
	})
	t.Run("complete-commit", func(t *testing.T) {
		m := providerHistoryExecutionUnitMeter()
		r := providerHistoryEventUnitRun(t, in, m)
		frame := &in.Blocks[0].Executions[0].Root.Calls[0]
		header := &types.Header{Number: big.NewInt(0)}
		if err := r.commitObserved(providerHistoryEventUnitLog(frame), header, r.outcomes.Events["ClaimCommitted"]); err != nil ||
			len(r.claims) != 1 || m.actual != quote.commit || m.failed || m.closed {
			t.Fatal("complete original commit row and final claim store", err, m.actual, quote.commit)
		}
		historyABIWithin(t, quote.commit, quote.event)
		upper := quote.commit
		upper[10]--
		short := providerHistoryContentUnitMeter(upper)
		other := providerHistoryEventUnitRun(t, in, short)
		if err := other.commitObserved(providerHistoryEventUnitLog(frame), header, other.outcomes.Events["ClaimCommitted"]); err == nil || !short.failed || len(other.claims) != 0 {
			t.Fatal("one-short final commit store escaped")
		}
		prefix := quote.commit
		prefix[10] -= uint64(reflect.TypeFor[replayClaimID]().Size()+reflect.TypeFor[replayClaimState]().Size()) + 1
		if short.actual != prefix {
			t.Fatal("late original commit store did not retain its exact prefix", short.actual, prefix)
		}
		if err := other.commitObserved(providerHistoryEventUnitLog(frame), header, other.outcomes.Events["ClaimCommitted"]); err == nil || short.actual != prefix || len(other.claims) != 0 {
			t.Fatal("failed commit DATA observer revived")
		}
	})
}

func TestProviderHistoryABIUpperFixedPairKeepsLateCodecPrefix(t *testing.T) {
	quote, ok := providerHistoryABIUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, 1, 1024)
	if !ok {
		t.Fatal("finite pair quote")
	}
	m := providerHistoryContentUnitMeter(quote.fixed)
	contract, outcomes, err := replayABIsObserved(m)
	if err != nil || len(contract.Methods) != 13 || len(outcomes.Events) != 2 || m.actual != quote.fixed || m.failed {
		t.Fatal("actual two fixed input copies and native codec observations", err, m.actual)
	}
	upper := quote.fixed
	upper[10]--
	short := providerHistoryContentUnitMeter(upper)
	contract, outcomes, err = replayABIsObserved(short)
	if err == nil || len(contract.Methods) != 0 || len(outcomes.Events) != 0 || !short.failed ||
		short.actual != (candidateWorkVector{2: 6790, 10: 46114118}) {
		t.Fatal("second codec shortage leaked maps or refunded first ABI/copy prefix", err, short.actual)
	}
	before := short.actual
	if _, _, err := replayABIsObserved(short); err == nil || short.actual != before {
		t.Fatal("failed original fixed ABI observer restarted")
	}
}

func TestProviderHistoryABIUpperCanonicalPoolUsesOriginalSharedByteCharge(t *testing.T) {
	in := replayTestInput(t)
	m := providerHistoryExecutionUnitMeter()
	b := replayBudget{limits: replayTestLimits(), observer: m}
	root := &in.Blocks[1].Executions[0].Root
	if _, err := replayFrameObserved(context.Background(), root, "", false, true, 0, &b); err != nil {
		t.Fatal("ordinary original frame/log bytes must be charged first", err)
	}
	frame := &root.Calls[0]
	contract, outcomes, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	method, err := contract.MethodById(frame.Input[:4])
	if err != nil {
		t.Fatal("ordinary mutation selector precursor", err)
	}
	fields, selected := replayABINonIndexedObserved(outcomes.Events["RecordChanged"].Inputs, m)
	call, event := frame.Input[4:], frame.Logs[0].Data
	if !selected || !replayCallFramingObserved(*method, frame.Input, m) {
		t.Fatal("same-observer ordinary mutation framing precursor")
	}
	if n, ok := replayABIFieldsObserved(event, fields, m); !ok || n != len(event) {
		t.Fatal("same-observer ordinary event framing precursor")
	}
	if len(call)+len(event) > b.bytes || b.bytes > b.limits.MaxEvidenceBytes || b.traceLogs < 1 {
		t.Fatal("two canonical inputs must share the same original frame/log byte budget")
	}
	quote, ok := providerHistoryABIUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1},
		uint64(b.limits.MaxLogs), uint64(b.limits.MaxEvidenceBytes))
	if !ok {
		t.Fatal("original shared budget quote")
	}
	start := m.actual
	if _, err := replayABICanonicalObserved(method.Inputs, call, m); err != nil {
		t.Fatal("ordinary actual mutation canonical input", err)
	}
	if _, err := replayABICanonicalObserved(fields, event, m); err != nil {
		t.Fatal("ordinary actual event canonical input", err)
	}
	n, k := uint64(len(call)), uint64(len(event))
	want := candidateWorkVector{2: n + k, 10: 2*n*n + 2*n + 3 + 2*k*k + 2*k + 3}
	if actual := historyABIActualDelta(t, m, start); actual != want {
		t.Fatal("canonical pool replaced actual output with a prospective bound", actual, want)
	}
	historyABIWithin(t, want, quote.canonicalWindow)
	// Independent finite extremal DATA: two canonical calls for each event,
	// mixed lengths and a tight shared sum. These are scalar byte lengths,
	// not generated peer messages, a runtime profile or another observation.
	for _, lengths := range [][]uint64{{148800, 512}, {32, 512, 148800, 512}, {64, 512, 64, 512}} {
		var total, logical uint64
		for _, n := range lengths {
			total += n
			logical += 2*n*n + 2*n + 3
		}
		l := uint64(len(lengths) / 2)
		q, ok := providerHistoryABIUpper(providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}, l, total)
		if !ok || logical > q.canonicalWindow[10] || total != q.canonicalWindow[2] {
			t.Fatal("shared canonical byte inequality", lengths, logical, q.canonicalWindow)
		}
	}
}

func TestProviderHistoryABIUpperRejectsInvalidOriginalDomains(t *testing.T) {
	base := providerHistoryCollectionSemanticLimits{Groups: 2, Requests: 3, Blocks: 2, Windows: 2}
	for _, mode := range []string{"zero-groups", "wide-groups", "group-product", "few-requests", "many-requests", "zero-blocks", "wide-blocks",
		"zero-windows", "wide-windows", "wide-versions", "many-stops", "zero-logs", "wide-logs", "zero-evidence", "wide-evidence"} {
		t.Run(mode, func(t *testing.T) {
			l, logs, evidence := base, uint64(1), uint64(1024)
			switch mode {
			case "zero-groups":
				l.Groups = 0
			case "wide-groups":
				l.Groups = (1<<20)/3 + 1
			case "group-product":
				l.Groups = ^uint64(0)
			case "few-requests":
				l.Requests = 1
			case "many-requests":
				l.Requests = 65
			case "zero-blocks":
				l.Blocks = 0
			case "wide-blocks":
				l.Blocks = 1<<20 + 1
			case "zero-windows":
				l.Windows = 0
			case "wide-windows":
				l.Windows = 3
			case "wide-versions":
				l.Versions = 1<<20 + 1
			case "many-stops":
				l.PresentStops = 3
			case "zero-logs":
				logs = 0
			case "wide-logs":
				logs = 65537
			case "zero-evidence":
				evidence = 0
			case "wide-evidence":
				evidence = 64<<20 + 1
			}
			if got, ok := providerHistoryABIUpper(l, logs, evidence); ok || got != (providerHistoryABIWork{}) {
				t.Fatal("invalid original ABI limits leaked a partial quote", mode, got, ok)
			}
		})
	}
}

func TestProviderHistoryABIUpperLateLogicalOverflowReturnsWholeZero(t *testing.T) {
	// Numeric source domains are valid; native feasibility/source binding is
	// not claimed. Decoded work and all per-window rows remain finite, while
	// repeating the complete logical window exceeds uint64. No finite count
	// or decoded prefix may survive the later failure.
	l := providerHistoryCollectionSemanticLimits{Groups: 100000, Requests: 100000, Blocks: 100000, Windows: 100000}
	if got, ok := providerHistoryABIUpper(l, 1, 64<<20); ok || got != (providerHistoryABIWork{}) {
		t.Fatal("late repeated ABI logical overflow leaked finite rows/counts", got, ok)
	}
}
