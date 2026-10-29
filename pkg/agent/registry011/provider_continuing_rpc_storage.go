package registry011

// Closed storage DATA for one selected method. The owner must derive capture
// from its own immutable request and original method profile before holding P.
// This arithmetic cannot authorize, reserve, consume or dispatch an operation.
func providerOperationStorageUpper(b providerIntegrationBounds, capture ProviderCaptureLimits, requestBytes uint64) (providerFirstOperationStorage, bool) {
	if b.firstAdmission == nil || !b.valid() || b.tls == nil || !b.tls.valid() || !capture.valid() || requestBytes == 0 || requestBytes > 16<<10 || requestBytes >= uint64(capture.WireBytes) {
		return providerFirstOperationStorage{}, false
	}
	// Capture's wire cap includes the already fixed request. Its remaining
	// response bytes form ONE separately retained object, not merely a term
	// in the aggregate disk quote. This subtracts wire bytes, never I/E1 funds.
	if uint64(capture.WireBytes)-requestBytes > b.maximumObjectBytes {
		return providerFirstOperationStorage{}, false
	}
	// JSON headers expand at most6 bytes per input byte. Each of at most128
	// parsed headers adds less than32 structural bytes. TLS identity retains
	// first-chain SHA256 digests, not DER: each quoted digest plus comma is69.
	headerBytes, a := providerUpperProduct(uint64(capture.HeaderBytes), 6)
	headerFields, c := providerUpperProduct(min(uint64(capture.HeaderBytes/4), 128), 32)
	digests, d := providerUpperProduct(uint64(b.tls.certificates), 69)
	// The remaining closed transport fields fit8KiB. The future serializer
	// must check that cap; it cannot use this quote as validation of sample DATA.
	transport, e := providerUpperSum(candidateLedgerMetadataLimit, headerBytes, headerFields, digests)
	// ceil(4*n/3) safely bounds unpadded base64url without subtracting any
	// already held first-E1 budget or response bytes from an earlier estimator.
	requestBase64, f := providerUpperProduct(requestBytes, 4)
	responseBase64, g := providerUpperProduct(uint64(capture.BodyBytes), 4)
	requestBase64, h := providerUpperSum(requestBase64, 2)
	responseBase64, i := providerUpperSum(responseBase64, 2)
	exchange, j := providerUpperSum(candidateLedgerMetadataLimit, transport, requestBase64/3, responseBase64/3)
	metadata, k := providerUpperProduct(b.maximumMetadataPerCommit, 3)
	// C:7 fresh objects and7 main/head files. S:11 fresh objects and7 main/
	// head files. Consumption, settlement and wire-binding each use8KiB caps;
	// E1/request/params are absent. X includes its own transport copy, while
	// the separately retained TLS object is charged again in S.
	consumption, l := providerUpperSum(8*candidateLedgerMetadataLimit, 3*(64<<10), metadata)
	settlement, m := providerUpperSum(9*candidateLedgerMetadataLimit, 3*(64<<10), metadata, exchange, uint64(capture.WireBytes), transport)
	raw, n := providerUpperSum(consumption, settlement)
	allocation, o := providerUpperProduct(b.fileOverheadBytes, 32)
	physical, p := providerUpperSum(raw, allocation)
	if !a || !c || !d || !e || !f || !g || !h || !i || !j || !k || !l || !m || !n || !o || !p || transport > b.maximumObjectBytes || exchange > b.maximumObjectBytes || physical > b.maximumStoredBytes {
		return providerFirstOperationStorage{}, false
	}
	return providerFirstOperationStorage{consumption, settlement, transport, exchange, raw, physical}, true
}
