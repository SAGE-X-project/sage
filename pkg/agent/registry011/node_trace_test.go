package registry011

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// This encoder represents owned offline execution fixtures only. It is not an
// endpoint, tracer, attack producer, or a claim that node options were honored.
func nodeTestFrame(f ReplayFrame) map[string]any {
	m := map[string]any{"type": f.Type, "from": f.From, "to": f.To, "input": nodeHex(f.Input), "value": f.Value, "gas": "0x100000", "gasUsed": "0x1"}
	if f.Error != "" {
		m["error"] = f.Error
		m["revertReason"] = "fixture refusal"
	}
	for _, child := range f.Calls {
		calls, _ := m["calls"].([]any)
		m["calls"] = append(calls, nodeTestFrame(child))
	}
	for _, l := range f.Logs {
		logs, _ := m["logs"].([]any)
		topics := []string{}
		for _, v := range l.Topics {
			topics = append(topics, v.Hex())
		}
		m["logs"] = append(logs, map[string]any{"address": l.Address, "topics": topics, "data": nodeHex(l.Data), "position": l.AfterChildren, "index": "0x0"})
	}
	return m
}

func TestNodeOwnedWalletCallAndReceiptJoin(t *testing.T) {
	in := replayTestInput(t)
	raw := in.Blocks[0]
	var h types.Header
	_ = rlp.DecodeBytes(raw.HeaderRLP, &h)
	h.Number = new(big.Int).SetUint64(sepoliaDeploymentBlock)
	tx := new(types.Transaction)
	if tx.UnmarshalBinary(raw.Transactions[0]) != nil {
		t.Fatal("tx")
	}
	block := types.NewBlockWithHeader(&h).WithBody(types.Body{Transactions: []*types.Transaction{tx}})
	encoded, e := rlp.EncodeToBytes(block)
	if e != nil {
		t.Fatal(e)
	}
	trace := nodeTestFrame(raw.Executions[0].Root)
	active := false
	r, _ := nodeFixture(t, func(method string, params []json.RawMessage, body []byte) []byte {
		if !active {
			return body
		}
		if method == "eth_getBlockByNumber" && string(params[0]) == `"0x0"` {
			return body
		}
		return nodeMutateResult(body, func(v *json.RawMessage) {
			switch method {
			case "eth_getBlockByNumber":
				*v, _ = json.Marshal(map[string]any{"hash": h.Hash().Hex(), "parentHash": h.ParentHash.Hex(), "number": "0xb6b80e", "timestamp": "0x0"})
			case "debug_getRawBlock":
				*v, _ = json.Marshal(nodeHex(encoded))
			case "debug_getRawReceipts":
				*v, _ = json.Marshal([]string{nodeHex(raw.Receipts[0])})
			case "debug_traceTransaction":
				*v, _ = json.Marshal(trace)
			}
		})
	})
	active = true
	r.header = replayHeader(&h)
	got, err := r.ReadNodeBlock(context.Background(), sepoliaDeploymentBlock, nodeTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Block.Executions) != 1 || got.Block.Executions[0].Root.Calls[0].From != replayTestWallet || got.Block.Executions[0].TransactionHash != tx.Hash().Hex() {
		t.Fatal("wallet caller")
	}
	if !got.Gas.LegacyGasCompatible {
		t.Fatal("owned fixture gas compatibility")
	}
	budget := replayBudget{limits: nodeTestLimits().Replay}
	if _, err := replayDecodeBlock(context.Background(), got.Block, &budget); err != nil {
		t.Fatal("strict owned block", err)
	}
	// Same inert fixture, inconsistent caller: public acquisition must refuse.
	trace["from"] = replayTestWallet
	if got, err := r.ReadNodeBlock(context.Background(), sepoliaDeploymentBlock, nodeTestLimits()); got != nil || err == nil {
		t.Fatal("trace/tx mismatch")
	}
}

