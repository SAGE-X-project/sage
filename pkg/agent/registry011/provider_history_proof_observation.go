package registry011

import (
	"encoding/binary"
	"reflect"
)

// This is the adopted five-field challenge, not a new signature domain or an
// arbitrary marshaler. Every output length is derived before its allocation.
func replayChallengeObserved(registry, agent string, key replayKey, meter *providerWorkMeter) ([]byte, bool) {
	if meter == nil {
		return replayChallenge(registry, agent, key), true
	}
	if !providerReplayTextEqual(registry, "eip155:11155111:"+SepoliaRegistryAddress, meter) || len(agent) > 64 || !meter.event(providerCompareEvent, uint64(len(agent))) || !rpcAgent(agent) || !replayCommandNameObserved(key.Name, meter) || (!providerReplayTextEqual(key.Alg, "ed25519", meter) && !providerReplayTextEqual(key.Alg, "x25519", meter)) {
		return nil, false
	}
	material, ok := replayCommandBinaryObserved(key.Key, 32, meter)
	if !ok {
		return nil, false
	}
	const domain = "sage-pop-0.11.0"
	fields := [...]string{registry, agent, key.Name, key.Alg}
	total := len(domain) + 2*(len(fields)+1) + len(material)
	for _, field := range fields {
		total += len(field)
	}
	if total > 512 || !meter.event(providerCopyEvent, uint64(total)) {
		return nil, false
	}
	out := make([]byte, total)
	if !meter.event(providerCopyEvent, uint64(len(domain))) {
		return nil, false
	}
	pos := copy(out, domain)
	for _, field := range fields {
		if !meter.event(providerEncodeEvent, 2) {
			return nil, false
		}
		binary.BigEndian.PutUint16(out[pos:pos+2], uint16(len(field)))
		pos += 2
		if !meter.event(providerCopyEvent, uint64(len(field))) {
			return nil, false
		}
		pos += copy(out[pos:], field)
	}
	if !meter.event(providerEncodeEvent, 2) {
		return nil, false
	}
	binary.BigEndian.PutUint16(out[pos:pos+2], uint16(len(material)))
	pos += 2
	if !meter.event(providerCopyEvent, uint64(len(material))) {
		return nil, false
	}
	pos += copy(out[pos:], material)
	if pos != len(out) {
		return nil, false
	}
	return out, true
}

// One counted proof is one actual strict verification invocation, including
// its original point/scalar checks. Input accounting and that attempt are
// admitted atomically; no proof is invented for a rejected input reservation.
// This bounded native call is not a native curve CPU/allocator/RAM certificate.
func replayStrictEdObserved(key, message, signature []byte, meter *providerWorkMeter) bool {
	if meter == nil {
		return replayStrictEd(key, message, signature)
	}
	if len(key) != 32 || len(signature) != 64 || len(message) > 512 {
		return false
	}
	work, ok := providerEventVector(providerCodecEvent, uint64(len(key)+len(message)+len(signature)))
	work[7] = 1
	return ok && meter.add(work) && replayStrictEd(key, message, signature)
}

// Actual keys have already passed the closed command/state schemas. Newly
// admitted signing proofs precede KEM endorsements, as in the nil legacy path.
func replayVerifyNewKeysObserved(registry, agent string, all, added []replayKey, now uint64, meter *providerWorkMeter) bool {
	if meter == nil {
		return replayVerifyNewKeys(registry, agent, all, added, now)
	}
	if len(all) > 128 || len(added) > 128 {
		return false
	}
	did := providerReplayJoin([]string{"did:sage:", registry, ":", agent}, meter)
	for i := range added {
		if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayKey]().Size())) {
			return false
		}
		key := added[i]
		if providerReplayTextEqual(key.Alg, "ed25519", meter) {
			material, a := replayCommandBinaryObserved(key.Key, 32, meter)
			signature, b := replayCommandBinaryObserved(key.Proof.Value, 64, meter)
			challenge, c := replayChallengeObserved(registry, agent, key, meter)
			if !a || !b || !c || !replayStrictEdObserved(material, challenge, signature, meter) {
				return false
			}
		} else if !providerReplayTextEqual(key.Alg, "x25519", meter) {
			return false
		}
	}
	for i := range added {
		if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayKey]().Size())) {
			return false
		}
		key := added[i]
		if !providerReplayTextEqual(key.Alg, "x25519", meter) {
			continue
		}
		found := false
		for j := range all {
			if !meter.event(providerCopyEvent, uint64(reflect.TypeFor[replayKey]().Size())) {
				return false
			}
			signer := all[j]
			name := providerReplayJoin([]string{did, "#", signer.Name}, meter)
			if !providerReplayTextEqual(key.Proof.Signer, name, meter) {
				continue
			}
			if !replayStateUsableObserved(signer, now, meter) {
				return false
			}
			material, a := replayCommandBinaryObserved(signer.Key, 32, meter)
			signature, b := replayCommandBinaryObserved(key.Proof.Value, 64, meter)
			challenge, c := replayChallengeObserved(registry, agent, key, meter)
			if !a || !b || !c || !replayStrictEdObserved(material, challenge, signature, meter) {
				return false
			}
			found = true
			break
		}
		if !found {
			return false
		}
	}
	return meter.preflightWork(candidateWorkVector{})
}
