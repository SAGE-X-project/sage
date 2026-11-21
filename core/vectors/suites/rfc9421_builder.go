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

package suites

import (
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

// sigSpec describes a received request for the reject vectors. The suite
// builds its signature base on its own (RFC 9421 §2.5), independently of
// httpsig, and signs with the Ed25519 key of key_label.
type sigSpec struct {
	method, url, body string
	fields            [][2]string // the field lines other than the signature fields
	covered           []string    // component identifiers
	sep               map[int]string
	params            string // ";keyid=...;..." as sent
	label             string
	input, signature  string // set to send fixed signature fields
	verifyAt          int64  // verification time; zero means reqCreated (E00-08)
	// es256k signs with the secp256k1 key of label (Keccak-256, 65-byte
	// r || s || v) instead of Ed25519; reencode, when set, changes the
	// signature bytes after signing.
	es256k   bool
	reencode func([]byte) []byte
	// prior holds received requests verified first, at the same verify_at
	// and with the same replay store, each of which must be accepted
	// (replay vectors).
	prior []map[string]any
}

func baseRequest() sigSpec {
	return sigSpec{
		method: reqMethod, url: reqURL, body: reqBody,
		fields: [][2]string{{"content-type", "application/json"}, {"date", reqDate},
			{"x-sage-did", reqDID}, {"content-digest", reqContentDigest}},
		covered: []string{`"@method"`, `"@target-uri"`, `"@authority"`, `"content-type"`, `"content-digest"`, `"x-sage-did"`, `"date"`},
		params:  `;keyid="` + reqKeyID + `";alg="ed25519";created=1788609600;nonce="` + reqNonce + `"`,
		label:   reqKeyLabel,
	}
}

func (s *sigSpec) drop(id string) {
	out := s.covered[:0:0]
	for _, c := range s.covered {
		if c != id {
			out = append(out, c)
		}
	}
	s.covered = out
}

func (s *sigSpec) replace(old, new string) {
	for i, c := range s.covered {
		if c == old {
			s.covered[i] = new
		}
	}
}

// member returns the Signature-Input member as sent.
func (s sigSpec) member() string {
	var b strings.Builder
	b.WriteString("sig1=(")
	for i, c := range s.covered {
		if i > 0 {
			sep, ok := s.sep[i-1]
			if !ok {
				sep = " "
			}
			b.WriteString(sep)
		}
		b.WriteString(c)
	}
	b.WriteString(")")
	b.WriteString(s.params)
	return b.String()
}

// base returns the signature base. Its @signature-params line is the
// canonical serialisation of the member (03 §2), whatever spacing member
// sends.
func (s sigSpec) base(resp *respSpec) string {
	var b strings.Builder
	canonical := func(x sigSpec) string {
		x.sep = nil
		return strings.TrimPrefix(x.member(), "sig1=")
	}
	covered, params := s.covered, canonical(s)
	if resp != nil {
		covered, params = resp.covered, canonical(resp.sigSpec)
	}
	for _, id := range covered {
		b.WriteString(id + ": " + s.value(id, resp) + "\n")
	}
	b.WriteString(`"@signature-params": ` + params)
	return b.String()
}

// value returns the component value of id (RFC 9421 §2.1, §2.2).
func (s sigSpec) value(id string, resp *respSpec) string {
	name, params, _ := strings.Cut(strings.TrimPrefix(id, `"`), `"`)
	fields := s.fields
	if resp != nil && !strings.Contains(params, ";req") {
		if name == "@status" {
			return fmt.Sprint(resp.status)
		}
		fields = resp.fields
	}
	scheme, rest, _ := strings.Cut(s.url, "://")
	authority, pathQuery, _ := strings.Cut(rest, "/")
	path, query, _ := strings.Cut("/"+pathQuery, "?")
	switch name {
	case "@method":
		return s.method
	case "@target-uri":
		return s.url
	case "@authority":
		return authority
	case "@scheme":
		return scheme
	case "@path":
		return path
	case "@query":
		return "?" + query
	case "@request-target":
		return "/" + pathQuery
	case "@query-param":
		want := strings.TrimSuffix(strings.TrimPrefix(params, `;name="`), `"`)
		for _, kv := range strings.Split(query, "&") {
			if k, v, _ := strings.Cut(kv, "="); k == want {
				return v
			}
		}
		return ""
	}
	var lines []string
	for _, f := range fields {
		if f[0] == name {
			lines = append(lines, strings.Trim(f[1], " \t"))
		}
	}
	return strings.Join(lines, ", ")
}

func (s sigSpec) sign() string {
	if !s.es256k {
		return edSign(s.label, s.base(nil))
	}
	var d secp256k1.ModNScalar
	d.SetByteSlice(labelSeed(s.label))
	compact := secpecdsa.SignCompact(secp256k1.NewPrivateKey(&d), keccak([]byte(s.base(nil))), false)
	sig := append(append([]byte(nil), compact[1:]...), compact[0]-27)
	if s.reencode != nil {
		sig = s.reencode(sig)
	}
	return "sig1=:" + base64.StdEncoding.EncodeToString(sig) + ":"
}

func edSign(label, base string) string {
	sig := ed25519.Sign(ed25519.NewKeyFromSeed(labelSeed(label)), []byte(base))
	return "sig1=:" + base64.StdEncoding.EncodeToString(sig) + ":"
}

// received returns the vector input of the request as received.
func (s sigSpec) received() map[string]any {
	input, signature := s.input, s.signature
	if input == "" {
		input, signature = s.member(), s.sign()
	}
	return receivedWith(s, input, signature)
}

// receivedWith returns the vector input of s with the given signature
// fields and extra field lines appended.
func receivedWith(s sigSpec, input, signature string, extra ...[2]string) map[string]any {
	fields := make([]any, 0, len(s.fields)+2+len(extra))
	for _, f := range s.fields {
		fields = append(fields, []any{f[0], f[1]})
	}
	fields = append(fields, []any{"signature-input", input}, []any{"signature", signature})
	for _, f := range extra {
		fields = append(fields, []any{f[0], f[1]})
	}
	at := s.verifyAt
	if at == 0 {
		at = reqCreated
	}
	in := map[string]any{
		"method": s.method, "url": s.url, "body": s.body, "fields": fields,
		"key_label": s.label, "verify_at": at,
	}
	if len(s.prior) > 0 {
		prior := make([]any, len(s.prior))
		for i, p := range s.prior {
			q := map[string]any{}
			for k, v := range p {
				q[k] = v
			}
			delete(q, "verify_at") // verified at the vector's verify_at
			prior[i] = q
		}
		in["prior"] = prior
	}
	return in
}

// respSpec describes a received response for the reject vectors. signFor
// is the request the response is signed for; request is the one it is
// checked against.
type respSpec struct {
	sigSpec
	status           int
	request, signFor sigSpec
}

func baseResponse() respSpec {
	req := baseRequest()
	req.fields = append(req.fields, [2]string{"signature-input", req.member()}, [2]string{"signature", req.sign()})
	return respSpec{
		sigSpec: sigSpec{
			body:   respBody,
			fields: [][2]string{{"content-type", "application/json"}, {"content-digest", respContentDigest}},
			covered: []string{`"@status"`, `"@method";req`, `"@target-uri";req`, `"@authority";req`,
				`"content-digest";req`, `"signature";req`, `"content-type"`, `"content-digest"`},
			params: `;keyid="` + respKeyID + `";alg="ed25519";created=1788609601;nonce="` + respNonce + `"`,
			label:  respKeyLabel,
		},
		status: 200, request: req, signFor: req,
	}
}

func (s respSpec) sign() string { return edSign(s.label, s.signFor.base(&s)) }

func (s respSpec) received() map[string]any {
	fields := make([]any, 0, len(s.fields)+2)
	for _, f := range s.fields {
		fields = append(fields, []any{f[0], f[1]})
	}
	fields = append(fields, []any{"signature-input", s.member()}, []any{"signature", s.sign()})
	req := s.request
	reqFields := req.fields[:len(req.fields)-2]
	in := receivedWith(sigSpec{method: req.method, url: req.url, body: req.body, fields: reqFields, label: req.label},
		req.fields[len(req.fields)-2][1], req.fields[len(req.fields)-1][1])
	delete(in, "key_label")
	delete(in, "verify_at")
	return map[string]any{"status": s.status, "body": s.body, "fields": fields, "request": in,
		"key_label": s.label, "verify_at": respCreated}
}

// respDID is the DID of respKeyID, another agent than the request's.
const respDID = "did:sage:ethereum:0x2222222222222222222222222222222222222222"

// setField replaces the value of the field line name.
func (s *sigSpec) setField(name, value string) {
	for i, f := range s.fields {
		if f[0] == name {
			s.fields[i][1] = value
			return
		}
	}
	s.fields = append(s.fields, [2]string{name, value})
}

// es256kRequest turns s into request-es256k (the secp256k1 key of
// labelSecp256k1A, alg="es256k") whose signature reencode changes.
func (s *sigSpec) es256kRequest(reencode func([]byte) []byte) {
	s.es256k, s.label, s.reencode = true, labelSecp256k1A, reencode
	s.params = strings.Replace(s.params, `;alg="ed25519"`, `;alg="es256k"`, 1)
}

func sha512Of(body string) string {
	sum := sha512.Sum512([]byte(body))
	return base64.StdEncoding.EncodeToString(sum[:])
}

// derOf re-encodes a 65-byte r || s || v signature as an ASN.1 DER
// SEQUENCE of r and s, dropping v.
func derOf(sig []byte) []byte {
	integer := func(b []byte) []byte {
		v := new(big.Int).SetBytes(b).Bytes()
		if len(v) == 0 || v[0]&0x80 != 0 {
			v = append([]byte{0}, v...)
		}
		return append([]byte{0x02, byte(len(v))}, v...)
	}
	body := append(integer(sig[:32]), integer(sig[32:64])...)
	return append([]byte{0x30, byte(len(body))}, body...)
}
