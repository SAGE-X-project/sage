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


package formats

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

// pemExporter implements KeyExporter for PEM format
type pemExporter struct{}

// NewPEMExporter creates a new PEM exporter
func NewPEMExporter() sagecrypto.KeyExporter {
	return &pemExporter{}
}

// Export exports the key pair in PEM format
func (e *pemExporter) Export(keyPair sagecrypto.KeyPair, format sagecrypto.KeyFormat) ([]byte, error) {
	if format != sagecrypto.KeyFormatPEM {
		return nil, sagecrypto.ErrInvalidKeyFormat
	}

	switch keyPair.Type() {
	case sagecrypto.KeyTypeEd25519:
		privateKey, ok := keyPair.PrivateKey().(ed25519.PrivateKey)
		if !ok {
			return nil, errors.New("invalid Ed25519 private key type")
		}
		
		// Use PKCS8 format for Ed25519
		derBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Ed25519 private key: %w", err)
		}
		
		block := &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: derBytes,
		}
		
		return pem.EncodeToMemory(block), nil

	case sagecrypto.KeyTypeSecp256k1:
		privateKey, ok := keyPair.PrivateKey().(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("invalid Secp256k1 private key type")
		}
		
		// For secp256k1, we'll use PKCS8 format with custom OID
		// First convert to a generic private key structure
		privKeyBytes := privateKey.D.Bytes()
		
		// Ensure the private key is 32 bytes
		if len(privKeyBytes) < 32 {
			// Pad with zeros at the beginning
			padded := make([]byte, 32)
			copy(padded[32-len(privKeyBytes):], privKeyBytes)
			privKeyBytes = padded
		}
		
		// NOTE: This is a non-standard format due to x509 package limitations.
		// Standard x509.MarshalPKCS8PrivateKey doesn't support secp256k1 curve.
		// We store raw 32-byte private key with a custom header to indicate the curve.
		// For better interoperability, consider using JWK format instead of PEM for secp256k1 keys.
		block := &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: privKeyBytes,
			Headers: map[string]string{
				"Curve": "secp256k1",
			},
		}
		
		return pem.EncodeToMemory(block), nil
		
	case sagecrypto.KeyTypeRSA:
        privateKey, ok := keyPair.PrivateKey().(*rsa.PrivateKey)
        if !ok {
            return nil, errors.New("invalid RSA private key type")
        }
        derBytes := x509.MarshalPKCS1PrivateKey(privateKey)
        block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: derBytes}
        return pem.EncodeToMemory(block), nil

	default:
		return nil, sagecrypto.ErrInvalidKeyType
	}
}

// ExportPublic exports only the public key in PEM format
func (e *pemExporter) ExportPublic(keyPair sagecrypto.KeyPair, format sagecrypto.KeyFormat) ([]byte, error) {
	if format != sagecrypto.KeyFormatPEM {
		return nil, sagecrypto.ErrInvalidKeyFormat
	}

	switch keyPair.Type() {
	case sagecrypto.KeyTypeEd25519:
		publicKey := keyPair.PublicKey()
		// Use PKIX format for Ed25519 public keys
		derBytes, err := x509.MarshalPKIXPublicKey(publicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Ed25519 public key: %w", err)
		}
		
		block := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: derBytes,
		}
		
		return pem.EncodeToMemory(block), nil
		
	case sagecrypto.KeyTypeSecp256k1:
		publicKey, ok := keyPair.PublicKey().(*ecdsa.PublicKey)
		if !ok {
			return nil, errors.New("invalid Secp256k1 public key type")
		}
		
		// For secp256k1, we'll store the raw public key bytes
		// X and Y coordinates, 32 bytes each
		xBytes := publicKey.X.Bytes()
		yBytes := publicKey.Y.Bytes()
		
		// Ensure each coordinate is 32 bytes
		if len(xBytes) < 32 {
			padded := make([]byte, 32)
			copy(padded[32-len(xBytes):], xBytes)
			xBytes = padded
		}
		if len(yBytes) < 32 {
			padded := make([]byte, 32)
			copy(padded[32-len(yBytes):], yBytes)
			yBytes = padded
		}
		
		pubKeyBytes := append(xBytes, yBytes...)
		
		block := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: pubKeyBytes,
			Headers: map[string]string{
				"Curve": "secp256k1",
			},
		}
		
		return pem.EncodeToMemory(block), nil

	case sagecrypto.KeyTypeRSA:
        publicKey, ok := keyPair.PublicKey().(*rsa.PublicKey)
        if !ok {
            return nil, errors.New("invalid RSA public key type")
        }
        derBytes, err := x509.MarshalPKIXPublicKey(publicKey)
        if err != nil {
            return nil, fmt.Errorf("failed to marshal RSA public key: %w", err)
        }
        block := &pem.Block{Type: "PUBLIC KEY", Bytes: derBytes}
        return pem.EncodeToMemory(block), nil

	default:
		return nil, sagecrypto.ErrInvalidKeyType
	}
}

