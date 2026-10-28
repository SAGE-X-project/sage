package registry011

import "testing"

// Quote DATA is collected BEFORE any new G/T so refusal diagnostics cannot
// reopen a failed role or reattach an earlier observer. Nothing is dispatched.
func providerContinuingPinQuoteFixture(t *testing.T, g *providerGuardianOwner, selected string) map[string]any {
	t.Helper()
	s, b := g.store, g.config.bounds
	before, ordinal, tip, observer := s.stored, g.ordinal, s.frontierFiles.pin, s.activeMeter.Load()
	clock := s.clock
	clock.mu.Lock()
	clockRead, clockMaximum := clock.read, clock.maximum
	clockIdentityOK := clock.owner == clock && !clock.failed
	clock.mu.Unlock()

	requests, err := providerPinRequests(selected)
	if err != nil {
		t.Fatal(err)
	}
	command := &providerContinuingPinCommand{requests: requests}
	for i, request := range requests {
		command.captures[i], err = b.methods.capture(request, g.config.capture)
		if err != nil {
			t.Fatal(err)
		}
	}
	plan, storage, maximum, sizes, resourceOK := providerContinuingPinResourceUpper(b, g.config.capture, command.requests, command.captures)
	memory, memoryOK := providerContinuingPinMemoryUpper(g, command, storage, plan.metadataRaw, maximum, sizes)
	old, oldOK := providerContinuingBootstrapPlan(g)
	var peaks [4]uint64
	var peakOK [4]bool
	for i, request := range requests {
		p, stored, maxValue, ok := providerContinuingRPCResourceUpper(b, command.captures[i], uint64(len(request.wire)))
		if ok {
			peaks[i], peakOK[i] = providerContinuingRPCMemoryUpper(g, request, command.captures[i], stored, p.metadataRaw, maxValue)
		}
	}
	var refused []int
	for i, value := range plan.reservation.work {
		if value > s.firstLimits.operation[i] || value > s.firstLimits.segment[i] {
			refused = append(refused, i)
		}
	}
	// A genuine next G takes its first sample before the constructor checks
	// its remaining11+claim-clocks capacity. These are prospective counts;
	// the quote itself never samples or advances the original clock.
	afterFirstG, firstClockOK := providerUpperSum(clockRead, 1)
	requiredAfterG, requiredClockOK := providerUpperSum(11, g.bounds.claim.clocks)
	clockOK := firstClockOK && requiredClockOK && clockIdentityOK && candidateCapacity(0, afterFirstG, requiredAfterG, clockMaximum)
	clock.mu.Lock()
	unchangedClock := clock.read == clockRead
	clock.mu.Unlock()
	if !unchangedClock || s.stored != before || g.ordinal != ordinal || s.frontierFiles.pin != tip || s.activeMeter.Load() != observer || g.active || !g.ready {
		t.Fatal("arithmetic quote changed original selection or retained state")
	}
	return map[string]any{"resource_ok": resourceOK, "memory_ok": memoryOK, "memory_upper": memory, "resident_cap": b.continuation.maximumResidentBytes, "old_bootstrap_memory": old.memory, "old_bootstrap_ok": oldOK, "child_peaks": peaks, "child_peak_ok": peakOK, "work": plan.reservation.work, "operation_cap": s.firstLimits.operation, "segment_cap": s.firstLimits.segment, "refused_coordinates": refused, "physical": plan.physical, "stored": before, "storage_cap": b.maximumStoredBytes, "policy_live_cap": s.firstLimits.live, "policy_retained_cap": s.firstLimits.retained, "global_storage_fits": candidateCapacity(0, before, plan.physical, b.maximumStoredBytes), "policy_live_fits": candidateCapacity(0, before, plan.physical, s.firstLimits.live), "policy_retained_fits": candidateCapacity(0, before, plan.physical, s.firstLimits.retained), "clock_read_before_G": clockRead, "prospective_clock_after_first_G": afterFirstG, "required_remaining_clocks_after_G": requiredAfterG, "clock_cap": clockMaximum, "prospective_clock_fits": clockOK, "metadata_raw": plan.metadataRaw, "metadata_physical": plan.metadataPhysical}
}

func TestProviderContinuingPinResourcesAreBoundedDataWithoutAnOwner(t *testing.T) {
	t.Parallel()
	config, _ := providerGuardianConfigWithSelectionFixture(t, 64<<20, 6, &providerScanFixtureSelection{pin: true})
	b, global := config.bounds, config.capture
	requests, err := providerPinRequests(admissionTestDigest("a"))
	if err != nil {
		t.Fatal(err)
	}
	var captures [4]ProviderCaptureLimits
	for i, request := range requests {
		captures[i], err = b.methods.capture(request, global)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []string{"normal", "guardian", "methods", "tls", "global", "missing-request", "invalid-capture", "oversized-request", "claim-budget"} {
		t.Run(mode, func(t *testing.T) {
			bounds, selected, limits, readers := b, global, captures, requests
			guardian := *bounds.guardian
			bounds.guardian = &guardian
			switch mode {
			case "guardian":
				bounds.guardian = nil
			case "methods":
				bounds.methods = nil
			case "tls":
				bounds.tls = nil
			case "global":
				selected.BodyBytes = 0
			case "missing-request":
				readers[0] = nil
			case "invalid-capture":
				limits[0].BodyBytes = 0
			case "oversized-request":
				copy := *readers[0]
				copy.wire = string(make([]byte, (16<<10)+1))
				readers[0] = &copy
			case "claim-budget":
				guardian.claim.work[2] = 1
			}
			p, storage, maximum, sizes, ok := providerContinuingPinResourceUpper(bounds, selected, readers, limits)
			if mode == "normal" {
				if !ok || p.reservation.work[0] != 0 || p.reservation.work[1] != 0 || p.reservation.execution[3] != 1 || p.reservation.remaining[3] != 0 || p.physical != storage.physical || p.reservation.live != p.physical || maximum < 1<<20 || p.memory != 0 {
					t.Fatal("normal arithmetic quote lost its independent dimensions")
				}
				for _, size := range sizes {
					if size == 0 || size > bounds.maximumObjectBytes {
						t.Fatal("fixed child class does not fit the bounded original profile")
					}
				}
			} else if ok || p != (providerContinuingPlan{}) || storage != (providerFirstOperationStorage{}) || maximum != 0 || sizes != ([4]uint64{}) {
				t.Fatal("invalid quote DATA returned a usable proposal", mode)
			}
		})
	}
}
