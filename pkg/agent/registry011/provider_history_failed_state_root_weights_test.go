package registry011

import (
	"context"
	"testing"
)

func TestProviderHistoryFailedStateRootWeightsMatchBeforeStateIntervals(t *testing.T) {
	for _, tc := range []struct{ prior, members, first, last, roots, copies uint64 }{
		{0, 1, 0, 0, 148, 36}, {0, 1, 1, 1, 0, 0}, {0, 32, 0, 1, 520, 98},
		{1, 3, 0, 0, 96, 24}, {1, 3, 1, 1, 184, 28}, {1, 3, 0, 2, 280, 52},
		{5, 32, 1, 4, 432, 48}, {5, 32, 5, 5, 532, 86}, {5, 32, 0, 6, 1060, 158},
		{349524, 32, 0, 349525, 37749112, 4194386},
	} {
		if v, ok := providerHistoryCollectionFailedStageStateCallsBound(tc.prior, tc.members, tc.first, tc.last); !ok || v != (providerHistoryCollectionStateRootCalls{tc.roots, tc.copies}) {
			t.Fatal("completed failed S did not retain original before-state placement", tc, v)
		}
	}
	for _, tc := range []struct{ groups, requests, first, last, roots, copies uint64 }{
		{1, 1, 0, 0, 152, 41}, {1, 1, 0, 1, 161, 41}, {1, 1, 1, 1, 9, 0},
		{2, 33, 0, 0, 636, 131}, {2, 33, 1, 1, 551, 89}, {2, 33, 0, 2, 821, 158},
		{3, 96, 0, 3, 1979, 351}, {349525, 11184800, 0, 349525, 7696731340649, 855207633775},
	} {
		if v, ok := providerHistoryCollectionTerminalStateCallsBound(tc.groups, tc.requests, tc.first, tc.last); !ok || v != (providerHistoryCollectionStateRootCalls{tc.roots, tc.copies}) {
			t.Fatal("prospective terminal root interval", tc, v)
		}
	}
	for _, tc := range [][4]uint64{{0, 0, 0, 0}, {0, 33, 0, 0}, {349525, 1, 0, 1}, {0, 1, 1, 0}, {0, 1, 0, 2}, {^uint64(0), 1, 0, 0}} {
		if v, ok := providerHistoryCollectionFailedStageStateCallsBound(tc[0], tc[1], tc[2], tc[3]); ok || v != (providerHistoryCollectionStateRootCalls{}) {
			t.Fatal("invalid failure row exposed partial counts", tc, v)
		}
	}
	if v, ok := providerHistoryCollectionTerminalStateCallsBound(2, 1, 0, 2); ok || v != (providerHistoryCollectionStateRootCalls{}) {
		t.Fatal("invalid inherited whole domain exposed terminal counts")
	}
	// Pure price DATA, not measured work: fewer total roots do not imply
	// dominance when before/after prices differ. Copies include their root.
	before, _ := providerHistoryCollectionStageStateCallsBound(0, 1, false, 0, 0)
	after, _ := providerHistoryCollectionStageStateCallsBound(0, 1, false, 1, 1)
	failed, _ := providerHistoryCollectionFailedStageStateCallsBound(0, 1, 0, 1)
	if before.roots*10+after.roots != 1459 || failed.roots*10 != 1480 || failed.roots >= before.roots+after.roots || failed.copies >= before.copies {
		t.Fatal("total-count comparison hid before-state failed-S price")
	}
}

// These are independently expanded original call phases, not a runtime of the
// owner/transport. Optional results change noncursor costs, never these roots.
func historyFailedStateRootStage(prior, attempts uint64) historyStateRootTimeline {
	s := historyStateRootTimeline{}
	for range 12 {
		s.prefix(prior)
	}
	for _, row := range [][2]uint64{{6, 0}, {9, 3}, {24, 6}, {22, 4}, {12 * attempts, 2 * attempts}} {
		s.add(prior, row[0], row[1])
	}
	for range 3 { // Coordinator, before rename, after rename.
		s.add(prior, 9, 2) // Own original consumed-C replay, excluding prefix.
		s.add(prior, 1, 1) // Owned ledger copy.
		s.add(prior, 1, 0) // Consumed-ledger validity.
		s.add(prior, 1, 0) // Failed terminal advance retains the same cursor.
		s.add(prior, 1, 0) // terminalDataValid, not live validity.
	}
	return s
}

