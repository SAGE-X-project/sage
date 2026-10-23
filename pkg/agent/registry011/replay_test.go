package registry011

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

const replayTestRegistry = "eip155:11155111:" + SepoliaRegistryAddress
const replayTestAgent = "replay.unit"
const replayTestWallet = "0x0000000000000000000000000000000000001000"
const replayTestOperator = "0x0000000000000000000000000000000000002000"

func replayTestLimits() ReplayLimits {
	return ReplayLimits{History: HistoryReadLimits{4096, 8 << 20}, MaxBlocks: 4096, MaxTransactions: 16384, MaxFrames: 65536, MaxLogs: 65536, MaxProofs: 4096, MaxEvidenceBytes: 64 << 20}
}
func replayTestJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, e := rfc8785.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func replayTestClone[T any](t *testing.T, v T) T {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	var out T
	if e = json.Unmarshal(b, &out); e != nil {
		t.Fatal(e)
	}
	return out
}

// RFC 8032 section 7.1 public sample seeds, never operator or transaction keys.
// Signatures are generated only inside owned, offline Go unit tests in CI.
func replayTestKey(t *testing.T, name string, second bool) replayKey {
	t.Helper()
	seed := "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"
	if second {
		seed = "4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb"
	}
	b, _ := hex.DecodeString(seed)
	key := ed25519.NewKeyFromSeed(b)
	k := replayKey{Name: name, Alg: "ed25519", Key: base64.RawURLEncoding.EncodeToString(key.Public().(ed25519.PublicKey)), Proof: replayProof{Signer: "did:sage:" + replayTestRegistry + ":" + replayTestAgent + "#" + name}, State: "accepted"}
	k.Proof.Value = base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, replayChallenge(replayTestRegistry, replayTestAgent, k)))
	return k
}
func replayTestKEM(t *testing.T, signer replayKey) replayKey {
	t.Helper()
	k := replayKey{Name: "kem", Alg: "x25519", Key: base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32)), Proof: replayProof{Signer: signer.Proof.Signer}, State: "accepted"}
	seed, _ := hex.DecodeString("9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60")
	k.Proof.Value = base64.RawURLEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed(seed), replayChallenge(replayTestRegistry, replayTestAgent, k)))
	return k
}
func replayTestCommand(t *testing.T, op, actor string, previous uint64, args map[string]any) replayCommand {
	t.Helper()
	expected := ""
	if previous > 0 {
		expected = strconv.FormatUint(previous, 10)
	}
	raw := replayTestJSON(t, map[string]any{"actor": actor, "arguments": args, "binding": "sage-registry-chain-0.11.0", "did": "did:sage:" + replayTestRegistry + ":" + replayTestAgent, "expectedVersion": expected, "operation": op, "registry": replayTestRegistry})
	a, _, e := replayABIs()
	if e != nil {
		t.Fatal(e)
	}
	c, ok := replayParseCommand(replayTestRegistry, replayTestAgent, raw, a)
	if !ok {
		t.Fatalf("invalid normal test command %s", op)
	}
	return c
}
func replayTestCreate(t *testing.T) replayCommand {
	k := replayTestKey(t, "a", false)
	return replayTestCommand(t, "create", replayTestWallet, 0, map[string]any{"controller": replayTestWallet, "keys": []replayKey{k, replayTestKEM(t, k)}, "services": []replayService{}, "salt": base64.RawURLEncoding.EncodeToString(make([]byte, 32))})
}

