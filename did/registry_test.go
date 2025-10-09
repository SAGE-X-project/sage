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
	"crypto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	
	sagecrypto "github.com/sage-x-project/sage/crypto"
)

// MockRegistry is a mock implementation of the Registry interface
type MockRegistry struct {
	mock.Mock
}

func (m *MockRegistry) Register(ctx context.Context, req *RegistrationRequest) (*RegistrationResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RegistrationResult), args.Error(1)
}

func (m *MockRegistry) Update(ctx context.Context, did AgentDID, updates map[string]interface{}, keyPair sagecrypto.KeyPair) error {
	args := m.Called(ctx, did, updates, keyPair)
	return args.Error(0)
}

func (m *MockRegistry) Deactivate(ctx context.Context, did AgentDID, keyPair sagecrypto.KeyPair) error {
	args := m.Called(ctx, did, keyPair)
	return args.Error(0)
}

func (m *MockRegistry) GetRegistrationStatus(ctx context.Context, txHash string) (*RegistrationResult, error) {
	args := m.Called(ctx, txHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RegistrationResult), args.Error(1)
}

// MockKeyPair is a mock implementation of KeyPair
type MockKeyPair struct {
	mock.Mock
}

func (m *MockKeyPair) Sign(message []byte) ([]byte, error) {
	args := m.Called(message)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockKeyPair) PublicKey() crypto.PublicKey {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(crypto.PublicKey)
}

func (m *MockKeyPair) PrivateKey() crypto.PrivateKey {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(crypto.PrivateKey)
}

func (m *MockKeyPair) Verify(message, signature []byte) error {
	args := m.Called(message, signature)
	return args.Error(0)
}

func (m *MockKeyPair) Type() sagecrypto.KeyType {
	args := m.Called()
	return args.Get(0).(sagecrypto.KeyType)
}

func (m *MockKeyPair) ID() string {
	args := m.Called()
	return args.String(0)
}


