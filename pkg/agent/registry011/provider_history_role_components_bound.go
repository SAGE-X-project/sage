package registry011

import (
	"math/big"
	"reflect"
	"strconv"
)

// Assigned source components at the SAME original I/B/P/M/C/A cuts. Each
// child retains its own incoming envelope. These are conditional cost DATA,
// not full role allowances, reservations or an executable source witness.
// First-P raw interpretation, catalog/raw/read/write/retention/context costs,
// capture, physical resources, native CPU/RAM, recovery and final work remain
// separate. In particular firstRead already contains cursor initialization.
type providerHistoryRoleComponentCost struct {
	control       providerHistoryRoleControlParts
	requests      providerHistoryRoleRequestControlParts
	bindings      providerHistoryRoleBindingParts
	results       providerHistoryRoleResultControlParts
	cursor        providerHistoryRoleCursorParts
	rpc, boundary providerHistoryBoundaryRoleCost
	content       providerHistoryBoundaryRoleCost
	mechanical    providerHistoryContentJourneyCost
	callbacks     providerHistoryCallbackJourneyCost
	joined        providerHistoryRoleControlParts
}

// Price the assigned components from one sealed prospective manifest. The
// original field/reader fences required by every child remain prerequisites;
// a valid manifest or finite quote does not authenticate a live source. No
// caller may turn this incomplete result into the production derivation.
func providerHistoryCollectionRoleComponentsUpper(manifest *providerHistoryCollectionManifest, methods providerMethodBounds, certificates providerTLSBounds, maximum, prior, members uint64, advancing bool) (providerHistoryRoleComponentCost, bool) {
	var empty providerHistoryRoleComponentCost
	if !manifest.validSnapshot() || prior >= manifest.counts.groups || advancing && prior+1 >= manifest.counts.groups ||
		members == 0 || members > providerHistoryGroupMembers || !methods.fits(manifest.input.capture) || !certificates.valid() {
		return empty, false
	}
	in := manifest.input
	control, controlOK := providerHistoryCollectionRoleControlPartsUpper(prior, maximum, advancing)
	requests, requestsOK := providerHistoryCollectionRoleRequestControlPartsUpper(prior, in.effects, maximum, advancing)
	bindings, bindingsOK := providerHistoryCollectionRoleBindingPartsUpper(prior, members, advancing)
	results, resultsOK := providerHistoryCollectionRoleResultControlPartsUpper(prior, members, certificates, advancing)
	rpcModel, rpcOK := providerHistoryCollectionBoundaryRoleModelUpper(manifest, methods)
	boundaryModel, boundaryOK := providerHistoryCollectionSemanticBoundaryRoleModelUpper(manifest)
	if !controlOK || !requestsOK || !bindingsOK || !resultsOK || !rpcOK || !boundaryOK {
		return empty, false
	}
	rpc, rpcOK := rpcModel.roleUpper(prior, advancing)
	boundary, boundaryOK := boundaryModel.roleUpper(prior, advancing)
	// Uniform prospective cardinalities bound every original cursor state.
	// An inactive window/empty state consumes a subset of these full rows.
	// This adds no new live limit or preflight; all original envelopes remain.
	limits := providerHistoryCollectionCursorLimits{
		Spine: in.blocks, Effects: min(in.effects, in.contentBlocks*uint64(in.journal.Replay.MaxTransactions)),
		PendingBlocks: in.contentBlocks, Transactions: uint64(in.journal.Replay.MaxTransactions),
		EvidenceBytes: uint64(in.journal.Replay.MaxEvidenceBytes), BodyBytes: uint64(in.capture.BodyBytes),
		Semantic: providerHistorySemanticCommitmentLimits{
			ConfirmedClaims: uint64(in.journal.MaxClaims), ConfirmedCreated: uint64(in.journal.MaxIdentities),
			WorkingClaims: uint64(in.journal.MaxClaims), WorkingCreated: uint64(in.journal.MaxIdentities),
			Queue: min(in.versions, in.journal.Replay.History.MaxItems), Window: true, Pending: true,
		},
	}
	cursor, cursorOK := providerHistoryCollectionRoleCursorUpper(prior, advancing, []providerHistoryRoleStateBand{{
		first: 0, last: prior + 1, limits: limits, historyBytes: uint64(in.journal.Replay.History.MaxBytes),
	}})
	native, nativeOK := providerHistoryNativeCollectionInputBounds(in.capture, uint64(strconv.IntSize/8), uint64(reflect.TypeFor[big.Int]().Size()))
	if !rpcOK || !boundaryOK || !cursorOK || !nativeOK {
		return empty, false
	}
	// ValueBytes is a canonical transaction/value ceiling, not the padded
	// big.Int copy-event size native.value. Original canonical evidence and
	// whole response body constrain the former along with the selected native
	// input ceiling. TransactionBytes retains the wider pre-canonical spare.
	evidence := uint64(in.journal.Replay.MaxEvidenceBytes)
	mechanical, mechanicalOK := providerHistoryContentJourneyUpper(providerHistoryContentJourneyLimits{
		Windows: manifest.counts.contentWindows, Blocks: in.blocks, PendingBlocks: in.contentBlocks,
		EvidenceBytes: evidence, BodyBytes: uint64(in.capture.BodyBytes),
		ValueBytes: min(evidence, uint64(in.capture.BodyBytes), native.transaction), TransactionBytes: native.transaction,
		Transactions: uint64(in.journal.Replay.MaxTransactions), Frames: uint64(in.journal.Replay.MaxFrames), Logs: uint64(in.journal.Replay.MaxLogs),
		Effects: in.effects, NativeBlocks: in.native,
	}, in.encoding, in.gas)
	callbacks, callbacksOK := providerHistoryCallbackJourneyUpper(manifest.counts.contentWindows, in.versions,
		uint64(in.journal.Replay.MaxLogs), uint64(in.journal.Replay.MaxEvidenceBytes))
	if !mechanicalOK || !callbacksOK {
		return empty, false
	}
	content, contentOK := providerHistoryContentRoleComponentsUpper(manifest.counts.groups, prior, advancing,
		providerHistoryControlPair{mechanical.normal, mechanical.envelope}, providerHistoryControlPair{callbacks.normal, callbacks.envelope})
	if !contentOK {
		return empty, false
	}
	// No child clock counter is inferred from work[10]. Only the control
	// child assigns the original owned sample events and their counts.
	part := func(i, b, p, m, c, a candidateWorkVector) providerHistoryRoleControlParts {
		return providerHistoryRoleControlParts{
			initial: providerHistoryStageWorkCost{work: i}, before: providerHistoryStageWorkCost{work: b},
			preparation: providerHistoryStageWorkCost{work: p}, member: providerHistoryStageWorkCost{work: m},
			closure: providerHistoryStageWorkCost{work: c}, advance: providerHistoryStageWorkCost{work: a},
		}
	}
	joined, joinedOK := providerHistoryJoinRoleComponentParts(
		control,
		part(requests.initial, requests.before, requests.preparation, requests.member, requests.closure, requests.advance),
		part(bindings.initial, bindings.before, bindings.preparation, bindings.member, bindings.closure, bindings.advance),
		part(results.initial, results.before, results.preparation, results.member, results.closure, results.advance),
		part(cursor.initial, cursor.before, cursor.preparation, cursor.member, cursor.closure, cursor.advance),
		part(rpc.initial, rpc.before, rpc.preparation, rpc.member, rpc.closure, rpc.advance),
		part(boundary.initial, boundary.before, boundary.preparation, boundary.member, boundary.closure, boundary.advance),
		part(content.initial, content.before, content.preparation, content.member, content.closure, content.advance),
	)
	if !joinedOK {
		return empty, false
	}
	return providerHistoryRoleComponentCost{control, requests, bindings, results, cursor, rpc, boundary, content, mechanical, callbacks, joined}, true
}

