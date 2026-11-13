package registry011

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The original closed selector supplies HEAD and its disk. These calls test
// only the partial file primitive; they cannot certify complete read counts,
// install a model, acknowledge a checkpoint or resume the failed producer.
func TestProviderHistoryRecoveryOpenedInputRequiresOriginalIdentityAndLiveContext(t *testing.T) {
	for _, mode := range []string{"intact", "missing-original", "different-identity", "nil-context", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			o, r := historyRecoveryInputFixture(t)
			ctx := context.Background()
			control, err := r.readCheckpointInputCandidate(ctx, r.confirmed, providerHistoryRecoveryHead, 0)
			if err != nil || control == nil || control.state != providerHistoryRecoveryWholeInput || string(control.raw) != r.confirmed.head {
				t.Fatal("genuine selected original HEAD prerequisite", err)
			}
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			disk, maximum, valid := r.checkpointInputPlanLocked(r.confirmed, providerHistoryRecoveryHead, 0)
			name := r.checkpointInputNameLocked(r.confirmed, providerHistoryRecoveryHead, 0)
			before, statErr := disk.root.Lstat(name)
			upper, bounded := r.catalogInputUpperLocked(maximum)
			if !valid || disk != r.custody || name != "HEAD" || statErr != nil || !before.Mode().IsRegular() ||
				!bounded || !r.meter.preflightWork(upper) || !r.inputReadyLocked(ctx) {
				t.Fatal("original bounded file primitive prerequisites", statErr)
			}
			switch mode {
			case "missing-original":
				if err := os.Remove(filepath.Join(r.config.custodyDirectory, name)); err != nil {
					t.Fatal("remove only the owned original HEAD", err)
				}
			case "different-identity":
				other, err := os.Stat(t.TempDir())
				if err != nil || os.SameFile(before, other) {
					t.Fatal("distinct owned metadata DATA prerequisite", err)
				}
				before = other
			case "nil-context":
				ctx = nil
			case "cancelled":
				input, cancel := context.WithCancel(context.Background())
				cancel()
				ctx = input
			}
			originals := providerHistoryProvisionedOriginalBytes(t, r.config)
			work, files, bytes, bounds := r.meter.actual, r.inputFiles, r.inputBytes, r.bounds
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			out, err := r.readOpenedInputLocked(ctx, disk, name, before)
			if mode == "intact" {
				if err != nil || out == nil || out.state != providerHistoryRecoveryWholeInput || string(out.raw) != r.confirmed.head ||
					r.inputBytes != bytes+uint64(len(control.raw)) {
					t.Fatal("intact original file primitive failed", err)
				}
			} else if err == nil || out != nil || r.inputBytes != bytes {
				t.Fatal("missing identity or cancelled input exposed partial original DATA", mode, err)
			}
			if r.meter.failed || r.meter.actual[10] <= work[10] || r.inputFiles != files || r.bounds != bounds ||
				!r.activeLocked(r.meter) || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) || r.catalogResult != nil ||
				o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				o.store.activeMeter.Load() != nil || !reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, r.config)) {
				t.Fatal("partial file inspection changed its original producer, bounds or retained inputs", mode)
			}
		})
	}
}
