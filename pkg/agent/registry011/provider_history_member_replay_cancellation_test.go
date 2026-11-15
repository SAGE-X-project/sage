package registry011

import (
	"context"
	"reflect"
	"strconv"
	"testing"
)

func TestProviderHistoryMemberReplayCancellationCannotPublishEitherRoleModel(t *testing.T) {
	for role := range 2 {
		for _, mode := range []string{"before-first-row", "after-copy"} {
			t.Run(strconv.Itoa(role)+"-"+mode, func(t *testing.T) {
				o, d := historyMemberDispatchFixture(t, 1)
				s, m := o.store, o.meter
				s.mu.Lock()
				defer s.mu.Unlock()
				// Reuse the existing original read-only preparation fixture. It
				// reserves no connection, native sample or attempted member order.
				historyMemberReplayEnter(t, o, d)
				defer o.failLocked()
				q := d.memberReplaySeal
				rows := len(q.reads[role].rows)
				if rows == 0 || !d.memberReplaySelectedLocked() {
					t.Fatal("genuine selected original replay transcript required")
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				limit := 2 // Entry, then the first row's direct context guard.
				if mode == "after-copy" {
					limit = rows + 2
				} // All rows, then the owned copy's final guard.
				input := &historySettlementFinalCancelContext{Context: ctx, cancel: cancel, limit: limit}
				work, clocks, reads, upper := m.actual, m.clocks, s.clock.read, m.upper
				pin, ack, held, order := o.pin, o.last, s.stored, *o.order
				confirmed, pending, pendingAck := o.parent.recovery.confirmed, o.parent.recovery.pending, o.parent.recovery.pendingAck
				models, replayReads, attempt := q.models, q.reads, d.attempt
				roots, dataRoots := q.modelRoots, q.readRoots
				originals := providerHistoryProvisionedOriginalBytes(t, s.config)
				out, err := d.replayMemberEquivalentLocked(input, role)
				if err == nil || out != nil || ctx.Err() != context.Canceled || input.checks != limit || m.failed || m.closed ||
					m.actual[10] <= work[10] || m.actual[0] != work[0] || m.actual[1] != work[1] || m.clocks != clocks || s.clock.read != reads || m.upper != upper ||
					o.pin != pin || o.last != ack || s.stored != held || *o.order != order || o.phase != "preparing-member" || o.failed ||
					o.parent.recovery.confirmed != confirmed || o.parent.recovery.pending != pending || o.parent.recovery.pendingAck != pendingAck ||
					q.models != models || q.reads != replayReads || q.modelRoots != roots || q.readRoots != dataRoots || d.attempt != attempt ||
					d.memberReplaySeal != q || !d.memberReplaySelectedLocked() || !o.activeLocked(m) || s.activeMeter.Load() != m ||
					!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
					t.Fatal("cancelled role replay published a partial model, renewed work or changed the original C", role, mode, err, input.checks, limit)
				}
				for coordinate, before := range work {
					if m.actual[coordinate] < before {
						t.Fatal("cancelled replay refunded observed input work", coordinate)
					}
				}
			})
		}
	}
}

func TestProviderHistoryMemberReplayCopyRefusesUnownedAndInconsistentModelData(t *testing.T) {
	for _, mode := range []string{"whole", "unowned", "changed-phase"} {
		t.Run(mode, func(t *testing.T) {
			// Constructor-produced semantic DATA and a private arithmetic observer
			// are never installed as a live store, reader, owner or dispatch model.
			model := historyLedgerFixture(t, historyLedgerMaximum())
			if mode == "unowned" {
				model = &providerHistoryGroupLedger{}
			}
			if mode == "changed-phase" {
				model.phase = "inconsistent-private-DATA"
			}
			before := *model
			meter := historyGroupCodecMeter()
			work, upper := meter.actual, meter.upper
			copy := providerHistoryMemberReplayCopyObserved(model, meter)
			if mode == "whole" {
				if copy == nil || copy == model || copy.head != model.head || !copy.validObserved(meter) {
					t.Fatal("ordinary independently owned semantic DATA copy failed")
				}
			} else if copy != nil {
				t.Fatal("unowned or inconsistent DATA was returned as a valid replay model", mode)
			}
			if !reflect.DeepEqual(*model, before) || meter.failed || meter.closed || meter.upper != upper || meter.actual[10] <= work[10] ||
				meter.actual[0] != work[0] || meter.actual[1] != work[1] || meter.clocks != 0 || meter.historyGroup != nil || meter.historyRecovery != nil {
				t.Fatal("private copy changed original DATA, refunded work or selected an execution role", mode)
			}
		})
	}
}
