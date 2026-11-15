package registry011

import "reflect"

// The assigned post-ABI/window events of the original parsed/applied fold.
// Opening/closing here mean CONTENT windows only. RPC decoding, block/effect
// replay, queue retention, roots, native CPU/RAM and factory authority remain
// separate. These parts must replace overlapping older aggregate rows, not
// be added to their stop/begin/finish/history comparison charges a second time.
type providerHistorySemanticBoundaryCost struct {
	opening, closing, absentState, state, grants, historyItem providerHistoryControlPair
}

func providerHistorySemanticBoundaryControlUpper(l providerHistorySemanticCloneLimits) (providerHistorySemanticBoundaryCost, bool) {
	var empty providerHistorySemanticBoundaryCost
	clone, cloneOK := providerHistorySemanticCloneUpper(l)
	record, recordOK := providerHistoryAppliedStateSerializationUpper(providerHistoryStateSerializationLimits{Keys: l.Keys, Services: l.Services, Record: true})
	grants, grantsOK := providerHistoryAppliedStateSerializationUpper(providerHistoryStateSerializationLimits{Grants: l.Grants})
	if !cloneOK || !recordOK || !grantsOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	logical := func(n uint64) providerHistoryControlPair {
		v := candidateWorkVector{10: n}
		return providerHistoryControlPair{v, v}
	}
	var out providerHistorySemanticBoundaryCost
	contract, outcomes := uint64(len(replayContractABI)), uint64(len(replayOutcomesABI))
	out.opening.normal[2] = a.sum(contract, outcomes)
	out.opening.normal[10] = a.sum(a.product(contract, contract), contract, 2,
		a.product(outcomes, outcomes), outcomes, 2, clone[10],
		uint64(reflect.TypeFor[providerHistoryExpectedWindow]().Size()), uint64(reflect.TypeFor[replayRun]().Size()), 1)
	// Original ABI and clone paths have no nonzero outstanding preflight.
	out.opening.envelope = out.opening.normal
	out.closing = logical(a.sum(uint64(reflect.TypeFor[replayStream]().Size()), uint64(reflect.TypeFor[int]().Size()),
		uint64(reflect.TypeFor[*providerHistoryExpectedWindow]().Size()), 2))
	// Two empty comparisons, chargeHistory copy16, observedCopy1, select16.
	// replayBudget.charge is scalar accounting, not a decoded-byte event.
	out.absentState = logical(38)
	for c := range out.state.normal {
		out.state.normal[c] = a.sum(record.normal[c], grants.normal[c])
		out.state.envelope[c] = a.sum(record.envelope[c], grants.envelope[c])
	}
	out.grants = out.state
	r, g := record.normalBytes, grants.normalBytes
	// Both serializers run even when record refuses at its late output cap.
	// Present state then compares/copies record; grants separately copies the
	// stopped record and compares BOTH serializations before charge/select.
	stateTail := a.sum(a.product(3, r), 2)
	out.state.normal[10] = a.sum(out.state.normal[10], stateTail)
	// A mismatching selected record can be65536 bytes even when this typed
	// state's smaller prospective graph encodes to less. Copy follows equality.
	out.state.envelope[10] = a.sum(out.state.envelope[10], max(stateTail, a.sum(65536, r, 1)))
	out.grants.normal[10] = a.sum(out.grants.normal[10], a.product(3, r), a.product(2, g), 39)
	// A failed grant equality may enter with the original262144-byte result,
	// although a successful one equals the much smaller typed fixed-scope JSON.
	out.grants.envelope[10] = a.sum(out.grants.envelope[10], a.product(3, r), 262144, g, 39)
	// ExpectedItem copy, two owned byte copies/equalities, chargeHistory16 and
	// compared-int copy. Both source and RPC entry/command caps are262144.
	out.historyItem = logical(a.sum(uint64(reflect.TypeFor[providerHistoryExpectedItem]().Size()),
		uint64(reflect.TypeFor[int]().Size()), 23, 6*262144))
	if a.failed {
		return empty, false
	}
	return out, true
}

