package wire011

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	clientDID = "did:sage:eip155:31337:0x2279b7a0a67db372996a5fab50d91eaa73d2ebe6:payment-agent"
	serverDID = "did:sage:eip155:31337:0x2279b7a0a67db372996a5fab50d91eaa73d2ebe6:resource-server"
	serverURL = "https://resource.example.invalid/pay"
)

type fixedClock struct{ t time.Time }

func (c *fixedClock) Now() (time.Time, error) { return c.t, nil }

type failingClock struct{}

func (failingClock) Now() (time.Time, error) { return time.Time{}, errors.New("no trusted time") }

type keyMap map[string]ed25519.PublicKey

func (k keyMap) ResolveSigningKey(_ context.Context, kid string) (ed25519.PublicKey, error) {
	if key, ok := k[kid]; ok {
		return key, nil
	}
	return nil, errors.New("unknown key")
}

type recorder struct {
	mu     sync.Mutex
	events []Event
}

func (r *recorder) Trace(_ context.Context, e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func (r *recorder) lastReason() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := len(r.events) - 1; i >= 0; i-- {
		if r.events[i].Decision == "refused" {
			return r.events[i].Reason
		}
	}
	return ""
}

type fixture struct {
	clock          *fixedClock
	client, server *Signer
	keys           keyMap
	receiver       *Receiver
	responses      *ResponseVerifier
	trace          *recorder
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	clock := &fixedClock{t: time.Unix(1_800_000_000, 0)}
	cpub, cpriv, _ := ed25519.GenerateKey(nil)
	spub, spriv, _ := ed25519.GenerateKey(nil)
	keys := keyMap{clientDID + "#sig-3": cpub, serverDID + "#sig-1": spub}
	rec := &recorder{}
	return &fixture{
		clock:  clock,
		client: &Signer{DID: clientDID, KID: clientDID + "#sig-3", Key: cpriv, Clock: clock, TTL: 120},
		server: &Signer{DID: serverDID, KID: serverDID + "#sig-1", Key: spriv, Clock: clock, TTL: 120},
		keys:   keys,
		receiver: &Receiver{DID: serverDID, Endpoint: serverURL, Keys: keys, Clock: clock, Trace: rec,
			Replay: &MemoryReplay{Capacity: 64, Clock: clock}},
		responses: &ResponseVerifier{Keys: keys, Clock: clock, Trace: rec, Replay: &MemoryReplay{Capacity: 64, Clock: clock}},
		trace:     rec,
	}
}

func (f *fixture) request(t *testing.T) (*HTTPRequest, *SentRequest) {
	t.Helper()
	req, sent, err := f.client.SignRequest(serverDID, serverURL, []byte(`{"tool":"pay","amount":"4000000"}`),
		RequestOptions{ContextID: "3b1b2a8e-2f5c-4c4e-9c1a-8d3f6e7a9b10"})
	if err != nil {
		t.Fatal(err)
	}
	return req, sent
}

func setField(fields []Field, name, value string) []Field {
	out := append([]Field(nil), fields...)
	for i := range out {
		if out[i].Name == name {
			out[i].Value = value
		}
	}
	return out
}

func (f *fixture) expectRefusal(t *testing.T, req HTTPRequest, reason string) {
	t.Helper()
	if _, err := f.receiver.VerifyRequest(context.Background(), req); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("expected refusal, got %v", err)
	}
	if got := f.trace.lastReason(); !strings.Contains(got, reason) {
		t.Fatalf("refused for %q, want %q", got, reason)
	}
}

