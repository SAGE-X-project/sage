package registry011

import (
	"context"
	"testing"
)

func TestProviderHistoryRecoveryInventoryMissingRolesCannotStartOrAdvanceInspection(t *testing.T) {
	var missingRecovery *providerHistoryRecoveryOwner
	for _, r := range []*providerHistoryRecoveryOwner{missingRecovery, {}} {
		if walk, err := r.newInventoryWalkCandidate(context.Background(), 0); err == nil || walk != nil {
			t.Fatal("missing original store started an inventory inspection")
		}
	}
	var missingWalk *providerHistoryRecoveryInventoryWalk
	for _, w := range []*providerHistoryRecoveryInventoryWalk{missingWalk, {}, {recovery: &providerHistoryRecoveryOwner{}}} {
		if record, err := w.nextInventoryCandidate(context.Background()); err == nil || record != nil {
			t.Fatal("missing original recovery role returned an inventory")
		}
		if object, err := w.nextObjectCandidate(context.Background()); err == nil || object != nil {
			t.Fatal("missing original recovery role returned an object")
		}
		if summary, err := w.finishCandidate(context.Background()); err == nil || summary != nil {
			t.Fatal("missing original recovery role published an inventory summary")
		}
	}
}

func TestProviderHistoryRecoveryInventoryInputPlanRequiresItsSelectedWalkAndInventory(t *testing.T) {
	o, r, _ := historyRecoveryInventoryFixture(t)
	w, err := r.newInventoryWalkCandidate(context.Background(), 0)
	if err != nil || w == nil {
		t.Fatal("genuine original inventory walk precursor", err)
	}
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	if disk, limit, ok := w.inputPlanLocked(r, false); !ok || disk != r.evidence || limit != 64<<10 || w.current != nil {
		t.Fatal("original selected walk did not identify its next inventory")
	}
	old, work, files, bytes, clocks := o.meter.actual, r.meter.actual, r.inputFiles, r.inputBytes, o.store.clock.read
	pin, ack, held := o.pin, o.last, o.store.stored
	if disk, limit, ok := w.inputPlanLocked(r, true); ok || disk != nil || limit != 0 {
		t.Fatal("object plan skipped its unread original inventory")
	}
	copy := *w
	copy.owner = &copy
	if disk, limit, ok := copy.inputPlanLocked(r, false); ok || disk != nil || limit != 0 {
		t.Fatal("copied walk selected the original input path")
	}
	if !w.selectedLocked(r) || !r.activeLocked(r.meter) || r.inventoryWalks[0] != w || w.current != nil || w.complete ||
		o.meter.actual != old || r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes || o.store.clock.read != clocks ||
		o.pin != pin || o.last != ack || o.store.stored != held || o.store.activeMeter.Load() != nil {
		t.Fatal("refused input plan changed the original role, performed I/O or renewed execution")
	}
}
