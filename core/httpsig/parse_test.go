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
	"bytes"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/obs"
)

// The signed request of the published vector rfc9421/request-ed25519.
const (
	vecSignatureInput = `sig1=("@method" "@target-uri" "@authority" "content-type" "content-digest" "x-sage-did" "date");keyid="did:sage:ethereum:0x1111111111111111111111111111111111111111#key-1";alg="ed25519";created=1788609600;nonce="8f0c5b1c-0d1a-4c8e-9a51-6d2e4b7f1a10"`
	vecSignature      = `sig1=:VO0yputjPcCYt/44t6UUG+tjhuZRPTG18gcSj9D+x42OFhGyt7pTc3hhMjqQKEvYsxl9KpD9nKEzoar80rGJAA==:`
	vecContentDigest  = `sha-256=:8LVpz99pqw3Qk15ZR9KrTw7BNYt9yrfJQf92GvVs0X0=:`
	vecDID            = "did:sage:ethereum:0x1111111111111111111111111111111111111111"
)

func header(input, sig string) http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	h.Set("Date", "Tue, 01 Sep 2026 12:00:00 GMT")
	h.Set("X-Sage-Did", vecDID)
	h.Set("Content-Digest", vecContentDigest)
	h.Set("Signature-Input", input)
	h.Set("Signature", sig)
	return h
}

func parse(h http.Header) (Parsed, error) { return Parse(h, "", core.DefaultLimits()) }

func wantReason(t *testing.T, err error, want obs.Reason) {
	t.Helper()
	var r *core.Rejection
	if !errors.As(err, &r) {
		t.Fatalf("error %v is not a core.Rejection", err)
	}
	if r.Reason() != want {
		t.Fatalf("reason %s, want %s", r.Reason(), want)
	}
	if err.Error() != core.AuthenticationFailedText {
		t.Fatalf("error text %q", err)
	}
}

func TestParsePublishedRequest(t *testing.T) {
	p, err := parse(header(vecSignatureInput, vecSignature))
	if err != nil {
		t.Fatal(err)
	}
	if p.Label() != "sig1" {
		t.Fatalf("label %q", p.Label())
	}
	var names []string
	for _, c := range p.Components() {
		names = append(names, c.Identifier())
	}
	if strings.Join(names, " ") != `"@method" "@target-uri" "@authority" "content-type" "content-digest" "x-sage-did" "date"` {
		t.Fatalf("components %v", names)
	}
	if got := "sig1=" + p.SignatureParams(); got != vecSignatureInput {
		t.Fatalf("@signature-params %q", got)
	}
	if k, ok := p.KeyID(); !ok || k != vecDID+"#key-1" {
		t.Fatalf("keyid %q", k)
	}
	if a, ok := p.Alg(); !ok || a != "ed25519" {
		t.Fatal("alg")
	}
	if c, ok := p.Created(); !ok || c != 1788609600 {
		t.Fatal("created")
	}
	if _, ok := p.Expires(); ok {
		t.Fatal("expires")
	}
	if n, ok := p.Nonce(); !ok || n != "8f0c5b1c-0d1a-4c8e-9a51-6d2e4b7f1a10" {
		t.Fatal("nonce")
	}
	if len(p.Signature()) != 64 {
		t.Fatalf("signature %d bytes", len(p.Signature()))
	}
	s := p.Signature()
	s[0] ^= 0xff
	if bytes.Equal(s, p.Signature()) {
		t.Fatal("Signature exposes internal state")
	}
	cs := p.Components()
	cs[0] = Component{}
	if p.Components()[0].Identifier() != `"@method"` {
		t.Fatal("Components exposes internal state")
	}
}

func TestComponentParameters(t *testing.T) {
	in := `sig1=("@status" "@method";req "content-digest";req "signature";req;key="sig1");keyid="k";created=1`
	p, err := parse(header(in, `sig1=:AAAA:`))
	if err != nil {
		t.Fatal(err)
	}
	cs := p.Components()
	if cs[1].Name() != "@method" || cs[1].Identifier() != `"@method";req` || !cs[1].Has("req") {
		t.Fatalf("%+v", cs[1])
	}
	if cs[3].Identifier() != `"signature";req;key="sig1"` {
		t.Fatalf("%q", cs[3].Identifier())
	}
	if cs[0].Has("req") {
		t.Fatal("Has")
	}
}

