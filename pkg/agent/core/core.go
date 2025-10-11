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

package core

import (
	"context"

	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"

	// Initialize crypto implementations
	_ "github.com/sage-x-project/sage/internal/cryptoinit"
)

// Version of the core module
const Version = "0.1.0"

// Core represents the main entry point for SAGE core functionality
type Core struct {
	cryptoManager       *crypto.Manager
	didManager          *did.Manager
	verificationService *VerificationService
}

// New creates a new Core instance
func New() *Core {
	didManager := did.NewManager()

	return &Core{
		cryptoManager:       crypto.NewManager(),
		didManager:          didManager,
		verificationService: NewVerificationService(didManager),
	}
}

// ConfigureDID configures DID support for a specific blockchain
func (c *Core) ConfigureDID(chain did.Chain, config *did.RegistryConfig) error {
	return c.didManager.Configure(chain, config)
}

// GenerateKeyPair generates a new cryptographic key pair
func (c *Core) GenerateKeyPair(keyType crypto.KeyType) (crypto.KeyPair, error) {
	return c.cryptoManager.GenerateKeyPair(keyType)
}

// RegisterAgent registers a new AI agent on the blockchain
func (c *Core) RegisterAgent(ctx context.Context, chain did.Chain, req *did.RegistrationRequest) (*did.RegistrationResult, error) {
	return c.didManager.RegisterAgent(ctx, chain, req)
}

// ResolveAgent retrieves agent metadata by DID
func (c *Core) ResolveAgent(ctx context.Context, agentDID string) (*did.AgentMetadata, error) {
	return c.didManager.ResolveAgent(ctx, did.AgentDID(agentDID))
}

// VerifyAgentMessage verifies a message from an AI agent
func (c *Core) VerifyAgentMessage(ctx context.Context, message *rfc9421.Message, opts *rfc9421.VerificationOptions) (*VerificationResult, error) {
	return c.verificationService.VerifyAgentMessage(ctx, message, opts)
}

// VerifyMessageFromHeaders verifies a message constructed from headers
func (c *Core) VerifyMessageFromHeaders(ctx context.Context, headers map[string]string, body []byte, signature []byte) (*VerificationResult, error) {
	return c.verificationService.VerifyMessageFromHeaders(ctx, headers, body, signature)
}

// QuickVerify performs a quick signature verification
func (c *Core) QuickVerify(ctx context.Context, agentDID string, message []byte, signature []byte) error {
	return c.verificationService.QuickVerify(ctx, agentDID, message, signature)
}

// SignMessage signs a message with a key pair
func (c *Core) SignMessage(keyPair crypto.KeyPair, message []byte) ([]byte, error) {
	return keyPair.Sign(message)
}

// CreateRFC9421Message creates a new RFC-9421 compliant message
func (c *Core) CreateRFC9421Message(agentDID string, body []byte) *rfc9421.MessageBuilder {
	return rfc9421.NewMessageBuilder().
		WithAgentDID(agentDID).
		WithBody(body)
}

// GetSupportedChains returns the list of configured blockchain chains
func (c *Core) GetSupportedChains() []did.Chain {
	return c.didManager.GetSupportedChains()
}

// GetCryptoManager returns the crypto manager for advanced operations
func (c *Core) GetCryptoManager() *crypto.Manager {
	return c.cryptoManager
}

// GetDIDManager returns the DID manager for advanced operations
func (c *Core) GetDIDManager() *did.Manager {
	return c.didManager
}

// GetVerificationService returns the verification service for advanced operations
func (c *Core) GetVerificationService() *VerificationService {
	return c.verificationService
}
