package registry011

import "strconv"

const providerFirstAdmissionLiveAccounting = "held-retained-including-all-temporary-copies-v1"

// These are explicit candidate inputs, never operating defaults or a physical
// filesystem qualification. The future bridge must compare its complete
// first-E1 file/copy/RAM plan to this prior hold before any write. T0 cannot
// finance that write. V1/V2/V3 callers leave this object nil.
type providerFirstAdmissionBounds struct {
	work                               candidateWorkVector
	retained, live, clocks, durationMS uint64
}

func (b providerFirstAdmissionBounds) valid(store providerIntegrationBounds) bool {
	if store.scan == nil || store.setupLive != store.setupRetained || b.retained == 0 || b.live != b.retained || b.retained > store.maximumStoredBytes || b.retained > store.scan.maximumPreIRetainedBytes || b.clocks == 0 || b.clocks > store.scan.maximumPreIClockReads || b.durationMS == 0 || b.durationMS > store.scan.totalDurationMS || b.work[2] == 0 || b.work[9] == 0 || b.work[9] > b.retained || b.work[10] == 0 {
		return false
	}
	for coordinate, n := range b.work {
		if n > store.scan.maximumPreIWork[coordinate] || coordinate != 2 && coordinate != 9 && coordinate != 10 && n != 0 {
			return false // No request, wire, EVM or protected operation before I.
		}
	}
	return true
}

func (b providerFirstAdmissionBounds) object() map[string]any {
	return map[string]any{"format": "sage-provider-first-admission-bounds-v2", "work_11": candidateLedgerWorkObject(b.work), "retained_bytes": strconv.FormatUint(b.retained, 10), "live_bytes": strconv.FormatUint(b.live, 10), "maximum_clock_reads": strconv.FormatUint(b.clocks, 10), "maximum_duration_ms": strconv.FormatUint(b.durationMS, 10)}
}

// The V4 rule derives both capacities from the SAME committed retained upper.
// No additional live field is smuggled into the existing exact control schema.
// All original and temporary copies must be covered by its finite derivation.
type providerFirstAdmissionLimits struct {
	work, operation, segment candidateWorkVector
	live, retained           uint64
}

func (s *providerCustodyStore) firstAdmissionLimits(x *providerScanCodec) bool {
	if s.config.bounds.firstAdmission == nil {
		return true
	}
	m, ok := x.decode(providerReplayPhaseBytes(s.config.policy.raw, x.meter)).(map[string]any)
	if !ok {
		return false
	}
	limits, ok := m["limits"].(map[string]any)
	if !ok {
		return false
	}
	work, a := candidateAdmissionWorkLimit(limits["bootstrap"])
	operation, operationOK := candidateAdmissionWorkLimit(limits["operation"])
	segment, segmentOK := candidateAdmissionWorkLimit(limits["segment"])
	live, b := providerCandidateUint(limits["live_disk_bytes"], true)
	retained, c := providerCandidateUint(limits["retained_bytes"], true)
	if !a || !b || !c || !operationOK || !segmentOK {
		return false
	}
	s.firstLimits = &providerFirstAdmissionLimits{work: work, operation: operation, segment: segment, live: live, retained: retained}
	return true
}

func providerFirstAdmissionCapacity(s *providerCustodyStore, held providerScanUpper) bool {
	if s.config.bounds.firstAdmission == nil {
		return true // Preserve the existing V3 DATA-only contract byte-for-byte.
	}
	if b := s.config.bounds.guardian; b != nil {
		var ok bool
		held, ok = providerScanUpperSum(held, providerScanUpper{work: b.lifetime.work, retained: b.lifetime.retained, clocks: b.lifetime.clocks})
		if !ok {
			return false
		}
	}
	limits := s.firstLimits
	if limits == nil || held.retained > limits.live || held.retained > limits.retained || held.retained > s.config.bounds.maximumStoredBytes {
		return false
	}
	for coordinate, n := range held.work {
		if n > limits.work[coordinate] {
			return false
		}
	}
	return true
}

// Quote only the first E1 physical files. This is one component of the
// prospective plan, not that full plan, a reservation, or permission to write.
// Fourteen fresh objects plus five main files and two head copies are21
// possible allocations. HEAD.pending stays charged on uncertain outcomes.
// Binding is capped by min(metadata cap,8192). This quote deliberately uses
// the metadata cap alone as a conservative overestimate of that one file.
func providerFirstAdmissionStorageUpper(b providerIntegrationBounds) (uint64, uint64, bool) {
	if b.firstAdmission == nil || !b.valid() || b.maximumEntries < 14 || b.maximumObjects < 21 {
		return 0, 0, false
	}
	// Conservative binding + intent/pending/commit:4m. I/ticket/enrollment + inventory/
	// frontier:5*64KiB. B0/4 nodes/3 mutations + two head copies:10*8KiB.
	metadata, a := providerUpperProduct(b.maximumMetadataPerCommit, 4)
	raw, c := providerUpperSum(metadata, 5*(64<<10), 10*candidateLedgerMetadataLimit, 16<<10, 8<<10)
	allocation, d := providerUpperProduct(b.fileOverheadBytes, 21)
	physical, e := providerUpperSum(raw, allocation)
	return raw, physical, a && c && d && e && physical <= b.maximumStoredBytes
}

// One proposed-I construction only: policy copy/decode twice, limits encode/
// hash twice, joined admission-ID copy/copy/hash, and I encode/decode/hash/copy.
// The15 actual primitives have input at most64KiB (plus fixed hash domains).
// This logical component neither reserves F nor covers the rest of E1.
func providerFirstAdmissionIntentUpper() candidateWorkVector {
	const n uint64 = 64 << 10
	var work candidateWorkVector
	work[2], work[10] = 3*n, 15*(n*n+n+64)
	return work
}

// Only first-ticket formatting, not the request constructor or prospective
// operation quote: codecs3/encodes4/hashes5/copies7/comparisons5=24 primitives.
// The fixed request uses native text comparisons, without raw conversion.
// Seven quadratic events cost at most7*(n*n+1), and17 linear events cost at
// most17*(2*n+65); at n=65536 they fit24*(n*n+n+64).
// No request execution, clock read, write or physical reservation occurs here.
func providerFirstAdmissionTicketUpper() candidateWorkVector {
	const n uint64 = 64 << 10
	var work candidateWorkVector
	work[2], work[10] = 3*n, 24*(n*n+n+64)
	return work
}

// Future-only C/S storage DATA. It does not include I/E1 and cannot fund that
// earlier write. The eventual V4 builders must enforce the quoted closed
// metadata caps and independently compare every original retained delta.
type providerFirstOperationStorage struct {
	consumption, settlement, transport, exchange uint64
	raw, physical                                uint64
}

func providerFirstOperationStorageUpper(b providerIntegrationBounds, global ProviderCaptureLimits, requestBytes uint64) (providerFirstOperationStorage, bool) {
	if b.firstAdmission == nil || !b.valid() || !b.methods.fits(global) || requestBytes == 0 || requestBytes > 16<<10 {
		return providerFirstOperationStorage{}, false
	}
	capture, err := b.methods.header.capture(global, int(requestBytes))
	if err != nil {
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
