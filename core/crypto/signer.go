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
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"

	"github.com/sage-x-project/sage/core/secret"
)

// Signer signs bytes that the core builds. The core never holds a long-term
// private key: the host passes a Signer, which may be backed by a key store
// or a remote KMS. A host implementation returns its signature through
// ParseSignature, so it must produce the one accepted encoding.
//
// The signer's DID is not part of this interface; it belongs to the
// identity layer (core/did).
type Signer interface {
	// Public returns the key that verifies the signer's signatures.
	Public() PublicKey
	// Sign signs msg for purpose p. The purpose lets a key store refuse or
	// audit a request; it is not mixed into msg. For PurposePoP, msg is the
	// 06-did-sage.md §4 challenge, which starts with "SAGE-PoP:"; the
	// signer computes SHA-256(challenge) itself and signs that digest under
	// the PoP rules (E01-07): an es256k signer signs it directly instead of
	// a Keccak-256, an Ed25519 signer signs the 32 bytes as the message, and
	// a P-256 signer refuses. Because the signer hashes, a PoP request
	// cannot obtain a signature over a digest of the caller's choosing.
	// SignPoP is the caller for that purpose.
	Sign(ctx context.Context, p Purpose, msg []byte) (Signature, error)
}

// ErrPurpose is returned by the signers of this package for PurposeUnknown,
// a value outside the defined purposes, PurposePoP with a message that is
// not a challenge ("SAGE-PoP:" prefix), and PurposePoP on a P-256 signer
// (06-did-sage.md §4 defines proofs for Ed25519 and secp256k1 only).
var ErrPurpose = errors.New("crypto: signature purpose not supported")

// ErrSigningKey is returned for private key bytes that are not a valid key.
var ErrSigningKey = errors.New("crypto: invalid signing key")

// The in-memory signers below are for tests and simple hosts. They keep the
// private key in memory that the Go runtime may copy and that the core
// cannot erase; they have no method that returns it, and fmt prints them as
// redacted text. The key sits behind a pointer, so fmt prints only an
// address even when it cannot call the methods (a signer in an unexported
// field). Reflection dumpers that follow pointers and ignore methods
// (go-spew with DisableMethods, as testify diffs use) can still print it;
// do not pass a signer to them.

// NewEd25519Signer returns a signer for the 32-byte RFC 8032 seed in seed.
func NewEd25519Signer(seed secret.Bytes) (Signer, error) {
	var s *ed25519Signer
	err := seed.Use(func(b []byte) error {
		if len(b) != ed25519.SeedSize {
			return ErrSigningKey
		}
		priv := ed25519.NewKeyFromSeed(b)
		pub := &ed25519Key{}
		copy(pub.b[:], priv[32:])
		s = &ed25519Signer{priv: &priv, pub: pub}
		return nil
	})
	return orNil(s, err)
}

type ed25519Signer struct {
	priv *ed25519.PrivateKey // a pointer, so fmt never prints the bytes
	pub  *ed25519Key
	redacted
}

func (s *ed25519Signer) Public() PublicKey { return s.pub }

func (s *ed25519Signer) Sign(ctx context.Context, p Purpose, msg []byte) (Signature, error) {
	if err := ready(ctx, p, msg); err != nil {
		return Signature{}, err
	}
	if p == PurposePoP {
		digest := sha256.Sum256(msg)
		msg = digest[:]
	}
	return ParseSignature(AlgEd25519, ed25519.Sign(*s.priv, msg))
}

