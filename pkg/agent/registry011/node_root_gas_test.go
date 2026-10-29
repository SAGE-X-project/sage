package registry011

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func rootGasWire(t *testing.T) []byte {
	t.Helper()
	b, e := os.ReadFile("testdata/erigon-root-gas.json")
	h := sha256.Sum256(b)
	if e != nil || hex.EncodeToString(h[:]) != "440ffdef20cc9b5389d795006673c1d5859c47a0306243fcca0cd4f17b07e75b" {
		t.Fatal("original root trace", e)
	}
	return b
}

// This transport only returns finite recorded data; it cannot target a service.
func rootGasFixture(t *testing.T, mutate func([]byte) []byte) *PinnedAgentReads {
	t.Helper()
	r, _ := nodeFixture(t, nil)
	base := r.rpc.client.Transport
	r.rpc.client.Transport = rpcRoundTrip(func(req *http.Request) (*http.Response, error) {
		b, e := io.ReadAll(req.Body)
		if e != nil {
			return nil, e
		}
		var request struct {
			Method string
			Params []json.RawMessage
		}
		if json.Unmarshal(b, &request) != nil {
			t.Fatal("request")
		}
		req.Body = io.NopCloser(bytes.NewReader(b))
		if request.Method != "debug_traceBlockByHash" {
			return base.RoundTrip(req)
		}
		var opts map[string]any
		if len(request.Params) != 2 || json.Unmarshal(request.Params[1], &opts) != nil {
			t.Fatal("root request framing")
		}
		config, ok := opts["tracerConfig"].(map[string]any)
		if string(request.Params[0]) != `"0x54138f0dbccac32a1cd6fb0ed049fa013e78ca33910dd0471b09efea663d365e"` || len(opts) != 4 || opts["tracer"] != "callTracer" || opts["timeout"] != "2s" || opts["reexec"] != float64(0) || !ok || len(config) != 3 || config["onlyTopCall"] != true || config["withLog"] != false || config["includePrecompiles"] != true {
			t.Fatal("unexpected gas read or implicit fallback")
		}
		body := rootGasWire(t)
		if mutate != nil {
			body = mutate(body)
		}
		return jsonResponse(string(body)), nil
	})
	return r
}

func rootGasEdit(body []byte, edit func([]map[string]any) []map[string]any) []byte {
	var envelope map[string]json.RawMessage
	_ = json.Unmarshal(body, &envelope)
	var rows []map[string]any
	_ = json.Unmarshal(envelope["result"], &rows)
	envelope["result"], _ = json.Marshal(edit(rows))
	b, _ := json.Marshal(envelope)
	return b
}

func TestRootGasRecordedTransactionReceiptAgreement(t *testing.T) {
	r := rootGasFixture(t, nil)
	out, e := r.ReadReplayBlockWithRootGas(context.Background(), sepoliaDeploymentBlock, nodeTestLimits())
	if e != nil || out == nil {
		t.Fatal(e)
	}
	if out.Block.Gas.Profile != SepoliaAmsterdamRootRegularGasProfile || len(out.Block.Transactions) != 120 || out.Gas.LegacyGasCompatible {
		t.Fatal("wrong observation contract")
	}
	var rows []struct{ Result map[string]string }
	if json.Unmarshal(out.Block.Gas.Trace, &rows) != nil || len(rows) != 120 {
		t.Fatal("derived witness")
	}
	for _, row := range rows {
		if len(row.Result) != 4 {
			t.Fatal("root input was not separated from derived dimensions")
		}
	}
	original := false
	for _, exchange := range out.Exchanges {
		if exchange.Method == "debug_traceBlockByHash" {
			original = bytes.Equal(exchange.Response, rootGasWire(t))
		}
	}
	if !original {
		t.Fatal("original root response lost")
	}
	if _, e := replayDecodeBlock(context.Background(), out.Block, &replayBudget{limits: nodeTestLimits().Replay}); e != nil {
		t.Fatal("full signed block and receipts", e)
	}
}

