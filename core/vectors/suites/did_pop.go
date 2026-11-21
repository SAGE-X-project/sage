// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package suites

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"golang.org/x/crypto/sha3"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/clock"
	"github.com/sage-x-project/sage/core/crypto"
	"github.com/sage-x-project/sage/core/did"
	"github.com/sage-x-project/sage/core/obs"
	"github.com/sage-x-project/sage/core/secret"
	"github.com/sage-x-project/sage/core/vectors"
)

const (
	popDID          = "did:sage:ethereum:0x1111111111111111111111111111111111111111"
	popOtherDID     = "did:sage:ethereum:0x2222222222222222222222222222222222222222"
	labelSecp256k1B = "sage-spec/vectors/secp256k1/agent-b"

	// The deployed registry's ECDSA "ownership" value (sage-contracts
	// v1/ethereum/contracts/AgentCardRegistry.sol:424-438): an EIP-191
	// signature by the owner over Keccak-256("SAGE Agent Registration:" ||
	// chainid || registry || owner). Sepolia and a fixed registry address.
	registrationChainID  = 11155111
	registrationRegistry = "0x3333333333333333333333333333333333333333"

	// evaluationTime is the clock of the record fixtures; each record is
	// observed at that instant, so freshness never decides them.
	evaluationTime = 1788609600
)

// popMaterial holds the keys and proofs the PoP and selection cases use.
type popMaterial struct {
	edA, edB, k1A, k1B []byte // key_data as stored
	edProof, k1Proof    []byte // valid §4 proofs for popDID
	k1BProof            []byte
	k1OtherDID          []byte // k1A's proof for popOtherDID
	edWrongKey          []byte // edB's signature over edA's digest
	k1HighS             []byte // k1Proof with s := n - s and v flipped
	edRawChallenge      []byte // edA's signature over the challenge itself
	k1Registration      []byte // the deployed registry's EIP-191 value, owner = k1A, v in {27, 28}
	k1RegistrationV01   []byte // the same with v - 27, so it passes the E01-05 v check
}

func newPoPMaterial() (*popMaterial, error) {
	ctx := context.Background()
	m := &popMaterial{}
	edA, err := crypto.NewEd25519Signer(secret.From(labelSeed(labelEd25519A)))
	if err != nil {
		return nil, err
	}
	edB, err := crypto.NewEd25519Signer(secret.From(labelSeed(labelEd25519B)))
	if err != nil {
		return nil, err
	}
	k1A, err := crypto.NewSecp256k1Signer(secret.From(labelSeed(labelSecp256k1A)))
	if err != nil {
		return nil, err
	}
	k1B, err := crypto.NewSecp256k1Signer(secret.From(labelSeed(labelSecp256k1B)))
	if err != nil {
		return nil, err
	}
	m.edA, m.edB, m.k1A, m.k1B = edA.Public().Bytes(), edB.Public().Bytes(), k1A.Public().Bytes(), k1B.Public().Bytes()
	for _, p := range []struct {
		dst  *[]byte
		s    crypto.Signer
		did  string
		data []byte
	}{
		{&m.edProof, edA, popDID, m.edA},
		{&m.k1Proof, k1A, popDID, m.k1A},
		{&m.k1BProof, k1B, popDID, m.k1B},
		{&m.k1OtherDID, k1A, popOtherDID, m.k1A},
	} {
		sig, err := crypto.SignPoP(ctx, p.s, p.did, p.data)
		if err != nil {
			return nil, err
		}
		*p.dst = sig.Bytes()
	}
	wrong, err := edB.Sign(ctx, crypto.PurposePoP, crypto.PoPChallenge(popDID, m.edA))
	if err != nil {
		return nil, err
	}
	m.edWrongKey = wrong.Bytes()
	raw, err := edA.Sign(ctx, crypto.PurposeHTTPRequest, crypto.PoPChallenge(popDID, m.edA))
	if err != nil {
		return nil, err
	}
	m.edRawChallenge = raw.Bytes()
	m.k1HighS = highS(m.k1Proof)
	m.k1Registration, err = registrationSignature(labelSeed(labelSecp256k1A), k1A.Public())
	if err != nil {
		return nil, err
	}
	m.k1RegistrationV01 = append([]byte(nil), m.k1Registration...)
	m.k1RegistrationV01[64] -= 27
	return m, nil
}

