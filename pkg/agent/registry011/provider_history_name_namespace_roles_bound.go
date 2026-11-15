package registry011

// Successful logical name work at the unchanged I/B/P/M/C/A cuts. Direct
// read/write/getter names and catalog allowed-map names are distinct calls.
// Legacy inventory/ref names, recorder name copies and native string/map
// CPU, allocation/backing/RAM are separate components, not priced again.
type providerHistoryNameNamespaceRoleCost struct {
	priorMembers, advanceMembers          uint64
	prefixNames, advancePrefixNames       uint64
	initialPerCatalog, prefixCatalog      uint64
	catalogs, mapCopies                   [6]uint64
	directMemberNames, catalogMemberNames [3]uint64
	names, namespace, joined              providerHistoryRoleControlParts
}

// setupRecoveries/catalogEntries must bind the selected original source
// before H0. This numeric DATA creates no source or allowance. The original
// firstRead sublimit is checked, never renewed or charged a second time here.
// P is entered preparation; full M contains P. The original remaining-work
// predicate still needs its additional P; no full-history bound is divided.
func providerHistoryCollectionNameNamespaceRoleUpper(manifest *providerHistoryCollectionManifest, limits providerHistoryLegacyControlLimits, setupRecoveries, catalogEntries, prior uint64, advancing bool) (providerHistoryNameNamespaceRoleCost, bool) {
	var empty providerHistoryNameNamespaceRoleCost
	if !manifest.validSnapshot() || prior >= manifest.counts.groups || advancing && prior+1 >= manifest.counts.groups ||
		setupRecoveries > 64 || !providerHistoryCatalogEntriesSupported(catalogEntries) ||
		limits.firstObjects != manifest.input.firstRead.objects || limits.firstRaw != manifest.input.firstRead.raw {
		return empty, false
	}
	first, firstOK := providerHistoryFirstPReadUpper(limits, manifest.input.capture)
	if !firstOK {
		return empty, false
	}
	for c, units := range first.work {
		if units > manifest.input.firstRead.work[c] {
			return empty, false
		}
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryNameNamespaceRoleCost
	// At least one original request belongs to the current C. An own
	// advance includes that S, so only its next prefix may use the full R.
	out.priorMembers = min(a.product(32, prior), manifest.counts.requests-1)
	out.prefixNames = a.sum(89, a.product(1158, prior), a.product(688, out.priorMembers))
	out.initialPerCatalog = a.sum(788, a.product(132, setupRecoveries), a.product(170, limits.attempts))
	out.prefixCatalog = a.sum(out.initialPerCatalog, a.product(1164, prior), a.product(688, out.priorMembers))
	// A prepare at index k has k<=31 prior pairs. Direct member calls
	// include 4k+4 before capture and 8k+12 in full M. Three S readers
	// read 2m spool names each, m<=32. These are copies at actual getters.
	out.directMemberNames = [3]uint64{128, 260, 192}
	// Eight before catalogs see 2k names each. After capture, the first
	// of eight sees 2k+1; seven see 2k+2. Full M thus has32k+15 names.
	// Eleven S catalogs see2m each. These are allowed-map insertions.
	out.catalogMemberNames = [3]uint64{496, 1007, 704}
	out.catalogs = [6]uint64{0, 4, 8, 16, 12, 0}
	out.mapCopies = [6]uint64{0, 2, 0, 0, 8, 0}
	var names, namespace [6]uint64
	// One initial model: highwater31+frontier30+commit28. B9/S3 are
	// original healthy prefix models. Current T/C/dispatch total5735;
	// current S total3759 excludes its separately counted member spools.
	names[0] = 178
	names[1] = a.sum(a.product(9, out.prefixNames), 5735)
	names[2] = a.sum(370, a.product(115, out.directMemberNames[0]), 2*229)
	names[3] = a.sum(370, a.product(115, out.directMemberNames[1]), 4*229)
	names[4] = a.sum(a.product(3, out.prefixNames), 3759, a.product(115, out.directMemberNames[2]))
	for role, count := range out.catalogs {
		namespace[role] = a.product(count, out.prefixCatalog)
	}
	// B: two dispatch catalogs extend T/C; C preparation extends T once.
	// T and C after-catalog preparation adds413+329, separately from the
	// two actual copies of both root maps (<=512*entries per copy).
	namespace[1] = a.sum(namespace[1], 2*742, 413, 742, a.product(out.mapCopies[1], a.product(512, catalogEntries)), 2*34)
	namespace[2] = a.sum(namespace[2], 8*742, a.product(115, out.catalogMemberNames[0]))
	namespace[3] = a.sum(namespace[3], 16*742, a.product(115, out.catalogMemberNames[1]))
	// Twelve S catalogs include one discarded initial catalog. Eleven
	// extend T/C, eight after-catalogs insert S and copy both maps once.
	namespace[4] = a.sum(namespace[4], 11*742, 8*416, a.product(out.mapCopies[4], a.product(512, catalogEntries)),
		a.product(115, out.catalogMemberNames[2]), 3*34)
	if advancing {
		out.catalogs[5] = 2
		out.advanceMembers = min(a.product(32, prior+1), manifest.counts.requests)
		out.advancePrefixNames = a.sum(89, a.product(1158, prior+1), a.product(688, out.advanceMembers))
		names[5] = a.product(2, out.advancePrefixNames)
		namespace[5] = a.product(2, a.sum(out.initialPerCatalog, a.product(1164, prior+1), a.product(688, out.advanceMembers)))
	}
	parts := func(values [6]uint64) providerHistoryRoleControlParts {
		var rows [6]providerHistoryStageWorkCost
		for role, units := range values {
			rows[role].work[10] = units
		}
		return providerHistoryRoleControlParts{rows[0], rows[1], rows[2], rows[3], rows[4], rows[5]}
	}
	out.names, out.namespace = parts(names), parts(namespace)
	var joinedOK bool
	out.joined, joinedOK = providerHistoryJoinRoleComponentParts(out.names, out.namespace)
	if a.failed || !joinedOK {
		return empty, false
	}
	return out, true
}
