package registry011

import (
	"context"
	"reflect"
	"testing"
)

func TestProviderHistorySettlementReadbackRequiresAnOriginalWriter(t *testing.T) {
	for _, o := range []*providerHistoryGroupOwner{nil, {}} {
		if out, err := o.retainSettlementCandidate(context.Background(), nil); err == nil || out != nil {
			t.Fatal("missing original store published settlement results", err)
		}
		if model, prefix, err := o.readSettlementLocked(context.Background(), nil, false); err == nil || model != nil || prefix != nil {
			t.Fatal("unselected settlement claim supplied a replay or original prefix", err)
		}
		if model, err := o.readFailedSettlementLocked(context.Background(), nil, false); err == nil || model != nil {
			t.Fatal("unselected failed-settlement claim supplied a replay", err)
		}
	}
}

func TestProviderHistoryCompletedMemberDataMismatchCannotEnterSettlement(t *testing.T) {
	for _, mode := range []string{"coordinator-head", "member-result"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyCompletedMemberFixture(t, 1)
			s, m := o.store, o.meter
			s.mu.Lock()
			if !d.completedMemberShapeLocked() || d.coordinator == o.coordinator || d.originals[0] == nil ||
				d.originals[0].result == "" || s.activeMeter.Load() != nil || m.failed || m.closed {
				s.mu.Unlock()
				t.Fatal("genuine complete member prefix under original C required")
			}
			work, reads, held, pin, ack, retention := m.actual, s.clock.read, s.stored, o.pin, o.last, o.retention
			order, coordinator, custodian, confirmed := *o.order, o.coordinator, o.custodian, o.parent.recovery.confirmed
			coordinatorHead, custodianHead := coordinator.head, custodian.head
			originals := providerHistoryProvisionedOriginalBytes(t, s.config)
			// Only this fixture's private producer DATA differs. The original C,
			// stored bytes, selected profile, holds, native clock and order stay fixed.
			switch mode {
			case "coordinator-head":
				d.coordinator.head = "private-inconsistent-completed-model-DATA"
			case "member-result":
				d.originals[0].result = ""
			}
			s.mu.Unlock()
			out, err := o.retainSettlementCandidate(context.Background(), d)
			s.mu.Lock()
			defer s.mu.Unlock()
			defer o.failLocked()
			if err == nil || out != nil || o.failed || m.failed || m.closed || o.phase != "dispatch-prepared" ||
				m.actual != work || s.clock.read != reads || s.stored != held || o.pin != pin || o.last != ack ||
				o.retention != retention || *o.order != order || s.activeMeter.Load() != nil || s.frontierFiles.historyPending != nil ||
				o.parent.recovery.confirmed != confirmed || o.parent.recovery.pending != nil ||
				o.coordinator != coordinator || o.custodian != custodian || coordinator.head != coordinatorHead || custodian.head != custodianHead ||
				!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("inconsistent completed DATA entered S, changed originals or renewed observation", mode, err)
			}
			// Deferred original cleanup is terminal; no mutation is restored and
			// neither the member request nor settlement is retried.
		})
	}
}

// The positive ordinary S fixes the final synchronous Err-check ordinal.
// Cancellation is supplied only by a standard context; this wrapper changes
// no timer, native clock sample, observer result or retained original.
type historySettlementFinalCancelContext struct {
	context.Context
	cancel        context.CancelFunc
	checks, limit int
}

func (c *historySettlementFinalCancelContext) Err() error {
	c.checks++
	if c.limit > 0 && c.checks == c.limit {
		c.cancel()
	}
	return c.Context.Err()
}

func TestProviderHistorySettlementFinalCancellationRetainsPhysicalSWithoutPublishingACK(t *testing.T) {
	o, d := historyCompletedMemberFixture(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	positive := &historySettlementFinalCancelContext{Context: ctx, cancel: cancel}
	reads := o.store.clock.read
	out, err := o.retainSettlementCandidate(positive, d)
	cancel()
	if err != nil || out == nil || out.ack == nil || o.phase != "settled" || o.failed || positive.checks < 2 ||
		o.store.clock.read != reads+2 || o.store.frontierFiles.historyPending != nil || o.parent.recovery.pending != nil {
		t.Fatal("ordinary original S and final context-check precursor", err)
	}
	limit := positive.checks
	t.Run("last-bracket", func(t *testing.T) {
		original, dispatch := historyCompletedMemberFixture(t, 1)
		store, meter, recovery := original.store, original.meter, original.parent.recovery
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		input := &historySettlementFinalCancelContext{Context: ctx, cancel: cancel, limit: limit}
		pin, ack, held, beforeReads, beforeWork := original.pin, original.last, store.stored, store.clock.read, meter.actual
		consumed, sent, next, confirmed := original.order.consumed, original.order.sent, original.order.next, recovery.confirmed
		result, err := original.retainSettlementCandidate(input, dispatch)
		claim := store.frontierFiles.historyPending
		if err == nil || result != nil || input.checks != limit || ctx.Err() != context.Canceled ||
			claim == nil || !claim.renamed || claim.used || claim.frame == nil || claim.frame.phase != "settle" ||
			claim.checkpoint == nil || recovery.pending != claim.checkpoint || recovery.confirmed != confirmed ||
			claim.frame.before != pin || claim.frame.after.sequence != pin.sequence+1 ||
			original.pin != pin || original.last != ack || store.frontierFiles.pin != pin || store.sequence != pin.sequence || store.stored != held ||
			store.clock.read != beforeReads+2 || !original.failed || !original.parent.failed || !store.failed || !meter.failed || !meter.closed ||
			store.activeMeter.Load() != nil || original.order.state != providerHistoryOrderFailed ||
			original.order.consumed != consumed || original.order.sent != sent || original.order.next != next ||
			meter.actual[0] != beforeWork[0] || meter.actual[1] != beforeWork[1] {
			t.Fatal("final cancellation published S ACK/results, skipped final clock or reset the consumed prefix", err)
		}
		for coordinate, actual := range beforeWork {
			if meter.actual[coordinate] < actual {
				t.Fatal("last-bracket refusal refunded observed work", coordinate)
			}
		}
		head, err := store.custody.root.ReadFile("HEAD")
		if err != nil || string(head) != claim.highwater || string(head) == store.frontierFiles.pinRaw {
			t.Fatal("final refusal must retain the physical S while the selected ACK remains original C", err)
		}
	})
}
