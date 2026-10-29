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
