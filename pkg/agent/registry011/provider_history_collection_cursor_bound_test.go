package registry011

import (
	"reflect"
	"strconv"
	"testing"
)

func TestProviderHistoryCollectionCursorBoundsMatchIndependentRootAndCopyRows(t *testing.T) {
	for _, tc := range []struct {
		spine, effects, blocks, txs, root64, copy64 uint64
		window                                      bool
	}{
		{1, 0, 0, 1, 4496778168, 5, false},
		{8, 2, 2, 3, 7888061374, 546, true},
		{1 << 20, 163840, 10, 16384, 90998673814648, 9078280, true},
	} {
		l := providerHistoryCollectionCursorLimits{Spine: tc.spine, Effects: tc.effects, PendingBlocks: tc.blocks, Transactions: tc.txs, EvidenceBytes: 64 << 20, BodyBytes: 8 << 20}
		snapshotLimits := providerHistorySemanticSnapshotLimits{Confirmed: providerHistorySemanticCloneLimits{Keys: 128, Services: 16, Grants: 128}}
		if tc.window {
			// Distinct counts ensure a stream cannot borrow the other one's
			// lifetime index maxima. Child quotes are independently closed;
			// these literal rows cover the cursor work outside those children.
			l.Semantic = providerHistorySemanticCommitmentLimits{3, 5, 7, 11, 2, true, true}
			snapshotLimits = providerHistorySemanticSnapshotLimits{
				Confirmed: providerHistorySemanticCloneLimits{3, 5, 128, 16, 128},
				Working:   providerHistorySemanticCloneLimits{7, 11, 128, 16, 128}, Queue: 2, Window: true, Pending: true,
			}
		}
		semantic, semanticOK := providerHistorySemanticCommitmentUpper(l.Semantic)
		snapshot, snapshotOK := providerHistorySemanticSnapshotUpper(snapshotLimits)
		if !semanticOK || !snapshotOK {
			t.Fatal("independently bounded semantic component precursors")
		}
		wantRoot := semantic
		wantRoot[10] += tc.root64
		// Independent numeric-copy literal plus only declared block/string/
		// original machine-word copies; no backing-graph/RAM adjustment.
		copies := tc.copy64 - tc.blocks*152 - tc.effects*16 - 4*tc.blocks*tc.txs*8
		copies += tc.blocks*uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size()) + tc.effects*uint64(reflect.TypeFor[string]().Size()) + 4*tc.blocks*tc.txs*uint64(strconv.IntSize/8)
		wantCopy := wantRoot
		for coordinate, units := range snapshot {
			wantCopy[coordinate] += units
		}
		wantCopy[10] += copies
		got, ok := providerHistoryCollectionCursorUpper(l)
		if !ok || got.commitment != wantRoot || got.copy != wantCopy {
			t.Fatal("cursor differs from independent manifest/spine/effects/pending/copy rows", tc, got, wantRoot, wantCopy, ok)
		}
	}
}

func historyCollectionCursorCostLimits(t *testing.T, c *providerHistoryCollectionCursor) providerHistoryCollectionCursorLimits {
	t.Helper()
	in := c.selection.input
	effects, ok := providerUpperProduct(in.contentBlocks, uint64(in.journal.Replay.MaxTransactions))
	if !ok {
		t.Fatal("prospective selected local effect ceiling")
	}
	// All optional working/queue/pending work is funded BEFORE inspection.
	// These selected configuration maxima are not observed size reports.
	return providerHistoryCollectionCursorLimits{
		Spine: in.blocks, Effects: effects, PendingBlocks: in.contentBlocks,
		Transactions: uint64(in.journal.Replay.MaxTransactions), EvidenceBytes: uint64(in.journal.Replay.MaxEvidenceBytes), BodyBytes: uint64(in.capture.BodyBytes),
		Semantic: providerHistorySemanticCommitmentLimits{uint64(in.journal.MaxClaims), uint64(in.journal.MaxIdentities), uint64(in.journal.MaxClaims), uint64(in.journal.MaxIdentities), 4096, true, true},
	}
}

func TestProviderHistoryCollectionCursorBoundsFundEveryNormalReadOnlyPhase(t *testing.T) {
	// Existing synthetic canonical blocks/public samples only. This private
	// DATA cursor cannot pass the real Sepolia deployment manifest constructor.
	c, in, methods := historyCollectionContentFixture(t, 4)
	seen := map[string]bool{}
	finished := false
	for groups := 0; groups < 200; groups++ {
		upper, ok := providerHistoryCollectionCursorUpper(historyCollectionCursorCostLimits(t, c))
		if !ok {
			t.Fatal("prospective full cursor allowance")
		}
		original := c.semantic.observer
		before := original.actual
		want := c.commitmentObserved(providerHistorySemanticUnitMeter())
		meter := providerHistoryContentUnitMeter(upper.commitment)
		if root := c.commitmentObserved(meter); root == "" || root != want || meter.failed || original.actual != before || c.semantic.observer != original {
			t.Fatal("normal read-only cursor root changed original work/state", c.phase, meter.actual)
		}
		for coordinate, actual := range meter.actual {
			if actual > upper.commitment[coordinate] {
				t.Fatal("normal phase escaped its prospective cursor row", c.phase, coordinate)
			}
		}
		seen[c.phase] = true
		if c.phase == "done" {
			finished = true
			break
		}
		c = historyCollectionContentStep(t, c, in, methods)
	}
	for _, phase := range []string{"open", "content", "effects", "state", "grants", "history", "close", "done"} {
		if !seen[phase] {
			t.Fatal("normal source cursor phase was never reached", phase)
		}
	}
	if !finished || !c.semantic.atEnd() || c.semantic.blocks != 12 || c.semantic.items != 11 {
		t.Fatal("quoted normal read-only sequence did not reach its handwritten endpoint")
	}
}

