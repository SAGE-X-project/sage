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

package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

var (
	ErrKeyNotFound       = errors.New("key not found")
	ErrInvalidPassphrase = errors.New("invalid passphrase")
	ErrInvalidKeyID      = errors.New("invalid key ID")
	ErrEncryptionFailed  = errors.New("encryption failed")
	ErrDecryptionFailed  = errors.New("decryption failed")
)

// SecureVault defines the interface for secure key storage
type SecureVault interface {
	StoreEncrypted(keyID string, key []byte, passphrase string) error
	LoadDecrypted(keyID string, passphrase string) ([]byte, error)
	SetPermissions(keyID string, mode os.FileMode) error
	Delete(keyID string) error
	Exists(keyID string) bool
	ListKeys() []string
}

// EncryptedKeyData represents the structure of an encrypted key file
type EncryptedKeyData struct {
	Version    string    `json:"version"`
	KeyID      string    `json:"key_id"`
	Algorithm  string    `json:"algorithm"`
	Salt       string    `json:"salt"`
	IV         string    `json:"iv"`
	Ciphertext string    `json:"ciphertext"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// FileVault implements SecureVault using filesystem storage with AES-256 encryption
type FileVault struct {
	basePath string
	mu       sync.RWMutex
}

// NewFileVault creates a new file-based secure vault
func NewFileVault(basePath string) (*FileVault, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create vault directory: %w", err)
	}

	return &FileVault{
		basePath: basePath,
	}, nil
}

// StoreEncrypted encrypts and stores a key with AES-256-GCM
func (v *FileVault) StoreEncrypted(keyID string, key []byte, passphrase string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if keyID == "" {
		return ErrInvalidKeyID
	}

	// Generate salt for key derivation
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive encryption key from passphrase using PBKDF2
	derivedKey := pbkdf2.Key([]byte(passphrase), salt, 100000, 32, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the key
	ciphertext := gcm.Seal(nil, nonce, key, nil)

	// Create encrypted key data
	encData := EncryptedKeyData{
		Version:    "1.0",
		KeyID:      keyID,
		Algorithm:  "AES-256-GCM",
		Salt:       base64.StdEncoding.EncodeToString(salt),
		IV:         base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(encData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal encrypted data: %w", err)
	}

	// Write to file with restricted permissions
	filePath := v.getKeyPath(keyID)
	if err := os.WriteFile(filePath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write encrypted key: %w", err)
	}

	return nil
}

// LoadDecrypted loads and decrypts a key
func (v *FileVault) LoadDecrypted(keyID string, passphrase string) ([]byte, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if keyID == "" {
		return nil, ErrInvalidKeyID
	}

	filePath := v.getKeyPath(keyID)

	// Read encrypted data
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to read encrypted key: %w", err)
	}

	// Unmarshal JSON
	var encData EncryptedKeyData
	if err := json.Unmarshal(jsonData, &encData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal encrypted data: %w", err)
	}

	// Decode base64 values
	salt, err := base64.StdEncoding.DecodeString(encData.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(encData.IV)
	if err != nil {
		return nil, fmt.Errorf("failed to decode IV: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encData.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	// Derive decryption key from passphrase
	derivedKey := pbkdf2.Key([]byte(passphrase), salt, 100000, 32, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidPassphrase
	}

	return plaintext, nil
}

// SetPermissions sets file permissions for a key
func (v *FileVault) SetPermissions(keyID string, mode os.FileMode) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if keyID == "" {
		return ErrInvalidKeyID
	}

	filePath := v.getKeyPath(keyID)
	if err := os.Chmod(filePath, mode); err != nil {
		if os.IsNotExist(err) {
			return ErrKeyNotFound
		}
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

// Delete removes a key from the vault
func (v *FileVault) Delete(keyID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if keyID == "" {
		return ErrInvalidKeyID
	}

	filePath := v.getKeyPath(keyID)
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return ErrKeyNotFound
		}
		return fmt.Errorf("failed to delete key: %w", err)
	}

	return nil
}

// Exists checks if a key exists in the vault
func (v *FileVault) Exists(keyID string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if keyID == "" {
		return false
	}

	filePath := v.getKeyPath(keyID)
	_, err := os.Stat(filePath)
	return err == nil
}

// ListKeys returns a list of all key IDs in the vault
func (v *FileVault) ListKeys() []string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var keys []string

	files, err := os.ReadDir(v.basePath)
	if err != nil {
		return keys
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			keyID := file.Name()[:len(file.Name())-5] // Remove .json extension
			keys = append(keys, keyID)
		}
	}

	return keys
}

// getKeyPath returns the file path for a key
func (v *FileVault) getKeyPath(keyID string) string {
	// Sanitize keyID to prevent path traversal
	safeKeyID := filepath.Base(keyID)
	return filepath.Join(v.basePath, safeKeyID+".json")
}

// MemoryVault implements SecureVault using in-memory storage (for testing)
type MemoryVault struct {
	keys map[string][]byte
	mu   sync.RWMutex
}

// NewMemoryVault creates a new in-memory vault (primarily for testing)
func NewMemoryVault() *MemoryVault {
	return &MemoryVault{
		keys: make(map[string][]byte),
	}
}

// StoreEncrypted stores an encrypted key in memory
func (m *MemoryVault) StoreEncrypted(keyID string, key []byte, passphrase string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if keyID == "" {
		return ErrInvalidKeyID
	}

	// For memory vault, we'll use simple XOR encryption for testing
	encrypted := make([]byte, len(key))
	passphraseBytes := []byte(passphrase)
	for i := range key {
		encrypted[i] = key[i] ^ passphraseBytes[i%len(passphraseBytes)]
	}

	m.keys[keyID] = encrypted
	return nil
}

// LoadDecrypted loads and decrypts a key from memory
func (m *MemoryVault) LoadDecrypted(keyID string, passphrase string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if keyID == "" {
		return nil, ErrInvalidKeyID
	}

	encrypted, exists := m.keys[keyID]
	if !exists {
		return nil, ErrKeyNotFound
	}

	// Decrypt using XOR
	decrypted := make([]byte, len(encrypted))
	passphraseBytes := []byte(passphrase)
	for i := range encrypted {
		decrypted[i] = encrypted[i] ^ passphraseBytes[i%len(passphraseBytes)]
	}

	return decrypted, nil
}

// SetPermissions is a no-op for memory vault
func (m *MemoryVault) SetPermissions(keyID string, mode os.FileMode) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.keys[keyID]; !exists {
		return ErrKeyNotFound
	}
	return nil
}

// Delete removes a key from memory
func (m *MemoryVault) Delete(keyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if keyID == "" {
		return ErrInvalidKeyID
	}

	if _, exists := m.keys[keyID]; !exists {
		return ErrKeyNotFound
	}

	delete(m.keys, keyID)
	return nil
}

// Exists checks if a key exists in memory
func (m *MemoryVault) Exists(keyID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.keys[keyID]
	return exists
}

// ListKeys returns all key IDs in memory
func (m *MemoryVault) ListKeys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.keys))
	for keyID := range m.keys {
		keys = append(keys, keyID)
	}
	return keys
}
