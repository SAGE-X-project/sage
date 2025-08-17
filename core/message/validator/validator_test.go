package validator

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// mockHeader implements message.ControlHeader for testing
// Uses Sequence, Nonce, Timestamp

type mockHeader struct {
	seq       uint64
	nonce     string
	timestamp time.Time
}

func (m *mockHeader) GetSequence() uint64    { return m.seq }
func (m *mockHeader) GetNonce() string        { return m.nonce }
func (m *mockHeader) GetTimestamp() time.Time { return m.timestamp }

func TestValidateMessage(t *testing.T) {
	t.Run("ValidAndStats", func(t *testing.T) {
		cfg := &ValidatorConfig{
			TimestampTolerance:  time.Second,
			NonceTTL:            time.Minute,
			DuplicateTTL:        time.Minute,
			MaxOutOfOrderWindow: time.Second,
			CleanupInterval:     time.Second,
		}
		mv := NewMessageValidator(cfg)

		now := time.Now()
		head := &mockHeader{seq: 1, nonce: uuid.NewString(), timestamp: now}
		res := mv.ValidateMessage(head, "sess1", uuid.NewString())
		require.True(t, res.IsValid)
		require.False(t, res.IsReplay)
		require.False(t, res.IsDuplicate)
		require.False(t, res.IsOutOfOrder)
		
		stats := mv.GetStats()
		require.EqualValues(t, 1, stats["tracked_nonces"], "one nonce marked")
		require.EqualValues(t, 1, stats["tracked_packets"], "one packet tracked")
	})

	t.Run("TimestampOutsideTolerance", func(t *testing.T) {
		cfg := &ValidatorConfig{
			TimestampTolerance:  10 * time.Millisecond,
			NonceTTL:            time.Minute,
			DuplicateTTL:        time.Minute,
			MaxOutOfOrderWindow: time.Second,
			CleanupInterval:     time.Hour,
		}
		mv := NewMessageValidator(cfg)
		
		old := time.Now().Add(-time.Second)
		head := &mockHeader{seq: 1, nonce: uuid.NewString(), timestamp: old}
		res := mv.ValidateMessage(head, "sess", uuid.NewString())
		require.False(t, res.IsValid)
		require.Contains(t, res.Error.Error(), "timestamp outside tolerance window")

		stats := mv.GetStats()
		require.EqualValues(t, 0, stats["tracked_nonces"], "no nonce used on invalid")
		require.EqualValues(t, 0, stats["tracked_packets"], "no packet tracked on invalid")
	})

	t.Run("ReplayDetection", func(t *testing.T) {
		mv := NewMessageValidator(nil)

		nonce := uuid.NewString()
		head := &mockHeader{seq: 1, nonce: nonce, timestamp: time.Now()}

		// first call: valid
		res1 := mv.ValidateMessage(head, "sess", uuid.NewString())
		require.True(t, res1.IsValid)

		// second call: same nonce => replay
		res2 := mv.ValidateMessage(head, "sess", uuid.NewString())
		require.False(t, res2.IsValid)
		require.True(t, res2.IsReplay)
		require.Equal(t, "nonce has been used before (replay attack detected)", res2.Error.Error())

		stats := mv.GetStats()
		require.EqualValues(t, 1, stats["tracked_nonces"], "only first nonce is tracked")
		require.EqualValues(t, 1, stats["tracked_packets"], "only first packet is tracked")
	})

	t.Run("OutOfOrderError", func(t *testing.T) {
		cfg := &ValidatorConfig{
			TimestampTolerance:  time.Second,
			NonceTTL:            time.Minute,
			DuplicateTTL:        time.Minute,
			MaxOutOfOrderWindow: 50 * time.Millisecond,
			CleanupInterval:     time.Hour,
		}
		mv := NewMessageValidator(cfg)

		t1 := time.Now()
		head1 := &mockHeader{seq: 1, nonce: uuid.NewString(), timestamp: t1}
		res1 := mv.ValidateMessage(head1, "sess", uuid.NewString())
		require.True(t, res1.IsValid)

		// second msg with earlier timestamp triggers order error
		head2 := &mockHeader{seq: 2, nonce: uuid.NewString(), timestamp: t1.Add(-100 * time.Millisecond)}
		res2 := mv.ValidateMessage(head2, "sess", uuid.NewString())
		require.False(t, res2.IsValid)
		require.Contains(t, res2.Error.Error(), "order validation failed")

		stats := mv.GetStats()
		require.EqualValues(t, 1, stats["tracked_nonces"], "no new nonce marked on error")
		require.EqualValues(t, 1, stats["tracked_packets"], "no new packet tracked on error")
	})
}
