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


package solana

import (
	"context"
	"crypto/ed25519"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto/chain"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSolanaProvider(t *testing.T) {
	provider := NewProvider()

	t.Run("ChainType", func(t *testing.T) {
		assert.Equal(t, chain.ChainTypeSolana, provider.ChainType())
	})

	t.Run("SupportedNetworks", func(t *testing.T) {
		networks := provider.SupportedNetworks()
		assert.Contains(t, networks, chain.NetworkSolanaMainnet)
		assert.Contains(t, networks, chain.NetworkSolanaDevnet)
		assert.Contains(t, networks, chain.NetworkSolanaTestnet)
	})

	t.Run("GenerateAddress", func(t *testing.T) {
		// Generate Ed25519 key pair
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Generate Solana address
		address, err := provider.GenerateAddress(keyPair.PublicKey(), chain.NetworkSolanaMainnet)
		require.NoError(t, err)
		assert.NotNil(t, address)
		assert.Equal(t, chain.ChainTypeSolana, address.Chain)
		assert.Equal(t, chain.NetworkSolanaMainnet, address.Network)
		assert.NotEmpty(t, address.Value)
	})

	t.Run("GenerateAddressInvalidKey", func(t *testing.T) {
		// Try with Secp256k1 key (not supported by Solana)
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		_, err = provider.GenerateAddress(keyPair.PublicKey(), chain.NetworkSolanaMainnet)
		assert.Error(t, err)
		assert.Equal(t, chain.ErrInvalidPublicKey, err)
	})

	t.Run("GetPublicKeyFromAddress", func(t *testing.T) {
		// Generate a key and address
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		address, err := provider.GenerateAddress(keyPair.PublicKey(), chain.NetworkSolanaMainnet)
		require.NoError(t, err)

		// Recover public key from address
		recoveredPubKey, err := provider.GetPublicKeyFromAddress(context.Background(), address.Value, chain.NetworkSolanaMainnet)
		require.NoError(t, err)

		// Compare with original
		originalPubKey := keyPair.PublicKey().(ed25519.PublicKey)
		recoveredEd25519 := recoveredPubKey.(ed25519.PublicKey)
		assert.Equal(t, originalPubKey, recoveredEd25519)
	})

	t.Run("ValidateAddress", func(t *testing.T) {
		// Generate valid address
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		address, err := provider.GenerateAddress(keyPair.PublicKey(), chain.NetworkSolanaMainnet)
		require.NoError(t, err)

		// Valid address should pass
		err = provider.ValidateAddress(address.Value, chain.NetworkSolanaMainnet)
		assert.NoError(t, err)

		// Invalid addresses
		invalidAddresses := []string{
			"",                          // empty
			"invalid-base58!@#",         // invalid characters
			"shortaddr",                 // too short
			"verylongaddressthatiswaytoolongtobevalidsolanaaddress", // wrong length after decode
		}

		for _, addr := range invalidAddresses {
			err := provider.ValidateAddress(addr, chain.NetworkSolanaMainnet)
			assert.Error(t, err, "Address %s should be invalid", addr)
		}
	})

	t.Run("DeterministicAddressGeneration", func(t *testing.T) {
		// Same key should generate same address
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		address1, err := provider.GenerateAddress(keyPair.PublicKey(), chain.NetworkSolanaMainnet)
		require.NoError(t, err)

		address2, err := provider.GenerateAddress(keyPair.PublicKey(), chain.NetworkSolanaMainnet)
		require.NoError(t, err)

		assert.Equal(t, address1.Value, address2.Value)
	})

	t.Run("VerifySignature", func(t *testing.T) {
		// Generate key pair
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Sign a message
		message := []byte("Test message for Solana")
		signature, err := keyPair.Sign(message)
		require.NoError(t, err)

		// Verify using provider
		err = provider.VerifySignature(keyPair.PublicKey(), message, signature)
		assert.NoError(t, err)

		// Verify with wrong message should fail
		wrongMessage := []byte("Wrong message")
		err = provider.VerifySignature(keyPair.PublicKey(), wrongMessage, signature)
		assert.Error(t, err)

		// Verify with wrong signature should fail
		wrongSignature := make([]byte, len(signature))
		copy(wrongSignature, signature)
		wrongSignature[0] ^= 0xFF
		err = provider.VerifySignature(keyPair.PublicKey(), message, wrongSignature)
		assert.Error(t, err)
	})

	t.Run("NetworkValidation", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Valid networks
		for _, network := range provider.SupportedNetworks() {
			_, err := provider.GenerateAddress(keyPair.PublicKey(), network)
			assert.NoError(t, err, "Network %s should be supported", network)
		}

		// Invalid network
		_, err = provider.GenerateAddress(keyPair.PublicKey(), chain.NetworkEthereumMainnet)
		assert.Error(t, err)
		assert.Equal(t, chain.ErrNetworkNotSupported, err)
	})
}
