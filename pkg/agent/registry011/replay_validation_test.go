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
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestReplayIndependentDomainsGrammarAndABIs(t *testing.T) {
	raw, e := os.ReadFile("testdata/replay-reference.json")
	if e != nil {
		t.Fatal(e)
	}
	var f struct {
		Registry, Agent, ChallengeHex, ClaimJCS, ClaimSHA256 string
		Key                                                  replayKey
		URICases                                             []struct {
			Name, Value string
			Accepted    bool
		}
	}
	// Explicit tags for independent snake-case fixture fields.
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		t.Fatal("fixture")
	}
	for name, into := range map[string]any{"registry": &f.Registry, "agent": &f.Agent, "challenge_hex": &f.ChallengeHex, "claim_jcs": &f.ClaimJCS, "claim_sha256": &f.ClaimSHA256, "key": &f.Key, "uri_cases": &f.URICases} {
		if json.Unmarshal(fields[name], into) != nil {
			t.Fatal(name)
		}
	}
	if hex.EncodeToString(replayChallenge(f.Registry, f.Agent, f.Key)) != f.ChallengeHex {
		t.Fatal("len16 challenge")
	}
	claim := contentJSON([]byte(f.ClaimJCS)).(map[string]any)
	c := replayCommand{agent: f.Agent, args: map[string]any{"controller": claim["controller"], "keys": claim["keys"], "services": claim["services"], "salt": claim["salt"]}}
	h := replayClaim(f.Registry, c)
	if hex.EncodeToString(h[:]) != f.ClaimSHA256 {
		t.Fatal("claim domain")
	}
	for _, c := range f.URICases {
		t.Run(c.Name, func(t *testing.T) {
			if replayURI(c.Value) != c.Accepted {
				t.Fatalf("URI %q", c.Value)
			}
		})
	}
	for source, want := range map[string]string{replayContractABI: "0f31f995b0169b27c2ce4797b967a754b4c0b2b0a733e4c21da1b54089380f5f", replayOutcomesABI: "7d2c1fd84fcc83e3718512c36089b42779fe999f295d4185c0e229b8cf2038c8"} {
		h := sha256.Sum256([]byte(source))
		if hex.EncodeToString(h[:]) != want {
			t.Fatal("adopted ABI changed")
		}
	}
}

func TestReplayClosedCommandsAndABIFraming(t *testing.T) {
	a, _, _ := replayABIs()
	base := replayTestCreate(t)
	changes := map[string]func(map[string]any){
		"binding": func(m map[string]any) { m["binding"] = "sage-0.10.0" }, "registry": func(m map[string]any) { m["registry"] = "wrong" }, "did": func(m map[string]any) { m["did"] = "wrong" },
		"extra-root": func(m map[string]any) { m["extra"] = 0 }, "unknown-operation": func(m map[string]any) { m["operation"] = "transfer" }, "actor": func(m map[string]any) { m["actor"] = "0X0000000000000000000000000000000000001000" },
		"expected-type": func(m map[string]any) { m["expectedVersion"] = 0 }, "create-expected": func(m map[string]any) { m["expectedVersion"] = "1" },
		"controller": func(m map[string]any) { m["arguments"].(map[string]any)["controller"] = replayTestOperator }, "salt": func(m map[string]any) { m["arguments"].(map[string]any)["salt"] = "bad" },
		"keys": func(m map[string]any) { m["arguments"].(map[string]any)["keys"] = []any{} }, "services": func(m map[string]any) { m["arguments"].(map[string]any)["services"] = nil },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			raw := contentEdit(t, base.raw, change)
			if _, ok := replayParseCommand(replayTestRegistry, replayTestAgent, raw, a); ok {
				t.Fatal("accepted")
			}
		})
	}
	for _, step := range replayTestSteps(t) {
		c, ok := replayParseCommand(replayTestRegistry, replayTestAgent, step.History.Command, a)
		if !ok {
			t.Fatal("parse")
		}
		got, ok := replayDecodeCommand(replayTestRegistry, c.actor, c.calldata, a)
		if !ok || !bytes.Equal(got.raw, c.raw) {
			t.Fatalf("round trip %s", c.op)
		}
		method, _ := a.MethodById(c.calldata[:4])
		if !replayCallFraming(*method, c.calldata) {
			t.Fatal("normal framing")
		}
		for _, mutate := range []func([]byte) []byte{func(x []byte) []byte { return x[:3] }, func(x []byte) []byte { x[0] ^= 255; return x }, func(x []byte) []byte { return append(x, make([]byte, 32)...) }, func(x []byte) []byte { x[4] = 255; return x }, func(x []byte) []byte { x[35] = 0; return x }} {
			bad := mutate(bytes.Clone(c.calldata))
			if _, ok := replayDecodeCommand(replayTestRegistry, c.actor, bad, a); ok {
				t.Fatalf("bad framing %s", c.op)
			}
		}
		if c.op != "create" {
			for _, expected := range []any{"", "0", "01", "18446744073709551616", 1} {
				bad := contentEdit(t, c.raw, func(m map[string]any) { m["expectedVersion"] = expected })
				if _, ok := replayParseCommand(replayTestRegistry, replayTestAgent, bad, a); ok {
					t.Fatal("expected version")
				}
			}
		}
	}
	// Parser supports explicit zero expiry without converting it to absence.
	zero := uint64(0)
	k := replayTestKey(t, "b", true)
	k.Expires = &zero
	c := replayTestCommand(t, "add-key", replayTestWallet, 1, map[string]any{"key": k})
	got, ok := replayDecodeCommand(replayTestRegistry, c.actor, c.calldata, a)
	if !ok || got.keys[0].Expires == nil || *got.keys[0].Expires != 0 {
		t.Fatal("zero expiry")
	}
	if _, ok := replayDecodeCommand(replayTestRegistry, replayTestWallet, []byte{0, 0, 0, 0}, a); ok {
		t.Fatal("unknown selector")
	}
	for _, name := range []string{"record", "registryId", "commitClaim"} {
		if replayCallFraming(a.Methods[name], make([]byte, 36)) {
			t.Fatal("non-mutation framing")
		}
	}
}

