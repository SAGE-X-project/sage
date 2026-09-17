package hpke

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"sync"
	"testing"
)

func recordPair010(t *testing.T, expiry int64) (*AuthenticatedCompletion010, *AuthenticatedCompletion010, *completionControl) {
	t.Helper()
	a, b, c := completionPair(t)
	c.expiry = expiry
	p, q, x := a.Start(context.Background(), completionBob, completionBob+"#signing-1", 300)
	if x != nil {
		t.Fatal(x)
	}
	s, r, x := b.Respond(context.Background(), q, 300)
	if x != nil {
		t.Fatal(x)
	}
	i, x := p.Complete(context.Background(), r)
	if x != nil {
		t.Fatal(x)
	}
	t.Cleanup(i.Close)
	t.Cleanup(s.Close)
	return i, s, c
}
func mutateRecord010(t *testing.T, wire []byte, kind string, s *AuthenticatedCompletion010) []byte {
	t.Helper()
	if kind == "duplicate-field" {
		return append(append([]byte{}, wire[:len(wire)-1]...), []byte(`,"version":"0.10.0"}`)...)
	}
	var w map[string]any
	if json.Unmarshal(wire, &w) != nil {
		t.Fatal("JSON")
	}
	switch kind {
	case "signature":
		w["signature"] = base64.RawURLEncoding.EncodeToString(make([]byte, 64))
		return canon010(w)
	case "tag":
		v, _ := base64.RawURLEncoding.DecodeString(w["payload"].(string))
		v[len(v)-1] ^= 1
		w["payload"] = base64.RawURLEncoding.EncodeToString(v)
	case "did":
		w[kind] = completionBob
	case "recipient":
		w[kind] = completionAlice
	case "kid":
		w[kind] = completionAlice + "#different"
	case "role":
		w[kind] = "responder"
	case "context_id":
		w[kind] = "11111111-1111-4111-8111-111111111111"
	case "session_id":
		w[kind] = base64.RawURLEncoding.EncodeToString(make([]byte, 16))
	case "version":
		w[kind] = "0.9.0"
	case "unknown-field":
		w["extra"] = "value"
	default:
		return wire
	}
	delete(w, "signature")
	return sign010(w, "sage-wire-request|0.10.0\n", s.endpoint.signing)
}
func TestAuthenticatedRecord010Scenarios(t *testing.T) {
	raw, x := os.ReadFile("testdata/authenticated-record010.json")
	if x != nil {
		t.Fatal(x)
	}
	var f struct {
		Cases []string `json:"cases"`
	}
	if json.Unmarshal(raw, &f) != nil {
		t.Fatal("fixture")
	}
	for _, kind := range f.Cases {
		t.Run(kind, func(t *testing.T) {
			expiry := int64(0)
			if kind == "key-expiry-during-commit" {
				expiry = 101
			}
			a, b, c := recordPair010(t, expiry)
			ctx := context.Background()
			if _, x := b.SealRequest(ctx, []byte("denied"), 300); x == nil || b.State() != "RESPONSE_SENT" {
				t.Fatal("provisional sent")
			}
			w, x := a.SealRequest(ctx, []byte("first"), 300)
			if x != nil {
				t.Fatal(x)
			}
			first := w
			want := []byte("first")
			if kind == "nonzero-first" || kind == "out-of-order" {
				w, x = a.SealRequest(ctx, []byte("second"), 300)
				if x != nil {
					t.Fatal(x)
				}
				want = []byte("second")
			}
			good := append([]byte{}, w...)
			w = mutateRecord010(t, w, kind, a)
			accept := false
			closed := false
			switch kind {
			case "valid", "nonzero-first", "out-of-order", "application-reject", "unrelated":
				accept = true
				if kind == "unrelated" {
					c.mode = "unrelated"
				}
			case "duplicate", "idle-expiry":
				if _, x = b.OpenRequest(ctx, w); x != nil {
					t.Fatal(x)
				}
				if kind == "idle-expiry" {
					c.mono = 600000
					closed = true
				}
			case "store-error":
				c.mode = kind
			case "store-delay":
				c.mode = kind
				closed = true
			case "pending-mono-expiry":
				c.mono = 300000
				closed = true
			case "pending-utc-expiry":
				c.utc = 400
				closed = true
			case "key-expiry-during-commit":
				c.mode = "utc-delay"
				closed = true
			case "revoke-init", "revoke-resp", "revoke-kem", "changed-material", "source-error", "clock-error":
				c.mode = kind
				closed = true
			case "closed":
				b.Close()
				closed = true
			case "transport-id", "transport-nonce":
				var m map[string]json.RawMessage
				_ = json.Unmarshal(w, &m)
				entry := reservation010(m)
				key := "id"
				value := entry.ID
				if kind == "transport-nonce" {
					key = "nonce"
					value = entry.Nonce
				}
				b.endpoint.replay.(*completionReplay).seen[entry.Sender+"|"+entry.Recipient+"|"+key+"|"+value] = true
			}
			replay := b.endpoint.replay.(*completionReplay)
			before := len(replay.seen)
			p, x := b.OpenRequest(ctx, w)
			if (x == nil) != accept {
				t.Fatalf("accept=%v err=%v", accept, x)
			}
			if accept {
				if !bytes.Equal(p, want) || b.State() != "ESTABLISHED" {
					t.Fatal("accept result")
				}
				if len(replay.seen) != before+2 {
					t.Fatal("missing atomic transport entries")
				}
				if _, x = b.OpenRequest(ctx, w); x == nil {
					t.Fatal("duplicate")
				}
				if kind == "out-of-order" {
					p, x = b.OpenRequest(ctx, first)
					if x != nil || string(p) != "first" {
						t.Fatal("unseen earlier record")
					}
				}
				// Application rejection is a caller decision after this boundary: no rollback.
				r, x := b.SealRequest(ctx, []byte("rejection"), 300)
				if x != nil {
					t.Fatal(x)
				}
				p, x = a.OpenRequest(ctx, r)
				if x != nil || string(p) != "rejection" {
					t.Fatal("reverse direction", x)
				}
			} else {
				if len(p) != 0 || len(replay.seen) != before {
					t.Fatal("partial acceptance")
				}
				if closed {
					if b.State() != "CLOSED" {
						t.Fatal("not closed")
					}
				} else if kind != "duplicate" && kind != "transport-id" && kind != "transport-nonce" {
					if b.State() != "RESPONSE_SENT" {
						t.Fatal("confirmed invalid")
					}
					c.mode = ""
					p, x = b.OpenRequest(ctx, good)
					if x != nil || !bytes.Equal(p, want) {
						t.Fatal("sequence consumed after failure", x)
					}
				}
			}
		})
	}
}
func TestAuthenticatedRecord010ConcurrentCopies(t *testing.T) {
	a, b, _ := recordPair010(t, 0)
	ctx := context.Background()
	w, x := a.SealRequest(ctx, []byte("one"), 300)
	if x != nil {
		t.Fatal(x)
	}
	var wg sync.WaitGroup
	results := make(chan bool, 16)
	for n := 0; n < 16; n++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, x := b.OpenRequest(ctx, w); results <- x == nil }()
	}
	wg.Wait()
	close(results)
	accepted := 0
	for ok := range results {
		if ok {
			accepted++
		}
	}
	if accepted != 1 || b.State() != "ESTABLISHED" {
		t.Fatal("non-atomic copies", accepted)
	}
}

