package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func historyRecoveryGroupStart(t *testing.T, o *providerHistoryGroupOwner) (*providerHistoryRecoveryOwner, map[string]providerIntegrationObjectRef) {
	t.Helper()
	s, r := o.store, o.parent.recovery
	want := make(map[string]providerIntegrationObjectRef, len(s.objects))
	for key, ref := range s.objects {
		want[key] = ref
	}
	if len(want) == 0 || uint64(len(want)) != r.initialDescriptor.objectCount {
		t.Fatal("genuine original initial object references")
	}
	s.mu.Lock()
	o.failLocked()
	// Remove only failed producer caches. Original provenance pointers and
	// sealed descriptor values remain; no live reader can use these caches.
	s.objects, s.frontierReplay, s.bindings = nil, nil, nil
	o.coordinator, o.custodian = nil, nil
	if o.dispatch != nil {
		o.dispatch.coordinator, o.dispatch.custodian = nil, nil
	}
	for c := r.confirmed; c != nil && c != r.initial; c = c.previous {
		c.claim.coordinator, c.claim.purpose, c.claim.members, c.claim.upper = nil, "", 0, providerHistoryGroupUpper{}
	}
	if c := r.pending; c != nil {
		c.claim.coordinator, c.claim.purpose, c.claim.members, c.claim.upper = nil, "", 0, providerHistoryGroupUpper{}
	}
	s.mu.Unlock()
	if r.beginInspectionCandidate(context.Background(), o) != nil {
		t.Fatal("genuine terminal original group recovery")
	}
	return r, want
}

func historyRecoveryGroupInitial(t *testing.T, r *providerHistoryRecoveryOwner, pass int, want map[string]providerIntegrationObjectRef) {
	t.Helper()
	historyRecoveryModelPrefix(t, r, pass, want)
	if out, err := r.replayInitialModelCandidate(context.Background(), pass); err != nil || out == nil || !out.selectedLocked(r) {
		t.Fatal("genuine initial full semantic model", pass, err)
	}
}

func historyRecoveryGroupFixture(t *testing.T, count int) (*providerHistoryGroupOwner, *providerHistoryGroupDispatch) {
	t.Helper()
	return historyMemberDispatchFixtureWithRecoveryGroupSelection(t, count, nil, true, true, true)
}

