package registry011

// Original opaque pack bytes and the fixed small descriptor only. Request
// table semantics, TLS/result contexts and HTTP/RPC results are separate.
// The parsed row includes one bounded original caller text-to-byte copy.
type providerHistoryPackControlCost struct{ built, parsed candidateWorkVector }

func providerHistoryOriginalPackControlUpper(entries, maximum uint64) (providerHistoryPackControlCost, bool) {
	var empty providerHistoryPackControlCost
	if entries == 0 || entries > 2*providerHistoryGroupMembers || maximum <= uint64(len(providerHistoryOriginalPackMagic)+4) || maximum > providerHistoryOriginalPackBytes {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	const descriptor uint64 = providerHistoryOriginalPackHeaderBytes
	magic, domain := uint64(len(providerHistoryOriginalPackMagic)), uint64(len(providerHistoryOriginalPackDomain))
	q := a.sum(a.product(descriptor, descriptor), 1)
	var out providerHistoryPackControlCost
	// Three payload passes plus two complete raw passes; the small descriptor
	// retains its original64KiB encode preflight even if maximum is smaller.
	out.built[10] = a.sum(q, a.product(5, maximum), descriptor, a.product(3, entries), magic, domain, 22)
	out.parsed[2] = a.sum(descriptor, 4)
	out.parsed[10] = a.sum(a.product(2, q), a.product(4, maximum), a.product(2, descriptor), a.product(134, entries), a.product(3, magic), domain, 157)
	if a.failed {
		return empty, false
	}
	return out, true
}

// In-memory original response parsing only. Read events here are logical
// parser work, not another network request or retained-file read. member adds
// response/TLS hashes and result bindings, but EXCLUDES the separately priced
// TLS parser (which already owns its first caller text-to-byte copy).
type providerHistoryResponseControlCost struct{ http, member candidateWorkVector }

func providerHistoryResponseControlUpper(limits ProviderCaptureLimits) (providerHistoryResponseControlCost, bool) {
	var empty providerHistoryResponseControlCost
	if !limits.valid() {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	h, b, w := uint64(limits.HeaderBytes), uint64(limits.BodyBytes), uint64(limits.WireBytes)
	quadratic := func(n uint64) uint64 { return a.sum(a.product(n, n), 1) }
	var out providerHistoryResponseControlCost
	// Include H+1 header read sentinel, B+1 body preflight, EOF probe, two
	// byte checks and THREE body-sized codecs. jsonrpc's raw value is decoded
	// BEFORE checking version==2.0; five assumed bytes would underfund it.
	out.http[2] = a.sum(h, w, a.product(3, b))
	out.http[10] = a.sum(a.product(13, h), a.product(4, b), 20,
		quadratic(h), quadratic(w), a.product(3, quadratic(b)))
	out.member = out.http
	out.member[10] = a.sum(out.member[10], a.product(2, w), 2*(64<<10), a.product(2, b),
		quadratic(candidateLedgerMetadataLimit), candidateLedgerMetadataLimit, uint64(len(providerHistoryMemberResultDomain)), 8)
	if a.failed {
		return empty, false
	}
	return out, true
}

// Normal result-spool byte subgraph, composed with the separate diagnostic
// TLS/context component. Each current prefix builds and reparses its pack;
// archived successful prefixes only reparse the retained pack. The original
// G/R, object maximum and capture MUST be bound by the complete compiler.
// Other T/request packs, I/O, native CPU/RAM, failure/final and factory remain.
func providerHistoryCollectionRepeatedResultBytesUpper(groups, requests, maximum uint64, capture ProviderCaptureLimits) (candidateWorkVector, bool) {
	var empty candidateWorkVector
	calls, callsOK := providerHistoryCollectionResultContextCallsBound(groups, requests)
	pack, packOK := providerHistoryOriginalPackControlUpper(2*providerHistoryGroupMembers, maximum)
	response, responseOK := providerHistoryResponseControlUpper(capture)
	if !callsOK || !packOK || !responseOK {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	builds := a.sum(a.product(4, requests), groups)
	var work candidateWorkVector
	for coordinate := range work {
		work[coordinate] = a.sum(a.product(builds, pack.built[coordinate]),
			a.product(calls.packParses, pack.parsed[coordinate]), a.product(calls.tlsParses, response.member[coordinate]))
	}
	if a.failed {
		return empty, false
	}
	return work, true
}