func TestReplayNestedKeyServiceAndProofRules(t *testing.T) {
	did := "did:sage:" + replayTestRegistry + ":" + replayTestAgent
	k := replayTestKey(t, "a", false)
	v := contentJSON(replayTestJSON(t, []replayKey{k})).([]any)
	for name, change := range map[string]func(map[string]any){
		"name-empty": func(m map[string]any) { m["name"] = "" }, "name-long": func(m map[string]any) { m["name"] = strings.Repeat("a", 33) }, "name-char": func(m map[string]any) { m["name"] = "a.b" },
		"unsupported-before-key": func(m map[string]any) { m["alg"] = "p256"; m["key"] = nil }, "key-type": func(m map[string]any) { m["key"] = 0 }, "key-length": func(m map[string]any) { m["key"] = "AA" },
		"key-alphabet": func(m map[string]any) { m["key"] = strings.Repeat("+", 43) }, "state": func(m map[string]any) { m["state"] = "revoked" }, "extra": func(m map[string]any) { m["extra"] = 0 },
		"proof-extra": func(m map[string]any) { m["proof"].(map[string]any)["extra"] = 0 }, "proof-value": func(m map[string]any) { m["proof"].(map[string]any)["value"] = "AA" },
		"proof-cross-record": func(m map[string]any) { m["proof"].(map[string]any)["signer"] = "did:other#a" }, "proof-bad-name": func(m map[string]any) { m["proof"].(map[string]any)["signer"] = did + "#a.b" },
		"proof-other-signer": func(m map[string]any) { m["proof"].(map[string]any)["signer"] = did + "#b" }, "expiry": func(m map[string]any) { m["expires"] = 9007199254740992 },
	} {
		t.Run(name, func(t *testing.T) {
			raw := contentEdit(t, replayTestJSON(t, v[0]), change)
			if _, ok := replayKeys([]any{contentJSON(raw)}, did); ok {
				t.Fatal("key accepted")
			}
		})
	}
	for _, bad := range []any{nil, 0, []any{}, []any{nil}, []any{v[0], v[0]}} {
		if _, ok := replayKeys(bad, did); ok {
			t.Fatal("key array")
		}
	}
	other := replayTestClone(t, v[0].(map[string]any))
	other["name"] = "b"
	other["proof"].(map[string]any)["signer"] = did + "#b"
	if _, ok := replayKeys([]any{v[0], other}, did); ok {
		t.Fatal("material reused")
	}
	for _, s := range []replayService{{"api", "A2A", "https://example.invalid"}, {"api", "\x00\n\x7f", "https://[::1]:"}} {
		if _, ok := replayServices(contentJSON(replayTestJSON(t, []replayService{s})), []replayKey{k}); !ok {
			t.Fatal("normal service")
		}
	}
	for _, bad := range []any{nil, []any{nil}, []any{map[string]any{"name": "a", "type": "A2A", "uri": "https://x"}}, []any{map[string]any{"name": "api", "type": "", "uri": "https://x"}}, []any{map[string]any{"name": "api", "type": "A2A", "uri": "http://x"}}} {
		if _, ok := replayServices(bad, []replayKey{k}); ok {
			t.Fatal("service accepted")
		}
	}
	if !replayVerifyNewKeys(replayTestRegistry, replayTestAgent, []replayKey{k, replayTestKEM(t, k)}, []replayKey{k, replayTestKEM(t, k)}, 0) {
		t.Fatal("normal proofs")
	}
	bad := k
	proof, _ := replayBinary(k.Proof.Value, 64)
	proof[0] ^= 1
	bad.Proof.Value = base64.RawURLEncoding.EncodeToString(proof)
	if replayVerifyNewKeys(replayTestRegistry, replayTestAgent, []replayKey{bad}, []replayKey{bad}, 0) {
		t.Fatal("invalid signature")
	}
	kem := replayTestKEM(t, k)
	if replayVerifyNewKeys(replayTestRegistry, replayTestAgent, nil, []replayKey{kem}, 0) {
		t.Fatal("missing endorsement signer")
	}
	revoked := k
	revoked.State = "revoked"
	if replayVerifyNewKeys(replayTestRegistry, replayTestAgent, []replayKey{revoked}, []replayKey{kem}, 0) {
		t.Fatal("revoked endorser")
	}
	zero := uint64(0)
	expired := k
	expired.Expires = &zero
	if replayVerifyNewKeys(replayTestRegistry, replayTestAgent, []replayKey{expired}, []replayKey{kem}, 0) {
		t.Fatal("expired endorser")
	}
	kem.Proof.Value = bad.Proof.Value
	if replayVerifyNewKeys(replayTestRegistry, replayTestAgent, []replayKey{k}, []replayKey{kem}, 0) {
		t.Fatal("bad endorsement")
	}
	// Old endorsements are retained as history, not re-proven by later mutations.
	if !replayVerifyNewKeys(replayTestRegistry, replayTestAgent, []replayKey{revoked, kem}, nil, 100) {
		t.Fatal("historical endorsement")
	}
	identity := make([]byte, 32)
	identity[0] = 1
	for _, point := range [][]byte{nil, identity, bytes.Repeat([]byte{255}, 32), bytes.Repeat([]byte{0}, 32)} {
		if replayPrimeEdPoint(point) {
			t.Fatal("non-prime/canonical point")
		}
	}
	pub, _ := replayBinary(k.Key, 32)
	sig, _ := replayBinary(k.Proof.Value, 64)
	if replayStrictEd(pub, nil, nil) {
		t.Fatal("short signature")
	}
	copy(sig[32:], bytes.Repeat([]byte{255}, 32))
	if replayStrictEd(pub, nil, sig) {
		t.Fatal("noncanonical S")
	}
	if !contentAddress("0x"+strings.Repeat("0", 40)) || contentAddress("0x"+strings.Repeat("A", 40)) {
		t.Fatal("principal grammar")
	}
}