func TestProviderHistoryRecoveryGroupReconstructsConsumedSuccessfulAndFailedPrefixesTwice(t *testing.T) {
	for _, mode := range []string{"consumed", "settled-one", "settled-several", "settled-maximum", "failed-zero", "failed-one", "failed-two"} {
		t.Run(mode, func(t *testing.T) {
			count, completed := 1, 0
			switch mode {
			case "settled-several", "failed-zero", "failed-one", "failed-two":
				count = 3
			case "settled-maximum":
				count = providerHistoryGroupMembers
			}
			o, d := historyRecoveryGroupFixture(t, count)
			s := o.store
			if mode != "consumed" {
				switch mode {
				case "failed-one":
					completed = 1
				case "failed-two":
					completed = 2
				case "settled-one", "settled-several", "settled-maximum":
					completed = count
				}
				historyRetainNormalPrefix(t, o, d, completed)
				if completed < count {
					member := d.coordinator.ticket.table.members[completed]
					conn := providerIntegrationOwnedTLS(t, member.request.body, `{}`)
					if out, err := o.retainMemberCandidateWithConnection(context.Background(), d, completed, conn); err == nil || out != nil || !o.failed {
						t.Fatal("genuine retained failed-S precursor", err)
					}
				} else if out, err := o.retainSettlementCandidate(context.Background(), d); err != nil || out == nil {
					t.Fatal("genuine successful S precursor", err)
				}
			}
			r, want := historyRecoveryGroupStart(t, o)
			if r.bounds.maximumFiles != 1024 || r.bounds.maximumBytes != 16<<20 || r.bounds.work[2] != 128<<20 || r.bounds.work[10] != 1<<48 {
				t.Fatal("separate original pre-H0 full-group UNIT row")
			}
			old, reads, held, pin, ack, confirmed := o.meter.actual, s.clock.read, s.stored, o.pin, o.last, r.confirmed
			before := providerHistoryProvisionedOriginalBytes(t, s.config)
			var first *providerHistoryRecoveryGroupModel
			for pass := range 2 {
				historyRecoveryGroupInitial(t, r, pass, want)
				files, bytes, work := r.inputFiles, r.inputBytes, r.meter.actual
				out, err := r.replayGroupModelCandidate(context.Background(), pass)
				if err != nil || out == nil || !out.selectedLocked(r) || out.physical != confirmed || out.pin != confirmed.pin ||
					out.confirmedModel != confirmed.model || out.ledger.head != confirmed.model || out.tailState != 0 ||
					r.inputFiles <= files || r.inputBytes <= bytes || r.meter.actual[10] <= work[10] || out.retention.physical == 0 {
					t.Fatal("actual independent group model, exact roots and cumulative original row", mode, pass, err)
				}
				if mode == "consumed" {
					if len(out.records) != 2 || out.ledger.phase != "consumed" || out.ledger.ticket == nil || out.ledger.consumption == nil || out.ledger.ticket.table.count != count {
						t.Fatal("T/C semantics skipped or changed actual request cardinality")
					}
				} else if len(out.records) != 3 || out.ledger.ticket != nil || out.ledger.consumption != nil ||
					confirmed.failedS && (!out.ledger.failed || !out.ledger.terminal || out.ledger.phase != "failed") ||
					!confirmed.failedS && (out.ledger.failed || out.ledger.terminal || out.ledger.phase != "settled") {
					t.Fatal("successful/failed S lost its exact typed terminal semantics")
				}
				if out.retention != o.retention {
					t.Fatal("exact actual main and member spool retention differs from original whole group")
				}
				if pass == 0 {
					first = out
				} else if first == out || first.ledger == out.ledger || first.initial == out.initial || &first.records[0] == &out.records[0] ||
					first.ledger.head != out.ledger.head || first.retention != out.retention ||
					first.ledger.ticket != nil && first.ledger.ticket == out.ledger.ticket || first.records[0].inventory == out.records[0].inventory {
					t.Fatal("two actual group passes share mutable reconstruction or disagree")
				}
			}
			if o.meter.actual != old || s.clock.read != reads || s.stored != held || o.pin != pin || o.last != ack || r.confirmed != confirmed || r.pending != nil ||
				s.activeMeter.Load() != nil || s.objects != nil || o.coordinator != nil || d.coordinator != nil || !r.active || r.failed || r.closed ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("group replay changed originals, promoted authority, used cached models or refunded old work")
			}
		})
	}
}

// This changes only context cancellation at an owned normal S rename. It does
// not manufacture wire traffic, bypass a host or retry the original execution.
func historyRecoveryUnacknowledgedS(t *testing.T) (*providerHistoryGroupOwner, *providerHistoryGroupDispatch) {
	t.Helper()
	o, d, _ := historyRecoveryUnacknowledgedMembersS(t, 1)
	return o, d
}

func historyRecoveryUnacknowledgedMembersS(t *testing.T, count int) (*providerHistoryGroupOwner, *providerHistoryGroupDispatch, providerHistoryStorageCharge) {
	t.Helper()
	o, d := historyRecoveryGroupFixture(t, count)
	confirmedRetention := o.retention
	historyRetainNormalPrefix(t, o, d, count)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	disk, reached := o.store.custody, false
	syncFile := disk.syncFile
	disk.syncFile = func(file *os.File) error {
		if err := syncFile(file); err != nil {
			return err
		}
		c := o.store.frontierFiles.historyPending
		info, err := file.Stat()
		if err != nil || c == nil || c.frame.phase != "settle" || !info.IsDir() {
			return err
		}
		head, err := os.ReadFile(filepath.Join(o.store.config.custodyDirectory, "HEAD"))
		if err != nil {
			return err
		}
		if string(head) == c.highwater {
			reached = true
			cancel()
		}
		return nil
	}
	if out, err := o.retainSettlementCandidate(ctx, d); err == nil || out != nil || !reached || !o.failed || o.parent.recovery.pending == nil ||
		o.parent.recovery.confirmed.frame.phase != "consume" || o.parent.recovery.pending.frame.phase != "settle" {
		t.Fatal("genuine unacknowledged physical S under original C", err)
	}
	disk.syncFile = syncFile
	return o, d, confirmedRetention
}

