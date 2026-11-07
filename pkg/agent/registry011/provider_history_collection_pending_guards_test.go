package registry011

import "testing"

func historyCollectionPendingGuardFixture(t *testing.T) *providerHistoryCollectionCursor {
	t.Helper()
	c, in, methods := historyCollectionContentFixture(t, 4)
	for steps := 0; c.phase != "effects" && steps < 8; steps++ {
		c = historyCollectionContentStep(t, c, in, methods)
	}
	if c.phase != "effects" || len(c.pending) != 2 || len(c.effects) != 2 || c.pendingApplied != 0 || c.effectNext != 0 ||
		c.pending[0].transactionCount != 1 || c.pending[1].transactionCount != 1 || c.pendingRootObserved(providerReplayUnitMeter()) == "" {
		t.Fatal("ordinary two-block pending DATA precursor")
	}
	return c
}

// Corrupt only detached, private reconstructed DATA. No selected manifest,
// original observer, execution owner or stored response is replaced here.
func TestProviderHistoryCollectionPendingGuardsKeepThePredecessor(t *testing.T) {
	base := historyCollectionPendingGuardFixture(t)
	root := base.commitmentObserved(providerReplayUnitMeter())
	for _, tc := range []struct {
		name   string
		change func(*providerHistoryCollectionCursor)
	}{
		{"negative-applied", func(c *providerHistoryCollectionCursor) { c.pendingApplied = -1 }},
		{"negative-next-effect", func(c *providerHistoryCollectionCursor) { c.effectNext = -1 }},
		{"next-effect-outside-array", func(c *providerHistoryCollectionCursor) { c.effectNext = len(c.effects) + 1 }},
		{"missing-trace-slot", func(c *providerHistoryCollectionCursor) { c.traceResults = c.traceResults[:1] }},
		{"effects-without-pending-blocks", func(c *providerHistoryCollectionCursor) { c.pending = nil }},
		{"wrong-phase", func(c *providerHistoryCollectionCursor) { c.phase = "content" }},
		{"all-pending-marked-applied", func(c *providerHistoryCollectionCursor) { c.pendingApplied = len(c.pending) }},
		{"missing-effects", func(c *providerHistoryCollectionCursor) { c.effects, c.traceResults = nil, nil }},
		{"wrong-header", func(c *providerHistoryCollectionCursor) { c.pending[0].header.Number++ }},
		{"negative-transaction-count", func(c *providerHistoryCollectionCursor) { c.pending[0].transactionCount = -1 }},
		{"missing-transaction", func(c *providerHistoryCollectionCursor) { c.pending[0].transactions = nil }},
		{"missing-header-bytes", func(c *providerHistoryCollectionCursor) { c.pending[0].headerRLP = "" }},
		{"pre-fork-gas-bytes", func(c *providerHistoryCollectionCursor) { c.pending[0].gasTrace = "[]" }},
		{"applied-block-keeps-bytes", func(c *providerHistoryCollectionCursor) { c.pendingApplied = 1 }},
		{"applied-block-does-not-match-semantic-last", func(c *providerHistoryCollectionCursor) {
			c.pendingApplied = 1
			c.pending[0] = providerHistoryCollectionBlock{header: c.pending[0].header, transactionCount: 1}
			c.semantic.last = c.pending[1].header
		}},
		{"effect-outside-pending-range", func(c *providerHistoryCollectionCursor) { c.effects[0].number = c.high + 1 }},
		{"effect-outside-transaction-array", func(c *providerHistoryCollectionCursor) { c.effects[0].index = 1 }},
		{"received-trace-has-no-bytes", func(c *providerHistoryCollectionCursor) { c.effectNext = 1 }},
		{"unreceived-trace-has-bytes", func(c *providerHistoryCollectionCursor) { c.traceResults[0] = "ordinary unreceived DATA" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := base.copyObserved(base.semantic.observer)
			if c == nil {
				t.Fatal("owned private DATA copy")
			}
			tc.change(c)
			if !c.valid() {
				t.Fatal("test did not reach the pending-specific guard")
			}
			if got := c.pendingRootObserved(providerReplayUnitMeter()); got != "" || base.commitmentObserved(providerReplayUnitMeter()) != root {
				t.Fatal("incomplete pending DATA acquired a root or changed its predecessor")
			}
		})
	}
}

func TestProviderHistoryCollectionPendingRootRehashesRetainedBytes(t *testing.T) {
	base := historyCollectionPendingGuardFixture(t)
	before := base.pendingRootObserved(providerReplayUnitMeter())
	for _, tc := range []struct {
		name   string
		change func(*providerHistoryCollectionCursor)
	}{
		{"header", func(c *providerHistoryCollectionCursor) { c.pending[0].headerRLP += "x" }},
		{"transaction", func(c *providerHistoryCollectionCursor) { c.pending[0].transactions[0] += "x" }},
		{"receipt", func(c *providerHistoryCollectionCursor) { c.pending[0].receipts[0] += "x" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := base.copyObserved(base.semantic.observer)
			if c == nil {
				t.Fatal("owned byte-copy precursor")
			}
			tc.change(c)
			root := c.pendingRootObserved(providerReplayUnitMeter())
			if root == "" || root == before || base.pendingRootObserved(providerReplayUnitMeter()) != before {
				t.Fatal("retained byte change was omitted or shared with its predecessor")
			}
		})
	}
	// This checks DATA freshness only. The modified bytes are not decoded or
	// accepted as consensus, replay evidence or a dispatch authorization.
}
