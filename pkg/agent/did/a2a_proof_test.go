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

func TestGenerateA2ACardWithProof_Ed25519(t *testing.T) {
	// Generate Ed25519 key pair
	keyPair, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	pubKey, ok := keyPair.PublicKey().(ed25519.PublicKey)
	require.True(t, ok)

	privKey, ok := keyPair.PrivateKey().(ed25519.PrivateKey)
	require.True(t, ok)

	// Create metadata with Ed25519 key
	metadata := &AgentMetadataV4{
		DID:         "did:sage:ethereum:0x1234567890abcdef",
		Name:        "Test Agent",
		Description: "Test agent with Ed25519 key",
		Endpoint:    "https://test.agent.com",
		Keys: []AgentKey{
			{
				Type:      KeyTypeEd25519,
				KeyData:   pubKey,
				Verified:  true,
				CreatedAt: time.Now(),
			},
		},
		Capabilities: map[string]interface{}{
			"test": true,
		},
		Owner:     "0x1234567890abcdef",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Generate card with proof
	cardWithProof, err := GenerateA2ACardWithProof(metadata, privKey, KeyTypeEd25519)
	require.NoError(t, err)
	require.NotNil(t, cardWithProof)
	require.NotNil(t, cardWithProof.Proof)

	// Verify proof structure
	assert.Equal(t, "Ed25519Signature2020", cardWithProof.Proof.Type)
	assert.Equal(t, "assertionMethod", cardWithProof.Proof.ProofPurpose)
	assert.NotEmpty(t, cardWithProof.Proof.ProofValue)
	assert.NotZero(t, cardWithProof.Proof.Created)

	// Verify the proof
	valid, err := VerifyA2ACardProof(cardWithProof)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestGenerateA2ACardWithProof_ECDSA(t *testing.T) {
	// Generate ECDSA key pair
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	pubKeyBytes, err := MarshalPublicKey(keyPair.PublicKey())
	require.NoError(t, err)

	// Create metadata with ECDSA key
	metadata := &AgentMetadataV4{
		DID:         "did:sage:ethereum:0x1234567890abcdef",
		Name:        "Test Agent",
		Description: "Test agent with ECDSA key",
		Endpoint:    "https://test.agent.com",
		Keys: []AgentKey{
			{
				Type:      KeyTypeECDSA,
				KeyData:   pubKeyBytes,
				Verified:  true,
				CreatedAt: time.Now(),
			},
		},
		Capabilities: map[string]interface{}{
			"test": true,
		},
		Owner:     "0x1234567890abcdef",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Generate card with proof
	cardWithProof, err := GenerateA2ACardWithProof(metadata, keyPair.PrivateKey(), KeyTypeECDSA)
	require.NoError(t, err)
	require.NotNil(t, cardWithProof)
	require.NotNil(t, cardWithProof.Proof)

	// Verify proof structure
	assert.Equal(t, "EcdsaSecp256k1Signature2019", cardWithProof.Proof.Type)
	assert.Equal(t, "assertionMethod", cardWithProof.Proof.ProofPurpose)
	assert.NotEmpty(t, cardWithProof.Proof.ProofValue)

	// Verify the proof
	valid, err := VerifyA2ACardProof(cardWithProof)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestVerifyA2ACardProof_NoProof(t *testing.T) {
	cardWithProof := &A2AAgentCardWithProof{
		A2AAgentCard: A2AAgentCard{
			ID:   "did:sage:ethereum:0x1234",
			Name: "Test",
		},
		Proof: nil,
	}

	valid, err := VerifyA2ACardProof(cardWithProof)
	assert.Error(t, err)
	assert.False(t, valid)
	assert.Contains(t, err.Error(), "no proof")
}

func TestVerifyA2ACardProof_InvalidSignature(t *testing.T) {
	// Generate two different key pairs
	keyPair1, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	keyPair2, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	pubKey1 := keyPair1.PublicKey().(ed25519.PublicKey)
	privKey2 := keyPair2.PrivateKey().(ed25519.PrivateKey)

	// Create metadata with key1 public key
	metadata := &AgentMetadataV4{
		DID:         "did:sage:ethereum:0x1234567890abcdef",
		Name:        "Test Agent",
		Description: "Test agent",
		Endpoint:    "https://test.agent.com",
		Keys: []AgentKey{
			{
				Type:      KeyTypeEd25519,
				KeyData:   pubKey1,
				Verified:  true,
				CreatedAt: time.Now(),
			},
		},
		Owner:     "0x1234567890abcdef",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Sign with key2 private key (mismatched)
	cardWithProof, err := GenerateA2ACardWithProof(metadata, privKey2, KeyTypeEd25519)
	require.NoError(t, err) // Generation succeeds

	// But verification should fail (wrong key)
	valid, err := VerifyA2ACardProof(cardWithProof)
	assert.Error(t, err)
	assert.False(t, valid)
}

func TestValidateA2ACardWithProof(t *testing.T) {
	// Generate key pair
	keyPair, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)

	pubKey := keyPair.PublicKey().(ed25519.PublicKey)
	privKey := keyPair.PrivateKey().(ed25519.PrivateKey)

	// Create valid metadata
	metadata := &AgentMetadataV4{
		DID:         "did:sage:ethereum:0x1234567890abcdef",
		Name:        "Test Agent",
		Description: "Test agent",
		Endpoint:    "https://test.agent.com",
		Keys: []AgentKey{
			{
				Type:      KeyTypeEd25519,
				KeyData:   pubKey,
				Verified:  true,
				CreatedAt: time.Now(),
			},
		},
		Capabilities: map[string]interface{}{
			"test": true,
		},
		Owner:     "0x1234567890abcdef",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Generate card with proof
	cardWithProof, err := GenerateA2ACardWithProof(metadata, privKey, KeyTypeEd25519)
	require.NoError(t, err)

	// Validate (should pass all checks)
	err = ValidateA2ACardWithProof(cardWithProof)
	assert.NoError(t, err)
}

func TestGenerateA2ACardWithProof_NoVerifiedKey(t *testing.T) {
	// Create metadata with unverified key
	metadata := &AgentMetadataV4{
		DID:      "did:sage:ethereum:0x1234567890abcdef",
		Name:     "Test Agent",
		Endpoint: "https://test.agent.com",
		Keys: []AgentKey{
			{
				Type:      KeyTypeEd25519,
				KeyData:   make([]byte, 32),
				Verified:  false, // Not verified
				CreatedAt: time.Now(),
			},
		},
		Owner:     "0x1234567890abcdef",
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Should fail - no verified key of requested type
	_, err := GenerateA2ACardWithProof(metadata, ed25519.PrivateKey(make([]byte, 64)), KeyTypeEd25519)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no verified")
}
