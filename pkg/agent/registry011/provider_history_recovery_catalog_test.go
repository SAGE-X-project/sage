package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func historyRecoveryCatalogFixture(t *testing.T, mode string) (*providerHistoryGroupOwner, *providerHistoryRecoveryOwner, map[string]providerIntegrationObjectRef) {
	t.Helper()
	var o *providerHistoryGroupOwner
	switch mode {
	case "pending-admission", "pending-consumption":
		var upper providerHistoryGroupUpper
		o, upper = historyRecoveryBeforeAdmission(t)
		request := historyGroupSimple(t)
		if mode == "pending-consumption" {
			if ack, err := o.retainAdmissionCandidate(context.Background(), "selection", []*providerIntegrationRequest{request}, upper); err != nil || ack == nil {
				t.Fatal("genuine confirmed T before partial C", err)
			}
		}
		disk, reached := o.store.evidence, false
		syncFile := disk.syncFile
		disk.syncFile = func(file *os.File) error {
			if err := syncFile(file); err != nil {
				return err
			}
			c := o.store.frontierFiles.historyPending
			if c != nil && filepath.Base(file.Name()) == providerHistoryStoredName(c.inventory.entries[0], nil) {
				reached = true
				return errProviderIntegration // Owned original I/O uncertainty.
			}
			return nil
		}
		var err error
		if mode == "pending-admission" {
			_, err = o.retainAdmissionCandidate(context.Background(), "selection", []*providerIntegrationRequest{request}, upper)
		} else {
			_, err = o.retainConsumptionCandidate(context.Background(), o.last)
		}
		disk.syncFile = syncFile
		if err == nil || !reached || !o.failed || o.parent.recovery.pending == nil {
			t.Fatal("genuine partial original T/C write before catalog", err)
		}
	case "pending", "partial":
		o, _ = historyRecoveryUnacknowledgedS(t)
		if mode == "partial" {
			r := o.parent.recovery
			path := filepath.Join(r.config.custodyDirectory, providerCustodyName("highwater", r.pending.pin.ordinal))
			if raw, err := os.ReadFile(path); err != nil || len(raw) == 0 || os.WriteFile(path, nil, 0600) != nil {
				t.Fatal("owned genuine pending highwater precursor", err)
			}
		}
	default:
		var d *providerHistoryGroupDispatch
		o, d = historyRecoveryGroupFixture(t, 2)
		if mode == "settled" || mode == "members" {
			historyRetainNormalPrefix(t, o, d, 2)
			if mode == "settled" {
				if out, err := o.retainSettlementCandidate(context.Background(), d); err != nil || out == nil {
					t.Fatal("genuine original settled prefix", err)
				}
			}
		}
	}
	r, want := historyRecoveryGroupStart(t, o)
	return o, r, want
}

func historyRecoveryCatalogModels(t *testing.T, r *providerHistoryRecoveryOwner, want map[string]providerIntegrationObjectRef) {
	t.Helper()
	for pass := range 2 {
		historyRecoveryGroupInitial(t, r, pass, want)
		if out, err := r.replayGroupModelCandidate(context.Background(), pass); err != nil || out == nil || !out.selectedLocked(r) {
			t.Fatal("genuine fresh semantic prefix under original C0", pass, err)
		}
		if out, err := r.replayCompletedMemberPayloadsCandidate(context.Background(), pass); err != nil || out == nil || !out.selectedLocked(r) {
			t.Fatal("actual completed member payloads before catalog closure", pass, err)
		}
		if out, err := r.replayTailPayloadsCandidate(context.Background(), pass); err != nil || out == nil || !out.selectedLocked(r) {
			t.Fatal("actual incomplete and temporary payloads before catalog closure", pass, err)
		}
	}
}

func TestProviderHistoryRecoveryClosesWholeCatalogAroundBothActualModelPasses(t *testing.T) {
	for _, mode := range []string{"consumed", "members", "settled", "pending", "partial", "pending-admission", "pending-consumption"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryCatalogFixture(t, mode)
			old, reads, held, ack, pin := o.meter.actual, o.store.clock.read, o.store.stored, o.last, o.pin
			original := providerHistoryProvisionedOriginalBytes(t, r.config)
			catalogStart := r.meter.actual
			c0, err := r.beginCatalogCandidate(context.Background())
			if err != nil || c0 == nil || !c0.selectedLocked(r) || r.inputFiles != 0 || r.inputBytes != 0 || len(c0.entries) != len(original) {
				t.Fatal("actual two-root C0 must precede every payload input", err)
			}
			historyAssertObservedCatalogSourceUpper(t, r, catalogStart)
			historyRecoveryCatalogModels(t, r, want)
			files, bytes := r.inputFiles, r.inputBytes
			out, err := r.finishCatalogCandidate(context.Background())
			if err != nil || out == nil || !out.selectedLocked(r) || out.before != c0 || out.after == c0 || out.after != r.catalogAfter ||
				out.entries != uint64(len(original)) || out.names != c0.names || out.names != out.after.names || out.files != files+1 ||
				out.bytes != bytes+uint64(len(r.groupModels[0].head)) || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.last != ack || o.pin != pin ||
				o.store.activeMeter.Load() != nil || !reflect.DeepEqual(original, providerHistoryProvisionedOriginalBytes(t, r.config)) {
				t.Fatal("full names/metadata/HEAD closure changed old execution or skipped original input", mode, err)
			}
			if mode == "partial" && (r.groupModels[0].tailState != providerHistoryRecoveryPartialInput || r.groupModels[0].physical != r.confirmed) {
				t.Fatal("optional physical pending names promoted an incomplete S")
			}
			if (mode == "pending-admission" || mode == "pending-consumption") &&
				(r.groupModels[0].tailState != providerHistoryRecoveryMissingInput || r.groupModels[0].physical != r.confirmed) {
				t.Fatal("optional uncreated write slots promoted an incomplete T/C")
			}
		})
	}
}

