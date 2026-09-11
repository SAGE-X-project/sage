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
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"io"

	"github.com/sage-x-project/sage/internal/metrics"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

// SecureSession implements Session with ChaCha20-Poly1305 AEAD
type SecureSession struct {
	mu           sync.RWMutex
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

	// Pre-allocated key buffer (192 bytes total, sliced for all keys)
	// Layout: [encryptKey:32][signingKey:32][c2sEnc:32][c2sSign:32][s2cEnc:32][s2cSign:32]
	keyMaterial []byte
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

	// Replay / ordering protection and key rotation (see wire format below).
	sendMu  sync.Mutex
	sendSeq uint64 // next outbound sequence number
	recvMu  sync.Mutex
	recv    replayWindow // inbound sliding window
	genMu   sync.Mutex
	genAEAD map[genKey]cipher.AEAD // AEADs for key generations > 0
}

// Wire format of every ciphertext produced by this package:
//
//	seq (8 bytes, big-endian) || nonce (12 bytes) || ChaCha20-Poly1305(plaintext)
//
// seq is a per-session, per-direction counter starting at 0. It is bound to
// the ciphertext as the first 8 bytes of the AEAD associated data, followed by
// the caller's AAD, so it cannot be altered without failing authentication.
// The receiver keeps a sliding window of ReplayWindowSize sequence numbers
// below the highest one seen: a sequence number already accepted is rejected
// (ErrReplayedMessage), one that fell out of the window is rejected
// (ErrStaleMessage), and messages within the window may arrive out of order.
// Only authenticated messages update the window, so forged headers cannot
// poison it.
//
// When Config.RekeyInterval is > 0 the AEAD key of a direction changes every
// RekeyInterval messages: message seq uses key generation seq / RekeyInterval.
// Generation 0 is the handshake-derived key; generation g > 0 is
// HKDF-SHA256(sessionSeed, salt = session id, info = "sage-session-rekey-v1" ||
// direction || g). Both peers derive the same generations, so rotation needs
// no extra messages.
const (
	// SeqSize is the size of the sequence-number header.
	SeqSize = 8
	// HeaderSize is the size of the header preceding the AEAD ciphertext.
	HeaderSize = SeqSize + chacha20poly1305.NonceSize
	// ReplayWindowSize is how many sequence numbers below the highest accepted
	// one are still accepted (if not seen before).
	ReplayWindowSize = 1024
	// DefaultRekeyInterval is the number of messages per key generation used
	// by Manager when Config.RekeyInterval is 0.
	DefaultRekeyInterval = 256
)

var (
	// ErrReplayedMessage is returned when a sequence number was already accepted.
	ErrReplayedMessage = errors.New("session: replayed message")
	// ErrStaleMessage is returned when a sequence number is older than the replay window.
	ErrStaleMessage = errors.New("session: message outside replay window")
	// ErrDataTooShort is returned when a ciphertext is shorter than its header.
	ErrDataTooShort = errors.New("session: data too short")
)

type genKey struct {
	direction string
	gen       uint64
}

// replayWindow is a sliding bitmap over the last ReplayWindowSize sequence numbers.
type replayWindow struct {
	seen    bool
	highest uint64
	bits    [ReplayWindowSize / 64]uint64
}

func (w *replayWindow) slot(seq uint64) (word int, mask uint64) {
	i := seq % ReplayWindowSize
	return int(i / 64), 1 << (i % 64)
}

// check reports whether seq may be accepted; it does not modify the window.
func (w *replayWindow) check(seq uint64) error {
	if !w.seen || seq > w.highest {
		return nil
	}
	if w.highest-seq >= ReplayWindowSize {
		return ErrStaleMessage
	}
	word, mask := w.slot(seq)
	if w.bits[word]&mask != 0 {
		return ErrReplayedMessage
	}
	return nil
}

// mark records seq as accepted. Callers must have called check first.
func (w *replayWindow) mark(seq uint64) {
	if !w.seen {
		w.seen = true
		w.highest = seq
	} else if seq > w.highest {
		if seq-w.highest >= ReplayWindowSize {
			w.bits = [ReplayWindowSize / 64]uint64{}
		} else {
			for i := w.highest + 1; i <= seq; i++ {
				word, mask := w.slot(i)
				w.bits[word] &^= mask
			}
		}
		w.highest = seq
	}
	word, mask := w.slot(seq)
	w.bits[word] |= mask
}

