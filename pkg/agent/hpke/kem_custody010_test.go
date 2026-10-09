package hpke

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/registry010"
)

// testKEM010 is an in-process stand-in for external KEM custody. It exercises
// the endpoint contract only; it is not a protected custody service.
type testKEM010 struct {
	key   *ecdh.PrivateKey
	mode  string
	calls atomic.Int64
}

func newTestKEM010(t *testing.T, private []byte) *testKEM010 {
	t.Helper()
	k, err := ecdh.X25519().NewPrivateKey(private)
	if err != nil {
		t.Fatal(err)
	}
	return &testKEM010{key: k}
}

func (k *testKEM010) PublicKey(context.Context) ([]byte, error) {
	if k.mode == "public-error" {
		return nil, errors.New("custody unavailable")
	}
	return k.key.PublicKey().Bytes(), nil
}

func (k *testKEM010) ECDH(ctx context.Context, peer []byte) ([]byte, error) {
	k.calls.Add(1)
	switch k.mode {
	case "error":
		return nil, errors.New("custody refused")
	case "short":
		return make([]byte, 31), nil
	case "zero":
		return make([]byte, 32), nil
	case "other-key":
		other, _ := ecdh.X25519().GenerateKey(rand.Reader)
		p, _ := ecdh.X25519().NewPublicKey(peer)
		return other.ECDH(p)
	case "panic":
		panic("inert custody failure")
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	p, err := ecdh.X25519().NewPublicKey(peer)
	if err != nil {
		return nil, err
	}
	return k.key.ECDH(p)
}

// The custody path reproduces every independent responder derivation vector
// whose KEM private key it holds, and refuses the same invalid inputs.
func TestKEMCustodyMatchesIndependentDerivationVectors(t *testing.T) {
	f := fixture010(t)
	checked := 0
	for _, c := range f.Cases {
		if c.Operation == "domains" {
			continue
		}
		t.Run(c.ID, func(t *testing.T) {
			d := func(k string) []byte { return decode010(t, c.Input[k]) }
			private := d("kem_private_hex")
			k, err := ecdh.X25519().NewPrivateKey(private)
			if err != nil {
				// Invalid private keys are a local-key input condition; custody
				// never exposes one, so these vectors do not apply.
				return
			}
			kem := &testKEM010{key: k}
			v, err := deriveResponderCustody010(context.Background(), d("initiation_hex"), kem, k.PublicKey().Bytes(), d("e2e_private_hex"), c.Input["kid"])
			var out map[string]string
			if err == nil {
				defer zeroBytes(v.Seed)
				out = map[string]string{"transcript_hex": hex.EncodeToString(v.Transcript), "th_hex": hex.EncodeToString(v.TH), "seed_hex": hex.EncodeToString(v.Seed), "ack_tag_hex": hex.EncodeToString(v.AckTag), "sid": v.SID}
			}
			if c.Expected == nil {
				if err == nil {
					t.Fatal("invalid input accepted")
				}
			} else if err != nil || !reflect.DeepEqual(out, c.Expected) {
				t.Fatal("independent vector mismatch", err)
			}
			checked++
		})
	}
	t.Logf("checked %d derivation vectors", checked)
	if checked < 20 {
		t.Fatalf("only %d derivation vectors checked", checked)
	}
}

// Fresh exchanges agree with the initiator and with the local-key path.
func TestKEMCustodyFreshExchangeMatchesLocalKey(t *testing.T) {
	f := fixture010(t)
	b := decode010(t, f.Cases[0].Input["binding_hex"])
	for n := 0; n < 16; n++ {
		local, _ := ecdh.X25519().GenerateKey(rand.Reader)
		kem := &testKEM010{key: local}
		s, init, err := StartInitiator010(b, local.PublicKey().Bytes())
		if err != nil {
			t.Fatal(err)
		}
		e2e, _ := ecdh.X25519().GenerateKey(rand.Reader)
		const kid = "00000000-0000-4000-8000-000000000001"
		custody, err := deriveResponderCustody010(context.Background(), init, kem, local.PublicKey().Bytes(), e2e.Bytes(), kid)
		if err != nil {
			t.Fatal(err)
		}
		direct, err := DeriveResponder010(init, local.Bytes(), e2e.Bytes(), kid)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(custody.Seed, direct.Seed) || !bytes.Equal(custody.Transcript, direct.Transcript) || !bytes.Equal(custody.AckTag, direct.AckTag) {
			t.Fatal("custody and local-key derivations differ")
		}
		l, err := s.Derive(custody.Transcript)
		if err != nil || !bytes.Equal(l.Seed, custody.Seed) {
			t.Fatal("initiator disagrees", err)
		}
		s.Close()
	}
}

func TestKEMCustodyRefusals(t *testing.T) {
	f := fixture010(t)
	b := decode010(t, f.Cases[0].Input["binding_hex"])
	local, _ := ecdh.X25519().GenerateKey(rand.Reader)
	s, init, err := StartInitiator010(b, local.PublicKey().Bytes())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var m map[string]string
	if err = json.Unmarshal(init, &m); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"error", "short", "zero", "other-key", "panic"} {
		t.Run(mode, func(t *testing.T) {
			kem := &testKEM010{key: local, mode: mode}
			d, err := respondFreshCustody010(context.Background(), init, kem, local.PublicKey().Bytes())
			if mode == "other-key" {
				// A wrong shared value derives different secrets; the initiator
				// then refuses the completion.
				if err == nil {
					if l, e := s.Derive(d.Transcript); e == nil && bytes.Equal(l.Seed, d.Seed) {
						t.Fatal("wrong custody key agreed")
					}
				}
				return
			}
			if err == nil || d != nil {
				t.Fatal("custody failure produced a derivation")
			}
		})
	}
	kem := &testKEM010{key: local}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if d, err := respondFreshCustody010(cancelled, init, kem, local.PublicKey().Bytes()); err == nil || d != nil {
		t.Fatal("cancelled custody derivation accepted")
	}
	if d, err := respondFreshCustody010(context.Background(), init, kem, make([]byte, 31)); err == nil || d != nil {
		t.Fatal("short recipient key accepted")
	}
}

