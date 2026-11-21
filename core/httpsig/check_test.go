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
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/crypto"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/secret"
)

func secretScalar(t *testing.T) secret.Bytes {
	t.Helper()
	return secret.From([]byte(strings.Repeat("\x07", 32)))
}

const reqParams = `;keyid="k";created=1;nonce="n"`

func parsedOf(t *testing.T, input string) Parsed {
	t.Helper()
	p, err := Parse(http.Header{"Signature-Input": {input}, "Signature": {"sig1=:AAAA:"}}, "", core.DefaultLimits())
	if err != nil {
		t.Fatalf("%s: %v", input, err)
	}
	return p
}

func TestCheckRequest(t *testing.T) {
	r := Request{Method: "POST", Target: testTarget, Header: http.Header{}, Body: []byte("b")}
	noBody := r
	noBody.Body = nil
	const must = `"@method" "@target-uri" "@authority"`
	if err := CheckRequest(parsedOf(t, `sig1=(`+must+` "content-digest" "@scheme" "@query" "x-a")`+reqParams), r); err != nil {
		t.Fatalf("allowed set: %v", err)
	}
	for name, c := range map[string]struct {
		input string
		r     Request
		want  obs.Reason
	}{
		"no keyid":         {`sig1=(` + must + `);created=1;nonce="n"`, noBody, obs.ReasonParameters},
		"no created":       {`sig1=(` + must + `);keyid="k";nonce="n"`, noBody, obs.ReasonParameters},
		"no nonce":         {`sig1=(` + must + `);keyid="k";created=1`, noBody, obs.ReasonParameters},
		"no @method":       {`sig1=("@target-uri" "@authority")` + reqParams, noBody, obs.ReasonCoverage},
		"no @target-uri":   {`sig1=("@method" "@authority")` + reqParams, noBody, obs.ReasonCoverage},
		"no @authority":    {`sig1=("@method" "@target-uri")` + reqParams, noBody, obs.ReasonCoverage},
		"body, no digest":  {`sig1=(` + must + `)` + reqParams, r, obs.ReasonCoverage},
		"H4 empty set":     {`sig1=()` + reqParams, noBody, obs.ReasonCoverage},
		"@request-target":  {`sig1=(` + must + ` "@request-target")` + reqParams, noBody, obs.ReasonCoverage},
		"@path":            {`sig1=(` + must + ` "@path")` + reqParams, noBody, obs.ReasonCoverage},
		"@query-param":     {`sig1=(` + must + ` "@query-param";name="q")` + reqParams, noBody, obs.ReasonCoverage},
		"@status":          {`sig1=(` + must + ` "@status")` + reqParams, noBody, obs.ReasonCoverage},
		"unknown derived":  {`sig1=(` + must + ` "@foo")` + reqParams, noBody, obs.ReasonCoverage},
		";req":             {`sig1=("@method";req "@target-uri" "@authority")` + reqParams, noBody, obs.ReasonCoverage},
		";sf":              {`sig1=(` + must + ` "content-type";sf)` + reqParams, noBody, obs.ReasonCoverage},
		";bs":              {`sig1=(` + must + ` "content-type";bs)` + reqParams, noBody, obs.ReasonCoverage},
		";key":             {`sig1=(` + must + ` "x";key="a")` + reqParams, noBody, obs.ReasonCoverage},
		";tr":              {`sig1=(` + must + ` "x";tr)` + reqParams, noBody, obs.ReasonCoverage},
		";name":            {`sig1=(` + must + ` "x";name="a")` + reqParams, noBody, obs.ReasonCoverage},
		"twice":            {`sig1=(` + must + ` "@method")` + reqParams, noBody, obs.ReasonCoverage},
		"upper-case field": {`sig1=(` + must + ` "Content-Type")` + reqParams, noBody, obs.ReasonCoverage},
		"field with space": {`sig1=(` + must + ` "a b")` + reqParams, noBody, obs.ReasonCoverage},
		"empty field":      {`sig1=(` + must + ` "")` + reqParams, noBody, obs.ReasonCoverage},
		"authority upper":  {`sig1=(` + must + `)` + reqParams, Request{Method: "GET", Target: "https://Agent-B.example/"}, obs.ReasonCoverage},
		"default port":     {`sig1=(` + must + `)` + reqParams, Request{Method: "GET", Target: "http://agent-b.example:80/"}, obs.ReasonCoverage},
		"relative target":  {`sig1=(` + must + `)` + reqParams, Request{Method: "GET", Target: "/a"}, obs.ReasonCanonicalize},
	} {
		t.Run(name, func(t *testing.T) { wantReason(t, CheckRequest(parsedOf(t, c.input), c.r), c.want) })
	}
}

