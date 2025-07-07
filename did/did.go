// Package did provides decentralized identifier (DID) management for AI agents.
// It supports multi-chain registration, resolution, and verification of agent identities.
package did

import (
	"context"
	"fmt"
)

// Version of the DID module
const Version = "0.1.0"

// Default DID manager instance
var defaultManager *Manager

func init() {
	defaultManager = NewManager()
}

// GetDefaultManager returns the default DID manager instance
func GetDefaultManager() *Manager {
	return defaultManager
}

// Configure configures the default manager for a specific chain
func Configure(chain Chain, config *RegistryConfig) error {
	return defaultManager.Configure(chain, config)
}

// RegisterAgent registers a new AI agent using the default manager
func RegisterAgent(ctx context.Context, chain Chain, req *RegistrationRequest) (*RegistrationResult, error) {
	return defaultManager.RegisterAgent(ctx, chain, req)
}

// ResolveAgent retrieves agent metadata using the default manager
func ResolveAgent(ctx context.Context, did AgentDID) (*AgentMetadata, error) {
	return defaultManager.ResolveAgent(ctx, did)
}

// ValidateAgent validates an agent's DID and metadata using the default manager
func ValidateAgent(ctx context.Context, did AgentDID, opts *ValidationOptions) (*AgentMetadata, error) {
	return defaultManager.ValidateAgent(ctx, did, opts)
}

// CheckCapabilities verifies if an agent has specific capabilities using the default manager
func CheckCapabilities(ctx context.Context, did AgentDID, requiredCapabilities []string) (bool, error) {
	return defaultManager.CheckCapabilities(ctx, did, requiredCapabilities)
}

// ValidateDID validates a DID format
func ValidateDID(did string) error {
	if len(did) < 10 {
		return fmt.Errorf("DID too short")
	}
	
	if did[:4] != "did:" {
		return fmt.Errorf("DID must start with 'did:'")
	}
	
	_, _, err := ParseDID(AgentDID(did))
	return err
}