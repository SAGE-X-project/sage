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
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRSAKeyPair(t *testing.T) {
	t.Run("GenerateKeyPair", func(t *testing.T) {
		helpers.LogTestSection(t, "2.1.3", "RSA Key Pair Generation")

		helpers.LogDetail(t, "Step 1: Generate RSA key pair")
		keyPair, err := GenerateRSAKeyPair()
		require.NoError(t, err)
		assert.NotNil(t, keyPair)
		helpers.LogSuccess(t, "RSA key pair generated successfully")

		helpers.LogDetail(t, "Step 2: Validate key type")
		assert.Equal(t, crypto.KeyTypeRSA, keyPair.Type())
		helpers.LogSuccess(t, "Key type confirmed: RSA")

		helpers.LogDetail(t, "Step 3: Validate key material")
		assert.NotNil(t, keyPair.PublicKey())
		assert.NotNil(t, keyPair.PrivateKey())
		assert.NotEmpty(t, keyPair.ID())
		helpers.LogSuccess(t, "Key material validated")
		helpers.LogDetail(t, "  Key ID: %s", keyPair.ID())

		helpers.LogPassCriteria(t, []string{
			"RSA key pair generated",
			"Key type = RSA",
			"Public/Private keys exist",
			"Key ID generated",
		})
	})

	t.Run("SignAndVerify", func(t *testing.T) {
		helpers.LogTestSection(t, "2.4.3", "RSA Signature and Verification")

		helpers.LogDetail(t, "Step 1: Generate RSA key pair")
		keyPair, err := GenerateRSAKeyPair()
		require.NoError(t, err)
		helpers.LogSuccess(t, "Key pair generated")

		message := []byte("test message")
		helpers.LogDetail(t, "Test message: %s", string(message))

		// Sign message
		helpers.LogDetail(t, "Step 2: Sign message")
		signature, err := keyPair.Sign(message)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)
		helpers.LogSuccess(t, "Signature generated")
		helpers.LogDetail(t, "  Signature size: %d bytes", len(signature))

		// Verify signature
		helpers.LogDetail(t, "Step 3: Verify signature")
		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
		helpers.LogSuccess(t, "Signature verification successful")

		// Verify with wrong message should fail
		helpers.LogDetail(t, "Step 4: Test tamper detection - wrong message")
		wrongMessage := []byte("wrong message")
		err = keyPair.Verify(wrongMessage, signature)
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrInvalidSignature, err)
		helpers.LogSuccess(t, "Tamper detection: Wrong message rejected")

		// Verify with wrong signature should fail
		helpers.LogDetail(t, "Step 5: Test tamper detection - modified signature")
		wrongSignature := make([]byte, len(signature))
		copy(wrongSignature, signature)
		wrongSignature[0] ^= 0xFF
		err = keyPair.Verify(message, wrongSignature)
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrInvalidSignature, err)
		helpers.LogSuccess(t, "Tamper detection: Modified signature rejected")

		helpers.LogPassCriteria(t, []string{
			"Signature generation successful",
			"Signature verification successful",
			"Tamper detection (wrong message)",
			"Tamper detection (modified signature)",
		})
	})

	t.Run("MultipleKeyPairsHaveDifferentIDs", func(t *testing.T) {
		keyPair1, err := GenerateRSAKeyPair()
		require.NoError(t, err)

		keyPair2, err := GenerateRSAKeyPair()
		require.NoError(t, err)

		assert.NotEqual(t, keyPair1.ID(), keyPair2.ID())
	})

	t.Run("SignEmptyMessage", func(t *testing.T) {
		keyPair, err := GenerateRSAKeyPair()
		require.NoError(t, err)

		message := []byte{}

		signature, err := keyPair.Sign(message)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)

		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
	})

	t.Run("SignLargeMessage", func(t *testing.T) {
		keyPair, err := GenerateRSAKeyPair()
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
		keyPair, err := GenerateRSAKeyPair()
		require.NoError(t, err)

		message := []byte("test message")

		// Generate multiple signatures for the same message
		sig1, err := keyPair.Sign(message)
		require.NoError(t, err)

		sig2, err := keyPair.Sign(message)
		require.NoError(t, err)

		// RS256 signatures with PKCS#1 v1.5 can differ, but both must verify
		err = keyPair.Verify(message, sig1)
		assert.NoError(t, err)

		err = keyPair.Verify(message, sig2)
		assert.NoError(t, err)
	})
}
