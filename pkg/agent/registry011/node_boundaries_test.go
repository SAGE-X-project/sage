package registry011

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestNodeNetworkBindingRefusals(t *testing.T) {
	for _, mode := range []string{"chain-error", "chain", "genesis-error", "genesis", "registry-error", "registry", "client-error", "client"} {
		t.Run(mode, func(t *testing.T) {
			active := false
			r, _ := nodeFixture(t, func(method string, params []json.RawMessage, body []byte) []byte {
				if !active {
					return body
				}
				target := map[string]string{"chain-error": "eth_chainId", "chain": "eth_chainId", "genesis-error": "eth_getBlockByNumber", "genesis": "eth_getBlockByNumber", "registry-error": "eth_call", "registry": "eth_call", "client-error": "web3_clientVersion", "client": "web3_clientVersion"}[mode]
				if method != target {
					return body
				}
				if len(params) > 0 && method == "eth_getBlockByNumber" && string(params[0]) != `"0x0"` {
					return body
				}
				if mode == "chain-error" || mode == "genesis-error" || mode == "registry-error" || mode == "client-error" {
					return []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"unavailable"}}`)
				}
				// A nonempty string is a valid version claim, not a credential.
				// Use an actual schema violation for this refusal fixture.
				if mode == "client" {
					return nodeMutateResult(body, func(v *json.RawMessage) { *v = json.RawMessage(`null`) })
				}
				return nodeMutateResult(body, func(v *json.RawMessage) { *v = json.RawMessage(`"0x0"`) })
			})
			active = true
			if got, e := r.ReadNodeBlock(context.Background(), sepoliaDeploymentBlock, nodeTestLimits()); got != nil || e == nil {
				t.Fatal(mode)
			}
		})
	}
}

func TestNodeOwnedBodyCommitmentRefusals(t *testing.T) {
	in := replayTestInput(t)
	raw := in.Blocks[0]
	for _, mode := range []string{"uncle", "withdrawals", "receipt", "receipt-hex", "raw-extra", "header-number", "post-error"} {
		t.Run(mode, func(t *testing.T) {
			var h types.Header
			_ = rlp.DecodeBytes(raw.HeaderRLP, &h)
			h.Number.SetUint64(sepoliaDeploymentBlock)
			h.Time = 0
			if mode == "uncle" {
				h.UncleHash = common.Hash{1}
			}
			if mode == "withdrawals" {
				v := common.Hash{2}
				h.WithdrawalsHash = &v
			}
			tx := new(types.Transaction)
			_ = tx.UnmarshalBinary(raw.Transactions[0])
			block := types.NewBlockWithHeader(&h).WithBody(types.Body{Transactions: []*types.Transaction{tx}})
			encoded, e := rlp.EncodeToBytes(block)
			if e != nil {
				t.Fatal(e)
			}
			if mode == "raw-extra" {
				encoded = append(encoded, 0)
			}
			active := false
			count := 0
			r, _ := nodeFixture(t, func(method string, params []json.RawMessage, body []byte) []byte {
				if !active {
					return body
				}
				if method == "eth_getBlockByNumber" && string(params[0]) == `"0x0"` {
					return body
				}
				if method == "eth_getBlockByNumber" {
					count++
					if mode == "post-error" && count > 1 {
						return []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"unavailable"}}`)
					}
				}
				return nodeMutateResult(body, func(v *json.RawMessage) {
					switch method {
					case "eth_getBlockByNumber":
						number := "0xb6b80e"
						if mode == "header-number" {
							number = "0xb6b80f"
						}
						*v, _ = json.Marshal(map[string]any{"hash": h.Hash().Hex(), "parentHash": h.ParentHash.Hex(), "number": number, "timestamp": "0x0"})
					case "debug_getRawBlock":
						*v, _ = json.Marshal(nodeHex(encoded))
					case "debug_getRawReceipts":
						receipt := nodeHex(raw.Receipts[0])
						if mode == "receipt" {
							receipt = "0x00"
						}
						if mode == "receipt-hex" {
							receipt = "nothex"
						}
						*v, _ = json.Marshal([]string{receipt})
					case "debug_traceTransaction":
						*v, _ = json.Marshal(nodeTestFrame(raw.Executions[0].Root))
					}
				})
			})
			active = true
			r.header = replayHeader(&h)
			if out, e := r.ReadNodeBlock(context.Background(), sepoliaDeploymentBlock, nodeTestLimits()); out != nil || e == nil {
				t.Fatal("accepted", mode)
			}
		})
	}
}

