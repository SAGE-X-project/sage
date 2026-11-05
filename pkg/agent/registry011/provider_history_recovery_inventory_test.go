package registry011

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func historyRecoveryInventoryFixture(t *testing.T) (*providerHistoryGroupOwner, *providerHistoryRecoveryOwner, map[string]providerIntegrationObjectRef) {
	return historyRecoveryInventoryFixtureWithLegacy(t, false)
}

func historyRecoveryInventoryFixtureWithLegacy(t *testing.T, legacy bool) (*providerHistoryGroupOwner, *providerHistoryRecoveryOwner, map[string]providerIntegrationObjectRef) {
	t.Helper()
	o, _ := historyMemberDispatchFixtureWithRecoverySelection(t, 1, nil, true, legacy)
	want := make(map[string]providerIntegrationObjectRef, len(o.store.objects))
	for key, ref := range o.store.objects {
		want[key] = ref
	}
	r := o.parent.recovery
	if len(want) == 0 || uint64(len(want)) != r.initialDescriptor.objectCount || r.initialDescriptor.main.sequence != 4 {
		t.Fatal("genuine original sealed four-inventory reference prefix")
	}
	o.store.mu.Lock()
	o.failLocked()
	// The failed producer's object cache is deliberately unavailable. This is
	// private unit DATA, not a retained file or replacement live owner.
	o.store.objects = nil
	o.store.mu.Unlock()
	if r.beginInspectionCandidate(context.Background(), o) != nil {
		t.Fatal("genuine original terminal inventory reader")
	}
	return o, r, want
}

func historyRecoveryInventoryDrain(t *testing.T, w *providerHistoryRecoveryInventoryWalk, want map[string]providerIntegrationObjectRef) (uint64, uint64) {
	t.Helper()
	seen := make(map[string]bool)
	var files, bytes uint64
	for w.remaining > 0 {
		sequence := w.remaining
		record, err := w.nextInventoryCandidate(context.Background())
		if err != nil || record == nil || record.sequence != sequence || record.objects == 0 || w.current == nil {
			t.Fatal("actual anchored inventory and original descending sequence", err)
		}
		raw, err := os.ReadFile(filepath.Join(w.recovery.config.evidenceDirectory, "inventory-"+record.digest[2:]))
		if err != nil || string(raw) != record.raw || providerCandidateHash(providerHistoryRecoveryInventoryDomain, raw) != record.digest {
			t.Fatal("inventory result differs from actual retained original", err)
		}
		files++
		bytes += uint64(len(raw))
		count, previous := record.objects, record.previous
		record.previous, record.raw, record.objects = "inert returned DATA mutation", "inert", 0
		for range count {
			object, err := w.nextObjectCandidate(context.Background())
			if err != nil || object == nil {
				t.Fatal("actual object selected only by validated inventory bytes", err)
			}
			key := providerIntegrationRefKey(providerIntegrationObjectRef{kind: object.kind, digest: object.digest})
			ref, exists := want[key]
			if !exists || seen[key] || uint64(len(object.raw)) != ref.bytes {
				t.Fatal("initial closure selected a cache entry, duplicate or wrong object length")
			}
			actual, err := os.ReadFile(filepath.Join(w.recovery.config.evidenceDirectory, "object-"+object.kind+"-"+object.digest[2:]))
			if err != nil || string(actual) != object.raw {
				t.Fatal("object DATA differs from actual retained original", err)
			}
			files++
			bytes += uint64(len(actual))
			seen[key] = true
			object.raw, object.kind = "inert returned DATA mutation", "inert"
		}
		if w.current != nil || w.index != 0 || w.next != previous || w.remaining != sequence-1 {
			t.Fatal("returned DATA mutation changed the next anchored selection")
		}
	}
	if len(seen) != len(want) || len(w.seen) != len(want) || w.complete {
		t.Fatal("reference walk skipped or prematurely completed the original closure")
	}
	return files, bytes
}

