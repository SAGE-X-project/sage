package registry011

import (
	"context"
	"strconv"
	"testing"
)

// The larger independent test input is selected before a new H0/clock. No
// existing namespace, public operation limit or CI threshold is increased.
func providerContinuingSettledFixture(t *testing.T, header bool) (*providerCustodyStore, *providerGuardianOwner) {
	t.Helper()
	return providerContinuingSettledWithProfileFixture(t, header, 64<<20, 6)
}

func providerContinuingSettledWithProfileFixture(t *testing.T, header bool, decoded, records uint64) (*providerCustodyStore, *providerGuardianOwner) {
	t.Helper()
	config, clock := providerGuardianConfigWithPageFixture(t, decoded, records)
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal("fresh continuation profile", err)
	}
	t.Cleanup(func() { _ = s.close() })
	if _, err := s.enrollBeforeScan(); err != nil {
		t.Fatal("fresh continuation genesis", err)
	}
	x := mustProviderBindingCodec(t, s.config)
	target, err := x.target(providerScanTarget{setup: s.setupTip, custody: s.tip, frontier: s.frontierFiles.pin.frontier, objects: s.objectsRoot, enrollment: s.enrollment})
	if err != nil {
		t.Fatal(err)
	}
	attempt := providerScanAttemptFixture(t, s, s.scanControl.pin)
	attempt.target = x.hash("sage-provider-scan-target-v1", target)
	if s.scanControl.claim(attempt) != nil {
		t.Fatal("original continuation scan hold")
	}
	if _, err := s.scanControl.inspectBeforeAdmission(target); err != nil {
		t.Fatal("full original continuation scan", err)
	}
	o, err := newProviderFirstOperationCandidate(s)
	if err != nil {
		t.Fatal("original first operation", err)
	}
	response := providerFirstOperationResponse()
	if header {
		response = `{"jsonrpc":"2.0","id":1,"result":{"hash":"` + admissionTestDigest("a") + `","parentHash":"` + admissionTestDigest("b") + `","number":"0x` + strconv.FormatUint(sepoliaDeploymentBlock+1, 16) + `","timestamp":"0x1"}}`
	}
	conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, response)
	if _, err := o.executeCandidateWithConnection(context.Background(), conn); err != nil {
		t.Fatalf("normal owned first X: %v; state=%d E=%v R=%+v", err, o.state, o.execution.actual, o.remaining)
	}
	return s, s.guardian
}

func providerContinuingPreparedFixture(t *testing.T) (*providerCustodyStore, *providerContinuingOwner) {
	t.Helper()
	s, _ := providerContinuingSettledFixture(t, true)
	o, err := newProviderContinuingBootstrapCandidate(s)
	if err != nil {
		t.Fatalf("next local T: %v; main=%d Gphase=%d Gwork=%+v", err, s.sequence, s.guardian.phase, s.guardian.meter)
	}
	return s, o
}