func TestNodeABIWireCompatibility(t *testing.T) {
	_, deployed, e := nodeIdentityManifest(nodeIdentityJSON, nodeABIJSON)
	if e != nil {
		t.Fatal(e)
	}
	replay, outcomes, e := replayABIs()
	if e != nil {
		t.Fatal(e)
	}
	args := func(a abi.Arguments) string {
		s := ""
		for _, v := range a {
			s += v.Type.String() + ","
		}
		return s
	}
	if len(deployed.Methods) != 13 || len(deployed.Events) != 2 {
		t.Fatal("ABI scope")
	}
	for name, method := range replay.Methods {
		other, ok := deployed.Methods[name]
		if !ok || method.Sig != other.Sig || args(method.Outputs) != args(other.Outputs) || method.StateMutability != other.StateMutability {
			t.Fatal("method", name)
		}
	}
	for name, event := range outcomes.Events {
		other, ok := deployed.Events[name]
		if !ok || event.ID != other.ID || args(event.Inputs) != args(other.Inputs) {
			t.Fatal("event", name)
		}
	}
	var absent *ReadRPC
	if _, _, e := absent.callEnvelope(context.Background(), "eth_chainId", []any{}, 1); e != ErrConfiguration {
		t.Fatal(e)
	}
	c := testReadRPC(t, func(*http.Request) (*http.Response, error) { t.Fatal("zero budget reached transport"); return nil, nil })
	if _, _, e := c.callEnvelope(context.Background(), "eth_chainId", []any{}, 0); e != ErrConfiguration {
		t.Fatal(e)
	}
}

// An empty body is a supported content observation, not an absence certificate.
func TestNodeEmptyContentGasBoundary(t *testing.T) {
	h := &types.Header{Number: common.Big0, Difficulty: common.Big0, UncleHash: types.EmptyUncleHash, TxHash: types.EmptyTxsHash, ReceiptHash: types.EmptyReceiptsHash}
	// Avoid aliasing common.Big0 while assigning the fixed deployment height.
	h.Number = new(big.Int).SetUint64(sepoliaDeploymentBlock)
	block := types.NewBlockWithHeader(h)
	encoded, err := rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal(err)
	}
	active := false
	r, _ := nodeFixture(t, func(method string, params []json.RawMessage, body []byte) []byte {
		if !active || (method == "eth_getBlockByNumber" && string(params[0]) == `"0x0"`) {
			return body
		}
		return nodeMutateResult(body, func(v *json.RawMessage) {
			switch method {
			case "eth_getBlockByNumber":
				*v, _ = json.Marshal(map[string]any{"hash": h.Hash().Hex(), "parentHash": h.ParentHash.Hex(), "number": "0xb6b80e", "timestamp": "0x0"})
			case "debug_getRawBlock":
				*v, _ = json.Marshal(nodeHex(encoded))
			case "debug_getRawReceipts":
				*v = json.RawMessage(`[]`)
			}
		})
	})
	active = true
	r.header = replayHeader(h)
	got, err := r.ReadNodeBlock(context.Background(), sepoliaDeploymentBlock, nodeTestLimits())
	if err != nil || len(got.Block.Transactions) != 0 || got.Gas != (NodeGasEvidence{LegacyGasCompatible: true}) {
		t.Fatal("empty content", err)
	}
	budget := replayBudget{limits: nodeTestLimits().Replay}
	if _, err := replayDecodeBlock(context.Background(), got.Block, &budget); err != nil {
		t.Fatal(err)
	}
	// Header-only mismatch remains observable, and strict replay still refuses.
	h.GasUsed = 1
	block = types.NewBlockWithHeader(h)
	encoded, err = rlp.EncodeToBytes(block)
	if err != nil {
		t.Fatal(err)
	}
	r.header = replayHeader(h)
	got, err = r.ReadNodeBlock(context.Background(), sepoliaDeploymentBlock, nodeTestLimits())
	if err != nil || got.Gas.LegacyGasCompatible || got.Gas.HeaderGasUsed != 1 || got.Gas.ReceiptCumulativeGasUsed != 0 {
		t.Fatal("unequal empty gas", err)
	}
	budget = replayBudget{limits: nodeTestLimits().Replay}
	if _, err := replayDecodeBlock(context.Background(), got.Block, &budget); err != ErrReplay {
		t.Fatal("strict unequal empty gas", err)
	}
}
