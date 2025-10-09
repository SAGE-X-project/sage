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


package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	_ "github.com/sage-x-project/sage/internal/cryptoinit" // Initialize wrappers
)

func TestNewManager(t *testing.T) {
	manager := sagecrypto.NewManager()
	assert.NotNil(t, manager)
}

func TestManager_SetStorage(t *testing.T) {
	manager := sagecrypto.NewManager()
	customStorage := sagecrypto.NewMemoryKeyStorage()

	// Should not panic
	manager.SetStorage(customStorage)
}

func TestManager_GenerateKeyPair(t *testing.T) {
	manager := sagecrypto.NewManager()

	t.Run("Generate Ed25519 key pair", func(t *testing.T) {
		keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
		require.NoError(t, err)
		assert.NotNil(t, keyPair)
		assert.Equal(t, sagecrypto.KeyTypeEd25519, keyPair.Type())
		assert.NotEmpty(t, keyPair.ID())
		assert.NotNil(t, keyPair.PublicKey())
		assert.NotNil(t, keyPair.PrivateKey())
	})

	t.Run("Generate Secp256k1 key pair", func(t *testing.T) {
		keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeSecp256k1)
		require.NoError(t, err)
		assert.NotNil(t, keyPair)
		assert.Equal(t, sagecrypto.KeyTypeSecp256k1, keyPair.Type())
		assert.NotEmpty(t, keyPair.ID())
		assert.NotNil(t, keyPair.PublicKey())
		assert.NotNil(t, keyPair.PrivateKey())
	})

	t.Run("Unsupported key type", func(t *testing.T) {
		_, err := manager.GenerateKeyPair(sagecrypto.KeyType("unsupported"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported key type")
	})

	t.Run("X25519 key type not supported by Manager", func(t *testing.T) {
		_, err := manager.GenerateKeyPair(sagecrypto.KeyTypeX25519)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported key type")
	})

	t.Run("RSA key type not supported by Manager", func(t *testing.T) {
		_, err := manager.GenerateKeyPair(sagecrypto.KeyTypeRSA)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported key type")
	})
}

func TestManager_StoreKeyPair(t *testing.T) {
	manager := sagecrypto.NewManager()

	keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
	require.NoError(t, err)

	err = manager.StoreKeyPair(keyPair)
	assert.NoError(t, err)
}

func TestManager_LoadKeyPair(t *testing.T) {
	manager := sagecrypto.NewManager()

	// Generate and store a key pair
	keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
	require.NoError(t, err)

	err = manager.StoreKeyPair(keyPair)
	require.NoError(t, err)

	t.Run("Load existing key pair", func(t *testing.T) {
		loadedKeyPair, err := manager.LoadKeyPair(keyPair.ID())
		assert.NoError(t, err)
		assert.NotNil(t, loadedKeyPair)
		assert.Equal(t, keyPair.ID(), loadedKeyPair.ID())
		assert.Equal(t, keyPair.Type(), loadedKeyPair.Type())
	})

	t.Run("Load non-existent key pair", func(t *testing.T) {
		_, err := manager.LoadKeyPair("non-existent-id")
		assert.Error(t, err)
		assert.Equal(t, sagecrypto.ErrKeyNotFound, err)
	})
}

func TestManager_DeleteKeyPair(t *testing.T) {
	manager := sagecrypto.NewManager()

	// Generate and store a key pair
	keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
	require.NoError(t, err)

	err = manager.StoreKeyPair(keyPair)
	require.NoError(t, err)

	t.Run("Delete existing key pair", func(t *testing.T) {
		err := manager.DeleteKeyPair(keyPair.ID())
		assert.NoError(t, err)

		// Verify it's deleted
		_, err = manager.LoadKeyPair(keyPair.ID())
		assert.Error(t, err)
		assert.Equal(t, sagecrypto.ErrKeyNotFound, err)
	})

	t.Run("Delete non-existent key pair", func(t *testing.T) {
		err := manager.DeleteKeyPair("non-existent-id")
		assert.Error(t, err)
		assert.Equal(t, sagecrypto.ErrKeyNotFound, err)
	})
}

func TestManager_ListKeyPairs(t *testing.T) {
	manager := sagecrypto.NewManager()

	t.Run("List empty storage", func(t *testing.T) {
		ids, err := manager.ListKeyPairs()
		assert.NoError(t, err)
		assert.Empty(t, ids)
	})

	t.Run("List with stored keys", func(t *testing.T) {
		// Generate and store multiple key pairs
		keyPair1, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
		require.NoError(t, err)
		err = manager.StoreKeyPair(keyPair1)
		require.NoError(t, err)

		keyPair2, err := manager.GenerateKeyPair(sagecrypto.KeyTypeSecp256k1)
		require.NoError(t, err)
		err = manager.StoreKeyPair(keyPair2)
		require.NoError(t, err)

		ids, err := manager.ListKeyPairs()
		assert.NoError(t, err)
		assert.Len(t, ids, 2)
		assert.Contains(t, ids, keyPair1.ID())
		assert.Contains(t, ids, keyPair2.ID())
	})
}

func TestManager_ExportKeyPair(t *testing.T) {
	manager := sagecrypto.NewManager()

	keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
	require.NoError(t, err)

	t.Run("Export as JWK", func(t *testing.T) {
		data, err := manager.ExportKeyPair(keyPair, sagecrypto.KeyFormatJWK)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		// Should be valid JSON
		assert.Contains(t, string(data), "kty")
	})

	t.Run("Export as PEM", func(t *testing.T) {
		data, err := manager.ExportKeyPair(keyPair, sagecrypto.KeyFormatPEM)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		// Should contain PEM markers
		assert.Contains(t, string(data), "BEGIN")
		assert.Contains(t, string(data), "END")
	})

	t.Run("Export with unsupported format", func(t *testing.T) {
		_, err := manager.ExportKeyPair(keyPair, sagecrypto.KeyFormat("unsupported"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported key format")
	})
}

func TestManager_ImportKeyPair(t *testing.T) {
	manager := sagecrypto.NewManager()

	// Generate a key pair and export it
	originalKeyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
	require.NoError(t, err)

	t.Run("Import from JWK", func(t *testing.T) {
		// Export as JWK
		jwkData, err := manager.ExportKeyPair(originalKeyPair, sagecrypto.KeyFormatJWK)
		require.NoError(t, err)

		// Import from JWK
		importedKeyPair, err := manager.ImportKeyPair(jwkData, sagecrypto.KeyFormatJWK)
		assert.NoError(t, err)
		assert.NotNil(t, importedKeyPair)
		assert.Equal(t, originalKeyPair.Type(), importedKeyPair.Type())

		// Verify the keys are functionally equivalent by signing and verifying
		message := []byte("test message")
		signature, err := originalKeyPair.Sign(message)
		require.NoError(t, err)

		err = importedKeyPair.Verify(message, signature)
		assert.NoError(t, err)
	})

	t.Run("Import from PEM", func(t *testing.T) {
		// Export as PEM
		pemData, err := manager.ExportKeyPair(originalKeyPair, sagecrypto.KeyFormatPEM)
		require.NoError(t, err)

		// Import from PEM
		importedKeyPair, err := manager.ImportKeyPair(pemData, sagecrypto.KeyFormatPEM)
		assert.NoError(t, err)
		assert.NotNil(t, importedKeyPair)
		assert.Equal(t, originalKeyPair.Type(), importedKeyPair.Type())

		// Verify the keys are functionally equivalent
		message := []byte("test message")
		signature, err := originalKeyPair.Sign(message)
		require.NoError(t, err)

		err = importedKeyPair.Verify(message, signature)
		assert.NoError(t, err)
	})

	t.Run("Import with unsupported format", func(t *testing.T) {
		_, err := manager.ImportKeyPair([]byte("data"), sagecrypto.KeyFormat("unsupported"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported key format")
	})

	t.Run("Import with invalid data", func(t *testing.T) {
		_, err := manager.ImportKeyPair([]byte("invalid data"), sagecrypto.KeyFormatJWK)
		assert.Error(t, err)
	})
}

func TestManager_Integration(t *testing.T) {
	manager := sagecrypto.NewManager()

	// Generate key pair
	keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
	require.NoError(t, err)

	// Store it
	err = manager.StoreKeyPair(keyPair)
	require.NoError(t, err)

	// Load it
	loadedKeyPair, err := manager.LoadKeyPair(keyPair.ID())
	require.NoError(t, err)

	// Export it
	jwkData, err := manager.ExportKeyPair(loadedKeyPair, sagecrypto.KeyFormatJWK)
	require.NoError(t, err)

	// Delete it
	err = manager.DeleteKeyPair(keyPair.ID())
	require.NoError(t, err)

	// Import it back
	importedKeyPair, err := manager.ImportKeyPair(jwkData, sagecrypto.KeyFormatJWK)
	require.NoError(t, err)

	// Store it again with its ID
	err = manager.StoreKeyPair(importedKeyPair)
	require.NoError(t, err)

	// Verify we can list it
	ids, err := manager.ListKeyPairs()
	require.NoError(t, err)
	assert.Contains(t, ids, importedKeyPair.ID())
}
