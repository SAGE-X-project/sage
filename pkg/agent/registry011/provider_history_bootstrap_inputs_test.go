package registry011

import (
	"reflect"
	"testing"
)

func TestProviderHistoryFirstOperationFundsItsPreselectedReadOwners(t *testing.T) {
	b, capture := historyBootstrapInputBounds(t)
	args, argsOK := providerHistoryBootstrapInputsFromBounds(b)
	history, historyOK := providerBootstrapCostingInputFromBounds(b)
	if !argsOK || !historyOK || history.kind != providerBootstrapCostingHistory {
		t.Fatal("fresh unminted memory-only UNIT input")
	}
	// A separate numeric kind comparison, never an opened/changed profile.
	ordinary := history
	ordinary.kind = providerBootstrapCostingOrdinary
	base, baseOK := ordinary.firstOperationPlanUpper(capture, args.ordinal, args.records, args.retained, args.output)
	got, gotOK := history.firstOperationPlanUpper(capture, args.ordinal, args.records, args.retained, args.output)
	// Seven concrete preselected structs and one allocation pointer each:
	// parent, start, two independent reader/meter pairs and final owner.
	wantExtra := uint64(reflect.TypeFor[providerHistoryProgramOwner]().Size()) +
		uint64(reflect.TypeFor[providerHistoryProgramStart]().Size()) +
		2*uint64(reflect.TypeFor[providerHistoryRecoveryOwner]().Size()) +
		2*uint64(reflect.TypeFor[providerWorkMeter]().Size()) +
		uint64(reflect.TypeFor[providerHistoryFinalOwner]().Size()) +
		7*uint64(reflect.TypeFor[*byte]().Size())
	if !baseOK || !gotOK || got.memory != base.memory+wantExtra || got.storage != base.storage || got.capture != base.capture {
		t.Fatal("first-operation live set omitted independent preselected owner pointees", baseOK, gotOK)
	}
	compiled, ok := providerHistoryCompileBootstrapSource(b, capture)
	if !ok || compiled.source.firstOperationResident != got.memory || b.valid() || b.history.derivation != nil {
		t.Fatal("source compiler lost complete fixed owner layout or granted execution", ok)
	}
}

func TestProviderHistoryFirstOperationRefusesRoomForOnlyTheParentPointers(t *testing.T) {
	b, capture := historyBootstrapInputBounds(t)
	args, argsOK := providerHistoryBootstrapInputsFromBounds(b)
	history, historyOK := providerBootstrapCostingInputFromBounds(b)
	if !argsOK || !historyOK {
		t.Fatal("fresh unminted memory-only UNIT input")
	}
	ordinary := history
	ordinary.kind = providerBootstrapCostingOrdinary
	base, ok := ordinary.firstOperationPlanUpper(capture, args.ordinal, args.records, args.retained, args.output)
	if !ok {
		t.Fatal("ordinary numeric precursor before selecting insufficient history RAM")
	}
	// Select only the old parent/start allowance on fresh scalar inputs.
	// No live owner, root, clock or meter is modified or opened with it.
	oldExtra := uint64(reflect.TypeFor[providerHistoryProgramOwner]().Size()) +
		uint64(reflect.TypeFor[providerHistoryProgramStart]().Size()) +
		2*uint64(reflect.TypeFor[*byte]().Size())
	history.scan.maximumResidentBytes = base.memory + oldExtra
	if !history.valid() {
		t.Fatal("insufficient live-set selection lost its scalar domain")
	}
	if got, ok := history.firstOperationPlanUpper(capture, args.ordinal, args.records, args.retained, args.output); ok || got != (providerFirstOperationPlan{}) {
		t.Fatal("parent-only memory room admitted two uncounted readers and meters", ok)
	}
}

func TestProviderScanBodyOutputPreservesLiteralOriginalEnvelopes(t *testing.T) {
	legacy, legacyOK := providerScanBodyOutputUpper(4, 3, 8, false)
	first, firstOK := providerScanBodyOutputUpper(4, 3, 8, true)
	if !legacyOK || !firstOK || legacy != 24 || first != 2520 {
		t.Fatal("three original chunks lost output copies, receipt maps or target scratch", legacy, first)
	}
}

func TestProviderScanBodyOutputRefusesInvalidFiniteInputs(t *testing.T) {
	for _, row := range []struct {
		name                       string
		metadata, chunks, overhead uint64
	}{
		{"metadata-zero", 0, 3, 8},
		{"metadata-high", (1 << 20) + 1, 3, 8},
		{"chunks-zero", 4, 0, 8},
		{"chunks-high", 4, (2 << 20) + 66, 8},
		{"overhead-zero", 4, 3, 0},
		{"overhead-high", 4, 3, (1 << 20) + 1},
	} {
		t.Run(row.name, func(t *testing.T) {
			if got, ok := providerScanBodyOutputUpper(row.metadata, row.chunks, row.overhead, true); ok || got != 0 {
				t.Fatal("invalid output domain published a partial upper", got)
			}
		})
	}
}

