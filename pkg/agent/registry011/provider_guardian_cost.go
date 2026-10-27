package registry011

import "reflect"

// Prospective logical event rows, never measured work or host CPU/RAM. N
// covers both successful closed schemas and the larger encoder preflight.
// Q=N*N+1; L=2*N+101 covers two full byte strings and every fixed domain.
func providerGuardianAddedWork(b providerIntegrationBounds, quadratic, linear, raw uint64) (candidateWorkVector, bool) {
	var work candidateWorkVector
	if b.guardian == nil || b.scan == nil || b.continuation == nil {
		return work, false
	}
	n := max(b.maximumMetadataPerCommit, uint64(64<<10))
	q, a := providerUpperProduct(n, n)
	q, c := providerUpperSum(q, 1)
	l, d := providerUpperSum(n, n, 101)
	q, e := providerUpperProduct(q, quadratic)
	l, f := providerUpperProduct(l, linear)
	decoded, g := providerUpperProduct(n, quadratic)
	units, h := providerUpperSum(q, l)
	work[2], work[9], work[10] = decoded, raw, units
	return work, a && c && d && e && f && g && h
}

// O additions: 13 bounds rebuilds (39Q/13L), three origin encodes/hashes
// (3Q/6L), H0/origin-original/head/deadline/new-object branches (12Q/64L),
// and one complete old E0 replay envelope (273Q/1252L) conservatively covers
// the eighth entry's inventory/order/retention/index traversal. The 17-file
// E0+G0 quote adds only origin and two head allocations to the old 14 files.
func providerGuardianBootstrapExtraUpper(b providerIntegrationBounds) (candidateWorkVector, uint64, bool) {
	var empty candidateWorkVector
	if b.guardian == nil || b.firstAdmission == nil || b.maximumEntries < 8 || b.maximumObjects < 8 {
		return empty, 0, false
	}
	raw, a := providerUpperProduct(3, candidateLedgerMetadataLimit)
	files, c := providerUpperProduct(3, b.fileOverheadBytes)
	physical, d := providerUpperSum(raw, files)
	work, e := providerGuardianAddedWork(b, 39+3+12+273, 13+6+64+1252, raw)
	return work, physical, a && c && d && e
}

// F additions remain separate from O/G. The first four rows are the R9
// independently counted bounds/origin/initial/H0-original branches. Two old
// E0 envelopes cover the new entry in the two distinct original-file passes.
// Two additional complete genesis-lineage checks are 40Q/48L each. None of
// these rows contains the exhausted O or a future operation's C/S work.
func providerGuardianFirstExtraUpper(b providerIntegrationBounds) (candidateWorkVector, bool) {
	return providerGuardianAddedWork(b, 102+14+48+18+2*273+2*40, 34+28+72+64+2*1252+2*48, 0)
}

// Peak additional candidates reachable at the same time: three closed
// bounds/profile parse/encode maps (<=150 entries), origin/head/initial maps
// (<=74), and their slice headers (rounded to256 entries); six raw immutable
// head/origin copies plus three metadata scratch buffers. The declared codec
// multiplier and native structs are added, not used as physical qualification.
func providerGuardianMemoryExtra(b providerIntegrationBounds) (uint64, bool) {
	if b.guardian == nil || b.scan == nil || b.continuation == nil {
		return 0, false
	}
	n := max(b.maximumMetadataPerCommit, uint64(64<<10))
	entries, a := providerUpperProduct(256, max(b.scan.filenameEntryOverhead, b.continuation.objectMemoryOverhead))
	scratch, c := providerUpperProduct(n, 3)
	scratch, d := providerUpperSum(scratch, 6*candidateLedgerMetadataLimit)
	scratch, e := providerUpperProduct(scratch, b.continuation.codecMemoryMultiplier)
	native, f := providerUpperSum(uint64(reflect.TypeFor[providerGuardianOwner]().Size()), 2*uint64(reflect.TypeFor[providerGuardianBounds]().Size()), 3*uint64(reflect.TypeFor[providerGuardianHeadData]().Size()), 2*uint64(reflect.TypeFor[providerGuardianInitialData]().Size()), 2*uint64(reflect.TypeFor[providerGuardianReplay]().Size()))
	total, g := providerUpperSum(entries, scratch, native)
	return total, a && c && d && e && f && g
}

// Complete scalar-only O check before H0/root/clock observation. The old
// open128 and bootstrap512 rows are disjoint; the new rows above are added
// explicitly, together with the old setup files and the V5 peak allocation.
func providerGuardianInitialFits(s *providerCustodyStore) bool {
	if s == nil || s.config.bounds.guardian == nil || s.firstLimits == nil || s.clock == nil {
		return false
	}
	b := s.config.bounds
	bootstrap, physical, ok := providerBootstrapUpper(b)
	n := max(b.maximumMetadataPerCommit, uint64(64<<10))
	units, a := providerUpperProduct(n, n)
	units, c := providerUpperSum(units, n, 16)
	units, d := providerUpperProduct(units, 128)
	decoded, e := providerUpperProduct(n, 128)
	raw, f := providerUpperProduct(n, 9)
	files, g := providerUpperProduct(b.fileOverheadBytes, 13)
	openPhysical, h := providerUpperSum(raw, files)
	whole, j := providerUpperSum(openPhysical, physical)
	clocks, k := providerUpperSum(16, 8)
	open := candidateWorkVector{0, 0, decoded, 0, 0, 0, 0, 0, 0, raw, units}
	if !ok || !a || !c || !d || !e || !f || !g || !h || !j || !k || whole > b.setupRetained || clocks > b.setupMaximumClockReads {
		return false
	}
	for i, amount := range bootstrap {
		if !candidateCapacity(0, open[i], amount, b.setupWork[i]) {
			return false
		}
	}
	extra, extraOK := providerGuardianMemoryExtra(b)
	scratch, scratchOK := providerUpperProduct(physical, 2)
	base, baseOK := b.continuation.residentUpper(1, 1, 1, 8, whole, scratch)
	memory, memoryOK := providerUpperSum(base, extra)
	clock := s.clock
	clock.mu.Lock()
	clockOK := clock.owner == clock && !clock.failed && clock.read == 0 && candidateCapacity(0, clocks, b.guardian.lifetime.clocks, clock.maximum)
	clock.mu.Unlock()
	return extraOK && scratchOK && baseOK && memoryOK && clockOK && memory <= b.scan.maximumResidentBytes && memory <= b.continuation.maximumResidentBytes && extra <= b.guardian.lifetime.resident
}
