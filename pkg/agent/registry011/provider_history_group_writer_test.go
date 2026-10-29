package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func historyAdmissionWriterFixture(t *testing.T) (*providerHistoryGroupOwner, providerHistoryGroupUpper) {
	t.Helper()
	s := providerHistoryProvisionedInspectionFixture(t, providerScanFixtureSelection{historyProgram: true, historyGroupWrite: true})
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
	if o.parent.firstAdmission == nil || o.parent.firstAdmission.body == nil || o.parent.firstAdmission.body.receipts != nil {
		t.Fatal("first admission must release its closed inspection receipt cache")
	}
	// These test-only positive scalars do not prove source-derived costs or
	// permit capture. The whole synthetic input predates anchor/H0/clock.
	upper := providerHistoryGroupUpper{physical: 8 << 20, retainedRaw: 4 << 20, payload: 128 << 10, metadata: 2 << 20, objects: 8, files: 64, clocks: 16}
	upper.execution[0], upper.execution[1], upper.execution[10] = 1, 1, 128
	upper.custody[2], upper.custody[9], upper.custody[10] = 16<<20, 16<<20, 1<<50
	return o, upper
}

func TestProviderHistoryAdmissionWriterRetainsIndependentOriginalsBeforePublishing(t *testing.T) {
	o, upper := historyAdmissionWriterFixture(t)
	s, before, initialCoordinator, initialCustodian := o.store, o.pin, o.coordinator, o.custodian
	held, raw, reads, meter := s.stored, s.retainedRaw, s.clock.read, o.meter
	ack, err := o.retainAdmissionCandidate("selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper)
	if err != nil || ack == nil {
		t.Fatal("funded original admission", err)
	}
	if ack != o.last || ack.owner != o || ack.claim.owner != ack.claim || !ack.claim.used || !ack.claim.renamed || ack.pin != o.pin ||
		o.phase != "admitted" || o.pin.sequence != before.sequence+1 || o.pin != s.frontierFiles.pin || s.sequence != o.pin.sequence || s.tip != o.pin.commit ||
		o.coordinator == initialCoordinator || o.custodian == initialCustodian || o.coordinator == o.custodian ||
		o.coordinator.ticket == o.custodian.ticket || o.coordinator.head != o.custodian.head || o.coordinator.ticket.table.members[0].request.wire != o.custodian.ticket.table.members[0].request.wire ||
		s.stored != held || s.retainedRaw != raw+o.retention.raw || s.objectsRoot != o.start.objects || s.frontierFiles.historyPending != nil ||
		s.activeMeter.Load() != nil || o.meter != meter || meter.failed || meter.closed || s.clock.read != reads+2 || meter.clocks != 4 ||
		initialCoordinator.phase != "prepared" || initialCustodian.phase != "prepared" || o.retention.physical == 0 || o.retention.files != 9 || o.retention.objects != 2 {
		t.Fatal("admission published shared models, changed its original hold or omitted retained data")
	}
	head, err := os.ReadFile(filepath.Join(s.config.custodyDirectory, "HEAD"))
	if err != nil || string(head) != s.frontierFiles.pinRaw {
		t.Fatal("published HEAD differs from its original", err)
	}
	for i := range ack.claim.count {
		ref := ack.claim.inventory.entries[i]
		body, err := os.ReadFile(filepath.Join(s.config.evidenceDirectory, providerHistoryStoredName(ref, nil)))
		if err != nil || string(body) != ack.claim.fresh[i].raw {
			t.Fatal("published admission did not retain exact original", ref.kind, err)
		}
	}
	if again, err := o.retainAdmissionCandidate("selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper); err == nil || again != nil || o.last != ack || s.stored != held || o.pin != ack.pin {
		t.Fatal("same admission minted another ACK or returned a consumed hold")
	}
}

func TestProviderHistoryAdmissionCatalogRequiresSelectedOriginals(t *testing.T) {
	for _, mode := range []string{"chunk-custody-missing", "chunk-evidence-missing", "chunk-custody-changed", "chunk-evidence-changed",
		"command-custody-missing", "command-evidence-missing", "command-custody-changed", "command-evidence-changed", "selected-count", "selected-tail"} {
		t.Run(mode, func(t *testing.T) {
			o, upper := historyAdmissionWriterFixture(t)
			s, first, pin, held, raw := o.store, o.parent.firstAdmission, o.pin, o.store.stored, o.store.retainedRaw
			chunkName := providerScanReceiptName("chunk", first.lastChunk, nil)
			original, err := os.ReadFile(filepath.Join(s.config.custodyDirectory, chunkName))
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]any
			if err := json.Unmarshal(original, &fields); err != nil {
				t.Fatal(err)
			}
			command := contentString(fields["command_digest"])
			if !lowerHash(command) || first.records == 0 || first.mainPin.ordinal != first.records-1 {
				t.Fatal("negative case requires the selected complete original chain")
			}
			switch mode {
			case "selected-count":
				first.records++
			case "selected-tail":
				first.lastChunk = command // A real digest in the wrong domain.
			default:
				kind, directory, remove := "chunk", s.config.custodyDirectory, true
				switch mode {
				case "chunk-evidence-missing", "chunk-evidence-changed", "command-evidence-missing", "command-evidence-changed":
					directory = s.config.evidenceDirectory
				}
				switch mode {
				case "command-custody-missing", "command-evidence-missing", "command-custody-changed", "command-evidence-changed":
					kind = "command"
				}
				switch mode {
				case "chunk-custody-changed", "chunk-evidence-changed", "command-custody-changed", "command-evidence-changed":
					remove = false
				}
				name := chunkName
				if kind == "command" {
					name = providerScanReceiptName(kind, command, nil)
				}
				// Finite inert input in this test's private roots only. Neither
				// a peer nor an external Registry or path is contacted.
				path := filepath.Join(directory, name)
				if remove {
					err = os.Remove(path)
				} else {
					err = os.WriteFile(path, []byte("inert changed receipt"), 0600)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			if ack, err := o.retainAdmissionCandidate("selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper); err == nil || ack != nil || o.last != nil ||
				!o.failed || !s.failed || !o.meter.failed || !o.meter.closed || s.activeMeter.Load() != nil || o.pin != pin || s.frontierFiles.pin != pin ||
				s.stored != held || s.retainedRaw != raw || o.retention != (providerHistoryStorageCharge{}) || s.frontierFiles.historyPending != nil {
				t.Fatal("invalid selected original reached retention or returned admission authority")
			}
		})
	}
}

func TestProviderHistoryAdmissionWriterRefusesBeforePublishingAndKeepsCharges(t *testing.T) {
	for _, mode := range []string{"storage", "catalog", "unknown-custody", "unknown-evidence", "late-catalog", "work", "clocks", "preexisting", "write", "sync"} {
		t.Run(mode, func(t *testing.T) {
			o, upper := historyAdmissionWriterFixture(t)
			s, pin, held, raw := o.store, o.pin, o.store.stored, o.store.retainedRaw
			if mode == "storage" {
				upper.physical, upper.retainedRaw, upper.payload, upper.metadata = 64, 64, 1, 1
			}
			if mode == "catalog" {
				// Existing unrelated regular files are inert unit input inside
				// this test's own roots, not an external corruption program.
				for i := range 2048 {
					name := providerCustodyName("unit-extra", uint64(i))
					if err := os.WriteFile(filepath.Join(s.config.evidenceDirectory, name), []byte("unit"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			if mode == "unknown-custody" || mode == "unknown-evidence" {
				directory := s.config.custodyDirectory
				if mode == "unknown-evidence" {
					directory = s.config.evidenceDirectory
				}
				if err := os.WriteFile(filepath.Join(directory, "unit-unrecognized"), []byte("inert unit data"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "work" {
				upper.execution[10], upper.custody[10] = 1, 1
			}
			if mode == "clocks" {
				upper.clocks = 1
			}
			if mode == "preexisting" {
				if err := os.WriteFile(filepath.Join(s.config.custodyDirectory, s.name("intent", pin.sequence+1)), []byte("unit exists"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			writeInjected, lateCatalogInjected := false, false
			if mode == "write" {
				syncFile := s.evidence.syncFile
				s.evidence.syncFile = func(file *os.File) error {
					if err := syncFile(file); err != nil {
						return err
					}
					pending := s.frontierFiles.historyPending
					if !writeInjected && pending != nil && filepath.Base(file.Name()) == providerHistoryStoredName(pending.inventory.entries[0], nil) {
						// The first original is already being retained. Place
						// inert next-file data only inside this owned fixture;
						// the following exclusive write must then refuse.
						name := providerHistoryStoredName(pending.inventory.entries[1], nil)
						if err := os.WriteFile(filepath.Join(s.config.evidenceDirectory, name), []byte("inert next original"), 0600); err != nil {
							return err
						}
						writeInjected = true
					}
					return nil
				}
			}
			if mode == "sync" {
				s.evidence.syncFile = func(*os.File) error { return errProviderIntegration }
			}
			if mode == "late-catalog" {
				syncFile := s.custody.syncFile
				s.custody.syncFile = func(file *os.File) error {
					if err := syncFile(file); err != nil {
						return err
					}
					pending := s.frontierFiles.historyPending
					if pending != nil {
						head, err := os.ReadFile(filepath.Join(s.config.custodyDirectory, "HEAD"))
						if err == nil && string(head) == pending.highwater {
							// A finite isolated fixture after physical rename;
							// no peer, external path or bypass tool is involved.
							if err := os.WriteFile(filepath.Join(s.config.evidenceDirectory, "unit-unrecognized-after-rename"), []byte("inert unit data"), 0600); err != nil {
								return err
							}
							lateCatalogInjected = true
						}
					}
					return nil
				}
			}
			if !upper.valid(1) {
				t.Fatal("negative writer scenario requires otherwise valid upper data")
			}
			if ack, err := o.retainAdmissionCandidate("selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper); err == nil || ack != nil || o.last != nil ||
				!o.failed || !s.failed || !s.history.failed || !s.clock.historyRevoked || !o.meter.failed || !o.meter.closed || s.activeMeter.Load() != nil ||
				o.pin != pin || s.frontierFiles.pin != pin || s.sequence != pin.sequence || s.tip != pin.commit || s.stored != held || s.retainedRaw != raw ||
				o.coordinator.phase != "prepared" || o.custodian.phase != "prepared" {
				t.Fatal("refused admission returned an ACK, published a provisional pin or refunded its parent")
			}
			if mode == "preexisting" && (o.retention != (providerHistoryStorageCharge{}) || s.frontierFiles.historyPending != nil) {
				t.Fatal("unexpected preexisting file entered a retention transaction")
			}
			if mode == "write" && !writeInjected {
				t.Fatal("write refusal did not reach the retained original boundary")
			}
			if mode == "late-catalog" && (!lateCatalogInjected || s.frontierFiles.historyPending == nil || !s.frontierFiles.historyPending.renamed) {
				t.Fatal("late catalog refusal did not follow the physical HEAD rename")
			}
			if mode == "write" || mode == "sync" || mode == "late-catalog" {
				if o.retention.physical == 0 || s.frontierFiles.historyPending == nil || s.frontierFiles.historyPending.used {
					t.Fatal("uncertain write erased its no-refund claim")
				}
			}
		})
	}
}

func TestProviderHistoryAdmissionIdentityChecksPreflightAllFourMetadataReads(t *testing.T) {
	for _, mode := range []string{"identity", "catalog"} {
		t.Run(mode, func(t *testing.T) {
			o, upper := historyAdmissionWriterFixture(t)
			s, m := o.store, o.meter
			s.mu.Lock()
			defer s.mu.Unlock()
			before, reads := m.actual, s.clock.read
			// A finite child window inside the same already held observer.
			// No actual, epoch, parent allowance or selected root is reset.
			work, ok := upper.work()
			if !ok {
				t.Fatal("valid test window")
			}
			work[10] = 3
			o.window, o.windowStart, o.windowUpper = true, m.actual, work
			o.clockStart, o.clockUpper, o.phase = m.clocks, upper.clocks, "preparing-admit"
			s.activeMeter.Store(m)
			defer o.failLocked()
			if !o.activeLocked(m) || !m.preflight(providerReadEvent, 0) {
				t.Fatal("negative case did not start with its exact live role")
			}
			got := false
			if mode == "identity" {
				got = o.historyDiskIdentityLocked(s.custody)
			} else {
				got = o.admissionCatalogFitsLocked(&providerHistoryWriteClaim{})
			}
			if got || !m.failed || m.actual != before || s.clock.read != reads {
				t.Fatal("four-read primitive was partially charged without being called")
			}
		})
	}
}

func TestProviderHistoryAdmissionCopiesCannotBorrowWriterOwnership(t *testing.T) {
	o, upper := historyAdmissionWriterFixture(t)
	copy := *o
	before := o.meter.actual
	if ack, err := copy.retainAdmissionCandidate("selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper); err == nil || ack != nil || o.failed || o.meter.actual != before || o.phase != "prepared" {
		t.Fatal("copied owner entered a writer phase")
	}
	for _, value := range []any{&providerHistoryWriteClaim{}, &providerHistoryGroupAcknowledgment{}, &providerHistoryInventory{}} {
		if json.Unmarshal([]byte(`{}`), value) == nil {
			t.Fatal("JSON reconstructed history authority")
		}
	}
}

func TestProviderHistoryAdmissionWriterRefusesInvalidObserverBeforeWindow(t *testing.T) {
	for _, mode := range []string{"missing", "copied", "foreign", "closed", "failed", "mixed-role", "bounds", "clock-cap"} {
		t.Run(mode, func(t *testing.T) {
			o, upper := historyAdmissionWriterFixture(t)
			s, original := o.store, o.meter
			before, clocks, reads, pin, held := original.actual, original.clocks, s.clock.read, o.pin, s.stored
			defer func() { o.meter = original }()
			switch mode {
			case "missing":
				o.meter = nil
			case "copied":
				o.meter = &providerWorkMeter{owner: original, historyGroup: o, upper: original.upper, maximumClocks: original.maximumClocks, authority: true}
			case "foreign":
				o.meter = newProviderWorkMeter(nil, providerOperationReservation{work: original.upper, execution: original.upper})
			case "closed":
				original.closed = true
			case "failed":
				original.failed = true
			case "mixed-role":
				original.firstOperation = &providerFirstOperationOwner{}
			case "bounds":
				original.upper[10]--
			case "clock-cap":
				original.maximumClocks--
			}
			if ack, err := o.retainAdmissionCandidate("selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper); err == nil || ack != nil || o.last != nil ||
				o.phase != "prepared" || o.window || o.retention != (providerHistoryStorageCharge{}) || s.frontierFiles.historyPending != nil ||
				s.activeMeter.Load() != nil || original.actual != before || original.clocks != clocks || s.clock.read != reads || o.pin != pin || s.stored != held {
				t.Fatal("invalid observer entered admission or changed the original lifetime")
			}
		})
	}
}

func TestProviderHistoryAdmissionWindowStopsAfterItsAllowedPrefix(t *testing.T) {
	for _, mode := range []string{"work", "clocks"} {
		t.Run(mode, func(t *testing.T) {
			o, upper := historyAdmissionWriterFixture(t)
			s, m := o.store, o.meter
			s.mu.Lock()
			defer s.mu.Unlock()
			work, ok := upper.work()
			if !ok {
				t.Fatal("valid finite window")
			}
			if mode == "work" {
				work[10] = 3
			}
			o.window, o.windowStart, o.windowUpper = true, m.actual, work
			o.clockStart, o.clockUpper, o.phase = m.clocks, 2, "preparing-admit"
			s.activeMeter.Store(m)
			defer o.failLocked()
			initial, reads, clocks := m.actual, s.clock.read, m.clocks
			if mode == "work" {
				for range 3 {
					if !m.preflight(providerReadEvent, 0) || !m.event(providerReadEvent, 0) {
						t.Fatal("allowed work prefix was refused")
					}
				}
				atBound := m.actual
				if atBound[10] != initial[10]+3 || m.event(providerReadEvent, 0) || !m.failed || m.actual != atBound || m.clocks != clocks || s.clock.read != reads {
					t.Fatal("work window renewed capacity or consumed beyond its bound")
				}
			} else {
				for range 2 {
					if sample, err := s.samplePrimitive(); err != nil || !o.clockFits(sample) {
						t.Fatal("allowed original clock prefix was refused", err)
					}
				}
				atBound := m.actual
				if s.clock.read != reads+2 || m.clocks != clocks+2 {
					t.Fatal("clock prefix did not use the original backend twice")
				}
				if _, err := s.samplePrimitive(); err == nil || !m.failed || m.actual != atBound || s.clock.read != reads+2 {
					t.Fatal("clock window allowed a third native read")
				}
			}
		})
	}
}
