package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/sage-x-project/sage/crypto/storage"
	"github.com/spf13/cobra"
)

var (
	keyFile      string
	keyFormat    string
	messageFile  string
	message      string
	signatureOut string
	base64Output bool
)

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Sign a message with a private key",
	Long: `Sign a message using a private key.

The key can be loaded from:
  - A file in JWK or PEM format
  - Key storage using storage directory and key ID

The message can be provided as:
  - Command line argument
  - File content
  - Stdin (if no message or file specified)`,
	Example: `  # Sign a message using a JWK key file
  sage-crypto sign --key mykey.jwk --message "Hello, World!"

  # Sign a file using a PEM key
  sage-crypto sign --key mykey.pem --format pem --message-file document.txt

  # Sign using a key from storage
  sage-crypto sign --storage-dir ./keys --key-id mykey --message "Test message"

  # Sign from stdin and output base64
  echo "Message to sign" | sage-crypto sign --key mykey.jwk --base64`,
	RunE: runSign,
}

func init() {
	rootCmd.AddCommand(signCmd)

	signCmd.Flags().StringVar(&keyFile, "key", "", "Key file path")
	signCmd.Flags().StringVar(&keyFormat, "key-format", "jwk", "Key file format (jwk, pem)")
	signCmd.Flags().StringVarP(&storageDir, "storage-dir", "s", "", "Storage directory")
	signCmd.Flags().StringVarP(&keyID, "key-id", "k", "", "Key ID for storage")
	signCmd.Flags().StringVarP(&message, "message", "m", "", "Message to sign")
	signCmd.Flags().StringVar(&messageFile, "message-file", "", "File containing message to sign")
	signCmd.Flags().StringVarP(&signatureOut, "output", "o", "", "Output file for signature")
	signCmd.Flags().BoolVar(&base64Output, "base64", false, "Output signature as base64")
}

func runSign(cmd *cobra.Command, args []string) error {
	// Load the key
	keyPair, err := loadKey()
	if err != nil {
		return err
	}

	// Get the message to sign
	messageBytes, err := getMessage()
	if err != nil {
		return err
	}

	// Sign the message
	signature, err := keyPair.Sign(messageBytes)
	if err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}

	// Output the signature
	return outputSignature(signature, keyPair)
}

func loadKey() (crypto.KeyPair, error) {
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

func getMessage() ([]byte, error) {
	// Priority: message flag, message file, stdin
	if message != "" {
		return []byte(message), nil
	}

	if messageFile != "" {
		data, err := os.ReadFile(messageFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read message file: %w", err)
		}
		return data, nil
	}

	// Read from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no message provided")
	}

	return data, nil
}

func outputSignature(signature []byte, keyPair crypto.KeyPair) error {
	// Prepare output
	var output []byte

	if base64Output {
		output = []byte(base64.StdEncoding.EncodeToString(signature))
	} else {
		// JSON output with metadata
		result := map[string]interface{}{
			"signature": base64.StdEncoding.EncodeToString(signature),
			"key_id":    keyPair.ID(),
			"key_type":  string(keyPair.Type()),
			"algorithm": getSignatureAlgorithm(keyPair.Type()),
		}
		
		jsonOutput, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal output: %w", err)
		}
		output = jsonOutput
	}

	// Write output
	if signatureOut != "" {
		if err := os.WriteFile(signatureOut, output, 0644); err != nil {
			return fmt.Errorf("failed to write signature file: %w", err)
		}
		fmt.Printf("Signature saved to: %s\n", signatureOut)
	} else {
		fmt.Println(string(output))
	}

	return nil
}

func getSignatureAlgorithm(keyType crypto.KeyType) string {
	switch keyType {
	case crypto.KeyTypeEd25519:
		return "EdDSA"
	case crypto.KeyTypeSecp256k1:
		return "ECDSA-secp256k1"
	default:
		return "unknown"
	}
}