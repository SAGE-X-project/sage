package registry011

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestProviderHistoryRecoveryBodyRejectsIncompleteBudgetTransitions(t *testing.T) {
	o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
	historyRecoveryModelPrefix(t, r, 0, want)
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
	files, bytes := r.inputFiles, r.inputBytes
	for _, mode := range []string{"same", "missing-anchor", "missing-budget", "missing-ledger", "missing-ticket", "changed-ticket", "terminal-ticket", "unknown-transition", "wrong-budget"} {
		t.Run(mode, func(t *testing.T) {
			// Each private model is independently reconstructed from the same
			// immutable original prefix. No selected model or limit changes.
			f, err := newProviderFrontierReplayObserved(r.config, r.meter)
			if err != nil {
				t.Fatal("fresh private model precursor", err)
			}
			f.meter, f.body.meter, f.indexes.meter = r.meter, r.meter, r.meter
			m := &providerHistoryRecoveryInitialModel{recovery: r, prefix: r.legacyPrefixes[0], frontier: f}
			m.owner = m
			for i, frame := range m.prefix.source.frames {
				m.used[i] = make([]bool, len(frame.objects))
			}
			for i := range 2 {
				m.through = uint64(i + 1)
				if m.apply(m.prefix.records[i]) != nil {
					t.Fatal("ordinary enrollment/admission prefix", i)
				}
			}
			record := m.prefix.records[2]
			if record.kind != "consume" || record.root == nil || f.body.anchor == nil || f.body.ledger == nil {
				t.Fatal("original consumption precursor")
			}
			root := *record.root
			m.through = 3
			key := r.initialDescriptor.ticket
			state, exists := f.body.ledger.tickets[key]
			if !exists || state.ticket == nil || state.consumption != nil || state.terminal != "" {
				t.Fatal("fresh unconsumed private ticket")
			}
			switch mode {
			case "missing-anchor":
				f.body.anchor = nil
			case "missing-budget":
				root.budget = admissionTestDigest("f")
			case "missing-ledger":
				f.body.ledger = nil
			case "missing-ticket":
				delete(f.body.ledger.tickets, key)
			case "changed-ticket":
				ticket := *state.ticket
				ticket.digest = admissionTestDigest("f")
				state.ticket = &ticket
				f.body.ledger.tickets[key] = state
			case "terminal-ticket":
				state.terminal = admissionTestDigest("f")
				f.body.ledger.tickets[key] = state
			case "unknown-transition":
				record.kind = "unsupported-unit-transition"
			case "wrong-budget":
				root.budget = m.prefix.records[1].root.budget
				if root.budget == record.root.budget {
					t.Fatal("distinct original admission and consumption budgets")
				}
			}
			out, err := m.applyBody(record, &root, providerSetupReference{})
			if (err == nil) != (mode == "same") || mode == "same" && out.kind != "consumption" || mode != "same" && out != (providerIntegrationObject{}) {
				t.Fatal("inconsistent private budget transition accepted", mode, err)
			}
			if r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) || !r.activeLocked(r.meter) || r.inputFiles != files || r.inputBytes != bytes ||
				o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held || o.store.activeMeter.Load() != nil {
				t.Fatal("private budget replay published a model, performed I/O or revived original execution", mode)
			}
		})
	}
}

func TestProviderHistoryRecoveryCapacityIncludesAllHeldResourceCoordinates(t *testing.T) {
	s := providerHistoryParentScalarFixture(t)
	for _, mode := range []string{"same", "guardian", "missing-history", "invalid-history", "pre-initial-excess", "sum-overflow", "malformed-policy", "missing-limits", "invalid-bootstrap", "live-low", "retained-low", "work-low"} {
		t.Run(mode, func(t *testing.T) {
			// The existing scalar-only fixture is never opened with this
			// fabricated derivation. These copies cannot authorize a Source.
			config := s.config
			h := *config.bounds.history
			witness := *h.derivation
			h.derivation = &witness
			witness.owner, witness.snapshot = &witness, h
			config.bounds.history = &h
			held, ok := providerHistoryWholeHold(config.bounds)
			if !ok || held.retained == 0 || held.work[10] == 0 || !h.valid(config.bounds) {
				t.Fatal("closed scalar capacity precursor")
			}
			work := candidateLedgerWorkObject(held.work)
			work["deadline_ms"] = "300000"
			limits := map[string]any{"bootstrap": work, "live_disk_bytes": strconv.FormatUint(held.retained, 10), "retained_bytes": strconv.FormatUint(held.retained, 10)}
			policy := map[string]any{"limits": limits}
			preI := providerScanUpper{}
			switch mode {
			case "guardian":
				config.bounds.guardian = &providerGuardianBounds{}
			case "missing-history":
				config.bounds.history = nil
			case "invalid-history":
				h.program = "invalid"
			case "pre-initial-excess":
				preI.retained = config.bounds.scan.maximumPreIRetainedBytes + 1
			case "sum-overflow":
				h.startup.work[10] = ^uint64(0)
				witness.snapshot = h
				if !h.valid(config.bounds) {
					t.Fatal("overflow must reach the whole-hold sum after scalar shape")
				}
			case "missing-limits":
				delete(policy, "limits")
			case "invalid-bootstrap":
				limits["bootstrap"] = nil
			case "live-low":
				limits["live_disk_bytes"] = strconv.FormatUint(held.retained-1, 10)
			case "retained-low":
				limits["retained_bytes"] = strconv.FormatUint(held.retained-1, 10)
			case "work-low":
				work["work_units"] = strconv.FormatUint(held.work[10]-1, 10)
			}
			raw, err := json.Marshal(policy)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "malformed-policy" {
				raw = []byte("{")
			}
			r := &providerHistoryRecoveryOwner{config: config, meter: providerReplayUnitMeter()}
			m := &providerHistoryRecoveryInitialModel{recovery: r, frontier: &providerFrontierReplay{body: &providerCustodyReplay{policy: &ProviderPolicyCandidate{raw: string(raw)}}}}
			if got := m.initialCapacity(preI); got != (mode == "same") || r.active || r.used || r.store != nil || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) {
				t.Fatal("insufficient whole-hold scalar capacity accepted or created a retaining role", mode, got)
			}
		})
	}
}
