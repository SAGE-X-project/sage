package registry011

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func providerScanReceiptStoreFixture(t *testing.T) (*providerCustodyStore, []byte) {
	t.Helper()
	first := providerFirstAdmissionBoundsFixture()
	s := providerScanEnrollmentOriginalsWithFirst(t, &first)
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

func TestProviderScanReceiptsRetainBothOriginalCopiesAndFinishBothCatalogs(t *testing.T) {
	s, raw := providerScanReceiptStoreFixture(t)
	held, stored, retained := s.scanControl.held, s.stored, s.retainedRaw
	result, err := s.scanControl.inspectBeforeAdmission(raw)
	if err != nil || result == nil {
		t.Fatal("ordinary V4 retained inspection", err)
	}
	b := s.scanBody
	r := b.receipts
	if r == nil || len(r.commands) != 2 || len(r.chunks) != 2 || len(result.chunks) != 2 || len(b.observed) != len(b.required) || s.scanControl.held != held || s.stored != stored || s.retainedRaw <= retained || s.last != nil || b.active || b.failed || b.meter.failed || b.beforeFirst == nil || s.activeMeter.Load() != nil {
		t.Fatal("retained inspection omitted originals, borrowed first-E1 or minted an ACK")
	}
	x, err := newProviderScanCodec(s.config, nil)
	if err != nil {
		t.Fatal(err)
	}
	var physicalBytes uint64
	for index, chunk := range result.chunks {
		m := r.decode(x, "chunk", chunk)
		if m == nil || m["format"] != providerScanChunkV2 || m["scan_head"] != r.head || m["scan_highwater"] != r.highwater || m["scan_attempt"] != s.scanControl.pin.intent || m["command_digest"] != r.commands[index] || providerCandidateHash(providerScanChunkV2, chunk) != r.chunks[index] {
			t.Fatal("chunk was borrowed from another control attempt")
		}
		for _, pair := range []struct{ kind, domain, digest string }{{"command", providerScanCommandV2, r.commands[index]}, {"chunk", providerScanChunkV2, r.chunks[index]}} {
			var first []byte
			for _, disk := range []*retentionDisk{s.custody, s.evidence} {
				original, err := disk.read(providerScanReceiptName(pair.kind, pair.digest, nil), candidateLedgerMetadataLimit)
				if err != nil || r.decode(x, pair.kind, original) == nil || providerCandidateHash(pair.domain, original) != pair.digest || first != nil && !bytes.Equal(first, original) {
					t.Fatal("missing exact immutable original pair", err)
				}
				physicalBytes += uint64(len(original))
				first = original
			}
		}
	}
	if s.retainedRaw != retained+physicalBytes || r.valid() || r.UnmarshalJSON([]byte(`{}`)) == nil || r.verifyPrevious(x, result.target, 2) {
		t.Fatal("receipt DATA restored ownership or physical copies were charged twice")
	}
}

// A finite observer-state unit is not a restored production scan or permit.
func providerScanReceiptObserverUnit(t *testing.T) (*providerCustodyStore, *providerScanReceipts, *providerScanCodec) {
	t.Helper()
	s, _ := providerScanReceiptStoreFixture(t)
	b := &providerScanBody{control: s.scanControl, required: make(map[string]byte), observed: make(map[string]bool), active: true, window: true, windowLimit: s.config.bounds.scan.chunkWork}
	b.owner = b
	m := &providerWorkMeter{scan: s.scanControl, scanBody: b, upper: s.scanControl.prepared.body.work, maximumClocks: s.scanControl.prepared.body.clocks}
	m.owner, b.meter = m, m
	s.scanBody = b
	s.activeMeter.Store(m)
	t.Cleanup(func() { s.activeMeter.Store(nil) })
	x, err := newProviderScanCodec(s.config, m)
	if err != nil {
		t.Fatal(err)
	}
	r, err := newProviderScanReceipts(b, x)
	if err != nil {
		t.Fatal(err)
	}
	b.receipts = r
	return s, r, x
}

func TestProviderScanReceiptUnitsRefuseMissingOriginalsAndForeignLiveOwners(t *testing.T) {
	for _, mode := range []string{"owner", "missing-original", "incomplete-pair", "command-order", "invalid-name", "invalid-data", "required-quota", "name-quota", "closed-second-root"} {
		t.Run(mode, func(t *testing.T) {
			s, r, x := providerScanReceiptObserverUnit(t)
			held := s.scanControl.held
			raw, err := r.command(x, s.scanControl.prepared.target, 0, "", providerScanCursor{"setup", 0})
			if err != nil {
				t.Fatal(err)
			}
			digest, kind := x.hash(providerScanCommandV2, raw), "command"
			switch mode {
			case "owner":
				r.body.owner = nil
			case "missing-original":
				r.commands, r.chunks = []string{digest}, []string{admissionTestDigest("b")}
				if r.verifyPrevious(x, s.scanControl.prepared.target, 1) || s.last != nil || s.scanControl.held != held {
					t.Fatal("missing fixed original state was adopted")
				}
				return
			case "incomplete-pair":
				r.commands = []string{digest}
				if r.verifyPrevious(x, s.scanControl.prepared.target, 1) {
					t.Fatal("missing chunk completed a pair")
				}
				return
			case "command-order":
				kind = "chunk"
			case "invalid-name":
				digest = "invalid"
			case "invalid-data":
				raw = nil
			case "required-quota":
				for n := uint64(0); n < s.config.bounds.scan.maximumCatalogEntries; n++ {
					r.body.required[providerCustodyName("inert-name", n)] = 1
				}
			case "name-quota":
				r.body.requiredNameBytes = s.config.bounds.scan.maximumCatalogNameBytes
			case "closed-second-root":
				_ = s.evidence.root.Close()
			}
			if r.retain(x, kind, digest, raw) || s.scanControl.held != held || s.last != nil {
				t.Fatal("finite invalid receipt advanced a claim or ACK", mode)
			}
		})
	}
}

func TestProviderScanReceiptClosedSchemaRejectsUnknownOrMissingFields(t *testing.T) {
	s, r, x := providerScanReceiptObserverUnit(t)
	raw, err := r.command(x, s.scanControl.prepared.target, 0, "", providerScanCursor{"setup", 0})
	if err != nil {
		t.Fatal(err)
	}
	for key := range contentJSON(raw).(map[string]any) {
		m := contentJSON(raw).(map[string]any)
		delete(m, key)
		if r.decode(x, "command", providerIntegrationJSON(m)) != nil {
			t.Fatal("missing closed field accepted", key)
		}
	}
	m := contentJSON(raw).(map[string]any)
	m["unrecognized"] = "finite DATA"
	if r.decode(x, "command", providerIntegrationJSON(m)) != nil || r.decode(x, "other", raw) != nil {
		t.Fatal("unknown receipt field or kind accepted")
	}
	if _, err := newProviderScanReceipts(r.body, x); err == nil {
		t.Fatal("a second live receipt owner was created")
	}
}

func TestProviderScanReceiptInspectionCannotBorrowFirstAdmissionFunds(t *testing.T) {
	for _, mode := range []string{"bytes", "clock", "final-catalog"} {
		t.Run(mode, func(t *testing.T) {
			_, r, _ := providerScanReceiptObserverUnit(t)
			b, meter := r.body, r.body.meter
			before := meter.actual
			upper := providerScanUpper{work: before, clocks: meter.maximumClocks}
			for k := range upper.work {
				upper.work[k] += 100
			}
			b.beforeFirst, b.encoding = &upper, true
			switch mode {
			case "bytes":
				upper.work[2] = before[2]
			case "clock":
				upper.clocks = meter.clocks
			case "final-catalog":
				b.encoding, b.finalCatalog = false, true
				b.finalCatalogStart = meter.actual
				b.finalCatalogLimit = candidateWorkVector{}
			}
			kind, eventBytes := providerCodecEvent, uint64(1)
			if mode == "clock" {
				kind, eventBytes = providerClockEvent, 0
			}
			if meter.event(kind, eventBytes) || !meter.failed || meter.actual != before {
				t.Fatal("inspection borrowed a first-E1 or final-catalog allowance")
			}
		})
	}
}

func TestProviderScanReceiptFailuresRetainTheCompleteClaim(t *testing.T) {
	s, raw := providerScanReceiptStoreFixture(t)
	if os.WriteFile(filepath.Join(s.config.evidenceDirectory, "unrelated-finite-note"), []byte("DATA"), 0600) != nil {
		t.Fatal("owned finite catalog fixture")
	}
	held, stored := s.scanControl.held, s.stored
	if result, err := s.scanControl.inspectBeforeAdmission(raw); err == nil || result != nil || !s.failed || s.scanControl.held != held || s.stored != stored || s.last != nil {
		t.Fatal("incomplete V4 catalog refunded a held claim or granted permission")
	}
}

func TestProviderScanReceiptNamesChargeEachActualCopyBeforeAllocation(t *testing.T) {
	_, r, _ := providerScanReceiptObserverUnit(t)
	m := r.body.meter
	digest := admissionTestDigest("a")
	want := providerScanReceiptName("command", digest, nil)
	units := uint64(len(want) + 1)
	before := m.actual
	for n := 0; n < 2; n++ {
		if got := providerScanReceiptName("command", digest, m); got != want {
			t.Fatal("ordinary finite filename copy")
		}
	}
	expected := before
	expected[10] += 2 * units
	if m.actual != expected || providerScanReceiptName("other", digest, m) != "" || providerScanReceiptName("chunk", "invalid", m) != "" || m.actual != expected {
		t.Fatal("name copy used a prior charge or invalid schema allocated output")
	}
	m.upper[10] = m.actual[10] + units - 1
	if got := providerScanReceiptName("command", digest, m); got != "" || !m.failed || m.actual != expected {
		t.Fatal("one-unit-short copy allocated a filename before admission")
	}
}
