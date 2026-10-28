package registry011

import "testing"

func TestProviderPageSealIndependentModelsRefuseInconsistentData(t *testing.T) {
	t.Parallel()
	s, g := providerPageSealReadyFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	o, err := newProviderPageSealCandidateLocked(s)
	if err != nil {
		t.Fatal("normal page preparation", err)
	}
	defer o.failLocked()
	cost, err := o.retainLocked()
	if err != nil || !providerPageSealModelsEqual(o, cost) {
		t.Fatal("normal independent retained models", err)
	}
	before, stored, objects := s.frontierFiles.pin, s.stored, len(s.objects)
	for _, mode := range []string{"index-count", "retained-budget", "cost", "seal-reference", "retained-page"} {
		t.Run(mode, func(t *testing.T) {
			// These are finite unit DATA comparisons after ordinary retention.
			// No original file, live role, proof or network peer is replaced.
			actual := cost
			switch mode {
			case "index-count":
				value := g.indexes.counts[1]
				g.indexes.counts[1]++
				defer func() { g.indexes.counts[1] = value }()
			case "retained-budget":
				model := s.frontierReplay.body.ledger.current
				value := model.privateMetadata
				model.privateMetadata++
				defer func() { model.privateMetadata = value }()
			case "cost":
				actual.metadata++
			case "seal-reference":
				value := o.binding.seal
				o.binding.seal = admissionTestDigest("f")
				defer func() { o.binding.seal = value }()
			case "retained-page":
				model := s.frontierReplay.body.ledger.continuation
				value := model.lastPage
				model.lastPage = admissionTestDigest("f")
				defer func() { model.lastPage = value }()
			}
			if providerPageSealModelsEqual(o, actual) {
				t.Fatal("independent model disagreement accepted", mode)
			}
		})
	}
	// The same original binding cannot be applied twice by its independent replay.
	if _, err := s.frontierReplay.body.guardianSealBefore(s); err == nil {
		t.Fatal("already retained binding was replayed twice")
	}
	if s.frontierFiles.pin != before || s.stored != stored || len(s.objects) != objects || g.meter.actual[0] != 0 || g.meter.actual[1] != 0 {
		t.Fatal("comparison DATA changed custody or performed RPC work")
	}
	// The one-use role is quarantined by cleanup, never released after these
	// deliberately inconsistent unit DATA comparisons.
}
