package registry011

import "io"

// The initial pre-I target has one setup record and no main record. This
// concrete candidate inspection closes that complete finite prefix only. It
// cannot inspect a post-I prefix, restore execution or act as a Source factory.
type providerScanInitialInspection struct {
	owner                           *providerScanInitialInspection
	control                         *providerScanControl
	target                          providerScanTarget
	meter                           *providerWorkMeter
	replay                          *providerFrontierReplay
	targetNames, required, observed map[string]bool
	objects, rawBytes, catalogNames uint64
	active, completed, failed       bool
}

func (*providerScanInitialInspection) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (i *providerScanInitialInspection) key(disk *retentionDisk, name string) string {
	if disk == i.control.store.custody {
		return "custody:" + name
	}
	if disk == i.control.store.evidence {
		return "evidence:" + name
	}
	return ""
}

func (i *providerScanInitialInspection) beforeRead(disk *retentionDisk, name string, maximum uint64) bool {
	if i == nil || i.owner != i || !i.active || i.failed || i.completed || i.meter == nil || i.control.store.activeMeter.Load() != i.meter {
		return false
	}
	key := i.key(disk, name)
	if key == "" {
		return false
	}
	if !i.targetNames[key] {
		return true
	} // Separately checked live control delta.
	b := i.control.store.config.bounds.scan
	return candidateCapacity(0, i.objects, 1, min(b.maximumObjects, b.chunkObjects)) && candidateCapacity(0, i.rawBytes, maximum, min(b.maximumRawBytes, b.chunkRawBytes))
}

func (i *providerScanInitialInspection) afterRead(disk *retentionDisk, name string, n uint64) bool {
	if !i.beforeRead(disk, name, n) {
		i.failed = true
		return false
	}
	if i.targetNames[i.key(disk, name)] {
		i.objects++
		i.rawBytes += n
	}
	return true
}

// The retained lock is deliberately empty. General immutable-object reads
// reject empty payloads, so inspect this already locked descriptor separately
// without weakening that primitive. A successful zero-byte original read is
// still one observed target object and one logical read invocation.
func (i *providerScanInitialInspection) emptyLock(disk *retentionDisk) bool {
	if !i.beforeRead(disk, "AUTHORITY.LOCK", 1) || !i.meter.preflight(providerReadEvent, 1) || disk.checkIdentity() != nil {
		return false
	}
	info, err := disk.lock.Stat()
	if err != nil || info.Size() != 0 {
		return false
	}
	var probe [1]byte
	n, err := disk.lock.ReadAt(probe[:], 0)
	if n != 0 || err != io.EOF || disk.checkIdentity() != nil || !i.meter.event(providerReadEvent, 0) {
		return false
	}
	return i.afterRead(disk, "AUTHORITY.LOCK", 0)
}

// Reserve the entire last nonempty setup record PLUS both complete catalogs
// before any target original read or catalog allocation. The finite profile
// may refuse; it never buys another chunk/deadline or reports partial success.
func providerScanInitialReservation(config providerCustodyConfig) (candidateWorkVector, uint64, bool) {
	if config.bounds.scan == nil || !config.bounds.valid() {
		return candidateWorkVector{}, 0, false
	}
	b, n := config.bounds.scan, config.bounds.maximumMetadataPerCommit
	upper, a := providerUpperProduct(n, 128)
	square, c := providerUpperProduct(n, n)
	units, d := providerUpperSum(square, n, 16)
	units, e := providerUpperProduct(units, 128)
	names, f := providerUpperProduct(b.maximumCatalogNameBytes, 4)
	catalog, g := providerUpperProduct(b.maximumCatalogEntries, 8)
	units, h := providerUpperSum(units, names, catalog, 16)
	entries, j := providerUpperProduct(b.maximumCatalogEntries, b.filenameEntryOverhead)
	entries, k := providerUpperProduct(entries, 2)
	strings, l := providerUpperProduct(b.maximumCatalogNameBytes, 2)
	scratch, m := providerUpperProduct(n, 16)
	scratch, q := providerUpperSum(scratch, 64<<10)
	scratch, r := providerUpperProduct(scratch, config.bounds.continuation.codecMemoryMultiplier)
	resident, t := providerUpperSum(entries, strings, scratch)
	clocks, v := providerUpperSum(b.maximumCatalogEntries, 16)
	work := candidateWorkVector{0, 0, upper, 0, 0, 0, 0, 0, 0, 0, units}
	if !a || !c || !d || !e || !f || !g || !h || !j || !k || !l || !m || !q || !r || !t || !v || b.maximumFilenameBytes < uint64(len("scan-highwater-")+20) || resident > b.maximumResidentBytes || clocks > b.maximumClockReads || 32 > b.chunkObjects || upper > b.chunkRawBytes {
		return candidateWorkVector{}, 0, false
	}
	for index, n := range work {
		if n > b.chunkWork[index] {
			return candidateWorkVector{}, 0, false
		}
	}
	return work, resident, true
}

