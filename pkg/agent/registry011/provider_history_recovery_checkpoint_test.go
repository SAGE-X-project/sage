package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProviderHistoryRecoveryCheckpointsKeepOriginalInitialAdmissionAndConsumption(t *testing.T) {
	o, _ := historyMemberDispatchFixture(t, 1)
	r := o.parent.recovery
	c := r.confirmed
	if c == nil || c != o.last.claim.checkpoint || c.ack != o.last || c.pin != o.pin || c.head != o.store.frontierFiles.pinRaw ||
		c.frame != *o.last.claim.frame || c.inventory != *o.last.claim.inventory || c.failedS || r.pending != nil || !r.checkpointSelectionLocked(o) {
		t.Fatal("confirmed C checkpoint differs from the original serialized commit")
	}
	a, initial := c.previous, r.initial
	if a == nil || a.claim != o.admission.claim || a.ack != o.admission || a.frame.phase != "admit" || a.previous != initial ||
		initial == nil || initial.previous != nil || initial.claim != nil || initial.ack != nil || initial.pin != o.start.main || initial.scope != o.scope ||
		c.frame.phase != "consume" || c.frame.before != a.pin || a.frame.before != initial.pin ||
		c.frame.previousModel != a.model || a.frame.previousModel != initial.model {
		t.Fatal("checkpoint chain omitted or replaced an original confirmed prefix")
	}
	// Change producer-side DATA temporarily. The sealed value snapshots must
	// not alias either mutable frame/inventory or their cached catalog maps.
	claim := o.last.claim
	frame, inventory := *claim.frame, *claim.inventory
	claim.frame.intent = "changed producer DATA"
	claim.inventory.raw = "changed producer inventory"
	claim.inventory.entries[0].kind = "changed producer reference"
	if c.frame != frame || c.inventory != inventory {
		t.Fatal("sealed checkpoint followed a mutable producer alias")
	}
	*claim.frame, *claim.inventory = frame, inventory
	if !r.checkpointSelectionLocked(o) || json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryCheckpoint{}) == nil {
		t.Fatal("original checkpoint selection changed or JSON supplied provenance")
	}
}

func TestProviderHistoryRecoveryCheckpointSelectionRefusesDetachedChainsWithoutObservation(t *testing.T) {
	o, _ := historyMemberDispatchFixture(t, 1)
	r := o.parent.recovery
	o.store.mu.Lock()
	o.failLocked()
	o.store.mu.Unlock()
	if !r.checkpointSelectionLocked(o) {
		t.Fatal("genuine confirmed checkpoint precursor")
	}
	for _, mode := range []string{"initial", "confirmed", "copied", "ack", "cycle", "before", "pending"} {
		t.Run(mode, func(t *testing.T) {
			initial, confirmed, pending := r.initial, r.confirmed, r.pending
			previous, ack, before := confirmed.previous, confirmed.ack, confirmed.frame.before
			actual, old, reads := r.meter.actual, o.meter.actual, o.store.clock.read
			switch mode {
			case "initial":
				r.initial = nil
			case "confirmed":
				r.confirmed = nil
			case "copied":
				copy := *confirmed
				r.confirmed = &copy
			case "ack":
				confirmed.ack = nil
			case "cycle":
				confirmed.previous = confirmed
			case "before":
				confirmed.frame.before = confirmed.pin
			case "pending":
				r.pending = &providerHistoryRecoveryCheckpoint{}
			}
			accepted := r.beginInspectionCandidate(context.Background(), o) == nil
			r.initial, r.confirmed, r.pending = initial, confirmed, pending
			confirmed.previous, confirmed.ack, confirmed.frame.before = previous, ack, before
			if accepted || r.used || r.active || r.meter.actual != actual || o.meter.actual != old || o.store.clock.read != reads || !r.checkpointSelectionLocked(o) {
				t.Fatal("detached chain observed data or replaced the original selection", mode)
			}
		})
	}
	if err := r.beginInspectionCandidate(context.Background(), o); err != nil || !r.used || !r.active {
		t.Fatal("otherwise-valid terminal entry after isolated refusals", err)
	}
}

