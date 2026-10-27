package registry011

import "strconv"

// These are explicit candidate inputs for a fresh V5 namespace. They do not
// select an operating principal, increase an existing profile, or qualify a
// filesystem/clock. Future metadata is held before H0 and before the first I.
type providerGuardianCapacity struct {
	work                                      candidateWorkVector
	retained, live, clocks, duration          uint64
	records, objects, raw, metadata, resident uint64
}

type providerGuardianBounds struct {
	lifetime, claim                            providerGuardianCapacity
	attempts, privateRecords, objects, commits uint64
	catalogEntries, catalogNameBytes           uint64
}

func (v providerGuardianCapacity) valid() bool {
	for i, n := range v.work {
		if i != 2 && i != 9 && i != 10 && n != 0 {
			return false
		}
	}
	return v.work[2] > 0 && v.work[9] > 0 && v.work[10] > 0 &&
		v.retained > 0 && v.live >= v.retained && v.clocks > 0 &&
		v.duration > 0 && v.records > 0 && v.objects > 0 &&
		v.raw > 0 && v.metadata >= v.raw && v.resident > 0
}

func (b providerGuardianBounds) valid(store providerIntegrationBounds) bool {
	if store.firstAdmission == nil || store.continuation == nil || store.maximumCommits <= 0 || store.maximumObjects <= 0 || store.maximumSetupRecoveries < 0 || !b.lifetime.valid() || !b.claim.valid() ||
		b.attempts == 0 || b.attempts > 1<<30 || b.claim.records != 6 || b.claim.objects != 4 ||
		b.claim.duration > 300000 || b.lifetime.duration > 1<<53 ||
		b.claim.live != b.claim.retained || b.lifetime.live != b.lifetime.retained ||
		b.lifetime.retained > store.maximumStoredBytes || b.lifetime.resident > store.continuation.maximumResidentBytes ||
		b.catalogEntries == 0 || b.catalogNameBytes == 0 {
		return false
	}
	page := store.continuation
	if page.pageRecords < 3 || page.pageRecords%3 != 0 || page.maximumPages == 0 {
		return false
	}
	// Admission, non-final page sealing and final inspection all consume the
	// SAME lifetime. The first operation is conservatively included here too.
	claims, claimsOK := providerUpperSum(page.maximumRecords/3, page.maximumPages-1, 1)
	pageBytes, pageOK := providerLedgerPageUpper(page.pageRecords)
	if !claimsOK || b.attempts < claims || !pageOK || pageBytes > store.maximumObjectBytes || providerLedgerSealLimit > store.maximumObjectBytes {
		return false
	}
	for i, n := range b.claim.work {
		total, ok := providerUpperProduct(n, b.attempts)
		if !ok || total > b.lifetime.work[i] {
			return false
		}
	}
	for _, row := range [][2]uint64{
		{b.claim.retained, b.lifetime.retained},
		{b.claim.live, b.lifetime.live},
		{b.claim.clocks, b.lifetime.clocks},
		{b.claim.records, b.lifetime.records},
		{b.claim.objects, b.lifetime.objects},
		{b.claim.raw, b.lifetime.raw},
		{b.claim.metadata, b.lifetime.metadata},
	} {
		total, ok := providerUpperProduct(row[0], b.attempts)
		if !ok || total > row[1] {
			return false
		}
	}
	// Records, objects, commits and directory names have a single global
	// lifetime cap alongside the existing T/C/S and main-custody counters.
	// The six auxiliary records per G claim are only a reserved subset. All
	// main objects (including page/seal/index/budget/root/T/C/S) retain their
	// own cap. Include both main capacities conservatively in the global cap.
	records, a := providerUpperSum(store.continuation.maximumRecords, uint64(store.maximumObjects), b.lifetime.records, 2)
	objects, c := providerUpperSum(uint64(store.maximumObjects), b.lifetime.objects, 1)
	commits, d := providerUpperSum(uint64(store.maximumCommits), b.attempts, uint64(store.maximumSetupRecoveries), 1)
	names, e := providerUpperProduct(b.attempts, 5)
	mainNames, mainOK := providerUpperProduct(uint64(store.maximumCommits), 7)
	setupCount, setupCountOK := providerUpperSum(uint64(store.maximumSetupRecoveries), 1)
	setupNames, setupOK := providerUpperProduct(setupCount, 5)
	names, f := providerUpperSum(names, mainNames, setupNames, uint64(store.maximumObjects), 32)
	nameBytes, g := providerUpperProduct(names, 255)
	return a && c && d && e && mainOK && setupCountOK && setupOK && f && g && records <= b.privateRecords &&
		objects <= b.objects && commits <= b.commits &&
		names <= b.catalogEntries && nameBytes <= b.catalogNameBytes &&
		b.claim.live <= b.lifetime.live && b.claim.resident <= b.lifetime.resident &&
		b.claim.duration <= b.lifetime.duration
}

