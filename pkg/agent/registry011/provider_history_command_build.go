package registry011

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strconv"

	"github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"
)

func replayGeneratedObjectObserved(names []string, values []any, meter *providerWorkMeter) (map[string]any, bool) {
	if meter == nil || len(names) != len(values) || len(names) > 8 || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[map[string]any]().Size())) {
		return nil, false
	}
	out := make(map[string]any, len(names))
	for i, name := range names {
		if len(name) == 0 || len(name) > 32 || !meter.event(providerCompareEvent, uint64(len(name))) || !contentASCII(name) {
			return nil, false
		}
		if _, exists := out[name]; exists {
			return nil, false
		}
		if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[string]().Size()+reflect.TypeFor[any]().Size())) {
			return nil, false
		}
		out[name] = values[i]
	}
	return out, true
}

func replayGeneratedArrayObserved(count int, meter *providerWorkMeter) ([]any, bool) {
	if meter == nil || count < 0 || count > 128 || !meter.event(providerCopyEvent, uint64(count)*uint64(reflect.TypeFor[any]().Size())) {
		return nil, false
	}
	return make([]any, count), true
}

func replayGeneratedArrayPutObserved(values []any, index int, value any, meter *providerWorkMeter) bool {
	if meter == nil || index < 0 || index >= len(values) || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[any]().Size())) {
		return false
	}
	values[index] = value
	return true
}

func replayGeneratedBase64Observed(raw []byte, meter *providerWorkMeter) (string, bool) {
	if meter == nil || len(raw) > 65 {
		return "", false
	}
	maximum := base64.RawURLEncoding.EncodedLen(len(raw))
	if !meter.preflight(providerEncodeEvent, uint64(maximum)) {
		return "", false
	}
	value := base64.RawURLEncoding.EncodeToString(raw)
	if len(value) != maximum || !meter.event(providerEncodeEvent, uint64(len(value))) {
		return "", false
	}
	return value, true
}

func replayGeneratedUintObserved(value uint64, meter *providerWorkMeter) (string, bool) {
	if meter == nil || !meter.preflight(providerEncodeEvent, 20) {
		return "", false
	}
	text := strconv.FormatUint(value, 10)
	return text, meter.event(providerEncodeEvent, uint64(len(text)))
}

// First visit the actual closed tree, including its keys and Number text.
// Six bytes per source byte cover ASCII escapes/decimal expansion; six per
// visited node cover quotes, separators, containers and null/bool literals.
// Both counts are independently capped by the original tree walker, not a
// producer's reported length. Native sorting/allocator/CPU costs are separate.
func replayGeneratedMarshalObserved(value any, meter *providerWorkMeter) ([]byte, bool) {
	limit := replayCommandJSONLimit{nodes: 262144, bytes: 262144}
	native, ok := replayCommandNativeObserved(value, 0, &limit, meter)
	if !ok {
		return nil, false
	}
	maximum := 6*((262144-limit.nodes)+(262144-limit.bytes)) + 64
	if !meter.preflight(providerEncodeEvent, uint64(maximum)) {
		return nil, false
	}
	raw, err := rfc8785.Marshal(native)
	if err != nil || len(raw) > maximum {
		meter.refuse()
		return nil, false
	}
	if !meter.event(providerEncodeEvent, uint64(len(raw))) {
		return nil, false
	}
	return raw, true
}

func replayGeneratedKeyObserved(key replayABIKey, meter *providerWorkMeter) (map[string]any, bool) {
	if !key.HasExpiry && key.Expires != 0 || key.HasExpiry && key.Expires > 1<<53-1 {
		return nil, false
	}
	material, a := replayGeneratedBase64Observed(key.Key, meter)
	proof, b := replayGeneratedBase64Observed(key.Proof, meter)
	if !a || !b {
		return nil, false
	}
	proofValue, ok := replayGeneratedObjectObserved([]string{"signer", "value"}, []any{key.Signer, proof}, meter)
	if !ok {
		return nil, false
	}
	if key.HasExpiry {
		expires, ok := replayGeneratedUintObserved(key.Expires, meter)
		if !ok {
			return nil, false
		}
		return replayGeneratedObjectObserved([]string{"name", "alg", "key", "proof", "state", "expires"}, []any{key.Name, key.Alg, material, proofValue, "accepted", json.Number(expires)}, meter)
	}
	return replayGeneratedObjectObserved([]string{"name", "alg", "key", "proof", "state"}, []any{key.Name, key.Alg, material, proofValue, "accepted"}, meter)
}

func replayGeneratedKeysObserved(keys []replayABIKey, meter *providerWorkMeter) ([]any, bool) {
	out, ok := replayGeneratedArrayObserved(len(keys), meter)
	if !ok {
		return nil, false
	}
	for i, key := range keys {
		value, ok := replayGeneratedKeyObserved(key, meter)
		if !ok || !replayGeneratedArrayPutObserved(out, i, value, meter) {
			return nil, false
		}
	}
	return out, true
}

func replayGeneratedServicesObserved(services []replayABIService, meter *providerWorkMeter) ([]any, bool) {
	if len(services) > 16 {
		return nil, false
	}
	out, ok := replayGeneratedArrayObserved(len(services), meter)
	if !ok {
		return nil, false
	}
	for i, service := range services {
		value, ok := replayGeneratedObjectObserved([]string{"name", "type", "uri"}, []any{service.Name, service.ServiceType, service.URI}, meter)
		if !ok || !replayGeneratedArrayPutObserved(out, i, value, meter) {
			return nil, false
		}
	}
	return out, true
}
