package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sage-x-project/sage/pkg/storage"
)

// Store implements the storage.Store interface with in-memory storage
type Store struct {
	sessions map[string]*storage.Session
	nonces   map[string]*storage.Nonce
	dids     map[string]*storage.DID

	sessionsMu sync.RWMutex
	noncesMu   sync.RWMutex
	didsMu     sync.RWMutex

	sessionStore *SessionStore
	nonceStore   *NonceStore
	didStore     *DIDStore
}

// NewStore creates a new in-memory store
func NewStore() *Store {
	s := &Store{
		sessions: make(map[string]*storage.Session),
		nonces:   make(map[string]*storage.Nonce),
		dids:     make(map[string]*storage.DID),
	}

	s.sessionStore = &SessionStore{store: s}
	s.nonceStore = &NonceStore{store: s}
	s.didStore = &DIDStore{store: s}

	return s
}

// SessionStore returns the session store
func (s *Store) SessionStore() storage.SessionStore {
	return s.sessionStore
}

// NonceStore returns the nonce store
func (s *Store) NonceStore() storage.NonceStore {
	return s.nonceStore
}

// DIDStore returns the DID store
func (s *Store) DIDStore() storage.DIDStore {
	return s.didStore
}

// Close closes the store (no-op for memory store)
func (s *Store) Close() error {
	return nil
}

// Ping checks the store (always succeeds for memory store)
func (s *Store) Ping(ctx context.Context) error {
	return nil
}

// Clear removes all data (useful for testing)
func (s *Store) Clear() {
	s.sessionsMu.Lock()
	s.sessions = make(map[string]*storage.Session)
	s.sessionsMu.Unlock()

	s.noncesMu.Lock()
	s.nonces = make(map[string]*storage.Nonce)
	s.noncesMu.Unlock()

	s.didsMu.Lock()
	s.dids = make(map[string]*storage.DID)
	s.didsMu.Unlock()
}

// SessionStore implements storage.SessionStore
type SessionStore struct {
	store *Store
}

func (s *SessionStore) Create(ctx context.Context, session *storage.Session) error {
	s.store.sessionsMu.Lock()
	defer s.store.sessionsMu.Unlock()

	if _, exists := s.store.sessions[session.ID]; exists {
		return fmt.Errorf("session already exists: %s", session.ID)
	}

	// Deep copy to avoid external modifications
	copy := *session
	if session.SessionKey != nil {
		copy.SessionKey = make([]byte, len(session.SessionKey))
		copyBytes := copy(copy.SessionKey, session.SessionKey)
		_ = copyBytes
	}
	if session.Metadata != nil {
		copy.Metadata = make(map[string]interface{})
		for k, v := range session.Metadata {
			copy.Metadata[k] = v
		}
	}

	s.store.sessions[session.ID] = &copy
	return nil
}

func (s *SessionStore) Get(ctx context.Context, id string) (*storage.Session, error) {
	s.store.sessionsMu.RLock()
	defer s.store.sessionsMu.RUnlock()

	session, exists := s.store.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired: %s", id)
	}

	// Return copy
	copy := *session
	return &copy, nil
}

func (s *SessionStore) Update(ctx context.Context, session *storage.Session) error {
	s.store.sessionsMu.Lock()
	defer s.store.sessionsMu.Unlock()

	if _, exists := s.store.sessions[session.ID]; !exists {
		return fmt.Errorf("session not found: %s", session.ID)
	}

	copy := *session
	s.store.sessions[session.ID] = &copy
	return nil
}

func (s *SessionStore) Delete(ctx context.Context, id string) error {
	s.store.sessionsMu.Lock()
	defer s.store.sessionsMu.Unlock()

	if _, exists := s.store.sessions[id]; !exists {
		return fmt.Errorf("session not found: %s", id)
	}

	delete(s.store.sessions, id)
	return nil
}

func (s *SessionStore) DeleteExpired(ctx context.Context) (int64, error) {
	s.store.sessionsMu.Lock()
	defer s.store.sessionsMu.Unlock()

	now := time.Now()
	var count int64

	for id, session := range s.store.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.store.sessions, id)
			count++
		}
	}

	return count, nil
}

func (s *SessionStore) List(ctx context.Context, clientDID string, limit, offset int) ([]*storage.Session, error) {
	s.store.sessionsMu.RLock()
	defer s.store.sessionsMu.RUnlock()

	var sessions []*storage.Session
	now := time.Now()

	for _, session := range s.store.sessions {
		if session.ClientDID == clientDID && now.Before(session.ExpiresAt) {
			copy := *session
			sessions = append(sessions, &copy)
		}
	}

	// Apply pagination
	if offset >= len(sessions) {
		return []*storage.Session{}, nil
	}

	end := offset + limit
	if end > len(sessions) {
		end = len(sessions)
	}

	return sessions[offset:end], nil
}

func (s *SessionStore) UpdateActivity(ctx context.Context, id string) error {
	s.store.sessionsMu.Lock()
	defer s.store.sessionsMu.Unlock()

	session, exists := s.store.sessions[id]
	if !exists {
		return fmt.Errorf("session not found: %s", id)
	}

	session.LastActivity = time.Now()
	return nil
}

func (s *SessionStore) Count(ctx context.Context) (int64, error) {
	s.store.sessionsMu.RLock()
	defer s.store.sessionsMu.RUnlock()

	now := time.Now()
	var count int64

	for _, session := range s.store.sessions {
		if now.Before(session.ExpiresAt) {
			count++
		}
	}

	return count, nil
}
