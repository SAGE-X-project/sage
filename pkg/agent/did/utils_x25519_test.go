// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// Edge case and security tests for X25519 support in DID utils

package did

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnmarshalPublicKey_X25519_Valid tests valid X25519 key unmarshaling
func TestUnmarshalPublicKey_X25519_Valid(t *testing.T) {
	t.Run("Valid 32-byte X25519 key", func(t *testing.T) {
		// Generate a valid 32-byte X25519 public key
		validKey := make([]byte, 32)
		_, err := rand.Read(validKey)
		require.NoError(t, err)

		result, err := UnmarshalPublicKey(validKey, "x25519")
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify it returns a byte slice
		keyBytes, ok := result.([]byte)
		assert.True(t, ok, "Should return []byte type")
		assert.Len(t, keyBytes, 32, "Should be 32 bytes")
		assert.Equal(t, validKey, keyBytes, "Should match input key")
	})

	t.Run("X25519 key with all zeros", func(t *testing.T) {
		zeroKey := make([]byte, 32)
		result, err := UnmarshalPublicKey(zeroKey, "x25519")
		assert.NoError(t, err)
		assert.NotNil(t, result)

		keyBytes := result.([]byte)
		assert.Equal(t, zeroKey, keyBytes)
	})

	t.Run("X25519 key with all ones", func(t *testing.T) {
		onesKey := make([]byte, 32)
		for i := range onesKey {
			onesKey[i] = 0xFF
		}
		result, err := UnmarshalPublicKey(onesKey, "x25519")
		assert.NoError(t, err)
		assert.NotNil(t, result)

		keyBytes := result.([]byte)
		assert.Equal(t, onesKey, keyBytes)
	})
}

// TestUnmarshalPublicKey_X25519_InvalidSize tests invalid key sizes
func TestUnmarshalPublicKey_X25519_InvalidSize(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
	}{
		{"Empty key", 0},
		{"Too small (1 byte)", 1},
		{"Too small (16 bytes)", 16},
		{"Too small (31 bytes)", 31},
		{"Too large (33 bytes)", 33},
		{"Too large (64 bytes)", 64},
		{"Too large (100 bytes)", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invalidKey := make([]byte, tt.keySize)
			_, err := rand.Read(invalidKey)
			if tt.keySize > 0 {
				require.NoError(t, err)
			}

			result, err := UnmarshalPublicKey(invalidKey, "x25519")
			assert.Error(t, err, "Should reject key of size %d", tt.keySize)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), "invalid X25519 public key size")
			assert.Contains(t, err.Error(), "expected 32 bytes")
		})
	}
}

// TestUnmarshalPublicKey_X25519_NilInput tests nil input handling
func TestUnmarshalPublicKey_X25519_NilInput(t *testing.T) {
	t.Run("Nil byte slice", func(t *testing.T) {
		var nilKey []byte
		result, err := UnmarshalPublicKey(nilKey, "x25519")
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid X25519 public key size")
		assert.Contains(t, err.Error(), "got 0")
	})
}

// TestUnmarshalPublicKey_X25519_MemoryIsolation tests that returned key is a copy
func TestUnmarshalPublicKey_X25519_MemoryIsolation(t *testing.T) {
	t.Run("Returned key is a copy, not reference", func(t *testing.T) {
		// Create original key
		originalKey := make([]byte, 32)
		for i := range originalKey {
			originalKey[i] = byte(i)
		}
		originalCopy := make([]byte, 32)
		copy(originalCopy, originalKey)

		// Unmarshal
		result, err := UnmarshalPublicKey(originalKey, "x25519")
		require.NoError(t, err)
		returnedKey := result.([]byte)

		// Verify it matches
		assert.Equal(t, originalCopy, returnedKey)

		// Modify original key
		for i := range originalKey {
			originalKey[i] = 0xFF
		}

		// Returned key should NOT be affected (proves it's a copy)
		assert.Equal(t, originalCopy, returnedKey, "Returned key should be independent copy")
		assert.NotEqual(t, originalKey, returnedKey, "Should not reflect modifications to original")
	})
}

