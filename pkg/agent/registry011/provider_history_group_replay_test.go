package registry011

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func historyDispatchReplayFixture(t *testing.T, count int) (*providerHistoryGroupOwner, *providerHistoryGroupAcknowledgment) {
	t.Helper()
	o, admission := historyConsumptionFixture(t, count)
	ack, err := o.retainConsumptionCandidate(admission)
	if err != nil || ack == nil || !o.consumptionShapeLocked() || o.phase != "consumed" {
		t.Fatal("original first exchange, retained T and retained C are mandatory", err)
	}
	return o, ack
}

func TestProviderHistoryDispatchPreparationReplaysCommittedCWithoutRewinding(t *testing.T) {
	for _, count := range []int{1, 3, providerHistoryGroupMembers} {
		name := "one"
		if count == 3 {
			name = "several"
		} else if count == providerHistoryGroupMembers {
			name = "maximum"
		}
		t.Run(name, func(t *testing.T) {
			o, ack := historyDispatchReplayFixture(t, count)
			s, m, pin, raw := o.store, o.meter, o.pin, o.store.frontierFiles.pinRaw
			coordinator, custodian, order, admission := o.coordinator, o.custodian, o.order, o.admission
			actual, reads, held, retained, payload := m.actual, s.clock.read, s.stored, o.retention, s.retainedRaw
			windowStart, windowUpper, clockStart, clockUpper := o.windowStart, o.windowUpper, o.clockStart, o.clockUpper
			d, err := o.prepareDispatchCandidate(ack)
			if err != nil || d == nil || d != o.dispatch || d.owner != d || d.group != o || d.acknowledgment != ack || d.order != order || d.meter != m ||
				d.coordinator == coordinator || d.custodian == custodian || d.coordinator == d.custodian ||
				d.coordinator.ticket == d.custodian.ticket || d.coordinator.consumption == d.custodian.consumption ||
				d.coordinator.head != coordinator.head || d.custodian.head != custodian.head || !d.coordinator.valid() || !d.custodian.valid() ||
				o.phase != "dispatch-prepared" || o.pin != pin || s.frontierFiles.pin != pin || s.frontierFiles.pinRaw != raw ||
				o.last != ack || o.admission != admission || o.coordinator != coordinator || o.custodian != custodian || o.order != order ||
				order.state != providerHistoryOrderConfirmed || order.next != 0 || order.sent != 0 || o.retention != retained || s.retainedRaw != payload || s.stored != held ||
				m.failed || m.closed || s.activeMeter.Load() != nil || s.frontierFiles.historyPending != nil || s.clock.read != reads+2 ||
				o.windowStart != windowStart || o.windowUpper != windowUpper || o.clockStart != clockStart || o.clockUpper != clockUpper ||
				m.actual[0] != actual[0] || m.actual[1] != actual[1] || m.actual[9] != actual[9] || m.actual[10] <= actual[10] || m.actual[2] <= actual[2] {
				t.Fatal("preparation changed live C, shared a replay, dispatched, rewound or renewed an original window", err)
			}
			after, clocks := m.actual, s.clock.read
			if repeated, err := o.prepareDispatchCandidate(ack); err == nil || repeated != nil || o.dispatch != d || m.actual != after || s.clock.read != clocks {
				t.Fatal("same C produced a second preparation or observed an idle role")
			}
		})
	}
}

