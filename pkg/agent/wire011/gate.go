package wire011

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Peer protection follows decision D17 and A2AB-07. Whether a peer must use
// SAGE is decided from trusted sources (configuration or the Registry), never
// from unsigned headers, because an attacker can strip those. A SAGE peer is
// always verified strictly with no fallback. A peer without SAGE evidence is
// unprotected: its messages are labelled as such and the application policy
// decides which operations they may reach.

// Classifier tells whether a peer must use SAGE. It returns true for SAGE
// peers, false for peers with no SAGE evidence, and an error when the trusted
// source cannot answer; callers then refuse instead of treating the peer as
// unprotected.
type Classifier interface {
	RequiresSAGE(ctx context.Context, peerDID, target string) (bool, error)
}

// StaticClassifier marks configured DIDs and endpoints as SAGE peers.
type StaticClassifier struct {
	DIDs      map[string]bool
	Endpoints map[string]bool
}

func (c StaticClassifier) RequiresSAGE(_ context.Context, peerDID, target string) (bool, error) {
	return c.DIDs[peerDID] || c.Endpoints[target], nil
}

// AnyOf requires SAGE if any classifier does; an error from any of them refuses.
type AnyOf []Classifier

func (a AnyOf) RequiresSAGE(ctx context.Context, peerDID, target string) (bool, error) {
	for _, c := range a {
		sage, err := c.RequiresSAGE(ctx, peerDID, target)
		if err != nil {
			return false, err
		}
		if sage {
			return true, nil
		}
	}
	return false, nil
}

// Protection labels what a handler or caller received.
type Protection struct {
	Protected bool             // true only after full wire011 verification
	Request   *VerifiedRequest // set when Protected
}

type protectionKey struct{}

// ProtectionFrom returns the label the Gate attached to a request context.
// A missing label means unprotected.
func ProtectionFrom(ctx context.Context) Protection {
	p, _ := ctx.Value(protectionKey{}).(Protection)
	return p
}

// UnprotectedPolicy decides whether a request without SAGE may reach the
// application. The default (nil) refuses all of them.
type UnprotectedPolicy func(r *http.Request) bool

// Gate is the default inbound entry point for the 0.11.0 path.
type Gate struct {
	Receiver *Receiver
	// AllowUnprotected admits requests that carry no SAGE marker. Nil refuses.
	AllowUnprotected UnprotectedPolicy
	// TrustedTarget returns the absolute target URI of a request as the trusted
	// HTTP stack observed it. It must not read Host, Forwarded or X-Forwarded.
	TrustedTarget func(r *http.Request) string
	Next          http.Handler
}

// sageMarkers are fields whose presence means the sender claims SAGE. Once
// claimed, a failed verification is final.
var sageMarkers = []string{"Signature", "Signature-Input", "X-Sage-Did", "X-Sage-Version", "X-Sage-Message-Id"}

func (g *Gate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if g == nil || g.Receiver == nil || g.Next == nil || g.TrustedTarget == nil {
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody+1))
	if err != nil || len(body) > maxBody {
		http.Error(w, "request too large", http.StatusRequestEntityTooLarge)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if !claimsSAGE(r.Header) {
		allowed := g.AllowUnprotected != nil && g.AllowUnprotected(r)
		trace(r.Context(), g.Receiver.Trace, Event{Stage: "gate.unprotected", To: g.Receiver.DID, Decision: decision(allowed)})
		if !allowed {
			http.Error(w, "authentication failed", http.StatusUnauthorized)
			return
		}
		g.Next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), protectionKey{}, Protection{})))
		return
	}
	req := HTTPRequest{Method: r.Method, TargetURI: g.TrustedTarget(r), Authority: r.Host, Body: body}
	for name, values := range r.Header {
		for _, v := range values {
			req.Fields = append(req.Fields, Field{Name: name, Value: v})
		}
	}
	verified, err := g.Receiver.VerifyRequest(r.Context(), req)
	if err != nil {
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}
	ctx := context.WithValue(r.Context(), protectionKey{}, Protection{Protected: true, Request: verified})
	g.Next.ServeHTTP(w, r.WithContext(ctx))
}