func TestProviderHistoryRecoveryCatalogRefusesNonregularEntriesWithoutPublishingSnapshot(t *testing.T) {
	o, r, _ := historyRecoveryCatalogFixture(t, "consumed")
	if os.Mkdir(filepath.Join(r.config.evidenceDirectory, "owned-unexpected-directory"), 0700) != nil {
		t.Fatal("owned directory entry scenario")
	}
	old, reads, held, ack := o.meter.actual, o.store.clock.read, o.store.stored, o.last
	if out, err := r.beginCatalogCandidate(context.Background()); err == nil || out != nil || r.catalogBefore != nil || r.catalogAfter != nil || r.catalogResult != nil ||
		r.active || !r.closed || !r.failed || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.last != ack {
		t.Fatal("nonregular entry produced a partial complete snapshot or execution role", err)
	}
}

func TestProviderHistoryRecoveryCatalogRefusesLateCopiedRepeatedAndUnselectedClosure(t *testing.T) {
	o, r, want := historyRecoveryCatalogFixture(t, "consumed")
	work := r.meter.actual
	if out, err := r.finishCatalogCandidate(context.Background()); err == nil || out != nil || r.meter.actual != work || !r.active {
		t.Fatal("namespace closure without C0 reached observation")
	}
	c0, err := r.beginCatalogCandidate(context.Background())
	if err != nil || c0 == nil {
		t.Fatal("original complete C0", err)
	}
	work = r.meter.actual
	if out, err := r.beginCatalogCandidate(context.Background()); err == nil || out != nil || r.meter.actual != work || !r.active {
		t.Fatal("repeated C0 renewed its snapshot or observer")
	}
	copy := *c0
	copy.owner = &copy
	if copy.selectedLocked(r) || json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryCatalog{}) == nil ||
		json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryCatalogResult{}) == nil {
		t.Fatal("copied or JSON DATA replaced an original catalog")
	}
	if out, err := r.finishCatalogCandidate(context.Background()); err == nil || out != nil || r.meter.actual != work || !r.active {
		t.Fatal("C1 before two fresh complete models reached observation")
	}
	historyRecoveryCatalogModels(t, r, want)
	result, err := r.finishCatalogCandidate(context.Background())
	if err != nil || result == nil || !result.selectedLocked(r) {
		t.Fatal("positive complete closure after isolated entry refusals", err)
	}
	resultCopy := *result
	resultCopy.owner = &resultCopy
	if resultCopy.selectedLocked(r) {
		t.Fatal("copied closure manufactured selected DATA")
	}
	work = r.meter.actual
	if out, err := r.finishCatalogCandidate(context.Background()); err == nil || out != nil || r.meter.actual != work || !r.active {
		t.Fatal("repeated C1 reset or renewed the recovery lane")
	}
	if !r.meter.event(providerCompareEvent, 0) || result.selectedLocked(r) || o.store.activeMeter.Load() != nil {
		t.Fatal("a closure remained current after later observed work")
	}
	_, late, _ := historyRecoveryCatalogFixture(t, "consumed")
	if out, err := late.readCheckpointInputCandidate(context.Background(), late.initial, providerHistoryRecoveryHead, 0); err != nil || out == nil {
		t.Fatal("genuine selected payload before late C0 refusal", err)
	}
	work, files := late.meter.actual, late.inputFiles
	if out, err := late.beginCatalogCandidate(context.Background()); err == nil || out != nil || late.meter.actual != work || late.inputFiles != files || !late.active {
		t.Fatal("a late catalog was substituted for a pre-payload snapshot")
	}
}

