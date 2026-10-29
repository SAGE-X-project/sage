package registry011

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func providerScanBoundsFixture(store providerIntegrationBounds) providerScanBounds {
	work := candidateWorkVector{0, 0, 16 << 20, 0, 0, 0, 0, 0, 0, 0, 1 << 50}
	return providerScanBounds{maximumAttempts: 4, maximumRecords: uint64(store.maximumCommits + store.maximumSetupRecoveries + 1), maximumObjects: 1 << 20, maximumRawBytes: 1 << 30, maximumPages: store.continuation.maximumPages, maximumChunks: 1024, chunkRecords: 8, chunkObjects: 4096, chunkRawBytes: 8 << 20, maximumResidentBytes: store.continuation.maximumResidentBytes, filenameEntryOverhead: 256, maximumFilenameBytes: 255, maximumClockReads: 65536, chunkDurationMS: 300000, totalDurationMS: 300000, maximumCatalogEntries: 4096, maximumCatalogNameBytes: 1 << 20, maximumPreIRetainedBytes: store.maximumStoredBytes, maximumPreIClockReads: 1 << 20, chunkWork: work, maximumPreIWork: candidateWorkVector{0, 0, 1 << 34, 0, 0, 0, 0, 0, 0, store.maximumStoredBytes, 1 << 62}}
}

// A fresh isolated candidate is provisioned before its new original clock.
// It has no registered Agent, selected principal, dispatch permit or Source.
func providerScanProvisionedFixture(t *testing.T) (providerCustodyConfig, *providerOwnedClock) {
	return providerScanProvisionedFixtureWithFirst(t, nil)
}

func providerScanProvisionedFixtureWithFirst(t *testing.T, first *providerFirstAdmissionBounds) (providerCustodyConfig, *providerOwnedClock) {
	return providerScanProvisionedFixtureWithFirstPolicy(t, first, false)
}

func providerScanProvisionedFixtureWithFirstPolicy(t *testing.T, first *providerFirstAdmissionBounds, operation bool) (providerCustodyConfig, *providerOwnedClock) {
	return providerScanProvisionedFixtureWithCapture(t, first, operation, nil)
}

// An optional separate test profile is fixed before its first anchor/H0/clock.
type providerScanFixtureSelection struct {
	capture        *ProviderCaptureLimits
	tls            *providerTLSBounds
	continuingWire bool
	pin            bool
	pinResidentGiB bool
	pinLargeObject bool
	historyProgram bool
}

func providerScanProvisionedFixtureWithCapture(t *testing.T, first *providerFirstAdmissionBounds, operation bool, selected *ProviderCaptureLimits) (providerCustodyConfig, *providerOwnedClock) {
	return providerScanProvisionedFixtureWithSelection(t, first, operation, &providerScanFixtureSelection{capture: selected})
}

