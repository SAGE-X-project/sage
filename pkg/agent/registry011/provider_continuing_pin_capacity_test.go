package registry011

import "testing"

// Keep the original512MiB subject and its complete original child/page
// sequence. The refusal must precede a new T or physical protected-work hold.
func TestProviderContinuingPinRejectsInsufficientOriginalResidentCapacity(t *testing.T) {
	t.Parallel()
	s, g, digests := providerContinuingPinReadyWithSelectionFixture(t, &providerScanFixtureSelection{pin: true})
	proposal := providerContinuingPinQuoteFixture(t, g, admissionTestDigest("a"))
	before, sequence, tip, ordinal := s.stored, s.sequence, s.frontierFiles.pin, g.ordinal
	ticket := g.integration.ledger.lastTicket
	if g.config.bounds.continuation.maximumResidentBytes != 512<<20 || g.config.bounds.scan.maximumResidentBytes != 512<<20 || g.bounds.claim.resident != 512<<20 || proposal["resource_ok"] != true || proposal["memory_ok"] != false || proposal["global_storage_fits"] != true || proposal["policy_live_fits"] != true || proposal["policy_retained_fits"] != true || proposal["prospective_clock_fits"] != true || len(proposal["refused_coordinates"].([]int)) != 0 || proposal["memory_upper"].(uint64) <= 512<<20 {
		t.Fatalf("original subject does not isolate resident refusal: %+v", proposal)
	}
	o, err := newProviderContinuingPinCandidate(s, "example-agent", digests)
	if err == nil || o != nil || !s.failed || !g.failed || g.ordinal != ordinal+1 || g.integration.ledger.lastTicket != ticket || s.stored != before || s.sequence != sequence || s.frontierFiles.pin != tip || g.integration.pin != "" || s.activeMeter.Load() != nil {
		t.Fatal("insufficient original resident capacity admitted T or changed protected work")
	}
}

func TestProviderContinuingPinCapacityIsAnExplicitFreshProfile(t *testing.T) {
	t.Parallel()
	old, _ := providerGuardianConfigWithSelectionFixture(t, 64<<20, 6, &providerScanFixtureSelection{pin: true})
	next, _ := providerGuardianConfigWithSelectionFixture(t, 64<<20, 6, &providerScanFixtureSelection{pin: true, pinResidentGiB: true})
	if old.bounds.continuation.maximumResidentBytes != 512<<20 || old.bounds.scan.maximumResidentBytes != 512<<20 || old.bounds.guardian.claim.resident != 512<<20 || next.bounds.continuation.maximumResidentBytes != 1<<30 || next.bounds.scan.maximumResidentBytes != 1<<30 || next.bounds.guardian.claim.resident != 1<<30 || old.profile == next.profile || old.anchor == next.anchor || old.expectedScanHead == next.expectedScanHead {
		t.Fatal("different resident test inputs shared an original profile/anchor/H0")
	}
	if old.policy.policy != next.policy.policy || old.capture != next.capture || old.bounds.maximumObjectBytes != next.bounds.maximumObjectBytes || old.bounds.maximumStoredBytes != next.bounds.maximumStoredBytes || old.bounds.guardian.claim.work != next.bounds.guardian.claim.work || old.bounds.guardian.claim.clocks != next.bounds.guardian.claim.clocks || old.bounds.guardian.attempts != next.bounds.guardian.attempts {
		t.Fatal("fresh resident input changed another policy or work boundary")
	}
}
