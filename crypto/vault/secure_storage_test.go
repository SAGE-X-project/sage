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

package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileVault(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "vault_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	vault, err := NewFileVault(tempDir)
	require.NoError(t, err)

	t.Run("StoreAndLoadKey", func(t *testing.T) {
		keyID := "test_key_1"
		originalKey := []byte("this is my secret key data")
		passphrase := "strong_passphrase_123"

		// Store encrypted key
		err := vault.StoreEncrypted(keyID, originalKey, passphrase)
		assert.NoError(t, err)

		// Verify file was created with correct permissions
		filePath := filepath.Join(tempDir, keyID+".json")
		info, err := os.Stat(filePath)
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

		// Load and decrypt key
		loadedKey, err := vault.LoadDecrypted(keyID, passphrase)
		assert.NoError(t, err)
		assert.Equal(t, originalKey, loadedKey)
	})

	t.Run("InvalidPassphrase", func(t *testing.T) {
		keyID := "test_key_2"
		originalKey := []byte("another secret key")
		correctPassphrase := "correct_passphrase"
		wrongPassphrase := "wrong_passphrase"

		// Store with correct passphrase
		err := vault.StoreEncrypted(keyID, originalKey, correctPassphrase)
		assert.NoError(t, err)

		// Try to load with wrong passphrase
		_, err = vault.LoadDecrypted(keyID, wrongPassphrase)
		assert.Equal(t, ErrInvalidPassphrase, err)
	})

	t.Run("KeyNotFound", func(t *testing.T) {
		_, err := vault.LoadDecrypted("non_existent_key", "passphrase")
		assert.Equal(t, ErrKeyNotFound, err)
	})

	t.Run("InvalidKeyID", func(t *testing.T) {
		err := vault.StoreEncrypted("", []byte("key"), "passphrase")
		assert.Equal(t, ErrInvalidKeyID, err)

		_, err = vault.LoadDecrypted("", "passphrase")
		assert.Equal(t, ErrInvalidKeyID, err)
	})

	t.Run("SetPermissions", func(t *testing.T) {
		keyID := "test_key_3"
		key := []byte("permission test key")
		passphrase := "passphrase"

		// Store key
		err := vault.StoreEncrypted(keyID, key, passphrase)
		assert.NoError(t, err)

		// Change permissions
		err = vault.SetPermissions(keyID, 0644)
		assert.NoError(t, err)

		// Verify permissions changed
		filePath := filepath.Join(tempDir, keyID+".json")
		info, err := os.Stat(filePath)
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0644), info.Mode().Perm())

		// Try to set permissions on non-existent key
		err = vault.SetPermissions("non_existent", 0600)
		assert.Equal(t, ErrKeyNotFound, err)
	})

	t.Run("DeleteKey", func(t *testing.T) {
		keyID := "test_key_4"
		key := []byte("key to delete")
		passphrase := "passphrase"

		// Store key
		err := vault.StoreEncrypted(keyID, key, passphrase)
		assert.NoError(t, err)
		assert.True(t, vault.Exists(keyID))

		// Delete key
		err = vault.Delete(keyID)
		assert.NoError(t, err)
		assert.False(t, vault.Exists(keyID))

		// Try to load deleted key
		_, err = vault.LoadDecrypted(keyID, passphrase)
		assert.Equal(t, ErrKeyNotFound, err)

		// Try to delete non-existent key
		err = vault.Delete("non_existent")
		assert.Equal(t, ErrKeyNotFound, err)
	})

	t.Run("ListKeys", func(t *testing.T) {
		// Clear vault
		for _, key := range vault.ListKeys() {
			vault.Delete(key)
		}

		// Add multiple keys
		keys := []string{"key_a", "key_b", "key_c"}
		for _, keyID := range keys {
			err := vault.StoreEncrypted(keyID, []byte("data"), "passphrase")
			assert.NoError(t, err)
		}

		// List keys
		listedKeys := vault.ListKeys()
		assert.Len(t, listedKeys, 3)
		for _, key := range keys {
			assert.Contains(t, listedKeys, key)
		}
	})

	t.Run("OverwriteKey", func(t *testing.T) {
		keyID := "test_key_5"
		originalKey := []byte("original data")
		newKey := []byte("new data")
		passphrase := "passphrase"

		// Store original
		err := vault.StoreEncrypted(keyID, originalKey, passphrase)
		assert.NoError(t, err)

		// Overwrite with new data
		err = vault.StoreEncrypted(keyID, newKey, passphrase)
		assert.NoError(t, err)

		// Load should return new data
		loadedKey, err := vault.LoadDecrypted(keyID, passphrase)
		assert.NoError(t, err)
		assert.Equal(t, newKey, loadedKey)
	})

	t.Run("LargeKey", func(t *testing.T) {
		keyID := "large_key"
		// Create a 10KB key
		largeKey := make([]byte, 10*1024)
		for i := range largeKey {
			largeKey[i] = byte(i % 256)
		}
		passphrase := "passphrase"

		// Store large key
		err := vault.StoreEncrypted(keyID, largeKey, passphrase)
		assert.NoError(t, err)

		// Load and verify
		loadedKey, err := vault.LoadDecrypted(keyID, passphrase)
		assert.NoError(t, err)
		assert.True(t, bytes.Equal(largeKey, loadedKey))
	})
}

