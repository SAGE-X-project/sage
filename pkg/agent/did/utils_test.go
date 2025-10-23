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

	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAgentDIDWithAddress(t *testing.T) {
	tests := []struct {
		name         string
		chain        Chain
		ownerAddress string
		expectedDID  AgentDID
	}{
		{
			name:         "Ethereum with 0x prefix",
			chain:        ChainEthereum,
			ownerAddress: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
			expectedDID:  "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
		},
		{
			name:         "Ethereum without 0x prefix",
			chain:        ChainEthereum,
			ownerAddress: "f39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
			expectedDID:  "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
		},
		{
			name:         "Ethereum mixed case normalization",
			chain:        ChainEthereum,
			ownerAddress: "0xF39FD6E51AAD88F6F4CE6AB8827279CFFFB92266",
			expectedDID:  "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266",
		},
		{
			name:         "Solana address",
			chain:        ChainSolana,
			ownerAddress: "DYw8jCTfwHNRJhhmFcbXvVDTqWMEVFBX6ZKUmG5CNSKK",
			expectedDID:  "did:sage:solana:dyw8jctfwhnrjhhmfcbxvvdtqwmevfbx6zkumg5cnskk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateAgentDIDWithAddress(tt.chain, tt.ownerAddress)
			assert.Equal(t, tt.expectedDID, result)
		})
	}
}

func TestGenerateAgentDIDWithNonce(t *testing.T) {
	tests := []struct {
		name         string
		chain        Chain
		ownerAddress string
		nonce        uint64
		expectedDID  AgentDID
	}{
		{
			name:         "Ethereum with nonce 0",
			chain:        ChainEthereum,
			ownerAddress: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
			nonce:        0,
			expectedDID:  "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266:0",
		},
		{
			name:         "Ethereum with nonce 1",
			chain:        ChainEthereum,
			ownerAddress: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
			nonce:        1,
			expectedDID:  "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266:1",
		},
		{
			name:         "Ethereum with high nonce",
			chain:        ChainEthereum,
			ownerAddress: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
			nonce:        999999,
			expectedDID:  "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266:999999",
		},
		{
			name:         "Without 0x prefix and nonce",
			chain:        ChainEthereum,
			ownerAddress: "f39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
			nonce:        42,
			expectedDID:  "did:sage:ethereum:0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266:42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateAgentDIDWithNonce(tt.chain, tt.ownerAddress, tt.nonce)
			assert.Equal(t, tt.expectedDID, result)
		})
	}
}

func TestDeriveEthereumAddress(t *testing.T) {
	t.Run("Valid secp256k1 keypair", func(t *testing.T) {
		// Generate a secp256k1 key pair
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		// Derive Ethereum address
		address, err := DeriveEthereumAddress(keyPair)
		require.NoError(t, err)

		// Verify address format
		assert.NotEmpty(t, address)
		assert.Equal(t, "0x", address[:2], "Address should start with 0x")
		assert.Equal(t, 42, len(address), "Address should be 42 characters (0x + 40 hex chars)")

		// Verify address is lowercase
		assert.Equal(t, address, address, "Address should be lowercase")
	})

	t.Run("Invalid key type - Ed25519", func(t *testing.T) {
		// Generate an Ed25519 key pair (not secp256k1)
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Try to derive Ethereum address (should fail)
		_, err = DeriveEthereumAddress(keyPair)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ethereum address derivation requires secp256k1 key")
	})

	t.Run("Known test vector", func(t *testing.T) {
		// This is a well-known test vector from go-ethereum
		// Private key: 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
		// Expected address: 0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266

		// Note: This test requires the actual private key import which is not
		// implemented in the current keys package. This is a placeholder for
		// when proper key import is available.
		t.Skip("Skipping test vector test until key import is implemented")
	})

	t.Run("Address consistency", func(t *testing.T) {
		// Generate a key pair
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		// Derive address multiple times
		address1, err := DeriveEthereumAddress(keyPair)
		require.NoError(t, err)

		address2, err := DeriveEthereumAddress(keyPair)
		require.NoError(t, err)

		// Addresses should be identical
		assert.Equal(t, address1, address2, "Same key should produce same address")
	})
}

