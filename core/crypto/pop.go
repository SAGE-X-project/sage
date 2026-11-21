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

package crypto

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/sage-x-project/sage/core/obs"
)

// popPrefix starts every proof-of-possession challenge (06-did-sage.md §4).
const popPrefix = "SAGE-PoP:"

// ErrPoP is returned by SignPoP for a signer whose key is not keyData, a
// P-256 signer, or a signer whose output is not a valid proof.
var ErrPoP = errors.New("crypto: cannot make a proof of possession")

// PoPChallenge returns the challenge of 06-did-sage.md §4:
// "SAGE-PoP:" || did || ":" || lower-case hex(keyData). did is the record's
// did member exactly as stored, and keyData the key exactly as stored, not
// a canonical form (E01-03, E06-10).
func PoPChallenge(did string, keyData []byte) []byte {
	out := make([]byte, 0, len(popPrefix)+len(did)+1+2*len(keyData))
	out = append(out, popPrefix...)
	out = append(out, did...)
	out = append(out, ':')
	return hex.AppendEncode(out, keyData)
}

// VerifyPoP checks proof as the proof of possession of 06-did-sage.md §4
// for the key pub, stored as keyData, and the DID did. It is a separate
// algorithm (E01-07): the signed message is SHA-256(challenge), and for
// secp256k1 that digest is signed directly, without Keccak. It is valid only
// as a proof of possession: did.Evaluate is its only caller outside this
// package (SignPoP checks its own output with it), and no other path of the
// core accepts a proof or verifies a signature this way. An Ed25519
// proof is, by the spec's construction, also an Ed25519 signature over the
// 32-byte digest as a message; no message the core verifies with Verify is
// such a digest, since every signed input there is structured text or JSON.
//
// keyData must decode to pub, so the proof covers the bytes the registry
// stores. The proof is parsed strictly (E01-04, E01-05): 64 bytes for
// Ed25519, 65 bytes r || s || v, low-S and v in {0, 1}, for secp256k1, whose
// recovered key must equal pub. A P-256 key has no proof of possession.
func VerifyPoP(pub PublicKey, keyData []byte, did string, proof []byte) error {
	if _, ok := canonicalBytes(pub); !ok {
		return reject(obs.ReasonKeyNotEligible, "no public key of this package")
	}
	if pub.Alg() != AlgEd25519 && pub.Alg() != AlgES256K {
		return reject(obs.ReasonAlgorithm, "no proof of possession for this key type")
	}
	stored, err := ParsePublicKey(pub.Alg(), keyData)
	if err != nil || !stored.Equal(pub) {
		return reject(obs.ReasonKeyNotEligible, "key data is not the key")
	}
	sig, err := ParseSignature(pub.Alg(), proof)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(PoPChallenge(did, keyData))
	var ok bool
	switch k := pub.(type) {
	case *ed25519Key:
		ok = ed25519.Verify(k.b[:], digest[:], sig.b)
	case *secp256k1Key:
		ok = verifyES256KDigest(k, sig.b, digest[:])
	}
	if !ok {
		return reject(obs.ReasonSignature, "proof of possession does not verify")
	}
	return nil
}

// SignPoP makes the proof of possession of 06-did-sage.md §4 for did with
// s, whose key must be keyData. It passes the challenge to s.Sign with
// PurposePoP, which signs SHA-256(challenge), and checks the result with
// VerifyPoP, so a host signer that hashes under other rules or returns
// another encoding fails.
func SignPoP(ctx context.Context, s Signer, did string, keyData []byte) (Signature, error) {
	if s == nil {
		return Signature{}, ErrPoP
	}
	pub := s.Public()
	if _, ok := canonicalBytes(pub); !ok || (pub.Alg() != AlgEd25519 && pub.Alg() != AlgES256K) {
		return Signature{}, ErrPoP
	}
	stored, err := ParsePublicKey(pub.Alg(), keyData)
	if err != nil || !stored.Equal(pub) {
		return Signature{}, ErrPoP
	}
	sig, err := s.Sign(ctx, PurposePoP, PoPChallenge(did, keyData))
	if err != nil {
		return Signature{}, err
	}
	if err := VerifyPoP(pub, keyData, did, sig.b); err != nil {
		return Signature{}, errors.Join(ErrPoP, err)
	}
	return sig, nil
}
