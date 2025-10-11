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
	"os"
	"path/filepath"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileKeyStorage(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := os.MkdirTemp("", "sage-key-storage-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	storage, err := NewFileKeyStorage(tempDir)
	require.NoError(t, err)

	t.Run("StoreAndLoadKeyPair", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Store the key pair
		err = storage.Store("test-key", keyPair)
		require.NoError(t, err)

		// Verify file was created
		keyFile := filepath.Join(tempDir, "test-key.key")
		assert.FileExists(t, keyFile)

		// Load the key pair
		loadedKeyPair, err := storage.Load("test-key")
		require.NoError(t, err)
		assert.NotNil(t, loadedKeyPair)
		assert.Equal(t, keyPair.Type(), loadedKeyPair.Type())

		// Test signing with loaded key
		message := []byte("test message")
		signature, err := loadedKeyPair.Sign(message)
		require.NoError(t, err)

		// Verify with original key
		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
	})

	t.Run("StoreSecp256k1KeyPair", func(t *testing.T) {
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		// Store the key pair
		err = storage.Store("secp256k1-key", keyPair)
		require.NoError(t, err)

		// Load the key pair
		loadedKeyPair, err := storage.Load("secp256k1-key")
		require.NoError(t, err)
		assert.NotNil(t, loadedKeyPair)
		assert.Equal(t, crypto.KeyTypeSecp256k1, loadedKeyPair.Type())
	})

	t.Run("LoadNonExistentKey", func(t *testing.T) {
		_, err := storage.Load("non-existent")
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrKeyNotFound, err)
	})

	t.Run("DeleteKey", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Store the key
		err = storage.Store("delete-test", keyPair)
		require.NoError(t, err)

		// Verify file exists
		keyFile := filepath.Join(tempDir, "delete-test.key")
		assert.FileExists(t, keyFile)

		// Delete the key
		err = storage.Delete("delete-test")
		require.NoError(t, err)

		// Verify file is gone
		assert.NoFileExists(t, keyFile)

		// Try to load deleted key
		_, err = storage.Load("delete-test")
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrKeyNotFound, err)
	})

	t.Run("ListKeys", func(t *testing.T) {
		// Create new storage in clean directory
		listDir, err := os.MkdirTemp("", "sage-list-test-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(listDir) }()

		listStorage, err := NewFileKeyStorage(listDir)
		require.NoError(t, err)

		// Add multiple keys
		keyPair1, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		keyPair2, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)
		keyPair3, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		err = listStorage.Store("key1", keyPair1)
		require.NoError(t, err)
		err = listStorage.Store("key2", keyPair2)
		require.NoError(t, err)
		err = listStorage.Store("key3", keyPair3)
		require.NoError(t, err)

		// List all keys
		ids, err := listStorage.List()
		require.NoError(t, err)
		assert.Len(t, ids, 3)
		assert.Contains(t, ids, "key1")
		assert.Contains(t, ids, "key2")
		assert.Contains(t, ids, "key3")
	})

	t.Run("InvalidKeyID", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Try to store with invalid ID containing path separator
		err = storage.Store("../invalid/key", keyPair)
		assert.Error(t, err)

		err = storage.Store("invalid\\key", keyPair)
		assert.Error(t, err)
	})

	t.Run("CorruptedKeyFile", func(t *testing.T) {
		// Create a corrupted key file
		corruptedFile := filepath.Join(tempDir, "corrupted.key")
		err := os.WriteFile(corruptedFile, []byte("corrupted data"), 0600)
		require.NoError(t, err)

		// Try to load corrupted key
		_, err = storage.Load("corrupted")
		assert.Error(t, err)
	})

	t.Run("FilePermissions", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Store the key
		err = storage.Store("perm-test", keyPair)
		require.NoError(t, err)

		// Check file permissions
		keyFile := filepath.Join(tempDir, "perm-test.key")
		info, err := os.Stat(keyFile)
		require.NoError(t, err)

		// Should be readable/writable by owner only (0600)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	})
}
