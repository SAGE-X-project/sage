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
	"crypto"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/sage-x-project/sage/internal/cryptoinit" // Initialize wrappers
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/crypto/storage"
)

// Mock implementations for testing
type mockKeyPair struct {
	id         string
	keyType    sagecrypto.KeyType
	publicKey  crypto.PublicKey
	privateKey crypto.PrivateKey
}

func (m *mockKeyPair) ID() string                             { return m.id }
func (m *mockKeyPair) Type() sagecrypto.KeyType               { return m.keyType }
func (m *mockKeyPair) PublicKey() crypto.PublicKey            { return m.publicKey }
func (m *mockKeyPair) PrivateKey() crypto.PrivateKey          { return m.privateKey }
func (m *mockKeyPair) Sign(message []byte) ([]byte, error)    { return nil, nil }
func (m *mockKeyPair) Verify(message, signature []byte) error { return nil }

type mockKeyStorage struct {
	data map[string]sagecrypto.KeyPair
}

func (m *mockKeyStorage) Store(id string, keyPair sagecrypto.KeyPair) error {
	m.data[id] = keyPair
	return nil
}

func (m *mockKeyStorage) Load(id string) (sagecrypto.KeyPair, error) {
	kp, exists := m.data[id]
	if !exists {
		return nil, sagecrypto.ErrKeyNotFound
	}
	return kp, nil
}

func (m *mockKeyStorage) Delete(id string) error {
	if _, exists := m.data[id]; !exists {
		return sagecrypto.ErrKeyNotFound
	}
	delete(m.data, id)
	return nil
}

func (m *mockKeyStorage) List() ([]string, error) {
	ids := make([]string, 0, len(m.data))
	for id := range m.data {
		ids = append(ids, id)
	}
	return ids, nil
}

func (m *mockKeyStorage) Exists(id string) bool {
	_, exists := m.data[id]
	return exists
}

type mockKeyExporter struct{}

func (m *mockKeyExporter) Export(keyPair sagecrypto.KeyPair, format sagecrypto.KeyFormat) ([]byte, error) {
	return []byte("exported"), nil
}

func (m *mockKeyExporter) ExportPublic(keyPair sagecrypto.KeyPair, format sagecrypto.KeyFormat) ([]byte, error) {
	return []byte("exported-public"), nil
}

type mockKeyImporter struct{}

func (m *mockKeyImporter) Import(data []byte, format sagecrypto.KeyFormat) (sagecrypto.KeyPair, error) {
	return &mockKeyPair{id: "imported", keyType: sagecrypto.KeyTypeEd25519}, nil
}

func (m *mockKeyImporter) ImportPublic(data []byte, format sagecrypto.KeyFormat) (crypto.PublicKey, error) {
	return nil, nil
}

func TestSetKeyGenerators(t *testing.T) {
	// Restore original generators at the end
	defer func() {
		sagecrypto.SetKeyGenerators(
			func() (sagecrypto.KeyPair, error) { return keys.GenerateEd25519KeyPair() },
			func() (sagecrypto.KeyPair, error) { return keys.GenerateSecp256k1KeyPair() },
		)
	}()

	// Set custom generators
	mockEd25519Called := false
	mockSecp256k1Called := false

	sagecrypto.SetKeyGenerators(
		func() (sagecrypto.KeyPair, error) {
			mockEd25519Called = true
			return &mockKeyPair{id: "mock-ed25519", keyType: sagecrypto.KeyTypeEd25519}, nil
		},
		func() (sagecrypto.KeyPair, error) {
			mockSecp256k1Called = true
			return &mockKeyPair{id: "mock-secp256k1", keyType: sagecrypto.KeyTypeSecp256k1}, nil
		},
	)

	// Test Ed25519 generator
	keyPair, err := sagecrypto.GenerateEd25519KeyPair()
	assert.NoError(t, err)
	assert.True(t, mockEd25519Called)
	assert.Equal(t, "mock-ed25519", keyPair.ID())

	// Test Secp256k1 generator
	keyPair, err = sagecrypto.GenerateSecp256k1KeyPair()
	assert.NoError(t, err)
	assert.True(t, mockSecp256k1Called)
	assert.Equal(t, "mock-secp256k1", keyPair.ID())
}

func TestSetStorageConstructors(t *testing.T) {
	// Restore original constructor at the end
	defer func() {
		sagecrypto.SetStorageConstructors(
			func() sagecrypto.KeyStorage { return storage.NewMemoryKeyStorage() },
		)
	}()

	// Set custom constructor
	mockStorageCalled := false

	sagecrypto.SetStorageConstructors(
		func() sagecrypto.KeyStorage {
			mockStorageCalled = true
			return &mockKeyStorage{data: make(map[string]sagecrypto.KeyPair)}
		},
	)

	// Test storage constructor
	str := sagecrypto.NewMemoryKeyStorage()
	assert.NotNil(t, str)
	assert.True(t, mockStorageCalled)
}

