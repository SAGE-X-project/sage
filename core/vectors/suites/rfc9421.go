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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/clock"
	"github.com/sage-x-project/sage/core/crypto"
	"github.com/sage-x-project/sage/core/did"
	"github.com/sage-x-project/sage/core/httpsig"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/replay"
	"github.com/sage-x-project/sage/core/secret"
	"github.com/sage-x-project/sage/core/vectors"
)

// RFC9421Description is the description of the rfc9421 suite file,
// unchanged from draft.1.
const RFC9421Description = "HTTP Message Signatures profile: covered components, signature base, Signature-Input/Signature headers, Content-Digest (sha-256), keyid = DID#fragment, and response signatures bound to the request with ;req."

// The published vector rfc9421/request-ed25519 and its response.
const (
	reqMethod         = "POST"
	reqURL            = "https://agent-b.example/mcp/tools/call"
	reqBody           = `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"echo","arguments":{"text":"hello"}}}`
	reqKeyLabel       = "sage-spec/vectors/ed25519/agent-a"
	reqKeyID          = "did:sage:ethereum:0x1111111111111111111111111111111111111111#key-1"
	reqDID            = "did:sage:ethereum:0x1111111111111111111111111111111111111111"
	reqNonce          = "8f0c5b1c-0d1a-4c8e-9a51-6d2e4b7f1a10"
	reqDate           = "Tue, 01 Sep 2026 12:00:00 GMT"
	reqCreated        = 1788609600
	reqSignatureInput = `sig1=("@method" "@target-uri" "@authority" "content-type" "content-digest" "x-sage-did" "date");keyid="did:sage:ethereum:0x1111111111111111111111111111111111111111#key-1";alg="ed25519";created=1788609600;nonce="8f0c5b1c-0d1a-4c8e-9a51-6d2e4b7f1a10"`
	reqSignature      = `sig1=:VO0yputjPcCYt/44t6UUG+tjhuZRPTG18gcSj9D+x42OFhGyt7pTc3hhMjqQKEvYsxl9KpD9nKEzoar80rGJAA==:`
	reqContentDigest  = `sha-256=:8LVpz99pqw3Qk15ZR9KrTw7BNYt9yrfJQf92GvVs0X0=:`

	respBody          = `{"jsonrpc":"2.0","id":7,"result":{"content":[{"type":"text","text":"hello"}]}}`
	respKeyLabel      = "sage-spec/vectors/ed25519/agent-b"
	respKeyID         = "did:sage:ethereum:0x2222222222222222222222222222222222222222#key-1"
	respNonce         = reqNonce + "-resp"
	respCreated       = reqCreated + 1
	respContentDigest = `sha-256=:RYDXwjjrCO/NMhfknKXVnCd+5BTGUptdGDRgWfgFWVc=:`
	respSignature     = `sig1=:9RhFjsLFirHiVYgTKvF3k3rAzD3j//L1Ev6LuFtzQd0WdrOUr5661LAliAkIU/7R+loapmepfaYQPMp8l+/IDA==:`
)

// RFC9421 returns the rfc9421 suite: the four published vectors, the
// errata §11 reject vectors for 03, each rejected at one step of 03 §5, and
// digest-mismatch-keeps-nonce. Every message is verified with every step of
// 03 §5 by httpsig.Verifier, whose registry (labelResolver) lists the
// key_label key for keyid's DID, at the fixed time verify_at (E00-08);
// request-ecdsa-p256-sha256, whose key type the registry cannot list, is
// checked by steps 1, 2, 3, 5, 6 and 7.
func RFC9421() vectors.Suite {
	cases := append(rfc9421Published(), rfc9421Rejects()...)
	return vectors.Suite{Name: "rfc9421", Description: RFC9421Description,
		Cases: append(cases, rfc9421KeepsNonce())}
}

func rfc9421Published() []vectors.Case {
	request := func(alg, label string) map[string]any {
		return map[string]any{
			"alg": alg, "body": reqBody,
			"covered":   []any{`"@method"`, `"@target-uri"`, `"@authority"`, `"content-type"`, `"content-digest"`, `"x-sage-did"`, `"date"`},
			"created":   reqCreated,
			"headers":   map[string]any{"content-type": "application/json", "date": reqDate, "x-sage-did": reqDID},
			"key_label": label, "keyid": reqKeyID, "method": reqMethod, "nonce": reqNonce,
			"signature_name": "sig1", "url": reqURL,
		}
	}
	desc := func(alg string) string {
		return "Signed POST request with alg=" + alg + ". Verify at the fixed clock now = created, nonce required, ExpectedDID bound to keyid."
	}
	return []vectors.Case{
		{Name: "request-ed25519", Mode: vectors.ModeDeterministic, Description: desc("ed25519"),
			Input: request("ed25519", labelEd25519A), Produce: produceRequest, Verify: verifyRequest},
		{Name: "request-es256k", Mode: vectors.ModeDeterministic, Description: desc("es256k"),
			Input: request("es256k", labelSecp256k1A), Produce: produceRequest, Verify: verifyRequest},
		{Name: "request-ecdsa-p256-sha256", Mode: vectors.ModeVerify, Description: desc("ecdsa-p256-sha256"),
			Input: request("ecdsa-p256-sha256", labelP256A), Produce: produceRequest, Verify: verifyRequest},
		{Name: "response-ed25519", Mode: vectors.ModeDeterministic,
			Description: "Response signed by agent B over @status plus the request's @method/@target-uri/@authority/content-digest/signature with ;req, and the response content-type/content-digest. The request carries the ed25519 request signature vector.",
			Input: map[string]any{
				"alg": "ed25519", "body": respBody,
				"covered":   []any{`"@status"`, `"@method";req`, `"@target-uri";req`, `"@authority";req`, `"content-digest";req`, `"signature";req`, `"content-type"`, `"content-digest"`},
				"created":   respCreated,
				"headers":   map[string]any{"content-type": "application/json"},
				"key_label": labelEd25519B, "keyid": respKeyID, "nonce": respNonce,
				"request":        request("ed25519", labelEd25519A),
				"signature_name": "sig1", "status": 200,
			},
			Produce: produceResponse, Verify: verifyResponse},
	}
}

