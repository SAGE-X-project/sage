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

Example:
  sage-did card validate agent-card.json`,
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
	cardValidateVerbose bool

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
	filePath := args[0]

	// Read file
	// #nosec G304 - User-specified file path is intentional for CLI tool
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse JSON
	var card did.A2AAgentCard
	if err := json.Unmarshal(data, &card); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	// Validate card
	if err := did.ValidateA2ACard(&card); err != nil {
		fmt.Printf("✗ Validation failed: %v\n", err)
		return err
	}

	// Success
	fmt.Printf("✓ A2A Agent Card is valid\n")

	if cardValidateVerbose {
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

	return nil
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
