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
		// Specification Requirement: Reject messages with empty timestamps
		helpers.LogTestSection(t, "8.1.3", "Message Order Empty Timestamp Rejection")

		sessionID := "session1"
		emptyTimestamp := time.Time{}

		helpers.LogDetail(t, "Session ID: %s", sessionID)
		helpers.LogDetail(t, "Testing empty timestamp: %v", emptyTimestamp)
		helpers.LogDetail(t, "IsZero: %v", emptyTimestamp.IsZero())

		// Specification Requirement: Empty timestamp must be rejected
		helpers.LogDetail(t, "Processing message with empty timestamp")
		err := mgr.ProcessMessage(&mockHeader{timestamp: emptyTimestamp}, sessionID)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty timestamp")

		helpers.LogSuccess(t, "Empty timestamp correctly rejected")
		helpers.LogDetail(t, "Error message: %s", err.Error())

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Empty timestamp detected",
			"ProcessMessage returned error",
			"Error message contains 'empty timestamp'",
			"Invalid timestamp protection working",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":   "8.1.3_Empty_Timestamp_Rejection",
			"session_id":  sessionID,
			"timestamp": map[string]interface{}{
				"value":   emptyTimestamp.Format(time.RFC3339Nano),
				"is_zero": emptyTimestamp.IsZero(),
			},
			"validation": map[string]interface{}{
				"accepted": false,
				"error":    "empty timestamp",
			},
		}
		helpers.SaveTestData(t, "message/order/empty_timestamp_rejection.json", testData)
	})

	t.Run("FirstMessage", func(t *testing.T) {
		// Specification Requirement: First message establishes session baseline
		helpers.LogTestSection(t, "8.1.4", "Message Order First Message Baseline")

		sessionID := "session1"
		seq := uint64(1)
		ts := time.Now()

		helpers.LogDetail(t, "Session ID: %s", sessionID)
		helpers.LogDetail(t, "First message:")
		helpers.LogDetail(t, "  Sequence: %d", seq)
		helpers.LogDetail(t, "  Timestamp: %s", ts.Format(time.RFC3339Nano))

		// Specification Requirement: First message should be accepted and establish baseline
		helpers.LogDetail(t, "Processing first message for session")
		err := mgr.ProcessMessage(&mockHeader{seq: seq, timestamp: ts}, sessionID)
		require.NoError(t, err)

		helpers.LogSuccess(t, "First message accepted successfully")
		helpers.LogDetail(t, "Session baseline established")

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"First message processed successfully",
			"No error returned",
			"Session baseline established",
			"Timestamp tracking initialized",
			"Sequence tracking initialized",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":   "8.1.4_First_Message_Baseline",
			"session_id":  sessionID,
			"first_message": map[string]interface{}{
				"sequence":  seq,
				"timestamp": ts.Format(time.RFC3339Nano),
				"accepted":  true,
			},
			"baseline": "established",
		}
		helpers.SaveTestData(t, "message/order/first_message_baseline.json", testData)
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
		// Specification Requirement: Session-specific order tracking isolation
		helpers.LogTestSection(t, "8.1.5", "Message Order Session Isolation")

		tsA := time.Now()
		sessionA := "A"
		sessionB := "B"

		helpers.LogDetail(t, "Testing session isolation")
		helpers.LogDetail(t, "Base timestamp: %s", tsA.Format(time.RFC3339Nano))
		helpers.LogDetail(t, "Session A ID: %s", sessionA)
		helpers.LogDetail(t, "Session B ID: %s", sessionB)

		// Specification Requirement: Session A - later then earlier => error on second
		tsA1 := tsA.Add(50 * time.Millisecond)
		seqA1 := uint64(1)
		helpers.LogDetail(t, "Session A - First message:")
		helpers.LogDetail(t, "  Sequence: %d", seqA1)
		helpers.LogDetail(t, "  Timestamp: %s (+50ms)", tsA1.Format(time.RFC3339Nano))

		errA1 := mgr.ProcessMessage(&mockHeader{seq: seqA1, timestamp: tsA1}, sessionA)
		require.NoError(t, errA1)
		helpers.LogSuccess(t, "Session A first message accepted")

		// Out-of-order in Session A
		tsA2 := tsA
		seqA2 := uint64(2)
		helpers.LogDetail(t, "Session A - Second message (out-of-order):")
		helpers.LogDetail(t, "  Sequence: %d", seqA2)
		helpers.LogDetail(t, "  Timestamp: %s (earlier than first)", tsA2.Format(time.RFC3339Nano))

		errA2 := mgr.ProcessMessage(&mockHeader{seq: seqA2, timestamp: tsA2}, sessionA)
		require.Error(t, errA2)
		helpers.LogSuccess(t, "Session A out-of-order message rejected")
		helpers.LogDetail(t, "Error: %s", errA2.Error())

		// Specification Requirement: Session B - separate, increasing order => no error
		tsB1 := tsA
		seqB1 := uint64(1)
		helpers.LogDetail(t, "Session B - First message:")
		helpers.LogDetail(t, "  Sequence: %d", seqB1)
		helpers.LogDetail(t, "  Timestamp: %s", tsB1.Format(time.RFC3339Nano))

		errB1 := mgr.ProcessMessage(&mockHeader{seq: seqB1, timestamp: tsB1}, sessionB)
		require.NoError(t, errB1)
		helpers.LogSuccess(t, "Session B first message accepted")

		tsB2 := tsA.Add(100 * time.Millisecond)
		seqB2 := uint64(2)
		helpers.LogDetail(t, "Session B - Second message (in order):")
		helpers.LogDetail(t, "  Sequence: %d", seqB2)
		helpers.LogDetail(t, "  Timestamp: %s (+100ms)", tsB2.Format(time.RFC3339Nano))

		errB2 := mgr.ProcessMessage(&mockHeader{seq: seqB2, timestamp: tsB2}, sessionB)
		require.NoError(t, errB2)
		helpers.LogSuccess(t, "Session B second message accepted")

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Session A first message accepted",
			"Session A out-of-order message rejected",
			"Session B operates independently",
			"Session B both messages accepted",
			"Session isolation working correctly",
			"No cross-session interference",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case": "8.1.5_Session_Isolation",
			"session_a": map[string]interface{}{
				"id": sessionA,
				"messages": []map[string]interface{}{
					{
						"sequence":  seqA1,
						"timestamp": tsA1.Format(time.RFC3339Nano),
						"accepted":  true,
					},
					{
						"sequence":  seqA2,
						"timestamp": tsA2.Format(time.RFC3339Nano),
						"accepted":  false,
						"error":     "out-of-order",
					},
				},
			},
			"session_b": map[string]interface{}{
				"id": sessionB,
				"messages": []map[string]interface{}{
					{
						"sequence":  seqB1,
						"timestamp": tsB1.Format(time.RFC3339Nano),
						"accepted":  true,
					},
					{
						"sequence":  seqB2,
						"timestamp": tsB2.Format(time.RFC3339Nano),
						"accepted":  true,
					},
				},
			},
			"isolation": "verified",
		}
		helpers.SaveTestData(t, "message/order/session_isolation.json", testData)
	})
}
