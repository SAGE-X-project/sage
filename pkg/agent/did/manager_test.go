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
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestManager(t *testing.T) {
	
	t.Run("NewManager", func(t *testing.T) {
		manager := NewManager()
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.registry)
		assert.NotNil(t, manager.resolver)
		assert.NotNil(t, manager.verifier)
		assert.NotNil(t, manager.configs)
	})
	
	t.Run("Configure", func(t *testing.T) {
		manager := NewManager()
		
		// Valid Ethereum configuration
		ethConfig := &RegistryConfig{
			Chain:           ChainEthereum,
			ContractAddress: "0x1234567890abcdef",
			RPCEndpoint:     "http://localhost:8545",
		}
		
		err := manager.Configure(ChainEthereum, ethConfig)
		// Should succeed now as we only store configuration
		assert.NoError(t, err)
		
		// Invalid configuration - missing contract address
		invalidConfig := &RegistryConfig{
			Chain:       ChainEthereum,
			RPCEndpoint: "http://localhost:8545",
		}
		
		err = manager.Configure(ChainEthereum, invalidConfig)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "contract address is required")
		
		// Invalid configuration - missing RPC endpoint
		invalidConfig2 := &RegistryConfig{
			Chain:           ChainEthereum,
			ContractAddress: "0x1234567890abcdef",
		}
		
		err = manager.Configure(ChainEthereum, invalidConfig2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "RPC endpoint is required")
		
		// Unknown chain should still succeed with just config storage
		err = manager.Configure(Chain("unknown"), ethConfig)
		assert.NoError(t, err)
	})
	
	t.Run("GetSupportedChains", func(t *testing.T) {
		manager := NewManager()
		
		// Initially no chains
		chains := manager.GetSupportedChains()
		assert.Empty(t, chains)
		
		// Add configuration (will fail to connect but config is stored)
		ethConfig := &RegistryConfig{
			Chain:           ChainEthereum,
			ContractAddress: "0x1234567890abcdef",
			RPCEndpoint:     "http://localhost:8545",
		}
		
		_ = manager.Configure(ChainEthereum, ethConfig)
		
		// Check if config was stored
		assert.True(t, manager.IsChainConfigured(ChainEthereum))
		assert.False(t, manager.IsChainConfigured(ChainSolana))
	})
}

func TestGenerateDID(t *testing.T) {
	tests := []struct {
		name       string
		chain      Chain
		identifier string
		expected   AgentDID
	}{
		{
			name:       "Ethereum DID",
			chain:      ChainEthereum,
			identifier: "agent001",
			expected:   "did:sage:ethereum:agent001",
		},
		{
			name:       "Solana DID",
			chain:      ChainSolana,
			identifier: "agent002",
			expected:   "did:sage:solana:agent002",
		},
		{
			name:       "DID with complex identifier",
			chain:      ChainEthereum,
			identifier: "org:department:agent003",
			expected:   "did:sage:ethereum:org:department:agent003",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			did := GenerateDID(tt.chain, tt.identifier)
			assert.Equal(t, tt.expected, did)
		})
	}
}

func TestParseDID(t *testing.T) {
	tests := []struct {
		name               string
		did                AgentDID
		expectedChain      Chain
		expectedIdentifier string
		expectError        bool
	}{
		{
			name:               "Valid Ethereum DID",
			did:                "did:sage:ethereum:agent001",
			expectedChain:      ChainEthereum,
			expectedIdentifier: "agent001",
			expectError:        false,
		},
		{
			name:               "Valid Ethereum DID with eth prefix",
			did:                "did:sage:eth:agent001",
			expectedChain:      ChainEthereum,
			expectedIdentifier: "agent001",
			expectError:        false,
		},
		{
			name:               "Valid Solana DID",
			did:                "did:sage:solana:agent002",
			expectedChain:      ChainSolana,
			expectedIdentifier: "agent002",
			expectError:        false,
		},
		{
			name:               "Valid Solana DID with sol prefix",
			did:                "did:sage:sol:agent002",
			expectedChain:      ChainSolana,
			expectedIdentifier: "agent002",
			expectError:        false,
		},
		{
			name:               "DID with complex identifier",
			did:                "did:sage:ethereum:org:department:agent003",
			expectedChain:      ChainEthereum,
			expectedIdentifier: "org:department:agent003",
			expectError:        false,
		},
		{
			name:               "Invalid format - too short",
			did:                "did:sage",
			expectedChain:      "",
			expectedIdentifier: "",
			expectError:        true,
		},
		{
			name:               "Invalid format - wrong prefix",
			did:                "invalid:sage:ethereum:agent001",
			expectedChain:      "",
			expectedIdentifier: "",
			expectError:        true,
		},
		{
			name:               "Unknown chain",
			did:                "did:sage:unknown:agent001",
			expectedChain:      "",
			expectedIdentifier: "",
			expectError:        true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, identifier, err := ParseDID(tt.did)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, chain)
				assert.Empty(t, identifier)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedChain, chain)
				assert.Equal(t, tt.expectedIdentifier, identifier)
			}
		})
	}
}

