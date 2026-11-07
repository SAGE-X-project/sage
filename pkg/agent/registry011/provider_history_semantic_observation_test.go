package registry011

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"
)

// Only actual full-block semantic units receive these explicit existing DATA
// counts. Execution-only and production limits remain unchanged.
func providerHistorySemanticUnitMeter() *providerWorkMeter {
	return providerHistoryContentUnitMeter(candidateWorkVector{2: 1 << 30, 3: 4096, 4: 16384, 5: 65536, 6: 65536, 7: 4096, 10: 1 << 60})
}

func historySemanticObservedFixture(t *testing.T, other bool) (*providerHistorySemanticFold, ReplayInput, *providerWorkMeter) {
	t.Helper()
	in := replayTestInput(t)
	c := journalTestConfig(t, in)
	if other {
		c.Agent = "other.selected.subject"
	}
	m := providerHistorySemanticUnitMeter()
	f, err := newProviderHistorySemanticFoldObserved("0x"+strings.Repeat("a", 64), c, in.Anchor, uint64(len(in.Blocks)), 11, m)
	if err != nil {
		t.Fatal("explicit original observer selection", err)
	}
	return f, in, m
}

func TestProviderHistorySemanticObserverSpansEveryNormalWindow(t *testing.T) {
	legacy, in := historySemanticFixture(t, 11)
	observed, _, m := historySemanticObservedFixture(t, false)
	before := replayTestClone(t, in)
	for _, interval := range [][2]int{{0, 3}, {4, 7}, {8, 11}} {
		actualBefore := m.actual
		for _, f := range []*providerHistorySemanticFold{legacy, observed} {
			historySemanticApply(t, f, in, interval[0], interval[1])
			if f == observed && (f.window.budget.observer != m || len(f.window.queue) != cap(f.window.queue)) {
				t.Fatal("new window replaced observer or used implicit queue growth")
			}
			historySemanticCompare(t, f, in, interval[0], interval[1])
		}
		if !reflect.DeepEqual(legacy.confirmed, observed.confirmed) || observed.items != legacy.items || observed.blocks != legacy.blocks || m.actual[10] <= actualBefore[10] || m.failed {
			t.Fatal("observed window changed confirmed state or reset cumulative work", interval, m.actual)
		}
	}
	if !observed.atEnd() || !legacy.atEnd() || m.actual[7] != 3 || m.actual[3] != uint64(len(in.Blocks)) || !reflect.DeepEqual(before, in) {
		t.Fatal("complete handwritten interval or actual proof/block count", m.actual)
	}
}

func TestProviderHistorySemanticObserverKeepsOtherSubjectTombstonesWithoutHistory(t *testing.T) {
	f, in, m := historySemanticObservedFixture(t, true)
	for i := range in.Blocks {
		historySemanticApply(t, f, in, i, i)
		if len(f.window.queue) != 0 || f.compareStateCandidate(context.Background(), false, nil, nil) != nil || f.finishWindowCandidate(context.Background()) != nil {
			t.Fatal("other-subject effects required invented selected history", i)
		}
	}
	if !f.atEnd() || f.items != 0 || m.actual[7] != 0 || !f.confirmed.created[replayTestAgent] || len(f.confirmed.claims) != 1 {
		t.Fatal("global lifetime indexes lost across observed windows", m.actual)
	}
	for _, claim := range f.confirmed.claims {
		if !claim.consumed {
			t.Fatal("global consumed tombstone forgotten")
		}
	}
}