// Explicit expected snapshots: no use of apply, its cleanup or ACL helper.
// The state changes below are the handwritten expected effects of 11 normal
// commands, covering all eight operations and both principal account classes.
func replayTestSteps(t *testing.T) []HistoryContentStep {
	t.Helper()
	create := replayTestCreate(t)
	second := replayTestKey(t, "b", true)
	commands := []replayCommand{create,
		replayTestCommand(t, "authorize-operator", replayTestWallet, 1, map[string]any{"operator": replayTestOperator, "scope": "activate"}),
		replayTestCommand(t, "authorize-operator", replayTestWallet, 2, map[string]any{"operator": replayTestOperator, "scope": "deactivate"}),
		replayTestCommand(t, "activate", replayTestOperator, 3, map[string]any{}),
		replayTestCommand(t, "authorize-operator", replayTestWallet, 4, map[string]any{"operator": replayTestOperator, "scope": "add-key"}),
		replayTestCommand(t, "add-key", replayTestOperator, 5, map[string]any{"key": second}),
		replayTestCommand(t, "revoke-operator", replayTestWallet, 6, map[string]any{"operator": replayTestOperator, "scope": "add-key"}),
		replayTestCommand(t, "update-services", replayTestWallet, 7, map[string]any{"services": []replayService{{"api", "A2A", "https://example.invalid/api"}}}),
		replayTestCommand(t, "revoke-key", replayTestWallet, 8, map[string]any{"name": "a"}),
		replayTestCommand(t, "revoke-key", replayTestWallet, 9, map[string]any{"name": "kem"}),
		replayTestCommand(t, "deactivate", replayTestOperator, 10, map[string]any{}),
	}
	r := replayRecord{ID: "did:sage:" + replayTestRegistry + ":" + replayTestAgent, Controller: replayTestWallet, Keys: append([]replayKey{}, create.keys...), Services: []replayService{}, State: "created"}
	g := []replayGrant{}
	steps := make([]HistoryContentStep, len(commands))
	for i, c := range commands {
		removed := []replayGrant{}
		switch i {
		case 1:
			g = []replayGrant{{replayTestOperator, "activate"}}
		case 2:
			g = []replayGrant{{replayTestOperator, "activate"}, {replayTestOperator, "deactivate"}}
		case 3:
			r.State = "active"
			removed = []replayGrant{{replayTestOperator, "activate"}}
			g = []replayGrant{{replayTestOperator, "deactivate"}}
		case 4:
			g = []replayGrant{{replayTestOperator, "add-key"}, {replayTestOperator, "deactivate"}}
		case 5:
			r.Keys = []replayKey{create.keys[0], second, create.keys[1]}
		case 6:
			g = []replayGrant{{replayTestOperator, "deactivate"}}
		case 7:
			r.Services = []replayService{{"api", "A2A", "https://example.invalid/api"}}
		case 8:
			r.Keys[0].State = "revoked"
		case 9:
			r.Keys[2].State = "revoked"
		case 10:
			r.State = "deactivated"
			removed = []replayGrant{{replayTestOperator, "deactivate"}}
			g = []replayGrant{}
		}
		r.Version = strconv.Itoa(i + 1)
		rb, gb := replayTestJSON(t, r), replayTestJSON(t, g)
		previous := ""
		if i > 0 {
			previous = strconv.Itoa(i)
		}
		var target any
		if c.op == "authorize-operator" || c.op == "revoke-operator" {
			target = c.args
		}
		h := replayTestJSON(t, map[string]any{"actor": c.actor, "admitted": i + 1, "automaticRemovals": removed, "commandDigest": contentHash(c.raw), "grantsDigest": contentHash(gb), "operation": c.op, "previousVersion": previous, "recordDigest": contentHash(rb), "target": target, "version": r.Version})
		steps[i] = HistoryContentStep{History: HistoryBytes{uint64(i + 1), h, c.raw}, Record: rb, Grants: gb}
	}
	return steps
}

