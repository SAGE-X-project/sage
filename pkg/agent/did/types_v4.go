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

package did

import (
	"time"
)

// KeyType represents the type of cryptographic key
type KeyType int

const (
	KeyTypeEd25519 KeyType = iota // Solana, Cardano, Polkadot
	KeyTypeECDSA                  // Ethereum, Bitcoin (secp256k1)
	KeyTypeX25519                 // HPKE key exchange
)

// String returns the string representation of KeyType
func (k KeyType) String() string {
	switch k {
	case KeyTypeEd25519:
		return "Ed25519"
	case KeyTypeECDSA:
		return "ECDSA"
	case KeyTypeX25519:
		return "X25519"
	default:
		return "Unknown"
	}
}

// AgentKey represents a single cryptographic key with metadata
type AgentKey struct {
	Type      KeyType   `json:"type"`
	KeyData   []byte    `json:"key_data"`   // Raw public key bytes
	Signature []byte    `json:"signature"`  // Signature proving ownership
	Verified  bool      `json:"verified"`   // Whether key has been verified
	CreatedAt time.Time `json:"created_at"` // When key was added
}

// AgentMetadataV4 contains metadata for agents with multi-key support
// This type supports SageRegistryV4 contract with multiple keys per agent
type AgentMetadataV4 struct {
	DID          AgentDID               `json:"did"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Endpoint     string                 `json:"endpoint"`
	Keys         []AgentKey             `json:"keys"` // Multiple keys (Ed25519, ECDSA, X25519)
	Capabilities map[string]interface{} `json:"capabilities"`
	Owner        string                 `json:"owner"` // Blockchain address
	IsActive     bool                   `json:"is_active"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// A2APublicKey represents a public key in A2A Agent Card format
type A2APublicKey struct {
	ID              string `json:"id"`               // Key identifier
	Type            string `json:"type"`             // Key type (e.g., "Ed25519VerificationKey2020")
	Controller      string `json:"controller"`       // DID that controls this key
	PublicKeyBase58 string `json:"publicKeyBase58"`  // Base58-encoded public key
	PublicKeyHex    string `json:"publicKeyHex,omitempty"` // Hex-encoded (alternative)
}

// A2AEndpoint represents a service endpoint in A2A Agent Card
type A2AEndpoint struct {
	Type string `json:"type"` // e.g., "grpc", "http", "websocket"
	URI  string `json:"uri"`  // Endpoint URL
}

// A2AAgentCard represents a Google A2A protocol Agent Card
// Spec: https://github.com/a2aproject/a2a
type A2AAgentCard struct {
	Context      []string              `json:"@context"`    // JSON-LD context
	ID           string                `json:"id"`          // Agent DID
	Type         []string              `json:"type"`        // e.g., ["Agent", "AIAgent"]
	Name         string                `json:"name"`        // Agent name
	Description  string                `json:"description"` // Agent description
	PublicKeys   []A2APublicKey        `json:"publicKey"`   // Multiple public keys
	Endpoints    []A2AEndpoint         `json:"service"`     // Service endpoints
	Capabilities []string              `json:"capabilities,omitempty"` // Agent capabilities
	Created      time.Time             `json:"created"`     // Creation timestamp
	Updated      time.Time             `json:"updated"`     // Last update timestamp
}

// GetKeyByType returns the first key of the specified type
func (m *AgentMetadataV4) GetKeyByType(keyType KeyType) *AgentKey {
	for i := range m.Keys {
		if m.Keys[i].Type == keyType {
			return &m.Keys[i]
		}
	}
	return nil
}

// GetVerifiedKeys returns all verified keys
func (m *AgentMetadataV4) GetVerifiedKeys() []AgentKey {
	verified := make([]AgentKey, 0, len(m.Keys))
	for _, key := range m.Keys {
		if key.Verified {
			verified = append(verified, key)
		}
	}
	return verified
}

// HasKeyType checks if the agent has a key of the specified type
func (m *AgentMetadataV4) HasKeyType(keyType KeyType) bool {
	return m.GetKeyByType(keyType) != nil
}

// ToAgentMetadata converts V4 metadata to legacy AgentMetadata
// Uses the first ECDSA key as PublicKey and first X25519 key as PublicKEMKey
func (m *AgentMetadataV4) ToAgentMetadata() *AgentMetadata {
	legacy := &AgentMetadata{
		DID:          m.DID,
		Name:         m.Name,
		Description:  m.Description,
		Endpoint:     m.Endpoint,
		Capabilities: m.Capabilities,
		Owner:        m.Owner,
		IsActive:     m.IsActive,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}

	// Use first ECDSA key as primary PublicKey
	if ecdsaKey := m.GetKeyByType(KeyTypeECDSA); ecdsaKey != nil {
		legacy.PublicKey = ecdsaKey.KeyData
	}

	// Use first X25519 key as PublicKEMKey
	if x25519Key := m.GetKeyByType(KeyTypeX25519); x25519Key != nil {
		legacy.PublicKEMKey = x25519Key.KeyData
	}

	return legacy
}

// FromAgentMetadata creates a V4 metadata from legacy AgentMetadata
func FromAgentMetadata(legacy *AgentMetadata) *AgentMetadataV4 {
	v4 := &AgentMetadataV4{
		DID:          legacy.DID,
		Name:         legacy.Name,
		Description:  legacy.Description,
		Endpoint:     legacy.Endpoint,
		Keys:         make([]AgentKey, 0, 2),
		Capabilities: legacy.Capabilities,
		Owner:        legacy.Owner,
		IsActive:     legacy.IsActive,
		CreatedAt:    legacy.CreatedAt,
		UpdatedAt:    legacy.UpdatedAt,
	}

	// Convert PublicKey to ECDSA key
	if legacy.PublicKey != nil {
		if keyBytes, ok := legacy.PublicKey.([]byte); ok {
			v4.Keys = append(v4.Keys, AgentKey{
				Type:      KeyTypeECDSA,
				KeyData:   keyBytes,
				Verified:  true,
				CreatedAt: legacy.CreatedAt,
			})
		}
	}

	// Convert PublicKEMKey to X25519 key
	if legacy.PublicKEMKey != nil {
		if keyBytes, ok := legacy.PublicKEMKey.([]byte); ok {
			v4.Keys = append(v4.Keys, AgentKey{
				Type:      KeyTypeX25519,
				KeyData:   keyBytes,
				Verified:  true,
				CreatedAt: legacy.CreatedAt,
			})
		}
	}

	return v4
}
