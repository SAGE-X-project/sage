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
	recovery                                 providerHistoryRecoveryBounds
	final                                    providerHistoryFinalBounds
	stages, records, commits, catalogEntries uint64
	catalogNameBytes, resident, durationMS   uint64
	derivation                               *providerHistoryProgramDerivation
	collection                               *providerHistoryCollectionManifest
}

// This is a source-derivation witness, NOT a budget, Source, principal, ACK or
// execution capability. Its eventual internal factory must close all source
// rows, failure/final work and live-set costs before constructing this token.
// Keeping the witness separate prevents an unfinished numeric profile from
// reaching H0 while those rows are still NOT_DERIVED.
type providerHistoryProgramDerivation struct {
	owner     *providerHistoryProgramDerivation
	snapshot  providerHistoryProgramBounds
	dispatch  providerHistoryDispatchBounds
	admission providerHistoryCollectionAdmissionBounds
}

func (*providerHistoryProgramBounds) UnmarshalJSON([]byte) error     { return ErrConfiguration }
func (*providerHistoryProgramDerivation) UnmarshalJSON([]byte) error { return ErrConfiguration }

func (b providerHistoryProgramBounds) valid(store providerIntegrationBounds) bool {
	if b.collection != nil && !b.collection.validSnapshot() {
		return false
	}
	final, finalOK := b.final.upper()
	if !finalOK || b.final != (providerHistoryFinalBounds{}) && !b.final.valid(store, b.catalogEntries, b.durationMS) {
		return false
	}
	if store.firstAdmission == nil || store.scan == nil || store.continuation == nil || store.guardian != nil ||
		!lowerHash(b.program) || !lowerHash(b.costs) || !b.future.valid() || !b.recovery.valid(store, b.catalogEntries) ||
		b.stages == 0 || b.stages > (1<<20)/3 || b.records < 3*b.stages || b.records > store.continuation.maximumRecords ||
		b.commits < 3*b.stages || b.commits > uint64(store.maximumCommits) || b.future.objects > uint64(store.maximumObjects) ||
		b.catalogEntries < b.future.files || b.catalogEntries == 0 || b.resident == 0 || b.resident > store.continuation.maximumResidentBytes ||
		b.durationMS == 0 || b.durationMS > 1<<53 || b.startup.retained == 0 || b.startup.clocks < 11 || b.startup.work[0] != 1 || b.startup.work[1] == 0 || b.startup.work[10] == 0 {
		return false
	}
	archiveSlots, archiveOK := providerHistoryRecoveryArchiveSlots(b.stages)
	names, ok := providerUpperProduct(b.catalogEntries, 255)
	physical, physicalOK := providerUpperSum(store.scan.maximumPreIRetainedBytes, b.startup.retained, b.future.physical)
	clocks, clockOK := providerUpperSum(store.scan.maximumPreIClockReads, b.startup.clocks, b.future.clocks, final.clocks)
	return archiveOK && archiveSlots <= b.resident && ok && names <= b.catalogNameBytes && physicalOK && physical <= store.maximumStoredBytes && clockOK && clocks <= 1<<20 &&
		providerHistoryUpperWithin(providerScanUpper{work: store.setupWork, retained: store.setupRetained, clocks: store.setupMaximumClockReads}, providerHistoryPreIUpper(store)) &&
		b.derivation != nil && b.derivation.owner == b.derivation && b.derivation.snapshot == b &&
		(b.stages == 1 || b.derivation.dispatch.advanceValid())
}

func (b providerHistoryProgramBounds) object() map[string]any {
	var maximum *providerHistoryGroupUpper
	if b.derivation != nil && b.derivation.admission != (providerHistoryCollectionAdmissionBounds{}) {
		maximum = &b.derivation.admission.maximum
	}
	return b.objectWithAdmissionMaximum(maximum)
}

