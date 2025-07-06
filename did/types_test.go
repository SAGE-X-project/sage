package did

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDIDError(t *testing.T) {
	tests := []struct {
		name     string
		err      DIDError
		expected string
	}{
		{
			name: "DID not found error",
			err:  ErrDIDNotFound,
			expected: "DID not found in registry",
		},
		{
			name: "DID already exists error",
			err:  ErrDIDAlreadyExists,
			expected: "DID already registered",
		},
		{
			name: "Invalid signature error",
			err:  ErrInvalidSignature,
			expected: "signature verification failed",
		},
		{
			name: "Custom error with details",
			err: DIDError{
				Code:    "CUSTOM_ERROR",
				Message: "custom error message",
				Details: map[string]interface{}{
					"field": "value",
				},
			},
			expected: "custom error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestChainConstants(t *testing.T) {
	// Test chain constants
	assert.Equal(t, Chain("ethereum"), ChainEthereum)
	assert.Equal(t, Chain("solana"), ChainSolana)
}

func TestNetworkConstants(t *testing.T) {
	// Test Ethereum networks
	assert.Equal(t, Network("ethereum-mainnet"), NetworkEthereumMainnet)
	assert.Equal(t, Network("ethereum-sepolia"), NetworkEthereumSepolia)
	assert.Equal(t, Network("ethereum-goerli"), NetworkEthereumGoerli)
	
	// Test Solana networks
	assert.Equal(t, Network("solana-mainnet"), NetworkSolanaMainnet)
	assert.Equal(t, Network("solana-devnet"), NetworkSolanaDevnet)
	assert.Equal(t, Network("solana-testnet"), NetworkSolanaTestnet)
}

func TestAgentMetadata(t *testing.T) {
	now := time.Now()
	metadata := &AgentMetadata{
		DID:         "did:sage:ethereum:agent001",
		Name:        "Test Agent",
		Description: "A test AI agent",
		Endpoint:    "https://api.example.com",
		PublicKey:   []byte("test-public-key"),
		Capabilities: map[string]interface{}{
			"chat": true,
			"code": true,
		},
		Owner:     "0x1234567890abcdef",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, AgentDID("did:sage:ethereum:agent001"), metadata.DID)
	assert.Equal(t, "Test Agent", metadata.Name)
	assert.Equal(t, "A test AI agent", metadata.Description)
	assert.Equal(t, "https://api.example.com", metadata.Endpoint)
	assert.Equal(t, []byte("test-public-key"), metadata.PublicKey)
	assert.True(t, metadata.IsActive)
	assert.Equal(t, "0x1234567890abcdef", metadata.Owner)
	assert.Equal(t, now, metadata.CreatedAt)
	assert.Equal(t, now, metadata.UpdatedAt)
	
	// Test capabilities
	require.NotNil(t, metadata.Capabilities)
	assert.Equal(t, true, metadata.Capabilities["chat"])
	assert.Equal(t, true, metadata.Capabilities["code"])
}

func TestRegistrationRequest(t *testing.T) {
	req := &RegistrationRequest{
		DID:         "did:sage:ethereum:agent001",
		Name:        "Test Agent",
		Description: "A test AI agent",
		Endpoint:    "https://api.example.com",
		Capabilities: map[string]interface{}{
			"version": "1.0",
			"features": []string{"chat", "code"},
		},
		KeyPair: nil, // Would be set with actual keypair
	}

	assert.Equal(t, AgentDID("did:sage:ethereum:agent001"), req.DID)
	assert.Equal(t, "Test Agent", req.Name)
	assert.Equal(t, "A test AI agent", req.Description)
	assert.Equal(t, "https://api.example.com", req.Endpoint)
	
	// Test nested capabilities
	assert.Equal(t, "1.0", req.Capabilities["version"])
	features, ok := req.Capabilities["features"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"chat", "code"}, features)
}

func TestRegistrationResult(t *testing.T) {
	now := time.Now()
	
	// Test Ethereum result
	ethResult := &RegistrationResult{
		TransactionHash: "0x123abc",
		BlockNumber:     12345,
		Timestamp:       now,
		GasUsed:         21000,
	}
	
	assert.Equal(t, "0x123abc", ethResult.TransactionHash)
	assert.Equal(t, uint64(12345), ethResult.BlockNumber)
	assert.Equal(t, now, ethResult.Timestamp)
	assert.Equal(t, uint64(21000), ethResult.GasUsed)
	assert.Equal(t, uint64(0), ethResult.Slot) // Should be zero for Ethereum
	
	// Test Solana result
	solResult := &RegistrationResult{
		TransactionHash: "5xyzdef",
		Slot:            98765,
		Timestamp:       now,
	}
	
	assert.Equal(t, "5xyzdef", solResult.TransactionHash)
	assert.Equal(t, uint64(98765), solResult.Slot)
	assert.Equal(t, now, solResult.Timestamp)
	assert.Equal(t, uint64(0), solResult.GasUsed) // Should be zero for Solana
}

func TestVerificationResult(t *testing.T) {
	now := time.Now()
	agent := &AgentMetadata{
		DID:      "did:sage:ethereum:agent001",
		Name:     "Test Agent",
		IsActive: true,
	}
	
	// Test successful verification
	successResult := &VerificationResult{
		Valid:      true,
		Agent:      agent,
		VerifiedAt: now,
	}
	
	assert.True(t, successResult.Valid)
	assert.Equal(t, agent, successResult.Agent)
	assert.Equal(t, "", successResult.Error)
	assert.Equal(t, now, successResult.VerifiedAt)
	
	// Test failed verification
	failResult := &VerificationResult{
		Valid:      false,
		Error:      "metadata mismatch",
		VerifiedAt: now,
	}
	
	assert.False(t, failResult.Valid)
	assert.Nil(t, failResult.Agent)
	assert.Equal(t, "metadata mismatch", failResult.Error)
	assert.Equal(t, now, failResult.VerifiedAt)
}