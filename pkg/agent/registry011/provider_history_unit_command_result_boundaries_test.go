package registry011

import (
	"reflect"
	"testing"
)

func TestProviderHistoryGeneratedCommandValuesCloseEveryUnfundedPrefix(t *testing.T) {
	create := replayTestCreate(t)
	keys, ok := replayCommandABIKeysObserved(create.keys, providerReplayUnitMeter())
	if !ok || len(keys) != 2 {
		t.Fatal("ordinary original key tuple precursor")
	}
	keys[0].HasExpiry, keys[0].Expires = true, 1000
	services := []replayABIService{{Name: "api", ServiceType: "A2A", URI: "https://example.invalid/api"}}
	values := providerHistoryTupleUnitValues(t)
	for _, tc := range []struct {
		name string
		run  func(*providerWorkMeter) (bool, bool)
	}{
		{"key", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayGeneratedKeyObserved(keys[0], m)
			return ok, ok || out == nil
		}},
		{"keys", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayGeneratedKeysObserved(keys, m)
			return ok, ok || out == nil
		}},
		{"services", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayGeneratedServicesObserved(services, m)
			return ok, ok || out == nil
		}},
		{"create-arguments", func(m *providerWorkMeter) (bool, bool) {
			out, ok := replayGeneratedArgumentsObserved("create", values, m)
			return ok, ok || out == nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) { historyUnitConversionRefusalPrefixes(t, tc.run) })
	}
}

func TestProviderHistoryExpectedCommandDataDoesNotPublishAnUnfundedPendingValue(t *testing.T) {
	command := replayTestCreate(t)
	before := string(command.raw)
	historyUnitConversionRefusalPrefixes(t, func(m *providerWorkMeter) (bool, bool) {
		// This private string-copy subject deliberately has no selected fold,
		// run or live authority. prepareSubject's separate caller seal is not
		// bypassed or represented by this lower-level DATA result.
		window := providerHistoryExpectedWindow{budget: replayBudget{observer: m}}
		ok := window.prepareSubjectObserved(command)
		if string(command.raw) != before || ok && (!window.pending || window.pendingVersion != 1 || window.pendingCommand != before) {
			t.Fatal("pending command copy changed its original or normal value")
		}
		return ok, ok || !window.pending && window.pendingVersion == 0 && window.pendingCommand == ""
	})
	var absent *providerHistoryExpectedWindow
	if absent.historyEqualObserved(1, []byte("entry"), []byte("command")) {
		t.Fatal("missing expected window supplied equality")
	}
	for _, fuel := range []uint64{0, 1 << 60} {
		m := providerReplayUnitMeter()
		m.upper[10] = fuel
		window := providerHistoryExpectedWindow{budget: replayBudget{observer: m},
			queue: []providerHistoryExpectedItem{{version: 1, entry: "entry", command: "command"}}}
		before := append([]providerHistoryExpectedItem(nil), window.queue...)
		got := window.historyEqualObserved(1, []byte("entry"), []byte("command"))
		if got != (fuel != 0) || window.compared != 0 || !reflect.DeepEqual(window.queue, before) || fuel == 0 && (!m.failed || m.actual != (candidateWorkVector{})) {
			t.Fatal("equality borrowed work or changed the expected queue", got, m.actual)
		}
	}
}

func TestProviderHistorySemanticStreamCopyClosesUnfundedOwnedGraphs(t *testing.T) {
	original := replayStream{state: providerHistoryStateUnitPrefix(t, 1),
		claims:  map[replayClaimID]replayClaimState{{actor: replayTestWallet, digest: [32]byte{1}}: {block: 1}},
		created: map[string]bool{replayTestAgent: true}}
	stateBefore := providerHistoryStateUnitSnapshot(t, original.state)
	config := JournalConfig{MaxClaims: 1, MaxIdentities: 1}
	historyUnitConversionRefusalPrefixes(t, func(m *providerWorkMeter) (bool, bool) {
		out, ok := providerHistorySemanticStreamCopyObserved(original, config, m)
		if !reflect.DeepEqual(original.state, stateBefore) || len(original.claims) != 1 || len(original.created) != 1 ||
			!original.created[replayTestAgent] || original.claims[replayClaimID{actor: replayTestWallet, digest: [32]byte{1}}] != (replayClaimState{block: 1}) {
			t.Fatal("semantic graph copy changed its original state or indexes")
		}
		if ok && !reflect.DeepEqual(out, original) {
			t.Fatal("semantic graph copy changed its normal complete value")
		}
		return ok, ok || reflect.DeepEqual(out, replayStream{})
	})
}

func TestProviderHistoryPrivateWorkingMutationRejectsUnsupportedOrRepeatedCreation(t *testing.T) {
	for _, mode := range []string{"unsupported-operation", "repeated-create", "missing-signer"} {
		state := providerHistoryStateUnitPrefix(t, 1)
		command := replayTestCreate(t)
		switch mode {
		case "unsupported-operation":
			command.op = "unsupported"
		case "missing-signer":
			state, command.keys = replayState{}, nil
		}
		before := providerHistoryStateUnitSnapshot(t, state)
		if out, ok := state.mutateObserved(replayTestRegistry, command, 2, providerReplayUnitMeter()); ok || out != nil || !reflect.DeepEqual(state, before) {
			t.Fatal("private refusal changed its working state", mode)
		}
	}
}
