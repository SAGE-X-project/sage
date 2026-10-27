package registry011

import (
	"os"
	"path/filepath"
	"testing"
)

func providerScanInitialTargetFixture(t *testing.T, s *providerCustodyStore) ([]byte, providerScanAttempt) {
	t.Helper()
	x, err := newProviderScanCodec(s.config, nil)
	if err != nil {
		t.Fatal(err)
	}
	// This is a bounded DATA proposal. The concrete body will rederive/check
	// every original under its already-held claim before returning DATA only.
	raw, err := x.target(providerScanTarget{setup: s.setupTip, custody: s.tip, objects: s.objectsRoot})
	if err != nil {
		t.Fatal(err)
	}
	a := providerScanAttemptFixture(t, s, providerScanHighwater{})
	a.target = x.hash("sage-provider-scan-target-v1", raw)
	return raw, a
}

func TestProviderScanInitialInspectsTheWholeFrozenEmptyPrefix(t *testing.T) {
	s := providerScanStoreFixture(t)
	raw, a := providerScanInitialTargetFixture(t, s)
	if err := s.scanControl.claim(a); err != nil {
		t.Fatal(err)
	}
	held, stored := s.scanControl.held, s.stored
	result, err := s.scanControl.inspectInitial(raw)
	if err != nil {
		t.Fatal(err)
	}
	i := s.initialInspection
	if result == nil || result.target != a.target || result.records != 1 || result.objects < 9 || result.rawBytes == 0 || !candidateLedgerClockAdvance(result.start, result.end) {
		t.Fatal("incomplete empty-prefix observation")
	}
	if i == nil || i.active || !i.completed || i.failed || i.replay == s.frontierReplay || i.replay.indexes == s.frontierReplay.indexes || len(i.observed) != len(i.required) || i.objects != result.objects || i.rawBytes != result.rawBytes || i.replay.body.intent != nil || s.last != nil || s.sequence != 0 || s.enrollment != "" || s.scanControl.held != held || s.stored != stored || s.activeMeter.Load() != nil {
		t.Fatal("inspection copied producer maps, reset a hold or minted authority")
	}
	if result.UnmarshalJSON([]byte(`{}`)) == nil {
		t.Fatal("diagnostic restored authority")
	}
	before := i.objects
	if _, err := s.scanControl.inspectInitial(raw); err == nil || i.objects != before {
		t.Fatal("completed scan resumed from old input")
	}
	if _, err := s.read(s.custody, "ANCHOR", int64(s.config.bounds.maximumMetadataPerCommit)); err != nil {
		t.Fatal("completed diagnostic left an active read observer", err)
	}
	if i.UnmarshalJSON([]byte(`{}`)) == nil {
		t.Fatal("JSON reconstructed a live scan")
	}
}

func TestProviderScanInitialRefusesUnknownTargetOrCatalogWithoutRefund(t *testing.T) {
	for _, mode := range []string{"wrong-digest", "unknown-file", "later-stage"} {
		t.Run(mode, func(t *testing.T) {
			s := providerScanStoreFixture(t)
			raw, a := providerScanInitialTargetFixture(t, s)
			switch mode {
			case "wrong-digest":
				a.target = admissionTestDigest("f")
			case "unknown-file":
				if os.WriteFile(filepath.Join(s.config.evidenceDirectory, "ordinary-note"), []byte("ordinary unrelated file"), 0600) != nil {
					t.Fatal("finite unrelated file fixture")
				}
			case "later-stage":
				x, _ := newProviderScanCodec(s.config, nil)
				var err error
				raw, err = x.target(providerScanTarget{setup: s.setupTip, custody: s.tip, objects: s.objectsRoot, frontier: admissionTestDigest("a"), enrollment: admissionTestDigest("b"), admission: admissionTestDigest("c"), budget: admissionTestDigest("d")})
				if err != nil {
					t.Fatal(err)
				}
				a.target = x.hash("sage-provider-scan-target-v1", raw)
			}
			if err := s.scanControl.claim(a); err != nil {
				t.Fatal(err)
			}
			held, stored := s.scanControl.held, s.stored
			if chunk, err := s.scanControl.inspectInitial(raw); err == nil || chunk != nil || !s.failed || !s.scanControl.failed || s.scanControl.held != held || s.stored != stored || s.last != nil || s.activeMeter.Load() != nil {
				t.Fatal("partial or wrong target became complete/refunded")
			}
		})
	}
}