func TestNormalAuthority(t *testing.T) {
	for in, want := range map[string]bool{
		"https://a.example/":       true,
		"https://a.example:8443/":  true,
		"http://a.example:443/":    true,
		"https://a.example:80/":    true,
		"https://[::1]:8443/":      true,
		"https://[::1]/":           true,
		"https://a.example:443/":   false,
		"http://a.example:80/":     false,
		"https://A.example/":       false,
		"https://a.example:/":      false,
		"https://a.example:0/":     false,
		"https://a.example:08443/": false,
		"https://a.example:65536/": false,
		"https://a.example./":      false,
		"https://a.example.:8443/": false,
	} {
		tg, err := parseTarget(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if got := tg.normalAuthority(); got != want {
			t.Errorf("%s: %v", in, got)
		}
	}
}

func TestCheckResponse(t *testing.T) {
	req := Request{Method: "POST", Target: testTarget, Header: http.Header{
		"Signature": {"sig1=:AAAA:"}, "Content-Digest": {ContentDigest([]byte("q"))}}, Body: []byte("q")}
	resp := Response{Status: 200, Header: http.Header{}, Body: []byte("b")}
	const must = `"@status" "@method";req "@target-uri";req "@authority";req "content-digest";req "signature";req "content-digest"`
	const params = `;keyid="k";created=1`
	if err := CheckResponse(parsedOf(t, `sig1=(`+must+` "@query";req "@scheme";req "x-a";req "x-b")`+params), resp, req); err != nil {
		t.Fatalf("allowed set: %v", err)
	}
	drop := func(id string) string {
		return `sig1=(` + strings.TrimSpace(strings.Replace(must, id, "", 1)) + `)` + params
	}
	for name, c := range map[string]struct {
		input string
		want  obs.Reason
	}{
		"no keyid":              {`sig1=(` + must + `);created=1`, obs.ReasonParameters},
		"no created":            {`sig1=(` + must + `);keyid="k"`, obs.ReasonParameters},
		"no @status":            {drop(`"@status" `), obs.ReasonCoverage},
		"no @method;req":        {drop(`"@method";req `), obs.ReasonCoverage},
		"no @target-uri;req":    {drop(`"@target-uri";req `), obs.ReasonCoverage},
		"no @authority;req":     {drop(`"@authority";req `), obs.ReasonCoverage},
		"no content-digest;req": {drop(`"content-digest";req `), obs.ReasonCoverage},
		"no signature;req":      {drop(`"signature";req `), obs.ReasonCoverage},
		"no content-digest":     {`sig1=(` + strings.TrimSuffix(must, ` "content-digest"`) + `)` + params, obs.ReasonCoverage},
		"@method without ;req":  {`sig1=(` + must + ` "@method")` + params, obs.ReasonCoverage},
		"@status;req":           {`sig1=(` + must + ` "@status";req)` + params, obs.ReasonCoverage},
		"@path;req":             {`sig1=(` + must + ` "@path";req)` + params, obs.ReasonCoverage},
		";req=?0":               {`sig1=(` + must + ` "x";req=?0)` + params, obs.ReasonCoverage},
		";req=1":                {`sig1=(` + must + ` "x";req=1)` + params, obs.ReasonCoverage},
		";req;sf":               {`sig1=(` + must + ` "x";req;sf)` + params, obs.ReasonCoverage},
		"twice":                 {`sig1=(` + must + ` "@status")` + params, obs.ReasonCoverage},
		"upper-case field ;req": {`sig1=(` + must + ` "X";req)` + params, obs.ReasonCoverage},
	} {
		t.Run(name, func(t *testing.T) { wantReason(t, CheckResponse(parsedOf(t, c.input), resp, req), c.want) })
	}
	// Without a request body, Content-Digest and response body, the
	// content-digest forms are not required; "signature";req always is,
	// and an unsigned request fails closed (E03-20).
	signedGet := Request{Method: "GET", Target: testTarget, Header: http.Header{"Signature": {"sig1=:AAAA:"}}}
	if err := CheckResponse(parsedOf(t, `sig1=("@status" "@method";req "@target-uri";req "@authority";req "signature";req)`+params),
		Response{Status: 204}, signedGet); err != nil {
		t.Fatal(err)
	}
	if err := CheckResponse(parsedOf(t, `sig1=("@status" "@method";req "@target-uri";req "@authority";req)`+params),
		Response{Status: 204}, Request{Method: "GET", Target: testTarget}); !errors.Is(err, ErrUnsignedRequest) {
		t.Fatalf("unsigned request: %v", err)
	}
	bodyNoDigest := signedGet
	bodyNoDigest.Body = []byte("q")
	wantReason(t, CheckResponse(parsedOf(t, `sig1=("@status" "@method";req "@target-uri";req "@authority";req "signature";req)`+params),
		Response{Status: 204}, bodyNoDigest), obs.ReasonCoverage)
	bad := req
	bad.Target = "https://agent-b.example:443/"
	wantReason(t, CheckResponse(parsedOf(t, `sig1=(`+must+`)`+params), resp, bad), obs.ReasonCoverage)
}

// rawBase builds the base without the step 2 checks, to test step 3 alone.
func rawBase(p Parsed, resp *Response, r Request) ([]byte, error) {
	return buildBase(p.components, p.params, resp, r)
}

func TestBase(t *testing.T) {
	r := Request{Method: "GET", Target: "https://agent-b.example/a/b?x=1&y", Header: http.Header{
		"X-A": {" one ", "two\t"}, "X-Empty": {""}}}
	p := parsedOf(t, `sig1=("@method" "@target-uri" "@authority" "@scheme" "@query" "x-a" "x-empty")`+reqParams)
	base, err := rawBase(p, nil, r)
	if err != nil {
		t.Fatal(err)
	}
	want := "\"@method\": GET\n\"@target-uri\": https://agent-b.example/a/b?x=1&y\n\"@authority\": agent-b.example\n" +
		"\"@scheme\": https\n\"@query\": ?x=1&y\n\"x-a\": one, two\n\"x-empty\": \n" +
		`"@signature-params": ("@method" "@target-uri" "@authority" "@scheme" "@query" "x-a" "x-empty")` + reqParams
	if string(base) != want {
		t.Fatalf("base\n got %q\nwant %q", base, want)
	}
	r.Target = "https://agent-b.example/a"
	if base, _ := rawBase(parsedOf(t, `sig1=("@query")`+reqParams), nil, r); !strings.HasPrefix(string(base), "\"@query\": ?\n") {
		t.Fatalf("@query without a query: %q", base)
	}
	for name, c := range map[string]struct {
		input string
		r     Request
	}{
		"absent field":       {`sig1=("x-missing")` + reqParams, r},
		"two keys":           {`sig1=("x-a")` + reqParams, Request{Header: http.Header{"X-A": {"1"}, "x-a": {"2"}}}},
		"control character":  {`sig1=("x-a")` + reqParams, Request{Header: http.Header{"X-A": {"a\nb"}}}},
		"non-ascii value":    {`sig1=("x-a")` + reqParams, Request{Header: http.Header{"X-A": {"caf\xc3\xa9"}}}},
		"del in value":       {`sig1=("x-a")` + reqParams, Request{Header: http.Header{"X-A": {"a\x7f"}}}},
		"method not a token": {`sig1=("@method")` + reqParams, Request{Method: "G T"}},
		"bad target":         {`sig1=("@authority")` + reqParams, Request{Target: "a b"}},
		"control in target":  {`sig1=("@target-uri")` + reqParams, Request{Target: "https://a/\n"}},
		"@path":              {`sig1=("@path")` + reqParams, r},
		"@status in request": {`sig1=("@status")` + reqParams, r},
		";req in request":    {`sig1=("@method";req)` + reqParams, r},
		";sf":                {`sig1=("x-a";sf)` + reqParams, r},
	} {
		_, err := rawBase(parsedOf(t, c.input), nil, c.r)
		t.Run(name, func(t *testing.T) { wantReason(t, err, obs.ReasonCanonicalize) })
	}
	// In a response, @status needs a valid status and request components
	// need ;req.
	resp := Response{Status: 99}
	_, err = rawBase(parsedOf(t, `sig1=("@status")`+reqParams), &resp, r)
	wantReason(t, err, obs.ReasonCanonicalize)
	_, err = rawBase(parsedOf(t, `sig1=("@method")`+reqParams), &Response{Status: 200}, r)
	wantReason(t, err, obs.ReasonCanonicalize)
	_, err = rawBase(parsedOf(t, `sig1=("x-a")`+reqParams), &Response{Status: 200}, r)
	wantReason(t, err, obs.ReasonCanonicalize)
}

func TestVerifySignature(t *testing.T) {
	s, cs := testSigner(t, 1, vecDID)
	r := signed(t, s, testRequest(), Params{Created: 1, Nonce: "n-0123456789abcdef"})
	p, _ := Parse(r.Header, "", core.DefaultLimits())
	base, err := RequestBase(p, r)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifySignature(base, cs.Public()); err != nil {
		t.Fatal(err)
	}
	wantReason(t, VerifySignature(base, nil), obs.ReasonKeyNotEligible)
	wantReason(t, VerifySignature(Base{}, cs.Public()), obs.ReasonCanonicalize)
	wantReason(t, VerifySignature(Base{p: p, b: append(base.Bytes(), 'x')}, cs.Public()), obs.ReasonSignature)
	k1, err := crypto.NewSecp256k1Signer(secretScalar(t))
	if err != nil {
		t.Fatal(err)
	}
	// alg="ed25519" with a secp256k1 key (E01-11).
	wantReason(t, VerifySignature(base, k1.Public()), obs.ReasonAlgorithm)
	// An unknown alg is rejected at step 2.
	unknown := parsedOf(t, `sig1=("@method" "@target-uri" "@authority");keyid="k";alg="rsa-v1_5-sha256";created=1;nonce="n"`)
	wantReason(t, CheckRequest(unknown, Request{Method: "GET", Target: testTarget}), obs.ReasonAlgorithm)
	// A signature of the wrong length for the key is rejected.
	short := parsedOf(t, `sig1=("@method" "@target-uri" "@authority")`+reqParams)
	sb, err := RequestBase(short, Request{Method: "GET", Target: testTarget})
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifySignature(sb, cs.Public()); err == nil {
		t.Fatal("3-byte signature accepted")
	}
}

// TestH4 is the H4 probe: a signature that covers nothing, reused on
// another method, target and body, has no base and no verification path.
func TestH4(t *testing.T) {
	s, cs := testSigner(t, 1, vecDID)
	r := signed(t, s, testRequest(), Params{Created: 1, Nonce: "n-0123456789abcdef"})
	params := strings.TrimPrefix(r.Header.Get("Signature-Input"), "sig1=")
	params = params[strings.Index(params, ")")+1:]
	empty := `sig1=()` + params
	b, err := rawBase(parsedOf(t, empty), nil, r)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := cs.Sign(context.Background(), crypto.PurposeHTTPRequest, b)
	if err != nil {
		t.Fatal(err)
	}
	// The probe of sage-security.md H4: the empty-coverage signature on
	// another method, target and body, and on the request it was made for.
	h := http.Header{"Signature-Input": {empty}, "Signature": {"sig1=:" + base64.StdEncoding.EncodeToString(sig.Bytes()) + ":"}}
	for name, other := range map[string]Request{
		"other method, target and body": {Method: "DELETE", Target: "https://agent-b.example/admin", Header: h, Body: []byte(`{"b":2}`)},
		"other method and target":       {Method: "DELETE", Target: "https://agent-b.example/admin", Header: h},
		"same request":                  {Method: r.Method, Target: r.Target, Header: h, Body: r.Body},
	} {
		t.Run(name, func(t *testing.T) {
			p, err := Parse(other.Header, "", core.DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			_, err = RequestBase(p, other)
			wantReason(t, err, obs.ReasonCoverage)
		})
	}
}

func TestTargetBrackets(t *testing.T) {
	for _, in := range []string{"https://a]b.example/p", "https://[::1/p", "https://a.example/p[", "https://[::1]x/", "https://[]/", "https://[[::1]]/", "https://a.example/?q=]"} {
		if _, err := parseTarget(in); err == nil {
			t.Errorf("%s accepted", in)
		}
	}
	for _, in := range []string{"https://[::1]/p", "https://[::1]:8443/p?q=1"} {
		if _, err := parseTarget(in); err != nil {
			t.Errorf("%s: %v", in, err)
		}
	}
}
