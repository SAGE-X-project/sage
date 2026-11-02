package registry011

import "reflect"

// Prospective original selections, not reported actuals or comparison weights.
// The complete pre-H0 factory must derive directoryBytes from BOTH original
// directory strings and bind the source ceilings below. Legacy paths acquire
// no new directory limit merely because this separate candidate quote exists.
type providerHistoryMemberContextInput struct {
	groups, requests, directoryBytes uint64
}

type providerHistoryMemberContextCost struct {
	// Slots: fixed context, C frame, read context, claim, ancestor value, fact.
	weights, walkCalls, equalityCalls [6]uint64
	sealCopies, claimCopies           uint64
	ancestorRows, ancestorFacts       uint64
	ancestorArrays, memberArrays      uint64
	walks, copies, comparisons        candidateWorkVector
	modelHeads, total                 candidateWorkVector
}

// SAME-C typed metadata only. Pointer contents are NOT traversed by equality:
// explicit claims and complete ancestor/fact snapshots remain separate rows.
// Actual originals already bound policy to64KiB, journal to9929, manifest and
// inventory to8KiB, request wire to16896, other metadata to1MiB and hashes/names
// to smaller ceilings. Fact.response/tls are hashes, NOT retained raw payloads.
// Only the two directory strings lack an existing byte cap; their aggregate
// must be selected and checked by the future complete factory before H0.
// Fixed type sizes here are logical copy/compare arguments, not native CPU,
// backing/peak/lifetime RAM or an all-eleven program certificate.
func providerHistoryMemberContextUpper(in providerHistoryMemberContextInput) (providerHistoryMemberContextCost, bool) {
	var empty providerHistoryMemberContextCost
	maximum, ok := providerUpperProduct(in.groups, providerHistoryGroupMembers)
	if !ok || in.groups == 0 || in.groups > (1<<20)/3 || in.requests < in.groups || in.requests > maximum ||
		in.directoryBytes < 2 || in.directoryBytes > 2<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryMemberContextCost
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

	out.sealCopies = in.groups
	out.claimCopies = a.sum(a.product(4, in.groups), a.product(8, in.requests))
	out.ancestorRows = a.product(3, a.sum(in.groups, tri))
	out.ancestorFacts = prior
	out.ancestorArrays, out.memberArrays = in.groups, tri
	out.walkCalls = [6]uint64{in.groups, in.groups, a.product(2, in.groups), a.product(2, in.groups), out.ancestorRows, prior}
	out.equalityCalls = [6]uint64{
		a.sum(in.groups, a.product(4, in.requests)),
		a.sum(in.groups, a.product(2, in.requests)),
		a.sum(a.product(2, in.groups), a.product(4, in.requests)),
		a.sum(a.product(2, in.groups), a.product(8, in.requests)),
		a.sum(out.ancestorRows, a.product(12, a.sum(in.requests, prior))),
		a.sum(prior, a.product(2, cross)),
	}
	// The current closed type shapes have227/44/44/83/140/28 string
	// leaves. Fixed/read contexts each contain BOTH directory strings.
	const metadata uint64 = 1 << 20
	strings := [6]uint64{
		a.sum(a.product(225, metadata), in.directoryBytes), 44 * metadata,
		a.sum(a.product(42, metadata), in.directoryBytes), 83 * metadata,
		140 * metadata, 28 * metadata,
	}
	sizes := [6]uint64{
		uint64(reflect.TypeFor[providerHistoryMemberReplayContext]().Size()),
		uint64(reflect.TypeFor[providerHistoryCustodyFrame]().Size()),
		uint64(reflect.TypeFor[providerHistoryMemberReadContext]().Size()),
		uint64(reflect.TypeFor[providerHistoryMemberClaimInput]().Size()),
		uint64(reflect.TypeFor[providerHistoryMemberAncestorValue]().Size()),
		uint64(reflect.TypeFor[providerHistoryMemberAncestorFact]().Size()),
	}
	// One compare(0) per visited node in the actual finite reflection walk.
	nodes := [6]uint64{1007, 73, 110, 196, 396, 122}
	for i := range out.weights {
		out.weights[i] = a.sum(sizes[i], strings[i])
		out.walks[10] = a.sum(out.walks[10], a.product(out.walkCalls[i], nodes[i]))
		out.comparisons[10] = a.sum(out.comparisons[10], a.product(out.equalityCalls[i], a.sum(a.product(2, out.weights[i]), 1)))
	}
	// Each original chain-count pass adds one compare(0) per ancestor.
	out.walks[10] = a.sum(out.walks[10], out.ancestorRows)
	out.copies[10] = a.sum(
		a.product(out.sealCopies, a.sum(uint64(reflect.TypeFor[providerHistoryMemberReplaySeal]().Size()), 1)),
		a.product(out.claimCopies, a.sum(sizes[3], 1)),
		a.product(out.ancestorRows, uint64(reflect.TypeFor[providerHistoryMemberAncestor]().Size())), out.ancestorArrays,
		a.product(prior, a.sum(sizes[5], 8)), out.memberArrays,
	)
	// These two CURRENT C model-head comparisons per role/member were
	// explicitly excluded from the retained-read row's532G+798R.
	out.modelHeads[10] = a.product(532, in.requests)
	out.total[10] = a.sum(out.walks[10], out.copies[10], out.comparisons[10], out.modelHeads[10])
	if a.failed {
		return empty, false
	}
	return out, true
}
