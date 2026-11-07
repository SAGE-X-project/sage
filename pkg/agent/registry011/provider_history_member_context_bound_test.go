package registry011

import (
	"context"
	"reflect"
	"testing"
)

func TestProviderHistoryMemberContextUpperMatchesIndependentLiterals(t *testing.T) {
	for _, tc := range []struct {
		input                              providerHistoryMemberContextInput
		walk, equal                        [6]uint64
		walkWork, copyBase, comparisonBase uint64
		claimCopies, ancestorRows, facts   uint64
		memberArrays, heads, totalBase     uint64
	}{
		{providerHistoryMemberContextInput{1, 1, 2}, [6]uint64{1, 1, 2, 2, 3, 0}, [6]uint64{5, 3, 6, 10, 15, 0}, 2917, 14, 9309257811, 12, 3, 0, 0, 532, 9309261274},
		{providerHistoryMemberContextInput{1, 32, 2048}, [6]uint64{1, 1, 2, 2, 3, 0}, [6]uint64{129, 65, 130, 258, 387, 0}, 2917, 262, 236851311561, 260, 3, 0, 0, 17024, 236851331764},
		{providerHistoryMemberContextInput{2, 3, 2048}, [6]uint64{2, 2, 4, 4, 9, 2}, [6]uint64{14, 8, 16, 28, 69, 10}, 7269, 53, 34473107601, 32, 9, 2, 1, 1596, 34473116519},
		{providerHistoryMemberContextInput{2, 4, 2048}, [6]uint64{2, 2, 4, 4, 9, 3}, [6]uint64{18, 10, 20, 36, 93, 19}, 7391, 69, 45864870084, 40, 9, 3, 1, 2128, 45864879672},
		{providerHistoryMemberContextInput{3, 96, 2048}, [6]uint64{3, 3, 6, 6, 18, 96}, [6]uint64{387, 195, 390, 774, 2322, 12384}, 24036, 1557, 1778616684612, 780, 18, 96, 3, 51072, 1778616761277},
	} {
		got, ok := providerHistoryMemberContextUpper(tc.input)
		sizes := [6]uint64{
			uint64(reflect.TypeFor[providerHistoryMemberReplayContext]().Size()), uint64(reflect.TypeFor[providerHistoryCustodyFrame]().Size()),
			uint64(reflect.TypeFor[providerHistoryMemberReadContext]().Size()), uint64(reflect.TypeFor[providerHistoryMemberClaimInput]().Size()),
			uint64(reflect.TypeFor[providerHistoryMemberAncestorValue]().Size()), uint64(reflect.TypeFor[providerHistoryMemberAncestorFact]().Size()),
		}
		strings := [6]uint64{235929600 + tc.input.directoryBytes, 46137344, 44040192 + tc.input.directoryBytes, 87031808, 146800640, 29360128}
		var compareSizes uint64
		for i := range sizes {
			if got.weights[i] != sizes[i]+strings[i] {
				t.Fatal("typed original string envelope", i, got.weights[i], sizes[i], strings[i])
			}
			compareSizes += 2 * tc.equal[i] * sizes[i]
		}
		copySizes := tc.input.groups*uint64(reflect.TypeFor[providerHistoryMemberReplaySeal]().Size()) + tc.claimCopies*sizes[3] +
			tc.ancestorRows*uint64(reflect.TypeFor[providerHistoryMemberAncestor]().Size()) + tc.facts*sizes[5]
		if !ok || got.walkCalls != tc.walk || got.equalityCalls != tc.equal || got.sealCopies != tc.input.groups ||
			got.claimCopies != tc.claimCopies || got.ancestorRows != tc.ancestorRows || got.ancestorFacts != tc.facts ||
			got.ancestorArrays != tc.input.groups || got.memberArrays != tc.memberArrays ||
			got.walks != (candidateWorkVector{10: tc.walkWork}) || got.copies != (candidateWorkVector{10: tc.copyBase + copySizes}) ||
			got.comparisons != (candidateWorkVector{10: tc.comparisonBase + compareSizes}) || got.modelHeads != (candidateWorkVector{10: tc.heads}) ||
			got.total != (candidateWorkVector{10: tc.totalBase + copySizes + compareSizes}) {
			t.Fatal("independent original topology/string/copy literals", tc, got, ok)
		}
	}
}

// Independent type-only inspection follows structs/arrays, never pointer
// children. It deliberately fails on a new kind instead of assuming that a
// changed source shape still has the former finite cost.
func historyMemberContextTypeShape(t *testing.T, typ reflect.Type, depth int) (uint64, uint64, int) {
	t.Helper()
	nodes, leaves, maximum := uint64(1), uint64(0), depth
	switch typ.Kind() {
	case reflect.String:
		leaves = 1
	case reflect.Struct:
		for i := range typ.NumField() {
			n, s, d := historyMemberContextTypeShape(t, typ.Field(i).Type, depth+1)
			nodes, leaves, maximum = nodes+n, leaves+s, max(maximum, d)
		}
	case reflect.Array:
		for range typ.Len() {
			n, s, d := historyMemberContextTypeShape(t, typ.Elem(), depth+1)
			nodes, leaves, maximum = nodes+n, leaves+s, max(maximum, d)
		}
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Pointer:
	default:
		t.Fatal("new kind needs its own original bound", typ)
	}
	return nodes, leaves, maximum
}

