package registry011

import (
	"bytes"
	"strings"
	"testing"
)

func TestProviderReplayPathCopiesObserveEveryInvocationBeforeAllocation(t *testing.T) {
	s := &providerCustodyStore{}
	s.config.bounds.firstAdmission = &providerFirstAdmissionBounds{}
	m := providerReplayUnitMeter()
	s.activeMeter.Store(m)
	digest := admissionTestDigest("a")
	var units uint64
	for n := 0; n < 2; n++ {
		name := providerCustodyName("intent", 7)
		if got := s.name("intent", 7); got != name {
			t.Fatal("exact immutable filename changed")
		}
		units += uint64(len(name) + 1)
		name = providerObjectName("enrollment", digest)
		if got := s.objectName("enrollment", digest); got != name {
			t.Fatal("exact object filename changed")
		}
		units += uint64(len(name) + 1)
		name = "inventory-" + digest[2:]
		if got := s.inventoryName(digest); got != name {
			t.Fatal("exact inventory filename changed")
		}
		units += uint64(len(name) + 1)
		if got := providerReplayBytes("ordinary", m); !bytes.Equal(got, []byte("ordinary")) {
			t.Fatal("exact input copy changed")
		}
		units += 9
	}
	if m.actual[10] != units || m.actual[2] != 0 || s.last != nil {
		t.Fatal("copy reused a previous charge, decoded metadata or granted authority")
	}
	before := m.actual
	if s.name("", 0) != "" || s.name(strings.Repeat("n", 65), 0) != "" || s.objectName("unknown", digest) != "" || s.objectName("enrollment", "invalid") != "" || s.inventoryName("invalid") != "" || m.actual != before {
		t.Fatal("invalid finite filename allocated output")
	}
}

func TestProviderReplayPathCopiesRefuseOneUnitShortWithoutOutput(t *testing.T) {
	for _, kind := range []string{"name", "object", "inventory", "bytes"} {
		t.Run(kind, func(t *testing.T) {
			s := &providerCustodyStore{}
			s.config.bounds.firstAdmission = &providerFirstAdmissionBounds{}
			m := providerReplayUnitMeter()
			s.activeMeter.Store(m)
			m.upper = candidateWorkVector{}
			digest := admissionTestDigest("a")
			var got string
			switch kind {
			case "name":
				m.upper[10] = uint64(len(providerCustodyName("intent", 0)))
				got = s.name("intent", 0)
			case "object":
				m.upper[10] = uint64(len(providerObjectName("enrollment", digest)))
				got = s.objectName("enrollment", digest)
			case "inventory":
				m.upper[10] = uint64(len("inventory-") + len(digest) - 2)
				got = s.inventoryName(digest)
			case "bytes":
				m.upper[10] = 8
				if providerReplayBytes("ordinary", m) != nil {
					t.Fatal("copy allocated before its admission")
				}
			}
			if got != "" || !m.failed || m.actual != (candidateWorkVector{}) {
				t.Fatal("copy returned uncharged filename bytes", kind)
			}
		})
	}
}

func TestProviderReplayLegacyPathsKeepBytesWithoutNewAccounting(t *testing.T) {
	s := &providerCustodyStore{}
	m := providerReplayUnitMeter()
	s.activeMeter.Store(m)
	digest := admissionTestDigest("a")
	if s.copyMeter() != nil || s.name("intent", 0) != providerCustodyName("intent", 0) || s.objectName("enrollment", digest) != providerObjectName("enrollment", digest) || s.inventoryName(digest) != "inventory-"+digest[2:] || m.actual != (candidateWorkVector{}) || !bytes.Equal(providerReplayBytes("ordinary", nil), []byte("ordinary")) {
		t.Fatal("older profile acquired new copy-accounting behavior")
	}
}