// The published vectors are signed by httpsig.Signer and verified by every
// step of 03 §5, with ExpectedDID bound to keyid's DID and the key_label key
// resolved for it, at the verification time now = created (E00-08). For
// response-ed25519 that is its own created, the request's created + 1.

func produceRequest(in map[string]any) (map[string]any, error) {
	r, s, p, err := signedMessage(in)
	if err != nil {
		return nil, err
	}
	f, err := s.SignRequest(context.Background(), r, p)
	if err != nil {
		return nil, err
	}
	f.Apply(r.Header)
	parsed, err := httpsig.Parse(r.Header, "sig1", core.DefaultLimits())
	if err != nil {
		return nil, err
	}
	if err := sameCovered(in, parsed); err != nil {
		return nil, err
	}
	base, err := httpsig.RequestBase(parsed, r)
	if err != nil {
		return nil, err
	}
	return map[string]any{"content_digest": f.ContentDigest, "signature": f.Signature,
		"signature_base": string(base.Bytes()), "signature_input": f.SignatureInput}, nil
}

func verifyRequest(in, out map[string]any) error {
	r, _, _, err := signedMessage(in)
	if err != nil {
		return err
	}
	if err := setOutputFields(r.Header, out, "signature_input", "signature", "content_digest"); err != nil {
		return err
	}
	if r.Header.Get(httpsig.FieldContentDigest) != httpsig.ContentDigest(r.Body) {
		return errors.New("content_digest is not the digest of the body")
	}
	created, err := integer(in["created"])
	if err != nil {
		return err
	}
	var base []byte
	if alg, _ := in["alg"].(string); alg == crypto.AlgP256.String() {
		base, err = checkP256Request(r, in, created)
	} else {
		var v *httpsig.Verifier
		var e httpsig.Expect
		if v, err = verifierAt(created, in["key_label"].(string)); err != nil {
			return err
		}
		if e, err = expectKeyID(in); err != nil {
			return err
		}
		base, err = checkRequest(r, v, e)
	}
	if err != nil {
		return err
	}
	if string(base) != out["signature_base"] {
		return errors.New("signature_base differs from the rebuilt base")
	}
	return nil
}

func produceResponse(in map[string]any) (map[string]any, error) {
	reqIn, ok := in["request"].(map[string]any)
	if !ok {
		return nil, errors.New("request must be an object")
	}
	reqOut, err := produceRequest(reqIn)
	if err != nil {
		return nil, err
	}
	req, _, _, err := signedMessage(reqIn)
	if err != nil {
		return nil, err
	}
	if err := setOutputFields(req.Header, reqOut, "signature_input", "signature", "content_digest"); err != nil {
		return nil, err
	}
	resp, s, p, err := signedResponse(in)
	if err != nil {
		return nil, err
	}
	f, err := s.SignResponse(context.Background(), resp, req, p)
	if err != nil {
		return nil, err
	}
	f.Apply(resp.Header)
	parsed, err := httpsig.Parse(resp.Header, "sig1", core.DefaultLimits())
	if err != nil {
		return nil, err
	}
	if err := sameCovered(in, parsed); err != nil {
		return nil, err
	}
	base, err := httpsig.ResponseBase(parsed, resp, req)
	if err != nil {
		return nil, err
	}
	return map[string]any{"content_digest": f.ContentDigest,
		"request_signature": reqOut["signature"], "request_signature_input": reqOut["signature_input"],
		"signature": f.Signature, "signature_base": string(base.Bytes()), "signature_input": f.SignatureInput}, nil
}

