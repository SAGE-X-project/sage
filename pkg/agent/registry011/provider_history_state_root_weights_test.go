package registry011

import "testing"

func TestProviderHistoryStateRootWeightsMatchIndependentIntervals(t *testing.T) {
	for _, tc := range []struct {
		prior, first, last, roots, copies uint64
	}{
		{0, 0, 0, 3, 1}, {1, 0, 0, 8, 2}, {1, 1, 1, 4, 0}, {3, 1, 2, 18, 2},
		{3, 0, 3, 30, 4}, {17, 1, 16, 144, 16}, {9226, 0, 9226, 83037, 9227},
	} {
		if got, ok := providerHistoryCollectionPrefixStateCallsBound(tc.prior, tc.first, tc.last); !ok || got != (providerHistoryCollectionStateRootCalls{tc.roots, tc.copies}) {
			t.Fatal("independent original prefix interval", tc, got)
		}
	}
	for _, tc := range []struct {
		prior, members, first, last, roots, copies uint64
		advance                                    bool
	}{
		{0, 1, 0, 0, 145, 39, false}, {0, 1, 1, 1, 9, 0, false},
		{0, 1, 0, 0, 161, 43, true}, {0, 1, 1, 1, 19, 0, true},
		{1, 3, 0, 0, 112, 28, true}, {1, 3, 1, 1, 199, 33, true}, {1, 3, 2, 2, 19, 0, true},
		{5, 32, 1, 4, 504, 56, true}, {5, 32, 5, 5, 547, 91, true}, {5, 32, 0, 6, 1182, 175, true},
	} {
		if got, ok := providerHistoryCollectionStageStateCallsBound(tc.prior, tc.members, tc.advance, tc.first, tc.last); !ok || got != (providerHistoryCollectionStateRootCalls{tc.roots, tc.copies}) {
			t.Fatal("independent same-window before/after-S interval", tc, got)
		}
	}
	for _, tc := range []struct{ groups, requests, first, last, roots, copies uint64 }{
		{1, 1, 0, 1, 158, 41}, {2, 33, 0, 0, 633, 131}, {2, 33, 1, 1, 548, 89},
		{2, 33, 2, 2, 9, 0}, {2, 33, 0, 2, 818, 158}, {3, 96, 0, 3, 1976, 351},
		{349525, 11184800, 0, 349525, 7696731340646, 855207633775},
	} {
		if got, ok := providerHistoryCollectionWholeStateCallsBound(tc.groups, tc.requests, tc.first, tc.last); !ok || got != (providerHistoryCollectionStateRootCalls{tc.roots, tc.copies}) {
			t.Fatal("independent prospective whole interval", tc, got)
		}
	}
	// Separate bands have independent request maxima; do not pretend both
	// attain the same globalR at once or call their sum an exact transcript.
	left, _ := providerHistoryCollectionWholeStateCallsBound(2, 33, 0, 0)
	right, _ := providerHistoryCollectionWholeStateCallsBound(2, 33, 1, 2)
	whole, _ := providerHistoryCollectionWholeStateCallsBound(2, 33, 0, 2)
	if left.roots+right.roots <= whole.roots || left.copies+right.copies <= whole.copies {
		t.Fatal("independent interval maxima were silently treated as coattainable")
	}
}

// Source-call timeline DATA, not project execution/Source authentication. The
// phase rows are from original caller inspection, not the closed-form helper.
type historyStateRootTimeline map[uint64]providerHistoryCollectionStateRootCalls

func (s historyStateRootTimeline) add(state, roots, copies uint64) {
	v := s[state]
	v.roots, v.copies = v.roots+roots, v.copies+copies
	s[state] = v
}

func (s historyStateRootTimeline) prefix(prior uint64) {
	s.add(0, 2, 1) // Original initial ledger root and cursor-copy root.
	for k := uint64(0); k < prior; k++ {
		s.add(k, 2, 0)   // T valid/advance.
		s.add(k, 2, 0)   // C valid/advance.
		s.add(k, 2, 1)   // S valid and original before-state copy.
		s.add(k+1, 3, 0) // Boundary root, ledger advance and S validation.
	}
	s.add(prior, 1, 0) // Final complete-prefix validation.
}

