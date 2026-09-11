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
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"

	"github.com/sage-x-project/sage/pkg/agent/crypto/jcs"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

// A2AProof represents a cryptographic proof for an A2A Agent Card
// Following W3C Verifiable Credentials Data Model 1.1 proof format
// https://www.w3.org/TR/vc-data-model/#proofs-signatures
type A2AProof struct {
	Type               string    `json:"type"`               // Signature type (e.g., "Ed25519Signature2020")
	Created            time.Time `json:"created"`            // When the proof was created
	VerificationMethod string    `json:"verificationMethod"` // Key ID used for signing
	ProofPurpose       string    `json:"proofPurpose"`       // Purpose (e.g., "assertionMethod")
	ProofValue         string    `json:"proofValue"`         // Base58-encoded signature
}

// A2AAgentCardWithProof extends A2AAgentCard with cryptographic proof
type A2AAgentCardWithProof struct {
	A2AAgentCard
	Proof *A2AProof `json:"proof,omitempty"` // Cryptographic proof

	// raw holds the JSON the card was parsed from (see
	// ParseA2AAgentCardWithProof). When present, the proof is verified over
	// the canonical form of these bytes rather than over a re-encoding of the
	// Go struct, so fields the struct does not model are still covered.
	raw json.RawMessage
}

// ParseA2AAgentCardWithProof decodes a signed card and retains the original
// JSON so that VerifyA2ACardProof verifies exactly what was received.
func ParseA2AAgentCardWithProof(data []byte) (*A2AAgentCardWithProof, error) {
	var card A2AAgentCardWithProof
	if err := json.Unmarshal(data, &card); err != nil {
		return nil, fmt.Errorf("invalid A2A card JSON: %w", err)
	}
	card.raw = append(json.RawMessage(nil), data...)
	return &card, nil
}

// a2aCardCanonicalBytes returns the RFC 8785 canonical JSON of the card
// without its "proof" member. This is the byte string A2A card proofs are
// computed over: Ed25519 signs it directly; secp256k1 signs Keccak-256 of it
// (Ethereum convention, see keys.SignSecp256k1Keccak).
func a2aCardCanonicalBytes(card *A2AAgentCardWithProof) ([]byte, error) {
	if len(card.raw) > 0 {
		var m map[string]interface{}
		dec := json.NewDecoder(bytes.NewReader(card.raw))
		dec.UseNumber()
		if err := dec.Decode(&m); err != nil {
			return nil, fmt.Errorf("failed to decode card: %w", err)
		}
		delete(m, "proof")
		return jcs.Marshal(m)
	}
	return jcs.Marshal(card.A2AAgentCard)
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

	// Canonical representation for signing (RFC 8785, without proof)
	canonical, err := jcs.Marshal(baseCard)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize card: %w", err)
	}

	// Sign based on key type
	var signature []byte
	var proofType string

	switch keyType {
	case KeyTypeEd25519:
		ed25519Key, ok := privateKey.(ed25519.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("invalid Ed25519 private key type")
		}
		signature = ed25519.Sign(ed25519Key, canonical)
		proofType = "Ed25519Signature2020"

	case KeyTypeECDSA:
		ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("invalid ECDSA private key type")
		}
		signature, err = keys.SignSecp256k1Keccak(ecdsaKey, canonical)
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
// against the key listed in the card itself.
//
// This is a self-attestation check: it proves that whoever produced the card
// held the private key for the listed verification method, but it does not
// prove that the key belongs to the DID named in the card. A forger can list
// their own key and sign. Use VerifyA2ACardProofWithDID to bind the proof to
// the on-chain DID document.
//
// Returns:
//   - true if the proof is valid
//   - false and error if verification fails
func VerifyA2ACardProof(cardWithProof *A2AAgentCardWithProof) (bool, error) {
	if cardWithProof == nil || cardWithProof.Proof == nil {
		return false, fmt.Errorf("card has no proof")
	}
	verificationKey := findA2APublicKey(&cardWithProof.A2AAgentCard, cardWithProof.Proof.VerificationMethod)
	if verificationKey == nil {
		return false, fmt.Errorf("verification key not found in card: %s", cardWithProof.Proof.VerificationMethod)
	}
	keyBytes, err := decodeA2APublicKey(verificationKey)
	if err != nil {
		return false, err
	}
	if err := verifyA2ACardProofWithKey(cardWithProof, keyBytes); err != nil {
		return false, err
	}
	return true, nil
}

