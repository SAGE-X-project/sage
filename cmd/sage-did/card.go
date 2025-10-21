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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mr-tron/base58"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/spf13/cobra"
)

var cardCmd = &cobra.Command{
	Use:   "card",
	Short: "Manage A2A Agent Cards",
	Long: `Manage Google A2A protocol Agent Cards for AI agent interoperability.

A2A Agent Cards are a standardized format for representing AI agent metadata
that enables interoperability between different AI agent platforms.

Available commands:
  generate - Generate an A2A Agent Card from a registered DID
  validate - Validate an A2A Agent Card file
  show     - Display an A2A Agent Card from a DID`,
}

var cardGenerateCmd = &cobra.Command{
	Use:   "generate [DID]",
	Short: "Generate an A2A Agent Card from a registered DID",
	Long: `Generate a Google A2A protocol Agent Card by resolving an agent DID
from the blockchain and converting it to A2A format.

The generated Agent Card will include:
- Agent metadata (name, description, endpoint)
- Verified public keys in multiple formats (Base58, Hex)
- Service endpoints
- Agent capabilities

Example:
  sage-did card generate did:sage:ethereum:agent_12345678 -o agent-card.json`,
	Args: cobra.ExactArgs(1),
	RunE: runCardGenerate,
}

var cardValidateCmd = &cobra.Command{
	Use:   "validate [FILE]",
	Short: "Validate an A2A Agent Card file",
	Long: `Validate the structure and content of a Google A2A Agent Card JSON file.

This command checks:
- Required fields are present (ID, name, publicKey, service)
- Public keys have valid format
- Endpoints are properly configured
- JSON structure conforms to A2A specification

With --with-proof flag:
- Verifies cryptographic proof signature
- Ensures card was signed by legitimate DID owner

With --verify-did flag:
- Cross-validates card against on-chain DID document
- Verifies all keys exist on blockchain
- Checks endpoint consistency

Examples:
  # Basic validation
  sage-did card validate agent-card.json

  # Validate with cryptographic proof
  sage-did card validate agent-card.json --with-proof

  # Validate against blockchain
  sage-did card validate agent-card.json --verify-did --rpc <url>

  # Full validation (proof + blockchain)
  sage-did card validate agent-card.json --with-proof --verify-did --rpc <url>`,
	Args: cobra.ExactArgs(1),
	RunE: runCardValidate,
}

var cardShowCmd = &cobra.Command{
	Use:   "show [DID]",
	Short: "Display an A2A Agent Card from a DID",
	Long: `Display a formatted A2A Agent Card by resolving the agent DID
from the blockchain.

This command resolves the DID, converts it to A2A format, and displays
the card in a human-readable format.

Example:
  sage-did card show did:sage:ethereum:agent_12345678`,
	Args: cobra.ExactArgs(1),
	RunE: runCardShow,
}

var (
	// Card generate flags
	cardGenerateRPC      string
	cardGenerateContract string
	cardGenerateOutput   string

	// Card validate flags
	cardValidateVerbose   bool
	cardValidateWithProof bool
	cardValidateVerifyDID bool
	cardValidateRPC       string
	cardValidateContract  string

	// Card show flags
	cardShowRPC      string
	cardShowContract string
	cardShowFormat   string
)

func init() {
	rootCmd.AddCommand(cardCmd)
	cardCmd.AddCommand(cardGenerateCmd)
	cardCmd.AddCommand(cardValidateCmd)
	cardCmd.AddCommand(cardShowCmd)

	// Generate command flags
	cardGenerateCmd.Flags().StringVar(&cardGenerateRPC, "rpc", "", "Blockchain RPC endpoint")
	cardGenerateCmd.Flags().StringVar(&cardGenerateContract, "contract", "", "DID registry contract address")
	cardGenerateCmd.Flags().StringVarP(&cardGenerateOutput, "output", "o", "", "Output file path (defaults to stdout)")

	// Validate command flags
	cardValidateCmd.Flags().BoolVarP(&cardValidateVerbose, "verbose", "v", false, "Show detailed validation information")
	cardValidateCmd.Flags().BoolVar(&cardValidateWithProof, "with-proof", false, "Verify cryptographic proof signature")
	cardValidateCmd.Flags().BoolVar(&cardValidateVerifyDID, "verify-did", false, "Cross-validate against on-chain DID document")
	cardValidateCmd.Flags().StringVar(&cardValidateRPC, "rpc", "", "Blockchain RPC endpoint (required with --verify-did)")
	cardValidateCmd.Flags().StringVar(&cardValidateContract, "contract", "", "DID registry contract address")

	// Show command flags
	cardShowCmd.Flags().StringVar(&cardShowRPC, "rpc", "", "Blockchain RPC endpoint")
	cardShowCmd.Flags().StringVar(&cardShowContract, "contract", "", "DID registry contract address")
	cardShowCmd.Flags().StringVar(&cardShowFormat, "format", "text", "Output format (text, json)")
}

