package registry011

import (
	"reflect"
	"testing"
)

func TestProviderHistoryAppliedEmptyStreamPartsKeepIndependentIndexesAndGuards(t *testing.T) {
	for _, tc := range []struct{ entries, rounds, normal, envelope uint64 }{
		{0, 0, 512799, 201328494}, {1, 0, 1250222, 202065917}, {3, 2, 2728412, 203544107}, {17, 5, 13089834, 213905529},
	} {
		copies, headers := 2*(3+tc.rounds)*tc.entries, 4*tc.rounds
		adjust := func(n uint64) uint64 {
			return n - copies*32 - headers*24 + copies*uint64(reflect.TypeFor[providerHistorySemanticIndexEntry]().Size()) + headers*uint64(reflect.TypeFor[[]providerHistorySemanticIndexEntry]().Size())
		}
		want := providerHistoryAppliedStreamRootCost{normal: candidateWorkVector{10: adjust(tc.normal)}, envelope: candidateWorkVector{10: adjust(tc.envelope)}}
		got, ok := providerHistoryAppliedEmptyStreamRootUpper(tc.entries, tc.entries)
		if !ok || got != want {
			t.Fatal("empty stream omitted independent lifetime indexes or changed event rows", tc, got, want)
		}
		// Unqualified finite DATA isolates the zero-state branch with separate
		// other-subject indexes; it authenticates neither a Source nor a fold.
		stream := historySemanticIndexCostData(int(tc.entries))
		meter := providerHistoryContentUnitMeter(got.envelope)
		if root := providerHistorySemanticStreamRootObserved(stream, JournalConfig{MaxClaims: max(1, int(tc.entries)), MaxIdentities: max(1, int(tc.entries))}, meter); root == "" || meter.failed {
			t.Fatal("original empty-state branch and both indexes", tc.entries)
		}
		for c, actual := range meter.actual {
			if actual > got.normal[c] {
				t.Fatal("original empty-state work escaped normal", tc.entries, c)
			}
		}
	}
	for _, pair := range [][2]uint64{{1<<20 + 1, 0}, {0, ^uint64(0)}} {
		if got, ok := providerHistoryAppliedEmptyStreamRootUpper(pair[0], pair[1]); ok || got != (providerHistoryAppliedStreamRootCost{}) {
			t.Fatal("invalid empty-stream child exposed a partial quote")
		}
	}
	for _, window := range []bool{false, true} {
		l, history := providerHistorySemanticCommitmentLimits{Window: window}, uint64(0)
		if window {
			history = 8 << 20
		}
		base, ok := providerHistoryAppliedCommitmentUpper(l, history)
		if !ok {
			t.Fatal("original complete applied fold precursor")
		}
		for _, kinds := range []providerHistoryAppliedStateKinds{{}, {ConfirmedEmpty: true}, {WorkingEmpty: true}, {true, true}} {
			got, valid := providerHistoryAppliedCommitmentKindsUpper(l, history, kinds)
			if kinds.WorkingEmpty && !window {
				if valid || got != (providerHistoryAppliedCommitmentCost{}) {
					t.Fatal("missing working branch acquired an empty kind")
				}
				continue
			}
			want := base
			full, _ := providerHistoryAppliedStreamRootUpper(0, 0)
			for _, zero := range []bool{kinds.ConfirmedEmpty, kinds.WorkingEmpty} {
				if zero {
					want.normal[2] -= 2048
					want.envelope[2] -= 2048
					want.normal[10] = want.normal[10] - full.normal[10] + 512799
					want.envelope[10] = want.envelope[10] - full.envelope[10] + 201328494
				}
			}
			if !valid || got != want {
				t.Fatal("conditional fold parts changed nonempty support or untouched child", kinds, got, want)
			}
		}
	}
	parts, _ := providerHistoryAppliedEmptyStreamRootUpper(0, 0)
	short := providerHistoryContentUnitMeter(parts.normal)
	if root := providerHistorySemanticStreamRootObserved(replayStream{}, JournalConfig{MaxClaims: 1, MaxIdentities: 1}, short); root != "" || !short.failed || short.actual[10] == 0 {
		t.Fatal("empty actual-only DATA bypassed original index metadata preflight")
	}
	prefix := short.actual
	if root := providerHistorySemanticStreamRootObserved(replayStream{}, JournalConfig{MaxClaims: 1, MaxIdentities: 1}, short); root != "" || !short.failed || short.actual != prefix {
		t.Fatal("empty-branch observer renewed its prefix after refusal")
	}
}

