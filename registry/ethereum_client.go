package registry

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/did/ethereum"
)

// EthereumClient implements the Client interface for Ethereum
type EthereumClient struct {
	client       *ethclient.Client
	didClient    *ethereum.EthereumClient // Use existing DID client
	contractAddr common.Address
}

// NewEthereumClient creates a new Ethereum registry client
func NewEthereumClient(config *ClientConfig) (*EthereumClient, error) {
	client, err := ethclient.Dial(config.RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum: %w", err)
	}

	contractAddr := common.HexToAddress(config.Contract)
	
	// Create DID client with proper configuration
	didConfig := &did.RegistryConfig{
		RPCEndpoint:      config.RPC,
		ContractAddress:  config.Contract,
	}
	didClient, err := ethereum.NewEthereumClient(didConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create DID client: %w", err)
	}

	return &EthereumClient{
		client:       client,
		didClient:    didClient,
		contractAddr: contractAddr,
	}, nil
}

// IsAgentRegistered checks if an agent is registered by DID
func (c *EthereumClient) IsAgentRegistered(ctx context.Context, agentDID string) (bool, error) {
	// TODO: Implement actual contract call when contract binding is available
	// For now, use DID client to check if agent exists
	_, err := c.didClient.Resolve(ctx, did.AgentDID(agentDID))
	if err != nil {
		// If agent doesn't exist, it's not registered
		return false, nil
	}
	return true, nil
}

// GetRegistrationStatus gets detailed registration status
func (c *EthereumClient) GetRegistrationStatus(ctx context.Context, agentDID string) (*did.RegistrationStatus, error) {
	// TODO: Implement actual contract call when contract binding is available
	// For now, use DID client to get agent metadata
	metadata, err := c.didClient.Resolve(ctx, did.AgentDID(agentDID))
	if err != nil {
		// If agent doesn't exist, return unregistered status
		return &did.RegistrationStatus{
			IsRegistered: false,
			IsActive:     false,
		}, nil
	}

	return &did.RegistrationStatus{
		IsRegistered: true,
		IsActive:     metadata.IsActive,
		RegisteredAt: metadata.CreatedAt,
		AgentID:      string(metadata.DID),
	}, nil
}

// RegisterAgent registers a new agent
func (c *EthereumClient) RegisterAgent(ctx context.Context, req *did.RegistrationRequest) (*did.RegistrationResult, error) {
	// This would require implementing the transaction signing logic
	// For now, returning a placeholder
	return nil, fmt.Errorf("not implemented")
}

// UpdateAgent updates agent metadata
func (c *EthereumClient) UpdateAgent(ctx context.Context, agentDID string, req *UpdateRequest) error {
	// This would require implementing the transaction signing logic
	return fmt.Errorf("not implemented")
}

// DeactivateAgent deactivates an agent
func (c *EthereumClient) DeactivateAgent(ctx context.Context, agentDID string) error {
	// This would require implementing the transaction signing logic
	return fmt.Errorf("not implemented")
}

// GetAgentByDID retrieves agent metadata by DID
func (c *EthereumClient) GetAgentByDID(ctx context.Context, agentDID string) (*did.AgentMetadata, error) {
	// Use DID client to resolve agent metadata
	return c.didClient.Resolve(ctx, did.AgentDID(agentDID))
}

// GetAgentsByOwner retrieves all agents owned by an address
func (c *EthereumClient) GetAgentsByOwner(ctx context.Context, owner string) ([]string, error) {
	// TODO: Implement actual contract call when contract binding is available
	// For now, return empty list
	return []string{}, nil
}