func TestProviderHistoryFailedStateRootWeightsCoverShorterCompletedPrograms(t *testing.T) {
	for _, members := range [][]uint64{{1}, {32}, {1, 32}, {32, 1}, {1, 1, 32}, {32, 1, 32, 1}} {
		whole := historyStateRootTimeline{}
		whole.add(0, 4, 2) // External initialization, outside firstPReading.
		var requests uint64
		for j, count := range members {
			prior, final := uint64(j), j+1 == len(members)
			s := historyStateRootStage(prior, count, true)
			if final {
				s = historyFailedStateRootStage(prior, count)
				for first := uint64(0); first <= prior+1; first++ {
					for last := first; last <= prior+1; last++ {
						if v, ok := providerHistoryCollectionFailedStageStateCallsBound(prior, count, first, last); !ok || v != s.interval(first, last) {
							t.Fatal("original failed-S chronology differs", members, first, last, v)
						}
					}
				}
				coarse, ok := providerHistoryCollectionStageRootCallsBound(prior, count, false)
				v := s.interval(0, prior+1)
				if !ok || v.roots+6 != coarse.cursorRoots || v.copies+3 != coarse.cursorCopies {
					t.Fatal("failed count difference from successful S changed")
				}
			}
			for k, v := range s {
				whole.add(k, v.roots, v.copies)
			}
			requests += count
		}
		groups := uint64(len(members))
		for maximumGroups := groups; maximumGroups <= groups+3; maximumGroups++ {
			for _, maximumRequests := range []uint64{max(requests, maximumGroups), 32 * maximumGroups} {
				for first := uint64(0); first <= maximumGroups; first++ {
					for last := first; last <= maximumGroups; last++ {
						v := whole.interval(first, last)
						upper, ok := providerHistoryCollectionTerminalStateCallsBound(maximumGroups, maximumRequests, first, last)
						if !ok || v.roots > upper.roots || v.copies > upper.copies {
							t.Fatal("shorter failed program exceeded own original interval", members, maximumGroups, first, last, v, upper)
						}
					}
				}
			}
		}
	}
}

func TestProviderHistoryFailedStateRootWeightsKeepThreeOriginalTerminalModels(t *testing.T) {
	for _, prior := range []uint64{0, 1} {
		c, headers, methods := historyCollectionBoundaryFixture(t, 13, 4)
		base := historyCollectionLedger(t, c)
		if prior == 1 {
			s, results := historyCollectionPendingSettlement(t, base, headers, methods)
			if base.settleOriginalsObserved(context.Background(), s, results, methods, base.collection.semantic.observer) != nil {
				t.Fatal("prior successful boundary")
			}
		}
		_, _ = historyCollectionPendingSettlement(t, base, headers, methods)
		original, cursor, charged := base.collection.semantic.observer, base.collection, base.charged
		root, head := cursor.commitmentObserved(providerHistorySemanticUnitMeter()), base.head
		failure, err := newProviderHistoryGroupSettlement(base.ticket, base.consumption, base.head, admissionTestDigest("2"), "failed", admissionTestDigest("f"), nil)
		if err != nil || root == "" {
			t.Fatal("original consumed-C failed-settlement DATA", err)
		}
		var models [3]*providerHistoryGroupLedger
		for i := range models {
			model := providerHistoryLedgerCopyObserved(base, original)
			if model == nil || model.collection == cursor || model.collection.semantic.observer != original ||
				model.settleOriginalsObserved(context.Background(), failure, nil, nil, original) != nil || !model.terminalDataValidObserved(original) ||
				model.validObserved(original) || model.stage != prior+1 || model.collection.completed != prior || model.collection.phase != cursor.phase ||
				model.collection.commitmentObserved(providerHistorySemanticUnitMeter()) != root || model.charged != charged {
				t.Fatal("terminal ledger stage advanced the successful cursor or refunded its upper", prior, i)
			}
			models[i] = model
			if i > 0 && (model.head != models[0].head || model.collection == models[0].collection || &model.collection.spine[0] == &models[0].collection.spine[0]) {
				t.Fatal("independent terminal models differed or shared mutable cursor backing")
			}
		}
		if original.failed || base.head != head || base.collection != cursor || base.charged != charged || cursor.completed != prior || cursor.semantic.observer != original ||
			cursor.commitmentObserved(providerHistorySemanticUnitMeter()) != root {
			t.Fatal("terminal DATA replaced the original observer or mutated the consumed source")
		}
		// This executes original primitives only. It does not authenticate the
		// three actual I/O readers, count their prefix work or price nonroots.
	}
}
