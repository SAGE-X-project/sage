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
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	sagecrypto "github.com/sage-x-project/sage/crypto"
)

// NewEd25519KeyPair creates a new Ed25519 key pair from an existing private key
func NewEd25519KeyPair(privateKey ed25519.PrivateKey, id string) (sagecrypto.KeyPair, error) {
	publicKey := privateKey.Public().(ed25519.PublicKey)
	
	// Use provided ID or generate from public key
	if id == "" {
		hash := sha256.Sum256(publicKey)
		id = hex.EncodeToString(hash[:8])
	}
	
	return &ed25519KeyPair{
		privateKey: privateKey,
		publicKey:  publicKey,
		id:         id,
	}, nil
}

// NewSecp256k1KeyPair creates a new Secp256k1 key pair from an existing private key
func NewSecp256k1KeyPair(privateKey *secp256k1.PrivateKey, id string) (sagecrypto.KeyPair, error) {
	publicKey := privateKey.PubKey()
	
	// Use provided ID or generate from public key
	if id == "" {
		pubKeyBytes := publicKey.SerializeCompressed()
		hash := sha256.Sum256(pubKeyBytes)
		id = hex.EncodeToString(hash[:8])
	}
	
	return &secp256k1KeyPair{
		privateKey: privateKey,
		publicKey:  publicKey,
		id:         id,
	}, nil
}

// NewX25519KeyPair creates a new X25519 key pair from an existing private key
func NewX25519KeyPair(privateKey *ecdh.PrivateKey, id string) (sagecrypto.KeyPair, error) {
	publicKey := privateKey.PublicKey()
	
	// Use provided ID or generate from public key
	if id == "" {
		pubKeyBytes := publicKey.Bytes()
		hash := sha256.Sum256(pubKeyBytes)
		id = hex.EncodeToString(hash[:8])
	}
	
	return &X25519KeyPair{
		privateKey: privateKey,
		publicKey:  publicKey,
		id:         id,
	}, nil
}


// NewRSAKeyPair creates a new RSA key pair for RS256 from an existing private key
func NewRSAKeyPair(privateKey *rsa.PrivateKey, id string) (sagecrypto.KeyPair, error) {
    publicKey := &privateKey.PublicKey
    if id == "" {
        // Derive ID from public key modulus hash
        hash := sha256.Sum256(publicKey.N.Bytes())
        id = hex.EncodeToString(hash[:8])
    }
    return &rsaKeyPair{
        privateKey: privateKey,
        publicKey:  publicKey,
        id:         id,
    }, nil
}

// PublicKeyOnlyEd25519 wraps an Ed25519 public key for verification only
type publicKeyOnlyEd25519 struct {
	publicKey ed25519.PublicKey
	id        string
}

func (pk *publicKeyOnlyEd25519) PublicKey() crypto.PublicKey {
	return pk.publicKey
}

func (pk *publicKeyOnlyEd25519) PrivateKey() crypto.PrivateKey {
	return nil
}

func (pk *publicKeyOnlyEd25519) Type() sagecrypto.KeyType {
	return sagecrypto.KeyTypeEd25519
}

func (pk *publicKeyOnlyEd25519) Sign(message []byte) ([]byte, error) {
	return nil, errors.New("cannot sign with public key only")
}

func (pk *publicKeyOnlyEd25519) Verify(message, signature []byte) error {
	if !ed25519.Verify(pk.publicKey, message, signature) {
		return sagecrypto.ErrInvalidSignature
	}
	return nil
}

func (pk *publicKeyOnlyEd25519) ID() string {
	return pk.id
}

// PublicKeyOnlyRSA wraps an RSA public key for verification only
type publicKeyOnlyRSA struct {
    publicKey *rsa.PublicKey
    id        string
}

func (pk *publicKeyOnlyRSA) PublicKey() crypto.PublicKey {
    return pk.publicKey
}

func (pk *publicKeyOnlyRSA) PrivateKey() crypto.PrivateKey {
    return nil
}

func (pk *publicKeyOnlyRSA) Type() sagecrypto.KeyType {
    return sagecrypto.KeyTypeRSA
}

func (pk *publicKeyOnlyRSA) Sign(message []byte) ([]byte, error) {
    return nil, errors.New("cannot sign with public key only")
}

func (pk *publicKeyOnlyRSA) Verify(message, signature []byte) error {
    hash := sha256.Sum256(message)
    if err := rsa.VerifyPKCS1v15(pk.publicKey, crypto.SHA256, hash[:], signature); err != nil {
        return sagecrypto.ErrInvalidSignature
    }
    return nil
}

func (pk *publicKeyOnlyRSA) ID() string {
    return pk.id
}