func historyStateRootStage(prior, members uint64, advance bool) historyStateRootTimeline {
	s := historyStateRootTimeline{}
	for range 12 {
		s.prefix(prior)
	}
	for _, row := range [][2]uint64{{6, 0}, {9, 3}, {24, 6}, {22, 4}, {12 * members, 2 * members}, {36, 12}} {
		s.add(prior, row[0], row[1]) // Planning/T/C/dispatch/member/S-before.
	}
	s.add(prior+1, 9, 0) // Three original S after-state branches.
	if advance {
		s.prefix(prior + 1)
		s.prefix(prior + 1)
		s.add(prior+1, 2, 0) // BOTH group models before this SAME window closes.
	}
	return s
}

func (s historyStateRootTimeline) interval(first, last uint64) providerHistoryCollectionStateRootCalls {
	var out providerHistoryCollectionStateRootCalls
	for k, v := range s {
		if first <= k && k <= last {
			out.roots, out.copies = out.roots+v.roots, out.copies+v.copies
		}
	}
	return out
}

func TestProviderHistoryStateRootWeightsPreserveChronologyAndShorterPrograms(t *testing.T) {
	for _, members := range [][]uint64{{1}, {32}, {1, 32}, {32, 1}, {1, 1, 32}, {32, 1, 32, 1}} {
		whole := historyStateRootTimeline{}
		whole.add(0, 4, 2) // External initial models, OUTSIDE firstPReading.
		var requests uint64
		for j, count := range members {
			prior, advance := uint64(j), j+1 < len(members)
			s := historyStateRootStage(prior, count, advance)
			for first := uint64(0); first <= prior+1; first++ {
				for last := first; last <= prior+1; last++ {
					want := s.interval(first, last)
					if got, ok := providerHistoryCollectionStageStateCallsBound(prior, count, advance, first, last); !ok || got != want {
						t.Fatal("state placement differs from original source-call timeline", members, prior, first, last, got, want)
					}
				}
			}
			coarse, ok := providerHistoryCollectionStageRootCallsBound(prior, count, advance)
			if v := s.interval(0, prior+1); !ok || v.roots != coarse.cursorRoots || v.copies != coarse.cursorCopies {
				t.Fatal("refinement changed original same-window totals")
			}
			for state, v := range s {
				whole.add(state, v.roots, v.copies)
			}
			requests += count
		}
		groups := uint64(len(members))
		coarse, ok := providerHistoryCollectionRootCallsBound(groups, requests)
		if v := whole.interval(0, groups); !ok || v.roots != coarse.cursorRoots || v.copies != coarse.cursorCopies {
			t.Fatal("external initialization omitted or firstReader constituents added twice")
		}
		// Every shorter admitted program is allowed. Larger maxG does not
		// force nonexistent groups to consume one request apiece.
		for maximumGroups := groups; maximumGroups <= groups+3; maximumGroups++ {
			maximumRequests := max(requests, maximumGroups)
			for first := uint64(0); first <= maximumGroups; first++ {
				for last := first; last <= maximumGroups; last++ {
					v := whole.interval(first, last)
					upper, valid := providerHistoryCollectionWholeStateCallsBound(maximumGroups, maximumRequests, first, last)
					if !valid || v.roots > upper.roots || v.copies > upper.copies {
						t.Fatal("shorter/differently packed original program escaped range", members, maximumGroups, first, last, v, upper)
					}
				}
			}
		}
	}
}

func TestProviderHistoryStateRootNormalKeepsOriginalObserverAndFullCopy(t *testing.T) {
	for _, content := range []bool{false, true} {
		var c *providerHistoryCollectionCursor
		if content {
			c = historyCollectionCursorCostPending(t)
		} else {
			c, _, _ = historyCollectionBoundaryFixture(t, 13, 4)
		}
		l := historyCollectionCursorCostLimits(t, c)
		history, kinds := uint64(0), providerHistoryAppliedStateKinds{ConfirmedEmpty: !content}
		if content {
			history = uint64(c.selection.input.journal.Replay.History.MaxBytes)
		} else {
			l.Effects, l.PendingBlocks, l.Semantic = 0, 0, providerHistorySemanticCommitmentLimits{}
		}
		calls, ok := providerHistoryCollectionPrefixStateCallsBound(2, 0, 0)
		upper, valid := calls.normalUpper(l, history, kinds)
		one, child := providerHistoryAppliedCursorUpper(l, history, kinds)
		if !ok || !valid || !child || calls != (providerHistoryCollectionStateRootCalls{8, 2}) {
			t.Fatal("independent8roots/2copies source interval")
		}
		for coordinate, v := range upper {
			if v != 6*one.normalCommitment[coordinate]+2*one.normalCopy[coordinate] {
				t.Fatal("copy-initial root counted twice or a work coordinate omitted", coordinate)
			}
		}
		original := c.semantic.observer
		before, root := original.actual, c.commitmentObserved(providerHistorySemanticUnitMeter())
		// Repeated original primitive DATA at one fixed source state checks
		// the conditional component, not execution of a complete prefix/T/S.
		for range 6 {
			if got := c.commitmentObserved(original); got == "" || got != root {
				t.Fatal("original root changed during conditional interval")
			}
		}
		for range 2 {
			copy := c.copyObserved(original)
			if copy == nil || copy == c || copy.semantic == c.semantic || copy.semantic.observer != original || copy.commitmentObserved(providerHistorySemanticUnitMeter()) != root {
				t.Fatal("original full owned copy changed state or observer")
			}
		}
		if original.failed || c.semantic.observer != original {
			t.Fatal("numeric weight transferred or replaced original execution ownership")
		}
		for coordinate, actual := range original.actual {
			if actual < before[coordinate] || actual-before[coordinate] > upper[coordinate] {
				t.Fatal("same original component exceeded entered-work row", coordinate)
			}
		}
	}
}