// TestRejects covers the errata §11 vectors of W-011 and the [0.4] rows.
func TestRejects(t *testing.T) {
	big := strings.Replace(vecSignatureInput, `nonce="`, `nonce="`+strings.Repeat("a", core.SpecMaxSignatureFieldBytes+1-len(vecSignatureInput)), 1)
	comps := "sig1=(" + strings.TrimSpace(strings.Repeat(`"a" `, core.SpecMaxCoveredComponents+1)) + `);keyid="k";created=1`
	bigSig := "sig1=:" + strings.Repeat("A", core.SpecMaxSignatureFieldBytes) + ":"
	cases := map[string]struct {
		h      http.Header
		reason obs.Reason
	}{
		"tag-present":                    {header(vecSignatureInput+`;tag="x"`, vecSignature), obs.ReasonParameters},
		"unknown-param":                  {header(vecSignatureInput+`;foo=1`, vecSignature), obs.ReasonParameters},
		"noncanonical-params":            {header(strings.Replace(vecSignatureInput, `"@method" "@target-uri"`, `"@method"  "@target-uri"`, 1), vecSignature), obs.ReasonNonCanonical},
		"two-labels":                     {header(vecSignatureInput+`, sig2=("@method");keyid="k";created=1`, vecSignature+`, sig2=:AAAA:`), obs.ReasonDuplicate},
		"label-sets-differ":              {header(vecSignatureInput, strings.Replace(vecSignature, "sig1=", "sig2=", 1)), obs.ReasonMalformed},
		"repeated-label":                 {header(vecSignatureInput+", "+vecSignatureInput, vecSignature), obs.ReasonDuplicate},
		"repeated-sig-label":             {header(vecSignatureInput, vecSignature+", "+vecSignature), obs.ReasonDuplicate},
		"signature-input-too-large":      {header(big, vecSignature), obs.ReasonTooLarge},
		"signature-too-large":            {header(vecSignatureInput, bigSig), obs.ReasonTooLarge},
		"33 components":                  {header(comps, vecSignature), obs.ReasonTooLarge},
		"duplicate param":                {header(vecSignatureInput+`;created=2`, vecSignature), obs.ReasonDuplicate},
		"created not integer":            {header(strings.Replace(vecSignatureInput, "created=1788609600", `created="1788609600"`, 1), vecSignature), obs.ReasonMalformed},
		"created negative":               {header(strings.Replace(vecSignatureInput, "created=1788609600", "created=-1", 1), vecSignature), obs.ReasonMalformed},
		"created decimal":                {header(strings.Replace(vecSignatureInput, "created=1788609600", "created=1.5", 1), vecSignature), obs.ReasonMalformed},
		"created leading zero":           {header(strings.Replace(vecSignatureInput, "created=1788609600", "created=01788609600", 1), vecSignature), obs.ReasonNonCanonical},
		"keyid boolean":                  {header(strings.Replace(vecSignatureInput, `keyid="did:sage:ethereum:0x1111111111111111111111111111111111111111#key-1"`, "keyid", 1), vecSignature), obs.ReasonMalformed},
		"item not inner list":            {header(`sig1="@method";keyid="k";created=1`, vecSignature), obs.ReasonMalformed},
		"component not string":           {header(`sig1=(@method);keyid="k";created=1`, vecSignature), obs.ReasonMalformed},
		"signature not bytes":            {header(vecSignatureInput, `sig1="AAAA"`), obs.ReasonMalformed},
		"signature with param":           {header(vecSignatureInput, vecSignature+`;x=1`), obs.ReasonUnknownMember},
		"signature not canonical base64": {header(vecSignatureInput, `sig1=:AAB=:`), obs.ReasonNonCanonical},
		"truncated signature":            {header(vecSignatureInput, `sig1=:`), obs.ReasonMalformed},
		"truncated input":                {header(`sig1=();keyid="`, vecSignature), obs.ReasonMalformed},
		"uppercase key":                  {header(strings.Replace(vecSignatureInput, "sig1=", "Sig1=", 1), vecSignature), obs.ReasonMalformed},
		"bad escape":                     {header(strings.Replace(vecSignatureInput, `keyid="`, `keyid="\n`, 1), vecSignature), obs.ReasonMalformed},
		"non-ascii string":               {header(strings.Replace(vecSignatureInput, `keyid="`, `keyid="é`, 1), vecSignature), obs.ReasonMalformed},
		"trailing comma":                 {header(vecSignatureInput+",", vecSignature), obs.ReasonMalformed},
		"empty input":                    {header("", vecSignature), obs.ReasonMalformed},
	}
	for name, c := range cases {
		_, err := parse(c.h)
		if err == nil {
			t.Errorf("%s: accepted", name)
			continue
		}
		t.Run(name, func(t *testing.T) { wantReason(t, err, c.reason) })
	}
}