// pemImporter implements KeyImporter for PEM format
type pemImporter struct{}

// NewPEMImporter creates a new PEM importer
func NewPEMImporter() sagecrypto.KeyImporter {
	return &pemImporter{}
}

// Import imports a key pair from PEM format
func (i *pemImporter) Import(data []byte, format sagecrypto.KeyFormat) (sagecrypto.KeyPair, error) {
	if format != sagecrypto.KeyFormatPEM {
		return nil, sagecrypto.ErrInvalidKeyFormat
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	switch block.Type {
	case "PRIVATE KEY":
		// Try to parse as PKCS8
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}
		
		switch privateKey := key.(type) {
		case ed25519.PrivateKey:
			return keys.NewEd25519KeyPair(privateKey, "")
		case *ecdsa.PrivateKey:
			// Convert to secp256k1 private key
			privKeyBytes := privateKey.D.Bytes()
			secp256k1PrivKey := secp256k1.PrivKeyFromBytes(privKeyBytes)
			return keys.NewSecp256k1KeyPair(secp256k1PrivKey, "")
		default:
			return nil, fmt.Errorf("unsupported private key type: %T", privateKey)
		}

	case "EC PRIVATE KEY":
		// Check if this is our custom secp256k1 format
		if curve, ok := block.Headers["Curve"]; ok && curve == "secp256k1" {
			// Direct secp256k1 private key bytes
			if len(block.Bytes) != 32 {
				return nil, fmt.Errorf("invalid secp256k1 private key length: %d", len(block.Bytes))
			}
			secp256k1PrivKey := secp256k1.PrivKeyFromBytes(block.Bytes)
			return keys.NewSecp256k1KeyPair(secp256k1PrivKey, "")
		}
		
		// Try standard EC private key parsing (won't work for secp256k1)
		return nil, errors.New("standard EC private key format not supported for secp256k1")
	case "RSA PRIVATE KEY":
        priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
        if err != nil {
            return nil, fmt.Errorf("failed to parse RSA private key: %w", err)
        }
        return keys.NewRSAKeyPair(priv, "")

	default:
		return nil, fmt.Errorf("unsupported PEM block type: %s", block.Type)
	}
}

// ImportPublic imports only a public key from PEM format
func (i *pemImporter) ImportPublic(data []byte, format sagecrypto.KeyFormat) (crypto.PublicKey, error) {
	if format != sagecrypto.KeyFormatPEM {
		return nil, sagecrypto.ErrInvalidKeyFormat
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("expected PUBLIC KEY, got %s", block.Type)
	}

	// Check if this is our custom secp256k1 format
	if curve, ok := block.Headers["Curve"]; ok && curve == "secp256k1" {
		// Direct secp256k1 public key bytes (X and Y coordinates)
		if len(block.Bytes) != 64 {
			return nil, fmt.Errorf("invalid secp256k1 public key length: %d", len(block.Bytes))
		}
		
		xBytes := block.Bytes[:32]
		yBytes := block.Bytes[32:]
		
		pubKey := &ecdsa.PublicKey{
			Curve: secp256k1.S256(),
			X:     new(big.Int).SetBytes(xBytes),
			Y:     new(big.Int).SetBytes(yBytes),
		}
		return pubKey, nil
	}

	// Try standard PKIX parsing for other key types
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PKIX public key: %w", err)
	}

	switch key := publicKey.(type) {
	case ed25519.PublicKey:
		return key, nil
	case *ecdsa.PublicKey:
		return key, nil
	case *rsa.PublicKey:
        return key, nil
	default:
		return nil, fmt.Errorf("unsupported public key type: %T", publicKey)
	}
}
