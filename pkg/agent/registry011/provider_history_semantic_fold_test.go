package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func historySemanticFixture(t *testing.T, maximumItems uint64) (*providerHistorySemanticFold, ReplayInput) {
	t.Helper()
	in := replayTestInput(t)
	f, err := newProviderHistorySemanticFold("0x"+strings.Repeat("a", 64), journalTestConfig(t, in), in.Anchor, uint64(len(in.Blocks)), maximumItems)
	if err != nil {
		t.Fatal("owned handwritten replay fixture", err)
	}
	return f, in
}

func historySemanticApply(t *testing.T, f *providerHistorySemanticFold, in ReplayInput, first, end int) {
	t.Helper()
	if err := f.beginWindowCandidate(context.Background()); err != nil {
		t.Fatal("begin explicit semantic DATA window", err)
	}
	for i := first; i <= end; i++ {
		if err := f.applyBlockCandidate(context.Background(), in.Blocks[i], journalTestHeader(t, in.Blocks[i])); err != nil {
			t.Fatal("derive from actual canonical block/receipt/trace", i, err)
		}
	}
}

func historySemanticCompare(t *testing.T, f *providerHistorySemanticFold, in ReplayInput, first, end int) {
	t.Helper()
	var record, grants []byte
	if end > 0 {
		step := replayTestSteps(t)[end-1]
		record, grants = step.Record, step.Grants
	}
	if err := f.compareStateCandidate(context.Background(), end > 0, record, grants); err != nil {
		t.Fatal("separate stop record/grants", err)
	}
	for i := max(1, first); i <= end; i++ {
		h := in.History[i-1]
		if err := f.compareHistoryItemCandidate(context.Background(), h.Version, h.Entry, h.Command); err != nil {
			t.Fatal("separate exact history original", i, err)
		}
	}
	if err := f.finishWindowCandidate(context.Background()); err != nil {
		t.Fatal("confirm only after every original comparison", err)
	}
}

func TestProviderHistorySemanticFoldDerivesAllNormalTransitionsBeforeHistoryReads(t *testing.T) {
	left, in := historySemanticFixture(t, 11)
	right, _ := historySemanticFixture(t, 11)
	before := replayTestClone(t, in)
	steps := replayTestSteps(t)
	for _, f := range []*providerHistorySemanticFold{left, right} {
		for i := range in.Blocks {
			historySemanticApply(t, f, in, i, i)
			w := f.window
			if w.run.history != nil || len(w.run.steps) != 0 || f.confirmed.state.version != uint64(max(0, i-1)) {
				t.Fatal("derivation used external history or prematurely committed state", i)
			}
			if i > 0 {
				expected := steps[i-1].History
				if len(w.queue) != 1 || w.queue[0].version != expected.Version ||
					!bytes.Equal([]byte(w.queue[0].entry), expected.Entry) || !bytes.Equal([]byte(w.queue[0].command), expected.Command) {
					t.Fatal("expected history differs from independent handwritten snapshots", i)
				}
			}
			historySemanticCompare(t, f, in, i, i)
			if len(f.confirmed.claims) != 1 || (i > 0 && !f.confirmed.created[replayTestAgent]) {
				t.Fatal("shared claim/create indexes were reset at a window boundary", i)
			}
		}
		if !f.atEnd() || f.failed || f.items != 11 || f.windows != len(in.Blocks) || f.confirmed.state.record.State != "deactivated" {
			t.Fatal("complete versioned terminal semantic DATA was not retained")
		}
	}
	if !reflect.DeepEqual(before, in) || left == right || left.confirmed.state.version != right.confirmed.state.version {
		t.Fatal("independent fold or input ownership")
	}
	left.confirmed.state.record.Keys[0].State = "changed-unit-copy"
	if right.confirmed.state.record.Keys[0].State != "revoked" {
		t.Fatal("two independent semantic states share mutable keys")
	}
	missing := replayTestClone(t, in)
	missing.History = nil
	if result, err := ReplayCommands(context.Background(), missing, replayTestLimits()); err == nil || result != nil {
		t.Fatal("legacy strict API switched to derivation on nil history")
	}
}