// This is an independently selected, unminted arithmetic input. The roots
// provisioned by the fixture keep their original profile and are not opened
// with these copied bounds or a synthetic history derivation.
func historyBootstrapInputBounds(t *testing.T) (providerIntegrationBounds, ProviderCaptureLimits) {
	t.Helper()
	b, capture := providerBootstrapCostingTestBounds(t)
	scan := *b.scan
	scan.maximumChunks = 2
	b.scan, b.history = &scan, &providerHistoryProgramBounds{}
	if !b.validStructure() || b.valid() {
		t.Fatal("unminted independent prefix input lost its original structural domain")
	}
	return b, capture
}

func TestProviderHistoryBootstrapCompilerDerivesItsOwnPrefixArguments(t *testing.T) {
	b, capture := historyBootstrapInputBounds(t)
	before, scan, first := b, *b.scan, *b.firstAdmission
	compiled, ok := providerHistoryCompileBootstrapSource(b, capture)
	want := providerHistoryBootstrapSourceInputs{ordinal: 1, records: 2, retained: 1261568, output: 85256}
	if !ok || compiled.inputs != want || compiled.source.selectedPreI != (providerScanUpper{b.scan.maximumPreIWork, b.scan.maximumPreIRetainedBytes, b.scan.maximumPreIClockReads}) {
		t.Fatal("source binding did not use the complete original prefix and output envelopes", ok, compiled.inputs)
	}
	conditional, conditionalOK := providerHistoryBootstrapSourceUpper(b, capture, want.ordinal, want.records, want.retained, want.output)
	if !conditionalOK || compiled.source != conditional || b != before || *b.scan != scan || *b.firstAdmission != first || b.history.derivation != nil || b.valid() {
		t.Fatal("binding changed source costs or created execution authority")
	}
}

func TestProviderHistoryBootstrapInputsCoverTheGenuineFreshFinisher(t *testing.T) {
	s, f := providerFirstOperationPreparedFixture(t)
	work, clocks, retained, held, state := f.meter.actual, s.clock.read, s.retainedRaw, s.scanControl.held, f.state
	b := s.config.bounds
	// Numeric history-kind selection only; the actual store keeps its genuine
	// ordinary profile and original prepared owner throughout this comparison.
	b.history = &providerHistoryProgramBounds{}
	inputs, ok := providerHistoryBootstrapInputsFromBounds(b)
	if !ok || inputs.ordinal != f.scanPin.ordinal || inputs.records != f.records || inputs.retained < retained || inputs.output != f.readerOutput {
		t.Fatal("derived prefix inputs did not cover the actual original first finisher", ok, inputs)
	}
	if f.meter.actual != work || s.clock.read != clocks || s.retainedRaw != retained || s.scanControl.held != held || f.state != state || s.config.bounds.history != nil || s.activeMeter.Load() != nil {
		t.Fatal("numeric input binding changed the genuine owner, clock or reservation")
	}
}

func TestProviderHistoryBootstrapInputsRefuseIncompleteOriginalSelection(t *testing.T) {
	base, capture := historyBootstrapInputBounds(t)
	for _, name := range []string{"history", "guardian", "first", "chunks", "metadata", "enrollment-entries", "retained"} {
		t.Run(name, func(t *testing.T) {
			b, scan, first := base, *base.scan, *base.firstAdmission
			b.scan, b.firstAdmission = &scan, &first
			switch name {
			case "history":
				b.history = nil
			case "guardian":
				b.guardian = &providerGuardianBounds{}
			case "first":
				b.firstAdmission = nil
			case "chunks":
				scan.maximumChunks = 1
			case "metadata":
				b.maximumMetadataPerCommit = 0
			case "enrollment-entries":
				b.maximumEntries = 6
			case "retained":
				b.maximumStoredBytes, b.setupLive, b.setupRetained = 1<<20, 1<<20, 1<<20
				scan.maximumPreIRetainedBytes = 1 << 20
				first.retained, first.live = 1<<20, 1<<20
				first.work[9] = min(first.work[9], first.retained)
			}
			if (name == "chunks" || name == "enrollment-entries" || name == "retained") && !b.validStructure() {
				t.Fatal("numeric refusal lost its positive structural precursor", name)
			}
			if got, ok := providerHistoryBootstrapInputsFromBounds(b); ok || got != (providerHistoryBootstrapSourceInputs{}) {
				t.Fatal("incomplete original selection published prefix arguments", name, got)
			}
			if got, ok := providerHistoryCompileBootstrapSource(b, capture); ok || got != (providerHistoryCompiledBootstrapCost{}) {
				t.Fatal("incomplete input selection published a compiled source prefix", name)
			}
		})
	}
}

func TestProviderHistoryBootstrapCompilerPublishesNoPartialBoundPrefix(t *testing.T) {
	base, capture := historyBootstrapInputBounds(t)
	for _, name := range []string{"capture", "first-work"} {
		t.Run(name, func(t *testing.T) {
			b, first, global := base, *base.firstAdmission, capture
			b.firstAdmission = &first
			if name == "capture" {
				global.Timeout = 0
			} else {
				first.work[2] = 1
			}
			if inputs, ok := providerHistoryBootstrapInputsFromBounds(b); !ok || inputs == (providerHistoryBootstrapSourceInputs{}) {
				t.Fatal("downstream source refusal did not retain valid derived inputs", name)
			}
			if got, ok := providerHistoryCompileBootstrapSource(b, global); ok || got != (providerHistoryCompiledBootstrapCost{}) {
				t.Fatal("failed source compilation published its otherwise valid input prefix", name)
			}
		})
	}
}