func TestProviderHistoryRecoveryInventoryReplaysActualClosureTwiceWithoutProducerCache(t *testing.T) {
	o, r, want := historyRecoveryInventoryFixture(t)
	s := o.store
	old, reads, held, pin, ack := o.meter.actual, s.clock.read, s.stored, o.pin, o.last
	originals := providerHistoryProvisionedOriginalBytes(t, s.config)
	manifest, bounds := s.config.bounds.history.collection, s.config.bounds
	quote, quoteOK := providerHistoryReconstructionInventoryUpper(providerHistoryLegacyControlLimits{
		bounds.maximumMetadataPerCommit, bounds.maximumObjectBytes, manifest.input.firstRead.objects, manifest.input.firstRead.raw,
		uint64(bounds.maximumEntries), bounds.scan.maximumAttempts, bounds.scan.maximumChunks,
	})
	if !quoteOK {
		t.Fatal("original immutable inventory limits")
	}
	readBefore := r.meter.actual
	var files, bytes, decoded uint64
	for pass := range 2 {
		work := r.meter.actual
		w, err := r.newInventoryWalkCandidate(context.Background(), pass)
		if err != nil || w == nil || w.owner != w || w.descriptor != r.initialDescriptor || w.next != r.initialDescriptor.objects || w.remaining != 4 || len(w.seen) != 0 {
			t.Fatal("fresh original-anchored inventory pass", pass, err)
		}
		if pass == 1 && (w == r.inventoryWalks[0] || &w.seen[:cap(w.seen)][0] == &r.inventoryWalks[0].seen[0]) {
			t.Fatal("second pass reused first-pass mutable reference storage")
		}
		f, b := historyRecoveryInventoryDrain(t, w, want)
		files, bytes = files+f, bytes+b
		// Only inventory/object codecs debit coordinate2 in this subgraph;
		// file/name/catalog I/O is coordinate10 and is deliberately excluded
		// from this runtime attribution. Use each pass's actual fresh frames.
		for _, frame := range w.frames {
			decoded += 2 * uint64(len(frame.inventory.raw))
			for _, object := range frame.objects {
				domain, known := providerIntegrationDomain(object.kind)
				if !known {
					t.Fatal("actual original had an unknown closed object kind")
				}
				if domain != "" {
					decoded += uint64(len(object.raw))
				}
			}
		}
		summary, err := w.finishCandidate(context.Background())
		if err != nil || summary == nil || summary.root != r.initialDescriptor.objects || summary.inventories != 4 || summary.objects != uint64(len(want)) ||
			summary.bytes != w.bytes || !w.complete || r.inputFiles != files || r.inputBytes != bytes || r.meter.actual[10] <= work[10] {
			t.Fatal("full original reference closure or cumulative shared row was not preserved", err)
		}
	}
	if r.meter.actual[2] < readBefore[2] || decoded == 0 || r.meter.actual[2]-readBefore[2] != decoded || decoded > quote.normal[2] {
		t.Fatal("two fresh original decode streams exceeded their same-source row or borrowed logical I/O slack", decoded)
	}
	if r.inputFiles != 2*(4+uint64(len(want))) || r.inputBytes == 0 || r.meter.actual[2] == 0 || o.meter.actual != old || s.clock.read != reads || s.stored != held ||
		o.pin != pin || o.last != ack || !o.failed || !o.meter.closed || s.activeMeter.Load() != nil || !r.active || r.closed || r.failed || s.objects != nil ||
		!reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, s.config)) {
		t.Fatal("inventory replay renewed execution, read its producer cache, changed originals or reset its row")
	}
}

func TestProviderHistoryRecoveryInventoryRefusesDetachedAndOutOfOrderRolesBeforeIO(t *testing.T) {
	o, r, _ := historyRecoveryInventoryFixture(t)
	for _, pass := range []int{-1, 1, 2} {
		work := r.meter.actual
		if w, err := r.newInventoryWalkCandidate(context.Background(), pass); err == nil || w != nil || r.meter.actual != work || r.inputFiles != 0 || !r.active {
			t.Fatal("unselected pass spent or renewed the original row", pass, err)
		}
	}
	w, err := r.newInventoryWalkCandidate(context.Background(), 0)
	if err != nil || w == nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"copy", "object-before-inventory", "finish-before-inventory", "duplicate-pass", "hybrid-selection", "orphan-object"} {
		t.Run(mode, func(t *testing.T) {
			work, files, bytes, old, reads := r.meter.actual, r.inputFiles, r.inputBytes, o.meter.actual, o.store.clock.read
			var err error
			switch mode {
			case "copy":
				copy := *w
				copy.owner = &copy
				_, err = copy.nextInventoryCandidate(context.Background())
			case "object-before-inventory":
				_, err = w.nextObjectCandidate(context.Background())
			case "finish-before-inventory":
				_, err = w.finishCandidate(context.Background())
			case "duplicate-pass":
				_, err = r.newInventoryWalkCandidate(context.Background(), 0)
			case "hybrid-selection":
				r.store.mu.Lock()
				_, err = r.readSelectedInputLocked(context.Background(), providerHistoryRecoveryInputSelection{checkpoint: r.initial, inventory: w})
				r.store.mu.Unlock()
			case "orphan-object":
				r.store.mu.Lock()
				_, err = r.readSelectedInputLocked(context.Background(), providerHistoryRecoveryInputSelection{object: true})
				r.store.mu.Unlock()
			}
			if err == nil || r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes || o.meter.actual != old || o.store.clock.read != reads || !r.active || r.closed {
				t.Fatal("detached or out-of-order selection reached a file or spent the lane", mode, err)
			}
		})
	}
	if record, err := w.nextInventoryCandidate(context.Background()); err != nil || record == nil || w.current == nil {
		t.Fatal("otherwise-valid actual inventory after isolated role refusals", err)
	}
	work, files, bytes := r.meter.actual, r.inputFiles, r.inputBytes
	if out, err := w.nextInventoryCandidate(context.Background()); err == nil || out != nil || r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes || !r.active {
		t.Fatal("unread referenced objects were skipped by selecting another inventory", err)
	}
	if json.Unmarshal([]byte(`{}`), &providerHistoryRecoveryInventoryWalk{}) == nil {
		t.Fatal("serialized DATA manufactured a selected original walk")
	}
}