func TestProviderHistoryAppliedSnapshotPartsKeepOwnedQueueAndEmptyBranches(t *testing.T) {
	for _, tc := range []struct {
		queue, history, queue64 uint64
		window, pending         bool
		kinds                   providerHistoryAppliedStateKinds
	}{
		{0, 0, 0, false, false, providerHistoryAppliedStateKinds{ConfirmedEmpty: true}},
		{0, 8388608, 1, true, false, providerHistoryAppliedStateKinds{true, true}},
		{1, 1048576, 1048662, true, false, providerHistoryAppliedStateKinds{}},
		{3, 65536, 131328, true, true, providerHistoryAppliedStateKinds{ConfirmedEmpty: true}},
		{4096, 8388608, 17125377, true, true, providerHistoryAppliedStateKinds{}},
	} {
		l := providerHistorySemanticSnapshotLimits{Confirmed: providerHistorySemanticCloneLimits{3, 5, 128, 16, 128}, Window: tc.window, Queue: tc.queue, Pending: tc.pending}
		confirmed := l.Confirmed
		if tc.kinds.ConfirmedEmpty {
			confirmed.Keys, confirmed.Services, confirmed.Grants = 0, 0, 0
		}
		clone, ok := providerHistorySemanticCloneUpper(confirmed)
		if !ok {
			t.Fatal("independently checked confirmed clone")
		}
		want := clone
		want[10] += uint64(reflect.TypeFor[providerHistorySemanticFold]().Size()) + 1
		if tc.window {
			l.Working = providerHistorySemanticCloneLimits{7, 11, 128, 16, 128}
			working := l.Working
			if tc.kinds.WorkingEmpty {
				working.Keys, working.Services, working.Grants = 0, 0, 0
			}
			part, _ := providerHistorySemanticCloneUpper(working)
			// Each text has two copies. The independent aggregate2B row keeps
			// array/item events and adjusts only their declared40-byte size.
			queue := tc.queue64 - 2*tc.queue*40 + 2*tc.queue*uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size())
			want[10] += part[10] + uint64(reflect.TypeFor[providerHistoryExpectedWindow]().Size()+reflect.TypeFor[replayRun]().Size()) + 1 + queue
			if tc.pending {
				want[10] += 524290
			}
		}
		if got, ok := providerHistoryAppliedSnapshotUpper(l, tc.history, tc.kinds); !ok || got != want {
			t.Fatal("snapshot parts differ from original copy events and selected byte aggregate", tc, got, want)
		}
	}
	for _, mode := range []string{"no-window", "pending", "queued", "confirmed-and-working"} {
		f, original, l := historySemanticSnapshotCostFixture(t, mode)
		history := uint64(0)
		if l.Window {
			history = uint64(f.selection.config.Replay.History.MaxBytes)
		}
		// The no-window fixture actually comes from the empty constructor;
		// other cases keep every full nonempty prospective branch funded.
		kinds := providerHistoryAppliedStateKinds{ConfirmedEmpty: mode == "no-window"}
		upper, ok := providerHistoryAppliedSnapshotUpper(l, history, kinds)
		before, wanted := original.actual, f.commitmentObserved(providerHistorySemanticUnitMeter())
		copy, copied := f.snapshotObserved(original)
		if !ok || !copied || copy == nil || copy.observer != original || f.observer != original || wanted == "" || original.failed || copy.commitmentObserved(providerHistorySemanticUnitMeter()) != wanted {
			t.Fatal("actual original snapshot lost state or observer", mode)
		}
		for c, actual := range original.actual {
			if actual < before[c] || actual-before[c] > upper[c] {
				t.Fatal("original snapshot exceeded prospective aggregate copy row", mode, c)
			}
		}
	}
	invalid := providerHistorySemanticSnapshotLimits{Confirmed: providerHistorySemanticCloneLimits{Keys: 129}}
	if got, ok := providerHistoryAppliedSnapshotUpper(invalid, 0, providerHistoryAppliedStateKinds{ConfirmedEmpty: true}); ok || got != (candidateWorkVector{}) {
		t.Fatal("unused invalid original child was hidden by an empty kind")
	}
	for _, history := range []uint64{0, 8<<20 + 1} {
		if got, ok := providerHistoryAppliedSnapshotUpper(providerHistorySemanticSnapshotLimits{Window: true}, history, providerHistoryAppliedStateKinds{}); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid selected history limit exposed a snapshot quote")
		}
	}
	for _, tc := range []struct {
		limits  providerHistorySemanticSnapshotLimits
		history uint64
		kinds   providerHistoryAppliedStateKinds
	}{
		{providerHistorySemanticSnapshotLimits{}, 1, providerHistoryAppliedStateKinds{}},
		{providerHistorySemanticSnapshotLimits{}, 0, providerHistoryAppliedStateKinds{WorkingEmpty: true}},
		{providerHistorySemanticSnapshotLimits{Window: true, Queue: 4097}, 8 << 20, providerHistoryAppliedStateKinds{}},
	} {
		if got, ok := providerHistoryAppliedSnapshotUpper(tc.limits, tc.history, tc.kinds); ok || got != (candidateWorkVector{}) {
			t.Fatal("invalid original snapshot shape exposed a partial row", tc)
		}
	}
}
