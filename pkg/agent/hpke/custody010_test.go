package hpke

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

func signKey010(m map[string]any, domain string, key ed25519.PrivateKey) []byte {
	m["signature"] = base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, append([]byte(domain), canon010(m)...)))
	return canon010(m)
}

// testCustody010 is an in-process stand-in for external signing custody. It
// exercises the endpoint contract only; it is not a protected custody service.
type testCustody010 struct {
	key       ed25519.PrivateKey
	mode      string
	failAfter int64
	calls     atomic.Int64
	cancel    context.CancelFunc
}

func (c *testCustody010) PublicKey(ctx context.Context) (ed25519.PublicKey, error) {
	switch c.mode {
	case "public-error":
		return nil, errors.New("custody unavailable")
	case "public-panic":
		panic("inert custody failure")
	case "public-short":
		return make([]byte, 31), nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return c.key.Public().(ed25519.PublicKey), nil
}

func (c *testCustody010) Sign(ctx context.Context, message []byte) ([]byte, error) {
	n := c.calls.Add(1)
	if c.failAfter == 0 || n < c.failAfter {
		return ed25519.Sign(c.key, message), nil
	}
	switch c.mode {
	case "sign-error":
		return nil, errors.New("custody refused")
	case "sign-panic":
		panic("inert custody failure")
	case "short-signature":
		return make([]byte, 63), nil
	case "other-key":
		return ed25519.Sign(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{9}, 32)), message), nil
	case "mutated-input":
		message[0] ^= 1
		return ed25519.Sign(c.key, message), nil
	case "cancel-during-sign":
		c.cancel()
		return ed25519.Sign(c.key, message), nil
	}
	return ed25519.Sign(c.key, message), nil
}

func custodyEndpoint010(t *testing.T, c *completionControl, did string, custody Ed25519Custody010, kem []byte) *CompletionEndpoint010 {
	t.Helper()
	j, x := registry010.OpenJournal(filepath.Join(t.TempDir(), "state"), true)
	if x != nil {
		t.Fatal(x)
	}
	t.Cleanup(func() { _ = j.Close() })
	g, x := registry010.NewGate(registry010.Config{Source: "fixture-authority", Registry: completionRegistry, Network: "local"}, c, c, j)
	if x != nil {
		t.Fatal(x)
	}
	e, x := NewCustodyCompletionEndpoint010(context.Background(), did, did+"#signing-1", custody, kem, g, c, &completionReplay{c, map[string]bool{}})
	if x != nil {
		t.Fatal(x)
	}
	t.Cleanup(e.Close)
	return e
}

func custodyKey010(seed byte) ed25519.PrivateKey {
	return ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32))
}

func TestCustodyCompletion010ConstructorRefusals(t *testing.T) {
	c := &completionControl{utc: 100}
	j, x := registry010.OpenJournal(filepath.Join(t.TempDir(), "state"), true)
	if x != nil {
		t.Fatal(x)
	}
	t.Cleanup(func() { _ = j.Close() })
	g, x := registry010.NewGate(registry010.Config{Source: "fixture-authority", Registry: completionRegistry, Network: "local"}, c, c, j)
	if x != nil {
		t.Fatal(x)
	}
	replay := &completionReplay{c, map[string]bool{}}
	for _, mode := range []string{"nil-context", "cancelled", "nil-custody", "typed-nil-custody", "public-error", "public-panic", "public-short", "bad-did", "foreign-kid", "short-kem", "nil-gate", "nil-clock", "nil-replay"} {
		t.Run(mode, func(t *testing.T) {
			ctx := context.Background()
			custody := &testCustody010{key: custodyKey010(1), mode: mode}
			var port Ed25519Custody010 = custody
			did, kid, kem := completionAlice, completionAlice+"#signing-1", []byte(nil)
			gate, clock, store := g, registry010.Clock(c), ReplayStore010(replay)
			switch mode {
			case "nil-context":
				ctx = nil
			case "cancelled":
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			case "nil-custody":
				port = nil
			case "typed-nil-custody":
				var missing *testCustody010
				port = missing
			case "bad-did":
				did = "did:example:alice"
			case "foreign-kid":
				kid = completionBob + "#signing-1"
			case "short-kem":
				kem = make([]byte, 31)
			case "nil-gate":
				gate = nil
			case "nil-clock":
				clock = nil
			case "nil-replay":
				store = nil
			}
			if e, x := NewCustodyCompletionEndpoint010(ctx, did, kid, port, kem, gate, clock, store); x == nil || e != nil {
				t.Fatal("invalid custody endpoint accepted")
			}
			if custody.calls.Load() != 0 {
				t.Fatal("constructor used the signing key")
			}
		})
	}
}

