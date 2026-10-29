package registry011

// Checked quote DATA only. This helper receives no owner, ledger, clock or
// filesystem. Its result cannot reserve, consume, retain or dispatch work.
func providerContinuingRPCResourceUpper(b providerIntegrationBounds, capture ProviderCaptureLimits, requestBytes uint64) (providerContinuingPlan, providerFirstOperationStorage, uint64, bool) {
	var empty providerContinuingPlan
	if b.guardian == nil {
		return empty, providerFirstOperationStorage{}, 0, false
	}
	storage, ok := providerOperationStorageUpper(b, capture, requestBytes)
	if !ok || storage.exchange > 4<<20 || storage.transport > 4<<20 {
		return empty, providerFirstOperationStorage{}, 0, false
	}
	n := max(b.maximumMetadataPerCommit, uint64(64<<10))
	maximum := max(uint64(1<<20), n, storage.transport, storage.exchange)
	headroom, a := providerUpperProduct(maximum, maximum)
	headroom, c := providerUpperSum(headroom, b.maximumObjectBytes, 2)
	// E small: common388, typed30, C builder27, coordinator100,
	// two composite80, affected6, X metadata3, bound-class recheck1.
	execution, e := providerFirstOperationPhaseUpper(n, 388+30+27+100+80+6+3+1, 2580+100+134+250+32+448+12+5+1, 7, 29, storage.consumption, headroom)
	// R small excludes common's two X formats, builder's one X format,
	// and result's twelve large codecs. Their actual large rows follow below.
	remaining, f := providerFirstOperationPhaseUpper(n, 386+30+32+100+2+6+6+5, 2580+100+306+250+32+5+11+12+11, 5, 37, storage.settlement, headroom)
	body, wire, head := uint64(capture.BodyBytes), uint64(capture.WireBytes), uint64(capture.HeaderBytes)
	b64, mulOK := providerUpperProduct(body, 4)
	b64, sumOK := providerUpperSum(b64, 2)
	b64 /= 3
	extraE := providerRPCAddSizeRows(&execution, []providerRPCSizeRow{{storage.exchange, 2, 3}, {body, 1, 0}, {storage.transport, 1, 1}, {wire, 0, 1}})
	extraR := providerRPCAddSizeRows(&remaining, []providerRPCSizeRow{{storage.exchange, 4, 18}, {b64, 1, 1}, {body, 5, 10}, {wire, 1, 18}, {head, 1, 0}, {storage.transport, 3, 21}})
	captureUpper, captureOK := providerFirstOperationCaptureUpper(requestBytes, capture, *b.tls)
	if !a || !c || !e || !f || !mulOK || !sumOK || !extraE || !extraR || !captureOK {
		return empty, providerFirstOperationStorage{}, 0, false
	}
	for i, value := range captureUpper {
		var fits bool
		execution[i], fits = providerUpperSum(execution[i], value)
		if !fits {
			return empty, providerFirstOperationStorage{}, 0, false
		}
	}
	execution[0], execution[8] = 1, 1
	headers, headerOK := providerUpperProduct(head, 13)
	remaining[10], ok = providerUpperSum(remaining[10], headers, 2, providerRPCClassRecursionUpper)
	work, workErr := providerFrontierWorkTotal(execution, remaining)
	if !headerOK || !ok || workErr != nil {
		return empty, providerFirstOperationStorage{}, 0, false
	}
	mainRaw, mainOK := providerUpperSum(10*candidateLedgerMetadataLimit, 16<<10, 8<<10, 64<<10, 4*b.maximumMetadataPerCommit, 2*candidateLedgerMetadataLimit)
	metadataRaw, rawOK := providerUpperSum(mainRaw, 5*candidateLedgerMetadataLimit)
	files, filesOK := providerUpperProduct(22, b.fileOverheadBytes)
	metadataPhysical, physicalOK := providerUpperSum(metadataRaw, files)
	// Existing G/T rows plus request builder3Q/class1Q and the two extra
	// ordinary-content objects' explicit bounded copies/order/name rows352L.
	metadataWork, metadataOK := providerGuardianAddedWork(b, 552+128+64+4, 3106+1024+512+352+9, metadataRaw)
	// Two independently replayed request normalizations plus the new live
	// builder recursion are disjoint from fixed metadata codec/linear rows.
	metadataWork[10], ok = providerUpperSum(metadataWork[10], 2*providerRPCClassRecursionUpper, 1026+8705, headroom)
	if !mainOK || !rawOK || !filesOK || !physicalOK || !metadataOK || !ok || metadataRaw > b.guardian.claim.raw || metadataRaw > b.guardian.claim.metadata ||
		metadataPhysical > b.guardian.claim.retained || metadataPhysical > b.guardian.claim.live || !providerContinuingWorkFits(metadataWork, b.guardian.claim.work) {
		return empty, providerFirstOperationStorage{}, 0, false
	}
	return providerContinuingPlan{reservation: providerOperationReservation{work: work, execution: execution, remaining: remaining, live: storage.physical, retained: storage.physical}, physical: storage.physical, consumption: storage.consumption, settlement: storage.settlement, metadataWork: metadataWork, metadataRaw: metadataRaw, metadataPhysical: metadataPhysical, headroom: headroom}, storage, maximum, true
}