func TestProviderHistoryRecoveryGroupKeepsCompleteOrPartialPendingSUnacknowledged(t *testing.T) {
	for _, mode := range []string{"complete", "highwater-empty", "intent-empty", "inventory-missing", "object-empty", "response-missing", "tls-empty"} {
		t.Run(mode, func(t *testing.T) {
			o, _ := historyRecoveryUnacknowledgedS(t)
			r := o.parent.recovery
			pending, confirmed, ack, pin := r.pending, r.confirmed, o.last, o.pin
			root, name, kind, state := r.config.custodyDirectory, "", providerHistoryRecoveryFileKind(0), providerHistoryRecoveryInputState(0)
			switch mode {
			case "highwater-empty":
				name, kind, state = providerCustodyName("highwater", pending.pin.ordinal), providerHistoryRecoveryHighwater, providerHistoryRecoveryPartialInput
			case "intent-empty":
				name, kind, state = providerCustodyName("intent", pending.pin.sequence), providerHistoryRecoveryIntent, providerHistoryRecoveryPartialInput
			case "inventory-missing":
				root, name, kind, state = r.config.evidenceDirectory, providerHistoryInventoryName(pending.inventory.digest, nil), providerHistoryRecoveryInventory, providerHistoryRecoveryMissingInput
			case "object-empty":
				root, name, kind, state = r.config.evidenceDirectory, providerHistoryStoredName(pending.inventory.entries[0], nil), providerHistoryRecoveryObject, providerHistoryRecoveryPartialInput
			case "response-missing":
				root, name, kind, state = r.config.evidenceDirectory, r.members[0].capture.intent.responseName, providerHistoryRecoveryMemberResponse, providerHistoryRecoveryMissingInput
			case "tls-empty":
				root, name, kind, state = r.config.evidenceDirectory, r.members[0].capture.intent.tlsName, providerHistoryRecoveryMemberTLS, providerHistoryRecoveryPartialInput
			}
			if name != "" {
				path := filepath.Join(root, name)
				if raw, err := os.ReadFile(path); err != nil || len(raw) == 0 {
					t.Fatal("actual owned nonempty pending original", err)
				}
				if state == providerHistoryRecoveryMissingInput {
					if os.Remove(path) != nil {
						t.Fatal("owned missing-file unit scenario")
					}
				} else if os.WriteFile(path, nil, 0600) != nil {
					t.Fatal("owned empty-file unit scenario")
				}
			}
			r, want := historyRecoveryGroupStart(t, o)
			for pass := range 2 {
				historyRecoveryGroupInitial(t, r, pass, want)
				old, reads, held := o.meter.actual, o.store.clock.read, o.store.stored
				out, err := r.replayGroupModelCandidate(context.Background(), pass)
				if err != nil || out == nil || !out.selectedLocked(r) || out.confirmedModel != confirmed.model || r.confirmed != confirmed || r.pending != pending ||
					o.last != ack || o.pin != pin || o.store.frontierFiles.pin != pin || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held ||
					o.store.activeMeter.Load() != nil || out.tailState != state || out.tailKind != kind {
					t.Fatal("pending physical originals changed logical C, ACK, reservation or partial classification", mode, err)
				}
				if mode == "complete" {
					if out.physical != pending || out.pin != pending.pin || out.ledger.head != pending.model || out.ledger.phase != "settled" || len(out.records) != 3 {
						t.Fatal("complete unacknowledged S was neither independently reconstructed nor kept separate")
					}
				} else if out.physical != confirmed || out.pin != confirmed.pin || out.ledger.head != confirmed.model || out.ledger.phase != "consumed" || len(out.records) != 2 {
					t.Fatal("partial pending original advanced the physical semantic prefix")
				}
			}

		})
	}
}

