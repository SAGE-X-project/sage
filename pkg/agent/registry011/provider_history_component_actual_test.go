package registry011

import (
	"reflect"
	"testing"
)

// These finite numeric subjects exercise composition DATA, never a source,
// admitted original window, native cost certificate or operating policy.
// The existing independent portable polynomial tests still check each Upper.
func TestProviderHistoryComponentActualRowsPreserveStandaloneVectors(t *testing.T) {
	l := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}
	zero := candidateWorkVector{}
	for _, encoding := range []string{EthereumJSONBlocks, "debug-raw-rlp-v1"} {
		entered, ok := providerHistoryBinaryWrapperActual(l, 65536, 1, 2048, 1024, 1, encoding)
		standalone, bounded := providerHistoryBinaryWrapperUpper(l, 65536, 1, 2048, 1024, 1, encoding)
		if !ok || !bounded || entered.actual[2] != 153600 || entered.actual[10] == 0 || entered.preflight != zero || entered.total != zero || standalone.preflight[10] == 0 {
			t.Fatal("binary actual DATA became a standalone reservation", encoding, entered, standalone)
		}
		standalone.preflight, standalone.total = zero, zero
		if entered != standalone {
			t.Fatal("binary component lost its original rows or shapes", encoding)
		}
		json, ok := providerHistoryJSONBranchActual(l, 65536, 1, 0, 2048, 1024, encoding)
		jsonBound, bounded := providerHistoryJSONBranchUpper(l, 65536, 1, 0, 2048, 1024, encoding)
		if !ok || !bounded || json.actual[10] == 0 || json.preflight != zero || json.total != zero {
			t.Fatal("JSON actual DATA carried admission room", encoding, json, jsonBound)
		}
		jsonBound.preflight, jsonBound.total = zero, zero
		if json != jsonBound {
			t.Fatal("JSON component lost its original rows or shapes", encoding)
		}
	}
	for _, profile := range []string{SepoliaAmsterdamGasProfile, SepoliaAmsterdamNethermindGasProfile, SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamRootExecutionGasProfile} {
		entered, ok := providerHistoryGasBranchActual(l, 65536, 1, 2048, 1024, 1, profile)
		standalone, bounded := providerHistoryGasBranchUpper(l, 65536, 1, 2048, 1024, 1, profile)
		if !ok || !bounded || entered.strict[2] != 373470 || entered.actual[10] == 0 || entered.preflight != zero || entered.total != zero || standalone.preflight[10] == 0 {
			t.Fatal("gas actual DATA lost the two strict passes or gained room", profile, entered, standalone)
		}
		standalone.preflight, standalone.total = zero, zero
		if entered != standalone {
			t.Fatal("gas component lost its original rows or shapes", profile)
		}
	}
	p := providerHistoryEffectBranchLimits{65536, 1, 2, 1, 2048, 1024, 1, 1}
	entered, ok := providerHistoryEffectBranchActual(l, p)
	standalone, bounded := providerHistoryEffectBranchUpper(l, p)
	if !ok || !bounded || entered.actual[2] != 4254015 || entered.actual[5] != 60 || entered.actual[6] != 30 || entered.actual[10] == 0 ||
		entered.preflight != zero || entered.total != zero || standalone.preflight[10] == 0 {
		t.Fatal("effect actual DATA lost separate normalization/execution entries or gained room", entered, standalone)
	}
	standalone.preflight, standalone.total = zero, zero
	if entered != standalone {
		t.Fatal("effect component lost its original rows or shapes")
	}
}

