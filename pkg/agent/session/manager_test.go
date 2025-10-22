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
	"crypto/rand"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/chacha20poly1305"
)

func TestManager_CreateGetRemove(t *testing.T) {
	// Specification Requirement: Session manager lifecycle operations
	helpers.LogTestSection(t, "9.2.1", "Session Manager Create, Get, Remove")

	mgr := NewManager()
	defer func() { _ = mgr.Close() }()
	helpers.LogDetail(t, "Session manager initialized")

	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)
	helpers.LogDetail(t, "Shared secret generated:")
	helpers.LogDetail(t, "  Secret (hex): %s", hex.EncodeToString(secret))

	t.Run("Create_and_retrieve_session", func(t *testing.T) {
		// Specification Requirement: Create session with unique ID
		sessionID := "id1"
		helpers.LogDetail(t, "Creating session with ID: %s", sessionID)

		sess, err := mgr.CreateSession(sessionID, secret)
		require.NoError(t, err)
		require.NotNil(t, sess)
		helpers.LogSuccess(t, "Session created successfully")
		helpers.LogDetail(t, "  Session ID: %s", sess.GetID())

		// Specification Requirement: Retrieve created session
		helpers.LogDetail(t, "Retrieving session by ID")
		got, exists := mgr.GetSession(sessionID)
		require.True(t, exists)
		require.Equal(t, sess.GetID(), got.GetID())
		helpers.LogSuccess(t, "Session retrieved successfully")
		helpers.LogDetail(t, "  Retrieved session ID matches: %v", sess.GetID() == got.GetID())
	})

	t.Run("Remove_session", func(t *testing.T) {
		// Specification Requirement: Remove session from manager
		sessionID := "id1"
		helpers.LogDetail(t, "Removing session: %s", sessionID)

		mgr.RemoveSession(sessionID)
		helpers.LogSuccess(t, "Session removed")

		// Specification Requirement: Verify session no longer exists
		helpers.LogDetail(t, "Verifying session removal")
		_, exists := mgr.GetSession(sessionID)
		require.False(t, exists)
		helpers.LogSuccess(t, "Session confirmed removed")
	})

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"Session manager initialized",
		"Shared secret generated (32 bytes)",
		"Session created with unique ID",
		"Session retrieved by ID",
		"Retrieved session matches created session",
		"Session removed from manager",
		"Removed session no longer exists",
	})

	// Save test data for CLI verification
	testData := map[string]interface{}{
		"test_case": "9.2.1_Session_Manager_Lifecycle",
		"operations": []string{"Create", "Get", "Remove"},
		"session_id": "id1",
		"secret_size": len(secret),
		"lifecycle_verified": true,
	}
	helpers.SaveTestData(t, "session/manager_lifecycle.json", testData)
}

// Verifies expiration and cleanup without relying on the background ticker.
// We wait past MaxAge and then call cleanupExpiredSessions() directly.
func TestManager_ExpirationCleanup(t *testing.T) {
	// Specification Requirement: Session expiration and automatic cleanup
	helpers.LogTestSection(t, "9.2.2", "Session Manager Expiration Cleanup")

	mgr := NewManager()
	defer func() { _ = mgr.Close() }()
	helpers.LogDetail(t, "Session manager initialized")

	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)
	helpers.LogDetail(t, "Shared secret generated (%d bytes)", len(secret))

	// Specification Requirement: Create session with short expiration time
	maxAge := 50 * time.Millisecond
	cfg := Config{MaxAge: maxAge, IdleTimeout: 0, MaxMessages: 0}
	helpers.LogDetail(t, "Creating session with expiration config:")
	helpers.LogDetail(t, "  Max age: %v", maxAge)
	helpers.LogDetail(t, "  Session ID: exp1")

	sess, err := mgr.CreateSessionWithConfig("exp1", secret, cfg)
	require.NoError(t, err)
	require.NotNil(t, sess)
	helpers.LogSuccess(t, "Session created with expiration config")

	// Specification Requirement: Verify session exists before expiration
	_, exists := mgr.GetSession("exp1")
	require.True(t, exists)
	helpers.LogSuccess(t, "Session exists before expiration")

	// Specification Requirement: Wait for expiration period
	waitTime := 60 * time.Millisecond
	helpers.LogDetail(t, "Waiting %v for session to expire", waitTime)
	time.Sleep(waitTime)
	helpers.LogSuccess(t, "Expiration period elapsed")

	// Specification Requirement: Trigger cleanup and verify session removed
	helpers.LogDetail(t, "Triggering synchronous cleanup")
	mgr.cleanupExpiredSessions()
	helpers.LogSuccess(t, "Cleanup executed")

	// Specification Requirement: Verify expired session no longer exists
	_, exists = mgr.GetSession("exp1")
	require.False(t, exists)
	helpers.LogSuccess(t, "Expired session successfully removed")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"Session manager initialized",
		"Session created with MaxAge expiration",
		"Session exists before expiration time",
		"Expiration period elapsed",
		"Cleanup triggered synchronously",
		"Expired session removed from manager",
	})

	// Save test data for CLI verification
	testData := map[string]interface{}{
		"test_case": "9.2.2_Session_Manager_Expiration",
		"session_id": "exp1",
		"max_age_ms": maxAge.Milliseconds(),
		"wait_time_ms": waitTime.Milliseconds(),
		"expiration_verified": true,
		"cleanup_successful": true,
	}
	helpers.SaveTestData(t, "session/manager_expiration.json", testData)
}

