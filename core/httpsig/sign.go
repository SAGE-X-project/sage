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
	"fmt"
	"net/http"
	"strings"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/crypto"
	"github.com/sage-x-project/sage/core/did"
)

// ErrSigner is returned by NewSigner for a zero keyid or a nil signer.
var ErrSigner = errors.New("httpsig: a signer needs a keyid and a crypto.Signer")

// ErrSignParams is returned for signature parameters the profile does not
// allow: a created or expires outside 0 to 999999999999999 (RFC 8941
// integers), an expires not after created, a missing request nonce, or a
// nonce that is not 16 to 64 characters of ALPHA, DIGIT, "-" and "_"
// (03 §5 Replay).
var ErrSignParams = errors.New("httpsig: invalid signature parameters")

// ErrSignMessage is returned for a message that cannot be signed as the
// profile requires: an invalid method or target URI, an @authority not in
// normal form, a Content-Digest that does not match the body, an
// X-SAGE-DID other than the signer's DID, a response to an unsigned
// request, or a message that already carries a signature.
var ErrSignMessage = errors.New("httpsig: message cannot be signed")

// Label is the signature label the signer emits (03 §1).
const Label = "sig1"

// Signer signs requests and responses with one key. It emits only what a
// verifier of this profile accepts; there is no lenient mode.
type Signer struct {
	keyid  string
	did    did.DID
	signer crypto.Signer
}

// NewSigner returns a signer that names key, as it was parsed, as keyid
// and signs with s.
func NewSigner(key did.KeyRef, s crypto.Signer) (*Signer, error) {
	if key.DID().IsZero() || s == nil || s.Public() == nil {
		return nil, ErrSigner
	}
	return &Signer{keyid: key.AsSent(), did: key.DID(), signer: s}, nil
}

// Params are the signature parameters a caller chooses. Created and Expires
// are Unix seconds; Expires 0 means none. Nonce is required on requests
// (03 §3) and optional on responses.
type Params struct {
	Created int64
	Expires int64
	Nonce   string
}

// Fields are the header fields a signature adds to its message.
type Fields struct {
	SignatureInput string
	Signature      string
	ContentDigest  string // empty when the message has no body
}

// Apply sets the fields on h, replacing every line of them under any key
// spelling, so h carries one line of each (E03-07).
func (f Fields) Apply(h http.Header) {
	for k := range h {
		for _, name := range []string{FieldSignatureInput, FieldSignature, FieldContentDigest} {
			if strings.EqualFold(k, name) && (name != FieldContentDigest || f.ContentDigest != "") {
				delete(h, k)
			}
		}
	}
	h.Set(FieldSignatureInput, f.SignatureInput)
	h.Set(FieldSignature, f.Signature)
	if f.ContentDigest != "" {
		h.Set(FieldContentDigest, f.ContentDigest)
	}
}

// SignRequest signs r. It covers @method, @target-uri and @authority, then
// content-type and content-digest when there is a body, x-sage-did and date
// when present, in the order of 03 §3.
func (s *Signer) SignRequest(ctx context.Context, r Request, p Params) (Fields, error) {
	if p.Nonce == "" {
		return Fields{}, fmt.Errorf("%w: a request signature needs a nonce", ErrSignParams)
	}
	h, digest, err := prepare(r.Header, r.Body)
	if err != nil {
		return Fields{}, err
	}
	if !validMethod(r.Method) {
		return Fields{}, fmt.Errorf("%w: method is not a token", ErrSignMessage)
	}
	if err := checkAuthority(r.Target); err != nil {
		return Fields{}, fmt.Errorf("%w: target URI or @authority: %v", ErrSignMessage, detail(err))
	}
	// x-sage-did carries the signer's DID (03 §3).
	if v, ok, err := fieldValue(h, "x-sage-did"); err != nil || (ok && !sameDID(v, s.did)) {
		return Fields{}, fmt.Errorf("%w: X-SAGE-DID is not the signer's DID", ErrSignMessage)
	}
	comps := []Component{field(compMethod), field(compTargetURI), field(compAuthority)}
	if len(r.Body) > 0 {
		if hasField(h, "content-type") {
			comps = append(comps, field("content-type"))
		}
		comps = append(comps, field(fieldContentDigestLower))
	}
	for _, name := range []string{"x-sage-did", "date"} {
		if hasField(h, name) {
			comps = append(comps, field(name))
		}
	}
	signed := r
	signed.Header = h
	f, err := s.sign(ctx, comps, p, crypto.PurposeHTTPRequest, func(params string) ([]byte, error) {
		return buildBase(comps, params, nil, signed)
	})
	if err != nil {
		return Fields{}, err
	}
	f.ContentDigest = digest
	// The signer checks its output as a verifier would (03 §5 steps 1, 2
	// and 6), so it never emits a signature this profile rejects.
	f.Apply(signed.Header)
	parsed, err := Parse(signed.Header, Label, core.DefaultLimits())
	if err == nil {
		var b Base
		if b, err = RequestBase(parsed, signed); err == nil {
			_, err = verifySignature(b, s.signer.Public())
		}
	}
	if err != nil {
		return Fields{}, fmt.Errorf("%w: %v", ErrSignMessage, detail(err))
	}
	return f, nil
}

