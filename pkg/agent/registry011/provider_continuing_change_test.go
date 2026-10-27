package registry011

import (
	"strconv"
	"testing"
)

// RPC-shaped values here are inert proposals. The live owner still has a
// different retained local T and cannot write, consume or dispatch them.
func TestProviderContinuingChangeReusesOnlyExactExistingRequestOriginals(t *testing.T) {
	t.Parallel()
	for _, reuse := range []bool{false, true} {
		t.Run(strconv.FormatBool(reuse), func(t *testing.T) {
			s, o := providerContinuingPreparedFixture(t)
			s.mu.Lock()
			defer s.mu.Unlock()
			defer o.failLocked()
			c, m := o.integration, o.execution
			method, params := "eth_chainId", []any{}
			if reuse {
				method, params = "eth_getBlockByNumber", []any{"finalized", false}
			}
			request, err := newProviderIntegrationRequestObserved(method, params, m)
			if err != nil {
				t.Fatal(err)
			}
			value := contentJSON([]byte(o.ticket.raw)).(map[string]any)
			value["sequence"], value["previous_ticket"] = strconv.FormatUint(o.ticket.sequence+1, 10), o.ticket.digest
			value["kind"], value["input_digest"], value["method"], value["params_digest"] = "rpc-capture", request.inputDigest, request.method, request.paramsDigest
			// This is a deliberately small inert DATA proposal, not a live
			// RPC execution quote. Do not copy the local E/R reservation.
			work := candidateWorkVector{1, 1000, 4096, 0, 0, 0, 0, 0, 0, 4096, 1 << 30}
			value["reservation"] = candidateLedgerWorkObject(work)
			ticket, err := parseCandidateAdmissionTicketObserved(c.policy, c.intent, o.ticket.segment, providerIntegrationJSON(value), 64<<10, m)
			if err != nil {
				t.Fatal("inert proposed ticket", err)
			}
			if !candidateFits(candidateBudgetTotals{}, ticket.work, c.ledger.limits[0]) ||
				!candidateFits(c.ledger.segments[ticket.segment], ticket.work, c.ledger.limits[1]) ||
				!candidateFits(c.ledger.current.totals.bootstrap, ticket.work, c.ledger.limits[2]) ||
				!candidateCapacity(c.ledger.current.totals.liveUsed, c.ledger.current.totals.liveReserved, ticket.live, c.ledger.live) ||
				!candidateCapacity(c.ledger.current.totals.retainedUsed, c.ledger.current.totals.retainedHeld, ticket.retained, c.ledger.retained) {
				t.Fatal("inert proposal does not fit the unchanged owned model's limits")
			}
			change, err := c.ledger.AppendCandidateTicket(ticket)
			if err != nil {
				t.Fatalf("inert proposed ledger: %v; valid=%v successor=%v segment=%+v bootstrap=%+v operationLimit=%v segmentLimit=%v bootstrapLimit=%v page=%d/%d", err, c.ledger.valid(), CheckCandidateTicketSuccessor(o.ticket, ticket), c.ledger.segments[ticket.segment], c.ledger.current.totals.bootstrap, c.ledger.limits[0], c.ledger.limits[1], c.ledger.limits[2], len(c.ledger.continuation.records), c.ledger.continuation.bounds.pageRecords)
			}
			x, err := newProviderScanCodec(s.config, m)
			if err != nil {
				t.Fatal(err)
			}
			binding := o.binding
			binding.ticket, binding.input, binding.kind, binding.purpose = ticket.digest, ticket.input, ticket.kind, "rpc-admission"
			binding.method, binding.params = ticket.method, ticket.params
			binding.target, err = x.guardianTicketTarget(binding.purpose, binding.admission, binding.kind, binding.input, binding.method, binding.params)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := x.guardianAdmissionBinding(binding)
			if err != nil {
				t.Fatal(err)
			}
			stored, sequence, pin := s.stored, s.sequence, s.frontierFiles.pin
			proposal, err := providerContinuingChangeData(s, c.root, c.ledger, o.indexes, s.frontierReplay.current, "admit", ticket.digest, change,
				[]providerIntegrationObjectInput{{"request", []byte(request.wire)}, {"params", []byte(request.params)}, {"guardian-admission-binding", raw}, {"ticket", []byte(ticket.raw)}}, s.frontierReplay.current.clock, m)
			if err != nil {
				t.Fatal("inert change proposal", err)
			}
			expected := 10
			if reuse {
				expected = 8
			}
			if len(proposal.fresh) != expected || s.stored != stored || s.sequence != sequence || s.frontierFiles.pin != pin ||
				o.writingLocked(s) || o.permit != nil || m.actual[0] != 0 || m.actual[1] != 0 || m.actual[9] != 0 {
				t.Fatal("proposal recreated an immutable original or obtained live authority")
			}
		})
	}
}

func TestProviderContinuingChangeRejectsMalformedUnretainedData(t *testing.T) {
	t.Parallel()
	s, o := providerContinuingPreparedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	defer o.failLocked()
	c, m := o.integration, o.execution
	clock, err := c.currentClock()
	if err != nil {
		t.Fatal(err)
	}
	consumption, err := providerContinuingConsumptionData(o.ticket, c.ledger.lastConsumption, providerClockDiagnostic(clock), m)
	if err != nil {
		t.Fatal(err)
	}
	change, err := c.ledger.AppendCandidateConsumption(consumption)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"store", "kind", "inputs", "sequence", "root-json", "root-format", "object", "operation", "node"} {
		t.Run(mode, func(t *testing.T) {
			selected, before := s, *c.root
			kind, operation := "consume", consumption.digest
			extra := []providerIntegrationObjectInput{{"consumption", []byte(consumption.raw)}}
			switch mode {
			case "store":
				selected = nil
			case "kind":
				kind = "unknown"
			case "inputs":
				extra = nil
			case "sequence":
				before.sequence = ^uint64(0)
			case "root-json":
				before.raw = "null"
			case "root-format":
				before.raw = `{"format":"unsupported-unit-format"}`
			case "object":
				extra[0].raw = []byte(`{"format":"unsupported-unit-format"}`)
			case "operation":
				operation = admissionTestDigest("f")
			case "node":
				nodes := change.nodes
				defer func() { change.nodes = nodes }()
				change.nodes = []string{`{"format":"unsupported-unit-format"}`}
			}
			if proposal, err := providerContinuingChangeData(selected, &before, c.ledger, o.indexes, s.frontierReplay.current, kind, operation, change, extra, providerClockDiagnostic(clock), m); err == nil || proposal != nil {
				t.Fatal("malformed unretained DATA became a change", mode)
			}
		})
	}
	if s.sequence != 5 || o.permit != nil || o.change != nil || m.actual[0] != 0 || m.actual[1] != 0 || m.actual[9] != 0 {
		t.Fatal("invalid DATA changed custody or execution")
	}
}
