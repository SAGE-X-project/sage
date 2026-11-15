package registry011

// Successful original collection call topology, not authentication or a full
// cost certificate. A copy already consumes one cursor commitment. Separate
// ledger encoding, semantic transitions, native work, RAM, failure/recovery
// and final publication remain outside this component.
type providerHistoryCollectionRootBound struct {
	initialModels, prefixes, boundaries    uint64
	ledgerRoots, cursorRoots, cursorCopies uint64
}

// Prospective G/R must cover the original nonempty groups/requests. The rows
// include both initialization models, all full prior-prefix reconstructions,
// both same-C member passes, and advancement only before a later group. No
// caller-supplied root count or claimed actual supplies these multiplicities.
func providerHistoryCollectionRootCallsBound(groups, requests uint64) (providerHistoryCollectionRootBound, bool) {
	var empty providerHistoryCollectionRootBound
	maximum, ok := providerUpperProduct(groups, providerHistoryGroupMembers)
	if !ok || groups == 0 || groups > (1<<20)/3 || requests < groups || requests > maximum {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	tri := a.product(groups, groups-1) / 2
	initial := a.product(14, groups)
	prefixes := a.difference(initial, 2)
	boundaries := a.sum(a.product(3, groups), a.product(14, tri))
	// One fresh prefix after j successes has 2+7j ledger roots,
	// 1+j cursor copies and j additional boundary commitments: 3+9j
	// cursor roots in total. Twelve prefixes per group plus two before
	// each later group preserve all repeated interpretation work.
	ledger := a.sum(a.difference(a.product(108, groups), 4), a.product(98, tri), a.product(10, requests))
	roots := a.sum(a.difference(a.product(150, groups), 4), a.product(126, tri), a.product(12, requests))
	copies := a.sum(a.product(39, groups), a.product(14, tri), a.product(2, requests))
	if a.failed {
		return empty, false
	}
	return providerHistoryCollectionRootBound{initial, prefixes, boundaries, ledger, roots, copies}, true
}

// The selected limits must bound EVERY original cursor state, including both
// confirmed/working streams and optional pending work, not today's reported
// lengths. This composes only cursor 2/10 work; success neither binds those
// limits nor proves the chosen complete history fits its original reservation.
func providerHistoryCollectionRepeatedCursorUpper(groups, requests uint64, limits providerHistoryCollectionCursorLimits) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	calls, callsOK := providerHistoryCollectionRootCallsBound(groups, requests)
	one, oneOK := providerHistoryCollectionCursorUpper(limits)
	if !callsOK || !oneOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	// fullCopy already includes its initial root. Charging every root and
	// every fullCopy independently would count that root twice.
	standalone := a.difference(calls.cursorRoots, calls.cursorCopies)
	var work candidateWorkVector
	for coordinate := range work {
		work[coordinate] = a.sum(a.product(standalone, one.commitment[coordinate]), a.product(calls.cursorCopies, one.copy[coordinate]))
	}
	if a.failed {
		return empty, false
	}
	return work, true
}