func TestReplayStateACLAndTerminalRecovery(t *testing.T) {
	steps := replayTestSteps(t)
	a, _, _ := replayABIs()
	fresh := func(n int) replayState {
		var s replayState
		for i := 0; i < n; i++ {
			c, ok := replayParseCommand(replayTestRegistry, replayTestAgent, steps[i].History.Command, a)
			if !ok {
				t.Fatal("parse")
			}
			if _, ok = s.apply(replayTestRegistry, c, uint64(i+1)); !ok {
				t.Fatal("normal prefix", i)
			}
		}
		return s
	}
	for name, row := range map[string]struct {
		n         int
		op, actor string
		args      map[string]any
	}{
		"operator-cannot-grant":              {3, "authorize-operator", replayTestOperator, map[string]any{"operator": "0x" + strings.Repeat("0", 40), "scope": "activate"}},
		"operator-cannot-revoke-created-key": {3, "revoke-key", replayTestOperator, map[string]any{"name": "a"}},
		"created-cannot-add":                 {1, "add-key", replayTestWallet, map[string]any{"key": replayTestKey(t, "b", true)}},
		"created-cannot-update":              {1, "update-services", replayTestWallet, map[string]any{"services": []replayService{}}},
		"already-active":                     {4, "activate", replayTestWallet, map[string]any{}}, "operator-scope": {4, "update-services", replayTestOperator, map[string]any{"services": []replayService{}}},
		"grant-controller":    {1, "authorize-operator", replayTestWallet, map[string]any{"operator": replayTestWallet, "scope": "activate"}},
		"wrong-created-scope": {1, "authorize-operator", replayTestWallet, map[string]any{"operator": replayTestOperator, "scope": "add-key"}},
		"wrong-active-scope":  {4, "authorize-operator", replayTestWallet, map[string]any{"operator": replayTestOperator, "scope": "activate"}},
		"duplicate-grant":     {2, "authorize-operator", replayTestWallet, map[string]any{"operator": replayTestOperator, "scope": "activate"}},
		"missing-grant":       {1, "revoke-operator", replayTestWallet, map[string]any{"operator": replayTestOperator, "scope": "activate"}},
		"revoke-controller":   {1, "revoke-operator", replayTestWallet, map[string]any{"operator": replayTestWallet, "scope": "activate"}},
		"missing-key":         {4, "revoke-key", replayTestWallet, map[string]any{"name": "missing"}}, "revoked-key": {9, "revoke-key", replayTestWallet, map[string]any{"name": "a"}},
		"reused-key":            {4, "add-key", replayTestWallet, map[string]any{"key": replayTestKey(t, "a", false)}},
		"service-key-collision": {4, "update-services", replayTestWallet, map[string]any{"services": []replayService{{"a", "A2A", "https://x"}}}},
		"terminal":              {11, "deactivate", replayTestWallet, map[string]any{}},
	} {
		t.Run(name, func(t *testing.T) {
			s := fresh(row.n)
			c := replayTestCommand(t, row.op, row.actor, uint64(row.n), row.args)
			if _, ok := s.apply(replayTestRegistry, c, 20); ok {
				t.Fatal("transition accepted")
			}
		})
	}
	s := fresh(1)
	s.record.Keys = s.record.Keys[:1]
	c := replayTestCommand(t, "revoke-key", replayTestWallet, 1, map[string]any{"name": "a"})
	step, ok := s.apply(replayTestRegistry, c, 2)
	if !ok || contentJSON(step.Record).(map[string]any)["state"] != "deactivated" {
		t.Fatal("last signer cleanup")
	}
	s = fresh(1)
	expiry := uint64(2)
	s.record.Keys[0].Expires = &expiry
	c = replayTestCommand(t, "activate", replayTestWallet, 1, map[string]any{})
	if _, ok := s.apply(replayTestRegistry, c, 2); ok {
		t.Fatal("expired signer activates")
	}
	s = fresh(4)
	s.record.Keys[0].Expires = &expiry
	c = replayTestCommand(t, "add-key", replayTestWallet, 4, map[string]any{"key": replayTestKey(t, "b", true)})
	if _, ok := s.apply(replayTestRegistry, c, 5); !ok {
		t.Fatal("expired signer prevents authorized recovery")
	}
	for _, now := range []uint64{0, 1 << 53} {
		s = fresh(1)
		c = replayTestCommand(t, "deactivate", replayTestWallet, 1, map[string]any{})
		if _, ok := s.apply(replayTestRegistry, c, now); ok {
			t.Fatal("time")
		}
	}
	s = fresh(1)
	s.version = ^uint64(0)
	c = replayTestCommand(t, "deactivate", replayTestWallet, ^uint64(0), map[string]any{})
	if _, ok := s.apply(replayTestRegistry, c, 2); ok {
		t.Fatal("version exhaustion")
	}
	s = fresh(1)
	c = replayTestCommand(t, "deactivate", replayTestWallet, 2, map[string]any{})
	if _, ok := s.apply(replayTestRegistry, c, 2); ok {
		t.Fatal("wrong predecessor")
	}
	// Zero principal is an allowed grant target (not an extra reserved value).
	s = fresh(1)
	c = replayTestCommand(t, "authorize-operator", replayTestWallet, 1, map[string]any{"operator": "0x" + strings.Repeat("0", 40), "scope": "activate"})
	if _, ok := s.apply(replayTestRegistry, c, 2); !ok {
		t.Fatal("zero operator")
	}
}

