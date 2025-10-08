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


package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlgorithmRegistry(t *testing.T) {
	t.Run("Get registered algorithm", func(t *testing.T) {
		// Ed25519 should be registered
		info, err := GetAlgorithmInfo(KeyTypeEd25519)
		require.NoError(t, err)
		assert.Equal(t, KeyTypeEd25519, info.KeyType)
		assert.NotEmpty(t, info.RFC9421Algorithm)
		assert.True(t, info.SupportsRFC9421)
		assert.True(t, info.SupportsKeyGeneration)
	})

	t.Run("Get unregistered algorithm", func(t *testing.T) {
		_, err := GetAlgorithmInfo(KeyType("unknown"))
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrAlgorithmNotSupported)
	})

	t.Run("List all supported algorithms", func(t *testing.T) {
		algorithms := ListSupportedAlgorithms()
		assert.NotEmpty(t, algorithms)

		// Should include at least Ed25519, Secp256k1, RSA
		var found []KeyType
		for _, alg := range algorithms {
			found = append(found, alg.KeyType)
		}

		assert.Contains(t, found, KeyTypeEd25519)
		assert.Contains(t, found, KeyTypeSecp256k1)
		assert.Contains(t, found, KeyTypeRSA)
	})

	t.Run("Get RFC 9421 algorithm name", func(t *testing.T) {
		tests := []struct {
			keyType  KeyType
			expected string
		}{
			{KeyTypeEd25519, "ed25519"},
			{KeyTypeSecp256k1, "es256k"},
			{KeyTypeRSA, "rsa-pss-sha256"},
		}

		for _, tt := range tests {
			t.Run(string(tt.keyType), func(t *testing.T) {
				algName, err := GetRFC9421AlgorithmName(tt.keyType)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, algName)
			})
		}
	})

	t.Run("Get key type from RFC 9421 algorithm", func(t *testing.T) {
		tests := []struct {
			rfc9421Alg string
			expected   KeyType
		}{
			{"ed25519", KeyTypeEd25519},
			{"es256k", KeyTypeSecp256k1},
			{"rsa-pss-sha256", KeyTypeRSA},
		}

		for _, tt := range tests {
			t.Run(tt.rfc9421Alg, func(t *testing.T) {
				keyType, err := GetKeyTypeFromRFC9421Algorithm(tt.rfc9421Alg)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, keyType)
			})
		}
	})

	t.Run("List RFC 9421 supported algorithms", func(t *testing.T) {
		algorithms := ListRFC9421SupportedAlgorithms()
		assert.NotEmpty(t, algorithms)

		// Should include RFC 9421 algorithm names
		assert.Contains(t, algorithms, "ed25519")
		assert.Contains(t, algorithms, "es256k")
		assert.Contains(t, algorithms, "rsa-pss-sha256")

		// X25519 should NOT be in RFC 9421 list (it's for key exchange, not signing)
		assert.NotContains(t, algorithms, "x25519")
	})

	t.Run("Check if algorithm supports RFC 9421", func(t *testing.T) {
		// Ed25519 supports RFC 9421
		assert.True(t, SupportsRFC9421(KeyTypeEd25519))

		// X25519 does NOT support RFC 9421 (key exchange only)
		assert.False(t, SupportsRFC9421(KeyTypeX25519))
	})

	t.Run("Check if algorithm supports key generation", func(t *testing.T) {
		// All registered algorithms should support key generation
		assert.True(t, SupportsKeyGeneration(KeyTypeEd25519))
		assert.True(t, SupportsKeyGeneration(KeyTypeSecp256k1))
		assert.True(t, SupportsKeyGeneration(KeyTypeRSA))
		assert.True(t, SupportsKeyGeneration(KeyTypeX25519))
	})

	t.Run("Check if algorithm supports signature", func(t *testing.T) {
		// Ed25519, Secp256k1, and RSA support signatures
		assert.True(t, SupportsSignature(KeyTypeEd25519))
		assert.True(t, SupportsSignature(KeyTypeSecp256k1))
		assert.True(t, SupportsSignature(KeyTypeRSA))

		// X25519 does NOT support signatures (key exchange only)
		assert.False(t, SupportsSignature(KeyTypeX25519))

		// Unknown key type should return false
		assert.False(t, SupportsSignature(KeyType("unknown")))
	})

	t.Run("Check if algorithm is supported", func(t *testing.T) {
		// Registered algorithms should return true
		assert.True(t, IsAlgorithmSupported(KeyTypeEd25519))
		assert.True(t, IsAlgorithmSupported(KeyTypeSecp256k1))
		assert.True(t, IsAlgorithmSupported(KeyTypeRSA))
		assert.True(t, IsAlgorithmSupported(KeyTypeX25519))

		// Unknown algorithm should return false
		assert.False(t, IsAlgorithmSupported(KeyType("unknown")))
	})

	t.Run("Validate algorithm capabilities", func(t *testing.T) {
		// Test that X25519 is registered but doesn't support RFC 9421
		info, err := GetAlgorithmInfo(KeyTypeX25519)
		require.NoError(t, err)
		assert.Equal(t, KeyTypeX25519, info.KeyType)
		assert.True(t, info.SupportsKeyGeneration)
		assert.False(t, info.SupportsRFC9421, "X25519 should not support RFC 9421")
		assert.Empty(t, info.RFC9421Algorithm, "X25519 should not have RFC 9421 algorithm")
	})
}

