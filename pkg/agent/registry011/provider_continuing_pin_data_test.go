package registry011

import "testing"

func TestProviderContinuingPinQuotesRefusePendingOrIncompleteData(t *testing.T) {
	t.Parallel()
	s, g := providerContinuingRPCReadyWithSelectionFixture(t, &providerScanFixtureSelection{pin: true, pinResidentGiB: true})
	digests := [4]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("d")}
	o, err := newProviderContinuingPinCandidate(s, "example-agent", digests)
	if err != nil {
		t.Fatal("original funded T", err)
	}
	stored, ordinal, observer := s.stored, g.ordinal, s.activeMeter.Load()
	// Read-only quotes cannot spend a prepared T twice. This tests the
	// outstanding reservation; it does not claim a separate final-page proof.
	if _, ok := providerContinuingPinPlan(g, o.pin); ok {
		t.Fatal("an outstanding original T produced another usable quote")
	}
	command := *o.pin
	command.requests[0] = nil
	if _, ok := providerContinuingPinMemoryUpper(g, &command, providerFirstOperationStorage{}, 0, 1<<20, [4]uint64{1, 1, 1, 1}); ok {
		t.Fatal("incomplete request DATA produced resident bounds")
	}
	if s.stored != stored || g.ordinal != ordinal || s.activeMeter.Load() != observer || o.state != providerContinuingPrepared {
		t.Fatal("pure quote changed original admission or storage")
	}
}

// A detached read-only unit view borrows immutable original bytes only. It
// has no owner, clock, custody writer or private role and cannot admit T/C/S.
// Every unit counter is new and unattached; none replaces an original observer.
func providerPinUnitReadView(s *providerCustodyStore, work uint64) (*providerCustodyStore, *providerWorkMeter) {
	meter := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{execution: candidateWorkVector{2: 64 << 20, 10: work}})
	view := &providerCustodyStore{config: s.config, evidence: s.evidence, objects: s.objects, bindings: s.bindings}
	view.activeMeter.Store(meter)
	return view, meter
}

func TestProviderContinuingPinSelectedDataRequiresOriginalAndExpectedAdmission(t *testing.T) {
	t.Parallel()
	s, g := providerContinuingRPCReadyWithSelectionFixture(t, &providerScanFixtureSelection{pin: true, pinResidentGiB: true})
	c := &providerIntegration{bootstrap: g.integration.bootstrap, intent: g.integration.intent, firstExchange: g.integration.firstExchange}
	stored, ordinal, tip := s.stored, g.ordinal, s.frontierFiles.pin
	view, normal := providerPinUnitReadView(s, 1<<60)
	selected, err := providerContinuingPinSelected(view, c, normal)
	if err != nil || selected != g.integration.selected.Hash || view.valid() {
		t.Fatal("read-only unit view yielded authority or wrong DATA", err)
	}
	copyCost := uint64(len(selected)) + 1
	if normal.actual[10] < copyCost {
		t.Fatal("unit copy observation missing")
	}
	for _, mode := range []string{"missing-original", "expected-admission", "copy-budget"} {
		view, meter := providerPinUnitReadView(s, 1<<60)
		candidate := &providerIntegration{bootstrap: c.bootstrap, intent: c.intent, firstExchange: c.firstExchange}
		switch mode {
		case "missing-original":
			view.objects = map[string]providerIntegrationObjectRef{}
		case "expected-admission":
			candidate.intent = &CandidateAdmissionIntent{digest: admissionTestDigest("f")}
		case "copy-budget":
			meter.upper[10] = normal.actual[10] - copyCost
		}
		if value, err := providerContinuingPinSelected(view, candidate, meter); err == nil || value != "" || view.valid() {
			t.Fatal("invalid unit DATA or short budget yielded selection", mode)
		}
		if mode == "copy-budget" && (!meter.failed || meter.actual[10] != normal.actual[10]-copyCost) {
			t.Fatal("short unit budget did not reach the final selected-string copy")
		}
	}
	view, meter := providerPinUnitReadView(s, 1<<60)
	r := &providerCustodyReplay{meter: meter, ledger: g.integration.ledger}
	requests, err := providerPinRequests(g.integration.selected.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if value, err := r.continuingPinSettlement(view, g.integration.firstExchange, requests[0]); err == nil || value != nil {
		t.Fatal("a different retained method supplied the requested unit settlement")
	}
	if s.failed || !g.ready || g.active || s.activeMeter.Load() != nil || s.stored != stored || g.ordinal != ordinal || s.frontierFiles.pin != tip {
		t.Fatal("detached unit DATA checks changed the original owner")
	}
}

func TestProviderContinuingPinReplayRejectsInvalidLookupDataBeforeAnyRead(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"bad-digest", "duplicate", "request-budget"} {
		meter := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{})
		s := &providerCustodyStore{}
		s.activeMeter.Store(meter)
		r := &providerCustodyReplay{meter: meter, ledger: &CandidateAdmissionLedger{}}
		digests := [4]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("d")}
		switch mode {
		case "bad-digest":
			digests[0] = "invalid"
		case "duplicate":
			digests[3] = digests[0]
		}
		// No original files or source role exist. Even valid lookup strings
		// cannot build the request using this zero unit formatting budget.
		values, err := r.continuingPinResults(s, digests, admissionTestDigest("a"))
		if err == nil {
			t.Fatal("invalid lookup DATA yielded results", mode)
		}
		for _, value := range values {
			if value != nil {
				t.Fatal("invalid unit lookup yielded an original response", mode)
			}
		}
	}
}
