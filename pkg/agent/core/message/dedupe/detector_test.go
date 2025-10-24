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

	"github.com/sage-x-project/sage/tests/helpers"
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
		// Specification Requirement: New detector should start with no tracked packets
		helpers.LogTestSection(t, "8.2.2", "Deduplication Detector Initialization")

		ttl := time.Second
		cleanupInterval := time.Second
		d := NewDetector(ttl, cleanupInterval)

		helpers.LogDetail(t, "Detector initialized:")
		helpers.LogDetail(t, "  TTL: %v", ttl)
		helpers.LogDetail(t, "  Cleanup interval: %v", cleanupInterval)
		helpers.LogSuccess(t, "Detector created successfully")

		// Specification Requirement: Check for duplicate with unseen message
		seq := uint64(1)
		nonce := "n1"
		timestamp := now
		h := &mockHeader{seq: seq, nonce: nonce, timestamp: timestamp}

		helpers.LogDetail(t, "Test message header:")
		helpers.LogDetail(t, "  Sequence: %d", seq)
		helpers.LogDetail(t, "  Nonce: %s", nonce)
		helpers.LogDetail(t, "  Timestamp: %s", timestamp.Format(time.RFC3339Nano))

		isDup := d.IsDuplicate(h)
		require.False(t, isDup, "new Detector should report no duplicates")
		helpers.LogSuccess(t, "No duplicate detected for unseen message")
		helpers.LogDetail(t, "Is duplicate: %v", isDup)

		count := d.GetSeenPacketCount()
		require.Equal(t, 0, count, "no packets should be tracked initially")
		helpers.LogDetail(t, "Seen packet count: %d", count)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Detector initialized successfully",
			"Unseen message not detected as duplicate",
			"No packets tracked initially",
			"Baseline state correct",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case": "8.2.2_Detector_Initialization",
			"detector_config": map[string]interface{}{
				"ttl_ms":              ttl.Milliseconds(),
				"cleanup_interval_ms": cleanupInterval.Milliseconds(),
			},
			"message": map[string]interface{}{
				"sequence":  seq,
				"nonce":     nonce,
				"timestamp": timestamp.Format(time.RFC3339Nano),
			},
			"detection": map[string]interface{}{
				"is_duplicate": isDup,
				"packet_count": count,
			},
		}
		helpers.SaveTestData(t, "message/dedupe/detector_initialization.json", testData)
	})

	t.Run("MarkAndDetectDuplicate", func(t *testing.T) {
		// Specification Requirement: Message deduplication for replay attack prevention
		helpers.LogTestSection(t, "8.2.1", "Message Deduplication Detection")

		ttl := time.Second
		cleanupInterval := time.Second
		d := NewDetector(ttl, cleanupInterval)

		helpers.LogDetail(t, "Detector TTL: %v", ttl)
		helpers.LogDetail(t, "Cleanup interval: %v", cleanupInterval)

		// Specification Requirement: Create message header with unique identifiers
		seq := uint64(1)
		nonce := "n1"
		timestamp := now
		h := &mockHeader{seq: seq, nonce: nonce, timestamp: timestamp}

		helpers.LogDetail(t, "Message header:")
		helpers.LogDetail(t, "  Sequence: %d", seq)
		helpers.LogDetail(t, "  Nonce: %s", nonce)
		helpers.LogDetail(t, "  Timestamp: %s", timestamp.Format(time.RFC3339Nano))

		// Specification Requirement: Mark packet as seen (first occurrence)
		d.MarkPacketSeen(h)
		helpers.LogSuccess(t, "Packet marked as seen")

		seenCount := d.GetSeenPacketCount()
		require.Equal(t, 1, seenCount, "seen packet count should be 1")
		helpers.LogDetail(t, "Seen packet count: %d", seenCount)

		// Specification Requirement: Detect duplicate (replay attack)
		isDup := d.IsDuplicate(h)
		require.True(t, isDup, "packet just marked should be detected as duplicate")
		helpers.LogSuccess(t, "Duplicate detected: Replay attack prevented")
		helpers.LogDetail(t, "Is duplicate: %v", isDup)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Detector initialized with TTL and cleanup",
			"Packet marked as seen successfully",
			"Seen packet count = 1",
			"Duplicate detection successful",
			"Replay attack prevented",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case": "8.2.1_Message_Deduplication",
			"detector_config": map[string]interface{}{
				"ttl_ms":              ttl.Milliseconds(),
				"cleanup_interval_ms": cleanupInterval.Milliseconds(),
			},
			"message": map[string]interface{}{
				"sequence":  seq,
				"nonce":     nonce,
				"timestamp": timestamp.Format(time.RFC3339Nano),
			},
			"detection": map[string]interface{}{
				"marked_seen":      true,
				"seen_count":       seenCount,
				"is_duplicate":     isDup,
				"replay_prevented": true,
			},
		}
		helpers.SaveTestData(t, "message/dedupe/deduplication_detection.json", testData)
	})

	t.Run("DifferentMessagesCount", func(t *testing.T) {
		// Specification Requirement: Track multiple distinct messages separately
		helpers.LogTestSection(t, "8.2.3", "Deduplication Multi-Message Tracking")

		ttl := time.Second
		cleanupInterval := time.Second
		d := NewDetector(ttl, cleanupInterval)

		helpers.LogDetail(t, "Detector configuration:")
		helpers.LogDetail(t, "  TTL: %v", ttl)
		helpers.LogDetail(t, "  Cleanup interval: %v", cleanupInterval)
		helpers.LogSuccess(t, "Detector initialized")

		// Specification Requirement: Create two distinct messages
		seq1 := uint64(1)
		nonce1 := "a"
		timestamp1 := now
		head1 := &mockHeader{seq: seq1, nonce: nonce1, timestamp: timestamp1}

		seq2 := uint64(2)
		nonce2 := "b"
		timestamp2 := now
		head2 := &mockHeader{seq: seq2, nonce: nonce2, timestamp: timestamp2}

		helpers.LogDetail(t, "First message:")
		helpers.LogDetail(t, "  Sequence: %d", seq1)
		helpers.LogDetail(t, "  Nonce: %s", nonce1)
		helpers.LogDetail(t, "  Timestamp: %s", timestamp1.Format(time.RFC3339Nano))

		helpers.LogDetail(t, "Second message:")
		helpers.LogDetail(t, "  Sequence: %d", seq2)
		helpers.LogDetail(t, "  Nonce: %s", nonce2)
		helpers.LogDetail(t, "  Timestamp: %s", timestamp2.Format(time.RFC3339Nano))

		// Specification Requirement: Mark both packets as seen
		d.MarkPacketSeen(head1)
		helpers.LogDetail(t, "First packet marked as seen")

		d.MarkPacketSeen(head2)
		helpers.LogDetail(t, "Second packet marked as seen")

		count := d.GetSeenPacketCount()
		require.Equal(t, 2, count, "should track two distinct packets")
		helpers.LogSuccess(t, "Both distinct packets tracked")
		helpers.LogDetail(t, "Total seen packet count: %d", count)

		// Specification Requirement: Verify both are detected as duplicates
		isDup1 := d.IsDuplicate(head1)
		isDup2 := d.IsDuplicate(head2)
		require.True(t, isDup1, "first packet should be duplicate")
		require.True(t, isDup2, "second packet should be duplicate")
		helpers.LogSuccess(t, "Both packets correctly detected as duplicates")
		helpers.LogDetail(t, "First is duplicate: %v", isDup1)
		helpers.LogDetail(t, "Second is duplicate: %v", isDup2)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Detector initialized successfully",
			"Two distinct messages created",
			"First packet marked as seen",
			"Second packet marked as seen",
			"Total packet count = 2",
			"Both packets detected as duplicates",
			"Multi-message tracking working correctly",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case": "8.2.3_Multi_Message_Tracking",
			"detector_config": map[string]interface{}{
				"ttl_ms":              ttl.Milliseconds(),
				"cleanup_interval_ms": cleanupInterval.Milliseconds(),
			},
			"messages": []map[string]interface{}{
				{
					"sequence":     seq1,
					"nonce":        nonce1,
					"timestamp":    timestamp1.Format(time.RFC3339Nano),
					"is_duplicate": isDup1,
				},
				{
					"sequence":     seq2,
					"nonce":        nonce2,
					"timestamp":    timestamp2.Format(time.RFC3339Nano),
					"is_duplicate": isDup2,
				},
			},
			"tracking": map[string]interface{}{
				"total_count":       count,
				"distinct_messages": 2,
			},
		}
		helpers.SaveTestData(t, "message/dedupe/multi_message_tracking.json", testData)
	})

	t.Run("IsDuplicateRemovesExpired", func(t *testing.T) {
		// Specification Requirement: Expired packets removed on duplicate check
		helpers.LogTestSection(t, "8.2.4", "Deduplication Expiration on Check")

		ttl := 20 * time.Millisecond
		cleanupInterval := time.Hour
		d := NewDetector(ttl, cleanupInterval)

		helpers.LogDetail(t, "Detector configuration:")
		helpers.LogDetail(t, "  TTL: %v (short)", ttl)
		helpers.LogDetail(t, "  Cleanup interval: %v (disabled for this test)", cleanupInterval)
		helpers.LogSuccess(t, "Detector initialized")

		// Specification Requirement: Create message and mark as seen
		seq := uint64(1)
		nonce := "x"
		timestamp := time.Now()
		h := &mockHeader{seq: seq, nonce: nonce, timestamp: timestamp}

		helpers.LogDetail(t, "Test message:")
		helpers.LogDetail(t, "  Sequence: %d", seq)
		helpers.LogDetail(t, "  Nonce: %s", nonce)
		helpers.LogDetail(t, "  Timestamp: %s", timestamp.Format(time.RFC3339Nano))

		d.MarkPacketSeen(h)
		initialCount := d.GetSeenPacketCount()
		helpers.LogSuccess(t, "Packet marked as seen")
		helpers.LogDetail(t, "Initial packet count: %d", initialCount)

		// Specification Requirement: Wait for TTL expiration
		sleepDuration := ttl + 10*time.Millisecond
		helpers.LogDetail(t, "Waiting %v for packet to expire", sleepDuration)
		time.Sleep(sleepDuration)
		helpers.LogSuccess(t, "TTL expiration period elapsed")

		// Specification Requirement: Expired entry should be removed by IsDuplicate call
		helpers.LogDetail(t, "Checking if expired packet is duplicate")
		isDup := d.IsDuplicate(h)
		require.False(t, isDup, "expired packet should not be detected as duplicate")
		helpers.LogSuccess(t, "Expired packet not detected as duplicate")
		helpers.LogDetail(t, "Is duplicate: %v", isDup)

		finalCount := d.GetSeenPacketCount()
		require.Equal(t, 0, finalCount, "expired packet should be removed from tracking")
		helpers.LogSuccess(t, "Expired packet removed from tracking")
		helpers.LogDetail(t, "Final packet count: %d", finalCount)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Detector initialized with short TTL",
			"Packet marked as seen initially",
			"TTL expiration period elapsed",
			"IsDuplicate call triggered expiration check",
			"Expired packet not detected as duplicate",
			"Expired packet removed from tracking",
			"Memory cleanup on check working",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case": "8.2.4_Expiration_On_Check",
			"detector_config": map[string]interface{}{
				"ttl_ms":              ttl.Milliseconds(),
				"cleanup_interval_ms": cleanupInterval.Milliseconds(),
			},
			"message": map[string]interface{}{
				"sequence":  seq,
				"nonce":     nonce,
				"timestamp": timestamp.Format(time.RFC3339Nano),
			},
			"expiration": map[string]interface{}{
				"sleep_duration_ms": sleepDuration.Milliseconds(),
				"initial_count":     initialCount,
				"final_count":       finalCount,
				"is_duplicate":      isDup,
				"expired":           true,
			},
		}
		helpers.SaveTestData(t, "message/dedupe/expiration_on_check.json", testData)
	})

	t.Run("CleanupLoopPurgesExpired", func(t *testing.T) {
		// Specification Requirement: Automatic cleanup loop for expired packet purging
		helpers.LogTestSection(t, "8.2.5", "Deduplication Automatic Cleanup Loop")

		ttl := 20 * time.Millisecond
		cleanup := 10 * time.Millisecond
		d := NewDetector(ttl, cleanup)

		helpers.LogDetail(t, "Detector configuration:")
		helpers.LogDetail(t, "  TTL: %v", ttl)
		helpers.LogDetail(t, "  Cleanup interval: %v (active)", cleanup)
		helpers.LogSuccess(t, "Detector initialized with active cleanup loop")

		// Specification Requirement: Create message and mark as seen
		seq := uint64(1)
		nonce := "y"
		timestamp := time.Now()
		h := &mockHeader{seq: seq, nonce: nonce, timestamp: timestamp}

		helpers.LogDetail(t, "Test message:")
		helpers.LogDetail(t, "  Sequence: %d", seq)
		helpers.LogDetail(t, "  Nonce: %s", nonce)
		helpers.LogDetail(t, "  Timestamp: %s", timestamp.Format(time.RFC3339Nano))

		d.MarkPacketSeen(h)
		initialCount := d.GetSeenPacketCount()
		require.Equal(t, 1, initialCount, "should start with one tracked packet")
		helpers.LogSuccess(t, "Packet marked as seen")
		helpers.LogDetail(t, "Initial packet count: %d", initialCount)

		// Specification Requirement: Wait for TTL + cleanupInterval for cleanupLoop to run
		sleepDuration := ttl + cleanup + 10*time.Millisecond
		helpers.LogDetail(t, "Waiting %v for cleanup loop to run", sleepDuration)
		time.Sleep(sleepDuration)
		helpers.LogSuccess(t, "Cleanup loop execution period elapsed")

		finalCount := d.GetSeenPacketCount()
		require.Equal(t, 0, finalCount, "cleanupLoop should purge expired packets")
		helpers.LogSuccess(t, "Cleanup loop purged expired packets")
		helpers.LogDetail(t, "Final packet count: %d", finalCount)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Detector initialized with active cleanup",
			"Packet marked as seen initially",
			"Cleanup loop execution period elapsed",
			"Expired packets purged automatically",
			"Final packet count = 0",
			"Background cleanup working correctly",
			"Memory management successful",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case": "8.2.5_Automatic_Cleanup_Loop",
			"detector_config": map[string]interface{}{
				"ttl_ms":              ttl.Milliseconds(),
				"cleanup_interval_ms": cleanup.Milliseconds(),
			},
			"message": map[string]interface{}{
				"sequence":  seq,
				"nonce":     nonce,
				"timestamp": timestamp.Format(time.RFC3339Nano),
			},
			"cleanup": map[string]interface{}{
				"sleep_duration_ms":  sleepDuration.Milliseconds(),
				"initial_count":      initialCount,
				"final_count":        finalCount,
				"cleanup_successful": finalCount == 0,
			},
		}
		helpers.SaveTestData(t, "message/dedupe/automatic_cleanup.json", testData)
	})
}
