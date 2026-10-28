package registry011

import (
	"context"
	"testing"
)

// An ordinary owned TLS connection closed before use supplies a local IO
// failure. No remote peer, injected transport callback or attack is created.
func TestProviderContinuingRPCClosedOwnedConnectionKeepsFailedSettlement(t *testing.T) {
	t.Parallel()
	s, g := providerContinuingRPCReadyFixture(t)
	o, err := newProviderContinuingRPCCandidate(s, "eth_chainId", []any{})
	if err != nil {
		t.Fatal("normal RPC admission", err)
	}
	stored, sequence := s.stored, s.sequence
	conn := providerIntegrationOwnedTLS(t, o.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
	if err := conn.Close(); err != nil {
		t.Fatal("normal local close", err)
	}
	value, err := o.executeRPCCandidateWithConnection(context.Background(), conn)
	if err == nil || value != nil || !s.failed || !g.failed || g.ready || o.state != providerContinuingFailed || !o.consumption || !o.settlement || o.permit == nil || !o.permit.used || o.permit.ack != nil ||
		s.stored != stored || s.sequence != sequence+2 || !o.execution.closed || o.execution.claimed || o.remaining == nil || !o.remaining.closed || o.execution.actual[0] != 1 || s.activeMeter.Load() != nil {
		t.Fatal("local transport failure was refunded, retried or published", err, o.state, s.sequence)
	}
	entry, exists := s.frontierReplay.body.ledger.tickets[o.ticket.digest]
	if !exists || entry.terminal == "" {
		t.Fatal("failed RPC omitted original settlement")
	}
	raw, readErr := s.evidence.read(providerObjectName("settlement", entry.terminal), candidateLedgerMetadataLimit)
	object := contentObject(contentJSON(raw), "format", "ticket_digest", "outcome", "actual_work", "retained_root", "clock_end")
	if readErr != nil || object == nil || object["outcome"] != "failed" || object["retained_root"] != nil {
		t.Fatal("failed outcome was converted into a complete exchange", readErr)
	}
	if again, err := o.executeRPCCandidate(context.Background()); err == nil || again != nil || s.stored != stored {
		t.Fatal("failed RPC restored a permit")
	}
}
