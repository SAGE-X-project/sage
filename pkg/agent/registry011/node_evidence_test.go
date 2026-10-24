package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"
)

func nodeTestLimits() NodeLimits {
	return NodeLimits{Replay: replayTestLimits(), MaxRequests: 32, MaxWireBytes: 8 << 20, AcquisitionTimeout: 120 * time.Second, TraceTimeout: 2 * time.Second}
}
func nodeRecorded(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	b, err := os.ReadFile("testdata/node-deployment-rpc.json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(b, &m) != nil {
		t.Fatal("fixture")
	}
	return m
}

// These normal public responses came only from the approved official endpoint.
// Refusals below are finite in-memory mutations, with no malformed network peer.
func nodeFixture(t *testing.T, mutate func(string, []json.RawMessage, []byte) []byte) (*PinnedAgentReads, *int) {
	t.Helper()
	recorded := nodeRecorded(t)
	calls := 0
	identity, _, err := nodeIdentityManifest(nodeIdentityJSON, nodeABIJSON)
	if err != nil {
		t.Fatal(err)
	}
	c := testReadRPC(t, func(req *http.Request) (*http.Response, error) {
		calls++
		var in struct {
			Method string
			Params []json.RawMessage
		}
		b, err := io.ReadAll(req.Body)
		if err != nil || json.Unmarshal(b, &in) != nil {
			t.Fatal("request")
		}
		if req.URL.String() != SepoliaRPCURL {
			t.Fatal("endpoint")
		}
		key := ""
		switch in.Method {
		case "eth_chainId":
			key = "chain"
		case "web3_clientVersion":
			key = "client"
		case "eth_getBlockByNumber":
			var tag string
			_ = json.Unmarshal(in.Params[0], &tag)
			if tag == "0x0" {
				key = "genesis"
			} else if tag == "0xb6b80e" {
				key = "canonical-deployment"
			} else {
				key = "finalized"
			}
		case "eth_call":
			key = "identity-at-anchor"
		case "eth_getCode":
			var address string
			_ = json.Unmarshal(in.Params[0], &address)
			for _, v := range identity.Contracts {
				if v.Address == address {
					key = "code-" + v.Name
				}
			}
			var pin map[string]any
			_ = json.Unmarshal(in.Params[1], &pin)
			if pin["requireCanonical"] != true {
				t.Fatal("unbound code")
			}
		case "debug_getRawBlock":
			key = "raw-block"
		case "debug_getRawReceipts":
			key = "raw-receipts"
		case "debug_traceTransaction":
			key = "deployment-call-trace"
			var opt map[string]any
			_ = json.Unmarshal(in.Params[1], &opt)
			if opt["tracer"] != "callTracer" || opt["reexec"] != float64(0) || opt["timeout"] != "2s" {
				t.Fatal("trace request")
			}
		default:
			t.Fatal("unexpected method", in.Method)
		}
		body := bytes.Clone(recorded[key])
		if len(body) == 0 {
			t.Fatal("missing fixture", key)
		}
		if mutate != nil {
			body = mutate(in.Method, in.Params, body)
		}
		return jsonResponse(string(body)), nil
	})
	c.limits.MaxDepth = 128
	c.limits.RequestTimeout = 20 * time.Second
	r, err := c.PinAgentReads(context.Background(), SepoliaRegistryAddress, "sage-deployment-check")
	if err != nil {
		t.Fatal(err)
	}
	return r, &calls
}

