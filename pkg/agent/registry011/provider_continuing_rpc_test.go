package registry011

import (
	"context"
	"encoding/json"
	"testing"
)

func providerContinuingRPCReadyFixture(t *testing.T) (*providerCustodyStore, *providerGuardianOwner) {
	return providerContinuingRPCReadyWithSelectionFixture(t, nil)
}

func providerContinuingRPCReadyWithSelectionFixture(t *testing.T, selected *providerScanFixtureSelection) (*providerCustodyStore, *providerGuardianOwner) {
	t.Helper()
	s, g := providerPageSealReadyWithSelectionFixture(t, 64<<20, 6, selected)
	if _, err := sealProviderContinuingPageCandidate(s); err != nil {
		t.Fatal("normal completed page before read-only RPC", err)
	}
	return s, g
}

func TestProviderContinuingRPCKeepsOriginalLifetimeAcrossRequestsAndPages(t *testing.T) {
	t.Parallel()
	s, g := providerContinuingRPCReadyWithSelectionFixture(t, &providerScanFixtureSelection{continuingWire: true})
	first := g.first
	origin, initial, oldE, oldR := s.setupOrigin, g.integration.firstExchange, first.execution.actual, first.remaining.actual
	methods := []string{"eth_chainId", "web3_clientVersion", "eth_chainId"}
	results := []string{`"0xaa36a7"`, `"SAGE fixture"`, `"0xaa36a7"`}
	var previous *providerContinuingOwner
	for i, method := range methods {
		if i == 2 {
			if _, err := sealProviderContinuingPageCandidate(s); err != nil {
				t.Fatal("normal second retained page", err)
			}
		}
		before, ordinal := s.stored, g.ordinal
		o, err := newProviderContinuingRPCCandidate(s, method, []any{})
		if err != nil {
			t.Fatalf("new retained request %d: %v Gphase=%d Ghead=%d ordinal=%d main=%d ticket=%d metadata=%d meter=%+v", i, err, g.phase, g.pin.ordinal, g.ordinal, s.sequence, g.integration.ledger.lastTicket.sequence, g.integration.ledger.current.privateMetadata, g.meter)
		}
		if o.request == nil || o.ticket.kind != "rpc-capture" || o.ticket.input != o.request.inputDigest || o.ticket.params != o.request.paramsDigest ||
			g.ordinal != ordinal+1 || s.stored != before+o.plan.physical || o.execution == first.execution || o.remaining != nil || o.execution.actual != (candidateWorkVector{}) {
			t.Fatal("RPC admission borrowed a closed role or changed its request")
		}
		if previous != nil && (o.execution == previous.execution || previous.state != providerContinuingDone || !previous.remaining.closed) {
			t.Fatal("next RPC restored a previous owner")
		}
		response := `{"jsonrpc":"2.0","id":1,"result":` + results[i] + `}`
		connection := providerIntegrationOwnedTLS(t, o.request.body, response)
		data, err := o.executeRPCCandidateWithConnection(context.Background(), connection)
		if err != nil || data == nil {
			t.Fatalf("ordinary owned RPC %d: %v state=%d E=%+v R=%+v", i, err, o.state, o.execution, o.remaining)
		}
		if string(data.result) != results[i] || string(data.response) != response || !lowerHash(data.exchange) || s.bindings[data.exchange] == "" ||
			o.state != providerContinuingDone || !o.consumption || !o.settlement || o.permit == nil || !o.permit.used || o.permit.ack != nil ||
			!o.execution.closed || !o.execution.claimed || !o.remaining.closed || o.execution.actual[0] != 1 || o.execution.actual[1] == 0 ||
			o.remaining.actual[0] != 0 || o.remaining.actual[1] != 0 || s.activeMeter.Load() != nil || !g.ready || g.active || g.continuing != nil ||
			s.setupOrigin != origin || g.original != origin || g.integration.firstExchange != initial || g.integration.lastExchange != data.exchange ||
			s.frontierReplay.body.firstExchange != initial || first.execution.actual != oldE || first.remaining.actual != oldR || s.stored != before+o.plan.physical {
			t.Fatal("normal RPC reset the lifetime, borrowed counters or released before independent retention")
		}
		if again, err := o.executeRPCCandidate(context.Background()); err == nil || again != nil || s.failed || !g.ready || s.stored != before+o.plan.physical {
			t.Fatal("completed RPC was reused or refunded")
		}
		previous = o
	}
	if s.sequence != 18 || g.ordinal != 6 || g.integration.ledger.continuation.page != 2 || len(g.integration.ledger.continuation.records) != 3 {
		t.Fatal("requests did not keep their exact page and ordinal boundaries")
	}
}

func TestProviderContinuingRPCDataDoesNotCreateAnExecutionRole(t *testing.T) {
	t.Parallel()
	if json.Unmarshal([]byte(`{}`), &providerContinuingRPCData{}) == nil {
		t.Fatal("JSON restored RPC result authority")
	}
	if o, err := newProviderContinuingRPCCandidate(nil, "eth_chainId", []any{}); err == nil || o != nil {
		t.Fatal("missing store admitted RPC")
	}
	if o, err := newProviderContinuingRPCCandidate(&providerCustodyStore{}, "eth_chainId", []any{}); err == nil || o != nil {
		t.Fatal("empty store DATA admitted an RPC")
	}
	if value, err := (*providerContinuingOwner)(nil).executeRPCCandidate(context.Background()); err == nil || value != nil {
		t.Fatal("missing owner executed RPC")
	}
	if value, ok := providerContinuingRPCPlan(nil, nil, ProviderCaptureLimits{}); ok || value != (providerContinuingPlan{}) {
		t.Fatal("missing owner produced RPC quote")
	}
	if _, ok := providerContinuingRPCMemoryUpper(nil, nil, ProviderCaptureLimits{}, providerFirstOperationStorage{}, 0, 0); ok {
		t.Fatal("missing owner produced RPC memory")
	}
	if value, err := providerRetainedMethodCapture(nil, nil, nil, nil); err == nil || value != (ProviderCaptureLimits{}) {
		t.Fatal("unretained request selected a class")
	}
	if (*providerContinuingOwner)(nil).bindRetainedExchangeLocked(nil, nil) == nil {
		t.Fatal("missing live R bound an exchange")
	}
}

func TestProviderRPCSizeArithmeticRefusesOverflow(t *testing.T) {
	t.Parallel()
	if providerRPCAddSizeRows(nil, nil) {
		t.Fatal("missing output accepted")
	}
	var work candidateWorkVector
	if providerRPCAddSizeRows(&work, make([]providerRPCSizeRow, 17)) {
		t.Fatal("unbounded row count accepted")
	}
	for _, row := range []providerRPCSizeRow{{^uint64(0), 1, 0}, {2, ^uint64(0), 0}, {2, 0, ^uint64(0)}} {
		var work candidateWorkVector
		if providerRPCAddSizeRows(&work, []providerRPCSizeRow{row}) {
			t.Fatal("size arithmetic wrapped", row)
		}
	}
	work[10] = ^uint64(0)
	if providerRPCAddSizeRows(&work, []providerRPCSizeRow{{1, 1, 1}}) {
		t.Fatal("work accumulation wrapped")
	}
	if _, ok := providerRPCClassExtraUpper(^uint64(0)); ok {
		t.Fatal("class quote wrapped")
	}
	normal, ok := providerRPCClassExtraUpper(65536)
	if !ok || normal[2] != 5*65536 || normal[10] != 5*(65536*65536+1)+11*(2*65536+101)+providerRPCClassRecursionUpper {
		t.Fatal("class rows lost an independent decode or recursion")
	}
}
