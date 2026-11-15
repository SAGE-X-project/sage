package registry011

import (
	"math"
	"testing"
)

// These cases change only a freshly reconstructed private DATA model or a
// copied record. The selected prefix, stored originals and original resource
// limits remain unchanged; no model is installed as a recovered owner.
func TestProviderHistoryRecoveryObjectLookupRejectsIncompletePrivateSelections(t *testing.T) {
	for _, mode := range []string{"copied-model", "complete", "before-first", "past-last", "missing-use-map"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			files, bytes := r.inputFiles, r.inputBytes
			m, _ := historyRecoveryMessageDataUnit(t, r)
			var selected providerIntegrationObject
			for _, object := range m.prefix.source.frames[0].objects {
				if object.kind == "enrollment" {
					selected = object
					break
				}
			}
			if raw, err := m.object(selected.kind, selected.digest); err != nil || len(raw) == 0 {
				t.Fatal("genuine selected enrollment original is required", err)
			}
			input := m
			switch mode {
			case "copied-model":
				copy := *m
				input = &copy
			case "complete":
				m.complete = true
			case "before-first":
				m.through = 0
			case "past-last":
				m.through = 5
			case "missing-use-map":
				m.used[0] = nil
			}
			if raw, err := input.object(selected.kind, selected.digest); err == nil || raw != nil || r.meter.failed {
				t.Fatal("incomplete private selection supplied retained bytes or poisoned its original observer", mode, err)
			}
			if o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				r.inputFiles != files || r.inputBytes != bytes || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) ||
				!r.activeLocked(r.meter) || o.store.activeMeter.Load() != nil {
				t.Fatal("private selection checks performed I/O, installed a model or revived execution")
			}
		})
	}
}

func TestProviderHistoryRecoverySetupRequiresItsReferenceAndClockOrder(t *testing.T) {
	for _, mode := range []string{"missing-reference", "reversed-clock", "past-setup"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			files, bytes := r.inputFiles, r.inputBytes
			m, _ := historyRecoveryMessageDataUnit(t, r)
			record := m.prefix.records[3]
			setup, err := m.setup(record)
			if err != nil {
				t.Fatal("genuine original setup selection", err)
			}
			switch mode {
			case "missing-reference":
				record.before.setup = admissionTestDigest("f")
			case "reversed-clock":
				record.frontier.clock.MonotonicMS = setup.clock.MonotonicMS - 1
			case "past-setup":
				m.frontier.setupOrdinal = setup.ordinal + 1
			}
			if value, err := m.setup(record); err == nil || value != (providerSetupReference{}) || r.meter.failed {
				t.Fatal("unselected or reversed private setup returned a reference", mode, err)
			}
			if o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				r.inputFiles != files || r.inputBytes != bytes || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) ||
				!r.activeLocked(r.meter) || o.store.activeMeter.Load() != nil {
				t.Fatal("private setup checks altered the original prefix or execution")
			}
		})
	}
}

func TestProviderHistoryRecoveryRetentionRefusesOverflowWithoutPartialAccounting(t *testing.T) {
	for _, mode := range []string{"metadata-overflow", "stored-overflow", "missing-highwater"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			files, bytes := r.inputFiles, r.inputBytes
			m, _ := historyRecoveryMessageDataUnit(t, r)
			record := m.prefix.records[3]
			setup, err := m.setup(record)
			if err != nil {
				t.Fatal("genuine original setup", err)
			}
			if value, err := m.retention(record, setup, providerFrontierRetention{}); err != nil || value.stored == 0 || value.metadata == 0 {
				t.Fatal("positive retained highwater accounting precursor", err)
			}
			var cost providerFrontierRetention
			switch mode {
			case "metadata-overflow":
				cost.metadata = math.MaxUint64
			case "stored-overflow":
				cost.stored = math.MaxUint64
			case "missing-highwater":
				record.after = providerFrontierHighwater{}
			}
			if value, err := m.retention(record, setup, cost); err == nil || value != (providerFrontierRetention{}) || r.meter.failed {
				t.Fatal("invalid DATA accounting returned a partial retained charge", mode, err)
			}
			if o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				r.inputFiles != files || r.inputBytes != bytes || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) ||
				!r.activeLocked(r.meter) || o.store.activeMeter.Load() != nil {
				t.Fatal("private arithmetic changed the original allowance, files or execution")
			}
		})
	}
}

func TestProviderHistoryRecoveryApplyRejectsChangedRecordBindingsBeforePublication(t *testing.T) {
	for _, mode := range []string{"sequence", "kind", "operation"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			files, bytes := r.inputFiles, r.inputBytes
			m, _ := historyRecoveryMessageDataUnit(t, r)
			record := m.prefix.records[3]
			before, root := m.frontier.current, m.root
			switch mode {
			case "sequence":
				record.after.sequence = 0
			case "kind":
				record.kind = "enroll"
			case "operation":
				record.operation = admissionTestDigest("f")
			}
			if err := m.apply(record); err == nil || m.frontier.current != before || m.root != root || r.meter.failed {
				t.Fatal("changed private record advanced a frontier or original observer", mode, err)
			}
			if o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				r.inputFiles != files || r.inputBytes != bytes || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) ||
				!r.activeLocked(r.meter) || o.store.activeMeter.Load() != nil {
				t.Fatal("changed copied DATA altered original files or published a model")
			}
		})
	}
}

// Hash-shaped expected metadata cannot replace byte-for-byte equality with
// the original selected object. This private lookup never opens a new file.
func TestProviderHistoryRecoveryExactObjectRejectsDifferentExpectedBytes(t *testing.T) {
	o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
	historyRecoveryModelPrefix(t, r, 0, want)
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	m, _ := historyRecoveryMessageDataUnit(t, r)
	var original providerIntegrationObject
	for _, object := range m.prefix.source.frames[0].objects {
		if object.kind == "enrollment" {
			original = object
			break
		}
	}
	if original.raw == "" || m.exactObject(original) != nil {
		t.Fatal("genuine original bytes must satisfy the private equality check")
	}
	old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
	files, bytes, work, frontier := r.inputFiles, r.inputBytes, r.meter.actual, m.frontier.current
	different := original
	different.raw += "\n"
	if m.exactObject(different) == nil || r.meter.failed || r.meter.actual[10] <= work[10] ||
		m.frontier.current != frontier || r.inputFiles != files || r.inputBytes != bytes ||
		o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
		r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) || !r.activeLocked(r.meter) || o.store.activeMeter.Load() != nil {
		t.Fatal("different expected bytes escaped the original comparison or changed execution")
	}
}
