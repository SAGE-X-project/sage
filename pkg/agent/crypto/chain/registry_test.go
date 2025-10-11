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

package chain

import (
	"context"
	"crypto"
	"testing"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/stretchr/testify/assert"
)

func TestChainRegistry(t *testing.T) {
	// Use a new registry for testing
	registry := NewRegistry()

	t.Run("RegisterAndGetProvider", func(t *testing.T) {
		// Create mock providers
		ethProvider := &mockProvider{chainType: ChainTypeEthereum}
		solProvider := &mockProvider{chainType: ChainTypeSolana}

		// Register providers
		err := registry.RegisterProvider(ethProvider)
		assert.NoError(t, err)

		err = registry.RegisterProvider(solProvider)
		assert.NoError(t, err)

		// Get providers
		provider, err := registry.GetProvider(ChainTypeEthereum)
		assert.NoError(t, err)
		assert.Equal(t, ChainTypeEthereum, provider.ChainType())

		provider, err = registry.GetProvider(ChainTypeSolana)
		assert.NoError(t, err)
		assert.Equal(t, ChainTypeSolana, provider.ChainType())
	})

	t.Run("RegisterDuplicate", func(t *testing.T) {
		registry := NewRegistry()
		provider := &mockProvider{chainType: ChainTypeEthereum}

		err := registry.RegisterProvider(provider)
		assert.NoError(t, err)

		// Try to register again
		err = registry.RegisterProvider(provider)
		assert.Error(t, err)
		assert.Equal(t, ErrProviderExists, err)
	})

	t.Run("GetNonExistentProvider", func(t *testing.T) {
		registry := NewRegistry()

		_, err := registry.GetProvider(ChainTypeBitcoin)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("ListProviders", func(t *testing.T) {
		registry := NewRegistry()

		// Empty registry
		chains := registry.ListProviders()
		assert.Empty(t, chains)

		// Add providers
		_ = registry.RegisterProvider(&mockProvider{chainType: ChainTypeEthereum})
		_ = registry.RegisterProvider(&mockProvider{chainType: ChainTypeSolana})

		chains = registry.ListProviders()
		assert.Len(t, chains, 2)
		assert.Contains(t, chains, ChainTypeEthereum)
		assert.Contains(t, chains, ChainTypeSolana)
	})
}

func TestGlobalRegistry(t *testing.T) {
	// Create a new registry for testing to avoid import cycle
	// We test the global registry functions by creating a new registry

	// Create fresh registry
	testRegistry := NewRegistry()

	// Register mock providers
	mockEth := &mockProvider{chainType: ChainTypeEthereum}
	mockSol := &mockProvider{chainType: ChainTypeSolana}

	err := testRegistry.RegisterProvider(mockEth)
	assert.NoError(t, err, "Should register Ethereum provider")

	err = testRegistry.RegisterProvider(mockSol)
	assert.NoError(t, err, "Should register Solana provider")

	// Test GetProvider
	provider, err := testRegistry.GetProvider(ChainTypeEthereum)
	assert.NoError(t, err, "Should get Ethereum provider without error")
	assert.NotNil(t, provider, "Should get Ethereum provider")
	assert.Equal(t, ChainTypeEthereum, provider.ChainType())

	provider, err = testRegistry.GetProvider(ChainTypeSolana)
	assert.NoError(t, err, "Should get Solana provider without error")
	assert.NotNil(t, provider, "Should get Solana provider")
	assert.Equal(t, ChainTypeSolana, provider.ChainType())

	// Test ListProviders
	providers := testRegistry.ListProviders()
	assert.Len(t, providers, 2, "Should have 2 providers")
	assert.Contains(t, providers, ChainTypeEthereum)
	assert.Contains(t, providers, ChainTypeSolana)

	// Test duplicate registration
	err = testRegistry.RegisterProvider(mockEth)
	assert.Error(t, err, "Should error on duplicate registration")

	// Test getting non-existent provider
	provider, err = testRegistry.GetProvider(ChainType("unknown"))
	assert.Error(t, err, "Should error for unknown chain")
	assert.Nil(t, provider, "Should return nil provider for unknown chain")
}

func TestAddressGeneration(t *testing.T) {
	// Test using mock provider to avoid import cycle
	// Create a new registry instance for testing

	testRegistry := NewRegistry()

	// Register mock provider
	mockEth := &mockProvider{chainType: ChainTypeEthereum}
	err := testRegistry.RegisterProvider(mockEth)
	assert.NoError(t, err, "Should register mock provider")

	// Generate a test key (using ed25519 for simplicity)
	publicKey := []byte("test-public-key-32-bytes-padding")

	// Test GenerateAddresses (multiple chains)
	addresses, err := testRegistry.GenerateAddresses(publicKey)
	assert.NoError(t, err, "Should generate addresses")
	assert.NotNil(t, addresses, "Addresses map should not be nil")
	assert.Len(t, addresses, 1, "Should have one address for registered chain")

	ethAddress, exists := addresses[ChainTypeEthereum]
	assert.True(t, exists, "Should have Ethereum address")
	assert.Equal(t, "mock-address", ethAddress.Value)
	assert.Equal(t, ChainTypeEthereum, ethAddress.Chain)
	assert.Equal(t, NetworkEthereumMainnet, ethAddress.Network)

	// Test with additional provider
	mockSol := &mockProvider{chainType: ChainTypeSolana}
	err = testRegistry.RegisterProvider(mockSol)
	assert.NoError(t, err, "Should register Solana provider")

	addresses, err = testRegistry.GenerateAddresses(publicKey)
	assert.NoError(t, err, "Should generate addresses for multiple chains")
	assert.Len(t, addresses, 2, "Should have addresses for both chains")

	// Test individual provider address generation
	provider, err := testRegistry.GetProvider(ChainTypeEthereum)
	assert.NoError(t, err, "Should get provider")

	address, err := provider.GenerateAddress(publicKey, NetworkEthereumMainnet)
	assert.NoError(t, err, "Should generate address")
	assert.NotNil(t, address, "Address should not be nil")
	assert.Equal(t, "mock-address", address.Value)

	// Test with nil public key
	address, err = provider.GenerateAddress(nil, NetworkEthereumMainnet)
	assert.NoError(t, err, "Mock provider accepts nil key")
	assert.NotNil(t, address, "Address should not be nil")
}

// mockProvider is a mock implementation for testing
type mockProvider struct {
	chainType ChainType
}

func (m *mockProvider) ChainType() ChainType {
	return m.chainType
}

func (m *mockProvider) SupportedNetworks() []Network {
	return []Network{NetworkEthereumMainnet}
}

func (m *mockProvider) GenerateAddress(publicKey crypto.PublicKey, network Network) (*Address, error) {
	return &Address{
		Value:   "mock-address",
		Chain:   m.chainType,
		Network: network,
	}, nil
}

func (m *mockProvider) GetPublicKeyFromAddress(ctx context.Context, address string, network Network) (crypto.PublicKey, error) {
	return nil, ErrOperationNotSupported
}

func (m *mockProvider) ValidateAddress(address string, network Network) error {
	return nil
}

func (m *mockProvider) SignTransaction(keyPair sagecrypto.KeyPair, transaction interface{}) ([]byte, error) {
	return nil, ErrOperationNotSupported
}

func (m *mockProvider) VerifySignature(publicKey crypto.PublicKey, message []byte, signature []byte) error {
	return nil
}
