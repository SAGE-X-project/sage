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

func responseMutate010(t *testing.T, raw, request []byte, kind, otherID string, s *AuthenticatedCompletion010) []byte {
	t.Helper()
	var w, q map[string]any
	if json.Unmarshal(raw, &w) != nil || json.Unmarshal(request, &q) != nil {
		t.Fatal("JSON")
	}
	if kind == "duplicate-field" {
		return append(append([]byte{}, raw[:len(raw)-1]...), []byte(`,"success":true}`)...)
	}
	switch kind {
	case "request-hash":
		w["request_hash"] = base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	case "message-id":
		w["message_id"] = "11111111-1111-4111-8111-111111111111"
	case "cross-request":
		w["message_id"] = otherID
	case "same-id":
		w["id"] = q["id"]
	case "same-nonce":
		w["nonce"] = q["nonce"]
	case "recipient":
		w["recipient"] = completionBob
	case "did":
		w["did"] = completionAlice
	case "kid":
		w["kid"] = completionBob + "#different"
	case "role":
		w["role"] = "initiator"
	case "context_id":
		w[kind] = "11111111-1111-4111-8111-111111111111"
	case "session_id":
		w[kind] = base64.RawURLEncoding.EncodeToString(make([]byte, 16))
	case "signature":
		w[kind] = base64.RawURLEncoding.EncodeToString(make([]byte, 64))
		return canon010(w)
	case "tag":
		v, _ := base64.RawURLEncoding.DecodeString(w["data"].(string))
		v[len(v)-1] ^= 1
		w["data"] = base64.RawURLEncoding.EncodeToString(v)
	case "missing-error":
		w["success"] = false
		delete(w, "error")
	case "unknown-error":
		w["success"] = false
		w["error"] = "other"
	case "success-error":
		w["success"] = true
		w["error"] = "policy_denied"
	case "success-type":
		w["success"] = "true"
	case "unknown-field":
		w["extra"] = "value"
	case "plain-encoding":
		w["encoding"] = "plain"
	default:
		return raw
	}
	delete(w, "signature")
	return sign010(w, "sage-wire-response|0.10.0\n", s.endpoint.signing)
}
func TestSessionResponse010(t *testing.T) {
	raw, x := os.ReadFile("testdata/session-response010.json")
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
			q, x := a.SealRequest(ctx, []byte("request"), 300)
			if x != nil {
				t.Fatal(x)
			}
			if _, x = b.OpenRequest(ctx, q); x != nil {
				t.Fatal(x)
			}
			if kind == "reverse" {
				a, b = b, a
				q, x = a.SealRequest(ctx, []byte("reverse"), 300)
				if x != nil {
					t.Fatal(x)
				}
				if _, x = b.OpenRequest(ctx, q); x != nil {
					t.Fatal(x)
				}
			}
			var request map[string]json.RawMessage
			_ = json.Unmarshal(q, &request)
			id := str010(request, "id")
			code := ""
			if responseError010(kind) {
				code = kind
			}
			data := []byte("result")
			if kind == "empty-data" {
				data = nil
			}
			r, x := b.SealResponse(ctx, id, data, code == "", code, 300)
			if x != nil {
				t.Fatal(x)
			}
			if _, x = b.SealResponse(ctx, id, data, code == "", code, 300); x == nil {
				t.Fatal("second response emitted")
			}
			otherID := ""
			var otherResponse []byte
			if kind == "cross-request" || kind == "out-of-order" {
				other, x := a.SealRequest(ctx, []byte("other"), 300)
				if x != nil {
					t.Fatal(x)
				}
				if _, x = b.OpenRequest(ctx, other); x != nil {
					t.Fatal(x)
				}
				var m map[string]json.RawMessage
				_ = json.Unmarshal(other, &m)
				otherID = str010(m, "id")
				otherResponse, x = b.SealResponse(ctx, otherID, []byte("other-result"), true, "", 300)
				if x != nil {
					t.Fatal(x)
				}
			}
			good := append([]byte{}, r...)
			r = responseMutate010(t, r, q, kind, otherID, b)
			accept := kind == "valid" || kind == "empty-data" || responseError010(kind) || kind == "reverse" || kind == "out-of-order"
			closed := false
			switch kind {
			case "duplicate-terminal":
				if _, x = a.OpenResponse(ctx, r); x != nil {
					t.Fatal(x)
				}
			case "out-of-order":
				if v, x := a.OpenResponse(ctx, otherResponse); x != nil || string(v.Data) != "other-result" {
					t.Fatal("reordered", x)
				}
			case "store-error":
				c.mode = kind
			case "store-delay", "revoke-init", "revoke-resp", "revoke-kem", "source-error":
				c.mode = kind
				closed = true
			case "key-expiry-during-commit":
				c.mode = "utc-delay"
				closed = true
			case "closed":
				a.Close()
				closed = true
			case "expired":
				c.utc = 400
			}
			journal := a.endpoint.replay.(*completionReplay)
			before := len(journal.seen)
			result, x := a.OpenResponse(ctx, r)
			if (x == nil) != accept {
				t.Fatal("accept", accept, x)
			}
			if accept {
				if result.MessageID != id || result.Success != (code == "") || result.Error != code || !bytes.Equal(result.Data, data) {
					t.Fatal("incorrect result")
				}
				if _, x = a.OpenResponse(ctx, r); x == nil {
					t.Fatal("second acceptance")
				}
			} else {
				if result != nil || len(journal.seen) != before {
					t.Fatal("partial acceptance")
				}
				if closed {
					if a.State() != "CLOSED" {
						t.Fatal("not closed")
					}
				} else if kind != "duplicate-terminal" && kind != "expired" {
					c.mode = ""
					v, x := a.OpenResponse(ctx, good)
					if x != nil || !bytes.Equal(v.Data, data) {
						t.Fatal("invalid response consumed pending/sequence", x)
					}
				}
			}
		})
	}
}
func TestSessionResponse010ConcurrentTerminal(t *testing.T) {
	a, b, _ := recordPair010(t, 0)
	ctx := context.Background()
	q, _ := a.SealRequest(ctx, nil, 300)
	_, _ = b.OpenRequest(ctx, q)
	var m map[string]json.RawMessage
	_ = json.Unmarshal(q, &m)
	r, x := b.SealResponse(ctx, str010(m, "id"), nil, true, "", 300)
	if x != nil {
		t.Fatal(x)
	}
	var wg sync.WaitGroup
	out := make(chan bool, 16)
	for n := 0; n < 16; n++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, x := a.OpenResponse(ctx, r); out <- x == nil }()
	}
	wg.Wait()
	close(out)
	count := 0
	for v := range out {
		if v {
			count++
		}
	}
	if count != 1 {
		t.Fatal("terminal count", count)
	}
}

