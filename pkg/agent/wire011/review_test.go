package wire011

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// signedRequest builds a request whose HTTP signature is valid for env. With
// innerValid=false it keeps a stale inner signature, so only the envelope
// signature check can refuse it.
func signedRequest(t *testing.T, f *fixture, env Request, innerValid bool) HTTPRequest {
	t.Helper()
	if innerValid {
		sig, err := signEnvelope(requestDomain, env.unsigned(), f.client.Key)
		if err != nil {
			t.Fatal(err)
		}
		env.Signature = sig
	}
	body, _ := json.Marshal(env)
	target, authority, _ := endpoint(serverURL)
	h := map[string]string{"content-type": "application/json", "content-digest": contentDigest(body),
		"x-sage-did": env.DID, "x-sage-version": Version}
	input := signatureInput(requestComponents, env.KID, env.Created, env.Expires, env.Nonce)
	h["signature-input"] = "sig1=" + input
	h["signature"] = "sig1=:" + base64Std(ed25519.Sign(f.client.Key, requestBase("POST", target, authority, h, input))) + ":"
	return HTTPRequest{Method: "POST", TargetURI: target, Authority: authority, Fields: fieldsOf(h), Body: body}
}

func envelopeOf(t *testing.T, r *HTTPRequest) Request {
	t.Helper()
	var env Request
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatal(err)
	}
	return env
}

func TestInnerSignatureIsChecked(t *testing.T) {
	f := newFixture(t)
	req, _ := f.request(t)
	env := envelopeOf(t, req)
	env.Payload = b64([]byte(`{"tool":"pay","amount":"9000000"}`))
	f.expectRefusal(t, signedRequest(t, f, env, false), "envelope signature")
}

func TestSameIDWithNewNonceIsReplay(t *testing.T) {
	f := newFixture(t)
	req, _ := f.request(t)
	if _, err := f.receiver.VerifyRequest(context.Background(), *req); err != nil {
		t.Fatal(f.trace.lastReason())
	}
	env := envelopeOf(t, req)
	env.Nonce = "BBBBBBBBBBBBBBBBBBBBBA"
	f.expectRefusal(t, signedRequest(t, f, env, true), "replay")
}

func TestConcurrentCopiesAcceptOnce(t *testing.T) {
	f := newFixture(t)
	req, _ := f.request(t)
	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.receiver.VerifyRequest(context.Background(), *req); err == nil {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("%d copies accepted", accepted.Load())
	}
}

func TestParameterRefusalReasons(t *testing.T) {
	cases := map[string]struct{ old, new, reason string }{
		"escaped keyid": {"#sig-3", `#sig\-3`, "escaped parameter"},
		"bad created":   {";created=", ";created=0", "parameter created"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			req, _ := f.request(t)
			req.Fields = setField(req.Fields, "signature-input", strings.Replace(fieldValue(req.Fields, "signature-input"), tc.old, tc.new, 1))
			f.expectRefusal(t, *req, tc.reason)
		})
	}
}

func TestEmptyProjectionHeaderIsMismatch(t *testing.T) {
	f := newFixture(t)
	req, _ := f.request(t)
	req.Fields = append(req.Fields, Field{"X-Sage-Task-Id", ""})
	f.expectRefusal(t, *req, "x-sage-task-id")
}

