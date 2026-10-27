package registry011

import "testing"

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