func TestProviderReplayPhaseCopiesKeepOldObserversAndChargeTheV4Owner(t *testing.T) {
	legacy := providerReplayUnitMeter()
	if !bytes.Equal(providerReplayPhaseBytes("ordinary", nil), []byte("ordinary")) || !bytes.Equal(providerReplayPhaseBytes("ordinary", legacy), []byte("ordinary")) || legacy.actual != (candidateWorkVector{}) {
		t.Fatal("legacy DATA observer acquired V4 copy charges")
	}
	s := &providerCustodyStore{}
	legacy.integration.store = s
	if !bytes.Equal(providerReplayPhaseBytes("ordinary", legacy), []byte("ordinary")) || legacy.actual != (candidateWorkVector{}) {
		t.Fatal("a legacy store acquired V4 accounting")
	}
	s.config.bounds.firstAdmission = &providerFirstAdmissionBounds{}
	for n := 0; n < 2; n++ {
		if !providerReplayPhaseTextEqual("original", "original", legacy) {
			t.Fatal("ordinary closed inventory text comparison")
		}
	}
	// Each real comparison has two separately admitted8-byte conversions
	// and one comparison of16 bytes: 2*9+17=35, twice without reuse.
	if legacy.actual[10] != 70 {
		t.Fatal("typed cost comparison omitted or reused a byte copy")
	}
	before := legacy.actual
	legacy.upper[10] = before[10] + 9 + 8 // One unit short of the second copy.
	if providerReplayPhaseTextEqual("original", "original", legacy) || !legacy.failed || legacy.actual[10] != before[10]+9 {
		t.Fatal("unfunded second inventory input became an equality proof")
	}
}

func TestProviderBootstrapPinChecksObserveRepeatedCopiesAndRefuseBeforeRead(t *testing.T) {
	for _, short := range []bool{false, true} {
		t.Run(map[bool]string{false: "repeated", true: "one-unit-short"}[short], func(t *testing.T) {
			s := providerBootstrapOwnerFixture(t)
			m := s.bootstrapOwner.meter
			s.activeMeter.Store(m)
			t.Cleanup(func() { s.activeMeter.Store(nil) })
			before := m.actual
			if short {
				m.upper[10] = before[10] + uint64(len(s.frontierFiles.pinRaw))
				if s.scanControl.checkMainPin() || !m.failed || m.actual != before || s.last != nil {
					t.Fatal("pin check read an original after its input copy was refused")
				}
				return
			}
			for n := 0; n < 2; n++ {
				if !s.scanControl.checkMainPin() {
					t.Fatal("ordinary retained main pin")
				}
			}
			perCheck := uint64(8*len(s.frontierFiles.pinRaw) + len(providerCustodyName("highwater", 0)) + 7)
			if m.actual[10] != before[10]+2*perCheck || m.actual[2] != before[2] || m.actual[9] != before[9] || s.last != nil {
				t.Fatal("pin checks omitted their repeated copies/readbacks or changed metadata")
			}
		})
	}
}

func TestProviderReplayRetentionCostSeparatesV4CopiesFromLegacyCodecEvents(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	record := u.records[0]
	legacy, current := providerReplayUnitMeter(), providerReplayUnitMeter()
	s := &providerCustodyStore{}
	s.config.bounds.firstAdmission = &providerFirstAdmissionBounds{}
	current.integration.store = s // Finite DATA cost observer, never a live store.
	for n := 0; n < 2; n++ {
		a, e := providerFrontierRetentionCostObserved(u.s.config.identity, record.inventory, record.fresh, record.intent, record.commit, record.frontier, u.s.config.bounds, legacy)
		b, e2 := providerFrontierRetentionCostObserved(u.s.config.identity, record.inventory, record.fresh, record.intent, record.commit, record.frontier, u.s.config.bounds, current)
		if e != nil || e2 != nil || a != b {
			t.Fatal("copy-accounting revision changed typed retention DATA", e, e2)
		}
	}
	extra := uint64(2 * (len(record.inventory.raw) + 1))
	for _, original := range record.fresh {
		extra += uint64(len(original.raw) + 1)
	}
	want := legacy.actual
	want[10] += 2 * extra
	if current.actual != want || current.failed || legacy.failed {
		t.Fatal("repeated typed cost replay omitted its distinct original-input copies")
	}
}
