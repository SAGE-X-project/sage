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
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"golang.org/x/crypto/sha3"
)

// MarshalPublicKey converts a public key to bytes for storage
func MarshalPublicKey(publicKey interface{}) ([]byte, error) {
	switch pk := publicKey.(type) {
	case ed25519.PublicKey:
		return pk, nil
	case *secp256k1.PublicKey:
		return pk.SerializeCompressed(), nil
	case *ecdsa.PublicKey:
		// Handle ECDSA public keys (including secp256k1 converted to ECDSA)
		// Check if this is a secp256k1 curve (Ethereum)
		// Note: We check curve name instead of pointer equality because
		// different libraries may use different curve instances
		if pk.Curve.Params().Name == "secp256k1" {
			// For secp256k1, use UNCOMPRESSED format (64 bytes: x || y)
			// V4 contract rejects compressed keys due to expensive decompression on-chain
			// Returns raw 64-byte format (without 0x04 prefix)
			// Contract accepts both 64-byte and 65-byte (with 0x04) formats
			byteLen := (pk.Curve.Params().BitSize + 7) / 8
			bytes := make([]byte, 2*byteLen)
			pk.X.FillBytes(bytes[0:byteLen])
			pk.Y.FillBytes(bytes[byteLen:])
			return bytes, nil
		}
		// For other ECDSA curves, use uncompressed format (0x04 || X || Y)
		// Manual construction to avoid deprecated elliptic.Marshal
		byteLen := (pk.Curve.Params().BitSize + 7) / 8
		bytes := make([]byte, 1+2*byteLen)
		bytes[0] = 0x04 // uncompressed point format
		pk.X.FillBytes(bytes[1 : 1+byteLen])
		pk.Y.FillBytes(bytes[1+byteLen:])
		return bytes, nil
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
		// Handle multiple formats: compressed (33 bytes), uncompressed (65 bytes), or raw (64 bytes)
		if len(data) == 64 {
			// Raw format (64 bytes: x || y) - prepend 0x04 for standard uncompressed format
			data = append([]byte{0x04}, data...)
		}
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

// GenerateAgentDIDWithAddress creates a DID that includes the owner's address.
//
// Format:
//   - Ethereum: did:sage:ethereum:0x{address}
//   - Solana: did:sage:solana:{address}
//
// This format enables:
//   - Off-chain ownership verification
//   - Cross-chain traceability
//   - Prevention of DID collisions across different owners
//
// The function automatically normalizes the address:
//   - Ethereum: Adds "0x" prefix if missing and converts to lowercase
//   - Solana: Converts to lowercase (no prefix)
//
// Example:
//
//	// Ethereum
//	chain := ChainEthereum
//	ownerAddr := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
//	agentDID := GenerateAgentDIDWithAddress(chain, ownerAddr)
//	// Returns: "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"
//
//	// Solana
//	chain := ChainSolana
//	ownerAddr := "DYw8jCTfwHNRJhhmFcbXvVDTqWMEVFBX6ZKUmG5CNSKK"
//	agentDID := GenerateAgentDIDWithAddress(chain, ownerAddr)
//	// Returns: "did:sage:solana:dyw8jctfwhnrjhhmfcbxvvdtqwmevfbx6zkumg5cnskk"
//
// This function is used by sage-a2a-go for creating DIDs that can be verified
// against on-chain ownership records.
func GenerateAgentDIDWithAddress(chain Chain, ownerAddress string) AgentDID {
	// For Ethereum, ensure address starts with 0x
	if chain == ChainEthereum {
		if !strings.HasPrefix(ownerAddress, "0x") {
			ownerAddress = "0x" + ownerAddress
		}
	}
	// Convert to lowercase for consistency
	ownerAddress = strings.ToLower(ownerAddress)
	return AgentDID(fmt.Sprintf("did:sage:%s:%s", chain, ownerAddress))
}

// GenerateAgentDIDWithNonce creates a DID with both owner address and nonce.
//
// Format:
//   - Ethereum: did:sage:ethereum:0x{address}:{nonce}
//   - Solana: did:sage:solana:{address}:{nonce}
//
// Use case: Creating multiple agents per owner
//
// The nonce enables a single address to register multiple distinct
// agent identities on the SAGE registry. This is useful for:
//   - Multi-agent systems controlled by one owner
//   - Agent versioning and migration
//   - Separating different agent roles/capabilities
//
// Example:
//
//	// Ethereum
//	chain := ChainEthereum
//	ownerAddr := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
//	agentDID1 := GenerateAgentDIDWithNonce(chain, ownerAddr, 0)
//	// Returns: "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266:0"
//	agentDID2 := GenerateAgentDIDWithNonce(chain, ownerAddr, 1)
//	// Returns: "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266:1"
//
// See also: GenerateAgentDIDWithAddress for single-agent-per-owner scenarios.
func GenerateAgentDIDWithNonce(chain Chain, ownerAddress string, nonce uint64) AgentDID {
	// For Ethereum, ensure address starts with 0x
	if chain == ChainEthereum {
		if !strings.HasPrefix(ownerAddress, "0x") {
			ownerAddress = "0x" + ownerAddress
		}
	}
	// Convert to lowercase for consistency
	ownerAddress = strings.ToLower(ownerAddress)
	return AgentDID(fmt.Sprintf("did:sage:%s:%s:%d", chain, ownerAddress, nonce))
}

// DeriveEthereumAddress derives the Ethereum address from a secp256k1 keypair.
//
// This function implements the standard Ethereum address derivation algorithm:
//  1. Extract the ECDSA public key (64 bytes: X || Y coordinates)
//  2. Compute Keccak256 hash of the public key
//  3. Take the last 20 bytes as the address
//  4. Format with "0x" prefix
//
// The derived address can be used with GenerateAgentDIDWithAddress() to create
// DIDs that include owner verification, enabling the enhanced DID format:
// did:sage:ethereum:0x{derived_address}
//
// Example:
//
//	keyPair, _ := crypto.GenerateSecp256k1KeyPair()
//	address, _ := DeriveEthereumAddress(keyPair)
//	// address: "0x742d35cc6634c0532925a3b844bc9e7595f0beef"
//	agentDID := GenerateAgentDIDWithAddress(ChainEthereum, address)
//	// agentDID: "did:sage:ethereum:0x742d35cc6634c0532925a3b844bc9e7595f0beef"
//
// Important: This function ONLY works with secp256k1 keys. For other key types,
// it returns an error.
//
// Returns:
//   - Ethereum address with "0x" prefix (lowercase)
//   - Error if the key is not secp256k1 or derivation fails
//
// See also:
//   - https://ethereum.org/en/developers/docs/accounts/#account-creation
//   - https://github.com/ethereum/go-ethereum/blob/master/crypto/crypto.go
func DeriveEthereumAddress(keyPair crypto.KeyPair) (string, error) {
	// Verify this is a secp256k1 key
	if keyPair.Type() != crypto.KeyTypeSecp256k1 {
		return "", fmt.Errorf("ethereum address derivation requires secp256k1 key, got %s", keyPair.Type())
	}

	// Extract ECDSA public key
	ecdsaPubKey, ok := keyPair.PublicKey().(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("failed to convert public key to ECDSA format")
	}

	// Convert public key to uncompressed format (64 bytes: 32 bytes X + 32 bytes Y)
	pubKeyBytes := make([]byte, 64)
	ecdsaPubKey.X.FillBytes(pubKeyBytes[:32])
	ecdsaPubKey.Y.FillBytes(pubKeyBytes[32:])

	// Keccak256 hash of the public key
	hash := sha3.NewLegacyKeccak256()
	hash.Write(pubKeyBytes)
	addressBytes := hash.Sum(nil)

	// Take the last 20 bytes as the address and format with 0x prefix
	address := "0x" + hex.EncodeToString(addressBytes[12:])

	return address, nil
}
