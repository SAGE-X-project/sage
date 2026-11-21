// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package httpsig

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/crypto"
	"github.com/sage-x-project/sage/core/did"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/secret"
)

const testTarget = "https://agent-b.example/mcp/tools/call"

func testSigner(t *testing.T, seed byte, keyid string) (*Signer, crypto.Signer) {
	t.Helper()
	cs, err := crypto.NewEd25519Signer(secret.From([]byte(strings.Repeat(string(rune(seed)), 32))))
	if err != nil {
		t.Fatal(err)
	}
	ref, err := did.ParseKeyRef(keyid)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSigner(ref, cs)
	if err != nil {
		t.Fatal(err)
	}
	return s, cs
}

func testRequest() Request {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("X-SAGE-DID", vecDID)
	h.Set("Date", "Tue, 01 Sep 2026 12:00:00 GMT")
	return Request{Method: "POST", Target: testTarget, Header: h, Body: []byte(`{"a":1}`)}
}

// verifyRequest runs stages S1, S2, S3 and S6 (03 §5 steps 1, 2 and 6).
func verifyRequest(r Request, key crypto.PublicKey) error {
	p, err := Parse(r.Header, "", core.DefaultLimits())
	if err != nil {
		return err
	}
	base, err := RequestBase(p, r)
	if err != nil {
		return err
	}
	_, err = verifySignature(base, key)
	return err
}

func signed(t *testing.T, s *Signer, r Request, p Params) Request {
	t.Helper()
	f, err := s.SignRequest(context.Background(), r, p)
	if err != nil {
		t.Fatal(err)
	}
	r.Header = r.Header.Clone()
	f.Apply(r.Header)
	return r
}

func TestNewSigner(t *testing.T) {
	_, cs := testSigner(t, 1, vecDID)
	ref, _ := did.ParseKeyRef(vecDID + "#key-2")
	if _, err := NewSigner(did.KeyRef{}, cs); !errors.Is(err, ErrSigner) {
		t.Errorf("zero keyid: %v", err)
	}
	if _, err := NewSigner(ref, nil); !errors.Is(err, ErrSigner) {
		t.Errorf("nil signer: %v", err)
	}
	s, err := NewSigner(ref, cs)
	if err != nil || s.keyid != vecDID+"#key-2" {
		t.Fatalf("keyid %q, %v", s.keyid, err)
	}
}

func TestSignRequestRoundTrip(t *testing.T) {
	s, cs := testSigner(t, 1, vecDID+"#key-1")
	r := signed(t, s, testRequest(), Params{Created: 100, Expires: 400, Nonce: "n-0123456789abcdef"})
	if err := verifyRequest(r, cs.Public()); err != nil {
		t.Fatal(err)
	}
	p, _ := Parse(r.Header, "", core.DefaultLimits())
	want := `sig1=("@method" "@target-uri" "@authority" "content-type" "content-digest" "x-sage-did" "date");keyid="` +
		vecDID + `#key-1";alg="ed25519";created=100;expires=400;nonce="n-0123456789abcdef"`
	if got := p.Label() + "=" + p.SignatureParams(); got != want {
		t.Fatalf("Signature-Input\n got %s\nwant %s", got, want)
	}
	if r.Header.Get("Content-Digest") != ContentDigest([]byte(`{"a":1}`)) {
		t.Fatal("Content-Digest")
	}
	// Another key does not verify it.
	_, other := testSigner(t, 2, vecDID)
	wantReason(t, verifyRequest(r, other.Public()), obs.ReasonSignature)
	// Nor does a changed body digest, method, target or covered header.
	for name, change := range map[string]func(*Request){
		"method":       func(r *Request) { r.Method = "PUT" },
		"target":       func(r *Request) { r.Target = "https://agent-b.example/mcp/tools/list" },
		"query":        func(r *Request) { r.Target += "?a=1" },
		"date":         func(r *Request) { r.Header.Set("Date", "Wed, 02 Sep 2026 12:00:00 GMT") },
		"content-type": func(r *Request) { r.Header.Add("Content-Type", "text/plain") },
	} {
		c := r
		c.Header = r.Header.Clone()
		change(&c)
		t.Run(name, func(t *testing.T) { wantReason(t, verifyRequest(c, cs.Public()), obs.ReasonSignature) })
	}
}

