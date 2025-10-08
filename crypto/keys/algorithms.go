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


package keys

import (
	"log"

	sagecrypto "github.com/sage-x-project/sage/crypto"
)

// init registers all supported cryptographic algorithms
func init() {
	// Register Ed25519
	if err := sagecrypto.RegisterAlgorithm(sagecrypto.AlgorithmInfo{
		KeyType:               sagecrypto.KeyTypeEd25519,
		Name:                  "Ed25519",
		Description:           "Edwards-curve Digital Signature Algorithm using Curve25519",
		RFC9421Algorithm:      "ed25519",
		SupportsRFC9421:       true,
		SupportsKeyGeneration: true,
		SupportsSignature:     true,
		SupportsEncryption:    false,
	}); err != nil {
		log.Fatalf("Failed to register Ed25519 algorithm: %v", err)
	}

	// Register Secp256k1 (Ethereum-compatible)
	if err := sagecrypto.RegisterAlgorithm(sagecrypto.AlgorithmInfo{
		KeyType:               sagecrypto.KeyTypeSecp256k1,
		Name:                  "Secp256k1",
		Description:           "ECDSA with secp256k1 curve (used by Bitcoin and Ethereum)",
		RFC9421Algorithm:      "es256k",
		SupportsRFC9421:       true,
		SupportsKeyGeneration: true,
		SupportsSignature:     true,
		SupportsEncryption:    false,
	}); err != nil {
		log.Fatalf("Failed to register Secp256k1 algorithm: %v", err)
	}

	// Note: ECDSA P-256 (ecdsa-p256) is not registered separately
	// The registry currently maps all ECDSA keys to Secp256k1
	// TODO: Add support for distinguishing between different ECDSA curves (P-256, Secp256k1, etc.)

	// Register X25519 (key exchange only, not for signing)
	if err := sagecrypto.RegisterAlgorithm(sagecrypto.AlgorithmInfo{
		KeyType:               sagecrypto.KeyTypeX25519,
		Name:                  "X25519",
		Description:           "Elliptic Curve Diffie-Hellman (ECDH) using Curve25519 for key exchange",
		RFC9421Algorithm:      "", // X25519 is for key exchange, not signing
		SupportsRFC9421:       false,
		SupportsKeyGeneration: true,
		SupportsSignature:     false,
		SupportsEncryption:    true,
	}); err != nil {
		log.Fatalf("Failed to register X25519 algorithm: %v", err)
	}

	// Register RSA
	if err := sagecrypto.RegisterAlgorithm(sagecrypto.AlgorithmInfo{
		KeyType:               sagecrypto.KeyTypeRSA,
		Name:                  "RSA-PSS-SHA256",
		Description:           "RSA with PSS padding and SHA-256",
		RFC9421Algorithm:      "rsa-pss-sha256",
		SupportsRFC9421:       true,
		SupportsKeyGeneration: true,
		SupportsSignature:     true,
		SupportsEncryption:    true,
	}); err != nil {
		log.Fatalf("Failed to register RSA algorithm: %v", err)
	}
}
