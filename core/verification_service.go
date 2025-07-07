package core

import (
	"context"
	"fmt"
	"time"

	"github.com/sage-x-project/sage/core/rfc9421"
	"github.com/sage-x-project/sage/did"
)

// DIDResolver interface for resolving agent information
type DIDResolver interface {
	ResolveAgent(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error)
	ResolvePublicKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error)
}

// VerificationService provides signature verification with DID integration
type VerificationService struct {
	didResolver DIDResolver
	verifier    *rfc9421.Verifier
}

// NewVerificationService creates a new verification service
func NewVerificationService(didResolver DIDResolver) *VerificationService {
	return &VerificationService{
		didResolver: didResolver,
		verifier:    rfc9421.NewVerifier(),
	}
}

// VerifyAgentMessage verifies a message from an AI agent using DID
func (s *VerificationService) VerifyAgentMessage(
	ctx context.Context,
	message *rfc9421.Message,
	opts *rfc9421.VerificationOptions,
) (*VerificationResult, error) {
	// Resolve agent metadata from DID
	agentMetadata, err := s.didResolver.ResolveAgent(ctx, did.AgentDID(message.AgentDID))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve agent DID: %w", err)
	}
	
	// Check if agent is active
	if opts.RequireActiveAgent && !agentMetadata.IsActive {
		return &VerificationResult{
			Valid:   false,
			Error:   "agent is deactivated",
			AgentID: message.AgentDID,
		}, nil
	}
	
	// Prepare metadata for verification
	expectedMetadata := map[string]interface{}{
		"endpoint": agentMetadata.Endpoint,
		"name":     agentMetadata.Name,
	}
	
	// Add agent capabilities to message metadata if not present
	if message.Metadata == nil {
		message.Metadata = make(map[string]interface{})
	}
	message.Metadata["capabilities"] = agentMetadata.Capabilities
	
	// Verify signature with metadata
	verifyResult, err := s.verifier.VerifyWithMetadata(
		agentMetadata.PublicKey,
		message,
		expectedMetadata,
		opts.RequiredCapabilities,
		opts,
	)
	if err != nil {
		return nil, fmt.Errorf("verification failed: %w", err)
	}
	
	return &VerificationResult{
		Valid:        verifyResult.Valid,
		Error:        verifyResult.Error,
		AgentID:      message.AgentDID,
		AgentName:    agentMetadata.Name,
		VerifiedAt:   verifyResult.VerifiedAt,
		AgentOwner:   agentMetadata.Owner,
		Capabilities: agentMetadata.Capabilities,
	}, nil
}

// VerifyMessageFromHeaders verifies a message constructed from headers
func (s *VerificationService) VerifyMessageFromHeaders(
	ctx context.Context,
	headers map[string]string,
	body []byte,
	signature []byte,
) (*VerificationResult, error) {
	// Parse message from headers
	message, err := rfc9421.ParseMessageFromHeaders(headers, body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}
	
	// Set signature
	message.Signature = signature
	
	// Parse metadata from headers
	if message.Metadata == nil {
		message.Metadata = make(map[string]interface{})
	}
	if endpoint, ok := headers["X-Metadata-Endpoint"]; ok {
		message.Metadata["endpoint"] = endpoint
	}
	if name, ok := headers["X-Metadata-Name"]; ok {
		message.Metadata["name"] = name
	}
	
	// Use default options
	opts := rfc9421.DefaultVerificationOptions()
	
	return s.VerifyAgentMessage(ctx, message, opts)
}

// QuickVerify performs a quick signature verification without full metadata checks
func (s *VerificationService) QuickVerify(
	ctx context.Context,
	agentDID string,
	message []byte,
	signature []byte,
) error {
	// Resolve public key only
	publicKey, err := s.didResolver.ResolvePublicKey(ctx, did.AgentDID(agentDID))
	if err != nil {
		return fmt.Errorf("failed to resolve public key: %w", err)
	}
	
	// Determine algorithm based on DID chain
	chain, _, err := did.ParseDID(did.AgentDID(agentDID))
	if err != nil {
		return fmt.Errorf("failed to parse DID: %w", err)
	}
	
	var algorithm string
	switch chain {
	case did.ChainEthereum:
		algorithm = string(rfc9421.AlgorithmECDSASecp256k1)
	case did.ChainSolana:
		algorithm = string(rfc9421.AlgorithmEdDSA)
	default:
		algorithm = string(rfc9421.AlgorithmEdDSA)
	}
	
	// Create a minimal message for verification
	msg := &rfc9421.Message{
		Body:         message,
		Signature:    signature,
		Algorithm:    algorithm,
		SignedFields: []string{"body"},
		Timestamp:    time.Now(), // Set current time to avoid zero value
	}
	
	// Use options that skip timestamp verification
	opts := &rfc9421.VerificationOptions{
		MaxClockSkew: 0, // Disable timestamp verification
	}
	
	return s.verifier.VerifySignature(publicKey, msg, opts)
}

// VerificationResult contains the result of agent message verification
type VerificationResult struct {
	Valid        bool                   `json:"valid"`
	Error        string                 `json:"error,omitempty"`
	AgentID      string                 `json:"agent_id"`
	AgentName    string                 `json:"agent_name,omitempty"`
	AgentOwner   string                 `json:"agent_owner,omitempty"`
	Capabilities map[string]interface{} `json:"capabilities,omitempty"`
	VerifiedAt   time.Time              `json:"verified_at"`
}