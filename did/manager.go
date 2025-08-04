package did

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sage-x-project/sage/crypto"
)

// Manager provides a unified interface for DID operations across multiple chains
type Manager struct {
	registry  *MultiChainRegistry
	resolver  *MultiChainResolver
	verifier  *MetadataVerifier
	configs   map[Chain]*RegistryConfig
	mu        sync.RWMutex
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

// IsAgentRegistered checks if an agent is registered by DID
func (m *Manager) IsAgentRegistered(ctx context.Context, did AgentDID) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	chain, _, err := ParseDID(did)
	if err != nil {
		return false, fmt.Errorf("invalid DID format: %w", err)
	}
	
	// Use resolver to check if agent exists
	resolver, ok := m.resolver.resolvers[chain]
	if !ok {
		return false, fmt.Errorf("resolver for chain %s not configured", chain)
	}
	
	// Check if agent exists
	_, err = resolver.Resolve(ctx, did)
	if err != nil {
		// Check if it's a DID not found error
		if isDIDNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	
	return true, nil
}

// GetRegistrationStatus gets detailed registration status
func (m *Manager) GetRegistrationStatus(ctx context.Context, did AgentDID) (*RegistrationStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	chain, _, err := ParseDID(did)
	if err != nil {
		return nil, fmt.Errorf("invalid DID format: %w", err)
	}
	
	// Use resolver to get agent metadata
	resolver, ok := m.resolver.resolvers[chain]
	if !ok {
		return nil, fmt.Errorf("resolver for chain %s not configured", chain)
	}
	
	// Try to resolve the agent
	agent, err := resolver.Resolve(ctx, did)
	if err != nil {
		// Check if it's a DID not found error
		if isDIDNotFoundError(err) {
			return &RegistrationStatus{
				IsRegistered: false,
				IsActive:     false,
			}, nil
		}
		return nil, err
	}
	
	return &RegistrationStatus{
		IsRegistered: true,
		IsActive:     agent.IsActive,
		RegisteredAt: agent.CreatedAt,
		AgentID:      string(agent.DID),
	}, nil
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

// isDIDNotFoundError checks if an error indicates that a DID was not found
func isDIDNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	
	// Check if it's a DIDError with the right code
	if didErr, ok := err.(DIDError); ok && didErr.Code == "DID_NOT_FOUND" {
		return true
	}
	
	// Check by error message for compatibility
	return err.Error() == ErrDIDNotFound.Error()
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