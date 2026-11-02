package registry011

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"testing"
)

// Only units that enter actual required-suite proof verification get this
// explicit existing4096-proof DATA allowance. Execution-only meters retain0.
func providerHistoryStateUnitMeter() *providerWorkMeter {
	return providerHistoryContentUnitMeter(candidateWorkVector{2: 1 << 30, 5: 65536, 6: 65536, 7: 4096, 10: 1 << 60})
}

func TestProviderHistoryProofChallengeMatchesAdoptedLengthFraming(t *testing.T) {
	create := replayTestCreate(t)
	for _, key := range create.keys {
		m := providerHistoryStateUnitMeter()
		got, ok := replayChallengeObserved(replayTestRegistry, replayTestAgent, key, m)
		want := replayChallenge(replayTestRegistry, replayTestAgent, key)
		if !ok || !bytes.Equal(got, want) || m.actual[7] != 0 {
			t.Fatal("actual adopted challenge bytes", key.Alg, m.actual)
		}
		prefix := []byte("sage-pop-0.11.0")
		if !bytes.HasPrefix(got, prefix) {
			t.Fatal("challenge domain changed")
		}
		pos := len(prefix)
		material, ok := replayBinary(key.Key, 32)
		if !ok {
			t.Fatal("normal material precursor")
		}
		for _, field := range [][]byte{[]byte(replayTestRegistry), []byte(replayTestAgent), []byte(key.Name), []byte(key.Alg), material} {
			size := int(binary.BigEndian.Uint16(got[pos : pos+2]))
			pos += 2
			if size != len(field) || !bytes.Equal(got[pos:pos+size], field) {
				t.Fatal("field order/length not derived from actual input")
			}
			pos += size
		}
		if pos != len(got) {
			t.Fatal("unexpected challenge suffix")
		}
		legacy, ok := replayChallengeObserved(replayTestRegistry, replayTestAgent, key, nil)
		if !ok || !bytes.Equal(legacy, want) {
			t.Fatal("nil challenge changed")
		}
	}
}

func TestProviderHistoryStrictProofCountsOnlyAnAdmittedNativeInvocation(t *testing.T) {
	key := replayTestKey(t, "a", false)
	public, _ := replayBinary(key.Key, 32)
	signature, _ := replayBinary(key.Proof.Value, 64)
	message := replayChallenge(replayTestRegistry, replayTestAgent, key)
	if !replayStrictEd(public, message, signature) {
		t.Fatal("existing public RFC fixture precursor")
	}
	n := uint64(96 + len(message))
	want := candidateWorkVector{2: n, 7: 1, 10: n*n + 1}
	m := providerHistoryContentUnitMeter(want)
	if !replayStrictEdObserved(public, message, signature, m) || m.actual != want {
		t.Fatal("one actual strict verification invocation", m.actual, want)
	}
	for _, shortIndex := range []int{2, 7, 10} {
		upper := want
		upper[shortIndex]--
		short := providerHistoryContentUnitMeter(upper)
		if replayStrictEdObserved(public, message, signature, short) || !short.failed || short.actual != (candidateWorkVector{}) {
			t.Fatal("atomic input/attempt reservation", shortIndex, short.actual)
		}
		before := short.actual
		if replayStrictEdObserved(public, message, signature, short) || short.actual != before {
			t.Fatal("failed native proof observer resumed")
		}
	}
	if !replayStrictEdObserved(public, message, signature, nil) {
		t.Fatal("legacy strict proof")
	}
}

func TestProviderHistoryObservedProofsKeepSigningAndKEMRolesSeparate(t *testing.T) {
	create := replayTestCreate(t)
	if !replayVerifyNewKeys(replayTestRegistry, replayTestAgent, create.keys, create.keys, 0) {
		t.Fatal("existing required-suite proof precursor")
	}
	m := providerHistoryStateUnitMeter()
	if !replayVerifyNewKeysObserved(replayTestRegistry, replayTestAgent, create.keys, create.keys, 0, m) || m.actual[7] != 2 {
		t.Fatal("Ed self proof and Ed endorsement of KEM", m.actual)
	}
	// Historical endorsements are not new proofs after their signer retires.
	old := append([]replayKey{}, create.keys...)
	old[0].State = "revoked"
	m = providerHistoryStateUnitMeter()
	if !replayVerifyNewKeysObserved(replayTestRegistry, replayTestAgent, old, nil, 100, m) || m.actual[7] != 0 {
		t.Fatal("historical key retained as new proof")
	}
	kem := create.keys[1]
	for _, state := range []string{"missing", "revoked", "expired", "kem-as-signer"} {
		signer := create.keys[0]
		all := []replayKey{signer}
		switch state {
		case "missing":
			all = nil
		case "revoked":
			all[0].State = "revoked"
		case "expired":
			zero := uint64(0)
			all[0].Expires = &zero
		case "kem-as-signer":
			all[0].Alg = "x25519"
		}
		m := providerHistoryStateUnitMeter()
		if replayVerifyNewKeysObserved(replayTestRegistry, replayTestAgent, all, []replayKey{kem}, 0, m) || m.actual[7] != 0 {
			t.Fatal("unusable signing role entered native verification", state, m.actual)
		}
	}
}

func TestProviderHistoryObservedProofFailuresRetainTheirActualAttempt(t *testing.T) {
	key := replayTestKey(t, "a", false)
	bad := key
	signature, _ := replayBinary(key.Proof.Value, 64)
	signature[0] ^= 1
	bad.Proof.Value = base64.RawURLEncoding.EncodeToString(signature)
	m := providerHistoryStateUnitMeter()
	if replayVerifyNewKeysObserved(replayTestRegistry, replayTestAgent, []replayKey{bad}, []replayKey{bad}, 0, m) || m.actual[7] != 1 {
		t.Fatal("failed actual strict proof was hidden", m.actual)
	}
	for _, change := range []string{"registry", "agent", "name", "algorithm", "material"} {
		registry, agent, bad := replayTestRegistry, replayTestAgent, key
		switch change {
		case "registry":
			registry = "other"
		case "agent":
			agent = ""
		case "name":
			bad.Name = "invalid.name"
		case "algorithm":
			bad.Alg = "p256"
		case "material":
			bad.Key = "AA"
		}
		if got, ok := replayChallengeObserved(registry, agent, bad, providerHistoryStateUnitMeter()); ok || got != nil {
			t.Fatal("invalid challenge precursor", change)
		}
	}
	if replayVerifyNewKeysObserved(replayTestRegistry, replayTestAgent, make([]replayKey, 129), nil, 0, providerHistoryStateUnitMeter()) || replayVerifyNewKeysObserved(replayTestRegistry, replayTestAgent, nil, make([]replayKey, 129), 0, providerHistoryStateUnitMeter()) {
		t.Fatal("proof array cap")
	}
	if replayStrictEdObserved(nil, nil, nil, providerHistoryStateUnitMeter()) || replayStrictEdObserved(make([]byte, 32), make([]byte, 513), make([]byte, 64), providerHistoryStateUnitMeter()) {
		t.Fatal("unbounded native verification input")
	}
}
