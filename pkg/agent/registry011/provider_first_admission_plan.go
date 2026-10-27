package registry011

// This is a live owner's prospective candidate plan, never a recovered hold,
// host qualification, future operation quote or execution acknowledgment.
type providerFirstAdmissionPlan struct {
	owner                         *providerFirstAdmissionOwner
	work                          candidateWorkVector
	clocks, raw, physical, memory uint64
	stored                        uint64
	held                          providerScanUpper
	used                          bool
}

func (*providerFirstAdmissionPlan) UnmarshalJSON([]byte) error { return errProviderIntegration }

// Only scalar arithmetic precedes the complete comparison. The two original
// E0 passes and the common writer use the function/loop envelopes recorded in
// the independent cost review. Proposal/control/receipt counts conservatively
// reserve BOTH quadratic and linear cost for each event, rather than claiming
// those event counts are measured work. Every arithmetic step is checked.
func providerFirstAdmissionPlanUpper(b providerIntegrationBounds, ordinal, records, retained, output uint64) (providerFirstAdmissionPlan, bool) {
	var p providerFirstAdmissionPlan
	if !b.valid() || b.firstAdmission == nil || ordinal == 0 || ordinal > b.scan.maximumAttempts || records != 2 || retained > b.maximumStoredBytes || output == 0 {
		return p, false
	}
	n := max(b.maximumMetadataPerCommit, uint64(64<<10))
	square, a := providerUpperProduct(n, n)
	quadratic, c := providerUpperSum(square, 1)
	linear, d := providerUpperSum(n, n, 101)
	controls, e := providerUpperProduct(ordinal, 26)
	controls, f := providerUpperSum(controls, 24)
	receipts, g := providerUpperProduct(records, 29)
	receipts, h := providerUpperSum(receipts, 5)
	closures, i := providerUpperSum(controls, receipts)
	closures, j := providerUpperProduct(closures, 2)
	// Known initial-only graphs: two E0(273q/1252b), one retain(538q/
	// 2930b), proposal611 events and two independent original closures.
	q, k := providerUpperSum(2*273, 538, 611, closures)
	// Entry compares the original main pin once on adoption and twice when
	// consuming the closed prepaid role. These are observed linear events.
	bytes, l := providerUpperSum(2*1252, 2930, 611, closures, 3)
	qWork, o := providerUpperProduct(q, quadratic)
	bWork, u := providerUpperProduct(bytes, linear)
	decoded, v := providerUpperProduct(q, n)
	// Eight E0 samples, three writer samples, two receipt walks, and entry
	// proposal/adoption/final samples. Sync43+rename1 are independent events.
	clockReceipts, w := providerUpperProduct(records, 2)
	clocks, x := providerUpperSum(14, clockReceipts)
	units, y := providerUpperSum(qWork, bWork, 84998, 44, clocks)
	// Canonical encoders first demand up to1MiB even for small results; one
	// old-enrollment read first demands maximumObjectBytes. Keep that extra
	// headroom to the end, without recording it as actual codec/read work.
	headroom, z := providerUpperSum((1<<20)*(1<<20)+1, b.maximumObjectBytes, 1)
	units, unitsOK := providerUpperSum(units, headroom)
	raw, physical, storageOK := providerFirstAdmissionStorageUpper(b)
	if !a || !c || !d || !e || !f || !g || !h || !i || !j || !k || !l || !o || !u || !v || !w || !x || !y || !z || !unitsOK || !storageOK {
		return p, false
	}
	// The original store/closed inspection, proposed delta, input reader and
	// fresh custodian can overlap. Include the full external output upper,
	// both ledger/schema copies, three inventories and maximum codec scratch.
	schemas, schemaOK := providerUpperSum(retained, output, raw)
	scratch, scratchOK := providerUpperSum(raw, raw, raw, 8*b.maximumMetadataPerCommit, 1<<20)
	model, modelOK := b.continuation.residentUpper(1, 3, 1, 21, schemas, scratch)
	old, oldOK := b.continuation.residentUpper(1, 2, 1, 7, retained, 0)
	// Use the reader's maximum incoming-entry reservation, not just the
	// seven originals it ultimately reads. Three such map/catalog/name sets
	// cover the closing inspection, coordinator and custodian lifetimes.
	count, countOK := providerUpperSum(21, uint64(b.maximumEntries))
	entries, entriesOK := providerUpperProduct(count, 3)
	catalog, catalogOK := providerUpperProduct(b.scan.maximumCatalogEntries, 2)
	entries, moreOK := providerUpperSum(entries, uint64(b.maximumEntries), 4, 3, 2, catalog, 8)
	mapBytes, mapOK := providerUpperProduct(entries, b.scan.filenameEntryOverhead)
	prefixes, prefixOK := providerUpperProduct(b.scan.maximumCatalogEntries, 9)
	names, namesOK := providerUpperSum(b.scan.maximumCatalogNameBytes, prefixes)
	reader, readerOK := providerUpperSum(mapBytes, names)
	readers, readersOK := providerUpperProduct(reader, 6)
	memory, memoryOK := providerUpperSum(model, old, readers, output)
	if !schemaOK || !scratchOK || !modelOK || !oldOK || !countOK || !entriesOK || !catalogOK || !moreOK || !mapOK || !prefixOK || !namesOK || !readerOK || !readersOK || !memoryOK || memory > b.scan.maximumResidentBytes || memory > b.continuation.maximumResidentBytes {
		return p, false
	}
	p.work[2], p.work[9], p.work[10] = decoded, raw, units
	p.clocks, p.raw, p.physical, p.memory = clocks, raw, physical, memory
	return p, true
}