func TestProviderHistoryRecoveryGroupPublishesRetentionOnlyForWholePendingSettlement(t *testing.T) {
	for _, subject := range []struct {
		name    string
		count   int
		ordinal int
		partial string
	}{
		{name: "two-complete", count: 2},
		{name: "two-second-response", count: 2, ordinal: 1, partial: "response"},
		{name: "two-second-tls", count: 2, ordinal: 1, partial: "tls"},
		{name: "three-complete", count: 3},
		{name: "three-second-response", count: 3, ordinal: 1, partial: "response"},
		{name: "three-second-tls", count: 3, ordinal: 1, partial: "tls"},
		{name: "three-last-response", count: 3, ordinal: 2, partial: "response"},
		{name: "three-last-tls", count: 3, ordinal: 2, partial: "tls"},
	} {
		t.Run(subject.name, func(t *testing.T) {
			o, _, confirmedRetention := historyRecoveryUnacknowledgedMembersS(t, subject.count)
			r := o.parent.recovery
			confirmed, pending, ack, pin := r.confirmed, r.pending, o.last, o.pin
			physicalRetention := o.retention
			if physicalRetention == confirmedRetention {
				t.Fatal("genuine pending S must retain additional originals")
			}
			kind, state := providerHistoryRecoveryFileKind(0), providerHistoryRecoveryInputState(0)
			switch subject.partial {
			case "response":
				kind, state = providerHistoryRecoveryMemberResponse, providerHistoryRecoveryMissingInput
				path := filepath.Join(r.config.evidenceDirectory, r.members[subject.ordinal].capture.intent.responseName)
				if raw, err := os.ReadFile(path); err != nil || len(raw) == 0 || os.Remove(path) != nil {
					t.Fatal("owned nonempty later response removal", err)
				}
			case "tls":
				kind, state = providerHistoryRecoveryMemberTLS, providerHistoryRecoveryPartialInput
				path := filepath.Join(r.config.evidenceDirectory, r.members[subject.ordinal].capture.intent.tlsName)
				if raw, err := os.ReadFile(path); err != nil || len(raw) == 0 || os.WriteFile(path, nil, 0600) != nil {
					t.Fatal("owned nonempty later TLS truncation", err)
				}
			}
			r, want := historyRecoveryGroupStart(t, o)
			old, reads, held := o.meter.actual, o.store.clock.read, o.store.stored
			before := providerHistoryProvisionedOriginalBytes(t, o.store.config)
			for pass := range 2 {
				historyRecoveryGroupInitial(t, r, pass, want)
				files, work := r.inputFiles, r.meter.actual
				out, err := r.replayGroupModelCandidate(context.Background(), pass)
				if err != nil || out == nil || !out.selectedLocked(r) || out.confirmedModel != confirmed.model ||
					out.tailKind != kind || out.tailState != state || r.inputFiles <= files || r.meter.actual[10] <= work[10] {
					t.Fatal("actual fresh replay lost its pending tail or cumulative observation", pass, err)
				}
				if subject.partial == "" {
					if out.physical != pending || out.pin != pending.pin || out.ledger.head != pending.model || out.ledger.phase != "settled" ||
						len(out.records) != 3 || out.retention != physicalRetention {
						t.Fatal("whole pending S did not publish its exact physical retention")
					}
				} else if out.physical != confirmed || out.pin != confirmed.pin || out.ledger.head != confirmed.model || out.ledger.phase != "consumed" ||
					len(out.records) != 2 || out.retention != confirmedRetention || out.tailIndex != subject.ordinal {
					t.Fatal("later partial spool published retention beyond the last complete C prefix")
				}
			}
			if r.confirmed != confirmed || r.pending != pending || o.last != ack || o.pin != pin || o.retention != physicalRetention ||
				o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.store.activeMeter.Load() != nil ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, o.store.config)) {
				t.Fatal("private prefix accounting refunded actual observation, hold, files or original ACK")
			}
		})
	}
}

func TestProviderHistoryRecoveryGroupRefusesEntryCopiesOrderAndRepetitionBeforeObservation(t *testing.T) {
	o, _ := historyRecoveryGroupFixture(t, 1)
	r, want := historyRecoveryGroupStart(t, o)
	work := r.meter.actual
	if out, err := r.replayGroupModelCandidate(context.Background(), 0); err == nil || out != nil || r.meter.actual != work || !r.active {
		t.Fatal("group model without actual initial semantics")
	}
	for pass := range 2 {
		historyRecoveryGroupInitial(t, r, pass, want)
	}
	for _, mode := range []string{"negative-pass", "excess-pass", "second-before-first", "nil-initial", "copied-initial", "incomplete-initial"} {
		t.Run(mode, func(t *testing.T) {
			original, pass := r.initialModels[0], 0
			switch mode {
			case "negative-pass":
				pass = -1
			case "excess-pass":
				pass = 2
			case "second-before-first":
				pass = 1
			case "nil-initial":
				r.initialModels[0] = nil
			case "copied-initial":
				copy := *original
				copy.owner = original
				r.initialModels[0] = &copy
			case "incomplete-initial":
				original.complete = false
			}
			work, files, bytes := r.meter.actual, r.inputFiles, r.inputBytes
			out, err := r.replayGroupModelCandidate(context.Background(), pass)
			r.initialModels[0], original.complete = original, true
			if err == nil || out != nil || r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes || !r.active || r.closed {
				t.Fatal("unselected source reached observation or closed the lane", mode, err)
			}
		})
	}
	positive, err := r.replayGroupModelCandidate(context.Background(), 0)
	if err != nil || positive == nil || !positive.selectedLocked(r) {
		t.Fatal("positive control after isolated entry refusals", err)
	}
	copy := *positive
	copy.owner = &copy
	work = r.meter.actual
	if copy.selectedLocked(r) {
		t.Fatal("copied DATA replaced the registered group model")
	}
	if out, err := r.replayGroupModelCandidate(context.Background(), 0); err == nil || out != nil || r.meter.actual != work || !r.active {
		t.Fatal("repeated group replay reset its row", err)
	}
	if json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryGroupModel{}) == nil || json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryAdmissionDescriptor{}) == nil {
		t.Fatal("JSON manufactured a selected model or original admission descriptor")
	}
}

