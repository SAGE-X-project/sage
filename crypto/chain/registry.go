package chain

import (
	"crypto"
	"fmt"
	"sync"
)

// defaultRegistry implements the ChainRegistry interface
type defaultRegistry struct {
	providers map[ChainType]ChainProvider
	mu        sync.RWMutex
}

// NewRegistry creates a new chain registry
func NewRegistry() ChainRegistry {
	return &defaultRegistry{
		providers: make(map[ChainType]ChainProvider),
	}
}

// RegisterProvider registers a new chain provider
func (r *defaultRegistry) RegisterProvider(provider ChainProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	chainType := provider.ChainType()
	if _, exists := r.providers[chainType]; exists {
		return ErrProviderExists
	}

	r.providers[chainType] = provider
	return nil
}

// GetProvider returns a provider for the specified chain
func (r *defaultRegistry) GetProvider(chain ChainType) (ChainProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[chain]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, chain)
	}

	return provider, nil
}

// ListProviders returns all registered chain types
func (r *defaultRegistry) ListProviders() []ChainType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	chains := make([]ChainType, 0, len(r.providers))
	for chain := range r.providers {
		chains = append(chains, chain)
	}

	return chains
}

// GenerateAddresses generates addresses for all registered chains
func (r *defaultRegistry) GenerateAddresses(publicKey crypto.PublicKey) (map[ChainType]*Address, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	addresses := make(map[ChainType]*Address)
	
	for chainType, provider := range r.providers {
		// Get the first supported network as default
		networks := provider.SupportedNetworks()
		if len(networks) == 0 {
			continue
		}

		address, err := provider.GenerateAddress(publicKey, networks[0])
		if err != nil {
			// Some chains might not support certain key types
			if err == ErrInvalidPublicKey {
				continue
			}
			return nil, fmt.Errorf("failed to generate %s address: %w", chainType, err)
		}

		addresses[chainType] = address
	}

	return addresses, nil
}

// Global registry instance
var globalRegistry = NewRegistry()

// RegisterProvider registers a provider to the global registry
func RegisterProvider(provider ChainProvider) error {
	return globalRegistry.RegisterProvider(provider)
}

// GetProvider gets a provider from the global registry
func GetProvider(chain ChainType) (ChainProvider, error) {
	return globalRegistry.GetProvider(chain)
}

// ListProviders lists all providers in the global registry
func ListProviders() []ChainType {
	return globalRegistry.ListProviders()
}

// GenerateAddresses generates addresses using the global registry
func GenerateAddresses(publicKey crypto.PublicKey) (map[ChainType]*Address, error) {
	return globalRegistry.GenerateAddresses(publicKey)
}