func TestReplayBlockRootLineageAndReceiptControls(t *testing.T) {
	base := replayTestInput(t)
	for name, change := range map[string]func(*types.Header){"height": func(h *types.Header) { h.Number = big.NewInt(2) }, "parent": func(h *types.Header) { h.ParentHash = common.Hash{1} }, "time": func(h *types.Header) { h.Time = 1 << 53 }, "gas": func(h *types.Header) { h.GasUsed++ }, "transactions-root": func(h *types.Header) { h.TxHash[0] ^= 1 }, "receipts-root": func(h *types.Header) { h.ReceiptHash[0] ^= 1 }} {
		t.Run(name, func(t *testing.T) {
			in := replayTestClone(t, base)
			var h types.Header
			if rlp.DecodeBytes(in.Blocks[1].HeaderRLP, &h) != nil {
				t.Fatal("decode")
			}
			change(&h)
			in.Blocks[1].HeaderRLP, _ = rlp.EncodeToBytes(&h)
			if got, e := ReplayCommands(context.Background(), in, replayTestLimits()); e == nil || got != nil {
				t.Fatal("bad block")
			}
		})
	}
	for name, change := range map[string]func(*types.Receipt){"type": func(r *types.Receipt) { r.Type = types.LegacyTxType }, "failed-logs": func(r *types.Receipt) { r.Status = 0 }, "gas": func(r *types.Receipt) { r.CumulativeGasUsed++ }} {
		t.Run(name, func(t *testing.T) {
			in := replayTestClone(t, base)
			var r types.Receipt
			if r.UnmarshalBinary(in.Blocks[0].Receipts[0]) != nil {
				t.Fatal("receipt")
			}
			change(&r)
			in.Blocks[0].Receipts[0], _ = r.MarshalBinary()
			if got, e := ReplayCommands(context.Background(), in, replayTestLimits()); e == nil || got != nil {
				t.Fatal("bad receipt")
			}
		})
	}
}

