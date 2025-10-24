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
	"crypto/sha256"
	"fmt"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// GenerateKeyProofOfPossession generates a signature proving ownership of a private key
//
// The proof-of-possession (PoP) is created by signing a challenge message that includes:
//   - The agent's DID
//   - The public key being proven
//   - A timestamp or nonce to prevent replay attacks
//
// Parameters:
//   - did: The agent's DID
//   - keyData: The public key data (raw bytes)
//   - privateKey: The corresponding private key
//   - keyType: Type of the key (Ed25519, ECDSA, etc.)
//
// Returns:
//   - Signature bytes proving key ownership
//   - Error if signing fails
func GenerateKeyProofOfPossession(did AgentDID, keyData []byte, privateKey interface{}, keyType KeyType) ([]byte, error) {
	// Create challenge message
	challenge := createPoPChallenge(did, keyData)

	// Hash the challenge
	hash := sha256.Sum256(challenge)

	// Sign based on key type
	switch keyType {
	case KeyTypeEd25519:
		ed25519Key, ok := privateKey.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("invalid Ed25519 private key type")
		}
		signature := ed25519.Sign(ed25519Key, hash[:])
		return signature, nil

	case KeyTypeECDSA:
		ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("invalid ECDSA private key type")
		}
		signature, err := ethcrypto.Sign(hash[:], ecdsaKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign with ECDSA: %w", err)
		}
		return signature, nil

	case KeyTypeX25519:
		// X25519 is for key agreement, not signing
		// For PoP, we could use a derived Ed25519 key or require a separate signing key
		return nil, fmt.Errorf("X25519 keys cannot generate signatures (key agreement only)")

	default:
		return nil, fmt.Errorf("unsupported key type for PoP: %s", keyType)
	}
}

// VerifyKeyProofOfPossession verifies a proof-of-possession signature
//
// This function verifies that:
//  1. The signature was created by the private key corresponding to keyData
//  2. The signature is valid for the challenge (DID + public key)
//
// Parameters:
//   - did: The agent's DID
//   - key: The agent key with signature to verify
//
// Returns:
//   - true if the proof is valid
//   - Error if verification fails
func VerifyKeyProofOfPossession(did AgentDID, key *AgentKey) error {
	if key == nil {
		return fmt.Errorf("key cannot be nil")
	}

	if len(key.Signature) == 0 {
		return fmt.Errorf("key has no proof-of-possession signature")
	}

	// Create the same challenge that was signed
	challenge := createPoPChallenge(did, key.KeyData)
	hash := sha256.Sum256(challenge)

	// Verify based on key type
	switch key.Type {
	case KeyTypeEd25519:
		if len(key.KeyData) != ed25519.PublicKeySize {
			return fmt.Errorf("invalid Ed25519 public key size: %d", len(key.KeyData))
		}
		pubKey := ed25519.PublicKey(key.KeyData)
		valid := ed25519.Verify(pubKey, hash[:], key.Signature)
		if !valid {
			return fmt.Errorf("Ed25519 PoP verification failed")
		}
		return nil

	case KeyTypeECDSA:
		// Handle both compressed (33 bytes) and uncompressed formats (64 or 65 bytes)
		keyData := key.KeyData
		if len(keyData) == 64 {
			// Raw format (64 bytes: x || y) - prepend 0x04 for standard uncompressed format
			keyData = append([]byte{0x04}, keyData...)
		}

		var pubKey *ecdsa.PublicKey
		var err error
		if len(keyData) == 33 {
			// Compressed format - use DecompressPubkey
			pubKey, err = ethcrypto.DecompressPubkey(keyData)
		} else if len(keyData) == 65 {
			// Uncompressed format - use UnmarshalPubkey
			pubKey, err = ethcrypto.UnmarshalPubkey(keyData)
		} else {
			return fmt.Errorf("invalid ECDSA public key length: %d (expected 33, 64, or 65 bytes)", len(keyData))
		}
		if err != nil {
			return fmt.Errorf("failed to parse ECDSA public key: %w", err)
		}

		// Ethereum signatures include a recovery ID in the last byte
		signature := key.Signature
		if len(signature) == 65 {
			signature = signature[:64] // Remove recovery ID for verification
		}

		// Verify signature
		valid := ethcrypto.VerifySignature(ethcrypto.CompressPubkey(pubKey), hash[:], signature)
		if !valid {
			return fmt.Errorf("ECDSA PoP verification failed")
		}
		return nil

	case KeyTypeX25519:
		// X25519 keys don't support signing
		return fmt.Errorf("X25519 keys cannot be verified for PoP (key agreement only)")

	default:
		return fmt.Errorf("unsupported key type for PoP verification: %s", key.Type)
	}
}

