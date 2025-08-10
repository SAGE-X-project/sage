package session

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"io"

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

    // Cryptographic materials
    // sessionSeed is the HKDF-Extract(PRK) derived from the ECDH shared secret and handshake salt.
    // It is NOT the raw ECDH output. Both peers must compute the same PRK.
    sessionSeed []byte
    encryptKey   []byte
    signingKey   []byte
    aead        cipher.AEAD
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
    Label   string
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
        sessionSeed: sessionSeed,
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
    // Generate random 12-byte nonce
    nonce := make([]byte, chacha20poly1305.NonceSize)
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, fmt.Errorf("failed to generate nonce: %w", err)
    }

    // Seal appends the ciphertext and authentication tag
    ciphertext := s.aead.Seal(nil, nonce, plaintext, nil)

    // Prepend nonce
    out := make([]byte, len(nonce)+len(ciphertext))
    copy(out, nonce)
    copy(out[len(nonce):], ciphertext)

	s.UpdateLastUsed()
    return out, nil
}

// Decrypt decrypts data produced by Encrypt.
// Expects input format: nonce || ciphertext.
func (s *SecureSession) Decrypt(data []byte) ([]byte, error) {
    if len(data) < chacha20poly1305.NonceSize {
        return nil, fmt.Errorf("data too short")
    }

    nonce := data[:chacha20poly1305.NonceSize]
    ciphertext := data[chacha20poly1305.NonceSize:]

    // Open verifies authenticity and decrypts
    plaintext, err := s.aead.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("decryption failed: %w", err)
    }
	s.UpdateLastUsed()
    return plaintext, nil
}

// EncryptAndSign encrypts plaintext and adds authentication
func (s *SecureSession) EncryptAndSign(plaintext []byte) ([]byte, error) {
    if s.IsExpired() {
        return nil, fmt.Errorf("session expired")
    }
    
    // Generate random nonce
    nonce := make([]byte, 12) // ChaCha20-Poly1305 nonce size
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, fmt.Errorf("failed to generate nonce: %w", err)
    }
    
    // Encrypt with AEAD
    ciphertext := s.aead.Seal(nil, nonce, plaintext, nil)
    
    // Create final payload: nonce || ciphertext
    result := make([]byte, len(nonce)+len(ciphertext))
    copy(result, nonce)
    copy(result[len(nonce):], ciphertext)
    
    s.UpdateLastUsed()
    return result, nil
}

// DecryptAndVerify decrypts and verifies ciphertext
func (s *SecureSession) DecryptAndVerify(ciphertext []byte) ([]byte, error) {
    if s.IsExpired() {
        return nil, fmt.Errorf("session expired")
    }
    
    if len(ciphertext) < 12 { // nonce size
        return nil, fmt.Errorf("ciphertext too short")
    }
    
    // Extract nonce and encrypted data
    nonce := ciphertext[:12]
    encrypted := ciphertext[12:]
    
    // Decrypt and verify
    plaintext, err := s.aead.Open(nil, nonce, encrypted, nil)
    if err != nil {
        return nil, fmt.Errorf("decryption/verification failed: %w", err)
    }
    
    s.UpdateLastUsed()
    return plaintext, nil
}
