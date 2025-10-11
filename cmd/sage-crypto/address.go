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
	"crypto/ecdsa"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/chain"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/storage"
	"github.com/spf13/cobra"
)

var (
	chainType string
	allChains bool
)

var addressCmd = &cobra.Command{
	Use:   "address",
	Short: "Generate blockchain addresses from keys",
	Long: `Generate blockchain addresses from cryptographic keys.

This command can generate addresses for various blockchains:
  - Ethereum: Requires secp256k1 keys
  - Solana: Requires Ed25519 keys

The key can be provided from a file or from storage.`,
}

var addressGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate blockchain addresses from a key",
	Example: `  # Generate Ethereum address from a key file
  sage-crypto address generate --key mykey.jwk --chain ethereum

  # Generate all compatible addresses from a stored key
  sage-crypto address generate --storage-dir ./keys --key-id mykey --all

  # Generate Solana address from PEM key
  sage-crypto address generate --key mykey.pem --format pem --chain solana`,
	RunE: runAddressGenerate,
}

var addressParseCmd = &cobra.Command{
	Use:   "parse [address]",
	Short: "Parse and validate a blockchain address",
	Args:  cobra.ExactArgs(1),
	Example: `  # Parse an Ethereum address
  sage-crypto address parse 0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80

  # Parse a Solana address
  sage-crypto address parse 9WzDXwBbmkg8ZTbNMqUxvQRAyrZzDsGYdLVL9zYtAWWM`,
	RunE: runAddressParse,
}

func init() {
	rootCmd.AddCommand(addressCmd)
	addressCmd.AddCommand(addressGenerateCmd)
	addressCmd.AddCommand(addressParseCmd)

	// Flags for address generate
	addressGenerateCmd.Flags().StringVar(&keyFile, "key", "", "Key file path")
	addressGenerateCmd.Flags().StringVar(&keyFormat, "key-format", "jwk", "Key file format (jwk, pem)")
	addressGenerateCmd.Flags().StringVarP(&storageDir, "storage-dir", "s", "", "Storage directory")
	addressGenerateCmd.Flags().StringVarP(&keyID, "key-id", "k", "", "Key ID for storage")
	addressGenerateCmd.Flags().StringVar(&chainType, "chain", "", "Blockchain type (ethereum, solana)")
	addressGenerateCmd.Flags().BoolVar(&allChains, "all", false, "Generate addresses for all compatible chains")
	addressGenerateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
}

func runAddressGenerate(cmd *cobra.Command, args []string) error {
	// Load the key
	keyPair, err := loadKeyForAddress()
	if err != nil {
		return err
	}

	// Determine which chains to generate addresses for
	var chains []chain.ChainType
	if allChains {
		// Generate for all compatible chains
		chains = chain.GetSupportedChainsForKey(keyPair)
		if len(chains) == 0 {
			return fmt.Errorf("no compatible chains found for key type %s", keyPair.Type())
		}
	} else if chainType != "" {
		// Generate for specific chain
		ct := chain.ChainType(strings.ToLower(chainType))

		// Validate chain is supported
		provider, err := chain.GetProvider(ct)
		if err != nil {
			return fmt.Errorf("unsupported chain: %s", chainType)
		}

		// Validate key type is compatible
		if err := chain.ValidateKeyForChain(keyPair.PublicKey(), ct); err != nil {
			return err
		}

		chains = []chain.ChainType{ct}
		_ = provider // provider is used for validation
	} else {
		// Default to all compatible chains
		chains = chain.GetSupportedChainsForKey(keyPair)
	}

	// Generate addresses
	addresses, err := chain.AddressFromKeyPair(keyPair, chains...)
	if err != nil {
		return fmt.Errorf("failed to generate addresses: %w", err)
	}

	if len(addresses) == 0 {
		return fmt.Errorf("no addresses could be generated")
	}

	// Output results
	return outputAddresses(addresses, keyPair)
}