func TestProviderHistoryMemberContextTypeCeilingsMatchActualWalks(t *testing.T) {
	for _, tc := range []struct {
		value         any
		nodes, leaves uint64
		depth         int
	}{
		{providerHistoryMemberReplayContext{}, 1041, 227, 6}, {providerHistoryCustodyFrame{}, 73, 44, 4},
		{providerHistoryMemberReadContext{}, 110, 44, 4}, {providerHistoryMemberClaimInput{}, 196, 83, 5},
		{providerHistoryMemberAncestorValue{}, 396, 140, 6}, {providerHistoryMemberAncestorFact{}, 122, 28, 4},
	} {
		nodes, leaves, depth := historyMemberContextTypeShape(t, reflect.TypeOf(tc.value), 0)
		m, remaining := providerReplayUnitMeter(), uint64(65536)
		weight, ok := providerHistoryMemberStringWeightObserved(reflect.ValueOf(tc.value), m, 0, &remaining)
		if nodes != tc.nodes || leaves != tc.leaves || depth != tc.depth || !ok || weight != 0 ||
			remaining != 65536-tc.nodes || m.actual != (candidateWorkVector{10: tc.nodes}) || m.failed {
			t.Fatal("closed type/string shape or actual per-node walk changed", reflect.TypeOf(tc.value), nodes, leaves, depth, m.actual)
		}
	}
}

func historyMemberContextOriginalDATA[T comparable](t *testing.T, value T, originalWeight, maximum, nodes uint64) {
	t.Helper()
	m := providerReplayUnitMeter() // DATA inspection; never an execution owner.
	weight, ok := providerHistoryMemberComparisonWeightObserved(value, m)
	if !ok || weight != originalWeight || weight > maximum || m.actual != (candidateWorkVector{10: nodes}) ||
		!providerHistoryMemberEqualObserved(value, value, weight, m) || m.actual != (candidateWorkVector{10: nodes + 2*weight + 1}) || m.failed {
		t.Fatal("actual original typed snapshot/weight/debit", reflect.TypeFor[T](), weight, originalWeight, maximum, m.actual)
	}
}

func TestProviderHistoryMemberContextBoundCoversGenuineClaimsAndArchivedFacts(t *testing.T) {
	o := historyMultistageOwner(t, 3)
	for stage := uint64(1); stage <= 2; stage++ {
		out := historyMultistageSettle(t, o, stage)
		if err := o.advanceStageCandidate(context.Background(), out.ack); err != nil {
			t.Fatal("genuine prior complete archived stage", stage, err)
		}
	}
	d := historyMultistageDispatch(t, o, 3)
	q, before, clock := d.memberReplaySeal, o.meter.actual, o.store.clock.read
	if q == nil || len(q.ancestors) != 9 || q.revoked || !d.memberReplaySelectedLocked() {
		t.Fatal("complete third-stage original SAME-C precursor")
	}
	directory := uint64(len(q.fixed.config.custodyDirectory) + len(q.fixed.config.evidenceDirectory))
	upper, ok := providerHistoryMemberContextUpper(providerHistoryMemberContextInput{3, 3, directory})
	if !ok {
		t.Fatal("original directories must fit the separate prospective selection")
	}
	historyMemberContextOriginalDATA(t, q.fixed, q.fixedComparison, upper.weights[0], 1041)
	historyMemberContextOriginalDATA(t, q.consumption.frame, q.frameComparison, upper.weights[1], 73)
	for role := range 2 {
		historyMemberContextOriginalDATA(t, q.readContexts[role], q.readComparisons[role], upper.weights[2], 110)
		historyMemberContextOriginalDATA(t, q.claims[role], q.claimComparisons[role], upper.weights[3], 196)
	}
	facts := 0
	for _, row := range q.ancestors {
		historyMemberContextOriginalDATA(t, row.value, row.comparison, upper.weights[4], 396)
		for j, fact := range row.members {
			if !lowerHash(fact.original.response) || !lowerHash(fact.original.tls) || !lowerHash(fact.capture.response) || !lowerHash(fact.capture.tls) {
				t.Fatal("archive stores original hashes, not response payload strings")
			}
			historyMemberContextOriginalDATA(t, fact, row.weights[j], upper.weights[5], 122)
			facts++
		}
	}
	if facts != 2 || o.meter.actual != before || o.store.clock.read != clock || o.meter.failed || q.revoked {
		t.Fatal("DATA inspection changed original execution or omitted archived children", facts)
	}
}

