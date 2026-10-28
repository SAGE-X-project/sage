package registry011

import (
	"context"
	"testing"
)

// The unrelated larger object ceiling is chosen before all original setup.
// Four class-sized original closures still repeat under independent E and R.
func TestProviderContinuingPinClassesRemainBoundedWithLargerObjectCeiling(t *testing.T) {
	t.Parallel()
	s, g, digests := providerContinuingPinReadyWithSelectionFixture(t, &providerScanFixtureSelection{pin: true, pinResidentGiB: true, pinLargeObject: true})
	if s.config.bounds.maximumObjectBytes != 1<<20 {
		t.Fatal("larger unrelated object input was not selected before setup")
	}
	origin, bootstrap, first := s.setupOrigin, g.integration.bootstrap, g.integration.firstExchange
	proposal := providerContinuingPinQuoteFixture(t, g, admissionTestDigest("a"))
	if proposal["resource_ok"] != true || proposal["memory_ok"] != true || len(proposal["refused_coordinates"].([]int)) != 0 {
		t.Fatalf("class-specific prospective quote refused: %+v", proposal)
	}
	o, err := newProviderContinuingPinCandidate(s, "example-agent", digests)
	if err != nil {
		t.Fatal("original class-bounded local admission", err)
	}
	data, err := o.executePinCandidate(context.Background())
	if err != nil || data == nil || data.exchanges != digests || data.pin != g.integration.pin || s.sequence != 25 || g.ordinal != 9 || g.integration.bootstrap != bootstrap || g.integration.firstExchange != first || s.setupOrigin != origin || g.original != origin || !g.ready || g.active || s.activeMeter.Load() != nil {
		t.Fatalf("larger unrelated object ceiling replaced an original child closure: %v E=%+v R=%+v", err, o.execution, o.remaining)
	}
	for i, value := range data.execution {
		if value > o.plan.reservation.execution[i] || data.remaining[i] > o.plan.reservation.remaining[i] {
			t.Fatal("independent class-sized original reads exceeded a phase hold", i)
		}
	}
}
