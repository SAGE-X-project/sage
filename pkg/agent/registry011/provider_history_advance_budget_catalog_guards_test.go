package registry011

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProviderHistoryStageBudgetRejectsIncompletePrivateInitialization(t *testing.T) {
	for _, mode := range []string{"same", "invalid-initialization", "initialization-short", "observer-short", "clock-short"} {
		t.Run(mode, func(t *testing.T) {
			// Numeric DATA only: no registered owner, clock, store root,
			// derivation or authority is created or changed by this predicate.
			u := historyRecordUpper(1)
			work, ok := u.work()
			b := providerIntegrationBounds{}
			initial, initialOK := providerHistoryGroupInitializationForProfileUpper(b)
			if !ok || !initialOK {
				t.Fatal("finite private initialization precursor")
			}
			future := historyLedgerMaximum()
			for i := range work {
				future.work[i] = initial[i] + work[i]
			}
			future.clocks = u.clocks + 2
			o := &providerHistoryGroupOwner{parent: &providerHistoryProgramOwner{}, store: &providerCustodyStore{}, coordinator: &providerHistoryGroupLedger{}, meter: &providerWorkMeter{}}
			o.parent.bounds.stages, o.parent.bounds.future = 2, future
			o.store.config.bounds = b
			o.meter.upper, o.meter.maximumClocks = future.work, future.clocks
			if !o.admissionBudgetFitsLocked(u, work) {
				t.Fatal("exact private initialization and stage accounting")
			}
			switch mode {
			case "invalid-initialization":
				o.store.config.bounds.maximumMetadataPerCommit = ^uint64(0)
			case "initialization-short":
				o.parent.bounds.future.work[10]--
			case "observer-short":
				o.meter.upper[2] = work[2] - 1
			case "clock-short":
				o.parent.bounds.future.clocks = u.clocks
			}
			if got := o.admissionBudgetFitsLocked(u, work); got != (mode == "same") ||
				o.coordinator.charged != (providerHistoryGroupTotals{}) || o.meter.actual != (candidateWorkVector{}) || o.meter.clocks != 0 ||
				o.meter.failed || o.owner != nil || o.store.owner != nil || o.parent.group != nil {
				t.Fatal("incomplete private budget fit or gained an original execution role", mode)
			}
		})
	}
}

func TestProviderHistoryStageAdvanceMissingOriginalCannotSelectABase(t *testing.T) {
	var absent *providerHistoryGroupOwner
	for _, o := range []*providerHistoryGroupOwner{absent, {}} {
		if o.advanceStageCandidate(context.Background(), nil) == nil || o.admissionBaseShapeLocked() || o.successfulSettlementShapeLocked() {
			t.Fatal("missing original owner selected a stage base or advance")
		}
	}
}

func TestProviderHistoryStageAdvanceRejectsUnexpectedOwnedCatalogBeforeClock(t *testing.T) {
	for _, root := range []string{"custody", "evidence"} {
		t.Run(root, func(t *testing.T) {
			o := historyMultistageOwner(t, 2)
			out := historyMultistageSettle(t, o, 1)
			s, m, r := o.store, o.meter, o.parent.recovery
			if !o.successfulSettlementShapeLocked() || !o.parent.matchesLocked() || !o.parent.dispatchBounds.advanceValid() || r.confirmed.ack != out.ack {
				t.Fatal("genuine successful first stage and unchanged advance precursor")
			}
			pin, ack, confirmed, base := o.pin, o.last, r.confirmed, o.stageBefore
			work, clocks, held, retained := m.actual, s.clock.read, s.stored, o.retention
			window, origin, upper, maximum := o.windowStart, o.clockStart, m.upper, o.parent.bounds.future
			directory := s.config.evidenceDirectory
			if root == "custody" {
				directory = s.config.custodyDirectory
			}
			if err := os.WriteFile(filepath.Join(directory, "owned-unexpected-catalog-entry"), []byte("finite UNIT data"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := o.advanceStageCandidate(context.Background(), out.ack); err == nil || !o.failed || !m.failed || !m.closed || s.activeMeter.Load() != nil ||
				o.pin != pin || s.frontierFiles.pin != pin || o.last != ack || r.confirmed != confirmed || o.stageBefore != base ||
				s.stored != held || o.retention != retained || s.clock.read != clocks || m.actual[10] <= work[10] ||
				m.actual[0] != work[0] || m.actual[1] != work[1] || m.actual[9] != work[9] ||
				o.windowStart != window || o.clockStart != origin || m.upper != upper || o.parent.bounds.future != maximum {
				t.Fatal("unknown owned name advanced the original stage, clock, pin or allowance", root, err)
			}
		})
	}
}
