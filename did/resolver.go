package did

import (
	"context"
	"fmt"
)

// Resolver defines the interface for DID resolution
type Resolver interface {
	// Resolve retrieves agent metadata by DID
	Resolve(ctx context.Context, did AgentDID) (*AgentMetadata, error)
	
	// ResolvePublicKey retrieves only the public key for an agent
	ResolvePublicKey(ctx context.Context, did AgentDID) (interface{}, error)
	
	// VerifyMetadata checks if the provided metadata matches the on-chain data
	VerifyMetadata(ctx context.Context, did AgentDID, metadata *AgentMetadata) (*VerificationResult, error)
	
	// ListAgentsByOwner retrieves all agents owned by a specific address
	ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*AgentMetadata, error)
	
	// Search finds agents matching the given criteria
	Search(ctx context.Context, criteria SearchCriteria) ([]*AgentMetadata, error)
}

// SearchCriteria defines search parameters for finding agents
type SearchCriteria struct {
	Name         string                 // Partial name match
	Capabilities map[string]interface{} // Required capabilities
	ActiveOnly   bool                   // Filter only active agents
	Limit        int                    // Maximum results
	Offset       int                    // Pagination offset
}

// MultiChainResolver aggregates multiple chain-specific resolvers
type MultiChainResolver struct {
	resolvers map[Chain]Resolver
}

// NewMultiChainResolver creates a new multi-chain resolver
func NewMultiChainResolver() *MultiChainResolver {
	return &MultiChainResolver{
		resolvers: make(map[Chain]Resolver),
	}
}

// AddResolver adds a chain-specific resolver
func (m *MultiChainResolver) AddResolver(chain Chain, resolver Resolver) {
	m.resolvers[chain] = resolver
}

// Resolve attempts to resolve a DID across all configured chains
func (m *MultiChainResolver) Resolve(ctx context.Context, did AgentDID) (*AgentMetadata, error) {
	chain, err := extractChainFromDID(did)
	if err != nil {
		// Try all chains if chain cannot be determined from DID
		for _, resolver := range m.resolvers {
			metadata, err := resolver.Resolve(ctx, did)
			if err == nil {
				return metadata, nil
			}
		}
		return nil, fmt.Errorf("DID not found on any chain: %s", did)
	}
	
	resolver, exists := m.resolvers[chain]
	if !exists {
		return nil, fmt.Errorf("no resolver for chain %s", chain)
	}
	
	return resolver.Resolve(ctx, did)
}

// ResolvePublicKey retrieves the public key for an agent from any chain
func (m *MultiChainResolver) ResolvePublicKey(ctx context.Context, did AgentDID) (interface{}, error) {
	metadata, err := m.Resolve(ctx, did)
	if err != nil {
		return nil, err
	}
	
	if !metadata.IsActive {
		return nil, ErrInactiveAgent
	}
	
	return metadata.PublicKey, nil
}

// VerifyMetadata verifies metadata against on-chain data
func (m *MultiChainResolver) VerifyMetadata(ctx context.Context, did AgentDID, metadata *AgentMetadata) (*VerificationResult, error) {
	chain, err := extractChainFromDID(did)
	if err != nil {
		return nil, err
	}
	
	resolver, exists := m.resolvers[chain]
	if !exists {
		return nil, fmt.Errorf("no resolver for chain %s", chain)
	}
	
	return resolver.VerifyMetadata(ctx, did, metadata)
}

// ListAgentsByOwner lists agents across all chains
func (m *MultiChainResolver) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*AgentMetadata, error) {
	var allAgents []*AgentMetadata
	
	for _, resolver := range m.resolvers {
		agents, err := resolver.ListAgentsByOwner(ctx, ownerAddress)
		if err != nil {
			// Continue with other chains even if one fails
			continue
		}
		allAgents = append(allAgents, agents...)
	}
	
	return allAgents, nil
}

// Search searches for agents across all chains
func (m *MultiChainResolver) Search(ctx context.Context, criteria SearchCriteria) ([]*AgentMetadata, error) {
	var allAgents []*AgentMetadata
	
	for _, resolver := range m.resolvers {
		agents, err := resolver.Search(ctx, criteria)
		if err != nil {
			continue
		}
		allAgents = append(allAgents, agents...)
	}
	
	// Apply limit after aggregating results
	if criteria.Limit > 0 && len(allAgents) > criteria.Limit {
		allAgents = allAgents[:criteria.Limit]
	}
	
	return allAgents, nil
}

// extractChainFromDID attempts to determine the chain from a DID
// Format: did:sage:chain:identifier
func extractChainFromDID(did AgentDID) (Chain, error) {
	didStr := string(did)
	if len(didStr) < 10 || didStr[:4] != "did:" {
		return "", fmt.Errorf("invalid DID format")
	}
	
	// Simple extraction - can be enhanced based on actual DID format
	if len(didStr) > 14 && didStr[4:9] == "sage:" {
		parts := didStr[9:]
		if len(parts) > 4 {
			switch parts[:3] {
			case "eth":
				return ChainEthereum, nil
			case "sol":
				return ChainSolana, nil
			}
		}
	}
	
	return "", fmt.Errorf("cannot determine chain from DID")
}