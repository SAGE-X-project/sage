package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func historyRecoveryScanSource(t *testing.T, r *providerHistoryRecoveryOwner, pass int, want map[string]providerIntegrationObjectRef) uint64 {
	t.Helper()
	w, err := r.newInventoryWalkCandidate(context.Background(), pass)
	if err != nil {
		t.Fatal("genuine independent original inventory source", err)
	}
	_, bytes := historyRecoveryInventoryDrain(t, w, want)
	if _, err := w.finishCandidate(context.Background()); err != nil || w.scanBinding == nil || w.scanBinding.raw == "" {
		t.Fatal("actual complete anchored source with separately retained scan binding", err)
	}
	return bytes
}

func historyRecoveryScanFixture(t *testing.T) (*providerHistoryGroupOwner, *providerHistoryRecoveryOwner, *providerHistoryRecoveryScanWalk, map[string]providerIntegrationObjectRef) {
	t.Helper()
	o, r, want := historyRecoveryInventoryFixture(t)
	historyRecoveryScanSource(t, r, 0, want)
	w, err := r.newScanWalkCandidate(context.Background(), 0)
	if err != nil || w == nil || w.input != 0 || w.remaining != r.initialDescriptor.records || w.controlBytes == 0 {
		t.Fatal("genuine actual scan-control reconstruction before receipt reading", err)
	}
	return o, r, w, want
}

func TestProviderHistoryRecoveryScanReadsBothOriginalChainsWithoutRenewingClockOrBudget(t *testing.T) {
	o, r, want := historyRecoveryInventoryFixture(t)
	s := o.store
	old, reads, held, pin, ack := o.meter.actual, s.clock.read, s.stored, o.pin, o.last
	before := providerHistoryProvisionedOriginalBytes(t, s.config)
	var bytes uint64
	for pass := range 2 {
		bytes += historyRecoveryScanSource(t, r, pass, want)
		w, err := r.newScanWalkCandidate(context.Background(), pass)
		if err != nil || w == nil || w.source != r.inventoryWalks[pass] || w.codec.meter != r.meter || w.input != 0 || w.complete ||
			pass == 1 && (w == r.scanWalks[0] || w.codec == r.scanWalks[0].codec || w.source.scanBinding == r.scanWalks[0].source.scanBinding) {
			t.Fatal("fresh actual scan binding/control pass", pass, err)
		}
		for !w.complete {
			remaining, digest := w.remaining, w.next
			record, err := w.nextRecordCandidate(context.Background())
			if err != nil || record == nil || record.index != remaining-1 || w.remaining != remaining-1 || w.input != 0 && !w.complete {
				t.Fatal("actual descending record/command chain", err)
			}
			if providerCandidateHash(providerScanChunkV2, []byte(record.chunk)) != digest || record.state != "" && !lowerHash(record.state) || !lowerHash(record.tickets) || !lowerHash(record.operations) || !lowerHash(record.segments) {
				t.Fatal("returned retained diagnostic is not its anchored original")
			}
			fields, ok := contentJSON([]byte(record.chunk)).(map[string]any)
			if !ok {
				t.Fatal("owned unit chunk oracle")
			}
			commandDigest := contentString(fields["command_digest"])
			for _, directory := range []string{s.config.custodyDirectory, s.config.evidenceDirectory} {
				chunk, a := os.ReadFile(filepath.Join(directory, providerScanReceiptName("chunk", digest, nil)))
				command, b := os.ReadFile(filepath.Join(directory, providerScanReceiptName("command", commandDigest, nil)))
				if a != nil || b != nil || string(chunk) != record.chunk || string(command) != record.command {
					t.Fatal("both actual originals differ from returned DATA", a, b)
				}
			}
			// Returned DATA cannot change internal cursor, predecessor or counts.
			record.index, record.chunk, record.command = ^uint64(0), "inert", "inert"
		}
		if w.result.records != r.initialDescriptor.records || w.result.metadata == 0 || w.next != "" || w.remaining != 0 || w.input != 4 ||
			w.wantObjects != 0 || w.wantBytes != 0 || w.wantWork != (candidateWorkVector{}) || w.cursor != (providerScanCursor{"setup", 0}) || !w.selectedLocked(r) {
			t.Fatal("scan closure did not terminate at original zero counters/setup cursor")
		}
		bytes += w.controlBytes + w.result.metadata
		work, files := r.meter.actual, r.inputFiles
		if out, err := w.nextRecordCandidate(context.Background()); err == nil || out != nil || r.meter.actual != work || r.inputFiles != files || !r.active {
			t.Fatal("completed scan chain was re-read or acquired another allowance", err)
		}
	}
	n := r.initialDescriptor
	wantFiles := 2 * (4 + uint64(len(want)) + 4 + 5*n.scan.ordinal + 4*n.records)
	if r.inputFiles != wantFiles || r.inputFiles > r.bounds.maximumFiles || r.inputBytes != bytes || o.meter.actual != old || s.clock.read != reads || s.stored != held || o.pin != pin || o.last != ack ||
		!o.failed || !o.meter.closed || s.activeMeter.Load() != nil || !r.active || r.failed || r.closed || !reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("actual initial scan changed old execution, originals or cumulative read-only quota")
	}
	if json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryScanWalk{}) == nil {
		t.Fatal("serialized diagnostics manufactured a selected scan reader")
	}
}

