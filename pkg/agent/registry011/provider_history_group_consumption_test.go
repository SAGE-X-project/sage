package registry011

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func historyConsumptionFixture(t *testing.T, count int) (*providerHistoryGroupOwner, *providerHistoryGroupAcknowledgment) {
	t.Helper()
	return historyConsumptionFixtureWithCap(t, count, "", 0)
}

func historyConsumptionFixtureWithCap(t *testing.T, count int, dimension string, maximum uint64) (*providerHistoryGroupOwner, *providerHistoryGroupAcknowledgment) {
	t.Helper()
	upper := providerHistoryGroupUpper{physical: 8 << 20, retainedRaw: 4 << 20, payload: 128 << 10, metadata: 2 << 20, objects: 8, files: 64, clocks: 16}
	upper.execution[0], upper.execution[1], upper.execution[10] = uint64(count), uint64(count), 128
	upper.custody[2], upper.custody[9], upper.custody[10] = 16<<20, 16<<20, 1<<50
	// Select independent synthetic group inputs before this namespace's H0.
	// Prior-fixture diagnostics never replace its original backend or hold.
	switch dimension {
	case "physical":
		upper.physical = min(upper.physical, maximum)
		upper.retainedRaw = min(upper.retainedRaw, maximum)
		upper.payload = min(upper.payload, maximum)
		upper.metadata = min(upper.metadata, maximum)
	case "raw":
		upper.retainedRaw = min(upper.retainedRaw, maximum)
		upper.payload = min(upper.payload, maximum)
		upper.metadata = min(upper.metadata, maximum)
	}
	s := providerHistoryProvisionedInspectionFixture(t, providerScanFixtureSelection{historyProgram: true, historyGroupWrite: true, historyGroupConsume: true})
	first, err := newProviderFirstOperationCandidate(s)
	if err != nil {
		t.Fatal(err)
	}
	connection := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	if _, err := first.executeCandidateWithConnection(context.Background(), connection); err != nil {
		t.Fatal(err)
	}
	o, err := newProviderHistoryGroupOwnerCandidate(s)
	if err != nil {
		t.Fatal(err)
	}
	requests := make([]*providerIntegrationRequest, count)
	for i := range requests {
		requests[i] = historyGroupSimple(t)
	}
	ack, err := o.retainAdmissionCandidate("selection", requests, upper)
	if err != nil || ack == nil {
		t.Fatal("genuine independently retained T", err)
	}
	return o, ack
}

func TestProviderHistoryConsumptionRetainsAllMembersWithoutRenewingItsWindow(t *testing.T) {
	for _, count := range []int{1, providerHistoryGroupMembers} {
		name := "one"
		if count == providerHistoryGroupMembers {
			name = "maximum"
		}
		t.Run(name, func(t *testing.T) {
			o, admission := historyConsumptionFixture(t, count)
			s, m, before := o.store, o.meter, o.pin
			held, raw, reads := s.stored, s.retainedRaw, s.clock.read
			workStart, clockStart, window, clocks := o.windowStart, o.clockStart, o.windowUpper, o.clockUpper
			priorCoordinator, priorCustodian, retained := o.coordinator, o.custodian, o.retention
			ack, err := o.retainConsumptionCandidate(admission)
			if err != nil || ack == nil {
				t.Fatal("funded C after original T", err)
			}
			all, _ := providerHistoryMemberPrefix(count)
			if ack != o.last || ack == admission || ack.owner != o || ack.claim.owner != ack.claim || !ack.claim.used || !ack.claim.renamed ||
				ack.claim.admission != admission || o.admission != admission || o.pin != ack.pin || o.pin.sequence != before.sequence+1 || o.pin != s.frontierFiles.pin ||
				o.phase != "consumed" || o.coordinator == priorCoordinator || o.custodian == priorCustodian || o.coordinator == o.custodian ||
				o.coordinator.consumption == nil || o.custodian.consumption == nil || o.coordinator.consumption == o.custodian.consumption ||
				o.coordinator.phase != "consumed" || o.custodian.phase != "consumed" || o.coordinator.stage != 0 || o.custodian.stage != 0 || o.coordinator.sequence != 2 || o.custodian.sequence != 2 ||
				o.coordinator.head != o.custodian.head || o.coordinator.consumption.consumed != all || priorCoordinator.phase != "admitted" || priorCustodian.phase != "admitted" ||
				o.order == nil || o.order.owner != o.order || o.order != ack.claim.order || o.order.state != providerHistoryOrderConfirmed || o.order.consumed != all || o.order.sent != 0 || o.order.next != 0 ||
				o.meter != m || m.failed || m.closed || s.activeMeter.Load() != nil || s.frontierFiles.historyPending != nil || s.stored != held ||
				o.windowStart != workStart || o.clockStart != clockStart || o.windowUpper != window || o.clockUpper != clocks || s.clock.read != reads+2 ||
				o.retention.files != retained.files+8 || o.retention.objects != retained.objects+1 || s.retainedRaw != raw+ack.claim.charge.raw ||
				ack.claim.inventory.previous != admission.claim.inventory.digest || ack.claim.inventory.count != 1 {
				t.Fatal("C reused a role, shared a model, renewed a window, dispatched a member or omitted retention")
			}
			file := providerHistoryStoredName(ack.claim.consumption, nil)
			original, err := os.ReadFile(filepath.Join(s.config.evidenceDirectory, file))
			if err != nil || string(original) != o.custodian.consumption.raw {
				t.Fatal("C bytes differ from independent replay", err)
			}
			head, err := os.ReadFile(filepath.Join(s.config.custodyDirectory, "HEAD"))
			if err != nil || string(head) != s.frontierFiles.pinRaw {
				t.Fatal("C HEAD is not its retained original", err)
			}
			afterWork, afterClocks := m.actual, s.clock.read
			if again, err := o.retainConsumptionCandidate(admission); err == nil || again != nil || o.last != ack || m.actual != afterWork || s.clock.read != afterClocks || s.stored != held {
				t.Fatal("consumed T produced a second C or renewed its original meter")
			}
		})
	}
}