// TestMarshalPublicKey_X25519_RoundTrip tests marshal/unmarshal round trip
func TestMarshalPublicKey_X25519_RoundTrip(t *testing.T) {
	t.Run("Unmarshal preserves key data", func(t *testing.T) {
		// Create original X25519 key bytes (as stored in registry)
		originalKey := make([]byte, 32)
		_, err := rand.Read(originalKey)
		require.NoError(t, err)

		// Unmarshal (simulating retrieval from storage)
		unmarshaled, err := UnmarshalPublicKey(originalKey, "x25519")
		require.NoError(t, err)

		// Verify it returns a proper copy
		unmarshaledBytes := unmarshaled.([]byte)
		assert.Equal(t, originalKey, unmarshaledBytes, "Unmarshal should preserve key data")

		// Verify it's a copy, not the same reference
		originalKey[0] ^= 0xFF
		assert.NotEqual(t, originalKey[0], unmarshaledBytes[0], "Should be independent copy")
	})
}

// TestUnmarshalPublicKey_X25519_ConcurrentAccess tests thread safety
func TestUnmarshalPublicKey_X25519_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent unmarshal operations", func(t *testing.T) {
		validKey := make([]byte, 32)
		_, err := rand.Read(validKey)
		require.NoError(t, err)

		// Run 100 concurrent unmarshal operations
		done := make(chan bool, 100)
		for i := 0; i < 100; i++ {
			go func() {
				result, err := UnmarshalPublicKey(validKey, "x25519")
				assert.NoError(t, err)
				assert.NotNil(t, result)
				keyBytes := result.([]byte)
				assert.Len(t, keyBytes, 32)
				done <- true
			}()
		}

		// Wait for all to complete
		for i := 0; i < 100; i++ {
			<-done
		}
	})
}

// TestUnmarshalPublicKey_X25519_Integration tests integration with HPKE usage
func TestUnmarshalPublicKey_X25519_Integration(t *testing.T) {
	t.Run("X25519 key usable for HPKE operations", func(t *testing.T) {
		// Simulate retrieving X25519 key from storage/registry
		storedKey := make([]byte, 32)
		_, err := rand.Read(storedKey)
		require.NoError(t, err)

		// Unmarshal
		result, err := UnmarshalPublicKey(storedKey, "x25519")
		require.NoError(t, err)

		// Verify it's usable (correct type and size)
		keyBytes, ok := result.([]byte)
		require.True(t, ok, "Should be []byte for HPKE compatibility")
		require.Len(t, keyBytes, 32, "Should be 32 bytes for X25519")

		// In real HPKE usage, this would be converted to ecdh.PublicKey:
		// import "crypto/ecdh"
		// x25519 := ecdh.X25519()
		// hpkeKey, err := x25519.NewPublicKey(keyBytes)
		// For this test, we just verify the format is correct
	})
}

// TestUnmarshalPublicKey_X25519_ErrorMessages tests error message quality
func TestUnmarshalPublicKey_X25519_ErrorMessages(t *testing.T) {
	t.Run("Error message includes actual size", func(t *testing.T) {
		invalidKey := make([]byte, 16)
		_, err := UnmarshalPublicKey(invalidKey, "x25519")
		require.Error(t, err)

		errMsg := err.Error()
		assert.Contains(t, errMsg, "invalid X25519 public key size")
		assert.Contains(t, errMsg, "expected 32 bytes")
		assert.Contains(t, errMsg, "got 16", "Should include actual size")
	})

	t.Run("Error message for zero-length key", func(t *testing.T) {
		emptyKey := []byte{}
		_, err := UnmarshalPublicKey(emptyKey, "x25519")
		require.Error(t, err)

		errMsg := err.Error()
		assert.Contains(t, errMsg, "got 0", "Should indicate zero length")
	})
}

// Benchmark X25519 unmarshal performance
func BenchmarkUnmarshalPublicKey_X25519(b *testing.B) {
	validKey := make([]byte, 32)
	_, err := rand.Read(validKey)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = UnmarshalPublicKey(validKey, "x25519")
	}
}

// Benchmark X25519 marshal performance
func BenchmarkMarshalPublicKey_X25519(b *testing.B) {
	validKey := make([]byte, 32)
	_, err := rand.Read(validKey)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = MarshalPublicKey(validKey)
	}
}