// One original healthy T/C/member/S window, excluding the two program-wide
// initialization models. Its optional advancement is AFTER this S and remains
// inside this SAME window. That advancement reconstructs prior+1 settled
// groups; assigning the before-next-T prefix to the next window would underfund
// the original owner's clock/work scope. These are call counts, not authority.
func providerHistoryCollectionStageRootCallsBound(prior, members uint64, advance bool) (providerHistoryCollectionRootBound, bool) {
	var empty providerHistoryCollectionRootBound
	const maximumGroups uint64 = (1 << 20) / 3
	if prior >= maximumGroups || members == 0 || members > providerHistoryGroupMembers || advance && prior+1 >= maximumGroups {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	// Twelve original prefixes at depth prior. Their current-window work
	// adds 78+10m ledger roots, 106+12m cursor roots, 25+2m copies and
	// three current-S boundary commitments. Each prefix itself has
	// 2+7prior ledger roots, 3+9prior cursor roots and 1+prior copies.
	models, prefixes := uint64(12), uint64(12)
	boundaries := a.sum(3, a.product(12, prior))
	ledger := a.sum(102, a.product(10, members), a.product(84, prior))
	roots := a.sum(142, a.product(12, members), a.product(108, prior))
	copies := a.sum(37, a.product(2, members), a.product(12, prior))
	if advance {
		// Two complete prefixes at depth prior+1, then both retained
		// original models' final validObserved calls. No new hold opens.
		models, prefixes = 14, 14
		boundaries = a.sum(boundaries, a.product(2, a.sum(prior, 1)))
		ledger = a.sum(ledger, 20, a.product(14, prior))
		roots = a.sum(roots, 26, a.product(18, prior))
		copies = a.sum(copies, 4, a.product(2, prior))
	}
	if a.failed {
		return empty, false
	}
	return providerHistoryCollectionRootBound{models, prefixes, boundaries, ledger, roots, copies}, true
}

// A common maximum for the existing pre-H0 admission.maximum. Derive the
// last-window and nonfinal-window rows separately, then take componentwise
// maxima. This is NOT a difference/average of whole-program upper bounds.
// Fewer realized optional groups and fewer members only reduce each direct
// row. The two initial models are disjoint and must still be funded once.
func providerHistoryCollectionUniformRootCallsBound(groups, requests uint64) (providerHistoryCollectionRootBound, bool) {
	var empty providerHistoryCollectionRootBound
	if _, ok := providerHistoryCollectionRootCallsBound(groups, requests); !ok {
		return empty, false
	}
	last, ok := providerHistoryCollectionStageRootCallsBound(groups-1, providerHistoryGroupMembers, false)
	if !ok {
		return empty, false
	}
	if groups == 1 {
		return last, true
	}
	beforeLast, ok := providerHistoryCollectionStageRootCallsBound(groups-2, providerHistoryGroupMembers, true)
	if !ok {
		return empty, false
	}
	return providerHistoryCollectionRootBound{
		max(last.initialModels, beforeLast.initialModels), max(last.prefixes, beforeLast.prefixes),
		max(last.boundaries, beforeLast.boundaries), max(last.ledgerRoots, beforeLast.ledgerRoots),
		max(last.cursorRoots, beforeLast.cursorRoots), max(last.cursorCopies, beforeLast.cursorCopies),
	}, true
}

// Cursor 2/10 subrow of that SAME uniform original stage window. Copy already
// includes one commitment, so charge roots-copies standalone roots. This row
// includes the existing per-call preflight ceiling; it is not entered actual,
// the complete all-coordinate maximum or a native/RAM/factory certificate.
func providerHistoryCollectionUniformCursorUpper(groups, requests uint64, limits providerHistoryCollectionCursorLimits) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	calls, callsOK := providerHistoryCollectionUniformRootCallsBound(groups, requests)
	one, oneOK := providerHistoryCollectionCursorUpper(limits)
	if !callsOK || !oneOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	standalone := a.difference(calls.cursorRoots, calls.cursorCopies)
	var work candidateWorkVector
	for coordinate := range work {
		work[coordinate] = a.sum(a.product(standalone, one.commitment[coordinate]), a.product(calls.cursorCopies, one.copy[coordinate]))
	}
	if a.failed {
		return empty, false
	}
	return work, true
}

// Successful source-state placements only. State k is the original cursor
// after k successful S boundaries, not the ledger's T/C phase. Copies already
// include one root. Counts authenticate no state, role, source or factory.
type providerHistoryCollectionStateRootCalls struct{ roots, copies uint64 }

