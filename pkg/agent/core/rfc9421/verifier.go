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
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/message/nonce"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/session"
)

// Verifier provides RFC-9421 signature verification
type Verifier struct {
	httpVerifier *HTTPVerifier
	replay       session.ReplayGuard // nonce replay protection, scoped by agent DID
}

// NewVerifier creates a verifier with an in-memory replay guard (5 minute window).
func NewVerifier() *Verifier {
	return &Verifier{
		httpVerifier: NewHTTPVerifier(),
		replay:       session.NewMemoryReplayGuard(5 * time.Minute),
	}
}

// NewVerifierWithReplayGuard creates a verifier that records nonces in the
// given guard (for example a shared or persistent store). A nil guard
// disables replay detection on the envelope path.
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

// replayScope is the nonce space a message's nonce is checked in.
func replayScope(message *Message) string { return message.AgentDID }

// VerifySignature verifies a signature according to RFC-9421
func (v *Verifier) VerifySignature(publicKey interface{}, message *Message, opts *VerificationOptions) error {
	if opts == nil {
		opts = DefaultVerificationOptions()
	}

	// Verify timestamp is within acceptable range
	if opts.MaxClockSkew > 0 {
		now := time.Now()
		diff := now.Sub(message.Timestamp)
		if diff < -opts.MaxClockSkew || diff > opts.MaxClockSkew {
			return fmt.Errorf("message timestamp outside acceptable range: %v", diff)
		}
	}

	// Fail fast on a nonce that is already recorded; the authoritative check
	// happens after signature verification so forged messages cannot poison
	// the guard.
	if message.Nonce != "" && v.replay != nil {
		if peek, ok := v.replay.(interface {
			Seen(scope, nonce string) bool
		}); ok && peek.Seen(replayScope(message), message.Nonce) {
			return fmt.Errorf("nonce replay attack detected: nonce %s has already been used", message.Nonce)
		}
	}

	// Construct the message to verify based on RFC-9421 partial signing
	signatureBase := v.ConstructSignatureBase(message)

	// Verify the signature
	if err := v.verifySignatureWithAlgorithm(publicKey, []byte(signatureBase), message.Signature, message.Algorithm); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Record the nonce after successful verification (atomic check-and-mark)
	if message.Nonce != "" && v.replay != nil {
		if !v.replay.CheckAndMark(replayScope(message), message.Nonce) {
			return fmt.Errorf("nonce replay attack detected: nonce %s has already been used", message.Nonce)
		}
	}

	return nil
}

// VerifyWithMetadata verifies a signature and checks metadata constraints
func (v *Verifier) VerifyWithMetadata(
	publicKey interface{},
	message *Message,
	expectedMetadata map[string]interface{},
	requiredCapabilities []string,
	opts *VerificationOptions,
) (*VerificationResult, error) {
	result := &VerificationResult{
		VerifiedAt: time.Now(),
	}

	// Basic signature verification
	if err := v.VerifySignature(publicKey, message, opts); err != nil {
		result.Valid = false
		result.Error = err.Error()
		return result, nil
	}

	// Verify metadata if requested
	if opts != nil && opts.VerifyMetadata && expectedMetadata != nil {
		if err := v.verifyMetadataMatch(message.Metadata, expectedMetadata); err != nil {
			result.Valid = false
			result.Error = fmt.Sprintf("metadata verification failed: %v", err)
			return result, nil
		}
	}

	// Check required capabilities
	if len(requiredCapabilities) > 0 && message.Metadata != nil {
		if capabilities, ok := message.Metadata["capabilities"].(map[string]interface{}); ok {
			if !hasRequiredCapabilities(capabilities, requiredCapabilities) {
				result.Valid = false
				result.Error = "agent missing required capabilities"
				return result, nil
			}
		} else {
			result.Valid = false
			result.Error = "agent capabilities not found in metadata"
			return result, nil
		}
	}

	result.Valid = true
	return result, nil
}

// ConstructSignatureBase builds the signature base string according to RFC-9421
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

// verifySignatureWithAlgorithm verifies the signature using the appropriate algorithm
func (v *Verifier) verifySignatureWithAlgorithm(publicKey interface{}, message, signature []byte, algorithm string) error {
	switch algorithm {
	case string(AlgorithmEdDSA):
		pk, ok := publicKey.(ed25519.PublicKey)
		if !ok {
			return fmt.Errorf("invalid public key type for EdDSA")
		}
		if !ed25519.Verify(pk, message, signature) {
			return fmt.Errorf("EdDSA signature verification failed")
		}

	case string(AlgorithmES256K), string(AlgorithmECDSA), string(AlgorithmECDSASecp256k1):
		ecdsaKey, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			return fmt.Errorf("invalid public key type for ECDSA")
		}

		// secp256k1 keys follow the Ethereum convention (Keccak-256, r||s[||v])
		// so that the same signature is valid for SAGE and for ecrecover.
		if keys.IsSecp256k1Curve(ecdsaKey.Curve) {
			if err := keys.VerifySecp256k1Keccak(ecdsaKey, message, signature); err != nil {
				return fmt.Errorf("ECDSA (secp256k1) signature verification failed: %w", err)
			}
			return nil
		}

		// Other curves (P-256): SHA-256 over the signature base, raw r || s.
		if len(signature) < 64 {
			return fmt.Errorf("invalid ECDSA signature length: %d", len(signature))
		}
		r := new(big.Int).SetBytes(signature[:32])
		s := new(big.Int).SetBytes(signature[32:64])
		hash := sha256.Sum256(message)
		if !ecdsa.Verify(ecdsaKey, hash[:], r, s) {
			return fmt.Errorf("ECDSA signature verification failed")
		}

	default:
		return fmt.Errorf("unsupported signature algorithm: %s", algorithm)
	}

	return nil
}

// verifyMetadataMatch checks if message metadata matches expected values
func (v *Verifier) verifyMetadataMatch(actual, expected map[string]interface{}) error {
	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		if !exists {
			return fmt.Errorf("missing expected metadata field: %s", key)
		}

		if !compareValues(expectedValue, actualValue) {
			return fmt.Errorf("metadata field %s mismatch", key)
		}
	}

	return nil
}

// VerifyHTTPRequest verifies an HTTP request signature according to RFC-9421
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

// Helper functions

func hasRequiredCapabilities(agentCaps map[string]interface{}, required []string) bool {
	for _, req := range required {
		if _, exists := agentCaps[req]; !exists {
			return false
		}
	}
	return true
}

func compareValues(v1, v2 interface{}) bool {
	// Simple comparison - can be enhanced for deep object comparison
	j1, _ := json.Marshal(v1)
	j2, _ := json.Marshal(v2)
	return string(j1) == string(j2)
}
