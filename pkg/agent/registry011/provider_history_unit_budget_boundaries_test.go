package registry011

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

func TestProviderHistoryABIParsingPreservesNativeErrorsAndObservedPrefixes(t *testing.T) {
	const definition = `[{"type":"function","name":"version","inputs":[],"outputs":[{"type":"uint256"}]}]`
	for _, raw := range []string{"", "{", "[true]"} {
		m := providerReplayUnitMeter()
		if _, err := replayABIObserved(raw, m); err == nil || m.failed || m.actual[2] != uint64(len(raw)) {
			t.Fatal("native ABI error was hidden or skipped observation", raw, err, m.actual)
		}
	}
	if _, err := replayABIObserved(definition, nil); err == nil {
		t.Fatal("observed ABI parsing accepted a missing observer")
	}
	for _, fuel := range []uint64{0, uint64(len(definition) + 1)} {
		m := providerReplayUnitMeter()
		m.upper[10] = fuel
		if got, err := replayABIObserved(definition, m); err == nil || len(got.Methods) != 0 || !m.failed || m.actual[2] != 0 || m.actual[10] != fuel {
			t.Fatal("ABI copy/parse boundary exceeded its original prefix", fuel, err, m.actual)
		}
		before := m.actual
		if _, err := replayABIObserved(definition, m); err == nil || m.actual != before {
			t.Fatal("failed ABI observer was reused")
		}
	}
	observed := providerReplayUnitMeter()
	contract, outcomes, err := replayABIsObserved(observed)
	legacy, oldOutcomes, oldErr := replayABIs()
	if err != nil || oldErr != nil || len(contract.Methods) != len(legacy.Methods) || len(outcomes.Events) != len(oldOutcomes.Events) ||
		observed.actual[2] != uint64(len(replayContractABI)+len(replayOutcomesABI)) {
		t.Fatal("fixed ABI parsing changed its definitions or native input accounting", err, oldErr)
	}
	for name, method := range legacy.Methods {
		if !bytes.Equal(method.ID, contract.Methods[name].ID) {
			t.Fatal("fixed method identity changed", name)
		}
	}
	for name, event := range oldOutcomes.Events {
		if event.ID != outcomes.Events[name].ID {
			t.Fatal("fixed event identity changed", name)
		}
	}
	for _, fuel := range []uint64{0, observed.actual[10] - 1} {
		m := providerReplayUnitMeter()
		m.upper[10] = fuel
		contract, outcomes, err := replayABIsObserved(m)
		if err == nil || len(contract.Methods) != 0 || len(outcomes.Events) != 0 || !m.failed || m.actual[10] > fuel {
			t.Fatal("partial fixed ABI pair escaped a failed original observer", fuel, err)
		}
	}
}

func TestProviderHistoryFrameStorageRefusesUnfundedAllocationAndKeepsItsPrefix(t *testing.T) {
	for _, tc := range []struct {
		name      string
		fuel      uint64
		keptCalls int
		keptWork  uint64
	}{
		{"before-calls", nodeFrameCopyBytes - 1, 0, 0},
		{"before-logs", nodeFrameCopyBytes + 1, 1, nodeFrameCopyBytes + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := providerReplayUnitMeter()
			m.upper[10] = tc.fuel
			frame := &ReplayFrame{}
			if nodeFrameStorageObserved(frame, 1, 1, m) || !m.failed || m.actual[10] != tc.keptWork ||
				len(frame.Calls) != tc.keptCalls || frame.Logs != nil {
				t.Fatal("unfunded frame storage allocated or lost a completed prefix", m.actual, frame)
			}
			before := m.actual
			if nodeFrameStorageObserved(&ReplayFrame{}, 1, 1, m) || m.actual != before {
				t.Fatal("refused storage renewed its original observer")
			}
		})
	}
}

func TestProviderHistoryCanonicalJSONKeepsExplicitExtensionBoundary(t *testing.T) {
	value := types.Withdrawal{Index: 1, Validator: 2, Amount: 3}
	fields, raw := providerHistoryNodeJSONUnitFields(t, value)
	fields["displayName"] = json.RawMessage(`"retained separately"`)
	before := replayTestJSON(t, fields)
	if nodeJSONCanonical(fields, value, "") || nodeJSONCanonicalObserved(fields, value, "", len(raw), providerReplayUnitMeter()) {
		t.Fatal("unlisted extension became a canonical typed field")
	}
	if !nodeJSONCanonical(fields, value, "displayName") || !nodeJSONCanonicalObserved(fields, value, "displayName", len(raw), providerReplayUnitMeter()) ||
		!bytes.Equal(before, replayTestJSON(t, fields)) {
		t.Fatal("explicit extension boundary changed the typed value or original fields")
	}
}