// highS returns the other encoding of an r || s || v signature: s := n - s
// with the recovery id flipped, which recovers the same key.
func highS(sig []byte) []byte {
	n, _ := new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	s := new(big.Int).Sub(n, new(big.Int).SetBytes(sig[32:64]))
	out := append([]byte(nil), sig...)
	s.FillBytes(out[32:64])
	out[64] ^= 1
	return out
}

// registrationSignature returns what the deployed registry stores as a
// type-0 key's signature: the owner's EIP-191 signature, v in {27, 28}, over
// the registration message, which does not involve the listed key.
func registrationSignature(scalar []byte, owner crypto.PublicKey) ([]byte, error) {
	addr, err := crypto.EthereumAddress(owner)
	if err != nil {
		return nil, err
	}
	ownerBytes, err := hex.DecodeString(addr[2:])
	if err != nil {
		return nil, err
	}
	registry, err := hex.DecodeString(registrationRegistry[2:])
	if err != nil {
		return nil, err
	}
	chain := make([]byte, 32)
	big.NewInt(registrationChainID).FillBytes(chain)
	msg := append([]byte("SAGE Agent Registration:"), chain...)
	msg = append(append(msg, registry...), ownerBytes...)
	inner := keccak(msg)
	hash := keccak(append([]byte("\x19Ethereum Signed Message:\n32"), inner...))
	var d secp256k1.ModNScalar
	if overflow := d.SetByteSlice(scalar); overflow || d.IsZero() {
		return nil, errors.New("invalid owner scalar")
	}
	compact := secpecdsa.SignCompact(secp256k1.NewPrivateKey(&d), hash, false)
	out := make([]byte, 65)
	copy(out, compact[1:])
	out[64] = compact[0] // 27 + v, as wallets return it
	return out, nil
}

func keccak(b []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(b)
	return h.Sum(nil)
}

// key is one keys[] entry of a record fixture.
type key struct {
	typ       did.KeyType
	data, sig []byte
	verified  bool
}

func record(active bool, keys ...key) map[string]any {
	list := make([]any, len(keys))
	for i, k := range keys {
		list[i] = map[string]any{
			"type": int(k.typ), "key_data": hex.EncodeToString(k.data),
			"signature": hex.EncodeToString(k.sig), "verified": k.verified,
		}
	}
	return map[string]any{"record": map[string]any{"did": popDID, "is_active": active, "keys": list}}
}

