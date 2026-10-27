package registry011

import (
	"io"
	"strconv"
)

// This is an unretained pre-I DATA inspection. It is not an S/ACK or a
// qualified Source. Post-I command/S admission and the O+Q/I binding are
// separate prerequisites; this entry point deliberately cannot scan after I.
type providerScanBody struct {
	owner                               *providerScanBody
	control                             *providerScanControl
	meter                               *providerWorkMeter
	reader                              *providerScanReplayReader
	required                            map[string]byte //1 target,2 verified live control
	observed                            map[string]bool
	objects, rawBytes, catalogNameBytes uint64
	requiredNameBytes                   uint64
	chunkObjects, chunkBytes            uint64
	windowStart, windowLimit            candidateWorkVector
	window, encoding, active, failed    bool
}

type providerScanDataResult struct {
	target                     string
	chunks                     [][]byte
	records, objects, rawBytes uint64
	work                       candidateWorkVector
	start, end                 CandidateClockDiagnostic
}

func (*providerScanBody) UnmarshalJSON([]byte) error       { return errProviderIntegration }
func (*providerScanDataResult) UnmarshalJSON([]byte) error { return errProviderIntegration }

func (b *providerScanBody) valid() bool {
	return b != nil && b.owner == b && b.active && !b.failed && b.control.valid() && b.control.store.scanBody == b && b.meter != nil && b.meter.owner == b.meter && b.meter.scanBody == b && b.control.store.activeMeter.Load() == b.meter
}

func (b *providerScanBody) key(disk *retentionDisk, name string) (string, bool) {
	prefix := ""
	switch disk {
	case b.control.store.custody:
		prefix = "custody:"
	case b.control.store.evidence:
		prefix = "evidence:"
	}
	if prefix == "" || uint64(len(name)) > b.control.store.config.bounds.scan.maximumFilenameBytes {
		return "", false
	}
	key := providerReplayJoin([]string{prefix, name}, b.meter)
	return key, key != ""
}

func (b *providerScanBody) beforeRead(disk *retentionDisk, name string, maximum uint64) bool {
	if !b.valid() || !b.window {
		return false
	}
	key, ok := b.key(disk, name)
	if !ok {
		return false
	}
	if b.required[key] == 2 {
		return true
	}
	limits := b.control.store.config.bounds.scan
	if b.required[key] == 0 {
		if uint64(len(b.required)) >= limits.maximumCatalogEntries || !candidateCapacity(0, b.requiredNameBytes, uint64(len(name)), limits.maximumCatalogNameBytes) {
			return false
		}
		b.required[key] = 1
		b.requiredNameBytes += uint64(len(name))
	}
	return candidateCapacity(0, b.objects, 1, limits.maximumObjects) && candidateCapacity(0, b.chunkObjects, 1, limits.chunkObjects) && candidateCapacity(0, b.rawBytes, maximum, limits.maximumRawBytes) && candidateCapacity(0, b.chunkBytes, maximum, limits.chunkRawBytes)
}

func (b *providerScanBody) afterRead(disk *retentionDisk, name string, n uint64) bool {
	if !b.valid() || !b.window {
		return false
	}
	key, ok := b.key(disk, name)
	if !ok {
		return false
	}
	if b.required[key] != 2 {
		limits := b.control.store.config.bounds.scan
		if !candidateCapacity(0, b.objects, 1, limits.maximumObjects) || !candidateCapacity(0, b.chunkObjects, 1, limits.chunkObjects) || !candidateCapacity(0, b.rawBytes, n, limits.maximumRawBytes) || !candidateCapacity(0, b.chunkBytes, n, limits.chunkRawBytes) {
			return false
		}
		// beforeRead already reserved the whole successful read's upper.
		b.objects++
		b.rawBytes += n
		b.chunkObjects++
		b.chunkBytes += n
	}
	return true
}

