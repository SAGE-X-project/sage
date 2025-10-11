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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
)

// fileKeyStorage implements KeyStorage interface using file system
type fileKeyStorage struct {
	directory string
	exporter  sagecrypto.KeyExporter
	importer  sagecrypto.KeyImporter
	mu        sync.RWMutex
}

// keyFileData represents the structure of a key file
type keyFileData struct {
	Type   sagecrypto.KeyType   `json:"type"`
	Format sagecrypto.KeyFormat `json:"format"`
	Data   string               `json:"data"`
	ID     string               `json:"id"`
}

// NewFileKeyStorage creates a new file-based key storage
func NewFileKeyStorage(directory string) (sagecrypto.KeyStorage, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, fmt.Errorf("failed to create key storage directory: %w", err)
	}

	return &fileKeyStorage{
		directory: directory,
		exporter:  formats.NewJWKExporter(),
		importer:  formats.NewJWKImporter(),
	}, nil
}

// validateKeyID validates that a key ID is safe for filesystem use
func validateKeyID(id string) error {
	if strings.Contains(id, "/") || strings.Contains(id, "\\") || strings.Contains(id, "..") {
		return fmt.Errorf("invalid key ID: %s", id)
	}
	return nil
}

// Store stores a key pair with the given ID
func (s *fileKeyStorage) Store(id string, keyPair sagecrypto.KeyPair) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate ID (no path traversal)
	if err := validateKeyID(id); err != nil {
		return err
	}

	// Export key to JWK format
	jwkData, err := s.exporter.Export(keyPair, sagecrypto.KeyFormatJWK)
	if err != nil {
		return fmt.Errorf("failed to export key: %w", err)
	}

	// Create key file data
	fileData := keyFileData{
		Type:   keyPair.Type(),
		Format: sagecrypto.KeyFormatJWK,
		Data:   string(jwkData),
		ID:     keyPair.ID(),
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal key data: %w", err)
	}

	// Write to file with secure permissions
	filename := filepath.Join(s.directory, id+".key")
	if err := os.WriteFile(filename, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

// Load loads a key pair by ID
func (s *fileKeyStorage) Load(id string) (sagecrypto.KeyPair, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Validate ID
	if err := validateKeyID(id); err != nil {
		return nil, err
	}

	filename := filepath.Join(s.directory, id+".key")

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, sagecrypto.ErrKeyNotFound
	}

	// Read file
	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Unmarshal JSON
	var fileData keyFileData
	if err := json.Unmarshal(jsonData, &fileData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal key data: %w", err)
	}

	// Import key from stored format
	keyPair, err := s.importer.Import([]byte(fileData.Data), fileData.Format)
	if err != nil {
		return nil, fmt.Errorf("failed to import key: %w", err)
	}

	return keyPair, nil
}

// Delete removes a key pair by ID
func (s *fileKeyStorage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate ID
	if err := validateKeyID(id); err != nil {
		return err
	}

	filename := filepath.Join(s.directory, id+".key")

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return sagecrypto.ErrKeyNotFound
	}

	// Remove file
	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("failed to delete key file: %w", err)
	}

	return nil
}

// List returns all stored key IDs in sorted order
func (s *fileKeyStorage) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read key directory: %w", err)
	}

	var ids []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".key") {
			// Remove .key extension
			id := strings.TrimSuffix(entry.Name(), ".key")
			ids = append(ids, id)
		}
	}

	// Sort for consistent output
	sort.Strings(ids)

	return ids, nil
}

// Exists checks if a key exists
func (s *fileKeyStorage) Exists(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Validate ID
	if err := validateKeyID(id); err != nil {
		return false
	}

	filename := filepath.Join(s.directory, id+".key")
	_, err := os.Stat(filename)
	return err == nil
}
