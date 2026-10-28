package registry011

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// Resource proposals are finite DATA. These checks never admit or execute
// the changed proposal, replace an original, or restore a closed owner.
func TestProviderContinuingRPCPlanRejectsIncompleteProspectiveData(t *testing.T) {
	t.Parallel()
	s, g := providerContinuingRPCReadyFixture(t)
	request, err := newProviderIntegrationRequest("eth_chainId", []any{})
	if err != nil {
		t.Fatal(err)
	}
	capture, err := s.config.bounds.methods.capture(request, s.config.capture)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := providerContinuingRPCPlan(g, request, capture); !ok {
		t.Fatal("normal prospective DATA")
	}
	before := s.frontierFiles.pin
	page := g.integration.ledger.continuation
	pending := page.pending
	page.pending = &providerLedgerBoundary{}
	_, ok := providerContinuingRPCPlan(g, request, capture)
	page.pending = pending
	if ok {
		t.Fatal("pending page produced a request quote")
	}
	large := *request
	large.wire = strings.Repeat("x", (16<<10)+1)
	if _, ok := providerContinuingRPCPlan(g, &large, capture); ok {
		t.Fatal("unbounded request DATA produced a quote")
	}
	limit := s.firstLimits.operation[1]
	s.firstLimits.operation[1] = 1
	_, ok = providerContinuingRPCPlan(g, request, capture)
	s.firstLimits.operation[1] = limit
	if ok {
		t.Fatal("unfunded operation produced a quote")
	}
	priorPeak := g.first.plan.memory
	g.first.plan.memory = ^uint64(0)
	_, ok = providerContinuingRPCPlan(g, request, capture)
	g.first.plan.memory = priorPeak
	if ok || s.frontierFiles.pin != before || s.activeMeter.Load() != nil || g.active || !g.ready {
		t.Fatal("resident overflow created a request or changed original state")
	}
}

func TestProviderContinuingRPCBurnRefusesDifferentRequestData(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"previous-ticket", "request-params", "capture-class"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			s, _ := providerContinuingRPCReadyFixture(t)
			o, err := newProviderContinuingRPCCandidate(s, "eth_chainId", []any{})
			if err != nil {
				t.Fatal(err)
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			defer o.failLocked()
			permit, err := o.consumeLocked()
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "previous-ticket":
				o.previousTicket = nil
			case "request-params":
				o.request.paramsDigest = admissionTestDigest("f")
			case "capture-class":
				o.capture.BodyBytes--
			}
			if o.burnLocked(permit) || !permit.used || o.burnLocked(permit) || o.execution.actual[0] != 0 || o.execution.actual[1] != 0 || s.sequence != 10 {
				t.Fatal("post-consumption refusal dispatched or restored a permit", mode)
			}
		})
	}
}

func TestProviderContinuingRPCCannotSettleWithoutConsumptionClockAndOriginal(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"missing-consumption", "unavailable-clock", "unavailable-settlement"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			s, _ := providerContinuingRPCReadyFixture(t)
			o, err := newProviderContinuingRPCCandidate(s, "eth_chainId", []any{})
			if err != nil {
				t.Fatal(err)
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			defer o.failLocked()
			if mode != "missing-consumption" {
				permit, err := o.consumeLocked()
				if err != nil || !o.burnLocked(permit) {
					t.Fatal("normal consumed command", err)
				}
			}
			main, stored := s.frontierFiles.pin, s.stored
			if mode == "unavailable-clock" {
				s.clock.failed = true
			}
			if mode == "unavailable-settlement" && s.custody.root.Mkdir(s.name("intent", s.sequence+1), 0700) != nil {
				t.Fatal("owned unavailable destination")
			}
			success := false
			data, err := o.finishRPCLocked(o.permit, nil, errProviderIntegration, &success)
			if err == nil || data != nil || success || s.frontierFiles.pin != main || s.stored != stored || o.execution.actual[1] != 0 {
				t.Fatal("incomplete settlement returned usable DATA or refunded a hold", mode)
			}
		})
	}
}

