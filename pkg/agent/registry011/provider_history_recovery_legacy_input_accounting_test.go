package registry011

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Fresh unselected bookkeeping DATA reads only through the existing original
// recovery selector. No selected prefix, live quota or counter is replaced.
func TestProviderHistoryRecoverySetupOriginalsRejectRepeatedAndUnfundedAccounting(t *testing.T) {
	for _, mode := range []string{"intact", "repeated", "missing-slot", "sum-overflow", "metadata-limit", "different-tip", "different-previous"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			original := r.legacyPrefixes[0]
			if len(original.setups) == 0 || original.setupOriginalBytes[0] == 0 || !original.selectedLocked(r) {
				t.Fatal("genuine original setup prefix prerequisite")
			}
			fresh := func() *providerHistoryRecoveryLegacyPrefix {
				return &providerHistoryRecoveryLegacyPrefix{recovery: r, setupOriginalBytes: make([]uint64, len(original.setups))}
			}
			control := fresh()
			setup, claim, commit, err := control.setupOriginalsLocked(context.Background(), 0, r.config.anchor, r.config.anchor)
			if err != nil || setup != original.setups[0] || len(claim) == 0 || len(commit) == 0 ||
				control.setupOriginalBytes[0] != original.setupOriginalBytes[0] || control.setupBytesTotal != original.setupOriginalBytes[0] {
				t.Fatal("positive actual setup read prerequisite", err)
			}
			p := fresh()
			tip, previous := r.config.anchor, r.config.anchor
			switch mode {
			case "repeated":
				p.setupOriginalBytes[0] = original.setupOriginalBytes[0]
			case "missing-slot":
				p.setupOriginalBytes = nil
			case "sum-overflow":
				p.setupBytesTotal = math.MaxUint64
			case "metadata-limit":
				p.setupBytesTotal = r.config.bounds.continuation.maximumMetadataBytes
			case "different-tip":
				tip += "x"
			case "different-previous":
				previous += "x"
			}
			initialTotal := p.setupBytesTotal
			initialSlots := append([]uint64(nil), p.setupOriginalBytes...)
			work, files, bytes, bounds := r.meter.actual, r.inputFiles, r.inputBytes, r.bounds
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			originals := providerHistoryProvisionedOriginalBytes(t, r.config)
			out, rawClaim, rawCommit, err := p.setupOriginalsLocked(context.Background(), 0, tip, previous)
			if mode == "intact" {
				if err != nil || out != setup || !reflect.DeepEqual(rawClaim, claim) || !reflect.DeepEqual(rawCommit, commit) ||
					p.setupOriginalBytes[0] != original.setupOriginalBytes[0] || p.setupBytesTotal != original.setupOriginalBytes[0] {
					t.Fatal("intact original setup accounting failed", err)
				}
			} else if err == nil || out != (providerSetupReference{}) || rawClaim != nil || rawCommit != nil ||
				p.setupBytesTotal != initialTotal || !reflect.DeepEqual(p.setupOriginalBytes, initialSlots) {
				t.Fatal("invalid setup accounting published a partial setup or refunded prior bookkeeping", mode, err)
			}
			if r.meter.failed || r.meter.actual[10] <= work[10] || r.inputFiles != files+3 || r.inputBytes != bytes+original.setupOriginalBytes[0] ||
				p.bytes != original.setupOriginalBytes[0] || p.selectedLocked(r) || r.legacyPrefixes[0] != original || !original.selectedLocked(r) ||
				r.bounds != bounds || !r.activeLocked(r.meter) || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) ||
				o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				o.store.activeMeter.Load() != nil || !reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, r.config)) {
				t.Fatal("private setup accounting changed original inputs, selected state or execution", mode)
			}
		})
	}
}

func TestProviderHistoryRecoveryLegacyReadKeepsActualInputChargesWhenBookkeepingRefuses(t *testing.T) {
	for _, mode := range []string{"intact", "missing-original", "sum-overflow", "stored-limit"} {
		t.Run(mode, func(t *testing.T) {
			o, r := historyRecoveryInputFixture(t)
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			p := &providerHistoryRecoveryLegacyPrefix{recovery: r}
			switch mode {
			case "missing-original":
				if err := os.Remove(filepath.Join(r.config.custodyDirectory, "HEAD")); err != nil {
					t.Fatal("remove only the owned original HEAD", err)
				}
			case "sum-overflow":
				p.bytes = math.MaxUint64
			case "stored-limit":
				p.bytes = r.config.bounds.maximumStoredBytes
			}
			prior := p.bytes
			work, files, bytes, bounds := r.meter.actual, r.inputFiles, r.inputBytes, r.bounds
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			originals := providerHistoryProvisionedOriginalBytes(t, r.config)
			out, err := p.readLocked(context.Background(), providerHistoryRecoveryHead, 0)
			if mode == "intact" {
				if err != nil || string(out) != r.confirmed.head || p.bytes != uint64(len(out)) {
					t.Fatal("intact original bytes were not accounted", err)
				}
			} else if err == nil || out != nil || p.bytes != prior {
				t.Fatal("missing or overflowing private original read published DATA or refunded bookkeeping", mode, err)
			}
			wantBytes := bytes
			if mode != "missing-original" {
				wantBytes += uint64(len(r.confirmed.head))
			}
			if r.meter.failed || r.meter.actual[10] <= work[10] || r.inputFiles != files+1 || r.inputBytes != wantBytes || r.bounds != bounds ||
				!r.activeLocked(r.meter) || p.selectedLocked(r) || r.legacyPrefixes != ([2]*providerHistoryRecoveryLegacyPrefix{}) ||
				o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				o.store.activeMeter.Load() != nil || !reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, r.config)) {
				t.Fatal("private bookkeeping refusal lost actual input charges or changed the original producer", mode)
			}
		})
	}
}
