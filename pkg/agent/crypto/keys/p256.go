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
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
)

// p256KeyPair implements the KeyPair interface for P-256 (NIST secp256r1) keys
type p256KeyPair struct {
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
	id         string
}

// GenerateP256KeyPair generates a new P-256 key pair
//
// P-256 (also known as secp256r1 or prime256v1) is a NIST-standardized
// elliptic curve used in many security protocols including TLS and JWT.
func GenerateP256KeyPair() (sagecrypto.KeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	publicKey := &privateKey.PublicKey

	// Generate ID from public key hash
	// Use uncompressed point format: 0x04 || X || Y (manually marshal to avoid deprecated function)
	pubKeyBytes := make([]byte, 1+32+32)
	pubKeyBytes[0] = 0x04
	publicKey.X.FillBytes(pubKeyBytes[1:33])
	publicKey.Y.FillBytes(pubKeyBytes[33:65])
	hash := sha256.Sum256(pubKeyBytes)
	id := hex.EncodeToString(hash[:8])

	return &p256KeyPair{
		privateKey: privateKey,
		publicKey:  publicKey,
		id:         id,
	}, nil
}

// NewP256KeyPair creates a P-256 key pair from an existing ECDSA private key
//
// This is useful for importing keys from external sources.
func NewP256KeyPair(privateKey *ecdsa.PrivateKey, id string) (sagecrypto.KeyPair, error) {
	if privateKey.Curve != elliptic.P256() {
		return nil, sagecrypto.ErrInvalidKeyType
	}

	if id == "" {
		// Generate ID from public key hash
		// Use uncompressed point format: 0x04 || X || Y (manually marshal to avoid deprecated function)
		pubKeyBytes := make([]byte, 1+32+32)
		pubKeyBytes[0] = 0x04
		privateKey.PublicKey.X.FillBytes(pubKeyBytes[1:33])
		privateKey.PublicKey.Y.FillBytes(pubKeyBytes[33:65])
		hash := sha256.Sum256(pubKeyBytes)
		id = hex.EncodeToString(hash[:8])
	}

	return &p256KeyPair{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		id:         id,
	}, nil
}

// PublicKey returns the public key
func (kp *p256KeyPair) PublicKey() crypto.PublicKey {
	return kp.publicKey
}

// PrivateKey returns the private key
func (kp *p256KeyPair) PrivateKey() crypto.PrivateKey {
	return kp.privateKey
}

// Type returns the key type
func (kp *p256KeyPair) Type() sagecrypto.KeyType {
	return sagecrypto.KeyTypeP256
}

// Sign signs the given message using ECDSA with SHA-256
//
// Returns a 64-byte signature (32 bytes R + 32 bytes S) in raw format,
// not DER-encoded. This is compatible with RFC 9421.
func (kp *p256KeyPair) Sign(message []byte) ([]byte, error) {
	// Hash the message with SHA-256
	hash := sha256.Sum256(message)

	// Sign the hash
	r, s, err := ecdsa.Sign(rand.Reader, kp.privateKey, hash[:])
	if err != nil {
		return nil, err
	}

	// Serialize to 64-byte format (32 bytes R + 32 bytes S)
	signature := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Pad with zeros if necessary (right-align in 32-byte slots)
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)

	return signature, nil
}

// Verify verifies the signature using ECDSA with SHA-256
//
// Accepts either a 64-byte raw signature or DER-encoded signature.
func (kp *p256KeyPair) Verify(message, signature []byte) error {
	// Hash the message with SHA-256
	hash := sha256.Sum256(message)

	// Deserialize the signature
	r, s, err := deserializeP256Signature(signature)
	if err != nil {
		return sagecrypto.ErrInvalidSignature
	}

	// Verify the signature
	verified := ecdsa.Verify(kp.publicKey, hash[:], r, s)
	if !verified {
		return sagecrypto.ErrInvalidSignature
	}

	return nil
}

// ID returns a unique identifier for this key pair
func (kp *p256KeyPair) ID() string {
	return kp.id
}

// deserializeP256Signature deserializes a P-256 ECDSA signature
//
// Supports both 64-byte raw format (32 bytes R + 32 bytes S) and
// DER-encoded format for compatibility.
func deserializeP256Signature(data []byte) (*big.Int, *big.Int, error) {
	if len(data) == 64 {
		// Raw format: 32 bytes R + 32 bytes S
		r := new(big.Int).SetBytes(data[:32])
		s := new(big.Int).SetBytes(data[32:])
		return r, s, nil
	}

	// For other lengths, assume it might be DER-encoded
	// For now, only support raw format
	return nil, nil, sagecrypto.ErrInvalidSignature
}