func TestSignRequestWithoutBody(t *testing.T) {
	s, cs := testSigner(t, 1, vecDID)
	r := Request{Method: "GET", Target: "http://agent-b.example:8080/a?b=c", Header: http.Header{}}
	r = signed(t, s, r, Params{Created: 1, Nonce: "n-0123456789abcdef"})
	if r.Header.Get("Content-Digest") != "" {
		t.Fatal("Content-Digest without a body")
	}
	p, _ := Parse(r.Header, "", core.DefaultLimits())
	if !strings.HasPrefix(p.SignatureParams(), `("@method" "@target-uri" "@authority");`) {
		t.Fatalf("covered %s", p.SignatureParams())
	}
	if err := verifyRequest(r, cs.Public()); err != nil {
		t.Fatal(err)
	}
	base, _ := RequestBase(p, r)
	if !strings.Contains(string(base.Bytes()), `"@authority": agent-b.example:8080`+"\n") {
		t.Fatalf("base %s", base.Bytes())
	}
	// A body added after signing must be covered by content-digest.
	r.Body = []byte("x")
	wantReason(t, verifyRequest(r, cs.Public()), obs.ReasonCoverage)
}

func TestSignRequestRefuses(t *testing.T) {
	s, _ := testSigner(t, 1, vecDID)
	ok := Params{Created: 1, Nonce: "n-0123456789abcdef"}
	signedAlready := testRequest()
	signedAlready.Header.Set("Signature", "sig1=:AAAA:")
	wrongDigest := testRequest()
	wrongDigest.Header.Set("Content-Digest", ContentDigest([]byte("other")))
	digestNoBody := testRequest()
	digestNoBody.Body = nil
	digestNoBody.Header.Set("Content-Digest", ContentDigest(nil))
	for name, c := range map[string]struct {
		r    func(Request) Request
		p    Params
		want error
	}{
		"no nonce":          {nil, Params{Created: 1}, ErrSignParams},
		"negative created":  {nil, Params{Created: -1, Nonce: "n"}, ErrSignParams},
		"expires = created": {nil, Params{Created: 5, Expires: 5, Nonce: "n"}, ErrSignParams},
		"control in nonce":  {nil, Params{Created: 1, Nonce: "a\nb"}, ErrSignParams},
		"non-ascii nonce":   {nil, Params{Created: 1, Nonce: "é"}, ErrSignParams},
		"bad method":        {func(r Request) Request { r.Method = "PO ST"; return r }, ok, ErrSignMessage},
		"relative target":   {func(r Request) Request { r.Target = "/mcp"; return r }, ok, ErrSignMessage},
		"fragment":          {func(r Request) Request { r.Target += "#x"; return r }, ok, ErrSignMessage},
		"userinfo":          {func(r Request) Request { r.Target = "https://u@agent-b.example/"; return r }, ok, ErrSignMessage},
		"upper-case scheme": {func(r Request) Request { r.Target = "HTTPS://agent-b.example/"; return r }, ok, ErrSignMessage},
		"ftp":               {func(r Request) Request { r.Target = "ftp://agent-b.example/"; return r }, ok, ErrSignMessage},
		"upper-case host":   {func(r Request) Request { r.Target = "https://Agent-B.example/"; return r }, ok, ErrSignMessage},
		"default port":      {func(r Request) Request { r.Target = "https://agent-b.example:443/"; return r }, ok, ErrSignMessage},
		"empty port":        {func(r Request) Request { r.Target = "https://agent-b.example:/"; return r }, ok, ErrSignMessage},
		"leading-zero port": {func(r Request) Request { r.Target = "https://agent-b.example:0443/"; return r }, ok, ErrSignMessage},
		"already signed":    {func(Request) Request { return signedAlready }, ok, ErrSignMessage},
		"wrong digest":      {func(Request) Request { return wrongDigest }, ok, ErrSignMessage},
		"digest no body":    {func(Request) Request { return digestNoBody }, ok, ErrSignMessage},
	} {
		r := testRequest()
		if c.r != nil {
			r = c.r(r)
		}
		if _, err := s.SignRequest(context.Background(), r, c.p); !errors.Is(err, c.want) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestSignResponseRoundTrip(t *testing.T) {
	reqSigner, _ := testSigner(t, 1, vecDID)
	req := signed(t, reqSigner, testRequest(), Params{Created: 100, Nonce: "n-0123456789abcdef"})
	s, cs := testSigner(t, 3, "did:sage:ethereum:0x2222222222222222222222222222222222222222#key-1")
	resp := Response{Status: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: []byte(`{"r":1}`)}
	f, err := s.SignResponse(context.Background(), resp, req, Params{Created: 101})
	if err != nil {
		t.Fatal(err)
	}
	resp.Header = resp.Header.Clone()
	f.Apply(resp.Header)
	verify := func(resp Response, req Request) error {
		p, err := Parse(resp.Header, "", core.DefaultLimits())
		if err != nil {
			return err
		}
		base, err := ResponseBase(p, resp, req)
		if err != nil {
			return err
		}
		_, err = verifySignature(base, cs.Public())
		return err
	}
	if err := verify(resp, req); err != nil {
		t.Fatal(err)
	}
	p, _ := Parse(resp.Header, "", core.DefaultLimits())
	if !strings.HasPrefix(p.SignatureParams(), `("@status" "@method";req "@target-uri";req "@authority";req "content-digest";req "signature";req "content-type" "content-digest");`) {
		t.Fatalf("covered %s", p.SignatureParams())
	}
	if _, ok := p.Nonce(); ok {
		t.Fatal("nonce is optional on responses")
	}
	// Bound to the request: another request, another status, or another
	// request signature fails.
	other := req
	other.Target = "https://agent-b.example/mcp/tools/list"
	wantReason(t, verify(resp, other), obs.ReasonSignature)
	changed := resp
	changed.Status = 500
	wantReason(t, verify(changed, req), obs.ReasonSignature)
	resigned := signed(t, reqSigner, testRequest(), Params{Created: 100, Nonce: "n-other-0123456789"})
	wantReason(t, verify(resp, resigned), obs.ReasonSignature)
	// A response binds a signed request (E03-20): neither the signer nor
	// the verifier accepts an unsigned one, and a request with a body but
	// no Content-Digest fails closed.
	unsigned := testRequest()
	if _, err := s.SignResponse(context.Background(), Response{Status: 204}, unsigned, Params{Created: 101}); !errors.Is(err, ErrSignMessage) {
		t.Fatalf("unsigned request: %v", err)
	}
	stripped := req
	stripped.Header = req.Header.Clone()
	stripped.Header.Del("Signature")
	if err := verify(resp, stripped); !errors.Is(err, ErrUnsignedRequest) {
		t.Fatalf("unsigned request: %v", err)
	}
	noDigest := req
	noDigest.Header = req.Header.Clone()
	noDigest.Header.Del("Content-Digest")
	wantReason(t, verify(resp, noDigest), obs.ReasonCoverage)
	// Without a request body, Content-Digest or response body, the
	// response covers neither content-digest form.
	bare := signed(t, reqSigner, Request{Method: "GET", Target: testTarget, Header: http.Header{}}, Params{Created: 1, Nonce: "n-0123456789abcdef"})
	f, err = s.SignResponse(context.Background(), Response{Status: 204, Header: http.Header{}}, bare, Params{Created: 101})
	if err != nil || !strings.Contains(f.SignatureInput, `"signature";req`) || strings.Contains(f.SignatureInput, "content-digest") {
		t.Fatalf("%s %v", f.SignatureInput, err)
	}
}

func TestSignResponseRefuses(t *testing.T) {
	s, _ := testSigner(t, 3, vecDID)
	reqSigner, _ := testSigner(t, 1, vecDID)
	req := signed(t, reqSigner, testRequest(), Params{Created: 1, Nonce: "n-0123456789abcdef"})
	for name, c := range map[string]struct {
		resp Response
		req  func(Request) Request
	}{
		"status 99":        {Response{Status: 99}, nil},
		"status 600":       {Response{Status: 600}, nil},
		"bad method":       {Response{Status: 200}, func(r Request) Request { r.Method = ""; return r }},
		"request target":   {Response{Status: 200}, func(r Request) Request { r.Target = "https://A/"; return r }},
		"already signed":   {Response{Status: 200, Header: http.Header{"Signature-Input": {"x"}}}, nil},
		"digest no body":   {Response{Status: 200, Header: http.Header{"Content-Digest": {"x"}}}, nil},
		"digest not match": {Response{Status: 200, Header: http.Header{"Content-Digest": {"x"}}, Body: []byte("b")}, nil},
	} {
		r := req
		if c.req != nil {
			r = c.req(r)
		}
		if _, err := s.SignResponse(context.Background(), c.resp, r, Params{Created: 1}); !errors.Is(err, ErrSignMessage) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if _, err := s.SignResponse(context.Background(), Response{Status: 200}, req, Params{Created: 1, Expires: 1}); !errors.Is(err, ErrSignParams) {
		t.Errorf("expires: %v", err)
	}
}

func TestSignerReviewCases(t *testing.T) {
	ctx := context.Background()
	ok := Params{Created: 1, Nonce: "n-0123456789abcdef"}
	// A keyid with a long alias DID is emitted as configured, and the
	// verifier's keyid parser accepts it (the canonical form is never
	// emitted).
	alias := "did:sage:eth:" + strings.Repeat("a", 243)
	s, cs := testSigner(t, 1, alias+"#key-1")
	r := testRequest()
	r.Header.Del("X-SAGE-DID")
	r = signed(t, s, r, ok)
	p, err := Parse(r.Header, "", core.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	keyid, _ := p.KeyID()
	if keyid != alias+"#key-1" {
		t.Fatalf("keyid %q", keyid)
	}
	if _, err := did.ParseKeyRef(keyid); err != nil {
		t.Fatalf("emitted keyid does not parse: %v", err)
	}
	if err := verifyRequest(r, cs.Public()); err != nil {
		t.Fatal(err)
	}

	s, _ = testSigner(t, 1, vecDID)
	// X-SAGE-DID must be the signer's DID (03 §3).
	other := testRequest()
	other.Header.Set("X-SAGE-DID", "did:sage:ethereum:0x2222222222222222222222222222222222222222")
	if _, err := s.SignRequest(ctx, other, ok); !errors.Is(err, ErrSignMessage) {
		t.Errorf("other X-SAGE-DID: %v", err)
	}
	// Nonces follow 03 §5 Replay on requests and responses.
	for _, n := range []string{"short", strings.Repeat("a", 65), `a "b" 0123456789ab`, "0123456789abcdef.", "0123456789abcdéf"} {
		if _, err := s.SignRequest(ctx, testRequest(), Params{Created: 1, Nonce: n}); !errors.Is(err, ErrSignParams) {
			t.Errorf("nonce %q: %v", n, err)
		}
	}
	for _, n := range []string{strings.Repeat("a", 16), strings.Repeat("Z", 64), "8f0c5b1c-0d1a-4c8e-9a51-6d2e4b7f1a10", "a_b-0123456789AB"} {
		if !validNonce(n) {
			t.Errorf("nonce %q rejected", n)
		}
	}
	// RFC 8941 integers have at most 15 digits.
	for _, p := range []Params{{Created: 1_000_000_000_000_000, Nonce: ok.Nonce}, {Created: 1, Expires: 1_000_000_000_000_000, Nonce: ok.Nonce}, {Created: 1, Expires: -1, Nonce: ok.Nonce}} {
		if _, err := s.SignRequest(ctx, testRequest(), p); !errors.Is(err, ErrSignParams) {
			t.Errorf("%+v: %v", p, err)
		}
	}
	// Targets that are not RFC 3986 URIs, or whose authority would not be
	// the text inside @target-uri, are refused.
	for _, target := range []string{
		"https://agent-b.example/p q", "https://agent-b.example/pé", "https://:8443/p",
		"https://[fe80::1%25en0]/p", "https://agent-b.example/?a=%ZZ", "https://agent-b.example/%4",
		"https://agent-b.example/\x00",
	} {
		r := testRequest()
		r.Target = target
		_, err := s.SignRequest(ctx, r, ok)
		if !errors.Is(err, ErrSignMessage) {
			t.Errorf("%q: %v", target, err)
		}
		// Signing errors are not inbound rejections.
		if errors.Is(err, core.ErrAuthentication) {
			t.Errorf("%q: a signing error reads as an authentication failure", target)
		}
	}
	// Apply leaves one line of each field whatever the key spelling.
	h := http.Header{"content-digest": {"x"}, "signature-input": {"y"}, "signature": {"z"}, "X-Other": {"k"}}
	f := Fields{SignatureInput: "a", Signature: "b", ContentDigest: "c"}
	f.Apply(h)
	if len(h) != 4 || h.Get("Content-Digest") != "c" || h.Get("Signature-Input") != "a" || h.Get("Signature") != "b" || h.Get("X-Other") != "k" {
		t.Fatalf("Apply: %v", h)
	}
	lower := testRequest()
	delete(lower.Header, "Content-Type")
	lower.Header["content-digest"] = []string{ContentDigest(lower.Body)}
	r = signed(t, s, lower, ok)
	if _, err := Parse(r.Header, "", core.DefaultLimits()); err != nil {
		t.Fatalf("lower-case Content-Digest key: %v", err)
	}
}
