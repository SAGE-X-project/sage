package registry011

import (
	"context"
	"math"
	"testing"
)

func TestProviderIntegrationHistoryCapacityPreservesLifetimeCap(t *testing.T) {
	base := FinalizedHeader{admissionTestDigest("a"), admissionTestDigest("b"), sepoliaDeploymentBlock, 1}
	selected := base
	selected.Number += 26791 // The retained old actual interval, as inert data.
	bounds := CandidateLedgerBounds{4096, 16 << 20, 64 << 20}
	if providerHistoryCapacity(bounds, [3]uint64{}, base, selected) {
		t.Fatal("long deployed history fit a reset or expanded ledger")
	}
	selected.Number = base.Number + 2
	if !providerHistoryCapacity(bounds, [3]uint64{1, 1, 1}, base, selected) {
		t.Fatal("small bounded interval")
	}
	for _, counts := range [][3]uint64{{4096, 0, 0}, {math.MaxUint64, 1, 0}, {4090, 0, 0}} {
		if providerHistoryCapacity(bounds, counts, base, selected) {
			t.Fatal("used claims or future local work disappeared")
		}
	}
	selected.Number = base.Number - 1
	if providerHistoryCapacity(bounds, [3]uint64{}, base, selected) || providerHistoryCapacity(CandidateLedgerBounds{}, [3]uint64{}, base, base) {
		t.Fatal("unsupported history bounds")
	}
	selected.Number = math.MaxUint64
	if providerHistoryCapacity(bounds, [3]uint64{}, base, selected) {
		t.Fatal("record lower-bound overflow")
	}
}

func TestProviderIntegrationCollectionScopeAndModeRefuseBeforeReads(t *testing.T) {
	for _, reads := range []*PinnedAgentReads{nil, {}, {rpc: &ReadRPC{}}, {rpc: &ReadRPC{acquisition: &rpcAcquisition{}}}} {
		if providerCollectionScope(reads) {
			t.Fatal("non-owned client became a provider scope")
		}
		if _, err := collectSourceBatchMode(context.Background(), reads, JournalConfig{}, JournalCheckpoint{}, SourceReadLimits{}, &ObservationChecks{}, nodeIdentity{}, sourceProviderStateGas); err == nil {
			t.Fatal("provider collection started without an owned scope")
		}
	}
	if _, err := collectSourceBatchMode(context.Background(), nil, JournalConfig{}, JournalCheckpoint{}, SourceReadLimits{}, nil, nodeIdentity{}, sourceAcquisitionMode(255)); err == nil {
		t.Fatal("unknown acquisition mode")
	}
}

func TestProviderCollectionPredicateRequiresAnUnsettledLocalCommand(t *testing.T) {
	for _, mode := range []string{"normal-data", "absent-meter", "foreign", "authority", "closed", "failed", "nested", "missing-ticket", "nil-ticket", "rpc-ticket", "unconsumed", "terminal", "header"} {
		t.Run(mode, func(t *testing.T) {
			c, s := providerPendingIntegrationUnit(t)
			// This is a finite DATA predicate test only. These labels and the
			// typed placeholders never come from retained completed work, so
			// they cannot qualify Source, authorize collection or reach I/O.
			c.bootstrap, c.pin = admissionTestDigest("a"), admissionTestDigest("b")
			c.selected = FinalizedHeader{admissionTestDigest("a"), admissionTestDigest("b"), sepoliaDeploymentBlock, 1}
			ticket := *c.firstTicket
			ticket.kind = "local-work"
			c.ledger.tickets[ticket.digest] = candidateLedgerTicket{ticket: &ticket, consumption: &CandidateConsumption{}}
			meter := newProviderWorkMeter(c, providerOperationReservation{})
			meter.ticket = ticket.digest
			s.activeMeter.Store(meter)
			defer s.activeMeter.Store(nil)
			reads := &PinnedAgentReads{rpc: &ReadRPC{acquisition: &rpcAcquisition{integration: c}}, header: c.selected}
			switch mode {
			case "absent-meter":
				s.activeMeter.Store(nil)
			case "foreign":
				meter.integration = &providerIntegration{}
			case "authority":
				meter.authority = true
			case "closed":
				meter.closed = true
			case "failed":
				meter.failed = true
			case "nested":
				meter.parent = &providerWorkMeter{}
			case "missing-ticket":
				delete(c.ledger.tickets, ticket.digest)
			case "nil-ticket":
				c.ledger.tickets[ticket.digest] = candidateLedgerTicket{}
			case "rpc-ticket":
				ticket.kind = "rpc-capture"
			case "unconsumed":
				c.ledger.tickets[ticket.digest] = candidateLedgerTicket{ticket: &ticket}
			case "terminal":
				c.ledger.tickets[ticket.digest] = candidateLedgerTicket{ticket: &ticket, consumption: &CandidateConsumption{}, terminal: admissionTestDigest("f")}
			case "header":
				reads.header.Number++
			}
			if providerCollectionScope(reads) != (mode == "normal-data") {
				t.Fatal("local-command DATA predicate did not match its scope", mode)
			}
		})
	}
}
