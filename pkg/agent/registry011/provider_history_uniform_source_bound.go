package registry011

// Compilation of the currently assigned source components, not the complete
// production program or its nonserializable derivation. Missing incoming,
// uncertain-failure, initializer/native/backing/RAM/recovery/final obligations
// remain separate. Finite DATA never opens H0 or qualifies an operating profile.
type providerHistoryUniformSourceCost struct {
	groups              uint64
	roles               providerHistoryRoleControlParts
	firstInitialization candidateWorkVector
	stage, future       providerHistoryStageWorkCost
	storage             providerHistoryUniformStorageCost
}

// Each original stage uses the same admitted maximum, apart from the existing
// RPC/wire table-count refinement. First take the coordinate maximum WITHIN
// each role over every prospective prior cut. Then assemble one complete stage
// and SUM that full stage G times, including unused final-advance room. A MAX
// across alternatives is never substituted for the future admitted-stage SUM.
//
// Enumeration is bounded by the existing manifest domain. It does not assume
// that all child formulae are monotone or that endpoints dominate intermediate
// cuts. No original input, owner, meter, reservation or source is changed.
func providerHistoryCollectionUniformSourceUpper(manifest *providerHistoryCollectionManifest, methods providerMethodBounds, certificates providerTLSBounds, maximum uint64, limits providerHistoryRoleSourceLimits) (providerHistoryUniformSourceCost, bool) {
	var empty providerHistoryUniformSourceCost
	if !manifest.validSnapshot() || manifest.counts.groups == 0 || manifest.counts.groups > (1<<20)/3 {
		return empty, false
	}
	storage, storageOK := providerHistoryCollectionUniformStorageUpper(manifest, methods, certificates, maximum, limits.legacy.metadata, limits.fileOverhead)
	if !storageOK {
		return empty, false
	}
	var selected [6]providerHistoryStageWorkCost
	var first candidateWorkVector
	for prior := uint64(0); prior < manifest.counts.groups; prior++ {
		source, ok := providerHistoryCollectionTransportSourceUpper(manifest, methods, certificates, maximum, limits, prior, providerHistoryGroupMembers, prior+1 < manifest.counts.groups)
		if !ok {
			return empty, false
		}
		p := source.joined
		if prior == 0 {
			first = source.source.files.firstInitialization
		} else if p.initial != selected[0] || source.source.files.firstInitialization != first {
			return empty, false // Initializer ownership cannot vary by prior cut.
		}
		for role, value := range [6]providerHistoryStageWorkCost{p.initial, p.before, p.preparation, p.member, p.closure, p.advance} {
			for coordinate, units := range value.work {
				selected[role].work[coordinate] = max(selected[role].work[coordinate], units)
			}
			selected[role].clocks = max(selected[role].clocks, value.clocks)
		}
	}
	// Local capture selects ACTIVE method classes, but current admission still
	// assigns global WireBytes+1 per member. Unused/inactive class differences
	// must not silently narrow that original admission row in the future SUM.
	wire, wireOK := providerUpperSum(uint64(manifest.input.capture.WireBytes), 1)
	if !wireOK || selected[3].work[1] > wire {
		return empty, false
	}
	selected[3].work[1] = wire
	roles := providerHistoryRoleControlParts{selected[0], selected[1], selected[2], selected[3], selected[4], selected[5]}
	stage, stageOK := providerHistoryAssignedStageWorkUpper(roles, providerHistoryGroupMembers)
	future, futureOK := providerHistoryAssignedFutureWorkUpper(roles.initial, first, stage, manifest.counts.groups)
	if !stageOK || !futureOK {
		return empty, false
	}
	return providerHistoryUniformSourceCost{manifest.counts.groups, roles, first, stage, future, storage}, true
}

// Source components only: tailMS and native qualification are deliberately not
// manufactured to satisfy providerHistoryDispatchBounds.valid. Full M already
// includes P; the literal remaining guard retains the entered P once MORE.
func providerHistoryAssignedStageWorkUpper(parts providerHistoryRoleControlParts, count int) (providerHistoryStageWorkCost, bool) {
	var empty providerHistoryStageWorkCost
	if count < 1 || count > providerHistoryGroupMembers || parts.member.work[0] != 1 || parts.member.work[1] == 0 ||
		parts.preparation.clocks > parts.member.clocks {
		return empty, false
	}
	for _, part := range [4]providerHistoryStageWorkCost{parts.before, parts.preparation, parts.closure, parts.advance} {
		if part.work[0] != 0 || part.work[1] != 0 {
			return empty, false
		}
	}
	for c, units := range parts.preparation.work {
		if units > parts.member.work[c] {
			return empty, false
		}
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryStageWorkCost
	for c := range out.work {
		out.work[c] = a.sum(parts.before.work[c], parts.preparation.work[c], a.product(uint64(count), parts.member.work[c]), parts.closure.work[c], parts.advance.work[c])
	}
	out.clocks = a.sum(parts.before.clocks, parts.preparation.clocks, a.product(uint64(count), parts.member.clocks), parts.closure.clocks, parts.advance.clocks)
	if a.failed {
		return empty, false
	}
	return out, true
}

// Initial assigned rows and the existing firstRead bridge are added ONCE.
// Other original initializer preflight/headroom must still be funded by the
// complete compiler. This component cannot replace that enclosing guard.
func providerHistoryAssignedFutureWorkUpper(initial providerHistoryStageWorkCost, first candidateWorkVector, stage providerHistoryStageWorkCost, groups uint64) (providerHistoryStageWorkCost, bool) {
	if groups == 0 || groups > (1<<20)/3 {
		return providerHistoryStageWorkCost{}, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryStageWorkCost
	for c := range out.work {
		out.work[c] = a.sum(initial.work[c], first[c], a.product(groups, stage.work[c]))
	}
	out.clocks = a.sum(initial.clocks, a.product(groups, stage.clocks))
	if a.failed {
		return providerHistoryStageWorkCost{}, false
	}
	return out, true
}
