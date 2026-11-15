package registry011

import "testing"

func TestProviderHistoryBootstrapSourceKeepsOriginalHoldsDisjoint(t *testing.T) {
	b, capture := providerBootstrapCostingTestBounds(t)
	b.history = &providerHistoryProgramBounds{}
	quote, ok := providerHistoryBootstrapSourceUpper(b, capture, 1, 2, 0, 1)
	if !ok || b.valid() || !b.validStructure() || b.history.derivation != nil {
		t.Fatal("positive prefix DATA must precede, never replace, a derivation", ok)
	}
	// The 16KiB metadata / 4KiB overhead fixture has 128 open and512
	// enrollment schema equivalents, including every distinct allocation.
	if quote.opening.retained != 643072 || quote.enrollment.retained != 372736 || quote.enrollment.work[2] != 33554432 || quote.enrollment.work[9] != 315392 || quote.enrollment.work[10] != 2199056818176 || quote.opening.clocks != 16 || quote.enrollment.clocks != 8 {
		t.Fatal("independent original opening/enrollment source rows changed", quote.opening, quote.enrollment)
	}
	if quote.selectedSetup != (providerScanUpper{b.setupWork, b.setupRetained, b.setupMaximumClockReads}) || quote.selectedPreI != (providerScanUpper{b.scan.maximumPreIWork, b.scan.maximumPreIRetainedBytes, b.scan.maximumPreIClockReads}) {
		t.Fatal("source minima replaced the original selected holds")
	}
	for coordinate := range quote.preI.work {
		if quote.preI.work[coordinate] != quote.selectedSetup.work[coordinate]+quote.preparation.work[coordinate]+quote.body.work[coordinate] ||
			quote.body.work[coordinate] != quote.inspection.work[coordinate]+b.firstAdmission.work[coordinate] {
			t.Fatal("open or selected F was charged twice, omitted or funded by startup", coordinate)
		}
	}
	if quote.preI.retained != quote.selectedSetup.retained+quote.preparation.retained+quote.body.retained || quote.body.retained != quote.inspection.retained+b.firstAdmission.retained || quote.preI.clocks != quote.selectedSetup.clocks+quote.preparation.clocks+quote.body.clocks || quote.preI.work[0] != 0 || quote.preI.work[1] != 0 {
		t.Fatal("pre-I acquired network permission or lost original physical/clock holding")
	}
	in, ok := providerBootstrapCostingInputFromBounds(b)
	first, firstOK := in.firstAdmissionPlanUpper(1, 2, 0, 1)
	operation, operationOK := in.firstOperationPlanUpper(capture, 1, 2, 0, 1)
	if !ok || !firstOK || !operationOK || quote.first.work[2] != first.work[2]+6815744 || quote.first.retained != first.physical || quote.first.clocks != 18 || quote.firstOperationResident != operation.memory {
		t.Fatal("firstExtra did not remain in its original F prefix")
	}
	if quote.startup.work != operation.reservation.work || quote.startup.retained != operation.storage.physical || quote.startup.clocks != 12 || quote.startup.work[0] != 1 || quote.startup.work[1] != uint64(operation.capture.WireBytes) {
		t.Fatal("first P borrowed or repeated F/open/scan work")
	}
}

func TestProviderHistoryBootstrapSourceRefusesUnfundedOriginalDomains(t *testing.T) {
	base, capture := providerBootstrapCostingTestBounds(t)
	base.history = &providerHistoryProgramBounds{}
	positive, ok := providerHistoryBootstrapSourceUpper(base, capture, 1, 2, 0, 1)
	if !ok {
		t.Fatal("positive source-prefix precursor")
	}
	for _, name := range []string{"history", "guardian", "first", "setup-work", "setup-physical", "setup-clock", "first-work", "first-physical", "first-clock", "pre-i-work", "pre-i-physical", "pre-i-body", "pre-i-preparation", "pre-i-clock", "ordinal", "output-overflow", "capture"} {
		t.Run(name, func(t *testing.T) {
			b, first, scan, global := base, *base.firstAdmission, *base.scan, capture
			b.firstAdmission, b.scan = &first, &scan
			ordinal, output := uint64(1), uint64(1)
			switch name {
			case "history":
				b.history = nil
			case "guardian":
				b.guardian = &providerGuardianBounds{}
			case "first":
				b.firstAdmission = nil
			case "setup-work":
				b.setupWork[2] = positive.opening.work[2] + positive.enrollment.work[2] - 1
			case "setup-physical":
				b.setupRetained = positive.opening.retained + positive.enrollment.retained - 1
				b.setupLive = b.setupRetained
			case "setup-clock":
				b.setupMaximumClockReads = 23
			case "first-work":
				first.work[2] = positive.first.work[2] - 1
			case "first-physical":
				first.retained = positive.first.retained - 1
				first.live, first.work[9] = first.retained, min(first.work[9], first.retained)
			case "first-clock":
				first.clocks = 17
			case "pre-i-work":
				scan.maximumPreIWork[2] = first.work[2]
			case "pre-i-physical":
				scan.maximumPreIRetainedBytes = first.retained
			case "pre-i-body":
				scan.maximumPreIRetainedBytes = positive.selectedSetup.retained + positive.body.retained - 1
			case "pre-i-preparation":
				scan.maximumPreIRetainedBytes = positive.preI.retained - 1
			case "pre-i-clock":
				scan.maximumPreIClockReads, scan.maximumClockReads = first.clocks, first.clocks
			case "ordinal":
				ordinal = 0
			case "output-overflow":
				output = ^uint64(0)
			case "capture":
				global.Timeout = 0
			}
			if name != "guardian" && !b.validStructure() {
				t.Fatal("intended source-capacity refusal lost its structural precursor", name)
			}
			if quote, ok := providerHistoryBootstrapSourceUpper(b, global, ordinal, 2, 0, output); ok || quote != (providerHistoryBootstrapSourceCost{}) {
				t.Fatal("incomplete or underfunded original prefix produced partial costs", name)
			}
		})
	}
}
