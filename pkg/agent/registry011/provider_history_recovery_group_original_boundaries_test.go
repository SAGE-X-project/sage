package registry011

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProviderHistoryRecoveryGroupReadCannotTreatConfirmedOriginalsAsOptionalTailData(t *testing.T) {
	for _, mode := range []string{"whole", "missing", "empty"} {
		t.Run(mode, func(t *testing.T) {
			o, _ := historyRecoveryGroupFixture(t, 1)
			r, _ := historyRecoveryGroupStart(t, o)
			path := filepath.Join(r.config.custodyDirectory, "HEAD")
			switch mode {
			case "missing":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "empty":
				if err := os.WriteFile(path, nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			// Only the exclusive UNIT file changes. This uninstalled model
			// cannot turn a missing confirmed original into a pending slot.
			originals := providerHistoryProvisionedOriginalBytes(t, r.config)
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			g := &providerHistoryRecoveryGroupModel{recovery: r}
			if r.pending != nil || r.confirmed == nil || r.confirmed == r.initial || !r.inputReadyLocked(context.Background()) {
				t.Fatal("genuine confirmed C and unchanged ready inspection lane")
			}
			work, files, bytes, bounds := r.meter.actual, r.inputFiles, r.inputBytes, r.bounds
			old, reads, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			raw, whole, err := g.read(context.Background(), r.confirmed, providerHistoryRecoveryHead, 0)
			wantBytes := bytes
			if mode == "whole" {
				wantBytes += uint64(len(r.confirmed.head))
				if err != nil || !whole || string(raw) != r.confirmed.head {
					t.Fatal("ordinary confirmed HEAD must remain readable", err)
				}
			} else if err == nil || whole || raw != nil {
				t.Fatal("incomplete confirmed original was treated as an optional pending tail", mode, err)
			}
			if r.inputFiles != files+1 || r.inputBytes != wantBytes || r.meter.actual[10] <= work[10] ||
				!r.activeLocked(r.meter) || r.meter.failed || r.bounds != bounds || g.owner != nil || g.complete ||
				g.tailState != 0 || g.tailKind != 0 || g.tailIndex != 0 || r.groupModels != ([2]*providerHistoryRecoveryGroupModel{}) ||
				o.meter.actual != old || o.store.clock.read != reads || o.pin != pin || o.last != ack || o.store.stored != held ||
				o.store.activeMeter.Load() != nil || !reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, r.config)) {
				t.Fatal("partial DATA read renewed execution, lost actual charges or published a tail/model", mode)
			}
		})
	}
}

func TestProviderHistoryRecoveryGroupReplayRefusesMissingConfirmedHeadWithoutPublishingAModel(t *testing.T) {
	var absent *providerHistoryRecoveryOwner
	if out, err := absent.replayGroupModelCandidate(context.Background(), 0); err == nil || out != nil {
		t.Fatal("absent original owner reconstructed a group model")
	}
	o, _ := historyRecoveryGroupFixture(t, 1)
	r, want := historyRecoveryGroupStart(t, o)
	historyRecoveryGroupInitial(t, r, 0, want)
	if err := os.Remove(filepath.Join(r.config.custodyDirectory, "HEAD")); err != nil {
		t.Fatal(err)
	}
	originals := providerHistoryProvisionedOriginalBytes(t, r.config)
	if !r.initialModels[0].selectedLocked(r) || !r.inputReadyLocked(context.Background()) || r.groupModels[0] != nil {
		t.Fatal("genuine complete initial model before missing confirmed HEAD")
	}
	initial, files, bytes, bounds := r.initialModels, r.inputFiles, r.inputBytes, r.bounds
	old, reads, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
	if out, err := r.replayGroupModelCandidate(context.Background(), 0); err == nil || out != nil ||
		r.active || !r.closed || !r.failed || !r.meter.closed || !r.meter.failed ||
		r.inputFiles != files+1 || r.inputBytes != bytes || r.bounds != bounds || r.initialModels != initial ||
		r.groupModels != ([2]*providerHistoryRecoveryGroupModel{}) || o.meter.actual != old || o.store.clock.read != reads ||
		o.pin != pin || o.last != ack || o.store.stored != held || o.store.activeMeter.Load() != nil ||
		!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, r.config)) {
		t.Fatal("public replay acknowledged missing confirmed HEAD or changed the original execution", err)
	}
}
