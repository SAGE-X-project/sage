package registry011

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProviderHistoryRecoveryGroupRejectsInconsistentInitialAndWholeChainData(t *testing.T) {
	for _, mode := range []string{"intact", "different-frontier", "different-exchange", "last-input-slot", "different-confirmed-model", "changed-head"} {
		t.Run(mode, func(t *testing.T) {
			o, _ := historyRecoveryGroupFixture(t, 1)
			r, refs := historyRecoveryGroupStart(t, o)
			historyRecoveryGroupInitial(t, r, 0, refs)
			s := o.store
			s.mu.Lock()
			locked := true
			defer func() {
				if locked {
					s.mu.Unlock()
				}
			}()
			initial := r.initialModels[0]
			if !initial.selectedLocked(r) || r.groupModels[0] != nil || !r.inputReadyLocked(context.Background()) {
				t.Fatal("complete original initial model and unused group pass required")
			}
			switch mode {
			case "different-frontier":
				if initial.frontier.current.budget == admissionTestDigest("f") {
					t.Fatal("different private frontier DATA required")
				}
				initial.frontier.current.budget = admissionTestDigest("f")
			case "different-exchange":
				if initial.frontier.body.firstExchange == admissionTestDigest("f") {
					t.Fatal("different private exchange DATA required")
				}
				initial.frontier.body.firstExchange = admissionTestDigest("f")
			case "last-input-slot":
				// Actually consume the existing file/byte/work allowance through
				// ordinary selected reads. Never edit a live counter or limit.
				probe := &providerHistoryRecoveryGroupModel{recovery: r}
				if r.inputFiles >= r.bounds.maximumFiles-1 {
					t.Fatal("unused finite original input slots required")
				}
				for r.inputFiles < r.bounds.maximumFiles-1 {
					if raw, whole, err := probe.read(context.Background(), r.confirmed, providerHistoryRecoveryHead, 0); err != nil || !whole || len(raw) == 0 {
						t.Fatal("actual selected read must consume each original input slot", err)
					}
				}
			case "different-confirmed-model":
				if r.confirmed.model == admissionTestDigest("f") {
					t.Fatal("different private model comparison DATA required")
				}
				r.confirmed.model = admissionTestDigest("f")
			case "changed-head":
				path := filepath.Join(r.config.custodyDirectory, "HEAD")
				raw, err := os.ReadFile(path)
				if err != nil || len(raw) == 0 {
					t.Fatal("ordinary owned HEAD required", err)
				}
				raw[0] = '!'
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal("finite owned HEAD DATA fault failed", err)
				}
			}
			if !initial.selectedLocked(r) || !r.inputReadyLocked(context.Background()) || r.groupModels[0] != nil {
				t.Fatal("private DATA fault must leave the original entry selection intact", mode)
			}
			originals := providerHistoryProvisionedOriginalBytes(t, s.config)
			files, bytes, work, bounds := r.inputFiles, r.inputBytes, r.meter.actual, r.bounds
			old, clocks, held, pin, ack, confirmed := o.meter.actual, s.clock.read, s.stored, o.pin, o.last, r.confirmed
			locked = false
			s.mu.Unlock()
			out, err := r.replayGroupModelCandidate(context.Background(), 0)
			if mode == "intact" {
				if err != nil || out == nil || !out.selectedLocked(r) || !r.active || r.closed || r.failed || r.groupModels[0] != out {
					t.Fatal("ordinary independent complete group replay required", err)
				}
			} else if err == nil || out != nil || r.active || !r.closed || !r.failed || !r.meter.failed || !r.meter.closed || r.groupModels[0] != nil {
				t.Fatal("inconsistent original replay published a model or retained its failed inspection role", mode, err)
			}
			wantFiles := files + 1
			if mode == "different-frontier" || mode == "different-exchange" {
				wantFiles = files
			}
			if mode != "intact" && mode != "different-confirmed-model" && r.inputFiles != wantFiles {
				t.Fatal("refusal reached an unexpected original input boundary", mode, r.inputFiles, wantFiles)
			}
			if mode == "different-confirmed-model" && r.inputFiles <= files+1 {
				t.Fatal("model comparison must reconstruct the actual acknowledged originals")
			}
			if r.inputBytes < bytes || r.meter.actual[10] <= work[10] || r.bounds != bounds || r.initialModels[0] != initial ||
				o.meter.actual != old || s.clock.read != clocks || s.stored != held || o.pin != pin || o.last != ack || r.confirmed != confirmed ||
				s.activeMeter.Load() != nil || !reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("failed private group replay changed originals, refunded input or revived execution", mode)
			}
		})
	}
}

func TestProviderHistoryRecoveryGroupReadRefusesAbsentContextBeforeOriginalInput(t *testing.T) {
	o, _ := historyRecoveryGroupFixture(t, 1)
	r, _ := historyRecoveryGroupStart(t, o)
	s := o.store
	s.mu.Lock()
	defer s.mu.Unlock()
	g := &providerHistoryRecoveryGroupModel{recovery: r}
	if !r.inputReadyLocked(context.Background()) || !r.activeLocked(r.meter) || r.meter.failed || r.meter.closed {
		t.Fatal("genuine ready original inspection before the absent-context refusal required")
	}
	files, bytes, work, reads, held := r.inputFiles, r.inputBytes, r.meter.actual, s.clock.read, s.stored
	var absentContext context.Context
	if raw, whole, err := g.read(absentContext, r.confirmed, providerHistoryRecoveryHead, 0); err == nil || whole || raw != nil ||
		r.inputFiles != files || r.inputBytes != bytes || r.meter.actual != work || s.clock.read != reads || s.stored != held ||
		r.active || !r.failed || !r.closed || !r.meter.failed || !r.meter.closed || g.owner != nil || g.complete || r.groupModels != ([2]*providerHistoryRecoveryGroupModel{}) {
		t.Fatal("missing context admitted a selected original or published private group DATA", err)
	}
}
