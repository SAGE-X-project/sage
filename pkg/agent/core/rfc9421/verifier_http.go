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
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/message/nonce"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	_ "github.com/sage-x-project/sage/pkg/agent/crypto/keys" // Import to register algorithms
)

// DefaultMaxClockSkew bounds how far in the future a "created" parameter may
// be. It mirrors the default MaxAge so the acceptance window is symmetric
// (5 minutes either side of the verifier's clock).
const DefaultMaxClockSkew = 5 * time.Minute

// ReplayGuard remembers (keyID, nonce) pairs for the verification window and
// reports reuse. CheckAndMark returns true when the pair has not been seen and
// records it; it returns false when the pair was already presented.
type ReplayGuard interface {
	CheckAndMark(keyID, nonce string) bool
}

// NonceReplayGuard is the in-memory ReplayGuard used by NewHTTPVerifier.
type NonceReplayGuard struct {
	m *nonce.Manager
}

// NewNonceReplayGuard returns an in-memory replay guard that forgets nonces
// after ttl. Call Close to stop its cleanup goroutine.
func NewNonceReplayGuard(ttl time.Duration) *NonceReplayGuard {
	cleanup := ttl / 2
	if cleanup < time.Second {
		cleanup = time.Second
	}
	return &NonceReplayGuard{m: nonce.NewManager(ttl, cleanup)}
}

// CheckAndMark implements ReplayGuard.
func (g *NonceReplayGuard) CheckAndMark(keyID, n string) bool {
	return g.m.CheckAndMark(keyID + "\x00" + n)
}

// Close stops the guard's cleanup goroutine.
func (g *NonceReplayGuard) Close() { g.m.Close() }

// HTTPVerifier provides RFC-9421 HTTP message signature verification
type HTTPVerifier struct {
	canonicalizer *Canonicalizer
	replay        ReplayGuard
	ownsGuard     bool
}

// NewHTTPVerifier creates a new HTTP signature verifier with an in-memory
// replay guard whose window matches DefaultHTTPVerificationOptions().MaxAge.
// Call Close when the verifier is no longer needed.
func NewHTTPVerifier() *HTTPVerifier {
	return &HTTPVerifier{
		canonicalizer: NewCanonicalizer(),
		replay:        NewNonceReplayGuard(DefaultHTTPVerificationOptions().MaxAge),
		ownsGuard:     true,
	}
}

// NewHTTPVerifierWithReplayGuard creates a verifier that records nonces in the
// given guard (for example a shared or persistent store). A nil guard disables
// replay detection.
func NewHTTPVerifierWithReplayGuard(guard ReplayGuard) *HTTPVerifier {
	return &HTTPVerifier{
		canonicalizer: NewCanonicalizer(),
		replay:        guard,
	}
}

// Close releases the verifier's own replay guard, if it created one.
func (v *HTTPVerifier) Close() {
	if v.ownsGuard {
		if c, ok := v.replay.(interface{ Close() }); ok {
			c.Close()
		}
	}
}

// SignRequest signs an HTTP request according to RFC 9421
func (v *HTTPVerifier) SignRequest(req *http.Request, sigName string, params *SignatureInputParams, privateKey crypto.Signer) error {
	// Build signature base
	signatureBase, err := v.canonicalizer.BuildSignatureBase(req, sigName, params)
	if err != nil {
		return fmt.Errorf("failed to build signature base: %w", err)
	}

	// Sign the signature base differently based on key type
	var signature []byte

	switch key := privateKey.(type) {
	case ed25519.PrivateKey:
		// Ed25519 signs the message directly, not a hash
		signature = ed25519.Sign(key, []byte(signatureBase))

	case *ecdsa.PrivateKey:
		// ECDSA requires hashed signing
		h := sha256.New()
		h.Write([]byte(signatureBase))
		digest := h.Sum(nil)

		r, s, err := ecdsa.Sign(rand.Reader, key, digest)
		if err != nil {
			return fmt.Errorf("failed to sign with ECDSA: %w", err)
		}

		// Convert to fixed-size byte arrays (P-256 = 32 bytes each)
		signature = make([]byte, 64)
		rBytes := r.Bytes()
		sBytes := s.Bytes()

		// Pad with zeros if necessary
		copy(signature[32-len(rBytes):32], rBytes)
		copy(signature[64-len(sBytes):64], sBytes)

	default:
		// Other algorithms use the standard crypto.Signer interface
		h := sha256.New()
		h.Write([]byte(signatureBase))
		digest := h.Sum(nil)

		signature, err = privateKey.Sign(rand.Reader, digest, crypto.SHA256)
		if err != nil {
			return fmt.Errorf("failed to sign: %w", err)
		}
	}

	// Set Signature-Input header
	inputHeader := v.formatSignatureInput(sigName, params)
	req.Header.Set("Signature-Input", inputHeader)

	// Set Signature header
	sigHeader := fmt.Sprintf("%s=:%s:", sigName, base64.StdEncoding.EncodeToString(signature))
	req.Header.Set("Signature", sigHeader)

	return nil
}

