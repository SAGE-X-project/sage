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


package rotation

import (
	"testing"
	"time"

	"github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/crypto/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyRotator(t *testing.T) {
	// Create storage and rotator
	keyStorage := storage.NewMemoryKeyStorage()
	rotator := NewKeyRotator(keyStorage)

	t.Run("RotateNonExistentKey", func(t *testing.T) {
		_, err := rotator.Rotate("non-existent")
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrKeyNotFound, err)
	})

	t.Run("RotateExistingKey", func(t *testing.T) {
		// Create and store initial key
		oldKeyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		
		err = keyStorage.Store("rotate-test", oldKeyPair)
		require.NoError(t, err)

		// Rotate the key
		newKeyPair, err := rotator.Rotate("rotate-test")
		require.NoError(t, err)
		assert.NotNil(t, newKeyPair)
		
		// Verify it's a different key
		assert.NotEqual(t, oldKeyPair.ID(), newKeyPair.ID())
		assert.Equal(t, oldKeyPair.Type(), newKeyPair.Type())

		// Verify new key is stored
		loadedKey, err := keyStorage.Load("rotate-test")
		require.NoError(t, err)
		assert.Equal(t, newKeyPair.ID(), loadedKey.ID())

		// Verify rotation history
		history, err := rotator.GetRotationHistory("rotate-test")
		require.NoError(t, err)
		assert.Len(t, history, 1)
		assert.Equal(t, oldKeyPair.ID(), history[0].OldKeyID)
		assert.Equal(t, newKeyPair.ID(), history[0].NewKeyID)
		assert.Equal(t, "Manual rotation", history[0].Reason)
	})

	t.Run("MultipleRotations", func(t *testing.T) {
		// Create and store initial key
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)
		
		err = keyStorage.Store("multi-rotate", keyPair)
		require.NoError(t, err)

		// Perform multiple rotations
		var keyIDs []string
		keyIDs = append(keyIDs, keyPair.ID())

		for i := 0; i < 3; i++ {
			newKeyPair, err := rotator.Rotate("multi-rotate")
			require.NoError(t, err)
			keyIDs = append(keyIDs, newKeyPair.ID())
		}

		// Verify rotation history
		history, err := rotator.GetRotationHistory("multi-rotate")
		require.NoError(t, err)
		assert.Len(t, history, 3)

		// Verify history order (most recent first)
		for i := 0; i < 3; i++ {
			assert.Equal(t, keyIDs[i], history[2-i].OldKeyID)
			assert.Equal(t, keyIDs[i+1], history[2-i].NewKeyID)
		}
	})

	t.Run("RotationWithKeepOldKeys", func(t *testing.T) {
		// Create new rotator with keep old keys config
		rotatorWithKeep := NewKeyRotator(keyStorage)
		rotatorWithKeep.SetRotationConfig(crypto.KeyRotationConfig{
			KeepOldKeys: true,
		})

		// Create and store initial key
		oldKeyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		
		err = keyStorage.Store("keep-old-test", oldKeyPair)
		require.NoError(t, err)

		// Rotate the key
		newKeyPair, err := rotatorWithKeep.Rotate("keep-old-test")
		require.NoError(t, err)

		// Verify new key is stored under the original ID
		loadedKey, err := keyStorage.Load("keep-old-test")
		require.NoError(t, err)
		assert.Equal(t, newKeyPair.ID(), loadedKey.ID())

		// Verify old key is also stored with a special ID
		oldKeyStored, err := keyStorage.Load("keep-old-test.old." + oldKeyPair.ID())
		require.NoError(t, err)
		assert.Equal(t, oldKeyPair.ID(), oldKeyStored.ID())
	})

	t.Run("GetRotationHistoryEmpty", func(t *testing.T) {
		history, err := rotator.GetRotationHistory("no-history")
		require.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("RotationConfigPersistence", func(t *testing.T) {
		rotator := NewKeyRotator(keyStorage)
		
		// Set custom config
		config := crypto.KeyRotationConfig{
			RotationInterval: 24 * time.Hour,
			MaxKeyAge:        7 * 24 * time.Hour,
			KeepOldKeys:      true,
		}
		rotator.SetRotationConfig(config)

		// Create and rotate a key
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		err = keyStorage.Store("config-test", keyPair)
		require.NoError(t, err)

		// Rotate with custom config
		_, err = rotator.Rotate("config-test")
		require.NoError(t, err)

		// Verify old key was kept
		oldKey, err := keyStorage.Load("config-test.old." + keyPair.ID())
		assert.NoError(t, err)
		assert.NotNil(t, oldKey)
	})

	t.Run("ConcurrentRotations", func(t *testing.T) {
		// Create initial key
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		err = keyStorage.Store("concurrent-test", keyPair)
		require.NoError(t, err)

		// Try concurrent rotations
		done := make(chan error, 5)
		for i := 0; i < 5; i++ {
			go func() {
				_, err := rotator.Rotate("concurrent-test")
				done <- err
			}()
		}

		// Collect results
		var errors []error
		for i := 0; i < 5; i++ {
			if err := <-done; err != nil {
				errors = append(errors, err)
			}
		}

		// At least one should succeed
		assert.Less(t, len(errors), 5)

		// Verify key exists and is valid
		finalKey, err := keyStorage.Load("concurrent-test")
		require.NoError(t, err)
		assert.NotNil(t, finalKey)
	})
}
