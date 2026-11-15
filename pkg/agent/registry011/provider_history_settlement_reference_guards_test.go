package registry011

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Only private expected metadata is varied during this UNIT's ordinary failed
// S write. Scalar probes never replace a clock, reservation, order or writer.
// The actual writer continues with its exact original metadata afterwards.
func TestProviderHistoryFailedSettlementShapesRequireOriginalFailureAndPrefixMetadata(t *testing.T) {
	for _, mode := range []string{"attempt-selection", "known-failure", "prefix-result", "terminal-model", "object-count", "model-readback"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyMemberDispatchFixture(t, 2)
			historyRetainNormalPrefix(t, o, d, 1)
			s, m, r := o.store, o.meter, o.parent.recovery
			ack, pin, confirmed := o.last, o.pin, r.confirmed
			syncFile := s.custody.syncFile
			defer func() { s.custody.syncFile = syncFile }()
			reached := false
			s.custody.syncFile = func(file *os.File) error {
				if err := syncFile(file); err != nil {
					return err
				}
				c := s.frontierFiles.historyPending
				if reached || c == nil || c.frame == nil || c.failure.kind != "failure" || filepath.Base(file.Name()) != providerCustodyName("HEAD.pending", c.frame.after.ordinal) {
					return nil
				}
				if c.count != 3 || c.used || c.renamed || !d.failedMemberShapeLocked() || !o.failedSettlementClaimShapeLocked(c) || !o.claimActiveLocked(c) {
					t.Fatal("genuine complete failed-S preparation required")
				}
				reached = true
				work, reads, clocks, upper := m.actual, s.clock.read, m.clocks, m.upper
				pending, pendingAck, held := r.pending, r.pendingAck, s.stored
				originals := providerHistoryProvisionedOriginalBytes(t, s.config)
				switch mode {
				case "attempt-selection":
					original := r.memberAttempt
					r.memberAttempt = nil
					accepted := d.failedMemberShapeLocked()
					r.memberAttempt = original
					if accepted {
						t.Fatal("missing original attempt selected a failed member")
					}
				case "known-failure":
					failure := d.attempt.failure
					failure.knownFailure = false
					accepted := d.failedMemberShapeLocked()
					failure.knownFailure = true
					if accepted {
						t.Fatal("unknown failure selected terminal settlement")
					}
				case "prefix-result":
					original := d.originals[0].result
					d.originals[0].result = "incomplete private result DATA"
					accepted := d.failedMemberShapeLocked()
					d.originals[0].result = original
					if accepted {
						t.Fatal("incomplete prefix reference selected a failed member")
					}
				case "terminal-model", "object-count":
					copy := *c
					if mode == "terminal-model" {
						copy.coordinator = nil
					} else {
						copy.count++
					}
					if o.failedSettlementClaimShapeLocked(&copy) {
						t.Fatal("incomplete private claim DATA satisfied failed-S shape")
					}
				case "model-readback":
					// Coherent comparison DATA still cannot replace reconstruction
					// from retained originals. Restore only these two strings before
					// the actual original writer resumes; no state/counter is reset.
					func() {
						head, sealed := c.coordinator.head, pending.model
						defer func() { c.coordinator.head, pending.model = head, sealed }()
						changed := admissionTestDigest("f")
						if changed == head {
							t.Fatal("different private comparison DATA required")
						}
						c.coordinator.head, pending.model = changed, changed
						if !o.claimActiveLocked(c) {
							t.Fatal("original owner selection must precede the independent model comparison")
						}
						if model, err := o.readFailedSettlementLocked(context.Background(), c, false); err == nil || model != nil {
							t.Fatal("coherent private DATA replaced the retained failure model", err)
						}
					}()
				}
				if !d.failedMemberShapeLocked() || !o.failedSettlementClaimShapeLocked(c) || !o.claimActiveLocked(c) ||
					m.failed || m.closed || m.upper != upper || m.clocks != clocks || s.clock.read != reads || s.stored != held ||
					m.actual[0] != work[0] || m.actual[1] != work[1] || o.pin != pin || o.last != ack || r.confirmed != confirmed ||
					r.pending != pending || r.pendingAck != pendingAck || c.used || c.renamed || s.activeMeter.Load() != m ||
					!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
					t.Fatal("private metadata refusal changed original failed-S state", mode)
				}
				for coordinate, before := range work {
					if m.actual[coordinate] < before {
						t.Fatal("private metadata probe refunded observed work", coordinate)
					}
				}
				return nil
			}
			member := d.coordinator.ticket.table.members[1]
			connection := providerIntegrationOwnedTLS(t, member.request.body, `{}`)
			out, err := o.retainMemberCandidateWithConnection(context.Background(), d, 1, connection)
			if err == nil || out != nil || !reached || o.last == ack || o.last == nil || o.last.claim.failure.kind != "failure" ||
				!o.last.claim.used || !o.last.claim.renamed || r.confirmed != o.last.claim.checkpoint || r.pending != nil || r.pendingAck != nil ||
				!o.failed || !m.failed || !m.closed || s.activeMeter.Load() != nil || d.retained != 1 {
				t.Fatal("private probes changed the actual known-failure outcome", mode, err)
			}
		})
	}
}

func TestProviderHistorySettlementRejectsChangedConsumedOriginalBeforePublication(t *testing.T) {
	o, d := historyMemberDispatchFixture(t, 1)
	historyRetainNormalPrefix(t, o, d, 1)
	s, m, r := o.store, o.meter, o.parent.recovery
	if !d.completedMemberShapeLocked() || r.pending != nil {
		t.Fatal("genuine complete successful prefix under C required")
	}
	path := filepath.Join(s.config.custodyDirectory, s.name("frontier", o.pin.sequence))
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 || raw[0] != '{' {
		t.Fatal("actual original C frontier required", err)
	}
	raw[0] = '!'
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal("finite owned original DATA fault failed", err)
	}
	originals := providerHistoryProvisionedOriginalBytes(t, s.config)
	ack, pin, confirmed, held, reads, work, upper := o.last, o.pin, r.confirmed, s.stored, s.clock.read, m.actual, m.upper
	if out, err := o.retainSettlementCandidate(context.Background(), d); err == nil || out != nil ||
		o.last != ack || o.pin != pin || r.confirmed != confirmed || r.pending != nil || s.frontierFiles.historyPending != nil ||
		s.stored != held || s.clock.read != reads+1 || m.upper != upper || !o.failed || !m.failed || !m.closed ||
		s.activeMeter.Load() != nil || m.actual[0] != work[0] || m.actual[1] != work[1] ||
		!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("changed original C became a successful S acknowledgment", err)
	}
}
