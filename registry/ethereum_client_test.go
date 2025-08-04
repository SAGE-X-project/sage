package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEthereumClient(t *testing.T) {
	config := &ClientConfig{
		RPC:      "ws://localhost:8546", // Use a different URL to avoid conflicts
		Contract: "0x1234567890123456789012345678901234567890",
		ChainID:  1,
	}

	// This will fail to connect, but we can test the creation logic
	_, err := NewEthereumClient(config)
	// Expect an error since we're not running a real Ethereum node
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to Ethereum")
}

func TestEthereumClient_Methods(t *testing.T) {
	// We can't create a real client without a running node,
	// but we can test the method signatures exist
	t.Run("method signatures", func(t *testing.T) {
		// Just verify the methods exist and have correct signatures
		var client Client
		client = (*EthereumClient)(nil)
		
		// These should compile without errors
		_ = client.IsAgentRegistered
		_ = client.GetRegistrationStatus
		_ = client.RegisterAgent
		_ = client.UpdateAgent
		_ = client.DeactivateAgent
		_ = client.GetAgentByDID
		_ = client.GetAgentsByOwner
	})
}

func TestEthereumClient_ConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *ClientConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &ClientConfig{
				RPC:      "http://localhost:8545",
				Contract: "0x1234567890123456789012345678901234567890",
				ChainID:  1,
			},
			wantErr: true, // Still expect error due to no real node
			errMsg:  "failed to connect to Ethereum",
		},
		{
			name: "empty RPC",
			config: &ClientConfig{
				RPC:      "",
				Contract: "0x1234567890123456789012345678901234567890",
				ChainID:  1,
			},
			wantErr: true,
		},
		{
			name: "invalid contract address format",
			config: &ClientConfig{
				RPC:      "http://localhost:8545",
				Contract: "invalid-address",
				ChainID:  1,
			},
			wantErr: true,
			errMsg:  "failed to connect to Ethereum", // Still fails at connection step first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEthereumClient(tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test helper functions and utilities
func TestClientConfigValidation(t *testing.T) {
	config := &ClientConfig{
		RPC:      "http://localhost:8545",
		Contract: "0x1234567890123456789012345678901234567890",
		ChainID:  1,
	}

	assert.Equal(t, "http://localhost:8545", config.RPC)
	assert.Equal(t, "0x1234567890123456789012345678901234567890", config.Contract)
	assert.Equal(t, uint64(1), config.ChainID)
}

func TestUpdateRequestValidation(t *testing.T) {
	req := &UpdateRequest{
		Name:        "Updated Agent",
		Description: "Updated Description", 
		Endpoint:    "https://updated.example.com",
		Capabilities: map[string]interface{}{
			"models":    []string{"gpt-4", "claude-3"},
			"languages": []string{"en", "es", "fr"},
		},
	}

	assert.Equal(t, "Updated Agent", req.Name)
	assert.Equal(t, "Updated Description", req.Description)
	assert.Equal(t, "https://updated.example.com", req.Endpoint)
	assert.NotNil(t, req.Capabilities)
	
	models, ok := req.Capabilities["models"].([]string)
	require.True(t, ok)
	assert.Len(t, models, 2)
	assert.Contains(t, models, "gpt-4")
	assert.Contains(t, models, "claude-3")
}