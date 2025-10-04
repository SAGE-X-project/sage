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


package session

import (
	"crypto/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/chacha20poly1305"
)

func TestManager_CreateGetRemove(t *testing.T) {
	mgr := NewManager()
	defer mgr.Close()

	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)

	t.Run("Create and retrieve session", func(t *testing.T) {
		sess, err := mgr.CreateSession("id1", secret)
		require.NoError(t, err)
		require.NotNil(t, sess)

		got, exists := mgr.GetSession("id1")
		require.True(t, exists)
		require.Equal(t, sess.GetID(), got.GetID())
	})

	t.Run("Remove session", func(t *testing.T) {
		mgr.RemoveSession("id1")
		_, exists := mgr.GetSession("id1")
		require.False(t, exists)
	})
}

// Verifies expiration and cleanup without relying on the background ticker.
// We wait past MaxAge and then call cleanupExpiredSessions() directly.
func TestManager_ExpirationCleanup(t *testing.T) {
	mgr := NewManager()
	defer mgr.Close()

	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)

	// Absolute expiration 50ms
	cfg := Config{MaxAge: 50 * time.Millisecond, IdleTimeout: 0, MaxMessages: 0}
	sess, err := mgr.CreateSessionWithConfig("exp1", secret, cfg)
	require.NoError(t, err)
	require.NotNil(t, sess)

	_, exists := mgr.GetSession("exp1")
	require.True(t, exists)

	// Wait until it should be expired
	time.Sleep(60 * time.Millisecond)

	// Trigger synchronous cleanup (avoid waiting for background ticker)
	mgr.cleanupExpiredSessions()

	_, exists = mgr.GetSession("exp1")
	require.False(t, exists)
}

// Lists and stats should reflect active sessions correctly.
func TestManager_ListAndStats(t *testing.T) {
	mgr := NewManager()
	defer mgr.Close()

	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)

	// Create multiple sessions
	_, _ = mgr.CreateSession("s1", secret)
	_, _ = mgr.CreateSession("s2", secret)

	list := mgr.ListSessions()
	require.Len(t, list, 2)

	stats := mgr.GetSessionStats()
	require.Equal(t, 2, stats.TotalSessions)
	require.Equal(t, 2, stats.ActiveSessions)
	require.Equal(t, 0, stats.ExpiredSessions)
}

func TestManager_ExistingSessionReuseAndCreateCollisions(t *testing.T) {
	mgr := NewManager()
	defer mgr.Close()

	secret := rb(chacha20poly1305.KeySize)

	t.Run("CreateSession returns error if session already exists", func(t *testing.T) {
		_, err := mgr.CreateSession("dup", secret)
		require.NoError(t, err)

		_, err = mgr.CreateSession("dup", secret)
		require.Error(t, err, "should not create a new session when the same id exists")
	})

	t.Run("EnsureSessionWithParams reuses existing session", func(t *testing.T) {
		// Pre-create the same SID using handshake params
		e1, e2 := rb(32), rb(32)
		p := Params{ContextID: "ctx-reuse", SharedSecret: secret, SelfEph: e1, PeerEph: e2, Label: "v1"}

		// First call should create
		s1, sid1, existed1, err := mgr.EnsureSessionWithParams(p, nil)
		require.NoError(t, err)
		require.False(t, existed1)
		require.NotEmpty(t, sid1)

		// Second call with same params should reuse (fast path)
		s2, sid2, existed2, err := mgr.EnsureSessionWithParams(p, nil)
		require.NoError(t, err)
		require.True(t, existed2)
		require.Equal(t, sid1, sid2)
		require.Equal(t, s1.GetID(), s2.GetID())

		// Manager should contain only one active session for that SID
		got, ok := mgr.GetSession(sid1)
		require.True(t, ok)
		require.Equal(t, s1.GetID(), got.GetID())
	})
}

