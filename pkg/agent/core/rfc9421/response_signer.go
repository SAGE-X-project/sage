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

package rfc9421

import (
	"bytes"
	"crypto"
	"net/http"
	"strconv"
	"time"
)

// ResponseSigner signs the responses an http.Handler produces so that a client
// can verify with VerifyResponse that a tool result or agent reply came from
// the holder of the key and answers exactly the request it sent.
//
// The middleware buffers the handler's output, sets Content-Digest over the
// body, and signs `"@status"`, `"content-type"` (when set), `"content-digest"`
// (when there is a body), and the request-binding components `"@method";req`,
// `"@target-uri";req`, `"@authority";req`, `"content-digest";req` (when the
// request carried one) and `"signature";req` (when the request was signed).
type ResponseSigner struct {
	Verifier   *HTTPVerifier
	PrivateKey crypto.Signer
	// KeyID is placed in the keyid parameter (for example the agent's DID key identifier).
	KeyID string
	// Algorithm is placed in the alg parameter (for example "ed25519", "es256k").
	Algorithm string
	// SignatureName is the label used for the signature; "sig1" when empty.
	SignatureName string
	// OnError, when set, is called if signing fails; the response is then sent
	// unsigned with status 500. When nil, the same happens silently.
	OnError func(*http.Request, error)
}

// Wrap returns a handler that signs every response written by next.
func (s *ResponseSigner) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &bufferedResponse{header: http.Header{}, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		resp := &http.Response{StatusCode: rec.status, Header: rec.header, Request: r}
		body := rec.body.Bytes()
		if len(body) > 0 {
			resp.Header.Set("Content-Digest", ComputeContentDigest(body))
			resp.ContentLength = int64(len(body))
		}

		params := &SignatureInputParams{
			CoveredComponents: s.coveredComponents(r, resp, len(body) > 0),
			KeyID:             s.KeyID,
			Algorithm:         s.Algorithm,
			Created:           time.Now().Unix(),
		}
		name := s.SignatureName
		if name == "" {
			name = "sig1"
		}
		verifier := s.Verifier
		if verifier == nil {
			verifier = NewHTTPVerifierWithReplayGuard(nil)
		}
		if err := verifier.SignResponse(resp, r, name, params, s.PrivateKey); err != nil {
			if s.OnError != nil {
				s.OnError(r, err)
			}
			http.Error(w, "failed to sign response", http.StatusInternalServerError)
			return
		}

		for k, vs := range resp.Header {
			w.Header()[k] = vs
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(rec.status)
		_, _ = w.Write(body)
	})
}

func (s *ResponseSigner) coveredComponents(r *http.Request, resp *http.Response, hasBody bool) []string {
	covered := []string{`"@status"`, `"@method";req`, `"@target-uri";req`, `"@authority";req`}
	if r.Header.Get("Content-Digest") != "" {
		covered = append(covered, `"content-digest";req`)
	}
	if r.Header.Get("Signature") != "" {
		covered = append(covered, `"signature";req`)
	}
	if resp.Header.Get("Content-Type") != "" {
		covered = append(covered, `"content-type"`)
	}
	if hasBody {
		covered = append(covered, `"content-digest"`)
	}
	return covered
}

// bufferedResponse captures a handler's status, headers and body.
type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
	wrote  bool
}

func (b *bufferedResponse) Header() http.Header { return b.header }

func (b *bufferedResponse) WriteHeader(status int) {
	if !b.wrote {
		b.status = status
		b.wrote = true
	}
}

func (b *bufferedResponse) Write(p []byte) (int, error) {
	b.wrote = true
	return b.body.Write(p)
}
