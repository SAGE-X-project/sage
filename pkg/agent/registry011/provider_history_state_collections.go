package registry011

import "reflect"

func replayStateUsableObserved(key replayKey, now uint64, meter *providerWorkMeter) bool {
	return meter != nil && providerReplayTextEqual(key.State, "accepted", meter) && providerReplayTextEqual(key.Alg, "ed25519", meter) && (key.Expires == nil || now < *key.Expires)
}

func replayStateHasSignerObserved(keys []replayKey, now uint64, meter *providerWorkMeter) bool {
	if meter == nil || len(keys) > 128 {
		return false
	}
	for _, key := range keys {
		if replayStateUsableObserved(key, now, meter) {
			return true
		}
	}
	return false
}

func (s *replayState) authorizedObserved(c replayCommand, meter *providerWorkMeter) bool {
	if s == nil || meter == nil || len(s.grants) > 128 {
		return false
	}
	if providerReplayTextEqual(c.actor, s.record.Controller, meter) {
		return true
	}
	if providerReplayTextEqual(c.op, "authorize-operator", meter) || providerReplayTextEqual(c.op, "revoke-operator", meter) || (providerReplayTextEqual(c.op, "revoke-key", meter) && !providerReplayTextEqual(s.record.State, "active", meter)) {
		return false
	}
	for _, grant := range s.grants {
		if providerReplayTextEqual(grant.Operator, c.actor, meter) && providerReplayTextEqual(grant.Scope, c.op, meter) {
			return true
		}
	}
	return false
}

func replayStateInsertKeyObserved(keys []replayKey, services []replayService, key replayKey, meter *providerWorkMeter) ([]replayKey, bool) {
	if meter == nil || len(keys) >= 128 || len(services) > 16 {
		return nil, false
	}
	position := len(keys)
	for i, old := range keys {
		if providerReplayTextEqual(old.Name, key.Name, meter) || providerReplayTextEqual(old.Key, key.Key, meter) || !meter.event(providerCompareEvent, uint64(len(key.Name)+len(old.Name))) {
			return nil, false
		}
		if position == len(keys) && key.Name < old.Name {
			position = i
		}
	}
	for _, service := range services {
		if providerReplayTextEqual(service.Name, key.Name, meter) {
			return nil, false
		}
	}
	owned, ok := replayStateKeyCloneObserved(key, meter)
	if !ok || !meter.event(providerCopyEvent, uint64(len(keys)+1)*uint64(reflect.TypeFor[replayKey]().Size())) {
		return nil, false
	}
	out := make([]replayKey, len(keys)+1)
	for i := range out {
		if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayKey]().Size())) {
			return nil, false
		}
		switch {
		case i < position:
			out[i] = keys[i]
		case i == position:
			out[i] = owned
		default:
			out[i] = keys[i-1]
		}
	}
	return out, true
}

func replayStateGrantAllowedObserved(state, scope string, meter *providerWorkMeter) bool {
	if meter == nil {
		return false
	}
	if providerReplayTextEqual(scope, "deactivate", meter) {
		return true
	}
	if providerReplayTextEqual(state, "created", meter) {
		return providerReplayTextEqual(scope, "activate", meter)
	}
	return providerReplayTextEqual(state, "active", meter) && (providerReplayTextEqual(scope, "add-key", meter) || providerReplayTextEqual(scope, "revoke-key", meter) || providerReplayTextEqual(scope, "update-services", meter))
}

func replayStateGrantChangeObserved(grants []replayGrant, controller, state string, c replayCommand, meter *providerWorkMeter) ([]replayGrant, bool) {
	if meter == nil || len(grants) > 128 || !replayCommandGrantObserved(c.args, meter) {
		return nil, false
	}
	actor, a := replayStateTextObserved(contentString(c.args["operator"]), 42, meter)
	scope, b := replayStateTextObserved(contentString(c.args["scope"]), 16, meter)
	if !a || !b || providerReplayTextEqual(actor, controller, meter) {
		return nil, false
	}
	grant := replayGrant{actor, scope}
	index, position := -1, len(grants)
	for i, old := range grants {
		sameActor := providerReplayTextEqual(old.Operator, actor, meter)
		if sameActor && providerReplayTextEqual(old.Scope, scope, meter) {
			index = i
		}
		if !meter.event(providerCompareEvent, uint64(len(old.Operator)+len(actor)+len(old.Scope)+len(scope))) {
			return nil, false
		}
		if position == len(grants) && (actor < old.Operator || sameActor && scope < old.Scope) {
			position = i
		}
	}
	if providerReplayTextEqual(c.op, "authorize-operator", meter) {
		if !replayStateGrantAllowedObserved(state, scope, meter) || index >= 0 || len(grants) >= 128 {
			return nil, false
		}
		out, ok := replayStateGrantStorageObserved(len(grants)+1, meter)
		if !ok {
			return nil, false
		}
		for i := range out {
			value := grant
			if i < position {
				value = grants[i]
			} else if i > position {
				value = grants[i-1]
			}
			if !replayStateGrantPutObserved(out, i, value, meter) {
				return nil, false
			}
		}
		return out, true
	}
	if !providerReplayTextEqual(c.op, "revoke-operator", meter) || index < 0 {
		return nil, false
	}
	out, ok := replayStateGrantStorageObserved(len(grants)-1, meter)
	if !ok {
		return nil, false
	}
	for i := range out {
		old := i
		if i >= index {
			old++
		}
		if !replayStateGrantPutObserved(out, i, grants[old], meter) {
			return nil, false
		}
	}
	return out, true
}

func replayStateSplitGrantsObserved(grants []replayGrant, deactivated, activating bool, meter *providerWorkMeter) ([]replayGrant, []replayGrant, bool) {
	if meter == nil || len(grants) > 128 || !meter.event(providerCopyEvent, uint64(len(grants))) {
		return nil, nil, false
	}
	mask := make([]bool, len(grants))
	count := 0
	for i, grant := range grants {
		remove := deactivated || activating && providerReplayTextEqual(grant.Scope, "activate", meter)
		if !meter.event(providerCopyEvent, 1) {
			return nil, nil, false
		}
		mask[i] = remove
		if remove {
			count++
		}
	}
	removed, a := replayStateGrantStorageObserved(count, meter)
	retained, b := replayStateGrantStorageObserved(len(grants)-count, meter)
	if !a || !b {
		return nil, nil, false
	}
	ri, ti := 0, 0
	for i, grant := range grants {
		if mask[i] {
			if !replayStateGrantPutObserved(removed, ri, grant, meter) {
				return nil, nil, false
			}
			ri++
		} else {
			if !replayStateGrantPutObserved(retained, ti, grant, meter) {
				return nil, nil, false
			}
			ti++
		}
	}
	return removed, retained, true
}
