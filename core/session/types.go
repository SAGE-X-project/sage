package session

import (
	"crypto"
	"encoding/json"
	"time"
)

const GeneralPrefix = "session"

// EphemeralStore specifies the API for managing ephemeral public keys per context.
type EphemeralStore interface {
	// SavePeer stores peer's ephemeral public key as JWK for a given context.
	SavePeer(id string, peerJWK json.RawMessage) error
    DeletePeer(id string) error 
	// EnsureLocal makes sure a local ephemeral keypair exists for the given context,
	// and returns its public JWK. The private part is not kept here.
	EnsureLocal(id string) (json.RawMessage, error)
    DeleteLocal(id string) error
	// Optional getters for different representations:
	GetPeerJWK(id string) (json.RawMessage, bool)
	GetPeerRaw(id string) ([]byte, bool)
	GetPeerPublicKey(id string) (crypto.PublicKey, bool, error)

	GetLocalJWK(id string) (json.RawMessage, bool)
	GetLocalRaw(id string) ([]byte, bool)
	GetLocalPublicKey(id string) (crypto.PublicKey, bool, error)

	// Optional setters (if caller prefers to provide raw bytes instead of JWK):
	SavePeerRaw(id string, peerRaw []byte) error
}
// Session represents an active cryptographic session between two agents
type Session interface {
    // Identification
    GetID() string
    GetCreatedAt() time.Time
    GetLastUsedAt() time.Time
    
    // Lifecycle
    IsExpired() bool
    UpdateLastUsed()
    Close() error
    
    // Cryptographic operations  
    Encrypt(plaintext []byte) ([]byte, error)
    Decrypt(data []byte) ([]byte, error)
    EncryptAndSign(plaintext []byte) ([]byte, error)
    DecryptAndVerify(ciphertext []byte) ([]byte, error)
    
    // Statistics
    GetMessageCount() int
    GetConfig() Config
}

// Config defines session policies and limits
type Config struct {
    MaxAge       time.Duration `json:"maxAge"`       // 절대 만료 시간 (예: 1시간)
    IdleTimeout  time.Duration `json:"idleTimeout"`  // 비활성 만료 시간 (예: 10분) 
    MaxMessages  int           `json:"maxMessages"`  // 최대 메시지 수 제한
}


// SessionStats provides information about session status
type SessionStats struct {
    TotalSessions   int `json:"totalSessions"`
    ActiveSessions  int `json:"activeSessions"`
    ExpiredSessions int `json:"expiredSessions"`
}
