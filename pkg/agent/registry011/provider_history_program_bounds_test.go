package registry011

import "testing"

func TestProviderHistoryProgramBoundsRejectUnfundedResourceDomains(t *testing.T) {
	for _, mode := range []string{"first", "scan", "continuation", "guardian", "program", "costs", "future", "stages", "stage-limit", "records", "commits", "objects", "catalog", "names", "resident", "duration", "startup", "clocks", "requests", "wire", "work", "physical", "clock-total", "pre-i-open-clock", "pre-i-open-work", "derivation", "derivation-owner", "derivation-snapshot"} {
		t.Run(mode, func(t *testing.T) {
			s := providerHistoryParentScalarFixture(t)
			store, h := s.config.bounds, *s.config.bounds.history
			switch mode {
			case "first":
				store.firstAdmission = nil
			case "scan":
				store.scan = nil
			case "continuation":
				store.continuation = nil
			case "guardian":
				store.guardian = &providerGuardianBounds{}
			case "program":
				h.program = "invalid"
			case "costs":
				h.costs = "invalid"
			case "future":
				h.future.physical = 0
			case "stages":
				h.stages = 0
			case "stage-limit":
				h.stages = 1 << 20
			case "records":
				h.records = 2
			case "commits":
				h.commits = 2
			case "objects":
				h.future.objects = uint64(store.maximumObjects) + 1
			case "catalog":
				h.catalogEntries = h.future.files - 1
			case "names":
				h.catalogNameBytes--
			case "resident":
				h.resident = store.continuation.maximumResidentBytes + 1
			case "duration":
				h.durationMS = 1<<53 + 1
			case "startup":
				h.startup.retained = 0
			case "clocks":
				h.startup.clocks = 10
			case "requests":
				h.startup.work[0] = 2
			case "wire":
				h.startup.work[1] = 0
			case "work":
				h.startup.work[10] = 0
			case "physical":
				h.startup.retained = store.maximumStoredBytes
			case "clock-total":
				h.startup.clocks = 1 << 20
			case "pre-i-open-clock":
				store.scan.maximumPreIClockReads = store.setupMaximumClockReads - 1
			case "pre-i-open-work":
				store.scan.maximumPreIWork[10] = store.setupWork[10] - 1
			case "derivation":
				h.derivation = nil
			case "derivation-owner":
				h.derivation.owner = &providerHistoryProgramDerivation{}
			case "derivation-snapshot":
				h.derivation.snapshot.program = admissionTestDigest("f")
			}
			if mode != "derivation" && mode != "derivation-owner" && mode != "derivation-snapshot" {
				// This remains test-only scalar DATA, never a production witness.
				h.derivation.snapshot = h
			}
			if h.valid(store) {
				t.Fatal("unfunded or mismatched domain accepted", mode)
			}
		})
	}
}

func TestProviderHistoryProgramSumRefusesOverflowAndKeepsOldSeedSeparate(t *testing.T) {
	s := providerHistoryParentScalarFixture(t)
	b := s.config.bounds
	held, ok := providerHistoryWholeHold(b)
	pre := providerHistoryPreIUpper(b)
	if !ok || held.work[0] != 2 || held.retained != pre.retained+b.history.startup.retained+b.history.future.physical || pre.work[0] != 0 || pre.work[1] != 0 || !b.firstAdmission.valid(b) {
		t.Fatal("future hold was mixed into the old initial seed")
	}
	for _, mode := range []string{"no-history", "no-scan", "work", "physical", "clocks"} {
		local, scan := b, *b.scan
		local.scan = &scan
		switch mode {
		case "no-history":
			local.history = nil
		case "no-scan":
			local.scan = nil
		case "work":
			scan.maximumPreIWork[10] = ^uint64(0)
		case "physical":
			scan.maximumPreIRetainedBytes = ^uint64(0)
		case "clocks":
			scan.maximumPreIClockReads = ^uint64(0)
		}
		if _, ok := providerHistoryWholeHold(local); ok {
			t.Fatal("incomplete or overflowing sum accepted", mode)
		}
	}
	for _, mode := range []string{"physical", "clocks", "work"} {
		v := pre
		switch mode {
		case "physical":
			v.retained++
		case "clocks":
			v.clocks++
		case "work":
			v.work[0]++
		}
		if providerHistoryUpperWithin(v, pre) {
			t.Fatal("inspection sublimit spent another domain", mode)
		}
	}
}