func providerScanProvisionedFixtureWithSelection(t *testing.T, first *providerFirstAdmissionBounds, operation bool, selected *providerScanFixtureSelection) (providerCustodyConfig, *providerOwnedClock) {
	t.Helper()
	model, policy := providerPrivateLedgerFixtureWithDeadline(t, 8, 32, 300000)
	b, capture := providerFrontierBoundsFixture()
	if selected != nil && selected.capture != nil {
		if !selected.capture.valid() || !b.methods.fits(*selected.capture) {
			t.Fatal("incomplete isolated capture profile")
		}
		capture = *selected.capture
	}
	if selected != nil && selected.tls != nil {
		if !selected.tls.valid() {
			t.Fatal("incomplete isolated TLS profile")
		}
		copied := *selected.tls
		b.tls = &copied
	}
	private := model.continuation.bounds
	if selected != nil && selected.pin {
		// Separate immutable test inputs chosen before every profile/anchor/H0.
		// Existing subjects and operating defaults remain unchanged.
		private.maximumResidentBytes = 512 << 20
		if selected.pinResidentGiB {
			private.maximumResidentBytes = 1 << 30
		}
	}
	b.continuation = &private
	b.maximumCommits, b.maximumObjects = int(private.maximumCommits), int(private.maximumObjects)
	b.ledger = CandidateLedgerBounds{int(private.pageRecords), min(private.pagePayloadBytes, 16<<20), min(private.pageMetadataBytes, 64<<20)}
	b.setupWork[2] = 16 << 20
	scan := providerScanBoundsFixture(b)
	scan.maximumChunks = scan.maximumRecords
	b.scan = &scan
	b.firstAdmission = first
	if first != nil {
		// Inert isolated candidate values, never an operating policy/profile.
		b.setupWork[2] = 64 << 20
		b.setupLive = b.setupRetained
		m := contentJSON(policy.CanonicalBytes()).(map[string]any)
		bootstrap := m["limits"].(map[string]any)["bootstrap"].(map[string]any)
		bootstrap["decoded_bytes"], bootstrap["disk_bytes_written"], bootstrap["work_units"] = strconv.FormatUint(1<<34, 10), strconv.FormatUint(1<<34, 10), strconv.FormatUint(1<<62, 10)
		if operation {
			// A different isolated policy is selected BEFORE profile/anchor/H0
			// or clock creation. No live namespace or existing cap is raised.
			// The selected header's X upper fits128KiB. A smaller fresh object
			// ceiling also keeps the complete E+R decoded quote below the
			// unchanged public operation policy maximum of256MiB.
			b.maximumObjectBytes = 128 << 10
			if selected != nil && selected.pin {
				b.maximumObjectBytes = 256 << 10
				if selected.pinLargeObject {
					b.maximumObjectBytes = 1 << 20
				}
			}
			for _, level := range []string{"operation", "segment"} {
				values := m["limits"].(map[string]any)[level].(map[string]any)
				values["decoded_bytes"], values["disk_bytes_written"], values["work_units"] = strconv.FormatUint(256<<20, 10), strconv.FormatUint(1<<34, 10), strconv.FormatUint(1<<62, 10)
			}
		}
		if operation && selected != nil && (selected.continuingWire || selected.pin) {
			// This fresh test input funds the header and three scalar reservations
			// before profile/anchor/H0/clock. Existing 65536-byte subjects stay intact.
			for _, level := range []string{"operation", "segment", "bootstrap"} {
				wire := "131072"
				if selected.pin {
					wire = "262144"
				}
				m["limits"].(map[string]any)[level].(map[string]any)["wire_bytes"] = wire
			}
		}
		var err error
		policy, err = ParseProviderPolicyCandidate(providerPolicyTestBytes(t, m), 64<<10)
		if err != nil {
			t.Fatal(err)
		}
	}
	if selected != nil && selected.historyProgram {
		providerHistoryProvisionedTestBounds(t, &b, capture)
	}
	profile := providerIntegrationProfile(b, capture)
	empty, err := newProviderFrontierInventory("", nil, b.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	id := model.intent.identity
	id.Policy = policy.policy
	anchor := providerIntegrationJSON(map[string]any{"format": "sage-provider-custody-anchor-v1", "namespace": id.Namespace, "instance_key": id.Instance, "profile": profile, "initial_objects_root": empty.digest})
	anchorHash := providerCandidateHash("sage-provider-custody-anchor-v1", anchor)
	root := t.TempDir()
	config := providerCustodyConfig{identity: id, policy: policy, bounds: b, capture: capture, profile: profile, anchor: anchorHash, expectedTip: anchorHash, expectedSetupTip: anchorHash, custodyDirectory: filepath.Join(root, "custody"), evidenceDirectory: filepath.Join(root, "evidence")}
	zero := providerScanHighwater{}
	h0, err := zero.bytes(config)
	head, headErr := zero.head(config)
	if err != nil || headErr != nil {
		t.Fatal("fresh profile", err, headErr)
	}
	config.expectedScanHead = providerCandidateHash(providerScanHeadDomain, head)
	for _, dir := range []string{config.custodyDirectory, config.evidenceDirectory} {
		if os.Mkdir(dir, 0700) != nil {
			t.Fatal("owned directory")
		}
		for name, raw := range map[string][]byte{"AUTHORITY.LOCK": nil, "ANCHOR": anchor, providerCustodyName("scan-highwater", 0): h0, "SCAN-HEAD": head} {
			if os.WriteFile(filepath.Join(dir, name), raw, 0600) != nil {
				t.Fatal("owned preprovisioned candidate originals")
			}
		}
	}
	maximumClocks := uint64(4096)
	if selected != nil && selected.historyProgram {
		maximumClocks = 1 << 20
	}
	clock, err := newProviderOwnedClock(policy, maximumClocks, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	return config, clock
}

func providerScanStoreFixture(t *testing.T) *providerCustodyStore {
	t.Helper()
	config, clock := providerScanProvisionedFixture(t)
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal("fresh candidate open", err)
	}
	t.Cleanup(func() { _ = s.close() })
	return s
}

func providerScanAttemptFixture(t *testing.T, s *providerCustodyStore, before providerScanHighwater) providerScanAttempt {
	t.Helper()
	raw, err := before.bytes(s.config)
	p, prepErr := providerScanPreparationUpper(s.config, before.ordinal+1)
	deadline, _, ok := providerScanDeadline(s.config, providerClockDiagnostic(s.setupOrigin))
	if err != nil || prepErr != nil || !ok {
		t.Fatal("closed normal attempt", err, prepErr)
	}
	body, bodyErr := providerScanBodyUpper(s.config)
	if bodyErr != nil {
		t.Fatal(bodyErr)
	}
	return providerScanAttempt{ordinal: before.ordinal + 1, previousCommit: before.commit, previousHighwater: providerCandidateHash(providerScanHighwaterDomain, raw), setup: s.setupTip, custody: s.tip, target: admissionTestDigest("a"), purpose: "initial-inspection", clock: providerClockDiagnostic(s.setupOrigin), deadline: deadline, preparation: p, body: body}
}

func TestProviderScanControlRetainsBothRootsBeforeMetadataSuccess(t *testing.T) {
	s := providerScanStoreFixture(t)
	c := s.scanControl
	beforeStored := s.stored
	a := providerScanAttemptFixture(t, s, c.pin)
	if err := c.claim(a); err != nil {
		t.Fatal(err)
	}
	held, ok := providerScanUpperSum(a.preparation, a.body)
	if !ok || c.held != held || c.pin.ordinal != 1 || c.prepared == nil || c.failed || s.failed || s.activeMeter.Load() != nil || s.stored != beforeStored+held.retained || s.last != nil || s.enrollment != "" || s.sequence != 0 || s.frontierReplay.body.intent != nil || !c.readPrefix() {
		t.Fatal("claim/readback/accounting created authority or lost originals")
	}
	for _, disk := range []*retentionDisk{s.custody, s.evidence} {
		raw, err := disk.read("SCAN-HEAD", candidateLedgerMetadataLimit)
		want, _ := c.pin.head(s.config)
		if err != nil || !bytes.Equal(raw, want) {
			t.Fatal("two-root HEAD mismatch", err)
		}
	}
	if err := c.claim(a); err == nil || s.stored != beforeStored+held.retained {
		t.Fatal("duplicate claim reset/reused a prepared attempt")
	}
}

func TestProviderScanControlDataReproducesEveryCumulativeHeldCoordinate(t *testing.T) {
	s := providerScanStoreFixture(t)
	h := providerScanHighwater{}
	for n := uint64(1); n <= s.config.bounds.scan.maximumAttempts; n++ {
		a := providerScanAttemptFixture(t, s, h)
		intent, err := a.bytes(s.config)
		parsed, parseErr := parseProviderScanAttempt(intent, s.config)
		after, commit, nextErr := providerScanSuccessor(s.config, h, a)
		held, ok := providerScanUpperSum(h.held, a.preparation, a.body)
		if err != nil || parseErr != nil || parsed != a || nextErr != nil || !ok || after.held != held || after.ordinal != n || after.intent != providerCandidateHash(providerScanIntentDomain, intent) || after.commit != providerCandidateHash(providerScanCommitDomain, commit) {
			t.Fatal("candidate exact disjoint equation", err, parseErr, nextErr)
		}
		raw, err := after.bytes(s.config)
		got, parseErr := parseProviderScanHighwater(raw, s.config)
		if err != nil || parseErr != nil || got != after {
			t.Fatal("closed highwater roundtrip", err, parseErr)
		}
		// Finite malformed record data only; it is never written or dispatched.
		for key := range contentJSON(intent).(map[string]any) {
			for _, mode := range []string{"missing", "invalid"} {
				m := contentJSON(intent).(map[string]any)
				if mode == "missing" {
					delete(m, key)
				} else {
					m[key] = "invalid"
				}
				if _, err := parseProviderScanAttempt(providerIntegrationJSON(m), s.config); err == nil {
					t.Fatalf("invalid attempt %s/%s", key, mode)
				}
			}
		}
		for key := range contentJSON(raw).(map[string]any) {
			m := contentJSON(raw).(map[string]any)
			delete(m, key)
			if _, err := parseProviderScanHighwater(providerIntegrationJSON(m), s.config); err == nil {
				t.Fatalf("missing highwater %s", key)
			}
		}
		h = after
	}
	if _, err := providerScanPreparationUpper(s.config, h.ordinal+1); err == nil {
		t.Fatal("attempts beyond selected ceiling")
	}
}

func TestProviderScanControlRejectsFiniteWrongLinksAndBudgets(t *testing.T) {
	s := providerScanStoreFixture(t)
	for _, mode := range []string{"ordinal", "previous", "commit", "target", "purpose", "clock", "deadline", "preparation", "body", "requests", "overflow", "global"} {
		t.Run(mode, func(t *testing.T) {
			a := providerScanAttemptFixture(t, s, providerScanHighwater{})
			config := s.config
			bounds := *config.bounds.scan
			config.bounds.scan = &bounds
			switch mode {
			case "ordinal":
				a.ordinal = 0
			case "previous":
				a.previousHighwater = admissionTestDigest("f")
			case "commit":
				a.previousCommit = admissionTestDigest("f")
			case "target":
				a.target = "bad"
			case "purpose":
				a.purpose = "dispatch"
			case "clock":
				a.clock.WallMS = -1
			case "deadline":
				a.deadline = 0
			case "preparation":
				a.preparation.retained = 0
			case "body":
				a.body.clocks = 0
			case "requests":
				a.body.work[0] = 1
			case "overflow":
				a.body.work[10] = ^uint64(0)
			case "global":
				bounds.maximumPreIRetainedBytes = a.preparation.retained + a.body.retained
			}
			if _, _, err := providerScanSuccessor(config, providerScanHighwater{}, a); err == nil {
				t.Fatal("bad candidate link/held upper")
			}
		})
	}
	for i := 0; i < 13; i++ {
		x := providerScanUpper{}
		if i < 11 {
			x.work[i] = ^uint64(0)
		} else if i == 11 {
			x.retained = ^uint64(0)
		} else {
			x.clocks = ^uint64(0)
		}
		one := x
		if i < 11 {
			one.work[i] = 1
		} else if i == 11 {
			one.retained = 1
		} else {
			one.clocks = 1
		}
		if _, ok := providerScanUpperSum(x, one); ok {
			t.Fatal("wrapped cumulative coordinate", i)
		}
	}
}

func TestProviderScanControlRefusesUnqualifiedReconstructionAndMeterUse(t *testing.T) {
	s := providerScanStoreFixture(t)
	c := s.scanControl
	if c.UnmarshalJSON([]byte(`{}`)) == nil {
		t.Fatal("restored owner from JSON")
	}
	m := &providerWorkMeter{scan: c, maximumClocks: 1, upper: candidateWorkVector{0, 0, 10, 0, 0, 0, 0, 0, 0, 10, 100}}
	m.owner = m
	if !m.event(providerClockEvent, 0) || m.event(providerClockEvent, 0) || !m.failed {
		t.Fatal("scan clock read ceiling")
	}
	// A scan meter can never be closed as an admission/settlement measurement.
	model, _ := providerPrivateLedgerFixture(t, 8, 32)
	if result, err := m.finish(nil, model.lastTicket, s.setupOrigin); err == nil || result != nil {
		t.Fatal("scan created an execution measurement")
	}
	before := s.stored
	a := providerScanAttemptFixture(t, s, c.pin)
	a.setup = admissionTestDigest("f")
	if c.claim(a) == nil || c.failed || s.stored != before {
		t.Fatal("different selected setup accepted")
	}
	config := s.config
	config.bounds.scan = nil
	if _, err := openProviderFrontierCandidate(config, s.clock); err == nil {
		t.Fatal("V3 files/profile adopted as V2")
	}
}
