package chain

import (
	"context"
	"crypto"
	"testing"

	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		registry.RegisterProvider(&mockProvider{chainType: ChainTypeEthereum})
		registry.RegisterProvider(&mockProvider{chainType: ChainTypeSolana})

		chains = registry.ListProviders()
		assert.Len(t, chains, 2)
		assert.Contains(t, chains, ChainTypeEthereum)
		assert.Contains(t, chains, ChainTypeSolana)
	})
}

func TestGlobalRegistry(t *testing.T) {
	// Test that the global registry has registered providers
	t.Run("HasEthereumProvider", func(t *testing.T) {
		chains := ListProviders()
		assert.Contains(t, chains, ChainTypeEthereum)

		provider, err := GetProvider(ChainTypeEthereum)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("HasSolanaProvider", func(t *testing.T) {
		chains := ListProviders()
		assert.Contains(t, chains, ChainTypeSolana)

		provider, err := GetProvider(ChainTypeSolana)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})
}

func TestAddressGeneration(t *testing.T) {
	t.Run("GenerateAllAddresses", func(t *testing.T) {
		// Test with Ed25519 key (Solana compatible)
		ed25519Key, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		addresses, err := GenerateAddresses(ed25519Key.PublicKey())
		require.NoError(t, err)
		
		// Should have Solana address
		assert.Contains(t, addresses, ChainTypeSolana)
		assert.NotNil(t, addresses[ChainTypeSolana])

		// Should not have Ethereum address (incompatible key type)
		assert.NotContains(t, addresses, ChainTypeEthereum)
	})

	t.Run("GenerateAllAddressesSecp256k1", func(t *testing.T) {
		// Test with Secp256k1 key (Ethereum compatible)
		secp256k1Key, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		addresses, err := GenerateAddresses(secp256k1Key.PublicKey())
		require.NoError(t, err)
		
		// Should have Ethereum address
		assert.Contains(t, addresses, ChainTypeEthereum)
		assert.NotNil(t, addresses[ChainTypeEthereum])

		// Should not have Solana address (incompatible key type)
		assert.NotContains(t, addresses, ChainTypeSolana)
	})
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