// Same original boundary call cuts as the RPC component, with a disjoint
// semantic assignment. Reusing the finite rank/role arithmetic does not merge
// these source costs or certify the eventual complete program. The production
// compiler must bind original parsed/applied state, whole-fold item counters,
// selected manifest and actual limits before it may use any of these DATA.
func providerHistoryCollectionSemanticBoundaryRoleModelUpper(manifest *providerHistoryCollectionManifest) (providerHistoryBoundaryRoleModel, bool) {
	var empty providerHistoryBoundaryRoleModel
	caps, capsOK := providerHistoryCollectionBoundaryPhaseCapsBound(manifest)
	if !capsOK || caps.grants > caps.state {
		return empty, false
	}
	j := manifest.input.journal
	if j.MaxClaims < 1 || j.MaxIdentities < 1 {
		return empty, false
	}
	parts, partsOK := providerHistorySemanticBoundaryControlUpper(providerHistorySemanticCloneLimits{
		Claims: uint64(j.MaxClaims), Created: uint64(j.MaxIdentities), Keys: 128, Services: 16, Grants: 128,
	})
	if !partsOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	scale := func(v candidateWorkVector, count uint64) candidateWorkVector {
		var out candidateWorkVector
		for c := range out {
			out[c] = a.product(v[c], count)
		}
		return out
	}
	var out providerHistoryBoundaryRoleModel
	out.groups = caps.groups
	out.tiers[0] = providerHistoryBoundaryNormalTier{caps.state, parts.opening.normal}
	out.tiers[1] = providerHistoryBoundaryNormalTier{caps.state, parts.closing.normal}
	out.tiers[2] = providerHistoryBoundaryNormalTier{caps.state - caps.grants, parts.absentState.normal}
	out.tiers[3] = providerHistoryBoundaryNormalTier{caps.grants, parts.state.normal}
	out.tiers[4] = providerHistoryBoundaryNormalTier{caps.grants, parts.grants.normal}
	full, rest := caps.historyItems/providerHistoryGroupMembers, caps.historyItems%providerHistoryGroupMembers
	used := full
	if rest != 0 {
		used = a.sum(used, 1)
		out.tiers[6] = providerHistoryBoundaryNormalTier{1, scale(parts.historyItem.normal, rest)}
	}
	if a.failed || used > caps.history {
		return empty, false
	}
	out.tiers[5] = providerHistoryBoundaryNormalTier{full, scale(parts.historyItem.normal, providerHistoryGroupMembers)}
	out.tiers[7].count = caps.history - used
	assigned := a.sum(a.product(3, caps.state), caps.grants, caps.history)
	if a.failed || assigned > caps.groups {
		return empty, false
	}
	out.tiers[8].count = caps.groups - assigned
	// Missing optional present stops may occupy the richer present-state
	// slots. Verify dominance before treating P as a cap rather than a count.
	for c := range out.delta {
		if parts.absentState.normal[c] > parts.state.normal[c] {
			return empty, false
		}
	}
	// The present-stop capacity guard is AFTER both serializers and record
	// equality. One further present state may therefore enter before refusing,
	// even when P==0. If an absent slot exists, its proxy N must fund that
	// complete failed present E; P is only a successful-stop cardinality cap.
	stateProxy := parts.state.normal
	if caps.grants < caps.state {
		stateProxy = parts.absentState.normal
	}
	for _, phase := range []struct {
		count uint64
		cost  providerHistoryControlPair
	}{
		{caps.state, parts.opening}, {caps.state, parts.closing},
		{caps.state, parts.absentState}, {caps.state, providerHistoryControlPair{stateProxy, parts.state.envelope}},
		{caps.grants, parts.grants},
	} {
		if phase.count == 0 {
			continue
		}
		for c := range out.delta {
			if phase.cost.normal[c] > phase.cost.envelope[c] {
				return empty, false
			}
			out.delta[c] = max(out.delta[c], phase.cost.envelope[c]-phase.cost.normal[c])
		}
	}
	// History per-item N==E; splitting V across windows adds positions, not
	// fresh32-item allowances. The original failed phase aborts its role.
	if a.failed || !out.valid() {
		return empty, false
	}
	return out, true
}
