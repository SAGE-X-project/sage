package handshake

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SessionBuilder constructs Session instances with a fluent API.
type SessionBuilder struct {
	session Session
}

// NewSessionBuilder initializes a builder with default values.
func NewSessionBuilder() *SessionBuilder {
	now := time.Now().UTC()
	return &SessionBuilder{
		session: Session{
			ID:        sessionIDPrefix + uuid.NewString(),
			CreatedAt: now.Format(time.RFC3339),
			Status:    "proposed",
		},
	}
}

// WithKeyInfo assigns a custom SignaturePolicy.
func (b *SessionBuilder) WithKeyInfo(sp KeyInfo) *SessionBuilder {
	b.session.KeyInfo = &sp
	return b
}

// WithStatus overrides the session status (e.g. "proposed", "active", "expired").
func (b *SessionBuilder) WithStatus(status string) *SessionBuilder {
	b.session.Status = status
	return b
}

// WithCreatedAt sets a custom creation timestamp.
func (b *SessionBuilder) WithCreatedAt(t time.Time) *SessionBuilder {
	b.session.CreatedAt = t.Format(time.RFC3339)
	return b
}

// WithExpiresAfter sets ExpiresAt to CreatedAt + duration.
func (b *SessionBuilder) WithExpiresAfter(d time.Duration) *SessionBuilder {
	created, err := time.Parse(time.RFC3339, b.session.CreatedAt)
	if err != nil {
		created = time.Now().UTC()
		b.session.CreatedAt = created.Format(time.RFC3339)
	}
	b.session.ExpiresAt = created.Add(d).Format(time.RFC3339)
	return b
}

// Build returns the constructed Session.
func (b *SessionBuilder) Build() *Session {
	return &b.session
}

// GenerateSalt generates a cryptographically secure 32-byte salt
func GenerateSalt() (string, error) {
    const saltSize = 32 // 256 bits
    saltBytes := make([]byte, saltSize)
    
    // crypto/rand.Read uses the system's CSPRNG
    if _, err := rand.Read(saltBytes); err != nil {
        return "", fmt.Errorf("failed to generate salt: %w", err)
    }
    
    // Encode to Base64URL without padding
    return base64.RawURLEncoding.EncodeToString(saltBytes), nil
}