func TestProviderHistoryDispatchPreparationRefusesDetachedInputsBeforeObservation(t *testing.T) {
	var absent *providerHistoryGroupOwner
	if d, err := absent.prepareDispatchCandidate(nil); err == nil || d != nil {
		t.Fatal("missing owner")
	}
	for _, mode := range []string{"nil-ack", "copied-ack", "old-T-ack", "copied-owner", "copied-order", "missing-order", "used-prefix", "unconfirmed", "closed-meter", "failed-meter", "copied-meter", "scan-body", "legacy-role", "authority", "changed-window", "changed-clock-window", "changed-upper", "pending-legacy"} {
		t.Run(mode, func(t *testing.T) {
			o, ack := historyDispatchReplayFixture(t, 1)
			input := ack
			switch mode {
			case "nil-ack":
				input = nil
			case "copied-ack":
				copy := *ack
				input = &copy
			case "old-T-ack":
				input = o.admission
			case "copied-owner":
				copy := *o
				o = &copy
			case "copied-order":
				copy := *o.order
				o.order = &copy
			case "missing-order":
				o.order = nil
			case "used-prefix":
				if !o.order.begin(0) || !o.order.complete(0) {
					t.Fatal("finite order DATA prefix")
				}
			case "unconfirmed":
				o.order.state = providerHistoryOrderConsumed
			case "closed-meter":
				o.meter.closed = true
			case "failed-meter":
				o.meter.failed = true
			case "copied-meter":
				copy := &providerWorkMeter{owner: o.meter, historyGroup: o, upper: o.meter.upper}
				o.meter = copy
			case "scan-body":
				o.meter.scanBody = &providerScanBody{}
			case "legacy-role":
				o.meter.integration = &providerIntegration{}
			case "authority":
				o.meter.authority = false
			case "changed-window":
				o.windowUpper[10]++
			case "changed-clock-window":
				o.clockUpper++
			case "changed-upper":
				o.meter.upper[10]++
			case "pending-legacy":
				o.store.frontierFiles.pending = &providerFrontierWriteClaim{}
			}
			m, s := o.meter, o.store
			actual, reads, pin, held, retained := m.actual, s.clock.read, o.pin, s.stored, o.retention
			if d, err := o.prepareDispatchCandidate(input); err == nil || d != nil || o.dispatch != nil || m.actual != actual || s.clock.read != reads ||
				o.pin != pin || s.frontierFiles.pin != pin || o.last != ack || s.stored != held || o.retention != retained || s.activeMeter.Load() != nil {
				t.Fatal("detached or already used C reached observation or dispatch")
			}
		})
	}
}

