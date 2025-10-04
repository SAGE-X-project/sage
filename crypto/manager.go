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


package crypto

import (
	"fmt"
)

// Manager provides centralized management of cryptographic operations
type Manager struct {
	storage KeyStorage
}

// NewManager creates a new crypto manager
func NewManager() *Manager {
	return &Manager{
		storage: NewMemoryKeyStorage(),
	}
}

// SetStorage sets the key storage backend
func (m *Manager) SetStorage(storage KeyStorage) {
	m.storage = storage
}

// GenerateKeyPair generates a new key pair of the specified type
func (m *Manager) GenerateKeyPair(keyType KeyType) (KeyPair, error) {
	switch keyType {
	case KeyTypeEd25519:
		return GenerateEd25519KeyPair()
	case KeyTypeSecp256k1:
		return GenerateSecp256k1KeyPair()
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}
}

// StoreKeyPair stores a key pair
func (m *Manager) StoreKeyPair(keyPair KeyPair) error {
	return m.storage.Store(keyPair.ID(), keyPair)
}

// LoadKeyPair loads a key pair by ID
func (m *Manager) LoadKeyPair(id string) (KeyPair, error) {
	return m.storage.Load(id)
}

// DeleteKeyPair deletes a key pair by ID
func (m *Manager) DeleteKeyPair(id string) error {
	return m.storage.Delete(id)
}

// ListKeyPairs lists all stored key pair IDs
func (m *Manager) ListKeyPairs() ([]string, error) {
	return m.storage.List()
}

// ExportKeyPair exports a key pair in the specified format
func (m *Manager) ExportKeyPair(keyPair KeyPair, format KeyFormat) ([]byte, error) {
	var exporter KeyExporter
	switch format {
	case KeyFormatJWK:
		exporter = NewJWKExporter()
	case KeyFormatPEM:
		exporter = NewPEMExporter()
	default:
		return nil, fmt.Errorf("unsupported key format: %s", format)
	}
	
	return exporter.Export(keyPair, format)
}

// ImportKeyPair imports a key pair from the specified format
func (m *Manager) ImportKeyPair(data []byte, format KeyFormat) (KeyPair, error) {
	var importer KeyImporter
	switch format {
	case KeyFormatJWK:
		importer = NewJWKImporter()
	case KeyFormatPEM:
		importer = NewPEMImporter()
	default:
		return nil, fmt.Errorf("unsupported key format: %s", format)
	}
	
	return importer.Import(data, format)
}