func TestProviderHistoryRecoveryGroupRejectsChangedConfirmedOriginalsAndSealedInputs(t *testing.T) {
	for _, mode := range []string{"highwater", "intent", "commit", "frontier", "pending", "inventory", "requests", "ticket", "consumption", "purpose", "upper"} {
		t.Run(mode, func(t *testing.T) {
			o, _ := historyRecoveryGroupFixture(t, 1)
			r, want := historyRecoveryGroupStart(t, o)
			historyRecoveryGroupInitial(t, r, 0, want)
			c := r.confirmed.previous
			root, name := r.config.custodyDirectory, ""
			switch mode {
			case "highwater":
				name = providerCustodyName("highwater", c.pin.ordinal)
			case "intent", "commit", "frontier":
				name = providerCustodyName(mode, c.pin.sequence)
			case "pending":
				root, name = r.config.evidenceDirectory, providerCustodyName("history-pending", c.pin.sequence)
			case "inventory":
				root, name = r.config.evidenceDirectory, providerHistoryInventoryName(c.inventory.digest, nil)
			case "requests", "ticket":
				index := 0
				if mode == "ticket" {
					index = 1
				}
				root, name = r.config.evidenceDirectory, providerHistoryStoredName(c.inventory.entries[index], nil)
			case "consumption":
				root, name = r.config.evidenceDirectory, providerHistoryStoredName(r.confirmed.inventory.entries[0], nil)
			case "purpose":
				c.admission.purpose = "identity" // Private unit descriptor DATA only.
			case "upper":
				c.admission.upper.metadata++
			}
			if name != "" {
				path := filepath.Join(root, name)
				raw, err := os.ReadFile(path)
				if err != nil || len(raw) == 0 {
					t.Fatal("actual nonempty owned precursor", err)
				}
				for i := range raw {
					raw[i] = '!'
				}
				if os.WriteFile(path, raw, 0600) != nil {
					t.Fatal("inert owned original replacement")
				}
			}
			old, reads, held, ack, pin := o.meter.actual, o.store.clock.read, o.store.stored, o.last, o.pin
			out, err := r.replayGroupModelCandidate(context.Background(), 0)
			if err == nil || out != nil || r.groupModels[0] != nil || r.active || !r.failed || !r.closed || o.meter.actual != old || o.store.clock.read != reads ||
				o.store.stored != held || o.last != ack || o.pin != pin || o.store.activeMeter.Load() != nil {
				t.Fatal("changed confirmed original or sealed selector produced a partial model or renewed authority", mode, err)
			}
		})
	}
}

func TestProviderHistoryRecoveryGroupCancellationStopsBeforeObservation(t *testing.T) {
	for i := range 2 {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			o, _ := historyRecoveryGroupFixture(t, 1)
			r, want := historyRecoveryGroupStart(t, o)
			historyRecoveryGroupInitial(t, r, 0, want)
			var ctx context.Context
			if i == 1 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			}
			work, files, bytes, old, reads := r.meter.actual, r.inputFiles, r.inputBytes, o.meter.actual, o.store.clock.read
			out, err := r.replayGroupModelCandidate(ctx, 0)
			if err == nil || out != nil || r.groupModels[0] != nil || r.active || !r.failed || !r.closed || r.meter.actual != work ||
				r.inputFiles != files || r.inputBytes != bytes || o.meter.actual != old || o.store.clock.read != reads {
				t.Fatal("cancelled valid group replay reached an effect or reset its observer", err)
			}
		})
	}
}

