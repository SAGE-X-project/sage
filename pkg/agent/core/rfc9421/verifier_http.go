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

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/session"
)

// DefaultMaxClockSkew bounds how far in the future a "created" parameter may
// be. It mirrors the default MaxAge so the acceptance window is symmetric
// (5 minutes either side of the verifier's clock).
const DefaultMaxClockSkew = 5 * time.Minute

// ReplayGuard is the replay-protection contract shared across SAGE
// (session.ReplayGuard). The HTTP verifier scopes nonces by keyid.
type ReplayGuard = session.ReplayGuard

// NonceReplayGuard is the in-memory ReplayGuard used by NewHTTPVerifier.
//
// Deprecated: it is session.MemoryReplayGuard; use that name.
type NonceReplayGuard = session.MemoryReplayGuard

// NewNonceReplayGuard returns an in-memory replay guard that forgets nonces
// after ttl. Call Close to stop its sweeper.
func NewNonceReplayGuard(ttl time.Duration) *NonceReplayGuard {
	return session.NewMemoryReplayGuard(ttl)
}

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
	signature, err := v.sign(signatureBase, privateKey)
	if err != nil {
		return err
	}
	v.setSignatureHeaders(req.Header, sigName, params, signature)
	return nil
}

// SignResponse signs an HTTP response according to RFC 9421. req is the
// request the response answers; covered components carrying the ";req"
// parameter (for example `"@method";req`, `"@target-uri";req`,
// `"content-digest";req`, `"signature";req`) are taken from it and bind the
// response to that exact request. resp.Header is created if nil.
func (v *HTTPVerifier) SignResponse(resp *http.Response, req *http.Request, sigName string, params *SignatureInputParams, privateKey crypto.Signer) error {
	if resp == nil {
		return fmt.Errorf("response is nil")
	}
	if resp.Header == nil {
		resp.Header = http.Header{}
	}
	signatureBase, err := v.canonicalizer.BuildResponseSignatureBase(resp, req, sigName, params)
	if err != nil {
		return fmt.Errorf("failed to build signature base: %w", err)
	}
	signature, err := v.sign(signatureBase, privateKey)
	if err != nil {
		return err
	}
	v.setSignatureHeaders(resp.Header, sigName, params, signature)
	return nil
}

// setSignatureHeaders writes the Signature-Input and Signature fields.
func (v *HTTPVerifier) setSignatureHeaders(h http.Header, sigName string, params *SignatureInputParams, signature []byte) {
	h.Set("Signature-Input", v.formatSignatureInput(sigName, params))
	h.Set("Signature", fmt.Sprintf("%s=:%s:", sigName, base64.StdEncoding.EncodeToString(signature)))
}

// sign produces the signature over a signature base for the given key type.
func (v *HTTPVerifier) sign(signatureBase string, privateKey crypto.Signer) ([]byte, error) {
	var signature []byte
	var err error

	switch key := privateKey.(type) {
	case ed25519.PrivateKey:
		// Ed25519 signs the message directly, not a hash
		signature = ed25519.Sign(key, []byte(signatureBase))

	case *ecdsa.PrivateKey:
		if keys.IsSecp256k1Curve(key.Curve) {
			// Ethereum convention: Keccak-256, deterministic, r || s || v.
			signature, err = keys.SignSecp256k1Keccak(key, []byte(signatureBase))
			if err != nil {
				return nil, fmt.Errorf("failed to sign with secp256k1: %w", err)
			}
			break
		}
		// Other curves (P-256): SHA-256 digest, raw r || s.
		h := sha256.New()
		h.Write([]byte(signatureBase))
		digest := h.Sum(nil)

		r, s, err := ecdsa.Sign(rand.Reader, key, digest)
		if err != nil {
			return nil, fmt.Errorf("failed to sign with ECDSA: %w", err)
		}
		// Fixed-size r || s (P-256: 32 bytes each), low-S normalised
		signature = keys.EncodeRawECDSASignature(key.Curve, r, s)

	default:
		// Other algorithms use the standard crypto.Signer interface
		h := sha256.New()
		h.Write([]byte(signatureBase))
		digest := h.Sum(nil)

		signature, err = privateKey.Sign(rand.Reader, digest, crypto.SHA256)
		if err != nil {
			return nil, fmt.Errorf("failed to sign: %w", err)
		}
	}
	return signature, nil
}

