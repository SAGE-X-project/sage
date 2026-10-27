package registry011

import "testing"

func TestProviderContinuingDataRecordsPreserveGlobalSequenceAndSegment(t *testing.T) {
	s, g := providerContinuingSettledFixture(t, true)
	s.mu.Lock()
	defer s.mu.Unlock()
	if g.startClaimLocked("bootstrap-admission") != nil || g.retainClaimLocked(admissionTestDigest("a")) != nil {
		t.Fatal("fresh finite DATA construction allowance")
	}
	defer g.failLocked()
	plan, ok := providerContinuingBootstrapPlan(g)
	if !ok {
		t.Fatal("closed prospective local plan")
	}
	c, m := g.integration, g.meter
	clock, err := g.sampleLocked()
	if err != nil {
		t.Fatal(err)
	}
	plan.reservation.execution[8], plan.reservation.work[8] = 0, 0
	local, err := providerContinuingTicketData(c.policy, c.intent, c.ledger.lastTicket, 0, "local-work", c.firstExchange, "", "", plan.reservation, providerClockDiagnostic(clock), m)
	if err != nil || local == nil || local.sequence != 1 || local.previous != c.ledger.lastTicket.digest || local.segment != 0 || local.work[8] != 0 || local.input != c.firstExchange {
		t.Fatal("next DATA ticket changed its original predecessor or segment", err)
	}
	for _, mode := range []string{"policy", "intent", "previous", "sequence", "segment", "input", "work", "live", "remaining", "kind", "local-method", "local-params", "policy-shape", "clock", "meter"} {
		t.Run(mode, func(t *testing.T) {
			policy, intent, previous := c.policy, c.intent, c.ledger.lastTicket
			prior := *previous
			quote, sample, observed := plan.reservation, providerClockDiagnostic(clock), m
			segment, kind, input, method, params := uint64(0), "local-work", c.firstExchange, "", ""
			switch mode {
			case "policy":
				policy = nil
			case "intent":
				intent = nil
			case "previous":
				previous = nil
			case "sequence":
				prior.sequence, previous = ^uint64(0), &prior
			case "segment":
				segment = 2
			case "input":
				input = "missing"
			case "work":
				quote.work[10]++
			case "live":
				quote.live = 0
			case "remaining":
				quote.remaining[0], quote.work[0] = 1, 1
			case "kind":
				kind = "unsupported-unit-command"
			case "local-method":
				method = "eth_chainId"
			case "local-params":
				params = admissionTestDigest("a")
			case "policy-shape":
				copy := *policy
				copy.raw, policy = "null", &copy
			case "clock":
				sample.Epoch = "unselected-unit-epoch"
			case "meter":
				observed = nil
			}
			if candidate, err := providerContinuingTicketData(policy, intent, previous, segment, kind, input, method, params, quote, sample, observed); err == nil || candidate != nil {
				t.Fatal("invalid DATA supplied a next ticket", mode)
			}
		})
	}
	rpc := plan.reservation
	rpc.execution[0], rpc.execution[1], rpc.work[0], rpc.work[1] = 1, 1000, 1, 1000
	if candidate, err := providerContinuingTicketData(c.policy, c.intent, c.ledger.lastTicket, 0, "rpc-capture", admissionTestDigest("a"), "eth_chainId", admissionTestDigest("b"), rpc, providerClockDiagnostic(clock), m); err != nil || candidate.input != admissionTestDigest("a") || candidate.method != "eth_chainId" || candidate.work[8] != 0 {
		t.Fatal("pure RPC DATA rewrote exact wire input", err)
	}
	if candidate, err := providerContinuingTicketData(c.policy, c.intent, c.ledger.lastTicket, 0, "rpc-capture", admissionTestDigest("a"), "eth_chainId", "missing", rpc, providerClockDiagnostic(clock), m); err == nil || candidate != nil {
		t.Fatal("missing RPC params accepted as DATA")
	}
	for _, mode := range []string{"valid", "ticket", "first", "previous", "sequence", "clock", "meter", "closed"} {
		t.Run("consumption-"+mode, func(t *testing.T) {
			// An unowned finite DATA observer is deliberately not attached to
			// the store and cannot commit C or issue an execution receipt.
			observed := newProviderWorkMeter(c, providerOperationReservation{execution: candidateWorkVector{2: 64 << 10, 10: 1 << 40}})
			ticket, previous, sample := local, c.ledger.lastConsumption, providerClockDiagnostic(clock)
			copy := *previous
			switch mode {
			case "ticket":
				ticket = nil
			case "first":
				ticket = c.firstTicket
			case "previous":
				previous = nil
			case "sequence":
				copy.sequence, previous = ^uint64(0), &copy
			case "clock":
				sample.Epoch = "unselected-unit-epoch"
			case "meter":
				observed = nil
			case "closed":
				observed.closed = true
			}
			consumed, err := providerContinuingConsumptionData(ticket, previous, sample, observed)
			if mode == "valid" {
				if err != nil || consumed.sequence != previous.sequence+1 || consumed.previous != previous.digest || consumed.ticket != local {
					t.Fatal("global consumption chain reset", err)
				}
			} else if err == nil || consumed != nil {
				t.Fatal("invalid consumption DATA accepted", mode)
			}
		})
	}
	if s.sequence != 4 || s.last != nil || g.ordinal != 1 || !g.active || g.ready || c.ledger.lastTicket.sequence != 0 {
		t.Fatal("formatting DATA created a retained T/C or dispatch authority")
	}
}

func TestProviderContinuingChangeInputsAreClosed(t *testing.T) {
	for _, inputs := range []struct {
		kind  string
		names []string
	}{
		{"admit", []string{"guardian-admission-binding", "ticket"}},
		{"admit", []string{"request", "params", "guardian-admission-binding", "ticket"}},
		{"consume", []string{"consumption"}},
		{"settle", []string{"settlement"}},
		{"settle", []string{"bootstrap", "settlement"}},
		{"settle", []string{"exchange", "response", "tls", "wire-binding", "settlement"}},
	} {
		values := make([]providerIntegrationObjectInput, len(inputs.names))
		for i, name := range inputs.names {
			values[i].kind = name
		}
		if !providerContinuingChangeInputs(inputs.kind, values) || providerContinuingChangeInputs("unknown", values) || providerContinuingChangeInputs(inputs.kind, nil) {
			t.Fatal("closed input list changed", inputs.kind)
		}
		values[len(values)-1].kind = "unsupported-unit-kind"
		if providerContinuingChangeInputs(inputs.kind, values) {
			t.Fatal("changed operation input kind accepted")
		}
	}
}
