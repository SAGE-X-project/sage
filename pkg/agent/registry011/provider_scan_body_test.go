package registry011

import (
	"os"
	"path/filepath"
	"testing"
)

func providerScanBodyFixture(t *testing.T) (*providerCustodyStore, []byte) {
	t.Helper()
	s := providerScanEnrollmentOriginals(t)
	x, err := newProviderScanCodec(s.config, nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := x.target(providerScanTarget{setup: s.setupTip, custody: s.tip, frontier: s.frontierFiles.pin.frontier, objects: s.objectsRoot, enrollment: s.enrollment})
	if err != nil {
		t.Fatal(err)
	}
	a := providerScanAttemptFixture(t, s, s.scanControl.pin)
	a.target = x.hash("sage-provider-scan-target-v1", raw)
	if err := s.scanControl.claim(a); err != nil {
		t.Fatal(err)
	}
	return s, raw
}

func TestProviderScanBodyClosesBothCatalogsWithOneOriginalMeter(t *testing.T) {
	s, raw := providerScanBodyFixture(t)
	held, stored := s.scanControl.held, s.stored
	result, err := s.scanControl.inspectBeforeAdmission(raw)
	if err != nil || result == nil || result.records != 2 || len(result.chunks) != 2 || result.objects == 0 || result.rawBytes == 0 || result.work[2] == 0 || result.work[10] == 0 || s.scanBody == nil || s.scanBody.active || s.scanBody.failed || len(s.scanBody.observed) != len(s.scanBody.required) || s.last != nil || s.activeMeter.Load() != nil || s.scanControl.held != held || s.stored != stored {
		t.Fatal("ordinary whole pre-I target did not close as DATA only", err)
	}
	var previous string
	var work candidateWorkVector
	for index, chunk := range result.chunks {
		m := contentJSON(chunk).(map[string]any)
		if m["format"] != "sage-provider-scan-chunk-v1" || m["index"] != []string{"0", "1"}[index] || m["records"] != "1" || m["previous_chunk"] != providerNullableDigest(previous) || m["target"] != result.target {
			t.Fatal("chunk does not name exact private cursor chain")
		}
		actual, ok := candidateAdmissionWork(m["work"])
		if !ok {
			t.Fatal("missing observed work")
		}
		for coordinate, n := range actual {
			work[coordinate] += n
		}
		previous = providerCandidateHash("sage-provider-scan-chunk-v1", chunk)
	}
	if work != result.work || s.scanBody.meter.actual[10] <= result.work[10] || result.UnmarshalJSON([]byte(`{}`)) == nil || s.scanBody.UnmarshalJSON([]byte(`{}`)) == nil {
		t.Fatal("output encoding was counted as earlier work or DATA restored an owner")
	}
	if got, err := s.scanControl.inspectBeforeAdmission(raw); err == nil || got != nil {
		t.Fatal("completed private body resumed or replenished its claim")
	}
	if _, err := s.read(s.custody, "ANCHOR", int64(s.config.bounds.maximumMetadataPerCommit)); err != nil {
		t.Fatal("inactive body intercepted ordinary later DATA reads", err)
	}
}

func TestProviderScanBodyIncompleteCatalogAndOriginalsRetainTheWholeClaim(t *testing.T) {
	for _, mode := range []string{"unknown-file", "directory", "nonempty-lock", "missing-anchor", "missing-head", "wrong-target", "clock", "chunk-objects", "chunk-bytes"} {
		t.Run(mode, func(t *testing.T) {
			s, raw := providerScanBodyFixture(t)
			switch mode {
			case "unknown-file":
				if os.WriteFile(filepath.Join(s.config.evidenceDirectory, "ordinary-note"), []byte("unrelated finite DATA"), 0600) != nil {
					t.Fatal("owned unrelated file")
				}
			case "directory":
				if os.Mkdir(filepath.Join(s.config.evidenceDirectory, "ordinary-directory"), 0700) != nil {
					t.Fatal("owned unrelated directory")
				}
			case "nonempty-lock":
				if os.WriteFile(filepath.Join(s.config.evidenceDirectory, "AUTHORITY.LOCK"), []byte("ordinary invalid lock DATA"), 0600) != nil {
					t.Fatal("owned finite lock-size scenario")
				}
			case "missing-anchor":
				_ = os.Remove(filepath.Join(s.config.evidenceDirectory, "ANCHOR"))
			case "missing-head":
				_ = os.Remove(filepath.Join(s.config.custodyDirectory, "HEAD"))
			case "wrong-target":
				raw = nil
			case "clock":
				s.clock.mu.Lock()
				s.clock.read = s.clock.maximum
				s.clock.mu.Unlock()
			case "chunk-objects", "chunk-bytes":
				// Only private finite bounds are reduced in this refusal unit.
				bounds := *s.config.bounds.scan
				if mode == "chunk-objects" {
					bounds.chunkObjects = 1
				} else {
					bounds.chunkRawBytes = 1
				}
				s.config.bounds.scan = &bounds
			}
			held, stored := s.scanControl.held, s.stored
			if result, err := s.scanControl.inspectBeforeAdmission(raw); err == nil || result != nil || s.scanControl.held != held || s.stored != stored || s.last != nil || s.activeMeter.Load() != nil {
				t.Fatal("an incomplete body returned completion or refunded its claim", mode)
			}
		})
	}
}

func TestProviderScanBodyMeterWindowsKeepCumulativeActualAndRefuseForeignOwners(t *testing.T) {
	s, _ := providerScanBodyFixture(t)
	b := &providerScanBody{control: s.scanControl, active: true, window: true, windowLimit: candidateWorkVector{0, 0, 100, 0, 0, 0, 0, 0, 0, 0, 1000}}
	b.owner = b
	m := &providerWorkMeter{scan: s.scanControl, scanBody: b, upper: candidateWorkVector{0, 0, 1000, 0, 0, 0, 0, 0, 0, 0, 10000}, maximumClocks: 100}
	m.owner, b.meter = m, m
	s.scanBody = b
	if !m.event(providerReadEvent, 8) {
		t.Fatal("ordinary first window")
	}
	first := m.actual
	b.windowStart = first
	if !m.event(providerReadEvent, 8) || m.actual[10] != 2*first[10] {
		t.Fatal("next window reset original actual")
	}
	b.windowLimit[10] = first[10]
	if m.preflight(providerReadEvent, 0) || !m.failed || m.actual[10] != 2*first[10] {
		t.Fatal("exhausted subwindow borrowed global unused work")
	}
	for _, mode := range []string{"owner", "body-owner", "inactive", "failed", "no-window", "underflow"} {
		t.Run(mode, func(t *testing.T) {
			body := &providerScanBody{control: s.scanControl, active: true, window: true, windowLimit: b.windowLimit}
			body.owner = body
			meter := &providerWorkMeter{scan: s.scanControl, scanBody: body, upper: m.upper}
			meter.owner, body.meter = meter, meter
			s.scanBody = body
			switch mode {
			case "owner":
				meter.owner = nil
			case "body-owner":
				body.owner = nil
			case "inactive":
				body.active = false
			case "failed":
				body.failed = true
			case "no-window":
				body.window = false
			case "underflow":
				body.windowStart[0] = 1
			}
			if meter.event(providerReadEvent, 1) || meter.actual != (candidateWorkVector{}) {
				t.Fatal("invalid owner or window advanced actual")
			}
		})
	}
}

func TestProviderScanBodyRejectsMissingOwnerOrUnclaimedEmptyTarget(t *testing.T) {
	var missing *providerScanControl
	if result, err := missing.inspectBeforeAdmission(nil); err == nil || result != nil {
		t.Fatal("nil control created scan DATA")
	}
	s := providerScanStoreFixture(t)
	if result, err := s.scanControl.inspectBeforeAdmission(nil); err == nil || result != nil || s.scanBody != nil {
		t.Fatal("unclaimed empty target became a completed scan")
	}
}

// Observer-state units do not restore a scan result or supply external counts.
// Successful count checks below go through the concrete original-file reader.
func providerScanBodyObserverUnit(t *testing.T) (*providerCustodyStore, *providerScanBody) {
	t.Helper()
	s, _ := providerScanBodyFixture(t)
	b := &providerScanBody{control: s.scanControl, active: true, window: true, required: make(map[string]byte), observed: make(map[string]bool), windowLimit: s.config.bounds.scan.chunkWork}
	b.owner = b
	m := &providerWorkMeter{scan: s.scanControl, scanBody: b, upper: s.scanControl.prepared.body.work, maximumClocks: s.config.bounds.scan.maximumClockReads}
	m.owner, b.meter = m, m
	s.scanBody = b
	s.activeMeter.Store(m)
	t.Cleanup(func() { s.activeMeter.Store(nil) })
	return s, b
}

func TestProviderScanBodyCountsRepeatedOriginalBytesSeparatelyFromControlReads(t *testing.T) {
	s, b := providerScanBodyObserverUnit(t)
	maximum := int64(s.config.bounds.maximumMetadataPerCommit)
	raw, err := s.read(s.custody, "ANCHOR", maximum)
	if err != nil || b.objects != 1 || b.rawBytes != uint64(len(raw)) {
		t.Fatal("concrete target read omitted its count", err)
	}
	if _, err := s.read(s.custody, "ANCHOR", maximum); err != nil || b.objects != 2 || b.rawBytes != uint64(2*len(raw)) {
		t.Fatal("repeated original read borrowed its first byte count", err)
	}
	key, ok := b.key(s.custody, "SCAN-HEAD")
	if !ok {
		t.Fatal("finite control classification")
	}
	b.required[key] = 2
	before := b.meter.actual
	if _, err := s.read(s.custody, "SCAN-HEAD", maximum); err != nil || b.objects != 2 || b.rawBytes != uint64(2*len(raw)) || b.meter.actual[10] <= before[10] {
		t.Fatal("control read changed target byte counts or omitted actual work", err)
	}
}

func TestProviderScanBodyObserverRefusesUnownedOrExhaustedTargetReads(t *testing.T) {
	for _, mode := range []string{"owner", "inactive", "window", "foreign-root", "filename", "required-names", "required-bytes", "objects", "bytes", "chunk-objects", "chunk-bytes"} {
		t.Run(mode, func(t *testing.T) {
			s, b := providerScanBodyObserverUnit(t)
			disk, name := s.custody, "ANCHOR"
			switch mode {
			case "owner":
				b.owner = nil
			case "inactive":
				b.active = false
			case "window":
				b.window = false
			case "foreign-root":
				disk = nil
			case "filename":
				name = string(make([]byte, 256)) // Bounded inert invalid name.
			case "required-names":
				for n := uint64(0); n < s.config.bounds.scan.maximumCatalogEntries; n++ {
					b.required[providerCustodyName("finite-unit", n)] = 1
				}
			case "required-bytes":
				b.requiredNameBytes = s.config.bounds.scan.maximumCatalogNameBytes
			case "objects":
				b.objects = s.config.bounds.scan.maximumObjects
			case "bytes":
				b.rawBytes = s.config.bounds.scan.maximumRawBytes
			case "chunk-objects":
				b.chunkObjects = s.config.bounds.scan.chunkObjects
			case "chunk-bytes":
				b.chunkBytes = s.config.bounds.scan.chunkRawBytes
			}
			objects, rawBytes := b.objects, b.rawBytes
			if b.beforeRead(disk, name, 1) || b.objects != objects || b.rawBytes != rawBytes {
				t.Fatal("unowned or exhausted read advanced target counters", mode)
			}
		})
	}
}

// These finite observer-state scenarios verify refusal without performing a
// filesystem read or treating a caller's byte count as retained evidence.
func TestProviderScanBodyAfterReadRefusesLostOwnerRootOrExhaustedCount(t *testing.T) {
	for _, mode := range []string{"inactive", "foreign-root", "exhausted"} {
		t.Run(mode, func(t *testing.T) {
			s, b := providerScanBodyObserverUnit(t)
			disk := s.custody
			switch mode {
			case "inactive":
				b.active = false
			case "foreign-root":
				disk = nil
			case "exhausted":
				b.objects = s.config.bounds.scan.maximumObjects
			}
			objects, raw := b.objects, b.rawBytes
			if b.afterRead(disk, "ANCHOR", 1) || b.objects != objects || b.rawBytes != raw {
				t.Fatal("post-read refusal changed the raw target counters", mode)
			}
		})
	}
}

func TestProviderScanBodyControlsRefuseIncompleteBothRootCatalogReservation(t *testing.T) {
	for _, entries := range []uint64{1, 3, 6} {
		t.Run(providerCustodyName("entries", entries), func(t *testing.T) {
			s, b := providerScanBodyObserverUnit(t)
			bounds := *s.config.bounds.scan
			bounds.maximumCatalogEntries = entries
			s.config.bounds.scan = &bounds
			if b.controls() || uint64(len(b.required)) != entries || s.last != nil {
				t.Fatal("partial control-name reservation completed both roots")
			}
		})
	}
}

func TestProviderScanBodyCatalogRequiresEveryOriginalAndSufficientClock(t *testing.T) {
	for _, mode := range []string{"ordinary", "clock", "work", "missing-required", "directory", "closed-root"} {
		t.Run(mode, func(t *testing.T) {
			s, b := providerScanBodyObserverUnit(t)
			for _, disk := range []*retentionDisk{s.custody, s.evidence} {
				names, err := providerDirectoryNames(disk, int(s.config.bounds.scan.maximumCatalogEntries))
				if err != nil {
					t.Fatal(err)
				}
				for name := range names {
					key, ok := b.key(disk, name)
					if !ok {
						t.Fatal("ordinary finite catalog name")
					}
					b.required[key] = 1
				}
			}
			switch mode {
			case "clock":
				s.clock.mu.Lock()
				s.clock.read = s.clock.maximum
				s.clock.mu.Unlock()
			case "work":
				b.windowLimit = candidateWorkVector{}
			case "missing-required":
				b.required["evidence:missing-unit-original"] = 1
			case "directory":
				_ = os.Mkdir(filepath.Join(s.config.evidenceDirectory, "ordinary-folder"), 0700)
			case "closed-root":
				_ = s.evidence.root.Close()
			}
			ok := b.catalog()
			if ok != (mode == "ordinary") || s.last != nil {
				t.Fatal("catalog result did not match bounded original closure", mode)
			}
		})
	}
}
