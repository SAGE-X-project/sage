// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


package keys

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"

	sagecrypto "github.com/sage-x-project/sage/crypto"
)

// rsaKeyPair implements the KeyPair interface for RSA keys (RS256)
type rsaKeyPair struct {
    privateKey *rsa.PrivateKey
    publicKey  *rsa.PublicKey
    id         string
}

// GenerateRSAKeyPair generates a new RSA key pair for RS256 (2048-bit)
func GenerateRSAKeyPair() (sagecrypto.KeyPair, error) {
    privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        return nil, err
    }

    publicKey := &privateKey.PublicKey

    // Generate ID from public key modulus hash (first 8 bytes of SHA-256)
    modBytes := publicKey.N.Bytes()
    hash := sha256.Sum256(modBytes)
    id := hex.EncodeToString(hash[:8])

    return &rsaKeyPair{
        privateKey: privateKey,
        publicKey:  publicKey,
        id:         id,
    }, nil
}

// PublicKey returns the public key
func (kp *rsaKeyPair) PublicKey() crypto.PublicKey {
    return kp.publicKey
}

// PrivateKey returns the private key
func (kp *rsaKeyPair) PrivateKey() crypto.PrivateKey {
    return kp.privateKey
}

// Type returns the key type
func (kp *rsaKeyPair) Type() sagecrypto.KeyType {
    return sagecrypto.KeyTypeRSA
}

// Sign signs the given message using RS256 (PKCS#1 v1.5 with SHA-256)
func (kp *rsaKeyPair) Sign(message []byte) ([]byte, error) {
    // Hash the message using SHA-256
    hash := sha256.Sum256(message)
    // Sign the hash with PKCS#1 v1.5
    signature, err := rsa.SignPKCS1v15(rand.Reader, kp.privateKey, crypto.SHA256, hash[:])
    if err != nil {
        return nil, err
    }
    return signature, nil
}

// Verify verifies the signature using RS256 (PKCS#1 v1.5 with SHA-256)
func (kp *rsaKeyPair) Verify(message, signature []byte) error {
    // Hash the message using SHA-256
    hash := sha256.Sum256(message)
    // Verify the signature
    err := rsa.VerifyPKCS1v15(kp.publicKey, crypto.SHA256, hash[:], signature)
    if err != nil {
        return sagecrypto.ErrInvalidSignature
    }
    return nil
}

// ID returns a unique identifier for this key pair
func (kp *rsaKeyPair) ID() string {
    return kp.id
}

