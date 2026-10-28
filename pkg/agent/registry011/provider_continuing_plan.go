package registry011

import "reflect"

// Unexported prospective DATA. Selection, actual work and concrete one-use
// retention are separate steps; this value is never an execution permission.
type providerContinuingPlan struct {
	reservation                       providerOperationReservation
	physical, consumption, settlement uint64
	metadataWork                      candidateWorkVector
	metadataRaw, metadataPhysical     uint64
	memory, headroom                  uint64
}

func (*providerContinuingPlan) UnmarshalJSON([]byte) error { return errProviderIntegration }

// This first continuation is the one closed BootstrapStart command. It reads
// the already retained method-specific first X and has no transport request.
// The C and S reference rows come from the separately reviewed first-operation
// graphs; the completed-X replay is additionally included in local E, rather
// than borrowed from the earlier first operation's closed measurement.
func providerContinuingBootstrapPlan(g *providerGuardianOwner) (providerContinuingPlan, bool) {
	var p providerContinuingPlan
	if g == nil || !g.selectedLocked() || !g.initialized || !g.admitted || g.integration == nil || g.first == nil || g.store.config.bounds.guardian == nil || g.store.firstLimits == nil || g.first.state != providerFirstOperationDone {
		return p, false
	}
	s, b := g.store, g.config.bounds
	storage, ok := providerFirstOperationStorageUpper(b, g.config.capture, uint64(len(providerFirstOperationRequestWire)))
	if !ok || s.frontierReplay == nil || s.frontierReplay.body == nil || s.frontierReplay.body.ledger == nil {
		return p, false
	}
	n := max(b.maximumMetadataPerCommit, uint64(64<<10), storage.transport, storage.exchange)
	maximum := max(uint64(1<<20), n)
	headroom, a := providerUpperProduct(maximum, maximum)
	headroom, c := providerUpperSum(headroom, b.maximumObjectBytes, 2)
	// E: original C graph552/3106 + completed-X/S result replay16/128,
	// two exact composite checks80/448, BootstrapStart formatting5/30, and
	// three affected-value comparisons6/12. The last comparisons encode each
	// side independently; old full-model native comparisons do not fund them.
	execution, e := providerFirstOperationPhaseUpper(n, 552+16+80+5+6, 3106+128+448+30+12, 7, 29, storage.consumption, headroom)
	// R: original S graph571/3284 + BootstrapStart closure16/64 + affected
	// typed values6/12. The old graph's larger RPC-only objects are a safe
	// overestimate for the smaller closed local result, not an actual vector.
	remaining, f := providerFirstOperationPhaseUpper(n, 571+16+6, 3284+64+12, 5, 37, storage.settlement, headroom)
	execution[3], execution[8] = 1, 1
	work, err := providerFrontierWorkTotal(execution, remaining)
	consumptionFiles, h := providerUpperProduct(14, b.fileOverheadBytes)
	settlementFiles, i := providerUpperProduct(18, b.fileOverheadBytes)
	consumption, j := providerUpperSum(storage.consumption, consumptionFiles)
	settlement, k := providerUpperSum(storage.settlement, settlementFiles)
	physical, l := providerUpperSum(consumption, settlement)
	// G pays the at-most10 T objects and seven main/head files in addition
	// to its five auxiliary allocations. All known typed metadata fits8KiB;
	// request/params have separate16/8KiB maxima and the inventory64KiB.
	mainRaw, m := providerUpperSum(10*candidateLedgerMetadataLimit, 16<<10, 8<<10, 64<<10, 4*b.maximumMetadataPerCommit, 2*candidateLedgerMetadataLimit)
	metadataRaw, metaOK := providerUpperSum(mainRaw, 5*candidateLedgerMetadataLimit)
	metadataFiles, filesOK := providerUpperProduct(22, b.fileOverheadBytes)
	metadataPhysical, physicalOK := providerUpperSum(metadataRaw, metadataFiles)
	// The separate closed count table covers G claim+originals+binding/T at
	//128Q/1024L. The64Q/512L difference row covers two extra T objects, its
	// third index mutation and their separate read/prepare/apply comparisons.
	metadataWork, metadataOK := providerGuardianAddedWork(b, 552+128+64, 3106+1024+512, metadataRaw)
	if !a || !c || !e || !f || err != nil || !h || !i || !j || !k || !l || !m || !metaOK || !filesOK || !physicalOK || !metadataOK ||
		physical != storage.physical || physical > b.maximumStoredBytes || metadataRaw > g.bounds.claim.raw ||
		metadataRaw > g.bounds.claim.metadata || metadataPhysical > g.bounds.claim.retained || !providerContinuingWorkFits(metadataWork, g.bounds.claim.work) {
		return p, false
	}
	reservation := providerOperationReservation{work: work, execution: execution, remaining: remaining, live: physical, retained: physical}
	for coordinate, amount := range work {
		if amount > s.firstLimits.operation[coordinate] || amount > s.firstLimits.segment[coordinate] {
			return p, false
		}
	}
	memory, memoryOK := providerContinuingMemoryUpper(g, storage, metadataRaw, maximum)
	if !memoryOK {
		return p, false
	}
	return providerContinuingPlan{reservation: reservation, physical: physical, consumption: consumption, settlement: settlement, metadataWork: metadataWork, metadataRaw: metadataRaw, metadataPhysical: metadataPhysical, memory: memory, headroom: headroom}, true
}