func verifyResponse(in, out map[string]any) error {
	reqIn, ok := in["request"].(map[string]any)
	if !ok {
		return errors.New("request must be an object")
	}
	req, _, _, err := signedMessage(reqIn)
	if err != nil {
		return err
	}
	if err := setOutputFields(req.Header, map[string]any{"signature_input": out["request_signature_input"],
		"signature": out["request_signature"], "content_digest": httpsig.ContentDigest(req.Body)},
		"signature_input", "signature", "content_digest"); err != nil {
		return err
	}
	resp, _, _, err := signedResponse(in)
	if err != nil {
		return err
	}
	if err := setOutputFields(resp.Header, out, "signature_input", "signature", "content_digest"); err != nil {
		return err
	}
	created, err := integer(in["created"])
	if err != nil {
		return err
	}
	label, _ := in["key_label"].(string)
	v, err := verifierAt(created, label)
	if err != nil {
		return err
	}
	e, err := expectKeyID(in)
	if err != nil {
		return err
	}
	base, err := checkResponse(resp, req, v, e)
	if err != nil {
		return err
	}
	if string(base) != out["signature_base"] {
		return errors.New("signature_base differs from the rebuilt base")
	}
	return nil
}

// labelResolver is the registry of the rfc9421 vectors: every DID resolves
// to an active record that lists exactly the key of key_label, with its §4
// proof of possession for that DID, observed at the verification time. So
// steps 4 and 6 of 03 §5 verify with the key_label key resolved for keyid's
// DID.
type labelResolver struct {
	signer crypto.Signer
	clock  clock.Clock
}

func (r labelResolver) Resolve(ctx context.Context, d did.DID) (did.Record, did.Observation, error) {
	data := r.signer.Public().Bytes()
	proof, err := crypto.SignPoP(ctx, r.signer, d.AsSent(), data)
	if err != nil {
		return did.Record{}, did.Observation{}, err
	}
	typ := did.KeyTypeEd25519
	if r.signer.Public().Alg() == crypto.AlgES256K {
		typ = did.KeyTypeECDSA
	}
	now, err := r.clock.Now()
	if err != nil {
		return did.Record{}, did.Observation{}, err
	}
	rec := did.Record{DID: d, IsActive: true, Keys: []did.Key{{Type: typ, KeyData: data, Signature: proof.Bytes(), Verified: true}}}
	return rec, did.Observation{At: now, Source: "vectors"}, nil
}

// verifierAt returns a verifier with the default limits at the
// verification time at (Unix seconds), whose replay store was restored an
// hour earlier, so the restore rule of E03-12 does not apply, and whose
// registry is the labelResolver of key_label.
func verifierAt(at int64, label string) (*httpsig.Verifier, error) {
	cs, err := labelSigner(label)
	if err != nil {
		return nil, err
	}
	now := time.Unix(at, 0)
	c := clock.NewFixed(now.Add(-time.Hour))
	store, err := replay.NewMemStore(c, replay.Capacity{PerOwner: 8, PerScope: 8, Total: 8})
	if err != nil {
		return nil, err
	}
	c.Set(now)
	s, err := core.NewSettings(core.WithSpecVersion(string(core.V0_9_0_Draft1_1)))
	if err != nil {
		return nil, err
	}
	return httpsig.NewVerifier(c, s, store, labelResolver{signer: cs, clock: c})
}

// checkRequest runs every step of 03 §5 on a received request with v and
// the expectation e, and returns the rebuilt base.
func checkRequest(r httpsig.Request, v *httpsig.Verifier, e httpsig.Expect) ([]byte, error) {
	p, err := httpsig.Parse(r.Header, "", core.DefaultLimits())
	if err != nil {
		return nil, err
	}
	base, err := httpsig.RequestBase(p, r)
	if err != nil {
		return nil, err
	}
	if _, err := v.VerifyRequest(context.Background(), r, e); err != nil {
		return nil, err
	}
	return base.Bytes(), nil
}

// checkResponse runs every step of 03 §5 on a received response for req,
// which reserves no nonce for a response (00-overview.md §7.4 DP-4), and
// returns the rebuilt base.
func checkResponse(resp httpsig.Response, req httpsig.Request, v *httpsig.Verifier, e httpsig.Expect) ([]byte, error) {
	p, err := httpsig.Parse(resp.Header, "", core.DefaultLimits())
	if err != nil {
		return nil, err
	}
	base, err := httpsig.ResponseBase(p, resp, req)
	if err != nil {
		return nil, err
	}
	if _, err := v.VerifyResponse(context.Background(), resp, req, e); err != nil {
		return nil, err
	}
	return base.Bytes(), nil
}

