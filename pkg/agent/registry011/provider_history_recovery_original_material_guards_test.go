package registry011

import "testing"

// Each case uses a newly reconstructed, uninstalled private model. It neither
// changes the selected prefix nor renews the original inspection allowance.
func TestProviderHistoryRecoveryBodyRejectsRepeatedInitializationAndAlteredMaterials(t *testing.T) {
	for _, mode := range []string{"intact-materials", "enrollment-repeat", "admission-repeat", "scan-seed-repeat", "changed-anchor-original", "changed-admission-original", "missing-operation-original"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			files, bytes, bounds := r.inputFiles, r.inputBytes, r.bounds
			m, _ := historyRecoveryMessageDataUnit(t, r)
			body := m.frontier.body
			if len(m.prefix.setups) == 0 || m.prefix.records[0].kind != "enroll" || m.prefix.records[1].kind != "admit" ||
				m.prefix.records[0].root == nil || m.prefix.records[1].root == nil || body.anchor == nil || body.intent == nil ||
				body.ledger == nil || !body.ledger.valid() || body.ledger.current == nil || m.through != 4 {
				t.Fatal("genuine private enrollment, admission and consumption required")
			}
			record := m.prefix.records[3]
			if record.kind != "settle" {
				t.Fatal("original pending settlement required")
			}
			switch mode {
			case "changed-anchor-original":
				changed := *body.anchor
				changed.raw += " "
				body.anchor = &changed
			case "changed-admission-original":
				changed := *body.intent
				changed.raw += " "
				body.intent = &changed
			case "missing-operation-original":
				record.operation += "x"
			}
			anchor, intent, ledger, current := body.anchor, body.intent, body.ledger, body.ledger.current
			frontier, metadata, through := m.frontier.current, body.scanMetadata, m.through
			work := r.meter.actual
			var err error
			switch mode {
			case "enrollment-repeat":
				var out providerIntegrationObject
				original := m.prefix.records[0]
				out, err = m.applyBody(original, original.root, m.prefix.setups[0])
				if out != (providerIntegrationObject{}) {
					t.Fatal("repeated enrollment supplied partial object data")
				}
			case "admission-repeat":
				var out providerIntegrationObject
				original := m.prefix.records[1]
				out, err = m.applyBody(original, original.root, providerSetupReference{})
				if out != (providerIntegrationObject{}) {
					t.Fatal("repeated admission supplied partial object data")
				}
			case "scan-seed-repeat":
				var seed *providerLedgerInitialCharge
				seed, err = m.scanSeed()
				if seed != nil {
					t.Fatal("existing private ledger received another scan seed")
				}
			default:
				err = m.oldMaterials(record)
			}
			if (err == nil) != (mode == "intact-materials") || r.meter.failed || r.meter.actual[10] <= work[10] ||
				body.anchor != anchor || body.intent != intent || body.ledger != ledger || ledger.current != current ||
				m.frontier.current != frontier || body.scanMetadata != metadata || m.through != through ||
				r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) || !r.activeLocked(r.meter) ||
				r.inputFiles != files || r.inputBytes != bytes || r.bounds != bounds || o.meter.actual != old ||
				o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held || o.store.activeMeter.Load() != nil {
				t.Fatal("repeated initialization or changed private materials supplied authority, changed state or hid unobserved work", mode, err)
			}
		})
	}
}
