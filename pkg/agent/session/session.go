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

package session

import (
	"bytes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"io"

	"github.com/sage-x-project/sage/internal/metrics"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

// SecureSession implements Session with ChaCha20-Poly1305 AEAD
type SecureSession struct {
	id           string
	createdAt    time.Time
	lastUsedAt   time.Time
	messageCount int
	config       Config
	closed       bool

	// Who initiated the HPKE/bootstrap for this session.
	// The initiator uses C2S keys for outbound and S2C for inbound.
	// The responder uses S2C for outbound and C2S for inbound.
	initiator bool

	// Cryptographic materials
	// sessionSeed is the HKDF-Extract(PRK) derived from the ECDH shared secret and handshake salt.
	// It is NOT the raw ECDH output. Both peers must compute the same PRK.
	sessionSeed []byte
	encryptKey  []byte
	signingKey  []byte
	aead        cipher.AEAD

	// Direction-separated keys
	outKey  []byte // AEAD key for outbound (Enc)
	inKey   []byte // AEAD key for inbound  (Enc)
	outSign []byte // HMAC-SHA256 key for outbound signatures
	inSign  []byte // HMAC-SHA256 key for inbound  signatures
	aeadOut cipher.AEAD
	aeadIn  cipher.AEAD
}

// Params describes the handshake context required to deterministically
type Params struct {
	// ContextID must be identical on both peers (e.g., the protocol's ContextID).
	ContextID string
	// SelfEph is this node's ephemeral public key bytes (as sent on the wire).
	SelfEph []byte
	// PeerEph is the peer's ephemeral public key bytes (as received).
	PeerEph []byte
	// Protocol version
	Label        string
	SharedSecret []byte
}

// NewSecureSession creates a new session with derived encryption/signing keys
func NewSecureSession(sid string, sessionSeed []byte, config Config) (*SecureSession, error) {
	if sid == "" || len(sessionSeed) == 0 {
		return nil, fmt.Errorf("invalid inputs")
	}
	now := time.Now()
	sess := &SecureSession{
		id:           sid,
		createdAt:    now,
		lastUsedAt:   now,
		messageCount: 0,
		config:       config,
		sessionSeed:  sessionSeed,
	}

	// Derive encryption and signing keys using HKDF
	if err := sess.deriveKeys(); err != nil {
		return nil, fmt.Errorf("failed to derive keys: %w", err)
	}

	// Initialize AEAD cipher
	aead, err := chacha20poly1305.New(sess.encryptKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD: %w", err)
	}
	sess.aead = aead

	return sess, nil
}

// NewSecureSessionFromExporterWithRole creates a session from an HPKE exporter secret,
// deriving direction-separated keys. 'initiator' is true for the side that ran HPKE Sender.
func NewSecureSessionFromExporterWithRole(sid string, exporter []byte, initiator bool, cfg Config) (*SecureSession, error) {
	if sid == "" || len(exporter) == 0 {
		return nil, fmt.Errorf("invalid inputs")
	}
	now := time.Now()
	sess := &SecureSession{
		id:           sid,
		createdAt:    now,
		lastUsedAt:   now,
		messageCount: 0,
		config:       cfg,
		sessionSeed:  append([]byte(nil), exporter...),
		initiator:    initiator,
	}
	if err := sess.deriveDirectionalKeys(); err != nil {
		return nil, fmt.Errorf("derive keys: %w", err)
	}
	if err := sess.initAEADs(); err != nil {
		return nil, err
	}
	return sess, nil
}

// NewSecureSessionFromExporter creates a session directly from an HPKE exporter secret.
// exporter must be the same 32-byte secret on both peers (e.g., HPKE Export(..., 32)).
func NewSecureSessionFromExporter(sid string, exporter []byte, cfg Config) (*SecureSession, error) {
	if sid == "" || len(exporter) == 0 {
		return nil, fmt.Errorf("invalid inputs")
	}
	now := time.Now()
	sess := &SecureSession{
		id:           sid,
		createdAt:    now,
		lastUsedAt:   now,
		messageCount: 0,
		config:       cfg,
		// We reuse sessionSeed slot to hold the HPKE exporter secret (PRK-like material).
		// This matches the comment that sessionSeed is a PRK, not raw ECDH.
		sessionSeed: append([]byte(nil), exporter...),
	}
	if err := sess.deriveKeys(); err != nil {
		return nil, fmt.Errorf("failed to derive keys: %w", err)
	}
	aead, err := chacha20poly1305.New(sess.encryptKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD: %w", err)
	}
	sess.aead = aead
	return sess, nil
}

// NewSecureSessionWithParams derives a sessionSeed (PRK) and a deterministic sessionID,
// then constructs the SecureSession so both peers get identical id+keys.
func NewSecureSessionWithParams(sharedSecret []byte, p Params, cfg Config) (*SecureSession, error) {
	seed, err := DeriveSessionSeed(sharedSecret, p)
	if err != nil {
		return nil, err
	}
	sid, err := ComputeSessionIDFromSeed(seed, p.Label)
	if err != nil {
		return nil, err
	}
	return NewSecureSession(sid, seed, cfg)
}

// DeriveSessionSeed returns PRK = HKDF-Extract(sharedSecret, salt(label, ctxID, ephs)).
func DeriveSessionSeed(sharedSecret []byte, p Params) ([]byte, error) {
	if len(sharedSecret) == 0 {
		return nil, fmt.Errorf("empty shared secret")
	}
	if p.ContextID == "" || len(p.SelfEph) == 0 || len(p.PeerEph) == 0 {
		return nil, fmt.Errorf("invalid params")
	}
	label := p.Label
	if label == "" {
		label = "a2a/handshake v1"
	}
	lo, hi := canonicalOrder(p.SelfEph, p.PeerEph)

	h := sha256.New()
	h.Write([]byte(label))
	h.Write([]byte(p.ContextID))
	h.Write(lo)
	h.Write(hi)
	salt := h.Sum(nil)

	seed := hkdfExtractSHA256(sharedSecret, salt) // PRK
	return seed, nil
}

// ComputeSessionIDFromSeed deterministically maps PRK -> compact session ID.
func ComputeSessionIDFromSeed(seed []byte, label string) (string, error) {
	if len(seed) == 0 {
		return "", fmt.Errorf("empty seed")
	}
	h := sha256.New()
	h.Write([]byte(label))
	h.Write(seed)
	full := h.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(full[:16]), nil
}

// deriveKeys derives encryption and signing keys from shared secret using HKDF
func (s *SecureSession) deriveKeys() error {
	salt := []byte(s.id) // Use session ID as salt

	// Derive encryption key
	hkdfEnc := hkdf.New(sha256.New, s.sessionSeed, salt, []byte("encryption"))
	s.encryptKey = make([]byte, 32) // ChaCha20-Poly1305 key size
	if _, err := io.ReadFull(hkdfEnc, s.encryptKey); err != nil {
		return fmt.Errorf("failed to derive encryption key: %w", err)
	}

	// Derive signing key
	hkdfSign := hkdf.New(sha256.New, s.sessionSeed, salt, []byte("signing"))
	s.signingKey = make([]byte, 32) // HMAC-SHA256 key size
	if _, err := io.ReadFull(hkdfSign, s.signingKey); err != nil {
		return fmt.Errorf("failed to derive signing key: %w", err)
	}

	return nil
}

// deriveDirectionalKeys derives c2s/s2c enc+sign keys from sessionSeed using HKDF.
// Salt = session ID (binds keys to session identity).
func (s *SecureSession) deriveDirectionalKeys() error {
	salt := []byte(s.id)
	expand := func(info string, n int) ([]byte, error) {
		r := hkdf.New(sha256.New, s.sessionSeed, salt, []byte(info))
		out := make([]byte, n)
		if _, err := io.ReadFull(r, out); err != nil {
			return nil, err
		}
		return out, nil
	}

	// 32B for ChaCha20-Poly1305 key, 32B for HMAC-SHA256 key.
	c2sEnc, err := expand("c2s|enc|v1", 32)
	if err != nil {
		return err
	}
	c2sSign, err := expand("c2s|sign|v1", 32)
	if err != nil {
		return err
	}
	s2cEnc, err := expand("s2c|enc|v1", 32)
	if err != nil {
		return err
	}
	s2cSign, err := expand("s2c|sign|v1", 32)
	if err != nil {
		return err
	}

	if s.initiator {
		// Client(initiator) sends C2S and receives S2C
		s.outKey, s.outSign = c2sEnc, c2sSign
		s.inKey, s.inSign = s2cEnc, s2cSign
	} else {
		// Server(responder) sends S2C and receives C2S
		s.outKey, s.outSign = s2cEnc, s2cSign
		s.inKey, s.inSign = c2sEnc, c2sSign
	}
	return nil
}

func (s *SecureSession) initAEADs() error {
	var err error
	s.aeadOut, err = chacha20poly1305.New(s.outKey)
	if err != nil {
		return fmt.Errorf("create outbound AEAD: %w", err)
	}
	s.aeadIn, err = chacha20poly1305.New(s.inKey)
	if err != nil {
		return fmt.Errorf("create inbound AEAD: %w", err)
	}
	return nil
}

// hkdfExtractSHA256 returns PRK = HKDF-Extract(sha256, ikm, salt).
func hkdfExtractSHA256(ikm, salt []byte) []byte {
	// In Go's x/crypto/hkdf, Extract is exposed via hkdf.Extract.
	prk := hkdf.Extract(sha256.New, ikm, salt)
	// Make a copy to avoid retaining an internal buffer.
	out := make([]byte, len(prk))
	copy(out, prk)
	return out
}

// canonicalOrder returns the two byte slices in lexicographic order.
// This ensures both peers produce identical salt bytes.
func canonicalOrder(a, b []byte) (lo, hi []byte) {
	if bytes.Compare(a, b) <= 0 {
		return a, b
	}
	return b, a
}

// GetID returns the session identifier
func (s *SecureSession) GetID() string {
	return s.id
}

// GetCreatedAt returns when the session was created
func (s *SecureSession) GetCreatedAt() time.Time {
	return s.createdAt
}

// GetLastUsedAt returns the last activity timestamp
func (s *SecureSession) GetLastUsedAt() time.Time {
	return s.lastUsedAt
}

// IsExpired checks if the session has expired based on configured policies
func (s *SecureSession) IsExpired() bool {
	if s.closed {
		return true
	}

	now := time.Now()

	// Check absolute expiration
	if s.config.MaxAge > 0 && now.After(s.createdAt.Add(s.config.MaxAge)) {
		return true
	}

	// Check idle timeout
	if s.config.IdleTimeout > 0 && now.After(s.lastUsedAt.Add(s.config.IdleTimeout)) {
		return true
	}

	// Check message count limit
	if s.config.MaxMessages > 0 && s.messageCount >= s.config.MaxMessages {
		return true
	}

	return false
}

// UpdateLastUsed updates the last activity timestamp and increments message count
func (s *SecureSession) UpdateLastUsed() {
	s.lastUsedAt = time.Now()
	s.messageCount++
}

// Close marks the session as closed
func (s *SecureSession) Close() error {
	s.closed = true

	// Clear sensitive key material
	if s.encryptKey != nil {
		for i := range s.encryptKey {
			s.encryptKey[i] = 0
		}
	}
	if s.signingKey != nil {
		for i := range s.signingKey {
			s.signingKey[i] = 0
		}
	}
	if s.sessionSeed != nil {
		for i := range s.sessionSeed {
			s.sessionSeed[i] = 0
		}
	}

	// Clear directional keys
	if s.outKey != nil {
		for i := range s.outKey {
			s.outKey[i] = 0
		}
	}
	if s.inKey != nil {
		for i := range s.inKey {
			s.inKey[i] = 0
		}
	}
	if s.outSign != nil {
		for i := range s.outSign {
			s.outSign[i] = 0
		}
	}
	if s.inSign != nil {
		for i := range s.inSign {
			s.inSign[i] = 0
		}
	}

	return nil
}

// GetMessageCount returns the number of messages processed
func (s *SecureSession) GetMessageCount() int {
	return s.messageCount
}

// GetConfig returns the session configuration
func (s *SecureSession) GetConfig() Config {
	return s.config
}

// Encrypt encrypts plaintext using ChaCha20-Poly1305.
// Output format: nonce || ciphertext.
func (s *SecureSession) Encrypt(plaintext []byte) ([]byte, error) {
	if s.IsExpired() {
		metrics.CryptoOperations.WithLabelValues("encrypt", "expired").Inc()
		return nil, fmt.Errorf("session expired")
	}

	if s.aeadOut != nil { // directional path
		return s.EncryptOutbound(plaintext)
	}
	// legacy single-AEAD path
	if s.aead == nil {
		metrics.CryptoOperations.WithLabelValues("encrypt", "not_initialized").Inc()
		return nil, fmt.Errorf("session not initialized: AEAD is nil")
	}
	// Generate random 12-byte nonce
	nonce := make([]byte, chacha20poly1305.NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		metrics.CryptoOperations.WithLabelValues("encrypt", "nonce_error").Inc()
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Seal appends the ciphertext and authentication tag
	ciphertext := s.aead.Seal(nil, nonce, plaintext, nil)

	// Prepend nonce
	out := make([]byte, len(nonce)+len(ciphertext))
	copy(out, nonce)
	copy(out[len(nonce):], ciphertext)

	s.UpdateLastUsed()
	metrics.CryptoOperations.WithLabelValues("encrypt", "success").Inc()
	metrics.SessionMessageSize.WithLabelValues("encrypted").Observe(float64(len(out)))
	return out, nil
}

// Decrypt decrypts data produced by Encrypt.
// Expects input format: nonce || ciphertext.
func (s *SecureSession) Decrypt(data []byte) ([]byte, error) {
	if s.IsExpired() {
		metrics.CryptoOperations.WithLabelValues("decrypt", "expired").Inc()
		return nil, fmt.Errorf("session expired")
	}

	if s.aeadIn != nil { // directional path
		return s.DecryptInbound(data)
	}
	// legacy single-AEAD path
	if s.aead == nil {
		metrics.CryptoOperations.WithLabelValues("decrypt", "not_initialized").Inc()
		return nil, fmt.Errorf("session not initialized: AEAD is nil")
	}
	if len(data) < chacha20poly1305.NonceSize {
		metrics.CryptoOperations.WithLabelValues("decrypt", "invalid_data").Inc()
		return nil, fmt.Errorf("data too short")
	}

	nonce := data[:chacha20poly1305.NonceSize]
	ciphertext := data[chacha20poly1305.NonceSize:]

	// Open verifies authenticity and decrypts
	plaintext, err := s.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		metrics.CryptoOperations.WithLabelValues("decrypt", "failure").Inc()
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	s.UpdateLastUsed()
	metrics.CryptoOperations.WithLabelValues("decrypt", "success").Inc()
	metrics.SessionMessageSize.WithLabelValues("decrypted").Observe(float64(len(plaintext)))
	return plaintext, nil
}

// EncryptAndSign encrypts plaintext and returns (cipher, mac) where:
//   - cipher = nonce || ciphertext (ChaCha20-Poly1305)
//   - mac    = HMAC-SHA256(signingKey, covered)
func (s *SecureSession) EncryptAndSign(plaintext []byte, covered []byte) (cipher []byte, mac []byte, err error) {
	if s.IsExpired() {
		return nil, nil, fmt.Errorf("session expired")
	}

	// Encrypt
	nonce := make([]byte, chacha20poly1305.NonceSize)
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	ct := s.aead.Seal(nil, nonce, plaintext, nil)

	out := make([]byte, len(nonce)+len(ct))
	copy(out, nonce)
	copy(out[len(nonce):], ct)

	// HMAC over your covered bytes
	h := hmac.New(sha256.New, s.signingKey)
	h.Write(covered)
	tag := h.Sum(nil)

	s.UpdateLastUsed()
	return out, tag, nil
}

// DecryptAndVerify verifies mac = HMAC-SHA256(signingKey, covered) and then decrypts cipher.
// cipher = nonce || ciphertext
func (s *SecureSession) DecryptAndVerify(cipher []byte, covered []byte, mac []byte) ([]byte, error) {
	if s.IsExpired() {
		return nil, fmt.Errorf("session expired")
	}

	// Verify HMAC first
	h := hmac.New(sha256.New, s.signingKey)
	h.Write(covered)
	want := h.Sum(nil)
	if !hmac.Equal(want, mac) {
		return nil, fmt.Errorf("signature verify failed")
	}

	// Then decrypt
	if len(cipher) < chacha20poly1305.NonceSize {
		return nil, fmt.Errorf("cipher too short")
	}
	nonce := cipher[:chacha20poly1305.NonceSize]
	ct := cipher[chacha20poly1305.NonceSize:]

	plain, err := s.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption/verification failed: %w", err)
	}

	s.UpdateLastUsed()
	return plain, nil
}

// EncryptWithAAD encrypts plaintext with optional AEAD AAD.
// Output: nonce || ciphertext
func (s *SecureSession) EncryptWithAAD(plaintext, aad []byte) ([]byte, error) {
	if s.aeadOut != nil {
		return s.EncryptWithAADOutbound(plaintext, aad)
	}
	if s.aead == nil {
		return nil, fmt.Errorf("session not initialized: AEAD is nil")
	}

	nonce := make([]byte, chacha20poly1305.NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	ct := s.aead.Seal(nil, nonce, plaintext, aad)
	out := make([]byte, len(nonce)+len(ct))
	copy(out, nonce)
	copy(out[len(nonce):], ct)
	s.UpdateLastUsed()
	return out, nil
}

// DecryptWithAAD decrypts data produced by EncryptWithAAD.
// Input: nonce || ciphertext
func (s *SecureSession) DecryptWithAAD(data, aad []byte) ([]byte, error) {
	if s.aeadIn != nil {
		return s.DecryptWithAADInbound(data, aad)
	}
	if s.aead == nil {
		return nil, fmt.Errorf("session not initialized: AEAD is nil")
	}

	if len(data) < chacha20poly1305.NonceSize {
		return nil, fmt.Errorf("data too short")
	}
	nonce := data[:chacha20poly1305.NonceSize]
	ct := data[chacha20poly1305.NonceSize:]
	pt, err := s.aead.Open(nil, nonce, ct, aad)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	s.UpdateLastUsed()
	return pt, nil
}

func (s *SecureSession) SignCovered(covered []byte) []byte {
	m := hmac.New(sha256.New, s.signingKey)
	m.Write(covered)
	s.UpdateLastUsed()
	return m.Sum(nil)
}

func (s *SecureSession) VerifyCovered(covered, sig []byte) error {
	m := hmac.New(sha256.New, s.signingKey)
	m.Write(covered)
	exp := m.Sum(nil)
	if !hmac.Equal(exp, sig) {
		return fmt.Errorf("bad signature")
	}
	s.UpdateLastUsed()
	return nil
}

// EncryptOutbound encrypts plaintext using the *outbound* AEAD.
// Output: nonce || ciphertext
func (s *SecureSession) EncryptOutbound(plaintext []byte) ([]byte, error) {
	if s.aeadOut == nil {
		return nil, fmt.Errorf("session not initialized: outbound AEAD is nil")
	}
	nonce := make([]byte, chacha20poly1305.NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	ct := s.aeadOut.Seal(nil, nonce, plaintext, nil)

	out := make([]byte, len(nonce)+len(ct))
	copy(out, nonce)
	copy(out[len(nonce):], ct)

	s.UpdateLastUsed()
	return out, nil
}

// DecryptInbound decrypts data using the *inbound* AEAD.
// Input: nonce || ciphertext
func (s *SecureSession) DecryptInbound(data []byte) ([]byte, error) {
	if s.aeadIn == nil {
		return nil, fmt.Errorf("session not initialized: inbound AEAD is nil")
	}
	if len(data) < chacha20poly1305.NonceSize {
		return nil, fmt.Errorf("data too short")
	}
	nonce := data[:chacha20poly1305.NonceSize]
	ct := data[chacha20poly1305.NonceSize:]

	pt, err := s.aeadIn.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	s.UpdateLastUsed()
	return pt, nil
}

// EncryptWithAADOutbound encrypts with AAD using *outbound* AEAD.
func (s *SecureSession) EncryptWithAADOutbound(plaintext, aad []byte) ([]byte, error) {
	if s.aeadOut == nil {
		return nil, fmt.Errorf("session not initialized: outbound AEAD is nil")
	}
	nonce := make([]byte, chacha20poly1305.NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	ct := s.aeadOut.Seal(nil, nonce, plaintext, aad)

	out := make([]byte, len(nonce)+len(ct))
	copy(out, nonce)
	copy(out[len(nonce):], ct)

	s.UpdateLastUsed()
	return out, nil
}

// DecryptWithAADInbound decrypts with AAD using *inbound* AEAD.
func (s *SecureSession) DecryptWithAADInbound(data, aad []byte) ([]byte, error) {
	if s.aeadIn == nil {
		return nil, fmt.Errorf("session not initialized: inbound AEAD is nil")
	}
	if len(data) < chacha20poly1305.NonceSize {
		return nil, fmt.Errorf("data too short")
	}
	nonce := data[:chacha20poly1305.NonceSize]
	ct := data[chacha20poly1305.NonceSize:]

	pt, err := s.aeadIn.Open(nil, nonce, ct, aad)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	s.UpdateLastUsed()
	return pt, nil
}