func TestNodeRecordedDeploymentConnection(t *testing.T) {
	r, calls := nodeFixture(t, nil)
	before := *calls
	got, err := r.ObserveDeployment(context.Background(), nodeTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.CodeHashes) != 13 || got.Deployment.Number != sepoliaDeploymentBlock || got.Creation.TransactionIndex != 62 || got.Creation.Root.Type != "CREATE" || len(got.Exchanges) != *calls-before || len(got.Block.Transactions) != 120 || len(got.Block.Receipts) != 120 {
		t.Fatal("bounded observation", got.Deployment, *calls)
	}
	if got.Anchor != r.header || got.Creation.Root.To != SepoliaRegistryAddress {
		t.Fatal("binding")
	}
	frames := 0
	var walk func(ReplayFrame)
	walk = func(f ReplayFrame) {
		frames++
		for _, c := range f.Calls {
			walk(c)
		}
	}
	walk(got.Creation.Root)
	if frames != 16 {
		t.Fatal(frames)
	}
	for _, e := range got.Exchanges {
		if len(e.Response) == 0 || !json.Valid(e.Params) {
			t.Fatal("raw evidence")
		}
	}
	block, err := r.ReadNodeBlock(context.Background(), sepoliaDeploymentBlock, nodeTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(block.Block.Executions) != 0 || block.Header != got.Deployment {
		t.Fatal("creation has no Registry event; do not claim live mutation")
	}
	// Returned buffers are private to each acquisition.
	got.Exchanges[0].Response[0] = 'x'
	again, err := r.ObserveDeployment(context.Background(), nodeTestLimits())
	if err != nil || again.Exchanges[0].Response[0] != '{' {
		t.Fatal("retained caller buffer", err)
	}
}

func TestNodeExplicitBudgetsAndContexts(t *testing.T) {
	r, calls := nodeFixture(t, nil)
	for name, change := range map[string]func(*NodeLimits){
		"requests0": func(x *NodeLimits) { x.MaxRequests = 0 }, "requests257": func(x *NodeLimits) { x.MaxRequests = 257 }, "bytes0": func(x *NodeLimits) { x.MaxWireBytes = 0 }, "bytes-over": func(x *NodeLimits) { x.MaxWireBytes = 64<<20 + 1 }, "time0": func(x *NodeLimits) { x.AcquisitionTimeout = 0 }, "time-over": func(x *NodeLimits) { x.AcquisitionTimeout = 5*time.Minute + 1 }, "trace0": func(x *NodeLimits) { x.TraceTimeout = 0 }, "trace-over": func(x *NodeLimits) { x.TraceTimeout = 5*time.Second + 1 }, "trace-acq": func(x *NodeLimits) { x.AcquisitionTimeout = time.Second }, "trace-http": func(x *NodeLimits) { x.TraceTimeout = 3 * time.Second; r.rpc.limits.RequestTimeout = time.Second }, "replay": func(x *NodeLimits) { x.Replay.MaxFrames = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			r.rpc.limits.RequestTimeout = 20 * time.Second
			l := nodeTestLimits()
			change(&l)
			before := *calls
			if out, e := r.ObserveDeployment(context.Background(), l); out != nil || e != ErrConfiguration || *calls != before {
				t.Fatal(e)
			}
		})
	}
	r.rpc.limits.RequestTimeout = 20 * time.Second
	for _, l := range []NodeLimits{func() NodeLimits { x := nodeTestLimits(); x.MaxRequests = 1; return x }(), func() NodeLimits { x := nodeTestLimits(); x.MaxWireBytes = 1; return x }(), func() NodeLimits { x := nodeTestLimits(); x.Replay.MaxTransactions = 1; return x }(), func() NodeLimits { x := nodeTestLimits(); x.Replay.MaxEvidenceBytes = 1; return x }(), func() NodeLimits { x := nodeTestLimits(); x.Replay.MaxFrames = 1; return x }()} {
		if out, e := r.ObserveDeployment(context.Background(), l); out != nil || e == nil {
			t.Fatal("budget", e)
		}
	}
	for name, ctx := range map[string]context.Context{"missing": nil, "cancelled": func() context.Context { c, cancel := context.WithCancel(context.Background()); cancel(); return c }()} {
		t.Run(name, func(t *testing.T) {
			if out, e := r.ObserveDeployment(ctx, nodeTestLimits()); out != nil || e == nil {
				t.Fatal("context")
			}
		})
	}
	for _, n := range []uint64{0, sepoliaDeploymentBlock - 1, r.header.Number + 1} {
		if out, e := r.ReadNodeBlock(context.Background(), n, nodeTestLimits()); out != nil || e == nil {
			t.Fatal("range")
		}
	}
	for _, absent := range []*PinnedAgentReads{nil, {}} {
		if out, e := absent.ReadNodeBlock(context.Background(), sepoliaDeploymentBlock, nodeTestLimits()); out != nil || e != ErrConfiguration {
			t.Fatal("zero")
		}
	}
}

func nodeMutateResult(body []byte, f func(*json.RawMessage)) []byte {
	var env map[string]json.RawMessage
	_ = json.Unmarshal(body, &env)
	v := env["result"]
	f(&v)
	env["result"] = v
	b, _ := json.Marshal(env)
	return b
}

