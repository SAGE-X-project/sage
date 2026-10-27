package registry011

// Parsed original diagnostics, never actual-work authority or a recovered
// inspection owner. The caller still owes independent frozen-target replay,
// first-I typed binding, current-pin verification and its failure closure.
type providerScanReceiptClosure struct {
	records, objects, bytes, metadata uint64
	work                              candidateWorkVector
	end                               CandidateClockDiagnostic
}

func (*providerScanReceiptClosure) UnmarshalJSON([]byte) error { return errProviderIntegration }

func providerScanReceiptCursor(value any) (providerScanCursor, bool) {
	m := contentObject(value, "phase", "next_sequence")
	n, ok := providerCandidateUint(m["next_sequence"], false)
	cursor := providerScanCursor{contentString(m["phase"]), n}
	return cursor, m != nil && ok && cursor.object() != nil
}

func providerScanReceiptStep(start, end providerScanCursor) bool {
	if start.phase == "setup" {
		return start.next != ^uint64(0) && end == (providerScanCursor{"setup", start.next + 1}) || end == (providerScanCursor{"main", 1})
	}
	return start.phase == "main" && (start.next != ^uint64(0) && end == (providerScanCursor{"main", start.next + 1}) || end == (providerScanCursor{"done", 0}))
}

func providerScanReceiptOriginalTime(origin, start, end CandidateClockDiagnostic, deadline, perRecord, total uint64) bool {
	return candidateLedgerClockAdvance(origin, start) && candidateLedgerClockAdvance(start, end) && !candidateLedgerExpired(end, deadline) && uint64(end.MonotonicMS-start.MonotonicMS) <= perRecord && uint64(end.MonotonicMS-origin.MonotonicMS) <= total
}

func providerScanReceiptObservedTime(origin, end, sample CandidateClockDiagnostic, deadline, total uint64) bool {
	return candidateLedgerClockAdvance(origin, sample) && candidateLedgerClockAdvance(end, sample) && !candidateLedgerExpired(sample, deadline) && uint64(sample.MonotonicMS-origin.MonotonicMS) <= total
}

// At most two original copies and their bounded parsing scratch coexist.
// No producer array/map or prior measurement is imported by this reader.
func readProviderAdmissionReceipt(s *providerCustodyStore, x *providerScanCodec, binding providerScanAdmissionBinding, kind, digest string) ([]byte, map[string]any, error) {
	name := providerScanReceiptName(kind, digest, x.meter)
	if name == "" {
		return nil, nil, errProviderIntegration
	}
	domain := providerScanCommandV2
	if kind == "chunk" {
		domain = providerScanChunkV2
	}
	var original []byte
	var value map[string]any
	data := &providerScanReceipts{} // Deliberately no live body/owner.
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		raw, readErr := s.read(disk, name, int64(s.config.bounds.maximumMetadataPerCommit))
		m := data.decode(x, kind, raw)
		canonical, encodeErr := x.encode(m)
		if readErr != nil || m == nil || encodeErr != nil || !x.equal(raw, canonical) || x.hash(domain, raw) != digest || m["format"] != domain || m["target"] != binding.target || m["scan_head"] != binding.head || m["scan_highwater"] != binding.highwater || m["scan_attempt"] != binding.attempt || original != nil && !x.equal(original, raw) {
			return nil, nil, errProviderIntegration
		}
		original, value = raw, m
	}
	return original, value, nil
}

