package registry011

import (
	"context"
	"testing"
)

func TestProviderHistoryRecoveryInitialModelCancellationClosesOnlyItsInspectionLane(t *testing.T) {
	var absent *providerHistoryRecoveryOwner
	if model, err := absent.replayInitialModelCandidate(context.Background(), 0); err == nil || model != nil {
		t.Fatal("missing recovery owner reconstructed an initial model")
	}
	for _, mode := range []string{"nil-context", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			if !r.legacyPrefixes[0].selectedLocked(r) || r.initialModels[0] != nil {
				t.Fatal("genuine selected original prefix before cancellation")
			}
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			work, files, bytes, bounds := r.meter.actual, r.inputFiles, r.inputBytes, r.bounds
			var ctx context.Context
			if mode == "cancelled" {
				input, cancel := context.WithCancel(context.Background())
				cancel()
				ctx = input
			}
			model, err := r.replayInitialModelCandidate(ctx, 0)
			if err == nil || model != nil || r.active || !r.closed || !r.failed || !r.meter.closed || !r.meter.failed ||
				r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes || r.bounds != bounds ||
				r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) || o.store.activeMeter.Load() != nil ||
				o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held {
				t.Fatal("cancelled initial replay observed more inputs, revived execution or published a model", mode, err)
			}
		})
	}
}

func TestProviderHistoryRecoveryScanStateRejectsAnIndexOutsideItsOriginalPair(t *testing.T) {
	o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
	historyRecoveryModelPrefix(t, r, 0, want)
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
	files, bytes := r.inputFiles, r.inputBytes
	f, err := newProviderFrontierReplayObserved(r.config, r.meter)
	if err != nil {
		t.Fatal("fresh private frontier before original enrollment", err)
	}
	f.meter, f.body.meter, f.indexes.meter = r.meter, r.meter, r.meter
	m := &providerHistoryRecoveryInitialModel{recovery: r, prefix: r.legacyPrefixes[0], frontier: f, through: 1}
	m.owner = m
	if !m.scanState(0) {
		t.Fatal("genuine original setup scan must match the fresh initial model")
	}
	if m.scanState(2) || r.meter.failed || o.meter.actual != old || o.store.clock.read != clocks ||
		o.pin != pin || o.last != ack || o.store.stored != held || r.inputFiles != files || r.inputBytes != bytes ||
		r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) || !r.activeLocked(r.meter) || o.store.activeMeter.Load() != nil {
		t.Fatal("out-of-pair scan index changed selected state or supplied a snapshot")
	}
}

func TestProviderHistoryRecoveryInitialReplayRequiresCompleteFreshReadCountEvidence(t *testing.T) {
	o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
	historyRecoveryModelPrefix(t, r, 0, want)
	ctx := context.Background()
	bounds := r.bounds
	if bounds.maximumFiles != 256 || r.inputFiles >= bounds.maximumFiles-1 || r.initialModels[0] != nil {
		t.Fatal("original finite legacy UNIT file allowance required")
	}
	old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
	// Leave one original file slot through actual closed HEAD/EOF reads.
	// No selected counter, bound or meter is rewritten or renewed.
	for r.inputFiles < bounds.maximumFiles-1 {
		input, err := r.readCheckpointInputCandidate(ctx, r.confirmed, providerHistoryRecoveryHead, 0)
		if err != nil || input == nil || input.state != providerHistoryRecoveryWholeInput || len(input.raw) == 0 {
			t.Fatal("actual bounded input precursor", err)
		}
	}
	r.store.mu.Lock()
	ready := r.inputReadyLocked(ctx)
	r.store.mu.Unlock()
	if !ready || r.inputFiles != bounds.maximumFiles-1 || !r.legacyPrefixes[0].selectedLocked(r) {
		t.Fatal("one remaining file slot must preserve the otherwise-ready original lane")
	}
	work, files, bytes := r.meter.actual, r.inputFiles, r.inputBytes
	// A new file observation proves entry into the final read-count check.
	// Refusal here does not identify a particular subsequent capacity branch.
	model, err := r.replayInitialModelCandidate(ctx, 0)
	if err == nil || model != nil || r.active || !r.closed || !r.failed || !r.meter.closed || !r.meter.failed ||
		r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) || r.inputFiles != files+1 || r.inputFiles != bounds.maximumFiles || r.inputBytes != bytes ||
		r.meter.actual[10] <= work[10] || r.meter.actual[2] <= work[2] || r.bounds != bounds ||
		o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held || o.store.activeMeter.Load() != nil {
		t.Fatal("initial model lacked complete fresh read-count evidence or never entered its fresh file reads", err)
	}
	after := r.meter.actual
	if again, err := r.replayInitialModelCandidate(ctx, 0); err == nil || again != nil ||
		r.meter.actual != after || r.inputFiles != files+1 || r.inputBytes != bytes {
		t.Fatal("failed read-count verification retried or restored its original allowance")
	}
}