// Pure final-schema projection. The ordinary path supplies its existing
// private admission value; cost compilation can supply the same value before
// any derivation exists. This serializer neither validates nor creates one.
func (b providerHistoryProgramBounds) objectWithAdmissionMaximum(maximum *providerHistoryGroupUpper) map[string]any {
	out := map[string]any{
		"format": providerHistoryProgramBoundsDomain, "program": b.program, "cost_revision": b.costs,
		"future": b.future.object(), "startup": b.startup.object(), "recovery_inspection": b.recovery.object(),
		"maximum_stages": strconv.FormatUint(b.stages, 10), "maximum_records": strconv.FormatUint(b.records, 10),
		"maximum_commits": strconv.FormatUint(b.commits, 10), "catalog_entries": strconv.FormatUint(b.catalogEntries, 10),
		"catalog_name_bytes": strconv.FormatUint(b.catalogNameBytes, 10), "resident_bytes": strconv.FormatUint(b.resident, 10),
		"duration_ms": strconv.FormatUint(b.durationMS, 10),
	}
	if b.collection != nil {
		out["collection_manifest"] = map[string]string{"digest": b.collection.digest, "raw": b.collection.raw}
		if maximum != nil {
			out["collection_admission_maximum"] = maximum.object()
		}
	}
	if b.final != (providerHistoryFinalBounds{}) {
		out["successful_final_inspection"] = b.final.object()
	}
	return out
}

func providerHistoryPreIUpper(b providerIntegrationBounds) providerScanUpper {
	return providerScanUpper{work: b.scan.maximumPreIWork, retained: b.scan.maximumPreIRetainedBytes, clocks: b.scan.maximumPreIClockReads}
}

// The old initial seed remains restricted to2/9/10. This separate sum covers
// the pre-I maximum (which ALREADY includes O), fixed first P, all future
// history work, disjoint read-only failure inspection and successful final
// inspection. It is never inserted into that old seed or charged a second
// time for each child. The internal derivation must justify each disjoint row.
func providerHistoryWholeHold(b providerIntegrationBounds) (providerScanUpper, bool) {
	if b.history == nil || b.scan == nil {
		return providerScanUpper{}, false
	}
	h := b.history
	final, ok := h.final.upper()
	if !ok {
		return providerScanUpper{}, false
	}
	return providerScanUpperSum(providerHistoryPreIUpper(b), h.startup, providerScanUpper{work: h.future.work, retained: h.future.physical, clocks: h.future.clocks}, providerScanUpper{work: h.recovery.work}, final)
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

// Work and clock DATA for a source-cost subgraph, not a budget witness,
// owner, reservation, native-time bound or execution capability. Storage and
// native/backing/peak/lifetime RAM remain separate domains.
type providerHistoryStageWorkCost struct {
	work   candidateWorkVector
	clocks uint64
}

// Compose ONE original T window. before covers the complete T/C/dispatch
// prefix, including its own incoming envelopes. preparation covers actual
// work already entered for the current member BEFORE remaining() preflight.
// Each full member row includes that preparation and every nested capture,
// retention and deferred envelope; closure must cover success AND known
// failure. Those source-completeness obligations are not proved by this sum.
//
// At ordinal k the unchanged guard adds (count-k)*FULLmember+closure to an
// actual prefix bounded by before+k*member+preparation. The extra preparation
// is therefore retained ONCE in EACH original T upper. Adding a whole extra
// member would incorrectly increase RPC/wire permission. Own advance remains
// inside this old T window; each full result still belongs in the irrevocable
// future SUM. Initialization, recovery, final/page, native time/RAM and the
// source derivation factory are not supplied or authorized by numeric DATA.
func providerHistoryCollectionStageWorkUpper(before, preparation providerHistoryStageWorkCost, rows providerHistoryDispatchBounds, count int, advancing bool) (providerHistoryStageWorkCost, bool) {
	var empty providerHistoryStageWorkCost
	if count < 1 || count > providerHistoryGroupMembers || !rows.valid() ||
		before.work[0] != 0 || before.work[1] != 0 || preparation.work[0] != 0 || preparation.work[1] != 0 ||
		preparation.clocks > rows.memberClocks || advancing && !rows.advanceValid() {
		return empty, false
	}
	for coordinate, value := range preparation.work {
		if value > rows.member[coordinate] {
			return empty, false
		}
	}
	remaining, clocks, ok := rows.remaining(count)
	if !ok {
		return empty, false
	}
	var a providerHistoryReadArithmetic
	var out providerHistoryStageWorkCost
	for coordinate := range out.work {
		out.work[coordinate] = a.sum(before.work[coordinate], preparation.work[coordinate], remaining[coordinate])
		if advancing {
			out.work[coordinate] = a.sum(out.work[coordinate], rows.advance[coordinate])
		}
	}
	out.clocks = a.sum(before.clocks, preparation.clocks, clocks)
	if advancing {
		out.clocks = a.sum(out.clocks, rows.advanceClocks)
	}
	if a.failed {
		return empty, false
	}
	return out, true
}