func TestProviderHistoryRecoveryCatalogRejectsUnknownNamesAndPostReadNamespaceChanges(t *testing.T) {
	for _, mode := range []string{"unknown-before", "unknown-after", "missing-after", "changed-after"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryCatalogFixture(t, "consumed")
			unknown := filepath.Join(r.config.evidenceDirectory, "owned-unexpected-tail")
			if mode == "unknown-before" && os.WriteFile(unknown, []byte("inert"), 0600) != nil {
				t.Fatal("owned unknown regular-file precursor")
			}
			if c0, err := r.beginCatalogCandidate(context.Background()); err != nil || c0 == nil {
				t.Fatal("complete C0 including observed DATA names", err)
			}
			historyRecoveryCatalogModels(t, r, want)
			path := filepath.Join(r.config.evidenceDirectory, "ANCHOR")
			switch mode {
			case "unknown-after":
				if os.WriteFile(unknown, []byte("inert"), 0600) != nil {
					t.Fatal("owned new tail after payloads")
				}
			case "missing-after":
				if os.Remove(path) != nil {
					t.Fatal("owned missing original after payloads")
				}
			case "changed-after":
				if file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600); err != nil {
					t.Fatal("owned original metadata scenario", err)
				} else {
					_, writeErr := file.Write([]byte("!"))
					closeErr := file.Close()
					if writeErr != nil || closeErr != nil {
						t.Fatal("owned original change", writeErr, closeErr)
					}
				}
			}
			old, reads, held, ack := o.meter.actual, o.store.clock.read, o.store.stored, o.last
			out, err := r.finishCatalogCandidate(context.Background())
			if err == nil || out != nil || r.catalogAfter != nil || r.catalogResult != nil || r.active || !r.failed || !r.closed ||
				o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.last != ack {
				t.Fatal("unknown/changed/missing namespace supplied a complete catalog or renewed authority", mode, err)
			}
		})
	}
}

func TestProviderHistoryRecoveryCatalogRejectsChangedOriginalBeforePayloadAndMalformedHead(t *testing.T) {
	for _, mode := range []string{"inventory-after-C0", "head-empty", "head-missing"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryCatalogFixture(t, "consumed")
			path := filepath.Join(r.config.custodyDirectory, "HEAD")
			if mode == "head-empty" && os.WriteFile(path, nil, 0600) != nil || mode == "head-missing" && os.Remove(path) != nil {
				t.Fatal("owned incomplete HEAD scenario")
			}
			if out, err := r.beginCatalogCandidate(context.Background()); err != nil || out == nil {
				t.Fatal("actual C0 before payloads", err)
			}
			old, reads, held, ack := o.meter.actual, o.store.clock.read, o.store.stored, o.last
			if mode == "inventory-after-C0" {
				path = filepath.Join(r.config.evidenceDirectory, "inventory-"+r.initialDescriptor.objects[2:])
				if raw, err := os.ReadFile(path); err != nil || os.WriteFile(path, append(raw, '!'), 0600) != nil {
					t.Fatal("owned selected inventory metadata change", err)
				}
				w, err := r.newInventoryWalkCandidate(context.Background(), 0)
				if err != nil || w == nil {
					t.Fatal("genuine selected inventory reader", err)
				}
				if out, err := w.nextInventoryCandidate(context.Background()); err == nil || out != nil || r.inputFiles != 1 || r.inputBytes != 0 {
					t.Fatal("changed C0 metadata reached content or a selected frame", err)
				}
			} else {
				historyRecoveryGroupInitial(t, r, 0, want)
				if out, err := r.replayGroupModelCandidate(context.Background(), 0); err == nil || out != nil || r.groupModels[0] != nil {
					t.Fatal("missing/empty actual HEAD became a complete semantic prefix", err)
				}
			}
			if r.active || !r.closed || !r.failed || r.catalogResult != nil || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.last != ack {
				t.Fatal("incomplete or changed originals resumed execution")
			}
		})
	}
}

func TestProviderHistoryRecoveryCatalogCancellationKeepsOriginalsAndClosesWithoutResult(t *testing.T) {
	for _, phase := range []string{"before", "after"} {
		t.Run(phase, func(t *testing.T) {
			o, r, want := historyRecoveryCatalogFixture(t, "consumed")
			if phase == "after" {
				if out, err := r.beginCatalogCandidate(context.Background()); err != nil || out == nil {
					t.Fatal("original complete C0", err)
				}
				historyRecoveryCatalogModels(t, r, want)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			work, files, bytes, old, reads := r.meter.actual, r.inputFiles, r.inputBytes, o.meter.actual, o.store.clock.read
			var err error
			if phase == "before" {
				_, err = r.beginCatalogCandidate(ctx)
			} else {
				_, err = r.finishCatalogCandidate(ctx)
			}
			if err == nil || r.catalogResult != nil || r.catalogAfter != nil || r.active || !r.closed || !r.failed || r.meter.actual != work ||
				r.inputFiles != files || r.inputBytes != bytes || o.meter.actual != old || o.store.clock.read != reads {
				t.Fatal("cancelled original catalog reached observation or a published result", phase, err)
			}
		})
	}
}
