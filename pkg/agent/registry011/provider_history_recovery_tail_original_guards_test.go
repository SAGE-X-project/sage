package registry011

import (
	"context"
	"testing"
)

func TestProviderHistoryRecoveryTailRequiresWholeBoundedOriginalInputs(t *testing.T) {
	o, r := historyRecoveryPendingInputFixture(t)
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
	for _, mode := range []string{"full-entry-array", "required-missing", "object-too-long"} {
		t.Run(mode, func(t *testing.T) {
			m := providerHistoryRecoveryTailPayloads{recovery: r}
			p := providerHistoryRecoveryTailPlan{checkpoint: r.pending, kind: providerHistoryRecoveryInventory,
				bytes: uint64(len(r.pending.inventory.raw)), digest: r.pending.inventory.digest}
			if p.bytes == 0 || !lowerHash(p.digest) {
				t.Fatal("sealed pending inventory precursor")
			}
			switch mode {
			case "full-entry-array":
				m.data.count = len(m.data.entries)
			case "required-missing":
				p.mustWhole = true
			case "object-too-long":
				p.kind, p.bytes, p.digest = providerHistoryRecoveryObject, 1, r.pending.inventory.entries[0].digest
			}
			data, files, bytes := m.data, r.inputFiles, r.inputBytes
			out, whole, err := m.read(context.Background(), p)
			if err == nil || out != nil || whole || m.data != data || !r.activeLocked(r.meter) || o.meter.actual != old || o.store.clock.read != clocks ||
				o.pin != pin || o.last != ack || o.store.stored != held || o.store.activeMeter.Load() != nil || r.catalogResult != nil {
				t.Fatal("incomplete bounded original tail supplied DATA or changed original execution", mode, err)
			}
			switch mode {
			case "full-entry-array":
				if r.inputFiles != files || r.inputBytes != bytes {
					t.Fatal("full private array reached an input read")
				}
			case "required-missing":
				if r.inputFiles != files+1 || r.inputBytes != bytes {
					t.Fatal("missing required original did not preserve its exact observed absence")
				}
			case "object-too-long":
				if r.inputFiles != files+1 || r.inputBytes <= bytes+1 {
					t.Fatal("small expected tail size was not compared to a whole original input")
				}
			}
		})
	}
}
