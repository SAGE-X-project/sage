package registry011

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestProviderScanOpenKeepsExactSetupAndMainHighwaterBytes(t *testing.T) {
	s := providerScanStoreFixture(t)
	x, err := newProviderScanCodec(s.config, nil)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := x.setup(providerClockDiagnostic(s.setupOrigin))
	commit, commitErr := x.setupCommit(claim)
	if err != nil || commitErr != nil || !x.equal(claim, providerCustodySetup(s.config, s.setupOrigin)) || !x.equal(commit, providerSetupCommit(claim)) {
		t.Fatal("setup byte contract changed", err, commitErr)
	}
	head, err := s.frontierFiles.pin.bytes(s.config)
	if err != nil || !x.equal(head, []byte(s.frontierFiles.pinRaw)) {
		t.Fatal("main highwater byte contract changed", err)
	}
	if len(s.setups) != 1 || s.setups[0].commit != s.setupTip || s.setupCommits[s.setupTip] != s.setups[0].claim || s.retainedRaw == 0 || s.retainedRaw >= s.stored {
		t.Fatal("fresh retained accounting")
	}
	if _, err := x.setupCommit([]byte(`[]`)); err == nil {
		t.Fatal("invalid finite setup input")
	}
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	if next, err := openProviderFrontierCandidate(s.config, s.clock); err == nil || next != nil {
		t.Fatal("normal closed prefix restored a fresh initial owner")
	}
}

func TestProviderScanOpenClosesOriginalRootsAfterSetupWriteFailure(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires an unprivileged owner with POSIX directory permissions")
	}
	for _, mode := range []string{"writable", "write-denied"} {
		t.Run(mode, func(t *testing.T) {
			config, clock := providerScanProvisionedFixture(t)
			originals := providerHistoryProvisionedOriginalBytes(t, config)
			if mode == "write-denied" {
				info, err := os.Stat(config.custodyDirectory)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := os.Chmod(config.custodyDirectory, info.Mode().Perm()); err != nil {
						t.Error(err)
					}
				})
				if err := os.Chmod(config.custodyDirectory, 0500); err != nil {
					t.Fatal(err)
				}
			}
			store, err := openProviderScanCandidate(config, clock)
			if mode == "writable" {
				if err != nil || store == nil {
					t.Fatal("ordinary writable original roots did not finish setup", err)
				}
				if err := store.close(); err != nil {
					t.Fatal(err)
				}
				return
			}
			if err != errProviderIntegration || store != nil || clock.read != 1 {
				t.Fatal("unwritable setup returned a store or reset the original clock", err)
			}
			after := providerHistoryProvisionedOriginalBytes(t, config)
			if len(after) != len(originals) {
				t.Fatal("failed first setup write left new original files")
			}
			for name, want := range originals {
				if after[name] != want {
					t.Fatal("failed setup replaced an original", name)
				}
			}
			for _, directory := range []string{config.custodyDirectory, config.evidenceDirectory} {
				disk, err := openRetentionDisk(directory)
				if err != nil {
					t.Fatal("failed setup retained an original root lock", err)
				}
				if err := disk.close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestProviderScanOpenRejectsInvalidSelectionBeforeRootUse(t *testing.T) {
	s := providerScanStoreFixture(t)
	for _, mode := range []string{"nil-scan", "nil-clock", "clock-policy", "attempt", "tip", "setup", "anchor", "scan-head", "custody", "evidence", "setup-upper", "profile", "generation", "policy"} {
		t.Run(mode, func(t *testing.T) {
			config, clock := s.config, s.clock
			switch mode {
			case "nil-scan":
				config.bounds.scan = nil
			case "nil-clock":
				clock = nil
			case "clock-policy":
				config.identity.Policy = admissionTestDigest("f")
			case "attempt":
				config.attempt = 1
			case "tip":
				config.expectedTip = admissionTestDigest("f")
			case "setup":
				config.expectedSetupTip = admissionTestDigest("f")
			case "anchor":
				config.anchor = "invalid"
			case "scan-head":
				config.expectedScanHead = "invalid"
			case "custody":
				config.custodyDirectory = ""
			case "evidence":
				config.evidenceDirectory = ""
			case "setup-upper":
				config.bounds.setupWork[2] = 1
			case "profile":
				config.profile = admissionTestDigest("f")
			case "generation":
				config.identity.Generation = 1
			case "policy":
				config.policy = nil
			}
			if next, err := openProviderScanCandidate(config, clock); err == nil || next != nil {
				t.Fatal("invalid initial selection opened the retained roots")
			}
		})
	}
}

func TestProviderScanDomainHashIncludesItsZeroSeparator(t *testing.T) {
	s := providerScanStoreFixture(t)
	m := &providerWorkMeter{scan: s.scanControl, upper: s.config.bounds.scan.maximumPreIWork}
	m.owner = m
	x, err := newProviderScanCodec(s.config, m)
	if err != nil {
		t.Fatal(err)
	}
	before := m.actual[10]
	if got := x.hash("abc", []byte("def")); got != providerCandidateHash("abc", []byte("def")) || m.actual[10]-before != 8 {
		t.Fatal("hash metric lost domain NUL or invocation")
	}
}

func TestProviderScanOpenRefusesMismatchedOriginalControlWithoutRepair(t *testing.T) {
	first := providerFirstAdmissionBoundsFixture()
	config, clock := providerScanProvisionedFixtureWithFirst(t, &first)
	name := filepath.Join(config.evidenceDirectory, "SCAN-HEAD")
	// Inert inconsistent DATA in a fresh owned fixture, never an operating root.
	original := []byte(`{"format":"incomplete-control-data"}`)
	if err := os.WriteFile(name, original, 0600); err != nil {
		t.Fatal(err)
	}
	if s, err := openProviderScanCandidate(config, clock); err == nil || s != nil {
		t.Fatal("mismatched original selected a fresh bootstrap owner")
	}
	got, err := os.ReadFile(name)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatal("refusal replaced or repaired a prior control original", err)
	}
	for _, path := range []string{filepath.Join(config.custodyDirectory, providerCustodyName("setup-intent", 0)), filepath.Join(config.evidenceDirectory, providerCustodyName("setup-claim", 0))} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("initial refusal created a new setup prefix", err)
		}
	}
}

