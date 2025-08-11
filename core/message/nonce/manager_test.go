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
