package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func historyRecoveryArchivedSettlement(t *testing.T, count int) (*providerHistoryGroupOwner, *providerHistoryGroupDispatch, *providerHistoryRecoveryCheckpoint) {
	t.Helper()
	o, d := historyRecoveryGroupFixture(t, count)
	historyRetainNormalPrefix(t, o, d, count)
	if out, err := o.retainSettlementCandidate(context.Background(), d); err != nil || out == nil {
		t.Fatal("genuine original acknowledged S", err)
	}
	c := o.parent.recovery.confirmed
	if c.frame.phase != "settle" || c.memberArchive == nil || !c.memberArchive.shapeLocked(o.parent.recovery, c) || c.ack != o.last {
		t.Fatal("original S did not retain its exact member archive")
	}
	return o, d, c
}

func TestProviderHistoryRecoveryArchiveSlotsAreBoundedBeforeOpen(t *testing.T) {
	for _, stages := range []uint64{0, 1, 2, (1 << 20) / 3, (1<<20)/3 + 1, ^uint64(0)} {
		slots, ok := providerHistoryRecoveryArchiveSlots(stages)
		valid := stages > 0 && stages <= (1<<20)/3
		if ok != valid || ok && slots != stages*providerHistoryRecoveryArchiveSlotBytes || !ok && slots != 0 {
			t.Fatal("fixed original archive slot bound", stages, slots, ok)
		}
	}
	s := providerHistoryParentScalarFixture(t)
	b, h := s.config.bounds, *s.config.bounds.history
	slots, ok := providerHistoryRecoveryArchiveSlots(h.stages)
	if !ok || slots > h.resident || !h.valid(b) {
		t.Fatal("genuine pre-H0 descriptor slot prerequisite")
	}
	h.resident = slots - 1
	h.derivation.snapshot = h // Test-only derivation; no opener or admission.
	if h.valid(b) {
		t.Fatal("archive descriptors fit only by exceeding the original resident row")
	}
	if json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryMemberArchive{}) == nil {
		t.Fatal("serialized DATA selected original stage members")
	}
}

func TestProviderHistoryRecoveryArchivePrecedesSuccessfulPendingAndFailedSWrites(t *testing.T) {
	for _, mode := range []string{"successful", "pending", "failed-empty", "failed-prefix"} {
		t.Run(mode, func(t *testing.T) {
			o, d := historyRecoveryGroupFixture(t, 2)
			r, s := o.parent.recovery, o.store
			completed := 2
			switch mode {
			case "failed-empty":
				completed = 0
			case "failed-prefix":
				completed = 1
			}
			historyRetainNormalPrefix(t, o, d, completed)
			consumption, ack, held := r.consumption, o.last, s.stored
			reached := false
			var selected *providerHistoryRecoveryMemberArchive
			syncFile := s.evidence.syncFile
			s.evidence.syncFile = func(file *os.File) error {
				if err := syncFile(file); err != nil {
					return err
				}
				claim := s.frontierFiles.historyPending
				if reached || claim == nil || claim.frame.phase != "settle" || filepath.Base(file.Name()) != providerHistoryStoredName(claim.inventory.entries[0], nil) {
					return nil
				}
				n := claim.checkpoint
				before, clocks := o.meter.actual, s.clock.read
				if n == nil || n.memberArchive == nil || n.memberArchive != claim.memberArchive || !n.memberArchive.shapeLocked(r, n) ||
					n.memberArchive.count != completed || n.memberArchive.consumption != consumption || n.memberArchive.members != r.members ||
					r.prepareMemberArchiveLocked(claim, n) != nil ||
					r.confirmed != consumption || o.last != ack || n.ack == ack || r.used || r.inputFiles != 0 || r.meter.actual != (candidateWorkVector{}) ||
					o.meter.actual != before || s.clock.read != clocks || s.stored != held {
					t.Fatal("first physical S write preceded its original non-authorizing archive")
				}
				selected, reached = n.memberArchive, true
				if mode == "pending" {
					return errProviderIntegration // Owned local sync uncertainty only.
				}
				return nil
			}
			if completed < 2 {
				member := d.coordinator.ticket.table.members[completed]
				conn := providerIntegrationOwnedTLS(t, member.request.body, `{}`)
				if out, err := o.retainMemberCandidateWithConnection(context.Background(), d, completed, conn); err == nil || out != nil || !o.failed {
					t.Fatal("genuine terminal failed-S precursor", err)
				}
			} else {
				out, err := o.retainSettlementCandidate(context.Background(), d)
				if (mode == "pending") != (err != nil) || (mode == "pending") != (out == nil) {
					t.Fatal("original successful or unacknowledged S", mode, err)
				}
			}
			s.evidence.syncFile = syncFile
			if !reached || selected == nil || !selected.shapeLocked(r, selected.checkpoint) || s.stored != held || r.used || s.activeMeter.Load() != nil {
				t.Fatal("S did not preserve its exact original archive")
			}
			if mode == "pending" {
				if r.pending != selected.checkpoint || r.confirmed != consumption || o.last != ack || !o.failed || r.pending.ack == o.last {
					t.Fatal("archive promoted an uncertain S or renewed its original execution")
				}
			} else if r.confirmed != selected.checkpoint || r.confirmed.ack != o.last || r.pending != nil || r.confirmed.failedS != (completed < 2) {
				t.Fatal("original S acknowledgment differs from the selected archive")
			}
		})
	}
}

