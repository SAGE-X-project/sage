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

package hpke

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// SignatureVerifier defines the interface for signature verification algorithms.
//
// Design Pattern: Strategy Pattern (Open/Closed Principle)
// - Each verifier implements a specific algorithm
// - New algorithms can be added without modifying existing code
// - CompositeVerifier selects appropriate verifier at runtime
type SignatureVerifier interface {
	// Verify verifies a signature against a payload using the provided public key
	Verify(payload, signature []byte, publicKey crypto.PublicKey) error

	// Supports checks if this verifier supports the given public key type
	Supports(publicKey crypto.PublicKey) bool
}

// ECDSAVerifier verifies ECDSA signatures, primarily for Ethereum (Secp256k1).
//
// Supports multiple signature formats:
// - 64-byte raw format (r || s)
// - 65-byte Ethereum format (r || s || v) - automatically strips v
// - ASN.1 DER encoded format
//
// Security: Uses Keccak256 for hashing (Ethereum standard)
type ECDSAVerifier struct{}

// NewECDSAVerifier creates a new ECDSA signature verifier.
func NewECDSAVerifier() *ECDSAVerifier {
	return &ECDSAVerifier{}
}

// Verify verifies an ECDSA signature against the payload.
//
// Algorithm:
// 1. Hash payload using Keccak256 (Ethereum standard)
// 2. Try raw 64-byte format verification
// 3. If raw fails, try ASN.1 DER format
//
// Parameters:
//   - payload: Original message that was signed
//   - signature: ECDSA signature (64, 65 bytes, or DER)
//   - publicKey: ECDSA public key
//
// Returns:
//   - error: nil if valid, error describing failure otherwise
func (e *ECDSAVerifier) Verify(payload, signature []byte, publicKey crypto.PublicKey) error {
	ecdsaPub, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("expected *ecdsa.PublicKey, got %T", publicKey)
	}

	// Hash payload with Keccak256 (Ethereum standard)
	hash := ethcrypto.Keccak256(payload)

	// Handle 65-byte Ethereum signature format (strip recovery ID)
	rawSig := signature
	if len(rawSig) == 65 {
		rawSig = rawSig[:64] // Remove last byte (recovery ID)
	}

	// Convert ECDSA public key to Ethereum format (uncompressed, with 0x04 prefix)
	pubBytes := ethcrypto.FromECDSAPub(ecdsaPub)

	// Try Method 1: Ethereum's VerifySignature (raw 64-byte format)
	if len(rawSig) == 64 && ethcrypto.VerifySignature(pubBytes, hash, rawSig) {
		return nil
	}

	// Try Method 2: Standard ECDSA with r, s split (raw 64-byte)
	if len(rawSig) == 64 {
		r := new(big.Int).SetBytes(rawSig[:32])
		s := new(big.Int).SetBytes(rawSig[32:])
		if ecdsa.Verify(ecdsaPub, hash, r, s) {
			return nil
		}
	}

	// Try Method 3: ASN.1 DER encoded signature
	type ecdsaSignature struct {
		R, S *big.Int
	}
	var derSig ecdsaSignature
	if _, err := asn1.Unmarshal(signature, &derSig); err == nil && derSig.R != nil && derSig.S != nil {
		if ecdsa.Verify(ecdsaPub, hash, derSig.R, derSig.S) {
			return nil
		}
	}

	return fmt.Errorf("ecdsa(secp256k1) signature verification failed")
}

// Supports checks if the public key is an ECDSA key.
func (e *ECDSAVerifier) Supports(publicKey crypto.PublicKey) bool {
	_, ok := publicKey.(*ecdsa.PublicKey)
	return ok
}

// Ed25519Verifier verifies Ed25519 signatures.
//
// Design: Simple wrapper around standard library Ed25519 verification
type Ed25519Verifier struct{}

// NewEd25519Verifier creates a new Ed25519 signature verifier.
func NewEd25519Verifier() *Ed25519Verifier {
	return &Ed25519Verifier{}
}

// Verify verifies an Ed25519 signature against the payload.
func (e *Ed25519Verifier) Verify(payload, signature []byte, publicKey crypto.PublicKey) error {
	ed25519Pub, ok := publicKey.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("expected ed25519.PublicKey, got %T", publicKey)
	}

	if !ed25519.Verify(ed25519Pub, payload, signature) {
		return errors.New("ed25519 signature verification failed")
	}

	return nil
}

// Supports checks if the public key is an Ed25519 key.
func (e *Ed25519Verifier) Supports(publicKey crypto.PublicKey) bool {
	_, ok := publicKey.(ed25519.PublicKey)
	return ok
}

// CompositeVerifier combines multiple signature verifiers.
//
// Design Pattern: Strategy Pattern + Composite Pattern
// - Selects appropriate verifier based on public key type
// - Tries verifiers in order until one supports the key
// - Extensible: new verifiers can be added easily
//
// Usage:
//
//	composite := NewCompositeVerifier()
//	err := composite.Verify(payload, sig, publicKey)
type CompositeVerifier struct {
	verifiers []SignatureVerifier
}

// NewCompositeVerifier creates a composite verifier with default verifiers.
//
// Default verifiers (in priority order):
// 1. Ed25519Verifier
// 2. ECDSAVerifier (Ethereum/Secp256k1)
func NewCompositeVerifier() *CompositeVerifier {
	return &CompositeVerifier{
		verifiers: []SignatureVerifier{
			NewEd25519Verifier(),
			NewECDSAVerifier(),
		},
	}
}

// Verify selects and uses the appropriate verifier for the public key type.
//
// Algorithm:
// 1. Iterate through registered verifiers
// 2. Find first verifier that supports the public key type
// 3. Delegate verification to that verifier
//
// Returns:
//   - error: nil if verified, error if unsupported or verification failed
func (c *CompositeVerifier) Verify(payload, signature []byte, publicKey crypto.PublicKey) error {
	for _, verifier := range c.verifiers {
		if verifier.Supports(publicKey) {
			return verifier.Verify(payload, signature, publicKey)
		}
	}

	return fmt.Errorf("unsupported public key type: %T", publicKey)
}

// Supports checks if any registered verifier supports the public key.
func (c *CompositeVerifier) Supports(publicKey crypto.PublicKey) bool {
	for _, verifier := range c.verifiers {
		if verifier.Supports(publicKey) {
			return true
		}
	}
	return false
}
