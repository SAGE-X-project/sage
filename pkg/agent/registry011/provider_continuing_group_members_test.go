package registry011

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func historyGroupBounds() (providerMethodBounds, ProviderCaptureLimits) {
	limit := providerMethodLimit{1024, 4096}
	b := providerMethodBounds{limit, limit, limit, limit, limit, limit, limit, limit}
	return b, ProviderCaptureLimits{time.Second, 512, 2048, 8192, 32}
}

func historyGroupRequest(t *testing.T, method string, params any) *providerIntegrationRequest {
	t.Helper()
	r, err := newProviderIntegrationRequest(method, params)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func historyGroupSimple(t *testing.T) *providerIntegrationRequest {
	t.Helper()
	return historyGroupRequest(t, "eth_chainId", []any{})
}

func TestProviderHistoryGroupTablePreservesEveryMethodClass(t *testing.T) {
	h := admissionTestDigest("a")
	address := "0x" + strings.Repeat("1", 40)
	bounds, global := historyGroupBounds()
	var requests []*providerIntegrationRequest
	var classes []string
	for _, row := range []struct {
		method, class string
		params        any
	}{
		{"eth_chainId", "scalar", []any{}},
		{"web3_clientVersion", "scalar", []any{}},
		{"eth_getCode", "code", []any{address, "0x1"}},
		{"eth_call", "state", []any{map[string]string{"to": address, "data": "0x"}, "0x1"}},
		{"debug_getRawBlock", "block", []any{h}},
		{"eth_getBlockReceipts", "receipts", []any{h}},
		{"debug_getRawReceipts", "receipts", []any{h}},
		{"eth_getBlockByHash", "header", []any{h, false}},
		{"eth_getBlockByNumber", "block", []any{"finalized", true}},
		{"debug_traceTransaction", "gas", []any{h, map[string]any{"tracer": "stateGasTracer", "reexec": 0}}},
		{"debug_traceBlockByHash", "execution", []any{h, map[string]string{"tracer": "callTracer"}}},
	} {
		requests = append(requests, historyGroupRequest(t, row.method, row.params))
		classes = append(classes, row.class)
	}
	table, err := newProviderHistoryGroupTable(h, "content", 1, requests, bounds, global)
	if err != nil || table == nil || table.count != len(requests) || table.digest != providerCandidateHash(providerHistoryGroupTableDomain, []byte(table.raw)) {
		t.Fatal("normal closed request table failed", err)
	}
	for i, class := range classes {
		if table.members[i].class != class || table.members[i].request != *requests[i] {
			t.Fatal("equal numeric caps confused symbolic class or exact request")
		}
	}
	parsed, err := parseProviderHistoryGroupTable([]byte(table.raw), h, "content", 1, requests, bounds, global)
	if err != nil || *parsed != *table {
		t.Fatal("independently reconstructed table differed", err)
	}
}

func TestProviderHistoryGroupTableCopiesRequestsWithoutDeduplication(t *testing.T) {
	bounds, global := historyGroupBounds()
	r := historyGroupSimple(t)
	requests := make([]*providerIntegrationRequest, providerHistoryGroupMembers)
	for i := range requests {
		requests[i] = r
	}
	table, err := newProviderHistoryGroupTable(admissionTestDigest("a"), "identity", 1, requests, bounds, global)
	if err != nil || table.count != 32 {
		t.Fatal("exact preparation member ceiling refused", err)
	}
	original := table.members[0].request
	r.method = "web3_clientVersion"
	requests[0] = historyGroupSimple(t)
	if table.members[0].request != original || table.members[31].request != original {
		t.Fatal("caller pointers or a caller slice remained in the table")
	}
	var object struct {
		Members []struct{ Ordinal, Request string }
	}
	if err := json.Unmarshal([]byte(table.raw), &object); err != nil || len(object.Members) != 32 || object.Members[0].Ordinal != "0" || object.Members[31].Ordinal != "31" || object.Members[0].Request != object.Members[31].Request {
		t.Fatal("identical request bytes erased distinct ordered invocations")
	}
	if json.Unmarshal([]byte(`{}`), table) == nil {
		t.Fatal("JSON reconstructed a checked table or role")
	}
	for _, invalid := range []*providerHistoryGroupTable{nil, {}, {count: providerHistoryGroupMembers + 1}} {
		if invalid.object() != nil {
			t.Fatal("missing or unbounded table produced metadata")
		}
	}
}

func TestProviderHistoryGroupTableRejectsUnboundPreparation(t *testing.T) {
	bounds, global := historyGroupBounds()
	base := historyGroupSimple(t)
	for _, state := range []string{"program", "purpose", "stage-zero", "stage-large", "empty", "too-many", "bad-class-limits", "bad-global", "missing-request", "bad-shape", "class-cap"} {
		t.Run(state, func(t *testing.T) {
			program, purpose, stage := admissionTestDigest("a"), "content", uint64(1)
			requests := []*providerIntegrationRequest{base}
			b, g := bounds, global
			switch state {
			case "program":
				program = ""
			case "purpose":
				purpose = "unknown"
			case "stage-zero":
				stage = 0
			case "stage-large":
				stage = 1<<53 + 1
			case "empty":
				requests = nil
			case "too-many":
				requests = make([]*providerIntegrationRequest, providerHistoryGroupMembers+1)
			case "bad-class-limits":
				b.scalar = providerMethodLimit{}
			case "bad-global":
				g.Timeout = 0
			case "missing-request":
				requests[0] = nil
			case "bad-shape":
				requests[0] = historyGroupRequest(t, "eth_getBlockByNumber", []any{"finalized"})
			case "class-cap":
				b.scalar = providerMethodLimit{128, 600}
				requests[0] = historyGroupRequest(t, "eth_chainId", []any{strings.Repeat("x", 1000)})
			}
			if table, err := newProviderHistoryGroupTable(program, purpose, stage, requests, b, g); err == nil || table != nil {
				t.Fatal("invalid preparation produced a table")
			}
		})
	}
	for _, purpose := range []string{"selection", "identity", "ancestry", "content", "effects", "state", "history", "final"} {
		if _, err := newProviderHistoryGroupTable(admissionTestDigest("a"), purpose, 1<<53, []*providerIntegrationRequest{base}, bounds, global); err != nil {
			t.Fatal("finite purpose or exact stage ceiling refused", err)
		}
	}
}

func TestProviderHistoryGroupRequestCopyRefusesMismatchedData(t *testing.T) {
	base := historyGroupSimple(t)
	if _, err := providerHistoryGroupRequestCopy(nil); err == nil {
		t.Fatal("missing request copied")
	}
	for _, field := range []string{"params-empty", "params-large", "params-json", "body-empty", "body-large", "head-empty", "head-large", "wire-empty", "wire-large", "method", "body", "head", "wire", "input-digest", "params-digest"} {
		x := *base
		switch field {
		case "params-empty":
			x.params = ""
		case "params-large":
			x.params = strings.Repeat("x", 8<<10+1)
		case "params-json":
			x.params = "["
		case "body-empty":
			x.body = ""
		case "body-large":
			x.body = strings.Repeat("x", 16<<10+1)
		case "head-empty":
			x.head = ""
		case "head-large":
			x.head = strings.Repeat("x", 513)
		case "wire-empty":
			x.wire = ""
		case "wire-large":
			x.wire = strings.Repeat("x", 16<<10+1)
		case "method":
			x.method = "unsupported"
		case "body":
			x.body = "different"
		case "head":
			x.head = "different"
		case "wire":
			x.wire = "different"
		case "input-digest":
			x.inputDigest = admissionTestDigest("b")
		case "params-digest":
			x.paramsDigest = admissionTestDigest("b")
		}
		if got, err := providerHistoryGroupRequestCopy(&x); err == nil || got != (providerIntegrationRequest{}) {
			t.Fatal("incorrect literal request reused a different byte identity", field)
		}
	}
}

func TestProviderHistoryGroupClassRefusesUnclosedShapes(t *testing.T) {
	for _, request := range []*providerIntegrationRequest{
		nil,
		{params: strings.Repeat("x", 8<<10+1)},
		{method: "unknown", params: `["x",false]`},
		{method: "eth_getBlockByHash", params: "["},
		{method: "eth_getBlockByHash", params: "[]"},
		{method: "eth_getBlockByHash", params: `["x","false"]`},
		{method: "debug_traceBlockByHash", params: `["x",false]`},
		{method: "debug_traceBlockByHash", params: `["x",{"tracer":"unknown"}]`},
	} {
		if class, err := providerHistoryGroupRequestClass(request); err == nil || class != "" {
			t.Fatal("an unknown class or malformed literal parameters became closed")
		}
	}
}

func TestProviderHistoryGroupTableBindsOrderAndExactOriginalMetadata(t *testing.T) {
	bounds, global := historyGroupBounds()
	program := admissionTestDigest("a")
	first := historyGroupSimple(t)
	second := historyGroupRequest(t, "web3_clientVersion", []any{})
	requests := []*providerIntegrationRequest{first, second}
	table, err := newProviderHistoryGroupTable(program, "identity", 1, requests, bounds, global)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range [][]byte{nil, []byte(strings.Repeat("x", providerHistoryGroupTableBytes+1)), []byte("different"), append([]byte(table.raw), '\n')} {
		if got, err := parseProviderHistoryGroupTable(raw, program, "identity", 1, requests, bounds, global); err == nil || got != nil {
			t.Fatal("nonidentical retained table was accepted")
		}
	}
	for _, state := range []string{"program", "purpose", "stage", "order", "profile", "invalid-preparation"} {
		p, purpose, stage := program, "identity", uint64(1)
		r := []*providerIntegrationRequest{first, second}
		b := bounds
		switch state {
		case "program":
			p = admissionTestDigest("b")
		case "purpose":
			purpose = "final"
		case "stage":
			stage = 2
		case "order":
			r[0], r[1] = r[1], r[0]
		case "profile":
			b.scalar.body--
		case "invalid-preparation":
			stage = 0
		}
		if got, err := parseProviderHistoryGroupTable([]byte(table.raw), p, purpose, stage, r, b, global); err == nil || got != nil {
			t.Fatal("changed order, program or selected capture reused an original table", state)
		}
	}
}
