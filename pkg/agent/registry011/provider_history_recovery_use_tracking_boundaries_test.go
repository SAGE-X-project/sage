package registry011

import (
	"reflect"
	"testing"
)

func TestProviderHistoryRecoveryOriginalLookupRequiresCompletePrivateUseTracking(t *testing.T) {
	o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
	historyRecoveryModelPrefix(t, r, 0, want)
	originals := providerHistoryProvisionedOriginalBytes(t, r.config)
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	m, _ := historyRecoveryMessageDataUnit(t, r)
	frame := m.prefix.source.frames[0]
	if len(frame.objects) == 0 || len(m.used[0]) != len(frame.objects) || m.complete || m.selectedLocked(r) {
		t.Fatal("genuine uninstalled private model with complete use-tracking slots")
	}
	object := frame.objects[0]
	if raw, err := m.object(object.kind, object.digest); err != nil || string(raw) != object.raw || len(raw) == 0 {
		t.Fatal("positive original lookup before private tracking DATA is removed", err)
	}
	// Remove only this uninstalled model's tracking DATA. Original frames,
	// selected prefix, observer, limits and real input counters stay intact.
	m.used[0] = nil
	work, files, bytes, bounds := r.meter.actual, r.inputFiles, r.inputBytes, r.bounds
	old, reads, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
	root, frontier, through := m.root, m.frontier.current, m.through
	if raw, err := m.object(object.kind, object.digest); err == nil || raw != nil || m.used[0] != nil ||
		m.root != root || m.frontier.current != frontier || m.through != through || m.complete || m.selectedLocked(r) ||
		r.meter.actual != work || r.meter.failed || !r.activeLocked(r.meter) || r.inputFiles != files || r.inputBytes != bytes || r.bounds != bounds ||
		r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) || o.meter.actual != old || o.store.clock.read != reads ||
		o.pin != pin || o.last != ack || o.store.stored != held || o.store.activeMeter.Load() != nil ||
		!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, r.config)) {
		t.Fatal("missing private tracking supplied original DATA, changed selected state or renewed execution", err)
	}
}
