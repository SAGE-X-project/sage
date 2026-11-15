package wire011

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestValidCommonRefusals(t *testing.T) {
	base := func() Request {
		return Request{Version: Version, ID: "3b1b2a8e-2f5c-4c4e-9c1a-8d3f6e7a9b10", DID: clientDID, Recipient: serverDID,
			KID: clientDID + "#sig-3", Created: 100, Expires: 200, Nonce: "AAAAAAAAAAAAAAAAAAAAAA", Encoding: "plain"}
	}
	check := func(r Request) error {
		return validCommon(r.Version, r.ID, r.DID, r.Recipient, r.KID, r.Created, r.Expires, r.Nonce, r.Encoding,
			r.Signature, r.ContextID, r.TaskID, r.Metadata)
	}
	if err := check(base()); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*Request){
		"version":        func(r *Request) { r.Version = "0.10.0" },
		"uuid version":   func(r *Request) { r.ID = "3b1b2a8e-2f5c-1c4e-9c1a-8d3f6e7a9b10" },
		"uppercase uuid": func(r *Request) { r.ID = strings.ToUpper(r.ID) },
		"bare did":       func(r *Request) { r.DID = "sage:x" },
		"self addressed": func(r *Request) { r.Recipient = r.DID },
		"foreign kid":    func(r *Request) { r.KID = serverDID + "#sig-1" },
		"bare kid":       func(r *Request) { r.KID = clientDID },
		"long lifetime":  func(r *Request) { r.Expires = r.Created + 301 },
		"zero lifetime":  func(r *Request) { r.Expires = r.Created },
		"short nonce":    func(r *Request) { r.Nonce = "AAAA" },
		"padded nonce":   func(r *Request) { r.Nonce = "AAAAAAAAAAAAAAAAAAAAAA==" },
		"session":        func(r *Request) { r.Encoding = "session" },
		"bad context":    func(r *Request) { r.ContextID = "not-a-uuid" },
		"bad signature":  func(r *Request) { r.Signature = "AAAA" },
		"metadata key":   func(r *Request) { r.Metadata = map[string]string{"Role": "x"} },
		"metadata value": func(r *Request) { r.Metadata = map[string]string{"role": strings.Repeat("x", 1025)} },
		"metadata count": func(r *Request) {
			r.Metadata = map[string]string{}
			for i := 0; i < 33; i++ {
				r.Metadata["k"+string(rune('a'+i%26))+string(rune('a'+i/26))] = "v"
			}
		},
		"metadata size": func(r *Request) {
			r.Metadata = map[string]string{}
			for i := 0; i < 8; i++ {
				r.Metadata["k"+string(rune('a'+i))] = strings.Repeat("v", 1000)
			}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			r := base()
			mutate(&r)
			if check(r) == nil {
				t.Fatal("accepted")
			}
		})
	}
}

