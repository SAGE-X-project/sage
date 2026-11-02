package registry011

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProviderHistoryRecoveryGroupAdmitsRemainingFilesBeforeCheckpointArrays(t *testing.T) {
	for _, subject := range []struct {
		name  string
		files uint64
	}{
		{name: "consumed", files: 17},
		{name: "settled-two", files: 29},
		{name: "failed-one-of-three", files: 28},
		{name: "pending-highwater-empty", files: 18},
	} {
		for _, short := range []bool{false, true} {
			name := subject.name + "-exact"
			if short {
				name = subject.name + "-one-short"
			}
			t.Run(name, func(t *testing.T) {
				count := 1
				if subject.name == "settled-two" {
					count = 2
				} else if subject.name == "failed-one-of-three" {
					count = 3
				}
				var o *providerHistoryGroupOwner
				if subject.name == "pending-highwater-empty" {
					o, _ = historyRecoveryUnacknowledgedS(t)
					p := o.parent.recovery.pending
					path := filepath.Join(o.store.config.custodyDirectory, providerCustodyName("highwater", p.pin.ordinal))
					if raw, err := os.ReadFile(path); err != nil || len(raw) == 0 || os.WriteFile(path, nil, 0600) != nil {
						t.Fatal("genuine owned pending highwater before partial-file scenario", err)
					}
				} else {
					var d *providerHistoryGroupDispatch
					o, d = historyRecoveryGroupFixture(t, count)
					switch subject.name {
					case "settled-two":
						historyRetainNormalPrefix(t, o, d, count)
						if out, err := o.retainSettlementCandidate(context.Background(), d); err != nil || out == nil {
							t.Fatal("genuine confirmed S", err)
						}
					case "failed-one-of-three":
						historyRetainNormalPrefix(t, o, d, 1)
						member := d.coordinator.ticket.table.members[1]
						conn := providerIntegrationOwnedTLS(t, member.request.body, `{}`)
						if out, err := o.retainMemberCandidateWithConnection(context.Background(), d, 1, conn); err == nil || out != nil || !o.failed ||
							o.parent.recovery.confirmed.frame.phase != "settle" || !o.parent.recovery.confirmed.failedS {
							t.Fatal("genuine failed S after one completed member", err)
						}
					}
				}
				r, want := historyRecoveryGroupStart(t, o)
				historyRecoveryGroupInitial(t, r, 0, want)
				bounds := r.bounds
				if bounds.maximumFiles != 1024 || r.groupModels[0] != nil || !r.initialModels[0].selectedLocked(r) {
					t.Fatal("original pre-H0 group UNIT row and complete initial semantic model required")
				}
				remaining := subject.files
				if short {
					remaining--
				}
				if r.inputFiles >= bounds.maximumFiles-remaining {
					t.Fatal("genuine precursor cannot prepare the selected remaining allowance")
				}
				// Spend the original allowance by actual closed HEAD/EOF reads.
				// Do not edit caps, counters, the original row or its selection.
				for r.inputFiles < bounds.maximumFiles-remaining {
					out, err := r.readCheckpointInputCandidate(context.Background(), r.confirmed, providerHistoryRecoveryHead, 0)
					if err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput || len(out.raw) == 0 {
						t.Fatal("actual cumulative input precursor", err)
					}
				}
				files, bytes, work := r.inputFiles, r.inputBytes, r.meter.actual
				old, reads, held, pin, ack, confirmed := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last, r.confirmed
				before := providerHistoryProvisionedOriginalBytes(t, o.store.config)
				out, err := r.replayGroupModelCandidate(context.Background(), 0)
				if short {
					if err == nil || out != nil || r.groupModels[0] != nil || r.active || !r.closed || !r.failed ||
						r.inputFiles != files+1 || r.inputBytes <= bytes || r.meter.actual[10] <= work[10] {
						t.Fatal("insufficient minimum allowance reached checkpoint reads or published a model", err)
					}
				} else {
					if err != nil || out == nil || !out.selectedLocked(r) || !r.active || r.closed || r.failed ||
						r.inputFiles != files+subject.files || r.inputFiles != bounds.maximumFiles || r.inputBytes <= bytes || r.meter.actual[10] <= work[10] ||
						out.pin != confirmed.pin || out.confirmedModel != confirmed.model {
						t.Fatal("exact remaining allowance refused an otherwise-valid whole semantic prefix", err)
					}
					if subject.name == "pending-highwater-empty" && (out.physical != confirmed || out.tailKind != providerHistoryRecoveryHighwater || out.tailState != providerHistoryRecoveryPartialInput) {
						t.Fatal("minimum pending probe changed the confirmed prefix or partial tail")
					}
				}
				if r.bounds != bounds || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.pin != pin || o.last != ack ||
					r.confirmed != confirmed || o.store.activeMeter.Load() != nil || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, o.store.config)) {
					t.Fatal("minimum file admission changed the original allowance, authority, clock, hold, ACK or retained files")
				}
			})
		}
	}
}