func TestSetFormatConstructors(t *testing.T) {
	// Restore original constructors at the end
	defer func() {
		sagecrypto.SetFormatConstructors(
			func() sagecrypto.KeyExporter { return formats.NewJWKExporter() },
			func() sagecrypto.KeyExporter { return formats.NewPEMExporter() },
			func() sagecrypto.KeyImporter { return formats.NewJWKImporter() },
			func() sagecrypto.KeyImporter { return formats.NewPEMImporter() },
		)
	}()

	// Set custom format constructors
	mockJWKExporterCalled := false
	mockPEMExporterCalled := false
	mockJWKImporterCalled := false
	mockPEMImporterCalled := false

	sagecrypto.SetFormatConstructors(
		func() sagecrypto.KeyExporter {
			mockJWKExporterCalled = true
			return &mockKeyExporter{}
		},
		func() sagecrypto.KeyExporter {
			mockPEMExporterCalled = true
			return &mockKeyExporter{}
		},
		func() sagecrypto.KeyImporter {
			mockJWKImporterCalled = true
			return &mockKeyImporter{}
		},
		func() sagecrypto.KeyImporter {
			mockPEMImporterCalled = true
			return &mockKeyImporter{}
		},
	)

	// Test JWK exporter
	exporter := sagecrypto.NewJWKExporter()
	assert.NotNil(t, exporter)
	assert.True(t, mockJWKExporterCalled)

	// Test PEM exporter
	exporter = sagecrypto.NewPEMExporter()
	assert.NotNil(t, exporter)
	assert.True(t, mockPEMExporterCalled)

	// Test JWK importer
	importer := sagecrypto.NewJWKImporter()
	assert.NotNil(t, importer)
	assert.True(t, mockJWKImporterCalled)

	// Test PEM importer
	importer = sagecrypto.NewPEMImporter()
	assert.NotNil(t, importer)
	assert.True(t, mockPEMImporterCalled)
}

func TestNewEd25519KeyPair(t *testing.T) {
	keyPair, err := sagecrypto.NewEd25519KeyPair()
	assert.NoError(t, err)
	assert.NotNil(t, keyPair)
	assert.Equal(t, sagecrypto.KeyTypeEd25519, keyPair.Type())
}

func TestNewSecp256k1KeyPair(t *testing.T) {
	keyPair, err := sagecrypto.NewSecp256k1KeyPair()
	assert.NoError(t, err)
	assert.NotNil(t, keyPair)
	assert.Equal(t, sagecrypto.KeyTypeSecp256k1, keyPair.Type())
}

func TestGenerateEd25519KeyPair(t *testing.T) {
	// This should be an alias for NewEd25519KeyPair
	keyPair, err := sagecrypto.GenerateEd25519KeyPair()
	assert.NoError(t, err)
	assert.NotNil(t, keyPair)
	assert.Equal(t, sagecrypto.KeyTypeEd25519, keyPair.Type())
}

func TestGenerateSecp256k1KeyPair(t *testing.T) {
	// This should be an alias for NewSecp256k1KeyPair
	keyPair, err := sagecrypto.GenerateSecp256k1KeyPair()
	assert.NoError(t, err)
	assert.NotNil(t, keyPair)
	assert.Equal(t, sagecrypto.KeyTypeSecp256k1, keyPair.Type())
}

func TestNewMemoryKeyStorage(t *testing.T) {
	storage := sagecrypto.NewMemoryKeyStorage()
	assert.NotNil(t, storage)
}

func TestNewJWKExporter(t *testing.T) {
	exporter := sagecrypto.NewJWKExporter()
	assert.NotNil(t, exporter)
}

func TestNewPEMExporter(t *testing.T) {
	exporter := sagecrypto.NewPEMExporter()
	assert.NotNil(t, exporter)
}

func TestNewJWKImporter(t *testing.T) {
	importer := sagecrypto.NewJWKImporter()
	assert.NotNil(t, importer)
}

func TestNewPEMImporter(t *testing.T) {
	importer := sagecrypto.NewPEMImporter()
	assert.NotNil(t, importer)
}

func TestPanicOnUninitializedGenerators(t *testing.T) {
	// Restore original generators at the end
	defer func() {
		sagecrypto.SetKeyGenerators(
			func() (sagecrypto.KeyPair, error) { return keys.GenerateEd25519KeyPair() },
			func() (sagecrypto.KeyPair, error) { return keys.GenerateSecp256k1KeyPair() },
		)
	}()

	// Set generators to nil
	sagecrypto.SetKeyGenerators(nil, nil)

	t.Run("Panic on uninitialized Ed25519 generator", func(t *testing.T) {
		assert.Panics(t, func() {
			_, _ = sagecrypto.NewEd25519KeyPair()
		})
	})

	t.Run("Panic on uninitialized Secp256k1 generator", func(t *testing.T) {
		assert.Panics(t, func() {
			_, _ = sagecrypto.NewSecp256k1KeyPair()
		})
	})
}

