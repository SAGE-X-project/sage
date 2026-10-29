package registry011

import (
	"context"
	"testing"
)

// Only tests can mint this synthetic witness. These isolated originals are
// provisioned with their new profile BEFORE anchor/H0/clock creation. They
// exercise real owner hooks, not a source-cost certificate, group ACK or any
// operating selection. Production has no corresponding derivation factory.
func providerHistoryProvisionedTestBounds(t *testing.T, b *providerIntegrationBounds, capture ProviderCaptureLimits) {
	t.Helper()
	if b.firstAdmission == nil || b.scan == nil || b.history != nil {
		t.Fatal("fresh first-operation test profile required")
	}
	scan := *b.scan
	scan.maximumPreIRetainedBytes /= 2
	scan.maximumPreIWork[2] /= 2
	scan.maximumPreIWork[9] /= 2
	scan.maximumPreIWork[10] /= 2
	scan.maximumPreIClockReads /= 2
	b.scan = &scan
	// This extracts only the fixed P work/storage components. Its two scalar
	// reader inputs do not affect those components and prove no RAM bound.
	plan, ok := providerFirstOperationPlanUpper(*b, capture, 1, 2, 0, 1)
	if !ok {
		t.Fatal("finite first-operation test components")
	}
	work := candidateWorkVector{}
	for i := range work {
		work[i] = 1
	}
	work[10] = 128
	h := &providerHistoryProgramBounds{
		program: admissionTestDigest("a"), costs: admissionTestDigest("b"),
		future:  providerHistoryGroupTotals{work: work, physical: 4096, retainedRaw: 1024, payload: 64, metadata: 64, objects: 3, files: 10, clocks: 32},
		startup: providerScanUpper{work: plan.reservation.work, retained: plan.storage.physical, clocks: 11},
		stages:  1, records: 3, commits: 3, catalogEntries: 10, catalogNameBytes: 2550, resident: 1 << 20, durationMS: 300000,
	}
	h.derivation = &providerHistoryProgramDerivation{}
	h.derivation.owner = h.derivation
	h.derivation.snapshot = *h
	b.history = h
	if !b.valid() {
		t.Fatal("fresh synthetic history shape")
	}
}

func providerHistoryProvisionedFirstFixture(t *testing.T) (*providerCustodyStore, *providerFirstOperationOwner) {
	t.Helper()
	first := providerFirstWriterBounds()
	config, clock := providerScanProvisionedFixtureWithSelection(t, &first, true, &providerScanFixtureSelection{historyProgram: true})
	if clock.read != 0 || clock.history != nil {
		t.Fatal("test profile was selected after observation")
	}
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal("fresh original history profile", err)
	}
	t.Cleanup(func() { _ = s.close() })
	held, ok := providerHistoryWholeHold(config.bounds)
	if !ok || s.history == nil || !s.history.matchesLocked() || s.history.original != s.setupOrigin || s.stored != held.retained || clock.history != s.history {
		t.Fatal("original open did not retain its one pre-observation hold")
	}
	if _, err := s.enrollBeforeScan(); err != nil {
		t.Fatal("original enrollment under the same O observer", err)
	}
	x := mustProviderBindingCodec(t, s.config)
	target, err := x.target(providerScanTarget{setup: s.setupTip, custody: s.tip, frontier: s.frontierFiles.pin.frontier, objects: s.objectsRoot, enrollment: s.enrollment})
	if err != nil {
		t.Fatal(err)
	}
	attempt := providerScanAttemptFixture(t, s, s.scanControl.pin)
	attempt.target = x.hash("sage-provider-scan-target-v1", target)
	if err := s.scanControl.claim(attempt); err != nil {
		t.Fatal("original inspection subclaim", err)
	}
	if _, err := s.scanControl.inspectBeforeAdmission(target); err != nil || s.firstAdmissionOwner == nil || s.history.inspectionPhase != 2 || s.stored != held.retained {
		t.Fatal("original scan charged the parent twice or lost its observer", err)
	}
	o, err := newProviderFirstOperationCandidate(s)
	if err != nil || o == nil {
		t.Fatal("original first operation under its pre-held reservation", err)
	}
	if !s.history.startupClaimed || s.stored != held.retained || s.scanControl.held != s.history.inspection || o.integration.ledger == s.frontierReplay.body.ledger || o.indexes == s.frontierReplay.indexes || s.firstAdmissionOwner.state != providerFirstAdmissionRetained || !s.firstAdmissionOwner.meter.closed {
		t.Fatal("first operation recharged storage or imported a replay model")
	}
	return s, o
}

func TestProviderHistoryProgramConnectsOriginalOpenInspectionAndFirstExchange(t *testing.T) {
	s, o := providerHistoryProvisionedFirstFixture(t)
	p, held, stored := s.history, s.history.held, s.stored
	conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	result, err := o.executeCandidateWithConnection(context.Background(), conn)
	if err != nil || result == nil {
		t.Fatalf("owned history exchange: %v; state=%d read=%d sequence=%d", err, o.state, s.clock.read, s.sequence)
	}
	if o.state != providerFirstOperationDone || o.clocks != 11 || !o.consumption || !o.settlement || s.stored != stored || p.held != held || !p.matchesLocked() || s.last != nil || s.sequence != 4 || string(result.result) != `"0x1"` || s.activeMeter.Load() != nil {
		t.Fatal("exchange lost its original hold or returned operating authority")
	}
	read, actual, clocks := s.clock.read, o.execution.actual, o.execution.clocks
	if _, err := s.samplePrimitive(); err == nil || p.issueClockClaimLocked(o.execution) != nil || s.clock.read != read || o.execution.actual != actual || o.execution.clocks != clocks {
		t.Fatal("completed owner spent the future program's clock allowance")
	}
	if s.close() != nil || !p.failed || !s.clock.historyRevoked || s.stored != stored {
		t.Fatal("close restored the parent or refunded its irreversible hold")
	}
}

func TestProviderHistoryProgramConnectionFailureKeepsTheWholeReservation(t *testing.T) {
	s, o := providerHistoryProvisionedFirstFixture(t)
	p, held, stored := s.history, s.history.held, s.stored
	conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	if conn.Close() != nil {
		t.Fatal("close isolated connection")
	}
	result, err := o.executeCandidateWithConnection(context.Background(), conn)
	if err == nil || result != nil || o.state != providerFirstOperationFailed || !s.failed || s.stored != stored || p.held != held {
		t.Fatal("failed connection was accepted or refunded", err)
	}
	if s.close() != nil || !p.failed || !s.clock.historyRevoked || s.stored != stored {
		t.Fatal("failed store cleanup restored the clock or storage")
	}
}
