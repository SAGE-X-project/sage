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
	"errors"

	"github.com/sage-x-project/sage/core/crypto"
	"github.com/sage-x-project/sage/core/obs"
)

// ErrUnsignedRequest is returned by CheckResponse and ResponseBase for a
// request without a signature: a response is verified only for a signed
// request (E03-20), so this is a fault of the caller, not of the peer.
var ErrUnsignedRequest = errors.New("httpsig: the request the response answers is not signed")

// The derived components a signature may cover (03 §2). A response covers
// @status and the request's derived components with ;req.
const (
	compMethod    = "@method"
	compTargetURI = "@target-uri"
	compAuthority = "@authority"
	compScheme    = "@scheme"
	compQuery     = "@query"
	compStatus    = "@status"
)

func requestDerived(name string) bool {
	switch name {
	case compMethod, compTargetURI, compAuthority, compScheme, compQuery:
		return true
	}
	return false
}

// CheckRequest is step 2 of 03 §5 for a request signature: the parameters
// and covered components of §1 to §3. It requires keyid, created and a
// nonce of the syntax of §5 (Replay); the coverage of @method, @target-uri
// and @authority, and of
// content-digest when the request has a body; only the derived components
// @method, @target-uri, @authority, @scheme and @query; no component
// parameter (;req is for responses, ;sf and the others are not allowed);
// and an @authority in its normal form.
//
// It does not verify anything; a request is verified only when every step
// of 03 §5 has passed.
func CheckRequest(p Parsed, r Request) error {
	if !p.keyid.ok || !p.created.ok || !p.nonce.ok {
		return reject(obs.ReasonParameters, "request signature lacks keyid, created or nonce")
	}
	if !validNonce(p.nonce.v) {
		return reject(obs.ReasonParameters, "nonce is not 16 to 64 characters of ALPHA, DIGIT, - and _")
	}
	if err := checkAlg(p); err != nil {
		return err
	}
	covered := map[string]bool{}
	for _, c := range p.components {
		isReq, ok := componentParams(c)
		switch {
		case !ok:
			return coverage("covered component has a parameter other than ;req")
		case isReq:
			return coverage(";req in a request signature")
		}
		if err := checkName(c.name, requestDerived); err != nil {
			return err
		}
		if covered[c.name] {
			return coverage("component covered twice")
		}
		covered[c.name] = true
	}
	if !covered[compMethod] || !covered[compTargetURI] || !covered[compAuthority] {
		return coverage("request signature does not cover @method, @target-uri and @authority")
	}
	if len(r.Body) > 0 && !covered[fieldContentDigestLower] {
		return coverage("request with a body does not cover content-digest")
	}
	return checkAuthority(r.Target)
}

// CheckResponse is step 2 of 03 §5 for a response signature, given the
// signed request it answers (E03-20). It requires keyid and created, and a
// nonce, when present, of the syntax of §5 (Replay); the coverage of
// @status, "@method";req, "@target-uri";req, "@authority";req
// and "signature";req, of "content-digest";req when the request carried
// Content-Digest, and of content-digest when the
// response has a body (§4); derived components only as §2 allows them in a
// response; no component parameter other than ;req; and the request's
// @authority in its normal form.
//
// It does not verify anything.
func CheckResponse(p Parsed, resp Response, req Request) error {
	if !p.keyid.ok || !p.created.ok {
		return reject(obs.ReasonParameters, "response signature lacks keyid or created")
	}
	if p.nonce.ok && !validNonce(p.nonce.v) {
		return reject(obs.ReasonParameters, "nonce is not 16 to 64 characters of ALPHA, DIGIT, - and _")
	}
	if err := checkAlg(p); err != nil {
		return err
	}
	// A response is verified only for a signed request (E03-20), and binds
	// it with "signature";req (03 §4, E03-10). A request with a body
	// carries Content-Digest (§3), so a body without one fails closed.
	if !hasField(req.Header, FieldSignature) {
		return ErrUnsignedRequest
	}
	requestDigest := hasField(req.Header, FieldContentDigest)
	if len(req.Body) > 0 && !requestDigest {
		return coverage("the request has a body but no Content-Digest")
	}
	covered := map[string]bool{}
	for _, c := range p.components {
		isReq, ok := componentParams(c)
		if !ok {
			return coverage("covered component has a parameter other than ;req")
		}
		allowed := func(name string) bool { return name == compStatus }
		if isReq {
			allowed = requestDerived
		}
		if err := checkName(c.name, allowed); err != nil {
			return err
		}
		id := c.Identifier()
		if covered[id] {
			return coverage("component covered twice")
		}
		covered[id] = true
	}
	required := []string{`"@status"`, `"@method";req`, `"@target-uri";req`, `"@authority";req`, `"signature";req`}
	if requestDigest {
		required = append(required, `"content-digest";req`)
	}
	if len(resp.Body) > 0 {
		required = append(required, `"content-digest"`)
	}
	for _, id := range required {
		if !covered[id] {
			return coverage("response signature does not cover " + id)
		}
	}
	return checkAuthority(req.Target)
}

// componentParams returns whether c carries ;req, and false for ok when it
// carries any other parameter, a repeated one, or req with a value other
// than true (03 §2, E03-09).
func componentParams(c Component) (isReq, ok bool) {
	switch {
	case len(c.params) == 0:
		return false, true
	case len(c.params) == 1 && c.params[0].key == "req" &&
		c.params[0].value.kind == kindBoolean && c.params[0].value.boolean:
		return true, true
	}
	return false, false
}

// checkName accepts a derived component that allowed admits and a
// lower-case field name. @request-target, @path, @query-param and every
// other derived component are rejected (03 §2, E03-08).
func checkName(name string, allowed func(string) bool) error {
	if len(name) > 0 && name[0] == '@' {
		if !allowed(name) {
			return coverage("derived component not allowed here")
		}
		return nil
	}
	if !validFieldName(name) {
		return coverage("covered field name is not a lower-case token")
	}
	return nil
}

// checkAlg rejects an alg that names no algorithm of this profile
// (01 §2); whether it names the key's algorithm is checked with the key.
func checkAlg(p Parsed) error {
	if name, ok := p.Alg(); ok {
		if _, err := crypto.ParseAlg(name); err != nil {
			return reject(obs.ReasonAlgorithm, "alg names no algorithm of this profile")
		}
	}
	return nil
}

func checkAuthority(targetURI string) error {
	t, err := parseTarget(targetURI)
	if err != nil {
		return err
	}
	if !t.normalAuthority() {
		return coverage("@authority is not the lower-case host with a non-default port")
	}
	return nil
}

const fieldContentDigestLower = "content-digest"

// validNonce is the nonce syntax of 03 §5 (Replay): 16 to 64 characters of
// ALPHA / DIGIT / "-" / "_". The signer and the verifier both apply it.
func validNonce(n string) bool {
	if len(n) < 16 || len(n) > 64 {
		return false
	}
	for i := 0; i < len(n); i++ {
		if c := n[i]; !isAlpha(c) && !isDigit(c) && c != '-' && c != '_' {
			return false
		}
	}
	return true
}
