package registry011

import (
	"fmt"
	"testing"
)

func historyCollectionInitialGuardFixture(t *testing.T) *providerHistoryCollectionCursor {
	t.Helper()
	in := historyCollectionInput(t)
	// All finite UNIT bounds are selected before the manifest and cursor.
	in.blocks, in.native, in.windowBlocks, in.maximumSelected = 3, 3, 2, sepoliaDeploymentBlock+2
	in.effects, in.versions, in.presentStops = 64, 64, 2
	header := in.journal.Deployment
	for i := 1; i < 3; i++ {
		header = FinalizedHeader{Hash: fmt.Sprintf("0x%064x", i+1), ParentHash: header.Hash, Number: header.Number + 1, Timestamp: header.Timestamp + 1}
	}
	c, err := newProviderHistoryCollectionCursorObserved(admissionTestDigest("a"), historyCollectionManifest(t, in), header, providerReplayUnitMeter())
	if err != nil || !c.initial() {
		t.Fatal("ordinary initial DATA precursor", err)
	}
	return c
}

func TestProviderHistoryCollectionInitialStateExcludesRetainedWork(t *testing.T) {
	base := historyCollectionInitialGuardFixture(t)
	root := base.commitmentObserved(providerReplayUnitMeter())
	for _, tc := range []struct {
		name   string
		change func(*providerHistoryCollectionCursor)
	}{
		{"present-stop", func(c *providerHistoryCollectionCursor) { c.presentStops = 1 }},
		{"pending-block", func(c *providerHistoryCollectionCursor) { c.pending = []providerHistoryCollectionBlock{{}} }},
		{"pending-count", func(c *providerHistoryCollectionCursor) { c.pendingApplied = 1 }},
		{"trace-bytes", func(c *providerHistoryCollectionCursor) { c.traceResults = []string{"ordinary DATA"} }},
		{"stop-record", func(c *providerHistoryCollectionCursor) { c.stopRecord = "ordinary DATA" }},
		{"semantic-block", func(c *providerHistoryCollectionCursor) { c.semantic.blocks = 1 }},
		{"semantic-item", func(c *providerHistoryCollectionCursor) { c.semantic.items = 1 }},
		{"semantic-window-count", func(c *providerHistoryCollectionCursor) { c.semantic.windows = 1 }},
		{"semantic-window", func(c *providerHistoryCollectionCursor) { c.semantic.window = &providerHistoryExpectedWindow{} }},
		{"confirmed-version", func(c *providerHistoryCollectionCursor) { c.semantic.confirmed.state.version = 1 }},
		{"confirmed-created", func(c *providerHistoryCollectionCursor) { c.semantic.confirmed.created = map[string]bool{"seen": true} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := base.copyObserved(base.semantic.observer)
			if c == nil {
				t.Fatal("owned initial DATA copy")
			}
			tc.change(c)
			if !c.valid() || c.initial() || base.commitmentObserved(providerReplayUnitMeter()) != root {
				t.Fatal("retained private DATA became an initial cursor or changed its predecessor")
			}
		})
	}
}

func TestProviderHistoryCollectionCommitmentRejectsBrokenRetainedContext(t *testing.T) {
	base := historyCollectionPendingGuardFixture(t)
	root := base.commitmentObserved(providerReplayUnitMeter())
	for _, tc := range []struct {
		name   string
		change func(*providerHistoryCollectionCursor)
	}{
		{"spine-number", func(c *providerHistoryCollectionCursor) { c.spine[1].Number++ }},
		{"spine-parent-link", func(c *providerHistoryCollectionCursor) { c.spine[1].Hash = admissionTestDigest("e") }},
		{"spine-time-order", func(c *providerHistoryCollectionCursor) { c.spine[1].Timestamp = c.spine[0].Timestamp }},
		{"deployment-context", func(c *providerHistoryCollectionCursor) {
			c.spine[len(c.spine)-1].ParentHash = admissionTestDigest("e")
		}},
		{"effect-order", func(c *providerHistoryCollectionCursor) { c.effects[1] = c.effects[0] }},
		{"stop-record-outside-grants", func(c *providerHistoryCollectionCursor) { c.stopRecord = "ordinary DATA" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := base.copyObserved(base.semantic.observer)
			if c == nil {
				t.Fatal("owned retained DATA copy")
			}
			tc.change(c)
			if !c.valid() || c.commitmentObserved(providerReplayUnitMeter()) != "" || base.commitmentObserved(providerReplayUnitMeter()) != root {
				t.Fatal("broken retained context acquired a commitment or changed its predecessor")
			}
		})
	}
}

func TestProviderHistoryCollectionPlanningRefusesInconsistentPrivateData(t *testing.T) {
	base := historyCollectionPendingGuardFixture(t)
	bounds, _ := providerFrontierBoundsFixture()
	root := base.commitmentObserved(providerReplayUnitMeter())
	for _, tc := range []struct {
		name   string
		change func(*providerHistoryCollectionCursor)
	}{
		{"ancestry-in-content-mode", func(c *providerHistoryCollectionCursor) { c.phase = "ancestry" }},
		{"content-in-ancestry-mode", func(c *providerHistoryCollectionCursor) { c.phase, c.mode = "content", "ancestry" }},
		{"content-outside-window", func(c *providerHistoryCollectionCursor) { c.phase, c.next = "content", c.high+1 }},
		{"state-in-ancestry-mode", func(c *providerHistoryCollectionCursor) { c.phase, c.mode = "state", "ancestry" }},
		{"state-without-stop-header", func(c *providerHistoryCollectionCursor) { c.phase, c.spine = "state", c.spine[:1] }},
		{"effect-outside-window", func(c *providerHistoryCollectionCursor) { c.effects[0].number = c.high + 1 }},
		{"effect-transaction-index", func(c *providerHistoryCollectionCursor) {
			c.effects[0].index = uint64(c.selection.input.journal.Replay.MaxTransactions)
		}},
		{"effect-order", func(c *providerHistoryCollectionCursor) { c.effects[1] = c.effects[0] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := base.copyObserved(base.semantic.observer)
			if c == nil {
				t.Fatal("owned planning DATA copy")
			}
			tc.change(c)
			if !c.valid() {
				t.Fatal("test did not reach the planning-specific guard")
			}
			table, err := c.nextTableObserved(c.completed+1, *bounds.methods, c.selection.input.capture, providerReplayUnitMeter())
			if err == nil || table != nil || base.commitmentObserved(providerReplayUnitMeter()) != root {
				t.Fatal("incomplete DATA selected a request table or changed its predecessor", err)
			}
		})
	}
	if err := base.UnmarshalJSON([]byte(`{}`)); err != ErrConfiguration || base.commitmentObserved(providerReplayUnitMeter()) != root {
		t.Fatal("private cursor could be restored from caller JSON", err)
	}
}