// checkP256Request checks request-ecdsa-p256-sha256, whose key type the
// registry cannot list (06 §3: types 0, 1 and 2), so it cannot be resolved
// and verified by the Verifier: steps 1, 2, 3, 5, 6 (with the key_label
// key) and 7 of 03 §5.
func checkP256Request(r httpsig.Request, in map[string]any, at int64) ([]byte, error) {
	key, err := labelKey(in)
	if err != nil {
		return nil, err
	}
	p, err := httpsig.Parse(r.Header, "", core.DefaultLimits())
	if err != nil {
		return nil, err
	}
	base, err := httpsig.RequestBase(p, r)
	if err != nil {
		return nil, err
	}
	e, err := expectKeyID(in)
	if err != nil {
		return nil, err
	}
	if _, err := httpsig.CheckSigner(p, r.Header, e); err != nil {
		return nil, err
	}
	c := clock.NewFixed(time.Unix(at, 0))
	store, err := replay.NewMemStore(c, replay.Capacity{PerOwner: 1, PerScope: 1, Total: 1})
	if err != nil {
		return nil, err
	}
	g, err := httpsig.NewGuard(c, core.DefaultLimits(), store)
	if err != nil {
		return nil, err
	}
	if err := g.CheckTime(p); err != nil {
		return nil, err
	}
	sig, err := crypto.ParseSignature(key.Alg(), p.Signature())
	if err != nil {
		return nil, err
	}
	if err := crypto.Verify(key, sig, base.Bytes()); err != nil {
		return nil, err
	}
	if err := httpsig.CheckDigest(r.Header, r.Body, core.DefaultLimits()); err != nil {
		return nil, err
	}
	return base.Bytes(), nil
}

// expectKeyID returns the expectation of the published request vectors:
// ExpectedDID bound to keyid's DID.
func expectKeyID(in map[string]any) (httpsig.Expect, error) {
	keyid, _ := in["keyid"].(string)
	ref, err := did.ParseKeyRef(keyid)
	if err != nil {
		return httpsig.Expect{}, err
	}
	return httpsig.Expect{DID: ref.DID()}, nil
}

// signedMessage reads the request and the signer of a published input.
func signedMessage(in map[string]any) (httpsig.Request, *httpsig.Signer, httpsig.Params, error) {
	h, err := headerObject(in["headers"])
	if err != nil {
		return httpsig.Request{}, nil, httpsig.Params{}, err
	}
	method, ok1 := in["method"].(string)
	url, ok2 := in["url"].(string)
	body, ok3 := in["body"].(string)
	if !ok1 || !ok2 || !ok3 {
		return httpsig.Request{}, nil, httpsig.Params{}, errors.New("method, url and body must be strings")
	}
	s, p, err := inputSigner(in)
	return httpsig.Request{Method: method, Target: url, Header: h, Body: []byte(body)}, s, p, err
}

func signedResponse(in map[string]any) (httpsig.Response, *httpsig.Signer, httpsig.Params, error) {
	h, err := headerObject(in["headers"])
	if err != nil {
		return httpsig.Response{}, nil, httpsig.Params{}, err
	}
	body, ok := in["body"].(string)
	status, err2 := integer(in["status"])
	if !ok || err2 != nil {
		return httpsig.Response{}, nil, httpsig.Params{}, errors.New("body must be a string and status an integer")
	}
	s, p, err := inputSigner(in)
	return httpsig.Response{Status: int(status), Header: h, Body: []byte(body)}, s, p, err
}

func inputSigner(in map[string]any) (*httpsig.Signer, httpsig.Params, error) {
	keyid, ok1 := in["keyid"].(string)
	alg, ok2 := in["alg"].(string)
	nonce, ok3 := in["nonce"].(string)
	label, ok4 := in["key_label"].(string)
	created, err := integer(in["created"])
	if !ok1 || !ok2 || !ok3 || !ok4 || err != nil {
		return nil, httpsig.Params{}, errors.New("keyid, alg, nonce, key_label and created are required")
	}
	if name, _ := in["signature_name"].(string); name != httpsig.Label {
		return nil, httpsig.Params{}, errors.New("signature_name must be sig1")
	}
	ref, err := did.ParseKeyRef(keyid)
	if err != nil {
		return nil, httpsig.Params{}, err
	}
	cs, err := labelSigner(label)
	if err != nil {
		return nil, httpsig.Params{}, err
	}
	if cs.Public().Alg().String() != alg {
		return nil, httpsig.Params{}, fmt.Errorf("key_label %s is not an %s key", label, alg)
	}
	s, err := httpsig.NewSigner(ref, cs)
	return s, httpsig.Params{Created: created, Nonce: nonce}, err
}

// labelSigner derives the key of a vector label (vectors/README.md): the
// algorithm is named by the label's second path element.
func labelSigner(label string) (crypto.Signer, error) {
	switch {
	case strings.HasPrefix(label, "sage-spec/vectors/ed25519/"):
		return crypto.NewEd25519Signer(secret.From(labelSeed(label)))
	case strings.HasPrefix(label, "sage-spec/vectors/secp256k1/"):
		return crypto.NewSecp256k1Signer(secret.From(labelSeed(label)))
	case strings.HasPrefix(label, "sage-spec/vectors/p256/"):
		return crypto.NewP256Signer(secret.From(p256Scalar(label)))
	}
	return nil, fmt.Errorf("unknown key label %q", label)
}

func labelKey(in map[string]any) (crypto.PublicKey, error) {
	label, ok := in["key_label"].(string)
	if !ok {
		return nil, errors.New("key_label must be a string")
	}
	s, err := labelSigner(label)
	if err != nil {
		return nil, err
	}
	return s.Public(), nil
}

