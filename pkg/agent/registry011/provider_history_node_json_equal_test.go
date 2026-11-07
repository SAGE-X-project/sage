package registry011

import "testing"

func TestProviderHistoryNodeJSONValueComparisonPreservesDecodedSemantics(t *testing.T) {
	for _, tc := range []struct {
		name string
		a, b any
		want bool
	}{
		{"null", nil, nil, true}, {"null-type", nil, false, false},
		{"string", "0x0012", "0x0012", true}, {"string-value", "a", "b", false}, {"string-type", "a", false, false},
		{"bool", true, true, true}, {"bool-value", true, false, false}, {"bool-type", false, "false", false},
		{"number", float64(1), float64(1), true}, {"number-value", float64(1), float64(2), false}, {"number-type", float64(1), "1", false},
		{"array", []any{nil, "a", true, float64(1)}, []any{nil, "a", true, float64(1)}, true},
		{"array-type", []any{}, nil, false}, {"array-size", []any{nil}, []any{}, false}, {"array-value", []any{"a"}, []any{"b"}, false},
		{"object", map[string]any{"a": []any{nil, "b"}}, map[string]any{"a": []any{nil, "b"}}, true},
		{"object-type", map[string]any{}, false, false}, {"object-size", map[string]any{}, map[string]any{"a": nil}, false},
		{"object-key", map[string]any{"a": nil}, map[string]any{"b": nil}, false},
		{"object-value", map[string]any{"a": "a"}, map[string]any{"a": "b"}, false},
		{"not-decoded-json", struct{}{}, struct{}{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := providerReplayUnitMeter()
			if got := nodeJSONValueEqualObserved(tc.a, tc.b, 0, m); got != tc.want || m.actual[10] == 0 {
				t.Fatal("bounded decoded value comparison", got, tc.want, m.actual)
			}
		})
	}
}

func TestProviderHistoryNodeJSONValueComparisonRefusesOriginalQuotaAndDepth(t *testing.T) {
	m := providerHistoryContentUnitMeter(candidateWorkVector{})
	if nodeJSONValueEqualObserved("a", "a", 0, m) || !m.failed {
		t.Fatal("zero original allowance")
	}
	before := m.actual
	if nodeJSONValueEqualObserved(nil, nil, 0, m) || m.actual != before {
		t.Fatal("failed original must not resume")
	}
	if nodeJSONValueEqualObserved(nil, nil, 33, providerReplayUnitMeter()) {
		t.Fatal("depth ceiling")
	}
	probe := providerReplayUnitMeter()
	if !nodeJSONValueEqualObserved(map[string]any{"a": "normal"}, map[string]any{"a": "normal"}, 0, probe) {
		t.Fatal("positive key/string precursor")
	}
	for n := uint64(0); n < probe.actual[10]; n++ {
		m := providerHistoryContentUnitMeter(candidateWorkVector{10: n})
		if nodeJSONValueEqualObserved(map[string]any{"a": "normal"}, map[string]any{"a": "normal"}, 0, m) || !m.failed {
			t.Fatal("every insufficient structural/string allowance", n)
		}
	}
	if !nodeJSONValueEqualObserved(map[string]any{"a": "normal"}, map[string]any{"a": "normal"}, 0, providerHistoryContentUnitMeter(probe.actual)) {
		t.Fatal("exact actual comparison allowance")
	}
}
