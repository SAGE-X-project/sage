// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


package did

import (
	"context"
	"fmt"
	
	"github.com/sage-x-project/sage/crypto"
)

// Registry defines the interface for DID registry operations
type Registry interface {
	// Register registers a new agent on the blockchain
	Register(ctx context.Context, req *RegistrationRequest) (*RegistrationResult, error)
	
	// Update updates agent metadata
	Update(ctx context.Context, did AgentDID, updates map[string]interface{}, keyPair crypto.KeyPair) error
	
	// Deactivate deactivates an agent
	Deactivate(ctx context.Context, did AgentDID, keyPair crypto.KeyPair) error
	
	// GetRegistrationStatus checks the status of a registration transaction
	GetRegistrationStatus(ctx context.Context, txHash string) (*RegistrationResult, error)
}

// RegistryConfig contains configuration for a DID registry
type RegistryConfig struct {
	Chain            Chain
	Network          Network
	ContractAddress  string
	RPCEndpoint      string
	PrivateKey       string // For paying gas fees
	GasPrice         uint64
	MaxRetries       int
	ConfirmationBlocks int
}

// MultiChainRegistry manages registries across multiple chains
type MultiChainRegistry struct {
	registries map[Chain]Registry
	configs    map[Chain]*RegistryConfig
}

// NewMultiChainRegistry creates a new multi-chain registry
func NewMultiChainRegistry() *MultiChainRegistry {
	return &MultiChainRegistry{
		registries: make(map[Chain]Registry),
		configs:    make(map[Chain]*RegistryConfig),
	}
}

// AddRegistry adds a chain-specific registry
func (m *MultiChainRegistry) AddRegistry(chain Chain, registry Registry, config *RegistryConfig) {
	m.registries[chain] = registry
	m.configs[chain] = config
}

// Register registers an agent on the specified chain
func (m *MultiChainRegistry) Register(ctx context.Context, chain Chain, req *RegistrationRequest) (*RegistrationResult, error) {
	registry, exists := m.registries[chain]
	if !exists {
		return nil, fmt.Errorf("no registry for chain %s", chain)
	}
	
	// Validate the registration request
	if err := validateRegistrationRequest(req); err != nil {
		return nil, err
	}
	
	// Add chain prefix to DID if not present
	if !hasChainPrefix(req.DID, chain) {
		req.DID = addChainPrefix(req.DID, chain)
	}
	
	return registry.Register(ctx, req)
}

// Update updates agent metadata on the appropriate chain
func (m *MultiChainRegistry) Update(ctx context.Context, did AgentDID, updates map[string]interface{}, keyPair crypto.KeyPair) error {
	chain, err := extractChainFromDID(did)
	if err != nil {
		return err
	}
	
	registry, exists := m.registries[chain]
	if !exists {
		return fmt.Errorf("no registry for chain %s", chain)
	}
	
	return registry.Update(ctx, did, updates, keyPair)
}

// Deactivate deactivates an agent on the appropriate chain
func (m *MultiChainRegistry) Deactivate(ctx context.Context, did AgentDID, keyPair crypto.KeyPair) error {
	chain, err := extractChainFromDID(did)
	if err != nil {
		return err
	}
	
	registry, exists := m.registries[chain]
	if !exists {
		return fmt.Errorf("no registry for chain %s", chain)
	}
	
	return registry.Deactivate(ctx, did, keyPair)
}

// GetRegistrationStatus checks registration status on the specified chain
func (m *MultiChainRegistry) GetRegistrationStatus(ctx context.Context, chain Chain, txHash string) (*RegistrationResult, error) {
	registry, exists := m.registries[chain]
	if !exists {
		return nil, fmt.Errorf("no registry for chain %s", chain)
	}
	
	return registry.GetRegistrationStatus(ctx, txHash)
}

// validateRegistrationRequest validates a registration request
func validateRegistrationRequest(req *RegistrationRequest) error {
	if req.DID == "" {
		return fmt.Errorf("DID is required")
	}
	
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	
	if req.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	
	if req.KeyPair == nil {
		return fmt.Errorf("key pair is required for signing")
	}
	
	return nil
}

// hasChainPrefix checks if a DID has the specified chain prefix
func hasChainPrefix(did AgentDID, chain Chain) bool {
	prefix := fmt.Sprintf("did:sage:%s:", chain)
	return len(string(did)) > len(prefix) && string(did)[:len(prefix)] == prefix
}

// addChainPrefix adds a chain prefix to a DID
func addChainPrefix(did AgentDID, chain Chain) AgentDID {
	if string(did)[:4] == "did:" {
		// Replace existing prefix
		parts := string(did)[4:]
		return AgentDID(fmt.Sprintf("did:sage:%s:%s", chain, parts))
	}
	return AgentDID(fmt.Sprintf("did:sage:%s:%s", chain, did))
}