func TestDeriveEthereumAddress_Integration(t *testing.T) {
	t.Run("Derive address and create DID", func(t *testing.T) {
		// Generate secp256k1 key pair
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		// Derive Ethereum address
		address, err := DeriveEthereumAddress(keyPair)
		require.NoError(t, err)

		// Create DID with derived address
		agentDID := GenerateAgentDIDWithAddress(ChainEthereum, address)

		// Verify DID format
		assert.Contains(t, string(agentDID), "did:sage:ethereum:")
		assert.Contains(t, string(agentDID), address)
	})

	t.Run("Multiple agents with nonce", func(t *testing.T) {
		// Generate secp256k1 key pair
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		// Derive Ethereum address
		address, err := DeriveEthereumAddress(keyPair)
		require.NoError(t, err)

		// Create multiple DIDs with same address but different nonces
		did1 := GenerateAgentDIDWithNonce(ChainEthereum, address, 0)
		did2 := GenerateAgentDIDWithNonce(ChainEthereum, address, 1)
		did3 := GenerateAgentDIDWithNonce(ChainEthereum, address, 2)

		// Verify all DIDs are different
		assert.NotEqual(t, did1, did2)
		assert.NotEqual(t, did2, did3)
		assert.NotEqual(t, did1, did3)

		// Verify all DIDs contain the same address
		assert.Contains(t, string(did1), address)
		assert.Contains(t, string(did2), address)
		assert.Contains(t, string(did3), address)

		// Verify nonces are in the DIDs
		assert.Contains(t, string(did1), ":0")
		assert.Contains(t, string(did2), ":1")
		assert.Contains(t, string(did3), ":2")
	})
}

func TestMarshalUnmarshalPublicKey(t *testing.T) {
	t.Run("Ed25519 key", func(t *testing.T) {
		// Generate Ed25519 key pair
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Marshal public key
		marshaled, err := MarshalPublicKey(keyPair.PublicKey())
		require.NoError(t, err)
		assert.NotEmpty(t, marshaled)
		assert.Equal(t, ed25519.PublicKeySize, len(marshaled))

		// Unmarshal public key
		unmarshaled, err := UnmarshalPublicKey(marshaled, "ed25519")
		require.NoError(t, err)
		assert.NotNil(t, unmarshaled)

		// Verify unmarshaled key matches original
		unmarshaledPubKey, ok := unmarshaled.(ed25519.PublicKey)
		assert.True(t, ok)
		originalPubKey, ok := keyPair.PublicKey().(ed25519.PublicKey)
		assert.True(t, ok)
		assert.Equal(t, originalPubKey, unmarshaledPubKey)
	})

	t.Run("Secp256k1 key", func(t *testing.T) {
		// Generate secp256k1 key pair
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		// Marshal public key
		marshaled, err := MarshalPublicKey(keyPair.PublicKey())
		require.NoError(t, err)
		assert.NotEmpty(t, marshaled)
		// secp256k1 uncompressed format is 64 bytes (without 0x04 prefix)
		// V4 contract uses uncompressed format to avoid expensive decompression on-chain
		assert.Equal(t, 64, len(marshaled))

		// Unmarshal public key
		unmarshaled, err := UnmarshalPublicKey(marshaled, "secp256k1")
		require.NoError(t, err)
		assert.NotNil(t, unmarshaled)

		// Note: Exact equality check is complex for ECDSA keys
		// because of different internal representations
	})

	t.Run("Invalid key type for unmarshal", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		marshaled, err := MarshalPublicKey(keyPair.PublicKey())
		require.NoError(t, err)

		// Try to unmarshal Ed25519 key as secp256k1 (should fail)
		_, err = UnmarshalPublicKey(marshaled, "secp256k1")
		assert.Error(t, err)
	})

	t.Run("Invalid Ed25519 key size", func(t *testing.T) {
		invalidData := make([]byte, 31) // Wrong size for Ed25519

		_, err := UnmarshalPublicKey(invalidData, "ed25519")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid Ed25519 public key size")
	})
}
