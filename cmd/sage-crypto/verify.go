// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"

	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/spf13/cobra"
)

var (
	publicKeyFile string
	signatureFile string
	signatureB64  string
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify a signature using a public key",
	Long: `Verify a signature using a public key.

The public key can be provided as:
  - A file in JWK or PEM format
  - Part of a private key file (will extract public key)

The signature can be provided as:
  - Base64 encoded string
  - JSON file containing signature data
  - Raw signature file`,
	Example: `  # Verify using public key and base64 signature
  sage-crypto verify --key public.jwk --message "Hello, World!" --signature-b64 "base64sig..."

  # Verify using signature file
  sage-crypto verify --key mykey.pem --format pem --message-file document.txt --signature-file sig.json

  # Verify from stdin
  echo "Message" | sage-crypto verify --key public.jwk --signature-b64 "base64sig..."`,
	RunE: runVerify,
}

func init() {
	rootCmd.AddCommand(verifyCmd)

	verifyCmd.Flags().StringVar(&publicKeyFile, "key", "", "Public key file path (required)")
	verifyCmd.Flags().StringVar(&keyFormat, "key-format", "jwk", "Key file format (jwk, pem)")
	verifyCmd.Flags().StringVarP(&message, "message", "m", "", "Message to verify")
	verifyCmd.Flags().StringVar(&messageFile, "message-file", "", "File containing message to verify")
	verifyCmd.Flags().StringVar(&signatureFile, "signature-file", "", "Signature file (JSON or raw)")
	verifyCmd.Flags().StringVar(&signatureB64, "signature-b64", "", "Base64 encoded signature")
	
	verifyCmd.MarkFlagRequired("key")
}

func runVerify(cmd *cobra.Command, args []string) error {
	// Load the public key
	publicKey, keyPair, err := loadPublicKey()
	if err != nil {
		return err
	}

	// Get the message
	messageBytes, err := getMessage()
	if err != nil {
		return err
	}

	// Get the signature
	signature, err := getSignature()
	if err != nil {
		return err
	}

	// Verify the signature
	var verifyErr error
	var keyType string
	
	if keyPair != nil {
		// We have a full key pair, use it for verification
		verifyErr = keyPair.Verify(messageBytes, signature)
		keyType = string(keyPair.Type())
	} else {
		// We only have a public key, verify directly
		verifyErr = verifyWithPublicKey(publicKey, messageBytes, signature)
		
		// Determine key type
		switch publicKey.(type) {
		case ed25519.PublicKey:
			keyType = "Ed25519"
		case *ecdsa.PublicKey:
			keyType = "Secp256k1"
		default:
			keyType = "Unknown"
		}
	}

	if verifyErr != nil {
		fmt.Println(" Signature verification FAILED")
		return fmt.Errorf("invalid signature: %w", verifyErr)
	}

	fmt.Println(" Signature verification PASSED")
	
	// Output additional information
	fmt.Printf("Key Type: %s\n", keyType)
	if keyPair != nil {
		fmt.Printf("Key ID: %s\n", keyPair.ID())
	}
	
	return nil
}

func loadPublicKey() (crypto.PublicKey, sagecrypto.KeyPair, error) {
	// Read key file
	keyData, err := os.ReadFile(publicKeyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Try to import as a full key pair first
	var importer sagecrypto.KeyImporter
	var format sagecrypto.KeyFormat

	switch keyFormat {
	case "jwk":
		importer = formats.NewJWKImporter()
		format = sagecrypto.KeyFormatJWK
		
		// Handle the wrapper format from sage-crypto generate
		var wrapper struct {
			PrivateKey json.RawMessage `json:"private_key"`
			PublicKey  json.RawMessage `json:"public_key"`
			KeyID      string          `json:"key_id"`
			KeyType    string          `json:"key_type"`
		}
		
		if err := json.Unmarshal(keyData, &wrapper); err == nil && (wrapper.PrivateKey != nil || wrapper.PublicKey != nil) {
			// It's a wrapper format
			if wrapper.PrivateKey != nil {
				// Use private key if available (contains both private and public)
				keyData = wrapper.PrivateKey
			} else if wrapper.PublicKey != nil {
				// Use public key only
				keyData = wrapper.PublicKey
			}
		}
		
	case "pem":
		importer = formats.NewPEMImporter()
		format = sagecrypto.KeyFormatPEM
	default:
		return nil, nil, fmt.Errorf("unsupported key format: %s", keyFormat)
	}

	// Try to import as key pair (might be a private key file)
	keyPair, err := importer.Import(keyData, format)
	if err == nil {
		return keyPair.PublicKey(), keyPair, nil
	}

	// Try to import as public key only
	publicKey, err := importer.ImportPublic(keyData, format)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to import key: %w", err)
	}

	return publicKey, nil, nil
}

func getSignature() ([]byte, error) {
	// Priority: base64 signature, signature file
	if signatureB64 != "" {
		signature, err := base64.StdEncoding.DecodeString(signatureB64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 signature: %w", err)
		}
		return signature, nil
	}

	if signatureFile != "" {
		data, err := os.ReadFile(signatureFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read signature file: %w", err)
		}

		// Try to parse as JSON first
		var sigData map[string]interface{}
		if err := json.Unmarshal(data, &sigData); err == nil {
			// It's JSON, extract signature field
			if sigStr, ok := sigData["signature"].(string); ok {
				signature, err := base64.StdEncoding.DecodeString(sigStr)
				if err != nil {
					return nil, fmt.Errorf("failed to decode signature from JSON: %w", err)
				}
				return signature, nil
			}
			return nil, fmt.Errorf("signature field not found in JSON")
		}

		// Not JSON, treat as raw signature
		return data, nil
	}

	// Try to read from stdin if it's not being used for message
	if message != "" || messageFile != "" {
		// Stdin might contain signature
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read signature from stdin: %w", err)
		}
		
		if len(data) > 0 {
			// Try base64 decode
			if signature, err := base64.StdEncoding.DecodeString(string(data)); err == nil {
				return signature, nil
			}
			// Return as is
			return data, nil
		}
	}

	return nil, fmt.Errorf("no signature provided")
}

// verifyWithPublicKey verifies a signature using only a public key
func verifyWithPublicKey(publicKey crypto.PublicKey, message, signature []byte) error {
	switch pk := publicKey.(type) {
	case ed25519.PublicKey:
		if !ed25519.Verify(pk, message, signature) {
			return fmt.Errorf("ed25519 signature verification failed")
		}
		return nil
		
	case *ecdsa.PublicKey:
		// For ECDSA, we need to hash the message and parse the signature
		hash := crypto.SHA256.New()
		hash.Write(message)
		hashed := hash.Sum(nil)
		
		// ECDSA signature should be 64 bytes (32 bytes for r, 32 bytes for s)
		if len(signature) != 64 {
			return fmt.Errorf("invalid ECDSA signature length: expected 64 bytes, got %d", len(signature))
		}
		
		// Split signature into r and s components
		r := new(big.Int).SetBytes(signature[:32])
		s := new(big.Int).SetBytes(signature[32:])
		
		if !ecdsa.Verify(pk, hashed, r, s) {
			return fmt.Errorf("ecdsa signature verification failed")
		}
		return nil
		
	default:
		return fmt.Errorf("unsupported public key type: %T", publicKey)
	}
}