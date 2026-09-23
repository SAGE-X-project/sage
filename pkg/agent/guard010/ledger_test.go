package guard010_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	g "github.com/sage-x-project/sage/pkg/agent/guard010"
)

func bridgeFixture(t *testing.T) guardFixture {
	t.Helper()
	for _, c := range cases(t) {
		if c.ID == "intent-valid" {
			var f guardFixture
			if json.Unmarshal(c.Input, &f) != nil {
				t.Fatal("fixture")
			}
			return f
		}
	}
	t.Fatal("missing fixture")
	return nil
}
func bridgeOpen(t *testing.T, recipients ...string) (*g.GuardLedger, string) {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("durable storage unsupported")
	}
	path := filepath.Join(t.TempDir(), "journal")
	f := bridgeFixture(t)
	recipient := f.s("expected_recipient")
	if len(recipients) > 0 {
		recipient = recipients[0]
	}
	l, e := g.OpenLedger(path, true, recipient)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l, path
}
func bridgeRaw(f guardFixture) []byte { b, _ := hex.DecodeString(f.s("envelope_hex")); return b }
func bridgeBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func resign(t *testing.T, f guardFixture, field string, v any, rotate bool) []byte {
	t.Helper()
	var env map[string]any
	if json.Unmarshal(bridgeRaw(f), &env) != nil {
		t.Fatal("envelope")
	}
	i := env["intent"].(map[string]any)
	if field != "" {
		i[field] = v
	}
	b, _ := json.Marshal(i)
	b, e := g.Canonicalize(b)
	if e != nil {
		t.Fatal(e)
	}
	seed := sha256.Sum256([]byte("public Guard fixture issuer"))
	if rotate {
		seed[0] ^= 1
	}
	key := ed25519.NewKeyFromSeed(seed[:])
	env["proof"] = base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, append([]byte("sage-execution-intent|0.10.0\x00"), b...)))
	f["public_key_hex"], _ = json.Marshal(hex.EncodeToString(key.Public().(ed25519.PublicKey)))
	raw, _ := json.Marshal(env)
	return raw
}
func TestBridgeIndependentIntentCases(t *testing.T) {
	for _, c := range cases(t) {
		if c.Operation != "sage.guard.intent.verify" {
			continue
		}
		t.Run(c.ID, func(t *testing.T) {
			var f guardFixture
			_ = json.Unmarshal(c.Input, &f)
			l, path := bridgeOpen(t, f.s("expected_recipient"))
			before := bridgeBytes(t, path)
			r, e := l.Reserve(context.Background(), bridgeRaw(f), f, f)
			accept := c.Expected["verdict"] == "ACCEPT"
			if (e == nil) != accept {
				t.Fatal(e, c.Expected)
			}
			if accept {
				if !r.Created() || r.State() != "RESERVED" {
					t.Fatal("missing reservation")
				}
			} else if !bytes.Equal(before, bridgeBytes(t, path)) {
				t.Fatal("invalid intent modified storage")
			}
		})
	}
}

type countingAuthority struct {
	guardFixture
	now, key int
}

func (a *countingAuthority) Now(ctx context.Context) (int64, error) {
	a.now++
	return a.guardFixture.Now(ctx)
}

func (a *countingAuthority) ActiveKey(ctx context.Context, issuer, kid string) (ed25519.PublicKey, error) {
	a.key++
	return a.guardFixture.ActiveKey(ctx, issuer, kid)
}

type countingPolicy struct {
	guardFixture
	bindings, authorize int
}

func (p *countingPolicy) Bindings(ctx context.Context, issuer, requestID string) (string, []byte, []byte, error) {
	p.bindings++
	return p.guardFixture.Bindings(ctx, issuer, requestID)
}

func (p *countingPolicy) Authorize(ctx context.Context, issuer, tool string, args []byte) error {
	p.authorize++
	return p.guardFixture.Authorize(ctx, issuer, tool, args)
}

