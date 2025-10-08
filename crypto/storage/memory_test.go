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


package storage

import (
	"fmt"
	"testing"

	"github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryKeyStorage(t *testing.T) {
	storage := NewMemoryKeyStorage()

	t.Run("StoreAndLoadKeyPair", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Store the key pair
		err = storage.Store("test-key", keyPair)
		require.NoError(t, err)

		// Load the key pair
		loadedKeyPair, err := storage.Load("test-key")
		require.NoError(t, err)
		assert.NotNil(t, loadedKeyPair)
		assert.Equal(t, keyPair.ID(), loadedKeyPair.ID())
		assert.Equal(t, keyPair.Type(), loadedKeyPair.Type())

		// Test signing with loaded key
		message := []byte("test message")
		signature, err := loadedKeyPair.Sign(message)
		require.NoError(t, err)

		// Verify with original key
		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
	})

	t.Run("LoadNonExistentKey", func(t *testing.T) {
		_, err := storage.Load("non-existent")
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrKeyNotFound, err)
	})

	t.Run("OverwriteExistingKey", func(t *testing.T) {
		keyPair1, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		keyPair2, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Store first key
		err = storage.Store("overwrite-test", keyPair1)
		require.NoError(t, err)

		// Overwrite with second key
		err = storage.Store("overwrite-test", keyPair2)
		require.NoError(t, err)

		// Load should return the second key
		loadedKeyPair, err := storage.Load("overwrite-test")
		require.NoError(t, err)
		assert.Equal(t, keyPair2.ID(), loadedKeyPair.ID())
	})

	t.Run("DeleteKey", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Store the key
		err = storage.Store("delete-test", keyPair)
		require.NoError(t, err)

		// Verify it exists
		assert.True(t, storage.Exists("delete-test"))

		// Delete the key
		err = storage.Delete("delete-test")
		require.NoError(t, err)

		// Verify it's gone
		assert.False(t, storage.Exists("delete-test"))

		// Try to load deleted key
		_, err = storage.Load("delete-test")
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrKeyNotFound, err)
	})

	t.Run("DeleteNonExistentKey", func(t *testing.T) {
		err := storage.Delete("non-existent")
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrKeyNotFound, err)
	})

	t.Run("ListKeys", func(t *testing.T) {
		// Clear storage first
		storage = NewMemoryKeyStorage()

		// Add multiple keys
		keyPair1, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		keyPair2, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)
		keyPair3, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		err = storage.Store("key1", keyPair1)
		require.NoError(t, err)
		err = storage.Store("key2", keyPair2)
		require.NoError(t, err)
		err = storage.Store("key3", keyPair3)
		require.NoError(t, err)

		// List all keys
		ids, err := storage.List()
		require.NoError(t, err)
		assert.Len(t, ids, 3)
		assert.Contains(t, ids, "key1")
		assert.Contains(t, ids, "key2")
		assert.Contains(t, ids, "key3")
	})

	t.Run("EmptyStorageList", func(t *testing.T) {
		emptyStorage := NewMemoryKeyStorage()
		ids, err := emptyStorage.List()
		require.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		storage := NewMemoryKeyStorage()
		done := make(chan bool)

		// Multiple goroutines storing keys
		for i := 0; i < 10; i++ {
			go func(id int) {
				keyPair, _ := keys.GenerateEd25519KeyPair()
				storage.Store(fmt.Sprintf("concurrent-%d", id), keyPair)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify all keys were stored
		ids, err := storage.List()
		require.NoError(t, err)
		assert.Len(t, ids, 10)
	})
}
