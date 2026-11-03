package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// This fault is confined to a fresh UNIT fixture's temporary files. It runs
// after the original pending checkpoint exists and all prospective S files
// have been written, but before the first S readback and physical HEAD rename.
// No caller can supply a path, peer, retained record or execution allowance.
func historySettlementReadbackFault(t *testing.T, o *providerHistoryGroupOwner, mode string, failed bool) *bool {
	t.Helper()
	s := o.store
	reached := new(bool)
	syncFile := s.custody.syncFile
	s.custody.syncFile = func(file *os.File) error {
		if err := syncFile(file); err != nil {
			return err
		}
		c := s.frontierFiles.historyPending
		if *reached || c == nil || c.frame == nil || c.frame.phase != "settle" ||
			filepath.Base(file.Name()) != providerCustodyName("HEAD.pending", c.frame.after.ordinal) {
			return nil
		}
		r := o.parent.recovery
		if !o.claimActiveLocked(c) || c.used || c.renamed || (c.failure.kind == "failure") != failed ||
			r.pending != c.checkpoint || r.pending == nil || r.pending.previous != r.confirmed ||
			r.confirmed.ack != o.last || r.pending.ack == o.last {
			t.Fatal("readback fault preceded the genuine pending S checkpoint")
		}
		var path string
		switch mode {
		case "highwater-missing":
			path = filepath.Join(s.config.custodyDirectory, providerCustodyName("highwater", c.frame.after.ordinal))
		case "inventory-missing", "inventory-same-width":
			path = filepath.Join(s.config.evidenceDirectory, providerHistoryInventoryName(c.inventory.digest, nil))
		case "object-same-width":
			path = filepath.Join(s.config.evidenceDirectory, providerHistoryStoredName(c.inventory.entries[0], nil))
		case "spool-same-width":
			if failed || c.dispatch.retained != 1 {
				t.Fatal("one genuinely completed member required for spool readback")
			}
			path = filepath.Join(s.config.evidenceDirectory, c.dispatch.originalNameLocked(0, "response"))
		default:
			t.Fatal("unknown owned readback case")
		}
		raw, err := os.ReadFile(path)
		if err != nil || len(raw) == 0 {
			t.Fatal("actual original file must exist before the isolated fault", mode, err)
		}
		*reached = true
		if mode == "object-same-width" || mode == "inventory-same-width" || mode == "spool-same-width" {
			raw[len(raw)/2] ^= 1
			return os.WriteFile(path, raw, 0600)
		}
		return os.Remove(path)
	}
	return reached
}

func TestProviderHistorySettlementRequiresOriginalReadbackBeforeAcknowledgment(t *testing.T) {
	for _, mode := range []string{"highwater-missing", "inventory-missing", "inventory-same-width", "object-same-width", "spool-same-width"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyCompletedMemberFixture(t, 1)
			s, m, r := o.store, o.meter, o.parent.recovery
			pin, ack, held, retained, rawBefore := o.pin, o.last, s.stored, o.retention, s.retainedRaw
			requests, sent, window, clockStart := m.actual[0], o.order.sent, o.windowStart, o.clockStart
			reached := historySettlementReadbackFault(t, o, mode, false)
			out, err := o.retainSettlementCandidate(context.Background(), d)
			c := s.frontierFiles.historyPending
			if err == nil || out != nil || !*reached || c == nil || c.used || c.renamed ||
				o.pin != pin || s.frontierFiles.pin != pin || o.last != ack || s.stored != held || s.retainedRaw != rawBefore ||
				o.retention.physical <= retained.physical || !o.failed || !m.failed || !m.closed || s.activeMeter.Load() != nil ||
				m.actual[0] != requests || o.order.sent != sent || o.order.state != providerHistoryOrderFailed ||
				o.windowStart != window || o.clockStart != clockStart || r.confirmed == nil || r.confirmed.ack != ack ||
				r.confirmed.pin != pin || r.pending != c.checkpoint || r.pending == nil || r.pending.previous != r.confirmed || r.pending.failedS {
				t.Fatal("incomplete original readback published S, renewed execution or lost the confirmed C", mode, err)
			}
			work, clocks := m.actual, s.clock.read
			if again, err := o.retainSettlementCandidate(context.Background(), d); err == nil || again != nil || m.actual != work || s.clock.read != clocks || o.last != ack {
				t.Fatal("terminal readback refusal retried the original execution")
			}
		})
	}
}

func TestProviderHistoryFailedSettlementRequiresOriginalReadbackBeforeAcknowledgment(t *testing.T) {
	for _, mode := range []string{"highwater-missing", "inventory-missing", "inventory-same-width", "object-same-width"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 1)
			s, m, r := o.store, o.meter, o.parent.recovery
			pin, ack, held, retained, rawBefore := o.pin, o.last, s.stored, o.retention, s.retainedRaw
			requests, window, clockStart := m.actual[0], o.windowStart, o.clockStart
			reached := historySettlementReadbackFault(t, o, mode, true)
			member := d.coordinator.ticket.table.members[0]
			// Reuse the existing finite owned invalid-RPC UNIT response.
			// No new peer, transport driver or attack payload is introduced.
			conn := providerIntegrationOwnedTLS(t, member.request.body, `{}`)
			out, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, conn)
			c := s.frontierFiles.historyPending
			if err == nil || out != nil || !*reached || c == nil || c.used || c.renamed ||
				o.pin != pin || s.frontierFiles.pin != pin || o.last != ack || s.stored != held || s.retainedRaw != rawBefore ||
				o.retention.physical <= retained.physical || !o.failed || !m.failed || !m.closed || s.activeMeter.Load() != nil ||
				m.actual[0] != requests+1 || o.order.sent != 1 || o.order.next != 0 || o.order.state != providerHistoryOrderFailed ||
				o.windowStart != window || o.clockStart != clockStart || r.confirmed == nil || r.confirmed.ack != ack ||
				r.confirmed.pin != pin || r.pending != c.checkpoint || r.pending == nil || r.pending.previous != r.confirmed || !r.pending.failedS {
				t.Fatal("uncertain failed-S readback acknowledged the attempt, refunded work or lost the confirmed C", mode, err)
			}
		})
	}
}

func TestProviderHistorySettlementReadersRejectCancellationBeforeOwnerAccess(t *testing.T) {
	var absent *providerHistoryGroupOwner
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, c := range []context.Context{nil, ctx} {
		if model, prefix, err := absent.readSettlementLocked(c, nil, false); err == nil || model != nil || prefix != nil {
			t.Fatal("cancelled settlement reader returned DATA without an original owner")
		}
		if model, err := absent.readFailedSettlementLocked(c, nil, false); err == nil || model != nil {
			t.Fatal("cancelled failed settlement reader returned DATA without an original owner")
		}
		if err := absent.applyHealthySettlementLocked(c, nil, nil, nil, [8]providerHistoryStoredObject{}); err == nil {
			t.Fatal("cancelled historical settlement reader reached absent originals")
		}
		if model, err := absent.replayCommittedAdmissionLocked(c, ""); err == nil || model != nil {
			t.Fatal("cancelled admission reader returned DATA without an original owner")
		}
		if model, err := absent.replayCommittedConsumptionLocked(c); err == nil || model != nil {
			t.Fatal("cancelled consumption reader returned DATA without an original owner")
		}
	}
	var dispatch providerHistoryGroupDispatch
	if json.Unmarshal([]byte(`{}`), &dispatch) == nil {
		t.Fatal("serialized DATA manufactured an original dispatch preparation")
	}
}
