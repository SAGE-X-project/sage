package registry011

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"testing"
)

// This normal recorded UNIT selects separate proof quotas before profile,
// anchor, H0 and either observer. The synthetic witness is test-only. It does
// not provide a production factory, operating qualification or new allowance.
func historyReadProofReadyFixture(t *testing.T, recoveryProofs, finalProofs uint64) (*providerCustodyStore, *providerHistorySettledResults) {
	t.Helper()
	manifest, recorded := historyCollectionRecordedRunManifest(t)
	in := manifest.input
	s := providerHistoryProvisionedInspectionFixture(t, providerScanFixtureSelection{
		capture: &in.capture, pin: true, pinLargeObject: true, historyProgram: true,
		historyGroupWrite: true, historyGroupConsume: true, historyGroupDispatch: true,
		historyRecoveryNamespace: true, historyRecoveryLegacy: true,
		historyCollection: manifest, historyCollectionAdmission: true, historyCollectionFirstP: true,
		historyCollectionFullPack: true, historyCollectionRecordedRun: true, historyFinalRead: true,
		historyRecoveryProofs: recoveryProofs, historyFinalReadProofs: finalProofs,
	})
	first, err := newProviderFirstOperationCandidate(s)
	if err != nil || first == nil {
		t.Fatal("original first operation for the selected read quotas", err)
	}
	connection := providerIntegrationOwnedTLS(t, providerFirstOperationRequestBody, string(recorded["canonical-deployment"]))
	if result, err := first.executeCandidateWithConnection(context.Background(), connection); err != nil || result == nil {
		t.Fatal("normal recorded first exchange", err)
	}
	settled, err := s.history.runCollectionWithConnectionsCandidate(context.Background(), historyCollectionRecordedRunConnections(t, in, recorded))
	if err != nil || settled == nil {
		t.Fatal("complete normal recorded collection before the read primitive", err)
	}
	return s, settled
}

// These are genuine read-owner primitive checks, not an assertion that the
// recorded absent subject contains a historical key or that a complete
// key-bearing readback has been qualified. Both callers keep their original
// prepaid row, strict verifier, lifetime, clock and exclusive owner.
func TestProviderHistoryReadProofUsesOnlyItsOriginalPreheldAllowance(t *testing.T) {
	key := replayTestKey(t, "a", false)
	public, _ := replayBinary(key.Key, 32)
	signature, _ := replayBinary(key.Proof.Value, 64)
	message := replayChallenge(replayTestRegistry, replayTestAgent, key)
	if !replayStrictEd(public, message, signature) {
		t.Fatal("existing benign public proof precursor")
	}
	n := uint64(96 + len(message))
	proof := candidateWorkVector{2: n, 7: 1, 10: n*n + 1}
	for _, role := range []string{"recovery", "final"} {
		for _, funded := range []bool{false, true} {
			name := role + "/unfunded"
			if funded {
				name = role + "/funded"
			}
			t.Run(name, func(t *testing.T) {
				// The OTHER role always has two unspent attempts. It cannot
				// supply this role's missing first or exhausted second proof.
				recoveryProofs, finalProofs := uint64(2), uint64(2)
				selected := uint64(0)
				if funded {
					selected = 1
				}
				if role == "recovery" {
					recoveryProofs = selected
				} else {
					finalProofs = selected
				}
				s, settled := historyReadProofReadyFixture(t, recoveryProofs, finalProofs)
				p, g := s.history, s.history.group
				r, other := p.recovery, p.final.reader
				if role == "recovery" {
					s.mu.Lock()
					g.failLocked()
					s.mu.Unlock()
					if r.beginInspectionCandidate(context.Background(), g) != nil {
						t.Fatal("original failed-inspection entry")
					}
				} else {
					r, other = p.final.reader, p.recovery
					if p.final.beginCandidate(context.Background(), settled) != nil {
						t.Fatal("original successful-final entry")
					}
				}
				s.mu.Lock()
				defer s.mu.Unlock()
				m := r.meter
				if !r.activeLocked(m) || m.authority || m.upper[7] != selected ||
					other.meter.upper[7] != 2 || other.meter.actual[7] != 0 || !m.event(providerCopyEvent, 0) {
					t.Fatal("genuine active read quota precursor")
				}
				before, execution, otherWork := m.actual, g.meter.actual, other.meter.actual
				reads, held, pin, original := s.clock.read, s.stored, g.pin, p.original
				upper, reservation, future := m.upper, m.reservation, p.bounds.future
				if accepted := replayStrictEdObserved(public, message, signature, m); accepted != funded {
					t.Fatal("strict proof ignored its original read allowance", role, funded)
				}
				want := before
				if funded {
					for i, units := range proof {
						want[i] += units
					}
				}
				if m.actual != want || m.failed == funded || m.closed {
					t.Fatal("proof inputs and attempt were not admitted atomically", m.actual, want)
				}
				// A refused first attempt or exhausted second attempt cannot
				// borrow the other reader's two proofs or the execution row.
				if replayStrictEdObserved(public, message, signature, m) || !m.failed || m.actual != want ||
					m.upper != upper || m.reservation != reservation || other.meter.actual != otherWork ||
					g.meter.actual != execution || s.clock.read != reads || s.stored != held || g.pin != pin ||
					p.original != original || p.bounds.future != future {
					t.Fatal("proof refusal renewed, borrowed or changed an original reservation")
				}
				for _, coordinate := range []int{0, 1, 8, 9} {
					if upper[coordinate] != 0 || m.actual[coordinate] != 0 {
						t.Fatal("historical verification acquired network, segment or write work", coordinate)
					}
				}
				for _, kind := range []providerWorkEvent{providerWriteEvent, providerSyncEvent, providerRenameEvent, providerTLSEvent} {
					if providerHistoryRecoveryEventAllowed(kind) {
						t.Fatal("historical proof quota enabled an effect primitive", kind)
					}
				}
				r.stopLocked(true)
				if !m.closed || !m.failed || r.active || !r.closed ||
					replayStrictEdObserved(public, message, signature, m) || m.actual != want ||
					s.clock.read != reads || s.stored != held || other.meter.actual != otherWork || g.meter.actual != execution {
					t.Fatal("closed proof reader resumed or refunded original work")
				}
			})
		}
	}
}

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
