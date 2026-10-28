package registry011

import "reflect"

// Retain the old local/first peak as well as both current models and every
// overlapping new native/raw/canonical/TLS family. This is prospective DATA,
// never evidence of a qualified allocator, clock or physical filesystem.
func providerContinuingRPCMemoryUpper(g *providerGuardianOwner, request *providerIntegrationRequest, capture ProviderCaptureLimits, storage providerFirstOperationStorage, metadataRaw, maximum uint64) (uint64, bool) {
	if g == nil || g.store == nil || g.integration == nil || g.integration.ledger == nil || request == nil || !capture.valid() || g.config.bounds.continuation == nil || g.config.bounds.scan == nil || g.config.bounds.tls == nil {
		return 0, false
	}
	old, oldOK := providerContinuingBootstrapPlan(g)
	if !oldOK {
		return 0, false
	}
	b, s, ledger := g.config.bounds, g.store, g.integration.ledger
	retained, a := providerUpperSum(s.retainedRaw, storage.raw, metadataRaw)
	combined, c := providerUpperSum(storage.raw, metadataRaw)
	raw, e := providerUpperProduct(combined, 5)
	metadata, f := providerUpperProduct(b.maximumMetadataPerCommit, 16)
	wire, h := providerUpperProduct(uint64(capture.WireBytes), 3)
	head, j := providerUpperProduct(uint64(capture.HeaderBytes), 3)
	body, k := providerUpperProduct(uint64(capture.BodyBytes), 8)
	requests, l := providerUpperProduct(uint64(len(request.wire)), 4)
	requestScratch, m := providerUpperSum(requests, 2*(16<<10), 8<<10)
	certificates, p := providerUpperProduct(uint64(b.tls.chains), uint64(b.tls.certificates))
	der, q := providerUpperProduct(certificates, uint64(b.tls.certificateBytes))
	digests, t := providerUpperProduct(certificates, max(uint64(69), b.continuation.objectMemoryOverhead))
	scratch, u := providerUpperSum(raw, metadata, wire, head, body, requestScratch, der, digests, maximum)
	operations, v := providerUpperSum(s.sequence, 3)
	models, x := b.continuation.residentUpper(uint64(len(ledger.tickets))+1, operations, uint64(len(ledger.segments))+1, uint64(len(s.objects))+32, retained, scratch)
	entries, y := providerUpperSum(uint64(len(s.objects)), 32, 3*32, 6*3, 256)
	collections, z := providerUpperProduct(entries, max(b.continuation.objectMemoryOverhead, b.scan.filenameEntryOverhead))
	var native uint64
	for _, row := range []struct{ bytes, count uint64 }{
		{uint64(reflect.TypeFor[providerContinuingOwner]().Size()), 1},
		{uint64(reflect.TypeFor[providerContinuingPlan]().Size()), 2},
		{uint64(reflect.TypeFor[providerWorkMeter]().Size()), 2},
		{uint64(reflect.TypeFor[providerIntegrationPermit]().Size()), 1},
		{uint64(reflect.TypeFor[providerFirstOperationChange]().Size()), 2},
		{uint64(reflect.TypeFor[providerIndexDelta]().Size()), 2},
		{uint64(reflect.TypeFor[providerGuardianClaimData]().Size()), 2},
		{uint64(reflect.TypeFor[providerGuardianAdmissionBinding]().Size()), 2},
		{uint64(reflect.TypeFor[providerContinuingRPCData]().Size()), 1},
		{uint64(reflect.TypeFor[providerIntegrationRequest]().Size()), 2},
		{uint64(reflect.TypeFor[ProviderTransportSample]().Size()), 1},
	} {
		amount, ok := providerUpperProduct(row.bytes, row.count)
		var fits bool
		native, fits = providerUpperSum(native, amount)
		if !ok || !fits {
			return 0, false
		}
	}
	total, totalOK := providerUpperSum(models, collections, native, old.memory)
	return total, a && c && e && f && h && j && k && l && m && p && q && t && u && v && x && y && z && totalOK && total <= b.scan.maximumResidentBytes && total <= g.bounds.claim.resident
}