func TestProviderHistorySemanticFoldKeepsOtherSubjectClaimAndCreatedIndexes(t *testing.T) {
	_, in := historySemanticFixture(t, 11)
	c := journalTestConfig(t, in)
	c.Agent, c.MaxClaims, c.MaxIdentities = "other.selected.subject", 1, 1
	f, err := newProviderHistorySemanticFold("0x"+strings.Repeat("a", 64), c, in.Anchor, uint64(len(in.Blocks)), 1)
	if err != nil {
		t.Fatal(err)
	}
	for i := range in.Blocks {
		historySemanticApply(t, f, in, i, i)
		if len(f.window.queue) != 0 || f.window.run.state.version != 0 ||
			f.compareStateCandidate(context.Background(), false, nil, nil) != nil || f.finishWindowCandidate(context.Background()) != nil {
			t.Fatal("uncreated selected subject versus shared Registry effects", i)
		}
	}
	if !f.atEnd() || f.items != 0 || len(f.confirmed.claims) != 1 || !f.confirmed.created[replayTestAgent] {
		t.Fatal("other subject effects disappeared from global indexes")
	}
	for _, claim := range f.confirmed.claims {
		if !claim.consumed {
			t.Fatal("other subject's original claim was not consumed")
		}
	}
}

func TestProviderHistorySemanticZeroHistoryKeepsStrictEmptyAndOtherSubjectBoundaries(t *testing.T) {
	for _, mode := range []string{"deployment-only", "uncreated-selected-subject", "first-selected-version"} {
		t.Run(mode, func(t *testing.T) {
			in := replayTestInput(t)
			c, selected := journalTestConfig(t, in), in.Anchor
			if mode == "deployment-only" {
				selected = c.Deployment
			}
			if mode == "uncreated-selected-subject" {
				c.Agent, c.MaxClaims, c.MaxIdentities = "other.selected.subject", 1, 1
			}
			f, err := newProviderHistorySemanticFold("0x"+strings.Repeat("a", 64), c, selected, uint64(len(in.Blocks)), 0)
			if err != nil {
				t.Fatal("explicit zero whole subject-history allowance", err)
			}
			historySemanticApply(t, f, in, 0, 0)
			if len(f.window.queue) != 0 || f.compareStateCandidate(context.Background(), false, nil, nil) != nil || f.finishWindowCandidate(context.Background()) != nil {
				t.Fatal("deployment-only no-subject state failed strict comparison")
			}
			if mode == "deployment-only" {
				if !f.atEnd() || f.items != 0 || f.confirmed.state.version != 0 {
					t.Fatal("complete one-block empty DATA required invented history")
				}
				return
			}
			if mode == "first-selected-version" {
				before := providerHistoryCloneSemanticState(f.confirmed)
				if f.beginWindowCandidate(context.Background()) != nil {
					t.Fatal("valid next-window precursor")
				}
				if f.applyBlockCandidate(context.Background(), in.Blocks[1], journalTestHeader(t, in.Blocks[1])) == nil || !f.failed || f.items != 0 || f.blocks != 1 ||
					!reflect.DeepEqual(before, f.confirmed) || f.finishWindowCandidate(context.Background()) == nil || f.atEnd() {
					t.Fatal("first selected version bypassed zero allowance or changed the confirmed prefix")
				}
				return
			}
			for i := 1; i < len(in.Blocks); i++ {
				historySemanticApply(t, f, in, i, i)
				if len(f.window.queue) != 0 || f.compareStateCandidate(context.Background(), false, nil, nil) != nil || f.finishWindowCandidate(context.Background()) != nil {
					t.Fatal("other-subject Registry effects required selected history", i)
				}
			}
			if !f.atEnd() || f.failed || f.items != 0 || len(f.confirmed.claims) != 1 || !f.confirmed.created[replayTestAgent] {
				t.Fatal("zero subject history discarded global Registry indexes")
			}
			for _, claim := range f.confirmed.claims {
				if !claim.consumed {
					t.Fatal("other-subject shared claim lost its consumed state")
				}
			}
		})
	}
}

