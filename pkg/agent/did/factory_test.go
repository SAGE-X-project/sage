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
	"testing"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient implements the Client interface for testing
type mockClient struct {
	chainType Chain
}

func (m *mockClient) Register(ctx context.Context, req *RegistrationRequest) (*RegistrationResult, error) {
	return &RegistrationResult{}, nil
}

func (m *mockClient) Resolve(ctx context.Context, agentDID AgentDID) (*AgentMetadata, error) {
	return &AgentMetadata{}, nil
}

func (m *mockClient) Update(ctx context.Context, agentDID AgentDID, updates map[string]interface{}, keyPair sagecrypto.KeyPair) error {
	return nil
}

func (m *mockClient) Deactivate(ctx context.Context, agentDID AgentDID, keyPair sagecrypto.KeyPair) error {
	return nil
}

func TestNewClientFactory(t *testing.T) {
	factory := NewClientFactory()
	assert.NotNil(t, factory)
}

func TestGetRecommendedKeyType(t *testing.T) {
	factory := NewClientFactory()

	tests := []struct {
		name      string
		chainType Chain
		expected  sagecrypto.KeyType
		wantErr   bool
	}{
		{
			name:      "Ethereum requires Secp256k1",
			chainType: ChainEthereum,
			expected:  sagecrypto.KeyTypeSecp256k1,
			wantErr:   false,
		},
		{
			name:      "Solana requires Ed25519",
			chainType: ChainSolana,
			expected:  sagecrypto.KeyTypeEd25519,
			wantErr:   false,
		},
		{
			name:      "Unknown chain returns error",
			chainType: Chain("unknown"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyType, err := factory.GetRecommendedKeyType(tt.chainType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, keyType)
			}
		})
	}
}

