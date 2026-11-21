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
	"time"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/clock"
	"github.com/sage-x-project/sage/core/crypto"
	"github.com/sage-x-project/sage/core/did"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/replay"
)

// ErrVerifier is returned by NewVerifier for a missing clock, store or
// resolver or invalid settings, and by the methods of a Verifier not made
// by NewVerifier.
var ErrVerifier = errors.New("httpsig: invalid verifier configuration")

// ErrExpect is returned for an Expect that cannot apply: Authorities on a
// response (03 §5 step 3 checks them for requests only).
var ErrExpect = errors.New("httpsig: expectation does not apply to this message")

// Expect holds what the host requires of a signature at step 3 of 03 §5.
// A zero field is not checked.
type Expect struct {
	// DID is the expected signer (ExpectedDID), compared in canonical form
	// with keyid's DID (06 §2).
	DID did.DID
	// KeyID is the expected keyid (ExpectedKeyID): its DID compared in
	// canonical form and its fragment exactly.
	KeyID did.KeyRef
	// Authorities (ExpectedAuthorities, requests only) lists the @authority
	// values the request may be addressed to, in the normal form of 03 §2
	// (lower-case host, a port only when it is not the scheme's default).
	Authorities []string
}

// Verifier verifies RFC 9421 request and response signatures with every
// step of 03 §5, in its order: parse, parameters and coverage, keyid and
// the expected signer, resolution and key selection, timing, signature,
// Content-Digest, and the replay reservation last, so a message that fails
// any earlier step keeps its nonce (E03-18). There is no lenient mode.
//
// It returns a *core.Rejection whose Error text is generic. The caller must
// answer every rejection with the same result (WriteAuthenticationFailure,
// E03-17) and may log the reason locally. Other errors (a cancelled context,
// a replay store that serves other limits) must be treated as failures too.
type Verifier struct {
	guard    *Guard
	resolver did.KeyResolver
	policy   did.Policy
	lim      core.Limits
}

// NewVerifier returns a Verifier that reads time from c, takes its limits
// and tracer from s, reserves nonces in store and resolves signers with r.
// The store and the resolver's observations should use the same clock
// (Guard, did.Observation).
func NewVerifier(c clock.Clock, s core.Settings, store replay.Store, r did.KeyResolver) (*Verifier, error) {
	if c == nil || store == nil || r == nil || s.Validate() != nil {
		return nil, ErrVerifier
	}
	g, err := NewGuard(c, s.Limits(), store)
	if err != nil {
		return nil, errors.Join(ErrVerifier, err)
	}
	return &Verifier{guard: g, resolver: r, policy: did.Policy{Clock: c, Settings: s}, lim: s.Limits()}, nil
}

func (v *Verifier) ok() bool { return v != nil && v.guard.ok() && v.resolver != nil }

// VerifyRequest verifies a request signature (03 §5). r.Target must be the
// URI the signer used, built from the configured external origin
// (TargetFrom, E03-22), and r.Body the complete body as transferred.
func (v *Verifier) VerifyRequest(ctx context.Context, r Request, e Expect) (VerifiedRequest, error) {
	if !v.ok() {
		return VerifiedRequest{}, ErrVerifier
	}
	if int64(len(r.Body)) > v.lim.MaxDigestBodyBytes {
		return VerifiedRequest{}, reject(obs.ReasonTooLarge, "body exceeds the Content-Digest limit")
	}
	p, err := Parse(r.Header, "", v.lim) // step 1
	if err != nil {
		return VerifiedRequest{}, err
	}
	base, err := RequestBase(p, r) // step 2, and the base of step 6
	if err != nil {
		return VerifiedRequest{}, err
	}
	ref, err := CheckSigner(p, r.Header, e) // step 3
	if err != nil {
		return VerifiedRequest{}, err
	}
	if len(e.Authorities) > 0 {
		if err := checkAuthorities(r.Target, e.Authorities); err != nil {
			return VerifiedRequest{}, err
		}
	}
	m, err := v.verify(ctx, p, base, ref, r.Header, r.Body)
	if err != nil {
		return VerifiedRequest{}, err
	}
	return VerifiedRequest{m: m}, nil
}

// VerifyResponse verifies a response signature (03 §4, §5) for the signed
// request req it answers, which supplies the components marked ;req. A
// client that sent a signed request to a responder known to sign responses
// must treat any error here, including for an error response without a
// signature, as a failure and must not use the response body (E03-20). A
// response reserves no nonce: the binding by "signature";req is its replay
// protection (00-overview.md §7.4 DP-4).
func (v *Verifier) VerifyResponse(ctx context.Context, resp Response, req Request, e Expect) (VerifiedResponse, error) {
	if !v.ok() {
		return VerifiedResponse{}, ErrVerifier
	}
	if len(e.Authorities) > 0 {
		return VerifiedResponse{}, ErrExpect
	}
	if int64(len(resp.Body)) > v.lim.MaxDigestBodyBytes {
		return VerifiedResponse{}, reject(obs.ReasonTooLarge, "body exceeds the Content-Digest limit")
	}
	p, err := Parse(resp.Header, "", v.lim)
	if err != nil {
		return VerifiedResponse{}, err
	}
	base, err := ResponseBase(p, resp, req)
	if err != nil {
		return VerifiedResponse{}, err
	}
	ref, err := CheckSigner(p, resp.Header, e)
	if err != nil {
		return VerifiedResponse{}, err
	}
	m, err := v.verify(ctx, p, base, ref, resp.Header, resp.Body)
	if err != nil {
		return VerifiedResponse{}, err
	}
	return VerifiedResponse{m: m}, nil
}