func TestStrictBodyMembers(t *testing.T) {
	f := newFixture(t)
	req, _ := f.request(t)
	base := string(req.Body)
	var out Request
	cases := map[string]string{
		"empty optional context": strings.Replace(base, `"encoding":"plain"`, `"encoding":"plain","task_id":""`, 1),
		"empty metadata":         strings.Replace(base, `"encoding":"plain"`, `"encoding":"plain","metadata":{}`, 1),
		"duplicate metadata key": strings.Replace(base, `"encoding":"plain"`, `"encoding":"plain","metadata":{"a":"x","a":"y"}`, 1),
		"null metadata value":    strings.Replace(base, `"encoding":"plain"`, `"encoding":"plain","metadata":{"a":null}`, 1),
		"number metadata value":  strings.Replace(base, `"encoding":"plain"`, `"encoding":"plain","metadata":{"a":1}`, 1),
		"invalid UTF-8":          strings.Replace(base, `"encoding":"plain"`, "\"encoding\":\"plain\",\"metadata\":{\"a\":\"\xff\"}", 1),
		"plain with session_id":  strings.Replace(base, `"encoding":"plain"`, `"encoding":"plain","session_id":"AAAAAAAAAAAAAAAAAAAAAA"`, 1),
	}
	for name, body := range cases {
		if decodeClosed([]byte(body), requestMembers, &out) == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	if err := decodeClosed([]byte(strings.Replace(base, `"encoding":"plain"`, `"encoding":"plain","metadata":{"a":"x"}`, 1)), requestMembers, &out); err != nil {
		t.Fatalf("valid metadata refused: %v", err)
	}
}

func TestTimingEdges(t *testing.T) {
	clock := &fixedClock{t: time.Unix(10_000, 0)}
	cases := []struct {
		created, expires int64
		ok               bool
	}{
		{10_030, 10_090, true},  // created == now+30
		{10_031, 10_090, false}, // created == now+31
		{9_800, 9_971, true},    // now == expires+29
		{9_800, 9_970, false},   // now == expires+30
	}
	for _, c := range cases {
		if got := checkTime(clock, c.created, c.expires) == ""; got != c.ok {
			t.Fatalf("created=%d expires=%d: accepted=%v", c.created, c.expires, got)
		}
	}
}

func TestResponseBindingRefusals(t *testing.T) {
	cases := map[string]struct {
		reason string
		mutate func(*Response)
	}{
		"request hash":       {"does not bind this request", func(r *Response) { r.RequestHash = b64(make([]byte, 32)) }},
		"response recipient": {"not from the request recipient", func(r *Response) { r.Recipient = serverDID + "-other" }},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			req, sent := f.request(t)
			v, err := f.receiver.VerifyRequest(context.Background(), *req)
			if err != nil {
				t.Fatal(f.trace.lastReason())
			}
			ok, _ := f.server.SignResponse(v, 200, true, nil, "")
			var env Response
			_ = json.Unmarshal(ok.Body, &env)
			tc.mutate(&env)
			if _, err := f.responses.VerifyResponse(context.Background(), sent, *resign(t, f, v, env, 200)); err == nil {
				t.Fatal("accepted")
			}
			if got := f.trace.lastReason(); !strings.Contains(got, tc.reason) {
				t.Fatalf("refused for %q, want %q", got, tc.reason)
			}
		})
	}
}

func TestReplayQuarantineAndRollback(t *testing.T) {
	clock := &fixedClock{t: time.Unix(1_000, 0)}
	m := &MemoryReplay{Capacity: 8, Clock: clock, Started: clock.t}
	if m.Reserve(context.Background(), "a", "b", "n", "i", clock.t.Add(time.Minute)) == nil {
		t.Fatal("reservation accepted during the restart quarantine")
	}
	clock.t = clock.t.Add(Quarantine)
	if err := m.Reserve(context.Background(), "a", "b", "n", "i", clock.t.Add(time.Minute)); err != nil {
		t.Fatalf("refused after quarantine: %v", err)
	}
	clock.t = clock.t.Add(-time.Second)
	if m.Reserve(context.Background(), "a", "b", "n2", "i2", clock.t.Add(time.Minute)) == nil {
		t.Fatal("reservation accepted after the clock moved backwards")
	}
}

func TestNonCanonicalEndpoints(t *testing.T) {
	for _, target := range []string{
		"https://a.example./pay", "https://a.example:/pay", "https://a.example:0443/pay", "https://a.example:443/pay",
		"https://a.example/pay?", "https://a.example/a/../pay", "https://a.example/%7epay", "https://a.example/p y",
		"https://A.example/pay", "https://a.example", "http://a.example/pay", "https://a.example/pay#x",
	} {
		if _, _, err := endpoint(target); err == nil {
			t.Fatalf("%q accepted", target)
		}
	}
	if _, authority, err := endpoint("https://a.example:8443/pay"); err != nil || authority != "a.example:8443" {
		t.Fatalf("canonical endpoint with port refused: %v", err)
	}
}

func TestSignerGuardsAndMetadataCopy(t *testing.T) {
	var nilSigner *Signer
	if _, err := nilSigner.SignResponse(&VerifiedRequest{}, 200, true, nil, ""); err == nil {
		t.Fatal("nil signer signed")
	}
	f := newFixture(t)
	meta := map[string]string{"purpose": "pay"}
	_, sent, err := f.client.SignRequest(serverDID, serverURL, nil, RequestOptions{Metadata: meta})
	if err != nil {
		t.Fatal(err)
	}
	meta["purpose"] = "changed"
	if sent.Envelope().Metadata["purpose"] != "pay" {
		t.Fatal("signed envelope aliases the caller's metadata")
	}
}
