package registry011

import (
	"encoding/base64"
	"testing"
)

func providerResultReplayUnit(t *testing.T) (*providerIntegration, *providerCustodyStore, *providerCustodyReplay, *CandidateAdmissionSettlement) {
	t.Helper()
	c, s := providerRetainedFirstExchangeFixture(t)
	view := providerReadDataView(t, s)
	for key, ref := range s.objects {
		view.objects[key] = ref
	}
	for key, digest := range s.bindings {
		view.bindings[key] = digest
	}
	r := &providerCustodyReplay{context: s.replay.context, policy: s.replay.policy, anchor: s.replay.anchor, intent: s.replay.intent, ledger: s.replay.ledger, firstExchange: s.replay.firstExchange}
	state := r.ledger.tickets[c.firstTicket.digest]
	raw, err := view.object("settlement", state.terminal)
	settlement, parseErr := ParseCandidateAdmissionSettlement(c.firstTicket, raw, 64<<10)
	if err != nil || parseErr != nil {
		t.Fatal("normal completed retained settlement", err, parseErr)
	}
	return c, view, r, settlement
}

// These altered records are finite parser inputs in an ownerless view. They
// are only refused: no live coordinator, transport, ACK or measured S uses them.
func TestProviderIntegrationExchangeDataRequiresExactIdentityAndTransportClosure(t *testing.T) {
	c, view, r, settlement := providerResultReplayUnit(t)
	raw, err := view.object("exchange", c.firstExchange)
	if err != nil || r.checkExchange(view, settlement) != nil {
		t.Fatal("normal retained exchange", err)
	}
	for _, mode := range []string{"identity", "clock", "request-base64", "request-body", "response-base64", "rpc", "transport", "chain-shape", "chain-empty", "certificate", "missing-binding"} {
		t.Run(mode, func(t *testing.T) {
			m := contentJSON(raw).(map[string]any)
			transport := m["transport"].(map[string]any)
			switch mode {
			case "identity":
				m["source_id"] = admissionTestDigest("f")
			case "clock":
				m["clock_start"].(map[string]any)["epoch"] = admissionTestDigest("f")
			case "request-base64":
				m["request"] = "!"
			case "request-body":
				m["request"] = base64.RawURLEncoding.EncodeToString([]byte("{}"))
			case "response-base64":
				m["response"] = "!"
			case "rpc":
				m["response"] = base64.RawURLEncoding.EncodeToString([]byte("{}"))
			case "transport":
				transport["endpoint"] = "invalid-unit-endpoint"
			case "chain-shape":
				transport["verified_chain"] = nil
			case "chain-empty":
				transport["verified_chain"] = []any{}
			case "certificate":
				transport["verified_chain"] = []any{"invalid"}
			case "missing-binding":
				// Change only the exchange digest; no corresponding wire binding
				// exists in this finite data view.
				m["clock_end"].(map[string]any)["uncertainty_ms"] = "9"
			}
			object := providerRetainReplayUnitData(t, view, "exchange", providerIntegrationJSON(m))
			copy := *settlement
			copy.retained = object.digest
			if r.checkExchange(view, &copy) == nil {
				t.Fatal("altered exchange input passed closed reconstruction", mode)
			}
		})
	}
	// Re-check the actual original after all independent refusal inputs.
	if r.checkExchange(view, settlement) != nil || c.bootstrap != "" {
		t.Fatal("unit refusals changed the retained producer state")
	}
}

func TestProviderIntegrationResponseDataRequiresOriginalWireHeadersAndBody(t *testing.T) {
	c, view, r, _ := providerResultReplayUnit(t)
	x, err := view.object("exchange", c.firstExchange)
	if err != nil {
		t.Fatal(err)
	}
	m := contentJSON(x).(map[string]any)
	body, ok := providerCanonicalBase64(m["response"], view.config.capture.BodyBytes)
	transport := m["transport"].(map[string]any)
	raw, err := view.object("wire-binding", view.bindings[c.firstExchange])
	binding := contentJSON(raw).(map[string]any)
	digest := contentString(binding["response_wire_digest"])
	if err != nil || !ok || r.matchResponse(view, digest, transport["response_headers"], body) != nil {
		t.Fatal("normal exact wire response", err)
	}
	for _, mode := range []string{"headers-shape", "headers-empty", "missing-wire", "wire-format", "wire-limit", "body", "headers"} {
		t.Run(mode, func(t *testing.T) {
			headers, selected, payload := transport["response_headers"], digest, body
			switch mode {
			case "headers-shape":
				headers = map[string]any{}
			case "headers-empty":
				headers = []any{}
			case "missing-wire":
				selected = admissionTestDigest("f")
			case "wire-format":
				object := providerRetainReplayUnitData(t, view, "response", []byte("finite invalid response data"))
				selected = object.digest
			case "wire-limit":
				original := view.config.capture.WireBytes
				view.config.capture.WireBytes = 1
				defer func() { view.config.capture.WireBytes = original }()
			case "body":
				payload = []byte("{}")
			case "headers":
				headers = []any{map[string]any{"name": "unit", "value": "different"}}
			}
			if r.matchResponse(view, selected, headers, payload) == nil {
				t.Fatal("incomplete response closure passed", mode)
			}
		})
	}
}

func TestProviderIntegrationStartDataReconstructsOnlyTheRetainedFinalizedHeader(t *testing.T) {
	view, r, _, _ := providerPinReplayDataFixture(t)
	raw, err := view.object("bootstrap", r.start)
	if err != nil || r.checkStart(view, r.start) != nil {
		t.Fatal("normal retained start", err)
	}
	x, err := view.object("exchange", r.firstExchange)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"identity", "selected", "missing-exchange", "response-base64", "response-rpc", "response-header"} {
		t.Run(mode, func(t *testing.T) {
			m := contentJSON(raw).(map[string]any)
			copy := *r
			switch mode {
			case "identity":
				m["admission_digest"] = admissionTestDigest("f")
			case "selected":
				m["selected_block_hash"] = admissionTestDigest("f")
			case "missing-exchange":
				copy.firstExchange = admissionTestDigest("f")
				m["selected_finality_exchange"] = copy.firstExchange
			default:
				exchange := contentJSON(x).(map[string]any)
				response := "!"
				if mode == "response-rpc" {
					response = base64.RawURLEncoding.EncodeToString([]byte("{}"))
				} else if mode == "response-header" {
					response = base64.RawURLEncoding.EncodeToString([]byte(`{"jsonrpc":"2.0","id":1,"result":{"hash":"invalid"}}`))
				}
				exchange["response"] = response
				object := providerRetainReplayUnitData(t, view, "exchange", providerIntegrationJSON(exchange))
				copy.firstExchange = object.digest
				m["selected_finality_exchange"] = object.digest
			}
			object := providerRetainReplayUnitData(t, view, "bootstrap", providerIntegrationJSON(m))
			if copy.checkStart(view, object.digest) == nil {
				t.Fatal("changed start data passed exact finalized reconstruction", mode)
			}
		})
	}
}
