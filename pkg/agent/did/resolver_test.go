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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockResolver is a mock implementation of the Resolver interface
type MockResolver struct {
	mock.Mock
}

func (m *MockResolver) Resolve(ctx context.Context, did AgentDID) (*AgentMetadata, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*AgentMetadata), args.Error(1)
}

func (m *MockResolver) ResolvePublicKey(ctx context.Context, did AgentDID) (interface{}, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
}

func (m *MockResolver) ResolveKEMKey(ctx context.Context, did AgentDID) (interface{}, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
}

func (m *MockResolver) VerifyMetadata(ctx context.Context, did AgentDID, metadata *AgentMetadata) (*VerificationResult, error) {
	args := m.Called(ctx, did, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*VerificationResult), args.Error(1)
}

func (m *MockResolver) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*AgentMetadata, error) {
	args := m.Called(ctx, ownerAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*AgentMetadata), args.Error(1)
}

func (m *MockResolver) Search(ctx context.Context, criteria SearchCriteria) ([]*AgentMetadata, error) {
	args := m.Called(ctx, criteria)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*AgentMetadata), args.Error(1)
}

func TestMultiChainResolver(t *testing.T) {
	ctx := context.Background()

	// Create mock resolvers for different chains
	ethResolver := new(MockResolver)
	solResolver := new(MockResolver)

	// Create multi-chain resolver
	multiResolver := NewMultiChainResolver()
	multiResolver.AddResolver(ChainEthereum, ethResolver)
	multiResolver.AddResolver(ChainSolana, solResolver)

	t.Run("Resolve with chain prefix", func(t *testing.T) {
		did := AgentDID("did:sage:eth:agent001")
		expectedMetadata := &AgentMetadata{
			DID:      did,
			Name:     "Test Agent",
			IsActive: true,
		}

		ethResolver.On("Resolve", ctx, did).Return(expectedMetadata, nil).Once()

		metadata, err := multiResolver.Resolve(ctx, did)
		require.NoError(t, err)
		assert.Equal(t, expectedMetadata, metadata)

		ethResolver.AssertExpectations(t)
	})

	t.Run("Resolve without chain prefix tries all chains", func(t *testing.T) {
		did := AgentDID("did:invalid:format")
		expectedMetadata := &AgentMetadata{
			DID:      did,
			Name:     "Solana Agent",
			IsActive: true,
		}

		// Ethereum resolver fails
		ethResolver.On("Resolve", ctx, did).Return(nil, errors.New("not found")).Once()
		// Solana resolver succeeds
		solResolver.On("Resolve", ctx, did).Return(expectedMetadata, nil).Once()

		metadata, err := multiResolver.Resolve(ctx, did)
		require.NoError(t, err)
		assert.Equal(t, expectedMetadata, metadata)

		ethResolver.AssertExpectations(t)
		solResolver.AssertExpectations(t)
	})

	t.Run("ResolvePublicKey with inactive agent", func(t *testing.T) {
		did := AgentDID("did:sage:eth:agent002")
		inactiveMetadata := &AgentMetadata{
			DID:       did,
			Name:      "Inactive Agent",
			IsActive:  false,
			PublicKey: ed25519.PublicKey(make([]byte, 32)),
		}

		ethResolver.On("Resolve", ctx, did).Return(inactiveMetadata, nil).Once()

		_, err := multiResolver.ResolvePublicKey(ctx, did)
		assert.Error(t, err)
		assert.Equal(t, ErrInactiveAgent, err)

		ethResolver.AssertExpectations(t)
	})

	t.Run("ResolvePublicKey with active agent", func(t *testing.T) {
		did := AgentDID("did:sage:sol:agent003")
		publicKey := ed25519.PublicKey(make([]byte, 32))
		activeMetadata := &AgentMetadata{
			DID:       did,
			Name:      "Active Agent",
			IsActive:  true,
			PublicKey: publicKey,
		}

		solResolver.On("Resolve", ctx, did).Return(activeMetadata, nil).Once()

		pk, err := multiResolver.ResolvePublicKey(ctx, did)
		require.NoError(t, err)
		assert.Equal(t, publicKey, pk)

		solResolver.AssertExpectations(t)
	})

	t.Run("ListAgentsByOwner aggregates from all chains", func(t *testing.T) {
		ownerAddress := "0x1234567890abcdef"

		ethAgents := []*AgentMetadata{
			{DID: "did:sage:eth:agent1", Name: "ETH Agent 1"},
			{DID: "did:sage:eth:agent2", Name: "ETH Agent 2"},
		}

		solAgents := []*AgentMetadata{
			{DID: "did:sage:sol:agent1", Name: "SOL Agent 1"},
		}

		ethResolver.On("ListAgentsByOwner", ctx, ownerAddress).Return(ethAgents, nil).Once()
		solResolver.On("ListAgentsByOwner", ctx, ownerAddress).Return(solAgents, nil).Once()

		allAgents, err := multiResolver.ListAgentsByOwner(ctx, ownerAddress)
		require.NoError(t, err)
		assert.Len(t, allAgents, 3)

		ethResolver.AssertExpectations(t)
		solResolver.AssertExpectations(t)
	})

	t.Run("Search with limit", func(t *testing.T) {
		criteria := SearchCriteria{
			Name:       "Test",
			ActiveOnly: true,
			Limit:      2,
		}

		ethAgents := []*AgentMetadata{
			{DID: "did:sage:eth:agent1", Name: "Test Agent 1"},
			{DID: "did:sage:eth:agent2", Name: "Test Agent 2"},
		}

		solAgents := []*AgentMetadata{
			{DID: "did:sage:sol:agent1", Name: "Test Agent 3"},
		}

		ethResolver.On("Search", ctx, criteria).Return(ethAgents, nil).Once()
		solResolver.On("Search", ctx, criteria).Return(solAgents, nil).Once()

		results, err := multiResolver.Search(ctx, criteria)
		require.NoError(t, err)
		assert.Len(t, results, 2) // Limited to 2

		ethResolver.AssertExpectations(t)
		solResolver.AssertExpectations(t)
	})
}

