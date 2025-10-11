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

package did

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

// MarshalPublicKey converts a public key to bytes for storage
func MarshalPublicKey(publicKey interface{}) ([]byte, error) {
	switch pk := publicKey.(type) {
	case ed25519.PublicKey:
		return pk, nil
	case *secp256k1.PublicKey:
		return pk.SerializeCompressed(), nil
	default:
		// Try to marshal as generic public key using x509
		return x509.MarshalPKIXPublicKey(publicKey)
	}
}

// UnmarshalPublicKey converts bytes back to a public key
func UnmarshalPublicKey(data []byte, keyType string) (interface{}, error) {
	switch keyType {
	case "ed25519":
		if len(data) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("invalid Ed25519 public key size: %d", len(data))
		}
		return ed25519.PublicKey(data), nil

	case "secp256k1":
		pk, err := secp256k1.ParsePubKey(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse secp256k1 public key: %w", err)
		}
		// Convert to standard ecdsa.PublicKey for compatibility
		return pk.ToECDSA(), nil

	default:
		// Try to unmarshal as generic public key
		block, _ := pem.Decode(data)
		if block != nil {
			data = block.Bytes
		}
		return x509.ParsePKIXPublicKey(data)
	}
}
