package registry011

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProviderFrontierCandidateOpensOnlyTheFreshFiniteMetadataPrefix(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	config := u.s.config
	root := t.TempDir()
	config.custodyDirectory, config.evidenceDirectory = filepath.Join(root, "custody"), filepath.Join(root, "evidence")
	anchor, err := u.s.read(u.s.custody, "ANCHOR", int64(config.bounds.maximumMetadataPerCommit))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{config.custodyDirectory, config.evidenceDirectory} {
		if os.Mkdir(path, 0700) != nil || os.WriteFile(filepath.Join(path, "AUTHORITY.LOCK"), nil, 0600) != nil || os.WriteFile(filepath.Join(path, "ANCHOR"), anchor, 0600) != nil {
			t.Fatal("fresh owned normal fixture provisioning")
		}
	}
	clock, err := newProviderOwnedClock(config.policy, 256, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal(err)
	}
	f := s.frontierFiles
	if !s.valid() || f == nil || !f.initialized || f.failed || s.frontierReplay == nil || s.frontierReplay.body.anchor != nil || s.last != nil || s.enrollment != "" || s.sequence != 0 || f.pin.ordinal != 0 || f.pin.commit != config.anchor || f.pin.setup != s.setupTip || !f.readExact("HEAD", []byte(f.pinRaw)) || s.stored != config.bounds.setupRetained {
		t.Fatal("candidate open skipped H0 or invented future I/ACK/authority")
	}
	if len(s.setups) != 1 || s.setups[0].clock != providerClockDiagnostic(s.setupOrigin) || s.retainedRaw == 0 || s.retainedRaw >= s.stored {
		t.Fatal("initial original prefix accounting/clock")
	}
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	// Normal close/reopen cannot create another live initial owner from the
	// existing original prefix. Inspection is a separate metadata-only API.
	if next, err := openProviderFrontierCandidate(config, clock); err == nil || next != nil {
		t.Fatal("a completed prefix was silently enrolled again")
	}
}

func TestProviderFrontierCandidateRejectsConfigurationBeforeEnvironmentUse(t *testing.T) {
	u := providerFrontierUnitFixture(t)
	config := u.s.config
	clock, err := newProviderOwnedClock(config.policy, 64, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"nil-clock", "attempt", "tip", "setup", "anchor", "generation", "custody", "evidence"} {
		t.Run(name, func(t *testing.T) {
			c, selected := config, clock
			switch name {
			case "nil-clock":
				selected = nil
			case "attempt":
				c.attempt = 1
			case "tip":
				c.expectedTip = admissionTestDigest("f")
			case "setup":
				c.expectedSetupTip = admissionTestDigest("f")
			case "anchor":
				c.anchor = "invalid"
			case "generation":
				c.identity.Generation = 1
			case "custody":
				c.custodyDirectory = ""
			case "evidence":
				c.evidenceDirectory = ""
			}
			if s, err := openProviderFrontierCandidate(c, selected); err == nil || s != nil {
				t.Fatal("invalid configuration opened an environment")
			}
		})
	}
}
