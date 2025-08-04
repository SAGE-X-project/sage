package did

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_AdditionalMethods(t *testing.T) {
	manager := NewManager()

	t.Run("IsChainConfigured", func(t *testing.T) {
		// Initially no chains are configured
		assert.False(t, manager.IsChainConfigured(ChainEthereum))
		assert.False(t, manager.IsChainConfigured(ChainSolana))

		// Test with unknown chain
		assert.False(t, manager.IsChainConfigured(Chain("unknown")))
	})

	t.Run("GetSupportedChains", func(t *testing.T) {
		// Initially no chains should be supported
		chains := manager.GetSupportedChains()
		assert.Empty(t, chains)
	})

	t.Run("Configure adds chain support", func(t *testing.T) {
		config := &RegistryConfig{
			Chain:           ChainEthereum,
			Network:         NetworkEthereumMainnet,
			RPCEndpoint:     "https://test.example.com",
			ContractAddress: "0x123",
		}

		// Configure should succeed but won't create a real client
		err := manager.Configure(ChainEthereum, config)
		assert.NoError(t, err) // Should succeed with stub implementation
	})
}

func TestManager_EdgeCases(t *testing.T) {
	ctx := context.Background()
	manager := NewManager()

	t.Run("IsAgentRegistered with empty manager", func(t *testing.T) {
		registered, err := manager.IsAgentRegistered(ctx, "did:sage:ethereum:test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resolver for chain ethereum not configured")
		assert.False(t, registered)
	})

	t.Run("GetRegistrationStatus with empty manager", func(t *testing.T) {
		status, err := manager.GetRegistrationStatus(ctx, "did:sage:ethereum:test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resolver for chain ethereum not configured")
		assert.Nil(t, status)
	})

	t.Run("ResolveAgent with empty manager", func(t *testing.T) {
		metadata, err := manager.ResolveAgent(ctx, "did:sage:ethereum:test")
		assert.Error(t, err)
		assert.Nil(t, metadata)
	})

	t.Run("ListAgentsByOwner with empty manager", func(t *testing.T) {
		agents, err := manager.ListAgentsByOwner(ctx, "0x123")
		assert.NoError(t, err) // MultiChainResolver returns empty list, not error
		assert.Empty(t, agents) // Should return empty list when no resolvers configured
	})

	t.Run("SearchAgents with empty manager", func(t *testing.T) {
		criteria := SearchCriteria{
			Name: "test",
		}
		agents, err := manager.SearchAgents(ctx, criteria)
		assert.NoError(t, err) // MultiChainResolver returns empty list, not error
		assert.Empty(t, agents) // Should return empty list when no resolvers configured
	})
}

func TestHelper_isDIDNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "DIDError with correct code",
			err:  DIDError{Code: "DID_NOT_FOUND", Message: "test"},
			want: true,
		},
		{
			name: "DIDError with wrong code",
			err:  DIDError{Code: "OTHER_ERROR", Message: "test"},
			want: false,
		},
		{
			name: "ErrDIDNotFound constant",
			err:  ErrDIDNotFound,
			want: true,
		},
		{
			name: "different error",
			err:  assert.AnError,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDIDNotFoundError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRegistryConfig(t *testing.T) {
	config := &RegistryConfig{
		Chain:            ChainEthereum,
		Network:          NetworkEthereumMainnet,
		ContractAddress:  "0x1234567890123456789012345678901234567890",
		RPCEndpoint:      "https://eth-mainnet.example.com",
		PrivateKey:       "test-private-key",
		GasPrice:         20000000000, // 20 gwei
		MaxRetries:       3,
		ConfirmationBlocks: 6,
	}

	assert.Equal(t, ChainEthereum, config.Chain)
	assert.Equal(t, NetworkEthereumMainnet, config.Network)
	assert.Equal(t, "0x1234567890123456789012345678901234567890", config.ContractAddress)
	assert.Equal(t, "https://eth-mainnet.example.com", config.RPCEndpoint)
	assert.Equal(t, "test-private-key", config.PrivateKey)
	assert.Equal(t, uint64(20000000000), config.GasPrice)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 6, config.ConfirmationBlocks)
}

func TestRegistryImplementations(t *testing.T) {
	t.Run("MultiChainRegistry creation", func(t *testing.T) {
		registry := NewMultiChainRegistry() 
		assert.NotNil(t, registry)
		assert.NotNil(t, registry.registries)
		assert.NotNil(t, registry.configs)
	})

	t.Run("MultiChainResolver creation", func(t *testing.T) {
		resolver := NewMultiChainResolver()
		assert.NotNil(t, resolver)
		assert.NotNil(t, resolver.resolvers)
	})
}

func TestRegistrationStatus(t *testing.T) {
	status := &RegistrationStatus{
		IsRegistered: true,
		IsActive:     true,
		AgentID:      "did:sage:ethereum:test123",
	}

	assert.True(t, status.IsRegistered)
	assert.True(t, status.IsActive) 
	assert.Equal(t, "did:sage:ethereum:test123", status.AgentID)
}