func providerContinuingWorkFits(work, upper candidateWorkVector) bool {
	for i, n := range work {
		if n > upper[i] {
			return false
		}
	}
	return true
}

func providerContinuingMemoryUpper(g *providerGuardianOwner, storage providerFirstOperationStorage, metadataRaw, maximum uint64) (uint64, bool) {
	if g == nil || g.store == nil || g.integration == nil || g.config.bounds.continuation == nil || g.config.bounds.scan == nil || g.config.bounds.methods == nil {
		return 0, false
	}
	b, s, ledger := g.config.bounds, g.store, g.integration.ledger
	if ledger == nil || ledger.continuation == nil || g.first == nil {
		return 0, false
	}
	retained, a := providerUpperSum(s.retainedRaw, storage.raw, metadataRaw)
	combinedRaw, combinedOK := providerUpperSum(storage.raw, metadataRaw)
	// Five independent new-byte families can overlap: proposed object strings,
	// proposed inventory/intent/frontier encodings, physical readback buffers,
	// custodian parser originals, and temporary canonical comparison encodes.
	// Retained coordinator/custodian strings are separately in residentUpper.
	raw, c := providerUpperProduct(combinedRaw, 5)
	// Sixteen metadata scratch blocks cover current/prior main+G originals,
	// expected G claim/head/commit and both three-node typed index deltas.
	metadata, d := providerUpperProduct(b.maximumMetadataPerCommit, 16)
	header, captureErr := b.methods.header.capture(g.config.capture, len(providerFirstOperationRequestWire))
	if captureErr != nil {
		return 0, false
	}
	bodies, bodiesOK := providerUpperProduct(uint64(header.BodyBytes), 4)
	// One old X read, decoded body/result and selected header copies are new
	// local-E scratch. The closed first owner's old peak cannot fund them.
	headerScratch, headerSum := providerUpperSum(storage.exchange, bodies, candidateLedgerMetadataLimit)
	scratch, e := providerUpperSum(raw, metadata, maximum, headerScratch)
	operations, operationsOK := providerUpperSum(s.sequence, 3)
	models, f := b.continuation.residentUpper(uint64(len(ledger.tickets))+1, operations, uint64(len(ledger.segments))+1, uint64(len(s.objects))+32, retained, scratch)
	// residentUpper funds two object-map copies. Add the third required-ref
	// map, three finite fresh/seen/inventory slices, six three-node mutation
	// arrays and256 closed G/schema map entries at the selected profile upper.
	entries, entriesOK := providerUpperSum(uint64(len(s.objects)), 32, 3*32, 6*3, 256)
	collections, collectionOK := providerUpperProduct(entries, max(b.continuation.objectMemoryOverhead, b.scan.filenameEntryOverhead))
	var native uint64
	for _, entry := range []struct{ bytes, count uint64 }{
		{uint64(reflect.TypeFor[providerContinuingOwner]().Size()), 1},
		{uint64(reflect.TypeFor[providerContinuingPlan]().Size()), 1},
		{uint64(reflect.TypeFor[providerWorkMeter]().Size()), 2},
		{uint64(reflect.TypeFor[providerIntegrationPermit]().Size()), 1},
		{uint64(reflect.TypeFor[providerFirstOperationChange]().Size()), 2},
		{uint64(reflect.TypeFor[providerIndexDelta]().Size()), 2},
		{uint64(reflect.TypeFor[providerGuardianClaimData]().Size()), 2},
		{uint64(reflect.TypeFor[providerGuardianAdmissionBinding]().Size()), 2},
		{uint64(reflect.TypeFor[providerContinuingBootstrapData]().Size()), 1},
	} {
		bytes, ok := providerUpperProduct(entry.bytes, entry.count)
		var added bool
		native, added = providerUpperSum(native, bytes)
		if !ok || !added {
			return 0, false
		}
	}
	// Closed F/first-owner pointers may still be reachable. Keep their entire
	// old peak as a conservative separate row; do not assume a GC release.
	total, h := providerUpperSum(models, collections, native, g.first.plan.memory)
	return total, a && combinedOK && c && d && bodiesOK && headerSum && e && operationsOK && f && entriesOK && collectionOK && h && total <= b.scan.maximumResidentBytes && total <= g.bounds.claim.resident
}
