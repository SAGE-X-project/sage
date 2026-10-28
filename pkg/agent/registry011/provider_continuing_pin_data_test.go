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
	capture, err := s.config.bounds.methods.capture(requests[0], s.config.capture)
	if err != nil {
		t.Fatal("unit expected method capture", err)
	}
	limits, ok := providerOperationStorageUpper(s.config.bounds, capture, uint64(len(requests[0].wire)))
	if !ok {
		t.Fatal("unit expected method storage bounds")
	}
	if value, err := r.continuingPinSettlement(view, g.integration.firstExchange, requests[0], capture, limits); err == nil || value != nil {
		t.Fatal("a different retained method supplied the requested unit settlement")
	}
	// Copy a typed lookup row into an invalid, unattached unit ledger. Only
	// its terminal hint is absent; no original S or actual ledger is changed.
	view, meter = providerPinUnitReadView(s, 1<<60)
	first := g.first.request
	firstTicket := g.integration.firstTicket
	if first == nil || firstTicket == nil || first != g.integration.first || firstTicket.kind != "rpc-capture" || firstTicket.input != first.inputDigest || firstTicket.method != first.method || firstTicket.params != first.paramsDigest {
		t.Fatal("first original request and ticket are not linked")
	}
	capture, err = s.config.bounds.methods.capture(first, s.config.capture)
	if err != nil {
		t.Fatal(err)
	}
	limits, ok = providerOperationStorageUpper(s.config.bounds, capture, uint64(len(first.wire)))
	if !ok {
		t.Fatal("unit first-exchange storage bounds")
	}
	entry, exists := g.integration.ledger.tickets[firstTicket.digest]
	if !exists || entry.ticket != firstTicket || entry.consumption == nil || !lowerHash(entry.terminal) {
		t.Fatal("original first ticket lacks its completed typed row")
	}
	entry.terminal = admissionTestDigest("f")
	if _, exists := view.objects[providerIntegrationRefKey(providerIntegrationObjectRef{kind: "settlement", digest: entry.terminal})]; exists {
		t.Fatal("unit absent settlement hint unexpectedly exists")
	}
	r = &providerCustodyReplay{meter: meter, ledger: &CandidateAdmissionLedger{tickets: map[string]candidateLedgerTicket{firstTicket.digest: entry}}}
	if value, err := r.continuingPinSettlement(view, g.integration.firstExchange, first, capture, limits); err == nil || value != nil || view.valid() {
		t.Fatal("absent terminal hint yielded an original unit settlement")
	}
	if s.failed || !g.ready || g.active || s.activeMeter.Load() != nil || s.stored != stored || g.ordinal != ordinal || s.frontierFiles.pin != tip {
		t.Fatal("detached unit DATA checks changed the original owner")
	}
}

func TestProviderContinuingPinReplayRejectsInvalidLookupDataBeforeAnyRead(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"bad-digest", "duplicate", "request-budget"} {
		meter := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{})
		s := &providerCustodyStore{config: providerCustodyConfig{bounds: providerIntegrationBounds{methods: &providerMethodBounds{}}}}
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

func TestProviderContinuingPinSelectedRequiresUnitInputs(t *testing.T) {
	t.Parallel()
	// No originals, filesystem, owner or counter are present in this guard.
	if value, err := providerContinuingPinSelected(nil, nil, nil); err == nil || value != "" {
		t.Fatal("missing unit inputs yielded selected DATA")
	}
}

