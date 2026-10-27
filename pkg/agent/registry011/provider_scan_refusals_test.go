package registry011

import (
	"os"
	"path/filepath"
	"testing"
)

// These borrowed readers are DATA-only, not selected actors. No ledger,
// coordinator maps, ACK, transport or execution permit is copied/constructed.
func providerScanBorrowedControl(s *providerCustodyStore) *providerScanControl {
	view := &providerCustodyStore{config: s.config, custody: s.custody, evidence: s.evidence, clock: s.clock, setupOrigin: s.setupOrigin, tip: s.tip, setupTip: s.setupTip, stored: s.stored, retainedRaw: s.retainedRaw, objects: make(map[string]providerIntegrationObjectRef), operations: make(map[string]bool)}
	return &providerScanControl{store: view, pin: s.scanControl.pin, held: s.scanControl.held}
}

func TestProviderScanOriginalReadersRefuseFiniteWrongContext(t *testing.T) {
	s := providerScanStoreFixture(t)
	for _, mode := range []string{"profile", "invalid-head", "missing-next", "clock"} {
		t.Run(mode, func(t *testing.T) {
			c := providerScanBorrowedControl(s)
			switch mode {
			case "profile":
				c.store.config.profile = admissionTestDigest("f")
			case "invalid-head":
				c.pin.ordinal = 1
			case "missing-next":
				c.pin.ordinal = 1
			case "clock":
				a := providerScanAttemptFixture(t, s, providerScanHighwater{})
				if err := s.scanControl.claim(a); err != nil {
					t.Fatal(err)
				}
				c.pin = s.scanControl.pin
				c.store.setupOrigin.WallMS++
			}
			if c.readPrefix() {
				t.Fatal("wrong original control prefix")
			}
			if mode != "clock" && c.readHighwater(c.pin) {
				t.Fatal("wrong original highwater")
			}
		})
	}
	if _, err := newProviderScanControl(nil); err == nil {
		t.Fatal("nil owner")
	}
	if _, err := newProviderScanControl(s); err == nil {
		t.Fatal("second live control owner")
	}
	if _, err := providerScanInitialFiles(nil); err == nil {
		t.Fatal("nil initial originals")
	}
	for _, mode := range []string{"profile", "head"} {
		c := providerScanBorrowedControl(s)
		if mode == "profile" {
			c.store.config.profile = admissionTestDigest("f")
		} else {
			c.store.config.expectedScanHead = admissionTestDigest("f")
		}
		if _, err := providerScanInitialFiles(c.store); err == nil {
			t.Fatal("wrong provisioned initial control", mode)
		}
		if c.checkMainPin() {
			t.Fatal("borrowed data became a current-tip principal")
		}
	}
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	c := providerScanBorrowedControl(s)
	if c.readPrefix() || c.readHighwater(c.pin) {
		t.Fatal("normal closed files retained read authority")
	}
	if _, err := providerScanInitialFiles(c.store); err == nil {
		t.Fatal("closed initial originals")
	}
}

func TestProviderScanPureSchemasRefuseMissingConfigAndBadInitialRecords(t *testing.T) {
	config := providerCustodyConfig{}
	h, a := providerScanHighwater{}, providerScanAttempt{}
	if _, err := h.bytes(config); err == nil {
		t.Fatal("zero config highwater")
	}
	if _, err := h.head(config); err == nil {
		t.Fatal("zero config head")
	}
	if _, err := a.bytes(config); err == nil {
		t.Fatal("zero config attempt")
	}
	if _, err := parseProviderScanHighwater([]byte(`{}`), config); err == nil {
		t.Fatal("zero config parser")
	}
	if _, err := parseProviderScanAttempt([]byte(`{}`), config); err == nil {
		t.Fatal("zero config parser")
	}
	if _, _, err := providerScanSuccessor(config, h, a); err == nil {
		t.Fatal("zero config successor")
	}
	if _, _, ok := providerScanDeadline(config, CandidateClockDiagnostic{}); ok {
		t.Fatal("zero config deadline")
	}
	s := providerScanStoreFixture(t)
	for _, mode := range []string{"initial-link", "ordinal", "missing-link"} {
		x := h
		if mode == "initial-link" {
			x.previous = admissionTestDigest("f")
		} else if mode == "ordinal" {
			x.ordinal = s.config.bounds.scan.maximumAttempts + 1
		} else {
			x.ordinal = 1
		}
		if _, err := x.bytes(s.config); err == nil {
			t.Fatal("bad initial/next highwater", mode)
		}
	}
}

func TestProviderScanPreparatoryRefusalsKeepTheClaimAndReturnNoCredential(t *testing.T) {
	var missing *providerScanControl
	if missing.claim(providerScanAttempt{}) == nil {
		t.Fatal("nil actor")
	}
	for _, mode := range []string{"wrong-prior", "clock-exhausted", "intent-tail", "highwater-tail", "pending-tail"} {
		t.Run(mode, func(t *testing.T) {
			s := providerScanStoreFixture(t)
			a := providerScanAttemptFixture(t, s, providerScanHighwater{})
			before := s.stored
			if mode == "wrong-prior" {
				a.previousHighwater = admissionTestDigest("f")
			}
			if mode == "clock-exhausted" {
				s.clock.mu.Lock()
				s.clock.read = s.clock.maximum
				s.clock.mu.Unlock()
			}
			// Finite ordinary crash-tail fixtures made from the same normal
			// canonical records; no remote peer or attack-producing runner.
			if mode == "intent-tail" || mode == "highwater-tail" || mode == "pending-tail" {
				after, _, err := providerScanSuccessor(s.config, providerScanHighwater{}, a)
				if err != nil {
					t.Fatal(err)
				}
				var name string
				var raw []byte
				switch mode {
				case "intent-tail":
					name = providerCustodyName("scan-intent", 1)
					raw, err = a.bytes(s.config)
				case "highwater-tail":
					name = providerCustodyName("scan-highwater", 1)
					raw, err = after.bytes(s.config)
				case "pending-tail":
					name = providerCustodyName("SCAN-HEAD.pending", 1)
					raw, err = after.head(s.config)
				}
				if err != nil || s.custody.write(name, raw) != nil {
					t.Fatal("ordinary finite preexisting tail", err)
				}
			}
			held, _ := providerScanUpperSum(a.preparation, a.body)
			if s.scanControl.claim(a) == nil || !s.failed || !s.scanControl.failed || s.scanControl.held != held || s.stored != before+held.retained || s.scanControl.prepared != nil || s.last != nil {
				t.Fatal("failed claim refunded or minted metadata success")
			}
		})
	}
}

