package registry011

import (
	"bytes"
	"reflect"
)

// Only the actual command reader selects this nonnil path. State is an owned
// working copy and is assigned after its complete successful result. This is
// not whole-event rollback: earlier private claim/index changes still require
// the collector to quarantine a failed run/window.
func (s *replayState) applyObserved(registry string, c replayCommand, now uint64, meter *providerWorkMeter) (HistoryContentStep, bool) {
	if meter == nil {
		return s.apply(registry, c, now)
	}
	var none HistoryContentStep
	if s == nil || now > 1<<53-1 || now < s.admitted || s.version == ^uint64(0) || c.expected != s.version || len(c.raw) == 0 || len(c.raw) > 262144 || len(c.agent) > 64 || !meter.event(providerCompareEvent, uint64(len(c.agent))) || !rpcAgent(c.agent) || !providerReplayTextEqual(registry, "eip155:11155111:"+SepoliaRegistryAddress, meter) {
		return none, false
	}
	working, ok := replayStateCloneObserved(*s, meter)
	if !ok {
		return none, false
	}
	added, ok := working.mutateObserved(registry, c, now, meter)
	if !ok {
		return none, false
	}
	previous := ""
	if working.version > 0 {
		previous, ok = replayGeneratedUintObserved(working.version, meter)
		if !ok {
			return none, false
		}
	}
	if !meter.event(providerCopyEvent, 16) {
		return none, false
	}
	working.version++
	working.admitted = now
	version, ok := replayGeneratedUintObserved(working.version, meter)
	if !ok || !replayStateSetTextObserved(&working.record.Version, version, meter) {
		return none, false
	}
	deactivated := providerReplayTextEqual(working.record.State, "deactivated", meter)
	activating := providerReplayTextEqual(c.op, "activate", meter)
	removed, retained, ok := replayStateSplitGrantsObserved(working.grants, deactivated, activating, meter)
	if !ok || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[[]replayGrant]().Size())) {
		return none, false
	}
	working.grants = retained
	record, ok := replayStateRecordBytesObserved(working.record, meter)
	if !ok {
		return none, false
	}
	// Keep the existing exact terminal/version-exhaustion headroom check
	// before any newly admitted proof verification.
	if !deactivated {
		if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayRecord]().Size())) {
			return none, false
		}
		future := working.record
		if !replayStateSetTextObserved(&future.State, "deactivated", meter) || !replayStateSetTextObserved(&future.Version, "18446744073709551615", meter) {
			return none, false
		}
		if _, ok := replayStateRecordBytesObserved(future, meter); !ok {
			return none, false
		}
	}
	if !replayVerifyNewKeysObserved(registry, c.agent, working.record.Keys, added, now, meter) {
		return none, false
	}
	grants, ok := replayStateGrantsBytesObserved(working.grants, meter)
	if !ok {
		return none, false
	}
	entry, ok := replayStateEntryObserved(version, previous, c, now, removed, record, grants, meter)
	if !ok || !meter.event(providerCopyEvent, uint64(len(c.raw))) {
		return none, false
	}
	command := bytes.Clone(c.raw)
	if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[HistoryContentStep]().Size())) {
		return none, false
	}
	step := HistoryContentStep{History: HistoryBytes{Version: working.version, Entry: entry, Command: command}, Record: record, Grants: grants}
	if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayState]().Size())) {
		return none, false
	}
	*s = working
	return step, true
}

func (s *replayState) mutateObserved(registry string, c replayCommand, now uint64, meter *providerWorkMeter) ([]replayKey, bool) {
	if providerReplayTextEqual(c.op, "create", meter) {
		if s.version != 0 || !replayStateHasSignerObserved(c.keys, now, meter) {
			return nil, false
		}
		keys, a := replayStateKeysCloneObserved(c.keys, meter)
		services, b := replayStateServicesCloneObserved(c.services, meter)
		grants, d := replayStateGrantStorageObserved(0, meter)
		if !a || !b || !d {
			return nil, false
		}
		if services == nil {
			if !meter.event(providerCopyEvent, 0) {
				return nil, false
			}
			services = make([]replayService, 0)
		}
		did := providerReplayJoin([]string{"did:sage:", registry, ":", c.agent}, meter)
		actor, ok := replayStateTextObserved(c.actor, 42, meter)
		if !ok || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayRecord]().Size()+reflect.TypeFor[[]replayGrant]().Size())) {
			return nil, false
		}
		s.record = replayRecord{ID: did, Controller: actor, Keys: keys, Services: services, State: "created"}
		s.grants = grants
		return c.keys, true
	}
	if s.version == 0 || providerReplayTextEqual(s.record.State, "deactivated", meter) || !s.authorizedObserved(c, meter) {
		return nil, false
	}
	switch c.op {
	case "activate":
		if !providerReplayTextEqual(s.record.State, "created", meter) || !replayStateHasSignerObserved(s.record.Keys, now, meter) || !replayStateSetTextObserved(&s.record.State, "active", meter) {
			return nil, false
		}
	case "deactivate":
		if !replayStateSetTextObserved(&s.record.State, "deactivated", meter) {
			return nil, false
		}
	case "add-key":
		if !providerReplayTextEqual(s.record.State, "active", meter) || len(c.keys) != 1 {
			return nil, false
		}
		keys, ok := replayStateInsertKeyObserved(s.record.Keys, s.record.Services, c.keys[0], meter)
		if !ok || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[[]replayKey]().Size())) {
			return nil, false
		}
		s.record.Keys = keys
		return c.keys, true
	case "revoke-key":
		found := false
		for i := range s.record.Keys {
			key := &s.record.Keys[i]
			if providerReplayTextEqual(key.Name, contentString(c.args["name"]), meter) && providerReplayTextEqual(key.State, "accepted", meter) {
				if !replayStateSetTextObserved(&key.State, "revoked", meter) {
					return nil, false
				}
				found = true
				break
			}
		}
		if !found {
			return nil, false
		}
		if !replayStateHasSignerObserved(s.record.Keys, now, meter) && !replayStateSetTextObserved(&s.record.State, "deactivated", meter) {
			return nil, false
		}
	case "update-services":
		if !providerReplayTextEqual(s.record.State, "active", meter) {
			return nil, false
		}
		services, ok := replayCommandServicesObserved(c.args["services"], s.record.Keys, meter)
		if !ok || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[[]replayService]().Size())) {
			return nil, false
		}
		s.record.Services = services
	case "authorize-operator", "revoke-operator":
		grants, ok := replayStateGrantChangeObserved(s.grants, s.record.Controller, s.record.State, c, meter)
		if !ok || !meter.event(providerCopyEvent, uint64(reflect.TypeFor[[]replayGrant]().Size())) {
			return nil, false
		}
		s.grants = grants
	default:
		return nil, false
	}
	return nil, meter.preflightWork(candidateWorkVector{})
}