func sameCovered(in map[string]any, p httpsig.Parsed) error {
	want, ok := in["covered"].([]any)
	got := p.Components()
	if !ok || len(want) != len(got) {
		return errors.New("covered components differ from the input")
	}
	for i, c := range got {
		if want[i] != c.Identifier() {
			return fmt.Errorf("covered component %d is %s, the input names %v", i, c.Identifier(), want[i])
		}
	}
	return nil
}

func headerObject(v any) (http.Header, error) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, errors.New("headers must be an object")
	}
	h := http.Header{}
	for k, x := range m {
		s, ok := x.(string)
		if !ok || k != strings.ToLower(k) {
			return nil, errors.New("headers must map lower-case names to strings")
		}
		h.Set(k, s)
	}
	return h, nil
}

func setOutputFields(h http.Header, out map[string]any, keys ...string) error {
	names := map[string]string{"signature_input": httpsig.FieldSignatureInput,
		"signature": httpsig.FieldSignature, "content_digest": httpsig.FieldContentDigest}
	for _, k := range keys {
		s, ok := out[k].(string)
		if !ok {
			return fmt.Errorf("%s must be a string", k)
		}
		h.Set(names[k], s)
	}
	return nil
}

func integer(v any) (int64, error) {
	switch n := v.(type) {
	case int:
		return int64(n), nil
	case int64:
		return n, nil
	case json.Number:
		if i, err := strconv.ParseInt(n.String(), 10, 64); err == nil {
			return i, nil
		}
	}
	return 0, fmt.Errorf("%v is not an integer", v)
}

