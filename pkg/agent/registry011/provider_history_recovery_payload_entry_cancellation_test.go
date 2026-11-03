package registry011

import (
	"context"
	"reflect"
	"testing"
)

// Each trial first constructs its genuine failed producer, catalog and all
// preceding original replay roles. Cancellation occurs only after those real
// prerequisites, without changing a live quota, owner, selection or clock.
func TestProviderHistoryRecoveryPayloadEntryCancellationPreservesPriorRoles(t *testing.T) {
	for _, mode := range []string{"group-model", "completed-members", "tail-payloads", "catalog-closure"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryCatalogFixture(t, "consumed")
			if out, err := r.beginCatalogCandidate(context.Background()); err != nil || out == nil || !out.selectedLocked(r) {
				t.Fatal("genuine original catalog prerequisite", err)
			}
			if mode == "catalog-closure" {
				historyRecoveryCatalogModels(t, r, want)
			} else {
				historyRecoveryGroupInitial(t, r, 0, want)
				if mode != "group-model" {
					if out, err := r.replayGroupModelCandidate(context.Background(), 0); err != nil || out == nil || !out.selectedLocked(r) {
						t.Fatal("genuine original group replay prerequisite", err)
					}
				}
				if mode == "tail-payloads" {
					if out, err := r.replayCompletedMemberPayloadsCandidate(context.Background(), 0); err != nil || out == nil || !out.selectedLocked(r) {
						t.Fatal("genuine completed-member replay prerequisite", err)
					}
				}
			}
			s := o.store
			if !o.failed || !o.meter.closed || !r.active || r.closed || r.failed || s.activeMeter.Load() != nil {
				t.Fatal("failed producer and still-active read-only lane prerequisite")
			}
			producer, reads, held, pin, ack := o.meter.actual, s.clock.read, s.stored, o.pin, o.last
			work, upper, files, rawBytes, clocks := r.meter.actual, r.meter.upper, r.inputFiles, r.inputBytes, r.meter.clocks
			initial, groups, members, tails := r.initialModels, r.groupModels, r.completedMemberModels, r.tailModels
			catalog, confirmed := r.catalogBefore, r.confirmed
			before := providerHistoryProvisionedOriginalBytes(t, s.config)
			invoke := func(ctx context.Context) (bool, error) {
				switch mode {
				case "group-model":
					out, err := r.replayGroupModelCandidate(ctx, 0)
					return out == nil, err
				case "completed-members":
					out, err := r.replayCompletedMemberPayloadsCandidate(ctx, 0)
					return out == nil, err
				case "tail-payloads":
					out, err := r.replayTailPayloadsCandidate(ctx, 0)
					return out == nil, err
				default:
					out, err := r.finishCatalogCandidate(ctx)
					return out == nil, err
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if empty, err := invoke(ctx); err == nil || !empty || r.active || !r.closed || !r.failed || !r.meter.closed || !r.meter.failed {
				t.Fatal("entered cancellation did not close the original inspection lane", err)
			}
			if empty, err := invoke(context.Background()); err == nil || !empty || r.active || !r.closed || !r.failed ||
				r.meter.actual != work || r.meter.upper != upper || r.meter.clocks != clocks || r.inputFiles != files || r.inputBytes != rawBytes ||
				r.initialModels != initial || r.groupModels != groups || r.completedMemberModels != members || r.tailModels != tails ||
				r.catalogBefore != catalog || r.catalogAfter != nil || r.catalogResult != nil || r.confirmed != confirmed ||
				o.meter.actual != producer || s.clock.read != reads || s.stored != held || o.pin != pin || o.last != ack ||
				!o.failed || !o.meter.closed || s.activeMeter.Load() != nil || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("cancelled inspection replaced a prior role, changed originals or renewed execution", err)
			}
		})
	}
}
