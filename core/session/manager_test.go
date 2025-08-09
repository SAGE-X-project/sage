package session

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
