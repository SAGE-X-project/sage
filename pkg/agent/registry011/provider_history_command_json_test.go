package registry011

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestProviderHistoryCommandJSONPreservesStrictUseNumberAndCanonicalBytes(t *testing.T) {
	for _, tc := range []struct{ name, raw string }{
		{"string", `"a"`}, {"number", `9007199254740991`}, {"exponent", `1e+21`},
		{"boolean", `false`}, {"null", `null`}, {"array", `[]`}, {"object", `{}`},
		{"tree", `{"a":[null,false,0,"x"],"n":9007199254740991}`},
		{"escaped-ascii", `{"a":"\u0000\n\"\\"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(tc.raw)
			want := contentJSON(raw)
			if want == nil && tc.name != "null" {
				t.Fatal("existing canonical JSON precursor")
			}
			m := providerHistoryExecutionUnitMeter()
			got := replayCommandJSONObserved(raw, m)
			if !reflect.DeepEqual(got, want) || m.failed || m.actual[2] < uint64(2*len(raw)) {
				t.Fatal("strict native canonical tree changed semantics", got, want, m.actual)
			}
			if !reflect.DeepEqual(replayCommandJSONObserved(raw, nil), want) {
				t.Fatal("legacy nil JSON")
			}
		})
	}
	v := replayCommandJSONObserved([]byte(`{"n":9007199254740991}`), providerHistoryExecutionUnitMeter()).(map[string]any)["n"]
	if v != json.Number("9007199254740991") {
		t.Fatal("returned command lost original Number text", v)
	}
}

func TestProviderHistoryCommandJSONRefusesNoncanonicalOrMalformedInputs(t *testing.T) {
	for _, tc := range []struct{ name, raw string }{
		{"empty", ""}, {"whitespace", ` {}`}, {"duplicate", `{"a":1,"a":2}`},
		{"unsorted", `{"b":0,"a":0}`}, {"trailing", `{}0`}, {"missing-value", `{"a":}`},
		{"unicode-value", `"é"`}, {"unicode-key", `{"é":0}`}, {"surrogate", `"\ud800"`},
		{"noncanonical-escape", `"\u0061"`}, {"negative-zero", `-0`}, {"exponent-expansion", `1e20`},
		{"nonfinite-number", `1e309`}, {"underflow-number", `1e-400`}, {"extra-container", `[][]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(tc.raw)
			if contentJSON(raw) != nil {
				t.Fatal("existing parser must first reject the changed input")
			}
			if replayCommandJSONObserved(raw, providerHistoryExecutionUnitMeter()) != nil {
				t.Fatal("invalid command JSON accepted")
			}
		})
	}
	for _, raw := range [][]byte{{255}, []byte(strings.Repeat("[", 33) + "0" + strings.Repeat("]", 33)), make([]byte, 262145)} {
		if replayCommandJSONObserved(raw, providerHistoryExecutionUnitMeter()) != nil {
			t.Fatal("UTF8/depth/input bound")
		}
	}
}

func TestProviderHistoryCommandJSONPreflightIsNotRecordedAsActualOutput(t *testing.T) {
	// UTF8/unicode8 + two decodes20 + entered/string/copy5 + encode10 + equality7.
	// Encoder preflight is82^2+1=6725 after actual33, not an actual output.
	m := providerHistoryContentUnitMeter(candidateWorkVector{2: 6, 10: 6758})
	if got := replayCommandJSONObserved([]byte(`"a"`), m); got != "a" || m.actual != (candidateWorkVector{2: 6, 10: 50}) {
		t.Fatal("native JSON actual/preflight arithmetic", got, m.actual)
	}
	for _, upper := range []candidateWorkVector{{}, {2: 5, 10: 6758}, {2: 6, 10: 6757}} {
		short := providerHistoryContentUnitMeter(upper)
		if got := replayCommandJSONObserved([]byte(`"a"`), short); got != nil || !short.failed {
			t.Fatal("unfunded original JSON observer", got, short.actual)
		}
		before := short.actual
		if got := replayCommandJSONObserved([]byte(`"a"`), short); got != nil || short.actual != before {
			t.Fatal("failed JSON observer resumed")
		}
	}
}

func TestProviderHistoryCommandNativeTreeClosesTypesShapesAndOwnedStorage(t *testing.T) {
	value := map[string]any{"a": []any{"text", json.Number("7")}}
	limit := replayCommandJSONLimit{nodes: 262144, bytes: 262144}
	got, ok := replayCommandNativeObserved(value, 0, &limit, providerHistoryExecutionUnitMeter())
	if !ok || !reflect.DeepEqual(got, map[string]any{"a": []any{"text", float64(7)}}) {
		t.Fatal("closed native tree", got)
	}
	got.(map[string]any)["a"].([]any)[0] = "different"
	if value["a"].([]any)[0] != "text" {
		t.Fatal("native clone aliases original containers")
	}
	for _, v := range []any{1, json.Number("1e309"), "é", make(chan int)} {
		limit := replayCommandJSONLimit{nodes: 262144, bytes: 262144}
		if _, ok := replayCommandNativeObserved(v, 0, &limit, providerHistoryExecutionUnitMeter()); ok {
			t.Fatal("unsupported native tree type or scalar")
		}
	}
	for _, limit := range []replayCommandJSONLimit{{}, {nodes: 1, bytes: 0}, {nodes: 1, bytes: 10}, {nodes: 262145, bytes: 10}, {nodes: 10, bytes: -1}, {nodes: 10, bytes: 262145}} {
		if _, ok := replayCommandNativeObserved(value, 0, &limit, providerHistoryExecutionUnitMeter()); ok {
			t.Fatal("native tree node/string budget")
		}
	}
	limit = replayCommandJSONLimit{nodes: 262144, bytes: 262144}
	if _, ok := replayCommandNativeObserved(value, 33, &limit, providerHistoryExecutionUnitMeter()); ok {
		t.Fatal("native tree depth")
	}
	if _, ok := replayCommandNativeObserved(value, -1, &limit, providerHistoryExecutionUnitMeter()); ok {
		t.Fatal("negative native tree depth")
	}
	if _, ok := replayCommandNativeObserved(value, 0, nil, providerHistoryExecutionUnitMeter()); ok {
		t.Fatal("missing shape budget")
	}
	if _, ok := replayCommandNativeObserved(value, 0, &limit, nil); ok {
		t.Fatal("missing original native observer")
	}
}
