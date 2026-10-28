package registry011

import (
	"os"
	"path/filepath"
	"testing"
)

// A different, explicitly finite isolated profile is selected BEFORE its
// new anchor/H0/clock. These values are test inputs, never operating defaults
// or an increase to an existing namespace or public protocol boundary.
func providerGuardianConfigFixture(t *testing.T) (providerCustodyConfig, *providerOwnedClock) {
	t.Helper()
	return providerGuardianConfigWithClaimFixture(t, 16<<20)
}

// Explicit fresh test-profile input is fixed before anchor/H0/clock creation.
// It is never applied to an existing namespace or an operating profile.
func providerGuardianConfigWithClaimFixture(t *testing.T, decoded uint64) (providerCustodyConfig, *providerOwnedClock) {
	t.Helper()
	return providerGuardianConfigWithPageFixture(t, decoded, 6)
}

func providerGuardianConfigWithPageFixture(t *testing.T, decoded, records uint64) (providerCustodyConfig, *providerOwnedClock) {
	t.Helper()
	if decoded != 16<<20 && decoded != 50<<20 && decoded != 64<<20 || records != 6 && records != 9 {
		t.Fatal("unsupported isolated test profile")
	}
	first := providerFirstWriterBounds()
	config, _ := providerScanProvisionedFixtureWithFirstPolicy(t, &first, true)
	b := &config.bounds
	page := *b.continuation
	page.pageRecords, page.maximumPages, page.maximumObjects = records, 6, 512
	page.maximumCommits = page.maximumRecords + page.maximumPages + 2
	b.continuation, b.maximumObjects, b.maximumCommits = &page, int(page.maximumObjects), int(page.maximumCommits)
	scan := *b.scan
	scan.maximumPages = page.maximumPages
	scan.maximumRecords = page.maximumCommits + uint64(b.maximumSetupRecoveries) + 1
	scan.maximumChunks = scan.maximumRecords
	b.scan = &scan
	b.ledger.MaximumRecords = int(page.pageRecords)
	b.setupLive, b.setupRetained, b.setupWork[2] = 2<<20, 2<<20, 128<<20
	claim := providerGuardianCapacity{work: candidateWorkVector{0, 0, decoded, 0, 0, 0, 0, 0, 0, 512 << 10, 1 << 52}, retained: 512 << 10, live: 512 << 10, clocks: 64, duration: 300000, records: 6, objects: 4, raw: 512 << 10, metadata: 512 << 10, resident: 128 << 20}
	lifetime := claim
	const attempts uint64 = 32
	for i := range lifetime.work {
		lifetime.work[i] *= attempts
	}
	lifetime.retained *= attempts
	lifetime.live *= attempts
	lifetime.clocks *= attempts
	lifetime.records *= attempts
	lifetime.objects *= attempts
	lifetime.raw *= attempts
	lifetime.metadata *= attempts
	b.guardian = &providerGuardianBounds{lifetime: lifetime, claim: claim, attempts: attempts, privateRecords: 1024, objects: 1024, commits: 128, catalogEntries: 4096, catalogNameBytes: 1 << 20}
	if !b.valid() {
		t.Fatal("closed isolated V5 profile")
	}
	config.profile = providerIntegrationProfile(*b, config.capture)
	empty, err := newProviderFrontierInventory("", nil, b.maximumEntries, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	anchor := providerIntegrationJSON(map[string]any{"format": "sage-provider-custody-anchor-v1", "namespace": config.identity.Namespace, "instance_key": config.identity.Instance, "profile": config.profile, "initial_objects_root": empty.digest})
	config.anchor = providerCandidateHash("sage-provider-custody-anchor-v1", anchor)
	config.expectedTip, config.expectedSetupTip = config.anchor, config.anchor
	root := t.TempDir()
	config.custodyDirectory, config.evidenceDirectory = filepath.Join(root, "custody"), filepath.Join(root, "evidence")
	h0, err := (providerScanHighwater{}).bytes(config)
	head, headErr := (providerScanHighwater{}).head(config)
	if err != nil || headErr != nil {
		t.Fatal("fresh V5 control", err, headErr)
	}
	config.expectedScanHead = providerCandidateHash(providerScanHeadDomain, head)
	for _, dir := range []string{config.custodyDirectory, config.evidenceDirectory} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		for name, raw := range map[string][]byte{"AUTHORITY.LOCK": nil, "ANCHOR": anchor, providerCustodyName("scan-highwater", 0): h0, "SCAN-HEAD": head} {
			if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	clock, err := newProviderOwnedClock(config.policy, 4096, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	return config, clock
}

func providerGuardianStoreFixture(t *testing.T) *providerCustodyStore {
	t.Helper()
	config, clock := providerGuardianConfigFixture(t)
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal("fresh V5 open", err)
	}
	t.Cleanup(func() { _ = s.close() })
	if _, err := s.enrollBeforeScan(); err != nil {
		t.Fatal("V5 E0 and genesis", err)
	}
	return s
}

func providerGuardianPreparedFixture(t *testing.T) *providerCustodyStore {
	t.Helper()
	s := providerGuardianStoreFixture(t)
	x := mustProviderBindingCodec(t, s.config)
	target, err := x.target(providerScanTarget{setup: s.setupTip, custody: s.tip, frontier: s.frontierFiles.pin.frontier, objects: s.objectsRoot, enrollment: s.enrollment})
	if err != nil {
		t.Fatal(err)
	}
	attempt := providerScanAttemptFixture(t, s, s.scanControl.pin)
	attempt.target = x.hash("sage-provider-scan-target-v1", target)
	if err := s.scanControl.claim(attempt); err != nil {
		t.Fatal("V5 independent original scan hold", err)
	}
	if _, err := s.scanControl.inspectBeforeAdmission(target); err != nil {
		t.Fatal("V5 independent original scan", err)
	}
	return s
}

func providerGuardianFirstFixture(t *testing.T) (*providerCustodyStore, *providerFirstOperationOwner) {
	t.Helper()
	s := providerGuardianPreparedFixture(t)
	o, err := newProviderFirstOperationCandidate(s)
	if err != nil {
		t.Fatalf("V5 first open: %v; seq=%d F=%+v", err, s.sequence, s.firstAdmissionOwner)
	}
	return s, o
}
