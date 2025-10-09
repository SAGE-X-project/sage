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
	"fmt"
	"strings"
	"sync"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
)

// Manager provides a unified interface for DID operations across multiple chains
type Manager struct {
	registry *MultiChainRegistry
	resolver *MultiChainResolver
	verifier *MetadataVerifier
	configs  map[Chain]*RegistryConfig
	mu       sync.RWMutex
}

// NewManager creates a new DID manager
func NewManager() *Manager {
	resolver := NewMultiChainResolver()
	return &Manager{
		registry: NewMultiChainRegistry(),
		resolver: resolver,
		verifier: NewMetadataVerifier(resolver),
		configs:  make(map[Chain]*RegistryConfig),
	}
}

// Configure adds configuration for a specific chain
func (m *Manager) Configure(chain Chain, config *RegistryConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate configuration
	if config.ContractAddress == "" {
		return fmt.Errorf("contract address is required")
	}
	if config.RPCEndpoint == "" {
		return fmt.Errorf("RPC endpoint is required")
	}

	// Store configuration
	m.configs[chain] = config

	// Note: In production, chain-specific clients should be initialized here
	// using a factory pattern or dependency injection to avoid import cycles.
	// For now, clients must be added separately using SetClient method.

	return nil
}

// SetClient sets a pre-initialized client for a specific chain
// This method is used to avoid import cycles in the package structure
func (m *Manager) SetClient(chain Chain, client interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify the client implements required interfaces
	registry, ok := client.(Registry)
	if !ok {
		return fmt.Errorf("client does not implement Registry interface")
	}

	resolver, ok := client.(Resolver)
	if !ok {
		return fmt.Errorf("client does not implement Resolver interface")
	}

	// Get configuration for the chain
	config, exists := m.configs[chain]
	if !exists {
		return fmt.Errorf("configuration not found for chain %s", chain)
	}

	// Add to registry and resolver
	m.registry.AddRegistry(chain, registry, config)
	m.resolver.AddResolver(chain, resolver)

	return nil
}

// RegisterAgent registers a new AI agent on the specified chain
func (m *Manager) RegisterAgent(ctx context.Context, chain Chain, req *RegistrationRequest) (*RegistrationResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry.Register(ctx, chain, req)
}

// ResolveAgent retrieves agent metadata by DID
func (m *Manager) ResolveAgent(ctx context.Context, did AgentDID) (*AgentMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.resolver.Resolve(ctx, did)
}

// ResolvePublicKey retrieves only the public key for an agent
func (m *Manager) ResolvePublicKey(ctx context.Context, did AgentDID) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.resolver.ResolvePublicKey(ctx, did)
}

// UpdateAgent updates agent metadata
func (m *Manager) UpdateAgent(ctx context.Context, did AgentDID, updates map[string]interface{}, keyPair crypto.KeyPair) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry.Update(ctx, did, updates, keyPair)
}

// DeactivateAgent deactivates an agent
func (m *Manager) DeactivateAgent(ctx context.Context, did AgentDID, keyPair crypto.KeyPair) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry.Deactivate(ctx, did, keyPair)
}

// ValidateAgent validates an agent's DID and metadata
func (m *Manager) ValidateAgent(ctx context.Context, did AgentDID, opts *ValidationOptions) (*AgentMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.verifier.ValidateAgent(ctx, did, opts)
}

// CheckCapabilities verifies if an agent has specific capabilities
func (m *Manager) CheckCapabilities(ctx context.Context, did AgentDID, requiredCapabilities []string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.verifier.CheckCapabilities(ctx, did, requiredCapabilities)
}

// ListAgentsByOwner lists all agents owned by a specific address
func (m *Manager) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*AgentMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.resolver.ListAgentsByOwner(ctx, ownerAddress)
}

// SearchAgents searches for agents matching criteria
func (m *Manager) SearchAgents(ctx context.Context, criteria SearchCriteria) ([]*AgentMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.resolver.Search(ctx, criteria)
}

// GetRegistrationStatus checks the status of a registration transaction
func (m *Manager) GetRegistrationStatus(ctx context.Context, chain Chain, txHash string) (*RegistrationResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry.GetRegistrationStatus(ctx, chain, txHash)
}

// GetSupportedChains returns the list of configured chains
func (m *Manager) GetSupportedChains() []Chain {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chains := make([]Chain, 0, len(m.configs))
	for chain := range m.configs {
		chains = append(chains, chain)
	}
	return chains
}

// IsChainConfigured checks if a chain is configured
func (m *Manager) IsChainConfigured(chain Chain) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.configs[chain]
	return exists
}

// GenerateDID generates a new DID for an agent
func GenerateDID(chain Chain, identifier string) AgentDID {
	return AgentDID(fmt.Sprintf("did:sage:%s:%s", chain, identifier))
}

// ParseDID parses a DID and extracts chain and identifier
func ParseDID(did AgentDID) (chain Chain, identifier string, err error) {
	parts := strings.Split(string(did), ":")
	if len(parts) < 4 || parts[0] != "did" || parts[1] != "sage" {
		return "", "", fmt.Errorf("invalid DID format")
	}

	switch parts[2] {
	case "ethereum", "eth":
		chain = ChainEthereum
	case "solana", "sol":
		chain = ChainSolana
	default:
		return "", "", fmt.Errorf("unknown chain: %s", parts[2])
	}

	identifier = strings.Join(parts[3:], ":")
	return chain, identifier, nil
}