func TestSessionResponse010RetainedCopyAndAdmission(t *testing.T) {
	a, b, _ := recordPair010(t, 0)
	ctx := context.Background()
	q, x := a.SealRequest(ctx, []byte("request"), 300)
	if x != nil {
		t.Fatal(x)
	}
	if _, x = b.SealResponse(ctx, "11111111-1111-4111-8111-111111111111", nil, true, "", 300); x == nil {
		t.Fatal("unsolicited emission")
	}
	_, x = b.OpenRequest(ctx, q)
	if x != nil {
		t.Fatal(x)
	}
	var m map[string]json.RawMessage
	_ = json.Unmarshal(q, &m)
	id := str010(m, "id")
	clear(q) // Both owners must retain private copies of the original signed envelope.
	for _, bad := range []struct {
		success bool
		code    string
	}{{true, "policy_denied"}, {false, ""}, {false, "unknown"}} {
		if _, x = b.SealResponse(ctx, id, nil, bad.success, bad.code, 300); x == nil {
			t.Fatal("invalid response status")
		}
	}
	if _, x = b.SealResponse(ctx, id, make([]byte, 16349), true, "", 300); x == nil {
		t.Fatal("size")
	}
	r, x := b.SealResponse(ctx, id, make([]byte, 16348), true, "", 300)
	if x != nil {
		t.Fatal(x)
	}
	v, x := a.OpenResponse(ctx, r)
	if x != nil || len(v.Data) != 16348 {
		t.Fatal("retained original/max boundary", x)
	}
}
