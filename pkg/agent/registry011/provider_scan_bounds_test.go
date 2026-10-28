package registry011

import "testing"

func TestProviderScanBoundsRefuseUnboundedAndInconsistentProfiles(t *testing.T) {
	s := providerScanStoreFixture(t)
	base := s.config.bounds
	for _, entry := range []struct {
		name   string
		change func(*providerScanBounds)
	}{
		{"attempt-zero", func(b *providerScanBounds) { b.maximumAttempts = 0 }},
		{"attempt-cap", func(b *providerScanBounds) { b.maximumAttempts = 65 }},
		{"records-zero", func(b *providerScanBounds) { b.maximumRecords = 0 }},
		{"records-cap", func(b *providerScanBounds) { b.maximumRecords = ^uint64(0) }},
		{"objects-zero", func(b *providerScanBounds) { b.maximumObjects = 0 }},
		{"objects-cap", func(b *providerScanBounds) { b.maximumObjects = 1<<30 + 1 }},
		{"raw-zero", func(b *providerScanBounds) { b.maximumRawBytes = 0 }},
		{"raw-cap", func(b *providerScanBounds) { b.maximumRawBytes = 1<<50 + 1 }},
		{"pages-zero", func(b *providerScanBounds) { b.maximumPages = 0 }},
		{"pages-cap", func(b *providerScanBounds) { b.maximumPages++ }},
		{"chunks-zero", func(b *providerScanBounds) { b.maximumChunks = 0 }},
		{"chunks-cap", func(b *providerScanBounds) { b.maximumChunks = b.maximumRecords + 1 }},
		{"chunk-records", func(b *providerScanBounds) { b.chunkRecords = 0 }},
		{"chunk-records-cap", func(b *providerScanBounds) { b.chunkRecords = b.maximumRecords + 1 }},
		{"chunk-objects", func(b *providerScanBounds) { b.chunkObjects = 0 }},
		{"chunk-objects-cap", func(b *providerScanBounds) { b.chunkObjects = b.maximumObjects + 1 }},
		{"chunk-raw", func(b *providerScanBounds) { b.chunkRawBytes = 0 }},
		{"chunk-raw-cap", func(b *providerScanBounds) { b.chunkRawBytes = b.maximumRawBytes + 1 }},
		{"ram-zero", func(b *providerScanBounds) { b.maximumResidentBytes = 0 }},
		{"ram-cap", func(b *providerScanBounds) { b.maximumResidentBytes++ }},
		{"ram-codec", func(b *providerScanBounds) { b.maximumResidentBytes = 1 }},
		{"entry-zero", func(b *providerScanBounds) { b.filenameEntryOverhead = 0 }},
		{"entry-cap", func(b *providerScanBounds) { b.filenameEntryOverhead = 1<<20 + 1 }},
		{"filename-zero", func(b *providerScanBounds) { b.maximumFilenameBytes = 0 }},
		{"filename-cap", func(b *providerScanBounds) { b.maximumFilenameBytes = 256 }},
		{"clock-zero", func(b *providerScanBounds) { b.maximumClockReads = 0 }},
		{"clock-cap", func(b *providerScanBounds) { b.maximumClockReads = 1<<20 + 1 }},
		{"duration-zero", func(b *providerScanBounds) { b.chunkDurationMS = 0 }},
		{"duration-cap", func(b *providerScanBounds) { b.chunkDurationMS = 300001 }},
		{"total-duration", func(b *providerScanBounds) { b.totalDurationMS = b.chunkDurationMS - 1 }},
		{"total-duration-cap", func(b *providerScanBounds) { b.totalDurationMS = 1<<53 + 1 }},
		{"catalog-zero", func(b *providerScanBounds) { b.maximumCatalogEntries = 0 }},
		{"catalog-cap", func(b *providerScanBounds) { b.maximumCatalogEntries = 32<<20 + 1 }},
		{"catalog-name-zero", func(b *providerScanBounds) { b.maximumCatalogNameBytes = 0 }},
		{"catalog-name-cap", func(b *providerScanBounds) { b.maximumCatalogNameBytes = 16<<30 + 1 }},
		{"retained-zero", func(b *providerScanBounds) { b.maximumPreIRetainedBytes = 0 }},
		{"retained-cap", func(b *providerScanBounds) { b.maximumPreIRetainedBytes++ }},
		{"pre-clock-zero", func(b *providerScanBounds) { b.maximumPreIClockReads = 0 }},
		{"pre-clock-cap", func(b *providerScanBounds) { b.maximumPreIClockReads = 1<<20 + 1 }},
		{"clock-common", func(b *providerScanBounds) { b.maximumPreIClockReads = 1 }},
		{"work-coordinate", func(b *providerScanBounds) { b.chunkWork[3] = 1 }},
		{"network", func(b *providerScanBounds) { b.chunkWork[0] = 1; b.maximumPreIWork[0] = 1 }},
		{"wire", func(b *providerScanBounds) { b.chunkWork[1] = 1; b.maximumPreIWork[1] = 1 }},
		{"decode", func(b *providerScanBounds) { b.chunkWork[2] = 0 }},
		{"work", func(b *providerScanBounds) { b.chunkWork[10] = 0 }},
		{"disk", func(b *providerScanBounds) { b.maximumPreIWork[9] = 0 }},
	} {
		t.Run(entry.name, func(t *testing.T) {
			store := base
			b := *base.scan
			entry.change(&b)
			store.scan = &b
			if store.valid() {
				t.Fatal("invalid finite scan profile accepted")
			}
		})
	}
	store := base
	store.continuation = nil
	if store.valid() {
		t.Fatal("V3 adopted the V1 lifetime")
	}
}

