package session

import (
	"fmt"
	"sync"
	"time"
)

// Manager handles session lifecycle, storage, and cleanup
type Manager struct {
    sessions       map[string]Session
    mu             sync.RWMutex
    cleanupTicker  *time.Ticker
    stopCleanup    chan struct{}
    defaultConfig  Config
}

// NewManager creates a new session manager with default configuration
func NewManager() *Manager {
    m := &Manager{
        sessions:     make(map[string]Session),
        stopCleanup:  make(chan struct{}),
        defaultConfig: Config{
            MaxAge:      time.Hour,        // 1-hour absolute expiration
            IdleTimeout: 10 * time.Minute, // 10-minute idle timeout
            MaxMessages: 1000,            
        },
    }
    
    // Start background cleanup every 30 seconds
    m.cleanupTicker = time.NewTicker(30 * time.Second)
    go m.runCleanup()
    
    return m
}

// CreateSession creates a new session with the given shared secret
func (m *Manager) CreateSession(sessionID string, sharedSecret []byte) (Session, error) {
    return m.CreateSessionWithConfig(sessionID, sharedSecret, m.defaultConfig)
}

// CreateSessionWithConfig creates a new session with custom configuration
func (m *Manager) CreateSessionWithConfig(sessionID string, sharedSecret []byte, config Config) (Session, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Check if session already exists
    if _, exists := m.sessions[sessionID]; exists {
        return nil, fmt.Errorf("session %s already exists", sessionID)
    }
    
    // Create new crypto session
    sess, err := NewSecureSession(sessionID, sharedSecret, config)
    if err != nil {
        return nil, fmt.Errorf("failed to create session: %w", err)
    }
    
    // Store in manager
    m.sessions[sessionID] = sess
    
    return sess, nil
}

// GetSession retrieves a session by ID, returns nil if not found or expired
func (m *Manager) GetSession(sessionID string) (Session, bool) {
    m.mu.RLock()
    sess, exists := m.sessions[sessionID]
    m.mu.RUnlock()
    
    if !exists {
        return nil, false
    }
    
    if sess.IsExpired() {
        // Remove expired session
        m.RemoveSession(sessionID)
        return nil, false
    }
    
    return sess, true
}

// RemoveSession removes a session from the manager
func (m *Manager) RemoveSession(sessionID string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if sess, exists := m.sessions[sessionID]; exists {
        sess.Close() // Clean up resources
        delete(m.sessions, sessionID)
    }
}

// ListSessions returns all active session IDs
func (m *Manager) ListSessions() []string {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    var sessionIDs []string
    for id := range m.sessions {
        sessionIDs = append(sessionIDs, id)
    }
    
    return sessionIDs
}

// GetSessionCount returns the number of active sessions
func (m *Manager) GetSessionCount() int {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return len(m.sessions)
}

// GetSessionStats returns statistics about sessions
func (m *Manager) GetSessionStats() SessionStats {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    stats := SessionStats{
        TotalSessions: len(m.sessions),
        ActiveSessions: 0,
        ExpiredSessions: 0,
    }
    
    for _, sess := range m.sessions {
        if sess.IsExpired() {
            stats.ExpiredSessions++
        } else {
            stats.ActiveSessions++
        }
    }
    
    return stats
}

// SetDefaultConfig updates the default session configuration
func (m *Manager) SetDefaultConfig(config Config) {
    m.defaultConfig = config
}

// Close stops the manager and cleans up all sessions
func (m *Manager) Close() error {
    // Stop cleanup goroutine
    close(m.stopCleanup)
    if m.cleanupTicker != nil {
        m.cleanupTicker.Stop()
    }
    
    // Close all sessions
    m.mu.Lock()
    defer m.mu.Unlock()
    
    for _, sess := range m.sessions {
        sess.Close()
    }
    
    m.sessions = make(map[string]Session)
    
    return nil
}

// runCleanup runs in background to remove expired sessions
func (m *Manager) runCleanup() {
    for {
        select {
        case <-m.cleanupTicker.C:
            m.cleanupExpiredSessions()
        case <-m.stopCleanup:
            return
        }
    }
}

// cleanupExpiredSessions removes all expired sessions
func (m *Manager) cleanupExpiredSessions() {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    var expiredIDs []string
    
    // Find expired sessions
    for id, sess := range m.sessions {
        if sess.IsExpired() {
            expiredIDs = append(expiredIDs, id)
        }
    }
    
    // Remove expired sessions
    for _, id := range expiredIDs {
        if sess, exists := m.sessions[id]; exists {
            sess.Close()
            delete(m.sessions, id)
        }
    }
    
    if len(expiredIDs) > 0 {
        // Log cleanup activity
        fmt.Printf("Cleaned up %d expired sessions\n", len(expiredIDs))
    }
}