func TestProviderHistoryRecoveryArchiveSelectsItsOwnOriginalsWithoutCurrentStageAliases(t *testing.T) {
	o, _, c := historyRecoveryArchivedSettlement(t, 2)
	r, s := o.parent.recovery, o.store
	archive := c.memberArchive
	o.store.mu.Lock()
	o.failLocked()
	o.store.mu.Unlock()
	old, clocks, held, work := o.meter.actual, s.clock.read, s.stored, r.meter.actual
	// Pure role-selection isolation only. This does not execute or claim a
	// healthy stage transition; all original current slots are restored before
	// the real read-only lane can be entered.
	members, order, consumption, dispatch, completed := r.members, r.order, r.consumption, r.originalDispatch, r.completed
	r.members, r.order, r.consumption, r.originalDispatch, r.completed = [providerHistoryGroupMembers]*providerHistoryRecoveryMemberComplete{}, nil, nil, nil, 0
	for i := range archive.count {
		if r.inputMemberIntentLocked(c, i) != archive.members[i].capture.intent || r.settledMemberLocked(c, i) != archive.members[i] {
			t.Fatal("archived S consulted current member/order/C/dispatch aliases")
		}
	}
	r.members, r.order, r.consumption, r.originalDispatch, r.completed = members, order, consumption, dispatch, completed
	if o.meter.actual != old || s.clock.read != clocks || s.stored != held || r.meter.actual != work || r.used || r.inputFiles != 0 {
		t.Fatal("pure archived selection performed or renewed an original operation")
	}
	if r.beginInspectionCandidate(context.Background(), o) != nil {
		t.Fatal("restored genuine original stage could not enter its read-only lane")
	}
	for i := range archive.count {
		for _, kind := range []providerHistoryRecoveryFileKind{providerHistoryRecoveryMemberResponse, providerHistoryRecoveryMemberTLS} {
			input, err := r.readCheckpointInputCandidate(context.Background(), c, kind, i)
			if err != nil || input == nil || input.state != providerHistoryRecoveryWholeInput || len(input.raw) == 0 {
				t.Fatal("actual original S spool under its closed archived role", i, kind, err)
			}
		}
	}
	if r.inputFiles != 4 || o.meter.actual != old || s.clock.read != clocks || s.stored != held || s.activeMeter.Load() != nil {
		t.Fatal("archived spool reads escaped the disjoint original read-only row")
	}
}

func TestProviderHistoryRecoveryArchiveRejectsDetachedOriginalStageFactsWithoutEffects(t *testing.T) {
	o, _, c := historyRecoveryArchivedSettlement(t, 1)
	r, s, a := o.parent.recovery, o.store, c.memberArchive
	s.mu.Lock()
	o.failLocked()
	s.mu.Unlock()
	for _, mode := range []string{"nil", "copy", "claim", "checkpoint", "consumption", "dispatch", "order", "stage", "count", "unused", "nil-member", "nil-capture", "nil-intent", "complete", "capture", "intent", "result", "files"} {
		t.Run(mode, func(t *testing.T) {
			originalArchive, originalClaim, data := c.memberArchive, c.claim.memberArchive, *a
			sealed := a.members[0]
			complete, capture, intent, attempt := *sealed, *sealed.capture, *sealed.capture.intent, *sealed.capture.intent.attempt
			old, work, clocks, held := o.meter.actual, r.meter.actual, s.clock.read, s.stored
			switch mode {
			case "nil":
				c.memberArchive = nil
			case "copy":
				copy := *a
				copy.owner, c.memberArchive = &copy, &copy
			case "claim":
				c.claim.memberArchive = nil
			case "checkpoint":
				a.checkpoint = nil
			case "consumption":
				copy := *a.consumption
				copy.owner, a.consumption = &copy, &copy
			case "dispatch":
				copy := *a.dispatch
				copy.owner, a.dispatch = &copy, &copy
			case "order":
				copy := *a.order
				copy.owner, a.order = &copy, &copy
			case "stage":
				a.stage++
			case "count":
				a.count++
			case "unused":
				a.members[providerHistoryGroupMembers-1] = sealed
			case "nil-member":
				a.members[0] = nil
			case "nil-capture":
				sealed.capture = nil
			case "nil-intent":
				sealed.capture.intent = nil
			case "complete":
				copy := *sealed
				copy.owner, a.members[0] = &copy, &copy
			case "capture":
				copy := *sealed.capture
				copy.owner, sealed.capture = &copy, &copy
			case "intent":
				copy := *sealed.capture.intent
				copy.owner, sealed.capture.intent = &copy, &copy
			case "result":
				sealed.result = admissionTestDigest("f")
			case "files":
				sealed.capture.intent.attempt.files = 1
			}
			if r.checkpointSelectionLocked(o) || r.settledMemberLocked(c, 0) != nil || r.beginInspectionCandidate(context.Background(), o) == nil ||
				r.used || r.active || r.inputFiles != 0 || o.meter.actual != old || r.meter.actual != work || s.clock.read != clocks || s.stored != held {
				t.Fatal("detached archived stage selected originals or performed an effect", mode)
			}
			c.memberArchive, c.claim.memberArchive, *a = originalArchive, originalClaim, data
			*intent.attempt = attempt
			*capture.intent = intent
			*complete.capture = capture
			*sealed = complete
			if !a.shapeLocked(r, c) || !r.checkpointSelectionLocked(o) {
				t.Fatal("genuine archived positive control was not restored", mode)
			}
		})
	}
	if r.beginInspectionCandidate(context.Background(), o) != nil {
		t.Fatal("genuine original archive could not enter after detached selections")
	}
}

