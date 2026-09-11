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
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/hex"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
)

// secp256k1KeyPair implements the KeyPair interface for Secp256k1 keys
type secp256k1KeyPair struct {
	privateKey *secp256k1.PrivateKey
	publicKey  *secp256k1.PublicKey
	id         string
}

// GenerateSecp256k1KeyPair generates a new Secp256k1 key pair
func GenerateSecp256k1KeyPair() (sagecrypto.KeyPair, error) {
	privateKey, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}

	publicKey := privateKey.PubKey()

	// Generate ID from public key hash
	pubKeyBytes := publicKey.SerializeCompressed()
	hash := sha256.Sum256(pubKeyBytes)
	id := hex.EncodeToString(hash[:8])

	return &secp256k1KeyPair{
		privateKey: privateKey,
		publicKey:  publicKey,
		id:         id,
	}, nil
}

// PublicKey returns the public key
func (kp *secp256k1KeyPair) PublicKey() crypto.PublicKey {
	return kp.publicKey.ToECDSA()
}

// PrivateKey returns the private key
func (kp *secp256k1KeyPair) PrivateKey() crypto.PrivateKey {
	return kp.privateKey.ToECDSA()
}

// Type returns the key type
func (kp *secp256k1KeyPair) Type() sagecrypto.KeyType {
	return sagecrypto.KeyTypeSecp256k1
}

// Sign signs the message with the Ethereum convention shared by every SAGE
// path (Keccak-256, RFC 6979, r || s || v). See SignSecp256k1Keccak.
func (kp *secp256k1KeyPair) Sign(message []byte) ([]byte, error) {
	return SignSecp256k1Keccak(kp.privateKey.ToECDSA(), message)
}

// Verify verifies a 64- or 65-byte Ethereum-style signature over the message.
func (kp *secp256k1KeyPair) Verify(message, signature []byte) error {
	return VerifySecp256k1Keccak(kp.publicKey.ToECDSA(), message, signature)
}

// ID returns a unique identifier for this key pair
func (kp *secp256k1KeyPair) ID() string {
	return kp.id
}

// Secp256k1Curve returns the secp256k1 curve as an elliptic.Curve so that
// crypto/ecdsa key structs can be built from decred key material.
// decred v4.4.1 deprecates S256 in favour of its specialised API, but
// crypto/ecdsa still requires an elliptic.Curve value; keep the single
// suppression here so every caller shares one source.
func Secp256k1Curve() elliptic.Curve {
	return secp256k1.S256() //nolint:staticcheck // SA1019: crypto/ecdsa needs elliptic.Curve
}
