package registry011

import "reflect"

// Successful retained replay/context DATA at the SAME original role cuts.
// No source, seal, descriptor, observer or allowance is installed. FirstRead
// already funds its own optional recording; only the outside subset is added
// here. Native reflection/maps/backing/RAM, changed-shape failure/recovery,
// all other raw/codecs and full feasibility/factory work remain separate.
type providerHistoryRetainedContextRoleCost struct {
	capacity, maximumRows, rows, outsideRows uint64
	priorMembers, ancestors                  uint64
	weights                                  [6]uint64
	beforeWalk, beforeEqual, memberEqual     [6]uint64
	recording, refresh, commitments          providerHistoryRoleControlParts
	comparisons, context, joined             providerHistoryRoleControlParts
}

// The manifest and explicit legacy/directory/catalog selections must still
// bind the genuine original source before H0. This quote checks the original
// firstRead sublimit and recorder capacity without enlarging either. P is its
// actual already-entered preparation; full M contains it. The extra P in the
// unchanged remaining-work composite is not removed or refunded.
func providerHistoryCollectionRetainedContextRoleUpper(manifest *providerHistoryCollectionManifest, limits providerHistoryLegacyControlLimits, catalogEntries, directoryBytes, prior uint64) (providerHistoryRetainedContextRoleCost, bool) {
	var empty providerHistoryRetainedContextRoleCost
	if !manifest.validSnapshot() || prior >= manifest.counts.groups || manifest.counts.requests == 0 ||
		catalogEntries == 0 || catalogEntries > 1<<20 || directoryBytes < 2 || directoryBytes > 2<<20 ||
		limits.firstObjects != manifest.input.firstRead.objects || limits.firstRaw != manifest.input.firstRead.raw {
		return empty, false
	}
	first, firstOK := providerHistoryFirstPReadUpper(limits, manifest.input.capture)
	// ONLY the invariant six type-shape weights are used from this child,
	// never its already repeated whole-program totals or source authority.
	shape, shapeOK := providerHistoryMemberContextUpper(providerHistoryMemberContextInput{groups: 1, requests: 1, directoryBytes: directoryBytes})
	if !firstOK || !shapeOK {
		return empty, false
	}
	for c, units := range first.work {
		if units > manifest.input.firstRead.work[c] {
			return empty, false
		}
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryRetainedContextRoleCost
	out.weights = shape.weights
	objects := min(limits.firstObjects, a.product(4, limits.entries))
	// Every current C has at least one current member. Its original prior
	// S archives therefore contain at most R-1 members, even at the last cut.
	priorMembers := func(q uint64) uint64 { return min(a.product(32, q), manifest.counts.requests-1) }
	rowCount := func(q uint64) uint64 {
		return a.sum(33, objects, a.product(23, q), a.product(2, priorMembers(q)))
	}
	out.capacity = a.sum(catalogEntries, limits.firstObjects, 128)
	out.maximumRows = rowCount(manifest.counts.groups - 1)
	out.rows, out.priorMembers = rowCount(prior), priorMembers(prior)
	if a.failed || out.maximumRows > out.capacity {
		return empty, false
	}
	out.outsideRows = a.difference(out.rows, a.sum(objects, 9))
	const h uint64 = candidateLedgerMetadataLimit
	d := max(limits.metadata, h)
	l := a.sum(uint64(manifest.input.capture.WireBytes), 64<<10)
	// Outside firstReading: initial model4+final HEAD, prior T/C/S and
	// archived spools, then current T10/C9 (one P pack plus18 D controls).
	outsideBytes := a.sum(a.product(5, d), a.product(prior, a.sum(a.product(21, d), a.product(2, limits.objectBytes))),
		a.product(out.priorMembers, l), limits.objectBytes, a.product(18, d))
	recordedOutside := a.product(2, outsideBytes)
	recordedRows := a.product(2, out.outsideRows)
	constructors := a.product(2, uint64(reflect.TypeFor[providerHistoryReplayReads]().Size())+1)
	recording := a.sum(constructors, recordedOutside, a.product(642, recordedRows))
	// File I/O for both fresh raw transcripts is already in the file role
	// component. These are its NEW SHA+digest-comparison events only.
	refreshedBytes := a.product(2, a.sum(outsideBytes, limits.firstRaw))
	refresh := a.sum(refreshedBytes, a.product(134, a.product(2, out.rows)))
	// B: two recorded roots and two seal checks. P: each role checks before
	// and after its raw refresh. All four retain the COMPLETE H+19 rows,
	// including firstReading; its rolling roots are outside that sublimit.
	base := h*h + 1 + h + uint64(len("sage-provider-history-replay-reads-v1")) + 2
	row := h*h + 1 + h + uint64(len("sage-provider-history-replay-read-v1")) + 2
	commitments := a.product(4, a.sum(base, a.product(out.rows, row)))
	out.ancestors = a.product(3, a.sum(prior, 1)) // initial + current T/C + 3 per prior S.
	out.beforeWalk = [6]uint64{1, 1, 2, 2, out.ancestors, out.priorMembers}
	out.beforeEqual = out.beforeWalk
	out.memberEqual = [6]uint64{4, 2, 4, 8, a.product(4, out.ancestors), a.product(4, out.priorMembers)}
	nodes := [6]uint64{1007, 73, 110, 196, 396, 122}
	var before, member uint64
	for i, weight := range out.weights {
		equality := a.sum(a.product(2, weight), 1)
		before = a.sum(before, a.product(out.beforeWalk[i], nodes[i]), a.product(out.beforeEqual[i], equality))
		member = a.sum(member, a.product(out.memberEqual[i], equality))
	}
	claim := uint64(reflect.TypeFor[providerHistoryMemberClaimInput]().Size()) + 1
	seal := uint64(reflect.TypeFor[providerHistoryMemberReplaySeal]().Size()) + 1
	ancestor := uint64(reflect.TypeFor[providerHistoryMemberAncestor]().Size())
	fact := uint64(reflect.TypeFor[providerHistoryMemberAncestorFact]().Size()) + 8
	// B captures the chain once: one count pass, one ancestor-array copy,
	// prior archive fact/weight arrays and four typed claim copies. P uses
	// eight NEW claim copies but never reseals or repeats the length walks.
	before = a.sum(before, out.ancestors, seal, a.product(4, claim), a.product(out.ancestors, ancestor), 1,
		a.product(out.priorMembers, fact), prior)
	member = a.sum(member, a.product(8, claim), 532) // Four current-C model-head comparisons.
	parts := func(b, p uint64) providerHistoryRoleControlParts {
		return providerHistoryRoleControlParts{
			before:      providerHistoryStageWorkCost{work: candidateWorkVector{10: b}},
			preparation: providerHistoryStageWorkCost{work: candidateWorkVector{10: p}},
			member:      providerHistoryStageWorkCost{work: candidateWorkVector{10: p}},
		}
	}
	out.recording, out.refresh = parts(recording, 0), parts(0, refresh)
	out.commitments, out.comparisons, out.context = parts(commitments, commitments), parts(532, 798), parts(before, member)
	var joinedOK bool
	out.joined, joinedOK = providerHistoryJoinRoleComponentParts(out.recording, out.refresh, out.commitments, out.comparisons, out.context)
	if a.failed || !joinedOK {
		return empty, false
	}
	return out, true
}
