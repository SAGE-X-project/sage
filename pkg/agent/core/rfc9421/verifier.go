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
	"crypto"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/message/nonce"
	"github.com/sage-x-project/sage/pkg/agent/session"
)

// Verifier provides RFC-9421 signature verification
type Verifier struct {
	httpVerifier *HTTPVerifier
	replay       session.ReplayGuard // used only by the removed Message format path (W-010)
}

// NewVerifier creates a verifier with an in-memory replay guard (5 minute window).
func NewVerifier() *Verifier {
	return &Verifier{
		httpVerifier: NewHTTPVerifier(),
		replay:       session.NewMemoryReplayGuard(5 * time.Minute),
	}
}

// NewVerifierWithReplayGuard creates a verifier with the given guard. The
// guard served only the removed Message format path and is not used; the
// HTTP methods use the replay protection of their HTTPVerifier.
func NewVerifierWithReplayGuard(guard session.ReplayGuard) *Verifier {
	return &Verifier{httpVerifier: NewHTTPVerifier(), replay: guard}
}

// NewVerifierWithNonceManager creates a verifier backed by a nonce.Manager.
//
// Deprecated: use NewVerifierWithReplayGuard with a session.ReplayGuard.
func NewVerifierWithNonceManager(nonceManager *nonce.Manager) *Verifier {
	if nonceManager == nil {
		return NewVerifierWithReplayGuard(nil)
	}
	return NewVerifierWithReplayGuard(nonceManagerGuard{m: nonceManager})
}

// nonceManagerGuard adapts the legacy nonce.Manager to session.ReplayGuard.
// The manager keeps a single global nonce space, so the scope is ignored to
// preserve its historical behaviour (callers can still query IsNonceUsed).
type nonceManagerGuard struct{ m *nonce.Manager }

func (g nonceManagerGuard) CheckAndMark(_, n string) bool { return g.m.CheckAndMark(n) }
func (g nonceManagerGuard) Seen(_, n string) bool         { return g.m.IsNonceUsed(n) }

// ErrMessageFormatRemoved is returned by every verifier of the SAGE "Message"
// signature format. The format is not part of SAGE 0.9.0-draft.1.1, which
// accepts only RFC 9421 HTTP message signatures (errata E03-19), and its
// verifier accepted forgeries by field aliasing (security review C1). The
// verifiers refuse every message until they are removed (W-010, W-079).
var ErrMessageFormatRemoved = errors.New("rfc9421: the SAGE Message signature format is not accepted (SAGE 0.9.0-draft.1.1, E03-19); use RFC 9421 HTTP message signatures")

// VerifySignature always returns ErrMessageFormatRemoved.
//
// Deprecated: the SAGE Message format is not part of SAGE 0.9.0-draft.1.1;
// sign and verify HTTP requests with SignHTTPRequest and VerifyHTTPRequest.
func (v *Verifier) VerifySignature(publicKey interface{}, message *Message, opts *VerificationOptions) error {
	return ErrMessageFormatRemoved
}

// VerifyWithMetadata always reports an invalid result and returns
// ErrMessageFormatRemoved, so a caller that checks either the error or
// Result.Valid refuses the message.
//
// Deprecated: the SAGE Message format is not part of SAGE 0.9.0-draft.1.1.
func (v *Verifier) VerifyWithMetadata(
	publicKey interface{},
	message *Message,
	expectedMetadata map[string]interface{},
	requiredCapabilities []string,
	opts *VerificationOptions,
) (*VerificationResult, error) {
	return &VerificationResult{Valid: false, Error: ErrMessageFormatRemoved.Error(), VerifiedAt: time.Now()}, ErrMessageFormatRemoved
}

// ConstructSignatureBase builds the signature base of the legacy SAGE
// "Message" format. It is not an RFC 9421 signature base, and no verifier
// accepts signatures over it (E03-19).
func (v *Verifier) ConstructSignatureBase(msg *Message) string {
	// RFC-9421 allows partial signing of message components
	var parts []string

	for _, field := range msg.SignedFields {
		switch field {
		case "agent_did":
			parts = append(parts, fmt.Sprintf("agent_did: %s", msg.AgentDID))
		case "message_id":
			parts = append(parts, fmt.Sprintf("message_id: %s", msg.MessageID))
		case "timestamp":
			parts = append(parts, fmt.Sprintf("timestamp: %s", msg.Timestamp.Format(time.RFC3339)))
		case "nonce":
			parts = append(parts, fmt.Sprintf("nonce: %s", msg.Nonce))
		case "body":
			parts = append(parts, fmt.Sprintf("body: %s", string(msg.Body)))
		default:
			// Check if it's a header field
			if strings.HasPrefix(field, "header.") {
				headerName := strings.TrimPrefix(field, "header.")
				if value, ok := msg.Headers[headerName]; ok {
					parts = append(parts, fmt.Sprintf("%s: %s", headerName, value))
				}
			}
		}
	}

	return strings.Join(parts, "\n")
}

// VerifyHTTPRequest verifies an HTTP request signature according to RFC-9421.
// With nil opts it uses DefaultHTTPVerificationOptions, which require no
// covered components, nonce or created (security review H4); the strict
// verifier of the core replaces this path (W-012).
func (v *Verifier) VerifyHTTPRequest(req *http.Request, publicKey interface{}, opts *HTTPVerificationOptions) error {
	return v.httpVerifier.VerifyRequest(req, publicKey, opts)
}

// SignHTTPRequest signs an HTTP request according to RFC-9421
func (v *Verifier) SignHTTPRequest(req *http.Request, sigName string, params *SignatureInputParams, privateKey interface{}) error {
	signer, ok := privateKey.(crypto.Signer)
	if !ok {
		return fmt.Errorf("private key must implement crypto.Signer interface")
	}
	return v.httpVerifier.SignRequest(req, sigName, params, signer)
}

// VerifyHTTPResponse verifies an HTTP response signature according to RFC-9421.
// req is the request the response answers (resp.Request when nil).
func (v *Verifier) VerifyHTTPResponse(resp *http.Response, req *http.Request, publicKey interface{}, opts *HTTPVerificationOptions) error {
	return v.httpVerifier.VerifyResponse(resp, req, publicKey, opts)
}

// SignHTTPResponse signs an HTTP response according to RFC-9421, binding it to req.
func (v *Verifier) SignHTTPResponse(resp *http.Response, req *http.Request, sigName string, params *SignatureInputParams, privateKey interface{}) error {
	signer, ok := privateKey.(crypto.Signer)
	if !ok {
		return fmt.Errorf("private key must implement crypto.Signer interface")
	}
	return v.httpVerifier.SignResponse(resp, req, sigName, params, signer)
}
