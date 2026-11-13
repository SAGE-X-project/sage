package registry011

import (
	"reflect"
	"strings"
)

func replayStateTextObserved(value string, maximum int, meter *providerWorkMeter) (string, bool) {
	if meter == nil || maximum < 1 || maximum > 2048 || len(value) > maximum || !meter.event(providerCompareEvent, uint64(len(value))) || !contentASCII(value) || !meter.event(providerCopyEvent, uint64(len(value))) {
		return "", false
	}
	return strings.Clone(value), true
}

func replayStateSetTextObserved(target *string, value string, meter *providerWorkMeter) bool {
	if target == nil || meter == nil || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[string]().Size())) {
		return false
	}
	*target = value
	return true
}

func replayStateKeyCloneObserved(key replayKey, meter *providerWorkMeter) (replayKey, bool) {
	if meter == nil || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayKey]().Size())) {
		return replayKey{}, false
	}
	out := key
	fields := [...]struct {
		target  *string
		maximum int
	}{{&out.Name, 32}, {&out.Alg, 24}, {&out.Key, 43}, {&out.Proof.Signer, 189}, {&out.Proof.Value, 86}, {&out.State, 16}}
	for _, field := range fields {
		value, ok := replayStateTextObserved(*field.target, field.maximum, meter)
		if !ok || !replayStateSetTextObserved(field.target, value, meter) {
			return replayKey{}, false
		}
	}
	if key.Expires != nil {
		if *key.Expires > 1<<53-1 || !meter.event(providerCopyEvent, 8) {
			return replayKey{}, false
		}
		expires := *key.Expires
		out.Expires = &expires
	}
	return out, true
}

func replayStateKeysCloneObserved(keys []replayKey, meter *providerWorkMeter) ([]replayKey, bool) {
	if meter == nil || len(keys) > 128 || !meter.event(providerCopyEvent, uint64(len(keys))*uint64(reflect.TypeFor[replayKey]().Size())) {
		return nil, false
	}
	if keys == nil {
		return nil, true
	}
	out := make([]replayKey, len(keys))
	for i, key := range keys {
		copy, ok := replayStateKeyCloneObserved(key, meter)
		if !ok || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayKey]().Size())) {
			return nil, false
		}
		out[i] = copy
	}
	return out, true
}

func replayStateServicesCloneObserved(services []replayService, meter *providerWorkMeter) ([]replayService, bool) {
	if meter == nil || len(services) > 16 || !meter.event(providerCopyEvent, uint64(len(services))*uint64(reflect.TypeFor[replayService]().Size())) {
		return nil, false
	}
	if services == nil {
		return nil, true
	}
	out := make([]replayService, len(services))
	for i, service := range services {
		name, a := replayStateTextObserved(service.Name, 32, meter)
		typeName, b := replayStateTextObserved(service.Type, 64, meter)
		uri, c := replayStateTextObserved(service.URI, 2048, meter)
		if !a || !b || !c || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayService]().Size())) {
			return nil, false
		}
		out[i] = replayService{name, typeName, uri}
	}
	return out, true
}

func replayStateGrantStorageObserved(count int, meter *providerWorkMeter) ([]replayGrant, bool) {
	if meter == nil || count < 0 || count > 128 || !meter.event(providerCopyEvent, uint64(count)*uint64(reflect.TypeFor[replayGrant]().Size())) {
		return nil, false
	}
	return make([]replayGrant, count), true
}

func replayStateGrantPutObserved(out []replayGrant, i int, grant replayGrant, meter *providerWorkMeter) bool {
	if meter == nil || i < 0 || i >= len(out) || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayGrant]().Size())) {
		return false
	}
	out[i] = grant
	return true
}

// This private working copy does not copy or reset a meter, lifetime claims,
// created identities or expected queue. Native backing/allocator costs remain
// separate from these explicit logical copies.
func replayStateCloneObserved(state replayState, meter *providerWorkMeter) (replayState, bool) {
	if meter == nil || len(state.grants) > 128 || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayState]().Size())) {
		return replayState{}, false
	}
	out := state
	fields := [...]struct {
		target  *string
		maximum int
	}{{&out.record.ID, 189}, {&out.record.Controller, 42}, {&out.record.State, 16}, {&out.record.Version, 20}}
	for _, field := range fields {
		value, ok := replayStateTextObserved(*field.target, field.maximum, meter)
		if !ok || !replayStateSetTextObserved(field.target, value, meter) {
			return replayState{}, false
		}
	}
	var ok bool
	out.record.Keys, ok = replayStateKeysCloneObserved(state.record.Keys, meter)
	if !ok {
		return replayState{}, false
	}
	out.record.Services, ok = replayStateServicesCloneObserved(state.record.Services, meter)
	if !ok {
		return replayState{}, false
	}
	if state.grants == nil {
		out.grants = nil
		return out, true
	}
	out.grants, ok = replayStateGrantStorageObserved(len(state.grants), meter)
	if !ok {
		return replayState{}, false
	}
	for i, grant := range state.grants {
		actor, a := replayStateTextObserved(grant.Operator, 42, meter)
		scope, b := replayStateTextObserved(grant.Scope, 16, meter)
		if !a || !b || !replayStateGrantPutObserved(out.grants, i, replayGrant{actor, scope}, meter) {
			return replayState{}, false
		}
	}
	return out, true
}
