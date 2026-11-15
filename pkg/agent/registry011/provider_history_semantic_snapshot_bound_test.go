package registry011

import (
	"context"
	"reflect"
	"testing"
)

func TestProviderHistorySemanticSnapshotBoundsMatchIndependentCompositionRows(t *testing.T) {
	confirmedLimits := providerHistorySemanticCloneLimits{3, 5, 2, 1, 2}
	workingLimits := providerHistorySemanticCloneLimits{17, 19, 4, 2, 3}
	confirmed, a := providerHistorySemanticCloneUpper(confirmedLimits)
	working, b := providerHistorySemanticCloneUpper(workingLimits)
	if !a || !b {
		t.Fatal("separately checked clone component precursor")
	}
	for _, tc := range []struct {
		window, pending bool
		queue, row64    uint64
	}{
		{false, false, 0, 0},
		{true, false, 0, 1},
		{true, true, 0, 524291},
		{true, false, 1, 1048662},
		{true, true, 3, 3670274},
		{true, true, 4096, 4295839747},
	} {
		l := providerHistorySemanticSnapshotLimits{Confirmed: confirmedLimits, Queue: tc.queue, Window: tc.window, Pending: tc.pending}
		want := confirmed[10] + uint64(reflect.TypeFor[providerHistorySemanticFold]().Size()) + 1
		if tc.window {
			l.Working = workingLimits
			// Literals count four bounded text copies and one item put per
			// entry, plus one array event. Only the two item copy byte sizes
			// per entry adjust from the independent40-byte DATA layout.
			row := tc.row64 - 2*tc.queue*40 + 2*tc.queue*uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size())
			want += working[10] + uint64(reflect.TypeFor[providerHistoryExpectedWindow]().Size()) + uint64(reflect.TypeFor[replayRun]().Size()) + 1 + row
		}
		got, ok := providerHistorySemanticSnapshotUpper(l)
		if !ok || got != (candidateWorkVector{10: want}) {
			t.Fatal("snapshot composition differs from independent event rows", tc, got, want, ok)
		}
	}
}

func historySemanticSnapshotCostFixture(t *testing.T, mode string) (*providerHistorySemanticFold, *providerWorkMeter, providerHistorySemanticSnapshotLimits) {
	t.Helper()
	f, in, meter := historySemanticObservedFixture(t, false)
	switch mode {
	case "pending":
		if f.beginWindowCandidate(context.Background()) != nil || !f.window.prepareSubject(f.window.run, replayTestCreate(t)) {
			t.Fatal("normal pending command precursor")
		}
	case "queued":
		historySemanticApply(t, f, in, 0, 3)
	case "confirmed-and-working":
		historySemanticApply(t, f, in, 0, 1)
		historySemanticCompare(t, f, in, 0, 1)
		historySemanticApply(t, f, in, 2, 4)
	case "no-window":
	default:
		t.Fatal("unknown finite semantic snapshot fixture")
	}
	// Original selected configuration supplies prospective maxima, not a
	// report of actual counts or queue bytes. No live authority is minted.
	config := f.selection.config
	counts := providerHistorySemanticCloneLimits{uint64(config.MaxClaims), uint64(config.MaxIdentities), 128, 16, 128}
	l := providerHistorySemanticSnapshotLimits{Confirmed: counts}
	if mode != "no-window" {
		l.Window, l.Pending, l.Queue, l.Working = true, true, 4096, counts
	}
	return f, meter, l
}