func TestFieldLines(t *testing.T) {
	for _, name := range []string{"Signature-Input", "Signature", "Content-Digest", "X-Sage-Did"} {
		h := header(vecSignatureInput, vecSignature)
		h.Add(name, h.Get(name))
		_, err := parse(h)
		t.Run("two "+name+" lines", func(t *testing.T) { wantReason(t, err, obs.ReasonDuplicate) })
	}
	for _, name := range []string{"Signature-Input", "Signature"} {
		h := header(vecSignatureInput, vecSignature)
		h.Del(name)
		_, err := parse(h)
		t.Run("no "+name, func(t *testing.T) { wantReason(t, err, obs.ReasonMalformed) })
	}
	h := header(vecSignatureInput, vecSignature)
	h.Del("Content-Digest")
	h.Del("X-Sage-Did")
	if _, err := parse(h); err != nil {
		t.Fatalf("Content-Digest and X-SAGE-DID are optional here: %v", err)
	}
}

func TestHeaderSection(t *testing.T) {
	h := header(vecSignatureInput, vecSignature)
	fill := core.SpecMaxHeaderBytes - SectionSize(h) - len("X-Fill: \r\n")
	h.Set("X-Fill", strings.Repeat("a", fill))
	if SectionSize(h) != core.SpecMaxHeaderBytes {
		t.Fatalf("section %d", SectionSize(h))
	}
	if _, err := parse(h); err != nil {
		t.Fatalf("a 64 KiB header section: %v", err)
	}
	h.Set("X-Fill", strings.Repeat("a", fill+1))
	_, err := parse(h)
	wantReason(t, err, obs.ReasonTooLarge)
}

