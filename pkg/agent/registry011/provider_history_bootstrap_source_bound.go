package registry011

// Source components for the original pre-I and first-P prefix. These DATA do
// not close future stages, recovery/final/cleanup, compiler copies or native
// peak/lifetime/backing costs, and cannot mint a history derivation or owner.
type providerHistoryBootstrapSourceCost struct {
	opening, enrollment, selectedSetup providerScanUpper
	preparation, inspection, first     providerScanUpper
	body, preI, selectedPreI, startup  providerScanUpper
	firstOperationResident             uint64
}

// The selected structural inputs must be privately frozen before this call.
// Snapshotting values here does not establish atomicity against a concurrent
// mutation of the original profile. Runtime wrappers still require full
// validity, while this pure compilation precedes that finished derivation.
func providerHistoryBootstrapSourceUpper(b providerIntegrationBounds, capture ProviderCaptureLimits, ordinal, records, retained, output uint64) (providerHistoryBootstrapSourceCost, bool) {
	var empty providerHistoryBootstrapSourceCost
	if b.history == nil || b.guardian != nil {
		return empty, false
	}
	in, ok := providerBootstrapCostingInputFromBounds(b)
	if !ok || in.kind != providerBootstrapCostingHistory {
		return empty, false
	}
	opening, openingOK := providerScanOpenCostUpper(in.maximumMetadataPerCommit, in.fileOverheadBytes)
	enrollmentWork, enrollmentBytes, enrollmentOK := in.bootstrapUpper()
	// The original enrollment independently requires room for eight samples;
	// these are additional to the opening's sixteen-sample source envelope.
	enrollment := providerScanUpper{work: enrollmentWork, retained: enrollmentBytes, clocks: 8}
	setupNeed, setupOK := providerScanUpperSum(opening, enrollment)
	setup := providerScanUpper{work: b.setupWork, retained: b.setupRetained, clocks: b.setupMaximumClockReads}
	if !openingOK || !enrollmentOK || !setupOK || !providerHistoryUpperWithin(setupNeed, setup) {
		return empty, false
	}
	preparation, preparationOK := providerScanSelectedPreparationCostUpper(in.maximumMetadataPerCommit, in.fileOverheadBytes, ordinal, in.scan.maximumAttempts, in.historyCollection)
	inspection, inspectionOK := providerScanInspectionCostUpper(in.scan, in.maximumMetadataPerCommit, in.fileOverheadBytes, true, setup)
	firstPlan, firstOK := in.firstAdmissionPlanUpper(ordinal, records, retained, output)
	operation, operationOK := in.firstOperationPlanUpper(capture, ordinal, records, retained, output)
	if !preparationOK || !inspectionOK || !firstOK || !operationOK {
		return empty, false
	}
	firstNeed := providerScanUpper{work: firstPlan.work, retained: firstPlan.physical, clocks: firstPlan.clocks}
	for coordinate, amount := range operation.firstExtra {
		var fits bool
		firstNeed.work[coordinate], fits = providerUpperSum(firstNeed.work[coordinate], amount)
		if !fits {
			return empty, false
		}
	}
	first := *b.firstAdmission
	selectedFirst := providerScanUpper{work: first.work, retained: first.retained, clocks: first.clocks}
	if !providerHistoryUpperWithin(firstNeed, selectedFirst) {
		return empty, false
	}
	body, bodyOK := providerScanBodyCostUpper(in.scan, setup, inspection, selectedFirst)
	preI, preIOK := providerScanUpperSum(setup, preparation, body)
	selectedPreI := providerScanUpper{work: in.scan.maximumPreIWork, retained: in.scan.maximumPreIRetainedBytes, clocks: in.scan.maximumPreIClockReads}
	if !bodyOK || !preIOK || !providerHistoryUpperWithin(preI, selectedPreI) {
		return empty, false
	}
	// C/X and S/R retain their separate seven- and five-sample envelopes.
	// firstExtra belongs to F above and is not charged again to this first P.
	startup := providerScanUpper{work: operation.reservation.work, retained: operation.storage.physical, clocks: 12}
	return providerHistoryBootstrapSourceCost{
		opening: opening, enrollment: enrollment, selectedSetup: setup,
		preparation: preparation, inspection: inspection, first: firstNeed,
		body: body, preI: preI, selectedPreI: selectedPreI, startup: startup,
		firstOperationResident: operation.memory,
	}, true
}
