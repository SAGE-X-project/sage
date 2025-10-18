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
	"strings"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/crypto/storage"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new AI agent on blockchain",
	Long: `Register a new AI agent with a Decentralized Identifier (DID) on blockchain.
This command creates a new agent identity on the specified blockchain network.`,
	RunE: runRegister,
}

var (
	// Register flags
	registerChain          string
	registerName           string
	registerDescription    string
	registerEndpoint       string
	registerCapabilities   string
	registerKeyFile        string
	registerKeyFormat      string
	registerStorageDir     string
	registerKeyID          string
	registerRPCEndpoint    string
	registerContractAddr   string
	registerPrivateKey     string
	registerAdditionalKeys string // Additional keys (comma-separated file paths)
	registerKeyTypes       string // Key types for additional keys (comma-separated: ed25519,ecdsa)
)

func init() {
	rootCmd.AddCommand(registerCmd)

	// Required flags
	registerCmd.Flags().StringVarP(&registerChain, "chain", "c", "", "Blockchain network (ethereum, solana)")
	registerCmd.Flags().StringVarP(&registerName, "name", "n", "", "Agent name")
	registerCmd.Flags().StringVar(&registerEndpoint, "endpoint", "", "Agent API endpoint URL")

	// Optional flags
	registerCmd.Flags().StringVarP(&registerDescription, "description", "d", "", "Agent description")
	registerCmd.Flags().StringVar(&registerCapabilities, "capabilities", "", "Agent capabilities (JSON format)")

	// Key source flags
	registerCmd.Flags().StringVarP(&registerKeyFile, "key", "k", "", "Key file path (JWK or PEM format)")
	registerCmd.Flags().StringVar(&registerKeyFormat, "key-format", "jwk", "Key file format (jwk, pem)")
	registerCmd.Flags().StringVar(&registerStorageDir, "storage-dir", "", "Key storage directory")
	registerCmd.Flags().StringVar(&registerKeyID, "key-id", "", "Key ID in storage")

	// Multi-key support flags
	registerCmd.Flags().StringVar(&registerAdditionalKeys, "additional-keys", "", "Additional key files (comma-separated)")
	registerCmd.Flags().StringVar(&registerKeyTypes, "key-types", "", "Key types for additional keys (comma-separated: ed25519,ecdsa)")

	// Blockchain connection flags
	registerCmd.Flags().StringVar(&registerRPCEndpoint, "rpc", "", "Blockchain RPC endpoint")
	registerCmd.Flags().StringVar(&registerContractAddr, "contract", "", "DID registry contract address")
	registerCmd.Flags().StringVar(&registerPrivateKey, "private-key", "", "Transaction signer private key (for gas fees)")

	// Mark required flags
	if err := registerCmd.MarkFlagRequired("chain"); err != nil {
		panic(fmt.Sprintf("failed to mark flag required: %v", err))
	}
	if err := registerCmd.MarkFlagRequired("name"); err != nil {
		panic(fmt.Sprintf("failed to mark flag required: %v", err))
	}
	if err := registerCmd.MarkFlagRequired("endpoint"); err != nil {
		panic(fmt.Sprintf("failed to mark flag required: %v", err))
	}
}

