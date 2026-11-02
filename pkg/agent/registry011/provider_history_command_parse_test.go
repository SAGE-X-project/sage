package registry011

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func TestProviderHistoryCommandParserAndDecodedCallerPreserveEightMutations(t *testing.T) {
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, step := range replayTestSteps(t) {
		want, accepted := replayParseCommand(replayTestRegistry, replayTestAgent, step.History.Command, contract)
		if !accepted {
			t.Fatal("normal command precursor")
		}
		m := providerHistoryExecutionUnitMeter()
		got, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, step.History.Command, contract, m)
		if !ok || !reflect.DeepEqual(got, want) || m.failed || m.actual[2] < uint64(2*len(step.History.Command)) {
			t.Fatal("strict command consumer semantics", want.op, m.actual)
		}
		decoded, ok := replayDecodeCommandObserved(replayTestRegistry, want.actor, want.calldata, contract, providerHistoryExecutionUnitMeter())
		if !ok || !reflect.DeepEqual(decoded, want) {
			t.Fatal("actual decoded caller did not use strict consumer", want.op)
		}
		legacy, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, step.History.Command, contract, nil)
		if !ok || !reflect.DeepEqual(legacy, want) {
			t.Fatal("nil legacy consumer changed")
		}
		seen[want.op] = true
		raw := bytes.Clone(step.History.Command)
		owned, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, raw, contract, providerHistoryExecutionUnitMeter())
		if !ok {
			t.Fatal("owned raw precursor")
		}
		raw[0] = '['
		if !bytes.Equal(owned.raw, step.History.Command) {
			t.Fatal("consumer retained mutable external raw bytes")
		}
	}
	if len(seen) != 8 {
		t.Fatal("all adopted mutation consumers required")
	}
}

func TestProviderHistoryCommandConsumerRejectsExistingRootViolations(t *testing.T) {
	contract, _, _ := replayABIs()
	base := replayTestCreate(t)
	for _, tc := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"binding", func(m map[string]any) { m["binding"] = "sage-0.10.0" }},
		{"registry", func(m map[string]any) { m["registry"] = "wrong" }},
		{"did", func(m map[string]any) { m["did"] = "did:other:a" }},
		{"operation", func(m map[string]any) { m["operation"] = "transfer" }},
		{"actor", func(m map[string]any) { m["actor"] = "invalid" }},
		{"field", func(m map[string]any) { m["extra"] = 0 }},
		{"expected-type", func(m map[string]any) { m["expectedVersion"] = 0 }},
		{"expected-create", func(m map[string]any) { m["expectedVersion"] = "1" }},
		{"controller", func(m map[string]any) { m["arguments"].(map[string]any)["controller"] = replayTestOperator }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := contentEdit(t, base.raw, tc.edit)
			if _, ok := replayParseCommand(replayTestRegistry, replayTestAgent, raw, contract); ok {
				t.Fatal("existing parser must first reject the single change")
			}
			if got, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, raw, contract, providerHistoryExecutionUnitMeter()); ok || !reflect.DeepEqual(got, replayCommand{}) {
				t.Fatal("invalid root returned a partial command")
			}
		})
	}
}

func TestProviderHistoryCommandPackPreflightFailureKeepsOriginalActualPrefix(t *testing.T) {
	contract, _, _ := replayABIs()
	base := replayTestCreate(t)
	probe := providerHistoryExecutionUnitMeter()
	got, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, base.raw, contract, probe)
	if !ok || !bytes.Equal(got.calldata, base.calldata) {
		t.Fatal("complete strict-consumer precursor")
	}
	encoded := uint64(len(base.calldata))*uint64(len(base.calldata)) + 1
	if probe.actual[10] < encoded {
		t.Fatal("actual pack invocation missing")
	}
	prefix := probe.actual[10] - encoded
	maximum := uint64(replayABICallMaximum("revealClaim"))
	// All earlier work is funded; the actual final encoder preflight is one
	// unit short. Its upper is not refunded or reported as encoded bytes.
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 1 << 30, 10: prefix + maximum*maximum})
	if got, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, base.raw, contract, m); ok || !reflect.DeepEqual(got, replayCommand{}) || !m.failed || m.actual[10] != prefix {
		t.Fatal("late Pack preflight or actual-prefix accounting", m.actual, prefix)
	}
	before := m.actual
	if got, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, base.raw, contract, m); ok || !reflect.DeepEqual(got, replayCommand{}) || m.actual != before {
		t.Fatal("failed original command consumer resumed")
	}
}

func TestProviderHistoryCommandConsumerAndParameterGuardsRefuseMissingInputs(t *testing.T) {
	contract, _, _ := replayABIs()
	base := replayTestCreate(t)
	for _, raw := range [][]byte{nil, []byte(`{}`), make([]byte, 262145)} {
		if got, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, raw, contract, providerHistoryExecutionUnitMeter()); ok || !reflect.DeepEqual(got, replayCommand{}) {
			t.Fatal("missing/oversized command")
		}
	}
	for _, tc := range []struct{ registry, agent string }{{"wrong", replayTestAgent}, {replayTestRegistry, ""}, {replayTestRegistry, "bad.name!"}} {
		if got, ok := replayParseCommandObserved(tc.registry, tc.agent, base.raw, contract, providerHistoryExecutionUnitMeter()); ok || !reflect.DeepEqual(got, replayCommand{}) {
			t.Fatal("pinned consumer identity")
		}
	}
	m := providerHistoryExecutionUnitMeter()
	if got, ok := replayParseCommandObserved(replayTestRegistry, replayTestAgent, base.raw, abi.ABI{}, m); ok || !reflect.DeepEqual(got, replayCommand{}) || !m.failed {
		t.Fatal("missing private ABI")
	}
	for _, count := range []int{1, 6} {
		if got, ok := replayCommandParamsObserved(count, providerHistoryExecutionUnitMeter()); ok || got != nil {
			t.Fatal("parameter count")
		}
	}
	if got, ok := replayCommandParamsObserved(2, nil); ok || got != nil {
		t.Fatal("missing parameter observer")
	}
	if replayCommandParamObserved(nil, 0, nil, providerHistoryExecutionUnitMeter()) || replayCommandParamObserved(make([]any, 2), -1, nil, providerHistoryExecutionUnitMeter()) || replayCommandParamObserved(make([]any, 2), 0, nil, nil) {
		t.Fatal("parameter index/observer")
	}
	if got, ok := replayCommandABIKeysObserved(make([]replayKey, 129), providerHistoryExecutionUnitMeter()); ok || got != nil {
		t.Fatal("ABI key count")
	}
	if got, ok := replayCommandABIServicesObserved(make([]replayService, 17), providerHistoryExecutionUnitMeter()); ok || got != nil {
		t.Fatal("ABI service count")
	}
}
