package registry011

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// The fixture is the exact public response retained by the failed official-RPC
// CI, not a network producer or an assertion about a running node binary.
func TestErigonRecordedCreationCall(t *testing.T) {
	b, e := os.ReadFile("testdata/erigon-core-trace.json")
	h := sha256.Sum256(b)
	if e != nil || hex.EncodeToString(h[:]) != "7f5663dba3e4f5c427dbe91d2e40304481d394f1631f7ef83cf14ea0e706f54a" {
		t.Fatal("original trace identity", e)
	}
	raw, e := DecodeRPCResult(b, 1, RPCResponseLimits{4 << 20, 128})
	if e != nil {
		t.Fatal(e)
	}
	f, e := DecodeCallTracer(context.Background(), raw, nodeTestLimits().Replay)
	if e != nil || f.Type != "CREATE" || f.To != SepoliaRegistryAddress {
		t.Fatal("complete creation", e)
	}
	n := 0
	var walk func(ReplayFrame)
	walk = func(v ReplayFrame) {
		n++
		for _, c := range v.Calls {
			walk(c)
		}
	}
	walk(f)
	if n != 16 {
		t.Fatal("incomplete recorded call tree", n)
	}
	r, _ := nodeFixture(t, func(method string, _ []json.RawMessage, response []byte) []byte {
		if method == "debug_traceTransaction" {
			return b
		}
		return response
	})
	got, e := r.ObserveDeployment(context.Background(), nodeTestLimits())
	if e != nil || got == nil || got.Creation.Root.To != SepoliaRegistryAddress {
		t.Fatal("recorded creation joined to signed block and receipts", e)
	}
}

func erigonOwnedFrame() map[string]any {
	r := nodeTestFrame(ReplayFrame{Type: "CALL", From: replayTestWallet, To: SepoliaRegistryAddress, Value: "0x0"})
	c := nodeTestFrame(ReplayFrame{Type: "STATICCALL", From: SepoliaRegistryAddress, To: replayTestWallet, Value: "0x0"})
	r["regularGasUsed"], r["stateGasUsed"], r["gasRefund"], r["stateGasReservoir"] = "0x1", "0x0", "0x0", "0x10"
	c["stateGasUsed"], c["stateGasReservoir"] = "0x1", "0x8"
	r["calls"] = []any{c}
	return r
}

func TestErigonChildSignedDiagnosticBoundaries(t *testing.T) {
	for _, signed := range []string{"0x0", "0x1", "0x7fffffffffffffff", "-0x1", "-0x8000000000000000"} {
		t.Run(signed, func(t *testing.T) {
			m := erigonOwnedFrame()
			c := m["calls"].([]any)[0].(map[string]any)
			c["stateGasUsed"] = signed
			delete(c, "stateGasReservoir") // zero reservoirs are omitted by the source
			b, _ := json.Marshal(m)
			f, e := DecodeCallTracer(context.Background(), b, replayTestLimits())
			if e != nil || len(f.Calls) != 1 || f.Calls[0].From != SepoliaRegistryAddress {
				t.Fatal("diagnostic changed execution context", e)
			}
		})
	}
}

func TestErigonDiagnosticShapeRefusals(t *testing.T) {
	for name, edit := range map[string]func(map[string]any, map[string]any){
		"no-root-discriminator": func(r, c map[string]any) { delete(r, "stateGasReservoir") },
		"root-reservoir-type":   func(r, c map[string]any) { r["stateGasReservoir"] = 1 },
		"root-negative-state":   func(r, c map[string]any) { r["stateGasUsed"] = "-0x1" },
		"root-missing-refund":   func(r, c map[string]any) { delete(r, "gasRefund") },
		"child-missing-state":   func(r, c map[string]any) { delete(c, "stateGasUsed") },
		"child-refund":          func(r, c map[string]any) { c["gasRefund"] = "0x0" },
		"child-regular":         func(r, c map[string]any) { c["regularGasUsed"] = "0x1" },
		"child-execution":       func(r, c map[string]any) { c["executionGasUsed"] = "0x1" },
		"negative-zero":         func(r, c map[string]any) { c["stateGasUsed"] = "-0x0" },
		"positive-overflow":     func(r, c map[string]any) { c["stateGasUsed"] = "0x8000000000000000" },
		"negative-overflow":     func(r, c map[string]any) { c["stateGasUsed"] = "-0x8000000000000001" },
		"signed-type":           func(r, c map[string]any) { c["stateGasUsed"] = 1 },
		"signed-alias":          func(r, c map[string]any) { c["stateGasUsed"] = "-0x01" },
		"signed-quantity":       func(r, c map[string]any) { c["stateGasUsed"] = strings.Repeat("-", 2) + "0x1" },
	} {
		t.Run(name, func(t *testing.T) {
			m := erigonOwnedFrame()
			c := m["calls"].([]any)[0].(map[string]any)
			edit(m, c)
			b, _ := json.Marshal(m)
			if _, e := DecodeCallTracer(context.Background(), b, replayTestLimits()); e == nil {
				t.Fatal("mixed or malformed frame diagnostics accepted")
			}
		})
	}
}
