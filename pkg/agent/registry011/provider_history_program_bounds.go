package registry011

import "strconv"

const providerHistoryProgramBoundsDomain = "sage-provider-history-program-bounds-v1"

// These immutable candidate bounds describe the COMPLETE future program, not
// just its T/C/S transcript. A positive data object is insufficient to open the
// profile: the closed source-cost compiler must also supply the original,
// nonserializable derivation below. No production derivation factory is
// provided by this unfinished connection; JSON and diagnostics cannot supply it.
type providerHistoryProgramBounds struct {
	program, costs                           string
	future                                   providerHistoryGroupTotals
	startup                                  providerScanUpper
	stages, records, commits, catalogEntries uint64
	catalogNameBytes, resident, durationMS   uint64
	derivation                               *providerHistoryProgramDerivation
}

// This is a source-derivation witness, NOT a budget, Source, principal, ACK or
// execution capability. Its eventual internal factory must close all source
// rows, failure/final work and live-set costs before constructing this token.
// Keeping the witness separate prevents an unfinished numeric profile from
// reaching H0 while those rows are still NOT_DERIVED.
type providerHistoryProgramDerivation struct {
	owner    *providerHistoryProgramDerivation
	snapshot providerHistoryProgramBounds
}

func (*providerHistoryProgramBounds) UnmarshalJSON([]byte) error     { return ErrConfiguration }
func (*providerHistoryProgramDerivation) UnmarshalJSON([]byte) error { return ErrConfiguration }

func (b providerHistoryProgramBounds) valid(store providerIntegrationBounds) bool {
	if store.firstAdmission == nil || store.scan == nil || store.continuation == nil || store.guardian != nil ||
		!lowerHash(b.program) || !lowerHash(b.costs) || !b.future.valid() ||
		b.stages == 0 || b.stages > (1<<20)/3 || b.records < 3*b.stages || b.records > store.continuation.maximumRecords ||
		b.commits < 3*b.stages || b.commits > uint64(store.maximumCommits) || b.future.objects > uint64(store.maximumObjects) ||
		b.catalogEntries < b.future.files || b.catalogEntries == 0 || b.resident == 0 || b.resident > store.continuation.maximumResidentBytes ||
		b.durationMS == 0 || b.durationMS > 1<<53 || b.startup.retained == 0 || b.startup.clocks < 11 || b.startup.work[0] != 1 || b.startup.work[1] == 0 || b.startup.work[10] == 0 {
		return false
	}
	names, ok := providerUpperProduct(b.catalogEntries, 255)
	physical, physicalOK := providerUpperSum(store.scan.maximumPreIRetainedBytes, b.startup.retained, b.future.physical)
	clocks, clockOK := providerUpperSum(store.scan.maximumPreIClockReads, b.startup.clocks, b.future.clocks)
	return ok && names <= b.catalogNameBytes && physicalOK && physical <= store.maximumStoredBytes && clockOK && clocks <= 1<<20 &&
		providerHistoryUpperWithin(providerScanUpper{work: store.setupWork, retained: store.setupRetained, clocks: store.setupMaximumClockReads}, providerHistoryPreIUpper(store)) &&
		b.derivation != nil && b.derivation.owner == b.derivation && b.derivation.snapshot == b
}

func (b providerHistoryProgramBounds) object() map[string]any {
	return map[string]any{
		"format": providerHistoryProgramBoundsDomain, "program": b.program, "cost_revision": b.costs,
		"future": b.future.object(), "startup": b.startup.object(),
		"maximum_stages": strconv.FormatUint(b.stages, 10), "maximum_records": strconv.FormatUint(b.records, 10),
		"maximum_commits": strconv.FormatUint(b.commits, 10), "catalog_entries": strconv.FormatUint(b.catalogEntries, 10),
		"catalog_name_bytes": strconv.FormatUint(b.catalogNameBytes, 10), "resident_bytes": strconv.FormatUint(b.resident, 10),
		"duration_ms": strconv.FormatUint(b.durationMS, 10),
	}
}

func providerHistoryPreIUpper(b providerIntegrationBounds) providerScanUpper {
	return providerScanUpper{work: b.scan.maximumPreIWork, retained: b.scan.maximumPreIRetainedBytes, clocks: b.scan.maximumPreIClockReads}
}

// The old initial seed remains restricted to2/9/10. This separate sum covers
// the pre-I maximum (which ALREADY includes O), fixed first P, and all future
// history work. It is never inserted into that old seed or charged a second
// time for each child. The internal derivation must justify each disjoint row.
func providerHistoryWholeHold(b providerIntegrationBounds) (providerScanUpper, bool) {
	if b.history == nil || b.scan == nil {
		return providerScanUpper{}, false
	}
	h := b.history
	return providerScanUpperSum(providerHistoryPreIUpper(b), h.startup, providerScanUpper{work: h.future.work, retained: h.future.physical, clocks: h.future.clocks})
}

func providerHistoryUpperWithin(value, maximum providerScanUpper) bool {
	if value.retained > maximum.retained || value.clocks > maximum.clocks {
		return false
	}
	for i, n := range value.work {
		if n > maximum.work[i] {
			return false
		}
	}
	return true
}