// Lists and stats should reflect active sessions correctly.
func TestManager_ListAndStats(t *testing.T) {
	// Specification Requirement: Session manager listing and statistics tracking
	helpers.LogTestSection(t, "9.2.3", "Session Manager List and Stats")

	mgr := NewManager()
	defer func() { _ = mgr.Close() }()
	helpers.LogDetail(t, "Session manager initialized")

	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	require.NoError(t, err)
	helpers.LogDetail(t, "Shared secret generated")

	// Specification Requirement: Create multiple sessions
	helpers.LogDetail(t, "Creating two sessions")
	sess1, err1 := mgr.CreateSession("s1", secret)
	sess2, err2 := mgr.CreateSession("s2", secret)
	require.NoError(t, err1)
	require.NoError(t, err2)
	helpers.LogSuccess(t, "Two sessions created")
	helpers.LogDetail(t, "  Session 1 ID: %s", sess1.GetID())
	helpers.LogDetail(t, "  Session 2 ID: %s", sess2.GetID())

	// Specification Requirement: List all sessions
	helpers.LogDetail(t, "Listing all sessions")
	list := mgr.ListSessions()
	require.Len(t, list, 2)
	helpers.LogSuccess(t, "Session list retrieved")
	helpers.LogDetail(t, "  Total sessions in list: %d", len(list))

	// Specification Requirement: Get session statistics
	helpers.LogDetail(t, "Retrieving session statistics")
	stats := mgr.GetSessionStats()
	require.Equal(t, 2, stats.TotalSessions)
	require.Equal(t, 2, stats.ActiveSessions)
	require.Equal(t, 0, stats.ExpiredSessions)
	helpers.LogSuccess(t, "Session statistics verified")
	helpers.LogDetail(t, "  Total sessions: %d", stats.TotalSessions)
	helpers.LogDetail(t, "  Active sessions: %d", stats.ActiveSessions)
	helpers.LogDetail(t, "  Expired sessions: %d", stats.ExpiredSessions)

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"Session manager initialized",
		"Two sessions created successfully",
		"ListSessions returns all sessions",
		"Session count matches expected (2)",
		"Statistics show correct total count",
		"Statistics show correct active count",
		"Statistics show zero expired sessions",
	})

	// Save test data for CLI verification
	testData := map[string]interface{}{
		"test_case": "9.2.3_Session_Manager_List_Stats",
		"sessions_created": 2,
		"sessions_listed": len(list),
		"statistics": map[string]interface{}{
			"total_sessions": stats.TotalSessions,
			"active_sessions": stats.ActiveSessions,
			"expired_sessions": stats.ExpiredSessions,
		},
		"verification_passed": true,
	}
	helpers.SaveTestData(t, "session/manager_list_stats.json", testData)
}

func TestManager_ExistingSessionReuseAndCreateCollisions(t *testing.T) {
	mgr := NewManager()
	defer func() { _ = mgr.Close() }()

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
	defer func() { _ = mgr.Close() }()

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
	defer func() { _ = mgr.Close() }()

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
	defer func() { _ = mgr.Close() }()

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
