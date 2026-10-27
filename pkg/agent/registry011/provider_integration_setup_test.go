package registry011

import "testing"

// Finite data-only interleavings. No clock injection into a live executor,
// attacker process or mutable operating authority is exposed by these units.
func TestProviderIntegrationSetupBoundarySelectionAndLineage(t *testing.T) {
	clock := func(n int64) CandidateClockDiagnostic {
		return CandidateClockDiagnostic{"owned", "epoch", 1000 + n, n, 1}
	}
	anchor, consumed := admissionTestDigest("a"), admissionTestDigest("b")
	end0, end1 := clock(10), clock(20)
	positions := map[string]providerCustodyPosition{
		anchor:   {clock: clock(0), next: &end0},
		consumed: {sequence: 2, clock: end1},
	}
	base := []providerSetupReference{
		{commit: "first", expectedTip: anchor, ordinal: 0, clock: clock(0)},
		{commit: "second", expectedTip: anchor, ordinal: 1, clock: clock(5)},
		{commit: "reopen", expectedTip: consumed, ordinal: 2, clock: clock(25)},
	}
	s := &providerCustodyStore{config: providerCustodyConfig{expectedTip: consumed}, setups: base}
	selected, found := s.setupForBoundary(anchor)
	if !found || selected.ordinal != 1 || selected.commit != "second" || !s.checkSetupLineage(positions) {
		t.Fatal("later reopen changed E0's setup")
	}
	if _, found := s.setupForBoundary("unknown"); found {
		t.Fatal("invented a setup boundary")
	}
	for _, name := range []string{"unknown-tip", "regression", "late-at-old-tip", "before-predecessor", "wrong-ordinal", "selected-tip-mismatch", "empty"} {
		t.Run(name, func(t *testing.T) {
			copy := append([]providerSetupReference(nil), base...)
			fixture := &providerCustodyStore{config: providerCustodyConfig{expectedTip: consumed}, setups: copy}
			switch name {
			case "unknown-tip":
				copy[2].expectedTip = admissionTestDigest("c")
			case "regression":
				copy[0].expectedTip, copy[2].expectedTip = consumed, anchor
			case "late-at-old-tip":
				copy[1].clock = clock(30)
			case "before-predecessor":
				copy[2].clock = clock(15)
			case "wrong-ordinal":
				copy[1].ordinal = 0
			case "selected-tip-mismatch":
				fixture.config.expectedTip = anchor
			case "empty":
				fixture.setups = nil
			}
			if fixture.checkSetupLineage(positions) {
				t.Fatal("ambiguous setup interleaving accepted")
			}
		})
	}
}

func TestProviderIntegrationStoredClockIsExactAndBounded(t *testing.T) {
	valid := map[string]any{"id": "owned", "epoch": "epoch", "wall_ms": "1000", "monotonic_ms": "10", "uncertainty_ms": "1"}
	if clock, ok := providerStoredClock(valid); !ok || clock.WallMS != 1000 || clock.MonotonicMS != 10 {
		t.Fatal("valid stored clock")
	}
	for _, value := range []any{nil, map[string]any{}, "clock", map[string]any{"id": "owned", "epoch": "epoch", "wall_ms": "18446744073709551615", "monotonic_ms": "10", "uncertainty_ms": "1"}} {
		if _, ok := providerStoredClock(value); ok {
			t.Fatal("malformed stored clock")
		}
	}
}