func protectedEndpoint010(t *testing.T, c *completionControl, did string, sign Ed25519Custody010, kem X25519Custody010) *CompletionEndpoint010 {
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
	e, x := NewProtectedCompletionEndpoint010(context.Background(), did, did+"#signing-1", sign, kem, g, c, &completionReplay{c, map[string]bool{}})
	if x != nil {
		t.Fatal(x)
	}
	t.Cleanup(e.Close)
	return e
}

// Bob holds neither private key: signatures and the KEM operation come from
// custody, and the handshake plus one record exchange complete.
func TestProtectedEndpointHoldsNoPrivateKeys(t *testing.T) {
	c := &completionControl{utc: 100}
	a := custodyEndpoint010(t, c, completionAlice, &testCustody010{key: custodyKey010(1)}, nil)
	kem := newTestKEM010(t, bytes.Repeat([]byte{3}, 32))
	b := protectedEndpoint010(t, c, completionBob, &testCustody010{key: custodyKey010(2)}, kem)
	if len(b.kem) != 0 || len(b.signing) != 0 {
		t.Fatal("protected endpoint retained private key bytes")
	}
	ctx := context.Background()
	pending, request, err := a.Start(ctx, completionBob, completionBob+"#signing-1", 300)
	if err != nil {
		t.Fatal(err)
	}
	responder, response, err := b.Respond(ctx, request, 300)
	if err != nil {
		t.Fatal(err)
	}
	defer responder.Close()
	initiator, err := pending.Complete(ctx, response)
	if err != nil {
		t.Fatal(err)
	}
	defer initiator.Close()
	if kem.calls.Load() != 1 {
		t.Fatalf("KEM custody used %d times", kem.calls.Load())
	}
	sealed, err := initiator.SealRequest(ctx, []byte("2+3"), 300)
	if err != nil {
		t.Fatal(err)
	}
	if opened, err := responder.OpenRequest(ctx, sealed); err != nil || string(opened) != "2+3" {
		t.Fatal("record not accepted", err)
	}
}

func TestProtectedEndpointKEMRefusals(t *testing.T) {
	for _, mode := range []string{"unregistered", "error", "zero", "other-key", "closed"} {
		t.Run(mode, func(t *testing.T) {
			c := &completionControl{utc: 100}
			a := custodyEndpoint010(t, c, completionAlice, &testCustody010{key: custodyKey010(1)}, nil)
			private := bytes.Repeat([]byte{3}, 32)
			if mode == "unregistered" {
				private = bytes.Repeat([]byte{4}, 32)
			}
			kem := newTestKEM010(t, private)
			if mode != "unregistered" && mode != "closed" {
				kem.mode = mode
			}
			b := protectedEndpoint010(t, c, completionBob, &testCustody010{key: custodyKey010(2)}, kem)
			if mode == "closed" {
				b.Close()
			}
			pending, request, err := a.Start(context.Background(), completionBob, completionBob+"#signing-1", 300)
			if err != nil {
				t.Fatal(err)
			}
			defer pending.destroy()
			s, response, err := b.Respond(context.Background(), request, 300)
			if mode == "other-key" && err == nil {
				// A wrong shared value yields a completion the initiator refuses.
				defer s.Close()
				if i, e := pending.Complete(context.Background(), response); e == nil {
					i.Close()
					t.Fatal("initiator accepted a completion from the wrong KEM key")
				}
				return
			}
			if err == nil || s != nil || response != nil {
				t.Fatal("KEM custody failure produced a completion")
			}
			if (mode == "unregistered" || mode == "closed") && kem.calls.Load() != 0 {
				t.Fatal("unregistered or closed KEM custody was used")
			}
		})
	}
	c := &completionControl{utc: 100}
	kem := newTestKEM010(t, bytes.Repeat([]byte{3}, 32))
	kem.mode = "public-error"
	j, _ := registry010.OpenJournal(filepath.Join(t.TempDir(), "state"), true)
	defer func() { _ = j.Close() }()
	g, _ := registry010.NewGate(registry010.Config{Source: "fixture-authority", Registry: completionRegistry, Network: "local"}, c, c, j)
	if e, err := NewProtectedCompletionEndpoint010(context.Background(), completionBob, completionBob+"#signing-1", &testCustody010{key: custodyKey010(2)}, kem, g, c, &completionReplay{c, map[string]bool{}}); err == nil || e != nil {
		t.Fatal("unavailable KEM custody accepted")
	}
}
