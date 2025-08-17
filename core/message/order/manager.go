package order

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sage-x-project/sage/core/message"
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
