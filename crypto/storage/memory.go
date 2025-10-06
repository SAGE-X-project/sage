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


package storage

import (
	"sort"
	"sync"

	sagecrypto "github.com/sage-x-project/sage/crypto"
)

// memoryKeyStorage implements KeyStorage interface using in-memory storage
type memoryKeyStorage struct {
	keys map[string]sagecrypto.KeyPair
	mu   sync.RWMutex
}

// NewMemoryKeyStorage creates a new in-memory key storage
func NewMemoryKeyStorage() sagecrypto.KeyStorage {
	return &memoryKeyStorage{
		keys: make(map[string]sagecrypto.KeyPair),
	}
}

// Store stores a key pair with the given ID
func (s *memoryKeyStorage) Store(id string, keyPair sagecrypto.KeyPair) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys[id] = keyPair
	return nil
}

// Load loads a key pair by ID
func (s *memoryKeyStorage) Load(id string) (sagecrypto.KeyPair, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keyPair, exists := s.keys[id]
	if !exists {
		return nil, sagecrypto.ErrKeyNotFound
	}

	return keyPair, nil
}

// Delete removes a key pair by ID
func (s *memoryKeyStorage) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.keys[id]; !exists {
		return sagecrypto.ErrKeyNotFound
	}

	delete(s.keys, id)
	return nil
}

// List returns all stored key IDs in sorted order
func (s *memoryKeyStorage) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.keys))
	for id := range s.keys {
		ids = append(ids, id)
	}
	
	// Sort for consistent output
	sort.Strings(ids)

	return ids, nil
}

// Exists checks if a key exists
func (s *memoryKeyStorage) Exists(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, exists := s.keys[id]
	return exists
}