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

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
)

// Client defines the interface for DID registry operations
type Client interface {
	// Register registers a new agent in the DID registry
	Register(ctx context.Context, req *RegistrationRequest) (*RegistrationResult, error)

	// Resolve retrieves agent metadata from the DID registry
	Resolve(ctx context.Context, agentDID AgentDID) (*AgentMetadata, error)

	// Update updates agent metadata in the DID registry
	Update(ctx context.Context, agentDID AgentDID, updates map[string]interface{}, keyPair sagecrypto.KeyPair) error

	// Deactivate marks an agent as inactive in the DID registry
	Deactivate(ctx context.Context, agentDID AgentDID, keyPair sagecrypto.KeyPair) error
}
