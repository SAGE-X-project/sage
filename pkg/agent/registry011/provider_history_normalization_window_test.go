package registry011

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

// Finite offline DATA only. No RPC, EVM, host hook, account operation or Source
// qualification is supplied by these normalization/window scenarios.
func historyNormalizationTrace(t *testing.T, frame ReplayFrame) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(nodeTestFrame(frame))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func historyNormalizationLeaf(log bool) ReplayFrame {
	f := ReplayFrame{Type: "CALL", From: replayTestOperator, To: replayTestWallet, Value: "0x0"}
	if log {
		f.Logs = []ReplayFrameLog{{Address: replayTestWallet, Data: []byte{1}}}
	}
	return f
}

func TestProviderHistoryNormalizationStandaloneKeepsPerTraceLimits(t *testing.T) {
	limits := replayTestLimits()
	limits.MaxFrames, limits.MaxLogs = 1, 1
	raw := historyNormalizationTrace(t, historyNormalizationLeaf(true))
	before := bytes.Clone(raw)
	want, err := DecodeCallTracer(context.Background(), raw, limits)
	if err != nil {
		t.Fatal("ordinary public DATA precursor", err)
	}
	m := providerHistoryExecutionUnitMeter()
	var first candidateWorkVector
	for pass := uint64(1); pass <= 2; pass++ {
		got, err := decodeCallTracerObserved(context.Background(), raw, limits, m)
		if err != nil || !reflect.DeepEqual(got, want) || m.failed || m.actual[5] != pass || m.actual[6] != pass {
			t.Fatal("standalone API inherited another trace's semantic quota", err, m.actual)
		}
		if pass == 1 {
			first = m.actual
		} else {
			for coordinate, n := range first {
				if m.actual[coordinate] != 2*n {
					t.Fatal("standalone original observer was replaced", coordinate, m.actual, first)
				}
			}
		}
	}
	if !bytes.Equal(raw, before) {
		t.Fatal("normalization changed original raw bytes")
	}
}

func TestProviderHistoryNormalizationSharedBudgetCarriesFramesAcrossRoots(t *testing.T) {
	limits := replayTestLimits()
	limits.MaxFrames, limits.MaxLogs = 2, 1
	m := providerHistoryExecutionUnitMeter()
	b := replayBudget{limits: limits, observer: m}
	raw := historyNormalizationTrace(t, historyNormalizationLeaf(false))
	for count := 1; count <= 2; count++ {
		if _, err := decodeCallTracerBudgetObserved(context.Background(), raw, &b); err != nil || b.frames != count || b.traceLogs != 0 {
			t.Fatal("one budget was not retained across ordinary roots", count, err, b.frames, b.traceLogs)
		}
	}
	before := m.actual
	if _, err := decodeCallTracerBudgetObserved(context.Background(), raw, &b); err == nil || b.frames != 2 || b.traceLogs != 0 || m.failed {
		t.Fatal("a third root renewed frame capacity", err, b.frames, m.actual)
	}
	// Complete raw preflight still precedes the root-frame capacity guard.
	if m.actual[2]-before[2] != 2*uint64(len(raw)) || m.actual[5] != before[5] || m.actual[10] <= before[10] {
		t.Fatal("the failed root lost its entered raw prefix or invented a frame", m.actual, before)
	}
}

func TestProviderHistoryNormalizationSharedBudgetCarriesLogsAcrossRoots(t *testing.T) {
	limits := replayTestLimits()
	limits.MaxFrames, limits.MaxLogs = 3, 2
	m := providerHistoryExecutionUnitMeter()
	b := replayBudget{limits: limits, observer: m}
	raw := historyNormalizationTrace(t, historyNormalizationLeaf(true))
	for count := 1; count <= 2; count++ {
		if _, err := decodeCallTracerBudgetObserved(context.Background(), raw, &b); err != nil || b.frames != count || b.traceLogs != count {
			t.Fatal("one budget was not retained across ordinary logs", count, err, b.frames, b.traceLogs)
		}
	}
	if _, err := decodeCallTracerBudgetObserved(context.Background(), raw, &b); err == nil || b.frames != 3 || b.traceLogs != 2 || m.failed || m.actual[5] != 3 || m.actual[6] != 2 {
		t.Fatal("a third root renewed log capacity or erased its entered frame", err, b.frames, b.traceLogs, m.actual)
	}
}

