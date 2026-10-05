package guard010_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

type issueFixture struct {
	clientFixture
	capture                                         *g.RootCapture
	signs, approvals, measures                      int
	deny, changed, failSign, wrongSign, panicPolicy bool
	path                                            string
	finalMeasure                                    int
	changeAfterSign                                 bool
}

func (f *issueFixture) ActiveKey(ctx context.Context, issuer, kid string) (ed25519.PublicKey, error) {
	if issuer == f.f.s("expected_issuer") {
		if kid != issuer+"#signing-1" {
			return nil, g.ErrInvalid
		}
		raw, err := hex.DecodeString(f.f.s("public_key_hex"))
		if err != nil || !f.f.yes("active_key") {
			return nil, g.ErrInvalid
		}
		return ed25519.PublicKey(raw), nil
	}
	return f.clientFixture.ActiveKey(ctx, issuer, kid)
}
func newIssueFixture(t *testing.T) *issueFixture {
	t.Helper()
	v := clientVectors(t)
	capture, err := g.NewRootCapture([][]byte{[]byte("trusted root input")}, "00000000-0000-4000-8000-000000000002")
	if err != nil {
		t.Fatal(err)
	}
	d, err := g.OriginalCommitment([][]byte{[]byte("trusted root input")})
	if err != nil {
		t.Fatal(err)
	}
	v.Input["original_digest"] = mustJSON(t, d)
	return &issueFixture{clientFixture: clientFixture{f: v.Input, utc: 1700000000000, clockOK: true, resultActive: true, pub: v.Public}, capture: capture}
}
func (f *issueFixture) ApproveIntent(_ context.Context, raw []byte) error {
	f.approvals++
	if f.panicPolicy {
		panic("bounded fixture evaluator failure")
	}
	var i map[string]any
	if f.deny || json.Unmarshal(raw, &i) != nil || i["recipient"] != f.f.s("expected_recipient") || i["alg"] != "ed25519" {
		return g.ErrInvalid
	}
	return nil
}
func (f *issueFixture) Check(_ context.Context, manifest, tool string) error {
	f.measures++
	md, err := g.ManifestCommitment(f.f["approved_manifest"])
	if f.changed || f.measures == f.finalMeasure || err != nil || manifest != md || tool != "read" {
		return g.ErrInvalid
	}
	return nil
}
func (f *issueFixture) Sign(_ context.Context, kid string, body []byte) ([]byte, error) {
	f.signs++
	if f.approvals == 0 || f.measures == 0 || kid != f.f.s("expected_issuer")+"#signing-1" ||
		!bytes.HasPrefix(body, []byte("sage-execution-intent|0.10.0\x00")) {
		return nil, g.ErrInvalid
	}
	if _, err := os.Stat(f.path + ".issuance"); err != nil {
		return nil, g.ErrInvalid
	}
	if f.failSign {
		return nil, g.ErrInvalid
	}
	seed := sha256.Sum256([]byte("public Guard fixture issuer"))
	if f.wrongSign {
		seed = sha256.Sum256([]byte("public unrelated fixture signer"))
	}
	proof := ed25519.Sign(ed25519.NewKeyFromSeed(seed[:]), body)
	if f.changeAfterSign {
		f.changed = true
	}
	return proof, nil
}
func (f *issueFixture) config() g.IssuerServices {
	c := f.services()
	c.IntentAuthority = f
	c.Policy = f
	return g.IssuerServices{Client: c, Policy: f, Signer: f, Measurement: f, KeyID: f.f.s("expected_issuer") + "#signing-1"}
}
func (f *issueFixture) issuer(t *testing.T) *g.IntentIssuer {
	t.Helper()
	i, err := g.NewIntentIssuer(f.capture, f.config())
	if err != nil {
		t.Fatal(err)
	}
	return i
}
func proposal() g.IntentProposal {
	return g.IntentProposal{Tool: "read", Arguments: []byte(`{"path":"notes.txt"}`), LifetimeSeconds: 300}
}
func issueToken(t *testing.T, i *g.IntentIssuer) *g.AuthorizedIntent {
	t.Helper()
	a, err := i.Authorize(context.Background(), proposal())
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestIntentIssuerJournalsBeforeSendAndReopensWithoutSigning(t *testing.T) {
	f := newIssueFixture(t)
	f.path = filepath.Join(t.TempDir(), "operation")
	issuer := f.issuer(t)
	p := proposal()
	a, err := issuer.Authorize(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	copy(p.Arguments, []byte(`{"path":"other.txt"}`))
	if f.signs != 0 || f.sends != 0 {
		t.Fatal("authorization used key or transport")
	}
	client, err := issuer.Issue(context.Background(), f.path, a)
	if err != nil {
		t.Fatal(err)
	}
	if f.signs != 1 || f.sends != 0 {
		t.Fatal("issue used transport or signed more than once")
	}
	first, err := client.Begin(context.Background(), "00000000-0000-4000-8000-000000000020")
	if err != nil {
		t.Fatal(err)
	}
	raw := first.Intent()
	v, err := g.VerifyIntent(context.Background(), raw, f.f.s("expected_recipient"), f, f)
	if err != nil || !bytes.Equal(v.Canonical(), raw) {
		t.Fatal("issued signature invalid", err)
	}
	var e map[string]any
	if json.Unmarshal(raw, &e) != nil {
		t.Fatal("envelope")
	}
	i := e["intent"].(map[string]any)
	if i["arguments"].(map[string]any)["path"] != "notes.txt" || i["keyid"] != f.f.s("expected_issuer")+"#signing-1" {
		t.Fatal("proposal mutated signed request")
	}
	if err = client.Close(); err != nil {
		t.Fatal(err)
	}
	resumed, err := f.issuer(t).Reopen(context.Background(), f.path)
	if err != nil {
		t.Fatal(err)
	}
	f.utc += 1000
	f.mono += 1000
	again, err := resumed.Begin(context.Background(), "00000000-0000-4000-8000-000000000021")
	if err != nil || !bytes.Equal(again.Intent(), raw) || f.signs != 1 || f.sends != 2 {
		t.Fatal("reopen changed identity or signed again", err)
	}
	if err = resumed.Close(); err != nil {
		t.Fatal(err)
	}
}
func TestIntentIssuerDeniesChangedAuthorityBeforeKeyUse(t *testing.T) {
	for _, mode := range []string{"policy", "epoch", "original", "measurement", "key", "expiry", "retirement", "cancellation"} {
		t.Run(mode, func(t *testing.T) {
			f := newIssueFixture(t)
			f.path = filepath.Join(t.TempDir(), "operation")
			issuer := f.issuer(t)
			a := issueToken(t, issuer)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch mode {
			case "policy":
				f.deny = true
			case "epoch":
				var descriptor map[string]any
				if json.Unmarshal(f.f["approved_policy"], &descriptor) != nil {
					t.Fatal("policy")
				}
				descriptor["epoch"] = "00000000-0000-4000-8000-000000000099"
				f.f["approved_policy"] = mustJSON(t, descriptor)
			case "original":
				f.f["original_digest"] = mustJSON(t, "0000000000000000000000000000000000000000000000000000000000000000")
			case "measurement":
				f.changed = true
			case "key":
				f.f["active_key"] = []byte("false")
			case "expiry":
				f.utc += 300000
			case "retirement":
				_ = issuer.Retire()
			case "cancellation":
				cancel()
			}
			if c, err := issuer.Issue(ctx, f.path, a); err == nil || c != nil || f.signs != 0 || f.sends != 0 {
				t.Fatal("denial used key or transport")
			}
			if _, err := os.Stat(f.path); !os.IsNotExist(err) {
				t.Fatal("denial created Client journal")
			}
			f.deny = false
			f.changed = false
			if c, err := issuer.Issue(context.Background(), f.path, a); err == nil || c != nil || f.signs != 0 {
				t.Fatal("consumed decision was reused")
			}
		})
	}
}
func TestIntentIssuerDurableFailureFence(t *testing.T) {
	for _, mode := range []string{"signer-failure", "wrong-proof", "partial-fence"} {
		t.Run(mode, func(t *testing.T) {
			f := newIssueFixture(t)
			f.path = filepath.Join(t.TempDir(), "operation")
			f.failSign = mode == "signer-failure"
			f.wrongSign = mode == "wrong-proof"
			if mode == "partial-fence" {
				if err := os.WriteFile(f.path+".issuance", nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			issuer := f.issuer(t)
			if c, err := issuer.Issue(context.Background(), f.path, issueToken(t, issuer)); err == nil || c != nil {
				t.Fatal("failed issuance returned Client")
			}
			before := f.signs
			f.failSign = false
			f.wrongSign = false
			restarted := f.issuer(t)
			if c, err := restarted.Issue(context.Background(), f.path, issueToken(t, restarted)); err == nil || c != nil || f.signs != before {
				t.Fatal("restart re-signed fenced operation")
			}
			if c, err := f.issuer(t).Reopen(context.Background(), f.path); err == nil || c != nil {
				t.Fatal("partial issuance resumed")
			}
			if _, err := os.Stat(f.path + ".issuance"); err != nil {
				t.Fatal("failure removed fence")
			}
		})
	}
}
func TestIntentIssuerOwnerBindingAndConcurrentSingleUse(t *testing.T) {
	f := newIssueFixture(t)
	f.path = filepath.Join(t.TempDir(), "operation")
	issuer := f.issuer(t)
	token := issueToken(t, issuer)
	if c, err := f.issuer(t).Issue(context.Background(), f.path, token); err == nil || c != nil || f.signs != 0 {
		t.Fatal("foreign token used key")
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := issuer.Issue(context.Background(), f.path, token)
			if err == nil {
				mu.Lock()
				success++
				mu.Unlock()
				_ = c.Close()
			}
		}()
	}
	wg.Wait()
	if success != 1 || f.signs != 1 {
		t.Fatal("decision reused", success, f.signs)
	}
}
func TestIntentIssuerRejectsInvalidProposalAndPanickedPolicy(t *testing.T) {
	for _, p := range []g.IntentProposal{{Tool: "read", Arguments: []byte(`{"path":"x","path":"y"}`), LifetimeSeconds: 300}, {Tool: "sage_secure_call", Arguments: []byte(`{}`), LifetimeSeconds: 300}, {Tool: "read", Arguments: []byte(`{"path":"x"}`), LifetimeSeconds: 301}, {Tool: "read", Arguments: []byte(`{"unexpected":"x"}`), LifetimeSeconds: 300}} {
		f := newIssueFixture(t)
		issuer := f.issuer(t)
		if a, err := issuer.Authorize(context.Background(), p); err == nil || a != nil || f.signs != 0 {
			t.Fatal("invalid proposal authorized")
		}
	}
	f := newIssueFixture(t)
	f.panicPolicy = true
	issuer := f.issuer(t)
	if a, err := issuer.Authorize(context.Background(), proposal()); err == nil || a != nil {
		t.Fatal("panic returned approval")
	}
	f.panicPolicy = false
	if a, err := issuer.Authorize(context.Background(), proposal()); err == nil || a != nil {
		t.Fatal("panicked issuer resumed")
	}
}
func TestHopIntentIssuerRequiresAdmittedParent(t *testing.T) {
	for _, allowed := range []bool{true, false} {
		t.Run(map[bool]string{true: "admitted", false: "retired"}[allowed], func(t *testing.T) {
			incoming, _, parent, child := hopTestInput(t)
			f := newIssueFixture(t)
			f.f = child.guardFixture
			capture, err := g.NewRootCapture([][]byte{incoming}, "00000000-0000-4000-8000-000000000011")
			if err != nil {
				t.Fatal(err)
			}
			f.capture = capture
			f.path = filepath.Join(t.TempDir(), "operation")
			issuer, err := g.NewHopIntentIssuer(capture, f.config(), incoming, g.HopServices{Authority: parent, Policy: parent, Parent: parent})
			if err != nil {
				t.Fatal(err)
			}
			token := issueToken(t, issuer)
			parent.parentAllowed = allowed
			c, err := issuer.Issue(context.Background(), f.path, token)
			if !allowed {
				if err == nil || c != nil || f.signs != 0 {
					t.Fatal("unadmitted parent signed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			ticket, err := c.Begin(context.Background(), "00000000-0000-4000-8000-000000000030")
			if err != nil {
				t.Fatal(err)
			}
			var env, upstream map[string]any
			_ = json.Unmarshal(ticket.Intent(), &env)
			_ = json.Unmarshal(incoming, &upstream)
			if env["intent"].(map[string]any)["parent_call_id"] != upstream["intent"].(map[string]any)["call_id"] {
				t.Fatal("wrong causal parent")
			}
			_ = c.Close()
		})
	}
}

func TestIntentIssuerProcessHelper(t *testing.T) {
	path := os.Getenv("SAGE_INTENT_ISSUANCE_TEST_PATH")
	if path == "" {
		t.Skip("bounded subprocess fixture")
	}
	f := newIssueFixture(t)
	f.path = path
	issuer := f.issuer(t)
	switch os.Getenv("SAGE_INTENT_ISSUANCE_TEST_MODE") {
	case "issue":
		client, err := issuer.Issue(context.Background(), path, issueToken(t, issuer))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = client.Begin(context.Background(), "00000000-0000-4000-8000-000000000040"); err != nil {
			t.Fatal(err)
		}
		if f.signs != 1 {
			t.Fatal("issue signature count")
		}
		if err = client.Close(); err != nil {
			t.Fatal(err)
		}
	case "reopen":
		client, err := issuer.Reopen(context.Background(), path)
		if err != nil {
			t.Fatal(err)
		}
		f.utc += 1000
		f.mono += 1000
		if _, err = client.Begin(context.Background(), "00000000-0000-4000-8000-000000000041"); err != nil {
			t.Fatal(err)
		}
		if f.signs != 0 {
			t.Fatal("process recovery signed again")
		}
		if err = client.Close(); err != nil {
			t.Fatal(err)
		}
	case "sign-failure", "fenced":
		f.failSign = os.Getenv("SAGE_INTENT_ISSUANCE_TEST_MODE") == "sign-failure"
		if c, err := issuer.Issue(context.Background(), path, issueToken(t, issuer)); err == nil || c != nil {
			t.Fatal("failure accepted")
		}
		if f.failSign && f.signs != 1 || !f.failSign && f.signs != 0 {
			t.Fatal("fence key-use count")
		}
	default:
		t.Fatal("unknown bounded fixture")
	}
}
func TestIntentIssuerProcessRestart(t *testing.T) {
	run := func(path, mode string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestIntentIssuerProcessHelper$")
		command.Env = append(os.Environ(), "SAGE_INTENT_ISSUANCE_TEST_PATH="+path, "SAGE_INTENT_ISSUANCE_TEST_MODE="+mode)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("%s process: %v: %s", mode, err, output)
		}
	}
	path := filepath.Join(t.TempDir(), "operation")
	run(path, "issue")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	run(path, "reopen")
	after, err := os.ReadFile(path)
	if err != nil || !bytes.HasPrefix(after, before) {
		t.Fatal("process restart replaced original journal")
	}
	failed := filepath.Join(t.TempDir(), "failed")
	run(failed, "sign-failure")
	run(failed, "fenced")
	if _, err = os.Stat(failed); !os.IsNotExist(err) {
		t.Fatal("failed process made a Client journal")
	}
}

func TestIntentIssuerRechecksAfterFenceAndAfterSigning(t *testing.T) {
	for _, postSign := range []bool{false, true} {
		t.Run(map[bool]string{false: "after-fence", true: "after-signing"}[postSign], func(t *testing.T) {
			f := newIssueFixture(t)
			f.path = filepath.Join(t.TempDir(), "operation")
			issuer := f.issuer(t)
			token := issueToken(t, issuer)
			if postSign {
				f.changeAfterSign = true
			} else {
				f.finalMeasure = 3
			}
			if client, err := issuer.Issue(context.Background(), f.path, token); err == nil || client != nil {
				t.Fatal("changed final instance returned Client")
			}
			expected := 0
			if postSign {
				expected = 1
			}
			if f.signs != expected || f.sends != 0 {
				t.Fatal("final check key/send count")
			}
			if _, err := os.Stat(f.path); !os.IsNotExist(err) {
				t.Fatal("failed final check created journal")
			}
			if _, err := os.Stat(f.path + ".issuance"); err != nil {
				t.Fatal("failed final check erased fence")
			}
		})
	}
}

func TestIntentIssuerRejectsOtherKeyRoleAndWeakKey(t *testing.T) {
	for _, role := range []string{"kem-1", "weak-signing-1", "foreign"} {
		t.Run(role, func(t *testing.T) {
			f := newIssueFixture(t)
			config := f.config()
			switch role {
			case "kem-1":
				config.KeyID = f.f.s("expected_issuer") + "#kem-1"
			case "weak-signing-1":
				f.f["public_key_hex"] = mustJSON(t, "0000000000000000000000000000000000000000000000000000000000000000")
			case "foreign":
				config.KeyID = "did:sage:web:agents.example.com:other#signing-1"
			}
			issuer, err := g.NewIntentIssuer(f.capture, config)
			if err == nil {
				if token, e := issuer.Authorize(context.Background(), proposal()); e == nil || token != nil {
					t.Fatal("wrong signing role or material approved")
				}
			}
			if f.signs != 0 {
				t.Fatal("fallback used private key")
			}
		})
	}
}