func TestProviderHistorySemanticFoldRejectsChangedMissingAndOutOfOrderStopEvidence(t *testing.T) {
	for _, mode := range []string{"missing-state", "record", "grants", "history-before-state", "history-missing", "entry", "command", "version", "duplicate", "block-after-state", "repeat-state", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			f, in := historySemanticFixture(t, 11)
			historySemanticApply(t, f, in, 0, 1)
			step := replayTestSteps(t)[0]
			ctx := context.Background()
			var err error
			switch mode {
			case "missing-state":
				err = f.compareStateCandidate(ctx, false, nil, nil)
			case "record":
				err = f.compareStateCandidate(ctx, true, append(bytes.Clone(step.Record), ' '), step.Grants)
			case "grants":
				err = f.compareStateCandidate(ctx, true, step.Record, []byte(`[{}]`))
			case "history-before-state":
				err = f.compareHistoryItemCandidate(ctx, 1, step.History.Entry, step.History.Command)
			case "cancelled":
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				err = f.compareStateCandidate(cancelled, true, step.Record, step.Grants)
			default:
				if f.compareStateCandidate(ctx, true, step.Record, step.Grants) != nil {
					t.Fatal("normal separately read stop precursor")
				}
				switch mode {
				case "history-missing":
					err = f.finishWindowCandidate(ctx)
				case "entry":
					err = f.compareHistoryItemCandidate(ctx, 1, append(bytes.Clone(step.History.Entry), ' '), step.History.Command)
				case "command":
					err = f.compareHistoryItemCandidate(ctx, 1, step.History.Entry, append(bytes.Clone(step.History.Command), ' '))
				case "version":
					err = f.compareHistoryItemCandidate(ctx, 2, step.History.Entry, step.History.Command)
				case "duplicate":
					if f.compareHistoryItemCandidate(ctx, 1, step.History.Entry, step.History.Command) != nil {
						t.Fatal("normal first original")
					}
					err = f.compareHistoryItemCandidate(ctx, 1, step.History.Entry, step.History.Command)
				case "block-after-state":
					err = f.applyBlockCandidate(ctx, in.Blocks[2], journalTestHeader(t, in.Blocks[2]))
				case "repeat-state":
					err = f.compareStateCandidate(ctx, true, step.Record, step.Grants)
				}
			}
			if err == nil || !f.failed || f.confirmed.state.version != 0 || len(f.confirmed.claims) != 0 || f.atEnd() || f.beginWindowCandidate(ctx) == nil {
				t.Fatal("failed semantic window changed its confirmed prefix or became reusable", mode, err)
			}
		})
	}
}

func TestProviderHistorySemanticFoldKeepsDeactivatedStatePresent(t *testing.T) {
	f, in := historySemanticFixture(t, 11)
	historySemanticApply(t, f, in, 0, len(in.Blocks)-1)
	if f.window.run.state.version != 11 || f.window.run.state.record.State != "deactivated" ||
		f.compareStateCandidate(context.Background(), false, nil, nil) == nil || !f.failed || f.confirmed.state.version != 0 {
		t.Fatal("deactivated was treated as absence, deletion or a version reset")
	}
}

func TestProviderHistorySemanticFoldRejectsCopiesJSONAndForeignDerivations(t *testing.T) {
	f, in := historySemanticFixture(t, 11)
	copyFold := *f
	if copyFold.beginWindowCandidate(context.Background()) == nil || f.failed || f.window != nil {
		t.Fatal("copied DATA fold changed the original")
	}
	historySemanticApply(t, f, in, 0, 0)
	w := f.window
	copyWindow := *w
	if copyWindow.selected(w.run) || (*providerHistoryExpectedWindow)(nil).selected(w.run) || w.selected(&replayRun{}) ||
		json.Unmarshal([]byte(`{}`), &providerHistorySemanticFold{}) == nil || json.Unmarshal([]byte(`{}`), &providerHistoryExpectedWindow{}) == nil {
		t.Fatal("detached/JSON DATA selected an original derivation")
	}
	historySemanticCompare(t, f, in, 0, 0)
	if w.selected(w.run) || f.failed {
		t.Fatal("closed expected window remains selectable")
	}
}

