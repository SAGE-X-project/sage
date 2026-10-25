package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func nodeJSONPut(m map[string]json.RawMessage, key string, value any) {
	m[key], _ = json.Marshal(value)
}

// Derive ordinary JSON from the existing finite raw fixture inside the test.
// Expected consensus bytes remain the independent original fixture, unchanged.
func nodeJSONFixtureBytes(t *testing.T) (*types.Block, []byte, []byte, [][]byte) {
	t.Helper()
	recorded := nodeRecorded(t)
	var env struct{ Result json.RawMessage }
	if json.Unmarshal(recorded["raw-block"], &env) != nil {
		t.Fatal("raw fixture envelope")
	}
	raw, ok := nodeData(env.Result)
	var block types.Block
	if !ok || rlp.DecodeBytes(raw, &block) != nil {
		t.Fatal("original RLP")
	}
	b, _ := json.Marshal(block.Header())
	var object map[string]json.RawMessage
	_ = json.Unmarshal(b, &object)
	rows := make([]map[string]json.RawMessage, 0, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		b, _ = json.Marshal(tx)
		var row map[string]json.RawMessage
		_ = json.Unmarshal(b, &row)
		sender, err := types.Sender(types.LatestSignerForChainID(big.NewInt(11155111)), tx)
		if err != nil {
			t.Fatal("original signed transaction", err)
		}
		nodeJSONPut(row, "from", replayAddress(sender))
		nodeJSONPut(row, "blockHash", block.Hash().Hex())
		nodeJSONPut(row, "blockNumber", fmt.Sprintf("0x%x", block.NumberU64()))
		nodeJSONPut(row, "blockTimestamp", fmt.Sprintf("0x%x", block.Time()))
		nodeJSONPut(row, "transactionIndex", fmt.Sprintf("0x%x", i))
		rows = append(rows, row)
	}
	nodeJSONPut(object, "transactions", rows)
	nodeJSONPut(object, "uncles", []string{})
	nodeJSONPut(object, "withdrawals", block.Withdrawals())
	nodeJSONPut(object, "size", fmt.Sprintf("0x%x", len(raw)))
	blockJSON, _ := json.Marshal(object)
	if json.Unmarshal(recorded["raw-receipts"], &env) != nil {
		t.Fatal("receipt envelope")
	}
	encoded, ok := nodeRawArray(env.Result, 1024)
	if !ok || len(encoded) != len(rows) {
		t.Fatal("original receipt count")
	}
	receipts := make([]*types.Receipt, 0, len(encoded))
	originals := make([][]byte, 0, len(encoded))
	var previous uint64
	var logIndex uint
	for i, row := range encoded {
		original, valid := nodeData(row)
		var receipt types.Receipt
		if !valid || receipt.UnmarshalBinary(original) != nil {
			t.Fatal("original receipt")
		}
		receipt.TxHash = block.Transactions()[i].Hash()
		receipt.BlockHash = block.Hash()
		receipt.BlockNumber = block.Number()
		receipt.TransactionIndex = uint(i)
		receipt.GasUsed = receipt.CumulativeGasUsed - previous
		previous = receipt.CumulativeGasUsed
		if receipt.Logs == nil {
			receipt.Logs = []*types.Log{}
		}
		for _, entry := range receipt.Logs {
			if entry.Topics == nil {
				entry.Topics = []common.Hash{}
			}
			entry.BlockHash, entry.BlockNumber = block.Hash(), block.NumberU64()
			entry.TxHash, entry.TxIndex, entry.Index = receipt.TxHash, uint(i), logIndex
			logIndex++
		}
		receipts = append(receipts, &receipt)
		originals = append(originals, original)
	}
	receiptJSON, _ := json.Marshal(receipts)
	var normalized []map[string]json.RawMessage
	_ = json.Unmarshal(receiptJSON, &normalized)
	for i, row := range normalized {
		nodeJSONPut(row, "type", fmt.Sprintf("0x%x", receipts[i].Type))
		logs, _ := nodeRawArray(row["logs"], 65536)
		for i, value := range logs {
			logs[i] = nodeJSONEditObject(value, func(m map[string]json.RawMessage) {
				if _, present := m["blockTimestamp"]; present {
					nodeJSONPut(m, "blockTimestamp", fmt.Sprintf("0x%x", block.Time()))
				}
			})
		}
		nodeJSONPut(row, "logs", logs)
	}
	receiptJSON, _ = json.Marshal(normalized)
	return &block, blockJSON, receiptJSON, originals
}

