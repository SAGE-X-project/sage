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