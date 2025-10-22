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

package keys

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEd25519KeyPair(t *testing.T) {
	t.Run("GenerateKeyPair", func(t *testing.T) {
		// Specification Requirement: Ed25519 key generation (32-byte public key, 64-byte private key)
		helpers.LogTestSection(t, "2.1.1", "Ed25519 Key Pair Generation")

		keyPair, err := GenerateEd25519KeyPair()
		require.NoError(t, err)
		assert.NotNil(t, keyPair)

		// Specification Requirement: Key type validation
		assert.Equal(t, crypto.KeyTypeEd25519, keyPair.Type())
		helpers.LogSuccess(t, "Key type confirmed: Ed25519")

		// Get raw key bytes for size validation
		pubKey := keyPair.PublicKey()
		assert.NotNil(t, pubKey)

		privKey := keyPair.PrivateKey()
		assert.NotNil(t, privKey)

		// Specification Requirement: Public key size must be 32 bytes
		pubKeyBytes, ok := pubKey.(ed25519.PublicKey)
		require.True(t, ok, "Public key should be ed25519.PublicKey type")
		assert.Equal(t, 32, len(pubKeyBytes), "Public key must be 32 bytes")

		// Specification Requirement: Private key size must be 64 bytes
		privKeyBytes, ok := privKey.(ed25519.PrivateKey)
		require.True(t, ok, "Private key should be ed25519.PrivateKey type")
		assert.Equal(t, 64, len(privKeyBytes), "Private key must be 64 bytes")

		helpers.LogSuccess(t, "Ed25519 key pair generation successful")
		helpers.LogDetail(t, "Public key size: %d bytes (expected: 32 bytes)", len(pubKeyBytes))
		helpers.LogDetail(t, "Private key size: %d bytes (expected: 64 bytes)", len(privKeyBytes))
		helpers.LogDetail(t, "Public key (hex): %x", pubKeyBytes)

		// Specification Requirement: JWK format with key ID
		keyID := keyPair.ID()
		assert.NotEmpty(t, keyID)
		helpers.LogDetail(t, "Key ID: %s", keyID)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Ed25519 key generation successful",
			"Key type = Ed25519",
			"Public key = 32 bytes",
			"Private key = 64 bytes",
			"Key ID present (JWK format)",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":       "2.1.1_Ed25519_Key_Generation",
			"key_type":        string(keyPair.Type()),
			"key_id":          keyID,
			"public_key_hex":  hex.EncodeToString(pubKeyBytes),
			"public_key_size": len(pubKeyBytes),
			"private_key_size": len(privKeyBytes),
			"expected_sizes": map[string]int{
				"public_key":  32,
				"private_key": 64,
			},
		}
		helpers.SaveTestData(t, "keys/ed25519_key_generation.json", testData)
	})

	t.Run("SignAndVerify", func(t *testing.T) {
		// Specification Requirement: Ed25519 signature/verification (64-byte signature)
		helpers.LogTestSection(t, "2.4.1", "Ed25519 Signature and Verification")

		keyPair, err := GenerateEd25519KeyPair()
		require.NoError(t, err)

		message := []byte("test message for ed25519 signature")
		helpers.LogDetail(t, "Test message: %s", string(message))
		helpers.LogDetail(t, "Message size: %d bytes", len(message))

		// Sign message
		signature, err := keyPair.Sign(message)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)

		// Specification Requirement: Signature size must be 64 bytes
		assert.Equal(t, 64, len(signature), "Ed25519 signature must be 64 bytes")

		helpers.LogSuccess(t, "Signature generation successful")
		helpers.LogDetail(t, "Signature size: %d bytes (expected: 64 bytes)", len(signature))
		helpers.LogDetail(t, "Signature (hex): %x", signature)

		// Verify signature
		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
		helpers.LogSuccess(t, "Signature verification successful")

		// Specification Requirement: Tamper detection - wrong message should fail
		wrongMessage := []byte("wrong message")
		err = keyPair.Verify(wrongMessage, signature)
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrInvalidSignature, err)
		helpers.LogSuccess(t, "Tamper detection: Wrong message rejected (expected behavior)")

		// Specification Requirement: Tamper detection - modified signature should fail
		wrongSignature := make([]byte, len(signature))
		copy(wrongSignature, signature)
		wrongSignature[0] ^= 0xFF
		err = keyPair.Verify(message, wrongSignature)
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrInvalidSignature, err)
		helpers.LogSuccess(t, "Tamper detection: Modified signature rejected (expected behavior)")

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Signature generation successful",
			"Signature size = 64 bytes",
			"Verification successful",
			"Tamper detection (wrong message)",
			"Tamper detection (modified signature)",
		})

		// Save test data for CLI verification
		pubKey := keyPair.PublicKey().(ed25519.PublicKey)
		privKey := keyPair.PrivateKey().(ed25519.PrivateKey)

		testData := map[string]interface{}{
			"test_case":        "2.4.1_Ed25519_Sign_Verify",
			"message":          string(message),
			"message_hex":      hex.EncodeToString(message),
			"public_key_hex":   hex.EncodeToString(pubKey),
			"private_key_hex":  hex.EncodeToString(privKey),
			"signature_hex":    hex.EncodeToString(signature),
			"signature_size":   len(signature),
			"expected_size":    64,
		}
		helpers.SaveTestData(t, "keys/ed25519_sign_verify.json", testData)
	})

	t.Run("MultipleKeyPairsHaveDifferentIDs", func(t *testing.T) {
		keyPair1, err := GenerateEd25519KeyPair()
		require.NoError(t, err)

		keyPair2, err := GenerateEd25519KeyPair()
		require.NoError(t, err)

		assert.NotEqual(t, keyPair1.ID(), keyPair2.ID())
	})

	t.Run("SignEmptyMessage", func(t *testing.T) {
		keyPair, err := GenerateEd25519KeyPair()
		require.NoError(t, err)

		message := []byte{}

		signature, err := keyPair.Sign(message)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)

		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
	})

	t.Run("SignLargeMessage", func(t *testing.T) {
		keyPair, err := GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Create a 1MB message
		message := make([]byte, 1024*1024)
		for i := range message {
			message[i] = byte(i % 256)
		}

		signature, err := keyPair.Sign(message)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)

		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
	})
}
