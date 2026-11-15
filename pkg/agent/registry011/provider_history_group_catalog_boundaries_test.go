package registry011

import (
	"os"
	"path/filepath"
	"testing"
)

// A genuine prepared controller supplies the already-held observer. The
// prospective filenames below are catalog DATA only: no pending transaction,
// retained original, native clock, ACK or write claim is installed by this
// fixture. Each test closes its one original observer without renewing it.
func historyAdmissionCatalogFixture(t *testing.T) (*providerHistoryGroupOwner, *providerHistoryWriteClaim) {
	t.Helper()
	o, upper := historyAdmissionWriterFixture(t)
	s, m := o.store, o.meter
	s.mu.Lock()
	t.Cleanup(func() { o.failLocked(); s.mu.Unlock() })
	work, ok := upper.work()
	if !ok {
		t.Fatal("finite child window")
	}
	o.window, o.windowStart, o.windowUpper = true, m.actual, work
	o.clockStart, o.clockUpper, o.phase = m.clocks, upper.clocks, "preparing-admit"
	s.activeMeter.Store(m)
	after := o.pin
	after.sequence++
	after.ordinal++
	c := &providerHistoryWriteClaim{group: o, count: 2,
		frame:     &providerHistoryCustodyFrame{before: o.pin, after: after},
		inventory: &providerHistoryInventory{count: 2, digest: admissionTestDigest("d")}}
	c.owner = c
	c.inventory.entries[0] = providerHistoryStoredReference{kind: "requests", digest: admissionTestDigest("a"), bytes: 1}
	c.inventory.entries[1] = providerHistoryStoredReference{kind: "ticket", digest: admissionTestDigest("b"), bytes: 1}
	if !o.activeLocked(m) || o.claimActiveLocked(c) || s.frontierFiles.historyPending != nil {
		t.Fatal("catalog data unexpectedly acquired a retained write role")
	}
	return o, c
}

func TestProviderHistoryAdmissionCatalogRejectsUnselectedPrefixAndIndexData(t *testing.T) {
	for _, mode := range []string{"claim", "setup-ordinal", "selected-inventory", "missing-inventory", "changed-inventory", "changed-object-index", "extra-object-index"} {
		t.Run(mode, func(t *testing.T) {
			o, c := historyAdmissionCatalogFixture(t)
			s, pin, held, raw, reads := o.store, o.pin, o.store.stored, o.store.retainedRaw, o.store.clock.read
			if !o.prepareAdmissionCatalogLocked(c) || !o.exactAdmissionCatalogLocked(c, false) {
				t.Fatal("separate original-prefix positive control")
			}
			path := filepath.Join(s.config.evidenceDirectory, s.inventoryName(o.start.objects))
			switch mode {
			case "claim":
				c.owner = nil
			case "setup-ordinal":
				s.setups[0].ordinal++
			case "selected-inventory":
				o.start.objects = "inert non-digest"
			case "missing-inventory":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "changed-inventory":
				if err := os.WriteFile(path, []byte("inert changed inventory"), 0600); err != nil {
					t.Fatal(err)
				}
			case "changed-object-index":
				for key, ref := range s.objects {
					ref.bytes++
					s.objects[key] = ref
					break
				}
			case "extra-object-index":
				for _, ref := range s.objects {
					s.objects["inert unselected index entry"] = ref
					break
				}
			}
			if o.prepareAdmissionCatalogLocked(c) || o.last != nil || s.frontierFiles.historyPending != nil ||
				o.retention != (providerHistoryStorageCharge{}) || s.stored != held || s.retainedRaw != raw || o.pin != pin || s.frontierFiles.pin != pin || s.clock.read != reads {
				t.Fatal("changed prefix or index supplied a catalog, allocation or authority")
			}
		})
	}
}