func nodeJSONRoute(t *testing.T, r *PinnedAgentReads, edit func(string, []byte) []byte) {
	t.Helper()
	_, block, receipts, _ := nodeJSONFixtureBytes(t)
	base := r.rpc.client.Transport
	r.rpc.client.Transport = rpcRoundTrip(func(req *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(b))
		var input struct {
			Method string
			Params []json.RawMessage
		}
		if json.Unmarshal(b, &input) != nil {
			t.Fatal("unit request")
		}
		if input.Method == "debug_getRawBlock" || input.Method == "debug_getRawReceipts" {
			t.Fatal("standard encoding retried a debug content method")
		}
		if input.Method != "eth_getBlockByHash" && input.Method != "eth_getBlockReceipts" {
			return base.RoundTrip(req)
		}
		if len(input.Params) == 0 || string(input.Params[0]) != `"0x54138f0dbccac32a1cd6fb0ed049fa013e78ca33910dd0471b09efea663d365e"` || input.Method == "eth_getBlockByHash" && (len(input.Params) != 2 || string(input.Params[1]) != "true") || input.Method == "eth_getBlockReceipts" && len(input.Params) != 1 {
			t.Fatal("fixed standard request")
		}
		value := block
		if input.Method == "eth_getBlockReceipts" {
			value = receipts
		}
		if edit != nil {
			value = edit(input.Method, bytes.Clone(value))
		}
		body, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "result": json.RawMessage(value)})
		return jsonResponse(string(body)), nil
	})
}

func nodeJSONLimits() NodeLimits {
	l := nodeTestLimits()
	l.BlockEncoding = EthereumJSONBlocks
	return l
}

func TestNodeJSONExactOriginalConsensusBytes(t *testing.T) {
	block, blockJSON, receiptsJSON, originals := nodeJSONFixtureBytes(t)
	limits := nodeTestLimits().Replay
	encoded, err := nodeJSONBlock(context.Background(), blockJSON, replayHeader(block.Header()), &replayBudget{limits: limits})
	want, _ := rlp.EncodeToBytes(block)
	if err != nil || !bytes.Equal(encoded, want) {
		t.Fatal("JSON changed original consensus block", err)
	}
	got, err := nodeJSONReceipts(context.Background(), receiptsJSON, block, &replayBudget{limits: limits})
	if err != nil || len(got) != len(originals) {
		t.Fatal("original receipts", err)
	}
	for i := range got {
		if !bytes.Equal(got[i], originals[i]) {
			t.Fatal("receipt consensus bytes changed", i)
		}
	}
	r, _ := nodeFixture(t, nil)
	nodeJSONRoute(t, r, nil)
	out, err := r.ObserveDeployment(context.Background(), nodeJSONLimits())
	if err != nil || out == nil || len(out.CodeHashes) != 13 || len(out.Block.Transactions) != 120 || out.Gas.LegacyGasCompatible {
		t.Fatal("standard deployment content", err)
	}
	seen := map[string]int{}
	for _, e := range out.Exchanges {
		seen[e.Method]++
	}
	if seen["eth_getBlockByHash"] != 1 || seen["eth_getBlockReceipts"] != 1 || seen["debug_getRawBlock"] != 0 || seen["debug_getRawReceipts"] != 0 {
		t.Fatal("encoding was not selected before acquisition", seen)
	}
	if _, err := replayDecodeBlock(context.Background(), out.Block, &replayBudget{limits: limits}); err == nil {
		t.Fatal("content conversion bypassed mandatory gas witness")
	}
}