func runCardGenerate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	agentDID := did.AgentDID(args[0])

	// Parse DID to get chain
	chain, _, err := did.ParseDID(agentDID)
	if err != nil {
		return fmt.Errorf("invalid DID: %w", err)
	}

	// Get config
	config := &did.RegistryConfig{
		RPCEndpoint:     cardGenerateRPC,
		ContractAddress: cardGenerateContract,
	}

	if config.RPCEndpoint == "" {
		config.RPCEndpoint = getDefaultRPCEndpoint(chain)
	}
	if config.ContractAddress == "" {
		config.ContractAddress = getDefaultContractAddress(chain)
	}

	// Create DID manager
	manager := did.NewManager()
	if err := manager.Configure(chain, config); err != nil {
		return fmt.Errorf("failed to configure DID manager: %w", err)
	}

	// Resolve agent
	fmt.Printf("Resolving %s...\n", agentDID)
	metadata, err := manager.ResolveAgent(ctx, agentDID)
	if err != nil {
		return fmt.Errorf("failed to resolve DID: %w", err)
	}

	// Convert to V4 metadata
	metadataV4 := did.FromAgentMetadata(metadata)

	// Generate A2A Agent Card
	card, err := did.GenerateA2ACard(metadataV4)
	if err != nil {
		return fmt.Errorf("failed to generate A2A card: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal card: %w", err)
	}

	// Write output
	if cardGenerateOutput != "" {
		if err := os.WriteFile(cardGenerateOutput, data, 0600); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Printf("✓ A2A Agent Card saved to %s\n", cardGenerateOutput)
	} else {
		fmt.Println("\n" + string(data))
	}

	// Display summary
	fmt.Printf("\n✓ Generated A2A Agent Card\n")
	fmt.Printf("  Agent: %s\n", card.Name)
	fmt.Printf("  DID: %s\n", card.ID)
	fmt.Printf("  Public Keys: %d\n", len(card.PublicKeys))
	fmt.Printf("  Endpoints: %d\n", len(card.Endpoints))
	fmt.Printf("  Capabilities: %d\n", len(card.Capabilities))

	return nil
}

func runCardValidate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	filePath := args[0]

	// Read file
	// #nosec G304 - User-specified file path is intentional for CLI tool
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	fmt.Printf("Validating A2A Agent Card: %s\n", filePath)

	// Level 1: Basic validation
	if cardValidateWithProof {
		// Parse as card with proof
		var cardWithProof did.A2AAgentCardWithProof
		if err := json.Unmarshal(data, &cardWithProof); err != nil {
			return fmt.Errorf("invalid JSON format: %w", err)
		}

		// Validate with proof
		fmt.Printf("  [1/3] Basic structure validation... ")
		if err := did.ValidateA2ACard(&cardWithProof.A2AAgentCard); err != nil {
			fmt.Printf("✗ FAILED\n")
			return fmt.Errorf("basic validation failed: %w", err)
		}
		fmt.Printf("✓ PASSED\n")

		// Level 2: Cryptographic proof validation
		fmt.Printf("  [2/3] Cryptographic proof verification... ")
		if err := did.ValidateA2ACardWithProof(&cardWithProof); err != nil {
			fmt.Printf("✗ FAILED\n")
			return fmt.Errorf("proof verification failed: %w", err)
		}
		fmt.Printf("✓ PASSED\n")

		// Level 3: DID cross-validation (optional)
		if cardValidateVerifyDID {
			fmt.Printf("  [3/3] On-chain DID cross-validation... ")
			if err := validateCardWithDID(ctx, &cardWithProof.A2AAgentCard); err != nil {
				fmt.Printf("✗ FAILED\n")
				return fmt.Errorf("DID validation failed: %w", err)
			}
			fmt.Printf("✓ PASSED\n")
		} else {
			fmt.Printf("  [3/3] On-chain DID cross-validation... ⊘ SKIPPED\n")
		}

		// Success message
		fmt.Printf("\n✓ A2A Agent Card is valid")
		if cardWithProof.Proof != nil {
			fmt.Printf(" (with cryptographic proof)")
		}
		if cardValidateVerifyDID {
			fmt.Printf(" (verified against blockchain)")
		}
		fmt.Printf("\n")

		// Verbose output
		if cardValidateVerbose {
			displayCardDetails(&cardWithProof.A2AAgentCard)
			if cardWithProof.Proof != nil {
				fmt.Printf("\n=== Cryptographic Proof ===\n")
				fmt.Printf("Type: %s\n", cardWithProof.Proof.Type)
				fmt.Printf("Created: %s\n", cardWithProof.Proof.Created.Format("2006-01-02 15:04:05"))
				fmt.Printf("Verification Method: %s\n", cardWithProof.Proof.VerificationMethod)
				fmt.Printf("Purpose: %s\n", cardWithProof.Proof.ProofPurpose)
				fmt.Printf("Signature: %s...\n", cardWithProof.Proof.ProofValue[:min(32, len(cardWithProof.Proof.ProofValue))])
			}
		}
	} else {
		// Basic validation without proof
		var card did.A2AAgentCard
		if err := json.Unmarshal(data, &card); err != nil {
			return fmt.Errorf("invalid JSON format: %w", err)
		}

		fmt.Printf("  [1/2] Basic structure validation... ")
		if err := did.ValidateA2ACard(&card); err != nil {
			fmt.Printf("✗ FAILED\n")
			return fmt.Errorf("validation failed: %w", err)
		}
		fmt.Printf("✓ PASSED\n")

		// DID cross-validation (optional)
		if cardValidateVerifyDID {
			fmt.Printf("  [2/2] On-chain DID cross-validation... ")
			if err := validateCardWithDID(ctx, &card); err != nil {
				fmt.Printf("✗ FAILED\n")
				return fmt.Errorf("DID validation failed: %w", err)
			}
			fmt.Printf("✓ PASSED\n")
		} else {
			fmt.Printf("  [2/2] On-chain DID cross-validation... ⊘ SKIPPED\n")
		}

		fmt.Printf("\n✓ A2A Agent Card is valid")
		if cardValidateVerifyDID {
			fmt.Printf(" (verified against blockchain)")
		}
		fmt.Printf("\n")

		if cardValidateVerbose {
			displayCardDetails(&card)
		}
	}

	return nil
}

