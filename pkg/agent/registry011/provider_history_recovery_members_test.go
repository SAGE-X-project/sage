package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProviderHistoryRecoveryMemberSelectionRequiresOriginalProgress(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 2)
	historyRetainNormalPrefix(t, o, d, 1)
	r := o.parent.recovery
	o.store.mu.Lock()
	o.failLocked()
	o.store.mu.Unlock()
	if !r.memberSelectionLocked(o) || r.completed != 1 || r.sent != 1 || r.members[0] == nil || r.members[1] != nil {
		t.Fatal("genuine original terminal one-of-two prefix")
	}
	for _, mode := range []string{"order", "dispatch", "consumed", "sent", "completed", "intent", "capture", "tail", "completed-copy"} {
		t.Run(mode, func(t *testing.T) {
			order, dispatch, consumed, sent, completed := r.order, r.originalDispatch, r.consumed, r.sent, r.completed
			intent, capture, first, tail := r.memberAttempt, r.memberCapture, r.members[0], r.members[1]
			actual, old, reads, held := r.meter.actual, o.meter.actual, o.store.clock.read, o.store.stored
			switch mode {
			case "order":
				copy := *r.order
				copy.owner = &copy
				r.order = &copy
			case "dispatch":
				copy := *r.originalDispatch
				copy.owner = &copy
				r.originalDispatch = &copy
			case "consumed":
				r.consumed ^= 1
			case "sent":
				r.sent ^= 1
			case "completed":
				r.completed++
			case "intent":
				r.memberAttempt = &providerHistoryRecoveryMemberAttempt{}
			case "capture":
				r.memberCapture = &providerHistoryRecoveryMemberCapture{}
			case "tail":
				r.members[1] = r.members[0]
			case "completed-copy":
				copy := *r.members[0]
				copy.owner = &copy
				r.members[0] = &copy
			}
			err := r.beginInspectionCandidate(context.Background(), o)
			r.order, r.originalDispatch, r.consumed, r.sent, r.completed = order, dispatch, consumed, sent, completed
			r.memberAttempt, r.memberCapture, r.members[0], r.members[1] = intent, capture, first, tail
			if err == nil || r.used || r.active || r.meter.actual != actual || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || !r.memberSelectionLocked(o) {
				t.Fatal("detached progress was observed, adopted or renewed an original", mode)
			}
		})
	}
	if err := r.beginInspectionCandidate(context.Background(), o); err != nil || !r.used || !r.active || r.completed != 1 || r.sent != 1 {
		t.Fatal("otherwise-valid original prefix could not enter read-only preparation", err)
	}
	for _, value := range []any{&providerHistoryRecoveryMemberAttempt{}, &providerHistoryRecoveryMemberCapture{}, &providerHistoryRecoveryMemberComplete{}} {
		if json.Unmarshal([]byte(`{}`), value) == nil {
			t.Fatal("serialized DATA manufactured original member progress")
		}
	}
}

func TestProviderHistoryRecoveryMemberSelectionRejectsCopiedConsumption(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 2)
	r := o.parent.recovery
	o.store.mu.Lock()
	o.failLocked()
	o.store.mu.Unlock()
	if !r.memberSelectionLocked(o) || !r.checkpointSelectionLocked(o) || r.completed != 0 || r.sent != 0 || d.attempt != nil || r.consumption != r.confirmed ||
		r.consumption != d.acknowledgment.claim.checkpoint {
		t.Fatal("genuine original terminal zero-member prefix")
	}
	original := r.consumption
	copy := *original
	copy.owner = &copy
	r.consumption = &copy
	actual, old, reads, held := r.meter.actual, o.meter.actual, o.store.clock.read, o.store.stored
	if !r.checkpointSelectionLocked(o) || r.memberSelectionLocked(o) || r.beginInspectionCandidate(context.Background(), o) == nil || r.used || r.active ||
		r.meter.actual != actual || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held {
		t.Fatal("a self-owned copied C checkpoint entered zero-prefix recovery or observed an effect")
	}
	r.consumption = original
	if !r.memberSelectionLocked(o) || r.beginInspectionCandidate(context.Background(), o) != nil || !r.used || !r.active {
		t.Fatal("otherwise-valid original C checkpoint could not enter read-only preparation")
	}
}

func TestProviderHistoryRecoveryMemberCaptureRejectsPointerAndNameChanges(t *testing.T) {
	for _, mode := range []string{"nil-capture", "copied-capture", "copied-original", "name", "premature-complete"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 1)
			s, r := o.store, o.parent.recovery
			data := *d
			data.owner, data.meter = &data, nil
			name := data.originalNameLocked(0, "response")
			pin, ack, held, retained := o.pin, o.last, s.stored, o.retention
			reached := false
			var nativeRead, requests uint64
			syncFile := s.evidence.syncFile
			s.evidence.syncFile = func(file *os.File) error {
				if err := syncFile(file); err != nil {
					return err
				}
				if reached || filepath.Base(file.Name()) != name {
					return nil
				}
				a, capture := d.attempt, r.memberCapture
				if a == nil || !a.used || a.files != 0 || capture == nil || capture.original != a.original || r.memberOriginal != a.original ||
					capture.intent != r.memberAttempt || r.memberAttempt.responseName != name || capture.responseBytes == 0 || capture.tlsBytes == 0 {
					t.Fatal("first physical spool preceded its original sealed capture")
				}
				reached, nativeRead, requests = true, s.clock.read, o.meter.actual[0]
				switch mode {
				case "nil-capture":
					a.recoveryCapture = nil
				case "copied-capture":
					copy := *capture
					copy.owner = &copy
					a.recoveryCapture = &copy
				case "copied-original":
					copy := *a.original
					copy.owner = &copy
					a.original = &copy
				case "name":
					r.memberAttempt.responseName = "inert owned-name mismatch"
				case "premature-complete":
					a.recoveryComplete = &providerHistoryRecoveryMemberComplete{}
				}
				return nil
			}
			member := d.coordinator.ticket.table.members[0]
			conn := providerIntegrationOwnedTLS(t, member.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
			if out, err := o.retainMemberCandidateWithConnection(context.Background(), d, 0, conn); err == nil || out != nil || !reached ||
				!o.failed || !o.meter.failed || !o.meter.closed || d.retained != 0 || d.originals[0] != nil || r.completed != 0 || r.members[0] != nil ||
				r.sent != 1 || o.order.sent != 1 || o.order.next != 0 || o.order.state != providerHistoryOrderFailed || o.pin != pin || o.last != ack ||
				s.stored != held || o.retention.physical <= retained.physical || s.clock.read != nativeRead || o.meter.actual[0] != requests || s.activeMeter.Load() != nil {
				t.Fatal("changed capture reached completion/end or refunded an attempted member", mode, err)
			}
		})
	}
}