func TestProviderHistoryAdmissionCatalogRejectsCorruptEarlyNameCaps(t *testing.T) {
	for _, limit := range []uint64{1, 2, 3} {
		t.Run([]string{"lock", "head", "highwater"}[limit-1], func(t *testing.T) {
			o, c := historyAdmissionCatalogFixture(t)
			before, reads, held := o.meter.actual, o.store.clock.read, o.store.stored
			// Corrupt scalar DATA only. It no longer matches the selected
			// immutable parent profile and cannot authorize any native IO.
			o.parent.bounds.catalogEntries = limit
			if o.prepareAdmissionCatalogLocked(c) || o.meter.actual != before || o.store.clock.read != reads || o.store.stored != held || o.last != nil || o.store.frontierFiles.historyPending != nil {
				t.Fatal("corrupt name cap reached an original read or authority")
			}
		})
	}
}

func TestProviderHistoryAdmissionExactCatalogRequiresItsDerivedNameSet(t *testing.T) {
	o, c := historyAdmissionCatalogFixture(t)
	if !o.prepareAdmissionCatalogLocked(c) || !o.exactAdmissionCatalogLocked(c, false) {
		t.Fatal("independently derived original catalog")
	}
	before, reads := o.meter.actual, o.store.clock.read
	copy := *c
	if o.exactAdmissionCatalogLocked(&copy, false) || o.meter.actual != before {
		t.Fatal("copied catalog data borrowed private ownership")
	}
	want := c.beforeCatalog[0]
	c.beforeCatalog[0] = nil
	if o.exactAdmissionCatalogLocked(c, false) || o.meter.actual != before {
		t.Fatal("absent derived names triggered an original directory read")
	}
	c.beforeCatalog[0] = want
	delete(want, "ANCHOR")
	want["inert unexpected replacement"] = true
	if o.exactAdmissionCatalogLocked(c, false) || o.meter.failed || o.store.clock.read != reads || o.last != nil || o.store.frontierFiles.historyPending != nil {
		t.Fatal("equal catalog sizes substituted for exact original names")
	}
}

func TestProviderHistoryAdmissionExactCatalogPreflightsItsWholeIdentityRead(t *testing.T) {
	o, c := historyAdmissionCatalogFixture(t)
	if !o.prepareAdmissionCatalogLocked(c) {
		t.Fatal("original catalog positive control")
	}
	before, reads := o.meter.actual, o.store.clock.read
	remaining := before[10] - o.windowStart[10] + 3
	if remaining >= o.windowUpper[10] {
		t.Fatal("negative case must only narrow its existing window")
	}
	o.windowUpper[10] = remaining
	if o.exactAdmissionCatalogLocked(c, false) || !o.meter.failed || o.meter.actual != before || o.store.clock.read != reads || o.last != nil {
		t.Fatal("four-read primitive partially consumed a three-read allowance")
	}
}

func TestProviderHistoryAdmissionCatalogCopyDepletionKeepsCumulativeWork(t *testing.T) {
	for _, test := range []struct {
		name    string
		divisor uint64
		missing uint64
	}{{"half", 2, 0}, {"quarter", 4, 0}, {"eighth", 8, 0}, {"late-name-copies", 0, 400}, {"new-object-names", 0, 256}, {"last-name-byte", 0, 1}} {
		t.Run(test.name, func(t *testing.T) {
			o, c := historyAdmissionCatalogFixture(t)
			before := o.meter.actual
			if !o.prepareAdmissionCatalogLocked(c) {
				t.Fatal("bounded catalog measurement positive control")
			}
			atBound, reads := o.meter.actual, o.store.clock.read
			copied := atBound[2] - before[2]
			if copied == 0 {
				t.Fatal("original catalog must pay for copies")
			}
			allowance := uint64(0)
			if test.divisor != 0 {
				allowance = copied / test.divisor
			} else {
				if copied <= test.missing {
					t.Fatal("positive catalog must exceed the selected missing copy suffix")
				}
				allowance = copied - test.missing
			}
			remaining := atBound[2] - o.windowStart[2] + allowance
			if remaining >= o.windowUpper[2] {
				t.Fatal("unit fault must only shrink the original child window")
			}
			o.windowUpper[2] = remaining
			if o.prepareAdmissionCatalogLocked(c) || !o.meter.failed || o.store.clock.read != reads || o.last != nil || o.store.frontierFiles.historyPending != nil {
				t.Fatal("copy depletion restored a catalog role or original call")
			}
			for i, n := range atBound {
				if o.meter.actual[i] < n {
					t.Fatal("failed catalog refunded cumulative work", i)
				}
			}
		})
	}
}

