package storage

import "time"

// Session represents a stored secure session
type Session struct {
	ID           string                 `json:"id"`
	ClientDID    string                 `json:"client_did"`
	ServerDID    string                 `json:"server_did"`
	SessionKey   []byte                 `json:"session_key"`
	CreatedAt    time.Time              `json:"created_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	LastActivity time.Time              `json:"last_activity"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Nonce represents a used nonce for replay prevention
type Nonce struct {
	Nonce     string    `json:"nonce"`
	SessionID string    `json:"session_id"`
	UsedAt    time.Time `json:"used_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// DID represents a cached DID from blockchain
type DID struct {
	DID          string    `json:"did"`
	PublicKey    []byte    `json:"public_key"`
	OwnerAddress string    `json:"owner_address"`
	KeyType      string    `json:"key_type"`
	Revoked      bool      `json:"revoked"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
