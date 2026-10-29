package registry011

import "reflect"

// Compile-time byte length only. The observed formatter is compared to these
// exact bytes before I; this literal is neither a credential nor a request.
const providerFirstOperationRequestBody = `{"jsonrpc":"2.0","id":1,"method":"eth_getBlockByNumber","params":["finalized",false]}`
const providerFirstOperationRequestHead = "POST / HTTP/1.1\r\nHost: rpc.sepolia.ethpandaops.io\r\nContent-Type: application/json\r\nAccept: application/json\r\nAccept-Encoding: identity\r\nConnection: close\r\nContent-Length: 85\r\n\r\n"
const providerFirstOperationRequestWire = providerFirstOperationRequestHead + providerFirstOperationRequestBody

type providerFirstOperationPlan struct {
	reservation providerOperationReservation
	storage     providerFirstOperationStorage
	capture     ProviderCaptureLimits
	firstExtra  candidateWorkVector
	headroom    uint64
	memory      uint64
}

func (*providerFirstOperationPlan) UnmarshalJSON([]byte) error { return errProviderIntegration }

// The independently reviewed callsite rows are prospective logical uppers.
// They never stand for measured work. Extra coordinator index traversal and
// bounded model/output comparisons are disjoint from the custodian's rows.
func providerFirstOperationPhaseUpper(n, q, linear, clocks, syncs, raw, headroom uint64) (candidateWorkVector, bool) {
	var work candidateWorkVector
	square, a := providerUpperProduct(n, n)
	quadratic, b := providerUpperSum(square, 1)
	bytes, c := providerUpperProduct(n, 2)
	bytes, d := providerUpperSum(bytes, 101)
	codec, e := providerUpperProduct(q, quadratic)
	primitive, f := providerUpperProduct(linear, bytes)
	decoded, g := providerUpperProduct(q, n)
	units, h := providerUpperSum(codec, primitive, 84998, clocks, syncs, 1, headroom)
	if !a || !b || !c || !d || !e || !f || !g || !h {
		return work, false
	}
	work[2], work[9], work[10] = decoded, raw, units
	return work, true
}

func providerFirstOperationPlanUpper(b providerIntegrationBounds, global ProviderCaptureLimits, ordinal, records, retained, output uint64) (providerFirstOperationPlan, bool) {
	var p providerFirstOperationPlan
	if !b.valid() || b.firstAdmission == nil || b.maximumObjects < providerInitialObjectCount(b)+31 || b.maximumEntries < 14 {
		return p, false
	}
	storage, ok := providerFirstOperationStorageUpper(b, global, uint64(len(providerFirstOperationRequestWire)))
	if !ok {
		return p, false
	}
	capture, err := b.methods.header.capture(global, len(providerFirstOperationRequestWire))
	if err != nil {
		return p, false
	}
	maximum := max(uint64(1<<20), b.maximumMetadataPerCommit, storage.transport, storage.exchange)
	headroom, a := providerUpperProduct(maximum, maximum)
	headroom, c := providerUpperSum(headroom, 1, b.maximumObjectBytes, 1)
	n := max(b.maximumMetadataPerCommit, uint64(64<<10), b.maximumObjectBytes, uint64(capture.WireBytes))
	if b.guardian != nil {
		// This first closed graph accepts only its typed E0/E1/C/X/S
		// schemas and the method-specific header transport/exchange sizes.
		// A larger global object ceiling belongs to later methods. Every
		// actual decoder still preflights its own length and fails closed.
		n = max(b.maximumMetadataPerCommit, uint64(64<<10), storage.transport, storage.exchange, uint64(capture.WireBytes))
	}
	// C/X:452Q/2824L, additional coordinator100Q/250L, and32 bounded
	// model/output/literal comparisons. R has its distinct S/result graph.
	execution, e := providerFirstOperationPhaseUpper(n, 552, 3106, 7, 29, storage.consumption, headroom)
	// The retained binding index adds two original decodes and five linear
	// events: reference/path copies, read, domain hash and exchange comparison.
	remaining, f := providerFirstOperationPhaseUpper(n, 571, 3284, 5, 37, storage.settlement, headroom)
	captureUpper, g := providerFirstOperationCaptureUpper(uint64(len(providerFirstOperationRequestWire)), capture, *b.tls)
	if !a || !c || !e || !f || !g {
		return p, false
	}
	for i, amount := range captureUpper {
		var fits bool
		execution[i], fits = providerUpperSum(execution[i], amount)
		if !fits {
			return p, false
		}
	}
	execution[0], execution[8] = 1, 1
	head, h := providerUpperProduct(uint64(global.HeaderBytes), 13)
	body, i := providerUpperProduct(uint64(global.BodyBytes), 4)
	remaining[10], ok = providerUpperSum(remaining[10], head, body, 5)
	if b.guardian != nil {
		// The V5 retained result derives its request's selected method class.
		// Fund that additional independent small reconstruction before I.
		class, classOK := providerRPCClassExtraUpper(n)
		if !classOK {
			return p, false
		}
		for i, amount := range class {
			var fits bool
			remaining[i], fits = providerUpperSum(remaining[i], amount)
			if !fits {
				return p, false
			}
		}
	}
	total, reservationErr := providerFrontierWorkTotal(execution, remaining)
	if !h || !i || !ok || reservationErr != nil {
		return p, false
	}
	// This closed32-file quote already includes every temporary copy. The
	// legacy frontier estimator's additional doubling is a different metric.
	reservation := providerOperationReservation{work: total, execution: execution, remaining: remaining, live: storage.physical, retained: storage.physical}
	first, firstOK := providerFirstAdmissionPlanUpper(b, ordinal, records, retained, output)
	// Separate provisional apply44Q/39L, conservatively100/250; four
	// newly observed segment encodes/hashes and32 bounded model comparisons.
	extraN := max(b.maximumMetadataPerCommit, uint64(64<<10))
	extra, extraOK := providerFirstOperationPhaseUpper(extraN, 104, 286, 0, 0, 0, 0)
	// That helper's Ureq/rename are not part of this extra F graph. Remove
	// their fixed conservative allowance; no actual counter is subtracted.
	if !firstOK || !extraOK || extra[10] < 84999 {
		return p, false
	}
	extra[10] -= 84999
	memory, memoryOK := providerFirstOperationMemoryUpper(b, capture, storage, first, retained, output, maximum)
	if !memoryOK {
		return p, false
	}
	p = providerFirstOperationPlan{reservation, storage, capture, extra, headroom, memory}
	return p, true
}

