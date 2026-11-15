package registry011

import (
	"reflect"
	"testing"
)

func TestProviderHistoryWorkingStateMutationsKeepOriginalInputsAcrossBudgetCuts(t *testing.T) {
	// mutateObserved operates on an uncommitted working copy. It may change
	// that private copy before refusal; only applyObserved publishes a state.
	// Every trial below reconstructs its private copy before entering a fresh
	// arithmetic UNIT observer and never lends it a live store role.
	key := replayTestKey(t, "b", true)
	for _, tc := range []struct {
		name, operation string
		prefix          int
		args            map[string]any
	}{
		{"create-empty-services", "create", 0, nil},
		{"activate", "activate", 1, map[string]any{}},
		{"deactivate", "deactivate", 4, map[string]any{}},
		{"add-key", "add-key", 4, map[string]any{"key": key}},
		{"revoke-signing-key", "revoke-key", 4, map[string]any{"name": "a"}},
		{"update-services", "update-services", 4, map[string]any{"services": []replayService{{Name: "api", Type: "A2A", URI: "https://example.invalid/api"}}}},
		{"authorize-operator", "authorize-operator", 1, map[string]any{"operator": replayTestOperator, "scope": "activate"}},
		{"revoke-operator", "revoke-operator", 2, map[string]any{"operator": replayTestOperator, "scope": "activate"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := providerHistoryStateUnitPrefix(t, tc.prefix)
			before := providerHistoryStateUnitSnapshot(t, original)
			command := replayTestCreate(t)
			if tc.operation == "create" {
				// The empty/nil service representation is private mutation DATA;
				// the resulting created record still normalizes it to [].
				command.services = nil
			} else {
				command = replayTestCommand(t, tc.operation, replayTestWallet, original.version, tc.args)
			}
			keysBefore := replayTestClone(t, command.keys)
			servicesBefore := replayTestClone(t, command.services)
			historyUnitConversionRefusalPrefixes(t, func(m *providerWorkMeter) (bool, bool) {
				working := providerHistoryStateUnitSnapshot(t, original)
				added, ok := working.mutateObserved(replayTestRegistry, command, uint64(tc.prefix+1), m)
				if !reflect.DeepEqual(original, before) || !reflect.DeepEqual(command.keys, keysBefore) ||
					!reflect.DeepEqual(command.services, servicesBefore) {
					t.Fatal("working mutation changed its original state or command")
				}
				if ok && tc.operation == "create" && (working.record.State != "created" || working.record.Services == nil || len(working.record.Services) != 0) {
					t.Fatal("normal creation did not preserve the empty service list")
				}
				return ok, ok || added == nil
			})
		})
	}
}

func TestProviderHistoryStateCollectionGuardsDoNotGrantAuthority(t *testing.T) {
	m := providerReplayUnitMeter()
	if replayStateHasSignerObserved(nil, 0, nil) || replayStateHasSignerObserved(make([]replayKey, 129), 1, m) {
		t.Fatal("missing observer or excessive key list supplied a signer")
	}
	var absent *replayState
	if absent.authorizedObserved(replayCommand{}, m) || (&replayState{}).authorizedObserved(replayCommand{}, nil) ||
		(&replayState{grants: make([]replayGrant, 129)}).authorizedObserved(replayCommand{}, m) {
		t.Fatal("missing or excessive state supplied authorization")
	}
	state := replayState{record: replayRecord{Controller: replayTestWallet, State: "active"},
		grants: []replayGrant{{Operator: replayTestOperator, Scope: "deactivate"}}}
	command := replayCommand{actor: "0x0000000000000000000000000000000000003000", op: "deactivate"}
	before := providerHistoryStateUnitSnapshot(t, state)
	if state.authorizedObserved(command, m) || !reflect.DeepEqual(state, before) {
		t.Fatal("an unrelated account acquired a grant or changed its source")
	}
	if replayStateGrantAllowedObserved("active", "deactivate", nil) {
		t.Fatal("missing observer allowed a grant")
	}
}
