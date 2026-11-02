package registry011

import (
	"context"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestProviderHistoryAppliedCursorPartsMatchIndependentComponentRows(t *testing.T) {
	for _, tc := range []struct {
		spine, effects, blocks, txs, normal10, copy64 uint64
		window                                        bool
	}{
		{1, 0, 0, 1, 143778564, 5, false},
		{8, 2, 2, 3, 716888597, 546, true},
		{1 << 20, 163840, 10, 16384, 2813522047279, 9078280, true},
	} {
		l := providerHistoryCollectionCursorLimits{Spine: tc.spine, Effects: tc.effects, PendingBlocks: tc.blocks, Transactions: tc.txs, EvidenceBytes: 64 << 20, BodyBytes: 8 << 20}
		kinds, history := providerHistoryAppliedStateKinds{ConfirmedEmpty: !tc.window}, uint64(0)
		snapshotLimits := providerHistorySemanticSnapshotLimits{Confirmed: providerHistorySemanticCloneLimits{Keys: 128, Services: 16, Grants: 128}}
		if tc.window {
			l.Semantic = providerHistorySemanticCommitmentLimits{3, 5, 7, 11, 3, true, true}
			history = 65536
			snapshotLimits = providerHistorySemanticSnapshotLimits{Confirmed: providerHistorySemanticCloneLimits{3, 5, 128, 16, 128},
				Working: providerHistorySemanticCloneLimits{7, 11, 128, 16, 128}, Queue: 3, Window: true, Pending: true}
		}
		semantic, a := providerHistoryAppliedCommitmentKindsUpper(l.Semantic, history, kinds)
		snapshot, b := providerHistoryAppliedSnapshotUpper(snapshotLimits, history, kinds)
		old, c := providerHistoryCollectionCursorUpper(l)
		oldSemantic, d := providerHistorySemanticCommitmentUpper(l.Semantic)
		if !a || !b || !c || !d {
			t.Fatal("independently closed child precursors")
		}
		want := providerHistoryAppliedCursorCost{normalCommitment: semantic.normal}
		want.normalCommitment[10] += tc.normal10
		for coordinate := range want.envelopeCommitment {
			want.envelopeCommitment[coordinate] = old.commitment[coordinate] - oldSemantic[coordinate] + semantic.envelope[coordinate]
			want.normalCopy[coordinate] = want.normalCommitment[coordinate] + snapshot[coordinate]
			want.envelopeCopy[coordinate] = want.envelopeCommitment[coordinate] + snapshot[coordinate]
		}
		// Portable event arguments only; no native/backing allocation claim.
		copies := tc.copy64 - tc.blocks*152 - tc.effects*16 - 4*tc.blocks*tc.txs*8 + tc.blocks*uint64(reflect.TypeFor[providerHistoryCollectionBlock]().Size()) +
			tc.effects*uint64(reflect.TypeFor[string]().Size()) + 4*tc.blocks*tc.txs*uint64(strconv.IntSize/8)
		normalCopies := copies - 4*tc.blocks*(tc.txs-min(tc.txs, l.BodyBytes/516))*uint64(strconv.IntSize/8)
		want.normalCopy[10] += normalCopies
		want.envelopeCopy[10] += copies
		if got, ok := providerHistoryAppliedCursorUpper(l, history, kinds); !ok || got != want {
			t.Fatal("cursor root/copy differs from independent disjoint rows", tc, got, want)
		}
	}
	valid := providerHistoryCollectionCursorLimits{Spine: 1, Transactions: 1, EvidenceBytes: 1, BodyBytes: 1}
	for _, change := range []func(*providerHistoryCollectionCursorLimits){
		func(l *providerHistoryCollectionCursorLimits) { l.Spine = 0 }, func(l *providerHistoryCollectionCursorLimits) { l.Effects = 163841 },
		func(l *providerHistoryCollectionCursorLimits) { l.PendingBlocks = 11 }, func(l *providerHistoryCollectionCursorLimits) { l.Transactions = 0 },
		func(l *providerHistoryCollectionCursorLimits) { l.EvidenceBytes = 64<<20 + 1 }, func(l *providerHistoryCollectionCursorLimits) { l.BodyBytes = 0 },
		func(l *providerHistoryCollectionCursorLimits) { l.Semantic.ConfirmedClaims = ^uint64(0) },
	} {
		l := valid
		change(&l)
		if got, ok := providerHistoryAppliedCursorUpper(l, 0, providerHistoryAppliedStateKinds{ConfirmedEmpty: true}); ok || got != (providerHistoryAppliedCursorCost{}) {
			t.Fatal("invalid original child returned partial cursor parts", l)
		}
	}
	if got, ok := providerHistoryAppliedCursorUpper(valid, 0, providerHistoryAppliedStateKinds{WorkingEmpty: true}); ok || got != (providerHistoryAppliedCursorCost{}) {
		t.Fatal("missing working branch acquired a source kind")
	}
}

func TestProviderHistoryAppliedCursorReceiptCountKeepsIncomingEnvelopes(t *testing.T) {
	// These finite numeric selections price one retained block, not decoding
	// permission. The literal 131762 is the two list-item rows plus their two
	// byte-root framing events per additional receipt/transaction pair.
	l := providerHistoryCollectionCursorLimits{Spine: 1, PendingBlocks: 1, Transactions: 16384, EvidenceBytes: 65536, BodyBytes: 1}
	kinds := providerHistoryAppliedStateKinds{ConfirmedEmpty: true}
	base, ok := providerHistoryAppliedCursorUpper(l, 0, kinds)
	if !ok {
		t.Fatal("zero retained-row cardinality precursor")
	}
	for _, tc := range []struct{ body, retained uint64 }{
		{1, 0}, {515, 0}, {516, 1}, {1031, 1}, {1032, 2}, {65536, 127}, {8 << 20, 16256},
	} {
		l.BodyBytes = tc.body
		got, valid := providerHistoryAppliedCursorUpper(l, 0, kinds)
		wantRoot, wantCopy := base.normalCommitment, base.normalCopy
		wantRoot[10] += tc.retained * 131762
		wantCopy[10] += tc.retained * (131762 + 4*uint64(strconv.IntSize/8))
		if !valid || got.normalCommitment != wantRoot || got.normalCopy != wantCopy ||
			got.envelopeCommitment != base.envelopeCommitment || got.envelopeCopy != base.envelopeCopy {
			t.Fatal("receipt boundary changed full incoming arrays or normal event rows", tc, valid)
		}
	}
	// The independently selected transaction cap still wins when smaller.
	l.Transactions = 1
	capped, valid := providerHistoryAppliedCursorUpper(l, 0, kinds)
	if !valid || capped.normalCommitment[10] != base.normalCommitment[10]+131762 ||
		capped.normalCopy[10] != base.normalCopy[10]+131762+4*uint64(strconv.IntSize/8) {
		t.Fatal("normal receipt count bypassed the original transaction cap")
	}
}

func TestProviderHistoryAppliedReceiptCardinalityFundsOriginalPendingCopy(t *testing.T) {
	c := historyCollectionCursorCostPending(t)
	l := historyCollectionCursorCostLimits(t, c)
	// Widen only this prospective DATA quote. The source selection, real
	// decoder, cursor, original observer and existing allowance stay intact.
	l.Transactions = 16384
	parts, ok := providerHistoryAppliedCursorUpper(l, uint64(c.selection.input.journal.Replay.History.MaxBytes), providerHistoryAppliedStateKinds{})
	if !ok {
		t.Fatal("prospective retained-receipt copy allowance")
	}
	for _, block := range c.pending {
		if uint64(block.transactionCount) > min(l.Transactions, l.BodyBytes/516) {
			t.Fatal("original successful decoder retained too many receipt rows")
		}
	}
	original := c.semantic.observer
	before, beforeUpper := original.actual, original.upper
	want := c.commitmentObserved(providerHistorySemanticUnitMeter())
	copied := c.copyObserved(original)
	if copied == nil || copied == c || copied.semantic.observer != original || original.failed || original.upper != beforeUpper ||
		want == "" || copied.commitmentObserved(providerHistorySemanticUnitMeter()) != want {
		t.Fatal("normal retained copy changed its original meaning, observer or allowance")
	}
	for coordinate, actual := range original.actual {
		if actual < before[coordinate] || actual-before[coordinate] > parts.normalCopy[coordinate] {
			t.Fatal("original pending copy escaped the retained-receipt row", coordinate)
		}
	}
}

func TestProviderHistoryAppliedCursorPartsKeepOriginalMetadataExtents(t *testing.T) {
	h, d := admissionTestDigest("a"), "18446744073709551615"
	header := map[string]any{"hash": h, "parent_hash": h, "number": d, "timestamp": "9007199254740991"}
	for _, tc := range []struct {
		kind   string
		value  map[string]any
		bytes  int
		actual uint64
	}{
		{"spine", map[string]any{"previous": h, "ordinal": "1048575", "header": header}, 402, 162056},
		{"effects", map[string]any{"previous": h, "ordinal": "163839", "block": h, "transaction": h, "number": d, "index": d}, 393, 154894},
		{"pending-empty", map[string]any{}, 77, 6064},
		{"pending-list", map[string]any{"count": 16384}, 90, 8247},
		{"pending-list-item", map[string]any{"previous": h, "ordinal": "16383", "bytes": h}, 256, 65854},
		{"pending-block", map[string]any{"previous": h, "ordinal": 9, "header": header, "applied": false, "transaction_count": 16384,
			"header_rlp": h, "transactions": h, "receipts": h, "gas_profile": SepoliaAmsterdamNethermindGasProfile, "gas_trace": h}, 829, 688128},
		{"pending-trace", map[string]any{"previous": h, "ordinal": 163839, "applied": false, "received": false, "bytes": h}, 284, 80998},
		{"pending-batch", map[string]any{"members": h, "blocks": 10, "applied": 9}, 180, 32638},
		{"cursor", map[string]any{"program": h, "manifest": h, "selected": header, "phase": "ancestry", "mode": "ancestry", "low": d, "high": d,
			"next": d, "completed_groups": d, "spine": h, "spine_count": "1048576", "effects": h, "effect_count": "163840", "effect_next": "163840",
			"traces": d, "native": d, "history_next": d, "stop_version": d, "genesis": header, "client": strings.Repeat("\x01", 256),
			"semantic": h, "pending": h, "stop_record": h, "present_stops": d}, 3062, 9378957},
	} {
		// Finite synthetic upper DATA. Some combinations are deliberately
		// wider than a reachable cursor; they are not selected source facts.
		meter := providerHistorySemanticUnitMeter()
		root := ""
		if tc.kind == "spine" {
			root = providerHistoryCollectionSpineDigestObserved(h, (1<<20)-1, FinalizedHeader{h, h, ^uint64(0), 1<<53 - 1}, meter)
			tc.value["format"], tc.value["kind"] = providerHistoryCollectionStateDomain, tc.kind
		} else {
			root = providerHistoryCollectionValueDigestObserved(tc.kind, tc.value, meter)
		}
		if root == "" || meter.failed || meter.actual != (candidateWorkVector{10: tc.actual}) || len(providerReplayCanonical(tc.value, candidateLedgerMetadataLimit, nil)) != tc.bytes {
			t.Fatal("original fixed metadata extent or event row changed", tc.kind, meter.actual)
		}
	}
}

func TestProviderHistoryAppliedCursorPartsKeepOriginalAncestryEmpty(t *testing.T) {
	// Original boundary DATA routines are used from constructor through every
	// parent window. This is not a live Source or proof of RPC/header trust.
	c, headers, methods := historyCollectionBoundaryFixture(t, 13, 4)
	finished := false
	for steps := 0; steps < 50; steps++ {
		l := historyCollectionCursorCostLimits(t, c)
		// Ancestry construction/boundaries have not entered semantic replay.
		// These prospective zero components follow that route, not a report
		// of the current state length or an arbitrary patch to c.mode.
		l.Effects, l.PendingBlocks, l.Semantic = 0, 0, providerHistorySemanticCommitmentLimits{}
		parts, ok := providerHistoryAppliedCursorUpper(l, 0, providerHistoryAppliedStateKinds{ConfirmedEmpty: true})
		if !ok || c.semantic.confirmed.state.version != 0 || len(c.semantic.confirmed.claims) != 0 || len(c.semantic.confirmed.created) != 0 || c.semantic.window != nil {
			t.Fatal("original ancestry route lost its empty semantic precursor", c.phase)
		}
		original, meter := c.semantic.observer, providerHistoryContentUnitMeter(parts.envelopeCommitment)
		before := original.actual
		want := c.commitmentObserved(providerHistorySemanticUnitMeter())
		if got := c.commitmentObserved(meter); got == "" || got != want || meter.failed || original.actual != before || c.semantic.observer != original {
			t.Fatal("empty ancestry root changed original meaning or observer")
		}
		for coordinate, actual := range meter.actual {
			if actual > parts.normalCommitment[coordinate] {
				t.Fatal("actual ancestry root escaped conditional normal", coordinate)
			}
		}
		if c.mode == "content" {
			finished = true
			break
		}
		table, err := c.nextTableObserved(c.completed+1, *methods, c.selection.input.capture, original)
		if err != nil {
			t.Fatal("exact original ancestry table", err)
		}
		results := historyCollectionBoundaryResults(t, c, headers, table)
		next, err := c.advanceBoundaryObserved(context.Background(), table, results, methods, original)
		if err != nil || next == nil || next.semantic.observer != original {
			t.Fatal("original parent/window transition", err)
		}
		c = next
	}
	if !finished || len(c.spine) != 13 || c.phase != "open" || c.semantic.blocks != 0 {
		t.Fatal("complete parent reconstruction did not reach pre-content boundary")
	}
}

func TestProviderHistoryAppliedCursorPartsPreserveEveryContentPhaseAndCopy(t *testing.T) {
	c, in, methods := historyCollectionContentFixture(t, 4)
	seen, copied, done := map[string]bool{}, false, false
	for steps := 0; steps < 200; steps++ {
		l := historyCollectionCursorCostLimits(t, c)
		parts, ok := providerHistoryAppliedCursorUpper(l, uint64(c.selection.input.journal.Replay.History.MaxBytes), providerHistoryAppliedStateKinds{})
		if !ok {
			t.Fatal("full nonempty conditional cursor parts")
		}
		original := c.semantic.observer
		before, want := original.actual, c.commitmentObserved(providerHistorySemanticUnitMeter())
		meter := providerHistoryContentUnitMeter(parts.envelopeCommitment)
		if root := c.commitmentObserved(meter); root == "" || root != want || meter.failed || original.actual != before || c.semantic.observer != original {
			t.Fatal("read-only original cursor changed", c.phase)
		}
		for coordinate, actual := range meter.actual {
			if actual > parts.normalCommitment[coordinate] {
				t.Fatal("original nonempty phase escaped entered part", c.phase, coordinate)
			}
		}
		if c.phase == "effects" && !copied {
			copy := c.copyObserved(original)
			if copy == nil || copy == c || copy.semantic == c.semantic || copy.semantic.observer != original || original.failed || copy.commitmentObserved(providerHistorySemanticUnitMeter()) != want {
				t.Fatal("original owned cursor copy changed meaning or observer")
			}
			for coordinate, actual := range original.actual {
				if actual < before[coordinate] || actual-before[coordinate] > parts.normalCopy[coordinate] {
					t.Fatal("owned copy escaped prospective entered work", coordinate)
				}
			}
			if &copy.pending[0] == &c.pending[0] || &copy.effects[0] == &c.effects[0] || &copy.spine[0] == &c.spine[0] || copy.semantic.window == c.semantic.window {
				t.Fatal("owned copy reused mutable original arrays/window")
			}
			copied = true
		}
		seen[c.phase] = true
		if c.phase == "done" {
			done = true
			break
		}
		c = historyCollectionContentStep(t, c, in, methods)
	}
	for _, phase := range []string{"open", "content", "effects", "state", "grants", "history", "close", "done"} {
		if !seen[phase] {
			t.Fatal("original nonempty phase missing", phase)
		}
	}
	if !done || !copied || !c.semantic.atEnd() || c.semantic.blocks != 12 || c.semantic.items != 11 {
		t.Fatal("original complete nonempty endpoint or owned copy not reached")
	}
}

func TestProviderHistoryAppliedCursorPartsCoverDeferredPendingAfterSemanticRefusal(t *testing.T) {
	c := historyCollectionCursorCostPending(t)
	l := historyCollectionCursorCostLimits(t, c)
	parts, ok := providerHistoryAppliedCursorUpper(l, uint64(c.selection.input.journal.Replay.History.MaxBytes), providerHistoryAppliedStateKinds{})
	if !ok || c.pendingApplied < 0 || c.pendingApplied >= len(c.pending) || c.pending[c.pendingApplied].headerRLP == "" {
		t.Fatal("real original-shaped unapplied pending block precursor")
	}
	// Deliberately unqualified finite in-memory DATA isolates the late record
	// cap and empty-index deterministic event sequence. It cannot supply a
	// parsed command, original Source, permit or authenticated state lineage.
	w := c.semantic.window
	c.semantic.confirmed.claims, c.semantic.confirmed.created, w.run.claims, w.run.created = nil, nil, nil, nil
	w.run.state = replayState{version: 1, admitted: 1, record: replayRecord{ID: "unit", State: "active", Version: "1", Services: make([]replayService, 16)},
		grants: []replayGrant{{Operator: replayTestWallet, Scope: "update-services"}}}
	for i := range w.run.state.record.Services {
		w.run.state.record.Services[i] = replayService{Name: "unit", Type: "unit", URI: strings.Repeat("\x01", 2048)}
	}
	original, before := c.semantic.observer, c.semantic.observer.actual
	semantic := providerHistoryContentUnitMeter(parts.envelopeCommitment)
	if root := c.semantic.commitmentObserved(semantic); root != "" || semantic.failed || semantic.actual[10] == 0 {
		t.Fatal("healthy late semantic refusal precursor")
	}
	wholeA, pendingA := providerHistoryContentUnitMeter(parts.envelopeCommitment), providerHistoryContentUnitMeter(parts.envelopeCommitment)
	if c.commitmentObserved(wholeA) != "" || wholeA.failed || c.pendingRootObserved(pendingA) == "" || pendingA.failed {
		t.Fatal("original semantic refusal or valid pending DATA precursor")
	}
	// Remove ONLY the first unapplied header text. Both tx/receipt sibling
	// list calls still run before pending's combined false. The whole root
	// already has a semantic false in BOTH trials, so its exact event-vector
	// delta must equal the separately measured pending-call delta.
	index, header := c.pendingApplied, c.pending[c.pendingApplied].headerRLP
	c.pending[index].headerRLP = ""
	wholeB, pendingB := providerHistoryContentUnitMeter(parts.envelopeCommitment), providerHistoryContentUnitMeter(parts.envelopeCommitment)
	root, pendingRoot := c.commitmentObserved(wholeB), c.pendingRootObserved(pendingB)
	c.pending[index].headerRLP = header
	if root != "" || pendingRoot != "" || wholeB.failed || pendingB.failed || pendingA.actual[10] <= pendingB.actual[10] || original.actual != before || c.semantic.observer != original {
		t.Fatal("healthy deferred refusal or original observer changed")
	}
	for coordinate, actual := range wholeA.actual {
		if actual < wholeB.actual[coordinate] || pendingA.actual[coordinate] < pendingB.actual[coordinate] ||
			actual-wholeB.actual[coordinate] != pendingA.actual[coordinate]-pendingB.actual[coordinate] || actual > parts.envelopeCommitment[coordinate] {
			t.Fatal("semantic false skipped deferred pending calls or escaped full caller envelope", coordinate)
		}
	}
	clean := historyCollectionCursorCostPending(t)
	cleanParts, valid := providerHistoryAppliedCursorUpper(historyCollectionCursorCostLimits(t, clean), uint64(clean.selection.input.journal.Replay.History.MaxBytes), providerHistoryAppliedStateKinds{})
	if !valid {
		t.Fatal("normal owned-copy precursor")
	}
	execution, start := clean.semantic.observer, clean.semantic.observer.actual
	replacement := providerHistoryContentUnitMeter(cleanParts.envelopeCopy)
	if copy := clean.copyObserved(replacement); copy != nil || replacement.failed || replacement.actual[10] == 0 || execution.actual != start || clean.semantic.observer != execution {
		t.Fatal("numeric read-only cost transferred original execution/snapshot ownership")
	}
}

func TestProviderHistoryRoleStateCallsMatchIndependentStateCuts(t *testing.T) {
	for _, tc := range []struct {
		prior, first, last uint64
		advancing          bool
		want               [6]providerHistoryCollectionStateRootCalls
	}{
		{0, 0, 0, true, [6]providerHistoryCollectionStateRootCalls{{88, 22}, {12, 2}, {12, 2}, {45, 15}, {48, 12}, {16, 4}}},
		{0, 1, 1, true, [6]providerHistoryCollectionStateRootCalls{{}, {}, {}, {9, 0}, {}, {10, 0}}},
		{1, 0, 0, true, [6]providerHistoryCollectionStateRootCalls{{72, 18}, {}, {}, {24, 6}, {24, 6}, {16, 4}}},
		{1, 1, 1, true, [6]providerHistoryCollectionStateRootCalls{{97, 13}, {12, 2}, {12, 2}, {48, 12}, {51, 9}, {18, 2}}},
		{1, 2, 2, true, [6]providerHistoryCollectionStateRootCalls{{}, {}, {}, {9, 0}, {}, {10, 0}}},
		{2, 1, 2, true, [6]providerHistoryCollectionStateRootCalls{{178, 22}, {12, 2}, {12, 2}, {75, 15}, {78, 12}, {36, 4}}},
		{2, 1, 3, true, [6]providerHistoryCollectionStateRootCalls{{178, 22}, {12, 2}, {12, 2}, {84, 15}, {78, 12}, {46, 4}}},
		{7, 0, 8, false, [6]providerHistoryCollectionStateRootCalls{{655, 85}, {12, 2}, {12, 2}, {243, 36}, {237, 33}, {}}},
	} {
		got, ok := providerHistoryCollectionRoleStateCallsBound(tc.prior, tc.first, tc.last, tc.advancing)
		if !ok || got != (providerHistoryRoleStateCalls{tc.want[0], tc.want[1], tc.want[2], tc.want[3], tc.want[4], tc.want[5]}) {
			t.Fatal("original role source cuts differ from independent counts", tc, got, ok)
		}
	}
}

func TestProviderHistoryRoleStateCallsConserveExistingSuccessfulAndFailedGrammar(t *testing.T) {
	for prior := uint64(0); prior <= 16; prior++ {
		for first := uint64(0); first <= prior+1; first++ {
			for last := first; last <= prior+1; last++ {
				for members := uint64(1); members <= 32; members++ {
					for _, advancing := range []bool{false, true} {
						r, ok := providerHistoryCollectionRoleStateCallsBound(prior, first, last, advancing)
						success, valid := providerHistoryCollectionStageStateCallsBound(prior, members, advancing, first, last)
						failed, failedOK := providerHistoryCollectionFailedStageStateCallsBound(prior, members, first, last)
						actual := providerHistoryCollectionStateRootCalls{
							r.before.roots + members*r.member.roots + r.successClosure.roots + r.advance.roots,
							r.before.copies + members*r.member.copies + r.successClosure.copies + r.advance.copies,
						}
						terminal := providerHistoryCollectionStateRootCalls{
							r.before.roots + members*r.member.roots + r.failedClosure.roots,
							r.before.copies + members*r.member.copies + r.failedClosure.copies,
						}
						if !ok || !valid || !failedOK || actual != success || terminal != failed || r.preparation != r.member {
							t.Fatal("role partition lost original successful/failed state placement", prior, first, last, members, advancing)
						}
						// Extra P belongs to the full incoming window, not actual
						// grammar or an additional RPC/member. Failed S has no A.
						if first <= prior && prior <= last && r.preparation != (providerHistoryCollectionStateRootCalls{12, 2}) {
							t.Fatal("already-entered preparation was moved to the after state")
						}
					}
				}
			}
		}
	}
}

func TestProviderHistoryRoleStateCallsRejectInvalidOriginalIntervals(t *testing.T) {
	for _, tc := range []struct {
		prior, first, last uint64
		advancing          bool
	}{
		{(1 << 20) / 3, 0, 0, false}, {^uint64(0), 0, 0, true}, {(1<<20)/3 - 1, 0, 0, true},
		{0, 1, 0, false}, {0, 0, 2, false}, {1, 3, 3, false}, {0, ^uint64(0), ^uint64(0), false},
	} {
		if got, ok := providerHistoryCollectionRoleStateCallsBound(tc.prior, tc.first, tc.last, tc.advancing); ok || got != (providerHistoryRoleStateCalls{}) {
			t.Fatal("invalid original role-state domain retained partial counts", tc, got, ok)
		}
	}
	last, ok := providerHistoryCollectionRoleStateCallsBound((1<<20)/3-1, (1<<20)/3, (1<<20)/3, false)
	if !ok || last != (providerHistoryRoleStateCalls{successClosure: providerHistoryCollectionStateRootCalls{9, 0}}) {
		t.Fatal("last legal after-state interval acquired an advance or failed after state", last, ok)
	}
}

func TestProviderHistoryRoleCursorPartsMatchIndependentUniformRowsAndPartitions(t *testing.T) {
	limits := providerHistoryCollectionCursorLimits{Spine: 1, Transactions: 1, EvidenceBytes: 1, BodyBytes: 1}
	kinds := providerHistoryAppliedStateKinds{ConfirmedEmpty: true}
	// Closed empty-fold/no-window root constants and full original envelope.
	// Copy's architecture-dependent explicit event sizes are adjusted only
	// as declared by the original structs; this is not allocator/native RAM.
	const normal, envelope uint64 = 145064941, 4765216463
	copyExtra := uint64(reflect.TypeFor[providerHistorySemanticFold]().Size()+reflect.TypeFor[replayStream]().Size()+reflect.TypeFor[replayState]().Size()+4*reflect.TypeFor[string]().Size()) + 558
	quote := func(roots, copies uint64) candidateWorkVector {
		return candidateWorkVector{10: roots*normal + copies*copyExtra + envelope - normal}
	}
	for _, prior := range []uint64{0, 1, 7} {
		for _, advancing := range []bool{false, true} {
			one := []providerHistoryRoleStateBand{{first: 0, last: prior + 1, limits: limits, kinds: kinds}}
			got, ok := providerHistoryCollectionRoleCursorUpper(prior, advancing, one)
			want := providerHistoryRoleCursorParts{
				initial: quote(4, 2), before: quote(88+81*prior, 22+9*prior), preparation: quote(12, 2), member: quote(12, 2), closure: quote(54+27*prior, 15+3*prior),
			}
			if advancing {
				want.advance = quote(26+18*prior, 4+2*prior)
			}
			if !ok || got != want {
				t.Fatal("role cursor DATA differs from independent uniform quotes", prior, advancing, got, want, ok)
			}
			split := make([]providerHistoryRoleStateBand, 0, int(prior)+2)
			for state := uint64(0); state <= prior+1; state++ {
				split = append(split, providerHistoryRoleStateBand{first: state, last: state, limits: limits, kinds: kinds})
			}
			partitioned, valid := providerHistoryCollectionRoleCursorUpper(prior, advancing, split)
			if !valid || partitioned != got || got.preparation != got.member {
				t.Fatal("same original source band split changed per-role headroom or P/M", prior)
			}
			control, valid := providerHistoryCollectionRoleControlPartsUpper(prior, 1<<20, advancing)
			if !valid {
				t.Fatal("finite independent CONTROL component unavailable")
			}
			// Compose the disjoint CONTROL/cursor DATA at original cuts.
			// These synthetic OTHER-domain values do not open a real profile.
			before, prep := control.before, control.preparation
			rows := providerHistoryDispatchBounds{member: control.member.work, closure: control.closure.work, memberClocks: 2, closureClocks: 2, tailMS: 1}
			for i := range rows.member {
				before.work[i] += got.before[i]
				prep.work[i] += got.preparation[i]
				rows.member[i] += got.member[i]
				rows.closure[i] += got.closure[i]
			}
			rows.member[0], rows.member[1], rows.member[9], rows.closure[9] = 1, 101, 1, 1
			if advancing {
				rows.advance = control.advance.work
				for i := range rows.advance {
					rows.advance[i] += got.advance[i]
				}
				rows.advanceClocks, rows.advanceTailMS = 2, 1
			}
			for count := 1; count <= 32; count++ {
				window, valid := providerHistoryCollectionStageWorkUpper(before, prep, rows, count, advancing)
				wantClocks := uint64(9 + 2*count)
				if advancing {
					wantClocks += 2
				}
				if !valid || window.work[0] != uint64(count) || window.work[1] != 101*uint64(count) || window.clocks != wantClocks {
					t.Fatal("combined disjoint rows changed original count/wire/clock or rejected P", prior, count, advancing)
				}
			}
		}
	}
}

func TestProviderHistoryRoleCursorClosureKeepsFailedBeforeStateCost(t *testing.T) {
	before := providerHistoryCollectionCursorLimits{Spine: 1, PendingBlocks: 10, Transactions: 16384, EvidenceBytes: 64 << 20, BodyBytes: 8 << 20}
	after := providerHistoryCollectionCursorLimits{Spine: 1, Transactions: 1, EvidenceBytes: 1, BodyBytes: 1}
	kinds := providerHistoryAppliedStateKinds{ConfirmedEmpty: true}
	// Prospective DATA: pending work may clear on success and remain on a
	// known failed terminal. No actual selected state/owner is authenticated.
	bands := []providerHistoryRoleStateBand{{first: 0, last: 0, limits: before, kinds: kinds}, {first: 1, last: 1, limits: after, kinds: kinds}}
	b, ok := providerHistoryAppliedCursorUpper(before, 0, kinds)
	a, valid := providerHistoryAppliedCursorUpper(after, 0, kinds)
	if !ok || !valid {
		t.Fatal("independent original-child prerequisites failed")
	}
	// Receipt-body limits retain16256 pairs while the incoming full copy
	// still allows16384. The extra128*4*10 word copies stay in headroom.
	if b.envelopeCopy[10]-b.normalCopy[10] != b.envelopeCommitment[10]-b.normalCommitment[10]+5120*uint64(strconv.IntSize/8) {
		t.Fatal("full incoming array-copy difference was lost")
	}
	got, ok := providerHistoryCollectionRoleCursorUpper(0, false, bands)
	var success, failed, want candidateWorkVector
	for i := range want {
		// Fixed source calls: success has45before roots incl15copies plus
		// nineafter roots. Failed has48before incl12copies and NOafter.
		bh := max(b.envelopeCommitment[i]-b.normalCommitment[i], b.envelopeCopy[i]-b.normalCopy[i])
		ah := max(a.envelopeCommitment[i]-a.normalCommitment[i], a.envelopeCopy[i]-a.normalCopy[i])
		success[i] = 30*b.normalCommitment[i] + 15*b.normalCopy[i] + 9*a.normalCommitment[i] + max(bh, ah)
		failed[i] = 36*b.normalCommitment[i] + 12*b.normalCopy[i] + bh
		want[i] = max(success[i], failed[i])
	}
	if !ok || got.closure != want || failed[10] <= success[10] || got.closure[10] != failed[10] || got.advance != (candidateWorkVector{}) {
		t.Fatal("cheaper after-success state replaced known failed before-state closure", got, success, failed, ok)
	}
}

func TestProviderHistoryRoleCursorRejectsMissingBandsAndLatePartialOverflow(t *testing.T) {
	limits := providerHistoryCollectionCursorLimits{Spine: 1, Transactions: 1, EvidenceBytes: 1, BodyBytes: 1}
	kinds := providerHistoryAppliedStateKinds{ConfirmedEmpty: true}
	band := func(first, last uint64) providerHistoryRoleStateBand {
		return providerHistoryRoleStateBand{first: first, last: last, limits: limits, kinds: kinds}
	}
	for _, bands := range [][]providerHistoryRoleStateBand{
		nil, {}, {band(0, 0)}, {band(1, 1)}, {band(0, 0), band(2, 3)}, {band(0, 1), band(1, 3)}, {band(1, 1), band(0, 3)}, {band(0, 4)}, {band(0, 0), band(1, 0)},
	} {
		if got, ok := providerHistoryCollectionRoleCursorUpper(2, false, bands); ok || got != (providerHistoryRoleCursorParts{}) {
			t.Fatal("missing/reordered/overlapping original state band retained partial DATA", bands, got, ok)
		}
	}
	for _, invalid := range []uint64{(1 << 20) / 3, ^uint64(0)} {
		if got, ok := providerHistoryCollectionRoleCursorUpper(invalid, true, []providerHistoryRoleStateBand{band(0, 0)}); ok || got != (providerHistoryRoleCursorParts{}) {
			t.Fatal("invalid prior overflowed original interval")
		}
	}
	if got, ok := providerHistoryCollectionRoleCursorUpper((1<<20)/3-1, true, []providerHistoryRoleStateBand{band(0, 0)}); ok || got != (providerHistoryRoleCursorParts{}) {
		t.Fatal("advancing outside original group domain accepted")
	}
	bad := band(1, 1)
	bad.limits.EvidenceBytes = 0
	if got, ok := providerHistoryCollectionRoleCursorUpper(0, false, []providerHistoryRoleStateBand{band(0, 0), bad}); ok || got != (providerHistoryRoleCursorParts{}) {
		t.Fatal("invalid later child kept successful initializer or earlier role parts")
	}
	bad = band(1, 1)
	bad.kinds.WorkingEmpty = true
	if got, ok := providerHistoryCollectionRoleCursorUpper(0, false, []providerHistoryRoleStateBand{band(0, 0), bad}); ok || got != (providerHistoryRoleCursorParts{}) {
		t.Fatal("invalid later source kind kept partial DATA")
	}
	large := providerHistoryCollectionCursorLimits{Spine: 1 << 20, Effects: 163840, PendingBlocks: 10, Transactions: 16384, EvidenceBytes: 64 << 20, BodyBytes: 8 << 20,
		Semantic: providerHistorySemanticCommitmentLimits{Window: true, Queue: 1, Pending: true}}
	one, valid := providerHistoryAppliedCursorUpper(large, 8<<20, providerHistoryAppliedStateKinds{})
	if !valid || one.normalCommitment[10] == 0 {
		t.Fatal("finite large child precursor unavailable")
	}

	// Each individual band is finite, but joining two same-role normal
	// prefixes exceeds uint64. Initial/other earlier parts must not escape.
	for _, count := range []providerHistoryCollectionStateRootCalls{{4049991, 450009}, {4050000, 450000}} {
		if _, valid := count.pointwisePartsUpper(large, 8<<20, providerHistoryAppliedStateKinds{}); !valid {
			t.Fatal("individually finite state band overflowed before the intended join", count)
		}
	}
	joinedBands := []providerHistoryRoleStateBand{
		{first: 0, last: 49999, limits: large, historyBytes: 8 << 20},
		{first: 50000, last: 99999, limits: large, historyBytes: 8 << 20},
		{first: 100000, last: 149999, limits: large, historyBytes: 8 << 20},
		{first: 150000, last: 200001, limits: large, historyBytes: 8 << 20},
	}
	if got, ok := providerHistoryCollectionRoleCursorUpper(200000, false, joinedBands); ok || got != (providerHistoryRoleCursorParts{}) {
		t.Fatal("late same-role join overflow exposed initial or prior-band DATA", got, ok)
	}

	last := (uint64(1)<<20)/3 - 1
	got, ok := providerHistoryCollectionRoleCursorUpper(last, false, []providerHistoryRoleStateBand{{first: 0, last: last + 1, limits: large, historyBytes: 8 << 20}})
	if ok || got != (providerHistoryRoleCursorParts{}) {
		t.Fatal("late repeated-role overflow leaked successful initial parts", got, ok)
	}
}