// VerifyRequest verifies an HTTP request signature
func (v *HTTPVerifier) VerifyRequest(req *http.Request, publicKey crypto.PublicKey, opts *HTTPVerificationOptions) error {
	if opts == nil {
		opts = DefaultHTTPVerificationOptions()
	}

	// Parse Signature-Input header
	inputHeader := req.Header.Get("Signature-Input")
	if inputHeader == "" {
		return fmt.Errorf("missing Signature-Input header")
	}

	sigInputs, err := ParseSignatureInput(inputHeader)
	if err != nil {
		return fmt.Errorf("failed to parse Signature-Input: %w", err)
	}

	// Parse Signature header
	sigHeader := req.Header.Get("Signature")
	if sigHeader == "" {
		return fmt.Errorf("missing Signature header")
	}

	signatures, err := ParseSignature(sigHeader)
	if err != nil {
		return fmt.Errorf("failed to parse Signature: %w", err)
	}

	// Find the signature to verify. Without an explicit name, pick the
	// lexicographically first label so the outcome does not depend on map order.
	var sigName string
	if opts.SignatureName != "" {
		sigName = opts.SignatureName
	} else {
		names := make([]string, 0, len(sigInputs))
		for name := range sigInputs {
			names = append(names, name)
		}
		sort.Strings(names)
		if len(names) > 0 {
			sigName = names[0]
		}
	}

	params, exists := sigInputs[sigName]
	if !exists {
		return fmt.Errorf("signature '%s' not found in Signature-Input", sigName)
	}

	signature, exists := signatures[sigName]
	if !exists {
		return fmt.Errorf("signature '%s' not found in Signature header", sigName)
	}

	// Check created/expires if present
	now := time.Now().Unix()
	if params.Created > 0 && opts.MaxAge > 0 {
		age := now - params.Created
		if age > int64(opts.MaxAge.Seconds()) {
			return fmt.Errorf("signature expired: created %d seconds ago (max %d)", age, int64(opts.MaxAge.Seconds()))
		}
	}

	if params.Created > 0 {
		skew := opts.MaxClockSkew
		if skew <= 0 {
			skew = DefaultMaxClockSkew
		}
		if params.Created > now+int64(skew.Seconds()) {
			return fmt.Errorf("signature created in the future: created=%d now=%d (max skew %s)", params.Created, now, skew)
		}
	}

	if params.Expires > 0 && now > params.Expires {
		return fmt.Errorf("signature expired at %d (now %d)", params.Expires, now)
	}

	// Required components: the verifier, not the signer, decides what must be covered.
	required := append([]string(nil), opts.RequiredComponents...)
	if opts.RequireContentDigest && requestHasBody(req) {
		required = append(required, "content-digest")
	}
	for _, comp := range required {
		if !IsComponentCovered(params.CoveredComponents, comp) {
			return fmt.Errorf("required component %q is not covered by the signature", strings.Trim(comp, `"`))
		}
	}
	if opts.RequireNonce && params.Nonce == "" {
		return fmt.Errorf("signature nonce is required but missing")
	}

	// Validate body integrity if Content-Digest is covered by signature
	// This prevents body tampering attacks where the body is modified but
	// the Content-Digest header remains unchanged (PR #118 security fix)
	bodyValidator := NewBodyIntegrityValidator()
	if err := bodyValidator.ValidateContentDigest(req, params.CoveredComponents); err != nil {
		return fmt.Errorf("body integrity validation failed: %w", err)
	}

	// Build signature base
	signatureBase, err := v.canonicalizer.BuildSignatureBase(req, sigName, params)
	if err != nil {
		return fmt.Errorf("failed to build signature base: %w", err)
	}

	// Verify signature
	if err := v.verifySignature(publicKey, []byte(signatureBase), signature, params.Algorithm); err != nil {
		return err
	}

	// Replay detection runs last so that unverifiable requests cannot poison the
	// nonce store. The nonce is scoped by keyid so distinct signers do not collide.
	if params.Nonce != "" && v.replay != nil && !opts.DisableReplayCheck {
		if !v.replay.CheckAndMark(params.KeyID, params.Nonce) {
			return fmt.Errorf("signature replay detected: nonce %q was already used for keyid %q", params.Nonce, params.KeyID)
		}
	}
	return nil
}

// requestHasBody reports whether the request carries a body that a signature
// should bind through content-digest.
func requestHasBody(req *http.Request) bool {
	if req.ContentLength > 0 {
		return true
	}
	return req.Body != nil && req.Body != http.NoBody && req.ContentLength != 0
}

