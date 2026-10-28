package registry011

import (
	"context"
	"encoding/json"
	"testing"
)

func TestProviderContinuingPinExecutionRefusesUnavailableOwnedResources(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"nil-context", "C-path", "selected-input", "different-network", "end-clock", "S-path", "release-clock"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			selection := &providerScanFixtureSelection{pin: true, pinResidentGiB: true}
			var s *providerCustodyStore
			var g *providerGuardianOwner
			digests := [4]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("d")}
			if mode == "nil-context" || mode == "C-path" || mode == "selected-input" {
				s, g = providerContinuingRPCReadyWithSelectionFixture(t, selection)
			} else {
				results := providerPinResultFixture(t)
				if mode == "different-network" {
					// An ordinary Mainnet chain ID from the existing owned unit
					// fixture must not identify this Sepolia Registry. No new
					// peer, callback, credential or external RPC is supplied.
					results[0] = json.RawMessage(`"0x1"`)
				}
				s, g, digests = providerContinuingPinReadyWithResultsFixture(t, selection, results)
			}
			o, err := newProviderContinuingPinCandidate(s, "example-agent", digests)
			if err != nil {
				t.Fatal("original funded local admission", err)
			}
			stored, origin := s.stored, s.setupOrigin
			ctx := context.Background()
			switch mode {
			case "nil-context":
				ctx = nil
			case "C-path":
				if err := s.custody.root.Mkdir(s.name("intent", s.sequence+1), 0700); err != nil {
					t.Fatal("owned unavailable C path", err)
				}
			case "selected-input":
				o.pin.selected = admissionTestDigest("b") // Finite in-memory input disagreement only.
			case "S-path":
				if err := s.custody.root.Mkdir(s.name("intent", s.sequence+2), 0700); err != nil {
					t.Fatal("owned unavailable S path", err)
				}
			case "end-clock", "release-clock":
				remaining := uint64(5)
				if mode == "release-clock" {
					remaining = 9
				}
				o.clock.mu.Lock()
				o.clock.maximum = o.clock.read + remaining
				o.clock.mu.Unlock()
			}
			data, err := o.executePinCandidate(ctx)
			if err == nil || data != nil || !s.failed || !g.failed || o.state != providerContinuingFailed || s.stored != stored || s.setupOrigin != origin || g.integration.pin != "" || s.activeMeter.Load() != nil || o.execution.actual[0] != 0 || o.execution.actual[1] != 0 {
				t.Fatalf("unavailable local command escaped or refunded: %s %v state=%d clocks=%d E=%+v R=%+v", mode, err, o.state, o.clocks, o.execution, o.remaining)
			}
			if mode == "selected-input" || mode == "different-network" || mode == "release-clock" {
				if !o.consumption || !o.settlement || o.permit == nil || !o.permit.used || s.sequence != o.before.sequence+3 {
					t.Fatal("later refusal did not retain its own C/S boundary", mode, o.clocks, s.sequence)
				}
			}
			if mode == "end-clock" && (!o.consumption || o.settlement || o.permit == nil || !o.permit.used || o.remaining != nil || o.clocks != 6 || s.sequence != o.before.sequence+2) {
				t.Fatal("clock refusal did not reach the E end after C and burn", o.clocks, s.sequence)
			}
			if mode == "release-clock" && (o.clocks != 10 || o.remaining == nil) {
				t.Fatal("clock refusal did not reach the post-S final sample", o.clocks)
			}
			if value, err := o.executePinCandidate(context.Background()); err == nil || value != nil || s.stored != stored {
				t.Fatal("failed command restarted or restored a physical hold", mode)
			}
		})
	}
}
