package registry011

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

func TestProviderHistoryObservedStateCloneOwnsArraysStringsAndExpiry(t *testing.T) {
	state := providerHistoryStateUnitPrefix(t, 1)
	expires := uint64(1000)
	state.record.Keys[0].Expires = &expires
	state.record.Services = []replayService{{Name: "api", Type: "A2A", URI: "https://example.invalid/api"}}
	state.grants = []replayGrant{{Operator: replayTestOperator, Scope: "activate"}}
	before := providerHistoryStateUnitSnapshot(t, state)
	got, ok := replayStateCloneObserved(state, providerHistoryStateUnitMeter())
	if !ok || !reflect.DeepEqual(got, before) || got.record.Keys[0].Expires == state.record.Keys[0].Expires {
		t.Fatal("owned actual typed state clone")
	}
	*got.record.Keys[0].Expires = 900
	got.record.Keys[0].Proof.Value = "changed"
	got.record.Services[0].URI = "changed"
	got.grants[0].Scope = "deactivate"
	if !reflect.DeepEqual(state, before) {
		t.Fatal("working copy changed its prior state")
	}
	zero, ok := replayStateCloneObserved(replayState{}, providerHistoryStateUnitMeter())
	if !ok || !reflect.DeepEqual(zero, replayState{}) {
		t.Fatal("zero/nil initial state shape")
	}
}

func TestProviderHistoryObservedStateStorageQuotasAndInputBounds(t *testing.T) {
	m := providerHistoryContentUnitMeter(candidateWorkVector{10: 6})
	if got, ok := replayStateTextObserved("ab", 2, m); !ok || got != "ab" || m.actual != (candidateWorkVector{10: 6}) {
		t.Fatal("owned state string compare3/copy3", m.actual)
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{10: 5})
	if got, ok := replayStateTextObserved("ab", 2, short); ok || got != "" || short.actual[10] != 3 || !short.failed {
		t.Fatal("state string late copy shortage", short.actual)
	}
	for _, value := range []string{strings.Repeat("a", 33), "é"} {
		if _, ok := replayStateTextObserved(value, 32, providerHistoryStateUnitMeter()); ok {
			t.Fatal("state text bound/ASCII")
		}
	}
	if _, ok := replayStateKeysCloneObserved(make([]replayKey, 129), providerHistoryStateUnitMeter()); ok {
		t.Fatal("key storage bound")
	}
	if _, ok := replayStateServicesCloneObserved(make([]replayService, 17), providerHistoryStateUnitMeter()); ok {
		t.Fatal("service storage bound")
	}
	for _, count := range []int{-1, 129} {
		if _, ok := replayStateGrantStorageObserved(count, providerHistoryStateUnitMeter()); ok {
			t.Fatal("grant storage bound")
		}
	}
	if replayStateSetTextObserved(nil, "a", providerHistoryStateUnitMeter()) || replayStateGrantPutObserved(nil, 0, replayGrant{}, providerHistoryStateUnitMeter()) {
		t.Fatal("missing destination")
	}
	if _, ok := replayStateCloneObserved(replayState{}, nil); ok {
		t.Fatal("missing clone observer")
	}
}

func TestProviderHistoryObservedKeyAndGrantInsertionPreservesSortedMeaning(t *testing.T) {
	create := replayTestCreate(t)
	second := replayTestKey(t, "b", true)
	got, ok := replayStateInsertKeyObserved(create.keys, nil, second, providerHistoryStateUnitMeter())
	if !ok || len(got) != 3 || got[0].Name != "a" || got[1].Name != "b" || got[2].Name != "kem" {
		t.Fatal("key insertion before retained KEM")
	}
	duplicate := second
	duplicate.Key = create.keys[0].Key
	if _, ok := replayStateInsertKeyObserved(create.keys, nil, duplicate, providerHistoryStateUnitMeter()); ok {
		t.Fatal("duplicate material")
	}
	if _, ok := replayStateInsertKeyObserved(create.keys, []replayService{{Name: "b"}}, second, providerHistoryStateUnitMeter()); ok {
		t.Fatal("service namespace collision")
	}
	if _, ok := replayStateInsertKeyObserved(make([]replayKey, 128), nil, second, providerHistoryStateUnitMeter()); ok {
		t.Fatal("key lifetime bound")
	}
	grants := []replayGrant{{Operator: replayTestOperator, Scope: "deactivate"}}
	c := replayTestCommand(t, "authorize-operator", replayTestWallet, 1, map[string]any{"operator": replayTestOperator, "scope": "activate"})
	added, ok := replayStateGrantChangeObserved(grants, replayTestWallet, "created", c, providerHistoryStateUnitMeter())
	if !ok || len(added) != 2 || added[0].Scope != "activate" || added[1].Scope != "deactivate" || len(grants) != 1 {
		t.Fatal("grant insertion ordering/ownership")
	}
	c = replayTestCommand(t, "revoke-operator", replayTestWallet, 2, map[string]any{"operator": replayTestOperator, "scope": "activate"})
	retained, ok := replayStateGrantChangeObserved(added, replayTestWallet, "created", c, providerHistoryStateUnitMeter())
	if !ok || !reflect.DeepEqual(retained, grants) || len(added) != 2 {
		t.Fatal("grant deletion ownership")
	}
	removed, retained, ok := replayStateSplitGrantsObserved(added, false, true, providerHistoryStateUnitMeter())
	if !ok || len(removed) != 1 || removed[0].Scope != "activate" || !reflect.DeepEqual(retained, grants) {
		t.Fatal("automatic activation cleanup")
	}
	removed, retained, ok = replayStateSplitGrantsObserved(added, true, false, providerHistoryStateUnitMeter())
	if !ok || !reflect.DeepEqual(removed, added) || retained == nil || len(retained) != 0 {
		t.Fatal("terminal cleanup preserves empty list")
	}
}

