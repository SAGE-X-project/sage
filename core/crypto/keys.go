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
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"filippo.io/edwards25519"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"golang.org/x/crypto/sha3"

	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/core/obs"
)

// Alg is a signature algorithm of 01-crypto.md §2. Each public key type has
// exactly one, and a verifier takes it from the key (E01-11).
type Alg uint8

// Signature algorithms. RSA is not part of the specification (E01-02).
const (
	AlgUnknown Alg = iota
	AlgEd25519     // pure Ed25519, 64-byte R || S
	AlgES256K      // secp256k1 over Keccak-256, 65-byte r || s || v
	AlgP256        // P-256 over SHA-256, 64-byte r || s
)

// String returns the RFC 9421 identifier of 01-crypto.md §3.
func (a Alg) String() string {
	switch a {
	case AlgEd25519:
		return "ed25519"
	case AlgES256K:
		return "es256k"
	case AlgP256:
		return "ecdsa-p256-sha256"
	default:
		return "unknown"
	}
}

// ParseAlg maps an RFC 9421 alg identifier to its algorithm. The comparison
// is exact. Every other identifier, including every RSA one, is rejected
// with reason algorithm.
func ParseAlg(s string) (Alg, error) {
	for _, a := range []Alg{AlgEd25519, AlgES256K, AlgP256} {
		if s == a.String() {
			return a, nil
		}
	}
	return AlgUnknown, reject(obs.ReasonAlgorithm, "algorithm not allowed")
}

// PublicKey is a checked public signing key. ParsePublicKey and the signers
// of this package make the only implementations, each of which has passed
// the checks of 01-crypto.md §1 and §2 (E01-04, E01-14). A type outside the
// package can satisfy the interface only by embedding one; Verify, KeyIDOf
// and EthereumAddress do not accept such a type. X25519 keys are not
// signing keys and are not a PublicKey (DP-3.2).
type PublicKey interface {
	// Alg returns the one algorithm the key verifies.
	Alg() Alg
	// Bytes returns a copy of the canonical encoding: 32 bytes for
	// Ed25519, 65-byte uncompressed SEC1 for secp256k1 and P-256.
	Bytes() []byte
	// Equal reports whether other is the same key.
	Equal(other PublicKey) bool

	sealed()
}

// ParsePublicKey decodes a public key of the type that alg verifies.
//   - Ed25519: 32 bytes, a canonical encoding (y < p) of a point that is
//     not of small order.
//   - secp256k1: 33-byte compressed (02, 03), 64-byte X || Y or 65-byte
//     uncompressed (04) SEC1 of a point on the curve; all three give the
//     same key, stored in the 65-byte form (E01-03).
//   - P-256: 65-byte uncompressed SEC1 of a point on the curve.
//
// A key that is not on its curve, of small order or of another length is
// rejected as malformed; a non-canonical Ed25519 encoding as non-canonical.
func ParsePublicKey(alg Alg, b []byte) (PublicKey, error) {
	switch alg {
	case AlgEd25519:
		return parseEd25519Key(b)
	case AlgES256K:
		return parseSecp256k1Key(b)
	case AlgP256:
		return parseP256Key(b)
	default:
		return nil, reject(obs.ReasonAlgorithm, "algorithm not allowed")
	}
}

type ed25519Key struct{ b [32]byte }

func (k *ed25519Key) Alg() Alg      { return AlgEd25519 }
func (k *ed25519Key) Bytes() []byte { return append([]byte(nil), k.b[:]...) }
func (k *ed25519Key) Equal(o PublicKey) bool {
	other, ok := o.(*ed25519Key)
	return ok && other != nil && other.b == k.b
}
func (k *ed25519Key) sealed() {}

func parseEd25519Key(b []byte) (PublicKey, error) {
	if len(b) != 32 {
		return nil, reject(obs.ReasonMalformed, "ed25519 public key is not 32 bytes")
	}
	p, err := decodeEd25519Point(b)
	if err != nil {
		return nil, err
	}
	if new(edwards25519.Point).MultByCofactor(p).Equal(edwards25519.NewIdentityPoint()) == 1 {
		return nil, reject(obs.ReasonMalformed, "ed25519 public key of small order")
	}
	k := &ed25519Key{}
	copy(k.b[:], b)
	return k, nil
}

// decodeEd25519Point decodes a point and requires the canonical encoding:
// edwards25519.SetBytes also accepts y >= p, which RFC 8032 §5.1.7 and
// E01-04 forbid.
func decodeEd25519Point(b []byte) (*edwards25519.Point, error) {
	p, err := new(edwards25519.Point).SetBytes(b)
	if err != nil {
		return nil, reject(obs.ReasonMalformed, "ed25519 point not on the curve")
	}
	if string(p.Bytes()) != string(b) {
		return nil, reject(obs.ReasonNonCanonical, "ed25519 point encoding is not canonical")
	}
	return p, nil
}

type secp256k1Key struct {
	b   [65]byte
	pub *secp256k1.PublicKey
}

func (k *secp256k1Key) Alg() Alg      { return AlgES256K }
func (k *secp256k1Key) Bytes() []byte { return append([]byte(nil), k.b[:]...) }
func (k *secp256k1Key) Equal(o PublicKey) bool {
	other, ok := o.(*secp256k1Key)
	return ok && other != nil && other.b == k.b
}
func (k *secp256k1Key) sealed() {}