func providerHistoryCollectionPrefixStateCallsBound(prior, first, last uint64) (providerHistoryCollectionStateRootCalls, bool) {
	var empty providerHistoryCollectionStateRootCalls
	if prior > (1<<20)/3 || first > last || last > prior {
		return empty, false
	}
	if prior == 0 {
		return providerHistoryCollectionStateRootCalls{3, 1}, true
	}
	var out providerHistoryCollectionStateRootCalls
	var a providerHistoryReadArithmetic
	if first == 0 {
		out.roots, out.copies = 8, 2
	}
	left, right := max(first, uint64(1)), min(last, prior-1)
	if left <= right {
		count := a.sum(a.difference(right, left), 1)
		out.roots = a.sum(out.roots, a.product(9, count))
		out.copies = a.sum(out.copies, count)
	}
	if last == prior {
		out.roots = a.sum(out.roots, 4)
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// One SAME original T window: twelve prefixes at prior, then direct before-S
// and after-S roots. Optional advancement is still inside this window. The
// external initial two models are outside this row AND outside firstPReading;
// a future complete initializer must assign their4roots/2copies exactly once.
func providerHistoryCollectionStageStateCallsBound(prior, members uint64, advance bool, first, last uint64) (providerHistoryCollectionStateRootCalls, bool) {
	var empty providerHistoryCollectionStateRootCalls
	if _, ok := providerHistoryCollectionStageRootCallsBound(prior, members, advance); !ok || first > last || last > prior+1 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryCollectionStateRootCalls
	if first <= prior {
		prefix, ok := providerHistoryCollectionPrefixStateCallsBound(prior, first, min(last, prior))
		if !ok {
			return empty, false
		}
		out.roots, out.copies = a.product(12, prefix.roots), a.product(12, prefix.copies)
	}
	if first <= prior && prior <= last {
		out.roots = a.sum(out.roots, 97, a.product(12, members))
		out.copies = a.sum(out.copies, 25, a.product(2, members))
	}
	if last == prior+1 {
		out.roots = a.sum(out.roots, 9)
	}
	if advance {
		prefix, ok := providerHistoryCollectionPrefixStateCallsBound(prior+1, first, last)
		if !ok {
			return empty, false
		}
		out.roots = a.sum(out.roots, a.product(2, prefix.roots))
		out.copies = a.sum(out.copies, a.product(2, prefix.copies))
		if last == prior+1 {
			out.roots = a.sum(out.roots, 2)
		}
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Prospective whole successful topology restricted to a cursor-state interval.
// Includes the external initial models; do not add them again to this whole
// row. G/R are maxima, so fewer actual groups can pack MORE members into this
// interval. Keep min(R,32*memberStates), never subtract mandatory members in
// nonexistent groups. Separate intervals' request maxima need not coexist;
// their sum is conservative, not exact resource feasibility or a new quota.
func providerHistoryCollectionWholeStateCallsBound(groups, requests, first, last uint64) (providerHistoryCollectionStateRootCalls, bool) {
	var empty providerHistoryCollectionStateRootCalls
	if _, ok := providerHistoryCollectionRootCallsBound(groups, requests); !ok || first > last || last > groups {
		return empty, false
	}
	var out providerHistoryCollectionStateRootCalls
	var a providerHistoryReadArithmetic
	if first == 0 {
		out.roots, out.copies = a.sum(a.product(112, groups), 25), a.sum(a.product(28, groups), 11)
	}
	left, right := max(first, uint64(1)), min(last, groups-1)
	if left <= right {
		count := a.sum(a.difference(right, left), 1)
		indices := a.product(count, a.sum(left, right)) / 2
		distance := a.difference(a.product(count, groups-1), indices)
		out.roots = a.sum(out.roots, a.product(164, count), a.product(126, distance))
		out.copies = a.sum(out.copies, a.product(25, count), a.product(14, distance))
	}
	if last == groups {
		out.roots = a.sum(out.roots, 9)
	}
	if first < groups {
		states := a.sum(a.difference(min(last, groups-1), first), 1)
		members := min(requests, a.product(providerHistoryGroupMembers, states))
		out.roots = a.sum(out.roots, a.product(12, members))
		out.copies = a.sum(out.copies, a.product(2, members))
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Conditional successful entered-work part for this source-derived interval.
// The SAME selected limits/kinds must bound EVERY actual state in the interval.
// Normal is NOT an allowance for original preflight/deferred-failure work;
// complete stage/future all11 headroom/native/RAM/factory proof remains open.
func (c providerHistoryCollectionStateRootCalls) normalUpper(l providerHistoryCollectionCursorLimits, historyBytes uint64, kinds providerHistoryAppliedStateKinds) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	if c.roots == 0 || c.copies > c.roots {
		return empty, false
	}
	one, ok := providerHistoryAppliedCursorUpper(l, historyBytes, kinds)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	rootOnly := a.difference(c.roots, c.copies)
	var out candidateWorkVector
	for coordinate := range out {
		out[coordinate] = a.sum(a.product(rootOnly, one.normalCommitment[coordinate]), a.product(c.copies, one.normalCopy[coordinate]))
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// A completed known transport-failure custody path still computes successful
// roots with the SAME healthy original observer. Its three S readers retain
// the before cursor, despite incrementing the terminal ledger stage. This
// counts that path only, not a refused root or an arbitrary early abort.
// The common interval domain includes prior+1, whose failed-path count is zero.
func providerHistoryCollectionFailedStageStateCallsBound(prior, members, first, last uint64) (providerHistoryCollectionStateRootCalls, bool) {
	var empty providerHistoryCollectionStateRootCalls
	if _, ok := providerHistoryCollectionStageRootCallsBound(prior, members, false); !ok || first > last || last > prior+1 {
		return empty, false
	}
	var out providerHistoryCollectionStateRootCalls
	var a providerHistoryReadArithmetic
	if first <= prior {
		prefix, ok := providerHistoryCollectionPrefixStateCallsBound(prior, first, min(last, prior))
		if !ok {
			return empty, false
		}
		out.roots, out.copies = a.product(12, prefix.roots), a.product(12, prefix.copies)
	}
	if first <= prior && prior <= last {
		out.roots = a.sum(out.roots, 100, a.product(12, members))
		out.copies = a.sum(out.copies, 22, a.product(2, members))
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Bound a successful program OR one ending with completed known-failure S.
// Relative to the same-length successful grammar, only the last S differs:
// +3 before roots, -3 before copies, -9 after roots. Keep the removals reserved
// and add three roots if this interval can contain the failed before state.
// Shorter programs and independently maximal request bands remain supported.
// This prices normal root/copy work, not failing/deferred primitives, incoming
// headroom, capture/storage/native/RAM, original factory or operating authority.
func providerHistoryCollectionTerminalStateCallsBound(groups, requests, first, last uint64) (providerHistoryCollectionStateRootCalls, bool) {
	out, ok := providerHistoryCollectionWholeStateCallsBound(groups, requests, first, last)
	if !ok {
		return providerHistoryCollectionStateRootCalls{}, false
	}
	if first < groups {
		var a providerHistoryReadArithmetic
		out.roots = a.sum(out.roots, 3)
		if a.failed {
			return providerHistoryCollectionStateRootCalls{}, false
		}
	}
	return out, true
}

// Actual root/copy COMPONENT parts for ONE fixed original accounting scope.
// They bind no source/state/role and are neither a complete T upper nor a
// future reservation. Nonroot prefixes and other composite incoming rows must
// be priced separately. Original first-false closure is required for this
// component; a completed known transport failure has successful cursor calls.
type providerHistoryCursorPointwiseParts struct{ normal, headroom candidateWorkVector }

func (c providerHistoryCollectionStateRootCalls) pointwisePartsUpper(l providerHistoryCollectionCursorLimits, historyBytes uint64, kinds providerHistoryAppliedStateKinds) (providerHistoryCursorPointwiseParts, bool) {
	var empty providerHistoryCursorPointwiseParts
	if c.roots == 0 || c.copies > c.roots {
		return empty, false
	}
	one, ok := providerHistoryAppliedCursorUpper(l, historyBytes, kinds)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	rootOnly := a.difference(c.roots, c.copies)
	var out providerHistoryCursorPointwiseParts
	for i := range out.normal {
		out.normal[i] = a.sum(a.product(rootOnly, one.normalCommitment[i]), a.product(c.copies, one.normalCopy[i]))
		// A normal retained copy can use receipt/body cardinality while its
		// incoming envelope keeps full arrays. The differences need not be
		// equal; preserve the larger first-refusal delta for either call.
		root := a.difference(one.envelopeCommitment[i], one.normalCommitment[i])
		copied := a.difference(one.envelopeCopy[i], one.normalCopy[i])
		out.headroom[i] = max(root, copied)
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Join qualified disjoint call/state bands INSIDE ONE actual-accounting scope.
// Add conditional normal work and keep the largest first-false/incoming delta.
// This MUST NOT replace SUM of complete admitted T uppers in the irrevocable
// future ledger: each full stage upper, including its own headroom, stays
// charged. Initialization and own advance origins also remain separate.
func (p providerHistoryCursorPointwiseParts) joinWithinScope(q providerHistoryCursorPointwiseParts) (providerHistoryCursorPointwiseParts, bool) {
	var out providerHistoryCursorPointwiseParts
	var a providerHistoryReadArithmetic
	for i := range out.normal {
		out.normal[i] = a.sum(p.normal[i], q.normal[i])
		out.headroom[i] = max(p.headroom[i], q.headroom[i])
	}
	if a.failed {
		return providerHistoryCursorPointwiseParts{}, false
	}
	return out, true
}

// Root/copy actual-prefix plus original incoming-primitive allowance ONLY.
// The source-bound nonroot K, all applicable complete window/advance rows and
// initializer+SUM(full admitted uppers) are additional unchanged obligations.
// This numerical result cannot construct the original production derivation.
func (p providerHistoryCursorPointwiseParts) actualAndIncomingUpper() (candidateWorkVector, bool) {
	var out candidateWorkVector
	var a providerHistoryReadArithmetic
	for i := range out {
		out[i] = a.sum(p.normal[i], p.headroom[i])
	}
	if a.failed {
		return candidateWorkVector{}, false
	}
	return out, true
}

// Counts at the original B/P/M/success-S/known-failed-S/own-A cuts, restricted
// to a cursor-state interval. P is already inside M, not another actual
// member. The two external initializer models remain separate at state zero.
// These numeric counts bind no actual source state, observer or authority.
type providerHistoryRoleStateCalls struct {
	before, preparation, member, successClosure, failedClosure, advance providerHistoryCollectionStateRootCalls
}

func providerHistoryCollectionRoleStateCallsBound(prior, first, last uint64, advancing bool) (providerHistoryRoleStateCalls, bool) {
	var empty providerHistoryRoleStateCalls
	const maximumGroups uint64 = (1 << 20) / 3
	if prior >= maximumGroups || advancing && prior+1 >= maximumGroups || first > last || last > prior+1 {
		return empty, false
	}
	var prefix providerHistoryCollectionStateRootCalls
	if first <= prior {
		var ok bool
		prefix, ok = providerHistoryCollectionPrefixStateCallsBound(prior, first, min(last, prior))
		if !ok {
			return empty, false
		}
	}
	var a providerHistoryReadArithmetic
	out := providerHistoryRoleStateCalls{
		before:         providerHistoryCollectionStateRootCalls{a.product(9, prefix.roots), a.product(9, prefix.copies)},
		successClosure: providerHistoryCollectionStateRootCalls{a.product(3, prefix.roots), a.product(3, prefix.copies)},
		failedClosure:  providerHistoryCollectionStateRootCalls{a.product(3, prefix.roots), a.product(3, prefix.copies)},
	}
	if first <= prior && prior <= last {
		out.before.roots, out.before.copies = a.sum(out.before.roots, 61), a.sum(out.before.copies, 13)
		out.preparation = providerHistoryCollectionStateRootCalls{12, 2}
		out.member = out.preparation
		out.successClosure.roots, out.successClosure.copies = a.sum(out.successClosure.roots, 36), a.sum(out.successClosure.copies, 12)
		out.failedClosure.roots, out.failedClosure.copies = a.sum(out.failedClosure.roots, 39), a.sum(out.failedClosure.copies, 9)
	}
	if last == prior+1 {
		out.successClosure.roots = a.sum(out.successClosure.roots, 9)
	}
	if advancing {
		p, ok := providerHistoryCollectionPrefixStateCallsBound(prior+1, first, last)
		if !ok {
			return empty, false
		}
		out.advance = providerHistoryCollectionStateRootCalls{a.product(2, p.roots), a.product(2, p.copies)}
		if last == prior+1 {
			out.advance.roots = a.sum(out.advance.roots, 2)
		}
	}
	if a.failed {
		return empty, false
	}
	return out, true
}

// Ordered, disjoint prospective state bands. Limits and kinds must bound
// EVERY original state in their interval, independently of reported actuals.
// A complete source factory must bind those facts before H0. No such factory
// or permission follows from this nonserializable numeric DATA description.
type providerHistoryRoleStateBand struct {
	first, last  uint64
	limits       providerHistoryCollectionCursorLimits
	historyBytes uint64
	kinds        providerHistoryAppliedStateKinds
}

type providerHistoryRoleCursorParts struct {
	initial, before, preparation, member, closure, advance candidateWorkVector
}

// Cursor-root/copy components at the same cuts as role CONTROL. Each role
// keeps its OWN complete first-refusal/incoming headroom. Join state bands
// only inside that role, never across admitted stages or I and future sums.
// Price successful and known-failed S separately, then take coordinate maxima;
// the failed path has no after-state or own advance. An unrealizable union of
// both paths' counts must not be substituted for either actual source path.
// Other K, boundary application, raw/capture/storage/deferred work, clocks,
// native/backing RAM, full original incoming scopes and factory stay separate.
func providerHistoryCollectionRoleCursorUpper(prior uint64, advancing bool, bands []providerHistoryRoleStateBand) (providerHistoryRoleCursorParts, bool) {
	var empty providerHistoryRoleCursorParts
	const maximumGroups uint64 = (1 << 20) / 3
	if prior >= maximumGroups || advancing && prior+1 >= maximumGroups || len(bands) == 0 || uint64(len(bands)) > prior+2 {
		return empty, false
	}
	var joined [7]providerHistoryCursorPointwiseParts
	var next uint64
	var a providerHistoryReadArithmetic
	for _, band := range bands {
		if band.first != next || band.first > band.last || band.last > prior+1 {
			return empty, false
		}
		roles, ok := providerHistoryCollectionRoleStateCallsBound(prior, band.first, band.last, advancing)
		if !ok {
			return empty, false
		}
		calls := [7]providerHistoryCollectionStateRootCalls{{}, roles.before, roles.preparation, roles.member, roles.successClosure, roles.failedClosure, roles.advance}
		if band.first == 0 {
			// Four external roots include the two copies' initial roots. Their
			// component is returned once; a compiler must add I only ONCE.
			calls[0] = providerHistoryCollectionStateRootCalls{4, 2}
		}
		for index, count := range calls {
			if count.roots == 0 {
				// Proven absent source band, not an untrusted q=0 allowance.
				continue
			}
			part, ok := count.pointwisePartsUpper(band.limits, band.historyBytes, band.kinds)
			if !ok {
				return empty, false
			}
			joined[index], ok = joined[index].joinWithinScope(part)
			if !ok {
				return empty, false
			}
		}
		next = a.sum(band.last, 1)
	}
	if a.failed || next != prior+2 {
		return empty, false
	}
	var values [7]candidateWorkVector
	for index, part := range joined {
		var ok bool
		values[index], ok = part.actualAndIncomingUpper()
		if !ok {
			return empty, false
		}
	}
	closure := values[4]
	for coordinate := range closure {
		closure[coordinate] = max(closure[coordinate], values[5][coordinate])
	}
	return providerHistoryRoleCursorParts{
		initial: values[0], before: values[1], preparation: values[2], member: values[3], closure: closure, advance: values[6],
	}, true
}