func TestProviderHistoryRecoveryScanRefusesDetachedAndOutOfOrderRolesBeforeIO(t *testing.T) {
	o, r, want := historyRecoveryInventoryFixture(t)
	work := r.meter.actual
	if out, err := r.newScanWalkCandidate(context.Background(), 0); err == nil || out != nil || r.meter.actual != work || r.inputFiles != 0 || !r.active {
		t.Fatal("scan selected without actual original inventory closure", err)
	}
	historyRecoveryScanSource(t, r, 0, want)
	w, err := r.newScanWalkCandidate(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	historyRecoveryScanSource(t, r, 1, want)
	for _, mode := range []string{"copy", "source-copy", "binding-copy", "duplicate-pass", "hybrid", "orphan-object", "second-before-first", "negative-pass", "excess-pass"} {
		t.Run(mode, func(t *testing.T) {
			actual, files, bytes, old, reads := r.meter.actual, r.inputFiles, r.inputBytes, o.meter.actual, o.store.clock.read
			var err error
			switch mode {
			case "copy":
				copy := *w
				copy.owner = &copy
				_, err = copy.nextRecordCandidate(context.Background())
			case "source-copy":
				source := w.source
				copy := *source
				copy.owner = &copy
				w.source = &copy
				_, err = w.nextRecordCandidate(context.Background())
				w.source = source
			case "duplicate-pass":
				_, err = r.newScanWalkCandidate(context.Background(), 0)
			case "binding-copy":
				binding := w.source.scanBinding
				copy := *binding
				w.source.scanBinding = &copy
				_, err = w.nextRecordCandidate(context.Background())
				w.source.scanBinding = binding
			case "hybrid":
				r.store.mu.Lock()
				_, err = r.readSelectedInputLocked(context.Background(), providerHistoryRecoveryInputSelection{scan: w, inventory: w.source})
				r.store.mu.Unlock()
			case "orphan-object":
				r.store.mu.Lock()
				_, err = r.readSelectedInputLocked(context.Background(), providerHistoryRecoveryInputSelection{scan: w, object: true})
				r.store.mu.Unlock()
			case "second-before-first":
				_, err = r.newScanWalkCandidate(context.Background(), 1)
			case "negative-pass":
				_, err = r.newScanWalkCandidate(context.Background(), -1)
			case "excess-pass":
				_, err = r.newScanWalkCandidate(context.Background(), 2)
			}
			if err == nil || r.meter.actual != actual || r.inputFiles != files || r.inputBytes != bytes || o.meter.actual != old || o.store.clock.read != reads || !r.active || r.closed {
				t.Fatal("detached scan role spent the row or reached original files", mode, err)
			}
		})
	}
	if out, err := w.nextRecordCandidate(context.Background()); err != nil || out == nil {
		t.Fatal("otherwise-valid actual scan after isolated role refusals", err)
	}
}

func TestProviderHistoryRecoveryScanRejectsChangedActualControlBeforeRegistration(t *testing.T) {
	for _, name := range []string{"head-custody", "head-evidence", "h0-custody", "h0-evidence", "intent", "claim", "commit", "hk-custody", "hk-evidence"} {
		t.Run(name, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixture(t)
			historyRecoveryScanSource(t, r, 0, want)
			n := r.initialDescriptor
			directory, selected := r.config.custodyDirectory, "SCAN-HEAD"
			switch name {
			case "head-evidence":
				directory = r.config.evidenceDirectory
			case "h0-custody":
				selected = providerCustodyName("scan-highwater", 0)
			case "h0-evidence":
				directory, selected = r.config.evidenceDirectory, providerCustodyName("scan-highwater", 0)
			case "intent":
				selected = providerCustodyName("scan-intent", n.scan.ordinal)
			case "claim":
				directory, selected = r.config.evidenceDirectory, providerCustodyName("scan-claim", n.scan.ordinal)
			case "commit":
				selected = providerCustodyName("scan-commit", n.scan.ordinal)
			case "hk-custody":
				selected = providerCustodyName("scan-highwater", n.scan.ordinal)
			case "hk-evidence":
				directory, selected = r.config.evidenceDirectory, providerCustodyName("scan-highwater", n.scan.ordinal)
			}
			path := filepath.Join(directory, selected)
			raw, err := os.ReadFile(path)
			if err != nil || len(raw) == 0 {
				t.Fatal("actual original control precursor", err)
			}
			for i := range raw {
				raw[i] = '!'
			}
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			old, reads, held, pin, ack, files := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last, r.inputFiles
			out, err := r.newScanWalkCandidate(context.Background(), 0)
			if err == nil || out != nil || r.scanWalks[0] != nil || !r.closed || !r.failed || r.active || r.inputFiles <= files || o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.pin != pin || o.last != ack {
				t.Fatal("changed actual control registered a scan reader or revived execution", name, err)
			}
		})
	}
}

