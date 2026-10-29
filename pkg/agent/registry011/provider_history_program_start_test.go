package registry011

import (
	"context"
	"encoding/json"
	"testing"
)

func providerHistoryProgramStartedFixture(t *testing.T) (*providerCustodyStore, *providerFirstOperationOwner) {
	t.Helper()
	s, o := providerHistoryProvisionedFirstFixture(t)
	if s.history.firstOwner != o || s.history.firstIntegration != o.integration || s.history.firstIndexes != o.indexes || s.history.firstMain != o.first.mainPin || s.history.start != nil || s.history.startReadyLocked() {
		t.Fatal("prepared first operation was replaced or treated as completed")
	}
	connection := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	if _, err := o.executeCandidateWithConnection(context.Background(), connection); err != nil || !s.history.startReadyLocked() {
		t.Fatal("original measured first exchange did not preserve its handoff", err)
	}
	return s, o
}

func TestProviderHistoryProgramStartPreservesTheOriginalComparedHandoff(t *testing.T) {
	s, o := providerHistoryProgramStartedFixture(t)
	p, start := s.history, s.history.start
	stored, reads, held := s.stored, s.clock.read, p.held
	if start == nil || start.owner != start || start.parent != p || start.first != o || start.main != s.frontierFiles.pin ||
		start.admission != o.integration.intent.digest || start.ticket != o.integration.firstTicket.digest || start.budget != o.integration.ledger.current.digest ||
		start.enrollment != s.enrollment || start.objects != s.objectsRoot || start.exchange != s.frontierReplay.body.firstExchange ||
		o.remaining.failed || !o.remaining.closed || o.clocks != 11 || s.activeMeter.Load() != nil {
		t.Fatal("handoff was not the same fully compared and retained first exchange")
	}
	if p.bindFirstOwnerLocked(o) || p.bindFirstIntegrationLocked(o) || p.handoffFirstLocked(o) {
		t.Fatal("DONE or retained DATA re-created an original owner")
	}
	for range 2 {
		if !p.startReadyLocked() || s.stored != stored || s.clock.read != reads || p.held != held {
			t.Fatal("scalar handoff check replenished or spent an allowance")
		}
	}
	if _, err := s.samplePrimitive(); err == nil || s.clock.read != reads {
		t.Fatal("handoff was accepted as a new clock or execution role")
	}
	if json.Unmarshal([]byte(`{}`), &providerHistoryProgramStart{}) == nil {
		t.Fatal("JSON reconstructed a live handoff")
	}
	if s.close() != nil || p.start != start || p.startReadyLocked() || !p.failed || !s.clock.historyRevoked || s.stored != stored {
		t.Fatal("closed parent forgot its history or renewed the handoff")
	}
}

func TestProviderHistoryProgramStartRefusesChangedOriginalsAndCopies(t *testing.T) {
	s, o := providerHistoryProgramStartedFixture(t)
	p, start := s.history, s.history.start
	stored, reads, held := s.stored, s.clock.read, p.held
	checks := []struct {
		name   string
		change func() func()
	}{
		{"start-copy", func() func() { copy := *start; p.start = &copy; return func() { p.start = start } }},
		{"parent-copy", func() func() { copy := *p; start.parent = &copy; return func() { start.parent = p } }},
		{"first-copy", func() func() { copy := *o; s.firstOperation = &copy; return func() { s.firstOperation = o } }},
		{"first-state", func() func() {
			o.state = providerFirstOperationSettlement
			return func() { o.state = providerFirstOperationDone }
		}},
		{"first-policy", func() func() { before := o.config; o.config.profile = ""; return func() { o.config = before } }},
		{"first-origin", func() func() { before := o.origin; o.origin.MonotonicMS++; return func() { o.origin = before } }},
		{"first-clock", func() func() { before := o.clock; o.clock = nil; return func() { o.clock = before } }},
		{"finishing-copy", func() func() { before := o.first; copy := *before; o.first = &copy; return func() { o.first = before } }},
		{"finishing-meter", func() func() {
			before := o.first.meter
			replacement := &providerWorkMeter{firstAdmission: o.first, closed: true}
			replacement.owner = replacement
			o.first.meter = replacement
			return func() { o.first.meter = before }
		}},
		{"finishing-meter-open", func() func() { o.first.meter.closed = false; return func() { o.first.meter.closed = true } }},
		{"first-main", func() func() { before := p.firstMain; p.firstMain.sequence++; return func() { p.firstMain = before } }},
		{"remaining-open", func() func() { o.remaining.closed = false; return func() { o.remaining.closed = true } }},
		{"execution-unclaimed", func() func() { o.execution.claimed = false; return func() { o.execution.claimed = true } }},
		{"pending-write", func() func() {
			s.frontierFiles.pending = &providerFrontierWriteClaim{}
			return func() { s.frontierFiles.pending = nil }
		}},
		{"current-main", func() func() {
			before := s.frontierFiles.pin
			s.frontierFiles.pin.ordinal++
			return func() { s.frontierFiles.pin = before }
		}},
		{"current-objects", func() func() { before := s.objectsRoot; s.objectsRoot = ""; return func() { s.objectsRoot = before } }},
		{"integration-replacement", func() func() {
			before := o.integration
			o.integration = &providerIntegration{}
			return func() { o.integration = before }
		}},
		{"budget", func() func() { before := start.budget; start.budget = ""; return func() { start.budget = before } }},
		{"exchange", func() func() {
			before := start.exchange
			start.exchange = ""
			return func() { start.exchange = before }
		}},
		{"independent-model", func() func() {
			before := s.frontierReplay.body.ledger
			s.frontierReplay.body.ledger = o.integration.ledger
			return func() { s.frontierReplay.body.ledger = before }
		}},
		{"independent-indexes", func() func() {
			before := s.frontierReplay.indexes
			s.frontierReplay.indexes = o.indexes
			return func() { s.frontierReplay.indexes = before }
		}},
	}
	for _, item := range checks {
		t.Run(item.name, func(t *testing.T) {
			restore := item.change()
			accepted := p.startReadyLocked()
			restore()
			if accepted || !p.startReadyLocked() || s.stored != stored || s.clock.read != reads || p.held != held {
				t.Fatal("changed original or shared model preserved the handoff")
			}
		})
	}
}

func TestProviderHistoryProgramStartRefusesPrematureAndFailedFirstExchange(t *testing.T) {
	var absent *providerHistoryProgramOwner
	if absent.startReadyLocked() || absent.bindFirstOwnerLocked(nil) || absent.bindFirstIntegrationLocked(nil) || absent.handoffFirstLocked(nil) {
		t.Fatal("absent parent supplied a handoff")
	}
	s, o := providerHistoryProvisionedFirstFixture(t)
	p, stored, reads := s.history, s.stored, s.clock.read
	if p.startReadyLocked() || p.handoffFirstLocked(o) || p.start != nil || s.stored != stored || s.clock.read != reads {
		t.Fatal("premature first exchange reached the future program")
	}
	connection := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, providerFirstOperationResponse())
	if connection.Close() != nil {
		t.Fatal("close only the isolated owned test connection")
	}
	if _, err := o.executeCandidateWithConnection(context.Background(), connection); err == nil || p.start != nil || p.startReadyLocked() || !p.failed || s.stored != stored {
		t.Fatal("failed first exchange initialized or refunded its future program")
	}
}
