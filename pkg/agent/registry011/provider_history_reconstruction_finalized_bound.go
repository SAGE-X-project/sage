package registry011

// Additional first-finalized selection from each completed original initial
// model. This does not fund that model, a new observation, cursor/root creation,
// native resources or authority. Original frames are looked up and hashed anew.
type providerHistoryReconstructionFinalizedCost struct {
	lookups, records, request, base64 candidateWorkVector
	rpc, header, comparisons          candidateWorkVector
	normal, incoming, envelope        candidateWorkVector
}

func providerHistoryReconstructionFinalizedUpper(l providerHistoryLegacyControlLimits, capture ProviderCaptureLimits, policyBytes uint64) (providerHistoryReconstructionFinalizedCost, bool) {
	var empty providerHistoryReconstructionFinalizedCost
	if !capture.valid() || policyBytes == 0 || policyBytes > 64<<10 {
		return empty, false
	}
	if _, ok := providerHistoryLegacyControlUpper(l); !ok {
		return empty, false
	}
	request, requestOK := providerHistoryLedgerControlUpper(1)
	base64, base64OK := providerHistoryBase64ControlUpper(uint64(capture.BodyBytes))
	header, headerOK := providerHistoryFirstPHeaderControlUpper(uint64(capture.BodyBytes))
	if !requestOK || !base64OK || !headerOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	q := func(n uint64) uint64 { return a.sum(a.product(n, n), 1) }
	p, body := l.objectBytes, uint64(capture.BodyBytes)
	settlement := min(p, uint64(64<<10))
	objects := min(l.firstObjects, a.product(4, l.entries))
	var out providerHistoryReconstructionFinalizedCost
	// Each pass retrieves exchange/request/settlement, scans its own immutable
	// frames, copies the selected bytes, hashes again and checks the digest.
	lookup := a.sum(a.product(objects, 2*26+2*66+2), p, 1)
	domains := uint64(len("sage-provider-exchange-v2") + len("sage-provider-budget-settlement-v2"))
	out.lookups[10] = a.product(2, a.sum(a.product(3, lookup), a.product(3, p), domains, 5, 3*133))
	// Exchange JSON plus the settlement's own raw and policy decodes. The
	// policy copy precedes its decode and is independent of object maximum P.
	out.records[2] = a.product(2, a.sum(p, settlement, policyBytes))
	out.records[10] = a.product(2, a.sum(q(p), q(settlement), q(policyBytes), settlement,
		uint64(len("sage-provider-budget-settlement-v2")), 2, settlement, 1, policyBytes, 1))
	for coordinate := range out.request {
		// This larger existing request-copy row includes the actual fixed
		// constructor's original8KiB/16KiB/512 encoder preflights, not a
		// successful-output-length substitution or another nextTable call.
		out.request[coordinate] = a.product(2, request.requestCopy[coordinate])
		out.base64[coordinate] = a.product(2, base64[coordinate])
		out.header[coordinate] = a.product(2, header[coordinate])
	}
	// Strict envelope parsing performs two whole-body decodes and separately
	// decodes jsonrpc BEFORE testing its version. Keep the full B bound for
	// that third codec even though the successful spelling is only five bytes.
	out.rpc[2] = a.product(6, body)
	out.rpc[10] = a.product(2, a.sum(a.product(3, q(body)), a.product(3, body), 3))
	// Exchange digest match; expected request phase-copy and raw comparison;
	// original manifest selection's four hash texts. Its returned header is
	// scalar DATA, so no copy event or native clock is invented for assignment.
	const wire uint64 = 16<<10 + 512
	out.comparisons[10] = a.product(2, a.sum(133, p, 2*wire+2, 265))
	for coordinate := range out.normal {
		out.normal[coordinate] = a.sum(out.lookups[coordinate], out.records[coordinate], out.request[coordinate],
			out.base64[coordinate], out.rpc[coordinate], out.header[coordinate], out.comparisons[coordinate])
	}
	// One outstanding primitive across both fresh selections. Base64's codec
	// consumes ENCODED bytes; preserve that wider bound before its raw decode.
	tail := [...]uint64{0, 2, 3}
	encoded := a.sum(a.product(body/3, 4), tail[body%3])
	out.incoming[2] = max(p, policyBytes, encoded, body, uint64(8192))
	out.incoming[10] = max(q(p), q(policyBytes), q(encoded), q(body), q(16<<10))
	for coordinate := range out.envelope {
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed || !providerHistoryReadWorkDomain(out.envelope) {
		return empty, false
	}
	return out, true
}
