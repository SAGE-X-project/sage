package registry011

import "testing"

func TestProviderHistoryCallerRequestUpperMatchesIndependentLiterals(t *testing.T) {
	for _, tc := range []struct{ groups, requests, fiveCalls, sevenCalls, five, seven, total uint64 }{
		{1, 1, 2, 1, 168074, 84303, 252377},
		{1, 32, 64, 32, 5378368, 2697696, 8076064},
		{2, 3, 6, 31, 504222, 2613393, 3117615},
		{2, 4, 8, 46, 672296, 3877938, 4550234},
		{2, 33, 66, 481, 5546442, 40549743, 46096185},
		{3, 96, 192, 1440, 16135104, 121396320, 137531424},
	} {
		got, ok := providerHistoryCallerRequestUpper(tc.groups, tc.requests)
		want := providerHistoryCallerRequestCost{tc.fiveCalls, tc.sevenCalls,
			candidateWorkVector{10: tc.five}, candidateWorkVector{10: tc.seven}, candidateWorkVector{10: tc.total}}
		if !ok || got != want {
			t.Fatal("independent complete caller request row", tc, got, want, ok)
		}
	}
}

func TestProviderHistoryCallerRequestFundsEachClosedRequestComparison(t *testing.T) {
	upper, ok := providerHistoryCallerRequestUpper(1, 1)
	if !ok || upper.fiveCalls != 2 || upper.sevenCalls != 1 {
		t.Fatal("one original member quote")
	}
	requests := historyRequestControlMethods(t)
	if len(requests) != 11 {
		t.Fatal("all closed ordinary methods required")
	}
	for _, original := range requests {
		before, copy := *original, *original
		var five uint64
		for _, value := range []string{before.method, before.params, before.head, before.body, before.wire} {
			five += 2*uint64(len(value)) + 1
		}
		// Expected work comes from immutable ordinary request bytes and two
		// independent fixed-length hash fields, not the product quote helper.
		if len(before.inputDigest) != 66 || len(before.paramsDigest) != 66 || five > 84037 {
			t.Fatal("ordinary request precursor escaped the source domain")
		}
		m := providerHistoryContentUnitMeter(upper.seven)
		if !providerHistoryGroupRequestEqualObserved(original, &copy, m) || m.failed || m.closed ||
			m.actual != (candidateWorkVector{10: five + 266}) || m.actual[10] > upper.seven[10] ||
			*original != before || copy != before {
			t.Fatal("actual seven-field observer changed bytes or escaped the disjoint row", before.method, m.actual)
		}
		// The five request-field prefix is the same primitive subgraph used
		// by original member intent matching. It is DATA, not an attempt seal.
		m = providerHistoryContentUnitMeter(candidateWorkVector{10: upper.five[10] / upper.fiveCalls})
		for _, pair := range [][2]string{{before.method, copy.method}, {before.params, copy.params}, {before.head, copy.head},
			{before.body, copy.body}, {before.wire, copy.wire}} {
			if !providerReplayTextEqual(pair[0], pair[1], m) {
				t.Fatal("ordinary five-field prefix refused", before.method)
			}
		}
		if m.actual != (candidateWorkVector{10: five}) || m.failed || *original != before || copy != before {
			t.Fatal("five-field DATA comparison claimed filename work or changed its input", before.method, m.actual)
		}
	}
}

// Enumerate real successful caller classes in chronological order, without
// the quote's triangular/weighted expression. A healthy prior S is visited
// twelve times during later work and twice during nonfinal advance.
func historyCallerRequestSequence(members []uint64) (five, seven uint64) {
	var prior uint64
	for group, count := range members {
		for range 12 {
			seven += prior
		}
		for range count {
			five++  // Before send.
			seven++ // Preparation's independently reconstructed request.
			five++  // Before the original retained-member end.
		}
		prior += count
		if group+1 < len(members) {
			for range 2 {
				seven += prior
			}
		}
	}
	return
}

func TestProviderHistoryCallerRequestUpperCoversSkewedAndOptionalGroups(t *testing.T) {
	maximum, ok := providerHistoryCallerRequestUpper(3, 96)
	if !ok {
		t.Fatal("prospectively selected whole-program maxima")
	}
	var sizes [3]uint64
	var visit func(int, int)
	visit = func(index, groups int) {
		if index < groups {
			for _, count := range []uint64{1, 2, 31, 32} {
				sizes[index] = count
				visit(index+1, groups)
			}
			return
		}
		var requests uint64
		for _, count := range sizes[:groups] {
			requests += count
		}
		got, ok := providerHistoryCallerRequestUpper(uint64(groups), requests)
		five, seven := historyCallerRequestSequence(sizes[:groups])
		if !ok || got.fiveCalls != five || got.sevenCalls < seven || maximum.fiveCalls < five || maximum.sevenCalls < seven {
			t.Fatal("original chronological caller sequence exceeded the quote", sizes[:groups], five, seven, got)
		}
	}
	for groups := 1; groups <= len(sizes); groups++ {
		visit(0, groups)
	}
}

func TestProviderHistoryCallerRequestLastHashShortageKeepsSixFieldPrefix(t *testing.T) {
	request := historyGroupSimple(t)
	before, copy := *request, *request
	var prefix uint64
	for _, value := range []string{before.method, before.params, before.body, before.head, before.wire, before.inputDigest} {
		prefix += 2*uint64(len(value)) + 1
	}
	if len(before.paramsDigest) != 66 || prefix == 0 {
		t.Fatal("ordinary exact-byte request with two hashes required")
	}
	// A separate DATA inspector can compare the first six fields, but is
	// one unit short of the final133-unit params digest comparison.
	m := providerHistoryContentUnitMeter(candidateWorkVector{10: prefix + 132})
	if providerHistoryGroupRequestEqualObserved(request, &copy, m) || !m.failed || m.actual != (candidateWorkVector{10: prefix}) ||
		*request != before || copy != before {
		t.Fatal("last real field shortage refunded or erased earlier request comparisons", m.actual, prefix)
	}
	if providerHistoryGroupRequestEqualObserved(request, &copy, m) || m.actual != (candidateWorkVector{10: prefix}) || *request != before || copy != before {
		t.Fatal("spent DATA observer retried or rewrote original request fields", m.actual)
	}
}

func TestProviderHistoryCallerRequestUpperRejectsInvalidOriginalLimits(t *testing.T) {
	for _, in := range [][2]uint64{{0, 0}, {0, 1}, {^uint64(0), 1}, {(1<<20)/3 + 1, (1<<20)/3 + 1},
		{1, 0}, {2, 1}, {1, 33}, {1, ^uint64(0)}} {
		if got, ok := providerHistoryCallerRequestUpper(in[0], in[1]); ok || got != (providerHistoryCallerRequestCost{}) {
			t.Fatal("invalid original limits yielded a partial caller comparison quote", in, got, ok)
		}
	}
}

func TestProviderHistoryCallerRequestUpperKeepsLargestOriginalDomainFinite(t *testing.T) {
	got, ok := providerHistoryCallerRequestUpper(349525, 11184800)
	want := providerHistoryCallerRequestCost{22369600, 27365503431200,
		candidateWorkVector{10: 1879874075200}, candidateWorkVector{10: 2306994035760453600}, candidateWorkVector{10: 2306995915634528800}}
	// No fixture/input-sized allocation or native feasibility is implied.
	// All valid arithmetic in this component is finite; no artificial valid
	// late-overflow branch is manufactured just to claim refusal coverage.
	if !ok || got != want {
		t.Fatal("independent largest original caller comparison row", got, want, ok)
	}
}
