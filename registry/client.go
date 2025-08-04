// Package registry provides client interfaces for agent registry operations
package registry

import (
	"context"

	"github.com/sage-x-project/sage/did"
)

// Client defines the interface for agent registry operations
type Client interface {
	// IsAgentRegistered checks if an agent is registered by DID
	IsAgentRegistered(ctx context.Context, agentDID string) (bool, error)

	// GetRegistrationStatus gets detailed registration status
	GetRegistrationStatus(ctx context.Context, agentDID string) (*did.RegistrationStatus, error)

	// RegisterAgent registers a new agent
	RegisterAgent(ctx context.Context, req *did.RegistrationRequest) (*did.RegistrationResult, error)

	// UpdateAgent updates agent metadata
	UpdateAgent(ctx context.Context, agentDID string, req *UpdateRequest) error

	// DeactivateAgent deactivates an agent
	DeactivateAgent(ctx context.Context, agentDID string) error

	// GetAgentByDID retrieves agent metadata by DID
	GetAgentByDID(ctx context.Context, agentDID string) (*did.AgentMetadata, error)

	// GetAgentsByOwner retrieves all agents owned by an address
	GetAgentsByOwner(ctx context.Context, owner string) ([]string, error)
}

// UpdateRequest contains the data needed to update an agent
type UpdateRequest struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Endpoint     string                 `json:"endpoint"`
	Capabilities map[string]interface{} `json:"capabilities"`
}

// ClientConfig contains configuration for registry clients
type ClientConfig struct {
	RPC      string `json:"rpc"`
	Contract string `json:"contract"`
	ChainID  uint64 `json:"chain_id,omitempty"` // For Ethereum
}