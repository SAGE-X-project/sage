package registry011

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

// Preserve the existing strict UTF-8/unicode, duplicate/depth, canonical byte,
// UseNumber and ASCII contract. The canonical writer receives only a closed
// native tree, never a caller marshaler or a producer-reported work result.
// The original decoded UseNumber tree is retained separately from that copy.
func replayCommandJSONObserved(raw []byte, meter *providerWorkMeter) any {
	if meter == nil {
		return contentJSON(raw)
	}
	if len(raw) == 0 || len(raw) > 262144 || !meter.event(providerCompareEvent, uint64(len(raw))) || !utf8.Valid(raw) || !meter.event(providerCompareEvent, uint64(len(raw))) || !validRPCUnicode(raw) || !meter.event(providerCodecEvent, uint64(len(raw))) {
		return nil
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if !readRPCValue(d, 0, 32) {
		return nil
	}
	if _, err := d.Token(); err != io.EOF {
		return nil
	}
	if !meter.event(providerCodecEvent, uint64(len(raw))) {
		return nil
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var value any
	if d.Decode(&value) != nil {
		return nil
	}
	limits := replayCommandJSONLimit{nodes: 262144, bytes: 262144}
	native, ok := replayCommandNativeObserved(value, 0, &limits, meter)
	if !ok {
		return nil
	}
	// An ECMAScript number can expand a compact exponent;6*n+64 bounds
	// that expansion as well as strings/punctuation for this <=256KiB input.
	maximum := 6*len(raw) + 64
	if !meter.preflight(providerEncodeEvent, uint64(maximum)) {
		return nil
	}
	canonical, err := rfc8785.Marshal(native)
	if err != nil || len(canonical) > maximum {
		meter.refuse()
		return nil
	}
	if !meter.event(providerEncodeEvent, uint64(len(canonical))) || !providerReplayEqual(raw, canonical, meter) {
		return nil
	}
	return value // Number strings and exact omission/null shape remain unchanged.
}

type replayCommandJSONLimit struct{ nodes, bytes int }

func replayCommandNativeObserved(value any, depth int, limit *replayCommandJSONLimit, meter *providerWorkMeter) (any, bool) {
	if meter == nil || limit == nil || depth < 0 || depth > 32 || limit.nodes <= 0 || limit.nodes > 262144 || limit.bytes < 0 || limit.bytes > 262144 || !meter.event(providerCompareEvent, 0) {
		return nil, false
	}
	limit.nodes--
	switch v := value.(type) {
	case nil, bool:
		return value, true
	case string:
		if len(v) > limit.bytes || !meter.event(providerCompareEvent, uint64(len(v))) || !contentASCII(v) || !meter.event(providerCopyEvent, uint64(len(v))) {
			return nil, false
		}
		limit.bytes -= len(v)
		return strings.Clone(v), true
	case json.Number:
		if len(v) > limit.bytes || !meter.event(providerCodecEvent, uint64(len(v))) {
			return nil, false
		}
		limit.bytes -= len(v)
		n, err := strconv.ParseFloat(string(v), 64)
		return n, err == nil && !math.IsInf(n, 0) && !math.IsNaN(n)
	case []any:
		if len(v) > limit.nodes || !meter.event(providerCopyEvent, uint64(len(v))*uint64(reflect.TypeFor[any]().Size())) {
			return nil, false
		}
		out := make([]any, len(v))
		for i, child := range v {
			copy, ok := replayCommandNativeObserved(child, depth+1, limit, meter)
			if !ok || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[any]().Size())) {
				return nil, false
			}
			out[i] = copy
		}
		return out, true
	case map[string]any:
		if len(v) > limit.nodes || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[map[string]any]().Size())) {
			return nil, false
		}
		out := make(map[string]any, len(v))
		for key, child := range v {
			copiedKey, ok := replayCommandNativeObserved(key, depth+1, limit, meter)
			if !ok {
				return nil, false
			}
			copied, ok := replayCommandNativeObserved(child, depth+1, limit, meter)
			if !ok || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[string]().Size()+reflect.TypeFor[any]().Size())) {
				return nil, false
			}
			out[copiedKey.(string)] = copied
		}
		return out, true
	}
	return nil, false
}

func replayCommandObjectObserved(value any, names []string, meter *providerWorkMeter) map[string]any {
	m, ok := value.(map[string]any)
	if meter == nil || !ok || len(m) != len(names) || len(names) > 8 {
		return nil
	}
	for _, name := range names {
		if !meter.event(providerCompareEvent, uint64(len(name))) {
			return nil
		}
		if _, present := m[name]; !present {
			return nil
		}
	}
	return m
}
