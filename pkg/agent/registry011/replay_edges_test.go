package registry011

import (
	"bytes"
	"context"
	"encoding/binary"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

func TestReplayPreflightScalarBounds(t *testing.T) {
	word := func(n uint64) []byte { b := make([]byte, 32); binary.BigEndian.PutUint64(b[24:], n); return b }
	for _, raw := range [][]byte{nil, bytes.Repeat([]byte{255}, 32), word(148805)} {
		if _, ok := replayABIWord(raw); ok {
			t.Fatal("word bound")
		}
	}
	for _, kind := range []string{"address", "uint64", "bool", "bytes32", "uint256", "bytes4", "int64"} {
		typ, e := abi.NewType(kind, "", nil)
		if e != nil {
			t.Fatal(e)
		}
		fields := abi.Arguments{{Name: "x", Type: typ}}
		b := word(0)
		n, ok := replayABIFields(b, fields)
		supported := kind == "address" || kind == "uint64" || kind == "bool" || kind == "bytes32"
		if ok != supported || ok && n != 32 {
			t.Fatal(kind)
		}
		if supported && kind != "bytes32" {
			b[0] = 1
			if _, ok := replayABIFields(b, fields); ok {
				t.Fatal("high bits", kind)
			}
		}
		if kind == "bool" {
			if _, ok := replayABIFields(word(2), fields); ok {
				t.Fatal("bool domain")
			}
		}
	}
	str, _ := abi.NewType("string", "", nil)
	field := abi.Argument{Name: "name", Type: str}
	for _, data := range [][]byte{nil, word(33), append(word(1), bytes.Repeat([]byte{128}, 32)...), append(word(1), append([]byte{'a'}, bytes.Repeat([]byte{1}, 31)...)...), word(1)} {
		if _, ok := replayABIDynamic(data, field); ok {
			t.Fatal("invalid string range")
		}
	}
	field.Name = "unknown"
	if _, ok := replayABIDynamic(word(0), field); ok {
		t.Fatal("unknown field")
	}
	array, _ := abi.NewType("string[]", "", nil)
	if _, ok := replayABIDynamic(word(1), abi.Argument{Name: "keys", Type: array}); ok {
		t.Fatal("truncated array")
	}
	if _, ok := replayABIDynamic(word(129), abi.Argument{Name: "keys", Type: array}); ok {
		t.Fatal("array cap")
	}
	if _, ok := replayABIDynamic(word(0), abi.Argument{Name: "unknown", Type: array}); ok {
		t.Fatal("unknown array")
	}
	if _, ok := replayABIFields(nil, abi.Arguments{{Name: "name", Type: str}}); ok {
		t.Fatal("truncated head")
	}
	fixed, _ := abi.NewType("bytes32", "", nil)
	if _, ok := replayABIDynamic(word(0), abi.Argument{Type: fixed}); ok {
		t.Fatal("unsupported dynamic")
	}
}

func TestReplayStateLifetimeBoundsAndProofRefusals(t *testing.T) {
	base := replayTestSteps(t)
	a, _, _ := replayABIs()
	fresh := func(n int) replayState {
		var s replayState
		for i := 0; i < n; i++ {
			c, _ := replayParseCommand(replayTestRegistry, replayTestAgent, base[i].History.Command, a)
			if _, ok := s.apply(replayTestRegistry, c, uint64(i+1)); !ok {
				t.Fatal("normal")
			}
		}
		return s
	}
	for name, mutate := range map[string]func(*replayState, *replayCommand){
		"absent":             func(s *replayState, c *replayCommand) { *s = replayState{}; c.expected = 0 },
		"key-cap":            func(s *replayState, c *replayCommand) { s.record.Keys = make([]replayKey, 128) },
		"service-name-reuse": func(s *replayState, c *replayCommand) { s.record.Services = []replayService{{"b", "A2A", "https://x"}} },
		"raw-material-reuse": func(s *replayState, c *replayCommand) { s.record.Keys[0].Key = c.keys[0].Key },
		"bad-new-proof":      func(s *replayState, c *replayCommand) { c.keys[0].Proof.Value = strings.Repeat("A", 86) },
	} {
		t.Run(name, func(t *testing.T) {
			s := fresh(4)
			c := replayTestCommand(t, "add-key", replayTestWallet, 4, map[string]any{"key": replayTestKey(t, "b", true)})
			mutate(&s, &c)
			if _, ok := s.apply(replayTestRegistry, c, 5); ok {
				t.Fatal("transition accepted")
			}
		})
	}
	s := fresh(1)
	s.grants = make([]replayGrant, 128)
	c := replayTestCommand(t, "authorize-operator", replayTestWallet, 1, map[string]any{"operator": replayTestOperator, "scope": "deactivate"})
	if _, ok := s.apply(replayTestRegistry, c, 2); ok {
		t.Fatal("grant cap")
	}
	s = fresh(1)
	c = replayTestCreate(t)
	c.expected = 1
	if _, ok := s.apply(replayTestRegistry, c, 2); ok {
		t.Fatal("recreate")
	}
	s = replayState{}
	c = replayTestCreate(t)
	zero := uint64(0)
	c.keys[0].Expires = &zero
	if _, ok := s.apply(replayTestRegistry, c, 1); ok {
		t.Fatal("create without usable signer")
	}
	s = fresh(1)
	c = replayCommand{op: "unknown", actor: replayTestWallet, expected: 1}
	if _, ok := s.apply(replayTestRegistry, c, 2); ok {
		t.Fatal("unknown operation")
	}
	// Prospective serialization/headroom is checked even for recovery actions.
	for _, terminal := range []bool{false, true} {
		s = fresh(4)
		s.record.Services = make([]replayService, 16)
		for i := range s.record.Services {
			s.record.Services[i] = replayService{Name: "svc", Type: strings.Repeat("a", 64), URI: "https://x/" + strings.Repeat("a", 2000)}
		}
		for len(s.record.Keys) < 128 {
			k := s.record.Keys[0]
			k.Name = strings.Repeat("b", 32)
			s.record.Keys = append(s.record.Keys, k)
		}
		if terminal {
			c = replayTestCommand(t, "deactivate", replayTestWallet, 4, map[string]any{})
		} else {
			c = replayTestCommand(t, "authorize-operator", replayTestWallet, 4, map[string]any{"operator": replayTestOperator, "scope": "update-services"})
		}
		if _, ok := s.apply(replayTestRegistry, c, 5); ok {
			t.Fatal("oversized prospective record")
		}
	}
}

func TestReplayEventIdentityRefusals(t *testing.T) {
	in := replayTestInput(t)
	a, out, _ := replayABIs()
	newRun := func() *replayRun {
		b := replayBudget{limits: replayTestLimits()}
		return &replayRun{registry: in.Registry, agent: in.Agent, contract: a, outcomes: out, claims: map[replayClaimID]replayClaimState{}, created: map[string]bool{}, history: in.History, budget: &b}
	}
	frame := in.Blocks[1].Executions[0].Root.Calls[0]
	for name, mutate := range map[string]func(*ReplayFrame){
		"topics-count": func(f *ReplayFrame) { f.Logs[0].Topics = f.Logs[0].Topics[:2] }, "event-selector": func(f *ReplayFrame) { f.Logs[0].Topics[0][0] ^= 1 },
		"did-hash": func(f *ReplayFrame) { f.Logs[0].Topics[1][0] ^= 1 }, "actor-topic": func(f *ReplayFrame) { f.Logs[0].Topics[2][0] ^= 1 },
		"event-data-short": func(f *ReplayFrame) { f.Logs[0].Data = nil }, "event-data-offset": func(f *ReplayFrame) { f.Logs[0].Data[0] = 1 }, "event-trailing": func(f *ReplayFrame) { f.Logs[0].Data = append(f.Logs[0].Data, make([]byte, 32)...) },
	} {
		t.Run(name, func(t *testing.T) {
			f := replayTestClone(t, frame)
			mutate(&f)
			l := f.Logs[0]
			r := newRun()
			if e := r.event(context.Background(), replayLog{&types.Log{Address: common.HexToAddress(l.Address), Topics: l.Topics, Data: l.Data}, &f, true}, &types.Header{Number: big.NewInt(1), Time: 1}); e == nil {
				t.Fatal("event accepted")
			}
		})
	}
	commit := in.Blocks[0].Executions[0].Root.Calls[0]
	for name, mutate := range map[string]func(*ReplayFrame){"short-input": func(f *ReplayFrame) { f.Input = nil }, "selector": func(f *ReplayFrame) { f.Input[0] ^= 1 }, "caller": func(f *ReplayFrame) { f.Logs[0].Topics[1][0] ^= 1 }, "digest": func(f *ReplayFrame) { f.Logs[0].Topics[2][0] ^= 1 }, "block": func(f *ReplayFrame) { f.Logs[0].Data[31] = 1 }} {
		t.Run("commit-"+name, func(t *testing.T) {
			f := replayTestClone(t, commit)
			mutate(&f)
			l := f.Logs[0]
			if e := newRun().event(context.Background(), replayLog{&types.Log{Topics: l.Topics, Data: l.Data}, &f, true}, &types.Header{Number: big.NewInt(0)}); e == nil {
				t.Fatal("commit accepted")
			}
		})
	}
	// A create for another identity still consumes that caller-scoped claim.
	r := newRun()
	r.agent = "another"
	create, _ := replayParseCommand(in.Registry, in.Agent, in.History[0].Command, a)
	id := replayClaimID{create.actor, replayClaim(in.Registry, create)}
	r.claims[id] = replayClaimState{block: 0}
	l := frame.Logs[0]
	if e := r.event(context.Background(), replayLog{&types.Log{Topics: l.Topics, Data: l.Data}, &frame, true}, &types.Header{Number: big.NewInt(1), Time: 1}); e != nil || !r.claims[id].consumed || len(r.steps) != 0 {
		t.Fatal("other create", e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := r.event(ctx, replayLog{}, &types.Header{}); e != context.Canceled {
		t.Fatal(e)
	}
}

func TestReplayExecutionDirectCallerAndSelectedFormats(t *testing.T) {
	in := replayTestInput(t)
	b := replayBudget{limits: replayTestLimits()}
	block, e := replayDecodeBlock(context.Background(), in.Blocks[0], &b)
	if e != nil {
		t.Fatal(e)
	}
	key, _ := crypto.HexToECDSA(strings.Repeat("0", 63) + "1")
	sender := crypto.PubkeyToAddress(key.PublicKey)
	to := common.HexToAddress(SepoliaRegistryAddress)
	for _, kind := range []uint8{types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType} {
		var data types.TxData
		switch kind {
		case types.LegacyTxType:
			data = &types.LegacyTx{To: &to, Gas: 100000, GasPrice: big.NewInt(1), Value: new(big.Int), Data: []byte{1}}
		case types.AccessListTxType:
			data = &types.AccessListTx{ChainID: big.NewInt(11155111), To: &to, Gas: 100000, GasPrice: big.NewInt(1), Value: new(big.Int), Data: []byte{1}}
		default:
			data = &types.DynamicFeeTx{ChainID: big.NewInt(11155111), To: &to, Gas: 100000, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2), Value: new(big.Int), Data: []byte{1}}
		}
		tx, e := types.SignTx(types.NewTx(data), types.LatestSignerForChainID(big.NewInt(11155111)), key)
		if e != nil {
			t.Fatal(e)
		}
		block.txs = types.Transactions{tx}
		block.receipts[0].Type = kind
		root := ReplayFrame{Type: "CALL", From: replayAddress(sender), To: SepoliaRegistryAddress, Value: "0x0", Input: []byte{1}, Logs: in.Blocks[0].Executions[0].Root.Calls[0].Logs}
		trace := ReplayExecution{BlockHash: replayAddressHash(block.header.Hash()), TransactionHash: replayAddressHash(tx.Hash()), Root: root}
		budget := replayBudget{limits: replayTestLimits()}
		logs, e := replayExecution(context.Background(), &trace, block, 0, &budget)
		if e != nil || len(logs) != 1 || !logs[0].eligible {
			t.Fatal("direct supported", kind, e)
		}
	}
	// An unprotected legacy transaction has no accepted chain binding.
	tx, e := types.SignTx(types.NewTx(&types.LegacyTx{To: &to, Gas: 100000, GasPrice: big.NewInt(1), Value: new(big.Int)}), types.HomesteadSigner{}, key)
	if e != nil {
		t.Fatal(e)
	}
	block.txs = types.Transactions{tx}
	trace := ReplayExecution{BlockHash: replayAddressHash(block.header.Hash()), TransactionHash: replayAddressHash(tx.Hash()), Root: ReplayFrame{Type: "CALL", From: replayAddress(sender), To: SepoliaRegistryAddress, Value: "0x0"}}
	b = replayBudget{limits: replayTestLimits()}
	if _, e := replayExecution(context.Background(), &trace, block, 0, &b); e != ErrReplayUnavailable {
		t.Fatal("unprotected", e)
	}
}
func replayAddressHash(h common.Hash) string { return strings.ToLower(h.Hex()) }

func TestReplayUnrelatedBlocksAndCanonicalRoots(t *testing.T) {
	in := replayTestInput(t)
	raw := in.Blocks[0]
	var h types.Header
	if rlp.DecodeBytes(raw.HeaderRLP, &h) != nil {
		t.Fatal("header")
	}
	h.GasUsed = 0
	h.TxHash = types.DeriveSha(types.Transactions{}, trie.NewStackTrie(nil))
	h.ReceiptHash = types.DeriveSha(types.Receipts{}, trie.NewStackTrie(nil))
	raw.HeaderRLP, _ = rlp.EncodeToBytes(&h)
	raw.Transactions = nil
	raw.Receipts = nil
	raw.Executions = nil
	b := replayBudget{limits: replayTestLimits()}
	decoded, e := replayDecodeBlock(context.Background(), raw, &b)
	if e != nil {
		t.Fatal(e)
	}
	r := replayRun{}
	if e := r.block(context.Background(), raw, decoded); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := replayDecodeBlock(ctx, raw, &b); e != context.Canceled {
		t.Fatal(e)
	}
	for _, bad := range []ReplayBlock{{HeaderRLP: bytes.Repeat([]byte{0}, 65537)}, {HeaderRLP: raw.HeaderRLP, Transactions: [][]byte{{1}}}} {
		b = replayBudget{limits: replayTestLimits()}
		if _, e := replayDecodeBlock(context.Background(), bad, &b); e == nil {
			t.Fatal("budget/frame")
		}
	}
	if b.charge(-1) {
		t.Fatal("negative charge")
	}
}
