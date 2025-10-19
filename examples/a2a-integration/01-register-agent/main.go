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

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	_ "github.com/sage-x-project/sage/internal/cryptoinit"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// Example 01: Multi-Key Agent Registration
//
// This example demonstrates how to register an agent with multiple cryptographic keys:
// - ECDSA (secp256k1) for Ethereum compatibility
// - Ed25519 for high-performance signing
// - X25519 for key agreement and encryption
//
// Prerequisites:
// 1. Local Hardhat node running: npx hardhat node
// 2. SageRegistryV4 deployed: npx hardhat run scripts/deploy-v4-local.js --network localhost
// 3. Environment variables set (see README.md)

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║     SAGE Example 01: Multi-Key Agent Registration        ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Get configuration from environment
	registryAddress := os.Getenv("REGISTRY_ADDRESS")
	rpcURL := os.Getenv("RPC_URL")
	privateKeyHex := os.Getenv("PRIVATE_KEY")

	if registryAddress == "" {
		registryAddress = "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9" // Default local deployment
	}
	if rpcURL == "" {
		rpcURL = "http://localhost:8545"
	}
	if privateKeyHex == "" {
		fmt.Println("❌ Error: PRIVATE_KEY environment variable not set")
		fmt.Println("   export PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
		os.Exit(1)
	}

	fmt.Println("📋 Configuration")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println("Registry Address:", registryAddress)
	fmt.Println("RPC URL:         ", rpcURL)
	fmt.Println()

	// Step 1: Generate cryptographic keys
	fmt.Println("🔑 Step 1: Generating Cryptographic Keys")
	fmt.Println("─────────────────────────────────────────────────────────")

	// Generate ECDSA key (primary key for Ethereum)
	fmt.Println("Generating ECDSA (secp256k1) key...")
	ecdsaKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	if err != nil {
		fmt.Printf("❌ Failed to generate ECDSA key: %v\n", err)
		os.Exit(1)
	}
	ecdsaPubKey, err := did.MarshalPublicKey(ecdsaKeyPair.PublicKey())
	if err != nil {
		fmt.Printf("❌ Failed to marshal ECDSA public key: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ ECDSA key generated (%d bytes)\n", len(ecdsaPubKey))

	// Generate Ed25519 key (for signing)
	fmt.Println("Generating Ed25519 key...")
	ed25519KeyPair, err := crypto.GenerateEd25519KeyPair()
	if err != nil {
		fmt.Printf("❌ Failed to generate Ed25519 key: %v\n", err)
		os.Exit(1)
	}
	ed25519PubKey, err := did.MarshalPublicKey(ed25519KeyPair.PublicKey())
	if err != nil {
		fmt.Printf("❌ Failed to marshal Ed25519 public key: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Ed25519 key generated (%d bytes)\n", len(ed25519PubKey))

	// Generate X25519 key (for encryption/key agreement)
	fmt.Println("Generating X25519 key...")
	x25519KeyPair, err := crypto.GenerateX25519KeyPair()
	if err != nil {
		fmt.Printf("❌ Failed to generate X25519 key: %v\n", err)
		os.Exit(1)
	}
	x25519PubKey, err := did.MarshalPublicKey(x25519KeyPair.PublicKey())
	if err != nil {
		fmt.Printf("❌ Failed to marshal X25519 public key: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ X25519 key generated (%d bytes)\n", len(x25519PubKey))
	fmt.Println()

	// Step 2: Create DID manager and configure chain
	fmt.Println("🔗 Step 2: Connecting to Blockchain")
	fmt.Println("─────────────────────────────────────────────────────────")

	manager := did.NewManager()
	config := &did.RegistryConfig{
		ContractAddress:    registryAddress,
		RPCEndpoint:        rpcURL,
		PrivateKey:         privateKeyHex,
		GasPrice:           0, // Use default
		MaxRetries:         10,
		ConfirmationBlocks: 0, // Instant for local network
	}

	err = manager.Configure(did.ChainEthereum, config)
	if err != nil {
		fmt.Printf("❌ Failed to configure manager: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Connected to Ethereum (localhost)")
	fmt.Println()

	// Step 3: Prepare registration request
	fmt.Println("📝 Step 3: Preparing Registration Request")
	fmt.Println("─────────────────────────────────────────────────────────")

	agentDID := did.GenerateDID(did.ChainEthereum, "example-agent-"+time.Now().Format("20060102150405"))
	fmt.Println("Agent DID:   ", agentDID)

	// Generate signatures for additional keys
	// Note: ECDSA signature is generated automatically by the client
	// Ed25519 and X25519 keys need to be prepared

	req := &did.RegistrationRequest{
		DID:         agentDID,
		Name:        "Multi-Key Example Agent",
		Description: "Example agent demonstrating multi-key registration with ECDSA, Ed25519, and X25519",
		Endpoint:    "https://agent.example.com",
		Capabilities: map[string]interface{}{
			"messaging":  true,
			"encryption": true,
			"signing":    true,
		},
		KeyPair: ecdsaKeyPair, // Primary key
		Keys: []did.AgentKey{
			{
				Type:    did.KeyTypeEd25519,
				KeyData: ed25519PubKey,
				// Signature will be empty for Ed25519 on Ethereum (requires approval)
			},
			{
				Type:    did.KeyTypeX25519,
				KeyData: x25519PubKey,
				// Signature will be empty for X25519 (no verification needed for KEM keys)
			},
		},
	}

	fmt.Println("Agent Name:  ", req.Name)
	fmt.Println("Endpoint:    ", req.Endpoint)
	fmt.Printf("Keys:         %d (ECDSA + Ed25519 + X25519)\n", 1+len(req.Keys))
	fmt.Println()

	// Step 4: Register agent
	fmt.Println("🚀 Step 4: Registering Agent on Blockchain")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println("⏳ Submitting transaction...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := manager.RegisterAgent(ctx, did.ChainEthereum, req)
	if err != nil {
		fmt.Printf("❌ Registration failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✅ Agent Registered Successfully!")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println("Transaction Hash:", result.TransactionHash)
	fmt.Println("Block Number:    ", result.BlockNumber)
	fmt.Println("Gas Used:        ", result.GasUsed)
	fmt.Println()

	// Step 5: Verify registration
	fmt.Println("🔍 Step 5: Verifying Registration")
	fmt.Println("─────────────────────────────────────────────────────────")

	agent, err := manager.ResolveAgent(ctx, agentDID)
	if err != nil {
		fmt.Printf("❌ Failed to resolve agent: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Agent DID:       ", agent.DID)
	fmt.Println("Agent Name:      ", agent.Name)
	fmt.Println("Agent Endpoint:  ", agent.Endpoint)
	fmt.Println("Owner Address:   ", agent.Owner)
	fmt.Println("Active:          ", agent.IsActive)
	fmt.Println()

	// Step 6: Ed25519 key approval notice
	fmt.Println("⚠️  Step 6: Ed25519 Key Approval Required")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println("Ed25519 keys require approval by the registry contract owner.")
	fmt.Println("Until approved, the Ed25519 key will be marked as 'unverified'.")
	fmt.Println()
	fmt.Println("To approve (as contract owner):")
	fmt.Println("  ./build/bin/sage-did key approve <keyhash> \\")
	fmt.Println("    --chain ethereum \\")
	fmt.Println("    --contract-address", registryAddress, "\\")
	fmt.Println("    --rpc-url", rpcURL, "\\")
	fmt.Println("    --private-key $OWNER_PRIVATE_KEY")
	fmt.Println()

	// Summary
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║     Registration Complete!                                ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("🎉 Success! Your multi-key agent is now registered on-chain.")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Approve the Ed25519 key (see command above)")
	fmt.Println("  2. Run example 02 to generate an A2A Agent Card")
	fmt.Println("  3. Use the agent for secure communication")
	fmt.Println()
	fmt.Println("Agent DID:", agentDID)
	fmt.Println()
}
