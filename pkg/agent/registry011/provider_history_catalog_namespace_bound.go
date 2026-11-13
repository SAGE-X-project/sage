package registry011

// Original pre-H0 limits, not observed directory lengths or producer costs.
// A complete factory must bind these to the selected immutable source limits.
type providerHistoryCatalogNamespaceInput struct {
	groups, requests, setupRecoveries, attempts, entries uint64
}

type providerHistoryCatalogNamespaceCost struct {
	catalogs, prefixStages, prefixMembers uint64
	currentPairs, afterCopies             uint64
	currentNames, pendingNames            uint64
	initialPerCatalog, copyPerCatalog     uint64
	initial, prior, extensions            candidateWorkVector
	preparation, copies, current, pending candidateWorkVector
	total                                 candidateWorkVector
}

// Successful allowed-set construction only: actual filename copies, prior
// chain-count comparisons and copying both allowed maps. It does not reprice
// scan/receipt/inventory/ref names already in the legacy catalog component,
// directory enumeration, read-recorder copies or native map hashing/CPU/RAM.
// Direct read/write/getter names outside catalog construction remain separate.
func providerHistoryCatalogNamespaceUpper(in providerHistoryCatalogNamespaceInput) (providerHistoryCatalogNamespaceCost, bool) {
	var empty providerHistoryCatalogNamespaceCost
	maximum, ok := providerUpperProduct(in.groups, providerHistoryGroupMembers)
	if !ok || in.groups == 0 || in.groups > (1<<20)/3 || in.requests < in.groups || in.requests > maximum ||
		in.setupRecoveries > 64 || in.attempts == 0 || in.attempts > 64 || !providerHistoryCatalogEntriesSupported(in.entries) {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryCatalogNamespaceCost
	tri := a.product(in.groups, in.groups-1) / 2
	extra := a.difference(in.requests, in.groups)
	full, tail := extra/31, extra%31
	prior := a.sum(tri, a.product(31, a.difference(a.product(full, in.groups), a.product(full, a.sum(full, 1))/2)))
	if tail != 0 {
		prior = a.sum(prior, a.product(tail, a.difference(a.difference(in.groups, full), 1)))
	}
	balanced, remainder := in.requests/in.groups, in.requests%in.groups
	minimumSquare := a.sum(a.product(in.groups, a.product(balanced, balanced)), a.product(remainder, a.sum(a.product(2, balanced), 1)))
	cross := a.difference(a.product(in.requests, in.requests), minimumSquare)
	// Fewer optional groups can have a larger CURRENT member prefix. Pack
	// groups of32 for this row instead of requiring exactly G nonempty groups.
	maximumSquare := a.sum(a.product(in.requests/32, 1024), a.product(in.requests%32, in.requests%32))
	out.catalogs = a.difference(a.sum(a.product(16, in.requests), a.product(16, in.groups)), 2)
	out.prefixStages = a.product(16, a.sum(prior, tri))
	out.prefixMembers = a.sum(a.product(8, cross), a.product(16, prior))
	// Two dispatch catalogs, member16m-2, and eleven S catalogs extend
	// both original T/C claims. C preparation additionally extends T once.
	out.currentPairs = a.sum(a.product(16, in.requests), a.product(11, in.groups))
	// T1/C1 and S8: S preparation, two reader pairs, generic pre/post and
	// outer completion. Each copies BOTH maps once, not once per element.
	out.afterCopies = a.product(10, in.groups)
	out.currentNames = a.sum(a.product(16, maximumSquare), a.product(21, in.requests))
	out.pendingNames = a.product(5, in.groups) // T1/C1/S3 before rename only.

	// At the original bridge sequence=4, ordinal-sequence<=recoveries and
	// setups<=recoveries+1. Thus H<=5+S, setup rows<=1+S. Initial indexed
	// names cost31H+101*(1+S)+170A+532 <=788+132S+170A.
	out.initialPerCatalog = a.sum(788, a.product(132, in.setupRecoveries), a.product(170, in.attempts))
	out.initial[10] = a.product(out.catalogs, out.initialPerCatalog)
	// One prior T/C/S has five stored names447, three inventory names249,
	// fifteen indexed names462 and three compare(1) events6 =>1164.
	// Each archived response/TLS name costs copy+equality<=344, twice/member.
	out.prior[10] = a.sum(a.product(1164, out.prefixStages), a.product(688, out.prefixMembers))
	out.extensions[10] = a.sum(a.product(742, out.currentPairs), a.product(413, in.groups))
	out.preparation[10] = a.product(4070, in.groups) // T413+C329+8*S416.
	// Each derived root map admits <=entries names of <=255 bytes. Charge
	// two roots and each actual copy's +1 without trusting a claimed byte sum.
	out.copyPerCatalog = a.product(512, in.entries)
	out.copies[10] = a.product(out.afterCopies, out.copyPerCatalog)
	out.current[10] = a.product(115, out.currentNames)
	out.pending[10] = a.product(34, out.pendingNames)
	out.total[10] = a.sum(out.initial[10], out.prior[10], out.extensions[10], out.preparation[10], out.copies[10], out.current[10], out.pending[10])
	if a.failed {
		return empty, false
	}
	return out, true
}