func TestValidateKeyTypeForChain(t *testing.T) {
	factory := NewClientFactory()

	tests := []struct {
		name      string
		keyType   sagecrypto.KeyType
		chainType Chain
		wantErr   bool
	}{
		{
			name:      "Secp256k1 is valid for Ethereum",
			keyType:   sagecrypto.KeyTypeSecp256k1,
			chainType: ChainEthereum,
			wantErr:   false,
		},
		{
			name:      "Ed25519 is valid for Solana",
			keyType:   sagecrypto.KeyTypeEd25519,
			chainType: ChainSolana,
			wantErr:   false,
		},
		{
			name:      "Ed25519 is invalid for Ethereum",
			keyType:   sagecrypto.KeyTypeEd25519,
			chainType: ChainEthereum,
			wantErr:   true,
		},
		{
			name:      "Secp256k1 is invalid for Solana",
			keyType:   sagecrypto.KeyTypeSecp256k1,
			chainType: ChainSolana,
			wantErr:   true,
		},
		{
			name:      "Unknown chain returns error",
			keyType:   sagecrypto.KeyTypeSecp256k1,
			chainType: Chain("unknown"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := factory.ValidateKeyTypeForChain(tt.keyType, tt.chainType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetRFC9421Algorithm(t *testing.T) {
	factory := NewClientFactory()

	tests := []struct {
		name     string
		keyType  sagecrypto.KeyType
		expected string
		wantErr  bool
	}{
		{
			name:     "Secp256k1 maps to es256k",
			keyType:  sagecrypto.KeyTypeSecp256k1,
			expected: "es256k",
			wantErr:  false,
		},
		{
			name:     "Ed25519 maps to ed25519",
			keyType:  sagecrypto.KeyTypeEd25519,
			expected: "ed25519",
			wantErr:  false,
		},
		{
			name:     "RSA maps to rsa-pss-sha256",
			keyType:  sagecrypto.KeyTypeRSA,
			expected: "rsa-pss-sha256",
			wantErr:  false,
		},
		{
			name:    "X25519 has no RFC 9421 mapping",
			keyType: sagecrypto.KeyTypeX25519,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			algorithm, err := factory.GetRFC9421Algorithm(tt.keyType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, algorithm)
			}
		})
	}
}

func TestCreateClient_RequiresConfig(t *testing.T) {
	factory := NewClientFactory()

	client, err := factory.CreateClient(nil)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestCreateClient_UnsupportedChain(t *testing.T) {
	factory := NewClientFactory()

	config := &RegistryConfig{
		Chain:           Chain("unsupported"),
		RPCEndpoint:     "https://example.com",
		ContractAddress: "0x0000000000000000000000000000000000000000",
	}

	client, err := factory.CreateClient(config)
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "not supported")
}

func TestConvenienceFunctions(t *testing.T) {
	t.Run("GetRecommendedKeyType convenience function", func(t *testing.T) {
		keyType, err := GetRecommendedKeyType(ChainEthereum)
		require.NoError(t, err)
		assert.Equal(t, sagecrypto.KeyTypeSecp256k1, keyType)
	})

	t.Run("ValidateKeyTypeForChain convenience function", func(t *testing.T) {
		err := ValidateKeyTypeForChain(sagecrypto.KeyTypeSecp256k1, ChainEthereum)
		assert.NoError(t, err)
	})

	t.Run("GetRFC9421Algorithm convenience function", func(t *testing.T) {
		algorithm, err := GetRFC9421Algorithm(sagecrypto.KeyTypeEd25519)
		require.NoError(t, err)
		assert.Equal(t, "ed25519", algorithm)
	})
}

func TestIntegrationScenario(t *testing.T) {
	factory := NewClientFactory()

	t.Run("Complete workflow: Ethereum agent", func(t *testing.T) {
		// Step 1: Get recommended key type for Ethereum
		keyType, err := factory.GetRecommendedKeyType(ChainEthereum)
		require.NoError(t, err)
		assert.Equal(t, sagecrypto.KeyTypeSecp256k1, keyType)
		t.Logf("Step 1: Recommended key type for Ethereum: %s", keyType)

		// Step 2: Validate the key type is supported
		err = factory.ValidateKeyTypeForChain(keyType, ChainEthereum)
		require.NoError(t, err)
		t.Logf("Step 2: Key type validated for Ethereum")

		// Step 3: Get RFC 9421 algorithm for message signing
		algorithm, err := factory.GetRFC9421Algorithm(keyType)
		require.NoError(t, err)
		assert.Equal(t, "es256k", algorithm)
		t.Logf("Step 3: RFC 9421 algorithm: %s", algorithm)

		// Note: In actual usage (E2E tests), Step 4 would be to create the DID client
		// using factory.CreateClient(config). In unit tests, we don't import the
		// ethereum/solana packages, so the creators are not registered.
		t.Logf("Step 4: In E2E tests, would create DID client using factory.CreateClient()")
	})

	t.Run("Complete workflow: Solana agent", func(t *testing.T) {
		// Step 1: Get recommended key type for Solana
		keyType, err := factory.GetRecommendedKeyType(ChainSolana)
		require.NoError(t, err)
		assert.Equal(t, sagecrypto.KeyTypeEd25519, keyType)
		t.Logf("Step 1: Recommended key type for Solana: %s", keyType)

		// Step 2: Validate the key type is supported
		err = factory.ValidateKeyTypeForChain(keyType, ChainSolana)
		require.NoError(t, err)
		t.Logf("Step 2: Key type validated for Solana")

		// Step 3: Get RFC 9421 algorithm for message signing
		algorithm, err := factory.GetRFC9421Algorithm(keyType)
		require.NoError(t, err)
		assert.Equal(t, "ed25519", algorithm)
		t.Logf("Step 3: RFC 9421 algorithm: %s", algorithm)
	})

	t.Run("Error scenario: Wrong key type for chain", func(t *testing.T) {
		// Try to use Ed25519 for Ethereum (should fail)
		err := factory.ValidateKeyTypeForChain(sagecrypto.KeyTypeEd25519, ChainEthereum)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
		t.Logf("Correctly rejected Ed25519 for Ethereum")

		// Try to use Secp256k1 for Solana (should fail)
		err = factory.ValidateKeyTypeForChain(sagecrypto.KeyTypeSecp256k1, ChainSolana)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
		t.Logf("Correctly rejected Secp256k1 for Solana")
	})
}

func TestRegisterClientCreators(t *testing.T) {
	// Save original creators
	origEth := createEthereumClient
	origSol := createSolanaClient

	// Restore after test
	defer func() {
		createEthereumClient = origEth
		createSolanaClient = origSol
	}()

	t.Run("Register and use Ethereum creator", func(t *testing.T) {
		called := false
		RegisterEthereumClientCreator(func(config *RegistryConfig) (Client, error) {
			called = true
			return &mockClient{chainType: ChainEthereum}, nil
		})

		factory := NewClientFactory()
		config := &RegistryConfig{
			Chain:           ChainEthereum,
			RPCEndpoint:     "https://example.com",
			ContractAddress: "0x0000000000000000000000000000000000000000",
		}

		client, err := factory.CreateClient(config)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.True(t, called, "Ethereum creator should have been called")
	})

	t.Run("Register and use Solana creator", func(t *testing.T) {
		called := false
		RegisterSolanaClientCreator(func(config *RegistryConfig) (Client, error) {
			called = true
			return &mockClient{chainType: ChainSolana}, nil
		})

		factory := NewClientFactory()
		config := &RegistryConfig{
			Chain:           ChainSolana,
			RPCEndpoint:     "https://api.devnet.solana.com",
			ContractAddress: "11111111111111111111111111111111",
		}

		client, err := factory.CreateClient(config)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.True(t, called, "Solana creator should have been called")
	})
}