func TestProviderHistoryConsumptionRefusesCopiedAndDetachedAdmissionBeforeObservation(t *testing.T) {
	var missing *providerHistoryGroupOwner
	if ack, err := missing.retainConsumptionCandidate(nil); err == nil || ack != nil {
		t.Fatal("missing owner")
	}
	for _, mode := range []string{"nil-ack", "copied-ack", "detached-owner", "missing-window", "closed-meter", "existing-order", "auxiliary-scan-body"} {
		t.Run(mode, func(t *testing.T) {
			o, admission := historyConsumptionFixture(t, 1)
			input := admission
			switch mode {
			case "nil-ack":
				input = nil
			case "copied-ack":
				copied := *admission
				input = &copied
			case "detached-owner":
				copied := *o
				o = &copied
			case "missing-window":
				o.window = false
			case "closed-meter":
				o.meter.closed = true
			case "existing-order":
				o.order = &providerHistoryGroupOrder{}
			case "auxiliary-scan-body":
				// Inert role DATA only: this owns no root, clock or scan.
				o.meter.scanBody = &providerScanBody{}
			}
			s, m, pin, reads, actual, held := o.store, o.meter, o.pin, o.store.clock.read, o.meter.actual, o.store.stored
			if ack, err := o.retainConsumptionCandidate(input); err == nil || ack != nil || o.last != admission || o.pin != pin || s.frontierFiles.pin != pin ||
				s.clock.read != reads || m.actual != actual || s.activeMeter.Load() != nil || s.frontierFiles.historyPending != nil || s.stored != held {
				t.Fatal("unbound C began observation or supplied authority")
			}
		})
	}
}

