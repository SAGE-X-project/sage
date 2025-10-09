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


package crypto

import (
	"crypto"
	"errors"
	"time"
)

// KeyType represents the type of cryptographic key
type KeyType string

const (
	KeyTypeEd25519   KeyType = "Ed25519"
	KeyTypeSecp256k1 KeyType = "Secp256k1"
	KeyTypeX25519    KeyType = "X25519"
	KeyTypeRSA       KeyType = "RSA256"
)

// KeyFormat represents the format for key export/import
type KeyFormat string

const (
	KeyFormatJWK KeyFormat = "JWK"
	KeyFormatPEM KeyFormat = "PEM"
)

// KeyPair represents a cryptographic key pair
type KeyPair interface {
	// PublicKey returns the public key
	PublicKey() crypto.PublicKey
	
	// PrivateKey returns the private key
	PrivateKey() crypto.PrivateKey
	
	// Type returns the key type
	Type() KeyType
	
	// Sign signs the given message
	Sign(message []byte) ([]byte, error)
	
	// Verify verifies the signature
	Verify(message, signature []byte) error
	
	// ID returns a unique identifier for this key pair
	ID() string
}

// KeyExporter handles key export operations
type KeyExporter interface {
	// Export exports the key pair in the specified format
	Export(keyPair KeyPair, format KeyFormat) ([]byte, error)
	
	// ExportPublic exports only the public key
	ExportPublic(keyPair KeyPair, format KeyFormat) ([]byte, error)
}

// KeyImporter handles key import operations
type KeyImporter interface {
	// Import imports a key pair from the specified format
	Import(data []byte, format KeyFormat) (KeyPair, error)
	
	// ImportPublic imports only a public key
	ImportPublic(data []byte, format KeyFormat) (crypto.PublicKey, error)
}

// KeyStorage provides secure storage for keys
type KeyStorage interface {
	// Store stores a key pair with the given ID
	Store(id string, keyPair KeyPair) error
	
	// Load loads a key pair by ID
	Load(id string) (KeyPair, error)
	
	// Delete removes a key pair by ID
	Delete(id string) error
	
	// List returns all stored key IDs
	List() ([]string, error)
	
	// Exists checks if a key exists
	Exists(id string) bool
}

// KeyRotationConfig represents configuration for key rotation
type KeyRotationConfig struct {
	// RotationInterval is the time between rotations
	RotationInterval time.Duration
	
	// MaxKeyAge is the maximum age for a key
	MaxKeyAge time.Duration
	
	// KeepOldKeys determines if old keys should be kept
	KeepOldKeys bool
}

// KeyRotator handles key rotation operations
type KeyRotator interface {
	// Rotate rotates the key for the given ID
	Rotate(id string) (KeyPair, error)
	
	// SetRotationConfig sets the rotation configuration
	SetRotationConfig(config KeyRotationConfig)
	
	// GetRotationHistory returns the rotation history for a key
	GetRotationHistory(id string) ([]KeyRotationEvent, error)
}

// KeyRotationEvent represents a key rotation event
type KeyRotationEvent struct {
	Timestamp   time.Time
	OldKeyID    string
	NewKeyID    string
	Reason      string
}

// KeyManager is the main interface for key management
type KeyManager interface {
	// GenerateKeyPair generates a new key pair
	GenerateKeyPair(keyType KeyType) (KeyPair, error)
	
	// GetExporter returns the key exporter
	GetExporter() KeyExporter
	
	// GetImporter returns the key importer
	GetImporter() KeyImporter
	
	// GetStorage returns the key storage
	GetStorage() KeyStorage
	
	// GetRotator returns the key rotator
	GetRotator() KeyRotator
}

// Common errors
var (
	ErrKeyNotFound      = errors.New("key not found")
	ErrInvalidKeyType   = errors.New("invalid key type")
	ErrInvalidKeyFormat = errors.New("invalid key format")
	ErrKeyExists        = errors.New("key already exists")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrSignNotSupported   = errors.New("signing not supported")
    ErrVerifyNotSupported = errors.New("verification not supported")
)
