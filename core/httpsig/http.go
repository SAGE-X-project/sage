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
	"net/http"
	"strings"
)

// ErrOrigin is returned by TargetFrom for an origin that is not
// "http://" or "https://" followed by a normal authority and nothing else,
// and for a request whose request-target is not in origin form.
var ErrOrigin = errors.New("httpsig: invalid external origin or request target")

// FailureBody is the body of the 401 answer to every verification failure
// (03 §5, Failure reporting, E03-17).
const FailureBody = "authentication failed"

// ErrChallenge is returned by WriteAuthenticationFailure for an empty
// WWW-Authenticate value or one with a control byte.
var ErrChallenge = errors.New("httpsig: invalid WWW-Authenticate value")

// TargetFrom returns the @target-uri of a received request (03 §2, E03-22):
// the configured external origin, such as "https://agent-b.example",
// followed by the request-target as received (path and query, as sent). A
// verifier behind a proxy must use its external origin, never the internal
// Host or the scheme of the connection it received the request on.
func TargetFrom(origin string, r *http.Request) (string, error) {
	t, err := parseTarget(origin + "/")
	if err != nil || !t.normalAuthority() || origin != t.scheme+"://"+t.authority {
		return "", ErrOrigin
	}
	if r == nil || !strings.HasPrefix(r.RequestURI, "/") {
		return "", ErrOrigin
	}
	return origin + r.RequestURI, nil
}

// WriteAuthenticationFailure writes the one answer of 03 §5 (Failure
// reporting, E03-17) to every failure before the application handler:
// HTTP 401, the body "authentication failed" and the deployment's constant
// WWW-Authenticate value (RFC 9110 §15.5.2). Nothing in it depends on which
// check failed; log the rejection's reason locally instead. An endpoint
// that carries 08-transport.md envelopes answers as 08 §2 fixes instead.
func WriteAuthenticationFailure(w http.ResponseWriter, challenge string) error {
	if challenge == "" || unsafeValue(challenge) {
		return ErrChallenge
	}
	h := w.Header()
	for k := range h {
		delete(h, k)
	}
	h.Set("WWW-Authenticate", challenge)
	h.Set("Content-Type", "text/plain; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusUnauthorized)
	_, err := w.Write([]byte(FailureBody))
	return err
}
