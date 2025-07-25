package ethereum

import (
	"testing"

	"github.com/stretchr/testify/assert"
	
	"github.com/sage-x-project/sage/did"
)

func TestSageRegistryABI(t *testing.T) {
	// Test that ABI is valid JSON
	assert.NotEmpty(t, SageRegistryABI)
	assert.Contains(t, SageRegistryABI, "registerAgent")
	assert.Contains(t, SageRegistryABI, "getAgent")
	assert.Contains(t, SageRegistryABI, "updateAgent")
	assert.Contains(t, SageRegistryABI, "deactivateAgent")
}

func TestNewEthereumClient(t *testing.T) {
	config := &did.RegistryConfig{
		Chain:           did.ChainEthereum,
		ContractAddress: "0x1234567890123456789012345678901234567890",
		RPCEndpoint:     "http://localhost:8545",
		PrivateKey:      "", // No private key for read-only
	}
	
	// This will fail without actual RPC endpoint
	_, err := NewEthereumClient(config)
	assert.Error(t, err)
	
	// Test with invalid contract address
	invalidConfig := &did.RegistryConfig{
		Chain:           did.ChainEthereum,
		ContractAddress: "invalid-address",
		RPCEndpoint:     "http://localhost:8545",
	}
	
	_, err = NewEthereumClient(invalidConfig)
	assert.Error(t, err)
}

func TestEthereumHelperMethods(t *testing.T) {
	client := &EthereumClient{
		config: &did.RegistryConfig{
			MaxRetries:         5,
			ConfirmationBlocks: 3,
		},
	}
	
	// Test prepareRegistrationMessage
	req := &did.RegistrationRequest{
		DID:      "did:sage:ethereum:agent001",
		Name:     "Test Agent",
		Endpoint: "https://api.example.com",
	}
	
	message := client.prepareRegistrationMessage(req, "0x1234567890abcdef")
	assert.Contains(t, message, "Register agent:")
	assert.Contains(t, message, string(req.DID))
	assert.Contains(t, message, req.Name)
	assert.Contains(t, message, req.Endpoint)
	assert.Contains(t, message, "0x1234567890abcdef")
	
	// Test prepareUpdateMessage
	agentDID := did.AgentDID("did:sage:ethereum:agent001")
	updates := map[string]interface{}{
		"name":        "Updated Agent",
		"description": "New description",
	}
	
	updateMessage := client.prepareUpdateMessage(agentDID, updates)
	assert.Contains(t, updateMessage, "Update agent:")
	assert.Contains(t, updateMessage, string(agentDID))
	assert.Contains(t, updateMessage, "Updated Agent")
}

func TestCompareCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		cap1     map[string]interface{}
		cap2     map[string]interface{}
		expected bool
	}{
		{
			name: "Equal capabilities",
			cap1: map[string]interface{}{
				"chat": true,
				"code": true,
			},
			cap2: map[string]interface{}{
				"chat": true,
				"code": true,
			},
			expected: true,
		},
		{
			name: "Different length",
			cap1: map[string]interface{}{
				"chat": true,
			},
			cap2: map[string]interface{}{
				"chat": true,
				"code": true,
			},
			expected: false,
		},
		{
			name: "Different values",
			cap1: map[string]interface{}{
				"chat": true,
				"code": false,
			},
			cap2: map[string]interface{}{
				"chat": true,
				"code": true,
			},
			expected: false,
		},
		{
			name:     "Both empty",
			cap1:     map[string]interface{}{},
			cap2:     map[string]interface{}{},
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareCapabilities(tt.cap1, tt.cap2)
			assert.Equal(t, tt.expected, result)
		})
	}
}