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
)

// LEGACY HELPER FUNCTIONS
// These functions are used by existing CLI commands (deactivate, update, key, list, etc.)
// TODO: Refactor other commands to use three-phase registration flow

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
