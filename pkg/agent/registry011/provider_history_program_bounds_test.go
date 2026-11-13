package registry011

import "testing"

func TestProviderHistoryProgramBoundsRejectUnfundedResourceDomains(t *testing.T) {
	for _, mode := range []string{"first", "scan", "continuation", "guardian", "program", "costs", "future", "stages", "stage-limit", "records", "commits", "objects", "catalog", "names", "resident", "archive-resident", "duration", "startup", "clocks", "requests", "wire", "work", "physical", "clock-total", "pre-i-open-clock", "pre-i-open-work", "derivation", "derivation-owner", "derivation-snapshot"} {
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
			case "archive-resident":
				h.resident = providerHistoryRecoveryArchiveSlotBytes - 1
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

func providerHistoryStageCompositionUnitRows() (providerHistoryStageWorkCost, providerHistoryStageWorkCost, providerHistoryDispatchBounds) {
	before := providerHistoryStageWorkCost{work: candidateWorkVector{2: 31, 3: 2, 4: 3, 5: 5, 6: 7, 7: 11, 8: 13, 9: 17, 10: 41}, clocks: 6}
	preparation := providerHistoryStageWorkCost{work: candidateWorkVector{2: 7, 3: 1, 4: 1, 5: 2, 6: 2, 7: 3, 8: 3, 9: 5, 10: 11}, clocks: 1}
	rows := providerHistoryDispatchBounds{
		member:       candidateWorkVector{0: 1, 1: 101, 2: 19, 3: 3, 4: 5, 5: 7, 6: 11, 7: 13, 8: 17, 9: 23, 10: 29},
		closure:      candidateWorkVector{2: 37, 3: 7, 4: 11, 5: 13, 6: 17, 7: 19, 8: 23, 9: 41, 10: 43},
		advance:      candidateWorkVector{2: 47, 3: 11, 4: 13, 5: 17, 6: 19, 7: 23, 8: 29, 10: 53},
		memberClocks: 3, closureClocks: 2, tailMS: 1, advanceClocks: 2, advanceTailMS: 1,
	}
	return before, preparation, rows
}

func TestProviderHistoryStageCompositionKeepsEveryRemainingGuardInOriginalWindow(t *testing.T) {
	before, preparation, rows := providerHistoryStageCompositionUnitRows()
	for count := 1; count <= providerHistoryGroupMembers; count++ {
		for _, advancing := range []bool{false, true} {
			held, ok := providerHistoryCollectionStageWorkUpper(before, preparation, rows, count, advancing)
			if !ok || held.work[0] != uint64(count) || held.work[1] != uint64(count)*101 {
				t.Fatal("finite original-window rows or exact request/wire changed", count, advancing, held, ok)
			}
			for ordinal := 0; ordinal < count; ordinal++ {
				remaining, clocks, valid := rows.remaining(count - ordinal)
				if !valid {
					t.Fatal("finite suffix unavailable")
				}
				// Pure UNIT prefix DATA, not patched live actuals or an owner.
				for coordinate, suffix := range remaining {
					entered := before.work[coordinate] + uint64(ordinal)*rows.member[coordinate] + preparation.work[coordinate]
					if entered+suffix > held.work[coordinate] || !advancing && entered+suffix != held.work[coordinate] {
						t.Fatal("already-entered preparation cannot fund full remaining suffix", count, ordinal, coordinate)
					}
				}
				enteredClocks := before.clocks + uint64(ordinal)*rows.memberClocks + preparation.clocks
				if enteredClocks+clocks > held.clocks || !advancing && enteredClocks+clocks != held.clocks {
					t.Fatal("remaining clocks left original T window", count, ordinal)
				}
			}
		}
	}
	for _, item := range []struct {
		advancing bool
		want      providerHistoryStageWorkCost
	}{
		{false, providerHistoryStageWorkCost{work: candidateWorkVector{3, 303, 132, 19, 30, 41, 59, 72, 90, 132, 182}, clocks: 18}},
		{true, providerHistoryStageWorkCost{work: candidateWorkVector{3, 303, 179, 30, 43, 58, 78, 95, 119, 132, 235}, clocks: 20}},
	} {
		got, ok := providerHistoryCollectionStageWorkUpper(before, preparation, rows, 3, item.advancing)
		if !ok || got != item.want {
			t.Fatal("independent all-coordinate UNIT oracle changed", item.advancing, got, item.want)
		}
	}
}

func TestProviderHistoryStageCompositionCannotBorrowAdvanceForEnteredPreparation(t *testing.T) {
	before, preparation, rows := providerHistoryStageCompositionUnitRows()
	held, ok := providerHistoryCollectionStageWorkUpper(before, preparation, rows, 3, false)
	if !ok {
		t.Fatal("finite final-stage UNIT row rejected")
	}
	remaining, clocks, ok := rows.remaining(3)
	if !ok {
		t.Fatal("finite suffix rejected")
	}
	for _, coordinate := range []int{2, 10} {
		withoutPreparation := held.work[coordinate] - preparation.work[coordinate]
		guardNeed := before.work[coordinate] + preparation.work[coordinate] + remaining[coordinate]
		if guardNeed != held.work[coordinate] || guardNeed <= withoutPreparation {
			t.Fatal("omitting entered preparation no longer exposes exact final-stage shortfall", coordinate)
		}
	}
	if before.clocks+preparation.clocks+clocks != held.clocks || held.clocks-preparation.clocks >= before.clocks+preparation.clocks+clocks {
		t.Fatal("entered preparation clock shortfall hidden")
	}
	rows.advance = candidateWorkVector{}
	rows.advanceClocks, rows.advanceTailMS = 0, 0
	last, accepted := providerHistoryCollectionStageWorkUpper(before, preparation, rows, 3, false)
	if !accepted || last != held {
		t.Fatal("final stage borrowed or required unused advance")
	}
	if got, accepted := providerHistoryCollectionStageWorkUpper(before, preparation, rows, 3, true); accepted || got != (providerHistoryStageWorkCost{}) {
		t.Fatal("missing own-advance row admitted", got, accepted)
	}
}

func TestProviderHistoryStageCompositionRefusesUnfundedOrRPCBearingPrefixes(t *testing.T) {
	for _, mode := range []string{"negative-count", "zero-count", "large-count", "member", "prefix-request", "prefix-wire", "preparation-request", "preparation-wire", "preparation-work", "preparation-clocks", "advance"} {
		t.Run(mode, func(t *testing.T) {
			before, preparation, rows := providerHistoryStageCompositionUnitRows()
			count, advancing := 3, false
			switch mode {
			case "negative-count":
				count = -1
			case "zero-count":
				count = 0
			case "large-count":
				count = providerHistoryGroupMembers + 1
			case "member":
				rows.member[0] = 0
			case "prefix-request":
				before.work[0] = 1
			case "prefix-wire":
				before.work[1] = 1
			case "preparation-request":
				preparation.work[0] = 1
			case "preparation-wire":
				preparation.work[1] = 1
			case "preparation-work":
				preparation.work[10] = rows.member[10] + 1
			case "preparation-clocks":
				preparation.clocks = rows.memberClocks + 1
			case "advance":
				advancing, rows.advanceTailMS = true, 0
			}
			if got, ok := providerHistoryCollectionStageWorkUpper(before, preparation, rows, count, advancing); ok || got != (providerHistoryStageWorkCost{}) {
				t.Fatal("unfunded or wrong-domain UNIT accepted", mode, got, ok)
			}
		})
	}
}

func TestProviderHistoryStageCompositionReturnsWholeZeroOnLateOverflow(t *testing.T) {
	for _, mode := range []string{"remaining-wire", "remaining-decoded", "remaining-clocks", "prefix-decoded", "preparation-logical", "advance-decoded", "window-clocks", "advance-clocks"} {
		t.Run(mode, func(t *testing.T) {
			before, preparation, rows := providerHistoryStageCompositionUnitRows()
			count, advancing := 3, false
			switch mode {
			case "remaining-wire":
				rows.member[1] = ^uint64(0)
			case "remaining-decoded":
				rows.member[2] = ^uint64(0)
			case "remaining-clocks":
				rows.memberClocks = ^uint64(0)
			case "prefix-decoded":
				before.work[2] = ^uint64(0)
			case "preparation-logical":
				count = 2
				before.work[10] = 0
				rows.member[10] = ^uint64(0)/2 - 100
				preparation.work[10] = rows.member[10]
			case "advance-decoded":
				advancing, rows.advance[2] = true, ^uint64(0)
			case "window-clocks":
				before.clocks = ^uint64(0)
			case "advance-clocks":
				advancing = true
				before.clocks = ^uint64(0) - 13
			}
			if got, ok := providerHistoryCollectionStageWorkUpper(before, preparation, rows, count, advancing); ok || got != (providerHistoryStageWorkCost{}) {
				t.Fatal("overflow left partial work or clocks", mode, got, ok)
			}
		})
	}
	before, preparation, rows := providerHistoryStageCompositionUnitRows()
	before.clocks = ^uint64(0) - 14
	exact, ok := providerHistoryCollectionStageWorkUpper(before, preparation, rows, 3, true)
	if !ok || exact.clocks != ^uint64(0) {
		t.Fatal("exact scalar clock maximum rejected", exact, ok)
	}
}
