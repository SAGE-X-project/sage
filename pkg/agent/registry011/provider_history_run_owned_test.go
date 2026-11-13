package registry011

import (
	"context"
	"testing"
)

// The existing synthetic UNIT selection covers one recorded public block.
// It is not a production derivation, operating profile or live-RPC result.
// All peers use the existing normal TLS fixture; no new transport hook,
// responder fault, endpoint or derivation constructor is introduced.
func historyCallerRecordedBundle(t *testing.T, config providerCustodyConfig) *providerHistoryOwnedConnections {
	t.Helper()
	recorded := nodeRecorded(t)
	return &providerHistoryOwnedConnections{
		first:   providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, string(recorded["canonical-deployment"])),
		members: historyCollectionRecordedRunConnections(t, config.bounds.history.collection.input, recorded),
	}
}

func TestProviderHistoryOwnedCallerReturnsVerifiedDataAfterClosingBothRoots(t *testing.T) {
	config, clock := historyCallerFreshConfiguration(t)
	bundle := historyCallerRecordedBundle(t, config)
	got, err := runProviderHistoryWithOwnedConnectionsCandidate(context.Background(), config, clock, bundle)
	p := clock.history
	if err != nil || got == (providerHistoryFinalData{}) || p == nil || p.store == nil || p.group == nil || p.final == nil ||
		p.firstOwner == nil || p.firstOwner.state != providerFirstOperationDone || !p.firstOwner.consumption || !p.firstOwner.settlement {
		t.Fatal("normal original invocation did not return verified detached DATA", err)
	}
	s, g, f := p.store, p.group, p.final
	if got.program != config.bounds.history.program || got.costs != config.bounds.history.costs || got.profile != config.profile ||
		got.manifest != config.bounds.history.collection.digest || got.selected != config.bounds.history.collection.input.journal.Deployment ||
		got.groups != 4 || !lowerHash(got.model) || !lowerHash(got.commit) || !lowerHash(got.collection) ||
		got.files == 0 || got.bytes == 0 || got.entries == 0 || got.names == 0 ||
		g.coordinator == nil || g.custodian == nil || g.coordinator.stage != 4 || g.custodian.stage != 4 || g.meter.actual[0] != 29 ||
		!f.used || !f.closed || f.reader == nil || !f.reader.closed || f.reader.active || f.reader.meter == nil ||
		got.work != f.reader.meter.actual || f.reader.meter.clocks != 2 || got.clockID != f.begin.ClockID || got.epoch != f.begin.Epoch ||
		!s.closed || !p.failed || !clock.historyRevoked || s.stored != p.held.retained || s.activeMeter.Load() != nil ||
		f.result != nil || f.preparedData != (providerHistoryFinalData{}) {
		t.Fatal("normal owning caller lost original identities, final verification or terminal cleanup")
	}
	// Acquiring both original locks proves cleanup completed before DATA left
	// the owning invocation. This observation must not re-enter its clock/work.
	before, reads, actual, held := got, clock.read, f.reader.meter.actual, s.stored
	historyAssertFinalInputSourceCost(t, config, got)
	historyFinalDataRootsReleased(t, s)
	if err := closeProviderHistoryCallCandidate(s, context.Background(), nil); err != nil ||
		got != before || clock.read != reads || f.reader.meter.actual != actual || s.stored != held {
		t.Fatal("terminal cleanup repeated original work or invalidated detached DATA", err)
	}
}

func TestProviderHistoryOwnedCallerClosesRootsAfterFirstMemberConnectionFailure(t *testing.T) {
	config, clock := historyCallerFreshConfiguration(t)
	bundle := historyCallerRecordedBundle(t, config)
	// Close only our first normal member connection. The first operation
	// still completes normally; no hostile response or external RPC is used.
	if err := bundle.members[0].Close(); err != nil {
		t.Fatal("ordinary owned closed-member prerequisite", err)
	}
	got, err := runProviderHistoryWithOwnedConnectionsCandidate(context.Background(), config, clock, bundle)
	p := clock.history
	if err == nil || got != (providerHistoryFinalData{}) || p == nil || p.store == nil ||
		p.firstOwner == nil || p.firstOwner.state != providerFirstOperationDone || !p.firstOwner.consumption || !p.firstOwner.settlement ||
		p.group == nil || p.group.order == nil || p.group.dispatch == nil || p.group.dispatch.attempt == nil || p.final == nil {
		t.Fatal("member refusal did not cross the completed original first operation", err)
	}
	s, g, a := p.store, p.group, p.group.dispatch.attempt
	if !a.used || a.ordinal != 0 || a.connection != nil || a.original != nil || a.failure != nil ||
		g.order.consumed != (1<<18)-1 || g.order.sent != 1 || g.meter.actual[0] != 1 || g.meter.actual[1] != 0 ||
		g.phase != "failed" || !g.failed || !p.failed || !g.meter.failed || !g.meter.closed || p.final.used ||
		!s.closed || !clock.historyRevoked || s.stored != p.held.retained || s.activeMeter.Load() != nil {
		t.Fatal("member connection refusal published DATA, renewed the hold or retained a live role")
	}
	reads, actual, held := clock.read, g.meter.actual, s.stored
	historyFinalDataRootsReleased(t, s)
	if clock.read != reads || g.meter.actual != actual || s.stored != held {
		t.Fatal("checking terminal member cleanup renewed original observations")
	}
}