// didPoP returns the published proof-of-possession vectors and the errata
// §11 PoP and key-selection cases of 06 (E06-10, E06-11, §3). Every record
// case runs did.Evaluate at a fixed clock with the record observed at that
// instant.
func didPoP() ([]vectors.Case, error) {
	m, err := newPoPMaterial()
	if err != nil {
		return nil, err
	}
	const (
		evaluated = "did.Evaluate, observed at the evaluation time"
		e10       = "E06-10: the proof is the §4 signature over SHA-256(challenge) for the record's did and key_data as stored; "
		e11       = "E06-11: "
	)
	on := true // the registry's verified flag
	cases := []vectors.Case{
		{
			Name: "pop-ed25519", Mode: vectors.ModeDeterministic,
			Description: "Key proof-of-possession: Ed25519 signature over SHA-256 of the challenge string.",
			Input:       map[string]any{"did": popDID, "seed_label": labelEd25519A},
			Produce:     producePoP(crypto.AlgEd25519), Verify: verifyPoP(did.KeyTypeEd25519),
		},
		{
			Name: "pop-secp256k1", Mode: vectors.ModeDeterministic,
			Description: "Key proof-of-possession: secp256k1 RFC 6979 signature (65 bytes r||s||v) over SHA-256 of the challenge string. Note: SHA-256, not Keccak, for this path.",
			Input:       map[string]any{"did": popDID, "scalar_label": labelSecp256k1A},
			Produce:     producePoP(crypto.AlgES256K), Verify: verifyPoP(did.KeyTypeECDSA),
		},
		{
			Name: "select-secp256k1-first", Mode: vectors.ModeDeterministic,
			Description: e11 + "an Ed25519 key listed before a secp256k1 key, both with valid proofs: the secp256k1 key is the signing key (" + evaluated + ").",
			Input:       record(true, key{did.KeyTypeEd25519, m.edA, m.edProof, on}, key{did.KeyTypeECDSA, m.k1A, m.k1Proof, on}),
			Produce:     produceSelection,
		},
		{
			Name: "select-skips-bad-pop", Mode: vectors.ModeDeterministic,
			Description: e11 + "the first secp256k1 entry carries the deployed registry's EIP-191 registration signature by the same key (no §4 proof; its v is given as 0 or 1, so only the proof check rejects it), the second a valid proof: the second is the signing key (" + evaluated + ").",
			Input:       record(true, key{did.KeyTypeECDSA, m.k1A, m.k1RegistrationV01, on}, key{did.KeyTypeECDSA, m.k1B, m.k1BProof, on}),
			Produce:     produceSelection,
		},
		{
			Name: "select-ignores-flag", Mode: vectors.ModeDeterministic,
			Description: e11 + "an entry with verified = true and no valid proof (the registry's EIP-191 value with v 27 or 28, as stored) is not eligible; a later entry with a valid proof and verified = false is the signing key (" + evaluated + ").",
			Input:       record(true, key{did.KeyTypeECDSA, m.k1A, m.k1Registration, true}, key{did.KeyTypeECDSA, m.k1B, m.k1BProof, false}),
			Produce:     produceSelection,
		},
	}
	malformed := append([]byte{0x04}, make([]byte, 39)...)
	for _, r := range []struct {
		name, desc string
		in         map[string]any
		reason     obs.Reason
	}{
		{"pop-wrong-did", e10 + "a proof made for another DID", record(true, key{did.KeyTypeECDSA, m.k1A, m.k1OtherDID, on}), obs.ReasonKeyNotEligible},
		{"pop-wrong-key", e10 + "another key's signature over this key's digest", record(true, key{did.KeyTypeEd25519, m.edA, m.edWrongKey, on}), obs.ReasonKeyNotEligible},
		{"pop-secp256k1-high-s", e10 + "a valid secp256k1 proof re-encoded with high S (E01-05)", record(true, key{did.KeyTypeECDSA, m.k1A, m.k1HighS, on}), obs.ReasonKeyNotEligible},
		{"pop-signed-raw-challenge", e10 + "an Ed25519 signature over the challenge itself instead of its SHA-256 (E01-07)", record(true, key{did.KeyTypeEd25519, m.edA, m.edRawChallenge, on}), obs.ReasonKeyNotEligible},
		{"select-no-eligible-key", e11 + "every signing entry lacks a valid proof, so the record has no signing key", record(true, key{did.KeyTypeECDSA, m.k1A, m.k1RegistrationV01, on}, key{did.KeyTypeEd25519, m.edA, m.edRawChallenge, on}), obs.ReasonKeyNotEligible},
		{"select-malformed-fails", e11 + "the first secp256k1 entry's key_data (40 bytes) does not decode; resolution fails and does not fall back to the second entry, which has a valid proof", record(true, key{did.KeyTypeECDSA, malformed, m.k1Proof, on}, key{did.KeyTypeECDSA, m.k1B, m.k1BProof, on}), obs.ReasonResolution},
		{"inactive-agent", "06 §3: an inactive agent (is_active false) has no usable keys, although its key has a valid proof", record(false, key{did.KeyTypeECDSA, m.k1A, m.k1Proof, on}), obs.ReasonInactive},
	} {
		cases = append(cases, vectors.Case{
			Name: r.name, Mode: vectors.ModeReject,
			Description: r.desc + " (" + evaluated + ").",
			Input:       r.in, Reject: rejectRecord, Reason: r.reason,
		})
	}
	return cases, nil
}

func producePoP(alg crypto.Alg) func(map[string]any) (map[string]any, error) {
	return func(in map[string]any) (map[string]any, error) {
		d, ok := in["did"].(string)
		if !ok {
			return nil, errors.New("did must be a string")
		}
		var s crypto.Signer
		var err error
		if alg == crypto.AlgEd25519 {
			label, _ := in["seed_label"].(string)
			s, err = crypto.NewEd25519Signer(secret.From(labelSeed(label)))
		} else {
			label, _ := in["scalar_label"].(string)
			s, err = crypto.NewSecp256k1Signer(secret.From(labelSeed(label)))
		}
		if err != nil {
			return nil, err
		}
		data := s.Public().Bytes()
		proof, err := crypto.SignPoP(context.Background(), s, d, data)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"challenge": string(crypto.PoPChallenge(d, data)),
			"key_data":  hex.EncodeToString(data),
			"proof":     hex.EncodeToString(proof.Bytes()),
		}, nil
	}
}