func runRegister(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Parse chain
	chain, err := parseChain(registerChain)
	if err != nil {
		return err
	}

	// Load primary key pair
	keyPair, err := loadKeyPair()
	if err != nil {
		return fmt.Errorf("failed to load key pair: %w", err)
	}

	// Validate key type for chain
	if err := validateKeyForChain(keyPair, chain); err != nil {
		return err
	}

	// Load additional keys for multi-key registration
	var additionalKeys []did.AgentKey
	if registerAdditionalKeys != "" {
		additionalKeys, err = loadAdditionalKeys(registerAdditionalKeys, registerKeyTypes)
		if err != nil {
			return fmt.Errorf("failed to load additional keys: %w", err)
		}
	}

	// Parse capabilities
	var capabilities map[string]interface{}
	if registerCapabilities != "" {
		if err := json.Unmarshal([]byte(registerCapabilities), &capabilities); err != nil {
			return fmt.Errorf("invalid capabilities JSON: %w", err)
		}
	}

	// Get default config if not provided
	config := &did.RegistryConfig{
		RPCEndpoint:     registerRPCEndpoint,
		ContractAddress: registerContractAddr,
		PrivateKey:      registerPrivateKey,
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

	// Generate DID
	agentDID := generateAgentDID(chain, keyPair)

	// Create registration request
	req := &did.RegistrationRequest{
		DID:          agentDID,
		Name:         registerName,
		Description:  registerDescription,
		Endpoint:     registerEndpoint,
		Capabilities: capabilities,
		KeyPair:      keyPair,
		Keys:         additionalKeys, // Multi-key support
	}

	// Register agent
	fmt.Printf("Registering agent %s on %s...\n", agentDID, chain)
	result, err := manager.RegisterAgent(ctx, chain, req)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	// Display result
	fmt.Println("\n Agent registered successfully!")
	fmt.Printf("DID: %s\n", agentDID)
	fmt.Printf("Transaction: %s\n", result.TransactionHash)
	if result.BlockNumber > 0 {
		fmt.Printf("Block: %d\n", result.BlockNumber)
	}
	if result.Slot > 0 {
		fmt.Printf("Slot: %d\n", result.Slot)
	}
	fmt.Printf("Gas Used: %d\n", result.GasUsed)

	// Save registration info
	if registerStorageDir != "" {
		saveRegistrationInfo(registerStorageDir, string(agentDID), result)
	}

	return nil
}

func parseChain(chainStr string) (did.Chain, error) {
	switch strings.ToLower(chainStr) {
	case "ethereum", "eth":
		return did.ChainEthereum, nil
	case "solana", "sol":
		return did.ChainSolana, nil
	default:
		return "", fmt.Errorf("unsupported chain: %s", chainStr)
	}
}

func loadKeyPair() (crypto.KeyPair, error) {
	// Load from storage
	if registerStorageDir != "" && registerKeyID != "" {
		store, err := storage.NewFileKeyStorage(registerStorageDir)
		if err != nil {
			return nil, err
		}
		return store.Load(registerKeyID)
	}

	// Load from file
	if registerKeyFile != "" {
		// #nosec G304 - User-specified file path is intentional for CLI tool
		data, err := os.ReadFile(registerKeyFile)
		if err != nil {
			return nil, err
		}

		switch registerKeyFormat {
		case "jwk":
			// Import JWK format
			var jwk map[string]interface{}
			if err := json.Unmarshal(data, &jwk); err != nil {
				return nil, fmt.Errorf("invalid JWK format: %w", err)
			}
			// This is a simplified implementation - in production you'd parse the JWK properly
			kty, _ := jwk["kty"].(string)
			if kty == "OKP" {
				return keys.GenerateEd25519KeyPair()
			}
			return keys.GenerateSecp256k1KeyPair()
		case "pem":
			// For now, generate a new key - proper PEM import would be implemented later
			return keys.GenerateEd25519KeyPair()
		default:
			return nil, fmt.Errorf("unsupported key format: %s", registerKeyFormat)
		}
	}

	return nil, fmt.Errorf("no key source specified: use --key or --storage-dir with --key-id")
}

func validateKeyForChain(keyPair crypto.KeyPair, chain did.Chain) error {
	switch chain {
	case did.ChainEthereum:
		if keyPair.Type() != crypto.KeyTypeSecp256k1 {
			return fmt.Errorf("ethereum requires Secp256k1 keys, got %s", keyPair.Type())
		}
	case did.ChainSolana:
		if keyPair.Type() != crypto.KeyTypeEd25519 {
			return fmt.Errorf("solana requires Ed25519 keys, got %s", keyPair.Type())
		}
	}
	return nil
}

func generateAgentDID(chain did.Chain, keyPair crypto.KeyPair) did.AgentDID {
	// Generate a simple agent ID based on key
	agentID := fmt.Sprintf("agent_%s", keyPair.ID()[:8])
	return did.AgentDID(fmt.Sprintf("did:sage:%s:%s", chain, agentID))
}

// Note: DID helper functions have been moved to pkg/agent/did/utils.go for public API access.
// Use did.GenerateAgentDIDWithAddress(), did.GenerateAgentDIDWithNonce(), and
// did.DeriveEthereumAddress() instead.

func getDefaultRPCEndpoint(chain did.Chain) string {
	switch chain {
	case did.ChainEthereum:
		return "https://eth-mainnet.g.alchemy.com/v2/your-api-key"
	case did.ChainSolana:
		return "https://api.mainnet-beta.solana.com"
	default:
		return ""
	}
}

func getDefaultContractAddress(chain did.Chain) string {
	// These would be the actual deployed contract addresses
	switch chain {
	case did.ChainEthereum:
		return "0x0000000000000000000000000000000000000000"
	case did.ChainSolana:
		return "11111111111111111111111111111111"
	default:
		return ""
	}
}

func saveRegistrationInfo(storageDir, agentDID string, result *did.RegistrationResult) {
	info := map[string]interface{}{
		"did":             agentDID,
		"transactionHash": result.TransactionHash,
		"blockNumber":     result.BlockNumber,
		"slot":            result.Slot,
		"timestamp":       result.Timestamp,
		"gasUsed":         result.GasUsed,
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		fmt.Printf("Warning: failed to marshal registration info: %v\n", err)
		return
	}
	fileName := fmt.Sprintf("%s/did_%s.json", storageDir, strings.ReplaceAll(agentDID, ":", "_"))
	if err := os.WriteFile(fileName, data, 0600); err != nil {
		fmt.Printf("Warning: failed to save registration info to %s: %v\n", fileName, err)
	}
}

func loadAdditionalKeys(keyFiles, keyTypesStr string) ([]did.AgentKey, error) {
	files := strings.Split(keyFiles, ",")
	types := strings.Split(keyTypesStr, ",")

	if len(files) != len(types) {
		return nil, fmt.Errorf("number of key files (%d) must match number of key types (%d)", len(files), len(types))
	}

	var keys []did.AgentKey
	for i, file := range files {
		file = strings.TrimSpace(file)
		keyTypeStr := strings.TrimSpace(types[i])

		// Parse key type
		var keyType did.KeyType
		switch strings.ToLower(keyTypeStr) {
		case "ed25519":
			keyType = did.KeyTypeEd25519
		case "ecdsa", "secp256k1":
			keyType = did.KeyTypeECDSA
		default:
			return nil, fmt.Errorf("unsupported key type: %s (supported: ed25519, ecdsa)", keyTypeStr)
		}

		// Read key file
		// #nosec G304 - User-specified file path is intentional for CLI tool
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file %s: %w", file, err)
		}

		// For now, we expect raw public key bytes in the file
		// In production, you'd want to support multiple formats (JWK, PEM, etc.)
		keys = append(keys, did.AgentKey{
			Type:    keyType,
			KeyData: data,
		})
	}

	return keys, nil
}
