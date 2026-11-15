package registry011

import "testing"

// A scalar arithmetic domain, not a complete store, selected operating profile
// or clock. Literal expectations below deliberately use small metadata inputs.
func providerScanCostingTestDomain() providerScanBounds {
	return providerScanBounds{
		maximumRecords: 2, maximumChunks: 2, maximumClockReads: 1,
		maximumPreIClockReads: 64, maximumPreIRetainedBytes: 1 << 20,
		chunkWork:       candidateWorkVector{2: 11, 9: 13, 10: 17},
		maximumPreIWork: candidateWorkVector{2: 1 << 20, 9: 1 << 20, 10: 1 << 50},
	}
}

func TestProviderScanCostingMatchesLiteralIndependentRows(t *testing.T) {
	open, ok := providerScanOpenCostUpper(4, 8)
	wantOpen := providerScanUpper{work: candidateWorkVector{2: 8388608, 9: 589824, 10: 549764204544}, retained: 589928, clocks: 16}
	if !ok || open != wantOpen {
		t.Fatal("128 open events, nine raw equivalents and thirteen allocations", ok, open)
	}
	preparation, ok := providerScanPreparationCostUpper(4, 8, 3, 4)
	wantPreparation := providerScanUpper{work: candidateWorkVector{2: 2560, 9: 28, 10: 17920}, retained: 84, clocks: 4}
	if !ok || preparation != wantPreparation {
		t.Fatal("640 preparation events and seven independent allocations", ok, preparation)
	}
	scan := providerScanCostingTestDomain()
	opening := providerScanUpper{work: candidateWorkVector{2: 3, 9: 5, 10: 7}, retained: 9, clocks: 2}
	legacy, ok := providerScanInspectionCostUpper(scan, 4, 8, false, opening)
	first, firstOK := providerScanInspectionCostUpper(scan, 4, 8, true, opening)
	wantLegacy := providerScanUpper{work: candidateWorkVector{2: 54, 9: 58, 10: 2338}, retained: 96, clocks: 1}
	wantFirst := providerScanUpper{work: candidateWorkVector{2: 70, 9: 58, 10: 2338}, retained: 96, clocks: 1}
	if !ok || !firstOK || legacy != wantLegacy || first != wantFirst {
		t.Fatal("two original receipt decoders lost or double charged", ok, firstOK, legacy, first)
	}
	held := providerScanUpper{work: candidateWorkVector{2: 19, 9: 23, 10: 29}, retained: 31, clocks: 3}
	body, ok := providerScanBodyCostUpper(scan, opening, first, held)
	wantBody := providerScanUpper{work: candidateWorkVector{2: 89, 9: 81, 10: 2367}, retained: 127, clocks: 4}
	if !ok || body != wantBody || body.work[0] != 0 || body.work[1] != 0 {
		t.Fatal("first-E1 hold did not remain separate inside B", ok, body)
	}
}

func TestProviderScanCostingPreservesOriginalWrappersAndWitnessRequirement(t *testing.T) {
	legacy, _ := providerScanProvisionedFixture(t)
	first := providerFirstWriterBounds()
	withFirst, _ := providerScanProvisionedFixtureWithFirstPolicy(t, &first, true)
	for _, config := range []providerCustodyConfig{legacy, withFirst} {
		b := config.bounds
		if !b.valid() || !providerScanSetupFits(config) {
			t.Fatal("original positive structural profile")
		}
		opening := providerScanOpenUpper(config)
		prep, ok := providerScanPreparationCostUpper(b.maximumMetadataPerCommit, b.fileOverheadBytes, 1, b.scan.maximumAttempts)
		livePrep, err := providerScanPreparationUpper(config, 1)
		if !ok || err != nil || prep != livePrep {
			t.Fatal("preparation wrapper changed", err)
		}
		inspection, ok := providerScanInspectionCostUpper(*b.scan, b.maximumMetadataPerCommit, b.fileOverheadBytes, b.firstAdmission != nil, opening)
		liveInspection, err := providerScanInspectionUpper(config)
		if !ok || err != nil || inspection != liveInspection {
			t.Fatal("inspection wrapper changed", err)
		}
		var held providerScanUpper
		if b.firstAdmission != nil {
			held = providerScanUpper{b.firstAdmission.work, b.firstAdmission.retained, b.firstAdmission.clocks}
		}
		body, ok := providerScanBodyCostUpper(*b.scan, opening, inspection, held)
		liveBody, err := providerScanBodyUpper(config)
		if !ok || err != nil || body != liveBody {
			t.Fatal("body wrapper changed", err)
		}
	}
	withFirst.bounds.history = &providerHistoryProgramBounds{}
	b := withFirst.bounds
	if b.valid() || !b.validStructure() || providerScanSetupFits(withFirst) {
		t.Fatal("unminted history acquired open validity")
	}
	if p, err := providerScanPreparationUpper(withFirst, 1); err == nil || p != (providerScanUpper{}) {
		t.Fatal("preparation ignored missing history witness")
	}
	if p, err := providerScanInspectionUpper(withFirst); err == nil || p != (providerScanUpper{}) {
		t.Fatal("inspection ignored missing history witness")
	}
	if p, err := providerScanBodyUpper(withFirst); err == nil || p != (providerScanUpper{}) {
		t.Fatal("body ignored missing history witness")
	}
	in, ok := providerBootstrapCostingInputFromBounds(b)
	if !ok {
		t.Fatal("unminted history DATA extraction")
	}
	if work, physical, ok := in.bootstrapUpper(); !ok || work[2] == 0 || physical == 0 {
		t.Fatal("original enrollment costing remained circular")
	}
	if work, physical, ok := providerBootstrapUpper(b); ok || work != (candidateWorkVector{}) || physical != 0 {
		t.Fatal("enrollment wrapper ignored the missing witness")
	}
}

