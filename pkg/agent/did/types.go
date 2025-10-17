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

	"github.com/sage-x-project/sage/pkg/agent/crypto"
)

// AgentDID represents a decentralized identifier for an AI agent
type AgentDID string

// AgentMetadata contains the metadata for a registered AI agent
type AgentMetadata struct {
	DID          AgentDID               `json:"did"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Endpoint     string                 `json:"endpoint"`
	PublicKey    interface{}            `json:"public_key"`     // crypto.PublicKey type
	PublicKEMKey interface{}            `json:"public_kem_key"` // crypto.PublicKey type
	Capabilities map[string]interface{} `json:"capabilities"`
	Owner        string                 `json:"owner"` // Blockchain address of the owner
	IsActive     bool                   `json:"is_active"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// RegistrationRequest contains the data needed to register a new agent
type RegistrationRequest struct {
	DID          AgentDID               `json:"did"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Endpoint     string                 `json:"endpoint"`
	Capabilities map[string]interface{} `json:"capabilities"`
	KeyPair      crypto.KeyPair         `json:"-"` // Used for signing, not serialized
	Keys         []AgentKey             `json:"keys,omitempty"` // Multiple keys for V4 (optional)
}

// RegistrationResult contains the result of a registration operation
type RegistrationResult struct {
	TransactionHash string    `json:"transaction_hash"`
	BlockNumber     uint64    `json:"block_number"`
	Timestamp       time.Time `json:"timestamp"`
	GasUsed         uint64    `json:"gas_used,omitempty"` // For Ethereum
	Slot            uint64    `json:"slot,omitempty"`     // For Solana
}

// VerificationResult contains the result of DID verification
type VerificationResult struct {
	Valid      bool           `json:"valid"`
	Agent      *AgentMetadata `json:"agent,omitempty"`
	Error      string         `json:"error,omitempty"`
	VerifiedAt time.Time      `json:"verified_at"`
}

// Chain represents the blockchain where the DID is registered
type Chain string

const (
	ChainEthereum Chain = "ethereum"
	ChainSolana   Chain = "solana"
)

// Network represents the specific network on a blockchain
type Network string

const (
	// Ethereum networks
	NetworkEthereumMainnet Network = "ethereum-mainnet"
	NetworkEthereumSepolia Network = "ethereum-sepolia"
	NetworkEthereumGoerli  Network = "ethereum-goerli"

	// Solana networks
	NetworkSolanaMainnet Network = "solana-mainnet"
	NetworkSolanaDevnet  Network = "solana-devnet"
	NetworkSolanaTestnet Network = "solana-testnet"
)

// DIDError represents a DID-specific error
type DIDError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e DIDError) Error() string {
	return e.Message
}

// Common DID errors
var (
	ErrDIDNotFound       = DIDError{Code: "DID_NOT_FOUND", Message: "DID not found in registry"}
	ErrDIDAlreadyExists  = DIDError{Code: "DID_EXISTS", Message: "DID already registered"}
	ErrInvalidSignature  = DIDError{Code: "INVALID_SIGNATURE", Message: "signature verification failed"}
	ErrInactiveAgent     = DIDError{Code: "INACTIVE_AGENT", Message: "agent is deactivated"}
	ErrUnauthorized      = DIDError{Code: "UNAUTHORIZED", Message: "unauthorized operation"}
	ErrChainNotSupported = DIDError{Code: "CHAIN_NOT_SUPPORTED", Message: "blockchain not supported"}
)
