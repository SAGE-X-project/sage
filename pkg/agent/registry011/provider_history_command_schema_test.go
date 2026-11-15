package registry011

import (
	"encoding/json"
	"reflect"
	"testing"
)

func providerHistoryCommandUnitKeys(t *testing.T) []any {
	t.Helper()
	c := replayTestCreate(t)
	return contentJSON(c.raw).(map[string]any)["arguments"].(map[string]any)["keys"].([]any)
}

func TestProviderHistoryCommandKeysAndABIArraysPreserveOwnedExpiryShape(t *testing.T) {
	did := "did:sage:" + replayTestRegistry + ":" + replayTestAgent
	keys := providerHistoryCommandUnitKeys(t)
	for _, expiry := range []bool{false, true} {
		if expiry {
			keys[0].(map[string]any)["expires"] = json.Number("0")
		}
		want, accepted := replayKeys(keys, did)
		m := providerHistoryExecutionUnitMeter()
		got, ok := replayCommandKeysObserved(keys, did, m)
		if !accepted || !ok || !reflect.DeepEqual(got, want) || m.failed {
			t.Fatal("existing suite/expiry shape", got, want, m.actual)
		}
		if expiry && (got[0].Expires == nil || *got[0].Expires != 0) {
			t.Fatal("explicit zero expiry became absence")
		}
		abiKeys, ok := replayCommandABIKeysObserved(got, m)
		if !ok || !reflect.DeepEqual(abiKeys, replayABIKeys(want)) {
			t.Fatal("actual ABI key conversion")
		}
		abiKeys[0].Key[0] ^= 1
		if got[0].Key != want[0].Key {
			t.Fatal("ABI binary material aliases source string")
		}
	}
	for _, value := range []any{nil, []any{}, make([]any, 129), []any{1}} {
		if got, ok := replayCommandKeysObserved(value, did, providerHistoryExecutionUnitMeter()); ok || got != nil {
			t.Fatal("key container/count/type")
		}
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{})
	if got, ok := replayCommandKeysObserved(keys, did, short); ok || got != nil || !short.failed {
		t.Fatal("unfunded key storage")
	}
	if got, ok := replayCommandKeysObserved(keys, did, nil); ok || got != nil {
		t.Fatal("missing key observer")
	}
}

func TestProviderHistoryCommandKeysRejectExistingSchemaViolations(t *testing.T) {
	did := "did:sage:" + replayTestRegistry + ":" + replayTestAgent
	base := providerHistoryCommandUnitKeys(t)
	for _, tc := range []struct {
		name string
		edit func([]any)
	}{
		{"algorithm", func(a []any) { a[0].(map[string]any)["alg"] = "rsa" }},
		{"name", func(a []any) { a[0].(map[string]any)["name"] = "bad.name" }},
		{"state", func(a []any) { a[0].(map[string]any)["state"] = "revoked" }},
		{"material", func(a []any) { a[0].(map[string]any)["key"] = "bad" }},
		{"duplicate-material", func(a []any) { a[1].(map[string]any)["key"] = a[0].(map[string]any)["key"] }},
		{"proof-shape", func(a []any) { a[0].(map[string]any)["proof"] = nil }},
		{"proof-value", func(a []any) { a[0].(map[string]any)["proof"].(map[string]any)["value"] = "bad" }},
		{"signer-prefix", func(a []any) { a[0].(map[string]any)["proof"].(map[string]any)["signer"] = "did:other:a#a" }},
		{"signer-name", func(a []any) { a[0].(map[string]any)["proof"].(map[string]any)["signer"] = did + "#bad.name" }},
		{"ed-self-signer", func(a []any) { a[0].(map[string]any)["proof"].(map[string]any)["signer"] = did + "#b" }},
		{"expiry", func(a []any) { a[0].(map[string]any)["expires"] = json.Number("9007199254740992") }},
		{"extra-field", func(a []any) { a[0].(map[string]any)["extra"] = true }},
		{"order", func(a []any) { a[0], a[1] = a[1], a[0] }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := contentJSON(replayTestJSON(t, base)).([]any)
			tc.edit(a)
			if _, ok := replayKeys(a, did); ok {
				t.Fatal("existing schema must first reject the single change")
			}
			if got, ok := replayCommandKeysObserved(a, did, providerHistoryExecutionUnitMeter()); ok || got != nil {
				t.Fatal("changed key schema returned partial output")
			}
		})
	}
}

func TestProviderHistoryCommandServicesPreserveURIAndKeyNamespaceChecks(t *testing.T) {
	base := []any{map[string]any{"name": "api", "type": "A2A", "uri": "https://example.invalid/api"}}
	keys, _ := replayKeys(providerHistoryCommandUnitKeys(t), "did:sage:"+replayTestRegistry+":"+replayTestAgent)
	want, accepted := replayServices(base, keys)
	m := providerHistoryExecutionUnitMeter()
	got, ok := replayCommandServicesObserved(base, keys, m)
	if !accepted || !ok || !reflect.DeepEqual(got, want) || m.failed {
		t.Fatal("existing services", got, want)
	}
	converted, ok := replayCommandABIServicesObserved(got, m)
	if !ok || !reflect.DeepEqual(converted, replayABIServices(want)) {
		t.Fatal("actual ABI service conversion")
	}
	for _, tc := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"name", func(m map[string]any) { m["name"] = "bad.name" }},
		{"key-namespace", func(m map[string]any) { m["name"] = "a" }},
		{"empty-type", func(m map[string]any) { m["type"] = "" }},
		{"non-ascii-type", func(m map[string]any) { m["type"] = "é" }},
		{"uri", func(m map[string]any) { m["uri"] = "http://example.invalid" }},
		{"field", func(m map[string]any) { m["extra"] = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := contentJSON(replayTestJSON(t, base)).([]any)
			tc.edit(a[0].(map[string]any))
			if _, ok := replayServices(a, keys); ok {
				t.Fatal("existing service schema must reject the change")
			}
			if got, ok := replayCommandServicesObserved(a, keys, providerHistoryExecutionUnitMeter()); ok || got != nil {
				t.Fatal("changed service schema returned partial output")
			}
		})
	}
	for _, value := range []any{nil, []any{nil}, make([]any, 17), []any{base[0], base[0]}} {
		if got, ok := replayCommandServicesObserved(value, keys, providerHistoryExecutionUnitMeter()); ok || got != nil {
			t.Fatal("service container/count/order")
		}
	}
	if got, ok := replayCommandServicesObserved([]any{}, keys, providerHistoryExecutionUnitMeter()); !ok || got == nil || len(got) != 0 {
		t.Fatal("empty service list changed to null")
	}
	short := providerHistoryContentUnitMeter(candidateWorkVector{})
	if got, ok := replayCommandServicesObserved(base, keys, short); ok || got != nil || !short.failed {
		t.Fatal("unfunded service storage")
	}
	if got, ok := replayCommandServicesObserved(base, keys, nil); ok || got != nil {
		t.Fatal("missing service observer")
	}
}