func TestRootGasExecutionAliasIsSeparateConservativeProfile(t *testing.T) {
	// Synthetic schema compatibility, not an observation of another node binary.
	r := rootGasFixture(t, func(b []byte) []byte {
		return rootGasEdit(b, func(rows []map[string]any) []map[string]any {
			for _, row := range rows {
				v := row["result"].(map[string]any)
				v["executionGasUsed"] = v["regularGasUsed"]
				delete(v, "regularGasUsed")
			}
			return rows
		})
	})
	out, e := r.ReadReplayBlockWithRootGas(context.Background(), sepoliaDeploymentBlock, nodeTestLimits())
	if e != nil || out.Block.Gas.Profile != SepoliaAmsterdamRootExecutionGasProfile {
		t.Fatal("profile or strict settlement", e)
	}
}

func TestRootGasRefusesChangedRootAndMixedDimensions(t *testing.T) {
	for name, edit := range map[string]func(map[string]any, map[string]any){
		"hash":              func(row, v map[string]any) { row["txHash"] = "0x" + strings.Repeat("a", 64) },
		"row-field":         func(row, v map[string]any) { row["extra"] = true },
		"nested-calls":      func(row, v map[string]any) { v["calls"] = []any{} },
		"unknown-root":      func(row, v map[string]any) { v["extra"] = true },
		"from":              func(row, v map[string]any) { v["from"] = replayTestWallet },
		"type":              func(row, v map[string]any) { v["type"] = "STATICCALL" },
		"gas":               func(row, v map[string]any) { v["gas"] = "0x0" },
		"input":             func(row, v map[string]any) { v["input"] = "0x" },
		"value":             func(row, v map[string]any) { v["value"] = "0x1" },
		"to":                func(row, v map[string]any) { v["to"] = SepoliaRegistryAddress },
		"output":            func(row, v map[string]any) { v["output"] = true },
		"success-error":     func(row, v map[string]any) { v["error"] = "not a receipt success" },
		"error-type":        func(row, v map[string]any) { v["error"] = true },
		"reservoir":         func(row, v map[string]any) { v["stateGasReservoir"] = "-0x1" },
		"no-dimension":      func(row, v map[string]any) { delete(v, "regularGasUsed") },
		"two-dimensions":    func(row, v map[string]any) { v["executionGasUsed"] = v["regularGasUsed"] },
		"invalid-dimension": func(row, v map[string]any) { v["stateGasUsed"] = "0x00" },
		"missing-refund":    func(row, v map[string]any) { delete(v, "gasRefund") },
	} {
		t.Run(name, func(t *testing.T) {
			r := rootGasFixture(t, func(b []byte) []byte {
				return rootGasEdit(b, func(rows []map[string]any) []map[string]any {
					edit(rows[0], rows[0]["result"].(map[string]any))
					return rows
				})
			})
			if out, e := r.ReadReplayBlockWithRootGas(context.Background(), sepoliaDeploymentBlock, nodeTestLimits()); e == nil || out != nil {
				t.Fatal("unbound or ambiguous gas root accepted")
			}
		})
	}
	for _, mode := range []string{"missing", "mixed", "empty-response", "row-type"} {
		t.Run(mode, func(t *testing.T) {
			r := rootGasFixture(t, func(b []byte) []byte {
				if mode == "empty-response" {
					return []byte(`{"jsonrpc":"2.0","id":1,"result":null}`)
				}
				if mode == "row-type" {
					return []byte(`{"jsonrpc":"2.0","id":1,"result":[1]}`)
				}
				return rootGasEdit(b, func(rows []map[string]any) []map[string]any {
					if mode == "missing" {
						return rows[:len(rows)-1]
					}
					v := rows[1]["result"].(map[string]any)
					v["executionGasUsed"] = v["regularGasUsed"]
					delete(v, "regularGasUsed")
					return rows
				})
			})
			if out, e := r.ReadReplayBlockWithRootGas(context.Background(), sepoliaDeploymentBlock, nodeTestLimits()); e == nil || out != nil {
				t.Fatal("partial or mixed root list accepted")
			}
		})
	}
}

func TestRootGasRefusesAmbiguousPaymentFloor(t *testing.T) {
	r := rootGasFixture(t, func(b []byte) []byte {
		return rootGasEdit(b, func(rows []map[string]any) []map[string]any {
			// Recorded simple transfer rows at the minimum have paid==execution,
			// no state usage and no refund. A nonzero raw refund is not inferred.
			for _, row := range rows {
				v := row["result"].(map[string]any)
				if v["gasUsed"] == "0x3a98" && v["stateGasUsed"] == "0x0" {
					v["gasRefund"] = "0x1"
					return rows
				}
			}
			t.Fatal("recorded floor row missing")
			return nil
		})
	})
	if out, e := r.ReadReplayBlockWithRootGas(context.Background(), sepoliaDeploymentBlock, nodeTestLimits()); e == nil || out != nil {
		t.Fatal("ambiguous floor accepted")
	}
}

