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


package crypto_test

import (
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

// FuzzKeyPairGeneration fuzzes key pair generation
func FuzzKeyPairGeneration(f *testing.F) {
	// Seed corpus
	f.Add(uint8(0)) // Ed25519
	f.Add(uint8(1)) // Secp256k1
	f.Add(uint8(2)) // X25519

	f.Fuzz(func(t *testing.T, keyTypeByte uint8) {
		// Map byte to key generation function
		var keyPair crypto.KeyPair
		var err error
		var expectedType crypto.KeyType

		switch keyTypeByte % 3 {
		case 0:
			keyPair, err = keys.GenerateEd25519KeyPair()
			expectedType = crypto.KeyTypeEd25519
		case 1:
			keyPair, err = keys.GenerateSecp256k1KeyPair()
			expectedType = crypto.KeyTypeSecp256k1
		case 2:
			keyPair, err = keys.GenerateX25519KeyPair()
			expectedType = crypto.KeyTypeX25519
		}

		if err != nil {
			t.Fatalf("Failed to generate key pair: %v", err)
		}

		// Verify key pair properties
		pubKey := keyPair.PublicKey()
		if pubKey == nil {
			t.Fatal("Public key is nil")
		}

		if keyPair.Type() != expectedType {
			t.Fatalf("Key type mismatch: expected %s, got %s", expectedType, keyPair.Type())
		}

		// Verify ID is set
		if keyPair.ID() == "" {
			t.Fatal("Key ID is empty")
		}
	})
}

// FuzzSignAndVerify fuzzes signing and verification
func FuzzSignAndVerify(f *testing.F) {
	// Seed corpus with various message sizes
	f.Add([]byte("hello"))
	f.Add([]byte(""))
	f.Add([]byte("a"))
	f.Add(make([]byte, 1024))

	keyPair, _ := keys.GenerateEd25519KeyPair()

	f.Fuzz(func(t *testing.T, message []byte) {
		// Sign the message
		signature, err := keyPair.Sign(message)
		if err != nil {
			t.Fatalf("Failed to sign message: %v", err)
		}

		// Verify the signature
		err = keyPair.Verify(message, signature)
		if err != nil {
			t.Fatalf("Failed to verify valid signature: %v", err)
		}

		// Verify that modified message fails
		if len(message) > 0 {
			modifiedMessage := make([]byte, len(message))
			copy(modifiedMessage, message)
			modifiedMessage[0] ^= 0xFF // Flip bits

			err = keyPair.Verify(modifiedMessage, signature)
			if err == nil {
				t.Fatal("Verification succeeded for modified message")
			}
		}

		// Verify that modified signature fails
		if len(signature) > 0 {
			modifiedSignature := make([]byte, len(signature))
			copy(modifiedSignature, signature)
			modifiedSignature[0] ^= 0xFF // Flip bits

			err = keyPair.Verify(message, modifiedSignature)
			if err == nil {
				t.Fatal("Verification succeeded for modified signature")
			}
		}
	})
}

// FuzzKeyExportImport fuzzes key export and import
func FuzzKeyExportImport(f *testing.F) {
	f.Add(uint8(0))
	f.Add(uint8(1))

	f.Fuzz(func(t *testing.T, keyTypeByte uint8) {
		var original crypto.KeyPair
		var err error

		if keyTypeByte%2 == 0 {
			original, err = keys.GenerateEd25519KeyPair()
		} else {
			original, err = keys.GenerateSecp256k1KeyPair()
		}

		if err != nil {
			t.Fatalf("Failed to generate key pair: %v", err)
		}

		// Test JWK export/import
		jwkExporter := formats.NewJWKExporter()
		jwkData, err := jwkExporter.Export(original, crypto.KeyFormatJWK)
		if err != nil {
			t.Fatalf("Failed to export JWK: %v", err)
		}

		jwkImporter := formats.NewJWKImporter()
		imported, err := jwkImporter.Import(jwkData, crypto.KeyFormatJWK)
		if err != nil {
			t.Fatalf("Failed to import JWK: %v", err)
		}

		// Verify types match
		if original.Type() != imported.Type() {
			t.Fatal("Key types don't match after JWK round-trip")
		}

		// Test PEM export/import
		pemExporter := formats.NewPEMExporter()
		pemData, err := pemExporter.Export(original, crypto.KeyFormatPEM)
		if err != nil {
			t.Fatalf("Failed to export PEM: %v", err)
		}

		pemImporter := formats.NewPEMImporter()
		imported2, err := pemImporter.Import(pemData, crypto.KeyFormatPEM)
		if err != nil {
			t.Fatalf("Failed to import PEM: %v", err)
		}

		// Verify types match
		if original.Type() != imported2.Type() {
			t.Fatal("Key types don't match after PEM round-trip")
		}
	})
}

// FuzzSignatureWithDifferentKeys fuzzes signature verification with different keys
func FuzzSignatureWithDifferentKeys(f *testing.F) {
	f.Add([]byte("message"))

	keyPair1, _ := keys.GenerateEd25519KeyPair()
	keyPair2, _ := keys.GenerateEd25519KeyPair()

	f.Fuzz(func(t *testing.T, message []byte) {
		// Sign with first key
		signature, err := keyPair1.Sign(message)
		if err != nil {
			t.Fatalf("Failed to sign: %v", err)
		}

		// Verify with second key should fail
		err = keyPair2.Verify(message, signature)
		if err == nil {
			t.Fatal("Verification succeeded with wrong key")
		}

		// Verify with correct key should succeed
		err = keyPair1.Verify(message, signature)
		if err != nil {
			t.Fatalf("Verification failed with correct key: %v", err)
		}
	})
}

// FuzzInvalidSignatureData fuzzes with invalid signature data
func FuzzInvalidSignatureData(f *testing.F) {
	f.Add([]byte("message"), []byte("invalid"))
	f.Add([]byte("test"), []byte(""))
	f.Add([]byte(""), []byte("sig"))

	keyPair, _ := keys.GenerateEd25519KeyPair()

	f.Fuzz(func(t *testing.T, message, invalidSig []byte) {
		// Try to verify with invalid signature
		// Should not crash, should return error
		err := keyPair.Verify(message, invalidSig)

		// We expect an error for invalid signatures
		// The important thing is it doesn't panic
		_ = err
	})
}

// FuzzKeyGeneration fuzzes key generation randomness
func FuzzKeyGeneration(f *testing.F) {
	f.Add([]byte("seed1"))
	f.Add([]byte(""))
	f.Add(make([]byte, 256))

	f.Fuzz(func(t *testing.T, _ []byte) {
		// Generate multiple keys to test randomness
		k1, err1 := keys.GenerateEd25519KeyPair()
		k2, err2 := keys.GenerateEd25519KeyPair()

		if err1 != nil || err2 != nil {
			t.Fatalf("Failed to generate keys: %v, %v", err1, err2)
		}

		// Verify keys are different
		if k1.ID() == k2.ID() {
			t.Fatal("Different key generations produced same ID")
		}

		// Both keys should be valid for signing
		testMsg := []byte("test message")
		sig1, err := k1.Sign(testMsg)
		if err != nil {
			t.Fatalf("Failed to sign with key1: %v", err)
		}

		sig2, err := k2.Sign(testMsg)
		if err != nil {
			t.Fatalf("Failed to sign with key2: %v", err)
		}

		// Signatures should be different
		if equalBytes(sig1, sig2) {
			t.Fatal("Different keys produced same signature")
		}
	})
}

// Helper function
func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