func TestProviderHistoryRecoveryScanRejectsMissingOrChangedReceiptCopiesWithoutAdvancing(t *testing.T) {
	for _, kind := range []string{"chunk", "command"} {
		for _, evidence := range []bool{false, true} {
			for _, missing := range []bool{false, true} {
				name := kind
				if evidence {
					name += "/evidence"
				} else {
					name += "/custody"
				}
				if missing {
					name += "/missing"
				} else {
					name += "/different"
				}
				t.Run(name, func(t *testing.T) {
					o, r, w, _ := historyRecoveryScanFixture(t)
					digest, probes := w.next, uint64(1)
					if kind == "command" {
						raw, err := os.ReadFile(filepath.Join(r.config.custodyDirectory, providerScanReceiptName("chunk", w.next, nil)))
						fields, ok := contentJSON(raw).(map[string]any)
						if err != nil || !ok {
							t.Fatal("actual owned chunk oracle", err)
						}
						digest, probes = contentString(fields["command_digest"]), 3
					}
					directory := r.config.custodyDirectory
					if evidence {
						directory, probes = r.config.evidenceDirectory, probes+1
					}
					path := filepath.Join(directory, providerScanReceiptName(kind, digest, nil))
					raw, err := os.ReadFile(path)
					if err != nil || len(raw) == 0 {
						t.Fatal("actual original receipt precursor", err)
					}
					// Inert files in this test's exclusively owned temporary roots.
					// No wire attack, host bypass, symlink or race helper is used.
					if missing {
						err = os.Remove(path)
					} else {
						for i := range raw {
							raw[i] = '!'
						}
						err = os.WriteFile(path, raw, 0600)
					}
					if err != nil {
						t.Fatal(err)
					}
					old, reads, held, pin, ack, files := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last, r.inputFiles
					remaining, next := w.remaining, w.next
					out, err := w.nextRecordCandidate(context.Background())
					if err == nil || out != nil || !r.closed || !r.failed || r.active || r.inputFiles != files+probes || w.remaining != remaining || w.next != next || w.have || w.complete || w.result != (providerScanReceiptClosure{}) ||
						o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.pin != pin || o.last != ack || o.store.activeMeter.Load() != nil {
						t.Fatal("partial physical receipt pair advanced semantic prefix or revived execution", err)
					}
				})
			}
		}
	}
}

func TestProviderHistoryRecoveryScanCancellationStopsBeforeNextEffect(t *testing.T) {
	for _, phase := range []string{"new", "new-nil", "record", "record-nil"} {
		t.Run(phase, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixture(t)
			historyRecoveryScanSource(t, r, 0, want)
			var w *providerHistoryRecoveryScanWalk
			if phase == "record" || phase == "record-nil" {
				var err error
				w, err = r.newScanWalkCandidate(context.Background(), 0)
				if err != nil {
					t.Fatal(err)
				}
			}
			var ctx context.Context
			if phase == "new" || phase == "record" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			}
			work, files, bytes, old, reads := r.meter.actual, r.inputFiles, r.inputBytes, o.meter.actual, o.store.clock.read
			var err error
			if w == nil {
				_, err = r.newScanWalkCandidate(ctx, 0)
			} else {
				_, err = w.nextRecordCandidate(ctx)
			}
			if err == nil || !r.closed || !r.failed || r.active || r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes || o.meter.actual != old || o.store.clock.read != reads {
				t.Fatal("cancelled selected scan action reached an effect or renewed its row", err)
			}
		})
	}
}