// VerifyRequest verifies an HTTP request signature
func (v *HTTPVerifier) VerifyRequest(req *http.Request, publicKey crypto.PublicKey, opts *HTTPVerificationOptions) error {
	if opts == nil {
		opts = DefaultHTTPVerificationOptions()
	}
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	sigName, params, signature, err := selectSignature(req.Header, opts)
	if err != nil {
		return err
	}
	if err := checkSignatureTimes(params, opts); err != nil {
		return err
	}

	// Required components: the verifier, not the signer, decides what must be covered.
	required := append([]string(nil), opts.RequiredComponents...)
	if opts.RequireContentDigest && requestHasBody(req) {
		required = append(required, "content-digest")
	}
	if len(opts.ExpectedAuthorities) > 0 {
		required = append(required, "@authority")
	}
	if err := checkCovered(params, required, opts.RequireNonce); err != nil {
		return err
	}
	if err := checkSignerIdentity(params, opts); err != nil {
		return err
	}
	if err := checkAudience(req, opts); err != nil {
		return err
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

	return v.checkReplay(params, opts)
}

// VerifyResponse verifies an HTTP response signature. req is the request the
// response answers (resp.Request when nil); components carrying the ";req"
// parameter are evaluated against it, so a response signed for another
// request fails verification. With RequireRequestBinding the signature must
// cover `"@method";req`, `"@target-uri";req` and `"@authority";req`, plus
// `"content-digest";req` when the request has a body.
func (v *HTTPVerifier) VerifyResponse(resp *http.Response, req *http.Request, publicKey crypto.PublicKey, opts *HTTPVerificationOptions) error {
	if opts == nil {
		opts = DefaultHTTPVerificationOptions()
	}
	if resp == nil {
		return fmt.Errorf("response is nil")
	}
	if req == nil {
		req = resp.Request
	}

	sigName, params, signature, err := selectSignature(resp.Header, opts)
	if err != nil {
		return err
	}
	if err := checkSignatureTimes(params, opts); err != nil {
		return err
	}

	required := append([]string(nil), opts.RequiredComponents...)
	if opts.RequireContentDigest && responseHasBody(resp) {
		required = append(required, "content-digest")
	}
	if opts.RequireRequestBinding {
		if req == nil {
			return fmt.Errorf("request binding required but no request is associated with the response")
		}
		required = append(required, `"@method";req`, `"@target-uri";req`, `"@authority";req`)
		if requestHasBody(req) {
			required = append(required, `"content-digest";req`)
		}
	}
	if err := checkCovered(params, required, opts.RequireNonce); err != nil {
		return err
	}
	if err := checkSignerIdentity(params, opts); err != nil {
		return err
	}

	bodyValidator := NewBodyIntegrityValidator()
	if err := bodyValidator.ValidateResponseContentDigest(resp, params.CoveredComponents); err != nil {
		return fmt.Errorf("body integrity validation failed: %w", err)
	}

	signatureBase, err := v.canonicalizer.BuildResponseSignatureBase(resp, req, sigName, params)
	if err != nil {
		return fmt.Errorf("failed to build signature base: %w", err)
	}
	if err := v.verifySignature(publicKey, []byte(signatureBase), signature, params.Algorithm); err != nil {
		return err
	}

	return v.checkReplay(params, opts)
}

// selectSignature parses the Signature-Input and Signature fields of h and
// picks the signature to verify. Without an explicit name the
// lexicographically first label is used so the outcome does not depend on
// map iteration order.
func selectSignature(h http.Header, opts *HTTPVerificationOptions) (string, *SignatureInputParams, []byte, error) {
	inputHeader := h.Get("Signature-Input")
	if inputHeader == "" {
		return "", nil, nil, fmt.Errorf("missing Signature-Input header")
	}
	sigInputs, err := ParseSignatureInput(inputHeader)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to parse Signature-Input: %w", err)
	}

	sigHeader := h.Get("Signature")
	if sigHeader == "" {
		return "", nil, nil, fmt.Errorf("missing Signature header")
	}
	signatures, err := ParseSignature(sigHeader)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to parse Signature: %w", err)
	}

	sigName := opts.SignatureName
	if sigName == "" {
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
		return "", nil, nil, fmt.Errorf("signature '%s' not found in Signature-Input", sigName)
	}
	signature, exists := signatures[sigName]
	if !exists {
		return "", nil, nil, fmt.Errorf("signature '%s' not found in Signature header", sigName)
	}
	return sigName, params, signature, nil
}

// checkSignatureTimes enforces created/expires freshness and the clock-skew bound.
func checkSignatureTimes(params *SignatureInputParams, opts *HTTPVerificationOptions) error {
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
	return nil
}

