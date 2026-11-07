package registry011

import (
	"reflect"
	"strconv"
	"testing"
)

func TestProviderHistoryFirstOriginalProspectiveBudgetsKeepFailuresAndResultsSeparate(t *testing.T) {
	derived, ok := providerHistoryFirstPReadUpper(historyFirstOriginalCompositionLimits(), historyCollectionInput(t).capture)
	if !ok || !derived.valid() {
		t.Fatal("ordinary independently selected UNIT reader row required")
	}
	budgets := []uint64{1, 32, 256, 1024, 4096, 16384, 65536, 262144, 1048576, 4194304, derived.work[10]}
	failed, succeeded := 0, 0
	for index, requested := range budgets {
		budget := min(requested, derived.work[10])
		t.Run(strconv.Itoa(index)+"-"+strconv.FormatUint(budget, 10), func(t *testing.T) {
			read := derived
			// Each independent profile is selected before H0 in a fresh owned UNIT
			// fixture. No failed owner, actual prefix or live quota is modified.
			read.work[10] = budget
			s := historyCollectionAdmissionStore(t, true, &read)
			config, bounds, held, pin, clocks := s.config, s.history.bounds, s.stored, s.frontierFiles.pin, s.clock.read
			originals := providerHistoryProvisionedOriginalBytes(t, config)
			out, err := newProviderHistoryGroupOwnerCandidate(s)
			g, r := s.history.group, s.history.recovery
			if g == nil || r == nil || g.firstPReading || g.firstPStart[10] == 0 || g.meter == nil ||
				g.meter.actual[10] < g.firstPStart[10] || g.meter.actual[0] != 0 || g.meter.actual[1] != 0 ||
				g.meter.upper != bounds.future.work || g.meter.maximumClocks != bounds.future.clocks ||
				s.config != config || s.history.bounds != bounds || s.stored != held || s.frontierFiles.pin != pin ||
				s.activeMeter.Load() != nil || !reflect.DeepEqual(originals, providerHistoryProvisionedOriginalBytes(t, config)) {
				t.Fatal("prospective reader limit changed original inputs, quota or retained selection", budget, err)
			}
			s.clock.mu.Lock()
			refusal := s.clock.nativeRefusal
			s.clock.mu.Unlock()
			if refusal != 0 {
				t.Fatal("native clock refusal cannot stand in for the bounded-reader scenario", budget, refusal)
			}
			if err == nil {
				succeeded++
				if out != g || g.failed || g.meter.failed || g.meter.closed || s.failed || s.history.failed ||
					g.coordinator == nil || g.custodian == nil || g.coordinator == g.custodian || g.coordinator.head != g.custodian.head ||
					r.initial == nil || r.confirmed != r.initial || r.pending != nil || s.clock.read != clocks+2 {
					t.Fatal("completed reader did not preserve independent initial models and the original bracket", budget)
				}
			} else {
				failed++
				if out != nil || !g.failed || !g.meter.failed || !g.meter.closed || !s.failed || !s.history.failed ||
					g.last != nil || r.initial != nil || r.confirmed != nil || r.pending != nil || s.clock.read != clocks+1 {
					t.Fatal("refused reader published partial models or renewed its original execution", budget, err)
				}
				work, reads, upper := g.meter.actual, s.clock.read, g.meter.upper
				if again, err := newProviderHistoryGroupOwnerCandidate(s); err == nil || again != nil || s.history.group != g ||
					g.meter.actual != work || g.meter.upper != upper || s.clock.read != reads || s.stored != held {
					t.Fatal("refused reader was retried or given another allowance", budget)
				}
			}
		})
	}
	if failed == 0 || succeeded == 0 {
		t.Fatal("matrix must include actual refusal and an ordinary complete control", failed, succeeded)
	}
}