func TestProviderHistoryComponentActualRowsKeepFiniteWorkPastReserveOverflow(t *testing.T) {
	// Old wide standalone room exceeds uint64, while the entered row fits.
	// This does NOT imply that a real complete history is feasible or funded.
	l := providerHistoryCollectionSemanticLimits{Groups: 293, Requests: 293, Blocks: 1, Windows: 1}
	const journeys = uint64(4103)
	json, ok := providerHistoryJSONBranchActual(l, 65536, 1, 0, 2048, 1024, EthereumJSONBlocks)
	jsonBound, bounded := providerHistoryJSONBranchUpper(l, 65536, 1, 0, 2048, 1024, EthereumJSONBlocks)
	if !ok || bounded || jsonBound != (providerHistoryJSONBranchCost{}) || json.journeys != journeys ||
		json.actual[2] != 111169*journeys || json.actual[3] != journeys || json.actual[4] != journeys || json.actual[10] == 0 || json.preflight != (candidateWorkVector{}) || json.total != (candidateWorkVector{}) {
		t.Fatal("standalone JSON room overflow erased finite entered DATA", json, jsonBound, ok, bounded)
	}
	gas, ok := providerHistoryGasBranchActual(l, 65536, 1, 2048, 1024, 1, SepoliaAmsterdamGasProfile)
	gasBound, bounded := providerHistoryGasBranchUpper(l, 65536, 1, 2048, 1024, 1, SepoliaAmsterdamGasProfile)
	if !ok || bounded || gasBound != (providerHistoryGasBranchCost{}) || gas.journeys != journeys ||
		gas.actual[2] != 31042*journeys || gas.actual[10] == 0 || gas.preflight != (candidateWorkVector{}) || gas.total != (candidateWorkVector{}) {
		t.Fatal("standalone gas room overflow erased finite entered DATA", gas, gasBound, ok, bounded)
	}
	p := providerHistoryEffectBranchLimits{65536, 1, 2, 1, 2048, 1024, 1, 1}
	effect, ok := providerHistoryEffectBranchActual(l, p)
	effectBound, bounded := providerHistoryEffectBranchUpper(l, p)
	if !ok || bounded || effectBound != (providerHistoryEffectBranchCost{}) || effect.journeys != journeys ||
		effect.actual[2] != 283601*journeys || effect.actual[5] != 4*journeys || effect.actual[6] != 2*journeys || effect.actual[10] == 0 ||
		effect.preflight != (candidateWorkVector{}) || effect.total != (candidateWorkVector{}) {
		t.Fatal("standalone effect room overflow erased finite entered DATA", effect, effectBound, ok, bounded)
	}
	// The wrapper uses C rather than64MiB room; choose a separate finite
	// tuple whose small original binary E still fits but repeated C room fails.
	l.Groups, l.Requests = 30000, 30000
	binary, ok := providerHistoryBinaryWrapperActual(l, 65536, 1, 8<<20, 1, 0, EthereumJSONBlocks)
	binaryBound, bounded := providerHistoryBinaryWrapperUpper(l, 65536, 1, 8<<20, 1, 0, EthereumJSONBlocks)
	if !ok || bounded || binaryBound != (providerHistoryBinaryWrapperCost{}) || binary.journeys != 420001 ||
		binary.actual[2] != 327680*420001 || binary.actual[10] == 0 || binary.preflight != (candidateWorkVector{}) || binary.total != (candidateWorkVector{}) {
		t.Fatal("standalone wrapper room overflow erased finite entered DATA", binary, binaryBound, ok, bounded)
	}
}

func TestProviderHistoryComponentActualRowsRefuseWholeOnInvalidOrOverflow(t *testing.T) {
	for _, overflow := range []bool{false, true} {
		l := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 1, Windows: 1}
		evidence := uint64(0)
		if overflow {
			l.Groups, l.Requests, evidence = 30000, 30000, 64<<20
		}
		binary, ok := providerHistoryBinaryWrapperActual(l, evidence, 16384, 8<<20, 8<<20, 1, EthereumJSONBlocks)
		if ok || binary != (providerHistoryBinaryWrapperCost{}) {
			t.Fatal("invalid/overflow binary row leaked partial output", overflow, binary)
		}
		json, ok := providerHistoryJSONBranchActual(l, evidence, 16384, 65536, 8<<20, 8<<20, EthereumJSONBlocks)
		if ok || json != (providerHistoryJSONBranchCost{}) {
			t.Fatal("invalid/overflow JSON row leaked partial output", overflow, json)
		}
		gas, ok := providerHistoryGasBranchActual(l, evidence, 16384, 8<<20, 8<<20, 1, SepoliaAmsterdamRootExecutionGasProfile)
		if ok || gas != (providerHistoryGasBranchCost{}) {
			t.Fatal("invalid/overflow gas row leaked partial output", overflow, gas)
		}
		effect, ok := providerHistoryEffectBranchActual(l, providerHistoryEffectBranchLimits{evidence, 16384, 65536, 65536, 8 << 20, 8 << 20, 16384, 1})
		if ok || effect != (providerHistoryEffectBranchCost{}) {
			t.Fatal("invalid/overflow effect row leaked partial output", overflow, effect)
		}
	}
}

func TestProviderHistoryComponentActualRowsKeepDormantPathsDisjoint(t *testing.T) {
	l := providerHistoryCollectionSemanticLimits{Groups: 1, Requests: 1, Blocks: 2, Windows: 2}
	gas, ok := providerHistoryGasBranchActual(l, 65536, 1, 8192, 1024, 0, SepoliaAmsterdamGasProfile)
	if !ok || gas.journeys != 15 || gas.windows != 0 || gas.native != 0 || gas.actual != (candidateWorkVector{}) || gas.preflight != (candidateWorkVector{}) || gas.total != (candidateWorkVector{}) {
		t.Fatal("no native block invented gas entries or headroom", gas, ok)
	}
	effect, ok := providerHistoryEffectBranchActual(l, providerHistoryEffectBranchLimits{65536, 1, 1, 1, 8192, 1024, 0, 1})
	want := candidateWorkVector{10: 5370 + 60*uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size())}
	if !ok || effect.journeys != 15 || effect.selected != 0 || effect.windows != 0 || effect.normalization != (candidateWorkVector{}) || effect.execution != (candidateWorkVector{}) ||
		effect.actual != want || effect.collection != want || effect.preflight != (candidateWorkVector{}) || effect.total != (candidateWorkVector{}) {
		t.Fatal("no selected trace dropped required pending/receipt scans or gained room", effect, ok, want)
	}
}
