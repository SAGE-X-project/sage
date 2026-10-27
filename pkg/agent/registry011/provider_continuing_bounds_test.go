package registry011

import "testing"

func TestProviderContinuingProspectiveBoundsAreDataAndRequireEveryOriginalDimension(t *testing.T) {
	s, g := providerContinuingSettledFixture(t, true)
	s.mu.Lock()
	defer s.mu.Unlock()
	if g.startClaimLocked("bootstrap-admission") != nil {
		t.Fatal("fresh original metadata allowance")
	}
	defer g.failLocked()
	plan, ok := providerContinuingBootstrapPlan(g)
	if !ok || plan.physical != plan.consumption+plan.settlement || plan.memory == 0 || plan.headroom == 0 ||
		plan.reservation.work[0] != 0 || plan.reservation.work[1] != 0 || plan.reservation.work[3] != 1 || plan.reservation.work[8] != 1 ||
		plan.metadataWork[2] != 744*max(s.config.bounds.maximumMetadataPerCommit, uint64(64<<10)) ||
		!providerContinuingWorkFits(plan.metadataWork, g.bounds.claim.work) {
		t.Fatal("prospective quote omitted an original dimension")
	}
	for coordinate, amount := range plan.metadataWork {
		if amount == 0 {
			continue
		}
		short := plan.metadataWork
		short[coordinate]--
		if providerContinuingWorkFits(plan.metadataWork, short) {
			t.Fatal("one-short metadata coordinate admitted the quote", coordinate)
		}
	}
	for _, mode := range []string{"nil", "failed", "first", "limits", "replay", "ledger", "memory"} {
		t.Run(mode, func(t *testing.T) {
			selected := g
			copy := *g
			copy.owner = &copy
			// A copied owner must already refuse; missing nested inputs below
			// additionally exercise the scalar memory quote's inert guards.
			switch mode {
			case "nil":
				selected = nil
			case "failed":
				copy.failed, selected = true, &copy
			case "first":
				copy.first, selected = nil, &copy
			case "limits":
				copy.store, selected = nil, &copy
			case "replay":
				copy.integration, selected = nil, &copy
			case "ledger":
				copy.config.bounds.continuation, selected = nil, &copy
			case "memory":
				copy.config.bounds.scan, selected = nil, &copy
			}
			if _, ok := providerContinuingBootstrapPlan(selected); ok {
				t.Fatal("copied or incomplete DATA supplied a live quote", mode)
			}
			if mode != "failed" && mode != "first" {
				if _, ok := providerContinuingMemoryUpper(selected, providerFirstOperationStorage{}, 0, 0); ok {
					t.Fatal("missing memory prerequisites accepted", mode)
				}
			}
		})
	}
}

func TestProviderContinuingPrefixDataDoesNotBecomeASelection(t *testing.T) {
	s, o := providerContinuingPreparedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	if !o.guardian.continuingPrefixMatchesLocked() {
		t.Fatal("actual typed T did not update its same-live marker")
	}
	for _, mode := range []string{"nil", "foreign", "stored", "budget", "page"} {
		t.Run(mode, func(t *testing.T) {
			g := o.guardian
			before := g.prefix
			defer func() { o.guardian.prefix = before }()
			switch mode {
			case "nil":
				g = nil
			case "foreign":
				g.prefix.owner = &providerGuardianOwner{}
			case "stored":
				g.prefix.stored++
			case "budget":
				g.prefix.budget.sequence++
			case "page":
				g.prefix.page.records++
			}
			if g.continuingPrefixMatchesLocked() {
				t.Fatal("changed private marker retained permission", mode)
			}
		})
	}
	if _, ok := providerContinuingPrefixData(nil); ok {
		t.Fatal("missing actor supplied prefix DATA")
	}
	if _, ok := providerContinuingPageData(nil); ok {
		t.Fatal("missing page supplied DATA")
	}
	if _, ok := providerContinuingPageData(&providerLedgerContinuation{final: true}); ok {
		t.Fatal("closed page supplied an active prefix")
	}
	if (&providerContinuingPrefix{}).UnmarshalJSON(nil) == nil || (&providerGuardianReplay{}).UnmarshalJSON(nil) == nil {
		t.Fatal("serialized prefix DATA restored a live role")
	}
}

func TestProviderGuardianOriginalReplayStateHasNoFallback(t *testing.T) {
	var missing *providerCustodyReplay
	if missing.guardianRawBytes() != 0 || missing.guardianConsumption(nil) || missing.guardianSettlement(nil) ||
		missing.guardianFirstTicket(nil, nil, nil) == nil || missing.guardianTicket(nil, nil) == nil {
		t.Fatal("missing replay supplied guardian state")
	}
	r := &providerCustodyReplay{}
	if r.guardianRawBytes() != 0 || r.guardianConsumption(nil) || r.guardianSettlement(nil) {
		t.Fatal("missing original G fell back to DATA")
	}
	r.guardian = &providerGuardianReplay{metadata: 123}
	if r.guardianRawBytes() != 123 || r.guardianConsumption(nil) || r.guardianSettlement(nil) {
		t.Fatal("raw metadata count became a consumption or settlement")
	}
	s, o := providerContinuingPreparedFixture(t)
	s.mu.Lock()
	defer s.mu.Unlock()
	reader := &providerScanReplayReader{view: s, replay: s.frontierReplay, meter: o.execution}
	if !reader.guardianFinalLineage() {
		t.Fatal("original latest G HEAD disagreed with retained typed T")
	}
	reader.meter = nil
	if reader.guardianFinalLineage() {
		t.Fatal("unobserved final lineage accepted")
	}
	if (&providerScanReplayReader{}).guardianFinalLineage() || (&providerScanReplayReader{}).guardianInitialLineage() {
		t.Fatal("missing original reader supplied lineage")
	}
}