func TestMemoryVault(t *testing.T) {
	vault := NewMemoryVault()

	t.Run("StoreAndLoadKey", func(t *testing.T) {
		keyID := "test_key_1"
		originalKey := []byte("this is my secret key data")
		passphrase := "strong_passphrase_123"

		// Store encrypted key
		err := vault.StoreEncrypted(keyID, originalKey, passphrase)
		assert.NoError(t, err)

		// Load and decrypt key
		loadedKey, err := vault.LoadDecrypted(keyID, passphrase)
		assert.NoError(t, err)
		assert.Equal(t, originalKey, loadedKey)
	})

	t.Run("KeyNotFound", func(t *testing.T) {
		_, err := vault.LoadDecrypted("non_existent_key", "passphrase")
		assert.Equal(t, ErrKeyNotFound, err)
	})

	t.Run("DeleteKey", func(t *testing.T) {
		keyID := "test_key_2"
		key := []byte("key to delete")
		passphrase := "passphrase"

		// Store key
		err := vault.StoreEncrypted(keyID, key, passphrase)
		assert.NoError(t, err)
		assert.True(t, vault.Exists(keyID))

		// Delete key
		err = vault.Delete(keyID)
		assert.NoError(t, err)
		assert.False(t, vault.Exists(keyID))
	})

	t.Run("ListKeys", func(t *testing.T) {
		// Clear vault
		for _, key := range vault.ListKeys() {
			vault.Delete(key)
		}

		// Add multiple keys
		keys := []string{"key_x", "key_y", "key_z"}
		for _, keyID := range keys {
			err := vault.StoreEncrypted(keyID, []byte("data"), "passphrase")
			assert.NoError(t, err)
		}

		// List keys
		listedKeys := vault.ListKeys()
		assert.Len(t, listedKeys, 3)
		for _, key := range keys {
			assert.Contains(t, listedKeys, key)
		}
	})

	t.Run("SetPermissions", func(t *testing.T) {
		// For memory vault, this is a no-op but should not error
		keyID := "test_key_3"
		err := vault.StoreEncrypted(keyID, []byte("data"), "pass")
		assert.NoError(t, err)

		err = vault.SetPermissions(keyID, 0600)
		assert.NoError(t, err)

		// Non-existent key should error
		err = vault.SetPermissions("non_existent", 0600)
		assert.Equal(t, ErrKeyNotFound, err)
	})
}

func BenchmarkFileVault(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "vault_bench")
	require.NoError(b, err)
	defer os.RemoveAll(tempDir)

	vault, err := NewFileVault(tempDir)
	require.NoError(b, err)

	key := []byte("benchmark test key data that is 32 bytes long!!")
	passphrase := "benchmark_passphrase"

	b.Run("StoreEncrypted", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			keyID := "bench_key_" + string(rune(i))
			vault.StoreEncrypted(keyID, key, passphrase)
		}
	})

	// Setup for load benchmark
	testKeyID := "bench_load_key"
	vault.StoreEncrypted(testKeyID, key, passphrase)

	b.Run("LoadDecrypted", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			vault.LoadDecrypted(testKeyID, passphrase)
		}
	})
}

func BenchmarkMemoryVault(b *testing.B) {
	vault := NewMemoryVault()

	key := []byte("benchmark test key data that is 32 bytes long!!")
	passphrase := "benchmark_passphrase"

	b.Run("StoreEncrypted", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			keyID := "bench_key_" + string(rune(i))
			vault.StoreEncrypted(keyID, key, passphrase)
		}
	})

	// Setup for load benchmark
	testKeyID := "bench_load_key"
	vault.StoreEncrypted(testKeyID, key, passphrase)

	b.Run("LoadDecrypted", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			vault.LoadDecrypted(testKeyID, passphrase)
		}
	})
}