func intentAlgorithmEnvelope(t *testing.T, f guardFixture, algorithm string) []byte {
	t.Helper()
	var envelope map[string]any
	if json.Unmarshal(bridgeRaw(f), &envelope) != nil {
		t.Fatal("envelope")
	}
	intent := envelope["intent"].(map[string]any)
	intent["alg"] = algorithm
	body, err := g.Canonicalize(mustJSON(t, intent))
	if err != nil {
		t.Fatal(err)
	}
	message := append([]byte("sage-execution-intent|0.10.0\x00"), body...)
	var proof []byte
	switch algorithm {
	case "ed25519":
		seed := sha256.Sum256([]byte("public Guard fixture issuer"))
		key := ed25519.NewKeyFromSeed(seed[:])
		proof = ed25519.Sign(key, message)
		if !ed25519.Verify(key.Public().(ed25519.PublicKey), message, proof) {
			t.Fatal("invalid Ed25519 control")
		}
		f["public_key_hex"], _ = json.Marshal(hex.EncodeToString(key.Public().(ed25519.PublicKey)))
	case "ecdsa-p256-sha256":
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(message)
		r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
		if err != nil || !ecdsa.Verify(&key.PublicKey, digest[:], r, s) {
			t.Fatal("invalid P-256 control", err)
		}
		proof = make([]byte, 64)
		r.FillBytes(proof[:32])
		s.FillBytes(proof[32:])
	case "secp256k1":
		key, err := ethcrypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		digest := ethcrypto.Keccak256(message)
		signature, err := ethcrypto.Sign(digest, key)
		if err != nil || !ethcrypto.VerifySignature(ethcrypto.FromECDSAPub(&key.PublicKey), digest, signature[:64]) {
			t.Fatal("invalid secp256k1 control", err)
		}
		proof = signature[:64]
	default:
		t.Fatal("unexpected algorithm")
	}
	envelope["proof"] = base64.RawURLEncoding.EncodeToString(proof)
	return mustJSON(t, envelope)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestBridgeIntentSignatureAlgorithmBoundary(t *testing.T) {
	for _, algorithm := range []string{"ed25519", "ecdsa-p256-sha256", "secp256k1"} {
		t.Run(algorithm, func(t *testing.T) {
			fixture := bridgeFixture(t)
			raw := intentAlgorithmEnvelope(t, fixture, algorithm)
			ledger, path := bridgeOpen(t, fixture.s("expected_recipient"))
			before := bridgeBytes(t, path)
			authority := &countingAuthority{guardFixture: fixture}
			policy := &countingPolicy{guardFixture: fixture}
			reservation, err := ledger.Reserve(context.Background(), raw, authority, policy)
			if algorithm == "ed25519" {
				if err != nil || reservation == nil || !reservation.Created() || authority.key == 0 || policy.bindings == 0 || policy.authorize == 0 || bytes.Equal(before, bridgeBytes(t, path)) {
					t.Fatal("Ed25519 did not continue complete validation", err)
				}
				return
			}
			if err == nil || reservation != nil || authority.now != 0 || authority.key != 0 || policy.bindings != 0 || policy.authorize != 0 || !bytes.Equal(before, bridgeBytes(t, path)) {
				t.Fatal("unsupported algorithm crossed the intent schema boundary", algorithm, err)
			}
		})
	}
}
func TestBridgeRetryAndRecovery(t *testing.T) {
	l, path := bridgeOpen(t)
	f := bridgeFixture(t)
	raw := bridgeRaw(f)
	r, e := l.Reserve(context.Background(), raw, f, f)
	if e != nil || !r.Created() {
		t.Fatal(e)
	}
	before := bridgeBytes(t, path)
	var pretty bytes.Buffer
	_ = json.Indent(&pretty, raw, "", " ")
	r, e = l.Reserve(context.Background(), pretty.Bytes(), f, f)
	if e != nil || r.Created() || r.State() != "RESERVED" || !bytes.Equal(before, bridgeBytes(t, path)) {
		t.Fatal("retry changed state", e)
	}
	for _, control := range []string{"active_key", "policy_allow", "clock_trusted"} {
		old := f[control]
		f[control] = json.RawMessage(`false`)
		if _, e = l.Reserve(context.Background(), raw, f, f); e == nil {
			t.Fatal(control)
		}
		f[control] = old
		if !bytes.Equal(before, bridgeBytes(t, path)) {
			t.Fatal("denial changed state")
		}
	}
	if e = l.Close(); e != nil {
		t.Fatal(e)
	}
	l, e = g.OpenLedger(path, false, f.s("expected_recipient"))
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = l.Close() }()
	r, e = l.Reserve(context.Background(), raw, f, f)
	if e != nil || r.Created() || r.State() != "UNKNOWN" {
		t.Fatal("recovery", e)
	}
	before = bridgeBytes(t, path)
	f["now"] = json.RawMessage(`1700000300`)
	if _, e = l.Reserve(context.Background(), raw, f, f); e == nil || !bytes.Equal(before, bridgeBytes(t, path)) {
		t.Fatal("expired query")
	}
}
func TestBridgeSignedConflicts(t *testing.T) {
	for _, kind := range []string{"arguments", "nonce", "call_id", "proof"} {
		t.Run(kind, func(t *testing.T) {
			l, path := bridgeOpen(t)
			f := bridgeFixture(t)
			if _, e := l.Reserve(context.Background(), bridgeRaw(f), f, f); e != nil {
				t.Fatal(e)
			}
			before := bridgeBytes(t, path)
			field := kind
			var v any
			switch kind {
			case "arguments":
				v = map[string]any{"path": "other.txt"}
			case "nonce":
				v = "AQECAwQFBgcICQoLDA0ODw"
			case "call_id":
				v = "00000000-0000-4000-8000-000000000004"
			case "proof":
				field = ""
			}
			raw := resign(t, f, field, v, kind == "proof")
			if _, e := g.VerifyIntent(context.Background(), raw, f.s("expected_recipient"), f, f); e != nil {
				t.Fatal("not a valid signed control", e)
			}
			if _, e := l.Reserve(context.Background(), raw, f, f); e == nil || !bytes.Equal(before, bridgeBytes(t, path)) {
				t.Fatal("conflicting signed identity changed state")
			}
		})
	}
}
func TestBridgeConcurrentIdentical(t *testing.T) {
	l, path := bridgeOpen(t)
	f := bridgeFixture(t)
	raw := bridgeRaw(f)
	var wg sync.WaitGroup
	results := make(chan *g.Reservation, 16)
	errs := make(chan error, 16)
	for range 16 {
		wg.Add(1)
		go func() { defer wg.Done(); r, e := l.Reserve(context.Background(), raw, f, f); results <- r; errs <- e }()
	}
	wg.Wait()
	close(results)
	close(errs)
	created := 0
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	for r := range results {
		if r.Created() {
			created++
		}
	}
	if created != 1 || bytes.Count(bridgeBytes(t, path), []byte("\n")) != 2 {
		t.Fatal("non-atomic retry", created)
	}
}