func TestReplayCallContextAndLogOrdering(t *testing.T) {
	// A wallet implementation reached via DELEGATECALL executes in wallet
	// storage context. Its subsequent Registry CALL has the wallet as From.
	root := ReplayFrame{Type: "CALL", From: replayTestOperator, To: replayTestWallet, Value: "0x0", Calls: []ReplayFrame{{Type: "DELEGATECALL", From: replayTestWallet, To: replayTestOperator, Value: "0x0", Calls: []ReplayFrame{{Type: "CALL", From: replayTestWallet, To: SepoliaRegistryAddress, Value: "0x0", Logs: []ReplayFrameLog{{Address: SepoliaRegistryAddress}}}}}}}
	b := replayBudget{limits: replayTestLimits()}
	got, e := replayFrame(context.Background(), &root, "", false, true, 0, &b)
	if e != nil || len(got) != 1 || !got[0].eligible || got[0].frame.From != replayTestWallet {
		t.Fatal(e)
	}
	root.Calls[0].Type = "CALLCODE"
	b = replayBudget{limits: replayTestLimits()}
	if got, e := replayFrame(context.Background(), &root, "", false, true, 0, &b); e != nil || !got[0].eligible {
		t.Fatal(e)
	}
	root.Calls[0].Type = "STATICCALL"
	root.Calls[0].Calls[0].From = replayTestOperator
	b = replayBudget{limits: replayTestLimits()}
	got, e = replayFrame(context.Background(), &root, "", false, true, 0, &b)
	if e != nil || got[0].eligible {
		t.Fatal("static ancestry", e)
	}
	root.Calls[0].Type = "CREATE2"
	b = replayBudget{limits: replayTestLimits()}
	if _, e := replayFrame(context.Background(), &root, "", false, true, 0, &b); e != nil {
		t.Fatal(e)
	}
	root = ReplayFrame{Type: "CALL", From: replayTestOperator, To: replayTestWallet, Value: "0x0", Calls: []ReplayFrame{{Type: "CALL", From: replayTestWallet, To: replayTestOperator, Value: "0x0", Logs: []ReplayFrameLog{{Address: replayTestOperator, Data: []byte{2}}}}}, Logs: []ReplayFrameLog{{Address: replayTestWallet, Data: []byte{1}}, {Address: replayTestWallet, Data: []byte{3}, AfterChildren: 1}}}
	b = replayBudget{limits: replayTestLimits()}
	got, e = replayFrame(context.Background(), &root, "", false, true, 0, &b)
	if e != nil || len(got) != 3 || got[0].log.Data[0] != 1 || got[1].log.Data[0] != 2 || got[2].log.Data[0] != 3 {
		t.Fatal("log order", e)
	}
	root.Calls[0].Error = "reverted"
	b = replayBudget{limits: replayTestLimits()}
	got, e = replayFrame(context.Background(), &root, "", false, true, 0, &b)
	if e != nil || len(got) != 2 {
		t.Fatal("reverted logs", e)
	}
	for _, s := range []string{"", "0x", "0x00", "0xA", "1", "0x" + strings.Repeat("f", 65)} {
		if _, ok := replayValue(s); ok {
			t.Fatal("value", s)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := replayFrame(ctx, &root, "", false, true, 0, &b); e != context.Canceled {
		t.Fatal(e)
	}
}

func TestReplayClaimPresenceConsumptionAndWindow(t *testing.T) {
	in := replayTestInput(t)
	a, out, _ := replayABIs()
	create, ok := replayParseCommand(in.Registry, in.Agent, in.History[0].Command, a)
	if !ok {
		t.Fatal("parse")
	}
	commit := in.Blocks[0].Executions[0].Root.Calls[0]
	reveal := in.Blocks[1].Executions[0].Root.Calls[0]
	digest := replayClaim(in.Registry, create)
	id := replayClaimID{replayTestWallet, digest}
	newRun := func() *replayRun {
		b := replayBudget{limits: replayTestLimits()}
		return &replayRun{registry: in.Registry, agent: in.Agent, contract: a, outcomes: out, claims: map[replayClaimID]replayClaimState{}, created: map[string]bool{}, budget: &b, history: in.History}
	}
	log := func(f *ReplayFrame) replayLog {
		l := f.Logs[0]
		return replayLog{&types.Log{Address: common.HexToAddress(l.Address), Topics: l.Topics, Data: l.Data}, f, true}
	}
	r := newRun()
	if e := r.event(context.Background(), log(&commit), &types.Header{Number: big.NewInt(0)}); e != nil {
		t.Fatal(e)
	}
	if e := r.event(context.Background(), log(&commit), &types.Header{Number: big.NewInt(0)}); e == nil {
		t.Fatal("duplicate refresh")
	}
	for name, state := range map[string]*replayClaimState{"absent": nil, "same-block": {block: 1}, "expired": {block: 0}, "consumed": {block: 0, consumed: true}, "future": {block: 300}} {
		t.Run(name, func(t *testing.T) {
			r := newRun()
			if state != nil {
				r.claims[id] = *state
			}
			height := int64(1)
			if name == "expired" {
				height = 257
			}
			if e := r.event(context.Background(), log(&reveal), &types.Header{Number: big.NewInt(height), Time: 1}); e == nil {
				t.Fatal("bad claim")
			}
		})
	}
	r = newRun()
	r.claims[id] = replayClaimState{block: 0}
	if e := r.event(context.Background(), log(&reveal), &types.Header{Number: big.NewInt(256), Time: 1}); e != nil || !r.claims[id].consumed {
		t.Fatal("inclusive window", e)
	}
	r = newRun()
	r.claims[id] = replayClaimState{block: 0, consumed: true}
	if e := r.event(context.Background(), log(&commit), &types.Header{Number: big.NewInt(0)}); e == nil {
		t.Fatal("consumed refresh")
	}
	// Same commitment under another caller cannot satisfy this caller's reveal.
	r = newRun()
	r.claims[replayClaimID{replayTestOperator, digest}] = replayClaimState{block: 0}
	if e := r.event(context.Background(), log(&reveal), &types.Header{Number: big.NewInt(1), Time: 1}); e == nil {
		t.Fatal("cross-caller")
	}
}

func TestReplayStrictSignatureSeparatesAccountAuthority(t *testing.T) {
	// A normal valid PoP does not authorize a different Registry principal.
	c := replayTestCreate(t)
	s := replayState{}
	if _, ok := s.apply(replayTestRegistry, c, 1); !ok {
		t.Fatal("create")
	}
	c = replayTestCommand(t, "deactivate", replayTestOperator, 1, map[string]any{})
	if _, ok := s.apply(replayTestRegistry, c, 2); ok {
		t.Fatal("signature substituted for ACL")
	}
	k := replayTestKey(t, "a", false)
	key, _ := replayBinary(k.Key, ed25519.PublicKeySize)
	sig, _ := replayBinary(k.Proof.Value, ed25519.SignatureSize)
	if replayStrictEd(key, append([]byte("sage-pop-0.10.0"), replayChallenge(replayTestRegistry, replayTestAgent, k)...), sig) {
		t.Fatal("cross-domain signature")
	}
}
