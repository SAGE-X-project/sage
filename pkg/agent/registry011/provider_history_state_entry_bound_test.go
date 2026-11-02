package registry011

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

func TestProviderHistoryStateEntryBoundsMatchHandwrittenEventRows(t *testing.T) {
	for _, tc := range []struct{ removed, units64 uint64 }{{0, 2474575515430}, {1, 2474669901254}, {128, 2486671531302}} {
		// Independent fixed-schema event totals use map8/string16/any16.
		// Only actual declared copy bytes change with the compiled target.
		maps, texts, values := 3+2*tc.removed, 22+4*tc.removed, 22+8*tc.removed
		want := tc.units64 - maps*8 - texts*16 - values*16
		want += maps*uint64(reflect.TypeFor[map[string]any]().Size()) + texts*uint64(reflect.TypeFor[string]().Size()) + values*uint64(reflect.TypeFor[any]().Size())
		got, ok := providerHistoryParsedStateEntryUpper(tc.removed)
		if !ok || got != (candidateWorkVector{2: 16, 10: want}) {
			t.Fatal("parsed history entry differs from handwritten rows", tc, got, want, ok)
		}
	}
}

func historyStateEntryCostCommand(t *testing.T, step HistoryContentStep) (replayCommand, []replayGrant) {
	t.Helper()
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal("normal adopted ABI fixture", err)
	}
	command, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, step.History.Command, contract, providerHistorySemanticUnitMeter())
	if !ok || len(command.op) > 18 || len(command.raw) > 262144 || len(step.Record) > 65536 || len(step.Grants) > 262144 {
		t.Fatal("observed parsed-command and apply-caller byte envelope precursor")
	}
	var expected struct {
		Removed []replayGrant `json:"automaticRemovals"`
	}
	if err := json.Unmarshal(step.History.Entry, &expected); err != nil {
		t.Fatal("independent handwritten removal list", err)
	}
	return command, expected.Removed
}

func TestProviderHistoryStateEntryBoundsFundOriginalNormalHistory(t *testing.T) {
	for i, step := range replayTestSteps(t) {
		command, removed := historyStateEntryCostCommand(t, step)
		upper, ok := providerHistoryParsedStateEntryUpper(128)
		if !ok {
			t.Fatal("prospective original removal ceiling")
		}
		previous := ""
		if i != 0 {
			previous = strconv.Itoa(i)
		}
		meter := providerHistoryContentUnitMeter(upper)
		raw, ok := replayStateEntryObserved(strconv.Itoa(i+1), previous, command, uint64(i+1), removed, step.Record, step.Grants, meter)
		if !ok || meter.failed || !bytes.Equal(raw, step.History.Entry) {
			t.Fatal("original parsed history differs from independent handwritten entry", i)
		}
		for coordinate, actual := range meter.actual {
			if actual > upper[coordinate] {
				t.Fatal("history entry exceeded prospective coordinate", i, coordinate)
			}
		}
	}
}

func TestProviderHistoryStateEntryBoundsKeepIndependentAdmittedDecodedQuota(t *testing.T) {
	step := replayTestSteps(t)[0]
	command, _ := historyStateEntryCostCommand(t, step)
	for _, count := range []int{0, 128} {
		// The removal list is ordinary bounded primitive DATA, not a claim
		// that a live create operation actually removed these operators.
		removed := make([]replayGrant, count)
		for i := range removed {
			removed[i] = replayGrant{Operator: replayTestOperator, Scope: "deactivate"}
		}
		var expected map[string]any
		if err := json.Unmarshal(step.History.Entry, &expected); err != nil {
			t.Fatal(err)
		}
		const now uint64 = 1<<53 - 1
		expected["admitted"], expected["automaticRemovals"] = now, removed
		want, err := rfc8785.Marshal(expected)
		if err != nil {
			t.Fatal("independent typed entry-byte oracle", err)
		}
		upper, ok := providerHistoryParsedStateEntryUpper(uint64(count))
		if !ok || upper[2] != 16 {
			t.Fatal("independent admitted Number decoded row")
		}
		normal := providerHistoryContentUnitMeter(upper)
		if raw, ok := replayStateEntryObserved("1", "", command, now, removed, step.Record, step.Grants, normal); !ok || !bytes.Equal(raw, want) || normal.failed || normal.actual[2] != 16 || normal.actual[10] > upper[10] {
			t.Fatal("normal admitted-number codec precursor", count, normal.actual)
		}
		short := upper
		short[2] = 15
		meter := providerHistoryContentUnitMeter(short)
		if raw, ok := replayStateEntryObserved("1", "", command, now, removed, step.Record, step.Grants, meter); ok || raw != nil || !meter.failed || meter.actual[2] != 0 || meter.actual[10] == 0 || meter.actual[10] > upper[10] {
			t.Fatal("decoded shortage was hidden by ample history logical work", count, meter.actual)
		}
		before := meter.actual
		if raw, ok := replayStateEntryObserved("1", "", command, now, removed, step.Record, step.Grants, meter); ok || raw != nil || !meter.failed || meter.actual != before {
			t.Fatal("failed original history observer revived on retry", count)
		}
	}
}

func TestProviderHistoryStateEntryBoundsCoverEnteredOutputRefusal(t *testing.T) {
	step := replayTestSteps(t)[0]
	command, _ := historyStateEntryCostCommand(t, step)
	upper, ok := providerHistoryParsedStateEntryUpper(0)
	if !ok {
		t.Fatal("prospective no-removal entry row")
	}
	meter := providerHistoryContentUnitMeter(upper)
	// Safe in-memory ASCII escaping DATA only: not a live version, executable
	// content or peer message. Source bytes fit the walker; canonical output
	// exceeds its later262144 cap and still spends the original encode work.
	raw, ok := replayStateEntryObserved(strings.Repeat("\x01", 45000), "", command, 1, nil, step.Record, step.Grants, meter)
	if ok || raw != nil || meter.failed || meter.actual[2] != 1 || meter.actual[10] == 0 || meter.actual[10] > upper[10] || meter.actual != (candidateWorkVector{2: 1, 10: meter.actual[10]}) {
		t.Fatal("entered output-cap refusal escaped its observed envelope", meter.actual)
	}
}

func TestProviderHistoryStateEntryBoundsRefuseInvalidDomainAndAdmissionTime(t *testing.T) {
	for _, count := range []uint64{129, ^uint64(0)} {
		if got, ok := providerHistoryParsedStateEntryUpper(count); ok || got != (candidateWorkVector{}) {
			t.Fatal("out-of-domain removed list returned a partial upper", count, got, ok)
		}
	}
	step := replayTestSteps(t)[0]
	command, _ := historyStateEntryCostCommand(t, step)
	upper, ok := providerHistoryParsedStateEntryUpper(0)
	if !ok {
		t.Fatal("normal scalar row")
	}
	meter := providerHistoryContentUnitMeter(upper)
	if raw, ok := replayStateEntryObserved("1", "", command, 1<<53, nil, step.Record, step.Grants, meter); ok || raw != nil || meter.failed || meter.actual != (candidateWorkVector{}) {
		t.Fatal("original invalid-time guard entered history work")
	}
}
