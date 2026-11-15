package registry011

import (
	"strings"
	"testing"
)

func historyStorageObjects(t *testing.T) []providerHistoryStoredObject {
	t.Helper()
	pack, _ := historyOriginalPackFixture(t, "requests", 0, 1)
	ticket := historyRecordTicket(t, 1, 1, admissionTestDigest("a"))
	objects := make([]providerHistoryStoredObject, 2)
	for i, value := range []struct{ kind, raw string }{{"requests", pack.raw}, {"ticket", ticket.raw}} {
		var err error
		objects[i], err = newProviderHistoryStoredObjectObserved(value.kind, value.raw, providerHistoryOriginalPackBytes, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	return objects
}

func TestProviderHistoryStorageUsesDistinctNamesAndExactOriginalInventory(t *testing.T) {
	objects := historyStorageObjects(t)
	previous := admissionTestDigest("b")
	m := historyGroupCodecMeter()
	inventory, err := newProviderHistoryInventoryObserved(previous, objects, candidateLedgerMetadataLimit, m)
	if err != nil || inventory.count != 2 || m.failed || m.actual[2] != 0 || m.actual[10] == 0 {
		t.Fatal("bounded history inventory", err)
	}
	parsed, err := parseProviderHistoryInventoryObserved([]byte(inventory.raw), previous, inventory.digest, objects, candidateLedgerMetadataLimit, m)
	if err != nil || parsed == inventory || *parsed != *inventory {
		t.Fatal("exact inventory did not independently reconstruct", err)
	}
	for _, ref := range inventory.entries[:inventory.count] {
		name := providerHistoryStoredName(ref, m)
		if !strings.HasPrefix(name, "history-object-") || name == providerObjectName(ref.kind, ref.digest) {
			t.Fatal("history original reused the old object namespace")
		}
	}
	if providerHistoryInventoryName(inventory.digest, m) != "history-inventory-"+inventory.digest[2:] || m.actual[0] != 0 || m.actual[9] != 0 || m.clocks != 0 {
		t.Fatal("history path work acquired an external role")
	}
	// The descriptor's strings are immutable despite replacing caller entries.
	objects[0].raw = "replaced caller value"
	if _, err := parseProviderHistoryInventoryObserved([]byte(inventory.raw), previous, inventory.digest, objects, candidateLedgerMetadataLimit, nil); err == nil {
		t.Fatal("changed original kept the inventory identity")
	}
}

func TestProviderHistoryStorageRejectsUnknownOversizedAndReorderedInputs(t *testing.T) {
	for _, mode := range []string{"kind", "empty", "maximum-zero", "maximum-large", "raw-large", "record-large", "previous", "count-zero", "count-large", "order", "digest", "metadata-zero", "metadata-large", "observer"} {
		t.Run(mode, func(t *testing.T) {
			objects := historyStorageObjects(t)
			previous, maximum, m := admissionTestDigest("b"), uint64(candidateLedgerMetadataLimit), historyGroupCodecMeter()
			if mode == "kind" || mode == "empty" || mode == "maximum-zero" || mode == "maximum-large" || mode == "raw-large" || mode == "record-large" {
				kind, raw, cap := "requests", objects[0].raw, uint64(providerHistoryOriginalPackBytes)
				switch mode {
				case "kind":
					kind = "legacy-unknown"
				case "empty":
					raw = ""
				case "maximum-zero":
					cap = 0
				case "maximum-large":
					cap++
				case "raw-large":
					cap = 1
				case "record-large":
					kind, raw = "ticket", strings.Repeat("x", candidateLedgerMetadataLimit+1)
				}
				if object, err := newProviderHistoryStoredObjectObserved(kind, raw, cap, m); err == nil || object != (providerHistoryStoredObject{}) || m.actual != (candidateWorkVector{}) {
					t.Fatal("invalid object reached payload work")
				}
				return
			}
			switch mode {
			case "previous":
				previous = "invalid"
			case "count-zero":
				objects = nil
			case "count-large":
				objects = make([]providerHistoryStoredObject, 9)
			case "order":
				objects[0], objects[1] = objects[1], objects[0]
			case "digest":
				objects[0].digest = admissionTestDigest("f")
			case "metadata-zero":
				maximum = 0
			case "metadata-large":
				maximum = candidateLedgerMetadataLimit + 1
			case "observer":
				m.upper[10] = 0
			}
			if inventory, err := newProviderHistoryInventoryObserved(previous, objects, maximum, m); err == nil || inventory != nil {
				t.Fatal("invalid history inventory")
			}
		})
	}
	for _, ref := range []providerHistoryStoredReference{{}, {"unknown", admissionTestDigest("a"), 1}, {"ticket", "bad", 1}, {"ticket", admissionTestDigest("a"), 0}, {"ticket", admissionTestDigest("a"), providerHistoryOriginalPackBytes + 1}} {
		if providerHistoryStoredName(ref, nil) != "" {
			t.Fatal("invalid original reference obtained a path")
		}
	}
	if providerHistoryInventoryName("bad", nil) != "" {
		t.Fatal("invalid inventory obtained a path")
	}
}

func TestProviderHistoryStorageChargesEveryNewFileWithoutRefund(t *testing.T) {
	objects := historyStorageObjects(t)
	inventory, err := newProviderHistoryInventoryObserved(admissionTestDigest("b"), objects, candidateLedgerMetadataLimit, nil)
	if err != nil {
		t.Fatal(err)
	}
	// This finite cost input is DATA only; it is never parsed or retained as a
	// valid custody frame, an ACK or a whole-program cost certificate.
	frame := &providerHistoryCustodyFrame{intent: "intent", commit: "commit", frontier: "frontier"}
	c, ok := providerHistoryStorageChargeFor(inventory, objects, frame, 64, 4096)
	wantRaw := uint64(len(objects[0].raw) + len(objects[1].raw) + len(inventory.raw) + 2*len(frame.intent) + len(frame.commit) + len(frame.frontier) + 128)
	if !ok || c.files != 9 || c.objects != 2 || c.raw != wantRaw || c.physical != wantRaw+9*4096 || c.payload != uint64(len(objects[0].raw)) || c.metadata+c.payload != c.raw {
		t.Fatal("actual delta allocation omitted an original or HEAD copy", c)
	}
	maximum := providerHistoryGroupTotals{physical: 2 * c.physical, retainedRaw: 2 * c.raw, payload: 2 * c.payload, metadata: 2 * c.metadata, objects: 2 * c.objects, files: 2 * c.files}
	first, ok := (providerHistoryStorageCharge{}).add(c, maximum)
	second, twice := first.add(c, maximum)
	if !ok || !twice || second.physical != maximum.physical || second.files != maximum.files {
		t.Fatal("cumulative history allocation lost the earlier HEAD charge")
	}
	if got, ok := second.add(c, maximum); ok || got != (providerHistoryStorageCharge{}) {
		t.Fatal("third delta refunded an earlier file")
	}
	for _, mode := range []string{"inventory", "frame", "count", "highwater", "overhead", "overflow"} {
		t.Run(mode, func(t *testing.T) {
			v, f, highwater, overhead := inventory, frame, uint64(64), uint64(4096)
			items := objects
			switch mode {
			case "inventory":
				v = nil
			case "frame":
				f = nil
			case "count":
				items = nil
			case "highwater":
				highwater = 0
			case "overhead":
				overhead = 0
			case "overflow":
				overhead = ^uint64(0)
			}
			if got, ok := providerHistoryStorageChargeFor(v, items, f, highwater, overhead); ok || got != (providerHistoryStorageCharge{}) {
				t.Fatal("invalid or unrepresentable storage delta")
			}
		})
	}
}