// verifyPoP checks a stored proof through did.Evaluate: a record listing
// the key with that proof must select it.
func verifyPoP(typ did.KeyType) func(in, out map[string]any) error {
	return func(in, out map[string]any) error {
		d, ok := in["did"].(string)
		if !ok {
			return errors.New("did must be a string")
		}
		data, err := hexField(out, "key_data")
		if err != nil {
			return err
		}
		proof, err := hexField(out, "proof")
		if err != nil {
			return err
		}
		if string(crypto.PoPChallenge(d, data)) != out["challenge"] {
			return errors.New("challenge differs")
		}
		rec := record(true, key{typ, data, proof, true})
		rec["record"].(map[string]any)["did"] = d
		a, err := evaluateRecord(rec)
		if err != nil {
			return err
		}
		if string(a.SigningKey().PublicKey().Bytes()) != string(mustKey(typ, data)) {
			return errors.New("the proven key was not selected")
		}
		return nil
	}
}

func mustKey(typ did.KeyType, data []byte) []byte {
	alg := crypto.AlgEd25519
	if typ == did.KeyTypeECDSA {
		alg = crypto.AlgES256K
	}
	k, err := crypto.ParsePublicKey(alg, data)
	if err != nil {
		return nil
	}
	return k.Bytes()
}

func produceSelection(in map[string]any) (map[string]any, error) {
	a, err := evaluateRecord(in)
	if err != nil {
		return nil, err
	}
	k := a.SigningKey()
	return map[string]any{"signing_key_type": int(k.Type()), "signing_key": hex.EncodeToString(k.PublicKey().Bytes())}, nil
}

func rejectRecord(in map[string]any) error {
	_, err := evaluateRecord(in)
	return err
}

// evaluateRecord parses the record fixture of in and runs did.Evaluate at
// evaluationTime with the record observed at that instant. A fixture that
// does not parse is a plain error, never a Rejection.
func evaluateRecord(in map[string]any) (did.Agent, error) {
	rec, err := parseRecord(in)
	if err != nil {
		return did.Agent{}, err
	}
	s, err := core.NewSettings(core.WithSpecVersion(string(core.V0_9_0_Draft1_1)))
	if err != nil {
		return did.Agent{}, err
	}
	at := time.Unix(evaluationTime, 0)
	return did.Evaluate(context.Background(), rec.DID, rec, did.Observation{At: at, Source: "fixture"},
		did.Policy{Clock: clock.NewFixed(at), Settings: s})
}

func parseRecord(in map[string]any) (did.Record, error) {
	if len(in) != 1 {
		return did.Record{}, errors.New("input must hold exactly record")
	}
	r, ok := in["record"].(map[string]any)
	if !ok || len(r) != 3 {
		return did.Record{}, errors.New("record must hold exactly did, is_active and keys")
	}
	ds, ok := r["did"].(string)
	if !ok {
		return did.Record{}, errors.New("record did must be a string")
	}
	d, err := did.Parse(ds)
	if err != nil {
		return did.Record{}, errors.New("record did does not parse")
	}
	active, ok := r["is_active"].(bool)
	if !ok {
		return did.Record{}, errors.New("is_active must be a boolean")
	}
	list, ok := r["keys"].([]any)
	if !ok {
		return did.Record{}, errors.New("keys must be a list")
	}
	rec := did.Record{DID: d, IsActive: active}
	for i, v := range list {
		k, err := parseKeyEntry(v)
		if err != nil {
			return did.Record{}, fmt.Errorf("keys[%d]: %w", i, err)
		}
		rec.Keys = append(rec.Keys, k)
	}
	return rec, nil
}

func parseKeyEntry(v any) (did.Key, error) {
	m, ok := v.(map[string]any)
	if !ok || len(m) != 4 {
		return did.Key{}, errors.New("must hold exactly type, key_data, signature and verified")
	}
	t, err := integer(m["type"])
	if err != nil || t < 0 || t > 2 {
		return did.Key{}, errors.New("type must be 0, 1 or 2")
	}
	data, err := hexField(m, "key_data")
	if err != nil {
		return did.Key{}, err
	}
	sig, err := hexField(m, "signature")
	if err != nil {
		return did.Key{}, err
	}
	verified, ok := m["verified"].(bool)
	if !ok {
		return did.Key{}, errors.New("verified must be a boolean")
	}
	return did.Key{Type: did.KeyType(t), KeyData: data, Signature: sig, Verified: verified}, nil
}