func TestProviderHistoryConsumptionReplaysItsSelectedAdmissionOriginals(t *testing.T) {
	for _, mode := range []string{"initial-highwater", "initial-frontier", "initial-commit", "request", "ticket", "inventory", "intent", "pending", "commit", "frontier", "cached-T-clock"} {
		t.Run(mode, func(t *testing.T) {
			o, admission := historyConsumptionFixture(t, 1)
			s, c := o.store, admission.claim
			directory, name := s.config.custodyDirectory, ""
			switch mode {
			case "cached-T-clock":
				c.frame.clock.MonotonicMS++
			case "initial-highwater":
				name = s.name("highwater", o.start.main.ordinal)
			case "initial-frontier":
				name = s.name("frontier", o.start.main.sequence)
			case "initial-commit":
				name = s.name("commit", o.start.main.sequence)
			case "request":
				directory, name = s.config.evidenceDirectory, providerHistoryStoredName(c.request, nil)
			case "ticket":
				directory, name = s.config.evidenceDirectory, providerHistoryStoredName(c.ticket, nil)
			case "inventory":
				directory, name = s.config.evidenceDirectory, providerHistoryInventoryName(c.inventory.digest, nil)
			case "pending":
				directory, name = s.config.evidenceDirectory, s.name("history-pending", c.frame.after.sequence)
			default:
				name = s.name(mode, c.frame.after.sequence)
			}
			if mode != "cached-T-clock" {
				if err := os.WriteFile(filepath.Join(directory, name), []byte("inert changed original in owned unit roots"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			pin, held, raw, retained := o.pin, s.stored, s.retainedRaw, o.retention
			if ack, err := o.retainConsumptionCandidate(admission); err == nil || ack != nil || o.last != admission || !o.failed || !s.failed || !o.meter.failed || !o.meter.closed ||
				o.pin != pin || s.frontierFiles.pin != pin || s.stored != held || s.retainedRaw != raw || o.retention != retained || o.order != nil || s.activeMeter.Load() != nil {
				t.Fatal("changed prior original reached C retention or renewed authority")
			}
		})
	}
}

func TestProviderHistoryConsumptionKeepsEveryMemberConsumedOnWriteOrReadbackFailure(t *testing.T) {
	for _, mode := range []string{"sync", "original-missing", "original-changed", "inventory-missing", "inventory-changed", "pending-changed", "frame-changed", "highwater-missing", "model-changed", "copied-order", "prior-ticket-changed", "late-original", "late-catalog", "late-auxiliary-role"} {
		t.Run(mode, func(t *testing.T) {
			o, admission := historyConsumptionFixture(t, 1)
			s, pin, held, raw := o.store, o.pin, o.store.stored, o.store.retainedRaw
			injected := false
			disk := s.custody
			if mode == "sync" {
				disk = s.evidence
			}
			syncFile := disk.syncFile
			disk.syncFile = func(file *os.File) error {
				if err := syncFile(file); err != nil {
					return err
				}
				c := s.frontierFiles.historyPending
				if c == nil || c.frame == nil || c.frame.phase != "consume" || injected {
					return nil
				}
				if mode == "sync" {
					injected = true
					return errProviderIntegration
				}
				if mode == "late-original" || mode == "late-catalog" {
					current, err := os.ReadFile(filepath.Join(s.config.custodyDirectory, "HEAD"))
					if err != nil || string(current) != c.highwater {
						return nil
					}
				} else if filepath.Base(file.Name()) != providerCustodyName("HEAD.pending", c.frame.after.ordinal) {
					return nil
				}
				injected = true
				if mode == "model-changed" {
					c.coordinator.head = admissionTestDigest("f")
					return nil
				}
				if mode == "copied-order" {
					copy := *o.order
					o.order = &copy
					return nil
				}
				if mode == "late-auxiliary-role" {
					o.meter.scanBody = &providerScanBody{} // Inert unit role data.
					return nil
				}
				directory, name := s.config.evidenceDirectory, providerHistoryStoredName(c.consumption, nil)
				switch mode {
				case "inventory-missing", "inventory-changed":
					name = providerHistoryInventoryName(c.inventory.digest, nil)
				case "pending-changed":
					name = s.name("history-pending", c.frame.after.sequence)
				case "frame-changed":
					directory, name = s.config.custodyDirectory, s.name("intent", c.frame.after.sequence)
				case "highwater-missing":
					directory, name = s.config.custodyDirectory, s.name("highwater", c.frame.after.ordinal)
				case "prior-ticket-changed":
					name = providerHistoryStoredName(admission.claim.ticket, nil)
				case "late-catalog":
					name = "unit-inert-unrecognized-after-C-rename"
				}
				path := filepath.Join(directory, name)
				if mode == "original-missing" || mode == "inventory-missing" || mode == "highwater-missing" {
					return os.Remove(path)
				}
				return os.WriteFile(path, []byte("inert owned unit fault"), 0600)
			}
			if ack, err := o.retainConsumptionCandidate(admission); err == nil || ack != nil || !injected || o.last != admission || !o.failed || !s.failed || !o.meter.failed || !o.meter.closed ||
				o.pin != pin || s.frontierFiles.pin != pin || s.stored != held || s.retainedRaw != raw || s.activeMeter.Load() != nil ||
				o.order == nil || o.order.consumed != 1 || o.retention.objects != 3 || o.retention.files != 17 {
				t.Fatal("C failure published its provisional pin, omitted consumption or refunded its admitted hold")
			}
			if pending := s.frontierFiles.historyPending; pending == nil || pending.used || pending.admission != admission || pending.order == nil || pending.order.state != providerHistoryOrderFailed || pending.order.consumed != 1 ||
				(mode == "late-original" || mode == "late-catalog") && !pending.renamed {
				t.Fatal("C failure lost its original pending transaction or rename boundary")
			}
		})
	}
}

func TestProviderHistoryConsumptionUsesTheOriginalClockAndRemainingGroupWork(t *testing.T) {
	for _, mode := range []string{"begin-clock", "end-clock", "remaining-work"} {
		t.Run(mode, func(t *testing.T) {
			o, admission := historyConsumptionFixture(t, 1)
			s, m, pin, held, raw, reads := o.store, o.meter, o.pin, o.store.stored, o.store.retainedRaw, o.store.clock.read
			actual, start, clockStart := m.actual, o.windowStart, o.clockStart
			allowed := uint64(2)
			switch mode {
			case "begin-clock":
				allowed = 0
				o.parent.clockCeiling = reads
			case "end-clock":
				allowed = 1
				o.parent.clockCeiling = reads + 1
			case "remaining-work":
				o.windowUpper[10] = actual[10] - start[10]
			}
			if ack, err := o.retainConsumptionCandidate(admission); err == nil || ack != nil || o.last != admission || !o.failed || !s.failed || !m.failed || !m.closed ||
				o.pin != pin || s.frontierFiles.pin != pin || s.stored != held || s.retainedRaw != raw || o.windowStart != start || o.clockStart != clockStart || s.activeMeter.Load() != nil {
				t.Fatal("C renewed its original clock, work window or hold after depletion")
			}
			if mode != "remaining-work" && s.clock.read != reads+allowed {
				t.Fatal("unavailable original native call was performed")
			}
			for i, before := range actual {
				if m.actual[i] < before {
					t.Fatal("C refunded prior work", i)
				}
			}
			if mode == "end-clock" && (o.order == nil || o.order.consumed != 1 || o.order.state != providerHistoryOrderFailed || s.frontierFiles.historyPending == nil || !s.frontierFiles.historyPending.renamed) {
				t.Fatal("post-rename clock refusal restored a member or omitted quarantine")
			}
		})
	}
}

// Finite diagnostic arithmetic only. These values never enter a native clock,
// writer, observer or operating claim. Actual C tests above use owned samples.
func TestProviderHistoryConsumptionKeepsTheOriginalAdmissionElapsedAnchor(t *testing.T) {
	origin := CandidateClockDiagnostic{ClockID: "unit-diagnostic", Epoch: admissionTestDigest("a"), WallMS: 1000, MonotonicMS: 10}
	for _, test := range []struct {
		name    string
		elapsed int64
		want    bool
	}{
		{"same", 0, true}, {"boundary", 300000, true}, {"new-short-C-bracket-cannot-renew-T", 300001, false}, {"backward", -1, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			current := origin
			current.WallMS += test.elapsed
			current.MonotonicMS += test.elapsed
			if got := providerHistoryGroupClockFits(origin, current); got != test.want {
				t.Fatal("original elapsed anchor", got)
			}
		})
	}
}

func TestProviderHistoryConsumptionRefusesCumulativeGroupStorageBeyondItsOriginalTicket(t *testing.T) {
	control, controlAdmission := historyConsumptionFixture(t, 1)
	diagnostic := control.retention
	controlC, err := control.retainConsumptionCandidate(controlAdmission)
	if err != nil || controlC == nil {
		t.Fatal("separate original T-through-C positive control", err)
	}
	delta := controlC.claim.charge
	if err := control.store.close(); err != nil {
		t.Fatal("separate diagnostic fixture close", err)
	}
	for _, dimension := range []string{"physical", "raw"} {
		t.Run(dimension, func(t *testing.T) {
			maximum := diagnostic.physical + 256
			if dimension == "raw" {
				maximum = diagnostic.raw + 256
			}
			o, admission := historyConsumptionFixtureWithCap(t, 1, dimension, maximum)
			s, pin, held, raw, retained := o.store, o.pin, o.store.stored, o.store.retainedRaw, o.retention
			selected := o.coordinator.charged.physical
			actual := retained.physical
			additional := delta.physical
			if dimension == "raw" {
				selected, actual = o.coordinator.charged.retainedRaw, retained.raw
				additional = delta.raw
			}
			if selected != maximum || actual > maximum || additional <= maximum-actual {
				t.Fatal("this independent T must actually fit its smaller admitted group cap")
			}
			if ack, err := o.retainConsumptionCandidate(admission); err == nil || ack != nil || o.last != admission || !o.failed || !s.failed || !o.meter.failed || !o.meter.closed ||
				o.pin != pin || s.frontierFiles.pin != pin || s.stored != held || s.retainedRaw != raw || o.retention != retained || o.order != nil ||
				s.activeMeter.Load() != nil || s.frontierFiles.historyPending != nil {
				t.Fatal("C borrowed the larger parent capacity or refunded its original T charge")
			}
		})
	}
}