func parseSecp256k1Key(b []byte) (PublicKey, error) {
	switch {
	case len(b) == 33 && (b[0] == 0x02 || b[0] == 0x03):
	case len(b) == 65 && b[0] == 0x04:
	case len(b) == 64:
		b = append([]byte{0x04}, b...)
	default:
		// ParsePubKey would also take the hybrid prefixes 06 and 07.
		return nil, reject(obs.ReasonMalformed, "secp256k1 public key encoding not allowed")
	}
	pub, err := secp256k1.ParsePubKey(b)
	if err != nil {
		return nil, reject(obs.ReasonMalformed, "secp256k1 point not on the curve")
	}
	k := &secp256k1Key{pub: pub}
	copy(k.b[:], pub.SerializeUncompressed())
	return k, nil
}

type p256Key struct {
	b   [65]byte
	pub *ecdsa.PublicKey
}

func (k *p256Key) Alg() Alg      { return AlgP256 }
func (k *p256Key) Bytes() []byte { return append([]byte(nil), k.b[:]...) }
func (k *p256Key) Equal(o PublicKey) bool {
	other, ok := o.(*p256Key)
	return ok && other != nil && other.b == k.b
}
func (k *p256Key) sealed() {}

func parseP256Key(b []byte) (PublicKey, error) {
	if len(b) != 65 || b[0] != 0x04 {
		return nil, reject(obs.ReasonMalformed, "p-256 public key is not 65-byte uncompressed")
	}
	// ParseUncompressedPublicKey rejects points off the curve and the point
	// at infinity.
	pub, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), b)
	if err != nil {
		return nil, reject(obs.ReasonMalformed, "p-256 point not on the curve")
	}
	k := &p256Key{pub: pub}
	copy(k.b[:], b)
	return k, nil
}

// X25519PublicKey is a key-agreement public key (HPKE, E2E). It is not a
// PublicKey: Verify cannot take it, so an X25519 key is never used to
// verify a signature (DP-3.2). The zero value holds no key.
type X25519PublicKey struct {
	b  [32]byte
	ok bool
}

// ParseX25519PublicKey accepts exactly 32 bytes (E01-14). Low-order peer
// keys are caught later by the all-zero shared-secret rule of 04-hpke.md
// §3, as E01-14 specifies.
func ParseX25519PublicKey(b []byte) (X25519PublicKey, error) {
	if len(b) != 32 {
		return X25519PublicKey{}, reject(obs.ReasonMalformed, "x25519 public key is not 32 bytes")
	}
	k := X25519PublicKey{ok: true}
	copy(k.b[:], b)
	return k, nil
}

// Bytes returns a copy of the 32 bytes, or nil for the zero value.
func (k X25519PublicKey) Bytes() []byte {
	if !k.ok {
		return nil
	}
	return append([]byte(nil), k.b[:]...)
}

// Equal reports whether both hold the same key.
func (k X25519PublicKey) Equal(o X25519PublicKey) bool { return k.ok && o.ok && k.b == o.b }

// KeyID returns the KeyID over the 32 bytes (E01-12), or the zero KeyID
// for the zero value.
func (k X25519PublicKey) KeyID() KeyID {
	if !k.ok {
		return KeyID{}
	}
	return keyIDOver(k.b[:])
}

// KeyID identifies a key in logs and diagnostics (01-crypto.md §4). It MUST
// NOT select a key, find a session or decide authorisation (E01-13), so no
// function of the core takes one.
type KeyID struct{ s string }

// String returns hex(SHA-256(canonical encoding)[0:8]).
func (k KeyID) String() string { return k.s }

// KeyIDOf returns the KeyID of pub over its canonical encoding (E01-12), or
// the zero KeyID for nil.
// A key type from outside this package also gives the zero KeyID, so a
// caller cannot choose the bytes it is computed over.
func KeyIDOf(pub PublicKey) KeyID {
	b, ok := canonicalBytes(pub)
	if !ok {
		return KeyID{}
	}
	return keyIDOver(b)
}

func keyIDOver(b []byte) KeyID {
	sum := sha256.Sum256(b)
	return KeyID{s: hex.EncodeToString(sum[:8])}
}

// canonicalBytes returns the encoding of a key of this package, and false
// for nil or any other implementation.
func canonicalBytes(pub PublicKey) ([]byte, bool) {
	switch k := pub.(type) {
	case *ed25519Key:
		if k != nil {
			return k.b[:], true
		}
	case *secp256k1Key:
		if k != nil {
			return k.b[:], true
		}
	case *p256Key:
		if k != nil {
			return k.b[:], true
		}
	}
	return nil, false
}

// EthereumAddress returns 0x || hex(Keccak-256(X || Y)[12:32]) in lower
// case for a secp256k1 key (01-crypto.md §4).
func EthereumAddress(pub PublicKey) (string, error) {
	k, ok := pub.(*secp256k1Key)
	if !ok || k == nil {
		return "", errors.New("crypto: an Ethereum address needs a secp256k1 key")
	}
	sum := keccak256(k.b[1:])
	return "0x" + hex.EncodeToString(sum[12:]), nil
}

func keccak256(b []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(b)
	return h.Sum(nil)
}

func reject(reason obs.Reason, detail string) error {
	return core.Reject(reason, obs.Known(detail))
}