func providerFirstOperationMemoryUpper(b providerIntegrationBounds, capture ProviderCaptureLimits, storage providerFirstOperationStorage, first providerFirstAdmissionPlan, retained, output, maximum uint64) (uint64, bool) {
	if b.continuation == nil || b.scan == nil || b.tls == nil || !capture.valid() {
		return 0, false
	}
	schemas, a := providerUpperSum(retained, first.raw, storage.raw, output)
	// Named simultaneous buffers: serialized strings+readback, temporary
	// delta encodes, live/replay wire, header, body/result, request, TLS DER
	// and identity, plus the largest declared serializer preflight scratch.
	raw, c := providerUpperProduct(storage.raw, 3)
	metadata, d := providerUpperProduct(b.maximumMetadataPerCommit, 8)
	// The original bytes are already in the readback row. Two additional
	// typed decodes use the binding's closed metadata cap, plus one map entry.
	bindingScratch, bindingOK := providerUpperProduct(candidateLedgerMetadataLimit, 2)
	wire, e := providerUpperProduct(uint64(capture.WireBytes), 3)
	head, f := providerUpperProduct(uint64(capture.HeaderBytes), 3)
	// Eight possible simultaneous bodies/results: live sample2, retained
	// base64/result decode2, HTTP replay body/result2 and final output2.
	body, g := providerUpperProduct(uint64(capture.BodyBytes), 8)
	request, h := providerUpperProduct(uint64(len(providerFirstOperationRequestWire)), 2)
	certificates, i := providerUpperProduct(uint64(b.tls.chains), uint64(b.tls.certificates))
	der, j := providerUpperProduct(certificates, uint64(b.tls.certificateBytes))
	digests, k := providerUpperProduct(certificates, 2*69)
	var classScratch uint64
	if b.guardian != nil {
		classScratch = 2*(16<<10) + 8<<10 + 2*uint64(reflect.TypeFor[providerIntegrationRequest]().Size())
	}
	scratch, l := providerUpperSum(raw, metadata, bindingScratch, wire, head, body, request, der, digests, maximum, classScratch)
	models, m := b.continuation.residentUpper(1, 6, 1, uint64(providerInitialObjectCount(b)+31), schemas, scratch)
	entries, n := providerUpperSum(3*14, 4*11, 128, certificates, 1)
	collections, o := providerUpperProduct(entries, max(b.continuation.objectMemoryOverhead, b.scan.filenameEntryOverhead))
	// Static layout only; allocator and host upper qualification are separate.
	// Each count names an actual simultaneously reachable fixed struct.
	var native uint64
	for _, layout := range []struct{ bytes, count uint64 }{
		{uint64(reflect.TypeFor[providerFirstOperationOwner]().Size()), 1},
		{uint64(reflect.TypeFor[providerFirstOperationPlan]().Size()), 1},
		{uint64(reflect.TypeFor[providerIntegration]().Size()), 1},
		{uint64(reflect.TypeFor[providerWorkMeter]().Size()), 2},
		{uint64(reflect.TypeFor[providerIntegrationPermit]().Size()), 1},
		{uint64(reflect.TypeFor[providerFirstOperationResultData]().Size()), 1},
		{uint64(reflect.TypeFor[providerFirstOperationChange]().Size()), 2},
		{uint64(reflect.TypeFor[providerIndexDelta]().Size()), 2},
		{uint64(reflect.TypeFor[providerFirstAdmissionProposal]().Size()), 1},
	} {
		bytes, fits := providerUpperSum(layout.bytes, uint64(reflect.TypeFor[*byte]().Size()))
		bytes, productOK := providerUpperProduct(bytes, layout.count)
		var totalOK bool
		native, totalOK = providerUpperSum(native, bytes)
		if !fits || !productOK || !totalOK {
			return 0, false
		}
	}
	if b.history != nil {
		// The original parent remains reachable while the first operation
		// preserves its scalar handoff. Count both concrete structs and their
		// allocation pointers; this is static layout, not allocator or host
		// qualification and not a complete future-program memory certificate.
		for _, size := range []uint64{uint64(reflect.TypeFor[providerHistoryProgramOwner]().Size()), uint64(reflect.TypeFor[providerHistoryProgramStart]().Size())} {
			allocation, fits := providerUpperSum(size, uint64(reflect.TypeFor[*byte]().Size()))
			var totalOK bool
			native, totalOK = providerUpperSum(native, allocation)
			if !fits || !totalOK {
				return 0, false
			}
		}
	}
	memory, p := providerUpperSum(first.memory, models, collections, native)
	return memory, a && c && d && bindingOK && e && f && g && h && i && j && k && l && m && n && o && p && memory <= b.scan.maximumResidentBytes && memory <= b.continuation.maximumResidentBytes
}
