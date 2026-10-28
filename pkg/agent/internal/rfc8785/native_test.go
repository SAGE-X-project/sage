package rfc8785

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"testing"
)

func TestNativeCanonicalMarshalMatchesOriginalNormalization(t *testing.T) {
	t.Parallel()
	cycle := map[string]any{}
	cycle["cycle"] = cycle
	var deep any = "leaf"
	for i := 0; i < 70; i++ {
		deep = []any{deep}
	}
	values := []any{
		nil, true, false, "", "<&>\u2028\u2029😀\x00\n", "\xff", float64(0), math.Copysign(0, -1),
		float64(1e-7), float64(1e-6), float64(1e20), float64(1e21), math.MaxFloat64, math.SmallestNonzeroFloat64,
		math.NaN(), math.Inf(1), math.Inf(-1), int64(9007199254740993), float32(0.1),
		json.Number("1.234567890123456789"), json.Number("01"), json.Number("1e400"),
		json.RawMessage(`{"b":2,"a":1}`), json.RawMessage(`invalid`),
		map[string]any(nil), map[string]string(nil), []any(nil), []string(nil),
		map[string]any{}, map[string]string{}, []any{}, []string{},
		map[string]int{"b": 2, "a": 1}, []int{2, 1},
		map[string]any{"a": nil, "b": false, "c": float64(12.5)},
		map[string]any{"\ufffd": "replacement", "😀": "supplementary", "a": "<&>", "\r": "control"},
		map[string]string{"\ufffd": "replacement", "😀": "supplementary", "a": "<&>", "\r": "control"},
		map[string]any{"\xff": "first", "\xfe": "second"}, map[string]string{"bad": "\xff"},
		map[string]any{"bad": "\xff"}, map[string]string{"\xff": "bad key"},
		[]string{"a", "\xff"}, []any{"a", struct{}{}},
		map[string]any{"nested": map[string]string{"z": "last", "a": "first"}, "strings": []string{"", "<&>", "😀"}, "values": []any{nil, true, float64(-3.25), map[string]any{"b": "2", "a": "1"}}},
		map[string]any{"raw": json.RawMessage(`null`)}, map[string]any{"number": json.Number("01")},
		[]map[string]any(nil), []map[string]string(nil), []map[string]any{}, []map[string]string{},
		[]map[string]any{{"b": "2", "a": "1"}, {"unicode": "😀", "value": float64(-3.25)}},
		[]map[string]string{{"b": "2", "a": "1"}, {"unicode": "😀", "control": "\n"}},
		[]map[string]any{nil}, []map[string]string{nil},
		[]map[string]any{{"unknown": json.Number("01")}}, []map[string]string{{"bad": "\xff"}},
		deep, cycle,
	}
	for i, value := range values {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			raw, expectedErr := json.Marshal(value)
			var expected []byte
			if expectedErr == nil {
				expected, expectedErr = Canonicalize(raw)
			}
			actual, actualErr := Marshal(value)
			if (actualErr != nil) != (expectedErr != nil) || !bytes.Equal(actual, expected) || actualErr != nil && actualErr.Error() != expectedErr.Error() {
				t.Fatalf("canonical bytes/error differ: actual=%q %v expected=%q %v", actual, actualErr, expected, expectedErr)
			}
		})
	}
}

type nativeCountingMarshaler struct {
	calls *int
	fail  bool
}

func (m nativeCountingMarshaler) MarshalJSON() ([]byte, error) {
	(*m.calls)++
	if m.fail {
		return nil, errors.New("finite marshaler refusal")
	}
	return []byte(`{"z":"<&>","a":1}`), nil
}

func TestNativeEligibilityNeverInvokesCustomMarshalers(t *testing.T) {
	t.Parallel()
	for _, fail := range []bool{false, true} {
		calls := 0
		value := map[string]any{"before": []any{"plain", true}, "custom": nativeCountingMarshaler{&calls, fail}}
		if nativeCanonicalValue(value, 0) || calls != 0 {
			t.Fatal("eligibility transformed a custom value or invoked a callback")
		}
		raw, expectedErr := json.Marshal(value)
		var expected []byte
		if expectedErr == nil {
			expected, expectedErr = Canonicalize(raw)
		}
		if calls != 1 {
			t.Fatal("baseline custom marshaler count")
		}
		calls = 0
		actual, actualErr := Marshal(value)
		if calls != 1 || (actualErr != nil) != (expectedErr != nil) || !bytes.Equal(actual, expected) || actualErr != nil && actualErr.Error() != expectedErr.Error() {
			t.Fatal("fallback changed original custom marshaler behavior", calls, actualErr, expectedErr)
		}
	}
}

func TestNativeEligibilityKeepsUnknownAndNormalizedShapesOnFallback(t *testing.T) {
	t.Parallel()
	for _, value := range []any{map[string]any(nil), map[string]string(nil), []any(nil), []string(nil), map[string]any{"bad": "\xff"}, map[string]any{"\xff": "bad"}, map[string]string{"bad": "\xff"}, map[string]string{"\xff": "bad"}, []any{struct{}{}}, []string{"\xff"}, struct{}{}, json.Number("1"), json.RawMessage(`null`), math.NaN(), math.Inf(1), math.Inf(-1)} {
		if nativeCanonicalValue(value, 0) {
			t.Fatal("normalization-dependent value entered the native writer")
		}
	}
	for _, value := range []any{[]map[string]any(nil), []map[string]string(nil), []map[string]any{nil}, []map[string]string{nil}, []map[string]any{{"bad": "\xff"}}, []map[string]string{{"bad": "\xff"}}} {
		if nativeCanonicalValue(value, 0) {
			t.Fatal("typed map slice changed normalization or nil semantics")
		}
	}
	if nativeCanonicalValue("bounded", 65) {
		t.Fatal("eligibility exceeded its recursion boundary")
	}
}

// Private writer error guards use finite DATA only. The public entry point
// rejects these shapes during eligibility and preserves the original fallback.
func TestNativeWriterRejectsInvalidUnvalidatedData(t *testing.T) {
	t.Parallel()
	for _, value := range []any{[]map[string]any{{"a": "valid"}, {"bad": struct{}{}}}, []map[string]string{{"a": "valid"}, {"bad": "\xff"}}, struct{}{}, "\xff", math.NaN(), map[string]any{"a": "valid", "\xff": "bad"}, map[string]any{"a": "valid", "b": struct{}{}}, map[string]string{"a": "valid", "\xff": "bad"}, map[string]string{"a": "valid", "b": "\xff"}, []any{"valid", struct{}{}}, []string{"valid", "\xff"}} {
		var buf bytes.Buffer
		if err := writeNativeCanonicalValue(&buf, value); err == nil {
			t.Fatal("invalid private writer DATA was accepted")
		}
	}
}
