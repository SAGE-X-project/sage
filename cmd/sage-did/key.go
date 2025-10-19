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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/spf13/cobra"
)

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage agent cryptographic keys",
	Long: `Manage cryptographic keys for registered agents.

SUBCOMMANDS:
  add      Add a new key to an existing agent
  list     List all keys for an agent
  revoke   Revoke a key from an agent
  approve  Approve an Ed25519 key (contract owner only)

EXAMPLES:
  # List all keys for an agent
  sage-did key list did:sage:eth:agent_12345678

  # Add a new Ed25519 key
  sage-did key add did:sage:eth:agent_12345678 keys/new_ed25519.jwk \
    --chain ethereum \
    --private-key <owner-private-key>

  # Revoke a key
  sage-did key revoke did:sage:eth:agent_12345678 0x1234... \
    --chain ethereum \
    --private-key <owner-private-key>

  # Approve an Ed25519 key (owner only)
  sage-did key approve 0x1234... \
    --chain ethereum \
    --private-key <registry-owner-key>`,
}

var (
	// Key command flags
	keyChain         string
	keyRPCEndpoint   string
	keyContractAddr  string
	keyPrivateKey    string
	keyType          string
	keyOutputFormat  string
)

// Key add command
var keyAddCmd = &cobra.Command{
	Use:   "add <did> <keyfile>",
	Short: "Add a new key to an existing agent",
	Long: `Add a new cryptographic key to an existing agent's DID document.

The key file can be in JWK, PEM, or raw format. The key type will be auto-detected
unless explicitly specified with --key-type flag.

REQUIREMENTS:
  - You must be the agent owner (provide --private-key)
  - ECDSA keys are verified on-chain immediately
  - Ed25519 keys require contract owner approval after addition

EXAMPLES:
  # Add Ed25519 key (auto-detect)
  sage-did key add did:sage:eth:agent_12345678 keys/ed25519.jwk \
    --chain ethereum \
    --private-key <your-private-key>

  # Add ECDSA key (explicit type)
  sage-did key add did:sage:eth:agent_12345678 keys/ecdsa.pem \
    --chain ethereum \
    --key-type ecdsa \
    --private-key <your-private-key>`,
	Args: cobra.ExactArgs(2),
	RunE: runKeyAdd,
}

