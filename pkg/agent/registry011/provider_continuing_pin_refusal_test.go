package registry011

import "testing"

func TestProviderContinuingPinUnavailableAdmissionNeverDispatches(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"prefix", "bootstrap-reader", "clock-capacity", "claim-original", "guardian-data", "segment-data", "main-original", "operation-data"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			s, g := providerContinuingRPCReadyWithSelectionFixture(t, &providerScanFixtureSelection{pin: true, pinResidentGiB: true})
			stored, main, origin, ordinal := s.stored, s.frontierFiles.pin, s.setupOrigin, g.ordinal
			priorTicket, metadata := g.integration.ledger.lastTicket, g.integration.ledger.current.privateMetadata
			digests := [4]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("d")}
			switch mode {
			case "prefix":
				g.prefix.mapSizes[0]++
			case "bootstrap-reader":
				if err := s.evidence.close(); err != nil {
					t.Fatal("normal owned evidence reader close", err)
				}
			case "clock-capacity":
				g.clock.mu.Lock()
				g.clock.maximum = g.clock.read + g.bounds.claim.clocks + 10
				g.clock.mu.Unlock()
			case "claim-original":
				if err := s.custody.root.Mkdir(s.name("guardian-intent", ordinal+1), 0700); err != nil {
					t.Fatal("owned unavailable new G path", err)
				}
			case "guardian-data":
				s.frontierReplay.body.guardian.terminal = false
			case "segment-data":
				g.integration.ledger.limits[1] = candidateWorkVector{}
			case "main-original":
				if err := s.custody.root.Mkdir(s.name("intent", s.sequence+1), 0700); err != nil {
					t.Fatal("owned unavailable new T path", err)
				}
			case "operation-data":
				s.firstLimits.operation[2] = 1
			}
			o, err := newProviderContinuingPinCandidate(s, "example-agent", digests)
			if err == nil || o != nil || !g.failed || !s.failed || g.ready || g.ordinal != ordinal+1 || s.frontierFiles.pin != main || s.setupOrigin != origin || s.stored < stored || s.last != nil || s.activeMeter.Load() != nil || g.integration.pin != "" {
				t.Fatal("refused admission dispatched, advanced main or restored a hold", mode, err)
			}
			if mode == "segment-data" || mode == "main-original" {
				if g.pin.ordinal != ordinal+1 || g.integration.ledger.current.privateMetadata <= metadata {
					t.Fatal("later refusal did not retain its own G boundary", mode)
				}
			}
			if mode == "main-original" && (g.integration.ledger.lastTicket == priorTicket || g.integration.ledger.lastTicket.sequence != priorTicket.sequence+1) {
				t.Fatal("unavailable main original was not reached after the next ticket append")
			}
		})
	}
}

func TestProviderContinuingPinCannotReplaceAnActiveOriginalClaim(t *testing.T) {
	t.Parallel()
	s, g := providerContinuingRPCReadyWithSelectionFixture(t, &providerScanFixtureSelection{pin: true, pinResidentGiB: true})
	o, err := newProviderContinuingRPCCandidate(s, "eth_chainId", []any{})
	if err != nil {
		t.Fatal("original RPC admission", err)
	}
	stored, main, ordinal, observer := s.stored, s.frontierFiles.pin, g.ordinal, s.activeMeter.Load()
	digests := [4]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("d")}
	if pin, err := newProviderContinuingPinCandidate(s, "example-agent", digests); err == nil || pin != nil || s.failed || g.failed || s.stored != stored || s.frontierFiles.pin != main || g.ordinal != ordinal || s.activeMeter.Load() != observer || o.state != providerContinuingPrepared {
		t.Fatal("active original request was replaced by local admission")
	}
}