// Both sides hold their signing keys only through custody. The exchange must
// produce the same verified handshake and record behavior as the seed path.
func TestCustodyCompletion010ExchangeAndRecords(t *testing.T) {
	c := &completionControl{utc: 100}
	alice := &testCustody010{key: custodyKey010(1)}
	bob := &testCustody010{key: custodyKey010(2)}
	a := custodyEndpoint010(t, c, completionAlice, alice, nil)
	b := custodyEndpoint010(t, c, completionBob, bob, bytes.Repeat([]byte{3}, 32))
	ctx := context.Background()
	pending, request, x := a.Start(ctx, completionBob, completionBob+"#signing-1", 300)
	if x != nil {
		t.Fatal(x)
	}
	responder, response, x := b.Respond(ctx, request, 300)
	if x != nil {
		t.Fatal(x)
	}
	initiator, x := pending.Complete(ctx, response)
	if x != nil {
		t.Fatal(x)
	}
	t.Cleanup(initiator.Close)
	t.Cleanup(responder.Close)
	if alice.calls.Load() != 1 || bob.calls.Load() != 2 {
		t.Fatalf("handshake custody uses alice=%d bob=%d", alice.calls.Load(), bob.calls.Load())
	}
	sealed, x := initiator.SealRequest(ctx, []byte("2+3"), 300)
	if x != nil {
		t.Fatal(x)
	}
	opened, x := responder.OpenRequest(ctx, sealed)
	if x != nil || !bytes.Equal(opened, []byte("2+3")) {
		t.Fatal("custody-signed request not accepted", x)
	}
	var fields map[string]json.RawMessage
	if x = json.Unmarshal(sealed, &fields); x != nil {
		t.Fatal(x)
	}
	reply, x := responder.SealResponse(ctx, str010(fields, "id"), []byte("5"), true, "", 300)
	if x != nil {
		t.Fatal(x)
	}
	result, x := initiator.OpenResponse(ctx, reply)
	if x != nil || !bytes.Equal(result.Data, []byte("5")) {
		t.Fatal("custody-signed response not accepted", x)
	}
}

// The seed and custody constructors must emit signatures that the same peer
// verifies, so either side can switch custody without a wire change.
func TestCustodyCompletion010InteroperatesWithSeedEndpoint(t *testing.T) {
	seedAlice, _, c := completionPair(t)
	bob := &testCustody010{key: custodyKey010(2)}
	b := custodyEndpoint010(t, c, completionBob, bob, bytes.Repeat([]byte{3}, 32))
	ctx := context.Background()
	pending, request, x := seedAlice.Start(ctx, completionBob, completionBob+"#signing-1", 300)
	if x != nil {
		t.Fatal(x)
	}
	responder, response, x := b.Respond(ctx, request, 300)
	if x != nil {
		t.Fatal(x)
	}
	t.Cleanup(responder.Close)
	initiator, x := pending.Complete(ctx, response)
	if x != nil {
		t.Fatal(x)
	}
	initiator.Close()
}

