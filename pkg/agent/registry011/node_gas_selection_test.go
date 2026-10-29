package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

func TestStateGasSelectionKeepsFailedTransaction(t *testing.T) {
	r := gasFixture(t, nil)
	nodeJSONRoute(t, r, nil)
	out, err := r.ReadReplayBlock(context.Background(), sepoliaDeploymentBlock, nodeJSONLimits())
	if err != nil || out == nil || out.Block.Gas == nil || out.Block.Gas.Profile != SepoliaAmsterdamGasProfile {
		t.Fatal("complete transaction-result gas witness", err)
	}
	var rows []struct {
		TxHash string                     `json:"txHash"`
		Result map[string]json.RawMessage `json:"result"`
	}
	if json.Unmarshal(out.Block.Gas.Trace, &rows) != nil || len(rows) != 120 || rows[65].TxHash != "0xd541a15936c139f2906d088de07a75269aee329e175ec49418fa7f1327b3f24c" {
		t.Fatal("failed transaction was omitted or reordered")
	}
	for _, name := range []string{"gasUsed", "executionGasUsed", "stateGasUsed", "gasRefund"} {
		if _, valid := nodeString(rows[65].Result[name]); !valid {
			t.Fatal("failed transaction dimension missing", name)
		}
	}
	traces := 0
	for _, exchange := range out.Exchanges {
		if exchange.Method != "debug_traceBlockByHash" {
			continue
		}
		traces++
		var params []json.RawMessage
		var options map[string]json.RawMessage
		if json.Unmarshal(exchange.Params, &params) != nil || len(params) != 2 || json.Unmarshal(params[1], &options) != nil || string(options["tracer"]) != `"stateGasTracer"` || !bytes.Equal(exchange.Response, gasWire(t)) {
			t.Fatal("selected gas method or original response changed")
		}
	}
	if traces != 1 {
		t.Fatal("gas acquisition retried", traces)
	}
	if _, err := replayDecodeBlock(context.Background(), out.Block, &replayBudget{limits: nodeTestLimits().Replay}); err != nil {
		t.Fatal("strict replay refused complete unchanged evidence", err)
	}
}

func TestStateGasSelectionRefusesIncompleteFailureWithoutRetry(t *testing.T) {
	for _, name := range []string{"executionGasUsed", "stateGasUsed", "gasRefund", "unsupported"} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			r := gasFixture(t, func(method string, body []byte) []byte {
				if method != "debug_traceBlockByHash" {
					return body
				}
				calls++
				if name == "unsupported" {
					return []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"unsupported tracer"}}`)
				}
				return rootGasEdit(body, func(rows []map[string]any) []map[string]any {
					delete(rows[65]["result"].(map[string]any), name)
					return rows
				})
			})
			nodeJSONRoute(t, r, nil)
			if out, err := r.ReadReplayBlock(context.Background(), sepoliaDeploymentBlock, nodeJSONLimits()); out != nil || err == nil || calls != 1 {
				t.Fatal("incomplete failure was accepted or another tracer tried", err, calls)
			}
		})
	}
}

func TestRootGasSelectionRetainsRuntimeOmissionRefusal(t *testing.T) {
	raw, err := os.ReadFile("testdata/runtime-oog-root-rpc.json")
	digest := sha256.Sum256(raw)
	if err != nil || hex.EncodeToString(digest[:]) != "82f084b9df92012822b8a159b403bda6ddccb566bb1bec3f463550fa0ae51e6e" {
		t.Fatal("original finite runtime failure response", err)
	}
	var envelope struct {
		Result []struct{ Result map[string]json.RawMessage }
	}
	if json.Unmarshal(raw, &envelope) != nil || len(envelope.Result) != 120 {
		t.Fatal("original transaction count")
	}
	for i, row := range envelope.Result {
		for _, name := range []string{"executionGasUsed", "stateGasUsed", "gasRefund"} {
			_, present := row.Result[name]
			if present != (i != 65) {
				t.Fatal("retained omission changed", i, name)
			}
		}
	}
	r := rootGasFixture(t, func([]byte) []byte { return raw })
	nodeJSONRoute(t, r, nil)
	if out, err := r.ReadReplayBlockWithRootGas(context.Background(), sepoliaDeploymentBlock, nodeJSONLimits()); out != nil || err != ErrReplayUnavailable {
		t.Fatal("root-only path inferred missing dimensions", err)
	}
}
