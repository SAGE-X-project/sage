package registry011

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestProviderHistoryRecoveryInitialDescriptorKeepsIndependentSelectedValues(t *testing.T) {
	o, _ := historyMemberDispatchFixture(t, 1)
	r, first := o.parent.recovery, o.parent.firstAdmission
	n := r.initialDescriptor
	if n == nil || !r.initialDescriptorShapeLocked(o) || n.checkpoint != r.initial || n.main != o.start.main || n.scan != first.scanPin ||
		n.attempt != first.attempt || n.scanMain != first.mainPin || n.scanMainRaw != first.mainRaw || n.targetRaw != first.targetRaw ||
		n.lastChunk != first.lastChunk || n.records != first.records || n.setups != uint64(len(o.store.setups)) || n.objectCount != uint64(len(o.store.objects)) {
		t.Fatal("original initial bridge/scan DATA was not selected by the initializer")
	}
	sealed := *n
	mainRaw, targetRaw, lastChunk, records, scan := first.mainRaw, first.targetRaw, first.lastChunk, first.records, first.scanPin
	first.mainRaw, first.targetRaw, first.lastChunk, first.records = "inert cache DATA", "inert target DATA", "inert chunk DATA", 0
	first.scanPin = providerScanHighwater{}
	unchanged := *n == sealed && r.initialDescriptorShapeLocked(o) && r.checkpointSelectionLocked(o)
	first.mainRaw, first.targetRaw, first.lastChunk, first.records, first.scanPin = mainRaw, targetRaw, lastChunk, records, scan
	if !unchanged || json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryInitialDescriptor{}) == nil {
		t.Fatal("immutable scan descriptor followed producer cache DATA or accepted JSON provenance")
	}
}

func TestProviderHistoryRecoveryInitialDescriptorRejectsDetachedSelectionBeforeObservation(t *testing.T) {
	o, _ := historyMemberDispatchFixture(t, 1)
	r := o.parent.recovery
	o.store.mu.Lock()
	o.failLocked()
	o.store.mu.Unlock()
	n := r.initialDescriptor
	if n == nil || !r.initialDescriptorShapeLocked(o) || !r.checkpointSelectionLocked(o) {
		t.Fatal("genuine terminal original descriptor")
	}
	for _, mode := range []string{"nil", "copy", "checkpoint-copy", "start-copy", "first-copy", "records", "setups", "exchange"} {
		t.Run(mode, func(t *testing.T) {
			selected, checkpoint, saved := r.initialDescriptor, r.initial.descriptor, *n
			actual, old, reads, held := r.meter.actual, o.meter.actual, o.store.clock.read, o.store.stored
			switch mode {
			case "nil":
				r.initialDescriptor = nil
			case "copy":
				copy := *n
				copy.owner = &copy
				r.initialDescriptor = &copy
			case "checkpoint-copy":
				copy := *n
				copy.owner = &copy
				r.initial.descriptor = &copy
			case "start-copy":
				copy := *n.start
				copy.owner = &copy
				n.start = &copy
			case "first-copy":
				copy := *n.first
				n.first = &copy
			case "records":
				n.records = 0
			case "setups":
				n.setups = 0
			case "exchange":
				n.exchange = "inert invalid selected hash"
			}
			err := r.beginInspectionCandidate(context.Background(), o)
			r.initialDescriptor, r.initial.descriptor, *n = selected, checkpoint, saved
			if err == nil || r.used || r.active || r.meter.actual != actual || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held ||
				!r.initialDescriptorShapeLocked(o) || !r.checkpointSelectionLocked(o) {
				t.Fatal("detached initial descriptor was read, adopted or renewed the original", mode)
			}
		})
	}
	if err := r.beginInspectionCandidate(context.Background(), o); err != nil || !r.used || !r.active {
		t.Fatal("otherwise-valid original descriptor refused read-only preparation", err)
	}
}

func TestProviderHistoryRecoveryInitialDescriptorRefusesUnsealedScanContext(t *testing.T) {
	for _, mode := range []string{"main", "target", "last-chunk", "records"} {
		t.Run(mode, func(t *testing.T) {
			s := providerHistoryGroupStartFixture(t, true)
			first := s.history.firstAdmission
			// Inert unit cache faults only. No retained original or network
			// message is replaced, and this does not recover a failed owner.
			switch mode {
			case "main":
				first.mainRaw = ""
			case "target":
				first.targetRaw = ""
			case "last-chunk":
				first.lastChunk = ""
			case "records":
				first.records = 0
			}
			reads, held, before := s.clock.read, s.stored, providerHistoryProvisionedOriginalBytes(t, s.config)
			out, err := newProviderHistoryGroupOwnerCandidate(s)
			g, r := s.history.group, s.history.recovery
			if err == nil || out != nil || g == nil || !g.failed || !g.meter.failed || !g.meter.closed || r.initial != nil || r.confirmed != nil || r.initialDescriptor != nil ||
				s.clock.read != reads+1 || g.meter.clocks != 1 || s.activeMeter.Load() != nil || s.stored != held ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("unsealed scan context reached initializer end/publication or changed originals", mode, err)
			}
		})
	}
}
