package registry011

import (
	"reflect"
	"strconv"
	"testing"
)

// This is only finite description DATA and a private arithmetic writer.
// No original store, clock, selected prefix or recovery owner is installed.
func historyRecoveryInitialNameUnitData(capacity int) (*providerHistoryRecoveryExpectedCatalog, *providerHistoryRecoveryLegacyPrefix) {
	w := historyRecoveryNameUnitWriter()
	w.names = make([]providerHistoryRecoveryExpectedName, 0, capacity)
	r := w.recovery
	r.parent.bounds.catalogEntries = uint64(capacity)
	r.parent.bounds.catalogNameBytes = 16 << 10
	r.initialDescriptor.scan.ordinal, r.initialDescriptor.main.ordinal, r.initialDescriptor.setups = 1, 1, 1
	p := &providerHistoryRecoveryLegacyPrefix{
		source: &providerHistoryRecoveryInventoryWalk{},
		scan: &providerHistoryRecoveryScanWalk{records: []providerHistoryRecoveryScanRecord{{
			chunk: "finite chunk description DATA", command: "finite command description DATA",
		}}},
	}
	inventories := [4]string{"a", "b", "c", "d"}
	objects := [4]string{"e", "f", "1", "2"}
	for index := range p.source.frames {
		p.source.frames[index] = &providerHistoryRecoveryInventoryFrame{
			inventory: &providerIntegrationInventory{digest: admissionTestDigest(inventories[index])},
			objects:   []providerIntegrationObject{{kind: "request", digest: admissionTestDigest(objects[index])}},
		}
	}
	return w, p
}

func TestProviderHistoryRecoveryInitialNamesKeepEveryFiniteCapacityPrefix(t *testing.T) {
	whole, prefix := historyRecoveryInitialNameUnitData(128)
	if !whole.addInitialNames(prefix) || len(whole.names) != 47 || whole.bytes == 0 ||
		whole.recovery.meter.failed || whole.recovery.meter.closed || whole.recovery.active || whole.recovery.store != nil ||
		prefix.selectedLocked(whole.recovery) || prefix.owner != nil || prefix.source.owner != nil || prefix.scan.owner != nil {
		t.Fatal("complete bounded initial name DATA must not select original recovery authority")
	}
	seen := make(map[providerHistoryRecoveryExpectedName]bool)
	for _, name := range whole.names {
		if name.name == "" || len(name.name) > 255 || name.optional || seen[name] {
			t.Fatal("initial names lost their required unique namespace", name)
		}
		seen[name] = true
	}
	for _, required := range []providerHistoryRecoveryExpectedName{
		{name: "HEAD"}, {evidence: true, name: "AUTHORITY.LOCK"},
		{name: "intent-00000000000000000001"}, {evidence: true, name: "pending-00000000000000000001"},
	} {
		if !seen[required] {
			t.Fatal("initial name closure omitted a required namespace", required)
		}
	}
	for capacity := range len(whole.names) {
		t.Run("capacity-"+strconv.Itoa(capacity), func(t *testing.T) {
			w, p := historyRecoveryInitialNameUnitData(capacity)
			upper := w.recovery.meter.upper
			if w.addInitialNames(p) || len(w.names) != capacity || !reflect.DeepEqual(w.names, whole.names[:capacity]) ||
				w.recovery.meter.upper != upper || w.recovery.meter.failed || w.recovery.meter.closed ||
				w.recovery.active || w.recovery.used || w.recovery.store != nil || w.recovery.catalogResult != nil ||
				p.owner != nil || p.source.owner != nil || p.scan.owner != nil {
				t.Fatal("initial name shortage skipped a prefix, renewed work or selected a recovery role", capacity)
			}
			var expectedBytes uint64
			for _, name := range whole.names[:capacity] {
				expectedBytes += uint64(len(name.name))
			}
			if w.bytes != expectedBytes {
				t.Fatal("refused initial name changed previously retained byte accounting", capacity)
			}
		})
	}
}