func (w *replayWindow) reset() {
	*w = replayWindow{}
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
	// The single-key methods (Encrypt/Decrypt, EncryptAndSign/DecryptAndVerify,
	// SignCovered/VerifyCovered) are part of the Session interface and must work
	// on exporter-derived sessions as well; derive their keys from the same
	// exporter with the legacy info label so both peers agree.
	if err := sess.deriveKeys(); err != nil {
		return nil, fmt.Errorf("derive legacy keys: %w", err)
	}
	aead, err := chacha20poly1305.New(sess.encryptKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AEAD: %w", err)
	}
	sess.aead = aead
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
// Optimized: Uses single HKDF expansion for all keys
// Reuses pre-allocated keyMaterial buffer if available (from session pool)
func (s *SecureSession) deriveKeys() error {
	salt := []byte(s.id) // Use session ID as salt

	// Reuse pre-allocated keyMaterial if available and large enough
	// Otherwise allocate new buffer (2 keys x 32 bytes = 64 bytes)
	if len(s.keyMaterial) < 64 {
		s.keyMaterial = make([]byte, 192) // Allocate max size for potential directional keys
	}

	// Single HKDF expansion with domain-separated info
	// Info: "sage-session-keys-v1" provides domain separation
	reader := hkdf.New(sha256.New, s.sessionSeed, salt, []byte("sage-session-keys-v1"))
	if _, err := io.ReadFull(reader, s.keyMaterial[:64]); err != nil {
		return fmt.Errorf("failed to derive keys: %w", err)
	}

	// Slice the key material
	s.encryptKey = s.keyMaterial[0:32]  // First 32 bytes for encryption
	s.signingKey = s.keyMaterial[32:64] // Next 32 bytes for signing

	return nil
}

// deriveDirectionalKeys derives c2s/s2c enc+sign keys from sessionSeed using HKDF.
// Salt = session ID (binds keys to session identity).
// Optimized: Uses single HKDF expansion for all directional keys
func (s *SecureSession) deriveDirectionalKeys() error {
	salt := []byte(s.id)

	// Extend keyMaterial buffer to hold directional keys
	// Layout: [encryptKey:32][signingKey:32][c2sEnc:32][c2sSign:32][s2cEnc:32][s2cSign:32]
	if len(s.keyMaterial) < 192 {
		// Preserve existing keys and extend buffer
		existing := make([]byte, len(s.keyMaterial))
		copy(existing, s.keyMaterial)
		s.keyMaterial = make([]byte, 192)
		copy(s.keyMaterial, existing)
		// Update pointers to encryptKey and signingKey
		s.encryptKey = s.keyMaterial[0:32]
		s.signingKey = s.keyMaterial[32:64]
	}

	// Single HKDF expansion for all 4 directional keys (128 bytes total)
	// Info: "sage-directional-keys-v1" provides domain separation from deriveKeys()
	reader := hkdf.New(sha256.New, s.sessionSeed, salt, []byte("sage-directional-keys-v1"))
	if _, err := io.ReadFull(reader, s.keyMaterial[64:192]); err != nil {
		return fmt.Errorf("failed to derive directional keys: %w", err)
	}

	// Slice the directional key material
	// Layout in keyMaterial[64:192]: [c2sEnc:32][c2sSign:32][s2cEnc:32][s2cSign:32]
	c2sEnc := s.keyMaterial[64:96]
	c2sSign := s.keyMaterial[96:128]
	s2cEnc := s.keyMaterial[128:160]
	s2cSign := s.keyMaterial[160:192]

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
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastUsedAt = time.Now()
	s.messageCount++
}

// Reset clears the session for reuse from the pool
// This method zeros all sensitive data and resets state fields
func (s *SecureSession) Reset() {
	s.closed = false
	s.id = ""
	s.createdAt = time.Time{}
	s.lastUsedAt = time.Time{}
	s.messageCount = 0
	s.initiator = false

	// Clear sensitive key material (zero the entire keyMaterial buffer)
	if s.keyMaterial != nil {
		for i := range s.keyMaterial {
			s.keyMaterial[i] = 0
		}
	}
	if s.sessionSeed != nil {
		for i := range s.sessionSeed {
			s.sessionSeed[i] = 0
		}
		s.sessionSeed = nil
	}

	// Clear slice references (they point into keyMaterial)
	s.encryptKey = nil
	s.signingKey = nil
	s.outKey = nil
	s.inKey = nil
	s.outSign = nil
	s.inSign = nil

	// Clear AEAD instances
	s.aead = nil
	s.aeadOut = nil
	s.aeadIn = nil

	s.resetSequencing()
}

// resetSequencing clears the sequence counter, replay window and rotated keys.
func (s *SecureSession) resetSequencing() {
	s.sendMu.Lock()
	s.sendSeq = 0
	s.sendMu.Unlock()
	s.recvMu.Lock()
	s.recv.reset()
	s.recvMu.Unlock()
	s.genMu.Lock()
	s.genAEAD = nil
	s.genMu.Unlock()
}

// InitializeSession initializes a pooled session with the given parameters
// This is used by the session pool to reuse session objects
func (s *SecureSession) InitializeSession(sid string, sessionSeed []byte, config Config) error {
	if sid == "" || len(sessionSeed) == 0 {
		return fmt.Errorf("invalid inputs")
	}

	now := time.Now()
	s.id = sid
	s.createdAt = now
	s.lastUsedAt = now
	s.messageCount = 0
	s.config = config
	s.closed = false
	s.sessionSeed = append([]byte(nil), sessionSeed...)
	s.resetSequencing()

	// Derive encryption and signing keys using HKDF
	if err := s.deriveKeys(); err != nil {
		return fmt.Errorf("failed to derive keys: %w", err)
	}

	// Initialize AEAD cipher
	aead, err := chacha20poly1305.New(s.encryptKey)
	if err != nil {
		return fmt.Errorf("failed to create AEAD: %w", err)
	}
	s.aead = aead

	return nil
}

// Close marks the session as closed
func (s *SecureSession) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	zeroBytes := func(b []byte) {
		for i := range b {
			b[i] = 0
		}
	}

	zeroBytes(s.encryptKey)
	zeroBytes(s.signingKey)
	zeroBytes(s.sessionSeed)
	zeroBytes(s.outKey)
	zeroBytes(s.inKey)
	zeroBytes(s.outSign)
	zeroBytes(s.inSign)

	s.aead = nil
	s.aeadOut = nil
	s.aeadIn = nil
	s.genMu.Lock()
	s.genAEAD = nil
	s.genMu.Unlock()

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

// Encrypt encrypts plaintext for the peer.
// Output format: seq || nonce || ciphertext (see the wire format above).
// Exporter-derived sessions use the outbound directional key; seed-derived
// sessions use the single shared key.
func (s *SecureSession) Encrypt(plaintext []byte) ([]byte, error) {
	if s.IsExpired() {
		metrics.CryptoOperations.WithLabelValues("encrypt", "expired").Inc()
		return nil, fmt.Errorf("session expired")
	}
	out, err := s.EncryptWithAAD(plaintext, nil)
	if err != nil {
		metrics.CryptoOperations.WithLabelValues("encrypt", "failure").Inc()
		return nil, err
	}
	metrics.CryptoOperations.WithLabelValues("encrypt", "success").Inc()
	metrics.SessionMessageSize.WithLabelValues("encrypted").Observe(float64(len(out)))
	return out, nil
}

// Decrypt decrypts data produced by the peer's Encrypt and enforces the
// replay window.
func (s *SecureSession) Decrypt(data []byte) ([]byte, error) {
	if s.IsExpired() {
		metrics.CryptoOperations.WithLabelValues("decrypt", "expired").Inc()
		return nil, fmt.Errorf("session expired")
	}
	pt, err := s.DecryptWithAAD(data, nil)
	if err != nil {
		metrics.CryptoOperations.WithLabelValues("decrypt", "failure").Inc()
		return nil, err
	}
	metrics.CryptoOperations.WithLabelValues("decrypt", "success").Inc()
	metrics.SessionMessageSize.WithLabelValues("decrypted").Observe(float64(len(pt)))
	return pt, nil
}

// EncryptAndSign encrypts plaintext with the single shared key and returns
// (cipher, mac) where:
//   - cipher = seq || nonce || ciphertext (ChaCha20-Poly1305)
//   - mac    = HMAC-SHA256(signingKey, covered)
func (s *SecureSession) EncryptAndSign(plaintext []byte, covered []byte) (cipher []byte, mac []byte, err error) {
	if s.IsExpired() {
		return nil, nil, fmt.Errorf("session expired")
	}
	if s.aead == nil {
		return nil, nil, fmt.Errorf("session not initialized: AEAD is nil")
	}
	out, err := s.seal(s.aead, "single", plaintext, nil)
	if err != nil {
		return nil, nil, err
	}

	h := hmac.New(sha256.New, s.signingKey)
	h.Write(covered)
	tag := h.Sum(nil)

	s.UpdateLastUsed()
	return out, tag, nil
}

// DecryptAndVerify verifies mac = HMAC-SHA256(signingKey, covered) and then
// decrypts cipher (seq || nonce || ciphertext) with the single shared key.
func (s *SecureSession) DecryptAndVerify(cipher []byte, covered []byte, mac []byte) ([]byte, error) {
	if s.IsExpired() {
		return nil, fmt.Errorf("session expired")
	}
	if s.aead == nil {
		return nil, fmt.Errorf("session not initialized: AEAD is nil")
	}

	// Verify HMAC first
	h := hmac.New(sha256.New, s.signingKey)
	h.Write(covered)
	want := h.Sum(nil)
	if !hmac.Equal(want, mac) {
		return nil, fmt.Errorf("signature verify failed")
	}

	plain, err := s.open(s.aead, "single", cipher, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption/verification failed: %w", err)
	}

	s.UpdateLastUsed()
	return plain, nil
}

// EncryptWithAAD encrypts plaintext with additional authenticated data.
// Output: seq || nonce || ciphertext
func (s *SecureSession) EncryptWithAAD(plaintext, aad []byte) ([]byte, error) {
	if s.aeadOut != nil {
		return s.EncryptWithAADOutbound(plaintext, aad)
	}
	if s.aead == nil {
		return nil, fmt.Errorf("session not initialized: AEAD is nil")
	}
	out, err := s.seal(s.aead, "single", plaintext, aad)
	if err != nil {
		return nil, err
	}
	s.UpdateLastUsed()
	return out, nil
}

// DecryptWithAAD decrypts data produced by EncryptWithAAD.
// Input: seq || nonce || ciphertext
func (s *SecureSession) DecryptWithAAD(data, aad []byte) ([]byte, error) {
	if s.aeadIn != nil {
		return s.DecryptWithAADInbound(data, aad)
	}
	if s.aead == nil {
		return nil, fmt.Errorf("session not initialized: AEAD is nil")
	}
	pt, err := s.open(s.aead, "single", data, aad)
	if err != nil {
		return nil, err
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
// Output: seq || nonce || ciphertext
func (s *SecureSession) EncryptOutbound(plaintext []byte) ([]byte, error) {
	return s.EncryptWithAADOutbound(plaintext, nil)
}

// DecryptInbound decrypts data using the *inbound* AEAD.
// Input: seq || nonce || ciphertext
func (s *SecureSession) DecryptInbound(data []byte) ([]byte, error) {
	return s.DecryptWithAADInbound(data, nil)
}

// EncryptWithAADOutbound encrypts with AAD using the *outbound* AEAD.
func (s *SecureSession) EncryptWithAADOutbound(plaintext, aad []byte) ([]byte, error) {
	if s.aeadOut == nil {
		return nil, fmt.Errorf("session not initialized: outbound AEAD is nil")
	}
	out, err := s.seal(s.aeadOut, s.directionLabel(true), plaintext, aad)
	if err != nil {
		return nil, err
	}
	s.UpdateLastUsed()
	return out, nil
}

// DecryptWithAADInbound decrypts with AAD using the *inbound* AEAD.
func (s *SecureSession) DecryptWithAADInbound(data, aad []byte) ([]byte, error) {
	if s.aeadIn == nil {
		return nil, fmt.Errorf("session not initialized: inbound AEAD is nil")
	}
	pt, err := s.open(s.aeadIn, s.directionLabel(false), data, aad)
	if err != nil {
		return nil, err
	}
	s.UpdateLastUsed()
	return pt, nil
}

// SendSequence returns the sequence number the next outbound message will carry.
func (s *SecureSession) SendSequence() uint64 {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	return s.sendSeq
}

// directionLabel names the key a direction uses, identically on both peers:
// the initiator's outbound key is the responder's inbound key.
func (s *SecureSession) directionLabel(outbound bool) string {
	if s.initiator == outbound {
		return "c2s"
	}
	return "s2c"
}

// generation returns the key generation used by sequence number seq.
func (s *SecureSession) generation(seq uint64) uint64 {
	if s.config.RekeyInterval == 0 {
		return 0
	}
	return seq / s.config.RekeyInterval
}

// aeadForSeq returns the AEAD for the given direction and sequence number,
// deriving and caching rotated keys as needed. Generations older than the
// previous one are dropped from the cache.
func (s *SecureSession) aeadForSeq(base cipher.AEAD, direction string, seq uint64) (cipher.AEAD, error) {
	gen := s.generation(seq)
	if gen == 0 {
		return base, nil
	}
	key := genKey{direction: direction, gen: gen}

	s.genMu.Lock()
	defer s.genMu.Unlock()
	if a, ok := s.genAEAD[key]; ok {
		return a, nil
	}
	if len(s.sessionSeed) == 0 {
		return nil, fmt.Errorf("session not initialized: no seed for key rotation")
	}
	info := make([]byte, 0, 32+len(direction)+8)
	info = append(info, "sage-session-rekey-v1"...)
	info = append(info, direction...)
	info = binary.BigEndian.AppendUint64(info, gen)
	k := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(hkdf.New(sha256.New, s.sessionSeed, []byte(s.id), info), k); err != nil {
		return nil, fmt.Errorf("derive rotated key: %w", err)
	}
	a, err := chacha20poly1305.New(k)
	if err != nil {
		return nil, fmt.Errorf("create rotated AEAD: %w", err)
	}
	if s.genAEAD == nil {
		s.genAEAD = make(map[genKey]cipher.AEAD)
	}
	for k := range s.genAEAD {
		if k.direction == direction && k.gen+1 < gen {
			delete(s.genAEAD, k)
		}
	}
	s.genAEAD[key] = a
	return a, nil
}

// seal produces seq || nonce || AEAD(plaintext, aad' = seq || aad).
func (s *SecureSession) seal(base cipher.AEAD, direction string, plaintext, aad []byte) ([]byte, error) {
	s.sendMu.Lock()
	seq := s.sendSeq
	s.sendSeq++
	s.sendMu.Unlock()

	aead, err := s.aeadForSeq(base, direction, seq)
	if err != nil {
		return nil, err
	}

	out := make([]byte, HeaderSize, HeaderSize+len(plaintext)+aead.Overhead())
	binary.BigEndian.PutUint64(out[:SeqSize], seq)
	nonce := out[SeqSize:HeaderSize]
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	// #nosec G407 - nonce is randomly generated using crypto/rand above
	return aead.Seal(out, nonce, plaintext, boundAAD(out[:SeqSize], aad)), nil
}

// open verifies seq against the replay window, authenticates and decrypts
// seq || nonce || ciphertext, and records seq only on success.
func (s *SecureSession) open(base cipher.AEAD, direction string, data, aad []byte) ([]byte, error) {
	if len(data) < HeaderSize {
		return nil, ErrDataTooShort
	}
	seq := binary.BigEndian.Uint64(data[:SeqSize])

	s.recvMu.Lock()
	defer s.recvMu.Unlock()
	if err := s.recv.check(seq); err != nil {
		return nil, err
	}
	aead, err := s.aeadForSeq(base, direction, seq)
	if err != nil {
		return nil, err
	}
	nonce := data[SeqSize:HeaderSize]
	pt, err := aead.Open(nil, nonce, data[HeaderSize:], boundAAD(data[:SeqSize], aad)) // #nosec G407 -- nonce extracted from data, not hardcoded
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}
	s.recv.mark(seq)
	return pt, nil
}

// boundAAD prefixes the caller's AAD with the sequence header.
func boundAAD(seqHeader, aad []byte) []byte {
	out := make([]byte, 0, len(seqHeader)+len(aad))
	out = append(out, seqHeader...)
	return append(out, aad...)
}
