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
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// Example 03: A2A Card Exchange and Verification
//
// This example demonstrates the complete card exchange workflow:
// 1. Agent A generates and exports a card
// 2. Agent B receives and validates the card
// 3. Agent B verifies the card against blockchain data
// 4. Both agents establish mutual trust
//
// This is the foundation for secure agent-to-agent communication.

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║     SAGE Example 03: A2A Card Exchange                   ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Get configuration
	registryAddress := os.Getenv("REGISTRY_ADDRESS")
	rpcURL := os.Getenv("RPC_URL")
	privateKey := os.Getenv("PRIVATE_KEY")

	if registryAddress == "" {
		registryAddress = "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
	}
	if rpcURL == "" {
		rpcURL = "http://localhost:8545"
	}
	if privateKey == "" {
		privateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	}

	fmt.Println(" Configuration")
	fmt.Println("─────────────────────────────────────────────────────────")
	fmt.Println("Registry Address:", registryAddress)
	fmt.Println("RPC URL:         ", rpcURL)
	fmt.Println()

	// Setup DID manager
	manager := did.NewManager()
	config := &did.RegistryConfig{
		ContractAddress:    registryAddress,
		RPCEndpoint:        rpcURL,
		PrivateKey:         privateKey,
		GasPrice:           0,
		MaxRetries:         10,
		ConfirmationBlocks: 0,
	}

	err := manager.Configure(did.ChainEthereum, config)
	if err != nil {
		fmt.Printf(" Failed to configure manager: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// ========================================================================
	// SCENARIO: Agent A and Agent B exchange cards
	// ========================================================================

	fmt.Println(" AGENT A: Registering and Generating Card")
	fmt.Println("═════════════════════════════════════════════════════════")
	fmt.Println()

	// Register Agent A
	agentA, cardA := registerAndGenerateCard(manager, ctx, "Agent-A")
	fmt.Printf(" Agent A registered: %s\n", agentA.DID)
	fmt.Println()

	// Save Agent A's card to file
	cardAJSON, _ := json.MarshalIndent(cardA, "", "  ")
	os.WriteFile("agent-a-card.json", cardAJSON, 0644)
	fmt.Println(" Agent A's card saved to: agent-a-card.json")
	fmt.Println()

	fmt.Println(" AGENT B: Registering and Generating Card")
	fmt.Println("═════════════════════════════════════════════════════════")
	fmt.Println()

	// Register Agent B
	agentB, cardB := registerAndGenerateCard(manager, ctx, "Agent-B")
	fmt.Printf(" Agent B registered: %s\n", agentB.DID)
	fmt.Println()

	// Save Agent B's card to file
	cardBJSON, _ := json.MarshalIndent(cardB, "", "  ")
	os.WriteFile("agent-b-card.json", cardBJSON, 0644)
	fmt.Println(" Agent B's card saved to: agent-b-card.json")
	fmt.Println()

	// ========================================================================
	// STEP 1: Agent B receives Agent A's card
	// ========================================================================

	fmt.Println(" Step 1: Agent B Receives Agent A's Card")
	fmt.Println("═════════════════════════════════════════════════════════")
	fmt.Println()

	// In a real scenario, this would be received via HTTP/HTTPS
	// For this example, we load it from the file
	receivedCardData, err := os.ReadFile("agent-a-card.json")
	if err != nil {
		fmt.Printf(" Failed to read Agent A's card: %v\n", err)
		os.Exit(1)
	}

	var receivedCard did.A2AAgentCard
	err = json.Unmarshal(receivedCardData, &receivedCard)
	if err != nil {
		fmt.Printf(" Failed to parse Agent A's card: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(" Agent B received Agent A's card")
	fmt.Println("  From:        ", receivedCard.Name)
	fmt.Println("  DID:         ", receivedCard.ID)
	fmt.Println("  Endpoint:    ", receivedCard.ServiceEndpoint)
	fmt.Printf("  Public Keys:  %d\n", len(receivedCard.PublicKeys))
	fmt.Println()

	// ========================================================================
	// STEP 2: Validate card structure
	// ========================================================================

	fmt.Println(" Step 2: Validating Card Structure")
	fmt.Println("═════════════════════════════════════════════════════════")
	fmt.Println()

	err = did.ValidateA2ACard(&receivedCard)
	if err != nil {
		fmt.Printf(" Card validation failed: %v\n", err)
		fmt.Println("   This card may be malformed or tampered with.")
		os.Exit(1)
	}

	fmt.Println(" Card structure is valid")
	fmt.Println("  - Correct type and version")
	fmt.Println("  - Valid DID format")
	fmt.Println("  - All required fields present")
	fmt.Println("  - Public keys are well-formed")
	fmt.Println()

	// ========================================================================
	// STEP 3: Verify card against blockchain
	// ========================================================================

	fmt.Println(" Step 3: Verifying Card Against Blockchain")
	fmt.Println("═════════════════════════════════════════════════════════")
	fmt.Println()

	// Resolve the agent from blockchain
	resolvedAgent, err := manager.ResolveAgent(ctx, did.AgentDID(receivedCard.ID))
	if err != nil {
		fmt.Printf(" Failed to resolve agent from blockchain: %v\n", err)
		fmt.Println("   The DID may not exist or the blockchain is unreachable.")
		os.Exit(1)
	}

	fmt.Println(" Agent resolved from blockchain")
	fmt.Println("  Name:     ", resolvedAgent.Name)
	fmt.Println("  Endpoint: ", resolvedAgent.Endpoint)
	fmt.Println("  Owner:    ", resolvedAgent.Owner)
	fmt.Println("  Active:   ", resolvedAgent.IsActive)
	fmt.Println()

	// Cross-check card data with blockchain data
	fmt.Println(" Cross-checking card with blockchain data...")

	verified := true

	// Check name
	if receivedCard.Name != resolvedAgent.Name {
		fmt.Println(" Name mismatch!")
		fmt.Printf("   Card: %s\n", receivedCard.Name)
		fmt.Printf("   Chain: %s\n", resolvedAgent.Name)
		verified = false
	} else {
		fmt.Println(" Name matches")
	}

	// Check endpoint
	if receivedCard.ServiceEndpoint != resolvedAgent.Endpoint {
		fmt.Println(" Endpoint mismatch!")
		fmt.Printf("   Card: %s\n", receivedCard.ServiceEndpoint)
		fmt.Printf("   Chain: %s\n", resolvedAgent.Endpoint)
		verified = false
	} else {
		fmt.Println(" Endpoint matches")
	}

	// Check if agent is active
	if !resolvedAgent.IsActive {
		fmt.Println("  Agent is deactivated on-chain")
		verified = false
	} else {
		fmt.Println(" Agent is active")
	}

	fmt.Println()

	if !verified {
		fmt.Println(" Card verification failed!")
		fmt.Println("   Do not trust this card.")
		os.Exit(1)
	}

	fmt.Println(" Card verification successful!")
	fmt.Println("   The card matches blockchain data.")
	fmt.Println()

	// ========================================================================
	// STEP 4: Establish trust
	// ========================================================================

	fmt.Println(" Step 4: Establishing Trust")
	fmt.Println("═════════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Println("Agent B now trusts Agent A because:")
	fmt.Println("  1.  Card structure is valid (proper format)")
	fmt.Println("  2.  DID is registered on blockchain")
	fmt.Println("  3.  Card data matches blockchain data")
	fmt.Println("  4.  Agent is active and not revoked")
	fmt.Println()

	fmt.Println("Agent B can now:")
	fmt.Println("  • Send encrypted messages using A's X25519 key")
	fmt.Println("  • Verify signatures using A's Ed25519 key")
	fmt.Println("  • Establish secure channels with A")
	fmt.Println("  • Collaborate on tasks with A")
	fmt.Println()

	// ========================================================================
	// Summary
	// ========================================================================

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║     Card Exchange Complete!                               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println(" Success! Agents A and B have exchanged and verified cards.")
	fmt.Println()
	fmt.Println("Files created:")
	fmt.Println("  • agent-a-card.json - Agent A's identity card")
	fmt.Println("  • agent-b-card.json - Agent B's identity card")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Run example 04 for secure messaging between A and B")
	fmt.Println("  2. Implement card exchange over HTTP/HTTPS")
	fmt.Println("  3. Build agent discovery services")
	fmt.Println()
}

// registerAndGenerateCard is a helper function that registers an agent and generates its card
func registerAndGenerateCard(manager *did.Manager, ctx context.Context, name string) (*did.AgentMetadata, *did.A2AAgentCard) {
	// Generate keys
	ecdsaKey, _ := crypto.GenerateSecp256k1KeyPair()
	ed25519Key, _ := crypto.GenerateEd25519KeyPair()
	x25519Key, _ := crypto.GenerateX25519KeyPair()

	ed25519Pub, _ := did.MarshalPublicKey(ed25519Key.PublicKey())
	x25519Pub, _ := did.MarshalPublicKey(x25519Key.PublicKey())

	// Create DID
	agentDID := did.GenerateDID(did.ChainEthereum, name+"-"+time.Now().Format("150405"))

	// Register
	req := &did.RegistrationRequest{
		DID:         agentDID,
		Name:        name,
		Description: "Example agent for card exchange demonstration",
		Endpoint:    "https://" + name + ".example.com",
		Capabilities: map[string]interface{}{
			"messaging": true,
		},
		KeyPair: ecdsaKey,
		Keys: []did.AgentKey{
			{Type: did.KeyTypeEd25519, KeyData: ed25519Pub},
			{Type: did.KeyTypeX25519, KeyData: x25519Pub},
		},
	}

	regCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := manager.RegisterAgent(regCtx, did.ChainEthereum, req)
	if err != nil {
		fmt.Printf(" Failed to register %s: %v\n", name, err)
		os.Exit(1)
	}

	// Resolve and generate card
	agent, err := manager.ResolveAgent(ctx, agentDID)
	if err != nil {
		fmt.Printf(" Failed to resolve %s: %v\n", name, err)
		os.Exit(1)
	}

	metadataV4 := did.FromAgentMetadata(agent)
	card, err := did.GenerateA2ACard(metadataV4)
	if err != nil {
		fmt.Printf(" Failed to generate card for %s: %v\n", name, err)
		os.Exit(1)
	}

	return agent, card
}