// Every window derives several commands before any separately read history.
// The expected bytes come from the original handwritten snapshots, not apply.
func TestProviderHistorySemanticFoldConfirmsMultiItemWindowsOnlyAfterAllComparisons(t *testing.T) {
	f, in := historySemanticFixture(t, 11)
	steps := replayTestSteps(t)
	windows := [][2]int{{0, 3}, {4, 7}, {8, 11}}
	for _, interval := range windows {
		first, end := interval[0], interval[1]
		beforeVersion := f.confirmed.state.version
		historySemanticApply(t, f, in, first, end)
		w := f.window
		if f.confirmed.state.version != beforeVersion || len(w.queue) != end-max(1, first)+1 || f.items != uint64(end) {
			t.Fatal("multi-item derivation committed early or renewed the lifetime count")
		}
		if f.compareStateCandidate(context.Background(), true, steps[end-1].Record, steps[end-1].Grants) != nil {
			t.Fatal("separately read multi-item window stop")
		}
		for i := max(1, first); i <= end; i++ {
			want := steps[i-1].History
			expected := w.queue[i-max(1, first)]
			if expected.version != want.Version || !bytes.Equal([]byte(expected.entry), want.Entry) || !bytes.Equal([]byte(expected.command), want.Command) {
				t.Fatal("ordered multi-item derivation differs from handwritten history", i)
			}
			if f.compareHistoryItemCandidate(context.Background(), want.Version, want.Entry, want.Command) != nil || f.confirmed.state.version != beforeVersion {
				t.Fatal("partial history comparison committed state or refused ordinary history", i)
			}
		}
		if f.finishWindowCandidate(context.Background()) != nil || f.confirmed.state.version != uint64(end) || len(f.confirmed.claims) != 1 || !f.confirmed.created[replayTestAgent] {
			t.Fatal("complete multi-item comparison lost confirmed state or shared indexes")
		}
	}
	if !f.atEnd() || f.windows != len(windows) || f.items != 11 || f.confirmed.state.record.State != "deactivated" {
		t.Fatal("three multi-item windows did not close the complete versioned DATA interval")
	}
}

func TestProviderHistorySemanticFoldKeepsConfirmedPrefixWhenLaterHistoryRowFails(t *testing.T) {
	for _, mode := range []string{"missing-later-row", "changed-later-row", "out-of-order-later-row", "cancel-after-prefix"} {
		t.Run(mode, func(t *testing.T) {
			f, in := historySemanticFixture(t, 11)
			historySemanticApply(t, f, in, 0, 3)
			historySemanticCompare(t, f, in, 0, 3)
			record, grants := f.confirmed.stateBytes()
			historySemanticApply(t, f, in, 4, 7)
			step := replayTestSteps(t)[6]
			ctx := context.Background()
			if f.compareStateCandidate(ctx, true, step.Record, step.Grants) != nil ||
				f.compareHistoryItemCandidate(ctx, 4, in.History[3].Entry, in.History[3].Command) != nil {
				t.Fatal("ordinary later window and first ordered history row")
			}
			var err error
			switch mode {
			case "missing-later-row":
				err = f.finishWindowCandidate(ctx)
			case "changed-later-row":
				err = f.compareHistoryItemCandidate(ctx, 5, append(bytes.Clone(in.History[4].Entry), ' '), in.History[4].Command)
			case "out-of-order-later-row":
				err = f.compareHistoryItemCandidate(ctx, 6, in.History[5].Entry, in.History[5].Command)
			case "cancel-after-prefix":
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				err = f.compareHistoryItemCandidate(cancelled, 5, in.History[4].Entry, in.History[4].Command)
			}
			afterRecord, afterGrants := f.confirmed.stateBytes()
			if err == nil || !f.failed || f.items != 7 || f.windows != 1 || f.confirmed.state.version != 3 ||
				!bytes.Equal(record, afterRecord) || !bytes.Equal(grants, afterGrants) || len(f.confirmed.claims) != 1 || !f.confirmed.created[replayTestAgent] ||
				f.window.compared != 1 || f.atEnd() || f.beginWindowCandidate(ctx) == nil || f.finishWindowCandidate(ctx) == nil {
				t.Fatal("later queue failure changed the earlier confirmed prefix, refunded derived items, or allowed retry", mode)
			}
			for _, claim := range f.confirmed.claims {
				if !claim.consumed {
					t.Fatal("later queue failure restored the original consumed commitment")
				}
			}
		})
	}
}