// TestNamedLabel: with a named label the other labels are ignored (E03-06).
func TestNamedLabel(t *testing.T) {
	in := `other=("@method");keyid="k";created=1, ` + vecSignatureInput
	sig := `other=:AAAA:, ` + vecSignature
	p, err := Parse(header(in, sig), "sig1", core.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	if p.Label() != "sig1" || "sig1="+p.SignatureParams() != vecSignatureInput {
		t.Fatal("named label")
	}
	if _, err := Parse(header(in, sig), "", core.DefaultLimits()); err == nil {
		t.Fatal("two labels without a named label")
	}
	_, err = Parse(header(in, sig), "missing", core.DefaultLimits())
	wantReason(t, err, obs.ReasonMalformed)
	// Other members are ignored whatever they hold: a repeated label,
	// repeated parameters, many members (E03-06).
	many := strings.Repeat(`o=("a");keyid="k", `, 100)
	for _, others := range []string{`o=("a");keyid="k", o=("b");keyid="k", `, `o;x;x, `, many} {
		if _, err := Parse(header(others+vecSignatureInput, `o=:AAAA:, o=:AAAA:, `+vecSignature), "sig1", core.DefaultLimits()); err != nil {
			t.Errorf("%.40q...: %v", others, err)
		}
	}
	// The named label itself must occur once in each dictionary.
	_, err = Parse(header(vecSignatureInput+", "+vecSignatureInput, vecSignature), "sig1", core.DefaultLimits())
	wantReason(t, err, obs.ReasonDuplicate)
	_, err = Parse(header(vecSignatureInput, vecSignature+", "+vecSignature), "sig1", core.DefaultLimits())
	wantReason(t, err, obs.ReasonDuplicate)
	// The named member must still be canonical, and the others must parse.
	_, err = Parse(header(`other=(bad, `+vecSignatureInput, sig), "sig1", core.DefaultLimits())
	wantReason(t, err, obs.ReasonMalformed)
	_, err = Parse(header("bad-Key=1, "+vecSignatureInput, sig), "sig1", core.DefaultLimits())
	wantReason(t, err, obs.ReasonMalformed)
}

func TestLimitsValidated(t *testing.T) {
	if _, err := Parse(header(vecSignatureInput, vecSignature), "", core.Limits{}); !errors.Is(err, core.ErrLimits) {
		t.Fatalf("%v", err)
	}
}

// TestLinear checks that parsing time grows linearly with the field size
// (H2: the legacy parser was quadratic). The limit is bypassed to measure.
func TestLinear(t *testing.T) {
	member := func(n int) string {
		return `sig1=(` + strings.TrimSpace(strings.Repeat(`"x-aaaaaaaaaaaaaaaaaaaaaaaaa" `, n)) + `);keyid="k";created=1`
	}
	small, large := member(256), member(2048) // about 8 KiB and 64 KiB
	measure := func(s string) time.Duration {
		best := time.Duration(1 << 62)
		for i := 0; i < 5; i++ {
			start := time.Now()
			if _, err := parseDictionary(s); err != nil {
				t.Fatal(err)
			}
			if d := time.Since(start); d < best {
				best = d
			}
		}
		return best
	}
	ratio := float64(measure(large)) / float64(measure(small))
	t.Logf("8x input, %.1fx time", ratio)
	if ratio > 24 { // linear is about 8; quadratic about 64
		t.Fatalf("parsing time grew %.1fx for 8x input", ratio)
	}
	// Allocated bytes per input byte stay flat too (design §9.2): a
	// string built up per rune would grow them with the input.
	allocated := func(s string) float64 {
		return float64(testing.Benchmark(func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				if _, err := parseDictionary(s); err != nil {
					b.Fatal(err)
				}
			}
		}).AllocedBytesPerOp()) / float64(len(s))
	}
	perSmall, perLarge := allocated(small), allocated(large)
	t.Logf("allocated bytes per input byte: %.1f at 8 KiB, %.1f at 64 KiB", perSmall, perLarge)
	if perLarge > 2*perSmall {
		t.Fatalf("allocation per byte grew from %.1f to %.1f", perSmall, perLarge)
	}
}

func FuzzParse(f *testing.F) {
	f.Add(vecSignatureInput, vecSignature)
	f.Add(`sig1=();keyid="`, `sig1=:`) // the H1 inputs: truncated Signature-Input and Signature
	f.Add(`sig1=:`, `sig1=();keyid="`)
	f.Add(`a=("x";p=?1 "y";q=:AA==:);k=1.5;t=tok, b=?0`, `a=:AA==:`)
	f.Fuzz(func(t *testing.T, input, sig string) {
		h := http.Header{"Signature-Input": {input}, "Signature": {sig}}
		p, err := Parse(h, "", core.DefaultLimits())
		if err != nil {
			var r *core.Rejection
			if !errors.As(err, &r) {
				t.Fatalf("non-rejection error %v", err)
			}
			return
		}
		// Steps 2 and 3 on an accepted signature return rejections, never
		// panic.
		req := Request{Method: "POST", Target: "https://a.example/p?q", Header: h, Body: []byte("b")}
		rb, rerr := RequestBase(p, req)
		sb, serr := ResponseBase(p, Response{Status: 200, Header: h}, req)
		raw, berr := rawBase(p, nil, req)
		rawResp, rerr2 := rawBase(p, &Response{Status: 200, Header: h}, req)
		for _, err := range []error{
			CheckRequest(p, req),
			CheckResponse(p, Response{Status: 200, Header: h, Body: []byte("b")}, req),
			rerr, serr, berr, rerr2,
		} {
			var r *core.Rejection
			if err != nil && !errors.As(err, &r) && !errors.Is(err, ErrUnsignedRequest) {
				t.Fatalf("non-rejection error %v", err)
			}
		}
		// A base has one line per covered component and no other control
		// byte: no value can add a line (RFC 9421 §2.5).
		for _, b := range [][]byte{rb.b, sb.b, raw, rawResp} {
			if b == nil {
				continue
			}
			if n := bytes.Count(b, []byte("\n")); n != len(p.components) {
				t.Fatalf("base has %d component lines for %d components: %q", n, len(p.components), b)
			}
			if unsafeValue(strings.ReplaceAll(string(b), "\n", "")) {
				t.Fatalf("base has a control or non-ASCII byte: %q", b)
			}
		}
		// Accepted input is canonical: the member re-serialises to itself.
		// RFC 8941 discards leading SP and the OWS after the last member.
		if got := p.Label() + "=" + p.SignatureParams(); got != strings.TrimRight(strings.TrimLeft(input, " "), " \t") {
			t.Fatalf("accepted %q, re-serialised %q", input, got)
		}
	})
}

