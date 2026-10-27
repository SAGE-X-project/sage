package registry011

import (
	"os"
	"path/filepath"
	"testing"
)

// Only ordinary isolated test directories are provisioned. They never select
// the Agent's operating custodian or authenticate an absent production history.
func TestProviderCustodyOpenRefusesMissingOrMismatchedOriginalRoots(t *testing.T) {
	for _, mode := range []string{"custody", "evidence", "anchor", "anchor-shape", "clock"} {
		t.Run(mode, func(t *testing.T) {
			s, _, _, _ := providerStoreFixture(t)
			config := s.config
			anchor, err := s.read(s.custody, "ANCHOR", 64<<10)
			if err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			config.custodyDirectory, config.evidenceDirectory = filepath.Join(root, "custody"), filepath.Join(root, "evidence")
			if mode == "anchor-shape" {
				anchor = []byte(`{}`)
				config.anchor = providerCandidateHash("sage-provider-custody-anchor-v1", anchor)
				config.expectedTip, config.expectedSetupTip = config.anchor, config.anchor
			}
			for i, path := range []string{config.custodyDirectory, config.evidenceDirectory} {
				if mode == "custody" && i == 0 || mode == "evidence" && i == 1 {
					continue
				}
				if os.Mkdir(path, 0700) != nil || os.WriteFile(filepath.Join(path, "AUTHORITY.LOCK"), nil, 0600) != nil {
					t.Fatal("normal owned fixture provision")
				}
				if mode != "anchor" && os.WriteFile(filepath.Join(path, "ANCHOR"), anchor, 0600) != nil {
					t.Fatal("normal fixture anchor")
				}
			}
			clock, err := newProviderOwnedClock(config.policy, 64, 10, 10)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "clock" {
				clock.failed = true
			}
			if opened, err := openProviderCustody(config, clock); err == nil || opened != nil {
				t.Fatal("incomplete originals became a store", mode)
			}
			if !s.valid() || s.sequence != 0 {
				t.Fatal("separate refused opener changed the original fixture")
			}
		})
	}
}

func TestProviderCustodyDirectoryEnumerationAndCloseAreBounded(t *testing.T) {
	s, _, _, _ := providerStoreFixture(t)
	if _, err := providerDirectoryNames(s.custody, 0); err == nil {
		t.Fatal("unfunded directory enumeration")
	}
	if _, err := providerDirectoryNames(s.custody, 1); err == nil {
		t.Fatal("directory entry cap silently truncated the inventory")
	}
	if os.Mkdir(filepath.Join(s.config.custodyDirectory, "ordinary-subdirectory"), 0700) != nil {
		t.Fatal("normal owned subdirectory")
	}
	if _, err := providerDirectoryNames(s.custody, 64); err == nil {
		t.Fatal("a subdirectory became a regular retained original")
	}
	if err := s.close(); err != nil {
		t.Fatal(err)
	}
	if s.close() == nil {
		t.Fatal("closed store closed a second time")
	}
	var absent *providerCustodyStore
	if absent.close() == nil {
		t.Fatal("absent store closed successfully")
	}
}

func TestProviderIntegrationConstructorCannotStartWithMissingObservationOrSpace(t *testing.T) {
	for _, mode := range []string{"read-observation", "physical-cap"} {
		t.Run(mode, func(t *testing.T) {
			s, policy, candidate, primitive := providerStoreFixture(t)
			if mode == "read-observation" {
				meter := &providerWorkMeter{failed: true}
				s.activeMeter.Store(meter)
				defer s.activeMeter.Store(nil)
			} else {
				s.stored = s.config.bounds.maximumStoredBytes
			}
			if c, err := newProviderIntegration(s, policy, candidate, primitive); err == nil || c != nil {
				t.Fatal("unobserved or unfunded enrollment started")
			}
			if s.sequence != 0 || s.enrollment != "" || s.last != nil {
				t.Fatal("refused enrollment produced a retained root or ACK")
			}
		})
	}
}
