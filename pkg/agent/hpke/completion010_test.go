package hpke

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/sage-x-project/sage/pkg/agent/registry010"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const completionRegistry = "web:agent.example"
const completionAlice = "did:sage:web:agent.example:alice"
const completionBob = "did:sage:web:agent.example:bob"

type completionControl struct {
	expiry    int64
	mono, utc int64
	mode      string
}

func (c *completionControl) Now() (registry010.Stamp, error) {
	if c.mode == "clock-error" {
		return registry010.Stamp{}, errors.New("clock")
	}
	return registry010.Stamp{MonoMS: c.mono, Unix: c.utc}, nil
}
func (c *completionControl) Read(_ context.Context, did string) (registry010.Snapshot, error) {
	if c.mode == "source-error" {
		return registry010.Snapshot{}, errors.New("source")
	}
	seed := byte(1)
	if did == completionBob {
		seed = 2
	}
	key := registry010.Key{Name: "signing-1", Alg: "ed25519", Material: hex.EncodeToString(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{seed}, 32)).Public().(ed25519.PublicKey)), State: "accepted"}
	keys := []registry010.Key{}
	if did == completionBob {
		p, _ := ecdh.X25519().NewPrivateKey(bytes.Repeat([]byte{3}, 32))
		kem := registry010.Key{Name: "kem-1", Alg: "x25519", Material: hex.EncodeToString(p.PublicKey().Bytes()), State: "accepted"}
		if c.mode == "revoke-kem" {
			kem.State = "revoked"
		}
		keys = append(keys, kem)
	}
	if (c.mode == "revoke-init" && did == completionAlice) || (c.mode == "revoke-resp" && did == completionBob) {
		key.State = "revoked"
	}
	if c.mode == "changed-material" && did == completionBob {
		key.Material = hex.EncodeToString(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{4}, 32)).Public().(ed25519.PublicKey))
	}
	keys = append(keys, key)
	version := "2"
	if strings.HasPrefix(c.mode, "revoke") || c.mode == "changed-material" || c.mode == "unrelated" {
		version = "3"
	}
	if c.mode == "unrelated" {
		extra := key
		extra.Name = "z-extra"
		extra.Material = hex.EncodeToString(ed25519.NewKeyFromSeed(bytes.Repeat([]byte{5}, 32)).Public().(ed25519.PublicKey))
		keys = append(keys, extra)
	}
	if c.expiry != 0 {
		for i := range keys {
			expiry := c.expiry
			keys[i].Expires = &expiry
		}
	}
	h := sha256.Sum256(canon010(keys))
	return registry010.Snapshot{Source: "fixture-authority", Registry: completionRegistry, Network: "local", DID: did, Version: version, State: "active", Digest: hex.EncodeToString(h[:]), Ready: true, Validated: true, Finalized: true, AcquiredMS: c.mono, Keys: keys}, nil
}

type completionReplay struct {
	c    *completionControl
	seen map[string]bool
}

func (r *completionReplay) Reserve(v Replay010) error {
	if r.c.mode == "store-error" {
		return errors.New("store")
	}
	prefix := v.Sender + "|" + v.Recipient
	ids := []string{prefix + "|id|" + v.ID, prefix + "|nonce|" + v.Nonce}
	if v.Context != "" {
		ids = append(ids, v.Sender+"|ctx|"+v.Context)
	}
	for _, id := range ids {
		if r.seen[id] {
			return errors.New("duplicate")
		}
	}
	for _, id := range ids {
		r.seen[id] = true
	}
	if r.c.mode == "utc-delay" {
		r.c.utc++
		r.c.mono += 1000
	}
	if r.c.mode == "store-delay" {
		r.c.mono += 5001
	}
	return nil
}
func completionPair(t *testing.T) (*CompletionEndpoint010, *CompletionEndpoint010, *completionControl) {
	t.Helper()
	c := &completionControl{utc: 100}
	makeEndpoint := func(did string, n byte) *CompletionEndpoint010 {
		j, x := registry010.OpenJournal(filepath.Join(t.TempDir(), "state"), true)
		if x != nil {
			t.Fatal(x)
		}
		t.Cleanup(func() { _ = j.Close() })
		g, x := registry010.NewGate(registry010.Config{Source: "fixture-authority", Registry: completionRegistry, Network: "local"}, c, c, j)
		if x != nil {
			t.Fatal(x)
		}
		kem := []byte{}
		if n == 2 {
			kem = bytes.Repeat([]byte{3}, 32)
		}
		e, x := NewCompletionEndpoint010(did, did+"#signing-1", bytes.Repeat([]byte{n}, 32), kem, g, c, &completionReplay{c, map[string]bool{}})
		if x != nil {
			t.Fatal(x)
		}
		t.Cleanup(e.Close)
		return e
	}
	return makeEndpoint(completionAlice, 1), makeEndpoint(completionBob, 2), c
}

