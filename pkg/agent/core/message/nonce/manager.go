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

package nonce

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// Manager prevents nonce reuse within TTL window
type Manager struct {
	ttl             time.Duration
	mu              sync.RWMutex
	usedNonces      map[string]time.Time
	lastCleanup     time.Time
	cleanupInterval time.Duration
	stop            chan struct{}
	stopOnce        sync.Once
}

// NewManager creates a new nonce tracker with the given TTL
func NewManager(ttl, cleanupInterval time.Duration) *Manager {
	m := &Manager{
		ttl:             ttl,
		usedNonces:      make(map[string]time.Time),
		cleanupInterval: cleanupInterval,
		lastCleanup:     time.Now(),
		stop:            make(chan struct{}),
	}
	go m.cleanupLoop()
	return m
}

// CheckAndMark atomically reports whether nonce is fresh and, if so, records it.
// It returns false when the nonce was already seen within the TTL window.
// Use this instead of IsNonceUsed followed by MarkNonceUsed, which is racy.
func (m *Manager) CheckAndMark(nonce string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	if seen, exists := m.usedNonces[nonce]; exists && now.Sub(seen) <= m.ttl {
		return false
	}
	m.usedNonces[nonce] = now
	return true
}

// Close stops the background cleanup goroutine. It is safe to call more than once.
func (m *Manager) Close() {
	m.stopOnce.Do(func() { close(m.stop) })
}

// GenerateNonce returns a cryptographically secure random nonce,
// encoded in Base64URL without padding. The sizeBytes specifies
// the length in bytes of the raw nonce (e.g., 16 for 128 bits).
func GenerateNonce() (string, error) {
	const size = 16
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	nonce := base64.RawURLEncoding.EncodeToString(b)
	return nonce, nil
}

// IsNonceUsed checks if a nonce has been used within the TTL window
func (m *Manager) IsNonceUsed(nonce string) bool {
	m.mu.RLock()
	timestamp, exists := m.usedNonces[nonce]
	m.mu.RUnlock()

	if !exists {
		return false
	}

	// Check if nonce is expired
	if time.Since(timestamp) > m.ttl {
		m.mu.Lock()
		delete(m.usedNonces, nonce)
		m.mu.Unlock()
		return false
	}

	return true
}

// MarkNonceUsed records a nonce as used
func (m *Manager) MarkNonceUsed(nonce string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.usedNonces[nonce] = time.Now()
}

// GetUsedNonceCount returns the number of nonces currently being tracked
func (m *Manager) GetUsedNonceCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.usedNonces)
}

// cleanupLoop periodically removes expired nonces
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.performCleanup()
		case <-m.stop:
			return
		}
	}
}

// performCleanup removes expired nonces
func (m *Manager) performCleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for nonce, timestamp := range m.usedNonces {
		if now.Sub(timestamp) > m.ttl {
			delete(m.usedNonces, nonce)
		}
	}
	m.lastCleanup = now
}
