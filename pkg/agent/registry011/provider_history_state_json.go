package registry011

import (
	"encoding/json"
	"reflect"
)

// History has ten fixed fields; command objects keep their separate eight
// field limit. Only the private actual typed state supplies these values.
func replayStateObjectObserved(names []string, values []any, meter *providerWorkMeter) (map[string]any, bool) {
	if meter == nil || len(names) != len(values) || len(names) > 10 || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[map[string]any]().Size())) {
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

func replayStateKeysValueObserved(keys []replayKey, meter *providerWorkMeter) ([]any, bool) {
	out, ok := replayGeneratedArrayObserved(len(keys), meter)
	if !ok {
		return nil, false
	}
	for i, key := range keys {
		proof, ok := replayGeneratedObjectObserved([]string{"signer", "value"}, []any{key.Proof.Signer, key.Proof.Value}, meter)
		if !ok {
			return nil, false
		}
		var value map[string]any
		if key.Expires != nil {
			if *key.Expires > 1<<53-1 {
				return nil, false
			}
			expires, ok := replayGeneratedUintObserved(*key.Expires, meter)
			if !ok {
				return nil, false
			}
			value, ok = replayGeneratedObjectObserved([]string{"name", "alg", "key", "proof", "state", "expires"}, []any{key.Name, key.Alg, key.Key, proof, key.State, json.Number(expires)}, meter)
			if !ok {
				return nil, false
			}
		} else {
			value, ok = replayGeneratedObjectObserved([]string{"name", "alg", "key", "proof", "state"}, []any{key.Name, key.Alg, key.Key, proof, key.State}, meter)
			if !ok {
				return nil, false
			}
		}
		if !replayGeneratedArrayPutObserved(out, i, value, meter) {
			return nil, false
		}
	}
	return out, true
}

func replayStateServicesValueObserved(services []replayService, meter *providerWorkMeter) ([]any, bool) {
	if len(services) > 16 {
		return nil, false
	}
	out, ok := replayGeneratedArrayObserved(len(services), meter)
	if !ok {
		return nil, false
	}
	for i, service := range services {
		value, ok := replayGeneratedObjectObserved([]string{"name", "type", "uri"}, []any{service.Name, service.Type, service.URI}, meter)
		if !ok || !replayGeneratedArrayPutObserved(out, i, value, meter) {
			return nil, false
		}
	}
	return out, true
}

func replayStateGrantsValueObserved(grants []replayGrant, meter *providerWorkMeter) ([]any, bool) {
	out, ok := replayGeneratedArrayObserved(len(grants), meter)
	if !ok {
		return nil, false
	}
	for i, grant := range grants {
		value, ok := replayGeneratedObjectObserved([]string{"operator", "scope"}, []any{grant.Operator, grant.Scope}, meter)
		if !ok || !replayGeneratedArrayPutObserved(out, i, value, meter) {
			return nil, false
		}
	}
	return out, true
}

func replayStateRecordBytesObserved(record replayRecord, meter *providerWorkMeter) ([]byte, bool) {
	keys, a := replayStateKeysValueObserved(record.Keys, meter)
	services, b := replayStateServicesValueObserved(record.Services, meter)
	if !a || !b {
		return nil, false
	}
	value, ok := replayGeneratedObjectObserved([]string{"id", "controller", "keys", "services", "state", "version"}, []any{record.ID, record.Controller, keys, services, record.State, record.Version}, meter)
	if !ok {
		return nil, false
	}
	raw, ok := replayGeneratedMarshalObserved(value, meter)
	if !ok || len(raw) > 65536 {
		return nil, false
	}
	return raw, true
}

func replayStateGrantsBytesObserved(grants []replayGrant, meter *providerWorkMeter) ([]byte, bool) {
	value, ok := replayStateGrantsValueObserved(grants, meter)
	if !ok {
		return nil, false
	}
	raw, ok := replayGeneratedMarshalObserved(value, meter)
	if !ok || len(raw) > 262144 {
		return nil, false
	}
	return raw, true
}

func replayStateEntryObserved(version, previous string, c replayCommand, now uint64, removed []replayGrant, record, grants []byte, meter *providerWorkMeter) ([]byte, bool) {
	if now > 1<<53-1 {
		return nil, false
	}
	commandDigest, a := replayDigestObserved(c.raw, meter)
	recordDigest, b := replayDigestObserved(record, meter)
	grantsDigest, d := replayDigestObserved(grants, meter)
	removals, e := replayStateGrantsValueObserved(removed, meter)
	admitted, f := replayGeneratedUintObserved(now, meter)
	if !a || !b || !d || !e || !f {
		return nil, false
	}
	var target any
	if providerReplayTextEqual(c.op, "authorize-operator", meter) || providerReplayTextEqual(c.op, "revoke-operator", meter) {
		args := replayCommandObjectObserved(c.args, []string{"operator", "scope"}, meter)
		if args == nil {
			return nil, false
		}
		target = args
	}
	value, ok := replayStateObjectObserved([]string{"version", "previousVersion", "actor", "operation", "commandDigest", "target", "automaticRemovals", "admitted", "recordDigest", "grantsDigest"}, []any{version, previous, c.actor, c.op, commandDigest, target, removals, json.Number(admitted), recordDigest, grantsDigest}, meter)
	if !ok {
		return nil, false
	}
	raw, ok := replayGeneratedMarshalObserved(value, meter)
	if !ok || len(raw) > 262144 {
		return nil, false
	}
	return raw, true
}
