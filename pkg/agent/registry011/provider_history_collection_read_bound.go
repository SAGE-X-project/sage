package registry011

// Successful-path content-read rows only, not bytes, metadata, CPU, RAM,
// clocks, failure/final work, a full eleven-coordinate upper or authority.
// Planning is a SUBSET of healthyPrefixes. All nested catalog content reads,
// current T/C/S suffix controls and dispatch construction are separate rows.
type providerHistoryCollectionReadBound struct {
	initialModels, planning, healthyPrefixes uint64
	memberReplay                             uint64
	// Spool content only: nested catalog inventory reads are NOT included.
	memberPreparation, memberRetention uint64
	advance                            uint64
}

// Sticky pure arithmetic for cost DATA: no failed intermediate may become a
// partial quote after a later successful operation. Every sum/product and
// subtraction remains checked; this creates no program derivation witness.
type providerHistoryReadArithmetic struct{ failed bool }

func (a *providerHistoryReadArithmetic) keep(value uint64, ok bool) uint64 {
	if a.failed || !ok {
		a.failed = true
		return 0
	}
	return value
}

func (a *providerHistoryReadArithmetic) sum(values ...uint64) uint64 {
	value, ok := providerUpperSum(values...)
	return a.keep(value, ok)
}

func (a *providerHistoryReadArithmetic) product(left, right uint64) uint64 {
	value, ok := providerUpperProduct(left, right)
	return a.keep(value, ok)
}

func (a *providerHistoryReadArithmetic) difference(left, right uint64) uint64 {
	if right > left {
		return a.keep(0, false)
	}
	return a.keep(left-right, true)
}

// G/R/O are prospective maxima for groups, requests and original first-P
// objects, never reported actuals. This O(1) component allocates no input-sized
// array, reads no files or clock and cannot mint a complete cost certificate.
func providerHistoryCollectionPrefixReadBound(groups, requests, firstObjects uint64) (providerHistoryCollectionReadBound, bool) {
	var empty providerHistoryCollectionReadBound
	maximumRequests, ok := providerUpperProduct(groups, providerHistoryGroupMembers)
	if !ok || groups == 0 || groups > (1<<20)/3 || requests < groups || requests > maximumRequests || firstObjects == 0 || firstObjects > 1<<20 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	// H(g)=14+O+sum(j<g)(23+2*m_j). Initial four controls,
	// four first-P inventories/O objects, five original first-P C/S controls,
	// final HEAD. A prior T/C/S contributes8+7+8 and2 archived files/member.
	h0 := a.sum(14, firstObjects)
	tri := a.product(groups, groups-1) / 2
	// Maximize sum((G-j)*m_j): one member per group, then fill31extras
	// earliest. Reversing the sequence gives the same maximum late weight.
	extra := a.difference(requests, groups)
	full, tail := extra/31, extra%31
	qg := a.product(full, groups)
	qt := a.product(full, a.sum(full, 1)) / 2
	weighted := a.sum(tri, a.product(31, a.difference(qg, qt)))
	if tail != 0 {
		weighted = a.sum(weighted, a.product(tail, a.difference(a.difference(groups, full), 1)))
	}
	sumH := a.sum(a.product(h0, groups), a.product(23, tri), a.product(2, weighted))
	// sum m*H <= h0*R +23*lateWeight +R^2-minSumSquares. Independent
	// maxima need not share a sequence; combining upper rows is conservative.
	balanced, remainder := requests/groups, requests%groups
	minimumSquare := a.sum(a.product(groups, a.product(balanced, balanced)), a.product(remainder, a.sum(a.product(2, balanced), 1)))
	requestSquare := a.product(requests, requests)
	memberH := a.sum(a.product(h0, requests), a.product(23, weighted), a.difference(requestSquare, minimumSquare))
	memberReads := a.sum(a.product(2, memberH), a.product(54, requests)) // two current T/C suffixes19 and two C/HEAD anchors8
	// Optional groups may not occur. Fewer realized groups can INCREASE
	// member-local spool prefix work. Pack32-member groups for at-most-G,
	// not exactly G nonempty groups. These rows exclude nested catalogs.
	maximumSquare := a.sum(a.product(requests/32, 1024), a.product(requests%32, requests%32))
	preparation := a.product(2, a.difference(maximumSquare, requests)) // twice before send:4*k spool reads/member
	retention := a.product(2, a.sum(maximumSquare, requests))          // twice after write:4*(k+1), including new member
	initial := a.product(2, a.sum(13, firstObjects))
	planning := a.product(2, sumH)
	healthy := a.product(12, sumH)                  // planning2 + original T2/C3/dispatch2/S3
	advance := a.product(2, a.difference(sumH, h0)) // no final-stage advance
	if a.failed {
		return empty, false
	}
	return providerHistoryCollectionReadBound{initial, planning, healthy, memberReads, preparation, retention, advance}, true
}
