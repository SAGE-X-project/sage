package did

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	assert.Equal(t, "0.1.0", Version)
}

func TestGetDefaultManager(t *testing.T) {
	manager := GetDefaultManager()
	assert.NotNil(t, manager)
	assert.Equal(t, defaultManager, manager)
}

func TestPackageLevelFunctions(t *testing.T) {
	ctx := context.Background()
	
	// Save original default manager
	originalManager := defaultManager
	defer func() {
		defaultManager = originalManager
	}()
	
	// Create a new manager with mocks
	defaultManager = NewManager()
	mockRegistry := new(MockRegistry)
	mockResolver := new(MockResolver)
	
	// Add mocks to the default manager
	defaultManager.registry.registries[ChainEthereum] = mockRegistry
	defaultManager.resolver.resolvers[ChainEthereum] = mockResolver
	
	t.Run("Configure", func(t *testing.T) {
		config := &RegistryConfig{
			Chain:           ChainEthereum,
			ContractAddress: "0x1234567890abcdef",
			RPCEndpoint:     "http://localhost:8545",
		}
		
		// This should succeed now as we only store configuration
		err := Configure(ChainEthereum, config)
		assert.NoError(t, err)
	})
	
	t.Run("RegisterAgent", func(t *testing.T) {
		mockKeyPair := new(MockKeyPair)
		req := &RegistrationRequest{
			DID:      "agent001",
			Name:     "Test Agent",
			Endpoint: "https://api.example.com",
			KeyPair:  mockKeyPair,
		}
		
		expectedResult := &RegistrationResult{
			TransactionHash: "0xabc123",
			BlockNumber:     12345,
		}
		
		mockRegistry.On("Register", ctx, mock.MatchedBy(func(r *RegistrationRequest) bool {
			return true
		})).Return(expectedResult, nil).Once()
		
		result, err := RegisterAgent(ctx, ChainEthereum, req)
		require.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		
		mockRegistry.AssertExpectations(t)
	})
	
	t.Run("ResolveAgent", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent001")
		expectedMetadata := &AgentMetadata{
			DID:      did,
			Name:     "Test Agent",
			IsActive: true,
		}
		
		mockResolver.On("Resolve", ctx, did).Return(expectedMetadata, nil).Once()
		
		metadata, err := ResolveAgent(ctx, did)
		require.NoError(t, err)
		assert.Equal(t, expectedMetadata, metadata)
		
		mockResolver.AssertExpectations(t)
	})
	
	t.Run("ValidateAgent", func(t *testing.T) {
		// Generate test keypair
		publicKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)
		
		did := AgentDID("did:sage:ethereum:agent001")
		
		agent := &AgentMetadata{
			DID:       did,
			Name:      "Test Agent",
			IsActive:  true,
			PublicKey: publicKey,
			Capabilities: map[string]interface{}{
				"messaging": true,
			},
		}
		
		mockResolver.On("Resolve", ctx, did).Return(agent, nil).Once()
		
		result, err := ValidateAgent(ctx, did, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, agent.Name, result.Name)
		
		mockResolver.AssertExpectations(t)
	})
}

func TestCheckCapabilities(t *testing.T) {
	ctx := context.Background()
	
	// Save original default manager
	originalManager := defaultManager
	defer func() {
		defaultManager = originalManager
	}()
	
	// Create a new manager with mocks
	defaultManager = NewManager()
	mockResolver := new(MockResolver)
	defaultManager.resolver.resolvers[ChainEthereum] = mockResolver
	
	t.Run("CheckCapabilities with all capabilities present", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent001")
		
		agent := &AgentMetadata{
			DID:      did,
			IsActive: true,
			Capabilities: map[string]interface{}{
				"messaging": true,
				"compute":   true,
				"storage":   true,
			},
		}
		
		mockResolver.On("Resolve", ctx, did).Return(agent, nil).Once()
		
		hasCapabilities, err := CheckCapabilities(ctx, did, []string{"messaging", "compute"})
		assert.NoError(t, err)
		assert.True(t, hasCapabilities)
		
		mockResolver.AssertExpectations(t)
	})
	
	t.Run("CheckCapabilities with missing capabilities", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent002")
		
		agent := &AgentMetadata{
			DID:      did,
			IsActive: true,
			Capabilities: map[string]interface{}{
				"messaging": true,
			},
		}
		
		mockResolver.On("Resolve", ctx, did).Return(agent, nil).Once()
		
		hasCapabilities, err := CheckCapabilities(ctx, did, []string{"messaging", "compute"})
		assert.NoError(t, err)
		assert.False(t, hasCapabilities)
		
		mockResolver.AssertExpectations(t)
	})
	
	t.Run("CheckCapabilities with inactive agent", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent003")
		
		agent := &AgentMetadata{
			DID:      did,
			IsActive: false,
			Capabilities: map[string]interface{}{
				"messaging": true,
			},
		}
		
		mockResolver.On("Resolve", ctx, did).Return(agent, nil).Once()
		
		hasCapabilities, err := CheckCapabilities(ctx, did, []string{"messaging"})
		assert.Error(t, err)
		assert.Equal(t, ErrInactiveAgent, err)
		assert.False(t, hasCapabilities)
		
		mockResolver.AssertExpectations(t)
	})
	
	t.Run("CheckCapabilities with resolve error", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent004")
		
		mockResolver.On("Resolve", ctx, did).
			Return(nil, ErrDIDNotFound).Once()
		
		hasCapabilities, err := CheckCapabilities(ctx, did, []string{"messaging"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve agent DID")
		assert.False(t, hasCapabilities)
		
		mockResolver.AssertExpectations(t)
	})
}

func TestValidateDID(t *testing.T) {
	tests := []struct {
		name        string
		did         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid Ethereum DID",
			did:         "did:sage:ethereum:agent001",
			expectError: false,
		},
		{
			name:        "Valid Solana DID",
			did:         "did:sage:solana:agent002",
			expectError: false,
		},
		{
			name:        "Valid short chain prefix",
			did:         "did:sage:eth:agent001",
			expectError: false,
		},
		{
			name:        "Too short",
			did:         "did:sage",
			expectError: true,
			errorMsg:    "DID too short",
		},
		{
			name:        "Wrong prefix",
			did:         "invalid:sage:ethereum:agent001",
			expectError: true,
			errorMsg:    "DID must start with 'did:'",
		},
		{
			name:        "Invalid format",
			did:         "did:invalid",
			expectError: true,
			errorMsg:    "invalid DID format",
		},
		{
			name:        "Unknown chain",
			did:         "did:sage:unknown:agent001",
			expectError: true,
			errorMsg:    "unknown chain",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDID(tt.did)
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}