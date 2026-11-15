package registry011

import "testing"

func TestProviderHistoryRecoveryFrontierRejectsInconsistentPrivateDerivation(t *testing.T) {
	for _, mode := range []string{"policy-shape", "duration-zero", "deadline-overflow", "previous-frontier", "intent-deadline", "checkpoint"} {
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
				t.Fatal("genuine selected setup before private frontier derivation", err)
			}
			// Complete only this fresh DATA model's body/delta, exactly as
			// original replay would. No frontier is installed or published.
			entry, err := m.applyBody(record, record.root, setup)
			if err != nil {
				t.Fatal("genuine private settlement body", err)
			}
			delta, err := m.frontier.indexes.prepare(record.kind, record.operation, m.through, entry, record.root, m.frontier.body.ledger)
			if err != nil {
				t.Fatal("genuine private index delta", err)
			}
			commit := []byte(record.commit)
			if value, err := m.deriveFrontier(record, record.root, delta, commit, setup); err != nil ||
				value.sequence != 4 || value.deadline == 0 || value.deadline != m.frontier.body.intent.deadline {
				t.Fatal("positive derivation from genuine original DATA is required", err)
			}
			root := m.root
			switch mode {
			case "policy-shape", "duration-zero", "deadline-overflow":
				// Copy before changing raw DATA; r.config.policy and the
				// originally selected policy remain unchanged.
				policy := *m.frontier.body.policy
				switch mode {
				case "policy-shape":
					policy.raw = `{}`
				case "duration-zero":
					policy.raw = `{"limits":{"bootstrap":{"deadline_ms":"0"}}}`
				case "deadline-overflow":
					policy.raw = `{"limits":{"bootstrap":{"deadline_ms":"18446744073709551615"}}}`
				}
				m.frontier.body.policy = &policy
			case "previous-frontier":
				m.frontier.current = providerFrontierRecord{}
			case "intent-deadline":
				intent := *m.frontier.body.intent
				intent.deadline++
				m.frontier.body.intent = &intent
			case "checkpoint":
				record.frontier.checkpoint += "x"
			}
			if value, err := m.deriveFrontier(record, record.root, delta, commit, setup); err == nil ||
				value != (providerFrontierRecord{}) || m.root != root || r.meter.failed {
				t.Fatal("inconsistent private frontier returned a partial record or poisoned the original observer", mode, err)
			}
			if o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				r.inputFiles != files || r.inputBytes != bytes || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) ||
				!r.activeLocked(r.meter) || o.store.activeMeter.Load() != nil {
				t.Fatal("private frontier derivation changed original files, execution or allowances")
			}
		})
	}
}
