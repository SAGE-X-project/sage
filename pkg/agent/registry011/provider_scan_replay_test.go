package registry011

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// One ordinary original enrollment is retained through the existing candidate
// writer. The future reader never receives these coordinator maps or inputs.
func providerScanEnrollmentOriginals(t *testing.T) *providerCustodyStore {
	t.Helper()
	s := providerScanStoreFixture(t)
	context := &ProviderCandidateContext{s.config.identity, true}
	m := admissionTestRootObject(t)
	m["policy"] = s.config.identity.Policy
	root, err := ParseCandidateEnrollmentRoot(context, providerPolicyTestBytes(t, m), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	operation := providerCandidateHash("sage-provider-enroll-operation-v1", []byte(s.config.anchor+root.identity.Instance+root.identity.Policy+root.identity.SourceID))
	indexes, anchors, err := newProviderLedgerIndexes(s.config.identity, 1, *s.config.bounds.continuation)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := newProviderIntegrationObject("enrollment", root.CanonicalBytes(), 64<<10)
	if err != nil {
		t.Fatal(err)
	}
	delta, err := indexes.prepare("enroll", operation, 1, entry, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	fresh := []providerIntegrationObject{entry}
	fresh = append(fresh, anchors...)
	for _, value := range []struct {
		kind, name string
		disk       *retentionDisk
	}{{"setup", providerCustodyName("setup-intent", 0), s.custody}, {"setup-commit", providerCustodyName("setup-commit", 0), s.custody}} {
		raw, err := s.read(value.disk, value.name, 64<<10)
		object, objectErr := newProviderIntegrationObject(value.kind, raw, 64<<10)
		if err != nil || objectErr != nil {
			t.Fatal(err, objectErr)
		}
		fresh = append(fresh, object)
	}
	heads := delta.previous
	for _, mutation := range delta.mutations {
		fresh = append(fresh, mutation.object)
		heads[mutation.chain] = mutation.object.digest
	}
	inventory, err := newProviderFrontierInventory(s.objectsRoot, fresh, s.config.bounds.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	link := providerFrontierLink{identity: s.config.identity, fence: 1, commit: s.tip, setup: s.setupTip, objects: s.objectsRoot}
	intent, err := providerFrontierIntent(link, root, inventory, "enroll", operation, s.setupOrigin, s.config.bounds)
	if err != nil {
		t.Fatal(err)
	}
	commit := providerFrontierCommit(intent)
	deadline, _, ok := providerScanDeadline(s.config, providerClockDiagnostic(s.setupOrigin))
	if !ok {
		t.Fatal("original deadline")
	}
	f := providerFrontierRecord{identity: s.config.identity, fence: 1, sequence: 1, commit: providerCandidateHash("sage-provider-custody-commit-v2", commit), setup: s.setupTip, enrollment: root.digest, objects: inventory.digest, indexes: heads, initial: providerClockDiagnostic(s.setupOrigin), clock: providerClockDiagnostic(s.setupOrigin), deadline: deadline}
	frontier, err := f.bytes()
	if err != nil {
		t.Fatal(err)
	}
	after := s.frontierFiles.pin
	after.ordinal, after.sequence, after.previous = 1, 1, providerCandidateHash("sage-provider-custody-highwater-v1", []byte(s.frontierFiles.pinRaw))
	after.commit, after.frontier = f.commit, providerCandidateHash("sage-provider-custody-frontier-v1", frontier)
	if _, err := s.frontierReplay.retain(s, inventory, fresh, intent, commit, frontier, after); err != nil {
		t.Fatal("ordinary retained enrollment", err)
	}
	return s
}

func providerScanReaderFixture(t *testing.T, enrolled bool) (*providerScanReplayReader, *providerCustodyStore, *providerWorkMeter) {
	t.Helper()
	s := providerScanStoreFixture(t)
	if enrolled {
		s = providerScanEnrollmentOriginals(t)
	}
	return providerScanReaderFromStoreFixture(t, s)
}

func providerScanReaderFromStoreFixture(t *testing.T, s *providerCustodyStore) (*providerScanReplayReader, *providerCustodyStore, *providerWorkMeter) {
	t.Helper()
	f := s.frontierReplay.current
	x, err := newProviderScanCodec(s.config, nil)
	if err != nil {
		t.Fatal(err)
	}
	target := providerScanTarget{setup: s.setupTip, custody: s.tip, frontier: s.frontierFiles.pin.frontier, objects: s.objectsRoot, enrollment: s.enrollment, admission: f.admission, budget: f.budget}
	raw, err := x.target(target)
	if err != nil {
		t.Fatal(err)
	}
	a := providerScanAttemptFixture(t, s, s.scanControl.pin)
	a.target = x.hash("sage-provider-scan-target-v1", raw)
	if err := s.scanControl.claim(a); err != nil {
		t.Fatal(err)
	}
	meter := &providerWorkMeter{scan: s.scanControl, upper: a.body.work, maximumClocks: a.body.clocks}
	meter.owner = meter
	s.activeMeter.Store(meter)
	t.Cleanup(func() { s.activeMeter.Store(nil) })
	i, err := newProviderScanReplayReader(s.scanControl, meter, raw)
	if err != nil {
		t.Fatal(err)
	}
	return i, s, meter
}

func TestProviderScanReaderReconstructsOriginalEnrollmentWithoutProducerMaps(t *testing.T) {
	i, s, meter := providerScanReaderFixture(t, true)
	before := s.frontierReplay.current
	held, stored := s.scanControl.held, s.stored
	if i.replay == s.frontierReplay || i.view == s || len(i.view.objects) != 0 || len(i.view.operations) != 0 || len(i.view.setups) != 0 || i.lineage() {
		t.Fatal("reader adopted a producer snapshot")
	}
	// Change only provisional in-memory dictionaries; originals are unchanged.
	s.objects = map[string]providerIntegrationObjectRef{"ordinary-unused": {}}
	s.operations = map[string]bool{"ordinary-unused": true}
	s.setupCommits = map[string]string{"ordinary-unused": "ordinary-unused"}
	s.setups = nil
	if err := i.setup(); err != nil || i.lineage() {
		t.Fatal("structural setup incorrectly completed the main lineage", err)
	}
	if err := i.main(); err != nil || !i.lineage() || i.replay.current != before || i.view.objectsRoot != before.objects || len(i.view.objects) < 3 || i.view.operations["ordinary-unused"] || i.view.setupCommits["ordinary-unused"] != "" || meter.actual[2] == 0 || meter.failed || s.last != nil || s.stored != stored || s.scanControl.held != held || s.frontierReplay.current != before {
		t.Fatal("original typed replay copied maps, lost work or granted authority", err)
	}
	if err := i.setup(); err == nil {
		t.Fatal("setup cursor reset")
	}
	if err := i.main(); err == nil {
		t.Fatal("main cursor reset")
	}
	if i.UnmarshalJSON([]byte(`{}`)) == nil {
		t.Fatal("JSON restored reader ownership")
	}
}

func TestProviderScanReaderEmptyLineageStillRequiresOriginalSetup(t *testing.T) {
	i, s, _ := providerScanReaderFixture(t, false)
	if i.lineage() {
		t.Fatal("unread setup treated as closed")
	}
	if err := i.setup(); err != nil || !i.lineage() || i.view.setupTip != s.setupTip || i.replay.current.sequence != 0 {
		t.Fatal("ordinary empty structural lineage", err)
	}
	if i.main() == nil {
		t.Fatal("empty main invented a record")
	}
	// Structural lineage says nothing about a catalog or retained scan result.
	if s.initialInspection != nil || s.last != nil {
		t.Fatal("reader created a completed scan or ACK")
	}
}

func TestProviderScanReaderRefusesFiniteMissingOrUnfundedOriginals(t *testing.T) {
	for _, mode := range []string{"inactive", "unfunded-setup", "closed-setup", "unfunded-main", "closed-main"} {
		t.Run(mode, func(t *testing.T) {
			i, s, meter := providerScanReaderFixture(t, true)
			if mode == "inactive" {
				s.activeMeter.Store(nil)
			}
			if mode == "unfunded-setup" {
				meter.upper = meter.actual
			}
			if mode == "closed-setup" {
				_ = s.custody.root.Close()
			}
			if mode == "inactive" || mode == "unfunded-setup" || mode == "closed-setup" {
				if i.setup() == nil || i.lineage() {
					t.Fatal("unavailable setup became closed")
				}
				return
			}
			if err := i.setup(); err != nil {
				t.Fatal(err)
			}
			if mode == "unfunded-main" {
				meter.upper = meter.actual
			} else {
				_ = s.custody.root.Close()
			}
			if i.main() == nil || !i.failed || i.lineage() || i.view.sequence != 0 || s.last != nil {
				t.Fatal("partial main replay completed or reset its cursor")
			}
		})
	}
}

func TestProviderReplayPolicyObservedCopyAndHashesPreservePublicBytes(t *testing.T) {
	_, policy := providerPrivateLedgerFixture(t, 8, 32)
	meter := providerReplayUnitMeter()
	raw := []byte(policy.raw)
	p, err := parseProviderPolicyCandidateObserved(raw, 64<<10, meter)
	if err != nil || *p != *policy || meter.actual[2] != uint64(len(raw)) || meter.actual[10] == 0 || !bytes.Equal(p.CanonicalBytes(), raw) {
		t.Fatal("observed policy changed DATA or omitted its actual operations", err)
	}
	for _, slot := range []int{2, 10} {
		m := providerReplayUnitMeter()
		m.upper[slot] = 0
		if p, err := parseProviderPolicyCandidateObserved(raw, 64<<10, m); err == nil || p != nil || !m.failed {
			t.Fatal("unfunded policy parse produced DATA", slot)
		}
	}
}

func TestProviderScanReaderRefusesUnselectedOrUnfundedConstruction(t *testing.T) {
	for _, mode := range []string{"meter", "pin", "target", "memory", "anchor"} {
		t.Run(mode, func(t *testing.T) {
			i, s, meter := providerScanReaderFixture(t, false)
			x, _ := newProviderScanCodec(s.config, nil)
			raw, _ := x.target(i.target)
			switch mode {
			case "meter":
				meter = nil
			case "pin":
				s.frontierFiles.initialized = false
			case "target":
				raw = nil // Finite invalid DATA, never a second claimed body.
			case "memory":
				bounds := *s.config.bounds.scan
				bounds.maximumCatalogNameBytes = 16 << 30
				s.config.bounds.scan = &bounds
				s.config.profile = providerIntegrationProfile(s.config.bounds, s.config.capture)
			case "anchor":
				if err := os.WriteFile(filepath.Join(s.config.evidenceDirectory, "ANCHOR"), []byte("ordinary invalid anchor DATA"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if got, err := newProviderScanReplayReader(s.scanControl, meter, raw); err == nil || got != nil || s.last != nil {
				t.Fatal("unselected input created a reader", mode)
			}
		})
	}
}

func TestProviderScanReaderOriginalInventoryAndObjectAbsenceRemainIncomplete(t *testing.T) {
	for _, mode := range []string{"memory", "inventory", "object", "setup-binding"} {
		t.Run(mode, func(t *testing.T) {
			i, s, _ := providerScanReaderFixture(t, true)
			if err := i.setup(); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "memory":
				bounds := *i.view.config.bounds.scan
				bounds.maximumResidentBytes = 1
				i.view.config.bounds.scan = &bounds
			case "inventory":
				if err := os.Remove(filepath.Join(s.config.evidenceDirectory, "inventory-"+i.target.objects[2:])); err != nil {
					t.Fatal(err)
				}
			case "object":
				for _, ref := range s.objects {
					if err := os.Remove(filepath.Join(s.config.evidenceDirectory, providerObjectName(ref.kind, ref.digest))); err != nil {
						t.Fatal(err)
					}
					break // One finite original-absence scenario.
				}
			case "setup-binding":
				i.view.setups[0].commit = admissionTestDigest("f")
			}
			if err := i.main(); err == nil || !i.failed || i.lineage() || i.view.sequence != 0 || s.last != nil {
				t.Fatal("incomplete original closure became a completed lineage", mode)
			}
		})
	}
}

func TestProviderScanReaderDeferredLineageCannotAcceptUnknownPosition(t *testing.T) {
	i, _, _ := providerScanReaderFixture(t, true)
	if i.setup() != nil || i.main() != nil || !i.lineage() {
		t.Fatal("ordinary lineage")
	}
	i.view.setups[0].expectedTip = admissionTestDigest("f")
	if i.lineage() {
		t.Fatal("an unresolved expected tip was accepted by presence alone")
	}
	if i.resident(^uint64(0), 0, 0) {
		t.Fatal("resident count wrapped")
	}
	var missing *providerScanReplayReader
	if missing.valid() || missing.setup() == nil || missing.main() == nil || missing.lineage() {
		t.Fatal("missing ownership recreated a scan")
	}
}

func TestProviderScanReaderConsumesAnOriginalSetupTailAtTheFinalMainPosition(t *testing.T) {
	s := providerScanEnrollmentOriginals(t)
	previous := s.frontierFiles.pin
	s.config.attempt, s.config.expectedTip, s.config.expectedSetupTip = 1, s.tip, s.setupTip
	clock, err := s.sample()
	if err != nil {
		t.Fatal(err)
	}
	// The normal candidate fixture retains its complete setup charge first.
	s.stored += s.config.bounds.setupRetained
	if err := s.writeSetup(clock, 0); err != nil {
		t.Fatal(err)
	}
	claim := providerCustodySetup(s.config, clock)
	commit := providerSetupCommit(claim)
	setup := providerSetupReference{commit: providerCandidateHash("sage-provider-authority-setup-commit-v1", commit), claim: providerCandidateHash("sage-provider-authority-setup-v1", claim), expectedTip: s.tip, ordinal: 1, clock: providerClockDiagnostic(clock)}
	after := previous
	after.ordinal++
	after.previous = providerCandidateHash("sage-provider-custody-highwater-v1", []byte(s.frontierFiles.pinRaw))
	after.setup = setup.commit
	if _, err := s.frontierFiles.advance(after, "setup", claim, commit, nil); err != nil {
		t.Fatal(err)
	}
	s.setups, s.setupTip = append(s.setups, setup), setup.commit
	s.setupCommits[setup.commit] = setup.claim
	i, _, _ := providerScanReaderFromStoreFixture(t, s)
	if i.setup() != nil || i.setup() != nil || i.main() != nil || !i.lineage() || i.replay.current.setup == i.target.setup || i.view.setupTip != i.target.setup {
		t.Fatal("final setup tail was skipped or retroactively applied to E0")
	}
	// Replacing only proposed target metadata cannot change the replayed I/B.
	i.target.admission = admissionTestDigest("f")
	if i.lineage() {
		t.Fatal("target metadata was accepted without matching typed replay")
	}
}

func TestProviderScanReaderProspectiveMapCapsRefuseBeforeOriginalReads(t *testing.T) {
	for _, phase := range []string{"setup", "main"} {
		t.Run(phase, func(t *testing.T) {
			i, _, meter := providerScanReaderFixture(t, true)
			if phase == "main" && i.setup() != nil {
				t.Fatal("ordinary structural setup")
			}
			// A finite DATA-only profile upper makes the prospective catalog
			// and reader maps exceed RAM. It is never an operating selection.
			bounds := *i.view.config.bounds.scan
			bounds.filenameEntryOverhead = 1 << 20
			i.view.config.bounds.scan = &bounds
			before, objects, setups, positions := meter.actual, len(i.view.objects), len(i.view.setups), len(i.positions)
			var err error
			if phase == "setup" {
				err = i.setup()
			} else {
				err = i.main()
			}
			if err == nil || !i.failed || meter.actual != before || len(i.view.objects) != objects || len(i.view.setups) != setups || len(i.positions) != positions {
				t.Fatal("RAM refusal followed original IO, decoding or map growth")
			}
		})
	}
}