// VerifyA2ACardProofWithDID verifies the proof of an A2A Agent Card against
// the on-chain DID document of the DID named in the card.
//
// The verification method must be a key identifier under the card's DID, the
// key it names must be present in the card, and the same key bytes must be
// listed as a verified key in the on-chain document. The signature is then
// checked with the on-chain key bytes, so a card that lists a key the chain
// does not know (or knows but has not verified) is rejected even when its
// signature is internally consistent.
func VerifyA2ACardProofWithDID(ctx context.Context, cardWithProof *A2AAgentCardWithProof, resolver Resolver) error {
	if cardWithProof == nil || cardWithProof.Proof == nil {
		return fmt.Errorf("card has no proof")
	}
	if resolver == nil {
		return fmt.Errorf("resolver cannot be nil")
	}
	proof := cardWithProof.Proof
	agentDID := AgentDID(cardWithProof.ID)
	if _, _, err := ParseDID(agentDID); err != nil {
		return fmt.Errorf("invalid DID in card: %w", err)
	}
	if !strings.HasPrefix(proof.VerificationMethod, cardWithProof.ID+"#") {
		return fmt.Errorf("verification method %s does not belong to DID %s", proof.VerificationMethod, cardWithProof.ID)
	}

	verificationKey := findA2APublicKey(&cardWithProof.A2AAgentCard, proof.VerificationMethod)
	if verificationKey == nil {
		return fmt.Errorf("verification key not found in card: %s", proof.VerificationMethod)
	}
	cardKeyBytes, err := decodeA2APublicKey(verificationKey)
	if err != nil {
		return err
	}

	metadata, err := resolver.Resolve(ctx, agentDID)
	if err != nil {
		return fmt.Errorf("failed to resolve DID from blockchain: %w", err)
	}
	if !metadata.IsActive {
		return fmt.Errorf("DID %s is not active on-chain", agentDID)
	}
	var onChainKey *AgentKey
	for _, key := range FromAgentMetadata(metadata).Keys {
		if key.Verified && bytes.Equal(key.KeyData, cardKeyBytes) {
			k := key
			onChainKey = &k
			break
		}
	}
	if onChainKey == nil {
		return fmt.Errorf("verification key %s is not a verified key of %s on-chain", proof.VerificationMethod, agentDID)
	}
	return verifyA2ACardProofWithKey(cardWithProof, onChainKey.KeyData)
}

// findA2APublicKey returns the card key with the given identifier, or nil.
func findA2APublicKey(card *A2AAgentCard, keyID string) *A2APublicKey {
	for i := range card.PublicKeys {
		if card.PublicKeys[i].ID == keyID {
			return &card.PublicKeys[i]
		}
	}
	return nil
}

// decodeA2APublicKey returns the raw key bytes of a card key, preferring the
// Base58 field and falling back to the hex field.
func decodeA2APublicKey(key *A2APublicKey) ([]byte, error) {
	switch {
	case key.PublicKeyBase58 != "":
		b, err := base58.Decode(key.PublicKeyBase58)
		if err != nil {
			return nil, fmt.Errorf("invalid key data in card for key %s: %w", key.ID, err)
		}
		return b, nil
	case key.PublicKeyHex != "":
		b, err := hex.DecodeString(strings.TrimPrefix(key.PublicKeyHex, "0x"))
		if err != nil {
			return nil, fmt.Errorf("invalid hex key data in card for key %s: %w", key.ID, err)
		}
		return b, nil
	default:
		return nil, fmt.Errorf("key %s has no public key data", key.ID)
	}
}

// verifyA2ACardProofWithKey checks the proof signature over the card (without
// the proof) using the given raw public key bytes.
func verifyA2ACardProofWithKey(cardWithProof *A2AAgentCardWithProof, pubKeyBytes []byte) error {
	proof := cardWithProof.Proof

	signature, err := base58.Decode(proof.ProofValue)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Canonical representation (RFC 8785, without proof) for verification
	canonical, err := a2aCardCanonicalBytes(cardWithProof)
	if err != nil {
		return err
	}

	switch proof.Type {
	case "Ed25519Signature2020":
		if len(pubKeyBytes) != ed25519.PublicKeySize {
			return fmt.Errorf("invalid Ed25519 public key size: %d", len(pubKeyBytes))
		}
		if !ed25519.Verify(ed25519.PublicKey(pubKeyBytes), canonical, signature) {
			return fmt.Errorf("Ed25519 signature verification failed")
		}
		return nil

	case "EcdsaSecp256k1Signature2019":
		// Accept compressed (33), raw x||y (64) and uncompressed (65) encodings.
		if len(pubKeyBytes) == 64 {
			pubKeyBytes = append([]byte{0x04}, pubKeyBytes...)
		}
		var pubKey *ecdsa.PublicKey
		switch len(pubKeyBytes) {
		case 33:
			pubKey, err = ethcrypto.DecompressPubkey(pubKeyBytes)
			if err != nil {
				return fmt.Errorf("failed to decompress public key: %w", err)
			}
		case 65:
			pubKey, err = ethcrypto.UnmarshalPubkey(pubKeyBytes)
			if err != nil {
				return fmt.Errorf("failed to unmarshal public key: %w", err)
			}
		default:
			return fmt.Errorf("invalid public key length: %d (expected 33, 64, or 65 bytes)", len(pubKeyBytes))
		}
		// Ethereum convention: Keccak-256 of the canonical card, r || s [|| v]
		if err := keys.VerifySecp256k1Keccak(pubKey, canonical, signature); err != nil {
			return fmt.Errorf("ECDSA signature verification failed: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported proof type: %s", proof.Type)
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

// ValidateA2ACardWithProofAndDID performs the complete validation of a signed
// A2A Agent Card: field validation, proof verification bound to the on-chain
// DID document (VerifyA2ACardProofWithDID) and cross-validation of every card
// key and the primary endpoint against the chain (ValidateA2ACardWithDID).
func ValidateA2ACardWithProofAndDID(ctx context.Context, cardWithProof *A2AAgentCardWithProof, resolver Resolver) error {
	if cardWithProof == nil {
		return fmt.Errorf("card cannot be nil")
	}
	if err := ValidateA2ACard(&cardWithProof.A2AAgentCard); err != nil {
		return fmt.Errorf("basic validation failed: %w", err)
	}
	if err := VerifyA2ACardProofWithDID(ctx, cardWithProof, resolver); err != nil {
		return fmt.Errorf("proof verification failed: %w", err)
	}
	return ValidateA2ACardWithDID(ctx, &cardWithProof.A2AAgentCard, resolver)
}
