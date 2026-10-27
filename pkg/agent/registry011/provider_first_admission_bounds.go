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
	work           candidateWorkVector
	live, retained uint64
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
	live, b := providerCandidateUint(limits["live_disk_bytes"], true)
	retained, c := providerCandidateUint(limits["retained_bytes"], true)
	if !a || !b || !c {
		return false
	}
	s.firstLimits = &providerFirstAdmissionLimits{work, live, retained}
	return true
}

func providerFirstAdmissionCapacity(s *providerCustodyStore, held providerScanUpper) bool {
	if s.config.bounds.firstAdmission == nil {
		return true // Preserve the existing V3 DATA-only contract byte-for-byte.
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
