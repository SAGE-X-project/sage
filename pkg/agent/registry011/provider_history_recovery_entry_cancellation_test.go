package registry011

import (
	"context"
	"reflect"
	"testing"
)

func TestProviderHistoryEnteredRecoveryCancellationClosesOnlyItsInspectionLane(t *testing.T) {
	for _, mode := range []string{"inventory", "catalog", "legacy-prefix", "initial-model"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			if mode == "legacy-prefix" {
				historyRecoveryLegacySource(t, r, 0, want)
			}
			if mode == "initial-model" {
				historyRecoveryModelPrefix(t, r, 0, want)
			}
			s := o.store
			producer, reads, held, pin, ack := o.meter.actual, s.clock.read, s.stored, o.pin, o.last
			work, upper, files, rawBytes, clocks := r.meter.actual, r.meter.upper, r.inputFiles, r.inputBytes, r.meter.clocks
			before := providerHistoryProvisionedOriginalBytes(t, s.config)
			if !o.failed || !o.meter.closed || !r.active || r.closed || r.failed || s.activeMeter.Load() != nil {
				t.Fatal("genuine failed producer and separate read-only inspection precursor")
			}
			invoke := func(ctx context.Context) error {
				switch mode {
				case "inventory":
					out, err := r.newInventoryWalkCandidate(ctx, 0)
					if out != nil {
						t.Fatal("cancelled inventory published a reader")
					}
					return err
				case "catalog":
					out, err := r.beginCatalogCandidate(ctx)
					if out != nil {
						t.Fatal("cancelled catalog published a snapshot")
					}
					return err
				case "legacy-prefix":
					out, err := r.replayLegacyPrefixCandidate(ctx, 0)
					if out != nil {
						t.Fatal("cancelled legacy replay published a prefix")
					}
					return err
				default:
					out, err := r.replayInitialModelCandidate(ctx, 0)
					if out != nil {
						t.Fatal("cancelled initial replay published a model")
					}
					return err
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := invoke(ctx); err == nil || r.active || !r.closed || !r.failed || !r.meter.closed || !r.meter.failed {
				t.Fatal("entered cancellation did not quarantine the same inspection lane", err)
			}
			// Clearing the caller's cancellation cannot reopen this original one-use
			// lane or lend the failed producer another clock, quota or execution role.
			if err := invoke(context.Background()); err == nil || r.active || !r.closed || !r.failed ||
				r.meter.actual != work || r.meter.upper != upper || r.meter.clocks != clocks || r.inputFiles != files || r.inputBytes != rawBytes ||
				o.meter.actual != producer || s.clock.read != reads || s.stored != held || o.pin != pin || o.last != ack ||
				!o.failed || !o.meter.closed || s.activeMeter.Load() != nil || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("inspection cancellation lost an original prefix or renewed authority", err)
			}
		})
	}
}
