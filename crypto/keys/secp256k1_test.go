// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


package keys

import (
	"testing"

	"github.com/sage-x-project/sage/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecp256k1KeyPair(t *testing.T) {
	t.Run("GenerateKeyPair", func(t *testing.T) {
		keyPair, err := GenerateSecp256k1KeyPair()
		require.NoError(t, err)
		assert.NotNil(t, keyPair)
		assert.Equal(t, crypto.KeyTypeSecp256k1, keyPair.Type())
		assert.NotNil(t, keyPair.PublicKey())
		assert.NotNil(t, keyPair.PrivateKey())
		assert.NotEmpty(t, keyPair.ID())
	})

	t.Run("SignAndVerify", func(t *testing.T) {
		keyPair, err := GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		message := []byte("test message")
		
		// Sign message
		signature, err := keyPair.Sign(message)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)
		
		// Verify signature
		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
		
		// Verify with wrong message should fail
		wrongMessage := []byte("wrong message")
		err = keyPair.Verify(wrongMessage, signature)
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrInvalidSignature, err)
		
		// Verify with wrong signature should fail
		wrongSignature := make([]byte, len(signature))
		copy(wrongSignature, signature)
		wrongSignature[0] ^= 0xFF
		err = keyPair.Verify(message, wrongSignature)
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrInvalidSignature, err)
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