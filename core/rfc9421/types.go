package rfc9421

import (
	"time"
)

// Message represents a message with RFC-9421 metadata for signature verification
type Message struct {
	// Message metadata
	AgentDID     string                 `json:"agent_did"`
	MessageID    string                 `json:"message_id"`
	Timestamp    time.Time              `json:"timestamp"`
	Nonce        string                 `json:"nonce"`
	
	// Message content
	Headers      map[string]string      `json:"headers"`
	Body         []byte                 `json:"body"`
	
	// Signature metadata
	Algorithm    string                 `json:"algorithm"`
	KeyID        string                 `json:"key_id"`
	Signature    []byte                 `json:"signature"`
	SignedFields []string               `json:"signed_fields"` // Which fields were included in signature
	
	// Additional metadata
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
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
	AlgorithmEdDSA         SignatureAlgorithm = "EdDSA"
	AlgorithmES256K        SignatureAlgorithm = "ES256K"
	AlgorithmECDSA         SignatureAlgorithm = "ECDSA"
	AlgorithmECDSASecp256k1 SignatureAlgorithm = "ECDSA-secp256k1"
)