func TestRootGasParserBudgetCancellationAndCreateDestination(t *testing.T) {
	raw := gasRecorded(t)
	budget := replayBudget{limits: replayTestLimits()}
	block, e := replayDecodeBlock(context.Background(), raw, &budget)
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, value := range []json.RawMessage{nil, []byte("[] []"), []byte(`[{"txHash":1,"txHash":2}]`)} {
		if _, _, e := nodeRootGas(context.Background(), value, block, &replayBudget{limits: replayTestLimits()}); e == nil {
			t.Fatal("invalid raw list accepted")
		}
	}
	if _, _, e := nodeRootGas(ctx, []byte("[]"), block, &replayBudget{limits: replayTestLimits()}); e == nil {
		t.Fatal("canceled")
	}
	tiny := replayTestLimits()
	tiny.MaxEvidenceBytes = 1
	if _, _, e := nodeRootGas(context.Background(), []byte("[]"), block, &replayBudget{limits: tiny}); e == nil {
		t.Fatal("byte budget")
	}
	var outer struct{ Result []map[string]json.RawMessage }
	if json.Unmarshal(rootGasWire(t), &outer) != nil {
		t.Fatal("fixture")
	}
	var creation map[string]json.RawMessage
	if json.Unmarshal(outer.Result[62]["result"], &creation) != nil {
		t.Fatal("creation")
	}
	if !nodeRootContext(creation, block.txs[62], 1) {
		t.Fatal("creation context")
	}
	delete(creation, "to")
	if nodeRootContext(creation, block.txs[62], 1) {
		t.Fatal("successful create destination missing")
	}
	creation["error"] = json.RawMessage(`"execution reverted"`)
	if !nodeRootContext(creation, block.txs[62], 0) {
		t.Fatal("failed create may omit destination")
	}
	creation["to"] = json.RawMessage(`"` + replayTestWallet + `"`)
	if nodeRootContext(creation, block.txs[62], 0) {
		t.Fatal("wrong failed create destination")
	}
}

func TestRootGasRequiresExplicitReceiptOutcome(t *testing.T) {
	for _, kind := range []string{"missing", "empty"} {
		t.Run(kind, func(t *testing.T) {
			r := rootGasFixture(t, func(b []byte) []byte {
				return rootGasEdit(b, func(rows []map[string]any) []map[string]any {
					for _, row := range rows {
						v := row["result"].(map[string]any)
						if _, failed := v["error"]; failed {
							if kind == "missing" {
								delete(v, "error")
							} else {
								v["error"] = ""
							}
							return rows
						}
					}
					t.Fatal("recorded receipt failure missing")
					return nil
				})
			})
			if out, e := r.ReadReplayBlockWithRootGas(context.Background(), sepoliaDeploymentBlock, nodeTestLimits()); e == nil || out != nil {
				t.Fatal("receipt outcome not bound")
			}
		})
	}
}

// The approved endpoint returned this finite original response in CI. Its
// ordinary callTracer frames contain no execution/state/refund dimensions.
// Preserve refusal rather than inventing those values or retrying a tracer.
func TestRootGasRecordedMissingDimensionsRemainUnavailable(t *testing.T) {
	b, err := os.ReadFile("testdata/geth-root-gas.json")
	h := sha256.Sum256(b)
	if err != nil || hex.EncodeToString(h[:]) != "062123b94d9281906d131f496be06691a5af2385577c4fa99ea7c86c344ee2c1" {
		t.Fatal("original dimensionless root trace", err)
	}
	calls := 0
	r := rootGasFixture(t, func([]byte) []byte {
		calls++
		return b
	})
	if out, err := r.ReadReplayBlockWithRootGas(context.Background(), sepoliaDeploymentBlock, nodeTestLimits()); out != nil || err != ErrReplayUnavailable || calls != 1 {
		t.Fatal("missing dimensions accepted or implicitly retried", err, calls)
	}
}