func TestDecodeClosedRefusals(t *testing.T) {
	allowed := memberSet("a b")
	var out map[string]any
	for name, raw := range map[string]string{
		"array": `[1]`, "unknown": `{"c":1}`, "duplicate": `{"a":1,"a":2}`, "null": `{"a":null}`,
		"trailing": `{"a":1} {"b":2}`, "broken": `{"a":`,
	} {
		if decodeClosed([]byte(raw), allowed, &out) == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	if decodeClosed([]byte(`{"a":1,"b":"x"}`), allowed, &out) != nil {
		t.Fatal("valid object refused")
	}
}

func TestDecodeB64Refusals(t *testing.T) {
	for _, s := range []string{"AA==", "A", "AB", "+/"} {
		if _, err := decodeB64(s, -1); err == nil {
			t.Fatalf("%q accepted", s)
		}
	}
	if _, err := decodeB64("AAAA", 2); err == nil {
		t.Fatal("wrong size accepted")
	}
}

func TestFieldRefusals(t *testing.T) {
	f := newFixture(t)
	req, _ := f.request(t)
	cases := map[string]func(*HTTPRequest){
		"content-length":  func(r *HTTPRequest) { r.Fields = append(r.Fields, Field{"Content-Length", "1"}) },
		"host":            func(r *HTTPRequest) { r.Fields = append(r.Fields, Field{"Host", "evil.example.invalid"}) },
		"control byte":    func(r *HTTPRequest) { r.Fields = append(r.Fields, Field{"X-Note", "a\x01"}) },
		"padded value":    func(r *HTTPRequest) { r.Fields = append(r.Fields, Field{"X-Note", " a"}) },
		"bad name":        func(r *HTTPRequest) { r.Fields = append(r.Fields, Field{"X Note", "a"}) },
		"too many fields": func(r *HTTPRequest) { r.Fields = append(r.Fields, make([]Field, 300)...) },
		"signature size":  func(r *HTTPRequest) { r.Fields = setField(r.Fields, "signature", strings.Repeat("a", 9000)) },
		"content-type":    func(r *HTTPRequest) { r.Fields = setField(r.Fields, "content-type", "application/json; charset=utf-8") },
		"missing did":     func(r *HTTPRequest) { r.Fields = setField(r.Fields, "x-sage-did", "") },
		"message-id":      func(r *HTTPRequest) { r.Fields = append(r.Fields, Field{"X-Sage-Message-Id", "other"}) },
		"escaped keyid": func(r *HTTPRequest) {
			r.Fields = setField(r.Fields, "signature-input", strings.Replace(fieldValue(r.Fields, "signature-input"), "#sig-3", `#sig\-3`, 1))
		},
		"bad created": func(r *HTTPRequest) {
			r.Fields = setField(r.Fields, "signature-input", strings.Replace(fieldValue(r.Fields, "signature-input"), ";created=", ";created=0", 1))
		},
		"missing tag": func(r *HTTPRequest) {
			r.Fields = setField(r.Fields, "signature-input", strings.Replace(fieldValue(r.Fields, "signature-input"), `;tag="`+Tag+`"`, "", 1))
		},
		"wrong label": func(r *HTTPRequest) {
			r.Fields = setField(r.Fields, "signature-input", "sig2"+fieldValue(r.Fields, "signature-input")[4:])
		},
		"signature member": func(r *HTTPRequest) {
			r.Fields = setField(r.Fields, "signature", "sig2"+fieldValue(r.Fields, "signature")[4:])
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			r := *req
			mutate(&r)
			if _, err := f.receiver.VerifyRequest(context.Background(), r); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}

func TestResponseFieldAndEnvelopeRefusals(t *testing.T) {
	cases := map[string]func(*fixture, *HTTPResponse){
		"204":           func(_ *fixture, r *HTTPResponse) { r.Status = 204 },
		"digest":        func(_ *fixture, r *HTTPResponse) { r.Body = append([]byte(nil), r.Body...); r.Body[5] ^= 1 },
		"input":         func(_ *fixture, r *HTTPResponse) { r.Fields = setField(r.Fields, "signature-input", "sig1=()") },
		"expired":       func(f *fixture, _ *HTTPResponse) { f.clock.t = f.clock.t.Add(time.Hour) },
		"did header":    func(_ *fixture, r *HTTPResponse) { r.Fields = setField(r.Fields, "x-sage-did", clientDID) },
		"unknown key":   func(f *fixture, _ *HTTPResponse) { delete(f.keys, serverDID+"#sig-1") },
		"replay store":  func(f *fixture, _ *HTTPResponse) { f.responses.Replay = &MemoryReplay{} },
		"not configure": func(f *fixture, _ *HTTPResponse) { f.responses.Clock = nil },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			req, sent := f.request(t)
			v, err := f.receiver.VerifyRequest(context.Background(), *req)
			if err != nil {
				t.Fatal(f.trace.lastReason())
			}
			resp, _ := f.server.SignResponse(v, 200, false, nil, "policy_denied")
			mutate(f, resp)
			if _, err := f.responses.VerifyResponse(context.Background(), sent, *resp); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}

func TestSlogTracerAndUnconfiguredParts(t *testing.T) {
	var buf bytes.Buffer
	tracer := SlogTracer{Logger: slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))}
	f := newFixture(t)
	f.receiver.Trace = tracer
	req, _ := f.request(t)
	req.Method = "GET"
	_, _ = f.receiver.VerifyRequest(context.Background(), *req)
	if !strings.Contains(buf.String(), `"reason":"route"`) || strings.Contains(buf.String(), "payload") {
		t.Fatalf("unexpected trace: %s", buf.String())
	}
	SlogTracer{}.Trace(context.Background(), Event{})
	if (&MemoryReplay{}).Reserve(context.Background(), "a", "b", "n", "i", time.Now()) == nil {
		t.Fatal("unconfigured replay store accepted")
	}
	if (&MemoryReplay{Capacity: 4, Clock: failingClock{}}).Reserve(context.Background(), "a", "b", "n", "i", time.Now()) == nil {
		t.Fatal("replay store without time accepted")
	}
	var nilReceiver *Receiver
	if _, err := nilReceiver.VerifyRequest(context.Background(), HTTPRequest{}); err == nil {
		t.Fatal("nil receiver accepted")
	}
	var nilVerifier *ResponseVerifier
	if _, err := nilVerifier.VerifyResponse(context.Background(), nil, HTTPResponse{}); err == nil {
		t.Fatal("nil response verifier accepted")
	}
	f.receiver.Endpoint = "http://resource.example.invalid/pay"
	if _, err := f.receiver.VerifyRequest(context.Background(), *req); err == nil {
		t.Fatal("plain http endpoint accepted")
	}
	if _, _, err := (&Signer{DID: clientDID, KID: clientDID + "#k", Key: f.client.Key, Clock: failingClock{}, TTL: 60}).
		SignRequest(serverDID, serverURL, nil, RequestOptions{}); err == nil {
		t.Fatal("signer without trusted time signed")
	}
	if _, _, err := f.client.SignRequest(serverDID, serverURL, nil, RequestOptions{ContextID: "x"}); err == nil {
		t.Fatal("invalid context accepted")
	}
}

func TestReplayEntriesExpire(t *testing.T) {
	clock := &fixedClock{t: time.Unix(1000, 0)}
	m := &MemoryReplay{Capacity: 2, Clock: clock}
	if err := m.Reserve(context.Background(), "a", "b", "n1", "i1", time.Unix(1010, 0)); err != nil {
		t.Fatal(err)
	}
	if m.Reserve(context.Background(), "a", "b", "n2", "i1", time.Unix(1010, 0)) == nil {
		t.Fatal("reused message id accepted")
	}
	clock.t = time.Unix(1011, 0)
	if err := m.Reserve(context.Background(), "a", "b", "n1", "i1", time.Unix(1020, 0)); err != nil {
		t.Fatalf("expired entry still blocks: %v", err)
	}
}

// resign rebuilds a correctly signed response from an altered envelope so a
// check after signature verification can be reached.
func resign(t *testing.T, f *fixture, v *VerifiedRequest, env Response, status int) *HTTPResponse {
	t.Helper()
	env.Signature = ""
	sig, err := signEnvelope(responseDomain, env, f.server.Key)
	if err != nil {
		t.Fatal(err)
	}
	env.Signature = sig
	body, _ := json.Marshal(env)
	h := map[string]string{"content-type": "application/json", "content-digest": contentDigest(body),
		"x-sage-did": env.DID, "x-sage-version": Version}
	input := signatureInput(responseComponents, env.KID, env.Created, env.Expires, env.Nonce)
	h["signature-input"] = "sig1=" + input
	h["signature"] = "sig1=:" + base64Std(ed25519.Sign(f.server.Key, responseBase(status, v.http, h, input))) + ":"
	return &HTTPResponse{Status: status, Fields: fieldsOf(h), Body: body}
}

func TestResponseChecksAfterSignatures(t *testing.T) {
	cases := map[string]struct {
		reason string
		mutate func(*Response)
	}{
		"context differs":   {"response context differs", func(r *Response) { r.ContextID = "" }},
		"success and error": {"response outcome", func(r *Response) { r.Error = "policy_denied" }},
		"unknown error":     {"response outcome", func(r *Response) { r.Success = false; r.Error = "teapot" }},
		"bad data":          {"data", func(r *Response) { r.Data = "AA==" }},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			req, sent := f.request(t)
			v, err := f.receiver.VerifyRequest(context.Background(), *req)
			if err != nil {
				t.Fatal(f.trace.lastReason())
			}
			ok, _ := f.server.SignResponse(v, 200, true, []byte("x"), "")
			var env Response
			if err := json.Unmarshal(ok.Body, &env); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&env)
			resp := resign(t, f, v, env, 200)
			if _, err := f.responses.VerifyResponse(context.Background(), sent, *resp); err == nil {
				t.Fatal("accepted")
			}
			if got := f.trace.lastReason(); !strings.Contains(got, tc.reason) {
				t.Fatalf("refused for %q, want %q", got, tc.reason)
			}
		})
	}
}

