package registry011

// Prospective original selections for the local source-component graph.
// catalogNameBytes bounds enumerated filename bytes; directoryBytes instead
// bounds BOTH original directory PATH strings. Neither is observed authority.
// The future complete factory must bind every field before H0, including a
// qualified physical per-file overhead. This DATA does not qualify a source.
type providerHistoryRoleSourceLimits struct {
	legacy                                            providerHistoryLegacyControlLimits
	setupRecoveries, catalogEntries, catalogNameBytes uint64
	directoryBytes, fileOverhead                      uint64
}

// Six conditional source components and their checked same-role sum. This
// retains all children for provenance and the separate existing firstRead
// initializer bridge. It is NOT complete role/stage/future work or a source
// derivation witness: incoming/failure/capture/native/resources/recovery/final
// and full feasible-profile/factory qualification remain required.
type providerHistoryRoleSourceCost struct {
	assigned providerHistoryRoleComponentCost
	files    providerHistoryOriginalFileRoleCost
	retained providerHistoryRetainedContextRoleCost
	names    providerHistoryNameNamespaceRoleCost
	raw      providerHistoryDirectRawRoleCost
	outer    providerHistoryOuterRoleCost
	joined   providerHistoryRoleControlParts
}

func providerHistoryCollectionRoleSourceUpper(manifest *providerHistoryCollectionManifest, methods providerMethodBounds, certificates providerTLSBounds, maximum uint64, limits providerHistoryRoleSourceLimits, prior, members uint64, advancing bool) (providerHistoryRoleSourceCost, bool) {
	var empty providerHistoryRoleSourceCost
	minimumNames, namesOK := providerUpperProduct(limits.catalogEntries, 255)
	if !manifest.validSnapshot() || !namesOK || maximum == 0 || maximum != limits.legacy.objectBytes ||
		!providerHistoryCatalogEntriesSupported(limits.catalogEntries) || limits.catalogNameBytes < minimumNames {
		return empty, false
	}
	// Every child keeps its own domain, exact original firstRead binding and
	// local guard. No child refusal is replaced by an earlier partial quote.
	assigned, assignedOK := providerHistoryCollectionRoleComponentsUpper(manifest, methods, certificates, maximum, prior, members, advancing)
	files, filesOK := providerHistoryCollectionOriginalFileRoleUpper(manifest, limits.legacy, limits.catalogEntries, limits.catalogNameBytes, limits.fileOverhead, prior, advancing)
	retained, retainedOK := providerHistoryCollectionRetainedContextRoleUpper(manifest, limits.legacy, limits.catalogEntries, limits.directoryBytes, prior)
	names, namesOK := providerHistoryCollectionNameNamespaceRoleUpper(manifest, limits.legacy, limits.setupRecoveries, limits.catalogEntries, prior, advancing)
	raw, rawOK := providerHistoryCollectionDirectRawRoleUpper(manifest, limits.legacy, prior, advancing)
	outer, outerOK := providerHistoryCollectionOuterRoleUpper(manifest, limits.legacy, prior, advancing)
	if !assignedOK || !filesOK || !retainedOK || !namesOK || !rawOK || !outerOK {
		return empty, false
	}
	// The constructor already adds files.firstInitialization (two complete
	// firstRead rows) to its original initializer. files.joined deliberately
	// excludes that bridge; adding it again here would duplicate it.
	joined, joinedOK := providerHistoryJoinRoleComponentParts(assigned.joined, files.joined, retained.joined, names.joined, raw.joined, outer.joined)
	if !joinedOK {
		return empty, false
	}
	return providerHistoryRoleSourceCost{assigned, files, retained, names, raw, outer, joined}, true
}
