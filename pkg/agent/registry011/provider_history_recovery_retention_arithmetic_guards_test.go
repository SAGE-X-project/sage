package registry011

import (
	"math"
	"testing"
)

func TestProviderHistoryRecoveryRetentionRejectsIncompleteAndOverflowedPrivateRows(t *testing.T) {
	for _, mode := range []string{"anchor-sum", "setup-sum", "missing-highwater", "zero-highwater", "highwater-sum", "highwater-index"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			files, bytes := r.inputFiles, r.inputBytes
			m, _ := historyRecoveryMessageDataUnit(t, r)
			record := m.prefix.records[0]
			// Isolated arithmetic inputs, explicitly NOT a selected prefix.
			// Copy the scalar struct and replace both length arrays before
			// changing them; no original prefix or live budget is edited.
			original := m.prefix
			prefix := *original
			prefix.anchorBytes = 3
			prefix.setupOriginalBytes = []uint64{4, 4, 4}
			prefix.highwaterOriginalBytes = []uint64{2, 2, 2, 2}
			m.prefix, m.through, m.frontier.setupOrdinal = &prefix, 1, -1
			setup := providerSetupReference{ordinal: 0}
			if prefix.owner == &prefix || prefix.selectedLocked(r) || m.selectedLocked(r) {
				t.Fatal("private arithmetic DATA unexpectedly selected a recovery role")
			}
			if value, err := m.retention(record, setup, providerFrontierRetention{}); err != nil || value.stored == 0 || value.metadata == 0 {
				t.Fatal("positive finite arithmetic row with genuine encoded highwater", err)
			}
			switch mode {
			case "anchor-sum":
				prefix.anchorBytes = math.MaxUint64
			case "setup-sum":
				prefix.setupOriginalBytes[0] = math.MaxUint64
			case "missing-highwater", "zero-highwater", "highwater-sum":
				m.frontier.setupOrdinal, setup.ordinal = 0, 1
				switch mode {
				case "missing-highwater":
					prefix.highwaterOriginalBytes = prefix.highwaterOriginalBytes[:1]
				case "zero-highwater":
					prefix.highwaterOriginalBytes[1] = 0
				case "highwater-sum":
					// Doubling is still representable; adding the already
					// included four setup bytes must overflow and refuse.
					prefix.highwaterOriginalBytes[1] = math.MaxUint64 / 2
				}
			case "highwater-index":
				m.frontier.setupOrdinal, setup.ordinal, m.through = 1, 2, math.MaxUint64
			}
			if value, err := m.retention(record, setup, providerFrontierRetention{}); err == nil || value != (providerFrontierRetention{}) || r.meter.failed {
				t.Fatal("incomplete or overflowing DATA row returned partial accounting", mode, err)
			}
			if original != r.legacyPrefixes[0] || original.setupOriginalBytes[0] == math.MaxUint64 ||
				original.highwaterOriginalBytes[0] == 0 || prefix.selectedLocked(r) ||
				o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				r.inputFiles != files || r.inputBytes != bytes || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) ||
				!r.activeLocked(r.meter) || o.store.activeMeter.Load() != nil {
				t.Fatal("private arithmetic changed selected originals, execution or allowances")
			}
		})
	}
}
