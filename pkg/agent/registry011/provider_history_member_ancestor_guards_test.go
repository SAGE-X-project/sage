package registry011

import (
	"reflect"
	"testing"
)

// A linked, private comparable-DATA subject only. It has no store, clock,
// original dispatch registration, selected replay, retained file or authority.
// The production constructor must still supply all of those prerequisites.
func historyMemberAncestorUnitGraph() *providerHistoryMemberReplaySeal {
	r := &providerHistoryRecoveryOwner{}
	initial := &providerHistoryRecoveryCheckpoint{recovery: r}
	initial.owner = initial
	consumption := &providerHistoryRecoveryCheckpoint{recovery: r, previous: initial}
	consumption.owner = consumption
	r.initial, r.consumption = initial, consumption
	q := &providerHistoryMemberReplaySeal{recovery: r, dispatch: &providerHistoryGroupDispatch{}}
	q.owner = q
	q.fixed.bounds.records = 1
	return q
}

func TestProviderHistoryMemberAncestorCaptureRejectsIncompletePrivateGraphs(t *testing.T) {
	for _, mode := range []string{"missing-initial", "overflow", "broken-chain", "missing-archive-order", "unselected-archive"} {
		t.Run(mode, func(t *testing.T) {
			q := historyMemberAncestorUnitGraph()
			switch mode {
			case "missing-initial":
				q.recovery.initial = nil
			case "overflow":
				q.fixed.bounds.records = ^uint64(0)
			case "broken-chain":
				q.recovery.consumption.previous = nil
			case "missing-archive-order":
				q.recovery.consumption.memberArchive = &providerHistoryRecoveryMemberArchive{}
			case "unselected-archive":
				q.recovery.consumption.memberArchive = &providerHistoryRecoveryMemberArchive{order: &providerHistoryGroupOrder{}}
			}
			meter := providerReplayUnitMeter()
			if q.captureAncestorsObservedLocked(meter) || meter.failed || q.dispatch.memberReplaySelectedLocked() || q.recovery.memberReplaySeal != nil {
				t.Fatal("incomplete private ancestry produced a complete capture or dispatch selection", mode)
			}
		})
	}
	// Two list visits each cost one. The next stage reserves exact array
	// storage; comparison-weight traversal starts only after that event.
	storage := 2*uint64(reflect.TypeFor[providerHistoryMemberAncestor]().Size()) + 1
	for _, fuel := range []uint64{0, 2, 2 + storage} {
		q := historyMemberAncestorUnitGraph()
		meter := providerReplayUnitMeter()
		meter.upper[10] = fuel
		if q.captureAncestorsObservedLocked(meter) || !meter.failed || meter.actual[10] != fuel || q.dispatch.memberReplaySelectedLocked() {
			t.Fatal("unfunded ancestry stage exposed a completed selection or lost charged work", fuel, meter.actual)
		}
		charged := meter.actual
		if q.captureAncestorsObservedLocked(meter) || meter.actual != charged {
			t.Fatal("partial ancestry capture renewed its observer")
		}
	}
}

func TestProviderHistoryMemberAncestorComparisonRejectsChangedPrivateShape(t *testing.T) {
	for _, mode := range []string{"same", "missing-initial", "too-many-rows", "missing-checkpoint", "missing-member-weights"} {
		t.Run(mode, func(t *testing.T) {
			q := historyMemberAncestorUnitGraph()
			meter := providerReplayUnitMeter()
			if !q.captureAncestorsObservedLocked(meter) || len(q.ancestors) != 2 || !q.ancestorsMatchObservedLocked(meter) {
				t.Fatal("complete private two-node ancestry DATA precursor")
			}
			switch mode {
			case "missing-initial":
				q.initial = nil
			case "too-many-rows":
				q.fixed.bounds.records = 0
			case "missing-checkpoint":
				q.recovery.consumption = nil
			case "missing-member-weights":
				q.ancestors[0].members = []providerHistoryMemberAncestorFact{{}}
			}
			if got := q.ancestorsMatchObservedLocked(meter); got != (mode == "same") || meter.failed ||
				q.dispatch.memberReplaySelectedLocked() || q.recovery.memberReplaySeal != nil {
				t.Fatal("inconsistent private ancestry matched or acquired a production selection", mode, got)
			}
		})
	}
}