func TestAlgorithmRegistry_Immutability(t *testing.T) {
	t.Run("Returned slice should be immutable", func(t *testing.T) {
		algorithms1 := ListSupportedAlgorithms()
		originalLen := len(algorithms1)

		// Try to modify the returned slice
		algorithms1 = append(algorithms1, AlgorithmInfo{})

		// Get the list again
		algorithms2 := ListSupportedAlgorithms()

		// Original should be unchanged
		assert.Equal(t, originalLen, len(algorithms2))
	})

	t.Run("Returned RFC 9421 list should be immutable", func(t *testing.T) {
		list1 := ListRFC9421SupportedAlgorithms()
		originalLen := len(list1)

		// Try to modify
		list1 = append(list1, "fake-algorithm")

		// Get again
		list2 := ListRFC9421SupportedAlgorithms()

		// Original should be unchanged
		assert.Equal(t, originalLen, len(list2))
		assert.NotContains(t, list2, "fake-algorithm")
	})
}

func TestAlgorithmRegistry_ThreadSafety(t *testing.T) {
	t.Run("Concurrent reads should be safe", func(t *testing.T) {
		done := make(chan bool)

		// Spawn multiple goroutines reading from registry
		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				_, _ = GetAlgorithmInfo(KeyTypeEd25519)
				_ = ListSupportedAlgorithms()
				_ = ListRFC9421SupportedAlgorithms()
				_, _ = GetRFC9421AlgorithmName(KeyTypeSecp256k1)
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// If we get here without panic, test passes
	})
}

func TestAlgorithmRegistry_Integration(t *testing.T) {
	t.Run("All key types should be registered", func(t *testing.T) {
		keyTypes := []KeyType{
			KeyTypeEd25519,
			KeyTypeSecp256k1,
			KeyTypeX25519,
			KeyTypeRSA,
		}

		for _, kt := range keyTypes {
			t.Run(string(kt), func(t *testing.T) {
				info, err := GetAlgorithmInfo(kt)
				require.NoError(t, err, "Key type %s should be registered", kt)
				assert.Equal(t, kt, info.KeyType)
				assert.NotEmpty(t, info.Name)
				assert.NotEmpty(t, info.Description)
			})
		}
	})

	t.Run("RFC 9421 algorithms should map back to key types", func(t *testing.T) {
		rfc9421Algorithms := ListRFC9421SupportedAlgorithms()

		for _, algName := range rfc9421Algorithms {
			t.Run(algName, func(t *testing.T) {
				keyType, err := GetKeyTypeFromRFC9421Algorithm(algName)
				require.NoError(t, err)

				// Reverse lookup should work
				rfc9421Name, err := GetRFC9421AlgorithmName(keyType)
				require.NoError(t, err)
				assert.Equal(t, algName, rfc9421Name)
			})
		}
	})
}
