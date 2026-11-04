package registry011

// Source-derived arguments for the one fresh opening/enrollment/inspection
// route. They are immutable numeric DATA, not an observed prefix, derivation,
// selected profile, store or permission to open a namespace. The eventual
// caller must use that same fresh route; this does not price recovery/resume.
type providerHistoryBootstrapSourceInputs struct {
	ordinal, records, retained, output uint64
}

func providerHistoryBootstrapInputsFromBounds(b providerIntegrationBounds) (providerHistoryBootstrapSourceInputs, bool) {
	var empty providerHistoryBootstrapSourceInputs
	if b.history == nil || b.guardian != nil {
		return empty, false
	}
	in, ok := providerBootstrapCostingInputFromBounds(b)
	if !ok || in.kind != providerBootstrapCostingHistory || in.scan.maximumRecords < 2 || in.scan.maximumChunks < 2 {
		return empty, false
	}
	// Fresh scan H0 has ordinal zero; claim is one-use. The sole E0 changes
	// the main ordinal from zero to one, so that inspection walks two records.
	// These constants come from the original route, not caller/UNIT counters.
	const ordinal, records uint64 = 1, 2
	opening, openOK := providerScanOpenCostUpper(in.maximumMetadataPerCommit, in.fileOverheadBytes)
	_, enrollmentPhysical, enrollOK := in.bootstrapUpper()
	preparation, prepareOK := providerScanPreparationCostUpper(in.maximumMetadataPerCommit, in.fileOverheadBytes, ordinal, in.scan.maximumAttempts)
	output, outputOK := providerScanBodyOutputUpper(in.maximumMetadataPerCommit, in.scan.maximumChunks, in.scan.filenameEntryOverhead, true)
	if !openOK || !enrollOK || !prepareOK || !outputOK {
		return empty, false
	}
	// Enrollment's original end check bounds retainedRaw by opening physical
	// plus its own physical quote. Keep those complete bounds, including
	// allocation overhead, rather than guessing successful encoded lengths.
	// The one claim adds seven metadata files. Each of the two record chunks
	// adds command+receipt copies in BOTH roots (four metadata files/record).
	var a providerHistoryReadArithmetic
	receipts := a.product(a.product(4, records), in.maximumMetadataPerCommit)
	retained := a.sum(opening.retained, enrollmentPhysical, preparation.work[9], receipts)
	if a.failed || retained > in.maximumStoredBytes {
		return empty, false
	}
	return providerHistoryBootstrapSourceInputs{ordinal, records, retained, output}, true
}

type providerHistoryCompiledBootstrapCost struct {
	inputs providerHistoryBootstrapSourceInputs
	source providerHistoryBootstrapSourceCost
}

// Bind the original prefix source calculation to its own derived maxima.
// Complete compiler/native/future/recovery/final qualification remains open;
// this result cannot mint the missing production derivation or execute I/O.
func providerHistoryCompileBootstrapSource(b providerIntegrationBounds, capture ProviderCaptureLimits) (providerHistoryCompiledBootstrapCost, bool) {
	inputs, ok := providerHistoryBootstrapInputsFromBounds(b)
	if !ok {
		return providerHistoryCompiledBootstrapCost{}, false
	}
	source, ok := providerHistoryBootstrapSourceUpper(b, capture, inputs.ordinal, inputs.records, inputs.retained, inputs.output)
	if !ok {
		return providerHistoryCompiledBootstrapCost{}, false
	}
	return providerHistoryCompiledBootstrapCost{inputs, source}, true
}
