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
	"crypto/ecdsa"
	"crypto/ed25519"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// This file contains types for multi-key agent support.
//
// TERMINOLOGY NOTE:
//   - "V4" in this filename means "multi-key support" (multiple key types per agent)
//   - These types are used by AgentCardRegistry contract
//   - KeyType values MUST match Solidity enum in AgentCardStorage.sol

// KeyType represents the type of cryptographic key
// CRITICAL: Values MUST match Solidity enum in AgentCardStorage.sol
type KeyType int

const (
	KeyTypeECDSA   KeyType = 0 // Ethereum, Bitcoin (secp256k1) - MUST be 0
	KeyTypeEd25519 KeyType = 1 // Solana, Cardano, Polkadot - MUST be 1
	KeyTypeX25519  KeyType = 2 // HPKE key exchange - MUST be 2
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

// AgentMetadataV4 is the former multi-key metadata type. AgentMetadata now
// carries the key list itself, so this is the same type.
//
// Deprecated: use AgentMetadata.
type AgentMetadataV4 = AgentMetadata

// A2APublicKey represents a public key in A2A Agent Card format
type A2APublicKey struct {
	ID              string `json:"id"`                     // Key identifier
	Type            string `json:"type"`                   // Key type (e.g., "Ed25519VerificationKey2020")
	Controller      string `json:"controller"`             // DID that controls this key
	PublicKeyBase58 string `json:"publicKeyBase58"`        // Base58-encoded public key
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
	Context      []string       `json:"@context"`               // JSON-LD context
	ID           string         `json:"id"`                     // Agent DID
	Type         []string       `json:"type"`                   // e.g., ["Agent", "AIAgent"]
	Name         string         `json:"name"`                   // Agent name
	Description  string         `json:"description"`            // Agent description
	PublicKeys   []A2APublicKey `json:"publicKey"`              // Multiple public keys
	Endpoints    []A2AEndpoint  `json:"service"`                // Service endpoints
	Capabilities []string       `json:"capabilities,omitempty"` // Agent capabilities
	Created      time.Time      `json:"created"`                // Creation timestamp
	Updated      time.Time      `json:"updated"`                // Last update timestamp
}

// GetKeyByType returns the first key of the specified type
func (m *AgentMetadata) GetKeyByType(keyType KeyType) *AgentKey {
	for i := range m.Keys {
		if m.Keys[i].Type == keyType {
			return &m.Keys[i]
		}
	}
	return nil
}

// GetVerifiedKeys returns all verified keys
func (m *AgentMetadata) GetVerifiedKeys() []AgentKey {
	verified := make([]AgentKey, 0, len(m.Keys))
	for _, key := range m.Keys {
		if key.Verified {
			verified = append(verified, key)
		}
	}
	return verified
}

// HasKeyType checks if the agent has a key of the specified type
func (m *AgentMetadata) HasKeyType(keyType KeyType) bool {
	return m.GetKeyByType(keyType) != nil
}

// KEMKey returns the raw X25519 key bytes: PublicKEMKey when it is set,
// otherwise the first verified X25519 entry of Keys. It returns nil when the
// agent has no KEM key.
func (m *AgentMetadata) KEMKey() []byte {
	if b, ok := m.PublicKEMKey.([]byte); ok && len(b) > 0 {
		return b
	}
	for _, k := range m.Keys {
		if k.Type == KeyTypeX25519 && k.Verified && len(k.KeyData) > 0 {
			return k.KeyData
		}
	}
	return nil
}

// Normalized returns a copy in which both views agree. When Keys is empty
// it is derived from PublicKey and PublicKEMKey (the derived entries are
// marked verified, since a resolver only exposes keys the registry
// accepted). When PublicKey is nil it becomes the raw bytes of the first
// verified ECDSA key, or failing that the first verified Ed25519 key; when
// PublicKEMKey is nil it becomes the first verified X25519 key. Unknown
// PublicKey types are left alone and produce no key entry. The receiver is
// not modified, so cached resolver results are safe to normalise.
func (m *AgentMetadata) Normalized() *AgentMetadata {
	out := *m
	out.Keys = append([]AgentKey(nil), m.Keys...)

	if len(out.Keys) == 0 {
		if m.PublicKey != nil {
			if keyType, keyBytes, ok := legacyPublicKeyBytes(m.PublicKey); ok {
				out.Keys = append(out.Keys, AgentKey{Type: keyType, KeyData: keyBytes, Verified: true, CreatedAt: m.CreatedAt})
			}
		}
		if kem, ok := m.PublicKEMKey.([]byte); ok && kem != nil {
			out.Keys = append(out.Keys, AgentKey{Type: KeyTypeX25519, KeyData: kem, Verified: true, CreatedAt: m.CreatedAt})
		}
	}

	if out.PublicKey == nil {
		if k := out.firstVerified(KeyTypeECDSA); k != nil {
			out.PublicKey = k.KeyData
		} else if k := out.firstVerified(KeyTypeEd25519); k != nil {
			out.PublicKey = k.KeyData
		}
	}
	if out.PublicKEMKey == nil {
		if k := out.firstVerified(KeyTypeX25519); k != nil {
			out.PublicKEMKey = k.KeyData
		}
	}
	return &out
}

func (m *AgentMetadata) firstVerified(keyType KeyType) *AgentKey {
	for i := range m.Keys {
		if m.Keys[i].Type == keyType && m.Keys[i].Verified {
			return &m.Keys[i]
		}
	}
	return nil
}

// ToAgentMetadata returns Normalized().
//
// Deprecated: AgentMetadata is the only metadata type; call Normalized.
func (m *AgentMetadata) ToAgentMetadata() *AgentMetadata {
	return m.Normalized()
}

// FromAgentMetadata returns legacy.Normalized().
//
// Deprecated: AgentMetadata is the only metadata type; call Normalized.
func FromAgentMetadata(legacy *AgentMetadata) *AgentMetadata {
	return legacy.Normalized()
}

// RegistrationParams represents parameters for AgentCardRegistry registration
// This matches the Solidity struct AgentCardStorage.RegistrationParams
type RegistrationParams struct {
	DID          string    `json:"did"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Endpoint     string    `json:"endpoint"`
	Capabilities string    `json:"capabilities"` // JSON-encoded capabilities
	Keys         [][]byte  `json:"keys"`         // Public key bytes
	KeyTypes     []KeyType `json:"key_types"`    // Type of each key
	Signatures   [][]byte  `json:"signatures"`   // Ownership proof signatures
	Salt         [32]byte  `json:"salt"`         // Salt for commit-reveal
}

// CommitmentState represents the state of a registration commitment
// This matches the return value of AgentCardRegistry.registrationCommitments()
type CommitmentState struct {
	CommitHash [32]byte  `json:"commit_hash"` // Hash of commitment
	Timestamp  time.Time `json:"timestamp"`   // When commitment was made
	Revealed   bool      `json:"revealed"`    // Whether commitment has been revealed
}

// CommitmentStatus represents the local tracking state for three-phase registration
type CommitmentStatus struct {
	Phase           RegistrationPhase   `json:"phase"`            // Current phase
	CommitHash      [32]byte            `json:"commit_hash"`      // Commitment hash
	CommitTimestamp time.Time           `json:"commit_timestamp"` // When committed
	Params          *RegistrationParams `json:"params,omitempty"` // Original params (cleared after reveal)
	AgentID         [32]byte            `json:"agent_id"`         // Set after registration
	CanActivateAt   time.Time           `json:"can_activate_at"`  // Earliest activation time
}

// RegistrationPhase represents the current phase of three-phase registration
type RegistrationPhase int

const (
	PhaseNotStarted RegistrationPhase = iota // Not yet committed
	PhaseCommitted                           // Committed, waiting to register
	PhaseRegistered                          // Registered, waiting activation delay
	PhaseActivated                           // Fully active
	PhaseFailed                              // Registration failed
)

// legacyPublicKeyBytes converts the PublicKey field of a legacy AgentMetadata
// into the on-chain byte encoding and the matching KeyType. Raw bytes are
// assumed to be ECDSA (the historical behaviour); parsed keys are encoded
// with MarshalPublicKey. Unknown types are skipped.
func legacyPublicKeyBytes(pk interface{}) (KeyType, []byte, bool) {
	switch k := pk.(type) {
	case []byte:
		return KeyTypeECDSA, k, true
	case ed25519.PublicKey:
		return KeyTypeEd25519, []byte(k), true
	case *ecdsa.PublicKey, *secp256k1.PublicKey:
		b, err := MarshalPublicKey(k)
		if err != nil {
			return 0, nil, false
		}
		return KeyTypeECDSA, b, true
	default:
		return 0, nil, false
	}
}
