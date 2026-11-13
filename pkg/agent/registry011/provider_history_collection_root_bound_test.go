package registry011

import "testing"

func TestProviderHistoryCollectionRootCallsMatchIndependentRows(t *testing.T) {
	for _, tc := range []struct {
		groups, requests uint64
		want             providerHistoryCollectionRootBound
	}{
		{1, 1, providerHistoryCollectionRootBound{14, 12, 3, 114, 158, 41}},
		{1, 32, providerHistoryCollectionRootBound{14, 12, 3, 424, 530, 103}},
		{2, 2, providerHistoryCollectionRootBound{28, 26, 20, 330, 446, 96}},
		{2, 33, providerHistoryCollectionRootBound{28, 26, 20, 640, 818, 158}},
		{3, 96, providerHistoryCollectionRootBound{42, 40, 51, 1574, 1976, 351}},
		{4, 28, providerHistoryCollectionRootBound{56, 54, 96, 1296, 1688, 296}},
		{100, 3200, providerHistoryCollectionRootBound{1400, 1398, 69600, 527896, 677096, 79600}},
		{349525, 11184800, providerHistoryCollectionRootBound{4893350, 4893348, 855172681275, 5986351025596, 7696731340646, 855207633775}},
	} {
		got, ok := providerHistoryCollectionRootCallsBound(tc.groups, tc.requests)
		if !ok || got != tc.want || got.cursorRoots != got.ledgerRoots+got.cursorCopies+got.boundaries || got.initialModels != got.prefixes+2 {
			t.Fatal("original call topology differs from independent rows", tc, got, ok)
		}
	}
}

// An independent chronological expansion of the original healthy caller
// grammar. This models call multiplicities only; it does not execute a real
// reader, authenticate a transcript or inspect the new closed-form equations.
func historyCollectionEnumeratedRootCalls(members []uint64) providerHistoryCollectionRootBound {
	total, _ := historyCollectionEnumeratedRootStages(members)
	return total
}

func historyCollectionEnumeratedRootStages(members []uint64) (providerHistoryCollectionRootBound, []providerHistoryCollectionRootBound) {
	var out providerHistoryCollectionRootBound
	var stages []providerHistoryCollectionRootBound
	previous := providerHistoryCollectionRootBound{initialModels: 2, ledgerRoots: 2, cursorRoots: 4, cursorCopies: 2}
	finish := func() {
		stages = append(stages, providerHistoryCollectionRootBound{
			out.initialModels - previous.initialModels, out.prefixes - previous.prefixes,
			out.boundaries - previous.boundaries, out.ledgerRoots - previous.ledgerRoots,
			out.cursorRoots - previous.cursorRoots, out.cursorCopies - previous.cursorCopies,
		})
		previous = out
	}
	root := func() {
		out.ledgerRoots++
		out.cursorRoots++
	}
	copyCursor := func() {
		out.cursorRoots++
		out.cursorCopies++
	}
	initial := func() {
		out.initialModels++
		root()
		copyCursor()
	}
	transition := func() {
		root() // valid
		root() // advance
	}
	settle := func() {
		transition()
		copyCursor()
		out.cursorRoots++ // successful boundary's explicit final commitment
		out.boundaries++
		root() // every original successful S caller validates the result
	}
	prefix := func(prior int) {
		out.prefixes++
		initial()
		for range prior {
			transition() // prior T
			transition() // prior C
			settle()     // prior original S, including its post-validation
		}
		root() // complete prefix's final validation
	}
	readTC := func() {
		copyCursor()
		transition()
	}
	admitted := func(prior int) {
		prefix(prior)
		readTC()
		root()
		root()
	}
	consumed := func(prior int) {
		prefix(prior)
		readTC()
		readTC()
		root()
		root()
		root()
	}
	initial()
	initial()
	for group, count := range members {
		if group > 0 {
			prefix(group)
			prefix(group)
			root()
			root()
			finish() // advancement closes the PREVIOUS original T window
		}
		for range 2 { // planning
			prefix(group)
			root()
			root()
			root()
		}
		readTC() // coordinator T
		for range 2 {
			prefix(group)
			readTC()
		}
		admitted(group) // coordinator C starts from fresh committed T
		readTC()
		for range 2 {
			admitted(group)
			readTC()
		}
		for range 2 {
			consumed(group)
			root()
		} // recorded dispatch passes
		root()
		root() // same-C seal validates both models
		for member := uint64(0); member < count; member++ {
			for range 2 {
				root()
				root()
				root() // fresh model/coordinator/custodian checks
				copyCursor()
				root() // owned replay copy is validated after rebuilding ABI maps
			}
			root()
			root() // original dispatch models checked after both roles
		}
		consumed(group) // coordinator S
		copyCursor()
		settle()
		for range 2 {
			consumed(group)
			copyCursor()
			settle()
		}
	}
	if len(members) != 0 {
		finish() // final S has no advancement
	}
	return out, stages
}

