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
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
)

// EthereumV4ClientCreator is a factory function type for creating Ethereum V4 clients
type EthereumV4ClientCreator func(*RegistryConfig) (interface{}, error)

var (
	ethereumV4ClientCreator EthereumV4ClientCreator
	ethereumV4CreatorMu     sync.RWMutex
)

// RegisterEthereumV4ClientCreator registers a factory function for Ethereum V4 clients
// This is called by the ethereum package's init() function to avoid import cycles
func RegisterEthereumV4ClientCreator(creator EthereumV4ClientCreator) {
	ethereumV4CreatorMu.Lock()
	defer ethereumV4CreatorMu.Unlock()
	ethereumV4ClientCreator = creator
}

// GetEthereumV4ClientCreator returns the registered Ethereum V4 client creator
func GetEthereumV4ClientCreator() EthereumV4ClientCreator {
	ethereumV4CreatorMu.RLock()
	defer ethereumV4CreatorMu.RUnlock()
	return ethereumV4ClientCreator
}

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

// Configure adds configuration for a specific chain and automatically initializes V4 client
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

	// Auto-initialize chain-specific V4 client
	// This avoids the need for manual SetClient calls in CLI code
	switch chain {
	case ChainEthereum:
		// Dynamically import and create V4 client to avoid import cycles
		// The ethereum package registers a factory function via init()
		if creator := GetEthereumV4ClientCreator(); creator != nil {
			client, err := creator(config)
			if err != nil {
				return fmt.Errorf("failed to create Ethereum V4 client: %w", err)
			}
			// Set the client immediately
			if err := m.setClientUnlocked(chain, client); err != nil {
				return fmt.Errorf("failed to set V4 client: %w", err)
			}
		}
		// Fallback: clients must be added separately using SetClient method
		// This maintains backward compatibility
	case ChainSolana:
		// Solana V4 client initialization would go here
		// For now, clients must be added separately using SetClient method
	default:
		return fmt.Errorf("unsupported chain: %s", chain)
	}

	return nil
}

// setClientUnlocked sets a client without acquiring the lock (internal use only)
func (m *Manager) setClientUnlocked(chain Chain, client interface{}) error {
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

// AddKey adds a new cryptographic key to an existing agent
func (m *Manager) AddKey(ctx context.Context, chain Chain, did AgentDID, key AgentKey) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get registry for the chain
	registry := m.registry.GetRegistry(chain)
	if registry == nil {
		return "", fmt.Errorf("no registry configured for chain %s", chain)
	}

	// Check if registry supports V4 interface (key management)
	v4Registry, ok := registry.(RegistryV4)
	if !ok {
		return "", fmt.Errorf("registry for chain %s does not support multi-key management", chain)
	}

	// Add key via V4 interface
	return v4Registry.AddKey(ctx, did, key)
}

// RevokeKey revokes a cryptographic key from an agent
func (m *Manager) RevokeKey(ctx context.Context, chain Chain, did AgentDID, keyHash string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get registry for the chain
	registry := m.registry.GetRegistry(chain)
	if registry == nil {
		return fmt.Errorf("no registry configured for chain %s", chain)
	}

	// Check if registry supports V4 interface (key management)
	v4Registry, ok := registry.(RegistryV4)
	if !ok {
		return fmt.Errorf("registry for chain %s does not support multi-key management", chain)
	}

	// Revoke key via V4 interface
	return v4Registry.RevokeKey(ctx, did, keyHash)
}

// ApproveEd25519Key approves an Ed25519 key (registry owner only)
func (m *Manager) ApproveEd25519Key(ctx context.Context, chain Chain, keyHashStr string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Get registry for the chain
	registry := m.registry.GetRegistry(chain)
	if registry == nil {
		return fmt.Errorf("no registry configured for chain %s", chain)
	}

	// Check if registry supports V4 interface (key management)
	v4Registry, ok := registry.(RegistryV4)
	if !ok {
		return fmt.Errorf("registry for chain %s does not support multi-key management", chain)
	}

	// Call ApproveEd25519Key via V4 interface
	// The interface expects a string (hex-encoded key hash)
	return v4Registry.ApproveEd25519Key(ctx, keyHashStr)
}