// Every cut is selected on a fresh arithmetic UNIT observer before invocation.
// These helpers never enter a store, reserve a live role, use a clock or send
// traffic. A failed attempt may retain its charged prefix but no partial output.
// Actual work is only a necessary minimum, not an incoming preflight upper.
// The separate decimal case below checks that distinction explicitly.
func TestProviderHistoryStateAndCommandCopiesKeepEveryBudgetPrefix(t *testing.T) {
	state := providerHistoryStateUnitPrefix(t, 1)
	expires := uint64(1000)
	state.record.Keys[0].Expires = &expires
	state.record.Services = []replayService{{Name: "api", Type: "A2A", URI: "https://example.invalid/api"}}
	state.grants = []replayGrant{{Operator: replayTestOperator, Scope: "activate"}}
	before := providerHistoryStateUnitSnapshot(t, state)
	for _, tc := range []struct {
		name string
		run  func(*providerWorkMeter) (bool, bool)
	}{
		{"command-object", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayGeneratedObjectObserved([]string{"name", "value"}, []any{"api", "ordinary"}, m)
			return ok, ok || out == nil
		}},
		{"state-object", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayStateObjectObserved([]string{"name", "value"}, []any{"api", "ordinary"}, m)
			return ok, ok || out == nil
		}},
		{"key-clone", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayStateKeyCloneObserved(state.record.Keys[0], m)
			return ok, ok || reflect.DeepEqual(out, replayKey{})
		}},
		{"service-clone", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayStateServicesCloneObserved(state.record.Services, m)
			return ok, ok || out == nil
		}},
		{"state-clone", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayStateCloneObserved(state, m)
			return ok, ok || reflect.DeepEqual(out, replayState{})
		}},
		{"keys-value", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayStateKeysValueObserved(state.record.Keys, m)
			return ok, ok || out == nil
		}},
		{"services-value", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayStateServicesValueObserved(state.record.Services, m)
			return ok, ok || out == nil
		}},
		{"grants-value", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayStateGrantsValueObserved(state.grants, m)
			return ok, ok || out == nil
		}},
		{"split-grants", func(m *providerWorkMeter) (bool, bool) {
			removed, retained, ok := replayStateSplitGrantsObserved(state.grants, true, false, m)
			return ok, ok || removed == nil && retained == nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			baseline := providerReplayUnitMeter()
			completed, closed := tc.run(baseline)
			if !completed || !closed || baseline.failed || baseline.actual[10] == 0 || baseline.actual[10] > 65536 {
				t.Fatal("finite normal copy subject did not complete", baseline.actual)
			}
			for fuel := uint64(0); fuel < baseline.actual[10]; fuel++ {
				m := providerReplayUnitMeter()
				m.upper[10] = fuel
				ok, closed := tc.run(m)
				if !closed || ok || !m.failed || m.actual[10] > fuel ||
					!reflect.DeepEqual(state, before) {
					t.Fatal("quota cut changed input or escaped its original prefix", fuel, ok, m.actual)
				}
				if !ok {
					prefix := m.actual
					again, closed := tc.run(m)
					if again || !closed || m.actual != prefix {
						t.Fatal("failed arithmetic observer accepted a second invocation", fuel)
					}
				}
			}
		})
	}
}

func TestProviderHistoryDecimalPreflightHeadroomRemainsDistinctFromActual(t *testing.T) {
	for _, tc := range []struct {
		value  uint64
		text   string
		actual uint64
	}{
		{0, "0", 2},
		{1000, "1000", 17},
		{^uint64(0), "18446744073709551615", 401},
	} {
		for _, fuel := range []uint64{tc.actual - 1, tc.actual, 400, 401} {
			m := providerReplayUnitMeter()
			m.upper[10] = fuel
			out, ok := replayGeneratedUintObserved(tc.value, m)
			if fuel >= 401 {
				if !ok || out != tc.text || m.failed || m.actual != (candidateWorkVector{10: tc.actual}) {
					t.Fatal("decimal actual replaced the original twenty-digit preflight", tc.value, fuel, out, m.actual)
				}
				continue
			}
			if ok || out != "" || !m.failed || m.actual != (candidateWorkVector{}) {
				t.Fatal("insufficient incoming headroom entered decimal encoding", tc.value, fuel, out, m.actual)
			}
			before := m.actual
			if out, ok := replayGeneratedUintObserved(tc.value, m); ok || out != "" || m.actual != before {
				t.Fatal("failed decimal observer was renewed", tc.value, fuel)
			}
		}
	}
}