func historyCollectionCursorCostPending(t *testing.T) *providerHistoryCollectionCursor {
	t.Helper()
	c, in, methods := historyCollectionContentFixture(t, 4)
	for steps := 0; c.phase != "effects" && steps < 8; steps++ {
		c = historyCollectionContentStep(t, c, in, methods)
	}
	if c.phase != "effects" || len(c.pending) == 0 || len(c.effects) == 0 || len(c.traceResults) != len(c.effects) {
		t.Fatal("actual original-shaped pending block/effect precursor")
	}
	return c
}

func TestProviderHistoryCollectionCursorBoundsFundOriginalOwnedCopy(t *testing.T) {
	c := historyCollectionCursorCostPending(t)
	upper, ok := providerHistoryCollectionCursorUpper(historyCollectionCursorCostLimits(t, c))
	if !ok {
		t.Fatal("prospective complete cursor copy allowance")
	}
	original := c.semantic.observer
	before := original.actual
	want := c.commitmentObserved(providerHistorySemanticUnitMeter())
	for _, coordinate := range []int{2, 10} {
		narrowed, valid := providerUpperSum(before[coordinate], upper.copy[coordinate])
		if !valid || narrowed > original.upper[coordinate] {
			t.Fatal("prospective copy does not fit original inert UNIT allowance")
		}
		original.upper[coordinate] = narrowed // same observer/actual, narrow only
	}
	copied := c.copyObserved(original)
	if copied == nil || copied == c || copied.semantic == c.semantic || copied.semantic.observer != original || original.failed || want == "" ||
		copied.commitmentObserved(providerHistorySemanticUnitMeter()) != want || c.commitmentObserved(providerHistorySemanticUnitMeter()) != want {
		t.Fatal("quoted copy changed its source meaning or original observer")
	}
	for coordinate, actual := range original.actual {
		if actual < before[coordinate] || actual-before[coordinate] > upper.copy[coordinate] {
			t.Fatal("original copy reset work or exceeded the complete row", coordinate)
		}
	}
	if &copied.spine[0] == &c.spine[0] || &copied.effects[0] == &c.effects[0] || &copied.pending[0] == &c.pending[0] || &copied.traceResults[0] == &c.traceResults[0] ||
		copied.semantic.window == c.semantic.window || copied.semantic.window.run == c.semantic.window.run {
		t.Fatal("cursor copy reused mutable arrays or working semantic owners")
	}
	transactionArrays := 0
	for i, block := range c.pending {
		if len(block.transactions) != 0 {
			transactionArrays++
			if &copied.pending[i].transactions[0] == &block.transactions[0] || &copied.pending[i].receipts[0] == &block.receipts[0] {
				t.Fatal("pending transaction/receipt arrays were shared")
			}
		}
	}
	if transactionArrays == 0 {
		t.Fatal("normal pending fixture did not exercise retained transaction arrays")
	}
}

func TestProviderHistoryCollectionCursorBoundsCannotTransferExecutionObserver(t *testing.T) {
	c := historyCollectionCursorCostPending(t)
	upper, ok := providerHistoryCollectionCursorUpper(historyCollectionCursorCostLimits(t, c))
	if !ok {
		t.Fatal("prospective cursor copy allowance")
	}
	original := c.semantic.observer
	before := original.actual
	want := c.commitmentObserved(providerHistorySemanticUnitMeter())
	replacement := providerHistoryContentUnitMeter(upper.copy)
	if copied := c.copyObserved(replacement); copied != nil || replacement.failed || replacement.actual[10] == 0 || original.actual != before || c.semantic.observer != original ||
		c.commitmentObserved(providerHistorySemanticUnitMeter()) != want || want == "" {
		t.Fatal("numeric read-only quote transferred execution/snapshot ownership")
	}
}

func TestProviderHistoryCollectionCursorBoundsRejectInvalidDomains(t *testing.T) {
	valid := providerHistoryCollectionCursorLimits{Spine: 1, Transactions: 1, EvidenceBytes: 1, BodyBytes: 1}
	for _, change := range []func(*providerHistoryCollectionCursorLimits){
		func(l *providerHistoryCollectionCursorLimits) { l.Spine = 0 },
		func(l *providerHistoryCollectionCursorLimits) { l.Spine = 1<<20 + 1 },
		func(l *providerHistoryCollectionCursorLimits) { l.Effects = 163841 },
		func(l *providerHistoryCollectionCursorLimits) { l.PendingBlocks = 11 },
		func(l *providerHistoryCollectionCursorLimits) { l.Transactions = 0 },
		func(l *providerHistoryCollectionCursorLimits) { l.Transactions = 16385 },
		func(l *providerHistoryCollectionCursorLimits) { l.EvidenceBytes = 0 },
		func(l *providerHistoryCollectionCursorLimits) { l.EvidenceBytes = 64<<20 + 1 },
		func(l *providerHistoryCollectionCursorLimits) { l.BodyBytes = 0 },
		func(l *providerHistoryCollectionCursorLimits) { l.BodyBytes = 8<<20 + 1 },
		func(l *providerHistoryCollectionCursorLimits) { l.Semantic.ConfirmedClaims = 1<<20 + 1 },
		func(l *providerHistoryCollectionCursorLimits) { l.Semantic.Queue = 4097 },
	} {
		l := valid
		change(&l)
		if got, ok := providerHistoryCollectionCursorUpper(l); ok || got != (providerHistoryCollectionCursorCost{}) {
			t.Fatal("invalid cursor domain returned a partial copy/root upper", l, got, ok)
		}
	}
}