func TestProviderScanInitialRefusesMissingOwnerOrInsufficientWholeRecord(t *testing.T) {
	var missing *providerScanControl
	if _, err := missing.inspectInitial(nil); err == nil {
		t.Fatal("nil owner")
	}
	s := providerScanStoreFixture(t)
	raw, a := providerScanInitialTargetFixture(t, s)
	if _, err := s.scanControl.inspectInitial(raw); err == nil || s.initialInspection != nil {
		t.Fatal("unclaimed scan started")
	}
	if err := s.scanControl.claim(a); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"clock", "memory", "bytes", "work", "objects"} {
		config := s.config
		scan := *config.bounds.scan
		config.bounds.scan = &scan
		switch mode {
		case "clock":
			scan.maximumClockReads = 1
		case "memory":
			scan.maximumCatalogEntries = 32 << 20
		case "bytes":
			scan.chunkRawBytes = 1
		case "work":
			scan.chunkWork[2] = 1
		case "objects":
			scan.chunkObjects = 1
		}
		if _, _, ok := providerScanInitialReservation(config); ok {
			t.Fatal("whole final record/catalog upper not reserved", mode)
		}
	}
	if _, _, ok := providerScanInitialReservation(providerCustodyConfig{}); ok {
		t.Fatal("missing selected bounds")
	}
}

func TestProviderScanTargetAndCommandAreClosedData(t *testing.T) {
	s := providerScanStoreFixture(t)
	x, _ := newProviderScanCodec(s.config, nil)
	raw, a := providerScanInitialTargetFixture(t, s)
	for key := range contentJSON(raw).(map[string]any) {
		for _, mode := range []string{"missing", "invalid"} {
			m := contentJSON(raw).(map[string]any)
			if mode == "missing" {
				delete(m, key)
			} else {
				m[key] = "invalid"
			}
			if _, err := x.parseTarget(providerIntegrationJSON(m)); err == nil {
				t.Fatal("invalid target", key, mode)
			}
		}
	}
	for _, cursor := range []providerScanCursor{{"unknown", 0}, {"done", 1}, {"main", 0}} {
		if cursor.object() != nil {
			t.Fatal("invalid cursor")
		}
	}
	for _, cursor := range []providerScanCursor{{"setup", 0}, {"main", 1}, {"done", 0}} {
		if cursor.object() == nil {
			t.Fatal("valid cursor")
		}
	}
	for _, mode := range []string{"target", "index", "previous", "initial-cursor", "done"} {
		target, index, previous, cursor := a.target, uint64(0), "", providerScanCursor{"setup", 0}
		switch mode {
		case "target":
			target = "bad"
		case "index":
			index = s.config.bounds.scan.maximumChunks
		case "previous":
			previous = admissionTestDigest("f")
		case "initial-cursor":
			cursor.next = 1
		case "done":
			cursor = providerScanCursor{"done", 0}
		}
		if _, err := x.command(target, index, previous, cursor); err == nil {
			t.Fatal("unsupported command", mode)
		}
	}
	for _, target := range []providerScanTarget{
		{}, {setup: s.setupTip, custody: s.tip, objects: s.objectsRoot, enrollment: admissionTestDigest("a")},
		{setup: s.setupTip, custody: s.tip, objects: s.objectsRoot, checkpoint: admissionTestDigest("a")},
		{setup: s.setupTip, custody: s.tip, objects: s.objectsRoot, admission: admissionTestDigest("a")},
	} {
		if _, err := x.target(target); err == nil {
			t.Fatal("target names a missing future root")
		}
	}
}