func TestCustodyCompletion010SigningFailuresPublishNothing(t *testing.T) {
	for _, mode := range []string{"sign-error", "sign-panic", "short-signature", "other-key", "mutated-input", "cancel-during-sign"} {
		t.Run("initiator/"+mode, func(t *testing.T) {
			c := &completionControl{utc: 100}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			alice := &testCustody010{key: custodyKey010(1), mode: mode, failAfter: 1, cancel: cancel}
			a := custodyEndpoint010(t, c, completionAlice, alice, nil)
			if pending, request, x := a.Start(ctx, completionBob, completionBob+"#signing-1", 300); x == nil || pending != nil || request != nil {
				t.Fatal("failed custody signature emitted a request")
			}
		})
		t.Run("responder/"+mode, func(t *testing.T) {
			c := &completionControl{utc: 100}
			alice := &testCustody010{key: custodyKey010(1)}
			a := custodyEndpoint010(t, c, completionAlice, alice, nil)
			pending, request, x := a.Start(context.Background(), completionBob, completionBob+"#signing-1", 300)
			if x != nil {
				t.Fatal(x)
			}
			defer pending.destroy()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			bob := &testCustody010{key: custodyKey010(2), mode: mode, failAfter: 1, cancel: cancel}
			b := custodyEndpoint010(t, c, completionBob, bob, bytes.Repeat([]byte{3}, 32))
			if s, response, x := b.Respond(ctx, request, 300); x == nil || s != nil || response != nil {
				t.Fatal("failed custody signature emitted a completion")
			}
		})
		t.Run("record/"+mode, func(t *testing.T) {
			c := &completionControl{utc: 100}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			alice := &testCustody010{key: custodyKey010(1), mode: mode, failAfter: 2, cancel: cancel}
			a := custodyEndpoint010(t, c, completionAlice, alice, nil)
			b := custodyEndpoint010(t, c, completionBob, &testCustody010{key: custodyKey010(2)}, bytes.Repeat([]byte{3}, 32))
			pending, request, x := a.Start(ctx, completionBob, completionBob+"#signing-1", 300)
			if x != nil {
				t.Fatal(x)
			}
			responder, response, x := b.Respond(context.Background(), request, 300)
			if x != nil {
				t.Fatal(x)
			}
			defer responder.Close()
			initiator, x := pending.Complete(context.Background(), response)
			if x != nil {
				t.Fatal(x)
			}
			defer initiator.Close()
			if sealed, x := initiator.SealRequest(ctx, []byte("2+3"), 300); x == nil || sealed != nil {
				t.Fatal("failed custody signature emitted a record")
			}
			// The record sequence was consumed before signing, so the session
			// must be retired rather than reused with a gap.
			if sealed, x := initiator.SealRequest(context.Background(), []byte("2+3"), 300); x == nil || sealed != nil {
				t.Fatal("session survived a failed record signature")
			}
		})
	}
}

func TestCustodyCompletion010RefusesUnregisteredCustodyAndClose(t *testing.T) {
	c := &completionControl{utc: 100}
	wrong := &testCustody010{key: custodyKey010(7)}
	a := custodyEndpoint010(t, c, completionAlice, wrong, nil)
	if _, request, x := a.Start(context.Background(), completionBob, completionBob+"#signing-1", 300); x == nil || request != nil {
		t.Fatal("unregistered custody key accepted")
	}
	if wrong.calls.Load() != 0 {
		t.Fatal("unregistered custody key was used")
	}
	alice := &testCustody010{key: custodyKey010(1)}
	closed := custodyEndpoint010(t, c, completionAlice, alice, nil)
	closed.Close()
	if _, request, x := closed.Start(context.Background(), completionBob, completionBob+"#signing-1", 300); x == nil || request != nil {
		t.Fatal("closed custody endpoint signed")
	}
	if alice.calls.Load() != 0 {
		t.Fatal("closed endpoint used custody")
	}
}