// NewSecp256k1Signer returns an es256k signer for the 32-byte big-endian
// scalar in scalar, which must be in [1, n-1]. Signatures use RFC 6979
// nonces and are low-S (01-crypto.md §2).
func NewSecp256k1Signer(scalar secret.Bytes) (Signer, error) {
	var s *secp256k1Signer
	err := scalar.Use(func(b []byte) error {
		if len(b) != 32 {
			return ErrSigningKey
		}
		var d secp256k1.ModNScalar
		if overflow := d.SetByteSlice(b); overflow || d.IsZero() {
			return ErrSigningKey
		}
		priv := secp256k1.NewPrivateKey(&d)
		d.Zero()
		pub := &secp256k1Key{pub: priv.PubKey()}
		copy(pub.b[:], pub.pub.SerializeUncompressed())
		s = &secp256k1Signer{priv: priv, pub: pub}
		return nil
	})
	return orNil(s, err)
}

type secp256k1Signer struct {
	priv *secp256k1.PrivateKey
	pub  *secp256k1Key
	redacted
}

func (s *secp256k1Signer) Public() PublicKey { return s.pub }

func (s *secp256k1Signer) Sign(ctx context.Context, p Purpose, msg []byte) (Signature, error) {
	if err := ready(ctx, p, msg); err != nil {
		return Signature{}, err
	}
	digest := keccak256(msg)
	if p == PurposePoP {
		sum := sha256.Sum256(msg)
		digest = sum[:]
	}
	// SignCompact returns 27 + v, then r || s.
	compact := secpecdsa.SignCompact(s.priv, digest, false)
	out := make([]byte, 65)
	copy(out, compact[1:])
	out[64] = compact[0] - 27
	return ParseSignature(AlgES256K, out)
}

// NewP256Signer returns a P-256 signer for the 32-byte big-endian scalar in
// scalar, which must be in [1, n-1]. Signing is randomised, which
// 01-crypto.md §2 permits, and the signature is normalised to low-S.
func NewP256Signer(scalar secret.Bytes) (Signer, error) {
	var s *p256Signer
	err := scalar.Use(func(b []byte) error {
		priv, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), b)
		if err != nil {
			return ErrSigningKey
		}
		pubBytes, err := priv.PublicKey.Bytes()
		if err != nil {
			return ErrSigningKey
		}
		pub := &p256Key{pub: &priv.PublicKey}
		copy(pub.b[:], pubBytes)
		s = &p256Signer{priv: priv, pub: pub}
		return nil
	})
	return orNil(s, err)
}

type p256Signer struct {
	priv *ecdsa.PrivateKey
	pub  *p256Key
	redacted
}

func (s *p256Signer) Public() PublicKey { return s.pub }

func (s *p256Signer) Sign(ctx context.Context, p Purpose, msg []byte) (Signature, error) {
	if p == PurposePoP {
		return Signature{}, ErrPurpose
	}
	if err := ready(ctx, p, msg); err != nil {
		return Signature{}, err
	}
	digest := sha256.Sum256(msg)
	r, sv, err := ecdsa.Sign(rand.Reader, s.priv, digest[:])
	if err != nil {
		return Signature{}, err
	}
	n := elliptic.P256().Params().N
	if sv.Cmp(new(big.Int).Rsh(n, 1)) > 0 {
		sv.Sub(n, sv)
	}
	out := make([]byte, 64)
	r.FillBytes(out[:32])
	sv.FillBytes(out[32:])
	return ParseSignature(AlgP256, out)
}

func ready(ctx context.Context, p Purpose, msg []byte) error {
	if p == PurposeUnknown || p >= purposeEnd || (p == PurposePoP && !bytes.HasPrefix(msg, []byte(popPrefix))) {
		return ErrPurpose
	}
	return ctx.Err()
}

// orNil keeps a typed nil pointer out of the Signer interface.
func orNil[T Signer](s T, err error) (Signer, error) {
	if err != nil {
		return nil, err
	}
	return s, nil
}

const redactedSigner = "signer " + secret.Redacted

// redacted keeps the private key out of every fmt verb, including %d and %x,
// which would otherwise print the struct fields.
type redacted struct{}

func (redacted) String() string             { return redactedSigner }
func (redacted) GoString() string           { return redactedSigner }
func (redacted) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, redactedSigner) }