func TestNodeJSONEffectivePriceMetadataAndParityAlias(t *testing.T) {
	block, raw, _, _ := nodeJSONFixtureBytes(t)
	var body map[string]json.RawMessage
	_ = json.Unmarshal(raw, &body)
	rows, _ := nodeRawArray(body["transactions"], 1024)
	for i, tx := range block.Transactions() {
		if tx.Type() < types.DynamicFeeTxType {
			continue
		}
		price := new(big.Int).Add(block.BaseFee(), tx.GasTipCap())
		if price.Cmp(tx.GasFeeCap()) > 0 {
			price.Set(tx.GasFeeCap())
		}
		rows[i] = nodeJSONEditObject(rows[i], func(m map[string]json.RawMessage) {
			nodeJSONPut(m, "gasPrice", "0x"+price.Text(16))
			delete(m, "yParity") // Explicit v preserves this documented alias.
		})
	}
	nodeJSONPut(body, "transactions", rows)
	nodeJSONPut(body, "totalDifficulty", "0x0")
	b, _ := json.Marshal(body)
	want, _ := rlp.EncodeToBytes(block)
	if out, err := nodeJSONBlock(context.Background(), b, replayHeader(block.Header()), &replayBudget{limits: nodeTestLimits().Replay}); err != nil || !bytes.Equal(out, want) {
		t.Fatal("standard dynamic-fee metadata changed signed body", err)
	}
	for _, value := range []json.RawMessage{[]byte("false"), []byte(`"0x00"`), []byte(`"0x0"`)} {
		bad := nodeJSONEditObject(b, func(m map[string]json.RawMessage) {
			for i, tx := range block.Transactions() {
				if tx.Type() >= types.DynamicFeeTxType {
					rows[i] = nodeJSONEditObject(rows[i], func(v map[string]json.RawMessage) { v["gasPrice"] = value })
					break
				}
			}
			nodeJSONPut(m, "transactions", rows)
		})
		if out, err := nodeJSONBlock(context.Background(), bad, replayHeader(block.Header()), &replayBudget{limits: nodeTestLimits().Replay}); out != nil || err == nil {
			t.Fatal("inconsistent dynamic price metadata")
		}
	}
}

func TestNodeJSONNestedArrayPreallocationBounds(t *testing.T) {
	hash := "0x" + strings.Repeat("1", 64)
	for _, field := range []string{"accessList", "blobVersionedHashes", "authorizationList"} {
		t.Run(field, func(t *testing.T) {
			var row any
			switch field {
			case "accessList":
				row = map[string]any{"address": replayTestWallet, "storageKeys": []string{hash}}
			case "blobVersionedHashes":
				row = hash
			case "authorizationList":
				row = map[string]any{"chainId": "0x1", "address": replayTestWallet, "nonce": "0x0", "yParity": "0x0", "r": "0x1", "s": "0x1"}
			}
			m := map[string]json.RawMessage{}
			nodeJSONPut(m, field, []any{row})
			if !nodeJSONTransactionLists(m) {
				t.Fatal("bounded valid list")
			}
			m[field] = json.RawMessage("true")
			if nodeJSONTransactionLists(m) {
				t.Fatal("non-list")
			}
			m[field] = json.RawMessage("[true]")
			if nodeJSONTransactionLists(m) {
				t.Fatal("invalid entry")
			}
			m[field] = json.RawMessage("[" + strings.Repeat("null,", 16384) + "null]")
			if nodeJSONTransactionLists(m) {
				t.Fatal("encoding list ceiling")
			}
		})
	}
	m := map[string]json.RawMessage{"accessList": []byte(`[{"address":"` + replayTestWallet + `","storageKeys":true}]`)}
	if nodeJSONTransactionLists(m) {
		t.Fatal("storage-key list")
	}
}

func nodeJSONEditObject(b []byte, edit func(map[string]json.RawMessage)) []byte {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(b, &m)
	edit(m)
	out, _ := json.Marshal(m)
	return out
}

