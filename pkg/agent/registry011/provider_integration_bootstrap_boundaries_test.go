package registry011

import (
	"encoding/json"
	"strconv"
	"testing"
)

func TestProviderBootstrapCannotInstallFromChangedOrIncompleteLocalData(t *testing.T) {
	for _, mode := range []string{"missing-exchange", "empty-result", "existing-start", "header-shape", "number", "timestamp", "old-height", "changed-hash", "clock", "preparations"} {
		t.Run(mode, func(t *testing.T) {
			// One normal owned TLS exchange is fully retained first. Every
			// changed value below is local finite UNIT data, never a malicious
			// peer or a payload sent over a transport.
			c, s := providerRetainedFirstExchangeFixture(t)
			// An RPC result is ordinary JSON, not retained canonical metadata.
			// Decode the normal wire object before making finite unit changes.
			var value map[string]any
			if err := json.Unmarshal([]byte(c.firstResult), &value); err != nil {
				t.Fatal("normal RPC result fixture", err)
			}
			switch mode {
			case "missing-exchange":
				c.firstExchange = ""
			case "empty-result":
				c.firstResult = ""
			case "existing-start":
				c.ledger.providerStart = admissionTestDigest("f")
			case "header-shape":
				c.firstResult = `{}`
			case "number":
				value["number"] = "0x00"
				c.firstResult = string(providerIntegrationJSON(value))
			case "timestamp":
				value["timestamp"] = "unknown"
				c.firstResult = string(providerIntegrationJSON(value))
			case "old-height":
				value["number"] = "0x" + strconv.FormatUint(sepoliaDeploymentBlock-1, 16)
				c.firstResult = string(providerIntegrationJSON(value))
			case "changed-hash":
				value["hash"] = admissionTestDigest("f")
				c.firstResult = string(providerIntegrationJSON(value))
			case "clock":
				c.clock.failed = true
			case "preparations":
				c.preparations = s.config.bounds.maximumCommits
			}
			before := c.intent
			if c.installBootstrap() == nil || c.bootstrap != "" || s.replay.start != "" || c.intent != before || c.root.status != "enrolling" || c.root.checkpoint != "" {
				t.Fatal("changed/unavailable first result installed a bootstrap", mode)
			}
		})
	}
}

func TestProviderPinReplayRequiresRetainedExchangeAndCompletedTicket(t *testing.T) {
	for _, mode := range []string{"normal", "missing-exchange", "missing-ledger", "unconsumed", "wrong-request", "missing-settlement"} {
		t.Run(mode, func(t *testing.T) {
			c, s := providerRetainedFirstExchangeFixture(t)
			view := providerReadDataView(t, s)
			for key, ref := range s.objects {
				view.objects[key] = ref
			}
			body := &providerCustodyReplay{context: s.replay.context, policy: s.replay.policy, anchor: s.replay.anchor, intent: s.replay.intent, ledger: s.replay.ledger, firstExchange: s.replay.firstExchange}
			request := c.first
			switch mode {
			case "missing-exchange":
				delete(view.objects, "exchange:"+c.firstExchange)
			case "missing-ledger":
				body.ledger = nil
			case "unconsumed":
				var err error
				body.ledger, err = NewCandidateAdmissionLedger(c.policy, c.intent, c.firstTicket, s.config.bounds.ledger)
				if err != nil {
					t.Fatal(err)
				}
			case "wrong-request":
				copy := *request
				copy.method = "eth_chainId"
				request = &copy
			case "missing-settlement":
				state := body.ledger.tickets[c.firstTicket.digest]
				delete(view.objects, "settlement:"+state.terminal)
			}
			result, err := body.completedExchangeResult(view, c.firstExchange, request)
			if mode == "normal" {
				if err != nil || string(result) != c.firstResult {
					t.Fatal("normal exact retained exchange data", err)
				}
			} else if err == nil || result != nil {
				t.Fatal("incomplete retained exchange produced a pin result", mode)
			}
			if c.bootstrap != "" || s.replay.start != "" || c.ledger.current.counts != [3]uint64{1, 1, 1} {
				t.Fatal("DATA replay changed operating enrollment or retained work")
			}
		})
	}
}