func TestProviderHistoryMemberContextCountsDominateIndependentSmallTopologies(t *testing.T) {
	// Count actual nested roles/boundaries for all1..3 groups with1..4
	// members, plus the32-member boundary. This uses no prefix formula.
	var sizes [3]uint64
	var visit func(int, int)
	visit = func(index, groups int) {
		if index < groups {
			for _, size := range []uint64{1, 2, 3, 4, 32} {
				sizes[index] = size
				visit(index+1, groups)
			}
			return
		}
		var walk, equal [6]uint64
		var requests, prior uint64
		for group := range groups {
			members, ancestors := sizes[group], 3*uint64(group+1)
			requests += members
			walk[0]++
			walk[1]++
			walk[2] += 2
			walk[3] += 2
			walk[4] += ancestors
			walk[5] += prior
			equal[0]++
			equal[1]++
			equal[2] += 2
			equal[3] += 2
			equal[4] += ancestors
			equal[5] += prior
			for range members {
				for range 2 { // Each independent original role.
					equal[1]++    // CURRENT C frame.
					for range 2 { // Before AND after the fresh read pass.
						equal[0]++
						equal[2]++
						equal[3] += 2 // Original T and C claims.
						equal[4] += ancestors
						equal[5] += prior
					}
				}
			}
			prior += members
		}
		upper, ok := providerHistoryMemberContextUpper(providerHistoryMemberContextInput{uint64(groups), requests, 2})
		if !ok {
			t.Fatal("valid finite small topology", groups, sizes, requests)
		}
		for i := range walk {
			if upper.walkCalls[i] < walk[i] || upper.equalityCalls[i] < equal[i] {
				t.Fatal("source multiplicity underfunds actual role/ancestor/member boundaries", groups, sizes, i, upper, walk, equal)
			}
		}
	}
	for groups := 1; groups <= len(sizes); groups++ {
		visit(0, groups)
	}
}

func TestProviderHistoryMemberContextShortageKeepsOriginalWalkPrefix(t *testing.T) {
	o, ack := historyDispatchReplayFixture(t, 1)
	d, err := o.prepareDispatchCandidate(context.Background(), ack)
	if err != nil || d == nil || d.memberReplaySeal == nil {
		t.Fatal("genuine original context", err)
	}
	q, before := d.memberReplaySeal, o.meter.actual
	weight := q.fixedComparison
	m := providerHistoryContentUnitMeter(candidateWorkVector{10: 1041 + 2*weight})
	got, ok := providerHistoryMemberComparisonWeightObserved(q.fixed, m)
	if !ok || got != weight || m.actual != (candidateWorkVector{10: 1041}) || m.failed {
		t.Fatal("real finite walk must precede its insufficient equality event")
	}
	if providerHistoryMemberEqualObserved(q.fixed, q.fixed, got, m) || !m.failed || m.actual != (candidateWorkVector{10: 1041}) {
		t.Fatal("one-short DATA equality spent or refunded its earlier walk")
	}
	if _, ok := providerHistoryMemberComparisonWeightObserved(q.fixed, m); ok ||
		providerHistoryMemberEqualObserved(q.fixed, q.fixed, got, m) || m.actual != (candidateWorkVector{10: 1041}) ||
		o.meter.actual != before || q.revoked || !d.memberReplaySelectedLocked() {
		t.Fatal("spent DATA observer retried or altered its original live owner")
	}
}

func TestProviderHistoryMemberContextUpperRejectsInvalidOriginalDomains(t *testing.T) {
	for _, in := range []providerHistoryMemberContextInput{
		{0, 1, 2}, {(1<<20)/3 + 1, (1<<20)/3 + 1, 2}, {^uint64(0), 1, 2},
		{1, 0, 2}, {2, 1, 2}, {1, 33, 2}, {1, ^uint64(0), 2},
		{1, 1, 0}, {1, 1, 1}, {1, 1, (2 << 20) + 1}, {1, 1, ^uint64(0)},
	} {
		if got, ok := providerHistoryMemberContextUpper(in); ok || got != (providerHistoryMemberContextCost{}) {
			t.Fatal("invalid prospective inputs returned a partial quote", in, got, ok)
		}
	}
}

func TestProviderHistoryMemberContextUpperLateOverflowReturnsWholeZero(t *testing.T) {
	// All scalar domains/counts and individual weights are valid. Repeated
	// equality bytes, not an invalid scalar, overflow the complete work row.
	input := providerHistoryMemberContextInput{20000, 640000, 2 << 20}
	if got, ok := providerHistoryMemberContextUpper(input); ok || got != (providerHistoryMemberContextCost{}) {
		t.Fatal("late repeated typed comparison overflow escaped as partial cost", got, ok)
	}
	if got, ok := providerHistoryMemberContextUpper(providerHistoryMemberContextInput{1, 1, 2 << 20}); !ok || got.total[10] == 0 {
		t.Fatal("valid directory-selection boundary was rejected", got, ok)
	}
}
