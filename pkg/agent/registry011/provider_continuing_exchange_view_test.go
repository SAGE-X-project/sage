package registry011

import (
	"bytes"
	"encoding/base64"
	"testing"
)

// Both views borrow the same normal owned originals but have different,
// unattached unit observers. Neither view has a writer, clock or capability.
func TestProviderHistoryExchangeViewMatchesIndependentOriginalChecks(t *testing.T) {
	t.Parallel()
	s, g, digests := providerContinuingPinReadyFixture(t)
	requests, err := providerPinRequests(g.integration.selected.Hash)
	if err != nil {
		t.Fatal(err)
	}
	stored, sequence, ordinal := s.stored, s.sequence, g.ordinal
	main, origin := s.frontierFiles.pin, s.setupOrigin
	oldE, oldR := g.first.execution.actual, g.first.remaining.actual
	for i, digest := range digests {
		view, meter := providerPinUnitReadView(s, 1<<60)
		r := &providerCustodyReplay{meter: meter, ledger: g.integration.ledger, intent: g.integration.intent, policy: g.integration.policy}
		capture, err := s.config.bounds.methods.capture(requests[i], s.config.capture)
		if err != nil {
			t.Fatal(err)
		}
		bounds, valid := providerOperationStorageUpper(s.config.bounds, capture, uint64(len(requests[i].wire)))
		if !valid {
			t.Fatal("expected method storage bound")
		}
		settlement, err := r.continuingPinSettlement(view, digest, requests[i], capture, bounds)
		if err != nil || r.checkExchange(view, settlement) != nil {
			t.Fatal("old complete original checks", i, err)
		}
		expected, err := r.completedExchangeResult(view, digest, requests[i])
		if err != nil || meter.failed {
			t.Fatal("old result from the same complete originals", i, err)
		}
		view, meter = providerPinUnitReadView(s, 1<<60)
		r = &providerCustodyReplay{meter: meter, ledger: g.integration.ledger, intent: g.integration.intent, policy: g.integration.policy}
		got, err := r.historyExchangeResultView(view, digest, requests[i])
		if err != nil || !bytes.Equal(got, expected) || meter.failed || view.valid() || view.guardian != nil || view.clock != nil || meter.authority {
			t.Fatal("phase-local DATA differs or restores authority", i, err)
		}
		actual := meter.actual
		// Fixed, finite prefix cuts of each independently observed unit budget.
		// These are observer-exhaustion scenarios, not production reservation
		// estimates or changes to any original closed E/R meter.
		for _, coordinate := range []int{2, 10} {
			if actual[coordinate] < 64 {
				t.Fatal("normal view did not observe its complete input work")
			}
			for part := uint64(0); part < 64; part++ {
				short, unit := providerPinUnitReadView(s, 1<<60)
				unit.upper[coordinate] = actual[coordinate] * part / 64
				check := &providerCustodyReplay{meter: unit, ledger: g.integration.ledger, intent: g.integration.intent, policy: g.integration.policy}
				if value, err := check.historyExchangeResultView(short, digest, requests[i]); err == nil || value != nil || !unit.failed || short.valid() {
					t.Fatal("incomplete observed unit work yielded a result", i, coordinate, part)
				}
			}
		}
	}
	if s.failed || !g.ready || g.active || s.activeMeter.Load() != nil || s.stored != stored || s.sequence != sequence || g.ordinal != ordinal || s.frontierFiles.pin != main || s.setupOrigin != origin || g.first.execution.actual != oldE || g.first.remaining.actual != oldR {
		t.Fatal("unit views changed original owner, lifetime or observers")
	}
}

