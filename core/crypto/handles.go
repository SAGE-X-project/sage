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

// Package crypto holds the key handles and value types of the SAGE core
// (spec 0.9.0-draft.1.1, 01-crypto.md).
//
// The core never holds a long-term private key. The host passes handles:
// a signer that signs bytes the core builds, and a KEM key that performs
// X25519 for the HPKE responder. No exported function or method of the core
// returns a private key.
package crypto

import (
	"crypto/ecdh"
	"errors"
	"fmt"
	"io"

	"github.com/sage-x-project/sage/core/secret"
)

// Purpose names what a signature is for. The core passes it to the host's
// signer so a key store can refuse or audit a request; it is not mixed into
// the signed bytes except where the spec defines a label (08-transport.md).
type Purpose uint8

// Signature purposes.
const (
	PurposeUnknown      Purpose = iota
	PurposeHTTPRequest          // 03: an HTTP request signature
	PurposeHTTPResponse         // 03: an HTTP response signature
	PurposeWireRequest          // 08: a WireMessage envelope
	PurposeWireResponse         // 08: a WireResponse envelope
	PurposeHPKEResponse         // 04: the responder's signature over the response
	PurposeCardProof            // 07: an agent card proof
	PurposePoP                  // 06: a registry proof of possession

	purposeEnd // sentinel, not a purpose
)

// String returns a stable name of the purpose.
func (p Purpose) String() string {
	switch p {
	case PurposeHTTPRequest:
		return "http_request"
	case PurposeHTTPResponse:
		return "http_response"
	case PurposeWireRequest:
		return "wire_request"
	case PurposeWireResponse:
		return "wire_response"
	case PurposeHPKEResponse:
		return "hpke_response"
	case PurposeCardProof:
		return "card_proof"
	case PurposePoP:
		return "pop"
	default:
		return "unknown"
	}
}

// KEMKey is the responder's static X25519 key (04-hpke.md). It is accepted
// by crypto/hpke.NewDHKEMPrivateKey. A host may back it with a hardware
// module; the core only calls these methods.
type KEMKey interface {
	ecdh.KeyExchanger
	// Public returns the 32-byte X25519 public key.
	Public() [32]byte
}

// ErrKEMKey is returned for a KEM key that is not a valid X25519 key.
var ErrKEMKey = errors.New("crypto: invalid X25519 KEM key")

// NewX25519KEMKey returns an in-memory KEM key for tests and simple hosts.
// The 32 bytes of sk are the X25519 scalar. The returned key has no method
// that yields the private key, and fmt prints it as redacted text.
// crypto/ecdh keeps its own copy of the scalar, which sk.Destroy does not
// erase and which lives until the key is no longer referenced. Reflection
// dumpers that ignore methods (go-spew with DisableMethods, as testify
// diffs use) can print that copy; do not pass the key to them. ECDH returns
// the shared secret as a plain slice, as crypto/ecdh.KeyExchanger requires.
func NewX25519KEMKey(sk secret.Bytes) (KEMKey, error) {
	var priv *ecdh.PrivateKey
	err := sk.Use(func(b []byte) error {
		if len(b) != 32 {
			return ErrKEMKey
		}
		k, err := ecdh.X25519().NewPrivateKey(b)
		if err != nil {
			return ErrKEMKey
		}
		priv = k
		return nil
	})
	if err != nil {
		return nil, err
	}
	var pub [32]byte
	copy(pub[:], priv.PublicKey().Bytes())
	return &x25519Key{priv: priv, pub: pub}, nil
}

type x25519Key struct {
	priv *ecdh.PrivateKey
	pub  [32]byte
}

func (k *x25519Key) Public() [32]byte { return k.pub }

func (k *x25519Key) PublicKey() *ecdh.PublicKey { return k.priv.PublicKey() }

func (k *x25519Key) Curve() ecdh.Curve { return ecdh.X25519() }

// ECDH computes the shared secret with remote. crypto/ecdh rejects a remote
// key that yields the all-zero secret (a low-order point).
func (k *x25519Key) ECDH(remote *ecdh.PublicKey) ([]byte, error) {
	if remote == nil || remote.Curve() != ecdh.X25519() {
		return nil, ErrKEMKey
	}
	return k.priv.ECDH(remote)
}

const redactedKEMKey = "x25519 KEM key " + secret.Redacted

// String, GoString and Format keep the private key out of every fmt verb,
// including %d and %x, which would otherwise print the struct fields.
func (k *x25519Key) String() string   { return redactedKEMKey }
func (k *x25519Key) GoString() string { return redactedKEMKey }

func (k *x25519Key) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, redactedKEMKey) }