func TestMultiChainRegistry(t *testing.T) {
	ctx := context.Background()
	
	// Create mock registries and configs
	ethRegistry := new(MockRegistry)
	solRegistry := new(MockRegistry)
	
	ethConfig := &RegistryConfig{
		Chain:           ChainEthereum,
		ContractAddress: "0x1234567890abcdef",
		RPCEndpoint:     "http://localhost:8545",
	}
	
	solConfig := &RegistryConfig{
		Chain:           ChainSolana,
		ContractAddress: "11111111111111111111111111111111",
		RPCEndpoint:     "http://localhost:8899",
	}
	
	// Create multi-chain registry
	multiRegistry := NewMultiChainRegistry()
	multiRegistry.AddRegistry(ChainEthereum, ethRegistry, ethConfig)
	multiRegistry.AddRegistry(ChainSolana, solRegistry, solConfig)
	
	t.Run("Register with valid request", func(t *testing.T) {
		mockKeyPair := new(MockKeyPair)
		req := &RegistrationRequest{
			DID:         "agent001",
			Name:        "Test Agent",
			Description: "A test agent",
			Endpoint:    "https://api.example.com",
			Capabilities: map[string]interface{}{
				"chat": true,
			},
			KeyPair: mockKeyPair,
		}
		
		expectedResult := &RegistrationResult{
			TransactionHash: "0xabc123",
			BlockNumber:     12345,
		}
		
		ethRegistry.On("Register", ctx, mock.MatchedBy(func(r *RegistrationRequest) bool {
			// Check that DID was prefixed
			return r.DID == "did:sage:ethereum:agent001"
		})).Return(expectedResult, nil).Once()
		
		result, err := multiRegistry.Register(ctx, ChainEthereum, req)
		require.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		
		ethRegistry.AssertExpectations(t)
	})
	
	t.Run("Register with invalid chain", func(t *testing.T) {
		mockKeyPair := new(MockKeyPair)
		req := &RegistrationRequest{
			DID:     "agent002",
			KeyPair: mockKeyPair,
		}
		
		_, err := multiRegistry.Register(ctx, Chain("unknown"), req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no registry for chain")
	})
	
	t.Run("Register with validation error", func(t *testing.T) {
		// Missing required fields
		req := &RegistrationRequest{
			DID: "",
		}
		
		_, err := multiRegistry.Register(ctx, ChainEthereum, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DID is required")
		
		// Missing name
		req.DID = "agent003"
		_, err = multiRegistry.Register(ctx, ChainEthereum, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
		
		// Missing endpoint
		req.Name = "Test"
		_, err = multiRegistry.Register(ctx, ChainEthereum, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint is required")
		
		// Missing keypair
		req.Endpoint = "https://example.com"
		_, err = multiRegistry.Register(ctx, ChainEthereum, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key pair is required")
	})
	
	t.Run("Update agent", func(t *testing.T) {
		mockKeyPair := new(MockKeyPair)
		did := AgentDID("did:sage:ethereum:agent001")
		updates := map[string]interface{}{
			"name":        "Updated Agent",
			"description": "Updated description",
		}
		
		ethRegistry.On("Update", ctx, did, updates, mockKeyPair).Return(nil).Once()
		
		err := multiRegistry.Update(ctx, did, updates, mockKeyPair)
		assert.NoError(t, err)
		
		ethRegistry.AssertExpectations(t)
	})
	
	t.Run("Deactivate agent", func(t *testing.T) {
		mockKeyPair := new(MockKeyPair)
		did := AgentDID("did:sage:solana:agent002")
		
		solRegistry.On("Deactivate", ctx, did, mockKeyPair).Return(nil).Once()
		
		err := multiRegistry.Deactivate(ctx, did, mockKeyPair)
		assert.NoError(t, err)
		
		solRegistry.AssertExpectations(t)
	})
	
	t.Run("GetRegistrationStatus", func(t *testing.T) {
		txHash := "0xdef456"
		expectedResult := &RegistrationResult{
			TransactionHash: txHash,
			BlockNumber:     67890,
		}
		
		ethRegistry.On("GetRegistrationStatus", ctx, txHash).Return(expectedResult, nil).Once()
		
		result, err := multiRegistry.GetRegistrationStatus(ctx, ChainEthereum, txHash)
		require.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		
		ethRegistry.AssertExpectations(t)
	})
}

func TestHasChainPrefix(t *testing.T) {
	tests := []struct {
		name     string
		did      AgentDID
		chain    Chain
		expected bool
	}{
		{
			name:     "Ethereum DID with prefix",
			did:      "did:sage:ethereum:agent001",
			chain:    ChainEthereum,
			expected: true,
		},
		{
			name:     "Solana DID with prefix",
			did:      "did:sage:solana:agent002",
			chain:    ChainSolana,
			expected: true,
		},
		{
			name:     "DID with wrong chain prefix",
			did:      "did:sage:ethereum:agent001",
			chain:    ChainSolana,
			expected: false,
		},
		{
			name:     "DID without prefix",
			did:      "agent001",
			chain:    ChainEthereum,
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasChainPrefix(tt.did, tt.chain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddChainPrefix(t *testing.T) {
	tests := []struct {
		name     string
		did      AgentDID
		chain    Chain
		expected AgentDID
	}{
		{
			name:     "Add Ethereum prefix to bare ID",
			did:      "agent001",
			chain:    ChainEthereum,
			expected: "did:sage:ethereum:agent001",
		},
		{
			name:     "Add Solana prefix to bare ID",
			did:      "agent002",
			chain:    ChainSolana,
			expected: "did:sage:solana:agent002",
		},
		{
			name:     "Replace existing DID prefix",
			did:      "did:example:agent003",
			chain:    ChainEthereum,
			expected: "did:sage:ethereum:example:agent003",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addChainPrefix(tt.did, tt.chain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateRegistrationRequest(t *testing.T) {
	mockKeyPair := new(MockKeyPair)
	
	tests := []struct {
		name        string
		req         *RegistrationRequest
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid request",
			req: &RegistrationRequest{
				DID:      "agent001",
				Name:     "Test Agent",
				Endpoint: "https://api.example.com",
				KeyPair:  mockKeyPair,
			},
			expectError: false,
		},
		{
			name: "Missing DID",
			req: &RegistrationRequest{
				Name:     "Test Agent",
				Endpoint: "https://api.example.com",
				KeyPair:  mockKeyPair,
			},
			expectError: true,
			errorMsg:    "DID is required",
		},
		{
			name: "Missing name",
			req: &RegistrationRequest{
				DID:      "agent001",
				Endpoint: "https://api.example.com",
				KeyPair:  mockKeyPair,
			},
			expectError: true,
			errorMsg:    "name is required",
		},
		{
			name: "Missing endpoint",
			req: &RegistrationRequest{
				DID:     "agent001",
				Name:    "Test Agent",
				KeyPair: mockKeyPair,
			},
			expectError: true,
			errorMsg:    "endpoint is required",
		},
		{
			name: "Missing keypair",
			req: &RegistrationRequest{
				DID:      "agent001",
				Name:     "Test Agent",
				Endpoint: "https://api.example.com",
			},
			expectError: true,
			errorMsg:    "key pair is required",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegistrationRequest(tt.req)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
