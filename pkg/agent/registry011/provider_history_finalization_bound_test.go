package registry011

import "testing"

// Passive comparison after the ORIGINAL final owner has closed its meter.
// No observer, selected upper, native sample, model or live config is changed.
// This finite owned runtime cannot qualify a full production interval/profile.
func historyAssertFinalClosingSourceCost(t *testing.T, f *providerHistoryFinalOwner, out *providerHistoryFinalResult) {
	t.Helper()
	if f == nil || out == nil || out.catalog == nil || out.cursor == nil || !f.closed || f.failed ||
		out.cursor.phase != "done" || !out.cursor.semantic.atEnd() || len(out.cursor.pending) != 0 ||
		len(out.cursor.effects) != 0 || len(out.cursor.traceResults) != 0 || out.cursor.stopRecord != "" {
		t.Fatal("closing source comparison lacks its genuine terminal predecessor")
	}
	quote, ok := providerHistoryFinalizationUpper(f.parent.bounds.collection, uint64(len(out.begin.ClockID)))
	if !ok || quote.clocks != 2 || f.reader.meter.clocks != 2 {
		t.Fatal("original closing selection could not be priced without replacing its two samples")
	}
	for coordinate, actual := range out.work {
		before := out.catalog.work[coordinate]
		if actual < before {
			t.Fatal("final closing comparison observed a reset work counter", coordinate)
		}
		delta := actual - before
		if coordinate == 10 {
			// catalog.work already includes C0. The one C1 unit belongs to
			// finalBounds.upper; the source quote contains neither clock unit.
			if delta == 0 {
				t.Fatal("successful original final read omitted C1")
			}
			delta--
		}
		if delta > quote.envelope[coordinate] {
			t.Fatal("original final closing escaped its compiled source row", coordinate, delta, quote.envelope[coordinate])
		}
	}
}

func TestProviderHistoryFinalizationRetainsTerminalSpineAndConfirmedIndexes(t *testing.T) {
	config := historySelectedSourceConfig(t)
	manifest := config.bounds.history.collection
	before := *manifest
	got, ok := providerHistoryFinalizationUpper(manifest, 64)
	if !ok || got.limits.Spine != manifest.input.blocks || got.limits.Semantic.ConfirmedClaims != uint64(manifest.input.journal.MaxClaims) ||
		got.limits.Semantic.ConfirmedCreated != uint64(manifest.input.journal.MaxIdentities) || got.clocks != 2 || got.clockIDBytes != 64 {
		t.Fatal("terminal source discarded original spine/index maxima or replaced clock selection")
	}
	if got.limits.Effects != 0 || got.limits.PendingBlocks != 0 || got.limits.Semantic.Window || got.limits.Semantic.Pending ||
		got.limits.Semantic.Queue != 0 || got.limits.Semantic.WorkingClaims != 0 || got.limits.Semantic.WorkingCreated != 0 {
		t.Fatal("terminal quote silently selected another working-window graph")
	}
	// Handwritten noncursor primitive sequence: begin copy, result copy,
	// hash equality, and BOTH roots' identity/Stat/metadata observations.
	meter := providerReplayUnitMeter()
	for range 2 {
		if !meter.event(providerCopyEvent, 1) {
			t.Fatal("fixed closing copy")
		}
	}
	if !meter.event(providerCompareEvent, 132) {
		t.Fatal("fixed root equality")
	}
	for range 2 {
		for range 5 {
			if !meter.event(providerReadEvent, 0) {
				t.Fatal("fixed original root observation")
			}
		}
		if !meter.event(providerCompareEvent, 0) {
			t.Fatal("fixed original root metadata comparison")
		}
	}
	if meter.actual != (candidateWorkVector{10: 149}) || meter.clocks != 0 || !providerHistoryReadWorkDomain(got.envelope) || *manifest != before {
		t.Fatal("closing arithmetic introduced a clock/effect or mutated its original manifest")
	}
	longer, longerOK := providerHistoryFinalizationUpper(manifest, 65)
	if !longerOK || longer.normal[10]-got.normal[10] != 1 || longer.incoming != got.incoming || longer.cursor != got.cursor {
		t.Fatal("one extra clock ID byte changed unrelated work or created another sample")
	}
}

func TestProviderHistoryFinalizationFundsBothCommitmentsBeforeTheirCombinedRefusal(t *testing.T) {
	config := historySelectedSourceConfig(t)
	got, ok := providerHistoryFinalizationUpper(config.bounds.history.collection, 64)
	if !ok {
		t.Fatal("original closing source inputs")
	}
	for coordinate := range got.envelope {
		rootNormal, rootEnvelope := got.cursor.normalCommitment[coordinate], got.cursor.envelopeCommitment[coordinate]
		copyNormal, copyEnvelope := got.cursor.normalCopy[coordinate], got.cursor.envelopeCopy[coordinate]
		if rootEnvelope < rootNormal || copyEnvelope < copyNormal {
			t.Fatal("child closed envelope lost a successful prefix", coordinate)
		}
		// These are distinct source closures, not two independent grants:
		// either direct commitment may fail without poisoning the observer;
		// the third commitment/copy requires BOTH direct roots to succeed.
		for _, entered := range []uint64{2 * rootEnvelope, rootEnvelope + rootNormal, 2*rootNormal + copyEnvelope} {
			if entered > got.envelope[coordinate] {
				t.Fatal("a paired or copy closure lost its entered work", coordinate, entered)
			}
		}
		if got.incoming[coordinate] != max(2*(rootEnvelope-rootNormal), copyEnvelope-copyNormal) ||
			got.envelope[coordinate] != got.normal[coordinate]+got.incoming[coordinate] {
			t.Fatal("combined first-false closure was replaced by one root or summed with its copy alternative", coordinate)
		}
	}
}

func TestProviderHistoryFinalizationRejectsUnsealedInputsAndLateOverflowWithoutPartialRows(t *testing.T) {
	config := historySelectedSourceConfig(t)
	for _, name := range []string{"nil-manifest", "copied-manifest", "changed-snapshot", "zero-clock-id", "clock-id-overflow", "late-total-overflow"} {
		t.Run(name, func(t *testing.T) {
			manifest, clockID := config.bounds.history.collection, uint64(64)
			switch name {
			case "nil-manifest":
				manifest = nil
			case "copied-manifest":
				copy := *manifest
				manifest = &copy
			case "changed-snapshot":
				copy := *manifest
				copy.owner = &copy
				copy.input.blocks++
				manifest = &copy
			case "zero-clock-id":
				clockID = 0
			case "clock-id-overflow":
				clockID = ^uint64(0)
			case "late-total-overflow":
				// The DATA preparation child still succeeds at exactly MaxUint64.
				// Adding preceding original root work must then reject the WHOLE
				// closing quote, without returning a successful cursor prefix.
				one, ok := providerHistoryFinalDataPreparationUpper(1)
				if !ok || one[10] == 0 {
					t.Fatal("bounded preparation precursor")
				}
				clockID = ^uint64(0) - one[10] + 1
				if upper, ok := providerHistoryFinalDataPreparationUpper(clockID); !ok || upper[10] != ^uint64(0) {
					t.Fatal("late-overflow input failed before its enclosing source sum")
				}
			}
			if out, ok := providerHistoryFinalizationUpper(manifest, clockID); ok || out != (providerHistoryFinalizationCost{}) {
				t.Fatal("invalid closing input leaked cursor work, DATA copies or a clock row", name)
			}
		})
	}
}
