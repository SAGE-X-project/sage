package registry011

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"filippo.io/edwards25519"
	"math/big"
)

// Replay proofs use the adopted 0.11 domain, never the old 0.10 challenge.
func replayChallenge(registry, agent string, k replayKey) []byte {
	raw, _ := replayBinary(k.Key, 32)
	out := []byte("sage-pop-0.11.0")
	for _, field := range [][]byte{[]byte(registry), []byte(agent), []byte(k.Name), []byte(k.Alg), raw} {
		out = binary.BigEndian.AppendUint16(out, uint16(len(field)))
		out = append(out, field...)
	}
	return out
}

func replayVerifyNewKeys(registry, agent string, all, added []replayKey, now uint64) bool {
	did := "did:sage:" + registry + ":" + agent
	// Only newly admitted signing proofs are verified, before KEM endorsements.
	for _, k := range added {
		if k.Alg == "ed25519" {
			raw, _ := replayBinary(k.Key, 32)
			proof, _ := replayBinary(k.Proof.Value, 64)
			if !replayStrictEd(raw, replayChallenge(registry, agent, k), proof) {
				return false
			}
		}
	}
	for _, k := range added {
		if k.Alg != "x25519" {
			continue
		}
		found := false
		for _, signer := range all {
			if k.Proof.Signer != did+"#"+signer.Name {
				continue
			}
			if !replayUsable(signer, now) {
				return false
			}
			raw, _ := replayBinary(signer.Key, 32)
			proof, _ := replayBinary(k.Proof.Value, 64)
			if !replayStrictEd(raw, replayChallenge(registry, agent, k), proof) {
				return false
			}
			found = true
			break
		}
		if !found {
			return false
		}
	}
	return true
}
func replayStrictEd(key, message, signature []byte) bool {
	if !replayPrimeEdPoint(key) || len(signature) != 64 || !replayPrimeEdPoint(signature[:32]) {
		return false
	}
	if _, e := new(edwards25519.Scalar).SetCanonicalBytes(signature[32:]); e != nil {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(key), message, signature)
}

func replayPrimeEdPoint(encoded []byte) bool {
	if len(encoded) != ed25519.PublicKeySize {
		return false
	}
	point, err := new(edwards25519.Point).SetBytes(encoded)
	if err != nil || !bytes.Equal(point.Bytes(), encoded) || point.Equal(edwards25519.NewIdentityPoint()) == 1 {
		return false
	}
	// [L]P must be identity. This also rejects mixed-torsion points that a
	// small-order-only check would leave in the input domain.
	order, _ := new(big.Int).SetString("7237005577332262213973186563042994240857116359379907606001950938285454250989", 10)
	accumulator := edwards25519.NewIdentityPoint()
	addend := point
	for i := 0; i < order.BitLen(); i++ {
		if order.Bit(i) == 1 {
			accumulator = new(edwards25519.Point).Add(accumulator, addend)
		}
		addend = new(edwards25519.Point).Add(addend, addend)
	}
	return accumulator.Equal(edwards25519.NewIdentityPoint()) == 1
}