// Key list command
var keyListCmd = &cobra.Command{
	Use:   "list <did>",
	Short: "List all keys for an agent",
	Long: `List all cryptographic keys registered for an agent.

Displays key hash, type, verification status, and registration timestamp.

EXAMPLES:
  # List keys in table format (default)
  sage-did key list did:sage:eth:agent_12345678 --chain ethereum

  # List keys in JSON format
  sage-did key list did:sage:eth:agent_12345678 \
    --chain ethereum \
    --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runKeyList,
}

// Key revoke command
var keyRevokeCmd = &cobra.Command{
	Use:   "revoke <did> <keyhash>",
	Short: "Revoke a key from an agent",
	Long: `Revoke a cryptographic key from an agent's DID document.

Only the agent owner can revoke keys. This operation is irreversible.

WARNING: Revoking keys may break existing integrations using those keys.

EXAMPLES:
  # Revoke a key
  sage-did key revoke did:sage:eth:agent_12345678 0x1234567890abcdef... \
    --chain ethereum \
    --private-key <your-private-key>`,
	Args: cobra.ExactArgs(2),
	RunE: runKeyRevoke,
}

// Key approve command
var keyApproveCmd = &cobra.Command{
	Use:   "approve <keyhash>",
	Short: "Approve an Ed25519 key (contract owner only)",
	Long: `Approve an Ed25519 key for verification.

This command is restricted to the contract registry owner. Ed25519 keys
on Ethereum require off-chain approval due to the lack of native on-chain
verification support.

REQUIREMENTS:
  - You must be the registry contract owner
  - The key must be Ed25519 type
  - The key must already be registered but not yet verified

EXAMPLES:
  # Approve an Ed25519 key
  sage-did key approve 0x1234567890abcdef... \
    --chain ethereum \
    --private-key <registry-owner-key>`,
	Args: cobra.ExactArgs(1),
	RunE: runKeyApprove,
}

func init() {
	rootCmd.AddCommand(keyCmd)
	keyCmd.AddCommand(keyAddCmd)
	keyCmd.AddCommand(keyListCmd)
	keyCmd.AddCommand(keyRevokeCmd)
	keyCmd.AddCommand(keyApproveCmd)

	// Common flags for all key commands
	keyCmd.PersistentFlags().StringVarP(&keyChain, "chain", "c", "", "Blockchain network (ethereum, solana)")
	keyCmd.PersistentFlags().StringVar(&keyRPCEndpoint, "rpc", "", "Blockchain RPC endpoint")
	keyCmd.PersistentFlags().StringVar(&keyContractAddr, "contract", "", "DID registry contract address")

	// Add command specific flags
	keyAddCmd.Flags().StringVar(&keyPrivateKey, "private-key", "", "Agent owner private key")
	keyAddCmd.Flags().StringVar(&keyType, "key-type", "", "Key type (ed25519, ecdsa, x25519) - auto-detected if omitted")
	if err := keyAddCmd.MarkFlagRequired("private-key"); err != nil {
		panic(fmt.Sprintf("failed to mark flag required: %v", err))
	}

	// List command specific flags
	keyListCmd.Flags().StringVar(&keyOutputFormat, "format", "table", "Output format (table, json)")

	// Revoke command specific flags
	keyRevokeCmd.Flags().StringVar(&keyPrivateKey, "private-key", "", "Agent owner private key")
	if err := keyRevokeCmd.MarkFlagRequired("private-key"); err != nil {
		panic(fmt.Sprintf("failed to mark flag required: %v", err))
	}

	// Approve command specific flags
	keyApproveCmd.Flags().StringVar(&keyPrivateKey, "private-key", "", "Registry owner private key")
	if err := keyApproveCmd.MarkFlagRequired("private-key"); err != nil {
		panic(fmt.Sprintf("failed to mark flag required: %v", err))
	}
}

func runKeyAdd(cmd *cobra.Command, args []string) error {
	agentDID := args[0]
	keyFile := args[1]

	// Parse chain
	chain, err := parseChain(keyChain)
	if err != nil {
		return err
	}

	// Read key file
	// #nosec G304 - User-specified file path is intentional for CLI tool
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	// Determine key type (explicit or auto-detect)
	var detectedKeyType did.KeyType
	if keyType != "" {
		// Explicit key type
		switch strings.ToLower(keyType) {
		case "ed25519":
			detectedKeyType = did.KeyTypeEd25519
		case "ecdsa", "secp256k1":
			detectedKeyType = did.KeyTypeECDSA
		case "x25519":
			detectedKeyType = did.KeyTypeX25519
		default:
			return fmt.Errorf("unsupported key type: %s (supported: ed25519, ecdsa, x25519)", keyType)
		}
	} else {
		// Auto-detect
		detectedKeyType, err = detectKeyType(keyFile, keyData)
		if err != nil {
			return fmt.Errorf("failed to detect key type: %w", err)
		}
		fmt.Printf("Auto-detected key type: %s\n", detectedKeyType)
	}

	// Parse key file
	parsedKeyData, err := parseKeyFile(keyData, detectedKeyType)
	if err != nil {
		return fmt.Errorf("failed to parse key file: %w", err)
	}

	// Get config
	config := &did.RegistryConfig{
		RPCEndpoint:     keyRPCEndpoint,
		ContractAddress: keyContractAddr,
		PrivateKey:      keyPrivateKey,
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

	// Add key
	fmt.Printf("Adding %s key to %s...\n", detectedKeyType, agentDID)
	ctx := context.Background()

	keyHash, err := manager.AddKey(ctx, chain, did.AgentDID(agentDID), did.AgentKey{
		Type:    detectedKeyType,
		KeyData: parsedKeyData,
	})
	if err != nil {
		return fmt.Errorf("failed to add key: %w", err)
	}

	fmt.Println("\n✓ Key added successfully!")
	fmt.Printf("Key Hash: %s\n", keyHash)

	// Show approval notice for Ed25519
	if detectedKeyType == did.KeyTypeEd25519 && chain == did.ChainEthereum {
		fmt.Println("\n⚠ IMPORTANT: Ed25519 Key Approval Required")
		fmt.Println("This Ed25519 key requires off-chain approval by the contract owner.")
		fmt.Println("Please contact the registry owner to approve this key:")
		fmt.Printf("  sage-did key approve %s --chain ethereum --private-key <owner-key>\n", keyHash)
	}

	return nil
}

func runKeyList(cmd *cobra.Command, args []string) error {
	agentDID := args[0]

	// Parse chain
	chain, err := parseChain(keyChain)
	if err != nil {
		return err
	}

	// Get config
	config := &did.RegistryConfig{
		RPCEndpoint:     keyRPCEndpoint,
		ContractAddress: keyContractAddr,
		PrivateKey:      keyPrivateKey,
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

	// Get agent metadata
	ctx := context.Background()
	metadata, err := manager.ResolveAgent(ctx, did.AgentDID(agentDID))
	if err != nil {
		return fmt.Errorf("failed to resolve agent: %w", err)
	}

	// Convert to V4 (ResolveAgent returns *AgentMetadata)
	metadataV4 := did.FromAgentMetadata(metadata)

	if len(metadataV4.Keys) == 0 {
		fmt.Println("No keys found for this agent")
		return nil
	}

	// Display keys
	if keyOutputFormat == "json" {
		// JSON format
		return displayKeysJSON(metadataV4.Keys)
	}

	// Table format (default)
	return displayKeysTable(metadataV4.Keys)
}

func runKeyRevoke(cmd *cobra.Command, args []string) error {
	agentDID := args[0]
	keyHash := args[1]

	// Parse chain
	chain, err := parseChain(keyChain)
	if err != nil {
		return err
	}

	// Confirm deletion
	fmt.Printf("⚠ WARNING: You are about to revoke key %s from agent %s\n", keyHash, agentDID)
	fmt.Println("This operation is irreversible and may break existing integrations.")
	fmt.Print("Are you sure you want to continue? (yes/no): ")

	var confirmation string
	fmt.Scanln(&confirmation)
	if strings.ToLower(confirmation) != "yes" {
		fmt.Println("Operation cancelled")
		return nil
	}

	// Get config
	config := &did.RegistryConfig{
		RPCEndpoint:     keyRPCEndpoint,
		ContractAddress: keyContractAddr,
		PrivateKey:      keyPrivateKey,
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

	// Revoke key
	fmt.Printf("Revoking key %s...\n", keyHash)
	ctx := context.Background()

	if err := manager.RevokeKey(ctx, chain, did.AgentDID(agentDID), keyHash); err != nil {
		return fmt.Errorf("failed to revoke key: %w", err)
	}

	fmt.Println("\n✓ Key revoked successfully!")
	return nil
}

func runKeyApprove(cmd *cobra.Command, args []string) error {
	keyHash := args[0]

	// Parse chain
	chain, err := parseChain(keyChain)
	if err != nil {
		return err
	}

	// Get config
	config := &did.RegistryConfig{
		RPCEndpoint:     keyRPCEndpoint,
		ContractAddress: keyContractAddr,
		PrivateKey:      keyPrivateKey,
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

	// Approve key
	fmt.Printf("Approving Ed25519 key %s...\n", keyHash)
	ctx := context.Background()

	if err := manager.ApproveEd25519Key(ctx, chain, keyHash); err != nil {
		return fmt.Errorf("failed to approve key: %w", err)
	}

	fmt.Println("\n✓ Key approved successfully!")
	fmt.Println("The Ed25519 key is now verified and fully functional.")
	return nil
}

func displayKeysTable(keys []did.AgentKey) error {
	fmt.Println("\nAgent Keys:")
	fmt.Println(strings.Repeat("=", 100))
	fmt.Printf("%-20s %-12s %-12s %-25s\n", "Key Hash", "Type", "Verified", "Created")
	fmt.Println(strings.Repeat("-", 100))

	for _, key := range keys {
		// Calculate key hash (same as contract)
		keyHash := hex.EncodeToString(key.KeyData[:min(8, len(key.KeyData))])
		verified := "No"
		if key.Verified {
			verified = "Yes"
		}
		createdAt := key.CreatedAt.Format(time.RFC3339)
		if key.CreatedAt.IsZero() {
			createdAt = "N/A"
		}

		fmt.Printf("0x%-18s %-12s %-12s %-25s\n",
			keyHash,
			key.Type.String(),
			verified,
			createdAt,
		)
	}

	fmt.Println(strings.Repeat("=", 100))
	fmt.Printf("\nTotal: %d keys\n", len(keys))
	return nil
}

func displayKeysJSON(keys []did.AgentKey) error {
	// Simple JSON output
	fmt.Println("[")
	for i, key := range keys {
		keyHash := hex.EncodeToString(key.KeyData[:min(8, len(key.KeyData))])
		fmt.Printf("  {\n")
		fmt.Printf("    \"keyHash\": \"0x%s\",\n", keyHash)
		fmt.Printf("    \"type\": \"%s\",\n", key.Type.String())
		fmt.Printf("    \"verified\": %t,\n", key.Verified)
		fmt.Printf("    \"createdAt\": \"%s\"\n", key.CreatedAt.Format(time.RFC3339))
		if i < len(keys)-1 {
			fmt.Printf("  },\n")
		} else {
			fmt.Printf("  }\n")
		}
	}
	fmt.Println("]")
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