func TestProviderHistoryCollectionRootCallsCoverEverySmallerSuccessfulPrefix(t *testing.T) {
	for maximumGroups := uint64(1); maximumGroups <= 4; maximumGroups++ {
		for maximumRequests := maximumGroups; maximumRequests <= maximumGroups+5; maximumRequests++ {
			upper, ok := providerHistoryCollectionRootCallsBound(maximumGroups, maximumRequests)
			if !ok {
				t.Fatal("prospective finite root envelope")
			}
			check := func(members []uint64) {
				got := historyCollectionEnumeratedRootCalls(members)
				for _, pair := range [6][2]uint64{{got.initialModels, upper.initialModels}, {got.prefixes, upper.prefixes}, {got.boundaries, upper.boundaries}, {got.ledgerRoots, upper.ledgerRoots}, {got.cursorRoots, upper.cursorRoots}, {got.cursorCopies, upper.cursorCopies}} {
					if pair[0] > pair[1] {
						t.Fatal("smaller original prefix exceeds a quoted row", members, got, upper)
					}
				}
			}
			check(nil) // initialization may succeed without entering the first T
			var visit func([]uint64, uint64, uint64)
			visit = func(prefix []uint64, groups, remaining uint64) {
				if groups == 0 {
					if remaining == 0 {
						check(prefix)
					}
					return
				}
				for count := uint64(1); count <= 32 && count <= remaining; count++ {
					visit(append(prefix, count), groups-1, remaining-count)
				}
			}
			for groups := uint64(1); groups <= maximumGroups; groups++ {
				for requests := groups; requests <= maximumRequests && requests <= 32*groups; requests++ {
					visit(nil, groups, requests)
				}
			}
		}
	}
	// The same request ceiling permits multiple distributions across two groups.
	upper, ok := providerHistoryCollectionRootCallsBound(2, 33)
	if !ok {
		t.Fatal("partial saturated group envelope")
	}
	for first := uint64(1); first <= 32; first++ {
		if got := historyCollectionEnumeratedRootCalls([]uint64{first, 33 - first}); got != upper {
			t.Fatal("chronological distribution changed these call-only counts", first, got, upper)
		}
	}
}

func TestProviderHistoryCollectionRepeatedCursorCostPreservesCopiesAndRefusals(t *testing.T) {
	l := providerHistoryCollectionCursorLimits{Spine: 8, Effects: 2, PendingBlocks: 2, Transactions: 3, EvidenceBytes: 64 << 20, BodyBytes: 8 << 20,
		Semantic: providerHistorySemanticCommitmentLimits{3, 5, 7, 11, 2, true, true}}
	one, ok := providerHistoryCollectionCursorUpper(l)
	if !ok {
		t.Fatal("separately closed cursor component precursor")
	}
	for _, tc := range []struct{ groups, requests, standalone, copies uint64 }{
		{1, 1, 117, 41}, {2, 33, 660, 158}, {4, 28, 1392, 296},
	} {
		var want candidateWorkVector
		for coordinate := range want {
			want[coordinate] = tc.standalone*one.commitment[coordinate] + tc.copies*one.copy[coordinate]
		}
		got, funded := providerHistoryCollectionRepeatedCursorUpper(tc.groups, tc.requests, l)
		if !funded || got != want || got[2] == 0 || got[10] == 0 {
			t.Fatal("cursor roots were omitted or full-copy roots counted twice", tc, got, want, funded)
		}
	}
	for _, tc := range []struct{ groups, requests uint64 }{
		{0, 1}, {1, 0}, {2, 1}, {1, 33}, {349526, 349526}, {^uint64(0), 1}, {1, ^uint64(0)},
	} {
		if got, valid := providerHistoryCollectionRootCallsBound(tc.groups, tc.requests); valid || got != (providerHistoryCollectionRootBound{}) {
			t.Fatal("invalid call DATA became a partial row", tc, got, valid)
		}
		if got, valid := providerHistoryCollectionRepeatedCursorUpper(tc.groups, tc.requests, l); valid || got != (candidateWorkVector{}) {
			t.Fatal("invalid call DATA became a cursor allowance", tc, got, valid)
		}
	}
	bad := l
	bad.Spine = 0
	if got, valid := providerHistoryCollectionRepeatedCursorUpper(1, 1, bad); valid || got != (candidateWorkVector{}) {
		t.Fatal("invalid child exposed a partial composition", got, valid)
	}
	// Finite call counts are not a claim that every cost product fits uint64.
	// A later factory must refuse the whole quote rather than cap or wrap it.
	if got, valid := providerHistoryCollectionRepeatedCursorUpper(349525, 11184800, l); valid || got != (candidateWorkVector{}) {
		t.Fatal("overflow revived a partial whole-prefix cursor quote", got, valid)
	}
}

