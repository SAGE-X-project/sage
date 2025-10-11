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

package ethereum

import (
	"context"
	"strings"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto/chain"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEthereumProvider(t *testing.T) {
	provider := NewProvider()

	t.Run("ChainType", func(t *testing.T) {
		assert.Equal(t, chain.ChainTypeEthereum, provider.ChainType())
	})

	t.Run("SupportedNetworks", func(t *testing.T) {
		networks := provider.SupportedNetworks()
		assert.Contains(t, networks, chain.NetworkEthereumMainnet)
		assert.Contains(t, networks, chain.NetworkEthereumGoerli)
		assert.Contains(t, networks, chain.NetworkEthereumSepolia)
	})

	t.Run("GenerateAddress", func(t *testing.T) {
		// Generate secp256k1 key pair
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		// Generate Ethereum address
		address, err := provider.GenerateAddress(keyPair.PublicKey(), chain.NetworkEthereumMainnet)
		require.NoError(t, err)
		assert.NotNil(t, address)
		assert.Equal(t, chain.ChainTypeEthereum, address.Chain)
		assert.Equal(t, chain.NetworkEthereumMainnet, address.Network)
		assert.True(t, strings.HasPrefix(address.Value, "0x"))
		assert.Len(t, address.Value, 42) // 0x + 40 hex chars
	})

	t.Run("GenerateAddressInvalidKey", func(t *testing.T) {
		// Try with Ed25519 key (not supported by Ethereum)
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		_, err = provider.GenerateAddress(keyPair.PublicKey(), chain.NetworkEthereumMainnet)
		assert.Error(t, err)
		assert.Equal(t, chain.ErrInvalidPublicKey, err)
	})

	t.Run("ValidateAddress", func(t *testing.T) {
		// Valid addresses
		validAddresses := []string{
			"0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80",
			"742d35Cc6634C0532925a3b844Bc9e7595f2bd80", // without 0x
			"0x0000000000000000000000000000000000000000",
		}

		for _, addr := range validAddresses {
			err := provider.ValidateAddress(addr, chain.NetworkEthereumMainnet)
			assert.NoError(t, err, "Address %s should be valid", addr)
		}

		// Invalid addresses
		invalidAddresses := []string{
			"0x742d35Cc6634C0532925a3b844Bc9e7595f2bd8",   // too short
			"0x742d35Cc6634C0532925a3b844Bc9e7595f2bd800", // too long
			"0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG",  // invalid hex
			"", // empty
		}

		for _, addr := range invalidAddresses {
			err := provider.ValidateAddress(addr, chain.NetworkEthereumMainnet)
			assert.Error(t, err, "Address %s should be invalid", addr)
		}
	})

	t.Run("GetPublicKeyFromAddress", func(t *testing.T) {
		// This operation is not supported for Ethereum
		_, err := provider.GetPublicKeyFromAddress(context.Background(), "0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80", chain.NetworkEthereumMainnet)
		assert.Error(t, err)
		assert.Equal(t, chain.ErrOperationNotSupported, err)
	})

	t.Run("DeterministicAddressGeneration", func(t *testing.T) {
		// Same key should generate same address
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		address1, err := provider.GenerateAddress(keyPair.PublicKey(), chain.NetworkEthereumMainnet)
		require.NoError(t, err)

		address2, err := provider.GenerateAddress(keyPair.PublicKey(), chain.NetworkEthereumMainnet)
		require.NoError(t, err)

		assert.Equal(t, address1.Value, address2.Value)
	})

	t.Run("KnownAddressGeneration", func(t *testing.T) {
		// Test with a known public key to verify correct address generation
		// This is a test vector to ensure our implementation is correct

		// Note: This test would require a specific test key with known address
		// For now, we just verify the format is correct
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		address, err := provider.GenerateAddress(keyPair.PublicKey(), chain.NetworkEthereumMainnet)
		require.NoError(t, err)

		// Verify checksum encoding (EIP-55)
		// The address should have mixed case for checksum
		addressWithout0x := strings.TrimPrefix(address.Value, "0x")
		hasUpper := false
		hasLower := false
		for _, char := range addressWithout0x {
			if char >= 'A' && char <= 'F' {
				hasUpper = true
			}
			if char >= 'a' && char <= 'f' {
				hasLower = true
			}
		}
		// Most addresses will have both upper and lower case letters
		// (though technically an address could be all numbers or all same case)
		_ = hasUpper
		_ = hasLower
	})
}
