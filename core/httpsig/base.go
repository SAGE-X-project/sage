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
	"strconv"
	"strings"

	"github.com/sage-x-project/sage/core/crypto"
	"github.com/sage-x-project/sage/core/obs"
)

// Base is the signature base of a signature that passed step 2 of 03 §5.
// Only RequestBase and ResponseBase make one, so a signature cannot be
// verified over a base that skipped the coverage checks (H4).
type Base struct {
	p        Parsed
	b        []byte
	response bool
}

// Bytes returns a copy of the base.
func (b Base) Bytes() []byte { return append([]byte(nil), b.b...) }

// RequestBase runs CheckRequest, then rebuilds the signature base from the
// received request (03 §2, RFC 9421 §2.5): one line per covered component,
// then the "@signature-params" line with the re-serialised member. It fails
// when a covered component has no value in r.
func RequestBase(p Parsed, r Request) (Base, error) {
	if err := CheckRequest(p, r); err != nil {
		return Base{}, err
	}
	b, err := buildBase(p.components, p.params, nil, r)
	if err != nil {
		return Base{}, err
	}
	return Base{p: p, b: b}, nil
}

// ResponseBase runs CheckResponse, then rebuilds the signature base from
// the received response and the request it answers, which supplies the
// components marked ;req (03 §4).
func ResponseBase(p Parsed, resp Response, req Request) (Base, error) {
	if err := CheckResponse(p, resp, req); err != nil {
		return Base{}, err
	}
	b, err := buildBase(p.components, p.params, &resp, req)
	if err != nil {
		return Base{}, err
	}
	return Base{p: p, b: b, response: true}, nil
}

func buildBase(comps []Component, params string, resp *Response, req Request) ([]byte, error) {
	var t target
	var tErr error
	targetOnce := false
	reqTarget := func() (target, error) {
		if !targetOnce {
			t, tErr = parseTarget(req.Target)
			targetOnce = true
		}
		return t, tErr
	}
	var b strings.Builder
	for _, c := range comps {
		isReq, ok := componentParams(c)
		if !ok || (isReq && resp == nil) {
			return nil, canonicalize("component parameter has no value")
		}
		v, err := componentValue(c.name, isReq, resp, req, reqTarget)
		if err != nil {
			return nil, err
		}
		b.WriteString(c.Identifier())
		b.WriteString(": ")
		b.WriteString(v)
		b.WriteByte('\n')
	}
	b.WriteString(`"@signature-params": `)
	b.WriteString(params)
	return []byte(b.String()), nil
}

func componentValue(name string, isReq bool, resp *Response, req Request, reqTarget func() (target, error)) (string, error) {
	fromRequest := resp == nil || isReq
	if name == compStatus {
		if fromRequest || resp.Status < 100 || resp.Status > 599 {
			return "", canonicalize("@status has no value")
		}
		return strconv.Itoa(resp.Status), nil
	}
	if len(name) > 0 && name[0] == '@' {
		if !fromRequest || !requestDerived(name) {
			return "", canonicalize("derived component has no value")
		}
		if name == compMethod {
			if !validMethod(req.Method) {
				return "", canonicalize("request method is not a token")
			}
			return req.Method, nil
		}
		t, err := reqTarget()
		if err != nil {
			return "", err
		}
		switch name {
		case compTargetURI:
			return t.uri, nil
		case compAuthority:
			return t.authority, nil
		case compScheme:
			return t.scheme, nil
		default:
			return t.query, nil
		}
	}
	h := req.Header
	if !fromRequest {
		h = resp.Header
	}
	v, ok, err := fieldValue(h, name)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", canonicalize("covered field is absent")
	}
	return v, nil
}

// Signed is a signature that passed step 2 of 03 §5 and verified at step 6
// over its rebuilt base. Only VerifySignature makes one, so the replay
// reservation (Guard.Reserve) cannot run for a signature that has not
// verified, and an unauthenticated sender cannot consume another signer's
// nonces or replay capacity.
type Signed struct {
	p        Parsed
	response bool
}

// Parsed returns the parsed signature.
func (s Signed) Parsed() Parsed { return s.p }

// VerifySignature checks the signature the base was built for, over that
// base, with key (03 §5 step 6 for the key the resolver chose). An alg
// parameter must name the key's algorithm (E01-11). It verifies the
// signature only: the message is verified when every step of 03 §5 has
// passed.
func VerifySignature(base Base, key crypto.PublicKey) (Signed, error) {
	p := base.p
	if base.b == nil {
		return Signed{}, canonicalize("no signature base")
	}
	if key == nil {
		return Signed{}, reject(obs.ReasonKeyNotEligible, "no verification key")
	}
	if name, ok := p.Alg(); ok {
		a, err := crypto.ParseAlg(name)
		if err != nil || a != key.Alg() {
			return Signed{}, reject(obs.ReasonAlgorithm, "alg does not name the key's algorithm")
		}
	}
	sig, err := crypto.ParseSignature(key.Alg(), p.signature)
	if err != nil {
		return Signed{}, err
	}
	if err := crypto.Verify(key, sig, base.b); err != nil {
		return Signed{}, err
	}
	return Signed{p: p, response: base.response}, nil
}
