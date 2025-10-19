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
	"encoding/json"
	"fmt"
	"time"

	"github.com/mr-tron/base58"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// A2AProof represents a cryptographic proof for an A2A Agent Card
// Following W3C Verifiable Credentials Data Model 1.1 proof format
// https://www.w3.org/TR/vc-data-model/#proofs-signatures
type A2AProof struct {
	Type               string    `json:"type"`                      // Signature type (e.g., "Ed25519Signature2020")
	Created            time.Time `json:"created"`                   // When the proof was created
	VerificationMethod string    `json:"verificationMethod"`        // Key ID used for signing
	ProofPurpose       string    `json:"proofPurpose"`              // Purpose (e.g., "assertionMethod")
	ProofValue         string    `json:"proofValue"`                // Base58-encoded signature
}

// A2AAgentCardWithProof extends A2AAgentCard with cryptographic proof
type A2AAgentCardWithProof struct {
	A2AAgentCard
	Proof *A2AProof `json:"proof,omitempty"` // Cryptographic proof
}

// GenerateA2ACardWithProof creates an A2A Agent Card with cryptographic proof
//
// The card is signed using the first verified key from the metadata, proving
// that the card was created by the legitimate DID owner.
//
// Parameters:
//   - metadata: Agent metadata containing keys
//   - privateKey: Private key corresponding to one of the agent's public keys
//   - keyType: Type of the signing key (Ed25519, ECDSA, etc.)
//
// Returns:
//   - Signed A2A Agent Card with proof
//   - Error if signing fails
func GenerateA2ACardWithProof(metadata *AgentMetadataV4, privateKey interface{}, keyType KeyType) (*A2AAgentCardWithProof, error) {
	// Generate base card
	baseCard, err := GenerateA2ACard(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to generate base card: %w", err)
	}

	// Find the corresponding public key
	var keyIndex int
	var signingKey *AgentKey
	for i, key := range metadata.Keys {
		if key.Type == keyType && key.Verified {
			signingKey = &metadata.Keys[i]
			keyIndex = i
			break
		}
	}

	if signingKey == nil {
		return nil, fmt.Errorf("no verified %s key found in metadata", keyType)
	}

	// Create canonical representation for signing (without proof)
	cardJSON, err := json.Marshal(baseCard)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal card: %w", err)
	}

	// Hash the card data
	hash := sha256.Sum256(cardJSON)

	// Sign based on key type
	var signature []byte
	var proofType string

	switch keyType {
	case KeyTypeEd25519:
		ed25519Key, ok := privateKey.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("invalid Ed25519 private key type")
		}
		signature = ed25519.Sign(ed25519Key, hash[:])
		proofType = "Ed25519Signature2020"

	case KeyTypeECDSA:
		ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("invalid ECDSA private key type")
		}
		signature, err = ethcrypto.Sign(hash[:], ecdsaKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign with ECDSA: %w", err)
		}
		proofType = "EcdsaSecp256k1Signature2019"

	default:
		return nil, fmt.Errorf("unsupported key type for signing: %s", keyType)
	}

	// Create proof
	keyID := fmt.Sprintf("%s#key-%d", metadata.DID, keyIndex+1)
	proof := &A2AProof{
		Type:               proofType,
		Created:            time.Now().UTC(),
		VerificationMethod: keyID,
		ProofPurpose:       "assertionMethod",
		ProofValue:         base58.Encode(signature),
	}

	// Return card with proof
	cardWithProof := &A2AAgentCardWithProof{
		A2AAgentCard: *baseCard,
		Proof:        proof,
	}

	return cardWithProof, nil
}

