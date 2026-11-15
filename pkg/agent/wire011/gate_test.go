package wire011

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type gateFixture struct {
	*fixture
	server   *httptest.Server
	endpoint string
	seen     *Protection
	allow    bool
}

// newGateFixture runs a TLS server whose handler answers protected requests
// with a signed response and unprotected ones with plain JSON.
func newGateFixture(t *testing.T) *gateFixture {
	t.Helper()
	g := &gateFixture{fixture: newFixture(t)}
	gate := &Gate{Receiver: g.receiver, AllowUnprotected: func(*http.Request) bool { return g.allow },
		TrustedTarget: func(r *http.Request) string { return g.endpoint },
		Next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := ProtectionFrom(r.Context())
			g.seen = &p
			if !p.Protected {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"unprotected":true}`)
				return
			}
			resp, err := g.server2().SignResponse(p.Request, 200, true, []byte(`{"paid":true}`), "")
			if err != nil {
				t.Error(err)
				return
			}
			_ = WriteResponse(w, resp)
		})}
	g.server = httptest.NewTLSServer(gate)
	t.Cleanup(g.server.Close)
	g.endpoint = g.server.URL + "/pay"
	g.receiver.Endpoint = g.endpoint
	return g
}

func (g *gateFixture) server2() *Signer { return g.fixture.server }

func (g *gateFixture) client(classifier Classifier, allowUnprotected bool) *Client {
	return &Client{Signer: g.fixture.client, Classifier: classifier, Responses: g.responses, HTTP: g.server.Client(),
		AllowUnprotected: allowUnprotected}
}

func TestGateProtectedExchange(t *testing.T) {
	g := newGateFixture(t)
	c := g.client(StaticClassifier{DIDs: map[string]bool{serverDID: true}}, false)
	res, err := c.Send(context.Background(), serverDID, g.endpoint, []byte(`{"tool":"pay"}`), RequestOptions{})
	if err != nil {
		t.Fatalf("protected exchange failed: %v (%s)", err, g.trace.lastReason())
	}
	if !res.Protected || string(res.Response.Data) != `{"paid":true}` || g.seen == nil || !g.seen.Protected {
		t.Fatal("exchange not marked protected")
	}
}

func TestGateRefusesUnprotectedByDefault(t *testing.T) {
	g := newGateFixture(t)
	resp, err := g.server.Client().Post(g.endpoint, "application/json", strings.NewReader(`{"tool":"pay"}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized || g.seen != nil {
		t.Fatalf("unprotected request reached the handler: %d", resp.StatusCode)
	}
}

func TestGateLabelsAllowedUnprotected(t *testing.T) {
	g := newGateFixture(t)
	g.allow = true
	res, err := g.client(StaticClassifier{}, true).Send(context.Background(), "did:web:example.invalid", g.endpoint, []byte(`{}`), RequestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Protected || g.seen == nil || g.seen.Protected || string(res.Body) != `{"unprotected":true}` {
		t.Fatal("unprotected exchange not labelled")
	}
}

func TestGateNeverFallsBackAfterSAGEClaim(t *testing.T) {
	g := newGateFixture(t)
	g.allow = true // even when unprotected requests are allowed
	req, err := http.NewRequest(http.MethodPost, g.endpoint, strings.NewReader(`{"tool":"pay"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sage-Did", clientDID)
	req.Header.Set("Signature", "sig1=:AAAA:")
	resp, err := g.server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized || g.seen != nil {
		t.Fatalf("request claiming SAGE reached the handler unverified: %d", resp.StatusCode)
	}
}

func TestClientNeverDowngradesSAGEPeer(t *testing.T) {
	plain := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"paid":true}`)
	}))
	defer plain.Close()
	f := newFixture(t)
	c := &Client{Signer: f.client, Classifier: StaticClassifier{DIDs: map[string]bool{serverDID: true}}, Responses: f.responses,
		HTTP: plain.Client(), AllowUnprotected: true}
	if _, err := c.Send(context.Background(), serverDID, plain.URL+"/pay", []byte(`{}`), RequestOptions{}); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("unsigned answer from a SAGE peer accepted: %v", err)
	}
}

type failingClassifier struct{}

func (failingClassifier) RequiresSAGE(context.Context, string, string) (bool, error) {
	return false, errors.New("registry unavailable")
}

func TestClientRefusalsWithoutEvidence(t *testing.T) {
	g := newGateFixture(t)
	if _, err := g.client(StaticClassifier{}, false).Send(context.Background(), "did:web:x", g.endpoint, nil, RequestOptions{}); !errors.Is(err, ErrUnprotectedRefused) {
		t.Fatalf("plain delivery without permission: %v", err)
	}
	if _, err := g.client(failingClassifier{}, true).Send(context.Background(), serverDID, g.endpoint, nil, RequestOptions{}); err == nil {
		t.Fatal("classifier failure treated as an unprotected peer")
	}
	if _, err := g.client(AnyOf{StaticClassifier{}, failingClassifier{}}, true).Send(context.Background(), serverDID, g.endpoint, nil, RequestOptions{}); err == nil {
		t.Fatal("AnyOf ignored a failing classifier")
	}
	var nilClient *Client
	if _, err := nilClient.Send(context.Background(), serverDID, g.endpoint, nil, RequestOptions{}); err == nil {
		t.Fatal("nil client sent")
	}
	if err := WriteResponse(httptest.NewRecorder(), nil); err == nil {
		t.Fatal("nil response written")
	}
	rec := httptest.NewRecorder()
	(&Gate{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/pay", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatal("unconfigured gate admitted a request")
	}
}

func TestGateAndClientEdgeFailures(t *testing.T) {
	g := newGateFixture(t)
	rec := httptest.NewRecorder()
	gate := &Gate{Receiver: g.receiver, TrustedTarget: func(*http.Request) string { return g.endpoint }, Next: http.NotFoundHandler()}
	gate.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/pay", strings.NewReader(strings.Repeat("a", maxBody+1))))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body: %d", rec.Code)
	}
	sage := StaticClassifier{DIDs: map[string]bool{serverDID: true}}
	if _, err := g.client(sage, false).Send(context.Background(), serverDID, g.server.URL, nil, RequestOptions{}); err == nil {
		t.Fatal("non-canonical target signed")
	}
	closed := httptest.NewTLSServer(http.NotFoundHandler())
	url, client := closed.URL+"/pay", closed.Client()
	closed.Close()
	c := &Client{Signer: g.fixture.client, Classifier: sage, Responses: g.responses, HTTP: client}
	if _, err := c.Send(context.Background(), serverDID, url, nil, RequestOptions{}); err == nil {
		t.Fatal("connection failure hidden")
	}
	c.Classifier, c.AllowUnprotected = StaticClassifier{}, true
	if _, err := c.Send(context.Background(), "did:web:x", url, nil, RequestOptions{}); err == nil {
		t.Fatal("connection failure hidden on the plain path")
	}
	if _, err := c.Send(context.Background(), "did:web:x", "://bad", nil, RequestOptions{}); err == nil {
		t.Fatal("bad URL accepted")
	}
}