func TestProviderHistoryExchangeViewRequiresPhaseInputs(t *testing.T) {
	t.Parallel()
	config, _ := providerGuardianConfigWithSelectionFixture(t, 64<<20, 6, &providerScanFixtureSelection{pin: true, pinResidentGiB: true})
	request, err := newProviderIntegrationRequest("eth_chainId", []any{})
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"nil-replay", "nil-store", "nil-request", "nil-meter", "closed", "failed", "other-observer", "nil-ledger", "nil-intent", "nil-policy", "nil-methods", "bad-digest", "invalid-class", "invalid-storage", "missing-original"} {
		meter := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{execution: candidateWorkVector{2: 64 << 20, 10: 1 << 60}})
		view := &providerCustodyStore{config: config, objects: map[string]providerIntegrationObjectRef{}}
		view.activeMeter.Store(meter)
		r := &providerCustodyReplay{meter: meter, ledger: &CandidateAdmissionLedger{}, intent: &CandidateAdmissionIntent{}, policy: config.policy}
		digest, expected := admissionTestDigest("a"), request
		switch mode {
		case "nil-replay":
			r = nil
		case "nil-store":
			view = nil
		case "nil-request":
			expected = nil
		case "nil-meter":
			r.meter = nil
		case "closed":
			meter.closed = true
		case "failed":
			meter.failed = true
		case "other-observer":
			view.activeMeter.Store(nil)
		case "nil-ledger":
			r.ledger = nil
		case "nil-intent":
			r.intent = nil
		case "nil-policy":
			r.policy = nil
		case "nil-methods":
			view.config.bounds.methods = nil
		case "bad-digest":
			digest = "invalid"
		case "invalid-class":
			view.config.bounds.methods = &providerMethodBounds{}
		case "invalid-storage":
			view.config.bounds.maximumObjectBytes = 1
		}
		if value, err := r.historyExchangeResultView(view, digest, expected); err == nil || value != nil || meter.actual[2] != 0 || meter.actual[9] != 0 {
			t.Fatal("unowned finite inputs read an original or yielded DATA", mode)
		}
	}
}

func TestProviderHistoryExchangeViewRefusesMissingLookupData(t *testing.T) {
	t.Parallel()
	s, g := providerContinuingRPCReadyWithSelectionFixture(t, &providerScanFixtureSelection{pin: true, pinResidentGiB: true})
	stored, sequence, ordinal, tip := s.stored, s.sequence, g.ordinal, s.frontierFiles.pin
	request, digest := g.first.request, g.integration.firstExchange
	for _, mode := range []string{"missing-exchange", "missing-binding", "different-request", "missing-settlement"} {
		view, meter := providerPinUnitReadView(s, 1<<60)
		r := &providerCustodyReplay{meter: meter, ledger: g.integration.ledger, intent: g.integration.intent, policy: g.integration.policy}
		expected := *request
		switch mode {
		case "missing-exchange":
			view.objects = map[string]providerIntegrationObjectRef{}
		case "missing-binding":
			view.bindings = map[string]string{}
		case "different-request":
			expected.inputDigest = admissionTestDigest("f")
			if expected.inputDigest == request.inputDigest {
				t.Fatal("finite unit request must differ")
			}
		case "missing-settlement":
			entry, exists := g.integration.ledger.tickets[g.integration.firstTicket.digest]
			if !exists || entry.terminal == "" {
				t.Fatal("normal original settlement precondition")
			}
			entry.terminal = admissionTestDigest("f")
			key := providerIntegrationRefKey(providerIntegrationObjectRef{kind: "settlement", digest: entry.terminal})
			if _, exists := view.objects[key]; exists {
				t.Fatal("finite missing original unexpectedly exists")
			}
			r.ledger = &CandidateAdmissionLedger{tickets: map[string]candidateLedgerTicket{g.integration.firstTicket.digest: entry}}
		}
		if result, err := r.historyExchangeResultView(view, digest, &expected); err == nil || result != nil || view.valid() {
			t.Fatal("incomplete detached DATA yielded a checked result", mode)
		}
	}
	if s.failed || !g.ready || g.active || s.activeMeter.Load() != nil || s.stored != stored || s.sequence != sequence || g.ordinal != ordinal || s.frontierFiles.pin != tip {
		t.Fatal("missing unit DATA changed the original owner")
	}
}

func TestProviderHistoryBodyBindingUsesExactCanonicalEncoding(t *testing.T) {
	t.Parallel()
	for _, raw := range [][]byte{[]byte(`{"jsonrpc":"2.0","id":1,"result":null}`), {0}, {0, 1}, {0, 1, 2}, {0, 1, 2, 3}} {
		encoded := base64.RawURLEncoding.EncodeToString(raw)
		meter := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{execution: candidateWorkVector{10: 1 << 30}})
		if !historyViewBase64Matches(encoded, raw, meter) || meter.failed {
			t.Fatal("exact canonical DATA did not bind")
		}
		for _, different := range []any{nil, 1, "", encoded + "=", encoded + "A"} {
			unit := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{execution: candidateWorkVector{10: 1 << 30}})
			if historyViewBase64Matches(different, raw, unit) {
				t.Fatal("different finite encoded DATA bound as identical")
			}
		}
	}
	if historyViewBase64Matches("AA", []byte{0}, nil) || historyViewBase64Matches("", nil, newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{})) {
		t.Fatal("missing DATA or observer bound a body")
	}
	meter := newProviderWorkMeter(&providerIntegration{}, providerOperationReservation{})
	if historyViewBase64Matches("AA", []byte{0}, meter) || !meter.failed {
		t.Fatal("unfunded unit encoding was not refused")
	}
}
