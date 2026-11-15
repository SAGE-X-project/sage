package registry011

import (
	"context"
	"testing"
)

func TestProviderHistoryCollectionTransitionsRejectIncompleteWorkingData(t *testing.T) {
	c, input, methods := historyCollectionContentFixture(t, 4)
	type phaseInput struct {
		cursor  *providerHistoryCollectionCursor
		results []providerHistoryMemberResult
	}
	phases := map[string]phaseInput{}
	for steps := 0; steps < 32; steps++ {
		table := historyCollectionBoundaryPlan(t, c, methods)
		if _, seen := phases[c.phase]; !seen {
			phases[c.phase] = phaseInput{c, historyCollectionContentResults(t, c, input, table)}
		}
		if c.phase == "close" {
			break
		}
		c = historyCollectionContentStep(t, c, input, methods)
	}
	for _, phase := range []string{"content", "effects", "state", "grants", "history", "close"} {
		if phases[phase].cursor == nil {
			t.Fatal("ordinary first-window phase precursor missing", phase)
		}
	}
	for _, tc := range []struct {
		name, phase string
		change      func(*providerHistoryCollectionCursor, []providerHistoryMemberResult) []providerHistoryMemberResult
	}{
		{"missing-window", "effects", func(c *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			c.semantic.window = nil
			return r
		}},
		{"effects-without-blocks", "effects", func(c *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			c.pending = nil
			return r
		}},
		{"effects-missing-trace-slot", "effects", func(c *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			c.traceResults = nil
			return r
		}},
		{"effects-missing-result", "effects", func(_ *providerHistoryCollectionCursor, _ []providerHistoryMemberResult) []providerHistoryMemberResult {
			return nil
		}},
		{"effects-empty-result", "effects", func(_ *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			r[0].result = ""
			return r
		}},
		{"effects-already-received", "effects", func(c *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			c.traceResults[0] = "ordinary DATA"
			return r
		}},
		{"state-missing-result", "state", func(_ *providerHistoryCollectionCursor, _ []providerHistoryMemberResult) []providerHistoryMemberResult {
			return nil
		}},
		{"state-pending-block", "state", func(c *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			c.pending = []providerHistoryCollectionBlock{{}}
			return r
		}},
		{"state-wrong-last-block", "state", func(c *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			c.semantic.last.Number++
			return r
		}},
		{"state-already-comparing", "state", func(c *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			c.semantic.window.comparing = true
			return r
		}},
		{"state-keeps-stop-record", "state", func(c *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			c.stopRecord = "ordinary DATA"
			return r
		}},
		{"grants-missing-record", "grants", func(c *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			c.stopRecord = ""
			return r
		}},
		{"grants-zero-version", "grants", func(c *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			c.stopVersion = 0
			return r
		}},
		{"grants-wrong-version", "grants", func(c *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			c.stopVersion++
			return r
		}},
		{"history-zero-version", "history", func(c *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			c.historyNext = 0
			return r
		}},
		{"history-missing-result", "history", func(_ *providerHistoryCollectionCursor, _ []providerHistoryMemberResult) []providerHistoryMemberResult {
			return nil
		}},
		{"content-missing-results", "content", func(_ *providerHistoryCollectionCursor, _ []providerHistoryMemberResult) []providerHistoryMemberResult {
			return nil
		}},
		{"content-extra-result", "content", func(_ *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			return append(r, providerHistoryMemberResult{result: `"ordinary extra DATA"`})
		}},
		{"close-is-not-a-content-step", "close", func(_ *providerHistoryCollectionCursor, r []providerHistoryMemberResult) []providerHistoryMemberResult {
			return r
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base := phases[tc.phase]
			root := base.cursor.commitmentObserved(providerReplayUnitMeter())
			working := base.cursor.copyObserved(base.cursor.semantic.observer)
			if working == nil || root == "" {
				t.Fatal("owned working DATA precursor")
			}
			results := append([]providerHistoryMemberResult(nil), base.results...)
			results = tc.change(working, results)
			if !working.valid() {
				t.Fatal("test did not reach the transition-specific guard")
			}
			if working.contentBoundaryObserved(context.Background(), results, working.semantic.observer) || base.cursor.commitmentObserved(providerReplayUnitMeter()) != root {
				t.Fatal("incomplete private working DATA advanced or changed its predecessor")
			}
			// A rejected working copy may contain partial private mutations.
			// It is discarded; no rollback of that copy or live owner is claimed.
		})
	}
	for _, mode := range []string{"missing-comparison", "base-after-stop", "stop-outside-selection", "queue-length"} {
		t.Run(mode, func(t *testing.T) {
			base := phases["history"].cursor
			root := base.commitmentObserved(providerReplayUnitMeter())
			working := base.copyObserved(base.semantic.observer)
			if working == nil {
				t.Fatal("owned compared-window precursor")
			}
			switch mode {
			case "missing-comparison":
				working.semantic.window.comparing = false
			case "base-after-stop":
				working.semantic.window.baseVersion = working.stopVersion + 1
			case "stop-outside-selection":
				working.stopVersion = working.selection.input.versions + 1
			case "queue-length":
				working.semantic.window.queue = nil
			}
			if working.selectHistoryObserved(working.semantic.observer) || base.commitmentObserved(providerReplayUnitMeter()) != root {
				t.Fatal("incomplete comparison selected a history range")
			}
		})
	}
	for _, mode := range []string{"pending-block", "pending-trace", "pending-record", "pending-history", "wrong-last-block"} {
		t.Run(mode, func(t *testing.T) {
			base := phases["close"].cursor
			root := base.commitmentObserved(providerReplayUnitMeter())
			working := base.copyObserved(base.semantic.observer)
			if working == nil {
				t.Fatal("owned completed-window precursor")
			}
			switch mode {
			case "pending-block":
				working.pending = []providerHistoryCollectionBlock{{}}
			case "pending-trace":
				working.traceResults = []string{"ordinary DATA"}
			case "pending-record":
				working.stopRecord = "ordinary DATA"
			case "pending-history":
				working.historyNext = 1
			case "wrong-last-block":
				working.semantic.last.Number++
			}
			if working.closeContentObserved(context.Background(), working.semantic.observer) || base.commitmentObserved(providerReplayUnitMeter()) != root {
				t.Fatal("incomplete working window closed or changed its predecessor")
			}
		})
	}
}
