package registry011

import "testing"

// Each call reconstructs a new private arithmetic model from the same genuine
// original read-only prefix. It neither installs an initial model nor changes
// any selected frame, file, ticket, clock, observer or execution allowance.
func historyRecoveryMessageDataUnit(t *testing.T, r *providerHistoryRecoveryOwner) (*providerHistoryRecoveryInitialModel, *CandidateAdmissionSettlement) {
	t.Helper()
	f, err := newProviderFrontierReplayObserved(r.config, r.meter)
	if err != nil {
		t.Fatal("fresh private message model", err)
	}
	f.meter, f.body.meter, f.indexes.meter = r.meter, r.meter, r.meter
	m := &providerHistoryRecoveryInitialModel{recovery: r, prefix: r.legacyPrefixes[0], frontier: f}
	m.owner = m
	for i, frame := range m.prefix.source.frames {
		m.used[i] = make([]bool, len(frame.objects))
	}
	for i := range 3 {
		m.through = uint64(i + 1)
		if m.apply(m.prefix.records[i]) != nil {
			t.Fatal("genuine private enrollment/admission/consumption", i)
		}
	}
	m.through = 4
	state := f.body.ledger.tickets[r.initialDescriptor.ticket]
	raw, err := m.object("settlement", m.prefix.records[3].operation)
	settlement, parseErr := parseCandidateAdmissionSettlementObserved(state.ticket, raw, 64<<10, r.meter)
	if err != nil || parseErr != nil || state.ticket == nil || state.consumption == nil ||
		m.request(state.ticket) != nil || m.exchange(settlement) != nil {
		t.Fatal("genuine original message and consumption prerequisites", err, parseErr)
	}
	return m, settlement
}

func TestProviderHistoryRecoveryRequestDataRequiresItsOriginalTicketBinding(t *testing.T) {
	for _, mode := range []string{"same", "nil-ticket", "kind", "sequence", "missing-request", "method", "params"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			files, bytes := r.inputFiles, r.inputBytes
			m, s := historyRecoveryMessageDataUnit(t, r)
			ticket := *s.ticket
			selected := &ticket
			switch mode {
			case "nil-ticket":
				selected = nil
			case "kind":
				ticket.kind = "local-work"
			case "sequence":
				ticket.sequence = 1
			case "missing-request":
				ticket.input = admissionTestDigest("f")
			case "method":
				ticket.method = "web3_clientVersion"
			case "params":
				ticket.params = admissionTestDigest("f")
			}
			if err := m.request(selected); (err == nil) != (mode == "same") || r.meter.failed {
				t.Fatal("inconsistent private request ticket accepted or original observer poisoned", mode, err)
			}
			if o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				r.inputFiles != files || r.inputBytes != bytes || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) || !r.activeLocked(r.meter) || o.store.activeMeter.Load() != nil {
				t.Fatal("private request checks performed I/O, installed a model or revived original execution")
			}
		})
	}
}

