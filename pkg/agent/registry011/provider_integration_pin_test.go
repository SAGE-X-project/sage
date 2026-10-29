package registry011

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func providerPinResultFixture(t *testing.T) [4]json.RawMessage {
	t.Helper()
	registry, _ := json.Marshal(returnABI(t, "registryId", "eip155:11155111:"+SepoliaRegistryAddress))
	return [4]json.RawMessage{
		json.RawMessage(`"0xaa36a7"`),
		json.RawMessage(`{"hash":"` + sepoliaGenesis + `","parentHash":"0x` + strings.Repeat("0", 64) + `","number":"0x0","timestamp":"0x6159af19"}`),
		json.RawMessage(`"owned-normal-provider"`),
		registry,
	}
}

func TestProviderPinRepeatedValidationCountsEveryDecoderInvocation(t *testing.T) {
	s, policy, candidate, primitive := providerStoreFixture(t)
	c, err := newProviderIntegration(s, policy, candidate, primitive)
	if err != nil {
		t.Fatal(err)
	}
	results := providerPinResultFixture(t)
	var decoded, units uint64
	for _, raw := range results {
		n := uint64(len(raw))
		decoded += n
		units += n*n + 1
	}
	m := newProviderWorkMeter(c, providerOperationReservation{execution: candidateWorkVector{2: 2 * decoded, 10: 2 * units}})
	s.activeMeter.Store(m)
	defer s.activeMeter.Store(nil)
	// The same retained header does not make a second parser invocation free.
	for run := uint64(1); run <= 2; run++ {
		if !s.checkPinResults(results) || m.actual[2] != run*decoded || m.actual[10] != run*units || m.actual[0] != 0 || m.actual[1] != 0 || m.actual[3] != 0 {
			t.Fatal("revalidation lost codec events or invented network/block work")
		}
	}
	before := m.actual
	if s.checkPinPart(0, results[0]) || !m.failed || m.actual != before {
		t.Fatal("decoder ran after its unchanged reservation was exhausted")
	}
}

func TestProviderPinMalformedResultRetainsItsObservedValidationCost(t *testing.T) {
	s, policy, candidate, primitive := providerStoreFixture(t)
	c, err := newProviderIntegration(s, policy, candidate, primitive)
	if err != nil {
		t.Fatal(err)
	}
	m := newProviderWorkMeter(c, providerOperationReservation{execution: candidateWorkVector{2: 16, 10: 256}})
	s.activeMeter.Store(m)
	defer s.activeMeter.Store(nil)
	if s.checkPinPart(0, json.RawMessage(`{}`)) || m.actual[2] != 2 || m.actual[10] != 5 || m.actual[0] != 0 || m.actual[1] != 0 {
		t.Fatal("invalid result was refunded or counted as network work")
	}
}

func TestProviderPinSettlementCountsBothInputsAndRefusesIncompleteMeasurement(t *testing.T) {
	owner, ticket, _, _ := providerMeterFixture(t)
	raw := providerPolicyTestBytes(t, ledgerTestSettlementObject(ticket, "complete", 1))
	// Inert decoder data: the proposed actual_work in this record is never
	// accepted as an owned measurement, persisted ACK or operating credential.
	a, b := uint64(len(raw)), uint64(len(ticket.policy))
	decoded := a + b
	units := a*a + 1 + b*b + 1 + uint64(len("sage-provider-budget-settlement-v2")+1) + a + 1 + a + 1
	for _, mode := range []string{"normal", "policy-budget", "hash-budget", "bad-record", "nil-ticket", "empty-record", "oversize-record"} {
		t.Run(mode, func(t *testing.T) {
			s := &providerCustodyStore{}
			m := newProviderWorkMeter(owner, providerOperationReservation{execution: candidateWorkVector{2: decoded, 10: units}})
			s.activeMeter.Store(m)
			selected, input := ticket, raw
			switch mode {
			case "policy-budget":
				m.upper[2] = decoded - 1
			case "hash-budget":
				m.upper[10] = units - 1
			case "bad-record":
				input = []byte(`{}`)
			case "nil-ticket":
				selected = nil
			case "empty-record":
				input = nil
			case "oversize-record":
				input = make([]byte, 64<<10+1)
			}
			result, err := s.parsePinSettlement(selected, input)
			if mode == "normal" {
				if err != nil || result == nil || m.failed || m.actual[2] != decoded || m.actual[10] != units || m.actual[0] != 0 || m.actual[1] != 0 || m.actual[9] != 0 {
					t.Fatal("two decoder inputs were merged, omitted or used as work authority", err)
				}
			} else if err == nil || result != nil {
				t.Fatal("invalid or underfunded decoder produced a result")
			} else if (mode == "policy-budget" || mode == "hash-budget" || mode == "bad-record") && !m.failed {
				t.Fatal("partial/failed parser left a usable measurement")
			}
		})
	}
}