func TestProviderHistoryDispatchPreparationRequiresRetainedInitialTAndCOriginals(t *testing.T) {
	for _, mode := range []string{"HEAD", "initial-highwater", "initial-frontier", "initial-commit", "request", "ticket", "T-inventory", "T-intent", "T-pending", "T-commit", "T-frontier", "T-highwater", "consumption", "C-inventory", "C-intent", "C-pending", "C-commit", "C-frontier", "C-highwater", "T-clock", "C-clock"} {
		t.Run(mode, func(t *testing.T) {
			o, ack := historyDispatchReplayFixture(t, 1)
			s, a, c := o.store, o.admission.claim, ack.claim
			directory, name := s.config.custodyDirectory, mode
			switch mode {
			case "T-clock":
				a.frame.clock.MonotonicMS++
			case "C-clock":
				c.frame.clock.MonotonicMS++
			case "initial-highwater":
				name = s.name("highwater", o.start.main.ordinal)
			case "initial-frontier":
				name = s.name("frontier", o.start.main.sequence)
			case "initial-commit":
				name = s.name("commit", o.start.main.sequence)
			case "request":
				directory, name = s.config.evidenceDirectory, providerHistoryStoredName(a.request, nil)
			case "ticket":
				directory, name = s.config.evidenceDirectory, providerHistoryStoredName(a.ticket, nil)
			case "consumption":
				directory, name = s.config.evidenceDirectory, providerHistoryStoredName(c.consumption, nil)
			default:
				if len(mode) > 2 && mode[1] == '-' {
					claim, kind := a, mode[2:]
					if mode[0] == 'C' {
						claim = c
					}
					switch kind {
					case "inventory":
						directory, name = s.config.evidenceDirectory, providerHistoryInventoryName(claim.inventory.digest, nil)
					case "pending":
						directory, name = s.config.evidenceDirectory, s.name("history-pending", claim.frame.after.sequence)
					case "highwater":
						name = s.name("highwater", claim.frame.after.ordinal)
					default:
						name = s.name(kind, claim.frame.after.sequence)
					}
				}
			}
			if mode != "T-clock" && mode != "C-clock" {
				if err := os.WriteFile(filepath.Join(directory, name), []byte("inert original fault in private unit roots"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			pin, held, retained, actual := o.pin, s.stored, o.retention, o.meter.actual
			if d, err := o.prepareDispatchCandidate(ack); err == nil || d != nil || o.dispatch != nil || !o.failed || !s.failed || !o.meter.failed || !o.meter.closed ||
				o.pin != pin || s.frontierFiles.pin != pin || o.last != ack || o.retention != retained || s.stored != held || s.activeMeter.Load() != nil ||
				o.order.consumed != 1 || o.order.sent != 0 || o.order.next != 0 || o.order.state != providerHistoryOrderFailed || o.meter.actual[10] <= actual[10] {
				t.Fatal("changed committed original granted preparation, rewound C or refunded consumed state")
			}
		})
	}
}

func TestProviderHistoryDispatchPreparationDerivesCatalogWithoutReusingCachedNames(t *testing.T) {
	for _, mode := range []string{"discarded-caches", "custody-extra", "evidence-extra", "missing-selected-chunk", "missing-selected-command"} {
		t.Run(mode, func(t *testing.T) {
			o, ack := historyDispatchReplayFixture(t, 1)
			s, a, c := o.store, o.admission.claim, ack.claim
			a.beforeCatalog, a.afterCatalog, c.beforeCatalog, c.afterCatalog = [2]map[string]bool{}, [2]map[string]bool{}, [2]map[string]bool{}, [2]map[string]bool{}
			if mode == "custody-extra" || mode == "evidence-extra" {
				directory := s.config.custodyDirectory
				if mode == "evidence-extra" {
					directory = s.config.evidenceDirectory
				}
				if err := os.WriteFile(filepath.Join(directory, "inert-unrecognized-unit-name"), []byte("inert"), 0600); err != nil {
					t.Fatal(err)
				}
			} else if mode == "missing-selected-chunk" || mode == "missing-selected-command" {
				first := o.parent.firstAdmission
				name := providerScanReceiptName("chunk", first.lastChunk, nil)
				if mode == "missing-selected-command" {
					// Read the selected local unit receipt as DATA to choose
					// which existing original this finite refusal removes.
					raw, err := os.ReadFile(filepath.Join(s.config.evidenceDirectory, name))
					if err != nil {
						t.Fatal(err)
					}
					var fields map[string]any
					if err := json.Unmarshal(raw, &fields); err != nil {
						t.Fatal(err)
					}
					name = providerScanReceiptName("command", contentString(fields["command_digest"]), nil)
					if name == "" {
						t.Fatal("selected original command name")
					}
				}
				if err := os.Remove(filepath.Join(s.config.evidenceDirectory, name)); err != nil {
					t.Fatal(err)
				}
			}
			d, err := o.prepareDispatchCandidate(ack)
			if mode == "discarded-caches" {
				if err != nil || d == nil || o.failed {
					t.Fatal("cached before/after maps replaced independently selected originals", err)
				}
			} else if err == nil || d != nil || !o.failed || o.order.consumed != 1 || o.order.sent != 0 {
				t.Fatal("observed unknown catalog or missing original was adopted")
			}
		})
	}
}

func TestProviderHistoryDispatchPreparationUsesOriginalNativeClockCeiling(t *testing.T) {
	for _, mode := range []string{"before-first-sample", "before-final-sample"} {
		t.Run(mode, func(t *testing.T) {
			o, ack := historyDispatchReplayFixture(t, 1)
			s, m, reads, pin, held := o.store, o.meter, o.store.clock.read, o.pin, o.store.stored
			allowed := uint64(0)
			if mode == "before-final-sample" {
				allowed = 1
			}
			o.parent.clockCeiling = reads + allowed
			actual, start, clockStart := m.actual, o.windowStart, o.clockStart
			if d, err := o.prepareDispatchCandidate(ack); err == nil || d != nil || o.dispatch != nil || !o.failed || !s.failed || !m.failed || !m.closed ||
				o.pin != pin || s.frontierFiles.pin != pin || o.last != ack || s.stored != held || s.clock.read != reads+allowed ||
				o.windowStart != start || o.clockStart != clockStart || s.activeMeter.Load() != nil || o.order.consumed != 1 || o.order.sent != 0 {
				t.Fatal("preparation renewed native lifetime, pin, hold or consumed bits")
			}
			for i, old := range actual {
				if m.actual[i] < old {
					t.Fatal("preparation refunded prior work", i)
				}
			}
		})
	}
}

func TestProviderHistoryDispatchPreparationRefusesAChangedCachedModel(t *testing.T) {
	for _, mode := range []string{"model-root", "member-wire", "consumption-mask"} {
		t.Run(mode, func(t *testing.T) {
			o, ack := historyDispatchReplayFixture(t, 1)
			switch mode {
			case "model-root":
				o.coordinator.head, o.custodian.head = admissionTestDigest("e"), admissionTestDigest("e")
			case "member-wire":
				o.coordinator.ticket.table.members[0].request.wire = "inert changed cached HTTP input"
				o.custodian.ticket.table.members[0].request.wire = "inert changed cached HTTP input"
			case "consumption-mask":
				o.coordinator.consumption.raw = "inert changed cached consumption"
				o.custodian.consumption.raw = "inert changed cached consumption"
			}
			if !o.consumptionShapeLocked() {
				t.Fatal("this finite model fault must reach independent original verification")
			}
			before, pin := o.meter.actual, o.pin
			if d, err := o.prepareDispatchCandidate(ack); err == nil || d != nil || o.dispatch != nil || !o.failed || !o.meter.failed || !o.meter.closed ||
				o.last != ack || o.pin != pin || o.store.frontierFiles.pin != pin || o.order.consumed != 1 || o.order.sent != 0 || o.meter.actual[10] <= before[10] {
				t.Fatal("matching caches replaced the independently replayed C")
			}
		})
	}
}

func TestProviderHistoryDispatchPreparationKeepsTheOriginalRemainingWork(t *testing.T) {
	control, controlC := historyDispatchReplayFixture(t, 1)
	used := control.meter.actual[2] - control.windowStart[2]
	before := control.meter.actual[2]
	if d, err := control.prepareDispatchCandidate(controlC); err != nil || d == nil || control.meter.actual[2]-before <= 4096 {
		t.Fatal("separate original positive control must actually decode the proposed original replay", err)
	}
	replayWork := control.meter.actual[2] - before
	if err := control.store.close(); err != nil {
		t.Fatal(err)
	}
	// This is a finite unit input, not a production cost certificate. The
	// smaller decoded-byte cap predates the new independent subject's H0.
	// Descriptor digit lengths can change, so this uses a wide interior
	// interval and confirms the actual new T/C precursor and remaining work.
	o, admission := historyConsumptionFixtureWithCap(t, 1, "decoded-work", used+replayWork/4)
	ack, err := o.retainConsumptionCandidate(admission)
	if err != nil || ack == nil || !o.consumptionShapeLocked() {
		t.Fatal("new T and C must genuinely fit their smaller original group upper", err)
	}
	usedNow := o.meter.actual[2] - o.windowStart[2]
	if usedNow > o.windowUpper[2] || o.windowUpper[2]-usedNow >= replayWork/2 {
		t.Fatal("refusal subject did not reach the intended funded-C remaining-work interval")
	}
	actual, pin, held, start, clockStart := o.meter.actual, o.pin, o.store.stored, o.windowStart, o.clockStart
	if d, err := o.prepareDispatchCandidate(ack); err == nil || d != nil || o.dispatch != nil || !o.failed || !o.meter.failed || !o.meter.closed ||
		o.pin != pin || o.store.frontierFiles.pin != pin || o.last != ack || o.store.stored != held || o.windowStart != start || o.clockStart != clockStart ||
		o.order.consumed != 1 || o.order.sent != 0 || o.order.next != 0 || o.store.activeMeter.Load() != nil {
		t.Fatal("unfunded preparation borrowed a fresh counter, renewed its hold or dispatched")
	}
	for i, old := range actual {
		if o.meter.actual[i] < old {
			t.Fatal("refused original window refunded prior actuals", i)
		}
	}
}