func TestProviderHistoryObservedStateJSONKeepsRevocationExpiryAndFullVersions(t *testing.T) {
	state := providerHistoryStateUnitPrefix(t, 11)
	zero := uint64(0)
	state.record.Keys[0].Expires = &zero
	state.record.Version = "18446744073709551615"
	want, err := rfc8785.Marshal(state.record)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := replayStateRecordBytesObserved(state.record, providerHistoryStateUnitMeter())
	if !ok || !bytes.Equal(got, want) || state.record.Keys[0].State != "revoked" {
		t.Fatal("record state/expiry/version must not use command accepted-only serializer")
	}
	want, err = rfc8785.Marshal(state.grants)
	if err != nil {
		t.Fatal(err)
	}
	got, ok = replayStateGrantsBytesObserved(state.grants, providerHistoryStateUnitMeter())
	if !ok || !bytes.Equal(got, want) || string(got) != "[]" {
		t.Fatal("empty grants canonical shape")
	}
	if _, ok := replayStateObjectObserved(make([]string, 11), make([]any, 11), providerHistoryStateUnitMeter()); ok {
		t.Fatal("history field ceiling")
	}
	if _, ok := replayStateObjectObserved([]string{"a", "a"}, []any{nil, nil}, providerHistoryStateUnitMeter()); ok {
		t.Fatal("duplicate state object fields")
	}
}

func TestProviderHistoryObservedStateRecordWidthAndTerminalHeadroom(t *testing.T) {
	// Typed serialization DATA only: repeated public sample entries determine
	// width; this is not an admitted set of128 distinct/verified keys.
	key := replayTestKey(t, "a", false)
	record := replayRecord{ID: "did:sage:" + replayTestRegistry + ":" + replayTestAgent, Controller: replayTestWallet, State: "created", Version: "1", Keys: make([]replayKey, 128), Services: make([]replayService, 16)}
	for i := range record.Keys {
		record.Keys[i] = key
	}
	for i := range record.Services {
		record.Services[i] = replayService{Name: "api", Type: "A2A", URI: "https://example.invalid/"}
	}
	base, err := rfc8785.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	extra := 65520 - len(base)
	if extra < 0 {
		t.Fatal("bounded width precursor no longer fits")
	}
	for i := range record.Services {
		count := extra / len(record.Services)
		if i < extra%len(record.Services) {
			count++
		}
		record.Services[i].URI += strings.Repeat("a", count)
		if len(record.Services[i].URI) > 2048 {
			t.Fatal("existing per-service URI bound")
		}
	}
	want, err := rfc8785.Marshal(record)
	if err != nil || len(want) != 65520 {
		t.Fatal("independent old writer width precursor", len(want), err)
	}
	if got, ok := replayStateRecordBytesObserved(record, providerHistoryStateUnitMeter()); !ok || !bytes.Equal(got, want) {
		t.Fatal("bounded present record")
	}
	record.State, record.Version = "deactivated", "18446744073709551615"
	future, err := rfc8785.Marshal(record)
	if err != nil || len(future) != 65543 {
		t.Fatal("terminal state/version growth precursor", len(future), err)
	}
	if got, ok := replayStateRecordBytesObserved(record, providerHistoryStateUnitMeter()); ok || got != nil {
		t.Fatal("terminal/version-exhaustion width was repaired or truncated")
	}
}
