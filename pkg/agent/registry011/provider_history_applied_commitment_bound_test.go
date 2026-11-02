package registry011

import (
	"reflect"
	"strings"
	"testing"
)

func TestProviderHistoryAppliedCommitmentPartsMatchIndependentRows(t *testing.T) {
	for _, tc := range []struct {
		entries, queue, history, normal, envelope, bytes uint64
		window, pending                                  bool
	}{
		{0, 0, 0, 4412353261, 321283784318, 0, false, false},
		{0, 0, 8388608, 8824383797, 642634678239, 0, true, false},
		{1, 1, 1048576, 8827044991, 642704311397, 524288, true, false},
		{3, 3, 65536, 8829883716, 642841094050, 65536, true, true},
		{17, 4096, 8388608, 9431153500, 917558612486, 8388608, true, true},
	} {
		l := providerHistorySemanticCommitmentLimits{ConfirmedClaims: tc.entries, ConfirmedCreated: tc.entries, Queue: tc.queue, Window: tc.window, Pending: tc.pending}
		streams := uint64(1)
		if tc.window {
			streams = 2
			l.WorkingClaims, l.WorkingCreated = tc.entries, tc.entries
		}
		rounds := uint64(0)
		for n := uint64(1); n < tc.entries; n *= 2 {
			rounds++
		}
		// Only declared copy arguments vary from the independent64-bit rows.
		// Remove each full ordinary stream state before adding its portable
		// components; positive remainders avoid unsigned negative deltas.
		state := historyAppliedSerializerLogicalUnits(providerHistoryStateSerializationLimits{Keys: 128, Services: 16, Record: true}, 4295314140) +
			historyAppliedSerializerLogicalUnits(providerHistoryStateSerializationLimits{Grants: 128}, 115676421)
		wideState := historyAppliedSerializerLogicalUnits(providerHistoryStateSerializationLimits{Keys: 128, Services: 16, Record: true}, 317600220444) +
			historyAppliedSerializerLogicalUnits(providerHistoryStateSerializationLimits{Grants: 128}, 3415049256)
		copies, headers := streams*2*(3+rounds)*tc.entries, streams*4*rounds
		normal := streams*state + (tc.normal - streams*4410990561) - copies*32 - headers*24 +
			copies*uint64(reflect.TypeFor[providerHistorySemanticIndexEntry]().Size()) + headers*uint64(reflect.TypeFor[[]providerHistorySemanticIndexEntry]().Size())
		envelope := streams*wideState + (tc.envelope - streams*321015269700) - copies*32 - headers*24 +
			copies*uint64(reflect.TypeFor[providerHistorySemanticIndexEntry]().Size()) + headers*uint64(reflect.TypeFor[[]providerHistorySemanticIndexEntry]().Size())
		want := providerHistoryAppliedCommitmentCost{normal: candidateWorkVector{2: streams * 2048, 10: normal}, envelope: candidateWorkVector{2: streams * 2048, 10: envelope}, queueBytes: tc.bytes}
		got, ok := providerHistoryAppliedCommitmentUpper(l, tc.history)
		if !ok || got != want {
			t.Fatal("one-fold parts differ from independent complete rows", tc, got, want, ok)
		}
	}
	for _, tc := range []struct {
		l       providerHistorySemanticCommitmentLimits
		history uint64
	}{
		{providerHistorySemanticCommitmentLimits{}, 1},
		{providerHistorySemanticCommitmentLimits{Pending: true}, 0},
		{providerHistorySemanticCommitmentLimits{WorkingClaims: 1}, 0},
		{providerHistorySemanticCommitmentLimits{WorkingCreated: 1}, 0},
		{providerHistorySemanticCommitmentLimits{Queue: 1}, 0},
		{providerHistorySemanticCommitmentLimits{Window: true}, 0},
		{providerHistorySemanticCommitmentLimits{Window: true}, 8<<20 + 1},
		{providerHistorySemanticCommitmentLimits{Window: true, Queue: 4097}, 8 << 20},
		{providerHistorySemanticCommitmentLimits{ConfirmedClaims: 1<<20 + 1}, 0},
		{providerHistorySemanticCommitmentLimits{ConfirmedCreated: ^uint64(0)}, 0},
		{providerHistorySemanticCommitmentLimits{Window: true, WorkingClaims: ^uint64(0)}, 8 << 20},
		{providerHistorySemanticCommitmentLimits{Window: true, WorkingCreated: 1<<20 + 1}, 8 << 20},
	} {
		if got, ok := providerHistoryAppliedCommitmentUpper(tc.l, tc.history); ok || got != (providerHistoryAppliedCommitmentCost{}) {
			t.Fatal("invalid one-fold DATA exposed a partial quote", tc, got, ok)
		}
	}
}

