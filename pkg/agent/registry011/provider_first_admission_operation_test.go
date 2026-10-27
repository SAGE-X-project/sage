package registry011

import (
	"context"
	"sync"
	"testing"
)

func providerFirstOperationPreparedFixture(t *testing.T) (*providerCustodyStore, *providerFirstAdmissionOwner) {
	t.Helper()
	first := providerFirstWriterBounds()
	config, clock := providerScanProvisionedFixtureWithFirstPolicy(t, &first, true)
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.close() })
	if _, err := s.enrollBeforeScan(); err != nil {
		t.Fatal("fresh isolated E0", err)
	}
	x := mustProviderBindingCodec(t, s.config)
	target, err := x.target(providerScanTarget{setup: s.setupTip, custody: s.tip, frontier: s.frontierFiles.pin.frontier, objects: s.objectsRoot, enrollment: s.enrollment})
	if err != nil {
		t.Fatal(err)
	}
	attempt := providerScanAttemptFixture(t, s, s.scanControl.pin)
	attempt.target = x.hash("sage-provider-scan-target-v1", target)
	if s.scanControl.claim(attempt) != nil {
		t.Fatal("fresh original scan hold")
	}
	if _, err := s.scanControl.inspectBeforeAdmission(target); err != nil || s.firstAdmissionOwner == nil {
		t.Fatal("fresh original scan", err)
	}
	return s, s.firstAdmissionOwner
}

func providerFirstOperationFixture(t *testing.T) (*providerCustodyStore, *providerFirstOperationOwner) {
	t.Helper()
	s, f := providerFirstOperationPreparedFixture(t)
	before, held := s.stored, s.scanControl.held
	o, err := newProviderFirstOperationCandidate(s)
	if err != nil || o == nil {
		t.Fatalf("connected candidate open: %v; Fstate=%d work=%v clocks=%d plan=%+v seq=%d", err, f.state, f.meter.actual, f.meter.clocks, f.plan, s.sequence)
	}
	if s.stored != before+o.plan.storage.physical || s.scanControl.held != held || s.last != nil || s.sequence != 2 || len(s.objects) != 21 || f.state != providerFirstAdmissionRetained || !f.meter.closed || f.meter.failed || o.integration.ledger == s.frontierReplay.body.ledger || o.indexes == s.frontierReplay.indexes || o.execution == f.meter || !o.fits(o.execution) || o.execution.actual != (candidateWorkVector{}) {
		t.Fatal("F was reopened, P repeated or independent state became authority")
	}
	return s, o
}

func providerFirstOperationResponse() string {
	return `{"jsonrpc":"2.0","id":1,"result":"0x1"}`
}

