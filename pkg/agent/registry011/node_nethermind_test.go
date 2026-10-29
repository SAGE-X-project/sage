package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

type nethermindGasRow struct {
	TxHash string            `json:"txHash"`
	Result map[string]string `json:"result"`
}

func TestNodeRecordedNethermindCallMetadata(t *testing.T) {
	// Untouched public response from the failed official-only e9 CI. This
	// proves offline adapter compatibility, not fresh state/finality or code.
	raw, e := os.ReadFile("testdata/node-deployment-nethermind-call-rpc.json")
	if e != nil {
		t.Fatal(e)
	}
	if len(raw) != 253142 || fmt.Sprintf("%x", sha256.Sum256(raw)) != "08eadb45ba0024b4b3374ad1ce6919354f2535efabfc67f29670608f8f79346e" {
		t.Fatal("original captured bytes")
	}
	r, _ := nodeFixture(t, func(method string, _ []json.RawMessage, body []byte) []byte {
		if method == "debug_traceTransaction" {
			return bytes.Clone(raw)
		}
		return body
	})
	got, e := r.ObserveDeployment(context.Background(), nodeTestLimits())
	if e != nil || got.Creation.Root.Type != "CREATE" || len(got.CodeHashes) != 13 {
		t.Fatal(e)
	}
	value, e := DecodeRPCResult(raw, 1, RPCResponseLimits{4 << 20, 128})
	if e != nil {
		t.Fatal(e)
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(value, &root) != nil {
		t.Fatal("trace")
	}
	for name, edit := range map[string]func(map[string]json.RawMessage){
		"hybrid":          func(m map[string]json.RawMessage) { m["executionGasUsed"] = m["regularGasUsed"] },
		"missing-state":   func(m map[string]json.RawMessage) { delete(m, "stateGasUsed") },
		"missing-refund":  func(m map[string]json.RawMessage) { delete(m, "gasRefund") },
		"missing-regular": func(m map[string]json.RawMessage) { delete(m, "regularGasUsed") },
		"bad-quantity":    func(m map[string]json.RawMessage) { m["regularGasUsed"] = json.RawMessage(`"0x00"`) },
	} {
		t.Run(name, func(t *testing.T) {
			m := replayTestClone(t, root)
			edit(m)
			bad, _ := json.Marshal(m)
			if _, e := DecodeCallTracer(context.Background(), bad, replayTestLimits()); e == nil {
				t.Fatal("accepted inconsistent optional metadata")
			}
		})
	}
}

func TestNethermindGasSchemaAndRawRefundBoundary(t *testing.T) {
	// Controlled FIELD-NAME changes to the previously pinned Reth sample test
	// the known mathematical overlap only. This is NOT an observed Nethermind
	// 120-row witness; actual collection/qualification remains a separate gate.
	r := gasFixture(t, func(method string, body []byte) []byte {
		if method == "debug_traceBlockByHash" {
			return bytes.ReplaceAll(body, []byte(`"executionGasUsed"`), []byte(`"regularGasUsed"`))
		}
		return body
	})
	out, e := r.ReadReplayBlock(context.Background(), sepoliaDeploymentBlock, nodeTestLimits())
	if e != nil || out.Block.Gas.Profile != SepoliaAmsterdamNethermindGasProfile {
		t.Fatal(e)
	}
	raw := out.Block
	b := replayBudget{limits: replayTestLimits()}
	decoded, e := replayDecodeBlock(context.Background(), raw, &b)
	if e != nil {
		t.Fatal(e)
	}
	var rows []nethermindGasRow
	if json.Unmarshal(raw.Gas.Trace, &rows) != nil {
		t.Fatal("shape")
	}
	floorIndex := -1
	for i, row := range rows {
		p, _ := rpcQuantity(row.Result["gasUsed"])
		f, e := replayGasFloor(decoded.txs[i])
		if e != nil {
			t.Fatal(e)
		}
		if p == f && row.Result["stateGasUsed"] == "0x0" && row.Result["gasRefund"] == "0x0" {
			floorIndex = i
			break
		}
	}
	if floorIndex < 0 {
		t.Fatal("missing known zero-state zero-refund floor fixture")
	}
	for name, edit := range map[string]func([]nethermindGasRow){
		"mixed-profile": func(v []nethermindGasRow) {
			v[1].Result["executionGasUsed"] = v[1].Result["regularGasUsed"]
			delete(v[1].Result, "regularGasUsed")
		},
		"raw-refund-at-floor": func(v []nethermindGasRow) { v[floorIndex].Result["gasRefund"] = "0x1" },
		"state-at-floor":      func(v []nethermindGasRow) { v[floorIndex].Result["stateGasUsed"] = "0x1" },
		"hybrid":              func(v []nethermindGasRow) { v[0].Result["executionGasUsed"] = v[0].Result["regularGasUsed"] },
	} {
		t.Run(name, func(t *testing.T) {
			x := replayTestClone(t, raw)
			v := replayTestClone(t, rows)
			edit(v)
			x.Gas.Trace, _ = json.Marshal(v)
			b := replayBudget{limits: replayTestLimits()}
			if _, e := replayDecodeBlock(context.Background(), x, &b); e == nil {
				t.Fatal("accepted incompatible raw refund or mixed data")
			}
		})
	}
}

func TestGasProfileClassificationIsClosedDataOnly(t *testing.T) {
	for name, raw := range map[string]string{"syntax": "[", "scalar": "0", "wrong-entry": `[{}]`, "unknown-entry": `[{"txHash":"x","result":{},"other":0}]`, "unknown-result": `[{"txHash":"x","result":{"unknown":0}}]`, "both": `[{"txHash":"x","result":{"gasUsed":"0x0","executionGasUsed":"0x0","regularGasUsed":"0x0","stateGasUsed":"0x0","gasRefund":"0x0"}}]`, "neither": `[{"txHash":"x","result":{"gasUsed":"0x0","stateGasUsed":"0x0","gasRefund":"0x0"}}]`} {
		t.Run(name, func(t *testing.T) {
			if _, e := replayGasProfile(json.RawMessage(raw), 1); e == nil {
				t.Fatal("accepted unknown profile")
			}
		})
	}
	if p, e := replayGasProfile(json.RawMessage(`[]`), 1); e != nil || p != SepoliaAmsterdamGasProfile {
		t.Fatal(e)
	}
	known := `[{"txHash":"x","result":{"gasUsed":"0x0","executionGasUsed":"0x0","stateGasUsed":"0x0","gasRefund":"0x0"}}]`
	if p, e := replayGasProfile(json.RawMessage(known), 1); e != nil || p != SepoliaAmsterdamGasProfile {
		t.Fatal(e)
	}
	if p, e := replayGasProfile(json.RawMessage(strings.ReplaceAll(known, "executionGasUsed", "regularGasUsed")), 1); e != nil || p != SepoliaAmsterdamNethermindGasProfile {
		t.Fatal(e)
	}
}
