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

package order

import (
	"testing"
	"time"

	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/require"
)

// mockHeader implements message.ControlHeader for testing ProcessMessage
// Only Timestamp is relevant here

type mockHeader struct {
	seq       uint64
	nonce     string
	timestamp time.Time
}

func (m *mockHeader) GetSequence() uint64     { return m.seq }
func (m *mockHeader) GetNonce() string        { return m.nonce }
func (m *mockHeader) GetTimestamp() time.Time { return m.timestamp }

func TestOrderManager(t *testing.T) {
	mgr := NewManager()

	t.Run("EmptyTimestamp", func(t *testing.T) {
		err := mgr.ProcessMessage(&mockHeader{timestamp: time.Time{}}, "session1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty timestamp")
	})

	t.Run("FirstMessage", func(t *testing.T) {
		ts := time.Now()
		err := mgr.ProcessMessage(&mockHeader{timestamp: ts}, "session1")
		require.NoError(t, err)
	})

	t.Run("SeqMonotonicity", func(t *testing.T) {
		// Specification Requirement: Sequence number monotonic increase validation for replay attack prevention
		helpers.LogTestSection(t, "8.1.1", "Message Sequence Number Monotonicity")

		ts := time.Now()
		sessionID := "sess2"
		helpers.LogDetail(t, "Session ID: %s", sessionID)
		helpers.LogDetail(t, "Base timestamp: %s", ts.Format(time.RFC3339Nano))

		// Specification Requirement: First message with sequence 1
		seq1 := uint64(1)
		helpers.LogDetail(t, "Processing message with sequence: %d", seq1)
		err := mgr.ProcessMessage(&mockHeader{seq: seq1, timestamp: ts}, sessionID)
		require.NoError(t, err)
		helpers.LogSuccess(t, "First message (seq=1) accepted")

		// Specification Requirement: Replay attack detection - duplicate sequence must be rejected
		helpers.LogDetail(t, "Attempting replay with same sequence: %d", seq1)
		err2 := mgr.ProcessMessage(&mockHeader{seq: seq1, timestamp: ts.Add(time.Millisecond)}, sessionID)
		require.Error(t, err2)
		require.Contains(t, err2.Error(), "invalid sequence")
		helpers.LogSuccess(t, "Replay attack detected: Duplicate sequence rejected")
		helpers.LogDetail(t, "Error message: %s", err2.Error())

		// Specification Requirement: Monotonic increase - higher sequence must be accepted
		seq2 := uint64(2)
		helpers.LogDetail(t, "Processing message with higher sequence: %d", seq2)
		err3 := mgr.ProcessMessage(&mockHeader{seq: seq2, timestamp: ts.Add(2 * time.Millisecond)}, sessionID)
		require.NoError(t, err3)
		helpers.LogSuccess(t, "Higher sequence (seq=2) accepted")

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"First message with seq=1 accepted",
			"Duplicate sequence rejected (replay attack)",
			"Higher sequence accepted (monotonic increase)",
			"Error message contains 'invalid sequence'",
			"Session-specific sequence tracking",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":   "8.1.1_Sequence_Monotonicity",
			"session_id":  sessionID,
			"sequence_1": map[string]interface{}{
				"value":    seq1,
				"accepted": true,
			},
			"sequence_replay": map[string]interface{}{
				"value":    seq1,
				"accepted": false,
				"error":    "invalid sequence",
			},
			"sequence_2": map[string]interface{}{
				"value":    seq2,
				"accepted": true,
			},
			"monotonicity": "enforced",
		}
		helpers.SaveTestData(t, "message/order/sequence_monotonicity.json", testData)
	})

	t.Run("TimestampOrder", func(t *testing.T) {
		// Specification Requirement: Timestamp ordering validation for temporal consistency
		helpers.LogTestSection(t, "8.1.2", "Message Timestamp Ordering")

		ts := time.Now()
		sessionID := "sess3"
		helpers.LogDetail(t, "Session ID: %s", sessionID)
		helpers.LogDetail(t, "Base timestamp: %s", ts.Format(time.RFC3339Nano))

		// Specification Requirement: First message establishes baseline
		seq1 := uint64(10)
		helpers.LogDetail(t, "First message - seq=%d, timestamp=%s", seq1, ts.Format(time.RFC3339Nano))
		err := mgr.ProcessMessage(&mockHeader{seq: seq1, timestamp: ts}, sessionID)
		require.NoError(t, err)
		helpers.LogSuccess(t, "Baseline timestamp established")

		// Specification Requirement: Earlier timestamp must be rejected (out-of-order)
		seq2 := uint64(11)
		earlierTS := ts.Add(-time.Second)
		helpers.LogDetail(t, "Second message - seq=%d, timestamp=%s (1 second earlier)", seq2, earlierTS.Format(time.RFC3339Nano))
		err2 := mgr.ProcessMessage(&mockHeader{seq: seq2, timestamp: earlierTS}, sessionID)
		require.Error(t, err2)
		require.Contains(t, err2.Error(), "out-of-order")
		helpers.LogSuccess(t, "Out-of-order timestamp rejected")
		helpers.LogDetail(t, "Error message: %s", err2.Error())

		// Specification Requirement: Later timestamp must be accepted
		seq3 := uint64(12)
		laterTS := ts.Add(time.Second)
		helpers.LogDetail(t, "Third message - seq=%d, timestamp=%s (1 second later)", seq3, laterTS.Format(time.RFC3339Nano))
		err3 := mgr.ProcessMessage(&mockHeader{seq: seq3, timestamp: laterTS}, sessionID)
		require.NoError(t, err3)
		helpers.LogSuccess(t, "Later timestamp accepted")

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Baseline timestamp established",
			"Earlier timestamp rejected (out-of-order)",
			"Later timestamp accepted",
			"Error message contains 'out-of-order'",
			"Temporal consistency enforced",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":   "8.1.2_Timestamp_Ordering",
			"session_id":  sessionID,
			"baseline": map[string]interface{}{
				"sequence":  seq1,
				"timestamp": ts.Format(time.RFC3339Nano),
				"accepted":  true,
			},
			"earlier_timestamp": map[string]interface{}{
				"sequence":       seq2,
				"timestamp":      earlierTS.Format(time.RFC3339Nano),
				"delta_seconds":  -1.0,
				"accepted":       false,
				"error":          "out-of-order",
			},
			"later_timestamp": map[string]interface{}{
				"sequence":      seq3,
				"timestamp":     laterTS.Format(time.RFC3339Nano),
				"delta_seconds": 1.0,
				"accepted":      true,
			},
		}
		helpers.SaveTestData(t, "message/order/timestamp_ordering.json", testData)
	})

	t.Run("SessionIsolation", func(t *testing.T) {
		// Session A: later then earlier => error on second
		tsA := time.Now()

		errA1 := mgr.ProcessMessage(&mockHeader{timestamp: tsA.Add(50 * time.Millisecond)}, "A")
		require.NoError(t, errA1)

		errA2 := mgr.ProcessMessage(&mockHeader{timestamp: tsA}, "A")
		require.Error(t, errA2)

		// Session B: separate, increasing order => no error
		errB1 := mgr.ProcessMessage(&mockHeader{timestamp: tsA}, "B")
		require.NoError(t, errB1)

		errB2 := mgr.ProcessMessage(&mockHeader{seq: 1, timestamp: tsA.Add(100 * time.Millisecond)}, "B")
		require.NoError(t, errB2)
	})
}
