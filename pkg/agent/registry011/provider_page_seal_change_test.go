package registry011

import (
	"encoding/json"
	"testing"
)

func TestProviderPageSealProposalRefusesInconsistentData(t *testing.T) {
	t.Parallel()
	s, g := providerPageSealReadyFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := newProviderPageSealCandidateLocked(s)
	if err != nil {
		t.Fatal("normal page preparation", err)
	}
	defer o.failLocked()
	proposal, root := o.change, g.integration.root
	before, stored, objects := s.frontierFiles.pin, s.stored, len(s.objects)
	clock, err := g.integration.currentClock()
	if err != nil {
		t.Fatal("original clock", err)
	}
	// Only unretained proposal DATA is varied. The concrete role came from
	// the normal constructor; no changed proposal is retained or dispatched.
	o.change = nil
	defer func() { o.change, g.integration.root = proposal, root }()
	for _, mode := range []string{"page", "seal", "root-type", "root-format"} {
		t.Run(mode, func(t *testing.T) {
			page, seal, binding := proposal.fresh[0], proposal.fresh[1], proposal.fresh[2]
			copyRoot := *root
			g.integration.root = &copyRoot
			defer func() { g.integration.root = root }()
			switch mode {
			case "page":
				page = providerIntegrationObject{}
			case "seal":
				seal.raw = ""
			case "root-type":
				copyRoot.raw = "null"
			case "root-format":
				var value map[string]any
				if json.Unmarshal([]byte(root.raw), &value) != nil {
					t.Fatal("normal root DATA")
				}
				value["format"] = "unsupported-unit-format"
				copyRoot.raw = string(providerIntegrationJSON(value))
			}
			if value, err := providerPageSealChangeData(o, page, seal, binding, providerClockDiagnostic(clock)); err == nil || value != nil {
				t.Fatal("inconsistent DATA produced a usable proposal", mode)
			}
			if s.frontierFiles.pin != before || s.stored != stored || len(s.objects) != objects || g.meter.actual[0] != 0 || g.meter.actual[1] != 0 {
				t.Fatal("refused DATA changed custody or performed RPC work", mode)
			}
		})
	}
}
