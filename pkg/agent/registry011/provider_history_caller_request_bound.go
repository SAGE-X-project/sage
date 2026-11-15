package registry011

type providerHistoryCallerRequestCost struct {
	fiveCalls, sevenCalls uint64
	five, seven, total    candidateWorkVector
}

// Successful direct caller comparisons, outside request reconstruction,
// ticket validation, whole-table equality, typed metadata and filename work.
// G/R are original prospective maxima. The result is coordinate-10 DATA,
// never authority or a sufficient all-eleven program reservation.
func providerHistoryCallerRequestUpper(groups, requests uint64) (providerHistoryCallerRequestCost, bool) {
	var empty providerHistoryCallerRequestCost
	maximum, ok := providerUpperProduct(groups, providerHistoryGroupMembers)
	if !ok || groups == 0 || groups > (1<<20)/3 || requests < groups || requests > maximum {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryCallerRequestCost
	tri := a.product(groups, groups-1) / 2
	extra := a.difference(requests, groups)
	full, tail := extra/31, extra%31
	prior := a.sum(tri, a.product(31, a.difference(a.product(full, groups), a.product(full, a.sum(full, 1))/2)))
	if tail != 0 {
		prior = a.sum(prior, a.product(tail, a.difference(a.difference(groups, full), 1)))
	}
	// Original intent matching occurs before send and before member end.
	// Its TWO expected filename comparisons are priced separately, not here.
	out.fiveCalls = a.product(2, requests)
	// One selected request comparison in preparation, plus one sealed request
	// comparison per prior archived member in each full healthy S replay.
	out.sevenCalls = a.sum(requests, a.product(14, prior))
	// method32/params8192/head512/body16384/wire16896. The original copy
	// validator caps input.wire at16384, but a reconstructed body+head may
	// reach16896 before equality. Using16896 on BOTH sides is conservative.
	// Every field emits its own compare event with its extra1. Hashes are66.
	five := a.sum(2*32+1, 2*8192+1, 2*512+1, 2*16384+1, 2*16896+1)
	seven := a.sum(five, 2*133)
	out.five[10] = a.product(out.fiveCalls, five)
	out.seven[10] = a.product(out.sevenCalls, seven)
	out.total[10] = a.sum(out.five[10], out.seven[10])
	if a.failed {
		return empty, false
	}
	return out, true
}
