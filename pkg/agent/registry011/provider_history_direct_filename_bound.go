package registry011

type providerHistoryDirectFilenameCost struct {
	initialModels, priorStages, priorMembers uint64
	currentNames, nameComparisons            uint64
	initial, admission, consumption          candidateWorkVector
	dispatch, settlement, prior              candidateWorkVector
	member, originals, comparisons, total    candidateWorkVector
}

// Successful direct read/write/getter filename events, outside allowed-map
// construction. G/R are original prospective maxima, not observed smaller
// counts. This logical-only row grants no owner, write, ACK or complete quote.
// Legacy first-P names, catalog namespaces, recorder copies and native string
// formatting/allocations are distinct work and must not be added again here.
func providerHistoryDirectFilenameUpper(groups, requests uint64) (providerHistoryDirectFilenameCost, bool) {
	var empty providerHistoryDirectFilenameCost
	maximum, ok := providerUpperProduct(groups, providerHistoryGroupMembers)
	if !ok || groups == 0 || groups > (1<<20)/3 || requests < groups || requests > maximum {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryDirectFilenameCost
	tri := a.product(groups, groups-1) / 2
	extra := a.difference(requests, groups)
	full, tail := extra/31, extra%31
	weighted := a.sum(tri, a.product(31, a.difference(a.product(full, groups), a.product(full, a.sum(full, 1))/2)))
	if tail != 0 {
		weighted = a.sum(weighted, a.product(tail, a.difference(a.difference(groups, full), 1)))
	}
	// Fewer optional groups can increase CURRENT spool-prefix work. Use
	// packed32-member groups, not exactly G nonempty observed groups.
	maximumSquare := a.sum(a.product(requests/32, 1024), a.product(requests%32, requests%32))
	out.initialModels = a.product(14, groups) // 2init+12Ghealthy+2(G-1)advance.
	out.priorStages = a.product(14, tri)
	out.priorMembers = a.product(14, weighted)
	out.currentNames = a.sum(a.product(4, maximumSquare), a.product(14, requests))
	out.nameComparisons = a.product(4, requests)
	// Initial model: highwater31/frontier30/commit28. Its first-P names
	// belong to the legacy component, and the final advance reads HEAD literal.
	out.initial[10] = a.product(89, out.initialModels)
	// Current T data444, C data360, C authentication185. Physical writers
	// T512/C428/S515 include BOTH HEAD.pending construction and rename.
	out.admission[10] = a.product(1400, groups)   // 2*T444+writer512.
	out.consumption[10] = a.product(2542, groups) // 3*T444+2*C360+2*outer31+writer428.
	out.dispatch[10] = a.product(1793, groups)    // 2*(T444+C360)+seal185.
	out.settlement[10] = a.product(3759, groups)  // 3*(T444+C360)+2*Ssuffix416+writer515.
	// Each prior T/C/S: indexed3*154+stored447+inventory249=1158.
	// Each archived response/TLS name pays copy+equality <=344, twice/member.
	// Chain-count events are separate from these direct filename operations.
	out.prior[10] = a.sum(a.product(1158, out.priorStages), a.product(688, out.priorMembers))
	out.member[10] = a.product(370, requests) // Two current-C authentication calls.
	// Four square-prefix reader calls,6R S-reader names, plus8R actual
	// prepare/match/write names. A current name is <=114 bytes; copy adds1.
	out.originals[10] = a.product(115, out.currentNames)
	// Two memberAttemptDataMatches calls compare TWO expected names each.
	// Its five other request-field comparisons are not priced by this row.
	out.comparisons[10] = a.product(229, out.nameComparisons)
	out.total[10] = a.sum(out.initial[10], out.admission[10], out.consumption[10], out.dispatch[10], out.settlement[10],
		out.prior[10], out.member[10], out.originals[10], out.comparisons[10])
	if a.failed {
		return empty, false
	}
	return out, true
}