// SignResponse signs resp as the answer to req. It covers @status,
// "@method";req, "@target-uri";req and "@authority";req, then
// "content-digest";req and "signature";req when the request carried them,
// content-type when set, and content-digest when there is a body (03 §4).
func (s *Signer) SignResponse(ctx context.Context, resp Response, req Request, p Params) (Fields, error) {
	h, digest, err := prepare(resp.Header, resp.Body)
	if err != nil {
		return Fields{}, err
	}
	if resp.Status < 100 || resp.Status > 599 {
		return Fields{}, fmt.Errorf("%w: status out of range", ErrSignMessage)
	}
	if !validMethod(req.Method) {
		return Fields{}, fmt.Errorf("%w: request method is not a token", ErrSignMessage)
	}
	if err := checkAuthority(req.Target); err != nil {
		return Fields{}, fmt.Errorf("%w: request target URI or @authority: %v", ErrSignMessage, detail(err))
	}
	// A response signature binds a signed request (03 §4, E03-20).
	if !hasField(req.Header, FieldSignature) {
		return Fields{}, fmt.Errorf("%w: the request is not signed", ErrSignMessage)
	}
	comps := []Component{field(compStatus), reqField(compMethod), reqField(compTargetURI), reqField(compAuthority)}
	if hasField(req.Header, FieldContentDigest) {
		comps = append(comps, reqField(fieldContentDigestLower))
	}
	comps = append(comps, reqField("signature"))
	if hasField(h, "content-type") {
		comps = append(comps, field("content-type"))
	}
	if len(resp.Body) > 0 {
		comps = append(comps, field(fieldContentDigestLower))
	}
	signed := resp
	signed.Header = h
	f, err := s.sign(ctx, comps, p, crypto.PurposeHTTPResponse, func(params string) ([]byte, error) {
		return buildBase(comps, params, &signed, req)
	})
	if err != nil {
		return Fields{}, err
	}
	f.ContentDigest = digest
	f.Apply(signed.Header)
	parsed, err := Parse(signed.Header, Label, core.DefaultLimits())
	if err == nil {
		var b Base
		if b, err = ResponseBase(parsed, signed, req); err == nil {
			_, err = verifySignature(b, s.signer.Public())
		}
	}
	if err != nil {
		return Fields{}, fmt.Errorf("%w: %v", ErrSignMessage, detail(err))
	}
	return f, nil
}

// prepare copies the header, refuses one that already carries a signature,
// and sets Content-Digest for a body, refusing a different one.
func prepare(header http.Header, body []byte) (http.Header, string, error) {
	h := header.Clone()
	if h == nil {
		h = http.Header{}
	}
	if hasField(h, FieldSignatureInput) || hasField(h, FieldSignature) {
		return nil, "", fmt.Errorf("%w: the message already carries a signature", ErrSignMessage)
	}
	if len(body) == 0 {
		if hasField(h, FieldContentDigest) {
			return nil, "", fmt.Errorf("%w: Content-Digest without a body", ErrSignMessage)
		}
		return h, "", nil
	}
	digest := ContentDigest(body)
	if v, ok, err := fieldValue(h, FieldContentDigest); err != nil || (ok && v != digest) {
		return nil, "", fmt.Errorf("%w: Content-Digest does not match the body", ErrSignMessage)
	}
	for k := range h {
		if strings.EqualFold(k, FieldContentDigest) {
			delete(h, k)
		}
	}
	h.Set(FieldContentDigest, digest)
	return h, digest, nil
}

// sign serialises the member, builds the base with base and signs it.
func (s *Signer) sign(ctx context.Context, comps []Component, p Params, purpose crypto.Purpose, base func(string) ([]byte, error)) (Fields, error) {
	if p.Created < 0 || p.Created > maxInteger || p.Expires < 0 || p.Expires > maxInteger ||
		(p.Expires != 0 && p.Expires <= p.Created) || (p.Nonce != "" && !validNonce(p.Nonce)) {
		return Fields{}, ErrSignParams
	}
	ps := []param{
		{key: "keyid", value: bare{kind: kindString, text: s.keyid}},
		{key: "alg", value: bare{kind: kindString, text: s.signer.Public().Alg().String()}},
		{key: "created", value: bare{kind: kindInteger, integer: p.Created}},
	}
	if p.Expires != 0 {
		ps = append(ps, param{key: "expires", value: bare{kind: kindInteger, integer: p.Expires}})
	}
	if p.Nonce != "" {
		ps = append(ps, param{key: "nonce", value: bare{kind: kindString, text: p.Nonce}})
	}
	list := make([]item, len(comps))
	for i, c := range comps {
		list[i] = item{value: bare{kind: kindString, text: c.name}, params: c.params}
	}
	member := member{key: Label, inner: true, list: list, params: ps}.serialise()
	b, err := base(strings.TrimPrefix(member, Label+"="))
	if err != nil {
		return Fields{}, fmt.Errorf("%w: %v", ErrSignMessage, detail(err))
	}
	sig, err := s.signer.Sign(ctx, purpose, b)
	if err != nil {
		return Fields{}, err
	}
	return Fields{
		SignatureInput: member,
		Signature:      Label + "=:" + base64.StdEncoding.EncodeToString(sig.Bytes()) + ":",
	}, nil
}

func field(name string) Component { return Component{name: name} }

func reqField(name string) Component {
	return Component{name: name, params: []param{{key: "req", value: bare{kind: kindBoolean, boolean: true}}}}
}

// maxInteger is the largest RFC 8941 integer (15 digits).
const maxInteger = 999_999_999_999_999

func sameDID(v string, d did.DID) bool {
	got, err := did.Parse(v)
	return err == nil && got.Equal(d)
}

// detail keeps a signing failure from reading as an inbound rejection: the
// returned error does not wrap the core.Rejection, so a host that maps
// rejections to 401 does not mistake a local signing fault for one.
func detail(err error) string {
	var r *core.Rejection
	if errors.As(err, &r) {
		return "rejected at " + r.Stage().String() + ": " + r.Reason().String()
	}
	return err.Error()
}
