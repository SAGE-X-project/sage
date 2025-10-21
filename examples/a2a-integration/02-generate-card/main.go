//go:build ignore

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
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/sage-x-project/sage/internal/cryptoinit"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// Example 02: A2A Agent Card Generation
//
// This example demonstrates how to:
// 1. Resolve an agent's DID from the blockchain
// 2. Generate an A2A (Agent-to-Agent) Agent Card
// 3. Export the card to JSON format
// 4. Validate the card structure
//
// A2A Agent Cards are portable, standardized identity documents that agents
// can exchange for discovery and trust establishment.
//
// Prerequisites:
// 1. Example 01 completed (agent registered)
// 2. Agent DID from example 01
// 3. Local Hardhat node running
// 4. Environment variables set

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║     SAGE Example 02: A2A Agent Card Generation           ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Get configuration from environment
	registryAddress := os.Getenv("REGISTRY_ADDRESS")
	rpcURL := os.Getenv("RPC_URL")
	agentDIDStr := os.Getenv("AGENT_DID")

	if registryAddress == "" {
		registryAddress = "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
	}
	if rpcURL == "" {
		rpcURL = "http://localhost:8545"
	}
	if agentDIDStr == "" {
		fmt.Println("❌ Error: AGENT_DID environment variable not set")
		fmt.Println("   Please run example 01 first to register an agent")
		fmt.Println("   Then set: export AGENT_DID=did:sage:ethereum:...")
		os.Exit(1)
	}

	agentDID := did.AgentDID(agentDIDStr)

	fmt.Println("📋 Configuration")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println("Registry Address:", registryAddress)
	fmt.Println("RPC URL:         ", rpcURL)
	fmt.Println("Agent DID:       ", agentDID)
	fmt.Println()

	// Step 1: Connect to blockchain
	fmt.Println("🔗 Step 1: Connecting to Blockchain")
	fmt.Println("─────────────────────────────────────────────────────────")

	manager := did.NewManager()
	config := &did.RegistryConfig{
		ContractAddress:    registryAddress,
		RPCEndpoint:        rpcURL,
		PrivateKey:         "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
		GasPrice:           0,
		MaxRetries:         10,
		ConfirmationBlocks: 0,
	}

	err := manager.Configure(did.ChainEthereum, config)
	if err != nil {
		fmt.Printf("❌ Failed to configure manager: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Connected to Ethereum (localhost)")
	fmt.Println()

	// Step 2: Resolve agent metadata
	fmt.Println("🔍 Step 2: Resolving Agent from Blockchain")
	fmt.Println("─────────────────────────────────────────────────────────")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	agent, err := manager.ResolveAgent(ctx, agentDID)
	if err != nil {
		fmt.Printf("❌ Failed to resolve agent: %v\n", err)
		fmt.Println()
		fmt.Println("Make sure:")
		fmt.Println("  1. The agent DID is correct")
		fmt.Println("  2. The agent was registered (run example 01 first)")
		fmt.Println("  3. The blockchain node is running")
		os.Exit(1)
	}

	fmt.Println("Agent resolved successfully!")
	fmt.Println("  Name:       ", agent.Name)
	fmt.Println("  Endpoint:   ", agent.Endpoint)
	fmt.Println("  Owner:      ", agent.Owner)
	fmt.Println("  Active:     ", agent.IsActive)
	fmt.Println()

	// Step 3: Generate A2A Agent Card
	fmt.Println("🎴 Step 3: Generating A2A Agent Card")
	fmt.Println("─────────────────────────────────────────────────────────")

	// Convert AgentMetadata to AgentMetadataV4 for card generation
	metadataV4 := did.FromAgentMetadata(agent)

	card, err := did.GenerateA2ACard(metadataV4)
	if err != nil {
		fmt.Printf("❌ Failed to generate A2A card: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ A2A Agent Card generated successfully!")
	fmt.Println()
	fmt.Println("Card Details:")
	fmt.Println("  Type:        ", card.Type)
	fmt.Println("  Version:     ", card.Version)
	fmt.Println("  ID (DID):    ", card.ID)
	fmt.Println("  Name:        ", card.Name)
	fmt.Println("  Description: ", card.Description)
	fmt.Println("  Service URL: ", card.ServiceEndpoint)
	fmt.Printf("  Public Keys:  %d keys\n", len(card.PublicKeys))
	fmt.Println()

	// Display public keys
	fmt.Println("Public Keys:")
	for i, pubKey := range card.PublicKeys {
		fmt.Printf("  [%d] %s\n", i+1, pubKey.ID)
		fmt.Printf("      Type:       %s\n", pubKey.Type)
		fmt.Printf("      Controller: %s\n", pubKey.Controller)
		fmt.Printf("      Key Length: %d bytes\n", len(pubKey.PublicKeyMultibase))
		if pubKey.Purpose != "" {
			fmt.Printf("      Purpose:    %s\n", pubKey.Purpose)
		}
		fmt.Println()
	}

	// Step 4: Validate card
	fmt.Println("✅ Step 4: Validating A2A Card")
	fmt.Println("─────────────────────────────────────────────────────────")

	err = did.ValidateA2ACard(card)
	if err != nil {
		fmt.Printf("❌ Card validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Card validation passed!")
	fmt.Println("  - Type and version are valid")
	fmt.Println("  - DID format is correct")
	fmt.Println("  - Required fields are present")
	fmt.Println("  - Public keys are well-formed")
	fmt.Println()

	// Step 5: Export to JSON
	fmt.Println("💾 Step 5: Exporting Card to JSON")
	fmt.Println("─────────────────────────────────────────────────────────")

	cardJSON, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		fmt.Printf("❌ Failed to marshal card to JSON: %v\n", err)
		os.Exit(1)
	}

	filename := "agent-card.json"
	err = os.WriteFile(filename, cardJSON, 0644)
	if err != nil {
		fmt.Printf("❌ Failed to write card file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Card exported to: %s\n", filename)
	fmt.Printf("  File size: %d bytes\n", len(cardJSON))
	fmt.Println()

	// Display JSON preview
	fmt.Println("JSON Preview (first 500 characters):")
	fmt.Println("─────────────────────────────────────────────────────────")
	preview := string(cardJSON)
	if len(preview) > 500 {
		preview = preview[:500] + "\n  ..."
	}
	fmt.Println(preview)
	fmt.Println()

	// Summary
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║     Card Generation Complete!                             ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("🎉 Success! Your A2A Agent Card is ready.")
	fmt.Println()
	fmt.Println("What you can do with this card:")
	fmt.Println("  1. Share it with other agents for discovery")
	fmt.Println("  2. Import it into A2A-compatible applications")
	fmt.Println("  3. Use it for agent-to-agent trust establishment")
	fmt.Println("  4. Exchange it over HTTP/HTTPS with other agents")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Run example 03 to exchange cards with another agent")
	fmt.Println("  2. Run example 04 for secure messaging")
	fmt.Println()
	fmt.Printf("Card file: %s\n", filename)
	fmt.Println()
}
