package registry011

import (
	"context"
	"testing"
)

func TestProviderGuardianOriginGenesisAndInitialHoldPrecedeFirstAdmission(t *testing.T) {
	s := providerGuardianStoreFixture(t)
	g := s.guardian
	if g == nil || !g.selectedLocked() || !g.initialized || g.admitted || g.ready || g.active || g.ordinal != 0 || g.original != s.setupOrigin || g.main != s.frontierFiles.pin || s.sequence != 1 || len(s.objects) != 8 || g.genesisMetadata == 0 || s.stored != s.config.bounds.setupRetained+g.bounds.lifetime.retained || g.meter != nil || s.activeMeter.Load() != nil || s.last != nil {
		t.Fatal("guardian was recreated from DATA or held after I")
	}
	for _, name := range []string{"GUARDIAN-HEAD", s.name("guardian-highwater", 0)} {
		raw, err := s.custody.read(name, candidateLedgerMetadataLimit)
		if err != nil || string(raw) != g.pinRaw {
			t.Fatal("genesis preceded exact original readbacks", name, err)
		}
	}
	stored := s.stored
	if g.startClaimLocked("bootstrap-admission") == nil || g.admitted || s.stored != stored || s.sequence != 1 {
		t.Fatal("pre-I metadata supplied later-work permission")
	}
	if g.installGenesisLocked(s.bootstrapOwner) == nil {
		t.Fatal("completed O regenerated genesis")
	}
	if _, err := g.originBeforeEnrollmentLocked(s.bootstrapOwner.meter); err == nil {
		t.Fatal("completed O regenerated origin")
	}
}

func TestProviderGuardianFirstSettlementHandsOffOnlyToDistinctFutureMetadata(t *testing.T) {
	s, o := providerGuardianFirstFixture(t)
	g := s.guardian
	if !g.admitted || g.ready || g.active || g.first != o || g.integration != o.integration || g.indexes != o.indexes {
		t.Fatal("first admission skipped the original guardian binding")
	}
	conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	result, err := o.executeCandidateWithConnection(context.Background(), conn)
	if err != nil || result == nil {
		t.Fatalf("V5 owned first exchange: %v state=%d seq=%d E=%v R=%+v", err, o.state, s.sequence, o.execution.actual, o.remaining)
	}
	if !g.ready || g.active || g.meter != nil || g.main != s.frontierFiles.pin || g.original != s.setupOrigin || g.pin.ordinal != 0 || o.state != providerFirstOperationDone || !o.remaining.closed || len(s.objects) != 39 {
		t.Fatal("first R was reopened or guardian advanced before its own claim")
	}
	stored, original, execution := s.stored, g.original, o.execution.actual
	s.mu.Lock()
	defer s.mu.Unlock()
	if g.bindFirstOwnerLocked(o) || g.bindFirstIntegrationLocked(o) || g.handoffFirstLocked(o) || g.checkFirstOriginalsLocked(nil, nil) {
		t.Fatal("completed first role was rebound")
	}
	if err := g.startClaimLocked("bootstrap-admission"); err != nil {
		t.Fatal("distinct prefunded guardian claim", err)
	}
	if g.ready || !g.active || g.ordinal != 1 || g.meter == o.remaining || g.meter == o.execution || g.original != original || g.meter.actual[0] != 0 || g.meter.actual[8] != 0 || g.meter.clocks != 1 {
		t.Fatal("new metadata borrowed old R or began protected work")
	}
	if err := g.retainClaimLocked(g.integration.firstExchange); err != nil {
		t.Fatalf("retained G claim: %v phase=%d work=%v", err, g.phase, g.meter.actual)
	}
	data, err := providerGuardianClaimOriginals(s, g.meter, g.before, g.main, g.integration.intent.digest, g.integration.firstExchange, "bootstrap-admission")
	if err != nil || data == nil || data.head != g.pin || data.metadata != g.metadata || g.pin.ordinal != 1 || g.phase != providerGuardianRecorded || s.stored != stored || g.physical == 0 || g.meter.actual[9] != g.raw || o.execution.actual != execution || g.meter.actual[0] != 0 || g.meter.actual[8] != 0 || s.sequence != 4 || s.last != nil {
		t.Fatal("claim originals or hold accounting changed", err)
	}
	if g.startClaimLocked("bootstrap-admission") == nil || g.retainClaimLocked(g.integration.firstExchange) == nil || o.burnLocked(o.permit) || o.beginRemainingLocked(o.permit) {
		t.Fatal("consumed G or DONE first owner was reused")
	}
}