func TestProviderLocalPrimitiveReservesOnlyEachClosedHeaderCommand(t *testing.T) {
	s, policy, candidate, primitive := providerStoreFixture(t)
	c, err := newProviderIntegration(s, policy, candidate, primitive)
	if err != nil {
		t.Fatal(err)
	}
	c.firstExchange, c.firstResult = admissionTestDigest("a"), `{"normal":"inert bounded input"}`
	bootstrap, err := c.localPrimitive(c.firstExchange, nil)
	if err != nil || bootstrap.blocks != 1 || bootstrap.transactions != 0 || bootstrap.frames != 0 || bootstrap.logs != 0 || bootstrap.proofs != 0 || bootstrap.capture != primitive.capture || bootstrap.maximumCreatedObjects != primitive.maximumCreatedObjects {
		t.Fatal("bootstrap reserved unrelated EVM replay work", err)
	}
	c.bootstrap = admissionTestDigest("b")
	raw := providerPinInput(c.bootstrap, "fixture-agent")
	input := providerCandidateHash("sage-provider-owned-pin-input-v1", raw)
	objects := []providerIntegrationObjectInput{{"pin-input", raw}}
	pin, err := c.localPrimitive(input, objects)
	if err != nil || pin != bootstrap {
		t.Fatal("closed pin did not keep the same exact header bound", err)
	}
	for _, mode := range []string{"unknown", "extra-object", "wrong-kind", "wrong-input", "wrong-raw", "already-pinned"} {
		t.Run(mode, func(t *testing.T) {
			selected := input
			extra := append([]providerIntegrationObjectInput(nil), objects...)
			switch mode {
			case "unknown":
				selected, extra = admissionTestDigest("f"), nil
			case "extra-object":
				extra = append(extra, extra[0])
			case "wrong-kind":
				extra[0].kind = "unknown"
			case "wrong-input":
				selected = admissionTestDigest("f")
			case "wrong-raw":
				extra[0].raw = []byte(`{}`)
			case "already-pinned":
				c.pin = admissionTestDigest("f")
				defer func() { c.pin = "" }()
			}
			if _, err := c.localPrimitive(selected, extra); err == nil {
				t.Fatal("unsupported input borrowed a closed command's reservation")
			}
		})
	}
}

func TestProviderIntegrationPinInputAndRequestsAreExact(t *testing.T) {
	start, selected := admissionTestDigest("a"), admissionTestDigest("b")
	raw := providerPinInput(start, "Agent_1")
	m := contentObject(contentJSON(raw), "format", "bootstrap_start", "registry", "agent")
	if m == nil || m["bootstrap_start"] != start || m["agent"] != "Agent_1" {
		t.Fatal("missing pin input binding")
	}
	for _, input := range []struct{ start, agent string }{{"", "Agent_1"}, {start, ""}, {start, strings.Repeat("a", 1025)}} {
		if providerPinInput(input.start, input.agent) != nil {
			t.Fatal("unsupported pin input")
		}
	}
	requests, err := providerPinRequests(selected)
	if err != nil || requests[0].method != "eth_chainId" || requests[1].params != `["0x0",false]` || requests[2].method != "web3_clientVersion" || requests[3].method != "eth_call" || requests[0].firstFinalized() {
		t.Fatal("pin substituted another finalized start", err)
	}
	if !strings.Contains(requests[3].params, selected) || !strings.Contains(requests[3].params, `"requireCanonical":true`) || !strings.Contains(requests[3].params, "0x04ef898d") {
		t.Fatal("registry identity not bound to selected block")
	}
	if _, err := providerPinRequests("unknown"); err == nil {
		t.Fatal("unbound selected hash")
	}
}

func TestProviderIntegrationPinResultsValidateAllIdentities(t *testing.T) {
	valid := providerPinResultFixture(t)
	if !providerCheckPinResults(valid) {
		t.Fatal("normal pin identities")
	}
	for _, bad := range []struct {
		name string
		at   int
		raw  string
	}{
		{"chain", 0, `"0x1"`}, {"chain-shape", 0, `{}`},
		{"genesis-shape", 1, `null`}, {"genesis-hash", 1, `{"hash":"` + admissionTestDigest("a") + `","parentHash":"` + admissionTestDigest("b") + `","number":"0x0","timestamp":"0x1"}`},
		{"version-shape", 2, `[]`}, {"version-empty", 2, `""`}, {"version-size", 2, `"` + strings.Repeat("a", 1025) + `"`},
		{"registry-abi", 3, `"0x00"`}, {"registry-id", 3, `"` + returnABI(t, "registryId", "eip155:1:other") + `"`},
	} {
		t.Run(bad.name, func(t *testing.T) {
			fixture := valid
			fixture[bad.at] = json.RawMessage(bad.raw)
			if providerCheckPinResults(fixture) {
				t.Fatal("foreign identity accepted")
			}
		})
	}
}

func TestProviderIntegrationHeaderParserRefusesIncompleteData(t *testing.T) {
	for _, raw := range []string{`{`, `{}`, `{"number":"0x00"}`, `{"hash":"` + admissionTestDigest("a") + `","parentHash":"` + admissionTestDigest("b") + `","number":"0x1","timestamp":"0x20000000000000"}`} {
		if _, err := providerDecodeHeader([]byte(raw)); err == nil {
			t.Fatal("unsupported header")
		}
	}
}

func TestProviderIntegrationPinRefusalsDoNotDispatch(t *testing.T) {
	var empty *providerIntegration
	if _, err := empty.pinAgent(context.Background(), "Agent_1"); err == nil {
		t.Fatal("nil integration")
	}
	store, policy, candidate, primitive := providerStoreFixture(t)
	c, err := newProviderIntegration(store, policy, candidate, primitive)
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []context.Context{nil, context.Background()} {
		if _, err := c.pinAgent(ctx, "Agent_1"); err == nil {
			t.Fatal("pin before the retained bootstrap")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.pinAgent(ctx, "Agent_1"); err == nil || c.ledger.lastConsumption != nil {
		t.Fatal("cancelled pin consumed a request")
	}
}