// VerifyA2ACardProof verifies the cryptographic proof of an A2A Agent Card
//
// This function:
//  1. Extracts the signature from the proof
//  2. Finds the verification key from the card's public keys
//  3. Verifies the signature against the card data
//
// Parameters:
//   - cardWithProof: A2A Agent Card with proof to verify
//
// Returns:
//   - true if the proof is valid
//   - false and error if verification fails
func VerifyA2ACardProof(cardWithProof *A2AAgentCardWithProof) (bool, error) {
	if cardWithProof.Proof == nil {
		return false, fmt.Errorf("card has no proof")
	}

	proof := cardWithProof.Proof

	// Find the verification key in the card
	var verificationKey *A2APublicKey
	for i := range cardWithProof.PublicKeys {
		if cardWithProof.PublicKeys[i].ID == proof.VerificationMethod {
			verificationKey = &cardWithProof.PublicKeys[i]
			break
		}
	}

	if verificationKey == nil {
		return false, fmt.Errorf("verification key not found in card: %s", proof.VerificationMethod)
	}

	// Decode signature
	signature, err := base58.Decode(proof.ProofValue)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Create canonical representation (without proof) for verification
	baseCard := cardWithProof.A2AAgentCard
	cardJSON, err := json.Marshal(baseCard)
	if err != nil {
		return false, fmt.Errorf("failed to marshal card: %w", err)
	}

	// Hash the card data
	hash := sha256.Sum256(cardJSON)

	// Verify based on proof type
	switch proof.Type {
	case "Ed25519Signature2020":
		// Decode Ed25519 public key
		pubKeyBytes, err := base58.Decode(verificationKey.PublicKeyBase58)
		if err != nil {
			return false, fmt.Errorf("failed to decode Ed25519 public key: %w", err)
		}

		if len(pubKeyBytes) != ed25519.PublicKeySize {
			return false, fmt.Errorf("invalid Ed25519 public key size: %d", len(pubKeyBytes))
		}

		pubKey := ed25519.PublicKey(pubKeyBytes)
		valid := ed25519.Verify(pubKey, hash[:], signature)
		if !valid {
			return false, fmt.Errorf("Ed25519 signature verification failed")
		}
		return true, nil

	case "EcdsaSecp256k1Signature2019":
		// Decode ECDSA public key
		var pubKeyBytes []byte
		if verificationKey.PublicKeyBase58 != "" {
			pubKeyBytes, err = base58.Decode(verificationKey.PublicKeyBase58)
		} else if verificationKey.PublicKeyHex != "" {
			pubKeyBytes, err = base58.Decode(verificationKey.PublicKeyHex)
		} else {
			return false, fmt.Errorf("no public key data in verification key")
		}
		if err != nil {
			return false, fmt.Errorf("failed to decode ECDSA public key: %w", err)
		}

		// Decompress public key
		pubKey, err := ethcrypto.DecompressPubkey(pubKeyBytes)
		if err != nil {
			return false, fmt.Errorf("failed to decompress public key: %w", err)
		}

		// Verify signature (Ethereum signature includes recovery ID in last byte)
		if len(signature) == 65 {
			signature = signature[:64] // Remove recovery ID
		}

		valid := ethcrypto.VerifySignature(ethcrypto.CompressPubkey(pubKey), hash[:], signature)
		if !valid {
			return false, fmt.Errorf("ECDSA signature verification failed")
		}
		return true, nil

	default:
		return false, fmt.Errorf("unsupported proof type: %s", proof.Type)
	}
}

// ValidateA2ACardWithProof performs comprehensive validation of an A2A Agent Card with proof
//
// This function combines:
//  1. Basic field validation (ValidateA2ACard)
//  2. Cryptographic proof verification
//
// Returns:
//   - Error if any validation fails
//   - nil if all validations pass
func ValidateA2ACardWithProof(cardWithProof *A2AAgentCardWithProof) error {
	// Basic field validation
	if err := ValidateA2ACard(&cardWithProof.A2AAgentCard); err != nil {
		return fmt.Errorf("basic validation failed: %w", err)
	}

	// Verify cryptographic proof
	valid, err := VerifyA2ACardProof(cardWithProof)
	if err != nil {
		return fmt.Errorf("proof verification failed: %w", err)
	}
	if !valid {
		return fmt.Errorf("proof verification returned false")
	}

	return nil
}
