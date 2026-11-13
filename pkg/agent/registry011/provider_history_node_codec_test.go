package registry011

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestProviderHistoryNodeScalarObservationPreservesValues(t *testing.T) {
	for _, raw := range []string{`"ordinary"`, ` "with\u0020space" `, `""`, `null`, `1`, `"unfinished`} {
		t.Run("string-"+raw, func(t *testing.T) {
			want, valid := nodeString(json.RawMessage(raw))
			got, ok := nodeStringObserved(json.RawMessage(raw), providerReplayUnitMeter())
			legacy, oldOK := nodeStringObserved(json.RawMessage(raw), nil)
			if ok != valid || got != want || legacy != want || oldOK != valid {
				t.Fatal("string byte contract", raw, got, ok, want, valid)
			}
		})
	}
	for _, raw := range []string{`"0x"`, `"0x00ff"`, `"\u0030x01"`, `"0xAB"`, `"0x0"`, `null`, `1`} {
		t.Run("data-"+raw, func(t *testing.T) {
			want, valid := nodeData(json.RawMessage(raw))
			got, ok := nodeDataObserved(json.RawMessage(raw), providerReplayUnitMeter())
			legacy, oldOK := nodeDataObserved(json.RawMessage(raw), nil)
			if ok != valid || !bytes.Equal(got, want) || oldOK != valid || !bytes.Equal(legacy, want) {
				t.Fatal("lower-case hex byte contract", raw, ok, valid)
			}
		})
	}
	for _, raw := range []string{`0`, `1`, `18446744073709551615`, `"0xffffffffffffffff"`, `"0x0"`, `01`, `+1`, ` 1`, `"0x00"`, `null`} {
		t.Run("index-"+raw, func(t *testing.T) {
			want, valid := nodeIndex(json.RawMessage(raw))
			got, ok := nodeIndexObserved(json.RawMessage(raw), providerReplayUnitMeter())
			legacy, oldOK := nodeIndexObserved(json.RawMessage(raw), nil)
			if ok != valid || ok && got != want || oldOK != valid || oldOK && legacy != want {
				t.Fatal("index conversion contract", raw, got, ok, want, valid)
			}
		})
	}
	for _, text := range []string{"0x0", "0xf", "0x" + strings.Repeat("f", 64), "0x00", "0X1", "0xg", "", "0x" + strings.Repeat("f", 65)} {
		want, valid := replayValue(text)
		got, ok := nodeValueObserved(text, providerReplayUnitMeter())
		legacy, oldOK := nodeValueObserved(text, nil)
		if ok != valid || ok && got.Cmp(want) != 0 || oldOK != valid || oldOK && legacy.Cmp(want) != 0 {
			t.Fatal("bounded integer contract", text, ok, valid)
		}
	}
	for _, raw := range []string{`"0x0"`, `"0x7fffffffffffffff"`, `"-0x8000000000000000"`, `"0x8000000000000000"`, `"-0x0"`, `"-0x8000000000000001"`, `0`} {
		if want := nodeSignedGas(json.RawMessage(raw)); nodeSignedGasObserved(json.RawMessage(raw), providerReplayUnitMeter()) != want || nodeSignedGasObserved(json.RawMessage(raw), nil) != want {
			t.Fatal("signed gas range", raw)
		}
	}
}

func TestProviderHistoryNodeObjectAndRequiredFieldsUseOriginalObserver(t *testing.T) {
	for _, raw := range []string{`{"a":1,"b":"ordinary"}`, ` {"b":2,"a":null} `, `{"a":1}`, `{"a":1,"extra":2}`, `[]`, `null`, `{`} {
		want, valid := nodeObject(json.RawMessage(raw), "a b")
		got, ok := nodeObjectObserved(json.RawMessage(raw), "a b", providerReplayUnitMeter())
		legacy, oldOK := nodeObjectObserved(json.RawMessage(raw), "a b", nil)
		if ok != valid || ok && !reflect.DeepEqual(got, want) || oldOK != valid || oldOK && !reflect.DeepEqual(legacy, want) {
			t.Fatal("closed object fields", raw, ok, valid)
		}
		want, valid = nodeJSONFields(json.RawMessage(raw), "a b", "a")
		got, ok = nodeJSONFieldsObserved(json.RawMessage(raw), "a b", "a", providerReplayUnitMeter())
		legacy, oldOK = nodeJSONFieldsObserved(json.RawMessage(raw), "a b", "a", nil)
		if ok != valid || ok && !reflect.DeepEqual(got, want) || oldOK != valid || oldOK && !reflect.DeepEqual(legacy, want) {
			t.Fatal("required non-null field contract", raw, ok, valid)
		}
	}
	probe := providerReplayUnitMeter()
	raw := json.RawMessage(`{"a":1}`)
	if _, ok := nodeJSONFieldsObserved(raw, "a", "a", probe); !ok || probe.actual[2] != uint64(len(raw)) {
		t.Fatal("normal original decoder precursor", probe.actual)
	}
	for units := uint64(0); units < probe.actual[10]; units++ {
		m := providerHistoryContentUnitMeter(candidateWorkVector{2: 1 << 30, 10: units})
		if _, ok := nodeJSONFieldsObserved(raw, "a", "a", m); ok || !m.failed {
			t.Fatal("failed null comparison became a non-null field", units, m.actual)
		}
	}
	exact := providerHistoryContentUnitMeter(probe.actual)
	if _, ok := nodeJSONFieldsObserved(raw, "a", "a", exact); !ok || exact.actual != probe.actual {
		t.Fatal("zero-cost final owner check demanded another codec invocation", exact.actual, probe.actual)
	}
}

