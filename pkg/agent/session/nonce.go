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

import "time"

// NonceCache is the previous name of the in-memory replay guard.
//
// Deprecated: use MemoryReplayGuard. NonceCache remains as a thin wrapper for
// one release: Seen is the inverse of CheckAndMark and DeleteKey is Forget.
type NonceCache struct {
	g *MemoryReplayGuard
}

// NewNonceCache returns a replay guard that forgets nonces after ttl.
//
// Deprecated: use NewMemoryReplayGuard.
func NewNonceCache(ttl time.Duration) *NonceCache {
	return &NonceCache{g: NewMemoryReplayGuard(ttl)}
}

// Seen records the pair and reports whether it had been presented before.
// Empty keyid or nonce values are never considered replays.
//
// Deprecated: use MemoryReplayGuard.CheckAndMark (note the inverted result).
func (n *NonceCache) Seen(keyid, nonce string) bool {
	if keyid == "" || nonce == "" {
		return false
	}
	return !n.g.CheckAndMark(keyid, nonce)
}

// DeleteKey forgets every nonce recorded for keyid.
//
// Deprecated: use MemoryReplayGuard.Forget.
func (n *NonceCache) DeleteKey(keyid string) { n.g.Forget(keyid) }

// Close stops the background sweeper.
func (n *NonceCache) Close() { n.g.Close() }
