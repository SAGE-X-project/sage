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

package rfc9421

import (
	"time"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	_ "github.com/sage-x-project/sage/pkg/agent/crypto/keys" // Import to register algorithms
)

// Message represents a message with RFC-9421 metadata for signature verification
type Message struct {
	// Message metadata
	AgentDID  string    `json:"agent_did"`
	MessageID string    `json:"message_id"`
	Timestamp time.Time `json:"timestamp"`
	Nonce     string    `json:"nonce"`

	// Message content
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body"`

	// Signature metadata
	Algorithm    string   `json:"algorithm"`
	KeyID        string   `json:"key_id"`
	Signature    []byte   `json:"signature"`
	SignedFields []string `json:"signed_fields"` // Which fields were included in signature

	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// VerificationOptions contains options for signature verification
type VerificationOptions struct {
	// RequireActiveAgent ensures the agent is active
	RequireActiveAgent bool

	// MaxClockSkew is the maximum allowed time difference
	MaxClockSkew time.Duration

	// RequiredCapabilities are capabilities the agent must have
	RequiredCapabilities []string

	// VerifyMetadata ensures message metadata matches expected values
	VerifyMetadata bool
}

// DefaultVerificationOptions returns default verification options
func DefaultVerificationOptions() *VerificationOptions {
	return &VerificationOptions{
		RequireActiveAgent: true,
		MaxClockSkew:       5 * time.Minute,
		VerifyMetadata:     true,
	}
}

// VerificationResult contains the result of signature verification
type VerificationResult struct {
	Valid      bool      `json:"valid"`
	Error      string    `json:"error,omitempty"`
	VerifiedAt time.Time `json:"verified_at"`
}

// SignatureAlgorithm represents supported signature algorithms
type SignatureAlgorithm string

const (
	AlgorithmEdDSA          SignatureAlgorithm = "EdDSA"
	AlgorithmES256K         SignatureAlgorithm = "ES256K"
	AlgorithmECDSA          SignatureAlgorithm = "ECDSA"
	AlgorithmECDSASecp256k1 SignatureAlgorithm = "ECDSA-secp256k1"
)

// GetSupportedAlgorithms returns a list of RFC 9421 supported algorithms
// This dynamically fetches from the centralized algorithm registry
func GetSupportedAlgorithms() []string {
	return sagecrypto.ListRFC9421SupportedAlgorithms()
}

// IsAlgorithmSupported checks if an RFC 9421 algorithm is supported
func IsAlgorithmSupported(algorithm string) bool {
	_, err := sagecrypto.GetKeyTypeFromRFC9421Algorithm(algorithm)
	return err == nil
}