// Stop at the real writer's allocation boundary, before it has retained the
// proposed S originals. A proposed binding must not become a retained index.
// All mutated inputs below remain in memory; no changed original is written.
func TestProviderContinuingRPCBindingNeedsItsNewRetainedOriginal(t *testing.T) {
	t.Parallel()
	s, _ := providerContinuingRPCReadyFixture(t)
	o, err := newProviderContinuingRPCCandidate(s, "eth_chainId", []any{})
	if err != nil {
		t.Fatal(err)
	}
	connection := providerIntegrationOwnedTLS(t, o.request.body, `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`)
	defer func() { _ = connection.Close() }()
	s.mu.Lock()
	defer s.mu.Unlock()
	defer o.failLocked()
	permit, err := o.consumeLocked()
	if err != nil || !o.burnLocked(permit) {
		t.Fatal("normal private consumption", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), o.capture.Timeout)
	defer cancel()
	sample, err := providerFirstOperationCapture(ctx, connection, o.capture, o.request, o.execution, *o.config.bounds.tls)
	if err != nil {
		t.Fatal("ordinary owned TLS response", err)
	}
	c := o.integration
	end, err := c.currentClock()
	if err != nil {
		t.Fatal(err)
	}
	extra, retained, err := providerOperationExchangeData(c, permit, sample, end, o.execution, o.capture)
	if err != nil {
		t.Fatal(err)
	}
	end, err = c.currentClock()
	if err != nil {
		t.Fatal(err)
	}
	measurement, err := o.execution.finish(c, permit.ticket, end)
	if err != nil || !o.beginRemainingLocked(permit) {
		t.Fatal("distinct remaining hold", err)
	}
	settlement, err := providerFirstOperationSettlementData(c, o.ticket, "complete", retained, end, measurement, o.remaining)
	if err != nil {
		t.Fatal(err)
	}
	change, err := c.ledger.AppendCandidateSettlement(settlement)
	if err != nil {
		t.Fatal(err)
	}
	extra = append(extra, providerIntegrationObjectInput{"settlement", providerReplayPhaseBytes(settlement.raw, o.remaining)})
	p, err := providerContinuingChangeData(s, c.root, c.ledger, o.indexes, s.frontierReplay.current, "settle", settlement.digest, change, extra, providerClockDiagnostic(end), o.remaining)
	if err != nil {
		t.Fatal(err)
	}
	o.change = p
	base, err := providerFrontierRetentionCostObserved(s.config.identity, p.inventory, p.fresh, p.intent, p.commit, p.frontier, s.config.bounds, o.remaining)
	if err != nil {
		t.Fatal(err)
	}
	head, err := p.after.bytesObserved(s.config, o.remaining)
	if err != nil {
		t.Fatal(err)
	}
	headCost, ok := providerHighwaterFileCost(uint64(len(head)), s.config.bounds)
	total, exact := providerUpperSum(base.stored, headCost.stored)
	if !ok || !exact || !o.consumeStorageLocked(total) || !o.settlement {
		t.Fatal("exact pre-write allocation")
	}
	main, stored, count := s.frontierFiles.pin, s.stored, len(s.bindings)
	for _, mode := range []string{"missing-reference", "duplicate-exchange", "unretained-original"} {
		fresh := append([]providerIntegrationObject(nil), p.fresh...)
		for i := range fresh {
			if mode == "missing-reference" && fresh[i].kind == "wire-binding" {
				fresh[i].kind = "finite-missing-binding"
			}
			if mode == "duplicate-exchange" && fresh[i].kind == "exchange" {
				fresh[i].digest = c.firstExchange
			}
		}
		if o.bindRetainedExchangeLocked(s, fresh) == nil || s.frontierFiles.pin != main || s.stored != stored || len(s.bindings) != count {
			t.Fatal("proposed binding replaced original retention", mode)
		}
	}
}

func TestProviderFrontierWriteClaimDataCannotRestoreAnOwner(t *testing.T) {
	t.Parallel()
	if json.Unmarshal([]byte(`{}`), &providerFrontierWriteClaim{}) == nil {
		t.Fatal("JSON restored a write claim")
	}
	var replay *providerFrontierReplay
	if _, err := replay.retainClosedOwnedLocked(nil, nil, nil, nil, nil, nil, providerFrontierHighwater{}, nil, nil, nil, nil, nil); err == nil {
		t.Fatal("missing store supplied custody")
	}
}