// Walk the immutable backward chain once, so memory is independent of its
// length. Counter differences, cursor transitions and both original copies
// must agree at every record, ending at index0/setup0 and zero prior totals.
func replayProviderAdmissionReceipts(s *providerCustodyStore, meter *providerWorkMeter, binding providerScanAdmissionBinding) (*providerScanReceiptClosure, error) {
	owned := meter != nil && (meter.scan != nil && meter.scan.store == s || meter.integration != nil && meter.integration.store == s)
	if !s.valid() || meter == nil || meter.owner != meter || meter.closed || meter.failed || s.activeMeter.Load() != meter || !owned || binding.clock != providerClockDiagnostic(s.setupOrigin) {
		return nil, errProviderIntegration
	}
	x, err := newProviderScanCodec(s.config, meter)
	if err != nil {
		return nil, errProviderIntegration
	}
	if _, err := x.admissionBinding(binding); err != nil {
		return nil, errProviderIntegration
	}
	b := s.config.bounds.scan
	scratch, scratchOK := providerUpperSum(8*s.config.bounds.maximumMetadataPerCommit, 64<<10)
	if _, ok := s.config.bounds.continuation.residentUpper(1, 1, 1, 0, 0, scratch); !scratchOK || !ok {
		return nil, errProviderIntegration
	}
	inspection, err := providerScanInspectionUpper(s.config)
	if err != nil {
		return nil, errProviderIntegration
	}
	result := &providerScanReceiptClosure{}
	data := &providerScanReceipts{head: binding.head, highwater: binding.highwater, attempt: binding.attempt}
	digest := binding.lastChunk
	wantCursor := providerScanCursor{"done", 0}
	var wantIndex, wantRecords, wantObjects, wantBytes uint64
	var wantWork candidateWorkVector
	var nextStart CandidateClockDiagnostic
	for count := uint64(0); count < b.maximumChunks; count++ {
		raw, chunk, readErr := readProviderAdmissionReceipt(s, x, binding, "chunk", digest)
		index, a := providerCandidateUint(chunk["index"], false)
		records, c := providerCandidateUint(chunk["records"], true)
		objects, d := providerCandidateUint(chunk["objects"], false)
		bytes, e := providerCandidateUint(chunk["bytes"], false)
		totalRecords, f := providerCandidateUint(chunk["cumulative_records"], true)
		totalObjects, g := providerCandidateUint(chunk["cumulative_objects"], false)
		totalBytes, h := providerCandidateUint(chunk["cumulative_bytes"], false)
		work, i := candidateAdmissionWork(chunk["work"])
		totalWork, j := candidateAdmissionWork(chunk["cumulative_work"])
		start, k := providerScanReceiptCursor(chunk["cursor_start"])
		end, l := providerScanReceiptCursor(chunk["cursor_end"])
		clockStart, n := providerStoredClock(chunk["clock_start"])
		clockEnd, p := providerStoredClock(chunk["clock_end"])
		previous := contentString(chunk["previous_chunk"])
		if readErr != nil || !a || !c || !d || !e || !f || !g || !h || !i || !j || !k || !l || !n || !p || index >= b.maximumChunks || index >= b.maximumRecords || records != 1 || records > b.chunkRecords || totalRecords != index+1 || objects > b.chunkObjects || bytes > b.chunkRawBytes || totalObjects > b.maximumObjects || totalBytes > b.maximumRawBytes || totalObjects < objects || totalBytes < bytes || end != wantCursor || !providerScanReceiptStep(start, end) || !providerIndexNullableDigest(chunk["previous_chunk"]) || (index == 0) != (previous == "") || index == 0 && start != (providerScanCursor{"setup", 0}) || !providerScanReceiptOriginalTime(binding.clock, clockStart, clockEnd, binding.deadline, b.chunkDurationMS, b.totalDurationMS) || count != 0 && !candidateLedgerClockAdvance(clockEnd, nextStart) {
			return nil, errProviderIntegration
		}
		if count == 0 {
			wantIndex, wantRecords, wantObjects, wantBytes, wantWork = index, totalRecords, totalObjects, totalBytes, totalWork
			result.records, result.objects, result.bytes, result.work, result.end = totalRecords, totalObjects, totalBytes, totalWork, clockEnd
		}
		if index != wantIndex || totalRecords != wantRecords || totalObjects != wantObjects || totalBytes != wantBytes || totalWork != wantWork {
			return nil, errProviderIntegration
		}
		for coordinate, value := range work {
			if value > inspection.work[coordinate]/b.maximumChunks || totalWork[coordinate] > inspection.work[coordinate] || totalWork[coordinate] < value {
				return nil, errProviderIntegration
			}
			wantWork[coordinate] = totalWork[coordinate] - value
		}
		commandDigest := contentString(chunk["command_digest"])
		command, _, commandErr := readProviderAdmissionReceipt(s, x, binding, "command", commandDigest)
		expected, encodeErr := data.command(x, binding.target, index, previous, start)
		metadata, metadataOK := providerUpperSum(result.metadata, uint64(len(raw)), uint64(len(command)))
		if commandErr != nil || encodeErr != nil || !x.equal(command, expected) || !metadataOK || metadata > s.config.bounds.continuation.maximumMetadataBytes {
			return nil, errProviderIntegration
		}
		result.metadata = metadata
		// A genuine owned sample follows each bounded nonempty record. This is
		// finishing work, not a restarted record, epoch or renewed deadline.
		var sample ClockSample
		if meter.firstAdmission != nil {
			sample, err = meter.firstAdmission.sampleLocked()
		} else {
			sample, err = s.sample()
		}
		if err != nil || !providerScanReceiptObservedTime(binding.clock, clockEnd, providerClockDiagnostic(sample), binding.deadline, b.totalDurationMS) {
			return nil, errProviderIntegration
		}
		wantRecords, wantObjects, wantBytes = totalRecords-records, totalObjects-objects, totalBytes-bytes
		if index == 0 {
			if wantRecords != 0 || wantObjects != 0 || wantBytes != 0 || wantWork != (candidateWorkVector{}) {
				return nil, errProviderIntegration
			}
			return result, nil
		}
		wantIndex, wantCursor, nextStart, digest = index-1, start, clockStart, previous
	}
	return nil, errProviderIntegration
}