type completionCase struct {
	ID            string `json:"id"`
	Accept        bool   `json:"accept"`
	Mutation      string `json:"mutation"`
	Mode          string `json:"mode"`
	Mono          *int64 `json:"mono_ms"`
	UTC           *int64 `json:"unix"`
	InitTTL       int64  `json:"init_ttl"`
	ResponseTTL   int64  `json:"response_ttl"`
	Abandon       bool   `json:"abandon"`
	EndpointClose bool   `json:"endpoint_close"`
}

func completionFixture(t *testing.T) []completionCase {
	b, x := os.ReadFile("testdata/completion010.json")
	if x != nil {
		t.Fatal(x)
	}
	var f struct {
		Cases      []completionCase
		PublicKeys map[string]string `json:"public_keys"`
	}
	if json.Unmarshal(b, &f) != nil {
		t.Fatal("fixture")
	}
	for _, n := range []byte{1, 2} {
		pub := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{n}, 32)).Public().(ed25519.PublicKey)
		if hex.EncodeToString(pub) != f.PublicKeys[string('0'+n)] {
			t.Fatal("independent public key")
		}
	}
	return f.Cases
}
func changedCompletion(t *testing.T, request, response []byte, kind string) []byte {
	t.Helper()
	if kind == "" {
		return response
	}
	var w, q map[string]any
	_ = json.Unmarshal(response, &w)
	_ = json.Unmarshal(request, &q)
	body, _ := base64.RawURLEncoding.DecodeString(w["data"].(string))
	var c map[string]any
	_ = json.Unmarshal(body, &c)
	key := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{2}, 32))
	inner := false
	switch kind {
	case "outer-signature":
		w["signature"] = base64.RawURLEncoding.EncodeToString(make([]byte, 64))
		return canon010(w)
	case "inner-signature":
		c["sigB64"] = base64.RawURLEncoding.EncodeToString(make([]byte, 64))
	case "ack":
		c["ackTagB64"] = base64.RawURLEncoding.EncodeToString(make([]byte, 32))
		inner = true
	case "echo":
		c["transcript"].(map[string]any)["ctx"] = "11111111-1111-4111-8111-111111111111"
		inner = true
	case "request-hash":
		w["request_hash"] = base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	case "message-id":
		w["message_id"] = "11111111-1111-4111-8111-111111111111"
	case "recipient":
		w["recipient"] = completionBob
	case "signing-key":
		w["kid"] = completionBob + "#different"
	case "response-nonce":
		w["nonce"] = q["nonce"]
	case "unknown-wire":
		w["extra"] = "value"
	case "unknown-completion":
		c["extra"] = "value"
	case "duplicate-wire":
		return append(append([]byte{}, response[:len(response)-1]...), []byte(",\"version\":\"0.10.0\"}")...)
	case "duplicate-completion":
		body = append(append([]byte{}, body[:len(body)-1]...), []byte(",\"v\":\"0.10.0\"}")...)
	case "duplicate-transcript":
		b := canon010(c["transcript"])
		c["transcript"] = json.RawMessage(append(b[:len(b)-1], []byte(",\"v\":\"0.10.0\"}")...))
		body, _ = json.Marshal(c)
	case "null-transcript":
		c["transcript"] = nil
	case "trailing-wire":
		return append(response, []byte(" {}")...)
	case "noncanonical-completion":
		body = append([]byte(" "), body...)
	default:
		t.Fatal("unknown mutation", kind)
	}
	if inner {
		delete(c, "sigB64")
		c["sigB64"] = base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, append([]byte("sage-hpke-complete|0.10.0\n"), canon010(c)...)))
	}
	if kind != "duplicate-completion" && kind != "duplicate-transcript" && kind != "noncanonical-completion" {
		body = canon010(c)
	}
	w["data"] = base64.RawURLEncoding.EncodeToString(body)
	delete(w, "signature")
	return sign010(w, "sage-wire-response|0.10.0\n", key)
}
func TestCompletion010Scenarios(t *testing.T) {
	cases := completionFixture(t)
	if len(cases) != 36 {
		t.Fatal("missing scenarios")
	}
	for _, q := range cases {
		t.Run(q.ID, func(t *testing.T) {
			a, b, c := completionPair(t)
			ttl := q.InitTTL
			if ttl == 0 {
				ttl = 300
			}
			pending, request, x := a.Start(context.Background(), completionBob, completionBob+"#signing-1", ttl)
			if x != nil {
				t.Fatal(x)
			}
			defer pending.Close()
			ttl = q.ResponseTTL
			if ttl == 0 {
				ttl = 300
			}
			provisional, response, x := b.Respond(context.Background(), request, ttl)
			if x != nil {
				t.Fatal(x)
			}
			defer provisional.Close()
			if provisional.State() != "RESPONSE_SENT" {
				t.Fatal("premature confirmation")
			}
			response = changedCompletion(t, request, response, q.Mutation)
			c.mode = q.Mode
			if q.Mono != nil {
				c.mono = *q.Mono
			}
			if q.UTC != nil {
				c.utc = *q.UTC
			}
			if q.Abandon {
				pending.Close()
			}
			if q.EndpointClose {
				a.Close()
			}
			result, x := pending.Complete(context.Background(), response)
			if (x == nil) != q.Accept {
				t.Fatalf("accept=%v got %v", q.Accept, x)
			}
			if pending.State() != "CLOSED" {
				t.Fatal("pending retained")
			}
			if result != nil {
				defer result.Close()
				if result.State() != "ESTABLISHED" {
					t.Fatal("state")
				}
				for k, v := range provisional.Tuple() {
					if result.Tuple()[k] != v {
						t.Fatal("tuple mismatch", k)
					}
				}
				copy := result.Tuple()
				copy["sid"] = "changed"
				if result.Tuple()["sid"] == "changed" {
					t.Fatal("tuple alias")
				}
			}
			if _, x = pending.Complete(context.Background(), response); x == nil {
				t.Fatal("second completion")
			}
		})
	}
}
func TestCompletion010Lifecycle(t *testing.T) {
	for _, mode := range []string{"revoke-init", "revoke-resp", "revoke-kem", "source-error", "unrelated", "expiry", "closed"} {
		t.Run(mode, func(t *testing.T) {
			a, b, c := completionPair(t)
			p, request, x := a.Start(context.Background(), completionBob, completionBob+"#signing-1", 300)
			if x != nil {
				t.Fatal(x)
			}
			defer p.Close()
			r, response, x := b.Respond(context.Background(), request, 300)
			if x != nil {
				t.Fatal(x)
			}
			defer r.Close()
			if _, _, x = b.Respond(context.Background(), request, 300); x == nil {
				t.Fatal("replayed initiation")
			}
			s, x := p.Complete(context.Background(), response)
			if x != nil {
				t.Fatal(x)
			}
			defer s.Close()
			c.mode = mode
			if mode == "expiry" {
				c.utc = 400
				c.mono = 300000
			}
			if mode == "closed" {
				r.Close()
			}
			x = r.Check(context.Background())
			if mode == "unrelated" {
				if x != nil || r.State() != "RESPONSE_SENT" {
					t.Fatal("unrelated update")
				}
			} else if x == nil || r.State() != "CLOSED" {
				t.Fatal("did not close", mode, x)
			}
			if mode != "expiry" && mode != "closed" {
				x = s.Check(context.Background())
				if (x == nil) != (mode == "unrelated") {
					t.Fatal("established key check", x)
				}
			}
		})
	}
}

func TestCompletion010KeyExpiresDuringCommit(t *testing.T) {
	a, b, c := completionPair(t)
	c.expiry = 101
	p, request, x := a.Start(context.Background(), completionBob, completionBob+"#signing-1", 300)
	if x != nil {
		t.Fatal(x)
	}
	defer p.Close()
	r, response, x := b.Respond(context.Background(), request, 300)
	if x != nil {
		t.Fatal(x)
	}
	defer r.Close()
	c.mode = "utc-delay"
	s, x := p.Complete(context.Background(), response)
	if x == nil || s != nil || p.State() != "CLOSED" {
		t.Fatal("expired signing key authorized completion")
	}
}