func runAddressParse(cmd *cobra.Command, args []string) error {
	addressStr := args[0]

	// Try to parse the address
	address, err := chain.ParseAddress(addressStr)
	if err != nil {
		return fmt.Errorf("failed to parse address: %w", err)
	}

	// Get the provider for validation
	provider, err := chain.GetProvider(address.Chain)
	if err != nil {
		return fmt.Errorf("failed to get provider: %w", err)
	}

	// Validate the address
	if err := provider.ValidateAddress(addressStr, address.Network); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	// Try to recover public key if possible
	pubKey, err := provider.GetPublicKeyFromAddress(cmd.Context(), addressStr, address.Network)
	canRecoverPubKey := err == nil

	// Output parsed information
	fmt.Printf("Address: %s\n", address.Value)
	fmt.Printf("Chain: %s\n", address.Chain)
	fmt.Printf("Network: %s\n", address.Network)
	fmt.Printf("Valid: \n")

	if canRecoverPubKey && pubKey != nil {
		fmt.Printf("Public Key Recoverable: Yes\n")

		// Show public key type
		switch pubKey.(type) {
		case ed25519.PublicKey:
			fmt.Printf("Key Type: Ed25519\n")
		case *ecdsa.PublicKey:
			fmt.Printf("Key Type: Secp256k1\n")
		}
	} else {
		fmt.Printf("Public Key Recoverable: No\n")
	}

	return nil
}

func outputAddresses(addresses map[chain.ChainType]*chain.Address, keyPair crypto.KeyPair) error {
	// Prepare output data
	output := map[string]interface{}{
		"key_id":    keyPair.ID(),
		"key_type":  string(keyPair.Type()),
		"addresses": make(map[string]string),
	}

	// Add addresses
	for chainType, address := range addresses {
		output["addresses"].(map[string]string)[string(chainType)] = address.Value
	}

	// Format output
	if outputFile != "" {
		// JSON output to file
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal output: %w", err)
		}

		if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		fmt.Printf("Addresses saved to: %s\n", outputFile)
	} else {
		// Table output to stdout
		fmt.Printf("Key Information:\n")
		fmt.Printf("  ID: %s\n", keyPair.ID())
		fmt.Printf("  Type: %s\n", keyPair.Type())
		fmt.Printf("\nGenerated Addresses:\n\n")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "CHAIN\tADDRESS\tNETWORK\n")
		fmt.Fprintf(w, "-----\t-------\t-------\n")

		// Sort chain types for consistent output
		var chainTypes []chain.ChainType
		for ct := range addresses {
			chainTypes = append(chainTypes, ct)
		}
		sort.Slice(chainTypes, func(i, j int) bool {
			return chainTypes[i] < chainTypes[j]
		})

		// Print addresses in sorted order
		for _, ct := range chainTypes {
			address := addresses[ct]
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				ct,
				address.Value,
				address.Network)
		}

		w.Flush()
	}

	return nil
}

// loadKeyForAddress loads a key from file or storage
func loadKeyForAddress() (crypto.KeyPair, error) {
	// Check if using storage
	if storageDir != "" && keyID != "" {
		keyStorage, err := storage.NewFileKeyStorage(storageDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create key storage: %w", err)
		}

		keyPair, err := keyStorage.Load(keyID)
		if err != nil {
			return nil, fmt.Errorf("failed to load key from storage: %w", err)
		}

		return keyPair, nil
	}

	// Load from file
	if keyFile == "" {
		return nil, fmt.Errorf("either --key or --storage-dir with --key-id must be specified")
	}

	// Read key file
	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Import the key
	var importer crypto.KeyImporter
	var format crypto.KeyFormat

	switch keyFormat {
	case "jwk":
		importer = formats.NewJWKImporter()
		format = crypto.KeyFormatJWK

		// Handle the wrapper format from sage-crypto generate
		var wrapper struct {
			PrivateKey json.RawMessage `json:"private_key"`
			PublicKey  json.RawMessage `json:"public_key"`
			KeyID      string          `json:"key_id"`
			KeyType    string          `json:"key_type"`
		}

		if err := json.Unmarshal(keyData, &wrapper); err == nil && wrapper.PrivateKey != nil {
			// It's a wrapper format, use the private key
			keyData = wrapper.PrivateKey
		}

	case "pem":
		importer = formats.NewPEMImporter()
		format = crypto.KeyFormatPEM
	default:
		return nil, fmt.Errorf("unsupported key format: %s", keyFormat)
	}

	keyPair, err := importer.Import(keyData, format)
	if err != nil {
		return nil, fmt.Errorf("failed to import key: %w", err)
	}

	return keyPair, nil
}