func TestProviderContinuingPinProfileDataDisagreementRefusesBeforeNewOriginal(t *testing.T) {
	t.Parallel()
	s, g := providerContinuingRPCReadyWithSelectionFixture(t, &providerScanFixtureSelection{pin: true, pinResidentGiB: true})
	stored, sequence, ordinal, main, head := s.stored, s.sequence, g.ordinal, s.frontierFiles.pin, g.pin
	priorTicket := g.integration.ledger.lastTicket
	profile, origin := s.config.profile, s.setupOrigin
	// Fixed in-memory configuration DATA only. Both local copies disagree
	// with their original profile bytes; no original, identity, key, permit,
	// closed observer or stored profile is changed or replaced.
	if profile == admissionTestDigest("f") {
		t.Fatal("unit disagreement must differ from the original profile")
	}
	s.config.profile = admissionTestDigest("f")
	g.config.profile = s.config.profile
	digests := [4]string{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("d")}
	o, err := newProviderContinuingPinCandidate(s, "example-agent", digests)
	// The constructor holds P before rebuilding the profile in its codec.
	// The wrong DATA digest must refuse there, before any new G original/T.
	failed := g.continuing
	if err == nil || o != nil || failed == nil || failed.state != providerContinuingFailed || !s.failed || !g.failed || g.ready || s.activeMeter.Load() != nil || g.ordinal != ordinal+1 || g.pin != head || g.phase != providerGuardianMetadata || s.sequence != sequence || s.frontierFiles.pin != main || g.integration.ledger.lastTicket != priorTicket || s.setupOrigin != origin || g.original != origin || g.first.config.profile != profile || failed.plan.physical == 0 || s.stored != stored+failed.plan.physical || failed.ticket != nil || failed.execution != nil || failed.remaining != nil || g.claim == nil || g.claim.target != "" || g.integration.pin != "" {
		t.Fatal("profile DATA disagreement did not reach the held pre-original refusal", err)
	}
	if value, err := newProviderContinuingPinCandidate(s, "example-agent", digests); err == nil || value != nil || s.stored != stored+failed.plan.physical || g.ordinal != ordinal+1 {
		t.Fatal("failed profile DATA restored an admission or a physical hold")
	}
}

// A read-only unit view cannot substitute missing or inconsistent lookup DATA
// for an original binding. It borrows the immutable first exchange only;
// it cannot create an admission, close an observer or change the owned store.
func TestProviderContinuingPinReadViewRequiresCompleteAncillaryData(t *testing.T) {
	t.Parallel()
	s, g := providerContinuingRPCReadyWithSelectionFixture(t, &providerScanFixtureSelection{pin: true, pinResidentGiB: true})
	stored, ordinal, tip, origin := s.stored, g.ordinal, s.frontierFiles.pin, s.setupOrigin
	first, digest := g.first.request, g.integration.firstExchange
	capture, err := s.config.bounds.methods.capture(first, s.config.capture)
	if err != nil {
		t.Fatal("original first request class", err)
	}
	limits, ok := providerOperationStorageUpper(s.config.bounds, capture, uint64(len(first.wire)))
	if !ok {
		t.Fatal("original first request storage bounds")
	}
	view, meter := providerPinUnitReadView(s, 1<<60)
	r := &providerCustodyReplay{meter: meter, ledger: g.integration.ledger}
	value, err := r.continuingPinSettlement(view, digest, first, capture, limits)
	if err != nil || value == nil || value.retained != digest || view.valid() {
		t.Fatal("original read-only settlement precondition", err)
	}
	for _, mode := range []string{"missing-binding-index", "different-request", "missing-transport", "different-transport"} {
		view, meter := providerPinUnitReadView(s, 1<<60)
		r := &providerCustodyReplay{meter: meter, ledger: g.integration.ledger}
		if mode == "missing-binding-index" {
			view.bindings = map[string]string{}
			if value, err := r.continuingPinSettlement(view, digest, first, capture, limits); err == nil || value != nil {
				t.Fatal("unattached missing lookup DATA yielded a settlement")
			}
		} else {
			request := *first
			exchange := map[string]any{}
			switch mode {
			case "different-request":
				request.inputDigest = admissionTestDigest("f")
				if request.inputDigest == first.inputDigest {
					t.Fatal("unit request DATA must differ")
				}
			case "different-transport":
				// Finite nil-valued local DATA, never written or dispatched.
				transport := map[string]any{}
				for _, key := range []string{"endpoint", "method", "http_version", "status", "response_headers", "tls_version", "cipher_suite", "server_name", "verified_chain", "roots_artifact", "redirects", "proxy"} {
					transport[key] = nil
				}
				exchange["transport"] = transport
			}
			if r.continuingPinAncillaryBounds(view, digest, &request, exchange, capture, limits) {
				t.Fatal("incomplete or inconsistent unit DATA replaced originals", mode)
			}
		}
		if view.valid() || meter.actual[9] != 0 || s.activeMeter.Load() != nil || s.stored != stored || g.ordinal != ordinal || s.frontierFiles.pin != tip || s.setupOrigin != origin || s.failed || !g.ready || g.active {
			t.Fatal("read-only ancillary unit changed original authority", mode)
		}
	}
}