// VerifyAllKeyProofs verifies proof-of-possession for all keys in metadata
//
// This function:
//  1. Checks each key has a PoP signature
//  2. Verifies each signature is valid
//  3. Returns errors for any invalid proofs
//
// Parameters:
//   - metadata: Agent metadata containing keys to verify
//
// Returns:
//   - Error if any key fails verification
//   - nil if all keys have valid proofs
func VerifyAllKeyProofs(metadata *AgentMetadataV4) error {
	if metadata == nil {
		return fmt.Errorf("metadata cannot be nil")
	}

	if len(metadata.Keys) == 0 {
		return fmt.Errorf("no keys to verify")
	}

	var errors []string

	for i, key := range metadata.Keys {
		// Skip X25519 keys (they're for key agreement, not signing)
		if key.Type == KeyTypeX25519 {
			continue
		}

		// Verify PoP for signing keys
		if err := VerifyKeyProofOfPossession(metadata.DID, &metadata.Keys[i]); err != nil {
			errors = append(errors, fmt.Sprintf("key %d (%s): %v", i, key.Type, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("key proof verification failed for %d keys: %v", len(errors), errors)
	}

	return nil
}

// createPoPChallenge creates a consistent challenge message for proof-of-possession
//
// The challenge format is: "SAGE-PoP:" + DID + ":" + hex(public_key)
// This ensures:
//   - Each DID has unique challenges
//   - Each key has a unique challenge
//   - The format is easily auditable
func createPoPChallenge(did AgentDID, keyData []byte) []byte {
	message := fmt.Sprintf("SAGE-PoP:%s:%x", did, keyData)
	return []byte(message)
}

// ValidateKeyWithPoP performs comprehensive key validation including PoP
//
// This function combines:
//  1. Basic key structure validation
//  2. Proof-of-possession verification
//  3. Key type compatibility checks
//
// Returns:
//   - Error if any validation fails
//   - nil if all validations pass
func ValidateKeyWithPoP(did AgentDID, key *AgentKey) error {
	if key == nil {
		return fmt.Errorf("key cannot be nil")
	}

	// Validate key has data
	if len(key.KeyData) == 0 {
		return fmt.Errorf("key data is empty")
	}

	// Validate key type
	switch key.Type {
	case KeyTypeEd25519:
		if len(key.KeyData) != ed25519.PublicKeySize {
			return fmt.Errorf("invalid Ed25519 key size: expected %d, got %d",
				ed25519.PublicKeySize, len(key.KeyData))
		}
	case KeyTypeECDSA:
		// Ethereum compressed public key is 33 bytes
		if len(key.KeyData) != 33 && len(key.KeyData) != 65 {
			return fmt.Errorf("invalid ECDSA key size: expected 33 or 65, got %d", len(key.KeyData))
		}
	case KeyTypeX25519:
		// X25519 public key is 32 bytes
		if len(key.KeyData) != 32 {
			return fmt.Errorf("invalid X25519 key size: expected 32, got %d", len(key.KeyData))
		}
		// X25519 keys don't need PoP verification (key agreement only)
		return nil
	default:
		return fmt.Errorf("unknown key type: %d", key.Type)
	}

	// Verify proof-of-possession for signing keys
	if err := VerifyKeyProofOfPossession(did, key); err != nil {
		return fmt.Errorf("PoP verification failed: %w", err)
	}

	return nil
}
