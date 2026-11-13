package registry011

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProviderHistoryRecoveryLegacyReferencesCannotReplaceCompleteOriginalFrames(t *testing.T) {
	for _, mode := range []string{"intact", "incomplete-walk", "excess-frame-objects", "different-prior-enrollment", "missing-ticket-reference"} {
		t.Run(mode, func(t *testing.T) {
			o, r, refs := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, refs)
			s := o.store
			s.mu.Lock()
			defer s.mu.Unlock()
			original := r.legacyPrefixes[0]
			walk := original.source
			if !original.selectedLocked(r) || !walk.framesReadyLocked(r) || len(walk.frames[1].objects) == 0 {
				t.Fatal("complete actual initial frames required")
			}
			files, bytes, work, bounds := r.inputFiles, r.inputBytes, r.meter.actual, r.bounds
			old, reads, pin, ack, held := o.meter.actual, s.clock.read, o.pin, o.last, s.stored
			before := providerHistoryProvisionedOriginalBytes(t, s.config)
			switch mode {
			case "incomplete-walk":
				walk.complete = false
				accepted := walk.framesReadyLocked(r)
				walk.complete = true
				if accepted {
					t.Fatal("incomplete walk supplied a closed frame list")
				}
			case "excess-frame-objects":
				frame := walk.frames[1]
				objects, inventory := frame.objects, frame.inventory
				copyInventory := *inventory
				copyInventory.entries = append(inventory.entries[:len(inventory.entries):len(inventory.entries)], inventory.entries[0])
				frame.objects = make([]providerIntegrationObject, len(objects)+1)
				copy(frame.objects, objects)
				frame.objects[len(objects)] = objects[0]
				frame.inventory = &copyInventory
				accepted := walk.framesReadyLocked(r)
				frame.objects, frame.inventory = objects, inventory
				if accepted {
					t.Fatal("extra private frame object exceeded the original sealed object count")
				}
			default:
				// A fresh unselected bookkeeping reader uses the actual original
				// controls. It never replaces this pass's selected legacy prefix.
				p := &providerHistoryRecoveryLegacyPrefix{recovery: r, source: walk}
				prior, current := original.records[0], original.records[1]
				enrollment := prior.root.digest
				if mode == "different-prior-enrollment" {
					enrollment = admissionTestDigest("f")
					if enrollment == prior.root.digest {
						t.Fatal("different prior enrollment DATA required")
					}
				}
				var changed *providerIntegrationObject
				kind := ""
				if mode == "missing-ticket-reference" {
					for i := range walk.frames[1].objects {
						object := &walk.frames[1].objects[i]
						if object.kind == "ticket" {
							changed, kind = object, object.kind
							object.kind = "unselected private object DATA"
							break
						}
					}
					if changed == nil || !walk.framesReadyLocked(r) {
						t.Fatal("ordinary ticket DATA and unchanged frame cardinality required")
					}
				}
				out, err := p.mainLocked(context.Background(), current.before, current.after, prior.frontier.objects, enrollment)
				if changed != nil {
					changed.kind = kind
				}
				if mode == "intact" {
					if err != nil || out.root == nil || out.before != current.before || out.after != current.after || out.kind != "admit" || out.operation != current.operation {
						t.Fatal("ordinary fresh initial main-record read required", err)
					}
				} else if err == nil || out != (providerHistoryRecoveryLegacyRecord{}) {
					t.Fatal("incomplete private reference supplied a legacy main record", mode, err)
				}
				if p.selectedLocked(r) || p.owner != nil || r.inputFiles != files+4 || r.inputBytes <= bytes || r.meter.actual[10] <= work[10] {
					t.Fatal("private original read lost actual input accounting or acquired a selected prefix", mode)
				}
			}
			if !original.selectedLocked(r) || !walk.framesReadyLocked(r) || r.legacyPrefixes[0] != original || r.bounds != bounds ||
				r.meter.failed || r.meter.closed || !r.active || r.failed || r.closed || o.meter.actual != old || s.clock.read != reads ||
				o.pin != pin || o.last != ack || s.stored != held || s.activeMeter.Load() != nil ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("private legacy reference probe changed originals, execution or the selected pass", mode)
			}
		})
	}
}

func TestProviderHistoryRecoveryLegacyRejectsUnplannedSetupAndDifferentFinalRoots(t *testing.T) {
	for _, mode := range []string{"different-final-root", "unplanned-setup"} {
		t.Run(mode, func(t *testing.T) {
			o, r, refs := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryLegacySource(t, r, 0, refs)
			s := o.store
			if r.initialDescriptor.setups != 1 || r.legacyPrefixes[0] != nil || !r.inputReadyLocked(context.Background()) {
				t.Fatal("single original setup and unused original legacy reader required")
			}
			if mode == "different-final-root" {
				if r.initialDescriptor.objects == admissionTestDigest("f") {
					t.Fatal("different final root comparison DATA required")
				}
				r.initialDescriptor.objects = admissionTestDigest("f")
			} else {
				path := filepath.Join(r.config.custodyDirectory, providerCustodyName("highwater", 3))
				raw, err := os.ReadFile(path)
				h, parseErr := parseProviderFrontierHighwater(raw, r.config)
				if err != nil || parseErr != nil || h.ordinal != 3 || h.sequence != 3 {
					t.Fatal("actual third main highwater required", err, parseErr)
				}
				// A canonical, locally owned DATA record describes an extra setup
				// instead of the next main record. The sealed setup count remains
				// one, and no new authenticated setup or capacity is supplied.
				h.sequence = 2
				raw, err = h.bytes(r.config)
				if err != nil {
					t.Fatal("finite canonical extra-setup description required", err)
				}
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal("owned highwater DATA fault failed", err)
				}
			}
			if !r.inputReadyLocked(context.Background()) || !r.inventoryWalks[0].framesReadyLocked(r) || !r.scanWalks[0].selectedLocked(r) {
				t.Fatal("original reader entry selection must precede semantic refusal", mode)
			}
			before := providerHistoryProvisionedOriginalBytes(t, s.config)
			old, reads, held, pin, ack, files, work := o.meter.actual, s.clock.read, s.stored, o.pin, o.last, r.inputFiles, r.meter.actual
			if out, err := r.replayLegacyPrefixCandidate(context.Background(), 0); err == nil || out != nil || r.legacyPrefixes[0] != nil ||
				r.active || !r.failed || !r.closed || !r.meter.failed || !r.meter.closed || r.inputFiles <= files || r.meter.actual[10] <= work[10] ||
				o.meter.actual != old || s.clock.read != reads || s.stored != held || o.pin != pin || o.last != ack || s.activeMeter.Load() != nil ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, s.config)) {
				t.Fatal("unplanned setup or wrong final root published a legacy prefix or revived execution", mode, err)
			}
		})
	}
}
