package session

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryReplayGuard_CheckAndMark(t *testing.T) {
	g := NewMemoryReplayGuard(time.Hour)
	defer g.Close()

	assert.False(t, g.Seen("k1", "n1"))
	assert.True(t, g.CheckAndMark("k1", "n1"), "first presentation is accepted")
	assert.True(t, g.Seen("k1", "n1"))
	assert.False(t, g.CheckAndMark("k1", "n1"), "second presentation is a replay")
	assert.True(t, g.CheckAndMark("k2", "n1"), "scopes are independent")

	g.Forget("k1")
	assert.True(t, g.CheckAndMark("k1", "n1"), "forgotten scope accepts the nonce again")
}

func TestMemoryReplayGuard_Expiry(t *testing.T) {
	g := NewMemoryReplayGuard(20 * time.Millisecond)
	defer g.Close()
	require.True(t, g.CheckAndMark("k", "n"))
	require.False(t, g.CheckAndMark("k", "n"))
	time.Sleep(40 * time.Millisecond)
	assert.False(t, g.Seen("k", "n"), "expired entries are not seen")
	assert.True(t, g.CheckAndMark("k", "n"), "expired entries may be presented again")
}

func TestMemoryReplayGuard_ConcurrentSingleWinner(t *testing.T) {
	g := NewMemoryReplayGuard(time.Hour)
	defer g.Close()
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if g.CheckAndMark("k", "n") {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, 1, wins)
	g.Close() // idempotent
}

func TestNonceCache_DeprecatedWrapper(t *testing.T) {
	c := NewNonceCache(time.Hour)
	defer c.Close()
	assert.False(t, c.Seen("k", "n"))
	assert.True(t, c.Seen("k", "n"))
	assert.False(t, c.Seen("", "n"), "empty key id is never a replay")
	c.DeleteKey("k")
	assert.False(t, c.Seen("k", "n"))
}