func TestManager_EnsureSessionWithParams_DeterminismAndConfig(t *testing.T) {
	mgr := NewManager()
	defer mgr.Close()

	secret := rb(chacha20poly1305.KeySize)
	eA, eB := rb(32), rb(32)

	t.Run("Deterministic SID with swapped ephemeral keys", func(t *testing.T) {
		pA := Params{ContextID: "ctx-det", SelfEph: eA, SharedSecret: secret, PeerEph: eB, Label: "label-x"}
		pB := Params{ContextID: "ctx-det", SelfEph: eB, SharedSecret: secret, PeerEph: eA, Label: "label-x"}

		s1, sid1, existed1, err := mgr.EnsureSessionWithParams(pA, nil)
		require.NoError(t, err)
		require.False(t, existed1)

		s2, sid2, existed2, err := mgr.EnsureSessionWithParams(pB, nil)
		require.NoError(t, err)
		require.True(t, existed2)

		require.Equal(t, sid1, sid2, "same handshake context must yield the same SID")
		require.Equal(t, s1.GetID(), s2.GetID())
	})

	t.Run("Custom config is applied on first creation, then reused", func(t *testing.T) {
		p := Params{ContextID: "ctx-cfg", SelfEph: eA, SharedSecret: secret, PeerEph: eB, Label: "label-y"}

		custom := &Config{
			MaxAge:      250 * time.Millisecond,
			IdleTimeout: 120 * time.Millisecond,
			MaxMessages: 7,
		}

		s1, sid, existed, err := mgr.EnsureSessionWithParams(p, custom)
		require.NoError(t, err)
		require.False(t, existed)
		require.Equal(t, *custom, s1.GetConfig())

		// Next call should ignore provided (different) cfg and reuse the existing one
		other := &Config{MaxAge: time.Hour, IdleTimeout: time.Hour, MaxMessages: 999999}
		s2, sid2, existed2, err := mgr.EnsureSessionWithParams(p, other)
		require.NoError(t, err)
		require.True(t, existed2)
		require.Equal(t, sid, sid2)
		require.Equal(t, s1.GetConfig(), s2.GetConfig(), "existing session config must be kept")
	})
}

func TestManager_EnsureSessionWithParams_DoubleCheckedLocking_Concurrency(t *testing.T) {
	mgr := NewManager()
	defer mgr.Close()

	secret := rb(chacha20poly1305.KeySize)
	e1, e2 := rb(32), rb(32)
	p := Params{ContextID: "ctx-concurrent", SelfEph: e1, SharedSecret: secret, PeerEph: e2, Label: "v1"}

	var wg sync.WaitGroup
	const N = 16

	type res struct {
		sid     string
		existed bool
		err     error
	}
	results := make([]res, N)

	t.Run("Concurrent EnsureSessionWithParams yields single stored session", func(t *testing.T) {
		for i := 0; i < N; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				s, sid, existed, err := mgr.EnsureSessionWithParams(p, nil)
				if err == nil && s != nil {
					results[i] = res{sid: sid, existed: existed, err: nil}
				} else {
					results[i] = res{sid: "", existed: false, err: err}
				}
			}(i)
		}
		wg.Wait()

		// No errors and all SIDs equal
		require.NotZero(t, len(results))
		firstSID := ""
		for i := range results {
			require.NoError(t, results[i].err)
			if firstSID == "" {
				firstSID = results[i].sid
			}
			require.Equal(t, firstSID, results[i].sid)
		}

		// Exactly one creation (existed=false) and the rest reused (existed=true)
		var created, reused int
		for _, r := range results {
			if r.existed {
				reused++
			} else {
				created++
			}
		}
		require.Equal(t, 1, created, "only one goroutine should create the session")
		require.Equal(t, N-1, reused, "all other goroutines should reuse it")

		// Manager must have exactly one session for that SID
		require.Equal(t, 1, mgr.GetSessionCount())
	})
}

func TestManager_EnsureSessionWithParams_ErrorPaths(t *testing.T) {
	mgr := NewManager()
	defer mgr.Close()

	secret := rb(chacha20poly1305.KeySize)
	e1 := rb(32)

	t.Run("Empty shared secret returns error", func(t *testing.T) {
		_, _, _, err := mgr.EnsureSessionWithParams(Params{ContextID: "c", SelfEph: e1, PeerEph: e1, Label: "v"}, nil)
		require.Error(t, err)
	})

	t.Run("Invalid params (missing eph/context) returns error", func(t *testing.T) {
		_, _, _, err := mgr.EnsureSessionWithParams(Params{ContextID: "", SelfEph: e1, PeerEph: e1, SharedSecret: secret, Label: "v"}, nil)
		require.Error(t, err)
	})
}

func rb(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}
