package registry011

import "strconv"

// Explicit fresh-profile candidate bounds. No values are operating defaults;
// a V1/V2 namespace is never upgraded by adding this object to its config.
type providerScanBounds struct {
	maximumAttempts, maximumRecords, maximumObjects, maximumRawBytes uint64
	maximumPages, maximumChunks, chunkRecords, chunkObjects          uint64
	chunkRawBytes, maximumResidentBytes, filenameEntryOverhead       uint64
	maximumFilenameBytes, maximumClockReads, chunkDurationMS         uint64
	totalDurationMS, maximumCatalogEntries, maximumCatalogNameBytes  uint64
	maximumPreIRetainedBytes, maximumPreIClockReads                  uint64
	chunkWork, maximumPreIWork                                       candidateWorkVector
}

func (b providerScanBounds) valid(store providerIntegrationBounds) bool {
	if store.continuation == nil || b.maximumAttempts == 0 || b.maximumAttempts > 64 ||
		b.maximumRecords == 0 || b.maximumRecords > uint64(store.maximumCommits)+uint64(store.maximumSetupRecoveries)+1 ||
		b.maximumObjects == 0 || b.maximumObjects > 1<<30 || b.maximumRawBytes == 0 || b.maximumRawBytes > 1<<50 ||
		b.maximumPages == 0 || b.maximumPages > store.continuation.maximumPages || b.maximumChunks == 0 || b.maximumChunks > b.maximumRecords ||
		b.chunkRecords == 0 || b.chunkRecords > b.maximumRecords || b.chunkObjects == 0 || b.chunkObjects > b.maximumObjects || b.chunkRawBytes == 0 || b.chunkRawBytes > b.maximumRawBytes ||
		b.maximumResidentBytes == 0 || b.maximumResidentBytes > store.continuation.maximumResidentBytes || b.filenameEntryOverhead == 0 || b.filenameEntryOverhead > 1<<20 ||
		b.maximumFilenameBytes == 0 || b.maximumFilenameBytes > 255 || b.maximumClockReads == 0 || b.maximumClockReads > 1<<20 ||
		b.chunkDurationMS == 0 || b.chunkDurationMS > 300000 || b.totalDurationMS < b.chunkDurationMS || b.totalDurationMS > 1<<53 ||
		b.maximumCatalogEntries == 0 || b.maximumCatalogEntries > 32<<20 || b.maximumCatalogNameBytes == 0 || b.maximumCatalogNameBytes > 16<<30 ||
		b.maximumPreIRetainedBytes == 0 || b.maximumPreIRetainedBytes > store.maximumStoredBytes || b.maximumPreIClockReads == 0 || b.maximumPreIClockReads > 1<<20 || b.maximumClockReads > b.maximumPreIClockReads {
		return false
	}
	codec, a := providerUpperProduct(store.maximumMetadataPerCommit, 12)
	codec, c := providerUpperSum(codec, 64<<10)
	codec, d := providerUpperProduct(codec, store.continuation.codecMemoryMultiplier)
	if !a || !c || !d || codec > b.maximumResidentBytes {
		return false
	}
	for i, n := range b.chunkWork {
		if n > b.maximumPreIWork[i] {
			return false
		}
	}
	return b.chunkWork[0] == 0 && b.chunkWork[1] == 0 && b.chunkWork[2] > 0 && b.chunkWork[10] > 0 && b.maximumPreIWork[0] == 0 && b.maximumPreIWork[1] == 0 && b.maximumPreIWork[9] > 0 && b.maximumPreIWork[10] > 0
}

// A V3 open has a distinct once-held O, including all fixed H0/HEAD0
// validation, bounded profile/policy codecs and initial directory observations.
// The minimum is conservative logical work; it is not actual CPU/FS evidence.
func providerScanSetupFits(config providerCustodyConfig) bool {
	b := config.bounds
	if b.scan == nil || !b.valid() {
		return false
	}
	open, ok := providerScanOpenCostUpper(b.maximumMetadataPerCommit, b.fileOverheadBytes)
	return ok && open.work[2] <= b.setupWork[2] && open.work[10] <= b.setupWork[10] && open.work[9] <= b.setupWork[9] && open.retained <= b.setupRetained && b.setupMaximumClockReads >= open.clocks
}