func TestNodeJSONHeaderAndBodyRefusals(t *testing.T) {
	block, raw, _, _ := nodeJSONFixtureBytes(t)
	selected := replayHeader(block.Header())
	for _, key := range strings.Fields(nodeJSONHeaderFields + " hash transactions uncles withdrawals") {
		for _, mode := range []string{"missing", "null"} {
			t.Run(key+"-"+mode, func(t *testing.T) {
				b := nodeJSONEditObject(raw, func(m map[string]json.RawMessage) {
					if mode == "missing" {
						delete(m, key)
					} else {
						m[key] = json.RawMessage("null")
					}
				})
				if out, e := nodeJSONBlock(context.Background(), b, selected, &replayBudget{limits: nodeTestLimits().Replay}); out != nil || e == nil {
					t.Fatal("missing field became zero/default", key, mode)
				}
			})
		}
	}
	for name, edit := range map[string]func(map[string]json.RawMessage){
		"unknown":             func(m map[string]json.RawMessage) { m["futureRoot"] = json.RawMessage(`"0x0"`) },
		"long":                func(m map[string]json.RawMessage) { nodeJSONPut(m, "extraData", strings.Repeat("0", 1031)) },
		"header-codec":        func(m map[string]json.RawMessage) { m["miner"] = json.RawMessage("false") },
		"header-hash":         func(m map[string]json.RawMessage) { nodeJSONPut(m, "hash", "0x"+strings.Repeat("0", 64)) },
		"number":              func(m map[string]json.RawMessage) { nodeJSONPut(m, "number", "0x00") },
		"difficulty":          func(m map[string]json.RawMessage) { nodeJSONPut(m, "difficulty", "-0x1") },
		"uncles":              func(m map[string]json.RawMessage) { nodeJSONPut(m, "uncles", []string{"0x0"}) },
		"transactions":        func(m map[string]json.RawMessage) { m["transactions"] = json.RawMessage("{}") },
		"withdrawals":         func(m map[string]json.RawMessage) { m["withdrawals"] = json.RawMessage("false") },
		"withdrawal-root":     func(m map[string]json.RawMessage) { nodeJSONPut(m, "withdrawals", []any{}) },
		"size":                func(m map[string]json.RawMessage) { nodeJSONPut(m, "size", "0x0") },
		"difficulty-metadata": func(m map[string]json.RawMessage) { m["totalDifficulty"] = json.RawMessage("false") },
	} {
		t.Run(name, func(t *testing.T) {
			b := nodeJSONEditObject(raw, edit)
			if out, e := nodeJSONBlock(context.Background(), b, selected, &replayBudget{limits: nodeTestLimits().Replay}); out != nil || e == nil {
				t.Fatal("invalid standard block accepted", name)
			}
		})
	}
	for _, value := range []string{"", "[]", "{} {}", `{"x":1,"x":2}`} {
		if out, e := nodeJSONBlock(context.Background(), []byte(value), selected, &replayBudget{limits: nodeTestLimits().Replay}); out != nil || e == nil {
			t.Fatal("preflight", value)
		}
	}
	old := selected
	old.Timestamp = sepoliaAmsterdamTime - 1
	if out, e := nodeJSONBlock(context.Background(), raw, old, &replayBudget{limits: nodeTestLimits().Replay}); out != nil || e == nil {
		t.Fatal("unsupported header interval")
	}
}

func TestNodeJSONTransactionAndWithdrawalRefusals(t *testing.T) {
	block, raw, _, _ := nodeJSONFixtureBytes(t)
	for name, edit := range map[string]func(map[string]json.RawMessage){
		"unknown":      func(m map[string]json.RawMessage) { m["extension"] = json.RawMessage("true") },
		"missing-type": func(m map[string]json.RawMessage) { delete(m, "type") },
		"missing-to":   func(m map[string]json.RawMessage) { delete(m, "to") },
		"signature":    func(m map[string]json.RawMessage) { nodeJSONPut(m, "r", "0x0") },
		"sender":       func(m map[string]json.RawMessage) { nodeJSONPut(m, "from", replayTestWallet) },
		"block":        func(m map[string]json.RawMessage) { nodeJSONPut(m, "blockHash", "0x"+strings.Repeat("0", 64)) },
		"ordinal":      func(m map[string]json.RawMessage) { nodeJSONPut(m, "transactionIndex", "0x1") },
		"timestamp":    func(m map[string]json.RawMessage) { nodeJSONPut(m, "blockTimestamp", "0x0") },
		"signed-field": func(m map[string]json.RawMessage) { nodeJSONPut(m, "value", "0x1") },
		"noncanonical": func(m map[string]json.RawMessage) { nodeJSONPut(m, "gas", "0x0249f0") },
	} {
		t.Run(name, func(t *testing.T) {
			b := nodeJSONEditObject(raw, func(m map[string]json.RawMessage) {
				rows, _ := nodeRawArray(m["transactions"], 1024)
				rows[0] = nodeJSONEditObject(rows[0], edit)
				nodeJSONPut(m, "transactions", rows)
			})
			if out, e := nodeJSONBlock(context.Background(), b, replayHeader(block.Header()), &replayBudget{limits: nodeTestLimits().Replay}); out != nil || e == nil {
				t.Fatal("transaction mismatch accepted", name)
			}
		})
	}
	for _, kind := range []string{"missing", "unknown", "codec", "changed"} {
		t.Run("withdrawal-"+kind, func(t *testing.T) {
			b := nodeJSONEditObject(raw, func(m map[string]json.RawMessage) {
				rows, _ := nodeRawArray(m["withdrawals"], 16)
				rows[0] = nodeJSONEditObject(rows[0], func(v map[string]json.RawMessage) {
					switch kind {
					case "missing":
						delete(v, "amount")
					case "unknown":
						v["other"] = json.RawMessage("true")
					case "codec":
						v["index"] = json.RawMessage("true")
					case "changed":
						nodeJSONPut(v, "amount", "0x0")
					}
				})
				nodeJSONPut(m, "withdrawals", rows)
			})
			if out, e := nodeJSONBlock(context.Background(), b, replayHeader(block.Header()), &replayBudget{limits: nodeTestLimits().Replay}); out != nil || e == nil {
				t.Fatal("withdrawal mismatch", kind)
			}
		})
	}
}