func TestProviderScanOpenRejectsADifferentWellFormedExpectedHeadBeforeSetup(t *testing.T) {
	config, clock := providerScanProvisionedFixture(t)
	config.expectedScanHead = admissionTestDigest("f")
	if s, err := openProviderScanCandidate(config, clock); err != errProviderIntegration || s != nil {
		t.Fatal("initial original head did not fail at retained control validation", err)
	}
	for _, path := range []string{filepath.Join(config.custodyDirectory, providerCustodyName("setup-intent", 0)), filepath.Join(config.evidenceDirectory, providerCustodyName("setup-claim", 0))} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("wrong selected head created a new setup prefix", err)
		}
	}
}

func TestProviderScanOpenRejectsANoncanonicalPolicyOriginalBeforeSetup(t *testing.T) {
	config, clock := providerScanProvisionedFixture(t)
	policy := *config.policy
	// Fixed invalid configuration DATA only. Keep the closed profile and
	// identity fields so the independent policy-original check is exercised.
	policy.raw += "\n"
	config.policy = &policy
	if s, err := openProviderScanCandidate(config, clock); err != errProviderIntegration || s != nil {
		t.Fatal("inconsistent policy original survived independent parsing", err)
	}
	for _, path := range []string{filepath.Join(config.custodyDirectory, providerCustodyName("setup-intent", 0)), filepath.Join(config.evidenceDirectory, providerCustodyName("setup-claim", 0))} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("invalid policy created a setup prefix", err)
		}
	}
}
