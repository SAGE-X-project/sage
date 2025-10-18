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

package ethereum

import (
	"context"
	"os"
	"testing"
	"time"

	_ "github.com/sage-x-project/sage/internal/cryptoinit" // Initialize crypto wrappers
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// TestV4SingleKeyRegistration tests single-key agent registration with V4 contract
func TestV4SingleKeyRegistration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	// Configuration for local Hardhat/Anvil network
	config := &did.RegistryConfig{
		ContractAddress: "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9", // From deployment (X25519 removed)
		RPCEndpoint:     "http://localhost:8545",
		PrivateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", // Hardhat test account #0
		GasPrice:        0, // Let the client determine gas price
		MaxRetries:      10,
		ConfirmationBlocks: 0, // No need to wait for confirmations on local network
	}

	// Create V4 client
	client, err := NewEthereumClientV4(config)
	if err != nil {
		t.Fatalf("Failed to create V4 client: %v", err)
	}

	// Generate a test keypair
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	// Create registration request
	testDID := did.AgentDID("did:sage:eth:test-agent-" + time.Now().Format("20060102150405"))
	req := &did.RegistrationRequest{
		DID:         testDID,
		Name:        "Test Agent V4",
		Description: "Single-key test agent for V4 contract",
		Endpoint:    "http://localhost:8080",
		Capabilities: map[string]interface{}{
			"protocols": []string{"http", "grpc"},
			"version":   "1.0.0",
		},
		KeyPair: keyPair,
	}

	// Register the agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("Registering agent with V4 contract...")
	result, err := client.Register(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	t.Logf("✓ Agent registered successfully!")
	t.Logf("  Transaction Hash: %s", result.TransactionHash)
	t.Logf("  Block Number: %d", result.BlockNumber)
	t.Logf("  Gas Used: %d", result.GasUsed)

	// Verify registration by resolving the agent
	t.Log("Resolving registered agent...")
	agent, err := client.Resolve(ctx, testDID)
	if err != nil {
		t.Fatalf("Failed to resolve agent: %v", err)
	}

	// Verify metadata
	if agent.DID != testDID {
		t.Errorf("DID mismatch: got %s, want %s", agent.DID, testDID)
	}
	if agent.Name != req.Name {
		t.Errorf("Name mismatch: got %s, want %s", agent.Name, req.Name)
	}
	if agent.Description != req.Description {
		t.Errorf("Description mismatch: got %s, want %s", agent.Description, req.Description)
	}
	if agent.Endpoint != req.Endpoint {
		t.Errorf("Endpoint mismatch: got %s, want %s", agent.Endpoint, req.Endpoint)
	}
	if !agent.IsActive {
		t.Error("Agent should be active")
	}

	t.Logf("✓ Agent resolved successfully!")
	t.Logf("  DID: %s", agent.DID)
	t.Logf("  Name: %s", agent.Name)
	t.Logf("  Owner: %s", agent.Owner)
	t.Logf("  Active: %v", agent.IsActive)
}

// TestV4MultiKeyRegistration tests multi-key agent registration with V4 contract
func TestV4MultiKeyRegistration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	// Configuration for local Hardhat/Anvil network
	config := &did.RegistryConfig{
		ContractAddress: "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9",
		RPCEndpoint:     "http://localhost:8545",
		PrivateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
		GasPrice:        0,
		MaxRetries:      10,
		ConfirmationBlocks: 0,
	}

	// Create V4 client
	client, err := NewEthereumClientV4(config)
	if err != nil {
		t.Fatalf("Failed to create V4 client: %v", err)
	}

	// Generate keypairs for multi-key test
	manager := crypto.NewManager()

	ecdsaKeyPair, err := manager.GenerateKeyPair(crypto.KeyTypeSecp256k1)
	if err != nil {
		t.Fatalf("Failed to generate ECDSA keypair: %v", err)
	}

	x25519KeyPair, err := manager.GenerateKeyPair(crypto.KeyTypeX25519)
	if err != nil {
		t.Fatalf("Failed to generate X25519 keypair: %v", err)
	}

	// Marshal public keys
	ecdsaPubKey, err := did.MarshalPublicKey(ecdsaKeyPair.PublicKey())
	if err != nil {
		t.Fatalf("Failed to marshal ECDSA public key: %v", err)
	}

	x25519PubKey, err := did.MarshalPublicKey(x25519KeyPair.PublicKey())
	if err != nil {
		t.Fatalf("Failed to marshal X25519 public key: %v", err)
	}

	// Prepare registration message
	testDID := did.AgentDID("did:sage:eth:multikey-" + time.Now().Format("20060102150405"))

	// Create registration request with multiple keys
	req := &did.RegistrationRequest{
		DID:         testDID,
		Name:        "Multi-Key Test Agent V4",
		Description: "Multi-key test agent for V4 contract",
		Endpoint:    "http://localhost:8080",
		Capabilities: map[string]interface{}{
			"protocols": []string{"http", "grpc"},
			"version":   "2.0.0",
			"multikey":  true,
		},
		KeyPair: ecdsaKeyPair, // Primary key for signing
		Keys: []did.AgentKey{
			{
				Type:    did.KeyTypeECDSA,
				KeyData: ecdsaPubKey,
			},
			{
				Type:    did.KeyTypeX25519,
				KeyData: x25519PubKey,
			},
		},
	}

	// Register the agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("Registering multi-key agent with V4 contract...")
	result, err := client.Register(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register multi-key agent: %v", err)
	}

	t.Logf("✓ Multi-key agent registered successfully!")
	t.Logf("  Transaction Hash: %s", result.TransactionHash)
	t.Logf("  Block Number: %d", result.BlockNumber)
	t.Logf("  Gas Used: %d", result.GasUsed)

	// Verify registration
	t.Log("Resolving multi-key agent...")
	agent, err := client.Resolve(ctx, testDID)
	if err != nil {
		t.Fatalf("Failed to resolve multi-key agent: %v", err)
	}

	t.Logf("✓ Multi-key agent resolved successfully!")
	t.Logf("  DID: %s", agent.DID)
	t.Logf("  Name: %s", agent.Name)
	t.Logf("  Keys: %d registered", len(req.Keys))
}