// Synthetic canonical transactions/headers/receipt tries and normalized traces,
// exclusively in memory. No RPC, node, EVM, file key, broadcast or live claim.
// Transaction test key 1 is public disposable fixture material.
func replayTestInput(t *testing.T) ReplayInput {
	t.Helper()
	steps := replayTestSteps(t)
	a, out, e := replayABIs()
	if e != nil {
		t.Fatal(e)
	}
	in := ReplayInput{Registry: replayTestRegistry, Agent: replayTestAgent, DeploymentBlock: 0, Record: bytes.Clone(steps[len(steps)-1].Record), Grants: bytes.Clone(steps[len(steps)-1].Grants)}
	create, ok := replayParseCommand(in.Registry, in.Agent, steps[0].History.Command, a)
	if !ok {
		t.Fatal("create")
	}
	claim := replayClaim(in.Registry, create)
	committed := out.Events["ClaimCommitted"]
	changed := out.Events["RecordChanged"]
	commitData, _ := a.Pack("commitClaim", claim)
	commitEvent, _ := committed.Inputs.NonIndexed().Pack(big.NewInt(0))
	commitLog := &types.Log{Address: common.HexToAddress(SepoliaRegistryAddress), Topics: []common.Hash{committed.ID, common.BytesToHash(common.HexToAddress(replayTestWallet).Bytes()), common.Hash(claim)}, Data: commitEvent}
	replayTestAppendBlock(t, &in, commitData, commitLog, replayTestWallet, 0)
	for i, s := range steps {
		c, ok := replayParseCommand(in.Registry, in.Agent, s.History.Command, a)
		if !ok {
			t.Fatal("command")
		}
		h := contentJSON(s.History.Entry).(map[string]any)
		data, _ := changed.Inputs.NonIndexed().Pack("did:sage:"+in.Registry+":"+in.Agent, c.expected, uint64(i+1), c.op, sha256.Sum256(c.raw), sha256.Sum256(s.Record), sha256.Sum256(s.Grants), uint64(i+1))
		didHash := sha256.Sum256([]byte("did:sage:" + in.Registry + ":" + in.Agent))
		l := &types.Log{Address: common.HexToAddress(SepoliaRegistryAddress), Topics: []common.Hash{changed.ID, common.Hash(didHash), common.BytesToHash(common.HexToAddress(contentString(h["actor"])).Bytes())}, Data: data}
		replayTestAppendBlock(t, &in, c.calldata, l, c.actor, uint64(i+1))
		in.History = append(in.History, s.History)
	}
	return in
}
func replayTestAppendBlock(t *testing.T, in *ReplayInput, data []byte, log *types.Log, caller string, stamp uint64) {
	t.Helper()
	key, e := crypto.HexToECDSA(strings.Repeat("0", 63) + "1")
	if e != nil {
		t.Fatal(e)
	}
	sender := crypto.PubkeyToAddress(key.PublicKey)
	wallet := common.HexToAddress(caller)
	tx, e := types.SignTx(types.NewTx(&types.DynamicFeeTx{ChainID: big.NewInt(11155111), Nonce: uint64(len(in.Blocks)), GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2), Gas: 1000000, To: &wallet, Value: new(big.Int), Data: []byte{1}}), types.LatestSignerForChainID(big.NewInt(11155111)), key)
	if e != nil {
		t.Fatal(e)
	}
	receipt := &types.Receipt{Type: tx.Type(), Status: 1, CumulativeGasUsed: 21000, Logs: []*types.Log{log}}
	txs, receipts := types.Transactions{tx}, types.Receipts{receipt}
	header := &types.Header{Number: new(big.Int).SetUint64(uint64(len(in.Blocks))), Difficulty: new(big.Int), Time: stamp, GasLimit: 1000000, GasUsed: 21000, TxHash: types.DeriveSha(txs, trie.NewStackTrie(nil)), ReceiptHash: types.DeriveSha(receipts, trie.NewStackTrie(nil)), UncleHash: types.EmptyUncleHash}
	if len(in.Blocks) > 0 {
		header.ParentHash = common.HexToHash(in.Anchor.Hash)
	}
	headerRLP, e := rlp.EncodeToBytes(header)
	if e != nil {
		t.Fatal(e)
	}
	txraw, _ := tx.MarshalBinary()
	rr, _ := receipt.MarshalBinary()
	child := ReplayFrame{Type: "CALL", From: caller, To: SepoliaRegistryAddress, Value: "0x0", Input: bytes.Clone(data), Logs: []ReplayFrameLog{{Address: SepoliaRegistryAddress, Topics: append([]common.Hash{}, log.Topics...), Data: bytes.Clone(log.Data)}}}
	root := ReplayFrame{Type: "CALL", From: replayAddress(sender), To: caller, Value: "0x0", Input: []byte{1}, Calls: []ReplayFrame{child}}
	trace := ReplayExecution{BlockHash: strings.ToLower(header.Hash().Hex()), TransactionHash: strings.ToLower(tx.Hash().Hex()), Root: root}
	in.Blocks = append(in.Blocks, ReplayBlock{HeaderRLP: headerRLP, Transactions: [][]byte{txraw}, Receipts: [][]byte{rr}, Executions: []ReplayExecution{trace}})
	in.Anchor = replayHeader(header)
}