func TestNodeCallTracerFieldsAndBounds(t *testing.T) {
	base := nodeTestFrame(ReplayFrame{Type: "CALL", From: replayTestWallet, To: SepoliaRegistryAddress, Value: "0x0", Input: []byte{}})
	base["logs"] = []any{map[string]any{"address": SepoliaRegistryAddress, "data": "0x", "topics": []any{}, "position": 0, "index": 0}}
	for _, kind := range []string{"CALL", "STATICCALL", "CALLCODE", "DELEGATECALL", "CREATE", "CREATE2"} {
		t.Run(kind, func(t *testing.T) {
			m := replayTestClone(t, base)
			m["type"] = kind
			m["output"] = "0x"
			m["executionGasUsed"] = "0x1"
			m["stateGasUsed"] = "0x0"
			m["gasRefund"] = "0x0"
			m["error"] = "reverted"
			m["revertReason"] = "fixture"
			if kind == "STATICCALL" {
				delete(m, "value")
			}
			b, _ := json.Marshal(m)
			f, e := DecodeCallTracer(context.Background(), b, replayTestLimits())
			if e != nil || f.Type != kind || f.Value != "0x0" {
				t.Fatal(e)
			}
		})
	}
	bad := map[string]func(map[string]any){
		"type": func(m map[string]any) { m["type"] = "SELFDESTRUCT" }, "type-null": func(m map[string]any) { m["type"] = nil }, "from": func(m map[string]any) { m["from"] = "bad" }, "to": func(m map[string]any) { delete(m, "to") }, "gas": func(m map[string]any) { m["gas"] = "0x00" }, "gas-used": func(m map[string]any) { delete(m, "gasUsed") }, "input": func(m map[string]any) { m["input"] = 1 }, "output": func(m map[string]any) { m["output"] = "nothex" }, "value": func(m map[string]any) { delete(m, "value") }, "value-bad": func(m map[string]any) { m["value"] = "0x00" }, "error": func(m map[string]any) { m["error"] = nil }, "error-long": func(m map[string]any) { m["error"] = strings.Repeat("x", 4097) }, "calls-null": func(m map[string]any) { m["calls"] = nil }, "logs-null": func(m map[string]any) { m["logs"] = nil }, "unknown": func(m map[string]any) { m["unknown"] = true }, "log-object": func(m map[string]any) { m["logs"] = []any{1} }, "log-unknown": func(m map[string]any) { m["logs"].([]any)[0].(map[string]any)["unknown"] = 1 }, "log-address": func(m map[string]any) { m["logs"].([]any)[0].(map[string]any)["address"] = "bad" }, "log-position": func(m map[string]any) { m["logs"].([]any)[0].(map[string]any)["position"] = 1 }, "log-index": func(m map[string]any) { m["logs"].([]any)[0].(map[string]any)["index"] = -1 }, "log-data": func(m map[string]any) { m["logs"].([]any)[0].(map[string]any)["data"] = "0x0" }, "topics-missing": func(m map[string]any) { delete(m["logs"].([]any)[0].(map[string]any), "topics") }, "topics-null": func(m map[string]any) { m["logs"].([]any)[0].(map[string]any)["topics"] = nil }, "topic": func(m map[string]any) { m["logs"].([]any)[0].(map[string]any)["topics"] = []any{"0x01"} }, "topics-cap": func(m map[string]any) { m["logs"].([]any)[0].(map[string]any)["topics"] = []any{1, 1, 1, 1, 1} }, "child": func(m map[string]any) { m["calls"] = []any{1} },
	}
	for name, mutate := range bad {
		t.Run(name, func(t *testing.T) {
			m := replayTestClone(t, base)
			mutate(m)
			b, _ := json.Marshal(m)
			if _, e := DecodeCallTracer(context.Background(), b, replayTestLimits()); e == nil {
				t.Fatal("accepted")
			}
		})
	}
	for _, raw := range []string{"", `null`, `[1]`, `{} {}`, `{"type":"CALL","type":"CALL"}`, `{"type":"\ud800"}`, string([]byte{'{', '"', 255, '"', ':', '1', '}'})} {
		if _, e := DecodeCallTracer(context.Background(), json.RawMessage(raw), replayTestLimits()); e == nil {
			t.Fatal(raw)
		}
	}
	for name, mutate := range map[string]func(*ReplayLimits){"bytes": func(l *ReplayLimits) { l.MaxEvidenceBytes = 1 }, "logs": func(l *ReplayLimits) { l.MaxLogs = 1 }, "frames": func(l *ReplayLimits) { l.MaxFrames = 1 }, "config": func(l *ReplayLimits) { l.MaxFrames = 0 }} {
		t.Run(name, func(t *testing.T) {
			l := replayTestLimits()
			mutate(&l)
			m := replayTestClone(t, base)
			m["logs"] = append(m["logs"].([]any), m["logs"].([]any)[0])
			m["calls"] = []any{nodeTestFrame(ReplayFrame{Type: "STATICCALL", From: SepoliaRegistryAddress, To: replayTestWallet, Value: "0x0"})}
			b, _ := json.Marshal(m)
			if _, e := DecodeCallTracer(context.Background(), b, l); e == nil {
				t.Fatal("budget")
			}
		})
	}
	m := base
	for range 34 {
		m = map[string]any{"type": "CALL", "from": replayTestWallet, "to": replayTestWallet, "input": "0x", "value": "0x0", "gas": "0x1", "gasUsed": "0x1", "calls": []any{m}}
	}
	b, _ := json.Marshal(m)
	if _, e := DecodeCallTracer(context.Background(), b, replayTestLimits()); e == nil {
		t.Fatal("depth")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := DecodeCallTracer(ctx, []byte(`{}`), replayTestLimits()); e != context.Canceled {
		t.Fatal(e)
	}
}

// Finite log index data exercise conversion on 32-bit and 64-bit consumers.
func TestNodeLogPositionConversionBound(t *testing.T) {
	for _, tc := range []struct {
		raw     string
		allowed bool
	}{
		{"65536", true}, {"65537", false}, {"4294967296", false}, {"18446744073709551615", false},
	} {
		raw := json.RawMessage(`{"address":"` + SepoliaRegistryAddress + `","topics":[],"data":"0x","position":` + tc.raw + `}`)
		got, ok := nodeLog(raw, 65536)
		if ok != tc.allowed || (ok && got.AfterChildren != 65536) {
			t.Fatal(tc.raw, ok, got.AfterChildren)
		}
	}
}
