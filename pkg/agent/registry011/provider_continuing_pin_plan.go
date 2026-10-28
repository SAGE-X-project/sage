package registry011

// This prospective local command funds both independent complete child
// replays. The original RPC reservations/closed observers do not pay it.
func providerContinuingPinPlan(g *providerGuardianOwner, command *providerContinuingPinCommand) (providerContinuingPlan, bool) {
	var empty providerContinuingPlan
	if g == nil || !g.selectedLocked() || !g.initialized || !g.admitted || g.integration == nil || g.integration.ledger == nil || command == nil || command.parent == nil || g.continuing != command.parent || command.parent.pin != command || command.input.kind != "pin-input" || !lowerHash(command.input.digest) || !lowerHash(command.selected) || g.config.bounds.methods == nil || g.config.bounds.tls == nil || g.store.firstLimits == nil {
		return empty, false
	}
	s, b, ledger := g.store, g.config.bounds, g.integration.ledger
	page := ledger.continuation
	if !ledger.valid() || page == nil || !page.valid(ledger) || page.pending != nil || page.final || page.bounds.pageRecords < 3 || uint64(len(page.records)) > page.bounds.pageRecords-3 || !lowerHash(g.integration.bootstrap) || ledger.providerStart != g.integration.bootstrap || s.sequence > ^uint64(0)-3 || s.sequence+3 > uint64(b.maximumCommits) || len(s.objects) > b.maximumObjects-32 {
		return empty, false
	}
	for _, value := range ledger.current.totals.bootstrap.outstanding {
		if value != 0 {
			return empty, false
		}
	}
	storage, ok := providerFirstOperationStorageUpper(b, g.config.capture, uint64(len(providerFirstOperationRequestWire)))
	if !ok || storage.exchange < 4096 {
		return empty, false
	}
	n, maximum := max(b.maximumMetadataPerCommit, uint64(64<<10)), uint64(1<<20)
	var sizes [4]uint64
	for i, request := range command.requests {
		capture := command.captures[i]
		if request == nil || !capture.valid() {
			return empty, false
		}
		child, valid := providerOperationStorageUpper(b, capture, uint64(len(request.wire)))
		base64, product := providerUpperProduct(uint64(capture.BodyBytes), 4)
		base64, sum := providerUpperSum(base64, 2)
		if !valid || !product || !sum || child.exchange > 4<<20 || child.transport > 4<<20 {
			return empty, false
		}
		sizes[i] = max(n, b.maximumObjectBytes, uint64(capture.WireBytes), base64/3)
		maximum = max(maximum, sizes[i])
	}
	headroom, a := providerUpperProduct(maximum, maximum)
	headroom, c := providerUpperSum(headroom, b.maximumObjectBytes, 2)
	// R3 closed table: actual child43Q/76L is additional to all metadata.
	// Two separate5Q/9L input rechecks cover C/S old-material invocations.
	execution, e := providerFirstOperationPhaseUpper(n, 655, 3595, 7, 29, storage.consumption, headroom)
	remaining, f := providerFirstOperationPhaseUpper(n, 594, 3327, 5, 37, storage.settlement, headroom)
	if !a || !c || !e || !f {
		return empty, false
	}
	validatorQ, validatorL := [4]uint64{1, 5, 1, 4}, [4]uint64{0, 8, 0, 7}
	for i, size := range sizes {
		row := []providerRPCSizeRow{{size, 43 + validatorQ[i], 76 + validatorL[i]}}
		capture := command.captures[i]
		head, h := providerUpperProduct(uint64(capture.HeaderBytes), 13)
		body, j := providerUpperProduct(uint64(capture.BodyBytes), 4)
		extra, k := providerUpperSum(head, body, 5, 9731, providerRPCClassRecursionUpper)
		if !h || !j || !k || !providerRPCAddSizeRows(&execution, row) || !providerRPCAddSizeRows(&remaining, row) {
			return empty, false
		}
		var left, right bool
		execution[10], left = providerUpperSum(execution[10], extra)
		remaining[10], right = providerUpperSum(remaining[10], extra)
		if !left || !right {
			return empty, false
		}
	}
	execution[3], execution[8] = 1, 1
	work, err := providerFrontierWorkTotal(execution, remaining)
	consumptionFiles, h := providerUpperProduct(14, b.fileOverheadBytes)
	settlementFiles, i := providerUpperProduct(18, b.fileOverheadBytes)
	consumption, j := providerUpperSum(storage.consumption, consumptionFiles)
	settlement, k := providerUpperSum(storage.settlement, settlementFiles)
	physical, l := providerUpperSum(consumption, settlement)
	mainRaw, m := providerUpperSum(10*candidateLedgerMetadataLimit, 16<<10, 8<<10, 64<<10, 4*b.maximumMetadataPerCommit, 2*candidateLedgerMetadataLimit)
	metadataRaw, rawOK := providerUpperSum(mainRaw, 5*candidateLedgerMetadataLimit)
	files, filesOK := providerUpperProduct(22, b.fileOverheadBytes)
	metadataPhysical, physicalOK := providerUpperSum(metadataRaw, files)
	// R3 G767/4870 plus four newly observed class selectors4Q/4L.
	metadataWork, metadataOK := providerGuardianAddedWork(b, 771, 4874, metadataRaw)
	metadataWork[10], ok = providerUpperSum(metadataWork[10], 2*providerRPCClassRecursionUpper, 4*9731, headroom)
	if err != nil || !h || !i || !j || !k || !l || !m || !rawOK || !filesOK || !physicalOK || !metadataOK || !ok || physical != storage.physical || metadataRaw > g.bounds.claim.raw || metadataRaw > g.bounds.claim.metadata || metadataPhysical > g.bounds.claim.retained || metadataPhysical > g.bounds.claim.live || !providerContinuingWorkFits(metadataWork, g.bounds.claim.work) {
		return empty, false
	}
	reservation := providerOperationReservation{work: work, execution: execution, remaining: remaining, live: physical, retained: physical}
	for coordinate, amount := range work {
		if amount > s.firstLimits.operation[coordinate] || amount > s.firstLimits.segment[coordinate] {
			return empty, false
		}
	}
	memory, memoryOK := providerContinuingPinMemoryUpper(g, command, storage, metadataRaw, maximum, sizes)
	if !memoryOK {
		return empty, false
	}
	return providerContinuingPlan{reservation: reservation, physical: physical, consumption: consumption, settlement: settlement, metadataWork: metadataWork, metadataRaw: metadataRaw, metadataPhysical: metadataPhysical, memory: memory, headroom: headroom}, true
}