func TestReplayAllOperationsWalletCallerAndSnapshots(t *testing.T) {
	in := replayTestInput(t)
	before := replayTestClone(t, in)
	got, e := ReplayCommands(context.Background(), in, replayTestLimits())
	if e != nil {
		t.Fatal(e)
	}
	want := replayTestSteps(t)
	if len(got) != len(want) {
		t.Fatal(len(got))
	}
	for i, s := range got {
		if !bytes.Equal(s.Record, want[i].Record) || !bytes.Equal(s.Grants, want[i].Grants) || !bytes.Equal(s.History.Entry, want[i].History.Entry) || !bytes.Equal(s.History.Command, want[i].History.Command) {
			t.Fatalf("snapshot %d", i)
		}
	}
	if !bytes.Equal(replayTestJSON(t, in), replayTestJSON(t, before)) {
		t.Fatal("input mutated")
	}
	got[0].Record[0] = 'x'
	if !bytes.Equal(in.Record, before.Record) {
		t.Fatal("retained final bytes")
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := ReplayCommands(context.Background(), in, replayTestLimits()); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
}

func TestReplayRefusesFiniteEvidenceMutations(t *testing.T) {
	// All cases mutate in-memory finite fixtures. No attack traffic or producer.
	cases := map[string]func(*ReplayInput){
		"no-blocks": func(x *ReplayInput) { x.Blocks = nil }, "missing-block": func(x *ReplayInput) { x.Blocks = x.Blocks[1:] },
		"wrong-start": func(x *ReplayInput) { x.DeploymentBlock = 1 }, "anchor-hash": func(x *ReplayInput) { x.Anchor.Hash = "0x" + strings.Repeat("a", 64) },
		"missing-trace": func(x *ReplayInput) { x.Blocks[0].Executions = nil }, "extra-trace": func(x *ReplayInput) {
			x.Blocks[0].Executions = append(x.Blocks[0].Executions, x.Blocks[0].Executions[0])
		},
		"trace-index": func(x *ReplayInput) { x.Blocks[0].Executions[0].TransactionIndex = 1 }, "trace-block": func(x *ReplayInput) { x.Blocks[0].Executions[0].BlockHash = x.Anchor.Hash },
		"trace-tx": func(x *ReplayInput) { x.Blocks[0].Executions[0].TransactionHash = x.Anchor.Hash }, "root-from": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.From = replayTestWallet },
		"root-to": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.To = replayTestOperator }, "root-kind": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Type = "CREATE" },
		"root-value": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Value = "0x1" }, "root-input": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Input = []byte{2} },
		"root-failed": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Error = "reverted" }, "child-caller": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Calls[0].From = replayTestOperator },
		"delegated-registry": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Calls[0].Type = "DELEGATECALL" },
		"static-registry":    func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Calls[0].Type = "STATICCALL" }, "failed-child": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Calls[0].Error = "reverted" },
		"nonzero-registry-value": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Calls[0].Value = "0x1" }, "unsupported-frame": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Calls[0].Type = "UNKNOWN" },
		"log-position": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Calls[0].Logs[0].AfterChildren = 1 }, "log-context": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Calls[0].Logs[0].Address = replayTestWallet },
		"log-data": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Calls[0].Logs[0].Data[0] ^= 1 }, "log-topic": func(x *ReplayInput) { x.Blocks[0].Executions[0].Root.Calls[0].Logs[0].Topics[1][0] ^= 1 },
		"short-calldata": func(x *ReplayInput) { x.Blocks[1].Executions[0].Root.Calls[0].Input = []byte{0} }, "trailing-calldata": func(x *ReplayInput) {
			x.Blocks[1].Executions[0].Root.Calls[0].Input = append(x.Blocks[1].Executions[0].Root.Calls[0].Input, 0)
		},
		"missing-history": func(x *ReplayInput) { x.History = x.History[:len(x.History)-1] }, "history-version": func(x *ReplayInput) { x.History[0].Version = 2 },
		"invented-command": func(x *ReplayInput) { x.History[0].Command = append(x.History[0].Command, ' ') }, "invented-entry": func(x *ReplayInput) { x.History[0].Entry = append(x.History[0].Entry, ' ') },
		"invented-final": func(x *ReplayInput) { x.Record = append(x.Record, ' ') }, "receipt-count": func(x *ReplayInput) { x.Blocks[0].Receipts = nil },
		"invalid-header": func(x *ReplayInput) { x.Blocks[0].HeaderRLP = []byte{0} }, "invalid-tx": func(x *ReplayInput) { x.Blocks[0].Transactions[0] = []byte{0} },
		"invalid-receipt": func(x *ReplayInput) { x.Blocks[0].Receipts[0] = []byte{0} },
	}
	base := replayTestInput(t)
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			in := replayTestClone(t, base)
			mutate(&in)
			got, e := ReplayCommands(context.Background(), in, replayTestLimits())
			if e == nil || got != nil {
				t.Fatalf("accepted mutation %v", e)
			}
		})
	}
}