func historyRecoveryBeforeAdmission(t *testing.T) (*providerHistoryGroupOwner, providerHistoryGroupUpper) {
	t.Helper()
	_, capture := providerFrontierBoundsFixture()
	capture.Timeout = 10 * time.Second
	s := providerHistoryProvisionedInspectionFixture(t, providerScanFixtureSelection{capture: &capture,
		pin: true, pinLargeObject: true, historyProgram: true, historyGroupWrite: true, historyGroupConsume: true, historyGroupDispatch: true,
		historyRecoveryNamespace: true, historyRecoveryLegacy: true, historyRecoveryGroup: true})
	first, err := newProviderFirstOperationCandidate(s)
	if err != nil {
		t.Fatal("original first operation", err)
	}
	conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	if out, err := first.executeCandidateWithConnection(context.Background(), conn); err != nil || out == nil {
		t.Fatal("original first complete exchange", err)
	}
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil {
		t.Fatal("original pre-T group", err)
	}
	upper := providerHistoryGroupUpper{physical: 8 << 20, retainedRaw: 4 << 20, payload: 2 << 20, metadata: 2 << 20, objects: 128, files: 256, clocks: 256}
	upper.execution[0], upper.execution[1], upper.execution[10] = 1, uint64(s.config.capture.WireBytes+1), 1<<48
	upper.custody[2], upper.custody[9], upper.custody[10] = 256<<20, 32<<20, 1<<58
	return o, upper
}

func TestProviderHistoryRecoveryGroupKeepsPendingAdmissionAndConsumptionUnacknowledged(t *testing.T) {
	for _, phase := range []string{"admit", "consume"} {
		for _, point := range []string{"first-object", "after-rename"} {
			t.Run(phase+"-"+point, func(t *testing.T) {
				o, upper := historyRecoveryBeforeAdmission(t)
				request := historyGroupSimple(t)
				if phase == "consume" {
					if ack, err := o.retainAdmissionCandidate(context.Background(), "selection", []*providerIntegrationRequest{request}, upper); err != nil || ack == nil {
						t.Fatal("genuine confirmed T before pending C", err)
					}
				}
				r, confirmed, ack, pin := o.parent.recovery, o.parent.recovery.confirmed, o.last, o.pin
				disk := o.store.evidence
				if point == "after-rename" {
					disk = o.store.custody
				}
				syncFile, reached := disk.syncFile, false
				disk.syncFile = func(file *os.File) error {
					if err := syncFile(file); err != nil {
						return err
					}
					c := o.store.frontierFiles.historyPending
					if c == nil || c.frame.phase != phase || reached {
						return nil
					}
					if point == "first-object" {
						if filepath.Base(file.Name()) != providerHistoryStoredName(c.inventory.entries[0], nil) {
							return nil
						}
					} else {
						info, err := file.Stat()
						if err != nil || !info.IsDir() {
							return err
						}
						head, err := os.ReadFile(filepath.Join(o.store.config.custodyDirectory, "HEAD"))
						if err != nil {
							return err
						}
						if string(head) != c.highwater {
							return nil
						}
					}
					reached = true
					return errProviderIntegration // Owned I/O failure only.
				}
				var out *providerHistoryGroupAcknowledgment
				var err error
				if phase == "admit" {
					out, err = o.retainAdmissionCandidate(context.Background(), "selection", []*providerIntegrationRequest{request}, upper)
				} else {
					out, err = o.retainConsumptionCandidate(context.Background(), ack)
				}
				disk.syncFile = syncFile
				if err == nil || out != nil || !reached || !o.failed || r.pending == nil || r.confirmed != confirmed || o.last != ack || o.pin != pin {
					t.Fatal("original failed write changed logical confirmation", err)
				}
				pending := r.pending
				r, want := historyRecoveryGroupStart(t, o)
				for pass := range 2 {
					historyRecoveryGroupInitial(t, r, pass, want)
					model, err := r.replayGroupModelCandidate(context.Background(), pass)
					if err != nil || model == nil || !model.selectedLocked(r) || model.confirmedModel != confirmed.model ||
						r.confirmed != confirmed || r.pending != pending || o.last != ack || o.pin != pin {
						t.Fatal("pending T/C changed logical confirmation", err)
					}
					if point == "after-rename" {
						if model.physical != pending || model.tailState != 0 || model.ledger.head != pending.model {
							t.Fatal("complete pending T/C not independently reconstructed")
						}
					} else if model.physical != confirmed || model.tailState != providerHistoryRecoveryMissingInput || model.tailKind != providerHistoryRecoveryHighwater {
						t.Fatal("incomplete pending T/C advanced the semantic prefix")
					}
				}

			})
		}
	}
}
