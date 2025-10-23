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
	"crypto/sha256"
	"encoding/hex"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
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

// Sign signs the given message (Ethereum-compatible signature)
func (kp *secp256k1KeyPair) Sign(message []byte) ([]byte, error) {
	// For Ethereum compatibility, use Keccak256 hash

	privateKey := kp.privateKey.ToECDSA()
	var hash []byte
	if len(message) == 32 {
		hash = message
	} else {
		// Ethereum 호환 경로: 메시지를 Keccak256으로 해시해서 서명
		hash = ethcrypto.Keccak256(message)
	}

	// Sign using Ethereum's method which includes recovery byte
	signature, err := ethcrypto.Sign(hash, privateKey)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// Verify verifies the signature (Ethereum-compatible)
func (kp *secp256k1KeyPair) Verify(message, signature []byte) error {
	// For Ethereum compatibility, use Keccak256 hash
	hash := ethcrypto.Keccak256(message)

	// Handle both 64-byte and 65-byte signatures
	if len(signature) == 65 {
		// Remove recovery byte for verification
		signature = signature[:64]
	}

	// Deserialize the signature
	r, s, err := deserializeSignature(signature)
	if err != nil {
		return sagecrypto.ErrInvalidSignature
	}

	// Verify the signature
	verified := ecdsa.Verify(kp.publicKey.ToECDSA(), hash, r, s)
	if !verified {
		return sagecrypto.ErrInvalidSignature
	}

	return nil
}

// ID returns a unique identifier for this key pair
func (kp *secp256k1KeyPair) ID() string {
	return kp.id
}

// deserializeSignature deserializes an ECDSA signature
func deserializeSignature(data []byte) (*big.Int, *big.Int, error) {
	if len(data) != 64 {
		return nil, nil, sagecrypto.ErrInvalidSignature
	}

	r := new(big.Int).SetBytes(data[:32])
	s := new(big.Int).SetBytes(data[32:])

	return r, s, nil
}