// verify runs steps 4 to 8 of 03 §5.
func (v *Verifier) verify(ctx context.Context, p Parsed, base Base, ref did.KeyRef, h http.Header, body []byte) (*verified, error) {
	agent, err := did.ResolveAgent(ctx, v.resolver, ref.DID(), v.policy) // step 4
	if err != nil {
		return nil, err
	}
	if err := v.guard.CheckTime(p); err != nil { // step 5
		return nil, err
	}
	signed, err := VerifySignature(base, agent) // step 6
	if err != nil {
		return nil, err
	}
	if err := CheckDigest(h, body, v.lim); err != nil { // step 7
		return nil, err
	}
	if err := v.guard.Reserve(ctx, signed); err != nil { // step 8, last
		return nil, err
	}
	created, _ := p.Created()
	return &verified{
		signer: agent.DID(), keyID: ref, key: agent.SigningKey().PublicKey(),
		created: time.Unix(created, 0), label: p.Label(),
	}, nil
}

// CheckSigner is step 3 of 03 §5: keyid must be a did:sage DID, optionally
// followed by #key-<n> (§3, E03-14); X-SAGE-DID, when present, must equal
// keyid's DID in canonical form (06 §2); and keyid must meet e.DID and
// e.KeyID when they are set. It returns keyid. Authorities are checked by
// VerifyRequest, which has the target.
func CheckSigner(p Parsed, h http.Header, e Expect) (did.KeyRef, error) {
	keyid, ok := p.KeyID()
	if !ok {
		return did.KeyRef{}, reject(obs.ReasonParameters, "signature lacks keyid")
	}
	ref, err := did.ParseKeyRef(keyid)
	if err != nil {
		return did.KeyRef{}, reject(obs.ReasonIdentifier, "keyid is not a did:sage DID with an optional #key-<n>")
	}
	header, err := oneLine(h, FieldSAGEDID, false)
	if err != nil {
		return did.KeyRef{}, err
	}
	if header != "" || hasField(h, FieldSAGEDID) {
		sent, err := did.Parse(header)
		if err != nil || !sent.Equal(ref.DID()) {
			return did.KeyRef{}, reject(obs.ReasonHeaderMismatch, "X-SAGE-DID is not keyid's DID")
		}
	}
	if !e.DID.IsZero() && !e.DID.Equal(ref.DID()) {
		return did.KeyRef{}, reject(obs.ReasonPeerMismatch, "keyid's DID is not the expected signer")
	}
	if !e.KeyID.DID().IsZero() && (!e.KeyID.DID().Equal(ref.DID()) || e.KeyID.Fragment() != ref.Fragment()) {
		return did.KeyRef{}, reject(obs.ReasonPeerMismatch, "keyid is not the expected key id")
	}
	return ref, nil
}

func checkAuthorities(targetURI string, allowed []string) error {
	t, err := parseTarget(targetURI)
	if err != nil {
		return err
	}
	for _, a := range allowed {
		if a == t.authority {
			return nil
		}
	}
	return reject(obs.ReasonPeerMismatch, "@authority is not an expected authority")
}

// verified is what Verifier established about a signature.
type verified struct {
	signer  did.DID
	keyID   did.KeyRef
	key     crypto.PublicKey
	created time.Time
	label   string
}

// VerifiedRequest is a request whose signature passed every step of 03 §5.
// Only VerifyRequest makes one; the zero value is not verified, and its
// methods return zero values.
type VerifiedRequest struct{ m *verified }

// VerifiedResponse is a response whose signature passed every step of
// 03 §5 for the request it answers. Only VerifyResponse makes one.
type VerifiedResponse struct{ m *verified }

// Valid reports whether r was made by VerifyRequest.
func (r VerifiedRequest) Valid() bool { return r.m != nil }

// Signer returns keyid's DID, which the key was resolved for.
func (r VerifiedRequest) Signer() did.DID { return r.m.get().signer }

// KeyID returns keyid as sent; its fragment is informative (E03-15).
func (r VerifiedRequest) KeyID() did.KeyRef { return r.m.get().keyID }

// Key returns the key that verified the signature.
func (r VerifiedRequest) Key() crypto.PublicKey { return r.m.get().key }

// Created returns the signer-asserted time that passed the timing check.
func (r VerifiedRequest) Created() time.Time { return r.m.get().created }

// Label returns the signature label.
func (r VerifiedRequest) Label() string { return r.m.get().label }

// Valid reports whether r was made by VerifyResponse.
func (r VerifiedResponse) Valid() bool { return r.m != nil }

// Signer returns keyid's DID, which the key was resolved for.
func (r VerifiedResponse) Signer() did.DID { return r.m.get().signer }

// KeyID returns keyid as sent.
func (r VerifiedResponse) KeyID() did.KeyRef { return r.m.get().keyID }

// Key returns the key that verified the signature.
func (r VerifiedResponse) Key() crypto.PublicKey { return r.m.get().key }

// Created returns the signer-asserted time that passed the timing check.
func (r VerifiedResponse) Created() time.Time { return r.m.get().created }

// Label returns the signature label.
func (r VerifiedResponse) Label() string { return r.m.get().label }

func (m *verified) get() verified {
	if m == nil {
		return verified{}
	}
	return *m
}