type expiryClock struct {
	guardFixture
	calls int
}

func (a *expiryClock) Now(ctx context.Context) (int64, error) {
	a.calls++
	if a.calls == 4 {
		return 1700000300, nil
	}
	return a.guardFixture.Now(ctx)
}
func TestBridgeExpiryDuringStorage(t *testing.T) {
	l, path := bridgeOpen(t)
	f := bridgeFixture(t)
	a := &expiryClock{guardFixture: f}
	if _, e := l.Reserve(context.Background(), bridgeRaw(f), a, f); e == nil {
		t.Fatal("expired response exposed")
	}
	if bytes.Count(bridgeBytes(t, path), []byte("\n")) != 2 {
		t.Fatal("durable denial lost")
	}
	r, e := l.Reserve(context.Background(), bridgeRaw(f), f, f)
	if e != nil || r.Created() {
		t.Fatal("duplicate created after denial", e)
	}
}
func TestBridgeMissingExclusiveAndClosed(t *testing.T) {
	l, path := bridgeOpen(t)
	f := bridgeFixture(t)
	if _, e := g.OpenLedger(path, false, f.s("expected_recipient")); e == nil {
		t.Fatal("second writer")
	}
	if _, e := g.OpenLedger(path+"missing", false, f.s("expected_recipient")); e == nil {
		t.Fatal("lost storage recreated")
	}
	_ = l.Close()
	if _, e := l.Reserve(context.Background(), bridgeRaw(f), f, f); e == nil {
		t.Fatal("closed store accepted")
	}
}

func TestBridgeConcurrentNonceConflict(t *testing.T) {
	l, path := bridgeOpen(t)
	var wg sync.WaitGroup
	accepted := make(chan bool, 2)
	for _, id := range []string{"00000000-0000-4000-8000-000000000004", "00000000-0000-4000-8000-000000000005"} {
		f := bridgeFixture(t)
		raw := resign(t, f, "call_id", id, false)
		if _, e := g.VerifyIntent(context.Background(), raw, f.s("expected_recipient"), f, f); e != nil {
			t.Fatal(e)
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := l.Reserve(context.Background(), raw, f, f)
			accepted <- e == nil && r.Created()
		}()
	}
	wg.Wait()
	close(accepted)
	count := 0
	for ok := range accepted {
		if ok {
			count++
		}
	}
	if count != 1 || bytes.Count(bridgeBytes(t, path), []byte("\n")) != 2 {
		t.Fatal("nonce reserved twice", count)
	}
}
