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
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"math/big"

	"filippo.io/edwards25519"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"

	"github.com/sage-x-project/sage/core/obs"
)

// Signature is a signature in the one encoding its algorithm accepts. Only
// ParseSignature and the signers of this package make one.
type Signature struct {
	alg Alg
	b   []byte
}

// Alg returns the algorithm the signature was parsed for.
func (s Signature) Alg() Alg { return s.alg }

// Bytes returns a copy of the encoding.
func (s Signature) Bytes() []byte { return append([]byte(nil), s.b...) }

// ParseSignature accepts exactly one encoding per algorithm and never
// normalises (01-crypto.md §2, E01-04 to E01-06):
//   - Ed25519: 64 bytes R || S with R canonical and S < L;
//   - es256k: 65 bytes r || s || v with 1 <= r, s < n, s <= n/2 and v in
//     {0, 1};
//   - P-256: 64 bytes r || s with 1 <= r, s < n and s <= n/2.
//
// Any other length, including DER, is malformed, and so is r or s equal to
// zero. An unreduced scalar (S >= L, r or s >= n), a high s and v outside
// {0, 1} are non-canonical: each is another encoding of a value that has a
// canonical one.
func ParseSignature(alg Alg, b []byte) (Signature, error) {
	var err error
	switch alg {
	case AlgEd25519:
		err = checkEd25519Signature(b)
	case AlgES256K:
		err = checkECDSASignature(b, 65, secp256k1Order())
		if err == nil && b[64] > 1 {
			err = reject(obs.ReasonNonCanonical, "es256k recovery id is not 0 or 1")
		}
	case AlgP256:
		err = checkECDSASignature(b, 64, elliptic.P256().Params().N)
	default:
		err = reject(obs.ReasonAlgorithm, "algorithm not allowed")
	}
	if err != nil {
		return Signature{}, err
	}
	return Signature{alg: alg, b: append([]byte(nil), b...)}, nil
}

// secp256k1Order returns the group order n of secp256k1 (SEC 2 §2.4.1).
func secp256k1Order() *big.Int {
	n, _ := new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	return n
}

func checkEd25519Signature(b []byte) error {
	if len(b) != ed25519.SignatureSize {
		return reject(obs.ReasonMalformed, "ed25519 signature is not 64 bytes")
	}
	if _, err := decodeEd25519Point(b[:32]); err != nil {
		return err
	}
	if _, err := new(edwards25519.Scalar).SetCanonicalBytes(b[32:]); err != nil {
		return reject(obs.ReasonNonCanonical, "ed25519 signature S is not reduced")
	}
	return nil
}

func checkECDSASignature(b []byte, size int, n *big.Int) error {
	if len(b) != size {
		return reject(obs.ReasonMalformed, "ecdsa signature has the wrong length")
	}
	r, s := new(big.Int).SetBytes(b[:32]), new(big.Int).SetBytes(b[32:64])
	if r.Sign() == 0 || s.Sign() == 0 {
		return reject(obs.ReasonMalformed, "ecdsa signature value is zero")
	}
	if r.Cmp(n) >= 0 || s.Cmp(n) >= 0 {
		return reject(obs.ReasonNonCanonical, "ecdsa signature value is not reduced")
	}
	if s.Cmp(new(big.Int).Rsh(n, 1)) > 0 {
		return reject(obs.ReasonNonCanonical, "ecdsa signature s is not low")
	}
	return nil
}

// Verify checks sig over msg with pub. The algorithm is the key's; a
// signature parsed for another algorithm is rejected without trying it
// (E01-11). es256k signatures must also recover to pub with their v. A nil
// key, or a key type from outside this package, is rejected like an unknown
// agent (DP-5), not with an error of its own.
func Verify(pub PublicKey, sig Signature, msg []byte) error {
	if _, ok := canonicalBytes(pub); !ok {
		return reject(obs.ReasonKeyNotEligible, "no public key of this package")
	}
	if sig.alg != pub.Alg() {
		return reject(obs.ReasonAlgorithm, "signature algorithm differs from the key")
	}
	var ok bool
	switch k := pub.(type) {
	case *ed25519Key:
		// crypto/ed25519 uses the cofactorless equation and requires S < L.
		ok = ed25519.Verify(k.b[:], msg, sig.b)
	case *secp256k1Key:
		ok = verifyES256K(k, sig.b, msg)
	case *p256Key:
		digest := sha256.Sum256(msg)
		r, s := new(big.Int).SetBytes(sig.b[:32]), new(big.Int).SetBytes(sig.b[32:])
		ok = ecdsa.Verify(k.pub, digest[:], r, s)
	}
	if !ok {
		return reject(obs.ReasonSignature, "signature does not verify")
	}
	return nil
}

func verifyES256K(k *secp256k1Key, sig, msg []byte) bool {
	return verifyES256KDigest(k, sig, keccak256(msg))
}

// verifyES256KDigest checks a 65-byte r || s || v over a 32-byte digest.
func verifyES256KDigest(k *secp256k1Key, sig, digest []byte) bool {
	// RecoverCompact takes 27 + v, then r || s, for an uncompressed key.
	compact := make([]byte, 65)
	compact[0] = 27 + sig[64]
	copy(compact[1:], sig[:64])
	recovered, _, err := secpecdsa.RecoverCompact(compact, digest)
	if err != nil || !recovered.IsEqual(k.pub) {
		return false
	}
	var r, s secp256k1.ModNScalar
	r.SetByteSlice(sig[:32])
	s.SetByteSlice(sig[32:64])
	return secpecdsa.NewSignature(&r, &s).Verify(digest, k.pub)
}