// verifySignature verifies the actual cryptographic signature
func (v *HTTPVerifier) verifySignature(publicKey crypto.PublicKey, message, signature []byte, algorithm string) error {
	// Validate algorithm compatibility with public key using the registry
	if err := sagecrypto.ValidateAlgorithmForPublicKey(publicKey, algorithm); err != nil {
		return fmt.Errorf("algorithm validation failed: %w", err)
	}

	// Hash the message
	h := sha256.New()
	h.Write(message)
	digest := h.Sum(nil)

	switch key := publicKey.(type) {
	case ed25519.PublicKey:
		if !ed25519.Verify(key, message, signature) {
			return fmt.Errorf("ed25519 signature verification failed")
		}

	case *ecdsa.PublicKey:
		// ECDSA signatures should be ASN.1 DER encoded
		var r, s *big.Int
		r, s, err := parseECDSASignature(signature)
		if err != nil {
			return fmt.Errorf("failed to parse ECDSA signature: %w", err)
		}

		if !ecdsa.Verify(key, digest, r, s) {
			return fmt.Errorf("ECDSA signature verification failed")
		}

	case *rsa.PublicKey:
		err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest, signature)
		if err != nil {
			return fmt.Errorf("RSA signature verification failed: %w", err)
		}

	default:
		return fmt.Errorf("unsupported key type: %T", publicKey)
	}

	return nil
}

// formatSignatureInput formats the Signature-Input header value
func (v *HTTPVerifier) formatSignatureInput(sigName string, params *SignatureInputParams) string {
	components := make([]string, len(params.CoveredComponents))
	copy(components, params.CoveredComponents)

	result := fmt.Sprintf("%s=(%s)", sigName, strings.Join(components, " "))

	if params.KeyID != "" {
		result += fmt.Sprintf(`;keyid="%s"`, params.KeyID)
	}
	if params.Algorithm != "" {
		result += fmt.Sprintf(`;alg="%s"`, params.Algorithm)
	}
	if params.Created > 0 {
		result += fmt.Sprintf(`;created=%d`, params.Created)
	}
	if params.Expires > 0 {
		result += fmt.Sprintf(`;expires=%d`, params.Expires)
	}
	if params.Nonce != "" {
		result += fmt.Sprintf(`;nonce="%s"`, params.Nonce)
	}

	return result
}

// HTTPVerificationOptions contains options for HTTP signature verification
type HTTPVerificationOptions struct {
	// SignatureName specifies which signature to verify (if multiple exist).
	// When empty, the lexicographically first signature label is used.
	SignatureName string

	// MaxAge specifies the maximum age for created timestamps
	MaxAge time.Duration

	// MaxClockSkew bounds how far in the future "created" may be. Zero means
	// DefaultMaxClockSkew.
	MaxClockSkew time.Duration

	// RequiredComponents lists components that the signature must cover
	// (for example "@method", "@target-uri", "content-digest"). Quotes optional.
	RequiredComponents []string

	// RequireContentDigest additionally requires "content-digest" to be covered
	// whenever the request has a body, binding the body to the signature.
	RequireContentDigest bool

	// RequireNonce rejects signatures without a nonce parameter.
	RequireNonce bool

	// DisableReplayCheck skips the verifier's replay guard (tests, offline replay).
	DisableReplayCheck bool
}

// DefaultHTTPVerificationOptions returns lenient options: signatures must be
// fresh (5 minutes), may not be dated in the future, and a nonce, when present,
// is accepted once. Component coverage is left to the signer; use
// StrictHTTPVerificationOptions to make the verifier enforce it.
func DefaultHTTPVerificationOptions() *HTTPVerificationOptions {
	return &HTTPVerificationOptions{
		MaxAge:       5 * time.Minute,
		MaxClockSkew: DefaultMaxClockSkew,
	}
}

// StrictHTTPVerificationOptions returns options that bind the method, target,
// authority and (for requests with a body) the body to the signature and
// require a nonce. This is the recommended setting for agent-to-agent traffic.
func StrictHTTPVerificationOptions() *HTTPVerificationOptions {
	return &HTTPVerificationOptions{
		MaxAge:               5 * time.Minute,
		MaxClockSkew:         DefaultMaxClockSkew,
		RequiredComponents:   []string{"@method", "@target-uri", "@authority"},
		RequireContentDigest: true,
		RequireNonce:         true,
	}
}

// parseECDSASignature parses an ECDSA signature
func parseECDSASignature(sig []byte) (r, s *big.Int, err error) {
	// For P-256, we expect 64 bytes (32 bytes each for r and s)
	if len(sig) == 64 {
		r = new(big.Int).SetBytes(sig[:32])
		s = new(big.Int).SetBytes(sig[32:])
		return r, s, nil
	}

	// Handle ASN.1 DER encoded signatures
	// This is a simplified parser - in production use crypto/x509
	if len(sig) < 8 || sig[0] != 0x30 {
		return nil, nil, fmt.Errorf("invalid ECDSA signature format")
	}

	return nil, nil, fmt.Errorf("ASN.1 parsing not implemented")
}
