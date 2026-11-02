package registry011

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestProviderHistoryStateApplyBoundsMatchIndependentFullEventVector(t *testing.T) {
	// Independent expanded event totals on the ordinary64-bit DATA layout.
	// Coefficients count declared copies across the whole operation, rather
	// than recomputing the implementation's intermediate expressions.
	want := uint64(10008433368160)
	for _, size := range []struct{ copies, ordinary, target uint64 }{
		{1, 216, uint64(reflect.TypeFor[replayStream]().Size())},
		{2, 152, uint64(reflect.TypeFor[replayState]().Size())},
		{6911, 16, uint64(reflect.TypeFor[string]().Size())},
		{17665, 104, uint64(reflect.TypeFor[replayKey]().Size())},
		{96, 48, uint64(reflect.TypeFor[replayService]().Size())},
		{768, 32, uint64(reflect.TypeFor[replayGrant]().Size())},
		{2, 112, uint64(reflect.TypeFor[replayRecord]().Size())},
		{5, 24, uint64(reflect.TypeFor[[]replayGrant]().Size())},
		{1, 104, uint64(reflect.TypeFor[HistoryContentStep]().Size())},
		{1607, 8, uint64(reflect.TypeFor[map[string]any]().Size())},
		{7534, 16, uint64(reflect.TypeFor[any]().Size())},
	} {
		want = want - size.copies*size.ordinary + size.copies*size.target
	}
	upper, ok := providerHistoryStateApplyUpper()
	if !ok || upper != (candidateWorkVector{2: 99128, 7: 128, 10: want}) {
		t.Fatal("full apply differs from independent copied-type and codec/proof rows", upper, want, ok)
	}
}

func providerHistoryApplyCostCommand(t *testing.T, raw []byte) replayCommand {
	t.Helper()
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	command, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, raw, contract, providerHistoryStateUnitMeter())
	if !ok {
		t.Fatal("actual observed parser precursor outside the apply quote")
	}
	return command
}

func TestProviderHistoryStateApplyBoundsFundAllNormalStepsWithoutMeterReset(t *testing.T) {
	steps := replayTestSteps(t)
	upper, ok := providerHistoryStateApplyUpper()
	if !ok {
		t.Fatal("prospective per-apply allowance")
	}
	var total candidateWorkVector
	for coordinate, units := range upper {
		total[coordinate], ok = providerUpperProduct(units, uint64(len(steps)))
		if !ok {
			t.Fatal("checked prospective repeated-apply allowance")
		}
	}
	meter := providerHistoryContentUnitMeter(total)
	var state replayState
	for i, want := range steps {
		command := providerHistoryApplyCostCommand(t, want.History.Command)
		before := meter.actual
		got, accepted := state.applyObserved(replayTestRegistry, command, uint64(i+1), meter)
		if !accepted || meter.failed || !reflect.DeepEqual(got, want) || state.version != uint64(i+1) {
			t.Fatal("full quote changed the handwritten normal state history", i, meter.actual)
		}
		for coordinate, actual := range meter.actual {
			if actual < before[coordinate] || actual-before[coordinate] > upper[coordinate] {
				t.Fatal("apply escaped its prospective upper or reset cumulative work", i, coordinate)
			}
		}
	}
	if meter.actual[7] != 3 {
		t.Fatal("retained historical keys were verified as new proofs", meter.actual)
	}
}

func TestProviderHistoryStateApplyBoundsKeepIndependentShortagesAndOriginalState(t *testing.T) {
	command := providerHistoryApplyCostCommand(t, replayTestCreate(t).raw)
	upper, ok := providerHistoryStateApplyUpper()
	if !ok {
		t.Fatal("prospective full apply allowance")
	}
	normal := providerHistoryContentUnitMeter(upper)
	var completed replayState
	if _, ok := completed.applyObserved(replayTestRegistry, command, 1, normal); !ok || normal.failed || normal.actual[7] != 2 {
		t.Fatal("normal original create precursor")
	}
	for _, coordinate := range []int{2, 7, 10} {
		short := upper
		short[coordinate] = normal.actual[coordinate] - 1
		if coordinate == 10 {
			// Canonicalization has an original prospective preflight larger
			// than actual output. Select a small clone-prefix allowance; do
			// not label actualCPU-1 a final-proof boundary.
			short[10] = 1024
		}
		meter := providerHistoryContentUnitMeter(short)
		var state replayState
		got, accepted := state.applyObserved(replayTestRegistry, command, 1, meter)
		if accepted || !reflect.DeepEqual(got, HistoryContentStep{}) || !reflect.DeepEqual(state, replayState{}) || !meter.failed || meter.actual[10] == 0 {
			t.Fatal("quota refusal published private working state or hid consumed work", coordinate, meter.actual)
		}
		for i, actual := range meter.actual {
			if actual > short[i] {
				t.Fatal("apply spent an unadmitted coordinate", coordinate, i)
			}
		}
		before := meter.actual
		if _, accepted := state.applyObserved(replayTestRegistry, command, 1, meter); accepted || !meter.failed || meter.actual != before || !reflect.DeepEqual(state, replayState{}) {
			t.Fatal("failed apply observer or original state revived", coordinate)
		}
	}
}

func TestProviderHistoryStateApplyBoundsKeepServiceURICodecSeparate(t *testing.T) {
	const prefix = "https://example.invalid/"
	uri := prefix + strings.Repeat("a", 2048-len(prefix))
	service := replayService{Name: "api", Type: "A2A", URI: uri}
	original := providerHistoryStateUnitPrefix(t, 7)
	before := providerHistoryStateUnitSnapshot(t, original)
	fixture := replayTestCommand(t, "update-services", replayTestWallet, original.version, map[string]any{"services": []replayService{service}})
	command := providerHistoryApplyCostCommand(t, fixture.raw)
	upper, ok := providerHistoryStateApplyUpper()
	if !ok {
		t.Fatal("prospective full apply allowance")
	}
	normal := providerHistoryContentUnitMeter(upper)
	got, accepted := original.applyObserved(replayTestRegistry, command, 8, normal)
	var record replayRecord
	if err := json.Unmarshal(got.Record, &record); err != nil || !accepted || normal.failed || !reflect.DeepEqual(record.Services, []replayService{service}) || record.Version != "8" || normal.actual[2] != 2049 || normal.actual[7] != 0 {
		t.Fatal("normal maximum URI codec plus one admitted digit", normal.actual, err)
	}
	short := upper
	short[2] = 2047
	meter := providerHistoryContentUnitMeter(short)
	state := providerHistoryStateUnitSnapshot(t, before)
	if step, accepted := state.applyObserved(replayTestRegistry, command, 8, meter); accepted || !reflect.DeepEqual(step, HistoryContentStep{}) || !reflect.DeepEqual(state, before) || !meter.failed || meter.actual[2] != 0 || meter.actual[10] == 0 {
		t.Fatal("URI shortage was hidden by proof/Number allowance", meter.actual)
	}
	actual := meter.actual
	if _, accepted := state.applyObserved(replayTestRegistry, command, 8, meter); accepted || meter.actual != actual || !reflect.DeepEqual(state, before) {
		t.Fatal("failed URI apply observer reset")
	}
}