func (b *providerScanBody) emptyLock(disk *retentionDisk) bool {
	if !b.beforeRead(disk, "AUTHORITY.LOCK", 1) || !b.meter.preflight(providerReadEvent, 1) || disk.checkIdentity() != nil {
		return false
	}
	info, err := disk.lock.Stat()
	if err != nil || info.Size() != 0 {
		return false
	}
	var probe [1]byte
	n, err := disk.lock.ReadAt(probe[:], 0)
	return n == 0 && err == io.EOF && disk.checkIdentity() == nil && b.meter.event(providerReadEvent, 0) && b.afterRead(disk, "AUTHORITY.LOCK", 0)
}

// Exact control names are classified separately from frozen target raw files.
// Original readPrefix must validate their content before catalog completion.
func (b *providerScanBody) controls() bool {
	s := b.control.store
	add := func(disk *retentionDisk, name string) bool {
		key, ok := b.key(disk, name)
		if !ok || uint64(len(b.required)) >= s.config.bounds.scan.maximumCatalogEntries || !candidateCapacity(0, b.requiredNameBytes, uint64(len(name)), s.config.bounds.scan.maximumCatalogNameBytes) {
			return false
		}
		b.required[key] = 2
		b.requiredNameBytes += uint64(len(name))
		return true
	}
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		if !add(disk, "SCAN-HEAD") {
			return false
		}
		for n := uint64(0); n <= b.control.pin.ordinal; n++ {
			if !add(disk, providerCustodyName("scan-highwater", n)) {
				return false
			}
		}
	}
	for n := uint64(1); n <= b.control.pin.ordinal; n++ {
		if !add(s.custody, providerCustodyName("scan-intent", n)) || !add(s.custody, providerCustodyName("scan-commit", n)) || !add(s.evidence, providerCustodyName("scan-claim", n)) {
			return false
		}
	}
	return b.control.readPrefix() && b.control.checkMainPin()
}

// Reserve the entire finite catalog in the LAST nonempty record's window,
// before starting that record. Names are logical work, not raw target bytes.
func providerScanCatalogWork(config providerCustodyConfig) (candidateWorkVector, bool) {
	var upper candidateWorkVector
	b := config.bounds.scan
	for _, item := range []struct {
		kind  providerWorkEvent
		bytes uint64
	}{{providerReadEvent, b.maximumFilenameBytes}, {providerCompareEvent, b.maximumFilenameBytes}, {providerCopyEvent, b.maximumFilenameBytes + 9}, {providerClockEvent, 0}} {
		work, ok := providerEventVector(item.kind, item.bytes)
		if !ok {
			return upper, false
		}
		for k, value := range work {
			part, a := providerUpperProduct(value, b.maximumCatalogEntries)
			upper[k], ok = providerUpperSum(upper[k], part)
			if !a || !ok {
				return upper, false
			}
		}
	}
	var ok bool
	upper[10], ok = providerUpperSum(upper[10], 5) // Both EOF reads and final clock room.
	return upper, ok
}

func (b *providerScanBody) catalog() bool {
	if !b.valid() || !b.window {
		return false
	}
	s, limits := b.control.store, b.control.store.config.bounds.scan
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
			if !b.meter.preflight(providerReadEvent, limits.maximumFilenameBytes) {
				ok = false
				break
			}
			entries, err := f.ReadDir(1)
			if err == io.EOF && len(entries) == 0 {
				ok = b.meter.event(providerReadEvent, 0)
				break
			}
			if err != nil || len(entries) != 1 || !entries[0].Type().IsRegular() {
				ok = false
				break
			}
			name := entries[0].Name()
			key, valid := b.key(disk, name)
			if !valid || uint64(len(b.observed)) >= limits.maximumCatalogEntries || !candidateCapacity(0, b.catalogNameBytes, uint64(len(name)), limits.maximumCatalogNameBytes) || b.required[key] == 0 || b.observed[key] || !b.meter.event(providerReadEvent, uint64(len(name))) || !b.meter.event(providerCompareEvent, uint64(len(name))) {
				ok = false
				break
			}
			b.observed[key] = true
			b.catalogNameBytes += uint64(len(name))
			if _, err := s.sample(); err != nil {
				ok = false
				break
			}
		}
		if f.Close() != nil || !ok || disk.checkIdentity() != nil {
			return false
		}
	}
	return len(b.observed) == len(b.required)
}

