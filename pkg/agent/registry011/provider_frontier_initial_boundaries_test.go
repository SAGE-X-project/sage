package registry011

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProviderFrontierInitialFilesRequireExactSelectedSetup(t *testing.T) {
	for _, mode := range []string{"attempt", "sequence", "tip", "setup-count", "setup-clock", "anchor-id", "objects-root", "headroom", "intent-tail", "head-tail"} {
		t.Run(mode, func(t *testing.T) {
			f, _ := providerFrontierFileFixture(t)
			s := f.store
			switch mode {
			case "attempt":
				s.config.attempt = 1
			case "sequence":
				s.sequence = 1
			case "tip":
				s.tip = admissionTestDigest("f")
			case "setup-count":
				s.setups = nil
			case "setup-clock":
				s.setupOrigin.MonotonicMS++
			case "anchor-id":
				s.config.identity.Namespace = admissionTestDigest("f")
			case "objects-root":
				s.objectsRoot = admissionTestDigest("f")
			case "headroom":
				s.config.bounds.setupRetained = 1
				s.stored = 1
			case "intent-tail", "head-tail":
				name := providerCustodyName("highwater", 0)
				if mode == "head-tail" {
					name = "HEAD"
				}
				if err := os.WriteFile(filepath.Join(s.config.custodyDirectory, name), []byte("finite ordinary unit tail"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			stored := s.stored
			if _, err := f.initialize(); err == nil || !f.failed || !s.failed || f.initialized || f.pinRaw != "" || s.stored != stored {
				t.Fatal("incomplete initial metadata installed a reusable highwater", mode)
			}
		})
	}
}

func TestProviderFrontierInitialReadersRefuseUnavailableOwnedDisks(t *testing.T) {
	for _, mode := range []string{"closed-lock", "anchor-read", "setup-read"} {
		t.Run(mode, func(t *testing.T) {
			f, _ := providerFrontierFileFixture(t)
			s := f.store
			switch mode {
			case "closed-lock":
				if err := s.custody.lock.Close(); err != nil {
					t.Fatal(err)
				}
			case "anchor-read":
				s.config.bounds.maximumMetadataPerCommit = 1
			case "setup-read":
				// The anchor remains readable; the larger setup original does not.
				raw, err := os.ReadFile(filepath.Join(s.config.custodyDirectory, "ANCHOR"))
				if err != nil {
					t.Fatal(err)
				}
				s.config.bounds.maximumMetadataPerCommit = uint64(len(raw))
			}
			if err := f.checkInitialPrefix(); err == nil {
				t.Fatal("unavailable bounded initial read was authenticated absence", mode)
			}
		})
	}
}
