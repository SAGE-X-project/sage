package registry011

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// Cancel only the independent read called directly by the outer S owner.
// Earlier reads inside the original writer keep their ordinary context.
// Stack inspection belongs to this UNIT context, never production policy.
type historySettlementOuterReadCancelContext struct {
	context.Context
	cancel         context.CancelFunc
	reader, caller string
	reached        int
}

func (c *historySettlementOuterReadCancelContext) Err() error {
	var pcs [4]uintptr
	n := runtime.Callers(1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	var names [3]string
	for i := range names {
		frame, more := frames.Next()
		names[i] = frame.Function
		if !more {
			break
		}
	}
	if names[1] == c.reader && names[2] == c.caller {
		c.reached++
		c.cancel()
	}
	return c.Context.Err()
}

func TestProviderHistorySettlementOuterReadCancellationRetainsUnacknowledgedOriginalS(t *testing.T) {
	o, d := historyCompletedMemberFixture(t, 1)
	s, m, r := o.store, o.meter, o.parent.recovery
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	prefix := reflect.TypeFor[providerHistoryGroupOwner]().PkgPath() + ".(*providerHistoryGroupOwner)."
	input := &historySettlementOuterReadCancelContext{Context: ctx, cancel: cancel,
		reader: prefix + "readSettlementLocked", caller: prefix + "retainSettlementCandidate"}
	pin, ack, confirmed, held := o.pin, o.last, r.confirmed, s.stored
	work, upper, window, clockStart := m.actual, m.upper, o.windowStart, o.clockStart
	consumed, sent, next := o.order.consumed, o.order.sent, o.order.next
	out, err := o.retainSettlementCandidate(input, d)
	c := s.frontierFiles.historyPending
	if err == nil || out != nil || ctx.Err() != context.Canceled || input.reached != 1 ||
		c == nil || !c.renamed || c.used || c.frame == nil || c.frame.phase != "settle" || c.failure.kind != "" ||
		c.checkpoint == nil || r.pending != c.checkpoint || r.pendingAck != c.checkpoint.ack || r.confirmed != confirmed ||
		c.frame.before != pin || c.frame.after.ordinal != pin.ordinal+1 || o.pin != pin || o.last != ack ||
		s.frontierFiles.pin != pin || s.stored != held || !o.failed || !o.parent.failed || !s.failed || !m.failed || !m.closed ||
		s.activeMeter.Load() != nil || m.upper != upper || o.windowStart != window || o.clockStart != clockStart ||
		o.order.state != providerHistoryOrderFailed || o.order.consumed != consumed || o.order.sent != sent || o.order.next != next ||
		m.actual[0] != work[0] || m.actual[1] != work[1] {
		t.Fatal("outer original read cancellation published S, changed its C selection or renewed execution", err)
	}
	for coordinate, before := range work {
		if m.actual[coordinate] < before {
			t.Fatal("outer read cancellation refunded observed work", coordinate)
		}
	}
	head, readErr := s.custody.root.ReadFile("HEAD")
	if readErr != nil || string(head) != c.highwater || string(head) == s.frontierFiles.pinRaw {
		t.Fatal("cancelled outer read must retain physical S under the confirmed original C", readErr)
	}
}

func TestProviderHistorySettlementNestedCancellationPreservesTheCurrentOriginalWriter(t *testing.T) {
	o, d := historyCompletedMemberFixture(t, 1)
	s, m, r := o.store, o.meter, o.parent.recovery
	syncFile := s.custody.syncFile
	defer func() { s.custody.syncFile = syncFile }()
	reached := false
	s.custody.syncFile = func(file *os.File) error {
		if err := syncFile(file); err != nil {
			return err
		}
		c := s.frontierFiles.historyPending
		if reached || c == nil || c.frame == nil || c.frame.phase != "settle" ||
			filepath.Base(file.Name()) != providerCustodyName("HEAD.pending", c.frame.after.ordinal) {
			return nil
		}
		if !o.claimActiveLocked(c) || c.renamed || c.used || c.failure.kind != "" || r.pending != c.checkpoint ||
			r.confirmed == nil || r.confirmed.ack != o.last || r.pending == nil || r.pending.ack == o.last {
			t.Fatal("genuine original pre-rename S writer required")
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// Exactly two direct pre-I/O checks: readSettlementLocked, then
		// replayCommittedConsumptionAtLocked. No native reader loop intervenes.
		input := &historySettlementFinalCancelContext{Context: ctx, cancel: cancel, limit: 2}
		work, clocks, reads, pin, ack, held := m.actual, m.clocks, s.clock.read, o.pin, o.last, s.stored
		upper, confirmed, pending, pendingAck := m.upper, r.confirmed, r.pending, r.pendingAck
		originals := providerHistoryProvisionedOriginalBytes(t, s.config)
		model, prefix, err := o.readSettlementLocked(input, c, false)
		if err == nil || model != nil || prefix != nil || input.checks != 2 || ctx.Err() != context.Canceled ||
			m.actual != work || m.clocks != clocks || s.clock.read != reads || m.upper != upper || m.failed || m.closed ||
			o.pin != pin || o.last != ack || s.stored != held || r.confirmed != confirmed || r.pending != pending || r.pendingAck != pendingAck ||
			s.frontierFiles.historyPending != c || !o.claimActiveLocked(c) || s.activeMeter.Load() != m ||
			!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
			t.Fatal("nested cancelled read touched original inputs, published DATA or stopped its separate writer context", err)
		}
		reached = true
		return nil
	}
	out, err := o.retainSettlementCandidate(context.Background(), d)
	if err != nil || out == nil || !reached || out.ack != o.last || o.phase != "settled" || o.failed || m.failed ||
		s.activeMeter.Load() != nil || s.frontierFiles.historyPending != nil || r.pending != nil || r.pendingAck != nil ||
		r.confirmed == nil || r.confirmed.ack != out.ack {
		t.Fatal("a cancelled private read changed the normal original S completion", err)
	}
}