func providerScanBodyResident(config providerCustodyConfig, output uint64) bool {
	b := config.bounds
	entries, a := providerUpperProduct(b.scan.maximumCatalogEntries, 4)
	entries, c := providerUpperProduct(entries, b.scan.filenameEntryOverhead)
	prefixes, d := providerUpperProduct(b.scan.maximumCatalogEntries, 9)
	names, e := providerUpperSum(b.scan.maximumCatalogNameBytes, prefixes)
	names, f := providerUpperProduct(names, 2)
	base, g := b.continuation.residentUpper(1, 1, 1, 0, output, 6*b.maximumMetadataPerCommit+64<<10)
	total, h := providerUpperSum(base, entries, names)
	return a && c && d && e && f && g && h && total <= b.scan.maximumResidentBytes
}

// One nonempty setup/main record uses one already prepaid subwindow. There
// is no JSON/cursor resume, renewed budget/deadline or zero-record heartbeat.
// The resulting closed chunk bytes remain unretained DATA with no S/ACK.
func (c *providerScanControl) inspectBeforeAdmission(rawTarget []byte) (*providerScanDataResult, error) {
	if c == nil || c.store == nil {
		return nil, errProviderIntegration
	}
	s := c.store
	s.mu.Lock()
	defer s.mu.Unlock()
	if !c.valid() || c.prepared == nil || s.scanBody != nil || s.initialInspection != nil || s.activeMeter.Load() != nil || s.frontierFiles == nil || s.frontierFiles.pin.sequence == 0 {
		return nil, errProviderIntegration
	}
	limits := s.config.bounds.scan
	records, ok := providerUpperSum(s.frontierFiles.pin.ordinal, 1)
	outputUpper, outputOK := providerUpperProduct(2*s.config.bounds.maximumMetadataPerCommit, limits.maximumChunks)
	catalogUpper, catalogOK := providerScanCatalogWork(s.config)
	if !ok || !outputOK || !catalogOK || !providerScanBodyResident(s.config, outputUpper) || records > limits.maximumRecords || records > limits.maximumChunks || limits.chunkRecords == 0 {
		return nil, errProviderIntegration
	}
	b := &providerScanBody{control: c, required: make(map[string]byte), observed: make(map[string]bool), active: true, window: true, windowLimit: limits.chunkWork}
	b.owner = b
	meter := &providerWorkMeter{scan: c, scanBody: b, upper: c.prepared.body.work, maximumClocks: c.prepared.body.clocks}
	meter.owner, b.meter = meter, meter
	s.scanBody = b
	s.activeMeter.Store(meter)
	defer s.activeMeter.Store(nil)
	defer func() { b.active = false }()
	succeeded := false
	defer func() {
		if !succeeded {
			b.failed, c.failed, s.failed = true, true, true
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
	b.reader, err = newProviderScanReplayReader(c, meter, rawTarget)
	if err != nil || b.reader.target.admission != "" || b.reader.target.budget != "" || b.reader.target.checkpoint != "" {
		return nil, errProviderIntegration // No post-I path without admitted T/C/S.
	}
	b.reader.outputUpper = outputUpper
	if !b.reader.resident(0, 0, 5*s.config.bounds.maximumMetadataPerCommit+64<<10) || !b.controls() {
		return nil, errProviderIntegration
	}
	// Existing typed index anchors must have actual original files even in the
	// setup chunk. Do not serialize computed future anchors before E0 exists.
	for _, anchor := range b.reader.replay.anchors {
		raw, err := s.read(s.evidence, providerObjectName(anchor.kind, anchor.digest), int64(s.config.bounds.maximumObjectBytes))
		if err != nil || !providerReplayEqual(raw, []byte(anchor.raw), meter) {
			return nil, errProviderIntegration
		}
	}
	result := &providerScanDataResult{target: c.prepared.target, start: providerClockDiagnostic(start)}
	cursor := providerScanCursor{"setup", 0}
	previous := ""
	for index := uint64(0); index < records; index++ {
		chunkStart := start
		if index != 0 {
			b.windowStart, b.windowLimit = meter.actual, limits.chunkWork
			b.chunkObjects, b.chunkBytes = 0, 0
			b.window = true
			chunkStart, err = s.sample()
			if err != nil {
				return nil, errProviderIntegration
			}
		}
		last := index+1 == records
		if last {
			for k, n := range catalogUpper {
				if n > b.windowLimit[k] {
					return nil, errProviderIntegration
				}
				b.windowLimit[k] -= n
			}
			if meter.clocks > meter.maximumClocks || limits.maximumCatalogEntries+2 > meter.maximumClocks-meter.clocks {
				return nil, errProviderIntegration
			}
		}
		command, err := x.command(result.target, index, previous, cursor)
		if err != nil {
			return nil, errProviderIntegration
		}
		commandDigest := x.hash("sage-provider-scan-command-v1", command)
		next := cursor
		if cursor.phase == "setup" {
			err = b.reader.setup()
			next.next++
			if next.next == b.reader.selected.ordinal-b.reader.selected.sequence+1 {
				next = providerScanCursor{"main", 1}
			}
		} else {
			err = b.reader.main()
			next.next++
		}
		if err != nil {
			return nil, errProviderIntegration
		}
		if last {
			if !b.reader.lineage() || !b.emptyLock(s.custody) || !b.emptyLock(s.evidence) || !c.checkMainPin() || !c.readHighwater(c.pin) {
				return nil, errProviderIntegration
			}
			// The catalog upper was held before this record began. Expanding
			// its sublimit now consumes that reservation, never a new claim.
			b.windowLimit = limits.chunkWork
			if !b.catalog() {
				return nil, errProviderIntegration
			}
			next = providerScanCursor{"done", 0}
		}
		end, err := s.sample()
		if err != nil || !candidateLedgerClockAdvance(providerClockDiagnostic(chunkStart), providerClockDiagnostic(end)) || !candidateLedgerClockAdvance(c.prepared.clock, providerClockDiagnostic(start)) || candidateLedgerExpired(providerClockDiagnostic(end), c.prepared.deadline) || uint64(end.MonotonicMS-chunkStart.MonotonicMS) > limits.chunkDurationMS || uint64(end.MonotonicMS-s.setupOrigin.MonotonicMS) > limits.totalDurationMS || meter.failed {
			return nil, errProviderIntegration
		}
		var work candidateWorkVector
		for k, n := range meter.actual {
			work[k] = n - b.windowStart[k]
			var valid bool
			result.work[k], valid = providerUpperSum(result.work[k], work[k])
			if !valid {
				return nil, errProviderIntegration
			}
		}
		result.records++
		result.objects, result.rawBytes = b.objects, b.rawBytes
		f := b.reader.replay.current
		heads := b.reader.replay.indexes.heads
		b.window, b.encoding = false, true
		raw, err := x.encode(map[string]any{"format": "sage-provider-scan-chunk-v1", "target": result.target, "command_digest": commandDigest, "index": strconv.FormatUint(index, 10), "previous_chunk": providerNullableDigest(previous), "cursor_start": cursor.object(), "cursor_end": next.object(), "records": "1", "objects": strconv.FormatUint(b.chunkObjects, 10), "bytes": strconv.FormatUint(b.chunkBytes, 10), "cumulative_records": strconv.FormatUint(result.records, 10), "cumulative_objects": strconv.FormatUint(result.objects, 10), "cumulative_bytes": strconv.FormatUint(result.rawBytes, 10), "replay_state_digest": providerNullableDigest(f.replay), "ticket_index_root": heads[0], "operation_index_root": heads[1], "segment_totals_root": heads[2], "clock_start": providerClockObject(chunkStart), "clock_end": providerClockObject(end), "work": candidateLedgerWorkObject(work), "cumulative_work": candidateLedgerWorkObject(result.work)})
		if err != nil {
			return nil, errProviderIntegration
		}
		previous = x.hash("sage-provider-scan-chunk-v1", raw)
		if previous == "" {
			return nil, errProviderIntegration
		}
		result.chunks = append(result.chunks, raw)
		b.encoding = false
		cursor, result.end = next, providerClockDiagnostic(end)
	}
	succeeded = true
	return result, nil // All bytes remain unretained DATA, not S/ACK/Source.
}