// Called immediately after STARTED, before any read, sample, I or growing
// collection. Insufficient original profiles fail; no ceiling is increased.
func (f *providerFirstAdmissionOwner) planLocked() bool {
	if f == nil || f.store == nil || f.state != providerFirstAdmissionStarted || f.plan != nil || f.replayStarted || f.custodianStarted || !f.fits(candidateWorkVector{}) || f.meter.actual != f.startWork || f.meter.clocks != f.startClocks || f.mainPin.sequence != 1 || f.mainPin.ordinal != 1 || f.store.sequence != 1 {
		return false
	}
	s, m := f.store, f.meter
	p, ok := providerFirstAdmissionPlanUpper(s.config.bounds, f.scanPin.ordinal, f.records, s.retainedRaw, f.readerOutput)
	if o := f.operation; o != nil {
		if !o.selectedLocked() || o.first != f || o.plan.memory == 0 {
			return false
		}
		for coordinate, amount := range o.plan.firstExtra {
			var fits bool
			p.work[coordinate], fits = providerUpperSum(p.work[coordinate], amount)
			if !fits {
				return false
			}
		}
		p.memory = o.plan.memory
	}
	if !ok || p.physical > f.limits.retained || p.raw > f.limits.work[9] || s.retainedRaw > s.stored || s.scanControl.held != f.scanPin.held || s.scanControl.pin != f.scanPin || s.stored < f.limits.retained || !f.fits(p.work) || !candidateCapacity(0, m.clocks, p.clocks, m.maximumClocks) || !candidateCapacity(0, m.clocks-f.startClocks, p.clocks, f.limits.clocks) {
		return false
	}
	for k, n := range p.work {
		if !candidateCapacity(0, m.actual[k], n, m.upper[k]) {
			return false
		}
	}
	// The F/B counters cannot replenish the original backend's finite calls.
	// Inspect its remaining capacity without taking or inventing a sample.
	c := s.clock
	c.mu.Lock()
	clockNeed := p.clocks
	if f.operation != nil {
		var fits bool
		clockNeed, fits = providerUpperSum(clockNeed, 11)
		if !fits {
			c.mu.Unlock()
			return false
		}
	}
	clockRoom := c.owner == c && !c.failed && c.id == f.origin.ClockID && c.epoch == f.origin.Epoch && c.policy == s.config.policy.policy && candidateCapacity(0, c.read, clockNeed, c.maximum)
	c.mu.Unlock()
	if !clockRoom {
		return false
	}
	p.owner, p.stored, p.held = f, s.stored, s.scanControl.held
	f.plan = &p
	return true
}