func TestProviderHistoryRecoveryResponseDataRequiresItsOriginalBodyAndHeaders(t *testing.T) {
	// Invalid private parser configuration must fail before any selected input
	// is needed. This detached zero role cannot select an original or a Source.
	absent := &providerHistoryRecoveryInitialModel{recovery: &providerHistoryRecoveryOwner{meter: providerReplayUnitMeter()}}
	if absent.response("", nil, nil) == nil || absent.recovery.meter.failed {
		t.Fatal("invalid private capture configuration reached an original response")
	}
	for _, mode := range []string{"same", "missing-headers", "missing-response", "different-body", "different-headers"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			files, bytes := r.inputFiles, r.inputBytes
			m, s := historyRecoveryMessageDataUnit(t, r)
			raw, err := m.object("exchange", s.retained)
			v, ok := providerReplayJSON(raw, r.meter).(map[string]any)
			if err != nil || !ok {
				t.Fatal("actual original exchange DATA", err)
			}
			transport, ok := v["transport"].(map[string]any)
			if !ok {
				t.Fatal("actual original transport DATA")
			}
			binding, err := m.wireBinding(s.retained)
			body, bodyOK := providerCanonicalBase64Observed(v["response"], r.config.capture.BodyBytes, r.meter)
			digest, headers := contentString(binding["response_wire_digest"]), transport["response_headers"]
			if err != nil || !bodyOK || len(body) == 0 || m.response(digest, headers, body) != nil {
				t.Fatal("exact original response body/header baseline", err)
			}
			switch mode {
			case "missing-headers":
				headers = nil
			case "missing-response":
				digest = admissionTestDigest("f")
			case "different-body":
				body[0] ^= 1 // Expected DATA only; no stored or network bytes change.
			case "different-headers":
				values, ok := replayTestClone(t, headers).([]any)
				if !ok || len(values) == 0 {
					t.Fatal("independent original header DATA copy")
				}
				first, ok := values[0].(map[string]any)
				if !ok {
					t.Fatal("original header shape")
				}
				first["value"] = contentString(first["value"]) + "-different"
				headers = values
			}
			if err := m.response(digest, headers, body); (err == nil) != (mode == "same") || r.meter.failed {
				t.Fatal("inconsistent expected response DATA accepted or original observer poisoned", mode, err)
			}
			if o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				r.inputFiles != files || r.inputBytes != bytes || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) || !r.activeLocked(r.meter) || o.store.activeMeter.Load() != nil {
				t.Fatal("private response checks performed I/O, installed a model or revived original execution")
			}
		})
	}
}

func TestProviderHistoryRecoveryExchangeDataRequiresTheOriginalConsumedTicket(t *testing.T) {
	for _, mode := range []string{"same", "nil-settlement", "missing-exchange", "missing-consumption"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			files, bytes := r.inputFiles, r.inputBytes
			m, original := historyRecoveryMessageDataUnit(t, r)
			s := *original
			selected := &s
			switch mode {
			case "nil-settlement":
				selected = nil
			case "missing-exchange":
				s.retained = admissionTestDigest("f")
			case "missing-consumption":
				state := m.frontier.body.ledger.tickets[s.ticket.digest]
				state.consumption = nil
				m.frontier.body.ledger.tickets[s.ticket.digest] = state
			}
			if err := m.exchange(selected); (err == nil) != (mode == "same") || r.meter.failed {
				t.Fatal("unconsumed or unbound private exchange accepted or original observer poisoned", mode, err)
			}
			if o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				r.inputFiles != files || r.inputBytes != bytes || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) || !r.activeLocked(r.meter) || o.store.activeMeter.Load() != nil {
				t.Fatal("private exchange checks performed I/O, installed a model or revived original execution")
			}
		})
	}
}

func TestProviderHistoryRecoveryWireBindingRejectsDifferentAndAbsentExpectedExchange(t *testing.T) {
	for _, mode := range []string{"same", "different-exchange", "no-private-records"} {
		t.Run(mode, func(t *testing.T) {
			o, r, want := historyRecoveryInventoryFixtureWithLegacy(t, true)
			historyRecoveryModelPrefix(t, r, 0, want)
			r.store.mu.Lock()
			defer r.store.mu.Unlock()
			old, clocks, pin, ack, held := o.meter.actual, o.store.clock.read, o.pin, o.last, o.store.stored
			files, bytes := r.inputFiles, r.inputBytes
			m, s := historyRecoveryMessageDataUnit(t, r)
			selected := s.retained
			if mode == "different-exchange" {
				selected = admissionTestDigest("f")
			}
			if mode == "no-private-records" {
				m.through = 0
			}
			binding, err := m.wireBinding(selected)
			if (err == nil) != (mode == "same") || mode != "same" && binding != nil || r.meter.failed {
				t.Fatal("unmatched private exchange selected original wire bindings", mode, err)
			}
			if o.meter.actual != old || o.store.clock.read != clocks || o.pin != pin || o.last != ack || o.store.stored != held ||
				r.inputFiles != files || r.inputBytes != bytes || r.initialModels != ([2]*providerHistoryRecoveryInitialModel{}) || !r.activeLocked(r.meter) || o.store.activeMeter.Load() != nil {
				t.Fatal("private binding checks performed I/O, installed a model or revived original execution")
			}
		})
	}
}
