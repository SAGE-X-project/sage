package registry011

import (
	"context"
	"reflect"
	"testing"
)

// These are detached partial-method inputs derived from genuine retained S
// originals. They never select a recovery owner, install an observer, open a
// file or replace a selected checkpoint, clock or resource allowance.
func TestProviderHistoryRecoverySettlementRejectsIncompletePrivateInputsBeforeReads(t *testing.T) {
	o, dispatch, original := historyRecoveryArchivedSettlement(t, 1)
	for _, mode := range []string{
		"nil-context", "cancelled", "missing-retention", "missing-ticket", "missing-consumption",
		"missing-previous", "previous-phase", "completed-negative", "completed-too-many", "clock-window",
		"failed-all-completed", "complete-incomplete", "inventory-count", "settlement-kind",
		"policy-shape", "provider-shape", "missing-roots", "failed-object-kind", "failed-attempt-missing",
		"result-kind", "invalid-result-pack",
	} {
		t.Run(mode, func(t *testing.T) {
			o.store.mu.Lock()
			defer o.store.mu.Unlock()
			meter := historyGroupCodecMeter()
			ledger := providerHistoryLedgerCopyObserved(dispatch.coordinator, meter)
			policy := *o.store.config.policy
			decoded, ok := providerReplayJSON(providerReplayBytes(policy.raw, meter), meter).(map[string]any)
			provider, providerOK := decoded["provider"].(map[string]any)
			roots := contentString(provider["tls_roots_artifact"])
			if ledger == nil || ledger.phase != "consumed" || ledger.ticket == nil || ledger.consumption == nil ||
				!ok || !providerOK || !lowerHash(roots) || original.previous == nil || original.admission == nil ||
				original.inventory.count != 2 || !providerHistoryGroupClockFits(original.admission.clock, original.frame.clock) ||
				!original.memberArchive.shapeLocked(o.parent.recovery, original) {
				t.Fatal("genuine complete original S and independent consumed-ledger DATA required")
			}
			objects := original.claim.fresh
			pack, parsed, err := parseProviderHistoryResultPackObserved(providerReplayBytes(objects[0].raw, meter), objects[0].digest,
				ledger.ticket, ledger.consumption, original.completed, roots, o.parent.tls, o.store.config.bounds.maximumObjectBytes, meter)
			if err != nil || pack == nil || parsed[0].digest != original.memberArchive.members[0].result {
				t.Fatal("positive retained original result-pack prerequisite", err)
			}
			// The private subject has no owner, selected pass, live store or
			// recovery lane. Only value copies and immutable originals enter it.
			recovery := &providerHistoryRecoveryOwner{meter: meter, parent: &providerHistoryProgramOwner{tls: o.parent.tls},
				config: providerCustodyConfig{bounds: o.store.config.bounds}}
			subject := &providerHistoryRecoveryGroupModel{recovery: recovery, ledger: ledger,
				initial: &providerHistoryRecoveryInitialModel{frontier: &providerFrontierReplay{body: &providerCustodyReplay{policy: &policy}}}}
			checkpoint, previous, frame := *original, *original.previous, original.frame
			checkpoint.previous = &previous
			record := providerHistoryRecoveryGroupRecord{frame: &frame, objects: objects}
			retained := previous.retention
			retention := &retained
			input := context.Background()
			switch mode {
			case "nil-context":
				input = nil
			case "cancelled":
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				input = ctx
			case "missing-retention":
				retention = nil
			case "missing-ticket":
				ledger.ticket = nil
			case "missing-consumption":
				ledger.consumption = nil
			case "missing-previous":
				checkpoint.previous = nil
			case "previous-phase":
				previous.frame.phase = "admit"
			case "completed-negative":
				checkpoint.completed = -1
			case "completed-too-many":
				checkpoint.completed = ledger.ticket.table.count + 1
			case "clock-window":
				frame.clock.MonotonicMS += 300001
			case "failed-all-completed":
				checkpoint.failedS = true
			case "complete-incomplete":
				checkpoint.completed = 0
			case "inventory-count":
				checkpoint.inventory.count = 1
			case "settlement-kind":
				record.objects[1].kind = "results"
			case "policy-shape":
				policy.raw = `{}`
			case "provider-shape":
				policy.raw = `{"provider":null}`
			case "missing-roots":
				policy.raw = `{"provider":{}}`
			case "failed-object-kind", "failed-attempt-missing":
				checkpoint.failedS, checkpoint.completed = true, 0
				if mode == "failed-attempt-missing" {
					record.objects[0].kind = "failure"
				}
			case "result-kind":
				record.objects[0].kind = "failure"
			case "invalid-result-pack":
				record.objects[0].raw = "!"
			}
			head, phase, expectedRetention, work := ledger.head, ledger.phase, retained, meter.actual
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			originalBytes := providerHistoryProvisionedOriginalBytes(t, o.store.config)
			whole, err := subject.settlement(input, &checkpoint, &record, retention)
			if err == nil || whole || meter.failed || ledger.head != head || ledger.phase != phase || ledger.failed || retained != expectedRetention ||
				subject.owner != nil || recovery.owner != nil || recovery.store != nil || recovery.active || recovery.used ||
				meter.actual[0] != 0 || meter.actual[1] != 0 || meter.actual[9] != 0 || meter.clocks != 0 ||
				o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				o.store.activeMeter.Load() != nil || !reflect.DeepEqual(originalBytes, providerHistoryProvisionedOriginalBytes(t, o.store.config)) {
				t.Fatal("incomplete private settlement reached originals, changed accounting or gained an owner", mode, err)
			}
			for index, before := range work {
				if meter.actual[index] < before {
					t.Fatal("private settlement observation refunded a preceding input", mode, index)
				}
			}
		})
	}
}