func TestProviderScanCostingRefusesInvalidDomainsAndOverflowWithoutPartialData(t *testing.T) {
	for _, args := range [][2]uint64{{0, 1}, {1<<20 + 1, 1}, {1, 0}, {1, 1<<20 + 1}} {
		if p, ok := providerScanOpenCostUpper(args[0], args[1]); ok || p != (providerScanUpper{}) {
			t.Fatal("invalid open parameters", args)
		}
		if p, ok := providerScanPreparationCostUpper(args[0], args[1], 1, 1); ok || p != (providerScanUpper{}) {
			t.Fatal("invalid preparation parameters", args)
		}
		if p, ok := providerScanInspectionCostUpper(providerScanCostingTestDomain(), args[0], args[1], true, providerScanUpper{}); ok || p != (providerScanUpper{}) {
			t.Fatal("invalid inspection parameters", args)
		}
	}
	for _, args := range [][2]uint64{{0, 1}, {2, 1}, {1, 0}, {1, 65}} {
		if p, ok := providerScanPreparationCostUpper(4, 8, args[0], args[1]); ok || p != (providerScanUpper{}) {
			t.Fatal("invalid ordinal or maximum", args)
		}
	}
	for _, name := range []string{"records", "chunks", "clock", "pre-i-clock", "physical", "requests", "wire", "decoded", "work", "written", "pre-i-wire", "chunk-capacity", "late-overflow"} {
		t.Run(name, func(t *testing.T) {
			b := providerScanCostingTestDomain()
			switch name {
			case "records":
				b.maximumRecords = (2 << 20) + 66
			case "chunks":
				b.maximumChunks = b.maximumRecords + 1
			case "clock":
				b.maximumClockReads = 0
			case "pre-i-clock":
				b.maximumPreIClockReads = 1<<20 + 1
			case "physical":
				b.maximumPreIRetainedBytes = 0
			case "requests":
				b.chunkWork[0] = 1
			case "wire":
				b.chunkWork[1] = 1
			case "decoded":
				b.chunkWork[2] = 0
			case "work":
				b.chunkWork[10] = 0
			case "written":
				b.maximumPreIWork[9] = 0
			case "pre-i-wire":
				b.maximumPreIWork[1] = 1
			case "chunk-capacity":
				b.chunkWork[3] = 1
			case "late-overflow":
				b.chunkWork[10], b.maximumPreIWork[10] = ^uint64(0), ^uint64(0)
			}
			if p, ok := providerScanInspectionCostUpper(b, 4, 8, true, providerScanUpper{}); ok || p != (providerScanUpper{}) {
				t.Fatal("invalid or overflowing inspection produced partial DATA", name)
			}
			if name != "late-overflow" {
				if providerScanCostingDomainValid(b) {
					t.Fatal("invalid-domain precursor remained valid", name)
				}
				if p, ok := providerScanBodyCostUpper(b, providerScanUpper{}, providerScanUpper{}, providerScanUpper{}); ok || p != (providerScanUpper{}) {
					t.Fatal("invalid domain produced a body quote", name)
				}
			}
		})
	}
	scan := providerScanCostingTestDomain()
	for _, first := range []providerScanUpper{{work: candidateWorkVector{10: ^uint64(0)}}, {retained: scan.maximumPreIRetainedBytes}, {clocks: scan.maximumPreIClockReads}} {
		inspection := providerScanUpper{work: candidateWorkVector{10: 1}, retained: 1, clocks: 1}
		if p, ok := providerScanBodyCostUpper(scan, providerScanUpper{}, inspection, first); ok || p != (providerScanUpper{}) {
			t.Fatal("overflowing or unfunded body produced partial DATA")
		}
	}
	b, _ := providerBootstrapCostingTestBounds(t)
	in, ok := providerBootstrapCostingInputFromBounds(b)
	if !ok {
		t.Fatal("positive enrollment costing precursor")
	}
	for _, entries := range []int{6, 7} {
		local := in
		local.maximumEntries = entries
		if entries == 7 {
			local.kind = providerBootstrapCostingGuardian
		}
		if !local.valid() {
			t.Fatal("intended enrollment object guard lost its structural precursor")
		}
		if work, physical, ok := local.bootstrapUpper(); ok || work != (candidateWorkVector{}) || physical != 0 {
			t.Fatal("initial seven/eight-object guard ignored", entries)
		}
	}
}
