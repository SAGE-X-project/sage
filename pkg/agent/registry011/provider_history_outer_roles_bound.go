package registry011

// Additional successful T/C/S outer codec work at original roles. The child
// frame/profile/predecessor graph is already included in each primitive.
// Initial legacy/firstReading, catalog codec, raw caller comparisons, names,
// I/O and ledger/record/TLS/result/cursor work remain separate components.
type providerHistoryOuterRoleCost struct {
	calls                                    [6]providerHistoryOuterControlCalls
	frames, inventories, objects, highwaters providerHistoryRoleControlParts
	readerInputs, joined                     providerHistoryRoleControlParts
}

// Explicit scalar selections must bind the original source before H0. A
// positive numeric result supplies no source, owner, quota or clock authority.
// It preserves the original firstRead sublimit and remaining-work composite.
// M includes its P, which that unchanged composite still requires separately.
func providerHistoryCollectionOuterRoleUpper(manifest *providerHistoryCollectionManifest, limits providerHistoryLegacyControlLimits, prior uint64, advancing bool) (providerHistoryOuterRoleCost, bool) {
	var empty providerHistoryOuterRoleCost
	if !manifest.validSnapshot() || prior >= manifest.counts.groups || advancing && prior+1 >= manifest.counts.groups ||
		limits.firstObjects != manifest.input.firstRead.objects || limits.firstRaw != manifest.input.firstRead.raw {
		return empty, false
	}
	first, firstOK := providerHistoryFirstPReadUpper(limits, manifest.input.capture)
	if !firstOK {
		return empty, false
	}
	for coordinate, units := range first.work {
		if units > manifest.input.firstRead.work[coordinate] {
			return empty, false
		}
	}
	one, oneOK := providerHistoryOuterControlUpper(limits.metadata, limits.objectBytes, 2)
	recordBytes := min(limits.objectBytes, uint64(candidateLedgerMetadataLimit))
	record, recordOK := providerHistoryOuterControlUpper(limits.metadata, recordBytes, 2)
	if !oneOK || !recordOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryOuterRoleCost
	// Current T7/C4 and the seal's C anchor give twelve B frame parsers.
	// Three T/C plus two S readers give eight S frames/inventories. Each
	// inventory child rebuilds its OWN objects; those are not added again.
	out.calls[1] = providerHistoryOuterControlCalls{2, 12, 2, 11, 8, 13, 7, 11, 6, 2}
	out.calls[2] = providerHistoryOuterControlCalls{0, 2, 0, 0, 0, 0, 0, 0, 0, 4}
	out.calls[3] = out.calls[2] // Only entered P authenticates the current C.
	out.calls[4] = providerHistoryOuterControlCalls{1, 8, 1, 8, 6, 9, 5, 8, 4, 0}
	priorRows := [6]uint64{0, a.product(9, prior), 0, 0, a.product(3, prior), 0}
	if advancing {
		priorRows[5] = a.product(2, prior+1)
	}
	for role, count := range priorRows {
		c := &out.calls[role]
		// One prior complete T/C/S per prefix: three frames, inventories,
		// record objects and parsed pins; two packs. These are local source
		// calls, not a whole-history polynomial apportioned across stages.
		three, two := a.product(3, count), a.product(2, count)
		c.frameParses = a.sum(c.frameParses, three)
		c.inventoryParses = a.sum(c.inventoryParses, three)
		c.packObjects = a.sum(c.packObjects, two)
		c.recordObjects = a.sum(c.recordObjects, three)
		c.packReads = a.sum(c.packReads, two)
		c.recordReads = a.sum(c.recordReads, three)
		c.highwaterParses = a.sum(c.highwaterParses, three)
	}
	var frames, inventories, objects, highwaters, readers [6]candidateWorkVector
	for role, c := range out.calls {
		for coordinate := range one.frameBuilt {
			frames[role][coordinate] = a.sum(a.product(c.frameBuilds, one.frameBuilt[coordinate]), a.product(c.frameParses, one.frameParsed[coordinate]))
			inventories[role][coordinate] = a.sum(a.product(c.inventoryBuilds, one.inventoryBuilt[coordinate]), a.product(c.inventoryParses, one.inventoryParsed[coordinate]))
			objects[role][coordinate] = a.sum(a.product(c.packObjects, one.storedObject[coordinate]), a.product(c.recordObjects, record.storedObject[coordinate]))
			highwaters[role][coordinate] = a.sum(a.product(c.highwaterBytes, one.highwaterBytes[coordinate]), a.product(c.highwaterParses, one.highwaterParsed[coordinate]))
		}
		// These are actual reader text-copy/digest comparisons OUTSIDE the
		// object constructor, unlike the inventory child's internal rebuilds.
		readers[role][10] = a.sum(a.product(c.packReads, a.sum(limits.objectBytes, 1, 133)), a.product(c.recordReads, a.sum(recordBytes, 1, 133)))
	}
	parts := func(values [6]candidateWorkVector) providerHistoryRoleControlParts {
		var rows [6]providerHistoryStageWorkCost
		for role, work := range values {
			rows[role].work = work
		}
		return providerHistoryRoleControlParts{rows[0], rows[1], rows[2], rows[3], rows[4], rows[5]}
	}
	out.frames, out.inventories, out.objects, out.highwaters, out.readerInputs = parts(frames), parts(inventories), parts(objects), parts(highwaters), parts(readers)
	var joinedOK bool
	out.joined, joinedOK = providerHistoryJoinRoleComponentParts(out.frames, out.inventories, out.objects, out.highwaters, out.readerInputs)
	if a.failed || !joinedOK {
		return empty, false
	}
	return out, true
}
