package registry011

import "reflect"

// Keep all four earlier child peaks and all three request-construction
// phases. No immediate GC or physical allocator qualification is inferred.
func providerContinuingPinMemoryUpper(g *providerGuardianOwner, command *providerContinuingPinCommand, storage providerFirstOperationStorage, metadataRaw, maximum uint64, sizes [4]uint64) (uint64, bool) {
	if g == nil || g.store == nil || g.integration == nil || g.integration.ledger == nil || command == nil || g.config.bounds.continuation == nil || g.config.bounds.scan == nil || g.config.bounds.tls == nil {
		return 0, false
	}
	s, b, ledger := g.store, g.config.bounds, g.integration.ledger
	var historical, childScratch uint64
	for i, request := range command.requests {
		if request == nil || !command.captures[i].valid() || sizes[i] == 0 {
			return 0, false
		}
		capture := command.captures[i]
		oldPlan, oldStorage, oldMaximum, ok := providerContinuingRPCResourceUpper(b, capture, uint64(len(request.wire)))
		if !ok {
			return 0, false
		}
		peak, valid := providerContinuingRPCMemoryUpper(g, request, capture, oldStorage, oldPlan.metadataRaw, oldMaximum)
		var added bool
		historical, added = providerUpperSum(historical, peak)
		if !valid || !added {
			return 0, false
		}
		// X/S originals, decoded models, base64/HTTP/JSON/canonical scratch,
		// and the G, E and R expected request strings can all remain allocated.
		originals, a := providerUpperProduct(sizes[i], 2)
		decoded, c := providerUpperProduct(sizes[i], b.continuation.codecMemoryMultiplier)
		intermediate, e := providerUpperProduct(sizes[i], 12)
		wire, f := providerUpperProduct(uint64(capture.WireBytes), 3)
		head, h := providerUpperProduct(uint64(capture.HeaderBytes), 3)
		body, j := providerUpperProduct(uint64(capture.BodyBytes), 8)
		requests, k := providerUpperProduct(uint64(len(request.wire)), 3)
		classScratch, l := providerUpperProduct(2*(16<<10)+(8<<10), 2)
		value, m := providerUpperSum(originals, decoded, intermediate, wire, head, body, requests, classScratch)
		childScratch, added = providerUpperSum(childScratch, value)
		if !a || !c || !e || !f || !h || !j || !k || !l || !m || !added {
			return 0, false
		}
	}
	retained, a := providerUpperSum(s.retainedRaw, storage.raw, metadataRaw)
	combined, c := providerUpperSum(storage.raw, metadataRaw)
	raw, e := providerUpperProduct(combined, 5)
	metadata, f := providerUpperProduct(b.maximumMetadataPerCommit, 16)
	certificates, h := providerUpperProduct(uint64(b.tls.chains), uint64(b.tls.certificates))
	der, j := providerUpperProduct(certificates, uint64(b.tls.certificateBytes))
	digests, k := providerUpperProduct(certificates, max(uint64(69), b.continuation.objectMemoryOverhead))
	scratch, l := providerUpperSum(raw, metadata, childScratch, 4*4096, der, digests, maximum)
	operations, m := providerUpperSum(s.sequence, 3)
	models, n := b.continuation.residentUpper(uint64(len(ledger.tickets))+1, operations, uint64(len(ledger.segments))+1, uint64(len(s.objects))+32, retained, scratch)
	entries, p := providerUpperSum(uint64(len(s.objects)), 32, 3*32, 6*3, 256)
	collections, q := providerUpperProduct(entries, max(b.continuation.objectMemoryOverhead, b.scan.filenameEntryOverhead))
	var native uint64
	for _, row := range []struct{ bytes, count uint64 }{
		{uint64(reflect.TypeFor[providerContinuingOwner]().Size()), 1},
		{uint64(reflect.TypeFor[providerContinuingPinCommand]().Size()), 1},
		{uint64(reflect.TypeFor[providerContinuingPlan]().Size()), 3},
		{uint64(reflect.TypeFor[providerWorkMeter]().Size()), 2},
		{uint64(reflect.TypeFor[providerIntegrationPermit]().Size()), 1},
		{uint64(reflect.TypeFor[providerFirstOperationChange]().Size()), 2},
		{uint64(reflect.TypeFor[providerIndexDelta]().Size()), 2},
		{uint64(reflect.TypeFor[providerGuardianClaimData]().Size()), 2},
		{uint64(reflect.TypeFor[providerGuardianAdmissionBinding]().Size()), 2},
		{uint64(reflect.TypeFor[providerContinuingPinData]().Size()), 1},
		{uint64(reflect.TypeFor[providerIntegrationBounds]().Size()), 1},
		{uint64(reflect.TypeFor[ProviderCaptureLimits]().Size()), 1},
		{uint64(reflect.TypeFor[[4]*providerIntegrationRequest]().Size()), 1},
		{uint64(reflect.TypeFor[[4]ProviderCaptureLimits]().Size()), 1},
		{uint64(reflect.TypeFor[[4]uint64]().Size()), 2},
		{uint64(reflect.TypeFor[providerIntegrationRequest]().Size()), 20},
		{uint64(reflect.TypeFor[CandidateAdmissionSettlement]().Size()), 18},
		{uint64(reflect.TypeFor[providerMeasuredWork]().Size()), 1},
		{uint64(reflect.TypeFor[FinalizedHeader]().Size()), 2},
	} {
		bytes, ok := providerUpperProduct(row.bytes, row.count)
		var sum bool
		native, sum = providerUpperSum(native, bytes)
		if !ok || !sum {
			return 0, false
		}
	}
	total, valid := providerUpperSum(historical, models, collections, native)
	return total, a && c && e && f && h && j && k && l && m && n && p && q && valid && total <= b.scan.maximumResidentBytes && total <= g.bounds.claim.resident && total <= b.continuation.maximumResidentBytes
}