// TestSurroundingWhitespace follows RFC 8941 §4.2 and §4.2.2: leading SP is
// discarded, OWS (SP or HTAB) after the last member is discarded, and a
// leading HTAB is invalid.
func TestSurroundingWhitespace(t *testing.T) {
	for _, in := range []string{"  " + vecSignatureInput, vecSignatureInput + " \t "} {
		if _, err := parse(header(in, vecSignature)); err != nil {
			t.Errorf("%q: %v", in, err)
		}
	}
	_, err := parse(header("\t"+vecSignatureInput, vecSignature))
	wantReason(t, err, obs.ReasonMalformed)
}

func TestAcceptedForms(t *testing.T) {
	exact := strings.Replace(vecSignatureInput, `nonce="`, `nonce="`+strings.Repeat("a", core.SpecMaxSignatureFieldBytes-len(vecSignatureInput)), 1)
	comps := "sig1=(" + strings.TrimSpace(strings.Repeat(`"a" `, core.SpecMaxCoveredComponents)) + `);keyid="k";created=1`
	for name, in := range map[string]string{
		"8 KiB Signature-Input":   exact,
		"32 components":           comps,
		"parameters out of order": `sig1=("@method");created=1;nonce="n";keyid="k"`,
		"expires, no alg":         `sig1=("@method");keyid="k";created=1;expires=2`,
		"escaped keyid":           `sig1=("@method");keyid="a\"b\\c";created=1`,
		"no components":           `sig1=();keyid="k";created=1`,
	} {
		p, err := parse(header(in, vecSignature))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if name == "escaped keyid" {
			if k, _ := p.KeyID(); k != `a"b\c` {
				t.Errorf("escaped keyid decoded to %q", k)
			}
		}
		if name == "expires, no alg" {
			if e, ok := p.Expires(); !ok || e != 2 {
				t.Error("expires")
			}
			if _, ok := p.Alg(); ok {
				t.Error("alg")
			}
		}
	}
	// The longest canonical Signature value within 8 KiB: base64 length is
	// a multiple of 4, so it is 8191 bytes.
	n := (core.SpecMaxSignatureFieldBytes - len("sig1=::")) / 4 * 4
	sig := "sig1=:" + strings.Repeat("A", n) + ":"
	if len(sig) != core.SpecMaxSignatureFieldBytes-1 {
		t.Fatalf("signature field %d bytes", len(sig))
	}
	if _, err := parse(header(vecSignatureInput, sig)); err != nil {
		t.Fatalf("%d-byte Signature: %v", len(sig), err)
	}
}

// TestUnpaddedSignature: RFC 8941 parsers should not fail on missing
// padding, but the verified signature must be canonical (DP-2.1).
func TestUnpaddedSignature(t *testing.T) {
	unpadded := strings.TrimSuffix(strings.TrimSuffix(vecSignature, ":"), "==") + ":"
	_, err := parse(header(vecSignatureInput, unpadded))
	wantReason(t, err, obs.ReasonNonCanonical)
	// In an ignored member it is fine.
	if _, err := Parse(header(vecSignatureInput, `o=:AAA:, `+vecSignature), "sig1", core.DefaultLimits()); err != nil {
		t.Fatalf("unpadded ignored member: %v", err)
	}
}

// TestNonCanonicalHeaderKeys: a header map built without net/http's
// canonical keys cannot hide a second field line (E03-07).
func TestNonCanonicalHeaderKeys(t *testing.T) {
	h := header(vecSignatureInput, vecSignature)
	h["signature-input"] = []string{vecSignatureInput}
	_, err := parse(h)
	wantReason(t, err, obs.ReasonDuplicate)
	h = header(vecSignatureInput, vecSignature)
	h["x-sage-did"] = []string{vecDID}
	_, err = parse(h)
	wantReason(t, err, obs.ReasonDuplicate)
}
