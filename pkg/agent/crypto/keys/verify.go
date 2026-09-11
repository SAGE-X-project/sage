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

package keys

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/asn1"
	"fmt"
	"math/big"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
)

// VerifySignature verifies signature over message with any public key type
// SAGE supports. It is the single signature-verification routine: the HPKE
// handshake, the DID proofs and the sage-crypto CLI all call it.
//
// Conventions (identical to KeyPair.Sign of the matching key type):
//   - Ed25519: pure Ed25519 over the message (RFC 8032).
//   - secp256k1: Keccak-256 of the message, signature r || s [|| v] (64 or 65
//     bytes) or DER, the Ethereum convention.
//   - P-256: SHA-256 of the message, signature raw r || s (64 bytes) or DER.
func VerifySignature(publicKey crypto.PublicKey, message, signature []byte) error {
	switch pub := publicKey.(type) {
	case ed25519.PublicKey:
		if !ed25519.Verify(pub, message, signature) {
			return fmt.Errorf("ed25519 signature verification failed: %w", sagecrypto.ErrInvalidSignature)
		}
		return nil
	case *ecdsa.PublicKey:
		if IsSecp256k1Curve(pub.Curve) {
			if raw, ok := derToRaw(pub.Curve, signature); ok {
				signature = raw
			}
			if err := VerifySecp256k1Keccak(pub, message, signature); err != nil {
				return fmt.Errorf("secp256k1 signature verification failed: %w", err)
			}
			return nil
		}
		if raw, ok := derToRaw(pub.Curve, signature); ok {
			signature = raw
		}
		r, s, err := deserializeP256Signature(signature)
		if err != nil {
			return fmt.Errorf("invalid ECDSA signature encoding: %w", sagecrypto.ErrInvalidSignature)
		}
		digest := sha256.Sum256(message)
		if !ecdsa.Verify(pub, digest[:], r, s) {
			return fmt.Errorf("ecdsa signature verification failed: %w", sagecrypto.ErrInvalidSignature)
		}
		return nil
	default:
		return fmt.Errorf("unsupported public key type %T: %w", publicKey, sagecrypto.ErrInvalidKeyType)
	}
}

// derToRaw converts a DER-encoded ECDSA signature to fixed-size r || s for
// the given curve. It returns false when sig is not DER.
func derToRaw(curve elliptic.Curve, sig []byte) ([]byte, bool) {
	var der struct{ R, S *big.Int }
	rest, err := asn1.Unmarshal(sig, &der)
	if err != nil || len(rest) != 0 || der.R == nil || der.S == nil {
		return nil, false
	}
	size := (curve.Params().BitSize + 7) / 8
	out := make([]byte, 2*size)
	der.R.FillBytes(out[:size])
	der.S.FillBytes(out[size:])
	return out, true
}