func TestNodeJSONReceiptAndLogRefusals(t *testing.T) {
	block, _, raw, _ := nodeJSONFixtureBytes(t)
	for name, edit := range map[string]func(map[string]json.RawMessage){
		"unknown": func(m map[string]json.RawMessage) { m["extension"] = json.RawMessage("true") },
		"missing": func(m map[string]json.RawMessage) { delete(m, "status") },
		"type":    func(m map[string]json.RawMessage) { nodeJSONPut(m, "type", "0x7f") },
		"status":  func(m map[string]json.RawMessage) { nodeJSONPut(m, "status", "0x2") },
		"ordinal": func(m map[string]json.RawMessage) { nodeJSONPut(m, "transactionIndex", "0x1") },
		"hash":    func(m map[string]json.RawMessage) { nodeJSONPut(m, "transactionHash", "0x"+strings.Repeat("0", 64)) },
		"paid":    func(m map[string]json.RawMessage) { nodeJSONPut(m, "gasUsed", "0x0") },
		"logs":    func(m map[string]json.RawMessage) { m["logs"] = json.RawMessage("{}") },
	} {
		t.Run(name, func(t *testing.T) {
			rows, _ := nodeRawArray(raw, 1024)
			rows[0] = nodeJSONEditObject(rows[0], edit)
			b, _ := json.Marshal(rows)
			if out, e := nodeJSONReceipts(context.Background(), b, block, &replayBudget{limits: nodeTestLimits().Replay}); out != nil || e == nil {
				t.Fatal("receipt mismatch", name)
			}
		})
	}
	for _, name := range []string{"missing", "unknown", "removed", "index", "block"} {
		t.Run("log-"+name, func(t *testing.T) {
			rows, _ := nodeRawArray(raw, 1024)
			rows[0] = nodeJSONEditObject(rows[0], func(m map[string]json.RawMessage) {
				logs, _ := nodeRawArray(m["logs"], 4096)
				if len(logs) == 0 {
					t.Fatal("normal first receipt log missing")
				}
				logs[0] = nodeJSONEditObject(logs[0], func(v map[string]json.RawMessage) {
					switch name {
					case "missing":
						delete(v, "blockHash")
					case "unknown":
						v["extra"] = json.RawMessage("true")
					case "removed":
						v["removed"] = json.RawMessage("true")
					case "index":
						nodeJSONPut(v, "logIndex", "0x1")
					case "block":
						nodeJSONPut(v, "blockNumber", "0x0")
					}
				})
				nodeJSONPut(m, "logs", logs)
			})
			b, _ := json.Marshal(rows)
			if out, e := nodeJSONReceipts(context.Background(), b, block, &replayBudget{limits: nodeTestLimits().Replay}); out != nil || e == nil {
				t.Fatal("unbound log", name)
			}
		})
	}
	for _, value := range []string{"[]", "null", "[false]"} {
		if out, e := nodeJSONReceipts(context.Background(), []byte(value), block, &replayBudget{limits: nodeTestLimits().Replay}); out != nil || e == nil {
			t.Fatal("missing receipt array")
		}
	}
}