func TestProviderHistoryNormalizationCountsEveryNodeOfSuccessfulWholeWalk(t *testing.T) {
	for _, mode := range []string{"committed", "reverted-root", "static-child"} {
		t.Run(mode, func(t *testing.T) {
			frame := providerHistoryExecutionUnitFrame() // Four frames, six logs.
			if mode == "reverted-root" {
				frame.Error = "ordinary unit refusal"
			}
			if mode == "static-child" {
				frame.Calls[0].Type = "STATICCALL"
			}
			raw := historyNormalizationTrace(t, frame)
			limits := replayTestLimits()
			limits.MaxFrames, limits.MaxLogs = 7, 8
			m := providerHistoryExecutionUnitMeter()
			normalization := replayBudget{limits: limits, frames: 3, traceLogs: 2, observer: m}
			got, err := decodeCallTracerBudgetObserved(context.Background(), raw, &normalization)
			if err != nil || normalization.frames != 7 || normalization.traceLogs != 8 || m.actual[5] != 4 || m.actual[6] != 6 {
				t.Fatal("normalizer omitted a reverted/static subtree", err, normalization.frames, normalization.traceLogs, m.actual)
			}
			// The unchanged legacy semantic walker independently visits the same
			// complete tree; it does not reuse the normalization budget or meter.
			semantic := replayBudget{limits: limits, frames: 3, traceLogs: 2}
			logs, err := replayFrame(context.Background(), &got, "", false, true, 0, &semantic)
			if err != nil || semantic.frames != normalization.frames || semantic.traceLogs != normalization.traceLogs || m.failed || mode == "reverted-root" && len(logs) != 0 {
				t.Fatal("successful semantic scope differs from normalization scope", err, semantic.frames, semantic.traceLogs)
			}
		})
	}
}

func TestProviderHistoryNormalizationInvalidSharedBudgetsDoNotObserve(t *testing.T) {
	raw := historyNormalizationTrace(t, historyNormalizationLeaf(false))
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, mode := range []string{"nil-context", "nil-budget", "invalid-limits", "negative-frames", "wide-frames", "negative-logs", "wide-logs", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			m := providerHistoryExecutionUnitMeter()
			b := &replayBudget{limits: replayTestLimits(), observer: m}
			ctx := context.Background()
			switch mode {
			case "nil-context":
				ctx = nil
			case "nil-budget":
				b = nil
			case "invalid-limits":
				b.limits.MaxFrames = 0
			case "negative-frames":
				b.frames = -1
			case "wide-frames":
				b.frames = b.limits.MaxFrames + 1
			case "negative-logs":
				b.traceLogs = -1
			case "wide-logs":
				b.traceLogs = b.limits.MaxLogs + 1
			case "cancelled":
				ctx = cancelled
			}
			if got, err := decodeCallTracerBudgetObserved(ctx, raw, b); err == nil || !reflect.DeepEqual(got, ReplayFrame{}) || m.actual != (candidateWorkVector{}) || m.failed {
				t.Fatal("invalid DATA budget entered parsing", err, got, m.actual)
			}
		})
	}
}

func TestProviderHistoryNormalizationFailedObserverRemainsSticky(t *testing.T) {
	m := providerHistoryContentUnitMeter(candidateWorkVector{})
	b := replayBudget{limits: replayTestLimits(), observer: m}
	raw := historyNormalizationTrace(t, historyNormalizationLeaf(false))
	if _, err := decodeCallTracerBudgetObserved(context.Background(), raw, &b); err == nil || !m.failed || b.frames != 0 || b.traceLogs != 0 {
		t.Fatal("empty original observer accepted JSON work", err, m.actual)
	}
	before := m.actual
	if _, err := decodeCallTracerBudgetObserved(context.Background(), raw, &b); err == nil || m.actual != before || b.frames != 0 || b.traceLogs != 0 {
		t.Fatal("the same failed observer resumed", err, m.actual, before)
	}
}

func TestProviderHistoryNormalizationCollectionCarriesWindowAcrossOriginalGroups(t *testing.T) {
	in := historyCollectionStreamingInput(t) // Two blocks; two then 38 roots.
	before := replayTestClone(t, in)
	for _, reader := range []string{"left", "right"} {
		t.Run(reader, func(t *testing.T) {
			limits := replayTestLimits()
			limits.MaxFrames, limits.MaxLogs = 80, 40
			c, _, methods := historyCollectionContentInputLimitsFixture(t, 2, in, SepoliaAmsterdamRootExecutionGasProfile, limits)
			m := c.semantic.observer
			c = historyCollectionContentStep(t, c, in, methods) // Open original window.
			c = historyCollectionContentStep(t, c, in, methods) // Retain both blocks.
			c = historyCollectionContentStep(t, c, in, methods) // First 32 results.
			if c.pendingApplied != 1 || c.semantic.window.budget.frames != 4 || c.semantic.window.budget.traceLogs != 2 || c.effectNext != 32 {
				t.Fatal("the waiting second block changed the first block's counters")
			}
			c = historyCollectionContentStep(t, c, in, methods) // Last eight release 38 roots.
			if c.phase != "state" || c.semantic.window.budget.frames != 80 || c.semantic.window.budget.traceLogs != 40 || m.actual[5] != 160 || m.actual[6] != 160 || m.failed {
				t.Fatal("window counters were reset, double charged or lost across groups", c.phase, m.actual)
			}
			c = historyCollectionContentStep(t, c, in, methods) // Absent record, no history.
			c = historyCollectionContentStep(t, c, in, methods) // Close exact same window.
			if c.phase != "done" || !c.semantic.atEnd() || c.semantic.windows != 1 || len(c.semantic.confirmed.claims) != 40 || c.semantic.observer != m {
				t.Fatal("original DATA completion or observer identity changed", c.phase)
			}
		})
	}
	if !reflect.DeepEqual(in, before) {
		t.Fatal("independent readers changed the original input")
	}
}

