package registry011

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// These faults affect only the ordinary sync of files inside t.TempDir.
// No hostile responder, arbitrary operation callback or external peer exists.
// The first stage is genuinely completed before the later-stage fault starts.
func TestProviderHistoryLaterStageFailuresPreserveEarlierArchivesThroughFullInspection(t *testing.T) {
	for _, mode := range []string{"admit-first-write", "admit-after-rename", "consume-first-write", "consume-after-rename",
		"member-response", "member-tls", "settle-first-write", "settle-after-rename", "failed-settlement", "failed-after-rename"} {
		t.Run(mode, func(t *testing.T) {
			o := historyMultistageOwner(t, 2)
			first := historyMultistageSettle(t, o, 1)
			r, s, m := o.parent.recovery, o.store, o.meter
			archive := r.confirmed.memberArchive
			if err := o.advanceStageCandidate(context.Background(), first.ack); err != nil {
				t.Fatal("genuine successful first-stage transition", err)
			}
			phase := "settle"
			var ticket *providerHistoryGroupAcknowledgment
			var dispatch *providerHistoryGroupDispatch
			switch {
			case strings.HasPrefix(mode, "admit-"):
				phase = "admit"
			case strings.HasPrefix(mode, "consume-"):
				phase = "consume"
				var err error
				ticket, err = o.retainAdmissionCandidate(context.Background(), "identity", []*providerIntegrationRequest{historyGroupRequest(t, "web3_clientVersion", []any{})}, historyMultistageTestUpper(s.config.capture))
				if err != nil || ticket == nil || ticket.claim.frame.stage != 2 {
					t.Fatal("genuine second T before the C fault", err)
				}
			default:
				dispatch = historyMultistageDispatch(t, o, 2)
				if strings.HasPrefix(mode, "settle-") {
					historyRetainNormalPrefix(t, o, dispatch, 1)
				}
			}
			pin, ack, confirmed, held, original := o.pin, o.last, r.confirmed, s.stored, o.parent.original
			reached := false
			custodySync, evidenceSync := s.custody.syncFile, s.evidence.syncFile
			if strings.Contains(mode, "after-rename") {
				s.custody.syncFile = func(file *os.File) error {
					if err := custodySync(file); err != nil {
						return err
					}
					c := s.frontierFiles.historyPending
					if c != nil && c.frame.stage == 2 && c.frame.phase == phase && c.renamed {
						reached = true
						return errProviderIntegration // Owned post-rename sync uncertainty.
					}
					return nil
				}
			} else if mode != "failed-settlement" {
				s.evidence.syncFile = func(file *os.File) error {
					if err := evidenceSync(file); err != nil {
						return err
					}
					name := filepath.Base(file.Name())
					if strings.HasPrefix(mode, "member-") {
						n := r.memberAttempt
						if n == nil || n.stage != 2 || o.phase != "retaining-member" {
							return nil
						}
						expected := n.responseName
						if mode == "member-tls" {
							expected = n.tlsName
						}
						if name != expected {
							return nil
						}
					} else {
						c := s.frontierFiles.historyPending
						if c == nil || c.frame.stage != 2 || c.frame.phase != phase || c.count < 1 || name != providerHistoryStoredName(c.inventory.entries[0], nil) {
							return nil
						}
					}
					reached = true
					return errProviderIntegration // Only this owned original's sync fails.
				}
			}
			var operationErr error
			switch phase {
			case "admit":
				var out *providerHistoryGroupAcknowledgment
				out, operationErr = o.retainAdmissionCandidate(context.Background(), "identity", []*providerIntegrationRequest{historyGroupRequest(t, "web3_clientVersion", []any{})}, historyMultistageTestUpper(s.config.capture))
				if out != nil {
					t.Fatal("uncertain T produced an acknowledgment")
				}
			case "consume":
				var out *providerHistoryGroupAcknowledgment
				out, operationErr = o.retainConsumptionCandidate(context.Background(), ticket)
				if out != nil {
					t.Fatal("uncertain C produced an acknowledgment")
				}
			default:
				if strings.HasPrefix(mode, "settle-") {
					var out *providerHistorySettledResults
					out, operationErr = o.retainSettlementCandidate(context.Background(), dispatch)
					if out != nil {
						t.Fatal("uncertain S returned a protected result")
					}
				} else {
					member := dispatch.coordinator.ticket.table.members[0]
					response := `{"jsonrpc":"2.0","id":1,"result":"owned-node"}`
					if strings.HasPrefix(mode, "failed-") {
						response = `{}` // Ordinary invalid RPC object, not an attack payload.
					}
					conn := providerIntegrationOwnedTLS(t, member.request.body, response)
					var out *providerHistoryMemberOriginal
					out, operationErr = o.retainMemberCandidateWithConnection(context.Background(), dispatch, 0, conn)
					if out != nil {
						t.Fatal("failed/uncertain member returned a completed original")
					}
				}
			}
			s.custody.syncFile, s.evidence.syncFile = custodySync, evidenceSync
			if operationErr == nil || mode != "failed-settlement" && !reached || !o.failed || !s.failed || !m.failed || !m.closed ||
				!o.parent.failed || !s.clock.historyRevoked || s.stored != held || o.parent.original != original || s.activeMeter.Load() != nil ||
				!archive.shapeLocked(r, archive.checkpoint) || archive.order.state != providerHistoryOrderClosed || archive.order.consumed != 1 || archive.order.sent != 1 {
				t.Fatal("later-stage failure renewed work or lost the earlier genuine S", operationErr)
			}
			if mode == "failed-settlement" {
				if r.confirmed == confirmed || !r.confirmed.failedS || r.confirmed.frame.stage != 2 || o.last == ack || r.pending != nil {
					t.Fatal("known RPC failure did not retain its exact terminal S")
				}
			} else if o.last != ack || r.confirmed != confirmed || o.pin != pin || s.frontierFiles.pin != pin {
				t.Fatal("uncertain later write promoted an ACK or logical pin")
			}
			if strings.Contains(mode, "after-rename") && (r.pending == nil || !r.pending.claim.renamed) {
				t.Fatal("post-rename failure did not retain the actual unacknowledged physical tail")
			}
			confirmed, ack, pin = r.confirmed, o.last, o.pin
			physical := confirmed
			if strings.Contains(mode, "after-rename") {
				physical = r.pending
			}
			tailFiles := 9 // Two pending T/S objects and seven planned controls.
			switch {
			case strings.HasPrefix(mode, "consume-"):
				tailFiles = 8 // One pending C object and seven planned controls.
			case strings.HasPrefix(mode, "member-"), mode == "failed-settlement":
				tailFiles = 2 // The original unfinished response/TLS slots only.
			case mode == "failed-after-rename":
				tailFiles = 11 // Pending failed-S objects/controls and unfinished slots.
			}
			incomplete := strings.HasSuffix(mode, "first-write") || mode == "member-response"
			work, clocks, before := m.actual, s.clock.read, providerHistoryProvisionedOriginalBytes(t, s.config)
			r, want := historyRecoveryGroupStart(t, o)
			if c, err := r.beginCatalogCandidate(context.Background()); err != nil || c == nil {
				t.Fatal("actual C0 before later-stage original reconstruction", err)
			}
			for pass := range 2 {
				historyRecoveryTailPrecursors(t, r, pass, want)
				out, err := r.replayTailPayloadsCandidate(context.Background(), pass)
				if err != nil || out == nil || !out.selectedLocked(r) {
					t.Fatal("actual later-stage completed, pending and unfinished payloads", pass, err)
				}
				if out.group.physical != physical || out.data.count != tailFiles || !out.data.unacknowledged ||
					out.data.incomplete != incomplete || out.data.unverifiedPartial || (out.data.attemptResult != "") != (mode == "member-tls") {
					t.Fatal("later-stage physical and unacknowledged tail expectations", mode, pass)
				}
			}
			if result, err := r.finishCatalogCandidate(context.Background()); err != nil || result == nil || !result.selectedLocked(r) {
				t.Fatal("actual C1 failed to close all earlier and current original names/metadata", err)
			}
			if r.confirmed != confirmed || o.last != ack || o.pin != pin || m.actual != work || s.clock.read != clocks || s.stored != held ||
				s.activeMeter.Load() != nil || !archive.shapeLocked(r, archive.checkpoint) || archive.order.consumed != 1 || archive.order.sent != 1 ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("read-only later-stage inspection changed the original lifetime or manufactured an ACK")
			}
		})
	}
}

