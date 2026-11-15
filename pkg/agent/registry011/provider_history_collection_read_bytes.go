package registry011

// Original pre-H0 numeric maxima only. Their actual store/manifest/profile
// qualification remains a production compiler obligation; no owner or budget
// may be constructed from these freely supplied DATA scalars.
type providerHistoryCollectionReadByteInput struct {
	groups, requests, firstObjects, attempts, chunks uint64
	firstRaw, metadata, object, wire                 uint64
}

type providerHistoryCollectionReadByteCost struct {
	counts                                                          providerHistoryCollectionSuccessfulReadCost
	initial, healthy, memberReplay, preparation, retention, advance uint64
	catalog, remaining, bytes                                       uint64
	work                                                            candidateWorkVector
}

// Content reads and their explicit read-event units only, initialization
// through final S. Codecs, metadata/native calls, clocks, writes, resident
// memory and failure/recovery/final/page work are separate components.
func providerHistoryCollectionSuccessfulReadByteUpper(in providerHistoryCollectionReadByteInput) (providerHistoryCollectionReadByteCost, bool) {
	var empty providerHistoryCollectionReadByteCost
	counts, ok := providerHistoryCollectionSuccessfulReadUpper(in.groups, in.requests, in.firstObjects, in.attempts, in.chunks)
	if !ok || in.firstRaw == 0 || in.firstRaw > 64<<20 || in.metadata == 0 || in.metadata > ^uint64(0)>>1 ||
		in.object == 0 || in.object > providerHistoryOriginalPackBytes || in.wire == 0 || in.wire > 16<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	// D covers preflight read maxima as well as successful small-object codec
	// caps. Using only actual 8KiB bytes could underfund a larger last preflight.
	d := max(in.metadata, uint64(candidateLedgerMetadataLimit))
	u := a.sum(a.product(5, d), in.firstRaw)
	v := a.sum(a.product(21, d), a.product(2, in.object))
	l := a.sum(in.wire, 64<<10)
	tri := a.product(in.groups, a.difference(in.groups, 1)) / 2
	extra := a.difference(in.requests, in.groups)
	full, tail := extra/31, extra%31
	leading := a.difference(a.product(full, in.groups), a.product(full, a.sum(full, 1))/2)
	weighted := a.sum(tri, a.product(31, leading))
	if tail != 0 {
		weighted = a.sum(weighted, a.product(tail, a.difference(a.difference(in.groups, full), 1)))
	}
	sumH := a.sum(a.product(in.groups, u), a.product(v, tri), a.product(l, weighted))
	q, r := in.requests/in.groups, in.requests%in.groups
	minimumSquare := a.sum(a.product(in.groups, a.product(q, q)), a.product(r, a.sum(a.product(2, q), 1)))
	// R^2-sum(m^2) is twice sum(i<j)m_i*m_j, hence even. Balanced
	// minimumSquare has the same parity as R, so this division is exact.
	cross := a.difference(a.product(in.requests, in.requests), minimumSquare) / 2
	memberH := a.sum(a.product(u, in.requests), a.product(v, weighted), a.product(l, cross))
	packedSquare := a.sum(a.product(in.requests/32, 1024), a.product(in.requests%32, in.requests%32))
	initial := a.product(2, a.difference(u, d))
	healthy := a.product(12, sumH)
	memberReplay := a.sum(a.product(2, memberH), a.product(a.product(2, in.requests), in.object), a.product(a.product(52, d), in.requests))
	preparation := a.product(l, a.difference(packedSquare, in.requests))
	retention := a.product(l, a.sum(packedSquare, in.requests))
	advance := a.product(2, a.difference(sumH, u))
	// All scan control/receipt reads use metadata maximum; each of the four
	// legacy catalog inventory reads instead has a 64KiB preflight bound.
	perCatalog := a.sum(a.product(d, a.sum(6, a.product(5, in.attempts), a.product(4, in.chunks))), 4*(64<<10))
	catalog := a.product(perCatalog, counts.catalog.calls)
	remaining := a.sum(a.product(in.groups, a.sum(a.product(12, in.object), a.product(185, d))),
		a.product(a.product(3, in.requests), l), a.product(a.difference(in.groups, 1), d))
	bytes := a.sum(initial, healthy, memberReplay, preparation, retention, advance, catalog, remaining)
	units := a.sum(bytes, counts.contentReads)
	if a.failed {
		return empty, false
	}
	var work candidateWorkVector
	work[10] = units
	return providerHistoryCollectionReadByteCost{counts, initial, healthy, memberReplay, preparation, retention, advance, catalog, remaining, bytes, work}, true
}