func TestProviderHistoryRecoveryArchiveReconstructsBothActualPassesAndCatalog(t *testing.T) {
	o, _, c := historyRecoveryArchivedSettlement(t, 2)
	r, want := historyRecoveryGroupStart(t, o)
	old, clocks, held, pin, ack := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last
	if _, err := r.beginCatalogCandidate(context.Background()); err != nil {
		t.Fatal("original C0 before archived payloads", err)
	}
	historyRecoveryCatalogModels(t, r, want)
	for pass := range 2 {
		g := r.groupModels[pass]
		if g.physical != c || len(g.records) != 3 || g.ledger.phase != "settled" || g.ledger.stage != c.frame.stage ||
			g.memberCheckpoint != c.previous || g.memberTicket == nil || g.memberConsumption == nil || g.retention != o.retention {
			t.Fatal("archived spools were not independently replayed with their own actual T/C", pass)
		}
	}
	if out, err := r.finishCatalogCandidate(context.Background()); err != nil || out == nil || !out.selectedLocked(r) ||
		o.meter.actual != old || o.store.clock.read != clocks || o.store.stored != held || o.pin != pin || o.last != ack || r.confirmed != c {
		t.Fatal("archived payload/catalog closure changed the original execution or ACK", err)
	}
}

func TestProviderHistoryRecoveryArchiveRejectsMissingWrongPhaseAndOutOfPrefixSelections(t *testing.T) {
	o, _, c := historyRecoveryArchivedSettlement(t, 1)
	r, a := o.parent.recovery, c.memberArchive
	old, work, clocks, held := o.meter.actual, r.meter.actual, o.store.clock.read, o.store.stored
	var absent *providerHistoryRecoveryMemberArchive
	if absent.shapeLocked(r, c) || a.shapeLocked(nil, c) || r.checkpointMemberArchiveShapeLocked(nil) ||
		r.settledMemberLocked(nil, 0) != nil || r.settledMemberLocked(c, -1) != nil || r.settledMemberLocked(c, 1) != nil ||
		r.settledMemberLocked(r.initial, 0) != nil || r.settledMemberLocked(c.previous, 0) != nil ||
		r.inputMemberIntentLocked(c, -1) != nil || r.inputMemberIntentLocked(c, providerHistoryGroupMembers) != nil ||
		r.inputMemberIntentLocked(c.previous.previous, 0) != nil || r.inputMemberIntentLocked(nil, 0) != nil {
		t.Fatal("missing, wrong-phase or out-of-prefix DATA selected an archived spool")
	}
	// Both T and C must have no S archive; adding even the genuine S archive
	// does not turn an earlier phase into a settled member selector.
	for _, other := range []*providerHistoryRecoveryCheckpoint{r.initial, c.previous.previous, c.previous} {
		if !r.checkpointMemberArchiveShapeLocked(other) {
			t.Fatal("genuine non-S empty archive control")
		}
		other.memberArchive = a
		if r.checkpointMemberArchiveShapeLocked(other) {
			t.Fatal("archive assigned to a different typed phase")
		}
		other.memberArchive = nil
	}
	if r.settledMemberLocked(c, 0) != a.members[0] || !a.shapeLocked(r, c) ||
		o.meter.actual != old || r.meter.actual != work || o.store.clock.read != clocks || o.store.stored != held || r.used || r.inputFiles != 0 {
		t.Fatal("role rejection performed work or prevented the genuine archive control")
	}
}
