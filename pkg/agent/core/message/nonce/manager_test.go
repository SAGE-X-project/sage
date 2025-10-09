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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNonceManager(t *testing.T) {
	const ttl = 50 * time.Millisecond
	const cleanup = 10 * time.Millisecond

	t.Run("GenerateNonce", func(t *testing.T) {
		n, err := GenerateNonce()
		require.NoError(t, err)
		require.NotEmpty(t, n)
	})

	t.Run("MarkNonceUsed", func(t *testing.T) {
		m := NewManager(time.Second, time.Second)
		n := "test-nonce"
		m.MarkNonceUsed(n)
		require.True(t, m.IsNonceUsed(n), "manually marked nonce should be used")
		require.Equal(t, 1, m.GetUsedNonceCount())
	})

	t.Run("IsNonceUsedExpiresOnCheck", func(t *testing.T) {
		m := NewManager(ttl, time.Hour)
		n := "expiring-nonce"
		m.MarkNonceUsed(n)
		time.Sleep(ttl + 20*time.Millisecond)
		// first call removes expired entry
		require.False(t, m.IsNonceUsed(n), "expired nonce should not be considered used")
		require.Equal(t, 0, m.GetUsedNonceCount(), "expired nonce should be removed")
	})

	t.Run("CleanupLoopPurgesExpired", func(t *testing.T) {
		m := NewManager(ttl, cleanup)
		n := "cleanup-nonce"
		m.MarkNonceUsed(n)
		require.Equal(t, 1, m.GetUsedNonceCount(), "nonce should be initially tracked")
		// wait for TTL + cleanup interval for cleanupLoop to run
		time.Sleep(ttl + cleanup + 20*time.Millisecond)
		require.Equal(t, 0, m.GetUsedNonceCount(), "cleanupLoop should purge expired nonces")
	})
}