func TestProviderHistoryAppliedCommitmentPartsKeepFixedMetadataSchemas(t *testing.T) {
	hash, decimal := "0x"+strings.Repeat("a", 64), "18446744073709551615"
	header := map[string]any{"hash": hash, "parent_hash": hash, "number": decimal, "timestamp": decimal}
	const signedMaximum = int64(1<<63 - 1)
	for _, tc := range []struct {
		kind   string
		value  map[string]any
		bytes  int
		actual uint64
	}{
		{"semantic-queue", map[string]any{"count": 4096}, 91, 8431},
		{"semantic-expected-item", map[string]any{"previous": hash, "ordinal": 4095, "version": decimal, "entry": hash, "command": hash}, 370, 137337},
		{"semantic-window", map[string]any{"state": hash, "base_version": decimal, "queue": hash, "queue_count": 4096,
			"pending_version": decimal, "pending_command": hash, "pending": false, "comparing": false, "compared": 4096,
			"history_bytes": signedMaximum, "blocks": signedMaximum, "budget": []int64{signedMaximum, signedMaximum, signedMaximum, signedMaximum, signedMaximum, signedMaximum}}, 664, 441620},
		{"semantic-fold", map[string]any{"program": hash, "selected": header, "last": header, "confirmed": hash, "window": hash,
			"blocks": ^uint64(0), "items": ^uint64(0), "windows": ^uint64(0)}, 879, 773578},
	} {
		// These are synthetic escape-free metadata maxima, not valid selected
		// counters. uint64 windows is wider than the original signed int.
		meter := providerHistorySemanticUnitMeter()
		if root := providerHistoryCollectionValueDigestObserved(tc.kind, tc.value, meter); root == "" || meter.failed ||
			meter.actual != (candidateWorkVector{10: tc.actual}) {
			t.Fatal("original metadata invocation differs from independent row", tc.kind, meter.actual)
		}
		if raw := providerReplayCanonical(tc.value, candidateLedgerMetadataLimit, nil); len(raw) != tc.bytes {
			t.Fatal("original encoder changed fixed schema extent", tc.kind, len(raw))
		}
	}
}

func TestProviderHistoryAppliedCommitmentPartsPreserveOwnedQueueAggregate(t *testing.T) {
	for _, mode := range []string{"no-window", "pending", "queued", "confirmed-and-working"} {
		f, original, snapshotLimits := historySemanticSnapshotCostFixture(t, mode)
		l := providerHistorySemanticCommitmentLimits{ConfirmedClaims: snapshotLimits.Confirmed.Claims, ConfirmedCreated: snapshotLimits.Confirmed.Created,
			Window: snapshotLimits.Window, Pending: snapshotLimits.Pending, Queue: snapshotLimits.Queue,
			WorkingClaims: snapshotLimits.Working.Claims, WorkingCreated: snapshotLimits.Working.Created}
		history := uint64(0)
		if f.window != nil {
			history = uint64(f.selection.config.Replay.History.MaxBytes)
		}
		parts, ok := providerHistoryAppliedCommitmentUpper(l, history)
		before := f.commitmentObserved(providerHistorySemanticUnitMeter())
		copy, copied := f.snapshotObserved(original)
		if !ok || !copied || copy == nil || before == "" || f.observer != original || copy.observer != original {
			t.Fatal("original observed fold/snapshot precursor", mode)
		}
		start := original.actual
		for _, candidate := range []*providerHistorySemanticFold{f, copy} {
			meter := providerHistoryContentUnitMeter(parts.envelope)
			if got := candidate.commitmentObserved(meter); got != before || meter.failed || !candidate.valid() {
				t.Fatal("one-fold quote changed original root/ownership", mode)
			}
			for c, units := range meter.actual {
				if units > parts.normal[c] {
					t.Fatal("successful fold escaped entered-only part", mode, c, units)
				}
			}
			if w := candidate.window; w != nil {
				bytes := 0
				for _, item := range w.queue {
					bytes += len(item.entry) + len(item.command)
				}
				if !w.selected(w.run) || bytes > w.historyBytes || uint64(bytes) > parts.queueBytes || uint64(w.historyBytes) > history {
					t.Fatal("actual charged immutable queue lost its selected aggregate", mode)
				}
			}
		}
		if original.actual != start || f.commitmentObserved(providerHistorySemanticUnitMeter()) != before {
			t.Fatal("read-only inspection changed execution observer or fold", mode)
		}
	}
}

