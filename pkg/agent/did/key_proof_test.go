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
	"testing"
	"time"

	_ "github.com/sage-x-project/sage/internal/cryptoinit" // Initialize crypto wrappers
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKeyProofOfPossession_Ed25519(t *testing.T) {
	// Generate key pair
	keyPair, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	pubKey := keyPair.PublicKey().(ed25519.PublicKey)
	privKey := keyPair.PrivateKey().(ed25519.PrivateKey)

	did := AgentDID("did:sage:ethereum:0x1234567890abcdef")

	// Generate PoP
	signature, err := GenerateKeyProofOfPossession(did, pubKey, privKey, KeyTypeEd25519)
	require.NoError(t, err)
	assert.NotEmpty(t, signature)

	// Verify PoP
	key := &AgentKey{
		Type:      KeyTypeEd25519,
		KeyData:   pubKey,
		Signature: signature,
		Verified:  true,
		CreatedAt: time.Now(),
	}

	err = VerifyKeyProofOfPossession(did, key)
	assert.NoError(t, err)
}

func TestGenerateKeyProofOfPossession_ECDSA(t *testing.T) {
	// Generate key pair
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	pubKeyBytes, err := MarshalPublicKey(keyPair.PublicKey())
	require.NoError(t, err)

	did := AgentDID("did:sage:ethereum:0x1234567890abcdef")

	// Generate PoP
	signature, err := GenerateKeyProofOfPossession(did, pubKeyBytes, keyPair.PrivateKey(), KeyTypeECDSA)
	require.NoError(t, err)
	assert.NotEmpty(t, signature)

	// Verify PoP
	key := &AgentKey{
		Type:      KeyTypeECDSA,
		KeyData:   pubKeyBytes,
		Signature: signature,
		Verified:  true,
		CreatedAt: time.Now(),
	}

	err = VerifyKeyProofOfPossession(did, key)
	assert.NoError(t, err)
}

func TestGenerateKeyProofOfPossession_X25519(t *testing.T) {
	// X25519 keys don't support signing
	did := AgentDID("did:sage:ethereum:0x1234567890abcdef")
	keyData := make([]byte, 32)

	_, err := GenerateKeyProofOfPossession(did, keyData, nil, KeyTypeX25519)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key agreement only")
}

