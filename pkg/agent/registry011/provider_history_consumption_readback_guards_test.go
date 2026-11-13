package registry011

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestProviderHistoryConsumptionRequiresOriginalReadbackBeforeAcknowledgment(t *testing.T) {
	for _, mode := range []string{"highwater-missing", "inventory-missing", "inventory-same-width", "consumption-same-width"} {
		t.Run(mode, func(t *testing.T) {
			o, admission := historyConsumptionFixture(t, 1)
			s, m, r := o.store, o.meter, o.parent.recovery
			pin, held, retained, rawBefore := o.pin, s.stored, o.retention, s.retainedRaw
			requests, window, clockStart := m.actual[0], o.windowStart, o.clockStart
			reached := false
			syncFile := s.custody.syncFile
			// Only the fresh fixture's own files are changed, after the
			// genuine pending C exists and before its first readback.
			s.custody.syncFile = func(file *os.File) error {
				if err := syncFile(file); err != nil {
					return err
				}
				c := s.frontierFiles.historyPending
				if reached || c == nil || c.frame == nil || c.frame.phase != "consume" ||
					filepath.Base(file.Name()) != providerCustodyName("HEAD.pending", c.frame.after.ordinal) {
					return nil
				}
				if !o.claimActiveLocked(c) || c.used || c.renamed || c.order == nil ||
					c.order.state != providerHistoryOrderConsumed || c.order.consumed != 1 || c.order.sent != 0 ||
					r.pending == nil || r.pending != c.checkpoint || r.pending.previous != r.confirmed ||
					r.confirmed == nil || r.confirmed.ack != admission || r.pending.ack == admission {
					t.Fatal("readback case preceded the original pending consumed checkpoint")
				}
				var path string
				switch mode {
				case "highwater-missing":
					path = filepath.Join(s.config.custodyDirectory, providerCustodyName("highwater", c.frame.after.ordinal))
				case "inventory-missing", "inventory-same-width":
					path = filepath.Join(s.config.evidenceDirectory, providerHistoryInventoryName(c.inventory.digest, nil))
				case "consumption-same-width":
					path = filepath.Join(s.config.evidenceDirectory, providerHistoryStoredName(c.consumption, nil))
				}
				raw, err := os.ReadFile(path)
				if err != nil || len(raw) == 0 {
					t.Fatal("actual original must precede the finite owned-file fault", err)
				}
				if mode == "inventory-same-width" || mode == "consumption-same-width" {
					raw[len(raw)/2] ^= 1
					err = os.WriteFile(path, raw, 0600)
				} else {
					err = os.Remove(path)
				}
				if err != nil {
					t.Fatal("owned readback fault was not installed", mode, err)
				}
				reached = true
				return nil
			}
			out, err := o.retainConsumptionCandidate(context.Background(), admission)
			c := s.frontierFiles.historyPending
			if err == nil || out != nil || !reached || c == nil || c.used || c.renamed ||
				o.pin != pin || s.frontierFiles.pin != pin || o.last != admission || o.admission != admission ||
				s.stored != held || s.retainedRaw != rawBefore || o.retention.physical <= retained.physical ||
				!o.failed || !s.failed || !m.failed || !m.closed || s.activeMeter.Load() != nil ||
				m.actual[0] != requests || o.order != c.order || o.order.consumed != 1 || o.order.sent != 0 ||
				o.order.state != providerHistoryOrderFailed || o.windowStart != window || o.clockStart != clockStart ||
				r.confirmed == nil || r.confirmed.ack != admission || r.confirmed.pin != pin ||
				r.pending != c.checkpoint || r.pending == nil || r.pending.previous != r.confirmed {
				t.Fatal("incomplete C readback published an acknowledgment, refunded consumption or lost confirmed T", mode, err)
			}
			work, clocks := m.actual, s.clock.read
			if again, err := o.retainConsumptionCandidate(context.Background(), admission); err == nil || again != nil ||
				m.actual != work || s.clock.read != clocks || o.last != admission {
				t.Fatal("terminal C readback refusal retried original consumption")
			}
		})
	}
}

func TestProviderHistoryConsumptionReadersRejectMissingOriginsBeforeObservation(t *testing.T) {
	var absent *providerHistoryGroupOwner
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, input := range []context.Context{nil, ctx} {
		if model, err := absent.readConsumptionLocked(input, nil, false); err == nil || model != nil {
			t.Fatal("cancelled consumption read returned DATA without an original")
		}
		if model, err := absent.replayCommittedConsumptionAtLocked(input, ""); err == nil || model != nil {
			t.Fatal("cancelled historical selection returned DATA without an original")
		}
	}
	if model, err := absent.replayCommittedConsumptionAtLocked(context.Background(), ""); err == nil || model != nil {
		t.Fatal("missing consumption owner selected retained DATA")
	}
	if model, err := absent.readConsumptionDataLocked(nil, providerFrontierHighwater{}, "", nil, ""); err == nil || model != nil {
		t.Fatal("missing origin selected a historical consumption model")
	}
}
