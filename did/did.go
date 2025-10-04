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