func TestProviderHistoryNodeArrayObservationHasOneOwnedAllocation(t *testing.T) {
	for i, tc := range []struct {
		raw string
		max int
	}{
		{`[]`, 0}, {`[1]`, 1}, {`[1,2]`, 2}, {`[null,{"a":1},"x"]`, 3}, {` [1,2] `, 8},
		{`[1,2]`, 1}, {`[1,]`, 3}, {`[1] {}`, 3}, {`{}`, 3}, {`[]`, 65536},
	} {
		t.Run(fmt.Sprintf("array-%d", i), func(t *testing.T) {
			want, valid := nodeRawArray(json.RawMessage(tc.raw), tc.max)
			m := providerReplayUnitMeter()
			got, ok := nodeRawArrayObserved(json.RawMessage(tc.raw), tc.max, m)
			legacy, oldOK := nodeRawArrayObserved(json.RawMessage(tc.raw), tc.max, nil)
			if ok != valid || ok && !reflect.DeepEqual(got, want) || oldOK != valid || oldOK && !reflect.DeepEqual(legacy, want) {
				t.Fatal("bounded streaming array contract", tc.raw, ok, valid)
			}
			if m.actual[2] != uint64(len(tc.raw)) {
				t.Fatal("array codec input was invented or omitted", m.actual)
			}
		})
	}
	for _, maximum := range []int{-1, 65537} {
		if _, ok := nodeRawArrayObserved([]byte(`[]`), maximum, providerReplayUnitMeter()); ok {
			t.Fatal("selected array count escaped the existing closed frame ceiling", maximum)
		}
	}
	var out []json.RawMessage
	fields := map[string]json.RawMessage{"items": []byte(`[1]`)}
	if !nodeArrayObserved(fields, "absent", &out, 1, providerReplayUnitMeter()) || out != nil || nodeArrayObserved(fields, "items", nil, 1, providerReplayUnitMeter()) {
		t.Fatal("optional absence or missing destination")
	}
	if !nodeArrayObserved(fields, "items", &out, 1, providerReplayUnitMeter()) || len(out) != 1 || !nodeArrayObserved(fields, "items", &out, 1, nil) {
		t.Fatal("selected optional array")
	}
	if uint64(reflect.TypeFor[json.RawMessage]().Size()) != uint64(3*(strconv.IntSize/8)) {
		t.Fatal("array header copy size differs from the concrete Go representation")
	}
}

func TestProviderHistoryNodeArrayLateCopyShortageReturnsNoPartialData(t *testing.T) {
	raw := json.RawMessage(`[1,2,3]`)
	probe := providerReplayUnitMeter()
	if _, ok := nodeRawArrayObserved(raw, 3, probe); !ok {
		t.Fatal("positive original precursor")
	}
	upper := probe.actual
	upper[10]--
	m := providerHistoryContentUnitMeter(upper)
	got, ok := nodeRawArrayObserved(raw, 3, m)
	if ok || got != nil || !m.failed || m.actual[2] != uint64(len(raw)) || m.actual[10] == 0 || m.actual[10] >= probe.actual[10] {
		t.Fatal("late element copy returned partial array or lost prior work", got, ok, m.actual)
	}
	before := m.actual
	if _, ok := nodeRawArrayObserved(raw, 3, m); ok || m.actual != before {
		t.Fatal("original array meter was reset or reused")
	}
	exact := providerHistoryContentUnitMeter(probe.actual)
	if got, ok := nodeRawArrayObserved(raw, 3, exact); !ok || len(got) != 3 || exact.actual != probe.actual {
		t.Fatal("exact original array precursor allowance", exact.actual, probe.actual)
	}
}

func TestProviderHistoryNodeScalarShortagesNeverSelectLegacyFallback(t *testing.T) {
	for _, mode := range []string{"object", "string", "data", "index", "integer", "literal", "signed", "array", "fields"} {
		m := providerHistoryContentUnitMeter(candidateWorkVector{})
		ok := false
		switch mode {
		case "object":
			_, ok = nodeObjectObserved([]byte(`{"a":1}`), "a", m)
		case "string":
			_, ok = nodeStringObserved([]byte(`"ordinary"`), m)
		case "data":
			_, ok = nodeDataObserved([]byte(`"0x01"`), m)
		case "index":
			_, ok = nodeIndexObserved([]byte(`1`), m)
		case "integer":
			_, ok = nodeValueObserved("0x1", m)
		case "literal":
			ok = nodeLiteralObserved([]byte(`null`), "null", m)
		case "signed":
			ok = nodeSignedGasObserved([]byte(`"-0x1"`), m)
		case "array":
			_, ok = nodeRawArrayObserved([]byte(`[1]`), 1, m)
		case "fields":
			_, ok = nodeJSONFieldsObserved([]byte(`{"a":1}`), "a", "a", m)
		}
		if ok || !m.failed || m.actual != (candidateWorkVector{}) {
			t.Fatal("unfunded scalar/array parser ran without its original observer", mode, m.actual)
		}
	}
	if !nodeLiteralObserved([]byte(` null `), "null", nil) || nodeLiteralObserved([]byte(`false`), "null", nil) {
		t.Fatal("legacy literal contract")
	}
}
