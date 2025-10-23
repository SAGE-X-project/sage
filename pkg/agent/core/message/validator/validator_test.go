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

package validator

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/require"
)

// mockHeader implements message.ControlHeader for testing
// Uses Sequence, Nonce, Timestamp

type mockHeader struct {
	seq       uint64
	nonce     string
	timestamp time.Time
}

func (m *mockHeader) GetSequence() uint64     { return m.seq }
func (m *mockHeader) GetNonce() string        { return m.nonce }
func (m *mockHeader) GetTimestamp() time.Time { return m.timestamp }

func TestValidateMessage(t *testing.T) {
	t.Run("ValidAndStats", func(t *testing.T) {
		// Specification Requirement: Message validation with statistics tracking
		helpers.LogTestSection(t, "8.3.2", "Message Validator Valid Message and Statistics")

		cfg := &ValidatorConfig{
			TimestampTolerance:  time.Second,
			NonceTTL:            time.Minute,
			DuplicateTTL:        time.Minute,
			MaxOutOfOrderWindow: time.Second,
			CleanupInterval:     time.Second,
		}
		helpers.LogDetail(t, "Validator configuration:")
		helpers.LogDetail(t, "  Timestamp tolerance: %v", cfg.TimestampTolerance)
		helpers.LogDetail(t, "  Nonce TTL: %v", cfg.NonceTTL)
		helpers.LogDetail(t, "  Duplicate TTL: %v", cfg.DuplicateTTL)
		helpers.LogDetail(t, "  Max out-of-order window: %v", cfg.MaxOutOfOrderWindow)

		mv := NewMessageValidator(cfg)
		helpers.LogSuccess(t, "Message validator initialized")

		// Specification Requirement: Create valid message with current timestamp
		now := time.Now()
		nonce := uuid.NewString()
		messageID := uuid.NewString()
		sessionID := "sess1"
		seq := uint64(1)
		head := &mockHeader{seq: seq, nonce: nonce, timestamp: now}

		helpers.LogDetail(t, "Test message:")
		helpers.LogDetail(t, "  Sequence: %d", seq)
		helpers.LogDetail(t, "  Nonce: %s", nonce)
		helpers.LogDetail(t, "  Timestamp: %s", now.Format(time.RFC3339Nano))
		helpers.LogDetail(t, "  Session ID: %s", sessionID)
		helpers.LogDetail(t, "  Message ID: %s", messageID)

		// Specification Requirement: Validate message
		res := mv.ValidateMessage(head, sessionID, messageID)
		require.True(t, res.IsValid)
		require.False(t, res.IsReplay)
		require.False(t, res.IsDuplicate)
		require.False(t, res.IsOutOfOrder)

		helpers.LogSuccess(t, "Message validated successfully")
		helpers.LogDetail(t, "Validation result:")
		helpers.LogDetail(t, "  Is valid: %v", res.IsValid)
		helpers.LogDetail(t, "  Is replay: %v", res.IsReplay)
		helpers.LogDetail(t, "  Is duplicate: %v", res.IsDuplicate)
		helpers.LogDetail(t, "  Is out-of-order: %v", res.IsOutOfOrder)

		// Specification Requirement: Verify statistics tracking
		stats := mv.GetStats()
		require.EqualValues(t, 1, stats["tracked_nonces"], "one nonce marked")
		require.EqualValues(t, 1, stats["tracked_packets"], "one packet tracked")

		helpers.LogSuccess(t, "Statistics verified")
		helpers.LogDetail(t, "Validator statistics:")
		helpers.LogDetail(t, "  Tracked nonces: %d", stats["tracked_nonces"])
		helpers.LogDetail(t, "  Tracked packets: %d", stats["tracked_packets"])

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Validator initialized with configuration",
			"Valid message created with current timestamp",
			"Message validated as valid",
			"No replay flag set",
			"No duplicate flag set",
			"No out-of-order flag set",
			"Statistics show one tracked nonce",
			"Statistics show one tracked packet",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":  "8.3.2_Validator_Valid_Message_Stats",
			"session_id": sessionID,
			"message": map[string]interface{}{
				"sequence":   seq,
				"nonce":      nonce,
				"message_id": messageID,
				"timestamp":  now.Format(time.RFC3339Nano),
			},
			"validation_result": map[string]interface{}{
				"is_valid":        res.IsValid,
				"is_replay":       res.IsReplay,
				"is_duplicate":    res.IsDuplicate,
				"is_out_of_order": res.IsOutOfOrder,
			},
			"statistics": map[string]interface{}{
				"tracked_nonces":  stats["tracked_nonces"],
				"tracked_packets": stats["tracked_packets"],
			},
		}
		helpers.SaveTestData(t, "message/validator/valid_stats.json", testData)
	})

	t.Run("TimestampOutsideTolerance", func(t *testing.T) {
		// Specification Requirement: Reject messages with timestamps outside tolerance window
		helpers.LogTestSection(t, "8.3.3", "Message Validator Timestamp Tolerance")

		cfg := &ValidatorConfig{
			TimestampTolerance:  10 * time.Millisecond,
			NonceTTL:            time.Minute,
			DuplicateTTL:        time.Minute,
			MaxOutOfOrderWindow: time.Second,
			CleanupInterval:     time.Hour,
		}
		helpers.LogDetail(t, "Validator configuration:")
		helpers.LogDetail(t, "  Timestamp tolerance: %v (strict)", cfg.TimestampTolerance)
		helpers.LogDetail(t, "  Nonce TTL: %v", cfg.NonceTTL)
		helpers.LogDetail(t, "  Duplicate TTL: %v", cfg.DuplicateTTL)

		mv := NewMessageValidator(cfg)
		helpers.LogSuccess(t, "Message validator initialized with strict tolerance")

		// Specification Requirement: Create message with old timestamp (outside tolerance)
		old := time.Now().Add(-time.Second)
		nonce := uuid.NewString()
		messageID := uuid.NewString()
		sessionID := "sess"
		seq := uint64(1)
		head := &mockHeader{seq: seq, nonce: nonce, timestamp: old}

		helpers.LogDetail(t, "Test message with old timestamp:")
		helpers.LogDetail(t, "  Sequence: %d", seq)
		helpers.LogDetail(t, "  Nonce: %s", nonce)
		helpers.LogDetail(t, "  Timestamp: %s (1 second ago)", old.Format(time.RFC3339Nano))
		helpers.LogDetail(t, "  Session ID: %s", sessionID)
		helpers.LogDetail(t, "  Message ID: %s", messageID)
		helpers.LogDetail(t, "  Age relative to now: %v", time.Since(old))

		// Specification Requirement: Validation should fail for old timestamp
		helpers.LogDetail(t, "Validating message with old timestamp")
		res := mv.ValidateMessage(head, sessionID, messageID)
		require.False(t, res.IsValid)
		require.Contains(t, res.Error.Error(), "timestamp outside tolerance window")

		helpers.LogSuccess(t, "Message correctly rejected due to timestamp")
		helpers.LogDetail(t, "Validation result:")
		helpers.LogDetail(t, "  Is valid: %v", res.IsValid)
		helpers.LogDetail(t, "  Error: %s", res.Error.Error())

		// Specification Requirement: Verify no tracking occurs for invalid message
		stats := mv.GetStats()
		require.EqualValues(t, 0, stats["tracked_nonces"], "no nonce used on invalid")
		require.EqualValues(t, 0, stats["tracked_packets"], "no packet tracked on invalid")

		helpers.LogSuccess(t, "Statistics confirmed no tracking")
		helpers.LogDetail(t, "Validator statistics:")
		helpers.LogDetail(t, "  Tracked nonces: %d", stats["tracked_nonces"])
		helpers.LogDetail(t, "  Tracked packets: %d", stats["tracked_packets"])

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Validator initialized with strict 10ms tolerance",
			"Message created with timestamp 1 second old",
			"Message validation failed as expected",
			"Error message contains 'timestamp outside tolerance window'",
			"No nonce tracked for invalid message",
			"No packet tracked for invalid message",
			"Timestamp protection working correctly",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":  "8.3.3_Validator_Timestamp_Tolerance",
			"session_id": sessionID,
			"configuration": map[string]interface{}{
				"timestamp_tolerance_ms": cfg.TimestampTolerance.Milliseconds(),
			},
			"message": map[string]interface{}{
				"sequence":    seq,
				"nonce":       nonce,
				"message_id":  messageID,
				"timestamp":   old.Format(time.RFC3339Nano),
				"age_seconds": time.Since(old).Seconds(),
			},
			"validation_result": map[string]interface{}{
				"is_valid": res.IsValid,
				"error":    "timestamp outside tolerance window",
			},
			"statistics": map[string]interface{}{
				"tracked_nonces":  stats["tracked_nonces"],
				"tracked_packets": stats["tracked_packets"],
			},
		}
		helpers.SaveTestData(t, "message/validator/timestamp_tolerance.json", testData)
	})

	t.Run("ReplayDetection", func(t *testing.T) {
		// Specification Requirement: Comprehensive message validation with replay attack detection
		helpers.LogTestSection(t, "8.3.1", "Message Validator Replay Detection")

		mv := NewMessageValidator(nil)
		helpers.LogSuccess(t, "Message validator initialized")

		// Specification Requirement: Generate unique nonce for message
		nonce := uuid.NewString()
		seq := uint64(1)
		timestamp := time.Now()
		sessionID := "sess"
		messageID := uuid.NewString()

		helpers.LogDetail(t, "Test message:")
		helpers.LogDetail(t, "  Sequence: %d", seq)
		helpers.LogDetail(t, "  Nonce: %s", nonce)
		helpers.LogDetail(t, "  Timestamp: %s", timestamp.Format(time.RFC3339Nano))
		helpers.LogDetail(t, "  Session ID: %s", sessionID)
		helpers.LogDetail(t, "  Message ID: %s", messageID)

		head := &mockHeader{seq: seq, nonce: nonce, timestamp: timestamp}

		// Specification Requirement: First message should be valid
		res1 := mv.ValidateMessage(head, sessionID, messageID)
		require.True(t, res1.IsValid)
		require.False(t, res1.IsReplay)
		require.False(t, res1.IsDuplicate)
		require.False(t, res1.IsOutOfOrder)

		helpers.LogSuccess(t, "First message validated successfully")
		helpers.LogDetail(t, "Validation result:")
		helpers.LogDetail(t, "  Is valid: %v", res1.IsValid)
		helpers.LogDetail(t, "  Is replay: %v", res1.IsReplay)
		helpers.LogDetail(t, "  Is duplicate: %v", res1.IsDuplicate)
		helpers.LogDetail(t, "  Is out-of-order: %v", res1.IsOutOfOrder)

		// Specification Requirement: Second message with same nonce must be rejected (replay attack)
		messageID2 := uuid.NewString()
		helpers.LogDetail(t, "Attempting replay with same nonce")
		helpers.LogDetail(t, "  Second message ID: %s", messageID2)

		res2 := mv.ValidateMessage(head, sessionID, messageID2)
		require.False(t, res2.IsValid)
		require.True(t, res2.IsReplay)
		require.Equal(t, "nonce has been used before (replay attack detected)", res2.Error.Error())

		helpers.LogSuccess(t, "Replay attack detected and prevented")
		helpers.LogDetail(t, "Validation result:")
		helpers.LogDetail(t, "  Is valid: %v", res2.IsValid)
		helpers.LogDetail(t, "  Is replay: %v", res2.IsReplay)
		helpers.LogDetail(t, "  Error: %s", res2.Error.Error())

		// Specification Requirement: Statistics validation
		stats := mv.GetStats()
		require.EqualValues(t, 1, stats["tracked_nonces"], "only first nonce is tracked")
		require.EqualValues(t, 1, stats["tracked_packets"], "only first packet is tracked")

		helpers.LogDetail(t, "Validator statistics:")
		helpers.LogDetail(t, "  Tracked nonces: %d", stats["tracked_nonces"])
		helpers.LogDetail(t, "  Tracked packets: %d", stats["tracked_packets"])

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Validator initialized successfully",
			"First message validated as valid",
			"Second message (replay) detected",
			"Replay flag set to true",
			"Error message correct",
			"Only first nonce tracked",
			"Only first packet tracked",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":  "8.3.1_Validator_Replay_Detection",
			"session_id": sessionID,
			"message": map[string]interface{}{
				"sequence":  seq,
				"nonce":     nonce,
				"timestamp": timestamp.Format(time.RFC3339Nano),
			},
			"first_validation": map[string]interface{}{
				"message_id":      messageID,
				"is_valid":        res1.IsValid,
				"is_replay":       res1.IsReplay,
				"is_duplicate":    res1.IsDuplicate,
				"is_out_of_order": res1.IsOutOfOrder,
			},
			"second_validation": map[string]interface{}{
				"message_id": messageID2,
				"is_valid":   res2.IsValid,
				"is_replay":  res2.IsReplay,
				"error":      "replay attack detected",
			},
			"statistics": map[string]interface{}{
				"tracked_nonces":  stats["tracked_nonces"],
				"tracked_packets": stats["tracked_packets"],
			},
		}
		helpers.SaveTestData(t, "message/validator/replay_detection.json", testData)
	})

	t.Run("OutOfOrderError", func(t *testing.T) {
		// Specification Requirement: Detect out-of-order messages exceeding window
		helpers.LogTestSection(t, "8.3.4", "Message Validator Out-of-Order Detection")

		cfg := &ValidatorConfig{
			TimestampTolerance:  time.Second,
			NonceTTL:            time.Minute,
			DuplicateTTL:        time.Minute,
			MaxOutOfOrderWindow: 50 * time.Millisecond,
			CleanupInterval:     time.Hour,
		}
		helpers.LogDetail(t, "Validator configuration:")
		helpers.LogDetail(t, "  Timestamp tolerance: %v", cfg.TimestampTolerance)
		helpers.LogDetail(t, "  Max out-of-order window: %v (strict)", cfg.MaxOutOfOrderWindow)
		helpers.LogDetail(t, "  Nonce TTL: %v", cfg.NonceTTL)

		mv := NewMessageValidator(cfg)
		helpers.LogSuccess(t, "Message validator initialized with strict order window")

		// Specification Requirement: First message establishes baseline
		t1 := time.Now()
		nonce1 := uuid.NewString()
		messageID1 := uuid.NewString()
		sessionID := "sess"
		seq1 := uint64(1)
		head1 := &mockHeader{seq: seq1, nonce: nonce1, timestamp: t1}

		helpers.LogDetail(t, "First message (baseline):")
		helpers.LogDetail(t, "  Sequence: %d", seq1)
		helpers.LogDetail(t, "  Nonce: %s", nonce1)
		helpers.LogDetail(t, "  Timestamp: %s", t1.Format(time.RFC3339Nano))
		helpers.LogDetail(t, "  Message ID: %s", messageID1)

		res1 := mv.ValidateMessage(head1, sessionID, messageID1)
		require.True(t, res1.IsValid)
		helpers.LogSuccess(t, "First message validated successfully")

		// Specification Requirement: Second message with earlier timestamp triggers order error
		t2 := t1.Add(-100 * time.Millisecond)
		nonce2 := uuid.NewString()
		messageID2 := uuid.NewString()
		seq2 := uint64(2)
		head2 := &mockHeader{seq: seq2, nonce: nonce2, timestamp: t2}

		helpers.LogDetail(t, "Second message (out-of-order):")
		helpers.LogDetail(t, "  Sequence: %d", seq2)
		helpers.LogDetail(t, "  Nonce: %s", nonce2)
		helpers.LogDetail(t, "  Timestamp: %s (100ms earlier)", t2.Format(time.RFC3339Nano))
		helpers.LogDetail(t, "  Message ID: %s", messageID2)
		helpers.LogDetail(t, "  Time difference: %v (exceeds %v window)", t1.Sub(t2), cfg.MaxOutOfOrderWindow)

		helpers.LogDetail(t, "Validating out-of-order message")
		res2 := mv.ValidateMessage(head2, sessionID, messageID2)
		require.False(t, res2.IsValid)
		require.Contains(t, res2.Error.Error(), "order validation failed")

		helpers.LogSuccess(t, "Out-of-order message correctly rejected")
		helpers.LogDetail(t, "Validation result:")
		helpers.LogDetail(t, "  Is valid: %v", res2.IsValid)
		helpers.LogDetail(t, "  Error: %s", res2.Error.Error())

		// Specification Requirement: Verify only first message tracked
		stats := mv.GetStats()
		require.EqualValues(t, 1, stats["tracked_nonces"], "no new nonce marked on error")
		require.EqualValues(t, 1, stats["tracked_packets"], "no new packet tracked on error")

		helpers.LogSuccess(t, "Statistics confirmed single message tracked")
		helpers.LogDetail(t, "Validator statistics:")
		helpers.LogDetail(t, "  Tracked nonces: %d", stats["tracked_nonces"])
		helpers.LogDetail(t, "  Tracked packets: %d", stats["tracked_packets"])

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Validator initialized with 50ms order window",
			"First message validated as baseline",
			"Second message with earlier timestamp created",
			"Time difference (100ms) exceeds allowed window (50ms)",
			"Out-of-order message validation failed",
			"Error message contains 'order validation failed'",
			"Only first message tracked in statistics",
			"Order protection working correctly",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":  "8.3.4_Validator_Out_Of_Order",
			"session_id": sessionID,
			"configuration": map[string]interface{}{
				"max_out_of_order_window_ms": cfg.MaxOutOfOrderWindow.Milliseconds(),
			},
			"first_message": map[string]interface{}{
				"sequence":   seq1,
				"nonce":      nonce1,
				"message_id": messageID1,
				"timestamp":  t1.Format(time.RFC3339Nano),
			},
			"second_message": map[string]interface{}{
				"sequence":     seq2,
				"nonce":        nonce2,
				"message_id":   messageID2,
				"timestamp":    t2.Format(time.RFC3339Nano),
				"time_diff_ms": t1.Sub(t2).Milliseconds(),
			},
			"validation_results": map[string]interface{}{
				"first_valid":  res1.IsValid,
				"second_valid": res2.IsValid,
				"second_error": "order validation failed",
			},
			"statistics": map[string]interface{}{
				"tracked_nonces":  stats["tracked_nonces"],
				"tracked_packets": stats["tracked_packets"],
			},
		}
		helpers.SaveTestData(t, "message/validator/out_of_order.json", testData)
	})
}