func TestAuthenticatedRecord010LifetimeAndBounds(t *testing.T) {
	ctx := context.Background()
	t.Run("absolute includes provisional time", func(t *testing.T) {
		a, b, c := recordPair010(t, 0)
		c.mono = 200000
		w, x := a.SealRequest(ctx, []byte("first"), 300)
		if x != nil {
			t.Fatal(x)
		}
		if _, x = b.OpenRequest(ctx, w); x != nil {
			t.Fatal(x)
		}
		for now := int64(500000); now <= 3500000; now += 500000 {
			c.mono = now
			if _, x = b.SealRequest(ctx, []byte("traffic"), 300); x != nil {
				t.Fatal(x)
			}
		}
		c.mono = 3600000
		if _, x = b.SealRequest(ctx, nil, 300); x == nil || b.State() != "CLOSED" {
			t.Fatal("confirmation reset lifetime")
		}
	})
	t.Run("invalid input cannot reset idle", func(t *testing.T) {
		a, b, c := recordPair010(t, 0)
		w, x := a.SealRequest(ctx, nil, 300)
		if x != nil {
			t.Fatal(x)
		}
		if _, x = b.OpenRequest(ctx, w); x != nil {
			t.Fatal(x)
		}
		c.mono = 599999
		if _, x = b.OpenRequest(ctx, mutateRecord010(t, w, "signature", a)); x == nil {
			t.Fatal("bad signature")
		}
		c.mono = 600000
		if b.Check(ctx) == nil || b.State() != "CLOSED" {
			t.Fatal("invalid traffic renewed idle")
		}
	})
	t.Run("bounded admission does not consume send sequence", func(t *testing.T) {
		a, b, _ := recordPair010(t, 0)
		if _, x := a.SealRequest(ctx, make([]byte, 16349), 300); x == nil {
			t.Fatal("oversize")
		}
		w, x := a.SealRequest(ctx, make([]byte, 16348), 300)
		if x != nil {
			t.Fatal(x)
		}
		p, x := b.OpenRequest(ctx, w)
		if x != nil || len(p) != 16348 {
			t.Fatal("maximum", x)
		}
	})
}