func TestVerifyKeyProofOfPossession_InvalidSignature(t *testing.T) {
	// Generate two different key pairs
	keyPair1, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	keyPair2, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	pubKey1 := keyPair1.PublicKey().(ed25519.PublicKey)
	privKey2 := keyPair2.PrivateKey().(ed25519.PrivateKey)

	did := AgentDID("did:sage:ethereum:0x1234567890abcdef")

	// Generate PoP with key2 private key
	signature, err := GenerateKeyProofOfPossession(did, pubKey1, privKey2, KeyTypeEd25519)
	require.NoError(t, err)

	// Try to verify with key1 public key (should fail)
	key := &AgentKey{
		Type:      KeyTypeEd25519,
		KeyData:   pubKey1,
		Signature: signature,
		Verified:  true,
		CreatedAt: time.Now(),
	}

	err = VerifyKeyProofOfPossession(did, key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verification failed")
}

func TestVerifyKeyProofOfPossession_NoSignature(t *testing.T) {
	keyPair, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	pubKey := keyPair.PublicKey().(ed25519.PublicKey)
	did := AgentDID("did:sage:ethereum:0x1234567890abcdef")

	// Key without signature
	key := &AgentKey{
		Type:      KeyTypeEd25519,
		KeyData:   pubKey,
		Signature: nil, // No signature
		Verified:  true,
		CreatedAt: time.Now(),
	}

	err = VerifyKeyProofOfPossession(did, key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no proof-of-possession")
}

func TestVerifyAllKeyProofs(t *testing.T) {
	// Generate key pairs
	ed25519KeyPair, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	ecdsaKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	ed25519PubKey := ed25519KeyPair.PublicKey().(ed25519.PublicKey)
	ed25519PrivKey := ed25519KeyPair.PrivateKey().(ed25519.PrivateKey)

	ecdsaPubKeyBytes, err := MarshalPublicKey(ecdsaKeyPair.PublicKey())
	require.NoError(t, err)

	did := AgentDID("did:sage:ethereum:0x1234567890abcdef")

	// Generate PoPs
	ed25519Sig, err := GenerateKeyProofOfPossession(did, ed25519PubKey, ed25519PrivKey, KeyTypeEd25519)
	require.NoError(t, err)

	ecdsaSig, err := GenerateKeyProofOfPossession(did, ecdsaPubKeyBytes, ecdsaKeyPair.PrivateKey(), KeyTypeECDSA)
	require.NoError(t, err)

	// Create metadata with multiple keys
	metadata := &AgentMetadataV4{
		DID:      did,
		Name:     "Test Agent",
		Endpoint: "https://test.agent.com",
		Keys: []AgentKey{
			{
				Type:      KeyTypeEd25519,
				KeyData:   ed25519PubKey,
				Signature: ed25519Sig,
				Verified:  true,
				CreatedAt: time.Now(),
			},
			{
				Type:      KeyTypeECDSA,
				KeyData:   ecdsaPubKeyBytes,
				Signature: ecdsaSig,
				Verified:  true,
				CreatedAt: time.Now(),
			},
			{
				Type:      KeyTypeX25519,
				KeyData:   make([]byte, 32),
				Signature: nil, // X25519 doesn't need PoP
				Verified:  true,
				CreatedAt: time.Now(),
			},
		},
		Owner:     "0x1234567890abcdef",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Verify all proofs
	err = VerifyAllKeyProofs(metadata)
	assert.NoError(t, err)
}

func TestVerifyAllKeyProofs_OneInvalid(t *testing.T) {
	// Generate two key pairs
	keyPair1, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	keyPair2, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	pubKey1 := keyPair1.PublicKey().(ed25519.PublicKey)
	privKey1 := keyPair1.PrivateKey().(ed25519.PrivateKey)
	pubKey2 := keyPair2.PublicKey().(ed25519.PublicKey)

	did := AgentDID("did:sage:ethereum:0x1234567890abcdef")

	// Generate valid PoP for key1
	sig1, err := GenerateKeyProofOfPossession(did, pubKey1, privKey1, KeyTypeEd25519)
	require.NoError(t, err)

	// Create metadata with one valid and one invalid key
	metadata := &AgentMetadataV4{
		DID:      did,
		Name:     "Test Agent",
		Endpoint: "https://test.agent.com",
		Keys: []AgentKey{
			{
				Type:      KeyTypeEd25519,
				KeyData:   pubKey1,
				Signature: sig1, // Valid signature
				Verified:  true,
				CreatedAt: time.Now(),
			},
			{
				Type:      KeyTypeEd25519,
				KeyData:   pubKey2,
				Signature: nil, // Missing signature
				Verified:  true,
				CreatedAt: time.Now(),
			},
		},
		Owner:     "0x1234567890abcdef",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Should fail - one key has no PoP
	err = VerifyAllKeyProofs(metadata)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key proof verification failed")
}

func TestValidateKeyWithPoP(t *testing.T) {
	// Generate key pair
	keyPair, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	pubKey := keyPair.PublicKey().(ed25519.PublicKey)
	privKey := keyPair.PrivateKey().(ed25519.PrivateKey)

	did := AgentDID("did:sage:ethereum:0x1234567890abcdef")

	// Generate PoP
	signature, err := GenerateKeyProofOfPossession(did, pubKey, privKey, KeyTypeEd25519)
	require.NoError(t, err)

	// Valid key
	key := &AgentKey{
		Type:      KeyTypeEd25519,
		KeyData:   pubKey,
		Signature: signature,
		Verified:  true,
		CreatedAt: time.Now(),
	}

	err = ValidateKeyWithPoP(did, key)
	assert.NoError(t, err)
}

func TestValidateKeyWithPoP_InvalidSize(t *testing.T) {
	did := AgentDID("did:sage:ethereum:0x1234567890abcdef")

	// Ed25519 key with wrong size
	key := &AgentKey{
		Type:      KeyTypeEd25519,
		KeyData:   make([]byte, 16), // Wrong size (should be 32)
		Signature: make([]byte, 64),
		Verified:  true,
		CreatedAt: time.Now(),
	}

	err := ValidateKeyWithPoP(did, key)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Ed25519 key size")
}

func TestValidateKeyWithPoP_X25519(t *testing.T) {
	did := AgentDID("did:sage:ethereum:0x1234567890abcdef")

	// X25519 key (no PoP required)
	key := &AgentKey{
		Type:      KeyTypeX25519,
		KeyData:   make([]byte, 32),
		Signature: nil, // X25519 doesn't need PoP
		Verified:  true,
		CreatedAt: time.Now(),
	}

	err := ValidateKeyWithPoP(did, key)
	assert.NoError(t, err) // Should pass without PoP verification
}
