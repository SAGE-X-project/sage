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
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
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
This command creates a new agent identity on the specified blockchain network.

MULTI-KEY REGISTRATION:
  Register an agent with multiple cryptographic keys for cross-chain compatibility.
  Supported key types: Ed25519, ECDSA (secp256k1), X25519

  Example (auto-detect key types from files):
    sage-did register \
      --chain ethereum \
      --name "Multi-Key Agent" \
      --endpoint https://agent.example.com \
      --key keys/primary.pem \
      --additional-keys keys/ed25519.jwk,keys/x25519.key

  Example (explicit key types):
    sage-did register \
      --chain ethereum \
      --name "My Agent" \
      --endpoint https://agent.example.com \
      --key keys/ecdsa.pem \
      --additional-keys keys/ed25519.jwk,keys/x25519.key \
      --key-types ed25519,x25519

NOTE: Ed25519 keys on Ethereum require off-chain approval by contract owner.`,
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
	fmt.Println("\n✓ Agent registered successfully!")
	fmt.Printf("DID: %s\n", agentDID)
	fmt.Printf("Transaction: %s\n", result.TransactionHash)
	if result.BlockNumber > 0 {
		fmt.Printf("Block: %d\n", result.BlockNumber)
	}
	if result.Slot > 0 {
		fmt.Printf("Slot: %d\n", result.Slot)
	}
	fmt.Printf("Gas Used: %d\n", result.GasUsed)

	// Check if Ed25519 keys need approval
	hasEd25519 := false
	for _, key := range additionalKeys {
		if key.Type == did.KeyTypeEd25519 {
			hasEd25519 = true
			break
		}
	}

	if hasEd25519 && chain == did.ChainEthereum {
		fmt.Println("\n⚠ IMPORTANT: Ed25519 Key Approval Required")
		fmt.Println("Your agent includes Ed25519 keys which require off-chain approval on Ethereum.")
		fmt.Println("Please contact the contract owner to approve your Ed25519 keys.")
		fmt.Println()
		fmt.Println("Once approved, your Ed25519 keys will be verified and fully functional.")
		fmt.Println("ECDSA and X25519 keys are already active.")
	}

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
	// Default contract addresses for SageRegistryV4
	// These are placeholder addresses. For production deployments:
	// 1. Use --contract-address flag with the actual deployed address
	// 2. Refer to contracts/DEPLOYED_ADDRESSES.md for network-specific addresses
	// 3. For local testing, use the address from deployment output
	switch chain {
	case did.ChainEthereum:
		// Placeholder - Update after mainnet/testnet deployment
		// See: contracts/DEPLOYED_ADDRESSES.md
		return "0x0000000000000000000000000000000000000000"
	case did.ChainSolana:
		// Placeholder for future Solana support
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

	// Parse explicit key types if provided
	var explicitTypes []string
	if keyTypesStr != "" {
		explicitTypes = strings.Split(keyTypesStr, ",")
		if len(files) != len(explicitTypes) {
			return nil, fmt.Errorf("number of key files (%d) must match number of key types (%d)", len(files), len(explicitTypes))
		}
	}

	var keys []did.AgentKey
	for i, file := range files {
		file = strings.TrimSpace(file)

		// Read key file
		// #nosec G304 - User-specified file path is intentional for CLI tool
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file %s: %w", file, err)
		}

		// Determine key type (explicit or auto-detect)
		var keyType did.KeyType
		if explicitTypes != nil {
			keyTypeStr := strings.TrimSpace(explicitTypes[i])
			switch strings.ToLower(keyTypeStr) {
			case "ed25519":
				keyType = did.KeyTypeEd25519
			case "ecdsa", "secp256k1":
				keyType = did.KeyTypeECDSA
			case "x25519":
				keyType = did.KeyTypeX25519
			default:
				return nil, fmt.Errorf("unsupported key type: %s (supported: ed25519, ecdsa, x25519)", keyTypeStr)
			}
		} else {
			// Auto-detect key type from file
			detectedType, err := detectKeyType(file, data)
			if err != nil {
				return nil, fmt.Errorf("failed to detect key type for %s: %w", file, err)
			}
			keyType = detectedType
			fmt.Printf("Auto-detected key type for %s: %s\n", filepath.Base(file), keyType)
		}

		// Parse key file based on format
		keyData, err := parseKeyFile(data, keyType)
		if err != nil {
			return nil, fmt.Errorf("failed to parse key file %s: %w", file, err)
		}

		keys = append(keys, did.AgentKey{
			Type:    keyType,
			KeyData: keyData,
		})
	}

	return keys, nil
}

// detectKeyType auto-detects key type from file extension and content
func detectKeyType(filename string, data []byte) (did.KeyType, error) {
	// Try extension first
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".ed25519", ".ed":
		return did.KeyTypeEd25519, nil
	case ".x25519":
		return did.KeyTypeX25519, nil
	case ".pem", ".crt", ".key":
		// PEM files need content inspection
		return detectPEMKeyType(data)
	case ".jwk", ".json":
		return detectJWKKeyType(data)
	}

	// Try content-based detection
	// Check if it's JWK (JSON)
	if json.Valid(data) {
		return detectJWKKeyType(data)
	}

	// Check if it's PEM
	if block, _ := pem.Decode(data); block != nil {
		return detectPEMKeyType(data)
	}

	// Try raw key detection by length
	switch len(data) {
	case 32:
		// Could be Ed25519 or X25519
		return did.KeyTypeEd25519, nil
	case 33:
		// Compressed secp256k1
		return did.KeyTypeECDSA, nil
	case 65:
		// Uncompressed secp256k1
		return did.KeyTypeECDSA, nil
	}

	return 0, fmt.Errorf("unable to detect key type (try using --key-types flag)")
}

// detectPEMKeyType detects key type from PEM content
func detectPEMKeyType(data []byte) (did.KeyType, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return 0, fmt.Errorf("not a valid PEM file")
	}

	// Try parsing as PKIX public key
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		switch pubKey.(type) {
		case *ecdsa.PublicKey:
			return did.KeyTypeECDSA, nil
		case ed25519.PublicKey:
			return did.KeyTypeEd25519, nil
		}
	}

	// Check PEM block type
	switch block.Type {
	case "EC PUBLIC KEY", "ECDSA PUBLIC KEY":
		return did.KeyTypeECDSA, nil
	case "PUBLIC KEY":
		// Generic, try parsing
		if pubKey, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
			switch pubKey.(type) {
			case *ecdsa.PublicKey:
				return did.KeyTypeECDSA, nil
			case ed25519.PublicKey:
				return did.KeyTypeEd25519, nil
			}
		}
	}

	return 0, fmt.Errorf("unsupported PEM key type: %s", block.Type)
}

// detectJWKKeyType detects key type from JWK content
func detectJWKKeyType(data []byte) (did.KeyType, error) {
	var jwk map[string]interface{}
	if err := json.Unmarshal(data, &jwk); err != nil {
		return 0, fmt.Errorf("invalid JWK format: %w", err)
	}

	kty, ok := jwk["kty"].(string)
	if !ok {
		return 0, fmt.Errorf("JWK missing 'kty' field")
	}

	switch kty {
	case "OKP":
		crv, _ := jwk["crv"].(string)
		switch crv {
		case "Ed25519":
			return did.KeyTypeEd25519, nil
		case "X25519":
			return did.KeyTypeX25519, nil
		default:
			return 0, fmt.Errorf("unsupported OKP curve: %s", crv)
		}
	case "EC":
		crv, _ := jwk["crv"].(string)
		if crv == "secp256k1" || crv == "P-256K" {
			return did.KeyTypeECDSA, nil
		}
		return 0, fmt.Errorf("unsupported EC curve: %s (only secp256k1 supported)", crv)
	default:
		return 0, fmt.Errorf("unsupported JWK key type: %s", kty)
	}
}

// parseKeyFile parses key data from various formats
func parseKeyFile(data []byte, keyType did.KeyType) ([]byte, error) {
	// Try JWK format
	if json.Valid(data) {
		return parseJWKKey(data, keyType)
	}

	// Try PEM format
	if block, _ := pem.Decode(data); block != nil {
		return parsePEMKey(block.Bytes, keyType)
	}

	// Try raw public key bytes
	if isValidRawKey(data, keyType) {
		return data, nil
	}

	return nil, fmt.Errorf("unsupported key file format (supported: JWK, PEM, raw bytes)")
}

// parseJWKKey parses public key from JWK format
func parseJWKKey(data []byte, keyType did.KeyType) ([]byte, error) {
	var jwk map[string]interface{}
	if err := json.Unmarshal(data, &jwk); err != nil {
		return nil, fmt.Errorf("invalid JWK: %w", err)
	}

	switch keyType {
	case did.KeyTypeEd25519, did.KeyTypeX25519:
		// OKP keys use 'x' parameter (base64url-encoded)
		xStr, ok := jwk["x"].(string)
		if !ok {
			return nil, fmt.Errorf("JWK missing 'x' parameter")
		}
		return base64.RawURLEncoding.DecodeString(xStr)

	case did.KeyTypeECDSA:
		// EC keys use 'x' and 'y' parameters
		xStr, ok1 := jwk["x"].(string)
		yStr, ok2 := jwk["y"].(string)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("JWK missing 'x' or 'y' parameters")
		}

		xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
		if err != nil {
			return nil, fmt.Errorf("invalid 'x' parameter: %w", err)
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(yStr)
		if err != nil {
			return nil, fmt.Errorf("invalid 'y' parameter: %w", err)
		}

		// Construct uncompressed public key (0x04 || x || y)
		pubKey := make([]byte, 1+len(xBytes)+len(yBytes))
		pubKey[0] = 0x04
		copy(pubKey[1:], xBytes)
		copy(pubKey[1+len(xBytes):], yBytes)
		return pubKey, nil
	}

	return nil, fmt.Errorf("unsupported key type for JWK parsing")
}

// parsePEMKey parses public key from PEM-encoded bytes
func parsePEMKey(derBytes []byte, keyType did.KeyType) ([]byte, error) {
	pubKey, err := x509.ParsePKIXPublicKey(derBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	switch keyType {
	case did.KeyTypeECDSA:
		ecdsaKey, ok := pubKey.(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("expected ECDSA key, got %T", pubKey)
		}
		// Return uncompressed public key (0x04 || x || y)
		return ethcrypto.FromECDSAPub(ecdsaKey), nil

	case did.KeyTypeEd25519:
		ed25519Key, ok := pubKey.(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("expected Ed25519 key, got %T", pubKey)
		}
		return ed25519Key, nil

	case did.KeyTypeX25519:
		// X25519 is not a standard x509 key type, return raw bytes
		return derBytes, nil
	}

	return nil, fmt.Errorf("unsupported key type for PEM parsing")
}

// isValidRawKey checks if raw bytes are valid for the key type
func isValidRawKey(data []byte, keyType did.KeyType) bool {
	switch keyType {
	case did.KeyTypeEd25519, did.KeyTypeX25519:
		return len(data) == 32
	case did.KeyTypeECDSA:
		// Uncompressed (65 bytes) or compressed (33 bytes)
		return len(data) == 65 || len(data) == 33
	}
	return false
}