// rfc9421Rejects returns the errata §11 vectors for 03 that stages S0 to
// S4, S6 and S8 (03 §5 steps 1, 2, 5, 6 and 8) reject, at the fixed
// verification time verify_at. Each is request-ed25519 (or
// response-ed25519) as received with one mutation, re-signed with the same key over the mutated
// message where the mutation changes the signed content, so the mutation
// is the only reason to reject it.
func rfc9421Rejects() []vectors.Case {
	const (
		sized   = "rejected by the field size limit before parsing (E03-21)"
		parsed  = "rejected when Signature-Input and Signature are parsed (03 §5 step 1)"
		checked = "rejected when the signature parameters are checked (03 §5 step 2)"
		covered = "rejected when the covered components are checked (03 §5 step 2)"
		signed  = "rejected when the signature is verified (03 §5 step 6)"
		signer  = "rejected when keyid and the expected signer are checked (03 §5 step 3)"
		digest  = "rejected when Content-Digest is checked against the body (03 §5 step 7)"
		timed   = "rejected when the timing is checked (03 §5 step 5)"
		reused  = "rejected when the nonce is reserved (03 §5 step 8)"
	)
	req := func(name, item, desc, stage string, in map[string]any, reason obs.Reason) vectors.Case {
		return vectors.Case{
			Name: name, Mode: vectors.ModeReject,
			Description: item + ": request-ed25519 as received, with " + desc + "; " + stage + ".",
			Input:       in, Reject: rejectHTTPRequest, Reason: reason,
		}
	}
	reqES := func(name, item, desc, stage string, in map[string]any, reason obs.Reason) vectors.Case {
		return vectors.Case{
			Name: name, Mode: vectors.ModeReject,
			Description: item + ": request-es256k as received, with " + desc + "; " + stage + ".",
			Input:       in, Reject: rejectHTTPRequest, Reason: reason,
		}
	}
	resp := func(name, item, desc, stage string, in map[string]any, reason obs.Reason) vectors.Case {
		return vectors.Case{
			Name: name, Mode: vectors.ModeReject,
			Description: item + ": response-ed25519 as received, with " + desc + "; " + stage + ".",
			Input:       in, Reject: rejectHTTPResponse, Reason: reason,
		}
	}
	base := baseRequest()
	with := func(f func(*sigSpec)) map[string]any {
		s := baseRequest()
		f(&s)
		return s.received()
	}
	withResp := func(f func(*respSpec)) map[string]any {
		s := baseResponse()
		f(&s)
		return s.received()
	}
	param := func(s *sigSpec, old, new string) { s.params = strings.Replace(s.params, old, new, 1) }
	insert := func(s *sigSpec, after, id string) {
		for i, c := range s.covered {
			if c == after {
				s.covered = append(s.covered[:i+1], append([]string{id}, s.covered[i+1:]...)...)
				return
			}
		}
	}
	other := base
	other.input, other.signature = base.member(), base.sign()
	second := strings.Replace(other.input, "sig1=", "sig2=", 1)
	return []vectors.Case{
		req("tag-present", "E03-03", `;tag="x" added to Signature-Input, re-signed`, checked,
			with(func(s *sigSpec) { s.params += `;tag="x"` }), obs.ReasonParameters),
		req("unknown-param", "E03-03", "the undefined parameter ;foo=1 added to Signature-Input, re-signed", checked,
			with(func(s *sigSpec) { s.params += `;foo=1` }), obs.ReasonParameters),
		req("noncanonical-params", "E03-05", "a second space between the first two covered components; the signature is unchanged, as it is over the canonical serialisation (03 §2)", parsed,
			with(func(s *sigSpec) { s.sep = map[int]string{0: "  "} }), obs.ReasonNonCanonical),
		req("two-labels", "E03-06", "a second label sig2 (a copy of sig1) in both dictionaries", parsed,
			receivedWith(base, other.input+", "+second, other.signature+", "+strings.Replace(other.signature, "sig1=", "sig2=", 1)), obs.ReasonDuplicate),
		req("label-sets-differ", "E03-06", "the Signature label renamed to sig2", parsed,
			receivedWith(base, other.input, strings.Replace(other.signature, "sig1=", "sig2=", 1)), obs.ReasonMalformed),
		req("repeated-label", "E03-06", "the sig1 member repeated in both dictionaries", parsed,
			receivedWith(base, other.input+", "+other.input, other.signature+", "+other.signature), obs.ReasonDuplicate),
		req("two-signature-input-lines", "E03-07", "a second Signature-Input field line with the same value", parsed,
			receivedWith(base, other.input, other.signature, [2]string{"signature-input", other.input}), obs.ReasonDuplicate),
		req("signature-input-too-large", "E03-21", "a covered field x-pad-aaa… (present in the request with the value 1) whose name makes Signature-Input 8 KiB + 1 bytes, re-signed", sized,
			with(func(s *sigSpec) {
				s.covered = append(s.covered, `"x-pad-"`)
				name := "x-pad-" + strings.Repeat("a", core.SpecMaxSignatureFieldBytes+1-len(s.member()))
				s.covered[len(s.covered)-1] = `"` + name + `"`
				s.fields = append(s.fields, [2]string{name, "1"})
			}), obs.ReasonTooLarge),
		req("no-created", "E03-02", "created removed from Signature-Input, re-signed", checked,
			with(func(s *sigSpec) { param(s, `;created=1788609600`, "") }), obs.ReasonParameters),
		req("no-nonce", "E03-10", "nonce removed from Signature-Input, re-signed", checked,
			with(func(s *sigSpec) { param(s, `;nonce="`+reqNonce+`"`, "") }), obs.ReasonParameters),
		req("body-without-digest-coverage", "E03-10", "content-digest removed from the covered components, re-signed (the body and its Content-Digest kept)", covered,
			with(func(s *sigSpec) { s.drop(`"content-digest"`) }), obs.ReasonCoverage),
		req("covers-request-target", "E03-08", `"@request-target" covered after "@authority", re-signed`, covered,
			with(func(s *sigSpec) { insert(s, `"@authority"`, `"@request-target"`) }), obs.ReasonCoverage),
		req("covers-path", "E03-08", `"@path" covered after "@authority", re-signed`, covered,
			with(func(s *sigSpec) { insert(s, `"@authority"`, `"@path"`) }), obs.ReasonCoverage),
		req("covers-query-param", "E03-08", `the query ?q=1 added to the target URI and "@query-param";name="q" covered after "@authority", re-signed`, covered,
			with(func(s *sigSpec) { s.url += "?q=1"; insert(s, `"@authority"`, `"@query-param";name="q"`) }), obs.ReasonCoverage),
		req("authority-not-normal", "E03-08", "the default port in the target URI (https://agent-b.example:443/...), so @authority is agent-b.example:443, re-signed over that value (a verifier that normalised @authority instead of rejecting it would fail the signature)", covered,
			with(func(s *sigSpec) { s.url = strings.Replace(s.url, "agent-b.example/", "agent-b.example:443/", 1) }), obs.ReasonCoverage),
		req("component-param-sf", "E03-09", `"content-type";sf covered instead of "content-type", re-signed`, covered,
			with(func(s *sigSpec) { s.replace(`"content-type"`, `"content-type";sf`) }), obs.ReasonCoverage),
		req("req-in-request", "03 §4", `"@method";req covered instead of "@method", re-signed`, covered,
			with(func(s *sigSpec) { s.replace(`"@method"`, `"@method";req`) }), obs.ReasonCoverage),
		resp("response-no-status", "E03-10", `"@status" removed from the covered components, re-signed`, covered,
			withResp(func(s *respSpec) { s.drop(`"@status"`) }), obs.ReasonCoverage),
		resp("response-no-signature-req", "E03-10", `"signature";req removed from the covered components, re-signed`, covered,
			withResp(func(s *respSpec) { s.drop(`"signature";req`) }), obs.ReasonCoverage),
		req("too-old", "E03-11", "the clock at created + 301 s, beyond any MaxAge (at most 300 s); the message is unchanged", timed,
			with(func(s *sigSpec) { s.verifyAt = reqCreated + 301 }), obs.ReasonStale),
		req("too-far-future", "E03-11", "created = now + 301 s, beyond any MaxClockSkew (at most 300 s), re-signed", timed,
			with(func(s *sigSpec) { param(s, `;created=1788609600`, `;created=1788609901`) }), obs.ReasonFuture),
		req("expired", "E03-11", "expires = created + 10 added and the clock at created + 11, so expires = now - 1, re-signed", timed,
			with(func(s *sigSpec) {
				param(s, `;created=1788609600`, `;created=1788609600;expires=1788609610`)
				s.verifyAt = reqCreated + 11
			}), obs.ReasonExpired),
		req("expires-before-created", "E03-11", "expires = created added and the clock at created - 1, so with a MaxClockSkew of at least 1 s (the default is 60 s) only expires <= created fails, re-signed", timed,
			with(func(s *sigSpec) {
				param(s, `;created=1788609600`, `;created=1788609600;expires=1788609600`)
				s.verifyAt = reqCreated - 1
			}), obs.ReasonExpired),
		req("replay-other-fragment", "E03-12", "keyid without the #key-1 fragment and the same nonce, re-signed, after request-ed25519 (in prior) was accepted by the same replay store; both name one signer DID", reused,
			with(func(s *sigSpec) {
				param(s, `;keyid="`+reqKeyID+`"`, `;keyid="`+reqDID+`"`)
				s.prior = []map[string]any{baseRequest().received()}
			}), obs.ReasonReplay),
		req("keyid-bad-did", "E03-14", "keyid did:sage:ethereum:a/b (a path character in the identifier), re-signed", signer,
			with(func(s *sigSpec) { param(s, `;keyid="`+reqKeyID+`"`, `;keyid="did:sage:ethereum:a/b"`) }), obs.ReasonIdentifier),
		req("keyid-bad-fragment", "E03-14", "keyid "+reqDID+"#1 (a fragment that is not key-<n>), re-signed", signer,
			with(func(s *sigSpec) { param(s, `;keyid="`+reqKeyID+`"`, `;keyid="`+reqDID+`#1"`) }), obs.ReasonIdentifier),
		req("x-sage-did-mismatch", "03 §3", "X-SAGE-DID naming another DID ("+respDID+"), re-signed", signer,
			with(func(s *sigSpec) { s.setField("x-sage-did", respDID) }), obs.ReasonHeaderMismatch),
		req("digest-mismatch", "E03-13", "the body replaced by another body of the same length; Content-Digest and the signature are unchanged (the signature covers Content-Digest, not the body)", digest,
			with(func(s *sigSpec) {
				s.input, s.signature = s.member(), s.sign()
				s.body = strings.Replace(s.body, `"hello"`, `"HELLO"`, 1)
			}), obs.ReasonDigest),
		req("digest-two-members", "E03-13", "a sha-512 member added to Content-Digest after the correct sha-256 member, re-signed", digest,
			with(func(s *sigSpec) { s.setField("content-digest", reqContentDigest+", sha-512=:"+sha512Of(reqBody)+":") }), obs.ReasonDigest),
		req("digest-sha512-only", "E03-13", "Content-Digest holding only the correct sha-512 member, re-signed", digest,
			with(func(s *sigSpec) { s.setField("content-digest", "sha-512=:"+sha512Of(reqBody)+":") }), obs.ReasonDigest),
		reqES("es256k-64-byte", "E03-23", "its 65-byte signature re-encoded as the 64 bytes r || s", signed,
			with(func(s *sigSpec) { s.es256kRequest(func(b []byte) []byte { return b[:64] }) }), obs.ReasonSignature),
		reqES("es256k-der", "E03-23", "its signature re-encoded as ASN.1 DER", signed,
			with(func(s *sigSpec) { s.es256kRequest(derOf) }), obs.ReasonSignature),
		req("alg-key-mismatch", "E01-11", `alg="es256k" with the resolved Ed25519 key, re-signed with that key`, signed,
			with(func(s *sigSpec) { param(s, `;alg="ed25519"`, `;alg="es256k"`) }), obs.ReasonAlgorithm),
		resp("response-other-request", "E03-10", `the response re-signed with "@target-uri";req https://agent-b.example/mcp/tools/list, another request's value, and checked against request-ed25519`, signed,
			withResp(func(s *respSpec) { s.signFor.url = strings.Replace(s.signFor.url, "/call", "/list", 1) }), obs.ReasonSignature),
	}
}