// checkCovered enforces the verifier's required components and nonce policy.
func checkCovered(params *SignatureInputParams, required []string, requireNonce bool) error {
	for _, comp := range required {
		if !IsComponentCovered(params.CoveredComponents, comp) {
			return fmt.Errorf("required component %s is not covered by the signature", normalizeComponentIdentifier(comp))
		}
	}
	if requireNonce && params.Nonce == "" {
		return fmt.Errorf("signature nonce is required but missing")
	}
	return nil
}

// KeyIDDID returns the DID part of a keyid of the form "<DID>#<fragment>" (or
// the keyid itself when it has no fragment). It is the DID a verifier should
// resolve the signing key from.
func KeyIDDID(keyID string) string {
	if i := strings.IndexByte(keyID, '#'); i >= 0 {
		return keyID[:i]
	}
	return keyID
}

// checkSignerIdentity ties the signature's keyid to the expected DID or key.
func checkSignerIdentity(params *SignatureInputParams, opts *HTTPVerificationOptions) error {
	if opts.ExpectedKeyID != "" && params.KeyID != opts.ExpectedKeyID {
		return fmt.Errorf("signature keyid %q does not match expected key %q", params.KeyID, opts.ExpectedKeyID)
	}
	if opts.ExpectedDID != "" && KeyIDDID(params.KeyID) != opts.ExpectedDID {
		return fmt.Errorf("signature keyid %q does not belong to DID %q", params.KeyID, opts.ExpectedDID)
	}
	return nil
}

// checkAudience rejects requests whose authority is not one this verifier
// serves. Combined with "@authority" coverage (enforced by the caller) this
// prevents a request signed for another agent from being accepted here.
func checkAudience(req *http.Request, opts *HTTPVerificationOptions) error {
	if len(opts.ExpectedAuthorities) == 0 {
		return nil
	}
	authority := req.Host
	if authority == "" && req.URL != nil {
		authority = req.URL.Host
	}
	for _, a := range opts.ExpectedAuthorities {
		if strings.EqualFold(strings.TrimSpace(a), authority) {
			return nil
		}
	}
	return fmt.Errorf("request authority %q is not served by this verifier", authority)
}

// checkReplay runs last so that unverifiable messages cannot poison the nonce
// store. The nonce is scoped by keyid so distinct signers do not collide.
func (v *HTTPVerifier) checkReplay(params *SignatureInputParams, opts *HTTPVerificationOptions) error {
	if params.Nonce != "" && v.replay != nil && !opts.DisableReplayCheck {
		if !v.replay.CheckAndMark(params.KeyID, params.Nonce) {
			return fmt.Errorf("signature replay detected: nonce %q was already used for keyid %q", params.Nonce, params.KeyID)
		}
	}
	return nil
}

// responseHasBody reports whether the response carries a body that a
// signature should bind through content-digest.
func responseHasBody(resp *http.Response) bool {
	if resp.ContentLength > 0 {
		return true
	}
	return resp.Body != nil && resp.Body != http.NoBody && resp.ContentLength != 0
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
		if keys.IsSecp256k1Curve(key.Curve) {
			if err := keys.VerifySecp256k1Keccak(key, message, signature); err != nil {
				return fmt.Errorf("secp256k1 signature verification failed: %w", err)
			}
			return nil
		}
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

	// ExpectedDID, when set, requires the signature's keyid to identify a key
	// of this DID: keyid must equal the DID or be "<DID>#<fragment>". Use it
	// to tie the signer to the DID whose key was resolved.
	ExpectedDID string

	// ExpectedKeyID, when set, requires the signature's keyid to equal it.
	ExpectedKeyID string

	// ExpectedAuthorities (requests only) lists the authorities (host[:port])
	// this verifier serves. When set, "@authority" must be covered and the
	// request's authority must be one of them, so a request signed for another
	// agent cannot be forwarded here.
	ExpectedAuthorities []string

	// RequireRequestBinding (responses only) requires the signature to cover
	// `"@method";req`, `"@target-uri";req` and `"@authority";req`, plus
	// `"content-digest";req` when the request has a body, so the response is
	// bound to the request it answers.
	RequireRequestBinding bool
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

// StrictHTTPResponseVerificationOptions returns options for verifying a
// response (for example an MCP tool result): the status and, for responses
// with a body, the body must be covered, and the signature must be bound to
// the request that was sent. Responses need no nonce because the binding to
// the request (which carries one) already prevents replay.
func StrictHTTPResponseVerificationOptions() *HTTPVerificationOptions {
	return &HTTPVerificationOptions{
		MaxAge:                5 * time.Minute,
		MaxClockSkew:          DefaultMaxClockSkew,
		RequiredComponents:    []string{"@status"},
		RequireContentDigest:  true,
		RequireRequestBinding: true,
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