// validateCardWithDID validates a card against on-chain DID document
func validateCardWithDID(ctx context.Context, card *did.A2AAgentCard) error {
	// Validate --verify-did prerequisites
	if cardValidateRPC == "" {
		return fmt.Errorf("--rpc flag is required with --verify-did")
	}

	// Parse DID from card
	agentDID := did.AgentDID(card.ID)
	chain, _, err := did.ParseDID(agentDID)
	if err != nil {
		return fmt.Errorf("invalid DID in card: %w", err)
	}

	// Setup configuration
	config := &did.RegistryConfig{
		RPCEndpoint:     cardValidateRPC,
		ContractAddress: cardValidateContract,
	}

	if config.ContractAddress == "" {
		config.ContractAddress = getDefaultContractAddress(chain)
	}

	// Create DID manager
	manager := did.NewManager()
	if err := manager.Configure(chain, config); err != nil {
		return fmt.Errorf("failed to configure DID manager: %w", err)
	}

	// Resolve on-chain metadata directly using manager
	metadata, err := manager.ResolveAgent(ctx, agentDID)
	if err != nil {
		return fmt.Errorf("failed to resolve DID from blockchain: %w", err)
	}

	// Convert to V4 metadata for key comparison
	metadataV4 := did.FromAgentMetadata(metadata)

	// Verify all public keys in card exist on-chain
	for _, cardKey := range card.PublicKeys {
		// Decode card key data
		var cardKeyData []byte
		if cardKey.PublicKeyBase58 != "" {
			cardKeyData, err = base58.Decode(cardKey.PublicKeyBase58)
			if err != nil {
				return fmt.Errorf("invalid key data in card for key %s: %w", cardKey.ID, err)
			}
		} else if cardKey.PublicKeyHex != "" {
			cardKeyData, err = hex.DecodeString(cardKey.PublicKeyHex)
			if err != nil {
				return fmt.Errorf("invalid hex key data in card for key %s: %w", cardKey.ID, err)
			}
		} else {
			return fmt.Errorf("key %s has no public key data", cardKey.ID)
		}

		// Check if this key exists on-chain
		found := false
		for _, onChainKey := range metadataV4.Keys {
			// Compare key data
			if hex.EncodeToString(onChainKey.KeyData) == hex.EncodeToString(cardKeyData) {
				// Key found - check if verified
				if !onChainKey.Verified {
					return fmt.Errorf("key %s exists on-chain but is not verified", cardKey.ID)
				}
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("key %s not found in on-chain DID document", cardKey.ID)
		}
	}

	// Verify endpoint consistency
	if len(card.Endpoints) > 0 {
		primaryEndpoint := card.Endpoints[0].URI
		if primaryEndpoint != metadata.Endpoint {
			return fmt.Errorf("primary endpoint mismatch: card has %s, on-chain has %s",
				primaryEndpoint, metadata.Endpoint)
		}
	}

	// Verify agent is active
	if !metadata.IsActive {
		return fmt.Errorf("DID %s is not active on-chain", agentDID)
	}

	return nil
}

// displayCardDetails shows detailed card information
func displayCardDetails(card *did.A2AAgentCard) {
	fmt.Printf("\n=== Card Details ===\n")
	fmt.Printf("ID: %s\n", card.ID)
	fmt.Printf("Name: %s\n", card.Name)
	fmt.Printf("Description: %s\n", card.Description)
	fmt.Printf("Type: %v\n", card.Type)
	fmt.Printf("\nPublic Keys: %d\n", len(card.PublicKeys))
	for i, pk := range card.PublicKeys {
		fmt.Printf("  [%d] ID: %s\n", i+1, pk.ID)
		fmt.Printf("      Type: %s\n", pk.Type)
		fmt.Printf("      Controller: %s\n", pk.Controller)
	}
	fmt.Printf("\nEndpoints: %d\n", len(card.Endpoints))
	for i, ep := range card.Endpoints {
		fmt.Printf("  [%d] %s: %s\n", i+1, ep.Type, ep.URI)
	}
	fmt.Printf("\nCapabilities: %v\n", card.Capabilities)
	fmt.Printf("\nCreated: %s\n", card.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", card.Updated.Format("2006-01-02 15:04:05"))
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func runCardShow(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	agentDID := did.AgentDID(args[0])

	// Parse DID to get chain
	chain, _, err := did.ParseDID(agentDID)
	if err != nil {
		return fmt.Errorf("invalid DID: %w", err)
	}

	// Get config
	config := &did.RegistryConfig{
		RPCEndpoint:     cardShowRPC,
		ContractAddress: cardShowContract,
	}

	if config.RPCEndpoint == "" {
		config.RPCEndpoint = getDefaultRPCEndpoint(chain)
	}
	if config.ContractAddress == "" {
		config.ContractAddress = getDefaultContractAddress(chain)
	}

	// Create DID manager
	manager := did.NewManager()
	if err := manager.Configure(chain, config); err != nil {
		return fmt.Errorf("failed to configure DID manager: %w", err)
	}

	// Resolve agent
	metadata, err := manager.ResolveAgent(ctx, agentDID)
	if err != nil {
		return fmt.Errorf("failed to resolve DID: %w", err)
	}

	// Convert to V4 metadata
	metadataV4 := did.FromAgentMetadata(metadata)

	// Generate A2A Agent Card
	card, err := did.GenerateA2ACard(metadataV4)
	if err != nil {
		return fmt.Errorf("failed to generate A2A card: %w", err)
	}

	// Format output
	switch cardShowFormat {
	case "json":
		data, err := json.MarshalIndent(card, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal card: %w", err)
		}
		fmt.Println(string(data))
	case "text":
		fmt.Println(formatA2ACardText(card))
	default:
		return fmt.Errorf("unsupported format: %s", cardShowFormat)
	}

	return nil
}

func formatA2ACardText(card *did.A2AAgentCard) string {
	output := "=== A2A Agent Card ===\n\n"
	output += fmt.Sprintf("ID: %s\n", card.ID)
	output += fmt.Sprintf("Name: %s\n", card.Name)
	if card.Description != "" {
		output += fmt.Sprintf("Description: %s\n", card.Description)
	}
	output += fmt.Sprintf("Type: %v\n", card.Type)

	if len(card.PublicKeys) > 0 {
		output += fmt.Sprintf("\nPublic Keys (%d):\n", len(card.PublicKeys))
		for i, pk := range card.PublicKeys {
			output += fmt.Sprintf("  [%d] %s\n", i+1, pk.Type)
			output += fmt.Sprintf("      ID: %s\n", pk.ID)
			output += fmt.Sprintf("      Controller: %s\n", pk.Controller)
			if pk.PublicKeyBase58 != "" {
				output += fmt.Sprintf("      Base58: %s\n", pk.PublicKeyBase58)
			}
			if pk.PublicKeyHex != "" {
				output += fmt.Sprintf("      Hex: %s\n", pk.PublicKeyHex)
			}
		}
	}

	if len(card.Endpoints) > 0 {
		output += fmt.Sprintf("\nService Endpoints (%d):\n", len(card.Endpoints))
		for i, ep := range card.Endpoints {
			output += fmt.Sprintf("  [%d] %s: %s\n", i+1, ep.Type, ep.URI)
		}
	}

	if len(card.Capabilities) > 0 {
		output += fmt.Sprintf("\nCapabilities (%d):\n", len(card.Capabilities))
		for i, cap := range card.Capabilities {
			output += fmt.Sprintf("  - %s\n", cap)
			if i >= 9 && i < len(card.Capabilities)-1 {
				output += fmt.Sprintf("  ... and %d more\n", len(card.Capabilities)-10)
				break
			}
		}
	}

	output += fmt.Sprintf("\nCreated: %s\n", card.Created.Format("2006-01-02 15:04:05 MST"))
	output += fmt.Sprintf("Updated: %s\n", card.Updated.Format("2006-01-02 15:04:05 MST"))

	return output
}