func TestExtractChainFromDID(t *testing.T) {
	tests := []struct {
		name          string
		did           AgentDID
		expectedChain Chain
		expectError   bool
	}{
		{
			name:          "Ethereum DID",
			did:           "did:sage:eth:agent001",
			expectedChain: ChainEthereum,
			expectError:   false,
		},
		{
			name:          "Solana DID",
			did:           "did:sage:sol:agent002",
			expectedChain: ChainSolana,
			expectError:   false,
		},
		{
			name:          "Invalid format - too short",
			did:           "did:sage",
			expectedChain: "",
			expectError:   true,
		},
		{
			name:          "Invalid format - wrong prefix",
			did:           "invalid:sage:eth:agent",
			expectedChain: "",
			expectError:   true,
		},
		{
			name:          "Unknown chain",
			did:           "did:sage:unknown:agent",
			expectedChain: "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, err := extractChainFromDID(tt.did)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, chain)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedChain, chain)
			}
		})
	}
}

func TestSearchCriteria(t *testing.T) {
	criteria := SearchCriteria{
		Name: "Test Agent",
		Capabilities: map[string]interface{}{
			"chat": true,
			"code": true,
		},
		ActiveOnly: true,
		Limit:      10,
		Offset:     5,
	}

	assert.Equal(t, "Test Agent", criteria.Name)
	assert.True(t, criteria.ActiveOnly)
	assert.Equal(t, 10, criteria.Limit)
	assert.Equal(t, 5, criteria.Offset)
	assert.Equal(t, true, criteria.Capabilities["chat"])
	assert.Equal(t, true, criteria.Capabilities["code"])
}

// TestMultiChainResolver_ResolveKEMKey tests KEM key resolution
func TestMultiChainResolver_ResolveKEMKey(t *testing.T) {
	ctx := context.Background()

	// Create mock resolver
	ethResolver := new(MockResolver)

	// Create multi-chain resolver
	multiResolver := NewMultiChainResolver()
	multiResolver.AddResolver(ChainEthereum, ethResolver)

	t.Run("ResolveKEMKey with active agent", func(t *testing.T) {
		did := AgentDID("did:sage:eth:agent001")
		kmeKey := make([]byte, 32) // X25519 public key
		for i := range kmeKey {
			kmeKey[i] = byte(i)
		}

		activeMetadata := &AgentMetadata{
			DID:          did,
			Name:         "KME Test Agent",
			IsActive:     true,
			PublicKEMKey: kmeKey,
		}

		ethResolver.On("Resolve", ctx, did).Return(activeMetadata, nil).Once()

		kemKey, err := multiResolver.ResolveKEMKey(ctx, did)
		require.NoError(t, err)
		assert.NotNil(t, kemKey)
		assert.Equal(t, kmeKey, kemKey)

		ethResolver.AssertExpectations(t)
	})

	t.Run("ResolveKEMKey with inactive agent", func(t *testing.T) {
		did := AgentDID("did:sage:eth:agent002")
		kmeKey := make([]byte, 32)

		inactiveMetadata := &AgentMetadata{
			DID:          did,
			Name:         "Inactive KME Agent",
			IsActive:     false,
			PublicKEMKey: kmeKey,
		}

		ethResolver.On("Resolve", ctx, did).Return(inactiveMetadata, nil).Once()

		_, err := multiResolver.ResolveKEMKey(ctx, did)
		assert.Error(t, err)
		assert.Equal(t, ErrInactiveAgent, err)

		ethResolver.AssertExpectations(t)
	})

	t.Run("ResolveKEMKey with no KEM key", func(t *testing.T) {
		did := AgentDID("did:sage:eth:agent003")

		activeMetadata := &AgentMetadata{
			DID:          did,
			Name:         "No KME Agent",
			IsActive:     true,
			PublicKEMKey: nil, // No KEM key
		}

		ethResolver.On("Resolve", ctx, did).Return(activeMetadata, nil).Once()

		kemKey, err := multiResolver.ResolveKEMKey(ctx, did)
		require.NoError(t, err)
		assert.Nil(t, kemKey)

		ethResolver.AssertExpectations(t)
	})

	t.Run("ResolveKEMKey with empty KEM key", func(t *testing.T) {
		did := AgentDID("did:sage:eth:agent004")

		activeMetadata := &AgentMetadata{
			DID:          did,
			Name:         "Empty KME Agent",
			IsActive:     true,
			PublicKEMKey: []byte{}, // Empty byte slice
		}

		ethResolver.On("Resolve", ctx, did).Return(activeMetadata, nil).Once()

		kemKey, err := multiResolver.ResolveKEMKey(ctx, did)
		require.NoError(t, err)
		assert.NotNil(t, kemKey)
		assert.Equal(t, 0, len(kemKey.([]byte)))

		ethResolver.AssertExpectations(t)
	})

	t.Run("ResolveKEMKey with resolution error", func(t *testing.T) {
		did := AgentDID("did:sage:eth:nonexistent")

		ethResolver.On("Resolve", ctx, did).Return(nil, ErrDIDNotFound).Once()

		_, err := multiResolver.ResolveKEMKey(ctx, did)
		assert.Error(t, err)
		assert.Equal(t, ErrDIDNotFound, err)

		ethResolver.AssertExpectations(t)
	})
}
