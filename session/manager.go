package session

import (
	"fmt"
	"sync"
	"time"
)

// Manager handles session lifecycle, storage, and cleanup
type Manager struct {
    sessions       map[string]Session
    byKeyID        map[string]string 
    keyIDsBySID    map[string]map[string]struct{}
    mu             sync.RWMutex
    cleanupTicker  *time.Ticker
    stopCleanup    chan struct{}
    defaultConfig  Config
    nonceCache     *NonceCache // replay guard
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
        nonceCache:  NewNonceCache(10 * time.Minute), // replay TTL
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

// EnsureSessionWithParams computes a deterministic sessionID and creates the session.
func (m *Manager) EnsureSessionWithParams(p Params, cfg *Config) (Session, string, bool, error) {
	seed, err := DeriveSessionSeed(p.SharedSecret, p)
	if err != nil {
		return nil, "", false, fmt.Errorf("derive seed: %w", err)
	}
	sid, err := ComputeSessionIDFromSeed(seed, p.Label)
	if err != nil {
		return nil, "", false, fmt.Errorf("compute id: %w", err)
	}

	// Fast path
	m.mu.RLock()
	if s, ok := m.sessions[sid]; ok {
		m.mu.RUnlock()
		return s, sid, true, nil
	}
	m.mu.RUnlock()

	newCfg := m.defaultConfig
	if cfg != nil {
		newCfg = withDefaults(*cfg)
	}
	s, err := NewSecureSession(sid, seed, newCfg)
	if err != nil {
		return nil, "", false, fmt.Errorf("new secure session: %w", err)
	}

	// Double-checked put
	m.mu.Lock()
	if exist, ok := m.sessions[sid]; ok {
		m.mu.Unlock()
		_ = s.Close()
		return exist, sid, true, nil
	}
	m.sessions[sid] = s
	m.mu.Unlock()

	return s, sid, false, nil
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

// BindKeyID associates an opaque keyid with an existing session ID and tracks reverse mapping.
func (m *Manager) BindKeyID(keyid, sid string) {
	m.mu.Lock()
	if m.byKeyID == nil {
		m.byKeyID = make(map[string]string)
	}
	if m.keyIDsBySID == nil {
		m.keyIDsBySID = make(map[string]map[string]struct{})
	}
	m.byKeyID[keyid] = sid
	set, ok := m.keyIDsBySID[sid]
	if !ok {
		set = make(map[string]struct{})
		m.keyIDsBySID[sid] = set
	}
	set[keyid] = struct{}{}
	m.mu.Unlock()
}

// UnbindKeyID removes a keyid mapping (call on session close or key rotation).
func (m *Manager) UnbindKeyID(keyid string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	sid, ok := m.byKeyID[keyid]
	if !ok {
		return false
	}
	delete(m.byKeyID, keyid)
	if set, ok := m.keyIDsBySID[sid]; ok {
		delete(set, keyid)
		if len(set) == 0 {
			delete(m.keyIDsBySID, sid)
		}
	}
	// drop replay entries for this key
	if m.nonceCache != nil {
		m.nonceCache.DeleteKey(keyid)
	}
	return true
}

// GetByKeyID returns the Session associated with the given keyid (if alive).
func (m *Manager) GetByKeyID(keyid string) (Session, bool) {
	m.mu.RLock()
	sid, ok := m.byKeyID[keyid]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return m.GetSession(sid)
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

// RemoveSession removes a session and unbinds all associated keyids.
func (m *Manager) RemoveSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, exists := m.sessions[sessionID]; exists {
		sess.Close()
		delete(m.sessions, sessionID)
	}
	// Unbind all keyids mapped to this sessionID
	if set, ok := m.keyIDsBySID[sessionID]; ok {
		for kid := range set {
			delete(m.byKeyID, kid)
			if m.nonceCache != nil {
				m.nonceCache.DeleteKey(kid)
			}
		}
		delete(m.keyIDsBySID, sessionID)
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

// ReplayGuardSeenOnce should be called per incoming request after parsing RFC-9421 `nonce`.
// Returns true if the (keyid, nonce) was already seen (reject request).
func (m *Manager) ReplayGuardSeenOnce(keyid, nonce string) bool {
	if m.nonceCache == nil {
		return false
	}
	return m.nonceCache.Seen(keyid, nonce)
}

// GetSessionCount returns the number of active sessions
func (m *Manager) GetSessionCount() int {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return len(m.sessions)
}

// GetSessionStats returns statistics about sessions
func (m *Manager) GetSessionStats() Status {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    stats := Status{
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

// Close stops the manager and cleans up all sessions and caches.
func (m *Manager) Close() error {
	close(m.stopCleanup)
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
	}
	if m.nonceCache != nil {
		m.nonceCache.Close()
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, sess := range m.sessions {
		sess.Close()
	}
	m.sessions = make(map[string]Session)
	m.byKeyID = nil
	m.keyIDsBySID = nil
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

// cleanupExpiredSessions removes expired sessions and unbinds their keyids.
func (m *Manager) cleanupExpiredSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	var expiredIDs []string
	for id, sess := range m.sessions {
		if sess.IsExpired() {
			expiredIDs = append(expiredIDs, id)
		}
	}
	for _, id := range expiredIDs {
		if sess, exists := m.sessions[id]; exists {
			sess.Close()
			delete(m.sessions, id)
		}
		// Unbind all keyids for this session
		if set, ok := m.keyIDsBySID[id]; ok {
			for kid := range set {
				delete(m.byKeyID, kid)
				if m.nonceCache != nil {
					m.nonceCache.DeleteKey(kid)
				}
			}
			delete(m.keyIDsBySID, id)
		}
	}
	if len(expiredIDs) > 0 {
		fmt.Printf("Cleaned up %d expired sessions\n", len(expiredIDs))
	}
}

func withDefaults(c Config) Config {
    if c.MaxAge == 0 {
        c.MaxAge = time.Hour // 기본 1시간
    }
    if c.IdleTimeout == 0 {
        c.IdleTimeout = 10 * time.Minute // 기본 10분
    }
    if c.MaxMessages == 0 {
        c.MaxMessages = 1000 // 기본 최대 메시지 수
    }
    return c
}