func TestEnvelopeSignatureMustMatchKey(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	if verifyEnvelope(requestDomain, Request{}, "AAAA", pub) {
		t.Fatal("short signature accepted")
	}
	f := newFixture(t)
	req, _ := f.request(t)
	var env Request
	_ = json.Unmarshal(req.Body, &env)
	if verifyEnvelope(requestDomain, env.unsigned(), env.Signature, pub) {
		t.Fatal("signature verified under another key")
	}
}

func TestSignerAndReceiverConfigurationRefusals(t *testing.T) {
	f := newFixture(t)
	if _, _, err := f.client.SignRequest(serverDID, serverURL, make([]byte, maxPayload+1), RequestOptions{}); err == nil {
		t.Fatal("oversized payload signed")
	}
	if _, _, err := f.client.SignRequest(serverDID, "https://resource.example.invalid", nil, RequestOptions{}); err == nil {
		t.Fatal("endpoint without path accepted")
	}
	req, _ := f.request(t)
	v, err := f.receiver.VerifyRequest(context.Background(), *req)
	if err != nil {
		t.Fatal(f.trace.lastReason())
	}
	f.server.Clock = failingClock{}
	if _, err := f.server.SignResponse(v, 200, true, nil, ""); err == nil {
		t.Fatal("response signed without trusted time")
	}
	f = newFixture(t)
	f.receiver.DID = "not-a-did"
	req, _ = f.request(t)
	f.expectRefusal(t, *req, "receiver not configured")
	f = newFixture(t)
	req, _ = f.request(t)
	req.Body = []byte(strings.Replace(string(req.Body), `"payload":"`, `"payload":"!`, 1))
	req.Fields = setField(req.Fields, "content-digest", contentDigest(req.Body))
	f.expectRefusal(t, *req, "payload")
}