func (v providerGuardianCapacity) object() map[string]any {
	m := map[string]any{"work_11": candidateLedgerWorkObject(v.work)}
	for k, n := range map[string]uint64{
		"retained_bytes": v.retained, "live_bytes": v.live,
		"clock_reads": v.clocks, "duration_ms": v.duration,
		"records": v.records, "objects": v.objects, "raw_bytes": v.raw,
		"metadata_bytes": v.metadata, "resident_bytes": v.resident,
	} {
		m[k] = strconv.FormatUint(n, 10)
	}
	return m
}

func (b providerGuardianBounds) object() map[string]any {
	m := map[string]any{"format": "sage-provider-guardian-bounds-v1", "lifetime": b.lifetime.object(), "per_claim": b.claim.object()}
	for k, n := range map[string]uint64{
		"maximum_attempts": b.attempts, "maximum_private_records": b.privateRecords,
		"maximum_objects": b.objects,
		"maximum_commits": b.commits, "maximum_catalog_entries": b.catalogEntries,
		"maximum_catalog_name_bytes": b.catalogNameBytes,
	} {
		m[k] = strconv.FormatUint(n, 10)
	}
	return m
}

func providerGuardianCapacityValue(value any) (providerGuardianCapacity, bool) {
	m := contentObject(value, "work_11", "retained_bytes", "live_bytes", "clock_reads", "duration_ms", "records", "objects", "raw_bytes", "metadata_bytes", "resident_bytes")
	var v providerGuardianCapacity
	work, ok := candidateAdmissionWork(m["work_11"])
	if m == nil || !ok {
		return v, false
	}
	v.work = work
	for key, target := range map[string]*uint64{
		"retained_bytes": &v.retained, "live_bytes": &v.live,
		"clock_reads": &v.clocks, "duration_ms": &v.duration,
		"records": &v.records, "objects": &v.objects, "raw_bytes": &v.raw,
		"metadata_bytes": &v.metadata, "resident_bytes": &v.resident,
	} {
		n, valid := providerCandidateUint(m[key], true)
		if !valid {
			return providerGuardianCapacity{}, false
		}
		*target = n
	}
	return v, v.valid()
}

// Four guardian records (intent/claim/commit/head), the main admission
// binding and an optional local command are six distinct typed records. Only
// the first four are outside the ordinary main object inventory. A claim has
// five possible filesystem allocations, including the pending HEAD copy;
// the new T has its own at-most-seven main files plus its fresh objects.
// Main T bytes/files and guardian bytes/files are G-funded, C/S files P-funded.
func providerGuardianBoundsValue(value any, store providerIntegrationBounds) (providerGuardianBounds, bool) {
	m := contentObject(value, "format", "lifetime", "per_claim", "maximum_attempts", "maximum_private_records", "maximum_objects", "maximum_commits", "maximum_catalog_entries", "maximum_catalog_name_bytes")
	var b providerGuardianBounds
	if m == nil || m["format"] != "sage-provider-guardian-bounds-v1" {
		return b, false
	}
	lifetime, a := providerGuardianCapacityValue(m["lifetime"])
	claim, c := providerGuardianCapacityValue(m["per_claim"])
	if !a || !c {
		return b, false
	}
	b.lifetime, b.claim = lifetime, claim
	for key, target := range map[string]*uint64{
		"maximum_attempts": &b.attempts, "maximum_private_records": &b.privateRecords,
		"maximum_objects": &b.objects, "maximum_commits": &b.commits,
		"maximum_catalog_entries": &b.catalogEntries, "maximum_catalog_name_bytes": &b.catalogNameBytes,
	} {
		n, ok := providerCandidateUint(m[key], true)
		if !ok {
			return providerGuardianBounds{}, false
		}
		*target = n
	}
	return b, b.valid(store)
}

func (x *providerScanCodec) guardianBoundsBytes() ([]byte, error) {
	if x == nil || x.config.bounds.guardian == nil {
		return nil, errProviderIntegration
	}
	raw, err := x.encode(x.config.bounds.guardian.object())
	if err != nil {
		return nil, errProviderIntegration
	}
	value, ok := providerGuardianBoundsValue(x.decode(raw), x.config.bounds)
	if !ok || value != *x.config.bounds.guardian {
		return nil, errProviderIntegration
	}
	canonical, err := x.encode(value.object())
	if err != nil || !x.equal(raw, canonical) {
		return nil, errProviderIntegration
	}
	return raw, nil
}
