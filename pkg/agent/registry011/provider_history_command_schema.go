package registry011

import (
	"reflect"
	"strings"
)

func replayCommandKeysObserved(value any, did string, meter *providerWorkMeter) ([]replayKey, bool) {
	a, ok := value.([]any)
	if meter == nil || !ok || len(a) == 0 || len(a) > 128 || len(did) > 189 || !meter.event(providerCopyEvent, uint64(len(a))*uint64(reflect.TypeFor[replayKey]().Size())) {
		return nil, false
	}
	out := make([]replayKey, len(a))
	if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[map[[32]byte]bool]().Size())) {
		return nil, false
	}
	materials := map[[32]byte]bool{}
	previous := ""
	prefix := providerReplayJoin([]string{did, "#"}, meter)
	if !meter.preflightWork(candidateWorkVector{}) {
		return nil, false
	}
	for i, value := range a {
		m, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		names := []string{"name", "alg", "key", "proof", "state"}
		if _, exists := m["expires"]; exists {
			names = []string{"name", "alg", "key", "proof", "state", "expires"}
		}
		if replayCommandObjectObserved(m, names, meter) == nil {
			return nil, false
		}
		k := replayKey{Name: contentString(m["name"]), Alg: contentString(m["alg"]), Key: contentString(m["key"]), State: contentString(m["state"])}
		if (!providerReplayTextEqual(k.Alg, "ed25519", meter) && !providerReplayTextEqual(k.Alg, "x25519", meter)) || !replayCommandNameObserved(k.Name, meter) || !meter.event(providerCompareEvent, uint64(len(k.Name)+len(previous))) || k.Name <= previous || !providerReplayTextEqual(k.State, "accepted", meter) {
			return nil, false
		}
		material, ok := replayCommandBinaryObserved(m["key"], 32, meter)
		if !ok || len(material) != 32 || !meter.event(providerCopyEvent, 32) {
			return nil, false
		}
		var id [32]byte
		copy(id[:], material)
		if materials[id] || !meter.event(providerCopyEvent, 33) {
			return nil, false
		}
		materials[id] = true
		proof := replayCommandObjectObserved(m["proof"], []string{"signer", "value"}, meter)
		if proof == nil {
			return nil, false
		}
		k.Proof = replayProof{Signer: contentString(proof["signer"]), Value: contentString(proof["value"])}
		if _, ok := replayCommandBinaryObserved(proof["value"], 64, meter); !ok {
			return nil, false
		}
		if !meter.event(providerCompareEvent, uint64(len(k.Proof.Signer)+len(prefix))) || !strings.HasPrefix(k.Proof.Signer, prefix) || !replayCommandNameObserved(strings.TrimPrefix(k.Proof.Signer, prefix), meter) {
			return nil, false
		}
		if providerReplayTextEqual(k.Alg, "ed25519", meter) {
			self := providerReplayJoin([]string{prefix, k.Name}, meter)
			if !providerReplayTextEqual(k.Proof.Signer, self, meter) {
				return nil, false
			}
		}
		if x, present := m["expires"]; present {
			n, ok := replayCommandSecondsObserved(x, meter)
			if !ok || !meter.event(providerCopyEvent, 8) {
				return nil, false
			}
			k.Expires = &n
		}
		if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayKey]().Size())) {
			return nil, false
		}
		out[i] = k
		previous = k.Name
	}
	if !meter.preflightWork(candidateWorkVector{}) {
		return nil, false
	}
	return out, true
}

func replayCommandServicesObserved(value any, keys []replayKey, meter *providerWorkMeter) ([]replayService, bool) {
	a, ok := value.([]any)
	if meter == nil || !ok || len(a) > 16 || len(keys) > 128 || !meter.event(providerCopyEvent, uint64(len(a))*uint64(reflect.TypeFor[replayService]().Size())) {
		return nil, false
	}
	out := make([]replayService, len(a))
	previous := ""
	for i, value := range a {
		m := replayCommandObjectObserved(value, []string{"name", "type", "uri"}, meter)
		if m == nil {
			return nil, false
		}
		s := replayService{Name: contentString(m["name"]), Type: contentString(m["type"]), URI: contentString(m["uri"])}
		if !replayCommandNameObserved(s.Name, meter) || !meter.event(providerCompareEvent, uint64(len(s.Name)+len(previous))) || s.Name <= previous || len(s.Type) == 0 || len(s.Type) > 64 || !meter.event(providerCompareEvent, uint64(len(s.Type))) || !contentASCII(s.Type) || !replayCommandURIObserved(s.URI, meter) {
			return nil, false
		}
		for _, k := range keys {
			if providerReplayTextEqual(k.Name, s.Name, meter) {
				return nil, false
			}
			if !meter.preflightWork(candidateWorkVector{}) {
				return nil, false
			}
		}
		if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayService]().Size())) {
			return nil, false
		}
		out[i] = s
		previous = s.Name
	}
	if !meter.preflightWork(candidateWorkVector{}) {
		return nil, false
	}
	return out, true
}
