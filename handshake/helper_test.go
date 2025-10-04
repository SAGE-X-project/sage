package handshake

import "time"

// HasPeer reports whether the peer cache contains the provided context identifier.
func HasPeer(s *Server, ctxID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.peers[ctxID]
	return ok
}

// SetPeerExpiry overrides the expiry timestamp for the given peer in tests.
func SetPeerExpiry(s *Server, ctxID string, expires time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	peer, ok := s.peers[ctxID]
	if !ok {
		return
	}
	peer.expires = expires
	s.peers[ctxID] = peer
}

// StopCleanupLoop stops the running cleanup loop and waits for it to exit.
func StopCleanupLoop(s *Server) {
	s.mu.Lock()
	stop := s.stopCleanup
	done := s.cleanupDone
	if stop == nil {
		s.mu.Unlock()
		if done != nil {
			<-done
		}
		return
	}
	s.stopCleanup = nil
	s.mu.Unlock()
	close(stop)
	if done != nil {
		<-done
	}
	s.mu.Lock()
	s.cleanupDone = nil
	s.mu.Unlock()
}

// RestartCleanupLoop restarts the cleanup loop with a new ticker interval.
func RestartCleanupLoop(s *Server, interval time.Duration) {
	StopCleanupLoop(s)
	s.mu.Lock()
	s.cleanupTicker = time.NewTicker(interval)
	s.stopCleanup = make(chan struct{})
	s.cleanupDone = make(chan struct{})
	go s.cleanupLoop()
	s.mu.Unlock()
}

// OverridePendingTTL sets the default TTL used when caching new peers.
func OverridePendingTTL(s *Server, ttl time.Duration) {
	s.mu.Lock()
	s.pendingTTL = ttl
	s.mu.Unlock()
}
