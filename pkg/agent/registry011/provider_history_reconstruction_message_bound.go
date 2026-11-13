package registry011

// The initial model reinterprets its first request at admission and again in
// consume/settle oldMaterials, then checks one exchange. These are TWO fresh
// read-only passes, not producer requests or the later finalized selector.
// Body/ledger/index/retention work, native resources and authority are separate.
type providerHistoryReconstructionMessageCost struct {
	lookups, requests, records, base64 candidateWorkVector
	rpc, http, transport               candidateWorkVector
	requestCalls, exchangeCalls        uint64
	bindingCandidates, objectLookups   uint64
	normal, incoming, envelope         candidateWorkVector
}

func providerHistoryReconstructionMessageUpper(l providerHistoryLegacyControlLimits, capture ProviderCaptureLimits, policyBytes uint64) (providerHistoryReconstructionMessageCost, bool) {
	var empty providerHistoryReconstructionMessageCost
	if !capture.valid() || policyBytes == 0 || policyBytes > 64<<10 {
		return empty, false
	}
	if _, ok := providerHistoryLegacyControlUpper(l); !ok {
		return empty, false
	}
	control, controlOK := providerHistoryLedgerControlUpper(1)
	response, responseOK := providerHistoryResponseControlUpper(capture)
	request64, requestOK := providerHistoryBase64ControlUpper(16 << 10)
	response64, response64OK := providerHistoryBase64ControlUpper(uint64(capture.BodyBytes))
	if !controlOK || !responseOK || !requestOK || !response64OK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	q := func(n uint64) uint64 { return a.sum(a.product(n, n), 1) }
	const params, body, head, wire, transport uint64 = 8192, 16384, 512, 16896, 1 << 20
	p, b := l.objectBytes, uint64(capture.BodyBytes)
	objects := min(l.firstObjects, a.product(4, l.entries))
	requestRaw, paramsRaw := min(p, body), min(p, params)
	scan := a.product(objects, 2*26+2*66+2)
	out := providerHistoryReconstructionMessageCost{requestCalls: 6, exchangeCalls: 2, bindingCandidates: 4, objectLookups: 24}
	// Each request has two frame lookups. Each exchange has exchange/request/
	// TLS/response plus TWO binding candidates: the second is decoded before
	// selected!=nil refuses. Price that whole prefix, not just its last codec.
	// m.object observes scan, phase-copy and a separate copy(1) use flag; it
	// does not call an object parser or hash. The binding search scans again.
	out.lookups[10] = a.product(2, a.sum(a.product(12, scan), a.product(4, requestRaw),
		a.product(3, paramsRaw), a.product(5, p), 36, a.product(39, objects)))
	// Raw request normalization visits the full decoded16KiB tree BEFORE the
	// closed builder's513th-node/8KiB guard. Keep both body codecs, UTF/unicode
	// checks and that wider normalization walk. The existing conservative
	// requestCopy row preserves all builder encoder preflights/intermediates;
	// its extra params codec is harmless over-reservation, not actual decoding.
	finalized := a.sum(2*32+1, 2*params+1, 2*body+1, q(head), head, 1,
		2*head+1, wire, 1, 2*wire+1)
	request := a.sum(control.requestCopy[10], a.product(2, q(body)), a.product(4, body+1),
		wire, 1, body, wire, 1, finalized, 5*params+4)
	out.requests[2] = a.product(6, a.sum(control.requestCopy[2], 2*body))
	out.requests[10] = a.product(6, request)
	// Exchange and up to two wire-binding JSON objects are independently
	// decoded at P. The separate original policy copy precedes its own decode.
	// Binding exchange text can be P bytes before its equality refusal.
	out.records[2] = a.product(2, a.sum(a.product(3, p), policyBytes))
	out.records[10] = a.product(2, a.sum(a.product(3, q(p)), q(policyBytes), policyBytes, 1,
		a.product(2, a.sum(p, 67))))
	for coordinate := range out.base64 {
		out.base64[coordinate] = a.product(2, a.sum(request64[coordinate], response64[coordinate]))
		out.http[coordinate] = a.product(2, response.http[coordinate])
	}
	// exchange decodes the RPC response independently of response's HTTP
	// parser. That HTTP row owns its own THREE codecs; these three are extra.
	out.rpc[2] = a.product(6, b)
	out.rpc[10] = a.product(2, a.sum(a.product(3, q(b)), a.product(3, b), 3))
	// One TLS metadata and two header encodings keep their original1MiB
	// preflights. Include TLS hash and comparison against up to P retained
	// bytes, request/response-body comparisons, parsed-header slot copy and
	// the independently encoded header-array comparison. No live TLS occurs.
	out.transport[10] = a.product(2, a.sum(a.product(3, q(transport)), 4*transport, p,
		2*body, a.product(2, b), 134))
	for coordinate := range out.normal {
		out.normal[coordinate] = a.sum(out.lookups[coordinate], out.requests[coordinate], out.records[coordinate],
			out.base64[coordinate], out.rpc[coordinate], out.http[coordinate], out.transport[coordinate])
	}
	tail := [...]uint64{0, 2, 3}
	encoded := a.sum(a.product(b/3, 4), tail[b%3])
	// One outstanding primitive across all six requests and both exchanges.
	// An encoder preflight can be larger than P or a successful output.
	out.incoming[2] = max(p, policyBytes, encoded, uint64(21846), uint64(capture.WireBytes), uint64(capture.HeaderBytes), body)
	out.incoming[10] = max(q(out.incoming[2]), q(transport))
	for coordinate := range out.envelope {
		out.envelope[coordinate] = a.sum(out.normal[coordinate], out.incoming[coordinate])
	}
	if a.failed || !providerHistoryReadWorkDomain(out.envelope) {
		return empty, false
	}
	return out, true
}