func claimsSAGE(h http.Header) bool {
	for _, name := range sageMarkers {
		if _, ok := h[http.CanonicalHeaderKey(name)]; ok {
			return true
		}
	}
	for name := range h {
		if strings.HasPrefix(strings.ToLower(name), "x-sage-") {
			return true
		}
	}
	return false
}

func decision(allowed bool) string {
	if allowed {
		return "allowed"
	}
	return "refused"
}

// ErrUnprotectedRefused means the caller asked for SAGE-only delivery to a
// peer that has no SAGE evidence.
var ErrUnprotectedRefused = errors.New("wire011: peer is not a SAGE peer")

// Result is what Client.Send returns.
type Result struct {
	Protected bool
	Response  *VerifiedResponse // set when Protected
	Status    int
	Body      []byte // raw body of an unprotected exchange
}

// Client sends to peers, signing for SAGE peers and never downgrading them.
type Client struct {
	Signer     *Signer
	Classifier Classifier
	Responses  *ResponseVerifier
	HTTP       *http.Client
	// AllowUnprotected permits plain delivery to peers without SAGE evidence.
	AllowUnprotected bool
}

// Send delivers payload to the peer. For a SAGE peer it signs the request and
// accepts only a verified response; any failure is returned as is, and the
// request is never resent without SAGE.
func (c *Client) Send(ctx context.Context, peerDID, target string, payload []byte, opts RequestOptions) (*Result, error) {
	if c == nil || c.Signer == nil || c.Classifier == nil || c.Responses == nil || c.HTTP == nil {
		return nil, errors.New("wire011: client not configured")
	}
	sage, err := c.Classifier.RequiresSAGE(ctx, peerDID, target)
	if err != nil {
		return nil, fmt.Errorf("wire011: classify peer: %w", err)
	}
	if !sage {
		if !c.AllowUnprotected {
			return nil, ErrUnprotectedRefused
		}
		status, body, _, err := c.post(ctx, target, nil, payload)
		if err != nil {
			return nil, err
		}
		return &Result{Status: status, Body: body}, nil
	}
	req, sent, err := c.Signer.SignRequest(peerDID, target, payload, opts)
	if err != nil {
		return nil, err
	}
	status, body, fields, err := c.post(ctx, req.TargetURI, req.Fields, req.Body)
	if err != nil {
		return nil, err
	}
	verified, err := c.Responses.VerifyResponse(ctx, sent, HTTPResponse{Status: status, Fields: fields, Body: body})
	if err != nil {
		return nil, err
	}
	return &Result{Protected: true, Response: verified, Status: status}, nil
}

func (c *Client) post(ctx context.Context, target string, fields []Field, body []byte) (int, []byte, []Field, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return 0, nil, nil, err
	}
	if fields == nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, f := range fields {
		req.Header.Add(f.Name, f.Value)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil || len(raw) > maxBody {
		return 0, nil, nil, errors.New("wire011: response too large")
	}
	var out []Field
	for name, values := range resp.Header {
		for _, v := range values {
			out = append(out, Field{Name: name, Value: v})
		}
	}
	return resp.StatusCode, raw, out, nil
}

// DownstreamUnprotected is the metadata key a relaying agent sets on its own
// signed request or response when part of the work went through a peer
// without SAGE. The value is the unprotected peer's endpoint. Metadata is
// authenticated but grants no authority (TRANSPORT-01).
const DownstreamUnprotected = "downstream_unprotected"

// WriteResponse writes a signed response produced by Signer.SignResponse.
func WriteResponse(w http.ResponseWriter, resp *HTTPResponse) error {
	if resp == nil {
		return errors.New("wire011: no response")
	}
	for _, f := range resp.Fields {
		w.Header().Add(f.Name, f.Value)
	}
	w.WriteHeader(resp.Status)
	_, err := w.Write(resp.Body)
	return err
}
