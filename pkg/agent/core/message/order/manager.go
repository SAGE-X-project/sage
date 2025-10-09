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
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/message"
)

var globalSeq uint64

// Manager now only tracks last timestamp per session.
type Manager struct {
    mu                sync.Mutex
    lastSeq           map[string]uint64
    lastProcessedTime map[string]time.Time // sessionID -> last timestamp
}

// NewManager creates the per-session timestamp tracker.
func NewManager() *Manager {
    return &Manager{
        lastSeq:           make(map[string]uint64),
        lastProcessedTime: make(map[string]time.Time),
    }
}

// GetNextSequence atomically returns the next sequence number.
func GetNextSequence() uint64 {
    return atomic.AddUint64(&globalSeq, 1)
}

// ProcessMessage validates timestamp monotonicity for a single-stream gRPC.
// Returns error if out-of-order (timestamp < last).
func (om *Manager) ProcessMessage(hdr message.ControlHeader, sessionID string) error {
    seq := hdr.GetSequence()
    ts  := hdr.GetTimestamp()

    if ts.IsZero() {
        return fmt.Errorf("empty timestamp")
    }

    om.mu.Lock()
    defer om.mu.Unlock()

    lastSeq, seqExists := om.lastSeq[sessionID]
    if seqExists && seq <= lastSeq {
        return fmt.Errorf("invalid sequence: %d >= last %d", seq, lastSeq)
    }

    last, exists := om.lastProcessedTime[sessionID]
    if exists && ts.Before(last) {
        return fmt.Errorf("out-of-order: %v before %v", ts, last)
    }

    om.lastSeq[sessionID] = seq
    om.lastProcessedTime[sessionID] = ts
    return nil
}
