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

package dedupe

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// mockHeader implements message.ControlHeader for testing Detector
// Only Sequence, Nonce, and Timestamp are used in hash

type mockHeader struct {
	seq       uint64
	nonce     string
	timestamp time.Time
}

func (f *mockHeader) GetSequence() uint64     { return f.seq }
func (f *mockHeader) GetNonce() string        { return f.nonce }
func (f *mockHeader) GetTimestamp() time.Time { return f.timestamp }

func TestDetector(t *testing.T) {
	now := time.Now()

	t.Run("NewDetector_NoDuplicate", func(t *testing.T) {
		d := NewDetector(time.Second, time.Second)
		h := &mockHeader{seq: 1, nonce: "n1", timestamp: now}
		require.False(t, d.IsDuplicate(h), "new Detector should report no duplicates")
		require.Equal(t, 0, d.GetSeenPacketCount(), "no packets should be tracked initially")
	})

	t.Run("MarkAndDetectDuplicate", func(t *testing.T) {
		d := NewDetector(time.Second, time.Second)
		h := &mockHeader{seq: 1, nonce: "n1", timestamp: now}
		d.MarkPacketSeen(h)
		require.True(t, d.IsDuplicate(h), "packet just marked should be detected as duplicate")
		require.Equal(t, 1, d.GetSeenPacketCount(), "seen packet count should be 1")
	})

	t.Run("DifferentMessagesCount", func(t *testing.T) {
		d := NewDetector(time.Second, time.Second)
		head1 := &mockHeader{seq: 1, nonce: "a", timestamp: now}
		head2 := &mockHeader{seq: 2, nonce: "b", timestamp: now}
		d.MarkPacketSeen(head1)
		d.MarkPacketSeen(head2)
		require.Equal(t, 2, d.GetSeenPacketCount(), "should track two distinct packets")
	})

	t.Run("IsDuplicateRemovesExpired", func(t *testing.T) {
		ttl := 20 * time.Millisecond
		d := NewDetector(ttl, time.Hour)
		h := &mockHeader{seq: 1, nonce: "x", timestamp: time.Now()}
		d.MarkPacketSeen(h)
		time.Sleep(ttl + 10*time.Millisecond)
		// expired entry should be removed by IsDuplicate
		require.False(t, d.IsDuplicate(h), "expired packet should not be detected as duplicate")
		require.Equal(t, 0, d.GetSeenPacketCount(), "expired packet should be removed from tracking")
	})

	t.Run("CleanupLoopPurgesExpired", func(t *testing.T) {
		ttl := 20 * time.Millisecond
		cleanup := 10 * time.Millisecond
		d := NewDetector(ttl, cleanup)
		h := &mockHeader{seq: 1, nonce: "y", timestamp: time.Now()}
		d.MarkPacketSeen(h)
		require.Equal(t, 1, d.GetSeenPacketCount(), "should start with one tracked packet")

		// wait for TTL + cleanupInterval for cleanupLoop to run
		time.Sleep(ttl + cleanup + 10*time.Millisecond)
		require.Equal(t, 0, d.GetSeenPacketCount(), "cleanupLoop should purge expired packets")
	})
}
