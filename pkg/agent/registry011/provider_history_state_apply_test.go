package registry011

import (
	"bytes"
	"encoding/base64"
	"reflect"
	"strconv"
	"testing"
)

func providerHistoryStateUnitPrefix(t *testing.T, count int) replayState {
	t.Helper()
	steps := replayTestSteps(t)
	contract, _, err := replayABIs()
	if err != nil {
		t.Fatal(err)
	}
	var state replayState
	for i := 0; i < count; i++ {
		c, ok := replayParseCommand(replayTestRegistry, replayTestAgent, steps[i].History.Command, contract)
		if !ok {
			t.Fatal("old parser precursor")
		}
		if step, ok := state.apply(replayTestRegistry, c, uint64(i+1)); !ok || !reflect.DeepEqual(step, steps[i]) {
			t.Fatal("independent normal prefix", i)
		}
	}
	return state
}

func providerHistoryStateUnitSnapshot(t *testing.T, state replayState) replayState {
	t.Helper()
	return replayState{record: replayTestClone(t, state.record), grants: replayTestClone(t, state.grants), version: state.version, admitted: state.admitted}
}

func TestProviderHistoryObservedStatePreservesAllHandwrittenMutations(t *testing.T) {
	steps := replayTestSteps(t)
	contract, _, _ := replayABIs()
	var state, legacy replayState
	m := providerHistoryStateUnitMeter()
	for i, want := range steps {
		c, ok := replayParseCommand(replayTestRegistry, replayTestAgent, want.History.Command, contract)
		if !ok {
			t.Fatal("existing actual command precursor")
		}
		old, accepted := legacy.apply(replayTestRegistry, c, uint64(i+1))
		got, ok := state.applyObserved(replayTestRegistry, c, uint64(i+1), m)
		if !accepted || !ok || !reflect.DeepEqual(old, want) || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(state, legacy) || m.failed {
			t.Fatal("same result/state for all eight operations", i, c.op, m.actual)
		}
		before := bytes.Clone(c.raw)
		got.History.Command[0] ^= 1
		if !bytes.Equal(c.raw, before) {
			t.Fatal("returned command aliases actual input")
		}
	}
	if m.actual[7] != 3 {
		t.Fatal("two initial proofs and one added key; retained proofs not repeated", m.actual)
	}
	var nilState replayState
	c := replayTestCreate(t)
	got, ok := nilState.applyObserved(replayTestRegistry, c, 1, nil)
	if !ok || !reflect.DeepEqual(got, steps[0]) {
		t.Fatal("nil legacy state path")
	}
}

func TestProviderHistoryObservedStateKeepsExistingACLAndLifecycleRefusals(t *testing.T) {
	key := replayTestKey(t, "b", true)
	for _, tc := range []struct {
		name, operation, actor string
		prefix                 int
		args                   map[string]any
	}{
		{"operator-cannot-grant", "authorize-operator", replayTestOperator, 3, map[string]any{"operator": "0x0000000000000000000000000000000000003000", "scope": "activate"}},
		{"operator-cannot-revoke-created", "revoke-key", replayTestOperator, 3, map[string]any{"name": "a"}},
		{"created-cannot-add", "add-key", replayTestWallet, 1, map[string]any{"key": key}},
		{"created-cannot-update", "update-services", replayTestWallet, 1, map[string]any{"services": []replayService{}}},
		{"active-cannot-activate", "activate", replayTestWallet, 4, map[string]any{}},
		{"controller-cannot-delegate-self", "authorize-operator", replayTestWallet, 4, map[string]any{"operator": replayTestWallet, "scope": "add-key"}},
		{"active-cannot-grant-activate", "authorize-operator", replayTestWallet, 4, map[string]any{"operator": replayTestOperator, "scope": "activate"}},
		{"created-cannot-grant-add", "authorize-operator", replayTestWallet, 1, map[string]any{"operator": replayTestOperator, "scope": "add-key"}},
		{"missing-key", "revoke-key", replayTestWallet, 4, map[string]any{"name": "absent"}},
		{"service-key-namespace", "update-services", replayTestWallet, 4, map[string]any{"services": []replayService{{Name: "a", Type: "A2A", URI: "https://example.invalid/api"}}}},
		{"missing-grant", "revoke-operator", replayTestWallet, 4, map[string]any{"operator": replayTestOperator, "scope": "add-key"}},
		{"duplicate-grant", "authorize-operator", replayTestWallet, 3, map[string]any{"operator": replayTestOperator, "scope": "activate"}},
		{"terminal-cannot-change", "activate", replayTestWallet, 11, map[string]any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := providerHistoryStateUnitPrefix(t, tc.prefix)
			before := providerHistoryStateUnitSnapshot(t, state)
			c := replayTestCommand(t, tc.operation, tc.actor, state.version, tc.args)
			legacy := providerHistoryStateUnitSnapshot(t, state)
			if _, accepted := legacy.apply(replayTestRegistry, c, uint64(tc.prefix+1)); accepted {
				t.Fatal("old refusal precursor unexpectedly valid")
			}
			if step, ok := state.applyObserved(replayTestRegistry, c, uint64(tc.prefix+1), providerHistoryStateUnitMeter()); ok || !reflect.DeepEqual(step, HistoryContentStep{}) || !reflect.DeepEqual(state, before) {
				t.Fatal("refusal changed the owned original state", tc.name)
			}
		})
	}
}