func TestProviderHistoryCollectionStageRootsKeepPostSettlementAdvance(t *testing.T) {
	for _, tc := range []struct {
		prior, members uint64
		advance        bool
		want           providerHistoryCollectionRootBound
	}{
		{0, 1, false, providerHistoryCollectionRootBound{12, 12, 3, 112, 154, 39}},
		{0, 32, false, providerHistoryCollectionRootBound{12, 12, 3, 422, 526, 101}},
		{0, 1, true, providerHistoryCollectionRootBound{14, 14, 5, 132, 180, 43}},
		{1, 1, true, providerHistoryCollectionRootBound{14, 14, 19, 230, 306, 57}},
		{5, 32, true, providerHistoryCollectionRootBound{14, 14, 75, 932, 1182, 175}},
	} {
		got, ok := providerHistoryCollectionStageRootCallsBound(tc.prior, tc.members, tc.advance)
		if !ok || got != tc.want || got.initialModels != got.prefixes || got.cursorRoots != got.ledgerRoots+got.cursorCopies+got.boundaries {
			t.Fatal("original window lost or reassigned its post-S advancement", tc, got, ok)
		}
	}
	for _, tc := range []struct {
		prior, members uint64
		advance        bool
	}{
		{0, 0, false}, {0, 33, false}, {349525, 1, false}, {349524, 1, true}, {^uint64(0), 1, true}, {0, ^uint64(0), false},
	} {
		if got, ok := providerHistoryCollectionStageRootCallsBound(tc.prior, tc.members, tc.advance); ok || got != (providerHistoryCollectionRootBound{}) {
			t.Fatal("invalid original stage domain exposed partial counts", tc, got, ok)
		}
	}
	for _, advance := range []bool{false, true} {
		prior := uint64(349524)
		if advance {
			prior--
		}
		if _, ok := providerHistoryCollectionStageRootCallsBound(prior, 32, advance); !ok {
			t.Fatal("last valid original stage domain refused", prior, advance)
		}
	}
}

func TestProviderHistoryCollectionStageRootsMatchChronologicalWindowCuts(t *testing.T) {
	for _, members := range [][]uint64{nil, {1}, {32}, {1, 32}, {32, 1}, {1, 32, 1}, {32, 32, 1, 32}, {1, 1, 1, 1, 1, 1, 32}} {
		total, stages := historyCollectionEnumeratedRootStages(members)
		if len(stages) != len(members) || total != historyCollectionEnumeratedRootCalls(members) {
			t.Fatal("chronological precursor differs", members, total, stages)
		}
		sum := providerHistoryCollectionRootBound{initialModels: 2, ledgerRoots: 2, cursorRoots: 4, cursorCopies: 2}
		for i, count := range members {
			want, ok := providerHistoryCollectionStageRootCallsBound(uint64(i), count, i+1 < len(members))
			if !ok || stages[i] != want {
				t.Fatal("window cut did not keep advance after its own S", members, i, stages[i], want, ok)
			}
			row := stages[i]
			sum.initialModels += row.initialModels
			sum.prefixes += row.prefixes
			sum.boundaries += row.boundaries
			sum.ledgerRoots += row.ledgerRoots
			sum.cursorRoots += row.cursorRoots
			sum.cursorCopies += row.cursorCopies
		}
		if sum != total {
			t.Fatal("initialization or advancement was omitted/double counted", members, sum, total)
		}
		if len(members) > 0 {
			var requests uint64
			for _, count := range members {
				requests += count
			}
			global, ok := providerHistoryCollectionRootCallsBound(uint64(len(members)), requests)
			if !ok || total != global {
				t.Fatal("old whole grammar changed", members, global, total, ok)
			}
		}
	}
}

