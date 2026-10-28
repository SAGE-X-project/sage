package registry011

// Checked prospective DATA only. No store, clock, ledger or owner is received.
// A result cannot reserve, consume, retain, dispatch or restore a capability.
func providerContinuingPinResourceUpper(b providerIntegrationBounds, global ProviderCaptureLimits, requests [4]*providerIntegrationRequest, captures [4]ProviderCaptureLimits) (providerContinuingPlan, providerFirstOperationStorage, uint64, [4]uint64, bool) {
	var empty providerContinuingPlan
	if b.guardian == nil || b.methods == nil || b.tls == nil {
		return empty, providerFirstOperationStorage{}, 0, [4]uint64{}, false
	}
	storage, ok := providerFirstOperationStorageUpper(b, global, uint64(len(providerFirstOperationRequestWire)))
	if !ok || storage.exchange < 4096 {
		return empty, providerFirstOperationStorage{}, 0, [4]uint64{}, false
	}
	n, maximum := max(b.maximumMetadataPerCommit, uint64(64<<10)), uint64(1<<20)
	var sizes [4]uint64
	for i, request := range requests {
		capture := captures[i]
		if request == nil || !capture.valid() {
			return empty, providerFirstOperationStorage{}, 0, [4]uint64{}, false
		}
		child, valid := providerOperationStorageUpper(b, capture, uint64(len(request.wire)))
		base64, product := providerUpperProduct(uint64(capture.BodyBytes), 4)
		base64, sum := providerUpperSum(base64, 2)
		if !valid || !product || !sum || child.exchange > 4<<20 || child.transport > 4<<20 {
			return empty, providerFirstOperationStorage{}, 0, [4]uint64{}, false
		}
		sizes[i] = max(n, child.exchange, child.transport, uint64(capture.WireBytes), base64/3)
		maximum = max(maximum, sizes[i])
	}
	headroom, a := providerUpperProduct(maximum, maximum)
	headroom, c := providerUpperSum(headroom, b.maximumObjectBytes, 2)
	// The original43Q/76L child closure remains. First-read bounds add
	// separately funded4Q/12L per child in EACH independent E and R.
	// Two separate5Q/9L input rechecks cover C/S old-material invocations.
	execution, e := providerFirstOperationPhaseUpper(n, 655, 3595, 7, 29, storage.consumption, headroom)
	remaining, f := providerFirstOperationPhaseUpper(n, 594, 3327, 5, 37, storage.settlement, headroom)
	if !a || !c || !e || !f {
		return empty, providerFirstOperationStorage{}, 0, [4]uint64{}, false
	}
	validatorQ, validatorL := [4]uint64{1, 5, 1, 4}, [4]uint64{0, 8, 0, 7}
	for i, size := range sizes {
		row := []providerRPCSizeRow{{size, 47 + validatorQ[i], 88 + validatorL[i]}}
		capture := captures[i]
		head, h := providerUpperProduct(uint64(capture.HeaderBytes), 13)
		body, j := providerUpperProduct(uint64(capture.BodyBytes), 4)
		extra, k := providerUpperSum(head, body, 5, 9731, providerRPCClassRecursionUpper)
		if !h || !j || !k || !providerRPCAddSizeRows(&execution, row) || !providerRPCAddSizeRows(&remaining, row) {
			return empty, providerFirstOperationStorage{}, 0, [4]uint64{}, false
		}
		var left, right bool
		execution[10], left = providerUpperSum(execution[10], extra)
		remaining[10], right = providerUpperSum(remaining[10], extra)
		if !left || !right {
			return empty, providerFirstOperationStorage{}, 0, [4]uint64{}, false
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
	if err != nil || !h || !i || !j || !k || !l || !m || !rawOK || !filesOK || !physicalOK || !metadataOK || !ok || physical != storage.physical || metadataRaw > b.guardian.claim.raw || metadataRaw > b.guardian.claim.metadata || metadataPhysical > b.guardian.claim.retained || metadataPhysical > b.guardian.claim.live || !providerContinuingWorkFits(metadataWork, b.guardian.claim.work) {
		return empty, providerFirstOperationStorage{}, 0, [4]uint64{}, false
	}
	reservation := providerOperationReservation{work: work, execution: execution, remaining: remaining, live: physical, retained: physical}
	return providerContinuingPlan{reservation: reservation, physical: physical, consumption: consumption, settlement: settlement, metadataWork: metadataWork, metadataRaw: metadataRaw, metadataPhysical: metadataPhysical, headroom: headroom}, storage, maximum, sizes, true
}