func TestProviderHistorySemanticSnapshotBoundsFundOriginalOwnedGraphs(t *testing.T) {
	for _, mode := range []string{"no-window", "pending", "queued", "confirmed-and-working"} {
		f, meter, l := historySemanticSnapshotCostFixture(t, mode)
		upper, ok := providerHistorySemanticSnapshotUpper(l)
		if !ok || upper[10] > meter.upper[10]-meter.actual[10] {
			t.Fatal("prospective snapshot allowance within original UNIT limit")
		}
		before := f.commitmentObserved(providerHistorySemanticUnitMeter())
		start := meter.actual
		// Reduce this inert UNIT meter's allowance before the scoped call;
		// retain the same original observer and cumulative actual counters.
		meter.upper[10] = start[10] + upper[10]
		got, success := f.snapshotObserved(meter)
		if !success || got == nil || before == "" || meter.failed || got.observer != meter || f.observer != meter || got == f ||
			got.commitmentObserved(providerHistorySemanticUnitMeter()) != before || f.commitmentObserved(providerHistorySemanticUnitMeter()) != before {
			t.Fatal("quoted original snapshot changed source/observer/graph meaning", mode)
		}
		for coordinate, actual := range meter.actual {
			if actual < start[coordinate] || actual-start[coordinate] > upper[coordinate] {
				t.Fatal("original snapshot exceeded scoped prospective row or reset actual", mode, coordinate)
			}
		}
		if mode != "no-window" && (got.window == f.window || got.window.run == f.window.run) {
			t.Fatal("snapshot reused the original working graph", mode)
		}
	}
}

func TestProviderHistorySemanticSnapshotBoundsCannotTransferOriginalObserver(t *testing.T) {
	f, original, l := historySemanticSnapshotCostFixture(t, "confirmed-and-working")
	upper, ok := providerHistorySemanticSnapshotUpper(l)
	if !ok {
		t.Fatal("prospective snapshot row")
	}
	before := f.commitmentObserved(providerHistorySemanticUnitMeter())
	replacement := providerHistoryContentUnitMeter(upper)
	got, success := f.snapshotObserved(replacement)
	if success || got != nil || before == "" || replacement.actual != (candidateWorkVector{}) || f.observer != original ||
		f.commitmentObserved(providerHistorySemanticUnitMeter()) != before || !f.valid() {
		t.Fatal("numeric quote transferred execution to another observer")
	}
}

func TestProviderHistorySemanticSnapshotBoundsKeepSourceAfterLateQuotaFailure(t *testing.T) {
	probe, meter, l := historySemanticSnapshotCostFixture(t, "confirmed-and-working")
	upper, ok := providerHistorySemanticSnapshotUpper(l)
	start := meter.actual[10]
	if got, success := probe.snapshotObserved(meter); !ok || !success || got == nil || meter.failed || meter.actual[10] <= start || meter.actual[10]-start > upper[10] {
		t.Fatal("normal original snapshot and prospective-bound precursor")
	}
	work := meter.actual[10] - start
	f, original, _ := historySemanticSnapshotCostFixture(t, "confirmed-and-working")
	before := f.commitmentObserved(providerHistorySemanticUnitMeter())
	if before == "" || work > original.upper[10]-original.actual[10] {
		t.Fatal("independent unchanged-source and original UNIT allowance precursor")
	}
	original.upper[10] = original.actual[10] + work - 1
	got, success := f.snapshotObserved(original)
	if success || got != nil || !original.failed || f.observer != original || f.commitmentObserved(providerHistorySemanticUnitMeter()) != before {
		t.Fatal("late quota shortage published a partial snapshot or changed its source")
	}
}

func TestProviderHistorySemanticSnapshotBoundsRefuseInvalidPlanCombinations(t *testing.T) {
	invalidCounts := []providerHistorySemanticCloneLimits{{Claims: 1<<20 + 1}, {Created: 1<<20 + 1}, {Keys: 129}, {Services: 17}, {Grants: 129}}
	for _, counts := range invalidCounts {
		for _, l := range []providerHistorySemanticSnapshotLimits{{Confirmed: counts}, {Window: true, Working: counts}} {
			if got, ok := providerHistorySemanticSnapshotUpper(l); ok || got != (candidateWorkVector{}) {
				t.Fatal("invalid cloned state returned a partial snapshot upper", l)
			}
		}
	}
	for _, l := range []providerHistorySemanticSnapshotLimits{{Queue: 1}, {Pending: true}, {Working: providerHistorySemanticCloneLimits{Keys: 1}},
		{Window: true, Queue: 4097}, {Window: true, Queue: ^uint64(0)}} {
		if got, ok := providerHistorySemanticSnapshotUpper(l); ok || got != (candidateWorkVector{}) {
			t.Fatal("impossible or out-of-range window returned an upper", l)
		}
	}
}