func TestNodeJSONLimitsCanonicalHelperAndNoFallback(t *testing.T) {
	block, b, receipts, _ := nodeJSONFixtureBytes(t)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, ctx := range []context.Context{nil, canceled} {
		if out, e := nodeJSONBlock(ctx, b, replayHeader(block.Header()), &replayBudget{limits: nodeTestLimits().Replay}); out != nil || e == nil {
			t.Fatal("context")
		}
		if out, e := nodeJSONReceipts(ctx, receipts, block, &replayBudget{limits: nodeTestLimits().Replay}); out != nil || e == nil {
			t.Fatal("receipt context")
		}
	}
	for _, kind := range []string{"bytes", "transactions", "logs"} {
		l := nodeTestLimits().Replay
		switch kind {
		case "bytes":
			l.MaxEvidenceBytes = 1
		case "transactions":
			l.MaxTransactions = 1
		case "logs":
			l.MaxLogs = 0
		}
		if kind != "logs" {
			if out, e := nodeJSONBlock(context.Background(), b, replayHeader(block.Header()), &replayBudget{limits: l}); out != nil || e == nil {
				t.Fatal("block limit", kind)
			}
		}
		if out, e := nodeJSONReceipts(context.Background(), receipts, block, &replayBudget{limits: l}); out != nil || e == nil {
			t.Fatal("receipt limit", kind)
		}
	}
	if nodeJSONPreflight(context.Background(), b, nil) || nodeJSONCanonical(nil, func() {}, "") || nodeJSONCanonical(nil, 1, "") || nodeJSONCanonical(nil, nil, "") {
		t.Fatal("invalid helper input")
	}
	for _, input := range []struct {
		raw     []byte
		receipt bool
	}{{b, false}, {receipts, true}} {
		limits := nodeTestLimits().Replay
		limits.MaxEvidenceBytes = len(input.raw)
		if input.receipt {
			if out, e := nodeJSONReceipts(context.Background(), input.raw, block, &replayBudget{limits: limits}); out != nil || e == nil {
				t.Fatal("receipt encoding work budget")
			}
		} else {
			if out, e := nodeJSONBlock(context.Background(), input.raw, replayHeader(block.Header()), &replayBudget{limits: limits}); out != nil || e == nil {
				t.Fatal("block encoding work budget")
			}
		}
	}
	r, calls := nodeFixture(t, nil)
	l := nodeTestLimits()
	l.BlockEncoding = "unknown"
	before := *calls
	if out, e := r.ReadNodeBlock(context.Background(), sepoliaDeploymentBlock, l); out != nil || e != ErrConfiguration || *calls != before {
		t.Fatal("unknown encoding issued request")
	}
	for _, method := range []string{"eth_getBlockByHash", "eth_getBlockReceipts"} {
		t.Run(method, func(t *testing.T) {
			r, _ := nodeFixture(t, nil)
			failed := 0
			nodeJSONRoute(t, r, func(got string, b []byte) []byte {
				if got == method {
					failed++
					return []byte("null")
				}
				return b
			})
			if out, e := r.ObserveDeployment(context.Background(), nodeJSONLimits()); out != nil || e == nil || failed != 1 {
				t.Fatal("standard failure retried or passed", e, failed)
			}
		})
	}
}

func TestNodeJSONStrictGasRequirementUnchanged(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprint(missing), func(t *testing.T) {
			r := rootGasFixture(t, func(original []byte) []byte {
				if missing {
					b, e := os.ReadFile("testdata/geth-root-gas.json")
					if e != nil {
						t.Fatal(e)
					}
					return b
				}
				return original
			})
			nodeJSONRoute(t, r, nil)
			out, e := r.ReadReplayBlockWithRootGas(context.Background(), sepoliaDeploymentBlock, nodeJSONLimits())
			if missing {
				if out != nil || e != ErrReplayUnavailable {
					t.Fatal("missing dimensions accepted", e)
				}
				return
			}
			if e != nil || out == nil || out.Block.Gas == nil {
				t.Fatal("complete original gas witness", e)
			}
		})
	}
}

// Only the positive standard-content exchange uses an owned TLS server. All
// malformed/omission scenarios above are inert units, never network producers.
func TestNodeJSONOwnedTLSRuntime(t *testing.T) {
	_, block, receipts, _ := nodeJSONFixtureBytes(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var in struct{ Method string }
		if json.NewDecoder(req.Body).Decode(&in) != nil {
			t.Error("normal request")
			return
		}
		value := block
		if in.Method == "eth_getBlockReceipts" {
			value = receipts
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": json.RawMessage(value)})
	}))
	defer server.Close()
	r, _ := nodeFixture(t, nil)
	base := r.rpc.client.Transport
	local := server.Client()
	r.rpc.client.Transport = rpcRoundTrip(func(req *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(b))
		var in struct{ Method string }
		_ = json.Unmarshal(b, &in)
		if in.Method != "eth_getBlockByHash" && in.Method != "eth_getBlockReceipts" {
			return base.RoundTrip(req)
		}
		clone := req.Clone(req.Context())
		u := *clone.URL
		clone.URL = &u
		clone.URL.Host = strings.TrimPrefix(server.URL, "https://")
		clone.Host = clone.URL.Host
		return local.Transport.RoundTrip(clone)
	})
	defer local.CloseIdleConnections()
	if out, e := r.ObserveDeployment(context.Background(), nodeJSONLimits()); e != nil || out == nil {
		t.Fatal("normal owned TLS", e)
	}
}
