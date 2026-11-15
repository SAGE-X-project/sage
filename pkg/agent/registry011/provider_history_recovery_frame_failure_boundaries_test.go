package registry011

import (
	"context"
	"reflect"
	"testing"
)

func TestProviderHistoryRecoveryInventoryFramesCannotRenewAFailedInspectionLane(t *testing.T) {
	o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
	historyRecoveryLegacySource(t, r, 0, want)
	s := o.store
	originals := providerHistoryProvisionedOriginalBytes(t, r.config)
	s.mu.Lock()
	w := r.inventoryWalks[0]
	if !w.framesReadyLocked(r) || len(w.frames[0].objects) == 0 || r.legacyPrefixes[0] != nil || !r.activeLocked(r.meter) {
		s.mu.Unlock()
		t.Fatal("genuine complete original frames before legacy replay")
	}
	object := w.frames[0].objects[0]
	if got := w.frameObjectObserved(object.kind, object.digest, 1); string(got) != object.raw || len(got) == 0 || r.meter.failed {
		s.mu.Unlock()
		t.Fatal("positive bounded lookup from original frame DATA")
	}
	for _, mode := range []string{"zero-prefix", "past-original-prefix", "invalid-digest"} {
		through, digest := uint64(1), object.digest
		switch mode {
		case "zero-prefix":
			through = 0
		case "past-original-prefix":
			through = uint64(len(w.frames)) + 1
		case "invalid-digest":
			digest = "not-a-digest"
		}
		before := r.meter.actual
		if out := w.frameObjectObserved(object.kind, digest, through); out != nil || r.meter.actual != before || r.meter.failed {
			s.mu.Unlock()
			t.Fatal("out-of-prefix lookup observed an original or returned DATA", mode)
		}
	}
	work, upper, files, bytes := r.meter.actual, r.meter.upper, r.inputFiles, r.inputBytes
	old, reads, pin, ack, held := o.meter.actual, s.clock.read, o.pin, o.last, s.stored
	s.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if prefix, err := r.replayLegacyPrefixCandidate(ctx, 0); err == nil || prefix != nil ||
		r.active || !r.closed || !r.failed || !r.meter.closed || !r.meter.failed || r.legacyPrefixes[0] != nil {
		t.Fatal("actual cancelled replay did not close its original inspection lane", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Original immutable DATA remains selected and structurally complete after
	// the lane closes. Refusal must come from its failed original observer;
	// an earlier shape failure would not establish this boundary.
	if !w.framesReadyLocked(r) || r.inventoryWalks[0] != w {
		t.Fatal("unchanged complete frame DATA required after original cancellation")
	}
	if out := w.frameObjectObserved(object.kind, object.digest, 1); out != nil ||
		r.meter.actual != work || r.meter.upper != upper || r.inputFiles != files || r.inputBytes != bytes ||
		r.active || !r.closed || !r.failed || !r.meter.closed || !r.meter.failed || r.legacyPrefixes[0] != nil ||
		o.meter.actual != old || s.clock.read != reads || o.pin != pin || o.last != ack || s.stored != held ||
		s.activeMeter.Load() != nil || !reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, r.config)) {
		t.Fatal("failed inspection lent old frame DATA a fresh observer, prefix or execution role")
	}
}
