package registry011

// Value DATA for the original source components selected by ONE config. This
// does not contain a derivation, owner or executable callback. Missing incoming,
// failure/recovery/final/cleanup and compiler/native/backing/lifetime RAM costs
// must still be closed before a production factory can grant execution.
type providerHistorySelectedSourceCost struct {
	limits     providerHistoryRoleSourceLimits
	bootstrap  providerHistoryCompiledBootstrapCost
	collection providerHistoryUniformSourceCost
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

// Compile the fresh prefix and FULL admitted future source components from
// the same selection. Every failure returns the entire zero result, including
// previously derived limits or a successful prefix. The unchanged whole-stage
// SUM, selected initializer guard, first-read rows and storage remain inside
// their original children. A positive result is not a complete/sufficient or
// feasible program, cannot open H0, and does not install a reservation.
func providerHistoryCompileSelectedSource(config providerCustodyConfig) (providerHistorySelectedSourceCost, bool) {
	var empty providerHistorySelectedSourceCost
	limits, limitsOK := providerHistorySourceLimitsFromConfig(config)
	if !limitsOK {
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
	return providerHistorySelectedSourceCost{limits, bootstrap, collection}, true
}