func TestProviderHistoryRecoveryInventoryRejectsMissingAndChangedActualInputsWithoutAdvancing(t *testing.T) {
	for _, kind := range []string{"inventory", "object"} {
		for _, fault := range []string{"missing", "empty", "different", "directory"} {
			t.Run(kind+"/"+fault, func(t *testing.T) {
				o, r, _ := historyRecoveryInventoryFixture(t)
				w, err := r.newInventoryWalkCandidate(context.Background(), 0)
				if err != nil {
					t.Fatal(err)
				}
				name := "inventory-" + w.next[2:]
				if kind == "object" {
					if _, err := w.nextInventoryCandidate(context.Background()); err != nil || w.current == nil {
						t.Fatal("genuine anchored inventory before owned object fixture", err)
					}
					ref := w.current.entries[w.index]
					name = "object-" + ref.kind + "-" + ref.digest[2:]
				}
				path := filepath.Join(r.config.evidenceDirectory, name)
				raw, err := os.ReadFile(path)
				if err != nil || len(raw) == 0 {
					t.Fatal("actual selected inert unit input precursor", err)
				}
				// Owned unit files only: no network payload, symlink/race,
				// external host, shared root or attack reproduction helper.
				switch fault {
				case "missing":
					err = os.Remove(path)
				case "empty":
					err = os.WriteFile(path, nil, 0600)
				case "different":
					for i := range raw {
						raw[i] = '!'
					}
					err = os.WriteFile(path, raw, 0600)
				case "directory":
					if err = os.Remove(path); err == nil {
						err = os.Mkdir(path, 0700)
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				remaining, index, seen, next := w.remaining, w.index, len(w.seen), w.next
				old, reads, held, pin, ack, files := o.meter.actual, o.store.clock.read, o.store.stored, o.pin, o.last, r.inputFiles
				if kind == "inventory" {
					if out, err := w.nextInventoryCandidate(context.Background()); err == nil || out != nil {
						t.Fatal("missing/changed inventory returned reference closure DATA", err)
					}
				} else if out, err := w.nextObjectCandidate(context.Background()); err == nil || out != nil {
					t.Fatal("missing/changed object returned reference closure DATA", err)
				}
				if !r.failed || !r.closed || r.active || r.inputFiles != files+1 || w.remaining != remaining || w.index != index || len(w.seen) != seen || w.next != next || w.complete ||
					o.meter.actual != old || o.store.clock.read != reads || o.store.stored != held || o.pin != pin || o.last != ack || o.store.activeMeter.Load() != nil {
					t.Fatal("failed actual input advanced the predecessor/object selection or revived execution")
				}
				work, bytes := r.meter.actual, r.inputBytes
				if r.beginInspectionCandidate(context.Background(), o) == nil || r.inputFiles != files+1 || r.inputBytes != bytes || r.meter.actual != work {
					t.Fatal("failed reference walk renewed its original row")
				}
			})
		}
	}
}

func TestProviderHistoryRecoveryInventoryCancellationClosesLaneBeforeNextEffect(t *testing.T) {
	for _, phase := range []string{"new", "inventory", "object", "finish"} {
		t.Run(phase, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixture(t)
			var w *providerHistoryRecoveryInventoryWalk
			if phase != "new" {
				var err error
				w, err = r.newInventoryWalkCandidate(context.Background(), 0)
				if err != nil {
					t.Fatal(err)
				}
			}
			if phase == "object" {
				if _, err := w.nextInventoryCandidate(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if phase == "finish" {
				historyRecoveryInventoryDrain(t, w, want)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			work, files, bytes, old, reads := r.meter.actual, r.inputFiles, r.inputBytes, o.meter.actual, o.store.clock.read
			var err error
			switch phase {
			case "new":
				_, err = r.newInventoryWalkCandidate(ctx, 0)
			case "inventory":
				_, err = w.nextInventoryCandidate(ctx)
			case "object":
				_, err = w.nextObjectCandidate(ctx)
			case "finish":
				_, err = w.finishCandidate(ctx)
			}
			if err == nil || !r.failed || !r.closed || r.active || r.meter.actual != work || r.inputFiles != files || r.inputBytes != bytes || o.meter.actual != old || o.store.clock.read != reads {
				t.Fatal("cancelled inventory action observed input, renewed execution or reset its row", err)
			}
		})
	}
}