func TestProviderScanBodyAndSetupUpperComeFromTheWholeSelectedProfile(t *testing.T) {
	s := providerScanStoreFixture(t)
	body, err := providerScanBodyUpper(s.config)
	if err != nil || body.clocks != s.config.bounds.scan.maximumClockReads || body.retained == 0 || body.work[2] <= s.config.bounds.scan.chunkWork[2] || body.work[10] <= s.config.bounds.scan.chunkWork[10] || !providerScanSetupFits(s.config) {
		t.Fatal("full selected scan upper missing", err)
	}
	a := providerScanAttemptFixture(t, s, providerScanHighwater{})
	a.body.work[2]--
	before := s.stored
	if s.scanControl.claim(a) == nil || s.stored != before || s.scanControl.failed {
		t.Fatal("smaller caller body upper reached admission")
	}
	config := s.config
	config.bounds.setupWork[2] = 1
	if providerScanSetupFits(config) {
		t.Fatal("V3 H0 reads received a free setup codec envelope")
	}
	if _, err := providerScanBodyUpper(providerCustodyConfig{}); err == nil {
		t.Fatal("missing profile")
	}
	config = s.config
	b := *config.bounds.scan
	config.bounds.scan = &b
	b.maximumPreIWork[10] = 1
	if _, err := providerScanBodyUpper(config); err == nil {
		t.Fatal("whole body escaped the common work ceiling")
	}
}

func TestProviderScanCodecMeasuresRepeatedCallsAndRefusesBeforeUnfundedEncoding(t *testing.T) {
	s := providerScanStoreFixture(t)
	c := s.scanControl
	if codec, err := newProviderScanCodec(s.config, &providerWorkMeter{}); err == nil || codec != nil {
		t.Fatal("unowned codec created")
	}
	m := &providerWorkMeter{scan: c, upper: s.config.bounds.scan.maximumPreIWork, maximumClocks: 1}
	m.owner = m
	x, err := newProviderScanCodec(s.config, m)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x.highwater(providerScanHighwater{})
	if err != nil {
		t.Fatal(err)
	}
	before := m.actual
	parsed, err := x.parseHighwater(raw)
	if err != nil || parsed != (providerScanHighwater{}) || m.actual[2] <= before[2] || m.actual[10] <= before[10] {
		t.Fatal("reparse/reencode was not observed", err)
	}
	if x.hash(providerScanHighwaterDomain, raw) == "" || !x.equal(raw, raw) || x.equal(raw, []byte("different")) {
		t.Fatal("observed digest/exact bytes")
	}
	bad := s.config
	bad.profile = admissionTestDigest("f")
	if _, err := newProviderScanCodec(bad, m); err == nil {
		t.Fatal("different profile binding")
	}
	for _, raw := range [][]byte{nil, []byte("[]"), make([]byte, candidateLedgerMetadataLimit+1)} {
		if _, err := x.parseHighwater(raw); err == nil {
			t.Fatal("finite invalid raw highwater")
		}
	}
	if _, err := x.encode(nil); err == nil {
		t.Fatal("nil codec object")
	}
	zero := &providerWorkMeter{scan: c}
	zero.owner = zero
	if _, err := newProviderScanCodec(s.config, zero); err == nil || !zero.failed || zero.actual != (candidateWorkVector{}) {
		t.Fatal("unfunded encoding consumed an unmeasured buffer")
	}
}

func TestProviderScanControlLateDeadlineRefusalKeepsTheEntireClaim(t *testing.T) {
	s := providerScanStoreFixture(t)
	a := providerScanAttemptFixture(t, s, providerScanHighwater{})
	a.deadline++
	before := s.stored
	held, _ := providerScanUpperSum(a.preparation, a.body)
	if s.scanControl.claim(a) == nil || !s.failed || !s.scanControl.failed || s.scanControl.held != held || s.stored != before+held.retained || s.scanControl.prepared != nil || s.activeMeter.Load() != nil {
		t.Fatal("late refusal refunded or completed an attempt")
	}
}