func TestProviderHistoryStateRootWeightsRejectInvalidAndOverflowingParts(t *testing.T) {
	for _, call := range []func() (providerHistoryCollectionStateRootCalls, bool){
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionPrefixStateCallsBound(349526, 0, 0)
		},
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionPrefixStateCallsBound(1, 1, 0)
		},
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionPrefixStateCallsBound(1, 0, 2)
		},
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionStageStateCallsBound(0, 0, false, 0, 1)
		},
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionStageStateCallsBound(0, 33, false, 0, 1)
		},
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionStageStateCallsBound(349525, 1, false, 0, 1)
		},
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionStageStateCallsBound(349524, 1, true, 0, 1)
		},
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionStageStateCallsBound(0, 1, false, 1, 0)
		},
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionStageStateCallsBound(0, 1, false, 0, 2)
		},
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionWholeStateCallsBound(0, 1, 0, 0)
		},
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionWholeStateCallsBound(2, 1, 0, 2)
		},
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionWholeStateCallsBound(1, 33, 0, 1)
		},
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionWholeStateCallsBound(1, 1, 1, 0)
		},
		func() (providerHistoryCollectionStateRootCalls, bool) {
			return providerHistoryCollectionWholeStateCallsBound(1, 1, 0, 2)
		},
	} {
		if v, ok := call(); ok || v != (providerHistoryCollectionStateRootCalls{}) {
			t.Fatal("invalid source interval exposed partial multiplicities", v)
		}
	}
	l := providerHistoryCollectionCursorLimits{Spine: 1, Transactions: 1, EvidenceBytes: 1, BodyBytes: 1}
	for _, calls := range []providerHistoryCollectionStateRootCalls{{}, {1, 2}} {
		if v, ok := calls.normalUpper(l, 0, providerHistoryAppliedStateKinds{ConfirmedEmpty: true}); ok || v != (candidateWorkVector{}) {
			t.Fatal("invalid numeric root/copy tuple exposed partial work")
		}
	}
	l.EvidenceBytes = 0
	if v, ok := (providerHistoryCollectionStateRootCalls{1, 0}).normalUpper(l, 0, providerHistoryAppliedStateKinds{}); ok || v != (candidateWorkVector{}) {
		t.Fatal("invalid original cursor child exposed partial work")
	}
	calls, ok := providerHistoryCollectionWholeStateCallsBound(349525, 11184800, 0, 349525)
	l = providerHistoryCollectionCursorLimits{Spine: 1 << 20, Effects: 163840, PendingBlocks: 10, Transactions: 16384, EvidenceBytes: 64 << 20, BodyBytes: 8 << 20}
	l.Semantic = providerHistorySemanticCommitmentLimits{Window: true, Queue: 1, Pending: true}
	child, valid := providerHistoryAppliedCursorUpper(l, 8<<20, providerHistoryAppliedStateKinds{})
	if !valid || child.normalCommitment[2] == 0 {
		t.Fatal("late-overflow precursor must first price nonzero decoded work")
	}
	if v, valid := calls.normalUpper(l, 8<<20, providerHistoryAppliedStateKinds{}); !ok || valid || v != (candidateWorkVector{}) {
		t.Fatal("late logical overflow exposed earlier nonzero work coordinates")
	}
}