func TestProviderScanObservedHelpersRefuseUnfundedWorkBeforeOriginalUse(t *testing.T) {
	s := providerScanStoreFixture(t)
	for _, mode := range []string{"replay", "indexes", "directory"} {
		m := &providerWorkMeter{scan: s.scanControl}
		m.owner = m
		switch mode {
		case "replay":
			if _, err := newProviderFrontierReplayObserved(s.config, m); err == nil {
				t.Fatal("unfunded replay")
			}
		case "indexes":
			if _, _, err := newProviderLedgerIndexesObserved(s.config.identity, 1, *s.config.bounds.continuation, m); err == nil {
				t.Fatal("unfunded indexes")
			}
		case "directory":
			if _, err := providerDirectoryNamesObserved(s.custody, 32, m); err == nil {
				t.Fatal("unfunded directory read")
			}
		}
		if !m.failed || m.actual != (candidateWorkVector{}) {
			t.Fatal("unfunded primitive worked first")
		}
	}
	config := s.config
	config.profile = admissionTestDigest("f")
	if _, err := newProviderFrontierReplay(config); err == nil {
		t.Fatal("wrong replay profile")
	}
	x, _ := newProviderScanCodec(s.config, nil)
	if _, _, ok := x.deadline(CandidateClockDiagnostic{WallMS: 0, UncertaintyMS: 1}); ok {
		t.Fatal("underflowed deadline")
	}
	m := &providerWorkMeter{scan: s.scanControl}
	m.owner = m
	x = &providerScanCodec{config: s.config, meter: m}
	if x.hash("a", []byte("b")) != "" {
		t.Fatal("unfunded hash")
	}
	if _, _, ok := x.deadline(providerClockDiagnostic(s.setupOrigin)); ok {
		t.Fatal("unfunded policy codec")
	}
	if _, err := x.setup(providerClockDiagnostic(s.setupOrigin)); err == nil {
		t.Fatal("unfunded setup hash")
	}
	x, _ = newProviderScanCodec(s.config, nil)
	if _, err := x.encode(make(chan int)); err == nil {
		t.Fatal("unsupported bounded codec value")
	}
	claim, err := x.setup(providerClockDiagnostic(s.setupOrigin))
	if err != nil {
		t.Fatal(err)
	}
	decode, _ := providerEventVector(providerCodecEvent, uint64(len(claim)))
	m = &providerWorkMeter{scan: s.scanControl, upper: decode}
	m.owner = m
	x = &providerScanCodec{config: s.config, meter: m}
	if _, err := x.setupCommit(claim); err == nil || !m.failed {
		t.Fatal("unfunded hash after a complete setup decode")
	}
}

func TestProviderScanFreshOpenRefusesMissingRootsOrLostClockWithoutReset(t *testing.T) {
	for _, mode := range []string{"missing-custody", "missing-evidence", "same-root", "empty-initial-root", "anchor-selection", "head-selection", "tiny-clock", "failed-clock"} {
		t.Run(mode, func(t *testing.T) {
			config, clock := providerScanProvisionedFixture(t)
			switch mode {
			case "missing-custody":
				config.custodyDirectory = filepath.Join(t.TempDir(), "not-provisioned")
			case "missing-evidence":
				config.evidenceDirectory = filepath.Join(t.TempDir(), "not-provisioned")
			case "same-root":
				config.evidenceDirectory = config.custodyDirectory
			case "empty-initial-root":
				config.custodyDirectory = t.TempDir()
				if os.WriteFile(filepath.Join(config.custodyDirectory, "AUTHORITY.LOCK"), nil, 0600) != nil {
					t.Fatal("ordinary empty initial root")
				}
			case "anchor-selection":
				config.anchor = admissionTestDigest("f")
				config.expectedTip = config.anchor
				config.expectedSetupTip = config.anchor
			case "head-selection":
				config.expectedScanHead = admissionTestDigest("f")
			case "tiny-clock":
				var err error
				clock, err = newProviderOwnedClock(config.policy, 1, 100, 100)
				if err != nil {
					t.Fatal(err)
				}
			case "failed-clock":
				clock.mu.Lock()
				clock.failed = true
				clock.mu.Unlock()
			}
			if next, err := openProviderFrontierCandidate(config, clock); err == nil || next != nil {
				t.Fatal("unqualified fresh environment silently adopted")
			}
			// No fixture cleanup removes or repairs any retained originals.
			if mode == "tiny-clock" {
				if _, err := os.Stat(filepath.Join(config.custodyDirectory, "HEAD")); err != nil {
					t.Fatal("late clock failure erased ordinary originals")
				}
			}
		})
	}
}