func TestPanicOnUninitializedStorage(t *testing.T) {
	// Restore original constructor at the end
	defer func() {
		sagecrypto.SetStorageConstructors(
			func() sagecrypto.KeyStorage { return storage.NewMemoryKeyStorage() },
		)
	}()

	// Set constructor to nil
	sagecrypto.SetStorageConstructors(nil)

	t.Run("Panic on uninitialized storage constructor", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = sagecrypto.NewMemoryKeyStorage()
		})
	})
}

func TestPanicOnUninitializedFormatConstructors(t *testing.T) {
	// Restore original constructors at the end
	defer func() {
		sagecrypto.SetFormatConstructors(
			func() sagecrypto.KeyExporter { return formats.NewJWKExporter() },
			func() sagecrypto.KeyExporter { return formats.NewPEMExporter() },
			func() sagecrypto.KeyImporter { return formats.NewJWKImporter() },
			func() sagecrypto.KeyImporter { return formats.NewPEMImporter() },
		)
	}()

	// Set all format constructors to nil
	sagecrypto.SetFormatConstructors(nil, nil, nil, nil)

	t.Run("Panic on uninitialized JWK exporter", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = sagecrypto.NewJWKExporter()
		})
	})

	t.Run("Panic on uninitialized PEM exporter", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = sagecrypto.NewPEMExporter()
		})
	})

	t.Run("Panic on uninitialized JWK importer", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = sagecrypto.NewJWKImporter()
		})
	})

	t.Run("Panic on uninitialized PEM importer", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = sagecrypto.NewPEMImporter()
		})
	})
}

func TestWrappers_ErrorHandling(t *testing.T) {
	// Test that errors from generators are properly propagated

	// Restore original generators at the end
	defer func() {
		sagecrypto.SetKeyGenerators(
			func() (sagecrypto.KeyPair, error) { return keys.GenerateEd25519KeyPair() },
			func() (sagecrypto.KeyPair, error) { return keys.GenerateSecp256k1KeyPair() },
		)
	}()

	// Set generators that return errors
	expectedError := errors.New("test error")
	sagecrypto.SetKeyGenerators(
		func() (sagecrypto.KeyPair, error) {
			return nil, expectedError
		},
		func() (sagecrypto.KeyPair, error) {
			return nil, expectedError
		},
	)

	t.Run("Ed25519 generator error propagation", func(t *testing.T) {
		_, err := sagecrypto.GenerateEd25519KeyPair()
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
	})

	t.Run("Secp256k1 generator error propagation", func(t *testing.T) {
		_, err := sagecrypto.GenerateSecp256k1KeyPair()
		assert.Error(t, err)
		assert.Equal(t, expectedError, err)
	})
}

func TestWrappers_Integration(t *testing.T) {
	// Test that all wrappers work together

	// Generate Ed25519 key pair
	ed25519KeyPair, err := sagecrypto.GenerateEd25519KeyPair()
	require.NoError(t, err)
	require.NotNil(t, ed25519KeyPair)

	// Generate Secp256k1 key pair
	secp256k1KeyPair, err := sagecrypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	require.NotNil(t, secp256k1KeyPair)

	// Create storage
	storage := sagecrypto.NewMemoryKeyStorage()
	require.NotNil(t, storage)

	// Store keys
	err = storage.Store(ed25519KeyPair.ID(), ed25519KeyPair)
	require.NoError(t, err)
	err = storage.Store(secp256k1KeyPair.ID(), secp256k1KeyPair)
	require.NoError(t, err)

	// List keys
	ids, err := storage.List()
	require.NoError(t, err)
	assert.Len(t, ids, 2)

	// Create exporters and importers
	jwkExporter := sagecrypto.NewJWKExporter()
	require.NotNil(t, jwkExporter)

	pemExporter := sagecrypto.NewPEMExporter()
	require.NotNil(t, pemExporter)

	jwkImporter := sagecrypto.NewJWKImporter()
	require.NotNil(t, jwkImporter)

	pemImporter := sagecrypto.NewPEMImporter()
	require.NotNil(t, pemImporter)

	// Export and import keys
	jwkData, err := jwkExporter.Export(ed25519KeyPair, sagecrypto.KeyFormatJWK)
	require.NoError(t, err)

	importedKeyPair, err := jwkImporter.Import(jwkData, sagecrypto.KeyFormatJWK)
	require.NoError(t, err)
	assert.NotNil(t, importedKeyPair)
}