func TestProviderFirstOperationConnectsOriginalConsumptionOneCaptureAndSettlement(t *testing.T) {
	s, o := providerFirstOperationFixture(t)
	stored, raw, held := s.stored, s.retainedRaw, s.scanControl.held
	conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	result, err := o.executeCandidateWithConnection(context.Background(), conn)
	if err != nil || result == nil {
		t.Fatalf("connected candidate exchange: %v; state=%d seq=%d E=%v R=%+v clocks=%d", err, o.state, s.sequence, o.execution.actual, o.remaining, o.clocks)
	}
	if o.state != providerFirstOperationDone || !o.consumption || !o.settlement || o.permit == nil || !o.permit.used || o.permit.ack != nil || s.last != nil || s.sequence != 4 || len(s.objects) != 38 || s.stored != stored || s.scanControl.held != held || o.clocks != 11 || o.execution.clocks != 7 || o.remaining.clocks != 4 || !o.execution.closed || !o.execution.claimed || o.execution.failed || !o.remaining.closed || o.remaining.failed || s.activeMeter.Load() != nil || string(result.result) != `"0x1"` || result.exchange != s.frontierReplay.body.firstExchange || result.highwater != s.frontierFiles.pin || result.enrollment != s.enrollment {
		t.Fatal("capture bypassed a boundary, reused a hold or returned operating authority")
	}
	var responseWire uint64
	for _, ref := range s.objects {
		if ref.kind == "response" {
			responseWire += ref.bytes
		}
	}
	if result.execution != o.execution.actual || result.remaining != o.remaining.actual || result.execution[0] != 1 || result.execution[8] != 1 || result.remaining[0] != 0 || result.remaining[1] != 0 || result.remaining[8] != 0 || responseWire == 0 || result.execution[1] != uint64(len(providerFirstOperationRequestWire))+responseWire || result.execution[9] == 0 || result.remaining[9] == 0 || o.physical != result.execution[9]+result.remaining[9]+31*s.config.bounds.fileOverheadBytes || s.retainedRaw != raw+result.execution[9]+result.remaining[9] {
		t.Fatal("uppers were reported as actual, or physical C/S file counts changed", result.execution, result.remaining)
	}
	for coordinate, actual := range result.execution {
		if actual > o.plan.reservation.execution[coordinate] || result.remaining[coordinate] > o.plan.reservation.remaining[coordinate] {
			t.Fatal("prospective execution/remaining bounds were exceeded", coordinate)
		}
	}
	state := s.frontierReplay.body.ledger.tickets[result.ticket]
	if state.consumption == nil || state.terminal == "" || s.frontierReplay.body.ledger.current.totals.bootstrap.outstanding != (candidateWorkVector{}) || o.integration.ledger.current.totals != s.frontierReplay.body.ledger.current.totals || o.integration.ledger.current.privatePayload != s.frontierReplay.body.ledger.current.privatePayload || o.integration.ledger.current.privateMetadata != s.frontierReplay.body.ledger.current.privateMetadata {
		t.Fatal("independent final model did not retain the complete charged T/C/S")
	}
	for _, ref := range s.objects {
		data, readErr := s.evidence.read(providerObjectName(ref.kind, ref.digest), int64(ref.bytes))
		digest, valid := providerIntegrationObjectDigest(ref.kind, data)
		if readErr != nil || !valid || digest != ref.digest {
			t.Fatal("candidate result preceded exact original custody", ref.kind, readErr)
		}
	}
	if again, err := o.executeCandidate(context.Background()); err == nil || again != nil || s.failed || s.stored != stored || o.state != providerFirstOperationDone || o.burnLocked(o.permit) || o.beginRemainingLocked(o.permit) || o.consumeStorageLocked(1) || o.writingLocked(s) || o.UnmarshalJSON(nil) == nil || result.UnmarshalJSON(nil) == nil || o.plan.UnmarshalJSON(nil) == nil {
		t.Fatal("completed diagnostic state restored a live owner or request")
	}
}

func TestProviderFirstOperationConcurrentUseHasOneCandidateRequest(t *testing.T) {
	s, o := providerFirstOperationFixture(t)
	conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	var group sync.WaitGroup
	results := make(chan bool, 2)
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			data, err := o.executeCandidateWithConnection(context.Background(), conn)
			results <- err == nil && data != nil
		}()
	}
	group.Wait()
	close(results)
	passed := 0
	for ok := range results {
		if ok {
			passed++
		}
	}
	if passed != 1 || s.failed || o.execution.actual[0] != 1 || s.sequence != 4 || o.state != providerFirstOperationDone {
		t.Fatal("concurrent candidate calls repeated capture or damaged the completed owner", passed)
	}
}

func TestProviderFirstOperationUnavailableOwnedConnectionKeepsAFullFailedCharge(t *testing.T) {
	s, o := providerFirstOperationFixture(t)
	stored, held := s.stored, s.scanControl.held
	conn := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	if conn.Close() != nil {
		t.Fatal("close owned fixture connection")
	}
	data, err := o.executeCandidateWithConnection(context.Background(), conn)
	if err == nil || data != nil || !s.failed || o.state != providerFirstOperationFailed || !o.consumption || !o.settlement || o.permit == nil || !o.permit.used || s.last != nil || s.sequence != 4 || len(s.objects) != 34 || s.stored != stored || s.scanControl.held != held || o.execution.claimed || !o.execution.failed || !o.execution.closed || o.remaining == nil || !o.remaining.closed || o.clocks > 11 || s.activeMeter.Load() != nil {
		t.Fatal("unavailable owned connection was retried, measured complete or refunded", err, o.state, s.sequence)
	}
	state := s.frontierReplay.body.ledger.tickets[o.integration.firstTicket.digest]
	if state.consumption == nil || state.terminal == "" {
		t.Fatal("failed capture lacked durable C/S originals")
	}
	raw, err := s.evidence.read(providerObjectName("settlement", state.terminal), candidateLedgerMetadataLimit)
	value := contentJSON(raw).(map[string]any)
	if err != nil || value["outcome"] != "failed" || value["actual_work"] != nil || value["retained_root"] != nil || s.frontierReplay.body.ledger.current.totals.bootstrap.outstanding != (candidateWorkVector{}) {
		t.Fatal("failed execution invented actual work or lost its full charge")
	}
}