func TestProviderHistoryNormalizationCollectionFailureKeepsPriorCursor(t *testing.T) {
	for _, tc := range []struct {
		name          string
		frames, logs  int
		enteredFrames uint64
	}{
		{"frame-capacity", 79, 40, 75},
		{"log-capacity", 80, 39, 76},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := historyCollectionStreamingInput(t)
			original := replayTestClone(t, in)
			limits := replayTestLimits()
			limits.MaxFrames, limits.MaxLogs = tc.frames, tc.logs
			c, _, methods := historyCollectionContentInputLimitsFixture(t, 2, in, SepoliaAmsterdamRootExecutionGasProfile, limits)
			c = historyCollectionContentStep(t, c, in, methods)
			c = historyCollectionContentStep(t, c, in, methods)
			c = historyCollectionContentStep(t, c, in, methods)
			m, completed := c.semantic.observer, c.completed
			root := c.commitmentObserved(providerReplayUnitMeter())
			table := historyCollectionBoundaryPlan(t, c, methods)
			results := historyCollectionContentResults(t, c, in, table)
			before := m.actual
			next, err := c.advanceBoundaryObserved(context.Background(), table, results, methods, m)
			if err == nil || next != nil || m.failed || c.completed != completed || c.commitmentObserved(providerReplayUnitMeter()) != root ||
				c.semantic.blocks != 1 || c.semantic.window.budget.frames != 4 || c.semantic.window.budget.traceLogs != 2 || c.effectNext != 32 || c.pendingApplied != 1 {
				t.Fatal("failed normalization published or mutated the prior candidate", err, c.completed, m.actual)
			}
			if m.actual[5]-before[5] != tc.enteredFrames || m.actual[6]-before[6] != 37 || m.actual[2] <= before[2] || m.actual[10] <= before[10] {
				t.Fatal("entered prefix was refunded, renewed or silently applied", m.actual, before)
			}
			if !reflect.DeepEqual(in, original) {
				t.Fatal("refusal changed original block/trace bytes")
			}
		})
	}
}

func TestProviderHistoryNormalizationCollectionRenewsOnlyAfterClosedWindow(t *testing.T) {
	in := historyCollectionStreamingInput(t)
	limits := replayTestLimits()
	limits.MaxFrames, limits.MaxLogs = 76, 38
	c, _, methods := historyCollectionContentInputLimitsFixture(t, 1, in, SepoliaAmsterdamRootExecutionGasProfile, limits)
	m := c.semantic.observer
	opens := 0
	for steps := 0; c.phase != "done" && steps < 50; steps++ {
		phase, before := c.phase, m.actual
		if phase == "open" {
			if c.semantic.window != nil {
				t.Fatal("an unclosed semantic window was renewed")
			}
			opens++
		}
		c = historyCollectionContentStep(t, c, in, methods)
		if phase == "open" && (c.semantic.window.budget.frames != 0 || c.semantic.window.budget.traceLogs != 0) {
			t.Fatal("a new original window inherited closed-window semantic counters")
		}
		for coordinate, n := range before {
			if m.actual[coordinate] < n {
				t.Fatal("closing a window refunded the original observer", coordinate)
			}
		}
	}
	if c.phase != "done" || opens != 2 || c.semantic.windows != 2 || c.semantic.observer != m || len(c.semantic.confirmed.claims) != 40 || m.actual[5] != 160 || m.actual[6] != 160 || m.failed {
		t.Fatal("window-local capacity became a whole-program quota or renewed the meter", c.phase, opens, m.actual)
	}
}

func TestProviderHistoryNormalizationReadyBlockRequiresOriginalWindowObserver(t *testing.T) {
	var absent *providerHistoryCollectionCursor
	if absent.applyReadyBlocksObserved(context.Background(), providerHistoryExecutionUnitMeter()) {
		t.Fatal("nil cursor accepted")
	}
	c, in, methods := historyCollectionContentFixture(t, 1)
	if c.applyReadyBlocksObserved(context.Background(), c.semantic.observer) {
		t.Fatal("an unopened semantic window accepted work")
	}
	c = historyCollectionContentStep(t, c, in, methods)
	m, before := c.semantic.observer, c.semantic.observer.actual
	if c.applyReadyBlocksObserved(nil, m) || c.applyReadyBlocksObserved(context.Background(), nil) ||
		c.applyReadyBlocksObserved(context.Background(), providerHistoryExecutionUnitMeter()) || m.actual != before {
		t.Fatal("a foreign or missing original context entered replay", m.actual, before)
	}
}