// Finite private-pointer faults in this test's owned writer only. The hook
// waits for actual HEAD rename and changes no wire message or retained file.
func historyCheckpointPostCASFault(t *testing.T, phase, mode string) {
	t.Helper()
	var o *providerHistoryGroupOwner
	var run func() error
	switch phase {
	case "admit":
		var upper providerHistoryGroupUpper
		o, upper = historyAdmissionWriterFixture(t)
		run = func() error {
			out, err := o.retainAdmissionCandidate(context.Background(), "selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper)
			if out != nil {
				t.Fatal("changed original checkpoint returned T")
			}
			return err
		}
	case "consume":
		var admission *providerHistoryGroupAcknowledgment
		o, admission = historyConsumptionFixture(t, 1)
		run = func() error {
			out, err := o.retainConsumptionCandidate(context.Background(), admission)
			if out != nil {
				t.Fatal("changed original checkpoint returned C")
			}
			return err
		}
	case "settle":
		var d *providerHistoryGroupDispatch
		o, d = historyCompletedMemberFixture(t, 1)
		run = func() error {
			out, err := o.retainSettlementCandidate(context.Background(), d)
			if out != nil {
				t.Fatal("changed original checkpoint returned successful S")
			}
			return err
		}
	case "failed-settle":
		var d *providerHistoryGroupDispatch
		o, d = historyMemberDispatchFixture(t, 1)
		run = func() error {
			member := d.coordinator.ticket.table.members[0]
			conn := providerIntegrationOwnedTLS(t, member.request.body, `{}`)
			out, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, conn)
			if out != nil {
				t.Fatal("changed original checkpoint returned failed capture")
			}
			return err
		}
	default:
		t.Fatal("unknown owned checkpoint test phase")
	}
	s, r := o.store, o.parent.recovery
	confirmed, ack, pin, held, raw := r.confirmed, o.last, o.pin, s.stored, s.retainedRaw
	reached := false
	var pending *providerHistoryRecoveryCheckpoint
	var faultRead, faultRequests uint64
	syncFile := s.custody.syncFile
	s.custody.syncFile = func(file *os.File) error {
		if err := syncFile(file); err != nil {
			return err
		}
		c := s.frontierFiles.historyPending
		info, err := file.Stat()
		if err != nil {
			return err
		}
		if reached || c == nil || !info.IsDir() {
			return nil
		}
		head, err := os.ReadFile(filepath.Join(s.config.custodyDirectory, "HEAD"))
		if err != nil || string(head) != c.highwater {
			return nil
		}
		pending = c.checkpoint
		if pending == nil || r.pending != pending || pending.previous != confirmed || pending.ack == nil {
			t.Fatal("actual physical write has no original sealed checkpoint")
		}
		reached, faultRead, faultRequests = true, s.clock.read, o.meter.actual[0]
		switch mode {
		case "nil":
			c.checkpoint = nil
		case "copy":
			copy := *pending
			copy.owner = &copy
			c.checkpoint = &copy
		case "previous":
			c.checkpoint = confirmed
		case "ack":
			copy := *pending.ack
			pending.ack = &copy
		case "frame-raw":
			pending.frame.intent = "inert sealed frame mismatch"
		case "inventory-raw":
			pending.inventory.raw = "inert sealed inventory mismatch"
		case "highwater-raw":
			pending.head = "inert sealed highwater mismatch"
		default:
			t.Fatal("unknown owned checkpoint fault")
		}
		return nil
	}
	if err := run(); err == nil || !reached || pending == nil || r.confirmed != confirmed || r.pending != pending || o.last != ack || o.pin != pin ||
		s.frontierFiles.pin != pin || s.stored != held || s.retainedRaw != raw || !o.failed || !s.failed || !o.meter.failed || !o.meter.closed ||
		s.clock.read != faultRead || o.meter.actual[0] != faultRequests || s.activeMeter.Load() != nil {
		t.Fatal("changed checkpoint reached final native sample/ACK or restored authority", phase, mode, err)
	}
}

func TestProviderHistoryRecoveryEveryWriterRejectsCheckpointReplacementAfterCAS(t *testing.T) {
	for _, phase := range []string{"admit", "consume", "settle", "failed-settle"} {
		t.Run(phase, func(t *testing.T) { historyCheckpointPostCASFault(t, phase, "nil") })
	}
}

func TestProviderHistoryRecoveryRejectsChangedSealedDataAndACKAfterCAS(t *testing.T) {
	for _, mode := range []string{"copy", "previous", "ack", "frame-raw", "inventory-raw", "highwater-raw"} {
		t.Run(mode, func(t *testing.T) { historyCheckpointPostCASFault(t, "settle", mode) })
	}
}