// Data-only one-use empty-prefix observation. Before E0, the computed empty
// index anchors have no retained inventory originals, so this path MUST NOT
// label its result as ScanChunkV1 or name those future retained roots. Actual
// command/chunk/S-bound scanning remains the separate planned implementation.
func (c *providerScanControl) inspectInitial(rawTarget []byte) (*providerScanEmptyResult, error) {
	if c == nil || c.store == nil {
		return nil, errProviderIntegration
	}
	s := c.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if !c.valid() || c.prepared == nil || s.initialInspection != nil || s.activeMeter.Load() != nil || s.sequence != 0 || s.enrollment != "" || len(s.objects) != 0 || len(s.setups) != 1 || s.setups[0].ordinal != 0 || s.frontierReplay == nil || s.frontierReplay.body.intent != nil {
		return nil, errProviderIntegration
	}
	if _, _, ok := providerScanInitialReservation(s.config); !ok {
		return nil, errProviderIntegration
	}
	files, filesOK := providerUpperProduct(c.pin.ordinal, 5)
	files, filesOK2 := providerUpperSum(files, 13)
	names, namesOK := providerUpperProduct(files, s.config.bounds.scan.maximumFilenameBytes)
	if !filesOK || !filesOK2 || !namesOK || files > s.config.bounds.scan.maximumCatalogEntries || names > s.config.bounds.scan.maximumCatalogNameBytes {
		return nil, errProviderIntegration
	}
	meter := &providerWorkMeter{scan: c, upper: c.prepared.body.work, maximumClocks: c.prepared.body.clocks}
	meter.owner = meter
	i := &providerScanInitialInspection{control: c, meter: meter, active: true, targetNames: make(map[string]bool), required: make(map[string]bool), observed: make(map[string]bool)}
	i.owner = i
	s.initialInspection = i
	s.activeMeter.Store(meter)
	defer s.activeMeter.Store(nil)
	defer func() { i.active = false }()
	succeeded := false
	defer func() {
		if !succeeded {
			i.failed, c.failed, s.failed = true, true, true
		}
	}()
	start, err := s.sample()
	if err != nil {
		return nil, errProviderIntegration
	}
	x, err := newProviderScanCodec(s.config, meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	i.replay, err = newProviderFrontierScanReplay(c, meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	target, err := x.parseTarget(rawTarget)
	targetDigest := x.hash("sage-provider-scan-target-v1", rawTarget)
	if err != nil || targetDigest != c.prepared.target || target != (providerScanTarget{setup: s.setupTip, custody: s.tip, objects: s.objectsRoot}) {
		return nil, errProviderIntegration
	}
	i.target = target
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		for _, name := range []string{"ANCHOR", "AUTHORITY.LOCK"} {
			i.targetNames[i.key(disk, name)] = true
		}
	}
	for _, name := range []string{providerCustodyName("setup-intent", 0), providerCustodyName("setup-commit", 0), providerCustodyName("highwater", 0), "HEAD"} {
		i.targetNames[i.key(s.custody, name)] = true
	}
	i.targetNames[i.key(s.evidence, providerCustodyName("setup-claim", 0))] = true
	for key := range i.targetNames {
		i.required[key] = true
	}
	// The exact live candidate control prefix is outside the frozen target.
	// Its original bytes must agree before its concrete names are classified.
	if !c.readPrefix() || !c.checkMainPin() {
		return nil, errProviderIntegration
	}
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		i.required[i.key(disk, "SCAN-HEAD")] = true
		for ordinal := uint64(0); ordinal <= c.pin.ordinal; ordinal++ {
			i.required[i.key(disk, providerCustodyName("scan-highwater", ordinal))] = true
		}
	}
	for ordinal := uint64(1); ordinal <= c.pin.ordinal; ordinal++ {
		i.required[i.key(s.custody, providerCustodyName("scan-intent", ordinal))] = true
		i.required[i.key(s.custody, providerCustodyName("scan-commit", ordinal))] = true
		i.required[i.key(s.evidence, providerCustodyName("scan-claim", ordinal))] = true
	}
	anchor, a := s.read(s.custody, "ANCHOR", int64(s.config.bounds.maximumMetadataPerCommit))
	local, b := s.read(s.evidence, "ANCHOR", int64(s.config.bounds.maximumMetadataPerCommit))
	value := contentObject(x.decode(anchor), "format", "namespace", "instance_key", "profile", "initial_objects_root")
	emptyRaw, emptyErr := x.encode(map[string]any{"format": "sage-provider-custody-objects-v2", "previous": nil, "entries": []map[string]string{}})
	emptyRoot := x.hash("sage-provider-custody-objects-v2", emptyRaw)
	if a != nil || b != nil || emptyErr != nil || value == nil || !x.equal(anchor, local) || x.hash("sage-provider-custody-anchor-v1", anchor) != s.config.anchor || value["format"] != "sage-provider-custody-anchor-v1" || value["namespace"] != s.config.identity.Namespace || value["instance_key"] != s.config.identity.Instance || value["profile"] != s.config.profile || value["initial_objects_root"] != emptyRoot || target.objects != emptyRoot {
		return nil, errProviderIntegration
	}
	claim, a := s.read(s.custody, providerCustodyName("setup-intent", 0), int64(s.config.bounds.maximumMetadataPerCommit))
	copy, b := s.read(s.evidence, providerCustodyName("setup-claim", 0), int64(s.config.bounds.maximumMetadataPerCommit))
	commit, d := s.read(s.custody, providerCustodyName("setup-commit", 0), int64(s.config.bounds.maximumMetadataPerCommit))
	want, expectedErr := x.setup(providerClockDiagnostic(s.setupOrigin))
	wantCommit, commitErr := x.setupCommit(claim)
	if a != nil || b != nil || d != nil || expectedErr != nil || commitErr != nil || !x.equal(claim, copy) || !x.equal(claim, want) || !x.equal(commit, wantCommit) || x.hash("sage-provider-authority-setup-commit-v1", commit) != target.setup {
		return nil, errProviderIntegration
	}
	// Independently derive original H0 from the verified anchor/setup, rather
	// than treating the producer's cached pin bytes as semantic evidence.
	id := s.config.identity
	mainHead, headErr := x.encode(map[string]any{"format": "sage-provider-custody-highwater-v1", "namespace": id.Namespace, "instance_key": id.Instance, "generation": "0", "fence": "1", "ordinal": "0", "previous_highwater": nil, "profile": s.config.profile, "sequence": "0", "custody_commit": target.custody, "setup_head": target.setup, "frontier": nil})
	reader := &providerScanControl{store: s}
	if headErr != nil || !reader.readExact(s.custody, "HEAD", mainHead) || !reader.readExact(s.custody, providerCustodyName("highwater", 0), mainHead) {
		return nil, errProviderIntegration
	}
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		if !i.emptyLock(disk) {
			return nil, errProviderIntegration
		}
	}
	// This is the final nonempty record's catalog, not a heartbeat chunk.
	if !i.catalog() || !c.checkMainPin() || !c.readHighwater(c.pin) {
		return nil, errProviderIntegration
	}
	end, err := s.sample()
	bounds := s.config.bounds.scan
	if err != nil || !candidateLedgerClockAdvance(providerClockDiagnostic(start), providerClockDiagnostic(end)) || !candidateLedgerClockAdvance(c.prepared.clock, providerClockDiagnostic(start)) || candidateLedgerExpired(providerClockDiagnostic(end), c.prepared.deadline) || uint64(end.MonotonicMS-start.MonotonicMS) > bounds.chunkDurationMS || uint64(end.MonotonicMS-s.setupOrigin.MonotonicMS) > bounds.totalDurationMS || meter.failed {
		return nil, errProviderIntegration
	}
	work := meter.actual
	for index, n := range work {
		if n > bounds.chunkWork[index] {
			return nil, errProviderIntegration
		}
	}
	result := &providerScanEmptyResult{target: targetDigest, records: 1, objects: i.objects, rawBytes: i.rawBytes, start: providerClockDiagnostic(start), end: providerClockDiagnostic(end), work: work}
	i.completed = true
	succeeded = true
	return result, nil // DATA only, never a retained ScanChunk/S/ACK credential.
}

