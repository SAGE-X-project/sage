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
	
	// Initialize chain-specific clients
	// Note: This is commented out to avoid import cycles in tests
	// In production, you would use a factory pattern or dependency injection
	// to create the appropriate client based on the chain
	/*
	switch chain {
	case ChainEthereum:
		ethClient, err := ethereum.NewEthereumClient(config)
		if err != nil {
			return fmt.Errorf("failed to create Ethereum client: %w", err)
		}
		m.registry.AddRegistry(chain, ethClient, config)
		m.resolver.AddResolver(chain, ethClient)
		
	case ChainSolana:
		solClient, err := solana.NewSolanaClient(config)
		if err != nil {
			return fmt.Errorf("failed to create Solana client: %w", err)
		}
		m.registry.AddRegistry(chain, solClient, config)
		m.resolver.AddResolver(chain, solClient)
		
	default:
		return fmt.Errorf("unsupported chain: %s", chain)
	}
	*/
	
	// For now, just return a placeholder error
	return fmt.Errorf("chain client initialization not implemented in test mode")
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