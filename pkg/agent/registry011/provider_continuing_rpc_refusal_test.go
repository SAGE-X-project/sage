package registry011

import (
	"context"
	"testing"
)

func TestProviderContinuingRPCRefusesUnavailableAdmissionBeforeDispatch(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"request", "class", "prefix", "clock-capacity", "claim-original", "guardian-data", "segment-data", "main-original"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			s, g := providerContinuingRPCReadyFixture(t)
			stored, main, origin, ordinal := s.stored, s.frontierFiles.pin, s.setupOrigin, g.ordinal
			priorTicket, metadata := g.integration.ledger.lastTicket, g.integration.ledger.current.privateMetadata
			method, params := "eth_chainId", []any{}
			switch mode {
			case "request":
				method = "unsupported-unit-method"
			case "class":
				method = "debug_traceTransaction"
				params = []any{admissionTestDigest("a"), map[string]any{"tracer": "unsupported-unit-class"}}
			case "prefix":
				g.prefix.mapSizes[0]++
			case "clock-capacity":
				g.clock.mu.Lock()
				g.clock.maximum = g.clock.read + g.bounds.claim.clocks + 10
				g.clock.mu.Unlock()
			case "claim-original":
				if s.custody.root.Mkdir(s.name("guardian-intent", ordinal+1), 0700) != nil {
					t.Fatal("normal owned unavailable G path")
				}
			case "guardian-data":
				s.frontierReplay.body.guardian.terminal = false
			case "segment-data":
				g.integration.ledger.limits[1] = candidateWorkVector{}
			case "main-original":
				if s.custody.root.Mkdir(s.name("intent", s.sequence+1), 0700) != nil {
					t.Fatal("normal owned unavailable T path")
				}
			}
			o, err := newProviderContinuingRPCCandidate(s, method, params)
			if err == nil || o != nil || !g.failed || !s.failed || g.ready || g.ordinal != ordinal+1 || s.frontierFiles.pin != main || s.setupOrigin != origin ||
				s.stored < stored || s.last != nil || s.activeMeter.Load() != nil {
				t.Fatal("refused RPC restored an original hold or advanced main", mode, err)
			}
			if mode == "segment-data" || mode == "main-original" {
				if g.pin.ordinal != ordinal+1 || g.integration.ledger.current.privateMetadata <= metadata {
					t.Fatal("later refusal was masked by an earlier empty-page metadata failure", mode)
				}
			}
			if mode == "main-original" && (g.integration.ledger.lastTicket == priorTicket || g.integration.ledger.lastTicket.sequence != priorTicket.sequence+1) {
				t.Fatal("unavailable main original was not reached after the next ticket append")
			}
		})
	}
}

func TestProviderContinuingRPCRefusesCancelledExecutionWithoutARequest(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"nil-context", "cancelled", "ticket-data", "method-profile", "tls-profile", "unavailable-C"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			s, g := providerContinuingRPCReadyFixture(t)
			o, err := newProviderContinuingRPCCandidate(s, "eth_chainId", []any{})
			if err != nil {
				t.Fatal("normal admitted RPC", err)
			}
			stored, main, origin := s.stored, s.frontierFiles.pin, s.setupOrigin
			ctx := context.Background()
			switch mode {
			case "nil-context":
				ctx = nil
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "ticket-data":
				o.ticket = nil
			case "method-profile":
				// Finite in-memory profile disagreement. No original is rewritten.
				o.config.bounds.methods.scalar.body--
			case "tls-profile":
				o.config.bounds.tls.certificates--
			case "unavailable-C":
				if s.custody.root.Mkdir(s.name("intent", s.sequence+1), 0700) != nil {
					t.Fatal("normal owned unavailable C path")
				}
			}
			data, err := o.executeRPCCandidate(ctx)
			if err == nil || data != nil || !g.failed || !s.failed || o.state != providerContinuingFailed || s.frontierFiles.pin != main || s.stored != stored ||
				s.setupOrigin != origin || s.activeMeter.Load() != nil || o.execution.actual[0] != 0 || o.execution.actual[1] != 0 {
				t.Fatal("cancelled or unavailable RPC dispatched or refunded", mode, err)
			}
		})
	}
}
