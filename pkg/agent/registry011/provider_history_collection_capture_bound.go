package registry011

// The fixed class order is scalar/header/code/state/block/receipts/gas/execution,
// matching the explicitly selected method limits below. These are prospective
// collection grammar maxima, not a returned transcript or dispatch authority.
type providerHistoryCollectionCaptureCost struct {
	classes [8]uint64
	work    candidateWorkVector
}

// Successful collection BYTE-engine envelopes plus one irrevocable attempt per
// member only. First P, table construction/selection, independent original
// replay, storage, native dial/TLS verification, clocks, failure closure and
// final publication remain separate. This DATA quote is not a whole-source
// cost certificate, production factory or permission to start collection.
// Journal/code identities and the immutable manifest must be validated by that
// separate compiler; accepting scalar input here does not qualify its source.
func providerHistoryCollectionCaptureUpper(in providerHistoryCollectionInput, methods providerMethodBounds, certificates providerTLSBounds) (providerHistoryCollectionCaptureCost, bool) {
	var empty providerHistoryCollectionCaptureCost
	if in.blocks > 1<<20 || in.versions > 1<<20 || !methods.fits(in.capture) || !certificates.valid() {
		return empty, false
	}
	counts, ok := providerHistoryCollectionMaximumCounts(in)
	if !ok {
		return empty, false
	}
	switch in.encoding {
	case EthereumJSONBlocks, "debug-raw-rlp-v1":
	default:
		return empty, false
	}
	var a providerHistoryReadArithmetic
	windows := a.sum(counts.ancestryWindows, counts.contentWindows)
	classes := [8]uint64{
		a.product(4, windows),
		a.sum(a.difference(in.blocks, 1), a.product(6, windows)),
		a.product(uint64(len(in.codes)), windows),
		a.sum(a.product(2, windows), counts.contentWindows, in.presentStops, in.versions),
		in.blocks,
		in.blocks,
		0,
		in.effects,
	}
	// Gas evidence uses the actual selected tracer. Root-call profiles use
	// callTracer and therefore the execution class, not the smaller gas class.
	switch in.gas {
	case SepoliaAmsterdamGasProfile, SepoliaAmsterdamNethermindGasProfile:
		classes[6] = in.native
	case SepoliaAmsterdamRootRegularGasProfile, SepoliaAmsterdamRootExecutionGasProfile:
		classes[7] = a.sum(classes[7], in.native)
	default:
		return empty, false
	}
	requests := a.sum(classes[:]...)
	if a.failed || requests != counts.requests {
		return empty, false
	}
	limits := [8]providerMethodLimit{methods.scalar, methods.header, methods.code, methods.state, methods.block, methods.receipts, methods.gas, methods.execution}
	var work candidateWorkVector
	work[0] = requests // consumed by the original owner before dial, once/member
	for i, limit := range limits {
		// One byte is a numeric envelope endpoint, not a fabricated RPC. For
		// valid request length r, q(r)-q(1)=(r-1)*(r+4-2W)<=0 since r<W
		// and W>=3; W=2 permits only r=1. Wire is constant and response
		// bytes also attain their upper at r=1. No request is made here.
		capture := in.capture
		capture.BodyBytes, capture.WireBytes = limit.body, limit.wire
		one, ok := providerHistoryCaptureUpper(1, capture, certificates)
		if !ok {
			return empty, false
		}
		for coordinate, units := range one {
			work[coordinate] = a.sum(work[coordinate], a.product(classes[i], units))
		}
	}
	if a.failed {
		return empty, false
	}
	return providerHistoryCollectionCaptureCost{classes, work}, true
}