func (i *providerScanInitialInspection) catalog() bool {
	s, b := i.control.store, i.control.store.config.bounds.scan
	if uint64(len(i.required)) > b.maximumCatalogEntries {
		return false
	}
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		if disk.checkIdentity() != nil {
			return false
		}
		f, err := disk.root.Open(".")
		if err != nil {
			return false
		}
		ok := true
		for {
			if !i.meter.preflight(providerReadEvent, b.maximumFilenameBytes) {
				ok = false
				break
			}
			entries, e := f.ReadDir(1)
			if e == io.EOF && len(entries) == 0 {
				ok = i.meter.event(providerReadEvent, 0)
				break
			}
			if e != nil || len(entries) != 1 || !entries[0].Type().IsRegular() {
				ok = false
				break
			}
			name := entries[0].Name()
			key := i.key(disk, name)
			if uint64(len(name)) > b.maximumFilenameBytes || uint64(len(i.observed)) >= b.maximumCatalogEntries || !candidateCapacity(0, i.catalogNames, uint64(len(name)), b.maximumCatalogNameBytes) || !i.required[key] || i.observed[key] || !i.meter.event(providerReadEvent, uint64(len(name))) || !i.meter.event(providerCompareEvent, uint64(len(name))) {
				ok = false
				break
			}
			i.observed[key] = true
			i.catalogNames += uint64(len(name))
			if _, err := s.sample(); err != nil {
				ok = false
				break
			}
		}
		if f.Close() != nil || !ok || disk.checkIdentity() != nil {
			return false
		}
	}
	return len(i.observed) == len(i.required)
}

// Private diagnostic fields convey no owner, future index roots, I/T/C/S or
// capability. There is no wire-format receipt or restoration/selection API.
type providerScanEmptyResult struct {
	target                     string
	records, objects, rawBytes uint64
	start, end                 CandidateClockDiagnostic
	work                       candidateWorkVector
}

func (*providerScanEmptyResult) UnmarshalJSON([]byte) error { return errProviderIntegration }
