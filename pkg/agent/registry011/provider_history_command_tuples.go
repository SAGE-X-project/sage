package registry011

import (
	"bytes"
	"reflect"
	"strings"
)

// Only anonymous tuples made by the pinned embedded ABI are expected here.
// Inspecting their fixed field tags/kinds avoids the former JSON marshal /
// unmarshal bridge and never invokes a value's custom marshaler. Output byte
// payloads and strings are owned copies; native reflection cost is separate.
func replayCommandTupleFieldsObserved(v reflect.Value, names []string, meter *providerWorkMeter) ([]reflect.Value, bool) {
	if meter == nil || !v.IsValid() || v.Kind() != reflect.Struct || v.Type().NumMethod() != 0 || v.NumField() != len(names) || len(names) > 7 || !meter.event(providerCopyEvent, uint64(len(names))*uint64(reflect.TypeFor[reflect.Value]().Size())) {
		return nil, false
	}
	fields := make([]reflect.Value, len(names))
	for i, name := range names {
		if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[reflect.StructField]().Size())) {
			return nil, false
		}
		field := v.Type().Field(i)
		if field.PkgPath != "" || !providerReplayTextEqual(field.Tag.Get("json"), name, meter) || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[reflect.Value]().Size())) {
			return nil, false
		}
		fields[i] = v.Field(i)
	}
	return fields, true
}

func replayCommandTupleStringObserved(value reflect.Value, maximum int, meter *providerWorkMeter) (string, bool) {
	if meter == nil || !value.IsValid() || value.Kind() != reflect.String || maximum < 1 || maximum > 2048 {
		return "", false
	}
	s := value.String()
	if len(s) > maximum || !meter.event(providerCompareEvent, uint64(len(s))) || !contentASCII(s) || !meter.event(providerCopyEvent, uint64(len(s))) {
		return "", false
	}
	return strings.Clone(s), true
}

func replayCommandTupleBytesObserved(value reflect.Value, meter *providerWorkMeter) ([]byte, bool) {
	if meter == nil || !value.IsValid() || value.Kind() != reflect.Slice || value.Type().Elem().Kind() != reflect.Uint8 || value.Len() > 65 || !meter.event(providerCopyEvent, uint64(value.Len())) {
		return nil, false
	}
	return bytes.Clone(value.Bytes()), true
}

func replayCommandTupleKeyObserved(value any, meter *providerWorkMeter) (replayABIKey, bool) {
	return replayCommandTupleKeyValueObserved(reflect.ValueOf(value), meter)
}

func replayCommandTupleKeyValueObserved(value reflect.Value, meter *providerWorkMeter) (replayABIKey, bool) {
	var empty replayABIKey
	fields, ok := replayCommandTupleFieldsObserved(value, []string{"name", "alg", "key", "signer", "proof", "hasExpiry", "expires"}, meter)
	if !ok {
		return empty, false
	}
	name, a := replayCommandTupleStringObserved(fields[0], 32, meter)
	alg, b := replayCommandTupleStringObserved(fields[1], 24, meter)
	key, c := replayCommandTupleBytesObserved(fields[2], meter)
	signer, d := replayCommandTupleStringObserved(fields[3], 189, meter)
	proof, e := replayCommandTupleBytesObserved(fields[4], meter)
	if !a || !b || !c || !d || !e || fields[5].Kind() != reflect.Bool || fields[6].Kind() != reflect.Uint64 || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayABIKey]().Size())) {
		return empty, false
	}
	return replayABIKey{Name: name, Alg: alg, Key: key, Signer: signer, Proof: proof, HasExpiry: fields[5].Bool(), Expires: fields[6].Uint()}, true
}

func replayCommandTupleServiceObserved(value any, meter *providerWorkMeter) (replayABIService, bool) {
	return replayCommandTupleServiceValueObserved(reflect.ValueOf(value), meter)
}

func replayCommandTupleServiceValueObserved(value reflect.Value, meter *providerWorkMeter) (replayABIService, bool) {
	var empty replayABIService
	fields, ok := replayCommandTupleFieldsObserved(value, []string{"name", "serviceType", "uri"}, meter)
	if !ok {
		return empty, false
	}
	name, a := replayCommandTupleStringObserved(fields[0], 32, meter)
	typeName, b := replayCommandTupleStringObserved(fields[1], 64, meter)
	uri, c := replayCommandTupleStringObserved(fields[2], 2048, meter)
	if !a || !b || !c || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayABIService]().Size())) {
		return empty, false
	}
	return replayABIService{Name: name, ServiceType: typeName, URI: uri}, true
}

func replayCommandTupleKeysObserved(value any, meter *providerWorkMeter) ([]replayABIKey, bool) {
	v := reflect.ValueOf(value)
	if meter == nil || !v.IsValid() || v.Kind() != reflect.Slice || v.Type().NumMethod() != 0 || v.Len() > 128 || !meter.preflightWork(candidateWorkVector{}) {
		return nil, false
	}
	if v.IsNil() {
		return nil, true
	}
	if !meter.event(providerCopyEvent, uint64(v.Len())*uint64(reflect.TypeFor[replayABIKey]().Size())) {
		return nil, false
	}
	out := make([]replayABIKey, v.Len())
	for i := range out {
		key, ok := replayCommandTupleKeyValueObserved(v.Index(i), meter)
		if !ok || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayABIKey]().Size())) {
			return nil, false
		}
		out[i] = key
	}
	return out, true
}

func replayCommandTupleServicesObserved(value any, meter *providerWorkMeter) ([]replayABIService, bool) {
	v := reflect.ValueOf(value)
	if meter == nil || !v.IsValid() || v.Kind() != reflect.Slice || v.Type().NumMethod() != 0 || v.Len() > 16 || !meter.preflightWork(candidateWorkVector{}) {
		return nil, false
	}
	if v.IsNil() {
		return nil, true
	}
	if !meter.event(providerCopyEvent, uint64(v.Len())*uint64(reflect.TypeFor[replayABIService]().Size())) {
		return nil, false
	}
	out := make([]replayABIService, v.Len())
	for i := range out {
		service, ok := replayCommandTupleServiceValueObserved(v.Index(i), meter)
		if !ok || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayABIService]().Size())) {
			return nil, false
		}
		out[i] = service
	}
	return out, true
}
