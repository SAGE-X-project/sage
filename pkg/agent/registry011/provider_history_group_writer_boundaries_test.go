package registry011

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProviderHistoryLedgerCopiesKeepSeparateActiveRecords(t *testing.T) {
	if providerHistoryLedgerCopy(nil) != nil || providerHistoryLedgerCopy(&providerHistoryGroupLedger{}) != nil {
		t.Fatal("unowned model became an independent replay model")
	}
	l := historyLedgerFixture(t, historyLedgerMaximum())
	ticket := historyRecordTicket(t, 1, 1, l.head)
	if err := l.admit(ticket); err != nil {
		t.Fatal(err)
	}
	consumption, err := newProviderHistoryGroupConsumption(ticket, l.head, admissionTestDigest("1"))
	if err != nil || l.consume(consumption) != nil {
		t.Fatal("pure transcript fixture", err)
	}
	copy := providerHistoryLedgerCopy(l)
	if copy == nil || copy == l || copy.owner != copy || copy.head != l.head || copy.charged != l.charged || copy.ticket == l.ticket || copy.consumption == l.consumption ||
		copy.ticket.digest != l.ticket.digest || copy.consumption.digest != l.consumption.digest {
		t.Fatal("active records were shared or transcript values changed")
	}
	copy.ticket.digest, copy.consumption.digest = admissionTestDigest("2"), admissionTestDigest("3")
	if l.ticket.digest != ticket.digest || l.consumption.digest != consumption.digest {
		t.Fatal("changing the diagnostic copy changed the original model")
	}
}

