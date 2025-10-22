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
			"test_case":   "8.3.1_Validator_Replay_Detection",
			"session_id":  sessionID,
			"message": map[string]interface{}{
				"sequence":  seq,
				"nonce":     nonce,
				"timestamp": timestamp.Format(time.RFC3339Nano),
			},
			"first_validation": map[string]interface{}{
				"message_id":    messageID,
				"is_valid":      res1.IsValid,
				"is_replay":     res1.IsReplay,
				"is_duplicate":  res1.IsDuplicate,
				"is_out_of_order": res1.IsOutOfOrder,
			},
			"second_validation": map[string]interface{}{
				"message_id":    messageID2,
				"is_valid":      res2.IsValid,
				"is_replay":     res2.IsReplay,
				"error":         "replay attack detected",
			},
			"statistics": map[string]interface{}{
				"tracked_nonces":  stats["tracked_nonces"],
				"tracked_packets": stats["tracked_packets"],
			},
		}
		helpers.SaveTestData(t, "message/validator/replay_detection.json", testData)
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