func TestProviderContinuingBootstrapRetainsOwnTicketConsumptionAndSettlement(t *testing.T) {
	t.Parallel()
	s, g := providerContinuingSettledFixture(t, true)
	first := g.first
	oldE, oldR, oldClocks, origin, before, raw := first.execution.actual, first.remaining.actual, first.clocks, s.setupOrigin, s.stored, s.retainedRaw
	objects := len(s.objects)
	o, err := newProviderContinuingBootstrapCandidate(s)
	if err != nil {
		t.Fatalf("next local T: %v; G=%+v", err, g)
	}
	if o.state != providerContinuingPrepared || s.sequence != 5 || g.ordinal != 1 || g.ready || !g.active || !g.meter.closed ||
		s.stored != before+o.plan.physical || s.last != nil || o.ticket.sequence != 1 || o.ticket.input != o.integration.firstExchange ||
		o.execution == first.execution || o.remaining != nil || o.execution.actual != (candidateWorkVector{}) {
		t.Fatal("later T borrowed an old owner or lacked its exact G/P hold")
	}
	// Diagnostic producer DATA is intentionally unavailable; only original X/S
	// can supply the selected header to this closed local operation.
	o.integration.firstResult = "unusable diagnostic DATA"
	result, err := o.executeBootstrapCandidate(context.Background())
	if err != nil || result == nil {
		t.Fatalf("next local C/S: %v; state=%d seq=%d E=%+v R=%+v", err, o.state, s.sequence, o.execution, o.remaining)
	}
	if o.state != providerContinuingDone || s.sequence != 7 || !o.consumption || !o.settlement || o.permit == nil || !o.permit.used || o.permit.ack != nil ||
		!g.ready || g.active || g.continuing != nil || g.claim != nil || g.meter != nil || g.ordinal != 1 || g.pin.ordinal != 1 || g.main != s.frontierFiles.pin ||
		s.setupOrigin != origin || s.stored != before+o.plan.physical || s.last != nil || s.activeMeter.Load() != nil || len(s.objects) != objects+22 ||
		first.state != providerFirstOperationDone || first.execution.actual != oldE || first.remaining.actual != oldR || first.clocks != oldClocks ||
		!o.execution.closed || !o.execution.claimed || o.execution.failed || !o.remaining.closed || o.remaining.failed ||
		result.selected.Number != sepoliaDeploymentBlock+1 || result.selected.Hash != admissionTestDigest("a") || result.bootstrap != o.integration.bootstrap ||
		result.bootstrap != s.frontierReplay.body.start || result.highwater != s.frontierFiles.pin || result.enrollment != s.enrollment {
		t.Fatal("continuation reset original authority, used diagnostics or escaped before durable S")
	}
	if result.execution != o.execution.actual || result.remaining != o.remaining.actual || result.execution[0] != 0 || result.execution[1] != 0 ||
		result.execution[3] != 1 || result.execution[8] != o.ticket.segment-o.previousTicket.segment || result.remaining[0] != 0 || result.remaining[1] != 0 || result.remaining[3] != 0 || result.remaining[8] != 0 ||
		o.physical != result.execution[9]+result.remaining[9]+28*s.config.bounds.fileOverheadBytes ||
		s.retainedRaw != raw+g.auxiliaryMetadata+o.metadataRaw+result.execution[9]+result.remaining[9] {
		t.Fatal("a prospective upper became actual or retained bytes were double counted", result.execution, result.remaining)
	}
	for coordinate, n := range result.execution {
		if n > o.plan.reservation.execution[coordinate] || result.remaining[coordinate] > o.plan.reservation.remaining[coordinate] {
			t.Fatal("closed phase exceeded its prospective envelope", coordinate)
		}
	}
	if again, err := o.executeBootstrapCandidate(context.Background()); err == nil || again != nil || s.failed ||
		o.burnLocked(o.permit) || o.beginRemainingLocked(o.permit) || o.releaseLocked() || o.consumeStorageLocked(1) ||
		o.UnmarshalJSON(nil) == nil || result.UnmarshalJSON(nil) == nil || o.plan.UnmarshalJSON(nil) == nil {
		t.Fatal("completed continuation DATA restored a live permission")
	}
}

func TestProviderContinuingInvalidOriginalResultGetsOnlyFullChargeFailedSettlement(t *testing.T) {
	t.Parallel()
	s, _ := providerContinuingSettledFixture(t, false)
	o, err := newProviderContinuingBootstrapCandidate(s)
	if err != nil {
		t.Fatal(err)
	}
	stored := s.stored
	result, err := o.executeBootstrapCandidate(context.Background())
	if err == nil || result != nil || !s.failed || !o.guardian.failed || o.state != providerContinuingFailed || s.sequence != 7 ||
		!o.consumption || !o.settlement || o.permit == nil || !o.permit.used || o.execution.claimed || o.remaining == nil ||
		!o.execution.closed || !o.remaining.closed || s.stored != stored || s.activeMeter.Load() != nil || o.integration.bootstrap != "" {
		t.Fatal("invalid local result was refunded, retried or published", err, o.state, s.sequence)
	}
	entry := s.frontierReplay.body.ledger.tickets[o.ticket.digest]
	raw, err := s.evidence.read(providerObjectName("settlement", entry.terminal), candidateLedgerMetadataLimit)
	value := contentObject(contentJSON(raw), "format", "ticket_digest", "outcome", "actual_work", "retained_root", "clock_end")
	if err != nil || value == nil || value["outcome"] != "failed" || value["actual_work"] != nil || value["retained_root"] != nil {
		t.Fatal("failed local E invented a measured success")
	}
}
