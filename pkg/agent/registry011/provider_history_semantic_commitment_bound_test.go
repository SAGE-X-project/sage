package registry011

import (
	"encoding/base64"
	"math/bits"
	"reflect"
	"testing"
)

func TestProviderHistorySemanticCommitmentBoundsMatchIndependentRootRows(t *testing.T) {
	for _, tc := range []struct {
		entries, queue, units64 uint64
		window, pending         bool
	}{
		{0, 0, 5004211076544, false, false},
		{0, 0, 10008489271023, true, false},
		{0, 4096, 10287698321152, true, true},
		{1, 0, 5004211813967, false, false},
		{1, 0, 10008490745869, true, false},
		{1, 4096, 10287699795998, true, true},
		{1 << 20, 0, 5786414171152, false, false},
		{1 << 20, 0, 11572895460239, true, false},
		{1 << 20, 4096, 11852104510368, true, true},
	} {
		l := providerHistorySemanticCommitmentLimits{ConfirmedClaims: tc.entries, ConfirmedCreated: tc.entries, Queue: tc.queue, Window: tc.window, Pending: tc.pending}
		streams := uint64(1)
		if tc.window {
			streams++
			l.WorkingClaims, l.WorkingCreated = tc.entries, tc.entries
		}
		rounds := uint64(0)
		if tc.entries > 1 {
			rounds = uint64(bits.Len64(tc.entries - 1))
		}
		// Independently expanded literal root totals; only original declared
		// type copies adjust from map8/string16/any16/index32/header24 DATA.
		want := tc.units64
		for _, size := range []struct{ copies, ordinary, target uint64 }{
			{streams * 802, 8, uint64(reflect.TypeFor[map[string]any]().Size())},
			{streams * 2668, 16, uint64(reflect.TypeFor[string]().Size())},
			{streams * 3756, 16, uint64(reflect.TypeFor[any]().Size())},
			{streams * 2 * tc.entries * (3 + rounds), 32, uint64(reflect.TypeFor[providerHistorySemanticIndexEntry]().Size())},
			{streams * 4 * rounds, 24, uint64(reflect.TypeFor[[]providerHistorySemanticIndexEntry]().Size())},
		} {
			want = want - size.copies*size.ordinary + size.copies*size.target
		}
		upper, ok := providerHistorySemanticCommitmentUpper(l)
		if !ok || upper != (candidateWorkVector{2: 2048 * streams, 10: want}) {
			t.Fatal("semantic commitment differs from independent complete root rows", tc, upper, want, ok)
		}
	}
}

func TestProviderHistorySemanticCommitmentBoundsFundActualReadOnlyRoots(t *testing.T) {
	for _, mode := range []string{"no-window", "pending", "queued", "confirmed-and-working"} {
		f, original, counts := historySemanticSnapshotCostFixture(t, mode)
		l := providerHistorySemanticCommitmentLimits{ConfirmedClaims: counts.Confirmed.Claims, ConfirmedCreated: counts.Confirmed.Created, WorkingClaims: counts.Working.Claims, WorkingCreated: counts.Working.Created, Queue: counts.Queue, Window: counts.Window, Pending: counts.Pending}
		upper, ok := providerHistorySemanticCommitmentUpper(l)
		if !ok {
			t.Fatal("prospective original semantic root envelope")
		}
		before := original.actual
		want := f.commitmentObserved(providerHistorySemanticUnitMeter())
		meter := providerHistoryContentUnitMeter(upper)
		got := f.commitmentObserved(meter)
		if want == "" || got != want || meter.failed || f.observer != original || original.actual != before || !f.valid() {
			t.Fatal("read-only root changed source/observer or exceeded its allowance", mode, meter.actual)
		}
		for coordinate, actual := range meter.actual {
			if actual > upper[coordinate] {
				t.Fatal("semantic root exceeded its prospective coordinate", mode, coordinate)
			}
		}
	}
}

func TestProviderHistorySemanticCommitmentBoundsKeepIndependentExpiryDecodedQuota(t *testing.T) {
	// A real normal observed-parsed/applied public-sample create supplies
	// this owned state. Its root is DATA, not authenticated Registry history.
	key := replayTestKey(t, "a", false)
	expires := uint64(1<<53 - 1)
	key.Expires = &expires
	fixture := replayTestCommand(t, "create", replayTestWallet, 0, map[string]any{
		"controller": replayTestWallet, "keys": []replayKey{key}, "services": []replayService{},
		"salt": base64.RawURLEncoding.EncodeToString(make([]byte, 32)),
	})
	command := providerHistoryApplyCostCommand(t, fixture.raw)
	var state replayState
	if _, ok := state.applyObserved(replayTestRegistry, command, 1, providerHistoryStateUnitMeter()); !ok {
		t.Fatal("normal max-safe expiry state precursor")
	}
	f, _, _ := historySemanticObservedFixture(t, false)
	stream := replayStream{state: state}
	before := providerHistoryStateUnitSnapshot(t, state)
	upper, ok := providerHistorySemanticStreamRootUpper(0, 0)
	if !ok {
		t.Fatal("prospective stream root envelope")
	}
	normal := providerHistoryContentUnitMeter(upper)
	want := providerHistorySemanticStreamRootObserved(stream, f.selection.config, normal)
	if want == "" || normal.failed || normal.actual[2] != 16 || normal.actual[10] > upper[10] || normal.actual[7] != 0 {
		t.Fatal("normal root lost record-expiry codec bytes", normal.actual)
	}
	for _, coordinate := range []int{2, 10} {
		short := upper
		short[coordinate] = 15
		if coordinate == 10 {
			short[10] = 1024 // observed traversal prefix, not final actual-1
		}
		meter := providerHistoryContentUnitMeter(short)
		if root := providerHistorySemanticStreamRootObserved(stream, f.selection.config, meter); root != "" || !meter.failed || meter.actual[10] == 0 || !reflect.DeepEqual(stream.state, before) {
			t.Fatal("root quota shortage hid work or changed the owned state", coordinate, meter.actual)
		}
		actual := meter.actual
		if root := providerHistorySemanticStreamRootObserved(stream, f.selection.config, meter); root != "" || !meter.failed || meter.actual != actual || !reflect.DeepEqual(stream.state, before) {
			t.Fatal("same failed root observer revived", coordinate)
		}
	}
}

func TestProviderHistorySemanticCommitmentBoundsRefuseInvalidDomains(t *testing.T) {
	for _, l := range []providerHistorySemanticCommitmentLimits{
		{ConfirmedClaims: 1<<20 + 1}, {ConfirmedCreated: ^uint64(0)},
		{Window: true, WorkingClaims: 1<<20 + 1}, {Window: true, WorkingCreated: ^uint64(0)},
		{Queue: 1}, {Pending: true}, {WorkingClaims: 1}, {WorkingCreated: 1},
		{Window: true, Queue: 4097}, {Window: true, Queue: ^uint64(0)},
	} {
		if upper, ok := providerHistorySemanticCommitmentUpper(l); ok || upper != (candidateWorkVector{}) {
			t.Fatal("invalid original root domain returned a partial upper", l, upper, ok)
		}
	}
}
