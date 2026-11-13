package registry011

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"testing"
)

// Only direct Err guards in one original S method count. Native read-loop
// checks and nested model guards never calibrate this private UNIT ordinal.
type historyFailedSettlementCancelContext struct {
	context.Context
	cancel        context.CancelFunc
	target        string
	limit, checks int
}

func (c *historyFailedSettlementCancelContext) Err() error {
	var pcs [3]uintptr
	n := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	_, more := frames.Next()
	if more {
		frame, _ := frames.Next()
		if frame.Function == c.target && c.Context.Err() == nil {
			c.checks++
			if c.checks == c.limit {
				c.cancel()
			}
		}
	}
	return c.Context.Err()
}

func TestProviderHistoryFailedSettlementCancellationCannotAcknowledgeAnIncompleteRead(t *testing.T) {
	for _, mode := range []string{"first-original", "complete-originals", "final-owner-guard"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 2)
			historyRetainNormalPrefix(t, o, d, 1)
			s, m, r := o.store, o.meter, o.parent.recovery
			work, upper, clocks := m.actual, m.upper, m.maximumClocks
			pin, ack, held, retainedRaw, confirmed := o.pin, o.last, s.stored, s.retainedRaw, r.confirmed
			window, clockStart := o.windowStart, o.clockStart
			originals := providerHistoryProvisionedOriginalBytes(t, s.config)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			method, limit := "readFailedSettlementLocked", 2
			if mode == "complete-originals" {
				limit = 5 // Entry, all three originals, then the final guard.
			}
			if mode == "final-owner-guard" {
				method = "retainFailedCaptureLocked"
			}
			prefix := reflect.TypeFor[providerHistoryGroupOwner]().PkgPath() + ".(*providerHistoryGroupOwner)."
			input := &historyFailedSettlementCancelContext{Context: ctx, cancel: cancel, target: prefix + method, limit: limit}
			member := d.coordinator.ticket.table.members[1]
			connection := providerIntegrationOwnedTLS(t, member.request.body, `{}`)
			out, err := o.retainMemberCandidateWithConnection(input, d, 1, connection)
			claim := s.frontierFiles.historyPending
			if err == nil || out != nil || ctx.Err() != context.Canceled || input.checks != limit ||
				claim == nil || claim.checkpoint == nil || claim.used || claim.dispatch != d || claim.count != 3 || claim.failure.kind != "failure" ||
				claim.renamed != (mode == "final-owner-guard") || r.pending != claim.checkpoint || r.pendingAck != claim.checkpoint.ack || r.confirmed != confirmed ||
				o.pin != pin || o.last != ack || s.frontierFiles.pin != pin || s.stored != held || s.retainedRaw != retainedRaw ||
				!o.failed || !o.parent.failed || !s.failed || !m.failed || !m.closed || s.activeMeter.Load() != nil ||
				m.upper != upper || m.maximumClocks != clocks || o.windowStart != window || o.clockStart != clockStart ||
				m.actual[0] != work[0]+1 || d.retained != 1 || o.order.next != 1 || o.order.state != providerHistoryOrderFailed {
				t.Fatal("cancelled failure S issued an acknowledgment or changed its original lifetime", mode, err, input.checks)
			}
			for coordinate, before := range work {
				if m.actual[coordinate] < before {
					t.Fatal("cancelled failed S refunded observed work", coordinate)
				}
			}
			after := providerHistoryProvisionedOriginalBytes(t, s.config)
			for path, before := range originals {
				if path == filepath.Join(s.config.custodyDirectory, "HEAD") && mode == "final-owner-guard" {
					continue
				}
				if raw, present := after[path]; !present || raw != before {
					t.Fatal("cancelled failed S changed an original C file", path)
				}
			}
			head, err := s.custody.root.ReadFile("HEAD")
			expected := s.frontierFiles.pinRaw
			if mode == "final-owner-guard" {
				expected = claim.highwater
			}
			if err != nil || string(head) != expected {
				t.Fatal("physical HEAD and logical C/S acknowledgment were confused", mode, err)
			}
		})
	}
}

func TestProviderHistoryFailedSettlementNestedReadCancellationPreservesTheOriginalWriter(t *testing.T) {
	for _, limit := range []int{2, 5} {
		t.Run(strconv.Itoa(limit), func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 2)
			historyRetainNormalPrefix(t, o, d, 1)
			s, m, r := o.store, o.meter, o.parent.recovery
			pin, ack, confirmed := o.pin, o.last, r.confirmed
			syncFile := s.custody.syncFile
			defer func() { s.custody.syncFile = syncFile }()
			reached := false
			var selected *providerHistoryWriteClaim
			s.custody.syncFile = func(file *os.File) error {
				if err := syncFile(file); err != nil {
					return err
				}
				c := s.frontierFiles.historyPending
				if reached || c == nil || c.frame == nil || c.failure.kind != "failure" || filepath.Base(file.Name()) != providerCustodyName("HEAD.pending", c.frame.after.ordinal) {
					return nil
				}
				if c.count != 3 || c.renamed || c.used || !o.claimActiveLocked(c) || r.pending != c.checkpoint || d.retained != 1 {
					t.Fatal("genuine complete failed-S originals before original HEAD rename required")
				}
				reached, selected = true, c
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				target := reflect.TypeFor[providerHistoryGroupOwner]().PkgPath() + ".(*providerHistoryGroupOwner).readFailedSettlementLocked"
				input := &historyFailedSettlementCancelContext{Context: ctx, cancel: cancel, target: target, limit: limit}
				work, reads, clocks, upper := m.actual, s.clock.read, m.clocks, m.upper
				pending, pendingAck := r.pending, r.pendingAck
				before := providerHistoryProvisionedOriginalBytes(t, s.config)
				model, err := o.readFailedSettlementLocked(input, c, false)
				if err == nil || model != nil || ctx.Err() != context.Canceled || input.checks != limit || m.failed || m.closed || o.failed ||
					m.actual[10] <= work[10] || m.actual[0] != work[0] || m.actual[1] != work[1] || m.clocks != clocks || m.upper != upper || s.clock.read != reads ||
					o.pin != pin || o.last != ack || r.confirmed != confirmed || r.pending != pending || r.pendingAck != pendingAck ||
					s.frontierFiles.historyPending != c || !o.claimActiveLocked(c) || s.activeMeter.Load() != m ||
					!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
					t.Fatal("private cancellation poisoned or renewed the original failure writer", err, input.checks)
				}
				for coordinate, before := range work {
					if m.actual[coordinate] < before {
						t.Fatal("private read refunded work", coordinate)
					}
				}
				return nil
			}
			member := d.coordinator.ticket.table.members[1]
			connection := providerIntegrationOwnedTLS(t, member.request.body, `{}`)
			out, err := o.retainMemberCandidateWithConnection(context.Background(), d, 1, connection)
			if err == nil || out != nil || !reached || selected == nil || !selected.renamed || !selected.used || o.last == ack || o.last == nil ||
				o.last.claim != selected || o.pin == pin || r.confirmed != selected.checkpoint || r.pending != nil || r.pendingAck != nil ||
				s.frontierFiles.historyPending != nil || !o.failed || !m.failed || !m.closed || s.activeMeter.Load() != nil ||
				d.retained != 1 || o.order.next != 1 || o.order.state != providerHistoryOrderFailed {
				t.Fatal("private cancelled read erased the original known-failure acknowledgment", err)
			}
		})
	}
}