// Normal owned TLS plus the actual original owned clock witnesses a successful
// S inside a positive, pre-H0 UNIT tail. This is a real test path, not a
// worst-case CPU/host qualification or a source-derived production certificate.
func TestProviderHistoryLaterStageSuccessfulSettlementHasAnActualPositiveNativeTail(t *testing.T) {
	_, capture := providerFrontierBoundsFixture()
	rows := providerHistoryDispatchBounds{
		member: candidateWorkVector{0: 1, 1: uint64(capture.WireBytes + 1), 2: 4 << 20, 9: 128 << 10, 10: 1 << 48},
		// Select a finite positive UNIT tail before H0 for full retained replay
		// with the race detector; the actual elapsed-tail assertion stays below.
		closure: candidateWorkVector{2: 32 << 20, 9: 1 << 20, 10: 1 << 50}, memberClocks: 2, closureClocks: 2, tailMS: 60000,
	}
	o := historyMultistageOwnerWithRows(t, 2, &rows)
	original, held, ceiling := o.parent.original, o.store.stored, o.parent.clockCeiling
	native := o.parent.clock
	for stage := uint64(1); stage <= 2; stage++ {
		dispatch := historyMultistageDispatch(t, o, stage)
		historyRetainNormalPrefix(t, o, dispatch, 1)
		if o.parent.clock != native || o.store.clock != native {
			t.Fatal("both stages must retain the original native backend", stage)
		}
		native.mu.Lock()
		memberEnd, beforeReads := native.last, native.read
		native.mu.Unlock()
		beforeWork := o.meter.actual
		out, err := o.retainSettlementCandidate(context.Background(), dispatch)
		if err != nil || out == nil {
			t.Fatal("ordinary successful retained S", stage, err)
		}
		native.mu.Lock()
		final, reads := native.last, native.read
		native.mu.Unlock()
		if reads != beforeReads+2 || o.parent.clock != native || o.parent.original != original || o.parent.clockCeiling != ceiling ||
			final.ClockID != original.ClockID || final.Epoch != original.Epoch || o.parent.dispatchBounds.tailMS != rows.tailMS ||
			!candidateLedgerClockAdvance(providerClockDiagnostic(memberEnd), providerClockDiagnostic(final)) ||
			!providerHistoryGroupClockFits(out.ack.claim.frame.clock, providerClockDiagnostic(final)) || !o.clockFits(final) ||
			final.MonotonicMS-memberEnd.MonotonicMS < 0 || uint64(final.MonotonicMS-memberEnd.MonotonicMS) > rows.tailMS ||
			o.meter.actual[0] != beforeWork[0] || o.meter.actual[1] != beforeWork[1] || o.store.stored != held ||
			out.ack != o.last || o.parent.recovery.confirmed != out.ack.claim.checkpoint || o.store.activeMeter.Load() != nil || o.failed {
			t.Fatal("successful S did not retain its actual positive original native tail", stage)
		}
		if stage < 2 && o.advanceStageCandidate(context.Background(), out.ack) != nil {
			t.Fatal("genuine next-stage transition after actual S tail")
		}
	}
}
