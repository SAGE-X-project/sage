package storage

import (
	"context"
	"time"
)

// SessionStore defines the interface for session persistence
type SessionStore interface {
	// Create creates a new session
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by ID
	Get(ctx context.Context, id string) (*Session, error)

	// Update updates an existing session
	Update(ctx context.Context, session *Session) error

	// Delete deletes a session by ID
	Delete(ctx context.Context, id string) error

	// DeleteExpired deletes all expired sessions
	DeleteExpired(ctx context.Context) (int64, error)

	// List lists all sessions for a client DID
	List(ctx context.Context, clientDID string, limit, offset int) ([]*Session, error)

	// UpdateActivity updates the last activity timestamp
	UpdateActivity(ctx context.Context, id string) error

	// Count returns the total number of active sessions
	Count(ctx context.Context) (int64, error)
}

// NonceStore defines the interface for nonce management
type NonceStore interface {
	// CheckAndStore atomically checks if nonce is used and stores it
	CheckAndStore(ctx context.Context, nonce string, sessionID string, expiresAt time.Time) error

	// IsUsed checks if a nonce has been used
	IsUsed(ctx context.Context, nonce string) (bool, error)

	// DeleteExpired deletes all expired nonces
	DeleteExpired(ctx context.Context) (int64, error)

	// Count returns the total number of stored nonces
	Count(ctx context.Context) (int64, error)
}

// DIDStore defines the interface for DID caching
type DIDStore interface {
	// Create creates a new DID entry
	Create(ctx context.Context, did *DID) error

	// Get retrieves a DID by its identifier
	Get(ctx context.Context, did string) (*DID, error)

	// Update updates an existing DID
	Update(ctx context.Context, did *DID) error

	// Delete deletes a DID
	Delete(ctx context.Context, did string) error

	// ListByOwner lists all DIDs owned by an address
	ListByOwner(ctx context.Context, ownerAddress string) ([]*DID, error)

	// Revoke marks a DID as revoked
	Revoke(ctx context.Context, did string) error

	// IsRevoked checks if a DID is revoked
	IsRevoked(ctx context.Context, did string) (bool, error)
}

// Store combines all storage interfaces
type Store interface {
	SessionStore() SessionStore
	NonceStore() NonceStore
	DIDStore() DIDStore

	// Close closes the storage connection
	Close() error

	// Ping checks the storage connection
	Ping(ctx context.Context) error
}
