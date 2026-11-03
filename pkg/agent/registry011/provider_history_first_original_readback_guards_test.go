package registry011

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Every fault is confined to a fresh UNIT fixture after its ordinary owned
// transport has completed. The original profile, references, clock and hold
// remain selected; no path, peer or execution authority comes from a caller.
func TestProviderHistoryFirstOriginalRequiresUnchangedRetainedInventoryAndObjects(t *testing.T) {
	for _, mode := range []string{"intact", "inventory-missing", "inventory-malformed", "object-missing", "object-same-width"} {
		t.Run(mode, func(t *testing.T) {
			manifest := historyCollectionManifest(t, historyCollectionInput(t))
			s := historyFirstFinalizedStore(t, historyFirstFinalizedResponse(t), manifest)
			config, held, pin, clocks := s.config, s.stored, s.frontierFiles.pin, s.clock.read
			if s.history.start == nil || s.history.bounds.collection != manifest || s.history.group != nil {
				t.Fatal("genuine completed first operation and original collection selection required")
			}
			path := filepath.Join(config.evidenceDirectory, s.inventoryName(s.history.start.objects))
			raw, err := os.ReadFile(path)
			if err != nil || len(raw) == 0 || raw[0] != '{' {
				t.Fatal("actual retained original inventory required", err)
			}
			if mode == "object-missing" || mode == "object-same-width" {
				var inventory struct {
					Entries []struct {
						Kind   string `json:"kind"`
						Digest string `json:"digest"`
					} `json:"entries"`
				}
				if err := json.Unmarshal(raw, &inventory); err != nil || len(inventory.Entries) == 0 {
					t.Fatal("actual inventory must select the owned object", err)
				}
				ref := inventory.Entries[0]
				name := s.objectName(ref.Kind, ref.Digest)
				if name == "" {
					t.Fatal("original inventory object reference required")
				}
				path = filepath.Join(config.evidenceDirectory, name)
				raw, err = os.ReadFile(path)
				if err != nil || len(raw) == 0 {
					t.Fatal("actual retained object required", err)
				}
			}
			switch mode {
			case "inventory-missing", "object-missing":
				err = os.Remove(path)
			case "inventory-malformed":
				raw[0] = '!'
				err = os.WriteFile(path, raw, 0600)
			case "object-same-width":
				raw[len(raw)/2] ^= 1
				err = os.WriteFile(path, raw, 0600)
			}
			if err != nil {
				t.Fatal("owned file fault installation failed", mode, err)
			}
			before := providerHistoryProvisionedOriginalBytes(t, config)
			owner, err := newProviderHistoryGroupOwnerCandidate(s)
			if mode == "intact" {
				if err != nil || owner == nil || owner.failed || owner.firstPReading || owner.coordinator == owner.custodian ||
					owner.coordinator == nil || owner.custodian == nil || owner.coordinator.head != owner.custodian.head ||
					s.history.recovery.initial == nil || s.clock.read != clocks+2 || s.activeMeter.Load() != nil ||
					s.config != config || s.stored != held || s.frontierFiles.pin != pin ||
					!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, config)) {
					t.Fatal("intact originals must reconstruct independent models without rewriting their selection", err)
				}
				return
			}
			failed := s.history.group
			if err == nil || owner != nil || !s.failed || !s.history.failed || failed == nil || !failed.failed ||
				failed.firstPReading || failed.firstPStart[10] == 0 || failed.last != nil || !failed.meter.closed || !failed.meter.failed ||
				s.history.recovery.initial != nil || s.history.recovery.confirmed != nil || s.history.recovery.pending != nil ||
				s.clock.read != clocks+1 || s.activeMeter.Load() != nil || s.config != config || s.stored != held || s.frontierFiles.pin != pin ||
				!reflect.DeepEqual(before, providerHistoryProvisionedOriginalBytes(t, config)) {
				t.Fatal("missing or changed first originals published a model, renewed the hold or rewrote retained bytes", mode, err)
			}
			work, reads := failed.meter.actual, s.clock.read
			if again, err := newProviderHistoryGroupOwnerCandidate(s); err == nil || again != nil ||
				failed.meter.actual != work || s.clock.read != reads || s.history.group != failed || s.history.recovery.initial != nil {
				t.Fatal("failed original reconstruction retried or installed fresh execution authority")
			}
		})
	}
}