func TestReplayExplicitLimitsAndCancellation(t *testing.T) {
	in := replayTestInput(t)
	for _, f := range []func(*ReplayLimits){func(x *ReplayLimits) { x.MaxBlocks = 0 }, func(x *ReplayLimits) { x.MaxTransactions = 0 }, func(x *ReplayLimits) { x.MaxFrames = 0 }, func(x *ReplayLimits) { x.MaxLogs = 0 }, func(x *ReplayLimits) { x.MaxProofs = 0 }, func(x *ReplayLimits) { x.MaxEvidenceBytes = 0 }, func(x *ReplayLimits) { x.History = HistoryReadLimits{} }, func(x *ReplayLimits) { x.MaxBlocks = 4097 }, func(x *ReplayLimits) { x.MaxTransactions = 16385 }, func(x *ReplayLimits) { x.MaxFrames = 65537 }, func(x *ReplayLimits) { x.MaxLogs = 65537 }, func(x *ReplayLimits) { x.MaxProofs = 4097 }, func(x *ReplayLimits) { x.MaxEvidenceBytes = 64<<20 + 1 }} {
		l := replayTestLimits()
		f(&l)
		if got, e := ReplayCommands(context.Background(), in, l); e != ErrConfiguration || got != nil {
			t.Fatal(e)
		}
	}
	for name, f := range map[string]func(*ReplayLimits){"blocks": func(x *ReplayLimits) { x.MaxBlocks = 1 }, "txs": func(x *ReplayLimits) { x.MaxTransactions = 1 }, "frames": func(x *ReplayLimits) { x.MaxFrames = 1 }, "logs": func(x *ReplayLimits) { x.MaxLogs = 1 }, "proofs": func(x *ReplayLimits) { x.MaxProofs = 1 }, "bytes": func(x *ReplayLimits) { x.MaxEvidenceBytes = 1 }, "history": func(x *ReplayLimits) { x.History.MaxBytes = 1 }} {
		t.Run(name, func(t *testing.T) {
			l := replayTestLimits()
			f(&l)
			if got, e := ReplayCommands(context.Background(), in, l); e == nil || got != nil {
				t.Fatal(e)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, e := ReplayCommands(ctx, in, replayTestLimits()); e != context.Canceled || got != nil {
		t.Fatal(e)
	}
	//lint:ignore SA1012 Exercise explicit nil-context rejection without starting an operation.
	if _, e := ReplayCommands(nil, in, replayTestLimits()); e != ErrConfiguration {
		t.Fatal(e)
	}
}