func TestRequestResponseRoundTrip(t *testing.T) {
	f := newFixture(t)
	req, sent := f.request(t)
	v, err := f.receiver.VerifyRequest(context.Background(), *req)
	if err != nil {
		t.Fatalf("request refused: %s", f.trace.lastReason())
	}
	if string(v.Payload) != `{"tool":"pay","amount":"4000000"}` || v.Envelope.DID != clientDID {
		t.Fatal("wrong verified request")
	}
	resp, err := f.server.SignResponse(v, 200, true, []byte(`{"paid":true}`), "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.responses.VerifyResponse(context.Background(), sent, *resp)
	if err != nil {
		t.Fatalf("response refused: %s", f.trace.lastReason())
	}
	if string(got.Data) != `{"paid":true}` || got.Envelope.ContextID != sent.Envelope().ContextID {
		t.Fatal("wrong verified response")
	}
	if _, err := f.responses.VerifyResponse(context.Background(), sent, *resp); err == nil ||
		!strings.Contains(f.trace.lastReason(), "second terminal response") {
		t.Fatalf("second response accepted: %v %q", err, f.trace.lastReason())
	}
}

func TestRequestRefusals(t *testing.T) {
	cases := []struct {
		name, reason string
		mutate       func(*fixture, *HTTPRequest)
	}{
		{"wrong method", "route", func(_ *fixture, r *HTTPRequest) { r.Method = "PUT" }},
		{"wrong target", "route", func(_ *fixture, r *HTTPRequest) { r.TargetURI = "https://resource.example.invalid/other" }},
		{"forged authority", "route", func(_ *fixture, r *HTTPRequest) { r.Authority = "evil.example.invalid" }},
		{"body changed", "content-digest", func(_ *fixture, r *HTTPRequest) { r.Body = append([]byte(nil), r.Body...); r.Body[10] ^= 1 }},
		{"duplicate digest", "duplicate content-digest", func(_ *fixture, r *HTTPRequest) {
			r.Fields = append(r.Fields, Field{"Content-Digest", contentDigest(r.Body)})
		}},
		{"meta header", "x-sage-meta-", func(_ *fixture, r *HTTPRequest) { r.Fields = append(r.Fields, Field{"X-Sage-Meta-Role", "admin"}) }},
		{"content encoding", "content-encoding", func(_ *fixture, r *HTTPRequest) { r.Fields = append(r.Fields, Field{"Content-Encoding", "gzip"}) }},
		{"wrong version header", "x-sage-version", func(_ *fixture, r *HTTPRequest) { r.Fields = setField(r.Fields, "x-sage-version", "0.10.0") }},
		{"header DID differs", "x-sage-did does not match", func(_ *fixture, r *HTTPRequest) {
			r.Fields = setField(r.Fields, "x-sage-did", serverDID)
		}},
		{"second signature", "second signature", func(_ *fixture, r *HTTPRequest) {
			r.Fields = setField(r.Fields, "signature-input", fieldValue(r.Fields, "signature-input")+`, sig2=("@method");alg="ed25519"`)
		}},
		{"algorithm downgrade", "alg", func(_ *fixture, r *HTTPRequest) {
			r.Fields = setField(r.Fields, "signature-input", strings.Replace(fieldValue(r.Fields, "signature-input"), `alg="ed25519"`, `alg="hmac-sha256"`, 1))
		}},
		{"0.10 tag", "tag", func(_ *fixture, r *HTTPRequest) {
			r.Fields = setField(r.Fields, "signature-input", strings.Replace(fieldValue(r.Fields, "signature-input"), Tag, "sage-0.10.0", 1))
		}},
		{"narrowed coverage", "coverage", func(_ *fixture, r *HTTPRequest) {
			r.Fields = setField(r.Fields, "signature-input", strings.Replace(fieldValue(r.Fields, "signature-input"), ` "content-digest"`, "", 1))
		}},
		{"unknown parameter", "unknown parameter", func(_ *fixture, r *HTTPRequest) {
			r.Fields = setField(r.Fields, "signature-input", fieldValue(r.Fields, "signature-input")+`;foo="1"`)
		}},
		{"header nonce differs", "http parameters do not match", func(_ *fixture, r *HTTPRequest) {
			r.Fields = setField(r.Fields, "signature-input", replaceNonce(fieldValue(r.Fields, "signature-input")))
		}},
		{"tampered http signature", "http signature", func(_ *fixture, r *HTTPRequest) {
			v := fieldValue(r.Fields, "signature")
			r.Fields = setField(r.Fields, "signature", v[:7]+flip(v[7])+v[8:])
		}},
		{"unknown key", "key resolution", func(f *fixture, _ *HTTPRequest) { delete(f.keys, clientDID+"#sig-3") }},
		{"expired", "expired", func(f *fixture, _ *HTTPRequest) { f.clock.t = f.clock.t.Add(200 * time.Second) }},
		{"created in the future", "created in the future", func(f *fixture, _ *HTTPRequest) { f.clock.t = f.clock.t.Add(-time.Minute) }},
		{"no trusted time", "clock unavailable", func(f *fixture, _ *HTTPRequest) { f.receiver.Clock = failingClock{} }},
		{"other recipient", "recipient", func(f *fixture, _ *HTTPRequest) { f.receiver.DID = clientDID + "x" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			req, _ := f.request(t)
			tc.mutate(f, req)
			f.expectRefusal(t, *req, tc.reason)
		})
	}
}

func TestEnvelopeTamperingIsRefused(t *testing.T) {
	f := newFixture(t)
	req, _ := f.request(t)
	var env map[string]any
	if err := json.Unmarshal(req.Body, &env); err != nil {
		t.Fatal(err)
	}
	env["payload"] = b64([]byte(`{"tool":"pay","amount":"9000000"}`))
	body, _ := json.Marshal(env)
	req.Body = body
	req.Fields = setField(req.Fields, "content-digest", contentDigest(body))
	f.expectRefusal(t, *req, "http signature")

	f = newFixture(t)
	req, _ = f.request(t)
	req.Body = []byte(strings.Replace(string(req.Body), `"encoding":"plain"`, `"encoding":"plain","admin":true`, 1))
	req.Fields = setField(req.Fields, "content-digest", contentDigest(req.Body))
	f.expectRefusal(t, *req, "member \"admin\" not allowed")
}

func TestReplayIsRefused(t *testing.T) {
	f := newFixture(t)
	req, _ := f.request(t)
	if _, err := f.receiver.VerifyRequest(context.Background(), *req); err != nil {
		t.Fatal(f.trace.lastReason())
	}
	f.expectRefusal(t, *req, "replay")
}

func TestReplayStoreFailsClosedWhenFull(t *testing.T) {
	f := newFixture(t)
	f.receiver.Replay = &MemoryReplay{Capacity: 2, Clock: f.clock}
	first, _ := f.request(t)
	if _, err := f.receiver.VerifyRequest(context.Background(), *first); err != nil {
		t.Fatal(f.trace.lastReason())
	}
	second, _ := f.request(t)
	f.expectRefusal(t, *second, "replay")
}

func TestResponseRefusals(t *testing.T) {
	cases := []struct {
		name, reason string
		mutate       func(*testing.T, *fixture, *VerifiedRequest, *HTTPResponse) *HTTPResponse
	}{
		{"signed by a third party", "not from the request recipient", func(_ *testing.T, f *fixture, v *VerifiedRequest, _ *HTTPResponse) *HTTPResponse {
			third := *f.server
			third.DID = clientDID + "-other"
			third.KID = third.DID + "#sig-1"
			v.Envelope.Recipient = third.DID
			resp, _ := third.SignResponse(v, 200, true, nil, "")
			return resp
		}},
		{"response for another request", "does not bind this request", func(t *testing.T, f *fixture, _ *VerifiedRequest, _ *HTTPResponse) *HTTPResponse {
			other, _ := f.request(t)
			v, _ := f.receiver.VerifyRequest(context.Background(), *other)
			resp, _ := f.server.SignResponse(v, 200, true, nil, "")
			return resp
		}},
		{"status changed", "http signature", func(_ *testing.T, _ *fixture, _ *VerifiedRequest, r *HTTPResponse) *HTTPResponse {
			r.Status = 500
			return r
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			req, sent := f.request(t)
			v, err := f.receiver.VerifyRequest(context.Background(), *req)
			if err != nil {
				t.Fatal(f.trace.lastReason())
			}
			resp, err := f.server.SignResponse(v, 200, true, []byte("ok"), "")
			if err != nil {
				t.Fatal(err)
			}
			resp = tc.mutate(t, f, v, resp)
			if _, err := f.responses.VerifyResponse(context.Background(), sent, *resp); !errors.Is(err, ErrAuthentication) {
				t.Fatalf("response accepted")
			}
			if got := f.trace.lastReason(); !strings.Contains(got, tc.reason) {
				t.Fatalf("refused for %q, want %q", got, tc.reason)
			}
		})
	}
}

func TestSignerRefusesInvalidOutcomes(t *testing.T) {
	f := newFixture(t)
	req, _ := f.request(t)
	v, _ := f.receiver.VerifyRequest(context.Background(), *req)
	if _, err := f.server.SignResponse(v, 204, true, nil, ""); err == nil {
		t.Fatal("204 shortcut accepted")
	}
	if _, err := f.server.SignResponse(v, 200, false, nil, "teapot"); err == nil {
		t.Fatal("unknown error code accepted")
	}
	if _, err := f.client.SignResponse(v, 200, true, nil, ""); err == nil {
		t.Fatal("non-recipient signed a response")
	}
	f.client.TTL = 301
	if _, _, err := f.client.SignRequest(serverDID, serverURL, nil, RequestOptions{}); err == nil {
		t.Fatal("lifetime above 300 seconds accepted")
	}
	f.client.TTL = 60
	if _, _, err := f.client.SignRequest(serverDID, "https://resource.example.invalid:443/pay", nil, RequestOptions{}); err == nil {
		t.Fatal("non-canonical endpoint accepted")
	}
}

func fieldValue(fields []Field, name string) string {
	for _, f := range fields {
		if f.Name == name {
			return f.Value
		}
	}
	return ""
}

func replaceNonce(input string) string {
	i := strings.Index(input, `nonce="`) + len(`nonce="`)
	return input[:i] + "AAAAAAAAAAAAAAAAAAAAAA" + input[i+22:]
}

func flip(c byte) string {
	if c == 'A' {
		return "B"
	}
	return "A"
}