func TestProviderHistoryAppliedCommitmentPartsCoverOuterDeferredRefusal(t *testing.T) {
	f, original, _ := historySemanticSnapshotCostFixture(t, "pending")
	w := f.window
	// Finite unqualified in-memory DATA isolates the original late refusal.
	// The real owned pending precursor supplies run/observer bindings, but
	// these synthetic state/queue values are not a parsed/applied credential.
	w.run.state = replayState{version: 1, admitted: 1, record: replayRecord{ID: "unit", State: "active", Version: "1", Services: make([]replayService, 16)},
		grants: []replayGrant{{Operator: replayTestWallet, Scope: "update-services"}}}
	for i := range w.run.state.record.Services {
		w.run.state.record.Services[i] = replayService{Name: "unit", Type: "unit", URI: strings.Repeat("\x01", 2048)}
	}
	w.queue = []providerHistoryExpectedItem{{version: 1, entry: "unit-entry", command: w.pendingCommand}}
	w.historyBytes = len(w.queue[0].entry) + len(w.queue[0].command)
	l := providerHistorySemanticCommitmentLimits{Window: true, Pending: true, Queue: 1}
	parts, ok := providerHistoryAppliedCommitmentUpper(l, uint64(w.budget.limits.History.MaxBytes))
	if !ok || !w.selected(w.run) {
		t.Fatal("bounded outer refusal DATA precursor")
	}
	before, originalActual := providerHistoryStateUnitSnapshot(t, w.run.state), original.actual
	expected := providerHistoryContentUnitMeter(parts.envelope)
	if providerHistorySemanticStreamRootObserved(f.confirmed, f.selection.config, expected) == "" ||
		providerHistorySemanticStreamRootObserved(replayStream{state: w.run.state}, f.selection.config, expected) != "" || expected.failed {
		t.Fatal("original working state late size refusal precursor")
	}
	streamPrefix := expected.actual
	base, a := providerHistorySemanticDecimalObserved(w.baseVersion, expected)
	pending, b := providerHistorySemanticDecimalObserved(w.pendingVersion, expected)
	queue := providerHistoryCollectionValueDigestObserved("semantic-queue", map[string]any{"count": 1}, expected)
	version, c := providerHistorySemanticDecimalObserved(1, expected)
	entry := providerHistorySemanticBytesRootObserved(w.queue[0].entry, expected)
	command := providerHistorySemanticBytesRootObserved(w.queue[0].command, expected)
	if !a || !b || !c || base == "" || pending == "" || queue == "" || entry == "" || command == "" {
		t.Fatal("deferred decimal/queue/byte calls")
	}
	queue = providerHistoryCollectionValueDigestObserved("semantic-expected-item", map[string]any{"previous": queue, "ordinal": 0, "version": version, "entry": entry, "command": command}, expected)
	if queue == "" || providerHistorySemanticBytesRootObserved(w.pendingCommand, expected) == "" || expected.failed || expected.actual[10] <= streamPrefix[10] {
		t.Fatal("full outer deferred closure did not execute")
	}
	actual := providerHistoryContentUnitMeter(parts.envelope)
	if root := f.commitmentObserved(actual); root != "" || actual.failed || actual.actual != expected.actual || original.actual != originalActual ||
		!reflect.DeepEqual(before, w.run.state) || !f.valid() {
		t.Fatal("working refusal skipped outer siblings, changed DATA or returned a root", actual.actual, expected.actual)
	}
	for c, units := range actual.actual {
		if units > parts.envelope[c] {
			t.Fatal("outer deferred call escaped complete one-fold envelope", c)
		}
	}
	short := parts.envelope
	short[10] = expected.actual[10] - 1
	refused := providerHistoryContentUnitMeter(short)
	if root := f.commitmentObserved(refused); root != "" || !refused.failed || refused.actual[10] == 0 || original.actual != originalActual {
		t.Fatal("unchanged preflight quota did not retain its entered prefix")
	}
	prefix := refused.actual
	if root := f.commitmentObserved(refused); root != "" || !refused.failed || refused.actual != prefix || !reflect.DeepEqual(before, w.run.state) {
		t.Fatal("failed read observer renewed consumed work or changed DATA")
	}
}