func TestProviderHistoryAdmissionWriterKeepsItsHoldWhenClockCallsAreUnavailable(t *testing.T) {
	var absent *providerHistoryGroupOwner
	if ack, err := absent.retainAdmissionCandidate("selection", nil, providerHistoryGroupUpper{}); err == nil || ack != nil {
		t.Fatal("absent writer returned authority")
	}
	for _, allowed := range []uint64{0, 1} {
		t.Run([]string{"begin", "end"}[allowed], func(t *testing.T) {
			o, upper := historyAdmissionWriterFixture(t)
			s, pin, held, raw, reads := o.store, o.pin, o.store.stored, o.store.retainedRaw, o.store.clock.read
			// Only reduce the existing test parent's available calls. Never
			// replace its backend, epoch, observer or cumulative counters.
			o.parent.clockCeiling = reads + allowed
			if ack, err := o.retainAdmissionCandidate("selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper); err == nil || ack != nil || o.last != nil ||
				!o.failed || !o.meter.failed || !o.meter.closed || !s.failed || !s.clock.historyRevoked || s.activeMeter.Load() != nil ||
				o.pin != pin || s.frontierFiles.pin != pin || s.stored != held || s.retainedRaw != raw || s.clock.read != reads+allowed {
				t.Fatal("native call refusal published a pin or renewed the original hold")
			}
			pending := s.frontierFiles.historyPending
			if allowed == 0 && (pending != nil || o.retention != (providerHistoryStorageCharge{})) ||
				allowed == 1 && (pending == nil || !pending.renamed || pending.used || o.retention.physical == 0) {
				t.Fatal("clock refusal lost its pre-write or physically renamed boundary")
			}
		})
	}
}

func TestProviderHistoryAdmissionReadbackRefusesOwnedOriginalAndContextFaults(t *testing.T) {
	for _, mode := range []string{"old-head", "request-missing", "request-changed", "inventory-missing", "inventory-changed", "request-context", "ticket-context", "model", "frame", "metadata-write", "directory-sync", "late-original"} {
		t.Run(mode, func(t *testing.T) {
			o, upper := historyAdmissionWriterFixture(t)
			s, pin, held, raw := o.store, o.pin, o.store.stored, o.store.retainedRaw
			injected := false
			inject := func(c *providerHistoryWriteClaim) error {
				path := filepath.Join(s.config.evidenceDirectory, providerHistoryStoredName(c.request, nil))
				switch mode {
				case "old-head":
					path = filepath.Join(s.config.custodyDirectory, "HEAD")
				case "inventory-missing", "inventory-changed":
					path = filepath.Join(s.config.evidenceDirectory, providerHistoryInventoryName(c.inventory.digest, nil))
				case "frame":
					path = filepath.Join(s.config.evidenceDirectory, providerCustodyName("history-pending", c.frame.after.sequence))
				case "request-context":
					c.purpose = "inert mismatched purpose"
					return nil
				case "ticket-context":
					c.upper.custody[10]++
					return nil
				case "model":
					c.coordinator.head = admissionTestDigest("f")
					return nil
				}
				if mode == "request-missing" || mode == "inventory-missing" {
					return os.Remove(path)
				}
				body, err := os.ReadFile(path)
				if err != nil || len(body) == 0 {
					return errProviderIntegration
				}
				body[len(body)/2] ^= 1
				return os.WriteFile(path, body, 0600)
			}
			if mode == "directory-sync" {
				syncFile := s.evidence.syncFile
				s.evidence.syncFile = func(file *os.File) error {
					info, err := file.Stat()
					if err != nil {
						return err
					}
					if info.IsDir() {
						injected = true
						return errProviderIntegration
					}
					return syncFile(file)
				}
			} else {
				disk := s.custody
				if mode == "metadata-write" {
					disk = s.evidence
				}
				syncFile := disk.syncFile
				disk.syncFile = func(file *os.File) error {
					if err := syncFile(file); err != nil {
						return err
					}
					c := s.frontierFiles.historyPending
					if c == nil || injected {
						return nil
					}
					if mode == "metadata-write" {
						if filepath.Base(file.Name()) != providerHistoryStoredName(c.request, nil) {
							return nil
						}
						injected = true
						return os.WriteFile(filepath.Join(s.config.custodyDirectory, providerCustodyName("commit", c.frame.after.sequence)), []byte("inert exclusive-write conflict"), 0600)
					}
					if mode == "late-original" {
						head, err := os.ReadFile(filepath.Join(s.config.custodyDirectory, "HEAD"))
						if err != nil || string(head) != c.highwater {
							return nil
						}
					} else if filepath.Base(file.Name()) != providerCustodyName("HEAD.pending", c.frame.after.ordinal) {
						return nil
					}
					injected = true
					return inject(c)
				}
			}
			// Every mutation is finite inert data in this fixture's own roots.
			// No peer, external path, transaction or reusable replay tool exists.
			if ack, err := o.retainAdmissionCandidate("selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper); err == nil || ack != nil || o.last != nil || !injected ||
				!o.failed || !s.failed || !o.meter.failed || !o.meter.closed || !s.clock.historyRevoked || s.activeMeter.Load() != nil ||
				o.pin != pin || s.frontierFiles.pin != pin || s.stored != held || s.retainedRaw != raw || o.retention.physical == 0 ||
				s.frontierFiles.historyPending == nil || s.frontierFiles.historyPending.used {
				t.Fatal("readback failure published authority, bypassed its boundary or refunded the retained claim")
			}
			if (mode == "late-original") != s.frontierFiles.historyPending.renamed {
				t.Fatal("refusal confused the physical rename with the unpublished logical pin")
			}
		})
	}
}

func TestProviderHistoryAdmissionPrivateHelpersRefuseDetachedInputs(t *testing.T) {
	o, upper := historyAdmissionWriterFixture(t)
	s, checked := o.store, false
	syncFile := s.custody.syncFile
	s.custody.syncFile = func(file *os.File) error {
		if err := syncFile(file); err != nil {
			return err
		}
		c := s.frontierFiles.historyPending
		if checked || c == nil || filepath.Base(file.Name()) != providerCustodyName("HEAD.pending", c.frame.after.ordinal) {
			return nil
		}
		checked = true
		before, clocks := o.meter.actual, s.clock.read
		copy := *c
		if o.writeAdmissionLocked(&copy) == nil {
			t.Fatal("copied claim entered an original write")
		}
		if model, err := o.readAdmissionLocked(&copy, false); err == nil || model != nil || o.historyDiskIdentityLocked(nil) || o.writeHistoryFileLocked(s.evidence, "", nil) == nil ||
			o.meter.actual != before || s.clock.read != clocks {
			t.Fatal("detached helper input consumed an event or supplied a model")
		}
		headPath := filepath.Join(s.config.custodyDirectory, "HEAD")
		head, err := os.ReadFile(headPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(headPath, []byte("inert mismatched old HEAD"), 0600); err != nil {
			return err
		}
		err = o.writeAdmissionLocked(c)
		if restoreErr := os.WriteFile(headPath, head, 0600); restoreErr != nil {
			return restoreErr
		}
		if err == nil || o.meter.failed || s.clock.read != clocks || c.used || c.renamed {
			t.Fatal("old-HEAD refusal changed the live claim or failed observer")
		}
		return nil
	}
	if ack, err := o.retainAdmissionCandidate("selection", []*providerIntegrationRequest{historyGroupSimple(t)}, upper); err != nil || ack == nil || !checked {
		t.Fatal("separate genuine admission positive control", err)
	}
}
