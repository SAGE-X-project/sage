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
