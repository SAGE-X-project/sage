package registry011

import (
	"context"
	"testing"
)

func providerPageSealReadyFixture(t *testing.T) (*providerCustodyStore, *providerGuardianOwner) {
	t.Helper()
	s, o := providerContinuingPreparedFixture(t)
	if _, err := o.executeBootstrapCandidate(context.Background()); err != nil {
		t.Fatal("normal local bootstrap before page boundary", err)
	}
	if len(o.integration.ledger.continuation.records) != 6 || !o.guardian.ready || s.sequence != 7 {
		t.Fatal("ordinary T/C/S pairs did not fill their selected page")
	}
	return s, o.guardian
}

func TestProviderPageSealRetainsBothBoundariesWithoutExecutionOrRefund(t *testing.T) {
	t.Parallel()
	s, g := providerPageSealReadyFixture(t)
	c := g.integration
	before, origin, raw := s.stored, s.setupOrigin, s.retainedRaw
	counts, budget := c.ledger.current.counts, c.ledger.current.digest
	charged, ticket, consumed := c.ledger.current.totals.bootstrap.charged, c.ledger.lastTicket, c.ledger.lastConsumption
	oldG, oldMain, oldPrefix := g.pin, g.main, g.prefix
	result, err := sealProviderContinuingPageCandidate(s)
	if err != nil {
		t.Fatalf("metadata-only full page: %v; G=%+v", err, g)
	}
	page := c.ledger.continuation
	if result == nil || result.main != s.frontierFiles.pin || result.guardian != g.pin || result.enrollment != s.enrollment ||
		result.main == oldMain || result.guardian == oldG || g.ordinal != 2 || s.sequence != 8 ||
		s.stored != before || s.retainedRaw <= raw || s.setupOrigin != origin || s.last != nil || s.activeMeter.Load() != nil ||
		g.sealing != nil || g.continuing != nil || !g.ready || g.active || g.claim != nil || g.meter != nil ||
		c.ledger.current.counts != counts || c.ledger.current.digest != budget || c.ledger.current.totals.bootstrap.charged != charged ||
		c.ledger.lastTicket != ticket || c.ledger.lastConsumption != consumed || page.page != 1 || len(page.records) != 0 ||
		page.sealedRecords != 6 || page.lastPage != result.page || page.lastSeal != result.seal || page.lastBudget != budget ||
		g.prefix == oldPrefix || !g.continuingPrefixMatchesLocked() || result.physical == 0 {
		t.Fatal("page boundary minted work, reset a lifetime, refunded G, or omitted independent retention")
	}
	// The next ordinal is still part of the original lifetime. This is only
	// its consumed metadata role, not proof of the not-yet-connected RPC path.
	s.mu.Lock()
	if g.startClaimLocked("rpc-admission") != nil || g.ordinal != 3 || g.original != origin || g.main != result.main ||
		s.stored != before || g.claim.clock != providerClockDiagnostic(origin) {
		s.mu.Unlock()
		t.Fatal("post-seal claim restarted the original lifetime")
	}
	g.failLocked()
	s.mu.Unlock()
}

func TestProviderPageSealQuarantinesUnavailableMainOriginal(t *testing.T) {
	t.Parallel()
	s, g := providerPageSealReadyFixture(t)
	stored, origin := s.stored, s.setupOrigin
	if s.custody.root.Mkdir(s.name("intent", 8), 0700) != nil {
		t.Fatal("normal owned unavailable main path")
	}
	data, err := sealProviderContinuingPageCandidate(s)
	if err == nil || data != nil || !g.failed || !s.failed || g.ordinal != 2 || g.ready ||
		s.stored != stored || s.setupOrigin != origin || s.sequence != 7 || s.activeMeter.Load() != nil ||
		g.sealing == nil || !g.sealing.closed || g.claim == nil || g.claim.target == "" {
		t.Fatal("uncertain seal recovered a consumed claim or advanced its main")
	}
}

func TestProviderPageSealRefusesMissingCandidateData(t *testing.T) {
	t.Parallel()
	var o *providerPageSealOwner
	if o.selectedLocked() || o.writingLocked(nil) || o.consumeStorageLocked(0) || providerPageSealModelsEqual(nil, providerFrontierRetention{}) {
		t.Fatal("missing live role obtained a capability")
	}
	o.failLocked()
	if data, err := sealProviderContinuingPageCandidate(nil); err == nil || data != nil {
		t.Fatal("missing store sealed a page")
	}
	if data, err := providerPageSealChangeData(nil, providerIntegrationObject{}, providerIntegrationObject{}, providerIntegrationObject{}, CandidateClockDiagnostic{}); err == nil || data != nil {
		t.Fatal("missing owner produced an adoptable change")
	}
	if _, ok := providerPageSealProspective(nil); ok {
		t.Fatal("missing original hold produced a quote")
	}
	if _, ok := providerGuardianSealBoundaryData(nil, nil, providerFrontierHighwater{}); ok {
		t.Fatal("missing independent ledger produced a boundary")
	}
	if _, err := (*providerCustodyReplay)(nil).guardianSealBefore(nil); err == nil {
		t.Fatal("missing retained replay advanced G")
	}
}
