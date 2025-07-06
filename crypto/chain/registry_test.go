package chain

import (
	"context"
	"crypto"
	"testing"

	sagecrypto "github.com/sage-x-project/sage/crypto"
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
		registry.RegisterProvider(&mockProvider{chainType: ChainTypeEthereum})
		registry.RegisterProvider(&mockProvider{chainType: ChainTypeSolana})

		chains = registry.ListProviders()
		assert.Len(t, chains, 2)
		assert.Contains(t, chains, ChainTypeEthereum)
		assert.Contains(t, chains, ChainTypeSolana)
	})
}

func TestGlobalRegistry(t *testing.T) {
	// Skip this test as it requires importing the provider packages
	// which creates an import cycle. The global registry is tested
	// via integration tests in cmd/sage-crypto
	t.Skip("Skipping global registry test to avoid import cycle")
}

func TestAddressGeneration(t *testing.T) {
	// Skip this test as it requires the global registry to have providers registered
	// which creates an import cycle. The address generation is tested
	// via integration tests in cmd/sage-crypto
	t.Skip("Skipping address generation test to avoid import cycle")
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