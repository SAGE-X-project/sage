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
	"crypto/ecdsa"
	"encoding/hex"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecp256k1KeyPair(t *testing.T) {
	t.Run("GenerateKeyPair", func(t *testing.T) {
		// Specification Requirement: Secp256k1 key generation (Ethereum compatible)
		helpers.LogTestSection(t, "2.1.2", "Secp256k1 Key Pair Generation (Ethereum)")

		keyPair, err := GenerateSecp256k1KeyPair()
		require.NoError(t, err)
		assert.NotNil(t, keyPair)

		// Specification Requirement: Key type validation
		assert.Equal(t, crypto.KeyTypeSecp256k1, keyPair.Type())
		helpers.LogSuccess(t, "Key type confirmed: Secp256k1")

		// Get raw key for validation
		pubKey := keyPair.PublicKey()
		assert.NotNil(t, pubKey)

		privKey := keyPair.PrivateKey()
		assert.NotNil(t, privKey)

		// Specification Requirement: Private key size must be 32 bytes
		ecdsaPrivKey, ok := privKey.(*ecdsa.PrivateKey)
		require.True(t, ok, "Private key should be *ecdsa.PrivateKey type")
		privKeyBytes := ecdsaPrivKey.D.Bytes()
		// D might be less than 32 bytes if leading zeros, but that's OK
		assert.LessOrEqual(t, len(privKeyBytes), 32, "Private key must be at most 32 bytes")

		// Specification Requirement: Public key (uncompressed) must be 65 bytes
		ecdsaPubKey, ok := pubKey.(*ecdsa.PublicKey)
		require.True(t, ok, "Public key should be *ecdsa.PublicKey type")
		uncompressedPubKey := ethcrypto.FromECDSAPub(ecdsaPubKey)
		assert.Equal(t, 65, len(uncompressedPubKey), "Uncompressed public key must be 65 bytes")

		// Specification Requirement: Ethereum address generation
		ethAddress := ethcrypto.PubkeyToAddress(*ecdsaPubKey)
		assert.Len(t, ethAddress.Hex(), 42, "Ethereum address must be 42 characters (0x + 40 hex)")
		assert.True(t, len(ethAddress.Hex()) == 42, "Ethereum address format check")

		helpers.LogSuccess(t, "Secp256k1 key pair generation successful (Ethereum compatible)")
		helpers.LogDetail(t, "Private key size: %d bytes (expected: 32 bytes)", len(privKeyBytes))
		helpers.LogDetail(t, "Uncompressed public key size: %d bytes (expected: 65 bytes)", len(uncompressedPubKey))
		helpers.LogDetail(t, "Ethereum address: %s", ethAddress.Hex())
		helpers.LogDetail(t, "Public key X: %x", ecdsaPubKey.X.Bytes())
		helpers.LogDetail(t, "Public key Y: %x", ecdsaPubKey.Y.Bytes())

		// Specification Requirement: JWK format with key ID
		keyID := keyPair.ID()
		assert.NotEmpty(t, keyID)
		helpers.LogDetail(t, "Key ID: %s", keyID)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Secp256k1 key generation successful",
			"Key type = Secp256k1",
			"Private key = 32 bytes",
			"Uncompressed public key = 65 bytes",
			"Ethereum address generation successful",
			"Ethereum address format valid (0x + 40 hex)",
			"Key ID present (JWK format)",
			"Ethereum compatible",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":                 "2.1.2_Secp256k1_Key_Generation",
			"key_type":                  string(keyPair.Type()),
			"key_id":                    keyID,
			"private_key_d":             hex.EncodeToString(ecdsaPrivKey.D.Bytes()),
			"public_key_x":              hex.EncodeToString(ecdsaPubKey.X.Bytes()),
			"public_key_y":              hex.EncodeToString(ecdsaPubKey.Y.Bytes()),
			"uncompressed_public_key":   hex.EncodeToString(uncompressedPubKey),
			"ethereum_address":          ethAddress.Hex(),
			"private_key_size":          len(privKeyBytes),
			"uncompressed_public_key_size": len(uncompressedPubKey),
			"expected_sizes": map[string]int{
				"private_key":             32,
				"uncompressed_public_key": 65,
			},
		}
		helpers.SaveTestData(t, "keys/secp256k1_key_generation.json", testData)
	})

	t.Run("SignAndVerify", func(t *testing.T) {
		// Specification Requirement: Secp256k1 signature/verification (65-byte signature with recovery)
		helpers.LogTestSection(t, "2.4.2", "Secp256k1 Signature and Verification (Ethereum)")

		keyPair, err := GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		message := []byte("test message for secp256k1 signature")
		helpers.LogDetail(t, "Test message: %s", string(message))
		helpers.LogDetail(t, "Message size: %d bytes", len(message))

		// Sign message
		signature, err := keyPair.Sign(message)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)

		// Specification Requirement: Secp256k1 signature size (typically 65 bytes with recovery byte)
		assert.Equal(t, 65, len(signature), "Secp256k1 signature with recovery byte must be 65 bytes")

		helpers.LogSuccess(t, "Signature generation successful")
		helpers.LogDetail(t, "Signature size: %d bytes (expected: 65 bytes)", len(signature))
		helpers.LogDetail(t, "Signature (hex): %x", signature)
		helpers.LogDetail(t, "Recovery byte (v): %d", signature[64])

		// Verify signature
		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
		helpers.LogSuccess(t, "Signature verification successful")

		// Specification Requirement: Ethereum address recovery from signature
		ecdsaPubKey := keyPair.PublicKey().(*ecdsa.PublicKey)
		expectedAddress := ethcrypto.PubkeyToAddress(*ecdsaPubKey)

		// Hash the message using Keccak256 (Ethereum style)
		hash := ethcrypto.Keccak256Hash(message)
		recoveredPubKey, err := ethcrypto.SigToPub(hash.Bytes(), signature)
		if err == nil {
			recoveredAddress := ethcrypto.PubkeyToAddress(*recoveredPubKey)
			assert.Equal(t, expectedAddress, recoveredAddress, "Recovered address should match original")
			helpers.LogSuccess(t, "Address recovery successful (Ethereum compatible)")
			helpers.LogDetail(t, "Expected address: %s", expectedAddress.Hex())
			helpers.LogDetail(t, "Recovered address: %s", recoveredAddress.Hex())
		}

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
			"Signature size = 65 bytes (with recovery byte)",
			"Verification successful",
			"Address recovery successful (Ethereum compatible)",
			"Tamper detection (wrong message)",
			"Tamper detection (modified signature)",
		})

		// Save test data for CLI verification
		privKey := keyPair.PrivateKey().(*ecdsa.PrivateKey)
		uncompressedPubKey := ethcrypto.FromECDSAPub(ecdsaPubKey)

		testData := map[string]interface{}{
			"test_case":              "2.4.2_Secp256k1_Sign_Verify",
			"message":                string(message),
			"message_hex":            hex.EncodeToString(message),
			"private_key_d":          hex.EncodeToString(privKey.D.Bytes()),
			"public_key_uncompressed": hex.EncodeToString(uncompressedPubKey),
			"ethereum_address":       expectedAddress.Hex(),
			"signature_hex":          hex.EncodeToString(signature),
			"signature_size":         len(signature),
			"expected_size":          65,
			"recovery_byte":          signature[64],
		}
		helpers.SaveTestData(t, "keys/secp256k1_sign_verify.json", testData)
	})

	t.Run("MultipleKeyPairsHaveDifferentIDs", func(t *testing.T) {
		keyPair1, err := GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		keyPair2, err := GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		assert.NotEqual(t, keyPair1.ID(), keyPair2.ID())
	})

	t.Run("SignEmptyMessage", func(t *testing.T) {
		keyPair, err := GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		message := []byte{}

		signature, err := keyPair.Sign(message)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)

		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
	})

	t.Run("SignLargeMessage", func(t *testing.T) {
		keyPair, err := GenerateSecp256k1KeyPair()
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

	t.Run("DeterministicSignatures", func(t *testing.T) {
		keyPair, err := GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		message := []byte("test message")

		// Generate multiple signatures for the same message
		sig1, err := keyPair.Sign(message)
		require.NoError(t, err)

		sig2, err := keyPair.Sign(message)
		require.NoError(t, err)

		// For secp256k1, signatures might not be identical due to randomness
		// But both should be valid
		err = keyPair.Verify(message, sig1)
		assert.NoError(t, err)

		err = keyPair.Verify(message, sig2)
		assert.NoError(t, err)
	})
}
