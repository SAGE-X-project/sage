package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/crypto/storage"
	"github.com/spf13/cobra"
)

var (
	keyType      string
	outputFormat string
	outputFile   string
	storageDir   string
	keyID        string
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new key pair",
	Long: `Generate a new cryptographic key pair.

Supported key types:
  - ed25519: Ed25519 signature algorithm
  - secp256k1: Secp256k1 elliptic curve (used in Bitcoin/Ethereum)

Supported output formats:
  - jwk: JSON Web Key format
  - pem: PEM (Privacy Enhanced Mail) format
  - storage: Store directly in key storage`,
	Example: `  # Generate Ed25519 key and output as JWK
  sage-crypto generate --type ed25519 --format jwk

  # Generate Secp256k1 key and save to file
  sage-crypto generate --type secp256k1 --format pem --output mykey.pem

  # Generate key and store in key storage
  sage-crypto generate --type ed25519 --format storage --storage-dir ./keys --key-id mykey`,
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&keyType, "type", "t", "ed25519", "Key type (ed25519, secp256k1)")
	generateCmd.Flags().StringVarP(&outputFormat, "format", "f", "jwk", "Output format (jwk, pem, storage)")
	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	generateCmd.Flags().StringVarP(&storageDir, "storage-dir", "s", "", "Storage directory (required for storage format)")
	generateCmd.Flags().StringVarP(&keyID, "key-id", "k", "", "Key ID (required for storage format)")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Generate key pair based on type
	var keyPair crypto.KeyPair
	var err error

	switch keyType {
	case "ed25519":
		keyPair, err = keys.GenerateEd25519KeyPair()
	case "secp256k1":
		keyPair, err = keys.GenerateSecp256k1KeyPair()
	default:
		return fmt.Errorf("unsupported key type: %s", keyType)
	}

	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Handle output based on format
	switch outputFormat {
	case "jwk":
		return outputJWK(keyPair)
	case "pem":
		return outputPEM(keyPair)
	case "storage":
		return storeKey(keyPair)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

func outputJWK(keyPair crypto.KeyPair) error {
	exporter := formats.NewJWKExporter()
	
	// Export private key
	privateJWK, err := exporter.Export(keyPair, crypto.KeyFormatJWK)
	if err != nil {
		return fmt.Errorf("failed to export private key: %w", err)
	}

	// Export public key
	publicJWK, err := exporter.ExportPublic(keyPair, crypto.KeyFormatJWK)
	if err != nil {
		return fmt.Errorf("failed to export public key: %w", err)
	}

	// Create output structure
	output := map[string]json.RawMessage{
		"private_key": privateJWK,
		"public_key":  publicJWK,
		"key_id":      json.RawMessage(fmt.Sprintf(`"%s"`, keyPair.ID())),
		"key_type":    json.RawMessage(fmt.Sprintf(`"%s"`, keyPair.Type())),
	}

	// Marshal to JSON
	jsonOutput, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}

	return writeOutput(jsonOutput)
}

func outputPEM(keyPair crypto.KeyPair) error {
	exporter := formats.NewPEMExporter()
	
	// Export private key
	privatePEM, err := exporter.Export(keyPair, crypto.KeyFormatPEM)
	if err != nil {
		return fmt.Errorf("failed to export private key: %w", err)
	}

	// Export public key
	publicPEM, err := exporter.ExportPublic(keyPair, crypto.KeyFormatPEM)
	if err != nil {
		return fmt.Errorf("failed to export public key: %w", err)
	}

	// Combine PEM blocks
	output := append(privatePEM, publicPEM...)
	
	// Add metadata as comments
	metadata := fmt.Sprintf("# Key ID: %s\n# Key Type: %s\n", keyPair.ID(), keyPair.Type())
	output = append([]byte(metadata), output...)

	return writeOutput(output)
}

func storeKey(keyPair crypto.KeyPair) error {
	if storageDir == "" {
		return fmt.Errorf("storage directory is required for storage format")
	}
	if keyID == "" {
		return fmt.Errorf("key ID is required for storage format")
	}

	// Create file storage
	keyStorage, err := storage.NewFileKeyStorage(storageDir)
	if err != nil {
		return fmt.Errorf("failed to create key storage: %w", err)
	}

	// Store the key
	if err := keyStorage.Store(keyID, keyPair); err != nil {
		return fmt.Errorf("failed to store key: %w", err)
	}

	fmt.Printf("Key successfully stored:\n")
	fmt.Printf("  Key ID: %s\n", keyID)
	fmt.Printf("  Key Type: %s\n", keyPair.Type())
	fmt.Printf("  Key Fingerprint: %s\n", keyPair.ID())
	fmt.Printf("  Storage Location: %s\n", filepath.Join(storageDir, keyID+".key"))

	return nil
}

func writeOutput(data []byte) error {
	if outputFile == "" {
		// Write to stdout
		fmt.Print(string(data))
		return nil
	}

	// Write to file
	if err := os.WriteFile(outputFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("Key saved to: %s\n", outputFile)
	return nil
}