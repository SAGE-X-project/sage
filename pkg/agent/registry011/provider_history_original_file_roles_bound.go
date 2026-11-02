package registry011

// Successful structural paths only, at the unchanged I/B/P/M/C/A cuts.
// Numeric original selections do not authenticate a store or create a hold.
// Native filesystem/CPU/backing/RAM, direct raw/codec/namespace/recording work
// outside firstReading, failure/recovery/final and complete feasibility remain
// separate. The existing constructor adds firstInitialization exactly once;
// joined.initial deliberately excludes that already present firstRead bridge.
type providerHistoryOriginalFileRoleCost struct {
	first               providerHistoryFirstPReadBounds
	firstInitialization candidateWorkVector
	legacy, reads       providerHistoryRoleControlParts
	catalog, metadata   providerHistoryRoleControlParts
	writes, joined      providerHistoryRoleControlParts
	counts              [6]providerHistoryOriginalFileRoleCounts
	storage             [6]providerHistoryStorageCharge
}

type providerHistoryOriginalFileRoleCounts struct {
	models, catalogs, directoryPairs, identities uint64
	readBytes, readCalls, writes, renames        uint64
}

// Every explicit limit must later bind the selected original store before H0.
// The firstRead objects/raw and its local work guard are checked here against
// the same sealed manifest. A larger global/stage allowance cannot replace a
// deficient firstRead sublimit. P is the already-entered preparation; full M
// includes it. Original dispatch.remaining still needs its additional P row.
func providerHistoryCollectionOriginalFileRoleUpper(manifest *providerHistoryCollectionManifest, limits providerHistoryLegacyControlLimits, catalogEntries, nameBytes, overhead, prior uint64, advancing bool) (providerHistoryOriginalFileRoleCost, bool) {
	var empty providerHistoryOriginalFileRoleCost
	if !manifest.validSnapshot() || prior >= manifest.counts.groups || advancing && prior+1 >= manifest.counts.groups ||
		catalogEntries == 0 || catalogEntries > 1<<20 || nameBytes == 0 || overhead == 0 ||
		limits.firstObjects != manifest.input.firstRead.objects || limits.firstRaw != manifest.input.firstRead.raw {
		return empty, false
	}
	legacy, legacyOK := providerHistoryLegacyControlUpper(limits)
	first, firstOK := providerHistoryFirstPReadUpper(limits, manifest.input.capture)
	if !legacyOK || !firstOK {
		return empty, false
	}
	for c, units := range first.work {
		if units > manifest.input.firstRead.work[c] {
			return empty, false
		}
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryOriginalFileRoleCost
	out.first = first
	for c, units := range first.work {
		out.firstInitialization[c] = a.product(2, units)
	}
	models := [6]uint64{2, 9, 0, 0, 3, 0}
	catalogs := [6]uint64{0, 4, 8, 16, 12, 0}
	pairs := [6]uint64{0, 10, 8, 16, 11, 0}
	// Each firstReading already funds its TWO identity calls (16 read(0)
	// units). Subtract those source-owned calls, not successful byte lengths.
	identities := [6]uint64{2, 11, 6, 9, 6, 0}
	if advancing {
		models[5], catalogs[5], pairs[5], identities[5] = 2, 2, 2, 2
	}
	var legacyRows, readRows, catalogRows, metadataRows, writeRows [6]providerHistoryStageWorkCost
	for role, count := range models {
		for c, units := range legacy.modelControl {
			if role != 0 {
				units = a.sum(units, first.work[c])
			}
			legacyRows[role].work[c] = a.product(count, units)
		}
		out.counts[role].models = count
	}
	const h, page uint64 = candidateLedgerMetadataLimit, 64 << 10
	d := max(limits.metadata, h)
	p, wire := limits.objectBytes, uint64(manifest.input.capture.WireBytes)
	l := a.sum(wire, page)
	objects := min(limits.firstObjects, a.product(4, limits.entries))
	// H(q) has firstReading plus five outside controls, 21D+2P per
	// preceding T/C/S and two archived spool files per preceding member.
	// M refreshes raw firstReading originals; it does NOT repeat its parsers.
	prefix := func(q uint64, includeFirst bool) (uint64, uint64) {
		members := min(a.product(32, q), manifest.counts.requests)
		bytes := a.sum(a.product(5, d), a.product(q, a.sum(a.product(21, d), a.product(2, p))), a.product(members, l))
		calls := a.sum(5, a.product(23, q), a.product(2, members))
		if includeFirst {
			bytes = a.sum(bytes, limits.firstRaw)
			calls = a.sum(calls, objects, 9)
		}
		return bytes, calls
	}
	outsideBytes, outsideCalls := prefix(prior, false)
	fullBytes, fullCalls := prefix(prior, true)
	// 12P+185D/197 covers all current noncatalog T/C/dispatch/S suffixes.
	// Using that complete one-stage row separately for B and S is a constant
	// conservative relaxation, NOT a whole-history F(G) repeated per stage.
	suffixBytes := a.sum(a.product(12, p), a.product(185, d))
	out.counts[0].readBytes, out.counts[0].readCalls = a.product(8, d), 8
	out.counts[1].readBytes = a.sum(a.product(9, outsideBytes), suffixBytes)
	out.counts[1].readCalls = a.sum(a.product(9, outsideCalls), 197)
	// P refreshes both retained raw transcripts and two C/HEAD anchors,
	// then reads at most31 old member spool pairs twice. Full M also reads
	// at most32 pairs twice after retention, including its new originals.
	out.counts[2].readBytes = a.sum(a.product(2, fullBytes), a.product(2, p), a.product(52, d), a.product(62, l))
	out.counts[2].readCalls = a.sum(a.product(2, fullCalls), 54, 124)
	out.counts[3].readBytes = a.sum(out.counts[2].readBytes, a.product(64, l))
	out.counts[3].readCalls = a.sum(out.counts[2].readCalls, 128)
	out.counts[4].readBytes = a.sum(a.product(3, outsideBytes), suffixBytes, a.product(96, l))
	out.counts[4].readCalls = a.sum(a.product(3, outsideCalls), 197, 192)
	if advancing {
		bytes, calls := prefix(prior+1, false)
		out.counts[5].readBytes, out.counts[5].readCalls = a.sum(a.product(2, bytes), d), a.sum(a.product(2, calls), 1)
	}
	// Catalog payload reads and their original legacy/scan codecs are distinct
	// from directory enumeration and from raw firstReading object loading.
	catalogCalls := a.sum(10, a.product(5, limits.attempts), a.product(4, limits.chunks))
	catalogBytes := a.sum(a.product(d, a.sum(6, a.product(5, limits.attempts), a.product(4, limits.chunks))), 4*page)
	directory := a.sum(a.product(2, nameBytes), a.product(2, catalogEntries), 265)
	for role := range out.counts {
		count := &out.counts[role]
		count.catalogs, count.directoryPairs, count.identities = catalogs[role], pairs[role], identities[role]
		readRows[role].work[10] = a.sum(count.readBytes, count.readCalls)
		for c, units := range legacy.catalog {
			catalogRows[role].work[c] = a.product(count.catalogs, units)
		}
		catalogRows[role].work[10] = a.sum(catalogRows[role].work[10], a.product(count.catalogs, a.sum(catalogBytes, catalogCalls)))
		metadataRows[role].work[10] = a.sum(a.product(count.directoryPairs, directory), a.product(8, count.identities))
	}
	// Source allocations: T9+C8, member response/TLS2, S9. The three
	// outer groups each contain eight small metadata files; T/S also have
	// their own P-sized pack. No write/refund is inferred for I/P/advance.
	for _, row := range []struct {
		role                                       int
		payload, metadata, objects, files, renames uint64
	}{{1, p, 16 * h, 3, 17, 2}, {3, wire, page, 2, 2, 0}, {4, p, 8 * h, 2, 9, 1}} {
		raw := a.sum(row.payload, row.metadata)
		out.storage[row.role] = providerHistoryStorageCharge{physical: a.sum(raw, a.product(row.files, overhead)), raw: raw,
			payload: row.payload, metadata: row.metadata, objects: row.objects, files: row.files}
		out.counts[row.role].writes, out.counts[row.role].renames = row.files, row.renames
		writeRows[row.role].work[9] = raw
		// write, two syncs, two four-event identities+open+close per file;
		// each rename has its own event and directory sync, hence13f+2r.
		writeRows[row.role].work[10] = a.sum(raw, a.product(13, row.files), a.product(2, row.renames))
	}
	parts := func(rows [6]providerHistoryStageWorkCost) providerHistoryRoleControlParts {
		return providerHistoryRoleControlParts{rows[0], rows[1], rows[2], rows[3], rows[4], rows[5]}
	}
	out.legacy, out.reads, out.catalog, out.metadata, out.writes = parts(legacyRows), parts(readRows), parts(catalogRows), parts(metadataRows), parts(writeRows)
	var joinedOK bool
	out.joined, joinedOK = providerHistoryJoinRoleComponentParts(out.legacy, out.reads, out.catalog, out.metadata, out.writes)
	if a.failed || !joinedOK {
		return empty, false
	}
	return out, true
}
