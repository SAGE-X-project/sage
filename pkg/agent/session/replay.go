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

package session

import (
	"sync"
	"time"
)

// ReplayGuard remembers (scope, nonce) pairs for a verification window and
// reports reuse. CheckAndMark returns true when the pair has not been seen
// within the window and records it; it returns false when the pair was already
// presented. scope separates nonce spaces (an RFC 9421 keyid, an HPKE context
// id, ...) so that unrelated peers cannot collide or exhaust each other.
//
// This is the one replay-protection contract in SAGE: the RFC 9421 verifiers,
// the HPKE server and the session manager all take a ReplayGuard, and
// MemoryReplayGuard is the default implementation. Persistent or shared
// stores implement the same interface.
type ReplayGuard interface {
	CheckAndMark(scope, nonce string) bool
}

// MemoryReplayGuard is an in-memory ReplayGuard with a fixed TTL per entry.
// Expired entries are ignored on lookup and swept by a background loop.
type MemoryReplayGuard struct {
	ttl  time.Duration
	mu   sync.Mutex
	seen map[string]map[string]time.Time // scope -> nonce -> expiry
	stop chan struct{}
	once sync.Once
}

// NewMemoryReplayGuard returns a guard that forgets pairs after ttl. Call
// Close to stop its sweeper.
func NewMemoryReplayGuard(ttl time.Duration) *MemoryReplayGuard {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	g := &MemoryReplayGuard{ttl: ttl, seen: make(map[string]map[string]time.Time), stop: make(chan struct{})}
	interval := ttl / 2
	if interval < time.Second {
		interval = time.Second
	}
	go g.sweep(interval)
	return g
}

// CheckAndMark implements ReplayGuard.
func (g *MemoryReplayGuard) CheckAndMark(scope, nonce string) bool {
	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	m, ok := g.seen[scope]
	if !ok {
		m = make(map[string]time.Time)
		g.seen[scope] = m
	}
	if exp, ok := m[nonce]; ok && exp.After(now) {
		return false
	}
	m[nonce] = now.Add(g.ttl)
	return true
}

// Seen reports whether the pair is currently recorded, without recording it.
// Verifiers use it to fail fast before doing expensive work; the authoritative
// decision is still CheckAndMark after the signature has been verified.
func (g *MemoryReplayGuard) Seen(scope, nonce string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	exp, ok := g.seen[scope][nonce]
	return ok && exp.After(time.Now())
}

// Forget drops every nonce recorded under scope (for example when a key id is
// unbound and its session removed).
func (g *MemoryReplayGuard) Forget(scope string) {
	g.mu.Lock()
	delete(g.seen, scope)
	g.mu.Unlock()
}

// Close stops the sweeper. The guard remains usable.
func (g *MemoryReplayGuard) Close() {
	g.once.Do(func() { close(g.stop) })
}

func (g *MemoryReplayGuard) sweep(interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-g.stop:
			return
		case now := <-t.C:
			g.mu.Lock()
			for scope, m := range g.seen {
				for n, exp := range m {
					if !exp.After(now) {
						delete(m, n)
					}
				}
				if len(m) == 0 {
					delete(g.seen, scope)
				}
			}
			g.mu.Unlock()
		}
	}
}