func TestProviderHistoryCollectionUniformRootsCoverEveryOriginalWindow(t *testing.T) {
	for _, tc := range []struct {
		groups, requests uint64
		want             providerHistoryCollectionRootBound
	}{
		{1, 1, providerHistoryCollectionRootBound{12, 12, 3, 422, 526, 101}},
		{2, 33, providerHistoryCollectionRootBound{14, 14, 15, 506, 634, 113}},
		{3, 96, providerHistoryCollectionRootBound{14, 14, 27, 590, 742, 125}},
		{7, 224, providerHistoryCollectionRootBound{14, 14, 75, 932, 1182, 175}},
		{100, 3200, providerHistoryCollectionRootBound{14, 14, 1377, 10046, 12900, 1477}},
	} {
		got, ok := providerHistoryCollectionUniformRootCallsBound(tc.groups, tc.requests)
		if !ok || got != tc.want {
			t.Fatal("uniform direct-window envelope", tc, got, ok)
		}
	}
	for groups := uint64(1); groups <= 4; groups++ {
		for requests := groups; requests <= groups+5; requests++ {
			upper, ok := providerHistoryCollectionUniformRootCallsBound(groups, requests)
			if !ok {
				t.Fatal("finite uniform precursor")
			}
			var visit func([]uint64, uint64)
			visit = func(members []uint64, left uint64) {
				if len(members) > 0 {
					_, stages := historyCollectionEnumeratedRootStages(members)
					for _, row := range stages {
						for _, pair := range [6][2]uint64{{row.initialModels, upper.initialModels}, {row.prefixes, upper.prefixes}, {row.boundaries, upper.boundaries}, {row.ledgerRoots, upper.ledgerRoots}, {row.cursorRoots, upper.cursorRoots}, {row.cursorCopies, upper.cursorCopies}} {
							if pair[0] > pair[1] {
								t.Fatal("smaller original window exceeds common maximum", groups, requests, members, row, upper)
							}
						}
					}
				}
				if uint64(len(members)) == groups {
					return
				}
				for count := uint64(1); count <= 32 && count <= left; count++ {
					visit(append(members, count), left-count)
				}
			}
			visit(nil, requests)
		}
	}
	for _, tc := range [][2]uint64{{0, 1}, {1, 0}, {2, 1}, {1, 33}, {349526, 349526}, {^uint64(0), 1}, {1, ^uint64(0)}} {
		if got, ok := providerHistoryCollectionUniformRootCallsBound(tc[0], tc[1]); ok || got != (providerHistoryCollectionRootBound{}) {
			t.Fatal("invalid whole domain became a uniform stage row", tc, got, ok)
		}
	}
}

func TestProviderHistoryCollectionUniformCursorKeepsCopyAndOverflowBoundaries(t *testing.T) {
	limits := providerHistoryCollectionCursorLimits{Spine: 8, Effects: 2, PendingBlocks: 2, Transactions: 3, EvidenceBytes: 64 << 20, BodyBytes: 8 << 20,
		Semantic: providerHistorySemanticCommitmentLimits{3, 5, 7, 11, 2, true, true}}
	one, ok := providerHistoryCollectionCursorUpper(limits)
	if !ok {
		t.Fatal("original closed cursor precursor")
	}
	for _, tc := range []struct{ groups, requests, standalone, copies uint64 }{
		{1, 1, 425, 101}, {2, 33, 521, 113}, {7, 224, 1007, 175}, {100, 3200, 11423, 1477},
	} {
		var want candidateWorkVector
		for i := range want {
			want[i] = tc.standalone*one.commitment[i] + tc.copies*one.copy[i]
		}
		got, valid := providerHistoryCollectionUniformCursorUpper(tc.groups, tc.requests, limits)
		if !valid || got != want || got[2] == 0 || got[10] == 0 {
			t.Fatal("uniform cursor omitted roots or charged copy roots twice", tc, got, want, valid)
		}
	}
	for _, tc := range [][2]uint64{{0, 1}, {1, 0}, {1, 33}, {349526, 349526}} {
		if got, valid := providerHistoryCollectionUniformCursorUpper(tc[0], tc[1], limits); valid || got != (candidateWorkVector{}) {
			t.Fatal("invalid whole domain exposed partial cursor quote", tc, got, valid)
		}
	}
	limits.Spine = 0
	if got, valid := providerHistoryCollectionUniformCursorUpper(1, 1, limits); valid || got != (candidateWorkVector{}) {
		t.Fatal("invalid child quote", got, valid)
	}
	limits.Spine = 1 << 20
	if got, valid := providerHistoryCollectionUniformCursorUpper(349525, 11184800, limits); valid || got != (candidateWorkVector{}) {
		t.Fatal("late uniform multiplication overflow exposed partial quote", got, valid)
	}
}
