// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


package order

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// mockHeader implements message.ControlHeader for testing ProcessMessage
// Only Timestamp is relevant here

type mockHeader struct {
	seq       uint64
	nonce     string
	timestamp time.Time
}

func (m *mockHeader) GetSequence() uint64    { return m.seq }
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
		ts := time.Now()
		// sequence 1
		err := mgr.ProcessMessage(&mockHeader{seq: 1, timestamp: ts}, "sess2")
		require.NoError(t, err)

		// repeat sequence 1 => error
		err2 := mgr.ProcessMessage(&mockHeader{seq: 1, timestamp: ts.Add(time.Millisecond)}, "sess2")
		require.Error(t, err2)
		require.Contains(t, err2.Error(), "invalid sequence")

		// higher sequence 2 => ok
		err3 := mgr.ProcessMessage(&mockHeader{seq: 2, timestamp: ts.Add(2*time.Millisecond)}, "sess2")
		require.NoError(t, err3)
	})

	t.Run("TimestampOrder", func(t *testing.T) {
		ts := time.Now()
		// first, timestamp ts
		err := mgr.ProcessMessage(&mockHeader{seq: 10, timestamp: ts}, "sess3")
		require.NoError(t, err)

		// earlier timestamp => error
		err2 := mgr.ProcessMessage(&mockHeader{seq: 11, timestamp: ts.Add(-time.Second)}, "sess3")
		require.Error(t, err2)
		require.Contains(t, err2.Error(), "out-of-order")

		// later timestamp => ok
		err3 := mgr.ProcessMessage(&mockHeader{seq: 12, timestamp: ts.Add(time.Second)}, "sess3")
		require.NoError(t, err3)
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

		errB2 := mgr.ProcessMessage(&mockHeader{seq: 1,timestamp: tsA.Add(100 * time.Millisecond)}, "B")
		require.NoError(t, errB2)
	})
}
