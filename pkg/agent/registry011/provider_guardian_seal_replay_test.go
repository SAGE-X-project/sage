package registry011

import "testing"

func TestProviderGuardianSealReplayRefusesMissingBoundaryData(t *testing.T) {
	t.Parallel()
	s, g := providerPageSealReadyFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := newProviderPageSealCandidateLocked(s)
	if err != nil {
		t.Fatal("normal page preparation", err)
	}
	defer o.failLocked()
	c, body := g.integration, s.frontierReplay.body
	before, stored, objects := s.frontierFiles.pin, s.stored, len(s.objects)
	if _, ok := providerGuardianSealBoundaryData(c.ledger, g.indexes, o.before); ok {
		t.Fatal("pending page was reused")
	}
	pending := c.ledger.continuation.pending
	c.ledger.continuation.pending = nil
	original := c.ledger.current.totals.bootstrap.outstanding[0]
	c.ledger.current.totals.bootstrap.outstanding[0] = 1
	_, ok := providerGuardianSealBoundaryData(c.ledger, g.indexes, o.before)
	c.ledger.current.totals.bootstrap.outstanding[0] = original
	c.ledger.continuation.pending = pending
	if ok {
		t.Fatal("outstanding work allowed a boundary")
	}
	references := s.newReferences
	defer func() { s.newReferences = references }()
	for _, mode := range []string{"missing", "invalid-digest", "duplicate", "absent-object"} {
		t.Run(mode, func(t *testing.T) {
			switch mode {
			case "missing":
				s.newReferences = nil
			case "invalid-digest":
				s.newReferences = []providerIntegrationObjectRef{{kind: "guardian-seal-binding"}}
			case "duplicate":
				ref := providerIntegrationObjectRef{kind: "guardian-seal-binding", digest: o.change.fresh[2].digest}
				s.newReferences = []providerIntegrationObjectRef{ref, ref}
			case "absent-object":
				s.newReferences = []providerIntegrationObjectRef{{kind: "guardian-seal-binding", digest: admissionTestDigest("f")}}
			}
			// Finite in-memory references are not written as evidence or credentials.
			if _, err := body.guardianSealBefore(s); err == nil {
				t.Fatal("incomplete references accepted", mode)
			}
			s.newReferences = references
			if s.frontierFiles.pin != before || s.stored != stored || len(s.objects) != objects || g.meter.actual[0] != 0 || g.meter.actual[1] != 0 {
				t.Fatal("refused references changed custody or performed RPC work", mode)
			}
		})
	}
	// The original observer has only one work unit left in this finite unit
	// model. A reference comparison must refuse before any original read.
	g.meter.actual[10] = g.meter.upper[10] - 1
	if _, err := body.guardianSealBefore(s); err == nil || !g.meter.failed ||
		s.frontierFiles.pin != before || s.stored != stored || len(s.objects) != objects {
		t.Fatal("unfunded replay reference comparison returned a boundary")
	}
}
