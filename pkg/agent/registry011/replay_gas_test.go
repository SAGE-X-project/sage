package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

func gasWire(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/node-deployment-gas-rpc.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func gasFixture(t *testing.T, mutate func(string, []byte) []byte) *PinnedAgentReads {
	t.Helper()
	r, _ := nodeFixture(t, nil)
	base := r.rpc.client.Transport
	r.rpc.client.Transport = rpcRoundTrip(func(req *http.Request) (*http.Response, error) {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		var in struct {
			Method string
			Params []json.RawMessage
		}
		if json.Unmarshal(b, &in) != nil {
			t.Fatal("request")
		}
		req.Body = io.NopCloser(bytes.NewReader(b))
		var body []byte
		if in.Method == "debug_traceBlockByHash" {
			var opt map[string]any
			_ = json.Unmarshal(in.Params[1], &opt)
			if string(in.Params[0]) != `"0x54138f0dbccac32a1cd6fb0ed049fa013e78ca33910dd0471b09efea663d365e"` || len(opt) != 3 || opt["tracer"] != "stateGasTracer" || opt["timeout"] != "2s" || opt["reexec"] != float64(0) {
				t.Fatal("gas acquisition binding")
			}
			body = gasWire(t)
		} else {
			response, e := base.RoundTrip(req)
			if e != nil {
				return nil, e
			}
			body, err = io.ReadAll(response.Body)
			_ = response.Body.Close()
			if err != nil {
				return nil, err
			}
		}
		if mutate != nil {
			body = mutate(in.Method, body)
		}
		return jsonResponse(string(body)), nil
	})
	return r
}

func gasRecorded(t *testing.T) ReplayBlock {
	t.Helper()
	r := gasFixture(t, nil)
	out, err := r.ReadReplayBlock(context.Background(), sepoliaDeploymentBlock, nodeTestLimits())
	if err != nil {
		t.Fatal(err)
	}
	if out.Block.Gas == nil || out.Block.Gas.Profile != SepoliaAmsterdamGasProfile || len(out.Block.Transactions) != 120 || out.Gas.LegacyGasCompatible {
		t.Fatal("complete fork gas evidence")
	}
	return out.Block
}

func TestReplayRecordedAmsterdamGas(t *testing.T) {
	raw := gasRecorded(t)
	before := replayTestClone(t, raw)
	b := replayBudget{limits: replayTestLimits()}
	d, err := replayDecodeBlock(context.Background(), raw, &b)
	if err != nil || d.header.GasUsed != 142149240 || d.receipts[119].CumulativeGasUsed != 206878059 {
		t.Fatal("strict gas boundary", err)
	}
	var rows []struct{ Result map[string]string }
	if json.Unmarshal(raw.Gas.Trace, &rows) != nil {
		t.Fatal("trace")
	}
	var exec, state uint64
	for _, row := range rows {
		e, _ := rpcQuantity(row.Result["executionGasUsed"])
		s, _ := rpcQuantity(row.Result["stateGasUsed"])
		exec += e
		state += s
	}
	if exec != 66635112 || state != 142149240 {
		t.Fatal("independent expected dimensions")
	}
	if !bytes.Equal(replayTestJSON(t, raw), replayTestJSON(t, before)) {
		t.Fatal("mutated input")
	}
	raw.Gas = nil
	b = replayBudget{limits: replayTestLimits()}
	if _, err := replayDecodeBlock(context.Background(), raw, &b); err != ErrReplay {
		t.Fatal("missing witness accepted", err)
	}
}

// Finite in-memory refusals never send malformed data to the network.
func TestReplayGasWitnessRefusals(t *testing.T) {
	base := gasRecorded(t)
	for name, mutate := range map[string]func(*ReplayBlock){
		"unknown-profile": func(b *ReplayBlock) { b.Gas.Profile = "unknown" },
		"wrong-block":     func(b *ReplayBlock) { b.Gas.BlockHash = testBlock },
		"missing-trace":   func(b *ReplayBlock) { b.Gas.Trace = nil },
		"unicode":         func(b *ReplayBlock) { b.Gas.Trace = []byte{255} },
		"surrogate":       func(b *ReplayBlock) { b.Gas.Trace = []byte(`[{"\ud800":0}]`) },
		"duplicate":       func(b *ReplayBlock) { b.Gas.Trace = []byte(`[{"result":{},"result":{}}]`) },
		"trailing":        func(b *ReplayBlock) { b.Gas.Trace = append(b.Gas.Trace, []byte(" true")...) },
		"deep":            func(b *ReplayBlock) { b.Gas.Trace = []byte(`[[[[[0]]]]]`) },
		"syntax":          func(b *ReplayBlock) { b.Gas.Trace = []byte(`[`) },
		"null":            func(b *ReplayBlock) { b.Gas.Trace = []byte(`null`) },
		"object":          func(b *ReplayBlock) { b.Gas.Trace = []byte(`{}`) },
		"missing":         func(b *ReplayBlock) { b.Gas.Trace = []byte(`[]`) },
		"header-over-limit": func(b *ReplayBlock) {
			var h types.Header
			_ = rlp.DecodeBytes(b.HeaderRLP, &h)
			h.GasLimit = h.GasUsed - 1
			b.HeaderRLP, _ = rlp.EncodeToBytes(&h)
		},
		"header-total": func(b *ReplayBlock) {
			var h types.Header
			_ = rlp.DecodeBytes(b.HeaderRLP, &h)
			h.GasUsed--
			b.HeaderRLP, _ = rlp.EncodeToBytes(&h)
			b.Gas.BlockHash = h.Hash().Hex()
		},
		"legacy-extra": func(b *ReplayBlock) {
			var h types.Header
			_ = rlp.DecodeBytes(b.HeaderRLP, &h)
			h.Time = sepoliaAmsterdamTime - 1
			b.HeaderRLP, _ = rlp.EncodeToBytes(&h)
		},
	} {
		t.Run(name, func(t *testing.T) {
			b := replayTestClone(t, base)
			mutate(&b)
			budget := replayBudget{limits: replayTestLimits()}
			if _, err := replayDecodeBlock(context.Background(), b, &budget); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	for _, mode := range []string{"extra-entry-field", "entry-error", "hash", "hash-type", "duplicate-tx", "reorder", "missing-row", "extra-row", "missing-field", "extra-field", "null-quantity", "leading-zero", "overflow", "paid", "execution", "state", "refund", "negative", "numeric"} {
		t.Run(mode, func(t *testing.T) {
			b := replayTestClone(t, base)
			var rows []map[string]json.RawMessage
			_ = json.Unmarshal(b.Gas.Trace, &rows)
			var v map[string]any
			_ = json.Unmarshal(rows[0]["result"], &v)
			switch mode {
			case "extra-entry-field":
				rows[0]["blockHash"] = json.RawMessage(`"extra"`)
			case "entry-error":
				delete(rows[0], "result")
				rows[0]["error"] = json.RawMessage(`"unavailable"`)
			case "hash":
				rows[0]["txHash"] = json.RawMessage(`"` + testBlock + `"`)
			case "hash-type":
				rows[0]["txHash"] = json.RawMessage(`null`)
			case "duplicate-tx":
				rows[1] = rows[0]
			case "reorder":
				rows[0], rows[1] = rows[1], rows[0]
			case "missing-row":
				rows = rows[:119]
			case "extra-row":
				rows = append(rows, rows[0])
			case "missing-field":
				delete(v, "stateGasUsed")
			case "extra-field":
				v["unknown"] = "0x0"
			case "null-quantity":
				v["gasUsed"] = nil
			case "leading-zero":
				v["gasUsed"] = "0x00"
			case "overflow":
				v["stateGasUsed"] = "0x10000000000000000"
			case "paid":
				v["gasUsed"] = "0x1"
			case "execution":
				v["executionGasUsed"] = "0x1000001"
			case "state":
				v["stateGasUsed"] = "0xffffffffffffffff"
			case "refund":
				v["gasRefund"] = "0xffffffffffffffff"
			case "negative":
				v["gasUsed"] = "-1"
			case "numeric":
				v["gasUsed"] = float64(53871)
			}
			if mode != "entry-error" && mode != "reorder" && mode != "duplicate-tx" {
				rows[0]["result"], _ = json.Marshal(v)
			}
			b.Gas.Trace, _ = json.Marshal(rows)
			budget := replayBudget{limits: replayTestLimits()}
			if _, err := replayDecodeBlock(context.Background(), b, &budget); err == nil {
				t.Fatal("accepted")
			}
		})
	}
	var decoded replayDecodedBlock
	budget := replayBudget{limits: replayTestLimits()}
	decoded, err := replayDecodeContent(context.Background(), base, &budget)
	if err != nil {
		t.Fatal(err)
	}
	budget.bytes = budget.limits.MaxEvidenceBytes
	if err := replayCheckGas(context.Background(), base, decoded, &budget); err != ErrReplayUnavailable {
		t.Fatal("witness byte budget", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	budget = replayBudget{limits: replayTestLimits()}
	if err := replayCheckGas(ctx, base, decoded, &budget); err != context.Canceled {
		t.Fatal("cancel", err)
	}
	budget = replayBudget{limits: replayTestLimits()}
	budget.limits.MaxTransactions = 119
	if err := replayCheckGas(context.Background(), base, decoded, &budget); err == nil {
		t.Fatal("array cap")
	}
}

func TestReplayGasSettlementFloorsAndRefunds(t *testing.T) {
	for _, v := range []struct {
		p, e, s, r, f, l uint64
		ok               bool
	}{
		{90000, 100000, 0, 10000, 15000, 120000, true},
		{88769372, 603968, 88201440, 0, 603968, 110000000, true},
		{100, 100, 0, 0, 100, 150, true}, {100, 110, 0, 0, 100, 150, true},
		{100, 125, 0, 0, 100, 150, false}, {100, 126, 0, 0, 100, 150, false},
		{100, 110, 0, 20, 100, 150, false}, {100, 120, 0, 20, 100, 150, true},
		{100, 100, 1, 0, 100, 150, false}, {99, 100, 0, 0, 100, 150, false},
		{120, 99, 0, 0, 100, 150, false}, {100, 100, 0, 200, 100, 150, false},
		{120, 120, 0, 40, 100, 150, false}, {120, 120, 121, 0, 100, 150, false},
		{120, 150, 0, 30, 100, 200, true}, {120, 151, 0, 31, 100, 200, false},
	} {
		if got := replayGasSettlement(v.p, v.e, v.s, v.r, v.f, v.l); got != v.ok {
			t.Fatalf("%+v got %v", v, got)
		}
	}
}

func TestReplayGasFloorSignedTransactionInputs(t *testing.T) {
	key, err := crypto.HexToECDSA(strings.Repeat("0", 63) + "1")
	if err != nil {
		t.Fatal(err)
	}
	self := crypto.PubkeyToAddress(key.PublicKey)
	other := common.HexToAddress(replayTestWallet)
	for _, v := range []struct {
		name   string
		to     *common.Address
		value  int64
		data   []byte
		access types.AccessList
		want   uint64
	}{
		{"create", nil, 0, nil, nil, 24000}, {"create-value", nil, 1, nil, nil, 24000},
		{"self", &self, 0, nil, nil, 12000}, {"self-value", &self, 1, nil, nil, 12000},
		{"call", &other, 0, nil, nil, 15000}, {"value", &other, 1, nil, nil, 21000},
		{"data-access", &other, 0, []byte{0, 1}, types.AccessList{{Address: other, StorageKeys: []common.Hash{{1}, {2}}}}, 15000 + 64*(2+20+64)},
	} {
		t.Run(v.name, func(t *testing.T) {
			tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(11155111), To: v.to, Value: big.NewInt(v.value), Data: v.data, AccessList: v.access})
			tx, e := types.SignTx(tx, types.LatestSignerForChainID(big.NewInt(11155111)), key)
			if e != nil {
				t.Fatal(e)
			}
			got, e := replayGasFloor(tx)
			if e != nil || got != v.want {
				t.Fatal(got, e, v.want)
			}
		})
	}
	for _, chain := range []int64{1, 11155111} {
		tx := types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(chain), To: &other, Value: new(big.Int)})
		if _, err := replayGasFloor(tx); err == nil {
			t.Fatal("unsigned or foreign transaction")
		}
	}
	tx := types.NewTx(&types.LegacyTx{To: &other, Value: new(big.Int)})
	tx, err = types.SignTx(tx, types.HomesteadSigner{}, key)
	if err != nil {
		t.Fatal(err)
	}
	if f, e := replayGasFloor(tx); e != nil || f != 15000 {
		t.Fatal("unprotected legacy", f, e)
	}
}

func TestNodeReplayBlockNoPartialOutput(t *testing.T) {
	var absent *PinnedAgentReads
	if out, e := absent.ReadReplayBlock(context.Background(), sepoliaDeploymentBlock, nodeTestLimits()); out != nil || e != ErrConfiguration {
		t.Fatal("configuration")
	}
	for _, mode := range []string{"network", "body", "trace-error", "gas", "recheck", "request-budget", "wire-budget"} {
		t.Run(mode, func(t *testing.T) {
			headers := 0
			r := gasFixture(t, func(method string, body []byte) []byte {
				if method == "eth_getBlockByNumber" {
					headers++
				}
				if mode == "network" && method == "eth_chainId" || mode == "trace-error" && method == "debug_traceBlockByHash" || mode == "recheck" && headers > 2 {
					return []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"unavailable"}}`)
				}
				if mode == "body" && method == "debug_getRawBlock" {
					return []byte(`{"jsonrpc":"2.0","id":1,"result":"0x00"}`)
				}
				if mode == "gas" && method == "debug_traceBlockByHash" {
					return []byte(`{"jsonrpc":"2.0","id":1,"result":[]}`)
				}
				return body
			})
			limits := nodeTestLimits()
			if mode == "request-budget" {
				limits.MaxRequests = 7
			}
			if mode == "wire-budget" {
				limits.MaxWireBytes = 32
			}
			if out, e := r.ReadReplayBlock(context.Background(), sepoliaDeploymentBlock, limits); out != nil || e == nil {
				t.Fatal("partial", fmt.Sprint(out), e)
			}
		})
	}
}

func TestReplayCommandsAmsterdamAdmissionAndMissingWitness(t *testing.T) {
	// Retain the original handwritten snapshots. Only admission timestamps and
	// canonical enclosing commitments move to the new, synthetic fork interval.
	in := replayTestInput(t)
	_, outcomes, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	changed := outcomes.Events["RecordChanged"]
	var parent common.Hash
	for i := range in.Blocks {
		b := &in.Blocks[i]
		var h types.Header
		if rlp.DecodeBytes(b.HeaderRLP, &h) != nil {
			t.Fatal("header")
		}
		h.Time = sepoliaAmsterdamTime + uint64(i)
		h.ParentHash = parent
		receipt := new(types.Receipt)
		if receipt.UnmarshalBinary(b.Receipts[0]) != nil {
			t.Fatal("receipt")
		}
		if i > 0 {
			entry := contentJSON(in.History[i-1].Entry).(map[string]any)
			entry["admitted"] = h.Time
			in.History[i-1].Entry = replayTestJSON(t, entry)
			values, e := changed.Inputs.NonIndexed().Unpack(receipt.Logs[0].Data)
			if e != nil {
				t.Fatal(e)
			}
			values[7] = h.Time
			receipt.Logs[0].Data, e = changed.Inputs.NonIndexed().Pack(values...)
			if e != nil {
				t.Fatal(e)
			}
			b.Executions[0].Root.Calls[0].Logs[0].Data = bytes.Clone(receipt.Logs[0].Data)
		}
		h.ReceiptHash = types.DeriveSha(types.Receipts{receipt}, trie.NewStackTrie(nil))
		b.Receipts[0], _ = receipt.MarshalBinary()
		b.HeaderRLP, _ = rlp.EncodeToBytes(&h)
		parent = h.Hash()
		b.Executions[0].BlockHash = h.Hash().Hex()
		tx := new(types.Transaction)
		if tx.UnmarshalBinary(b.Transactions[0]) != nil {
			t.Fatal("tx")
		}
		trace, _ := json.Marshal([]any{map[string]any{"txHash": tx.Hash().Hex(), "result": map[string]string{"gasUsed": "0x5208", "executionGasUsed": "0x5208", "stateGasUsed": "0x0", "gasRefund": "0x0"}}})
		b.Gas = &ReplayGasEvidence{SepoliaAmsterdamGasProfile, h.Hash().Hex(), trace}
		in.Anchor = replayHeader(&h)
	}
	steps, err := ReplayCommands(context.Background(), in, replayTestLimits())
	if err != nil || len(steps) != len(in.History) {
		t.Fatal("all operation replay with strict gas", err)
	}
	in.Blocks[0].Gas = nil
	if steps, err := ReplayCommands(context.Background(), in, replayTestLimits()); steps != nil || err == nil {
		t.Fatal("equal totals cannot replace new-fork witness")
	}
}

func TestReplayGasActivationEmptyBlock(t *testing.T) {
	for _, stamp := range []uint64{sepoliaAmsterdamTime - 1, sepoliaAmsterdamTime, sepoliaAmsterdamTime + 1} {
		h := &types.Header{Number: new(big.Int), Difficulty: new(big.Int), Time: stamp, GasLimit: 1000000, TxHash: types.EmptyTxsHash, ReceiptHash: types.EmptyReceiptsHash, UncleHash: types.EmptyUncleHash}
		raw, _ := rlp.EncodeToBytes(h)
		b := ReplayBlock{HeaderRLP: raw}
		if stamp >= sepoliaAmsterdamTime {
			b.Gas = &ReplayGasEvidence{SepoliaAmsterdamGasProfile, h.Hash().Hex(), json.RawMessage(`[]`)}
		}
		budget := replayBudget{limits: replayTestLimits()}
		if _, err := replayDecodeBlock(context.Background(), b, &budget); err != nil {
			t.Fatal("activation", stamp, err)
		}
	}
}