// rfc9421KeepsNonce returns digest-mismatch-keeps-nonce (errata §11, SN-6,
// E03-18): request-ed25519 with its body replaced is rejected at step 7, and
// then request-ed25519 itself, with the same nonce, is accepted by the same
// verifier, because a message that fails a step before 8 does not consume
// its nonce.
func rfc9421KeepsNonce() vectors.Case {
	mutated := baseRequest()
	mutated.input, mutated.signature = mutated.member(), mutated.sign()
	mutated.body = strings.Replace(mutated.body, `"hello"`, `"HELLO"`, 1)
	first := mutated.received()
	delete(first, "verify_at")
	in := baseRequest().received()
	in["rejected_first"] = []any{first}
	return vectors.Case{
		Name: "digest-mismatch-keeps-nonce", Mode: vectors.ModeDeterministic,
		Description: "E03-18 (SN-6): request-ed25519 with its body replaced (Content-Digest and signature unchanged), in rejected_first, " +
			"is rejected when Content-Digest is checked (03 §5 step 7); request-ed25519 with the same nonce is then accepted by the same verifier at verify_at.",
		Input: in, Produce: produceKeepsNonce,
	}
}

func produceKeepsNonce(in map[string]any) (map[string]any, error) {
	v, err := inputVerifier(in)
	if err != nil {
		return nil, err
	}
	list, ok := in["rejected_first"].([]any)
	if !ok || len(list) == 0 {
		return nil, errors.New("rejected_first must be a non-empty list")
	}
	codes := make([]any, len(list))
	for i, x := range list {
		m, ok := x.(map[string]any)
		if !ok {
			return nil, errors.New("each rejected_first request must be an object")
		}
		r, err := receivedRequestOf(m)
		if err != nil {
			return nil, err
		}
		_, err = checkRequest(r, v, httpsig.Expect{})
		var rej *core.Rejection
		if !errors.As(err, &rej) {
			return nil, fmt.Errorf("rejected_first request %d was not rejected: %v", i, err)
		}
		codes[i] = rej.Reason().String()
	}
	r, err := receivedRequestOf(in)
	if err != nil {
		return nil, err
	}
	if _, err := checkRequest(r, v, httpsig.Expect{}); err != nil {
		return nil, fmt.Errorf("the request after rejected_first was not accepted: %s", detail(err))
	}
	return map[string]any{"rejected_first": codes, "accepted": true}, nil
}

