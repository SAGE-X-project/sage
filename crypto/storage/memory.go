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