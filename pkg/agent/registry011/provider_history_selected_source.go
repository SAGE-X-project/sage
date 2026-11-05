package registry011

// Value DATA for the original source components selected by ONE config. This
// does not contain a derivation, owner or executable callback. The assigned
// prefix, full admitted future, failed reconstruction and successful final
// reader stay distinct. Compiler-own/cleanup/native/backing/lifetime RAM costs
// and whole-profile feasibility must still close before execution is granted.
type providerHistorySelectedSourceCost struct {
	limits              providerHistoryRoleSourceLimits
	clock               providerHistoryClockSelection
	bootstrap           providerHistoryCompiledBootstrapCost
	collection          providerHistoryUniformSourceCost
	recovery, finalRead providerHistoryReconstructionSourceCost
	finalization        providerHistoryFinalizationCost
	finalWork           candidateWorkVector
	assigned            providerHistoryStageWorkCost
}

// Bind the role inputs to their original store, sealed manifest and two path
// strings. No caller-supplied role maxima or small test defaults are accepted.
// These are pre-H0 numeric inputs: they neither authenticate the directories
// nor validate/mint the still-missing complete history derivation. The eventual
// enclosing compiler must fund and freeze the same private selection.
func providerHistorySourceLimitsFromConfig(config providerCustodyConfig) (providerHistoryRoleSourceLimits, bool) {
	var empty providerHistoryRoleSourceLimits
	b := config.bounds
	if b.history == nil || b.scan == nil || b.firstAdmission == nil || b.guardian != nil || !b.validStructure() ||
		b.history.derivation != nil || config.attempt != 0 || config.custodyDirectory == "" || config.evidenceDirectory == "" {
		return empty, false
	}
	h, manifest := b.history, b.history.collection
	if !manifest.validSnapshot() || config.capture != manifest.input.capture ||
		h.stages != manifest.counts.groups || h.stages == 0 || h.stages > (1<<20)/3 ||
		h.catalogEntries == 0 || h.catalogEntries > 1<<20 {
		return empty, false
	}
	names, namesOK := providerUpperProduct(h.catalogEntries, 255)
	directories, directoriesOK := providerUpperSum(uint64(len(config.custodyDirectory)), uint64(len(config.evidenceDirectory)))
	if !namesOK || h.catalogNameBytes < names || !directoriesOK || directories > 2<<20 {
		return empty, false
	}
	return providerHistoryRoleSourceLimits{
		legacy: providerHistoryLegacyControlLimits{
			metadata: b.maximumMetadataPerCommit, objectBytes: b.maximumObjectBytes,
			firstObjects: manifest.input.firstRead.objects, firstRaw: manifest.input.firstRead.raw,
			entries: uint64(b.maximumEntries), attempts: b.scan.maximumAttempts, chunks: b.scan.maximumChunks,
		},
		setupRecoveries: uint64(b.maximumSetupRecoveries), catalogEntries: h.catalogEntries,
		catalogNameBytes: h.catalogNameBytes, directoryBytes: directories, fileOverhead: b.fileOverheadBytes,
	}, true
}

// Compile every assigned source role from one unminted config and explicit
// clock snapshot. The eventual factory must freeze that SAME native backend;
// matching value DATA here cannot prove that a backend exists or qualify it.
// Every refusal returns zero, including any earlier positive child. No child
// result, profile or meter is changed to make a later role fit.
func providerHistoryCompileSelectedSource(config providerCustodyConfig, clock providerHistoryClockSelection) (providerHistorySelectedSourceCost, bool) {
	var empty providerHistorySelectedSourceCost
	limits, limitsOK := providerHistorySourceLimitsFromConfig(config)
	if !limitsOK || !providerHistorySourceClockMatchesPolicy(config.policy, clock) {
		return empty, false
	}
	bootstrap, bootstrapOK := providerHistoryCompileBootstrapSource(config.bounds, config.capture)
	if !bootstrapOK {
		return empty, false
	}
	b := config.bounds
	collection, collectionOK := providerHistoryCollectionUniformSourceUpper(b.history.collection, *b.methods, *b.tls, b.maximumObjectBytes, limits)
	if !collectionOK {
		return empty, false
	}
	recovery, recoveryOK := providerHistoryCompileReconstructionSource(config, true)
	finalRead, finalOK := providerHistoryCompileReconstructionSource(config, false)
	finalization, closingOK := providerHistoryFinalizationUpper(b.history.collection, uint64(len(clock.id)))
	if !recoveryOK || !finalOK || !closingOK {
		return empty, false
	}
	out := providerHistorySelectedSourceCost{
		limits: limits, clock: clock, bootstrap: bootstrap, collection: collection,
		recovery: recovery, finalRead: finalRead, finalization: finalization,
	}
	var a providerHistoryReadArithmetic
	for coordinate := range out.assigned.work {
		// Reconstruction stops at its first refusal. Closing is entered only
		// after reconstruction succeeds; its paired late-commit extras have
		// already SUMmed inside finalization. Only one phase can leave a new
		// outstanding incoming envelope in this successful-final invocation.
		out.finalWork[coordinate] = a.sum(finalRead.normal[coordinate], finalization.normal[coordinate],
			max(finalRead.incoming[coordinate], finalization.incoming[coordinate]))
		if coordinate == 10 {
			// The two clock events are not in either nonclock child above.
			out.finalWork[coordinate] = a.sum(out.finalWork[coordinate], finalization.clocks)
		}
		// These are distinct PRE-HELD roles. Failed inspection and successful
		// final inspection must SUM here, even though execution chooses one.
		// Keep the whole selected pre-I upper, its separate first P, and the
		// complete I+G*FULLstage future; no initial or future guard is refunded.
		out.assigned.work[coordinate] = a.sum(bootstrap.source.selectedPreI.work[coordinate], bootstrap.source.startup.work[coordinate],
			collection.future.work[coordinate], recovery.envelope[coordinate], out.finalWork[coordinate])
	}
	out.assigned.clocks = a.sum(bootstrap.source.selectedPreI.clocks, bootstrap.source.startup.clocks,
		collection.future.clocks, finalization.clocks)
	if a.failed {
		return empty, false
	}
	// Preserve ALL eleven dimensions, including historical proofs in R/F.
	// Their current live read domain still refuses proofs: do not erase that
	// mismatch or install these source DATA as an executable recovery bound.
	return out, true
}

// Bounded policy DATA matching only. No native sample or clock construction
// occurs. Parsing and transient copies here belong to the still-open compiler
// cost row, rather than being borrowed from an execution or final-read budget.
func providerHistorySourceClockMatchesPolicy(policy *ProviderPolicyCandidate, clock providerHistoryClockSelection) bool {
	if policy == nil || policy.raw == "" || len(policy.raw) > 64<<10 || clock.id == "" || len(clock.id) > 256 ||
		clock.policy != policy.policy || !lowerHash(clock.policy) || !lowerHash(clock.epoch) || clock.origin.IsZero() ||
		clock.window <= 0 || clock.uncertainty <= 0 || clock.maximum == 0 || clock.maximum > 1<<20 {
		return false
	}
	root, ok := contentJSON([]byte(policy.raw)).(map[string]any)
	if !ok {
		return false
	}
	selected, ok := root["clock"].(map[string]any)
	return ok && selected["id"] == clock.id
}