func TestProviderHistoryObservedStateProofFailurePublishesNoWorkingState(t *testing.T) {
	state := providerHistoryStateUnitPrefix(t, 4)
	before := providerHistoryStateUnitSnapshot(t, state)
	key := replayTestKey(t, "b", true)
	signature, _ := replayBinary(key.Proof.Value, 64)
	signature[0] ^= 1
	key.Proof.Value = base64.RawURLEncoding.EncodeToString(signature)
	c := replayTestCommand(t, "add-key", replayTestWallet, state.version, map[string]any{"key": key})
	m := providerHistoryStateUnitMeter()
	if step, ok := state.applyObserved(replayTestRegistry, c, 5, m); ok || !reflect.DeepEqual(step, HistoryContentStep{}) || !reflect.DeepEqual(state, before) || m.actual[7] != 1 || m.actual[10] == 0 {
		t.Fatal("late failed proof published its working state or hid work", m.actual)
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{})
	if step, ok := state.applyObserved(replayTestRegistry, c, 5, short); ok || !reflect.DeepEqual(step, HistoryContentStep{}) || !reflect.DeepEqual(state, before) || !short.failed {
		t.Fatal("unfunded state observer")
	}
	actual := short.actual
	if _, ok := state.applyObserved(replayTestRegistry, c, 5, short); ok || short.actual != actual || !reflect.DeepEqual(state, before) {
		t.Fatal("failed state observer reset")
	}
}

func TestProviderHistoryObservedStateRevokingLastSignerRemovesGrants(t *testing.T) {
	key := replayTestKey(t, "a", false)
	create := replayTestCommand(t, "create", replayTestWallet, 0, map[string]any{"controller": replayTestWallet, "keys": []replayKey{key}, "services": []replayService{}, "salt": base64.RawURLEncoding.EncodeToString(make([]byte, 32))})
	var state, legacy replayState
	m := providerHistoryStateUnitMeter()
	commands := []replayCommand{create,
		replayTestCommand(t, "activate", replayTestWallet, 1, map[string]any{}),
		replayTestCommand(t, "authorize-operator", replayTestWallet, 2, map[string]any{"operator": replayTestOperator, "scope": "deactivate"}),
		replayTestCommand(t, "revoke-key", replayTestWallet, 3, map[string]any{"name": "a"}),
	}
	for i, c := range commands {
		want, accepted := legacy.apply(replayTestRegistry, c, uint64(i+1))
		got, ok := state.applyObserved(replayTestRegistry, c, uint64(i+1), m)
		if !accepted || !ok || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(state, legacy) {
			t.Fatal("last signer terminal cleanup", c.op)
		}
	}
	if state.record.State != "deactivated" || len(state.grants) != 0 || state.record.Keys[0].State != "revoked" || m.actual[7] != 1 {
		t.Fatal("terminal result or historical proof count")
	}
}

func TestProviderHistoryObservedStateKeepsVersionAndTimeBoundaries(t *testing.T) {
	state := providerHistoryStateUnitPrefix(t, 4)
	state.version = ^uint64(0) - 1
	state.record.Version = strconv.FormatUint(state.version, 10)
	c := replayTestCommand(t, "deactivate", replayTestWallet, state.version, map[string]any{})
	legacy := providerHistoryStateUnitSnapshot(t, state)
	want, accepted := legacy.apply(replayTestRegistry, c, 5)
	got, ok := state.applyObserved(replayTestRegistry, c, 5, providerHistoryStateUnitMeter())
	if !accepted || !ok || !reflect.DeepEqual(got, want) || state.version != ^uint64(0) || state.record.Version != "18446744073709551615" {
		t.Fatal("full uint64 terminal version changed")
	}
	before := providerHistoryStateUnitSnapshot(t, state)
	c = replayTestCommand(t, "deactivate", replayTestWallet, state.version, map[string]any{})
	if _, ok := state.applyObserved(replayTestRegistry, c, 6, providerHistoryStateUnitMeter()); ok || !reflect.DeepEqual(state, before) {
		t.Fatal("exhausted state changed")
	}
	for _, now := range []uint64{0, 1 << 53} {
		state = providerHistoryStateUnitPrefix(t, 1)
		before = providerHistoryStateUnitSnapshot(t, state)
		c = replayTestCommand(t, "activate", replayTestWallet, 1, map[string]any{})
		if _, ok := state.applyObserved(replayTestRegistry, c, now, providerHistoryStateUnitMeter()); ok || !reflect.DeepEqual(state, before) {
			t.Fatal("invalid monotonic/safe admitted time")
		}
	}
}