// ONE journey upper, not an already repeated whole-program F(G). Every
// original independent prefix reader replays a subset of that same monotone
// journey. At most nine before, three S and two own-advance readers apply it;
// member copies do not replay content. A zero prior has no prior content.
//
// Place the complete journey in the first marginal slot as a conservative
// relaxation. It does not assert that the first real group applied content.
// The incoming MAX belongs to one serialized reader scope. Full admitted
// stage uppers, initializer, own advance and future SUM are NEVER replaced by
// this component's MAX. Numeric finiteness is not selected-plan feasibility.
func providerHistoryContentRoleComponentsUpper(groups, prior uint64, advancing bool, mechanical, callbacks providerHistoryControlPair) (providerHistoryBoundaryRoleCost, bool) {
	var empty providerHistoryBoundaryRoleCost
	if groups == 0 || groups > (1<<20)/3 || prior >= groups || advancing && prior+1 >= groups {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	model := providerHistoryBoundaryRoleModel{groups: groups}
	model.tiers[0].count, model.tiers[1].count = 1, groups-1
	for c := range model.delta {
		if mechanical.envelope[c] < mechanical.normal[c] || callbacks.envelope[c] < callbacks.normal[c] {
			return empty, false
		}
		model.tiers[0].work[c] = a.sum(mechanical.normal[c], callbacks.normal[c])
		model.delta[c] = max(mechanical.envelope[c]-mechanical.normal[c], callbacks.envelope[c]-callbacks.normal[c])
	}
	if a.failed {
		return empty, false
	}
	return model.roleUpper(prior, advancing)
}

// Sum complete ASSIGNED components only within each identical original role.
// This never consumes/refunds a meter or joins different role allowances.
// At most eight fixed children are used; every dimension and clock sum is
// checked and a late overflow returns the entire zero result.
func providerHistoryJoinRoleComponentParts(parts ...providerHistoryRoleControlParts) (providerHistoryRoleControlParts, bool) {
	var empty providerHistoryRoleControlParts
	if len(parts) == 0 || len(parts) > 8 {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var sums [6]providerHistoryStageWorkCost
	for _, p := range parts {
		for role, value := range [6]providerHistoryStageWorkCost{p.initial, p.before, p.preparation, p.member, p.closure, p.advance} {
			for c, units := range value.work {
				sums[role].work[c] = a.sum(sums[role].work[c], units)
			}
			sums[role].clocks = a.sum(sums[role].clocks, value.clocks)
		}
	}
	if a.failed {
		return empty, false
	}
	return providerHistoryRoleControlParts{sums[0], sums[1], sums[2], sums[3], sums[4], sums[5]}, true
}