func TestProviderHistoryExpectedQueueOwnsDerivedBytes(t *testing.T) {
	f, _, m := historySemanticObservedFixture(t, false)
	if f.beginWindowCandidate(context.Background()) != nil {
		t.Fatal("normal empty owned window")
	}
	w := f.window
	c := replayTestCreate(t)
	if !w.prepareSubject(w.run, c) {
		t.Fatal("valid command pending precursor")
	}
	pending := w.pendingCommand
	step, ok := w.run.state.applyObserved(replayTestRegistry, c, 1, m)
	if !ok || !w.retainExpected(w.run, step) || len(w.queue) != 1 || cap(w.queue) != 1 {
		t.Fatal("actual generated step queue precursor")
	}
	entry, command := w.queue[0].entry, w.queue[0].command
	step.History.Entry[0] ^= 1
	step.History.Command[0] ^= 1
	c.raw[0] ^= 1
	if w.queue[0].entry != entry || w.queue[0].command != command || !bytes.Equal([]byte(pending), []byte(command)) || w.pending || f.items != 1 {
		t.Fatal("queue retained mutable derivation buffers")
	}
	if w.retainExpected(w.run, step) || w.prepareSubject(w.run, c) {
		t.Fatal("retained item repeated or stale expected version admitted")
	}
}

func TestProviderHistorySemanticObserverQuarantinesFailedComparisons(t *testing.T) {
	for _, mode := range []string{"record", "history", "wrong-observer", "unfunded-next-window"} {
		t.Run(mode, func(t *testing.T) {
			f, in, m := historySemanticObservedFixture(t, false)
			historySemanticApply(t, f, in, 0, 0)
			historySemanticCompare(t, f, in, 0, 0)
			before := providerHistoryCloneSemanticState(f.confirmed)
			if mode == "unfunded-next-window" {
				// Preserve the consumed original actuals; do not install a new meter.
				m.upper = m.actual
				if f.beginWindowCandidate(context.Background()) == nil || !f.failed || !m.failed || !reflect.DeepEqual(f.confirmed, before) {
					t.Fatal("unfunded clone changed the confirmed prefix")
				}
				return
			}
			historySemanticApply(t, f, in, 1, 1)
			step := replayTestSteps(t)[0]
			var err error
			switch mode {
			case "wrong-observer":
				f.window.budget.observer = providerHistorySemanticUnitMeter()
				err = f.compareStateCandidate(context.Background(), true, step.Record, step.Grants)
			case "record":
				err = f.compareStateCandidate(context.Background(), true, append(bytes.Clone(step.Record), ' '), step.Grants)
			default:
				if f.compareStateCandidate(context.Background(), true, step.Record, step.Grants) != nil {
					t.Fatal("normal separate stop state precursor")
				}
				err = f.compareHistoryItemCandidate(context.Background(), 1, append(bytes.Clone(step.History.Entry), ' '), step.History.Command)
			}
			if err == nil || !f.failed || !reflect.DeepEqual(f.confirmed, before) || f.finishWindowCandidate(context.Background()) == nil || f.beginWindowCandidate(context.Background()) == nil {
				t.Fatal("failed observed window changed confirmed state or reopened", mode)
			}
		})
	}
}

func TestProviderHistorySemanticObserverRejectsMissingAndCopiedBinding(t *testing.T) {
	f, in, _ := historySemanticObservedFixture(t, false)
	c := journalTestConfig(t, in)
	if got, err := newProviderHistorySemanticFoldObserved("0x"+strings.Repeat("a", 64), c, in.Anchor, 12, 11, nil); err == nil || got != nil {
		t.Fatal("implicit nil observer selection")
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{})
	if got, err := newProviderHistorySemanticFoldObserved("0x"+strings.Repeat("a", 64), c, in.Anchor, 12, 11, short); err == nil || got != nil || !short.failed {
		t.Fatal("unfunded constructor")
	}
	invalid := c
	invalid.MaxClaims = 0
	if got, err := newProviderHistorySemanticFoldObserved("0x"+strings.Repeat("a", 64), invalid, in.Anchor, 12, 11, providerHistorySemanticUnitMeter()); err == nil || got != nil {
		t.Fatal("observer bypassed original configuration validation")
	}
	copyFold := *f
	if copyFold.beginWindowCandidate(context.Background()) == nil || f.failed {
		t.Fatal("copied fold changed original")
	}
	f.observer = providerHistorySemanticUnitMeter()
	if f.beginWindowCandidate(context.Background()) == nil || !f.failed {
		t.Fatal("replacement original observer admitted")
	}
}