// One genuine retained T supplies the closed observer and acknowledged claim.
// Each helper result is catalog DATA only: no C, pending write, transport or
// new clock allowance is installed, and negative cases keep cumulative charges.
func TestProviderHistoryCatalogExtensionRefusesUnretainedOrRepeatedOriginals(t *testing.T) {
	o, admission := historyConsumptionFixture(t, 1)
	s, m, claim := o.store, o.meter, admission.claim
	s.mu.Lock()
	t.Cleanup(func() { o.failLocked(); s.mu.Unlock() })
	o.phase = "preparing-consume"
	s.activeMeter.Store(m)
	if !o.activeLocked(m) || !o.admissionShapeLocked() || !claim.used || !claim.renamed {
		t.Fatal("genuine retained T must remain the only acknowledged prefix")
	}
	initial, ok := o.initialHistoryCatalogLocked()
	if !ok {
		t.Fatal("original initial namespace")
	}
	clone := func() [2]map[string]bool {
		var names [2]map[string]bool
		for root := range names {
			names[root] = make(map[string]bool, len(initial[root])+9)
			for name := range initial[root] {
				names[root][name] = true
			}
		}
		return names
	}
	positive := clone()
	if !o.extendHistoryCatalogLocked(positive, claim) || len(positive[0]) != len(initial[0])+4 || len(positive[1]) != len(initial[1])+claim.count+2 {
		t.Fatal("exact original T namespace positive control")
	}
	memberName := providerHistoryStoredName(claim.inventory.entries[0], m)
	inventoryName := providerHistoryInventoryName(claim.inventory.digest, m)
	if memberName == "" || inventoryName == "" || initial[1][memberName] || initial[1][inventoryName] || !positive[1][memberName] || !positive[1][inventoryName] {
		t.Fatal("negative names must select the positive original extension")
	}
	pin, raw, held, retained, payload, reads := o.pin, s.frontierFiles.pinRaw, s.stored, o.retention, s.retainedRaw, s.clock.read
	coordinator, custodian := o.coordinator, o.custodian
	for _, mode := range []string{"nil-claim", "copied-claim", "unretained", "unrenamed", "foreign-meter", "nil-derived-evidence", "duplicate-member", "duplicate-inventory", "duplicate-control"} {
		t.Run(mode, func(t *testing.T) {
			names := clone()
			candidate := *claim
			candidate.owner = &candidate
			input := &candidate
			switch mode {
			case "nil-claim":
				input = nil
			case "copied-claim":
				candidate.owner = claim
			case "unretained":
				candidate.used = false
			case "unrenamed":
				candidate.renamed = false
			case "foreign-meter":
				candidate.meter = &providerWorkMeter{}
			case "nil-derived-evidence":
				names[1] = nil
			case "duplicate-member":
				names[1][memberName] = true
			case "duplicate-inventory":
				names[1][inventoryName] = true
			case "duplicate-control":
				names[0][s.name("intent", claim.frame.after.sequence)] = true
			}
			before := m.actual
			if o.extendHistoryCatalogLocked(names, input) || m.failed || m.closed || !o.activeLocked(m) ||
				o.pin != pin || s.frontierFiles.pin != pin || s.frontierFiles.pinRaw != raw || o.last != admission || o.admission != admission ||
				o.coordinator != coordinator || o.custodian != custodian || o.order != nil || o.dispatch != nil ||
				s.stored != held || o.retention != retained || s.retainedRaw != payload || s.clock.read != reads ||
				s.frontierFiles.historyPending != nil || m.actual[0] != before[0] || m.actual[1] != before[1] || m.actual[9] != before[9] {
				t.Fatal("invalid or repeated originals supplied an extended namespace or changed authority")
			}
			for i, n := range before {
				if m.actual[i] < n {
					t.Fatal("catalog refusal refunded original cumulative work", i)
				}
			}
		})
	}
}