func providerPinReplayDataFixture(t *testing.T) (*providerCustodyStore, *providerCustodyReplay, string, string) {
	t.Helper()
	c, s := providerRetainedFirstExchangeFixture(t)
	if err := c.installBootstrap(); err != nil {
		t.Fatal("normal owned retained bootstrap", err)
	}
	view := providerReadDataView(t, s)
	for key, ref := range s.objects {
		view.objects[key] = ref
	}
	body := &providerCustodyReplay{context: s.replay.context, policy: s.replay.policy, anchor: s.replay.anchor, intent: s.replay.intent, ledger: s.replay.ledger, start: s.replay.start, firstExchange: s.replay.firstExchange}
	raw := providerPinInput(body.start, "finite-unit-agent")
	input := providerRetainReplayUnitData(t, view, "pin-input", raw)
	return view, body, input.digest, c.selected.Hash
}

// Only finite UNIT originals in this fixture's isolated directory are added.
// The view has no owner/ACK, and no coordinator or transport consumes its data.
func providerRetainReplayUnitData(t *testing.T, view *providerCustodyStore, kind string, raw []byte) providerIntegrationObject {
	t.Helper()
	object, err := newProviderIntegrationObject(kind, raw, view.config.bounds.maximumObjectBytes)
	if err != nil {
		t.Fatal(err)
	}
	key := kind + ":" + object.digest
	if _, exists := view.objects[key]; !exists {
		if view.evidence.write(providerObjectName(kind, object.digest), raw) != nil {
			t.Fatal("normal immutable unit data retention")
		}
		view.objects[key] = providerIntegrationObjectRef{kind, object.digest, uint64(len(raw))}
	}
	return object
}

func TestProviderPinInputDataRequiresTheSameRetainedBootstrap(t *testing.T) {
	for _, mode := range []string{"normal", "missing-input", "changed-start", "missing-bootstrap", "invalid-selected"} {
		t.Run(mode, func(t *testing.T) {
			view, body, input, selected := providerPinReplayDataFixture(t)
			switch mode {
			case "missing-input":
				delete(view.objects, "pin-input:"+input)
			case "changed-start":
				raw := providerPinInput(admissionTestDigest("f"), "finite-unit-agent")
				input = providerRetainReplayUnitData(t, view, "pin-input", raw).digest
			case "missing-bootstrap":
				delete(view.objects, "bootstrap:"+body.start)
			case "invalid-selected":
				// Rejected DATA only: neither this small typed placeholder nor
				// its digest can qualify a bootstrap or completed local command.
				raw := providerIntegrationJSON(map[string]any{"format": "sage-provider-start-v2", "selected_block_hash": "invalid"})
				body.start = providerRetainReplayUnitData(t, view, "bootstrap", raw).digest
				input = providerRetainReplayUnitData(t, view, "pin-input", providerPinInput(body.start, "finite-unit-agent")).digest
			}
			agent, hash, err := body.pinInput(view, input)
			if mode == "normal" {
				if err != nil || agent != "finite-unit-agent" || hash != selected {
					t.Fatal("normal retained pin input data", err)
				}
			} else if err == nil || agent != "" || hash != "" {
				t.Fatal("crossed/missing input produced a selected pin", mode)
			}
		})
	}
}

func TestProviderPinClosureRefusesFiniteUnsettledData(t *testing.T) {
	for _, mode := range []string{"missing-pin", "input", "selected", "shape", "count", "digest", "unretained-exchanges"} {
		t.Run(mode, func(t *testing.T) {
			view, body, input, selected := providerPinReplayDataFixture(t)
			value := map[string]any{"format": "sage-provider-owned-pin-v1", "input": input, "selected_block_hash": selected, "exchanges": []any{admissionTestDigest("a"), admissionTestDigest("b"), admissionTestDigest("c"), admissionTestDigest("d")}}
			switch mode {
			case "input":
				value["input"] = admissionTestDigest("f")
			case "selected":
				value["selected_block_hash"] = admissionTestDigest("f")
			case "shape":
				value["exchanges"] = map[string]any{}
			case "count":
				value["exchanges"] = []any{}
			case "digest":
				value["exchanges"].([]any)[0] = "invalid"
			}
			pin := providerRetainReplayUnitData(t, view, "pin", providerIntegrationJSON(value))
			if mode == "missing-pin" {
				delete(view.objects, "pin:"+pin.digest)
			}
			// These typed placeholders are exclusively refused unit data. There
			// is no S parser, owned measurement, actual settlement or work ACK.
			settlement := &CandidateAdmissionSettlement{ticket: &CandidateAdmissionTicket{input: input}, retained: pin.digest}
			if body.checkPin(view, settlement) == nil {
				t.Fatal("unsettled/incomplete pin DATA became a complete result", mode)
			}
		})
	}
}