func (b providerScanBounds) object() map[string]any {
	m := map[string]any{"chunk_work_11": candidateLedgerWorkObject(b.chunkWork), "maximum_pre_i_total_work_11": candidateLedgerWorkObject(b.maximumPreIWork)}
	values := map[string]uint64{
		"maximum_attempts": b.maximumAttempts, "maximum_records": b.maximumRecords, "maximum_objects": b.maximumObjects,
		"maximum_raw_bytes": b.maximumRawBytes, "maximum_pages": b.maximumPages, "maximum_chunks": b.maximumChunks,
		"chunk_records": b.chunkRecords, "chunk_objects": b.chunkObjects, "chunk_raw_bytes": b.chunkRawBytes,
		"maximum_resident_bytes": b.maximumResidentBytes, "filename_entry_overhead": b.filenameEntryOverhead,
		"maximum_filename_bytes": b.maximumFilenameBytes, "maximum_clock_reads": b.maximumClockReads,
		"chunk_duration_ms": b.chunkDurationMS, "total_duration_ms": b.totalDurationMS,
		"maximum_catalog_entries": b.maximumCatalogEntries, "maximum_catalog_name_bytes": b.maximumCatalogNameBytes,
		"maximum_pre_i_retained_bytes": b.maximumPreIRetainedBytes, "maximum_pre_i_clock_reads": b.maximumPreIClockReads,
	}
	for name, n := range values {
		m[name] = strconv.FormatUint(n, 10)
	}
	return m
}

// O and each P+B are disjoint held uppers, never actual work. Checked addition
// is shared by writer/replay so a refusal cannot refund a prior reservation.
type providerScanUpper struct {
	work             candidateWorkVector
	retained, clocks uint64
}

func (u providerScanUpper) object() map[string]any {
	return map[string]any{"work": candidateLedgerWorkObject(u.work), "retained_bytes": strconv.FormatUint(u.retained, 10), "clock_reads": strconv.FormatUint(u.clocks, 10)}
}

func providerScanUpperValue(value any) (providerScanUpper, bool) {
	m := contentObject(value, "work", "retained_bytes", "clock_reads")
	work, a := candidateAdmissionWork(m["work"])
	retained, b := providerCandidateUint(m["retained_bytes"], false)
	clocks, c := providerCandidateUint(m["clock_reads"], false)
	return providerScanUpper{work, retained, clocks}, m != nil && a && b && c
}

func providerScanUpperSum(parts ...providerScanUpper) (providerScanUpper, bool) {
	var out providerScanUpper
	for _, part := range parts {
		for i, n := range part.work {
			var ok bool
			out.work[i], ok = providerUpperSum(out.work[i], n)
			if !ok {
				return providerScanUpper{}, false
			}
		}
		var a, b bool
		out.retained, a = providerUpperSum(out.retained, part.retained)
		out.clocks, b = providerUpperSum(out.clocks, part.clocks)
		if !a || !b {
			return providerScanUpper{}, false
		}
	}
	return out, true
}

func (b providerScanBounds) fits(open, held providerScanUpper) bool {
	total, ok := providerScanUpperSum(open, held)
	if !ok || total.retained > b.maximumPreIRetainedBytes || total.clocks > b.maximumPreIClockReads {
		return false
	}
	for i, n := range total.work {
		if n > b.maximumPreIWork[i] {
			return false
		}
	}
	return true
}

// Bj is fixed by the complete selected scan profile, not by a caller's choice
// of a small reservation. Chunk snapshots exclude their later encoding, so a
// separate finite output/retention upper is included for EVERY possible chunk.
func providerScanBodyUpper(config providerCustodyConfig) (providerScanUpper, error) {
	inspection, err := providerScanInspectionUpper(config)
	if err != nil {
		return providerScanUpper{}, err
	}
	var held providerScanUpper
	if first := config.bounds.firstAdmission; first != nil {
		held = providerScanUpper{first.work, first.retained, first.clocks}
	}
	body, ok := providerScanBodyCostUpper(*config.bounds.scan, providerScanOpenUpper(config), inspection, held)
	if !ok {
		return providerScanUpper{}, errProviderIntegration
	}
	return body, nil
}

// The inspection sublimit excludes the separately held first-E1 envelope.
// V4 includes four original readbacks and two closed receipt decoders per pair.
func providerScanInspectionUpper(config providerCustodyConfig) (providerScanUpper, error) {
	if config.bounds.scan == nil || !config.bounds.valid() {
		return providerScanUpper{}, errProviderIntegration
	}
	b := config.bounds
	body, ok := providerScanInspectionCostUpper(*b.scan, b.maximumMetadataPerCommit, b.fileOverheadBytes, b.firstAdmission != nil, providerScanOpenUpper(config))
	if !ok {
		return providerScanUpper{}, errProviderIntegration
	}
	return body, nil
}