func TestManagerWithMockRegistry(t *testing.T) {
	ctx := context.Background()
	
	// Create manager and add mock registry/resolver
	manager := NewManager()
	mockRegistry := new(MockRegistry)
	mockResolver := new(MockResolver)
	
	// Directly add mocks to bypass actual client creation
	manager.registry.registries[ChainEthereum] = mockRegistry
	manager.resolver.resolvers[ChainEthereum] = mockResolver
	
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
		
		result, err := manager.RegisterAgent(ctx, ChainEthereum, req)
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
		
		metadata, err := manager.ResolveAgent(ctx, did)
		require.NoError(t, err)
		assert.Equal(t, expectedMetadata, metadata)
		
		mockResolver.AssertExpectations(t)
	})
	
	t.Run("ResolvePublicKey", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent001")
		publicKey := ed25519.PublicKey(make([]byte, 32))
		expectedMetadata := &AgentMetadata{
			DID:       did,
			Name:      "Test Agent",
			IsActive:  true,
			PublicKey: publicKey,
		}
		
		// MultiChainResolver.ResolvePublicKey calls Resolve internally
		mockResolver.On("Resolve", ctx, did).Return(expectedMetadata, nil).Once()
		
		pk, err := manager.ResolvePublicKey(ctx, did)
		require.NoError(t, err)
		assert.Equal(t, publicKey, pk)
		
		mockResolver.AssertExpectations(t)
	})
	
	t.Run("UpdateAgent", func(t *testing.T) {
		mockKeyPair := new(MockKeyPair)
		did := AgentDID("did:sage:ethereum:agent001")
		updates := map[string]interface{}{
			"name": "Updated Agent",
		}
		
		mockRegistry.On("Update", ctx, did, updates, mockKeyPair).Return(nil).Once()
		
		err := manager.UpdateAgent(ctx, did, updates, mockKeyPair)
		assert.NoError(t, err)
		
		mockRegistry.AssertExpectations(t)
	})
	
	t.Run("DeactivateAgent", func(t *testing.T) {
		mockKeyPair := new(MockKeyPair)
		did := AgentDID("did:sage:ethereum:agent001")
		
		mockRegistry.On("Deactivate", ctx, did, mockKeyPair).Return(nil).Once()
		
		err := manager.DeactivateAgent(ctx, did, mockKeyPair)
		assert.NoError(t, err)
		
		mockRegistry.AssertExpectations(t)
	})
	
	t.Run("ListAgentsByOwner", func(t *testing.T) {
		ownerAddress := "0x1234567890abcdef"
		expectedAgents := []*AgentMetadata{
			{DID: "did:sage:ethereum:agent1", Name: "Agent 1"},
			{DID: "did:sage:ethereum:agent2", Name: "Agent 2"},
		}
		
		mockResolver.On("ListAgentsByOwner", ctx, ownerAddress).
			Return(expectedAgents, nil).Once()
		
		agents, err := manager.ListAgentsByOwner(ctx, ownerAddress)
		require.NoError(t, err)
		assert.Equal(t, expectedAgents, agents)
		
		mockResolver.AssertExpectations(t)
	})
	
	t.Run("SearchAgents", func(t *testing.T) {
		criteria := SearchCriteria{
			Name:       "Test",
			ActiveOnly: true,
		}
		
		expectedAgents := []*AgentMetadata{
			{DID: "did:sage:ethereum:agent1", Name: "Test Agent 1"},
		}
		
		mockResolver.On("Search", ctx, criteria).Return(expectedAgents, nil).Once()
		
		agents, err := manager.SearchAgents(ctx, criteria)
		require.NoError(t, err)
		assert.Equal(t, expectedAgents, agents)
		
		mockResolver.AssertExpectations(t)
	})
	
	t.Run("GetRegistrationStatus", func(t *testing.T) {
		txHash := "0xdef456"
		expectedResult := &RegistrationResult{
			TransactionHash: txHash,
			BlockNumber:     67890,
		}
		
		mockRegistry.On("GetRegistrationStatus", ctx, txHash).
			Return(expectedResult, nil).Once()
		
		result, err := manager.GetRegistrationStatus(ctx, ChainEthereum, txHash)
		require.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		
		mockRegistry.AssertExpectations(t)
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
				"compute":   true,
			},
		}
		
		mockResolver.On("Resolve", ctx, did).Return(agent, nil).Once()
		
		result, err := manager.ValidateAgent(ctx, did, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, agent.Name, result.Name)
		assert.True(t, result.IsActive)
		
		mockResolver.AssertExpectations(t)
	})
	
	t.Run("CheckCapabilities", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent002")
		
		agent := &AgentMetadata{
			DID:      did,
			IsActive: true,
			Capabilities: map[string]interface{}{
				"messaging": true,
				"compute":   true,
			},
		}
		
		mockResolver.On("Resolve", ctx, did).Return(agent, nil).Once()
		
		hasCapabilities, err := manager.CheckCapabilities(ctx, did, []string{"messaging"})
		assert.NoError(t, err)
		assert.True(t, hasCapabilities)
		
		mockResolver.AssertExpectations(t)
	})
}
