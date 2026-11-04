package registry011

import (
	"bytes"
	"testing"
)

func TestProviderHistoryInitialScanInputsFundAllObservedPrimitiveEnvelopes(t *testing.T) {
	// Independent literals include three full metadata encode preflights,
	// three distinct domain hashes, and one bounded policy copy/decode.
	for _, row := range []struct{ metadata, decoded, units uint64 }{
		{1, 1, 116}, {4096, 4096, 67125356}, {8192, 8192, 268468332},
		{65536, 8192, 12952100972}, {1048576, 8192, 3298603065452},
	} {
		got, ok := providerHistoryInitialScanInputUpper(row.metadata)
		want := providerScanUpper{work: candidateWorkVector{2: row.decoded, 10: row.units}}
		if !ok || got != want {
			t.Fatal("observed preparation lost a primitive, preflight or separate dimension", row, got, ok)
		}
	}
	for _, metadata := range []uint64{0, (1 << 20) + 1, ^uint64(0)} {
		if got, ok := providerHistoryInitialScanInputUpper(metadata); ok || got != (providerScanUpper{}) {
			t.Fatal("invalid metadata returned a partial preparation", metadata, got)
		}
	}
}

func TestProviderHistoryInitialScanSelectionPreservesOriginalPreparation(t *testing.T) {
	for _, collection := range []bool{false, true} {
		for _, ordinal := range []uint64{1, 2} {
			old, oldOK := providerScanPreparationCostUpper(4096, 64, ordinal, 2)
			got, ok := providerScanSelectedPreparationCostUpper(4096, 64, ordinal, 2, collection)
			want := old
			if collection && ordinal == 1 {
				want.work[2] += 4096
				want.work[10] += 67125356
			}
			if !oldOK || !ok || got != want {
				t.Fatal("selection replaced old P or changed retained storage/clocks", collection, ordinal, got, want)
			}
		}
	}
	if got, ok := providerScanSelectedPreparationCostUpper(4096, 64, 3, 2, true); ok || got != (providerScanUpper{}) {
		t.Fatal("invalid original ordinal retained an extra-only result")
	}
}

// Existing synthetic UNIT selections precede every root/clock observation.
// They exercise the original short E0/P seam, never production D or a real
// deployment-to-current profile. No new witness factory is introduced here.
func historyCallerFreshConfiguration(t *testing.T) (providerCustodyConfig, *providerOwnedClock) {
	t.Helper()
	manifest, _ := historyCollectionRecordedRunManifest(t)
	first := providerFirstWriterBounds()
	return providerScanProvisionedFixtureWithSelection(t, &first, true, &providerScanFixtureSelection{
		capture: &manifest.input.capture, pin: true, pinLargeObject: true, historyProgram: true,
		historyGroupWrite: true, historyGroupConsume: true, historyGroupDispatch: true,
		historyRecoveryNamespace: true, historyRecoveryLegacy: true,
		historyCollection: manifest, historyCollectionAdmission: true, historyCollectionFirstP: true,
		historyCollectionFullPack: true, historyCollectionRecordedRun: true, historyFinalRead: true,
	})
}

func historyCallerEnrollmentPrefix(t *testing.T) *providerCustodyStore {
	t.Helper()
	config, clock := historyCallerFreshConfiguration(t)
	s, err := openProviderFrontierCandidate(config, clock)
	if err != nil || s == nil {
		t.Fatal("fresh isolated original open", err)
	}
	t.Cleanup(func() {
		if !s.closed {
			_ = s.close()
		}
	})
	if _, err := s.enrollBeforeScan(); err != nil || !s.bootstrapOwner.done || s.sequence != 1 || s.scanControl.prepared != nil {
		t.Fatal("genuine short enrollment prerequisite", err)
	}
	return s
}

func TestProviderHistoryInitialTargetKeepsOneHeldObservedPreparation(t *testing.T) {
	s := historyCallerEnrollmentPrefix(t)
	c, p := s.scanControl, s.history
	// DATA-only reference construction is an oracle, never live preparation.
	x := mustProviderBindingCodec(t, s.config)
	want, err := x.target(providerScanTarget{setup: s.setupTip, custody: s.tip, frontier: s.frontierFiles.pin.frontier, objects: s.objectsRoot, enrollment: s.enrollment})
	previous, previousErr := c.pin.bytes(s.config)
	deadline, _, deadlineOK := providerScanDeadline(s.config, providerClockDiagnostic(s.setupOrigin))
	if err != nil || previousErr != nil || !deadlineOK {
		t.Fatal("bounded reference DATA", err, previousErr)
	}
	input, inputOK := providerBootstrapCostingInputFromBounds(s.config.bounds)
	selected, selectedOK := providerScanSelectedPreparationCostUpper(input.maximumMetadataPerCommit, input.fileOverheadBytes, 1, input.scan.maximumAttempts, input.historyCollection)
	runtime, runtimeErr := providerScanPreparationUpper(s.config, 1)
	if !inputOK || !input.historyCollection || !selectedOK || runtimeErr != nil || selected != runtime {
		t.Fatal("compiler snapshot and actual wrapper selected different P", runtimeErr)
	}
	reads, held := s.clock.read, s.stored
	target, err := c.claimInitialTargetCandidate()
	m := p.preparationMeter
	if err != nil || !bytes.Equal(target, want) || c.prepared == nil || c.prepared.preparation != selected ||
		c.prepared.target != providerCandidateHash("sage-provider-scan-target-v1", want) ||
		c.prepared.previousHighwater != providerCandidateHash(providerScanHighwaterDomain, previous) || c.prepared.deadline != deadline ||
		m == nil || m.scan != c || m.actual[2] == 0 || m.actual[10] == 0 || m.clocks != 2 || s.clock.read != reads+2 ||
		c.pin.ordinal != 1 || c.failed || s.failed || s.activeMeter.Load() != nil || s.stored != held || p.inspection != c.held {
		t.Fatal("initial target escaped its original held preparation", err)
	}
	if !providerHistoryUpperWithin(providerScanUpper{work: m.actual, clocks: m.clocks}, selected) {
		t.Fatal("new observed preparation exceeded its selected row")
	}
	actual, now, pin := m.actual, s.clock.read, c.pin
	if again, err := c.claimInitialTargetCandidate(); err == nil || again != nil || m.actual != actual || s.clock.read != now || c.pin != pin || s.stored != held {
		t.Fatal("duplicate initial entry renewed a claim or original observation")
	}
}