func TestNodeFiniteResponseRefusals(t *testing.T) {
	for _, name := range []string{"code", "rawblock", "receipts", "receipt-count", "trace", "trace-context", "rpc-error", "postcanonical", "finalized-rollback", "missing-creation"} {
		t.Run(name, func(t *testing.T) {
			active := false
			headerCalls := 0
			r, _ := nodeFixture(t, func(method string, params []json.RawMessage, body []byte) []byte {
				if !active {
					return body
				}
				if method == "eth_getBlockByNumber" {
					headerCalls++
				}
				target := map[string]string{"code": "eth_getCode", "rawblock": "debug_getRawBlock", "receipts": "debug_getRawReceipts", "receipt-count": "debug_getRawReceipts", "trace": "debug_traceTransaction", "trace-context": "debug_traceTransaction", "rpc-error": "eth_getCode", "missing-creation": "eth_getBlockByNumber", "postcanonical": "eth_getBlockByNumber", "finalized-rollback": "eth_getBlockByNumber"}[name]
				if method != target {
					return body
				}
				if name == "postcanonical" && headerCalls < 3 {
					return body
				}
				if name == "finalized-rollback" && string(params[0]) != `"finalized"` {
					return body
				}
				if name == "rpc-error" {
					return []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"unavailable"}}`)
				}
				return nodeMutateResult(body, func(v *json.RawMessage) {
					switch name {
					case "code", "rawblock", "receipts", "trace":
						*v = json.RawMessage(`"0x00"`)
					case "receipt-count":
						*v = json.RawMessage(`[]`)
					case "trace-context":
						var m map[string]any
						_ = json.Unmarshal(*v, &m)
						m["from"] = replayTestWallet
						*v, _ = json.Marshal(m)
					case "postcanonical", "missing-creation":
						var m map[string]any
						_ = json.Unmarshal(*v, &m)
						m["hash"] = testBlock
						*v, _ = json.Marshal(m)
					case "finalized-rollback":
						var m map[string]any
						_ = json.Unmarshal(*v, &m)
						m["number"] = "0x1"
						*v, _ = json.Marshal(m)
					}
				})
			})
			active = true
			if out, e := r.ObserveDeployment(context.Background(), nodeTestLimits()); out != nil || e == nil {
				t.Fatal("accepted", name)
			}
		})
	}
}

func TestNodeHeaderAndPreallocationRefusals(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `{}`, `{"Hash":"bad"}`, `{"Hash":"` + testBlock + `","ParentHash":"` + testParent + `","Number":"0x00","Timestamp":"0x1"}`} {
		r, _ := nodeFixture(t, nil)
		r.rpc.client.Transport = rpcRoundTrip(func(*http.Request) (*http.Response, error) {
			return jsonResponse(`{"jsonrpc":"2.0","id":1,"result":` + raw + `}`), nil
		})
		if out, e := r.ReadNodeBlock(context.Background(), sepoliaDeploymentBlock, nodeTestLimits()); out != nil || e == nil {
			t.Fatal(raw)
		}
	}
	for _, b := range [][]byte{nil, {0xc0}, {0xc1, 0x80}, {0xc2, 0xc0, 0x80}, {0xc2, 0xc0, 0xc0, 0}} {
		if nodeBlockCount(b, 1) {
			t.Fatal("RLP", fmt.Sprintf("%x", b))
		}
	}
	for _, raw := range []string{"null", "{}", "[1", `[1] true`, `[1,2]`} {
		if _, ok := nodeRawArray(json.RawMessage(raw), 1); ok {
			t.Fatal(raw)
		}
	}
	for _, s := range []string{"1e0", "-1", "1.0", "01", "null", `"0x00"`} {
		if _, ok := nodeIndex(json.RawMessage(s)); ok {
			t.Fatal(s)
		}
	}
	if n, ok := nodeIndex(json.RawMessage(`"0x2"`)); !ok || n != 2 {
		t.Fatal("hexposition")
	}
	if n, ok := nodeIndex(json.RawMessage(strconv.Itoa(2))); !ok || n != 2 {
		t.Fatal("decimalposition")
	}
	if _, _, e := nodeIdentityManifest([]byte(`{}`), nodeABIJSON); e != ErrConfiguration {
		t.Fatal("manifest")
	}
	if _, _, e := nodeIdentityManifest(nodeIdentityJSON, []byte(`[]`)); e != ErrConfiguration {
		t.Fatal("ABI pin")
	}
	if _, ok := nodeString(json.RawMessage(`null`)); ok {
		t.Fatal("null string")
	}
	if _, ok := nodeObject(json.RawMessage(`{"type from":1}`), "type from"); ok {
		t.Fatal("unknown merged name")
	}
	if _, ok := nodeData(json.RawMessage(`"0xAB"`)); ok {
		t.Fatal("hexcase")
	}
}