// rejectHTTPRequest reads the received request of a vector and runs
// checkRequest on it at verify_at, after the requests in prior, which share
// its replay store and must each be accepted.
func rejectHTTPRequest(in map[string]any) error {
	r, err := receivedRequestOf(in)
	if err != nil {
		return err
	}
	v, err := inputVerifier(in)
	if err != nil {
		return err
	}
	if x, ok := in["prior"]; ok {
		list, ok := x.([]any)
		if !ok {
			return errors.New("prior must be a list")
		}
		for i, x := range list {
			p, ok := x.(map[string]any)
			if !ok {
				return errors.New("each prior request must be an object")
			}
			pr, err := receivedRequestOf(p)
			if err != nil {
				return err
			}
			if _, err := checkRequest(pr, v, httpsig.Expect{}); err != nil {
				return fmt.Errorf("prior request %d was not accepted: %s", i, detail(err))
			}
		}
	}
	_, err = checkRequest(r, v, httpsig.Expect{})
	return err
}

// inputVerifier returns the verifier of a received message at its
// verify_at, with the registry of its key_label.
func inputVerifier(in map[string]any) (*httpsig.Verifier, error) {
	at, err := integer(in["verify_at"])
	if err != nil {
		return nil, errors.New("verify_at must be an integer")
	}
	label, ok := in["key_label"].(string)
	if !ok {
		return nil, errors.New("key_label must be a string")
	}
	return verifierAt(at, label)
}

// detail keeps a rejection of a prior request from counting as the
// vector's expected rejection.
func detail(err error) string {
	var r *core.Rejection
	if errors.As(err, &r) {
		return r.Reason().String()
	}
	return err.Error()
}

// rejectHTTPResponse reads the received response and request of a vector
// and runs checkResponse on the response at verify_at.
func rejectHTTPResponse(in map[string]any) error {
	reqIn, ok := in["request"].(map[string]any)
	if !ok {
		return errors.New("request must be an object")
	}
	req, err := receivedRequestOf(reqIn)
	if err != nil {
		return err
	}
	h, err := fieldLines(in["fields"])
	if err != nil {
		return err
	}
	body, ok := in["body"].(string)
	status, err := integer(in["status"])
	if !ok || err != nil {
		return errors.New("body must be a string and status an integer")
	}
	v, err := inputVerifier(in)
	if err != nil {
		return err
	}
	_, err = checkResponse(httpsig.Response{Status: int(status), Header: h, Body: []byte(body)}, req, v, httpsig.Expect{})
	return err
}

func receivedRequestOf(in map[string]any) (httpsig.Request, error) {
	h, err := fieldLines(in["fields"])
	if err != nil {
		return httpsig.Request{}, err
	}
	method, ok1 := in["method"].(string)
	url, ok2 := in["url"].(string)
	body, ok3 := in["body"].(string)
	if !ok1 || !ok2 || !ok3 {
		return httpsig.Request{}, errors.New("method, url and body must be strings")
	}
	return httpsig.Request{Method: method, Target: url, Header: h, Body: []byte(body)}, nil
}

func fieldLines(v any) (http.Header, error) {
	list, ok := v.([]any)
	if !ok {
		return nil, errors.New("fields must be a list")
	}
	h := http.Header{}
	for _, f := range list {
		pair, ok := f.([]any)
		if !ok || len(pair) != 2 {
			return nil, errors.New("each field must be [name, value]")
		}
		name, ok1 := pair[0].(string)
		value, ok2 := pair[1].(string)
		if !ok1 || !ok2 || name == "" || strings.ToLower(name) != name {
			return nil, errors.New("field names must be lower-case strings")
		}
		h.Add(name, value)
	}
	return h, nil
}
