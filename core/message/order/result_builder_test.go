package order

import (
	"testing"
	"time"

	"github.com/sage-x-project/sage/core/message"
	"github.com/stretchr/testify/require"
)

func TestResultBuilder(t *testing.T) {
	t.Run("DefaultValues", func(t *testing.T) {
		builder := NewResultBuilder()
		res := builder.Build()

		require.False(t, res.IsProcessed, "default IsProcessed should be false")
		require.False(t, res.IsDuplicate, "default IsDuplicate should be false")
		require.False(t, res.IsWaiting, "default IsWaiting should be false")
		require.Empty(t, res.ReadyMessages, "default ReadyMessages should be empty slice")
	})

	t.Run("WithProcessed", func(t *testing.T) {
		res := NewResultBuilder().WithProcessed(true).Build()
		require.True(t, res.IsProcessed, "WithProcessed(true) should set IsProcessed=true")
	})

	t.Run("WithDuplicate", func(t *testing.T) {
		res := NewResultBuilder().WithDuplicate(true).Build()
		require.True(t, res.IsDuplicate, "WithDuplicate(true) should set IsDuplicate=true")
	})

	t.Run("WithWaiting", func(t *testing.T) {
		res := NewResultBuilder().WithWaiting(true).Build()
		require.True(t, res.IsWaiting, "WithWaiting(true) should set IsWaiting=true")
	})

	t.Run("WithReadyMessages", func(t *testing.T) {
		now := time.Now()
		head1 := &mockHeader{seq: 1, nonce: "a", timestamp: now}
		head2 := &mockHeader{seq: 2, nonce: "b", timestamp: now.Add(time.Second)}
		expected := []message.ControlHeader{head1, head2}

		res := NewResultBuilder().WithReadyMessages(expected).Build()
		require.Equal(t, expected, res.ReadyMessages, "WithReadyMessages should set the slice correctly")
	})

	t.Run("ChainedSettings", func(t *testing.T) {
		now := time.Now()
		head := &mockHeader{seq: 3, nonce: "c", timestamp: now}
		res := NewResultBuilder().
			WithProcessed(true).
			WithDuplicate(true).
			WithWaiting(true).
			WithReadyMessages([]message.ControlHeader{head}).
			Build()

		require.True(t, res.IsProcessed)
		require.True(t, res.IsDuplicate)
		require.True(t, res.IsWaiting)
		require.Len(t, res.ReadyMessages, 1)
		require.Equal(t, head, res.ReadyMessages[0])
	})
}
