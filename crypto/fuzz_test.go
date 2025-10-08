package crypto

import (
	"testing"
)

// FuzzKeyPairGeneration fuzzes key pair generation
func FuzzKeyPairGeneration(f *testing.F) {
	// Seed corpus
	f.Add(uint8(KeyTypeEd25519))
	f.Add(uint8(KeyTypeSecp256k1))
	f.Add(uint8(KeyTypeX25519))

	f.Fuzz(func(t *testing.T, keyTypeByte uint8) {
		// Map byte to valid key type
		var keyType KeyType
		switch keyTypeByte % 3 {
		case 0:
			keyType = KeyTypeEd25519
		case 1:
			keyType = KeyTypeSecp256k1
		case 2:
			keyType = KeyTypeX25519
		}

		// Generate key pair
		keyPair, err := GenerateKeyPair(keyType)
		if err != nil {
			t.Fatalf("Failed to generate key pair: %v", err)
		}

		// Verify key pair properties
		if len(keyPair.PublicKey()) == 0 {
			t.Fatal("Public key is empty")
		}

		if keyPair.Type() != keyType {
			t.Fatalf("Key type mismatch: expected %s, got %s", keyType, keyPair.Type())
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

	keyPair, _ := GenerateKeyPair(KeyTypeEd25519)

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
	f.Add(uint8(KeyTypeEd25519))
	f.Add(uint8(KeyTypeSecp256k1))

	f.Fuzz(func(t *testing.T, keyTypeByte uint8) {
		var keyType KeyType
		if keyTypeByte%2 == 0 {
			keyType = KeyTypeEd25519
		} else {
			keyType = KeyTypeSecp256k1
		}

		// Generate original key pair
		original, err := GenerateKeyPair(keyType)
		if err != nil {
			t.Fatalf("Failed to generate key pair: %v", err)
		}

		// Test JWK export/import
		jwk, err := original.ExportJWK()
		if err != nil {
			t.Fatalf("Failed to export JWK: %v", err)
		}

		imported, err := ImportJWK(jwk)
		if err != nil {
			t.Fatalf("Failed to import JWK: %v", err)
		}

		// Verify keys match
		if !equalBytes(original.PublicKey(), imported.PublicKey()) {
			t.Fatal("Public keys don't match after JWK round-trip")
		}

		// Test PEM export/import
		pem, err := original.ExportPEM()
		if err != nil {
			t.Fatalf("Failed to export PEM: %v", err)
		}

		imported2, err := ImportPEM(pem)
		if err != nil {
			t.Fatalf("Failed to import PEM: %v", err)
		}

		if !equalBytes(original.PublicKey(), imported2.PublicKey()) {
			t.Fatal("Public keys don't match after PEM round-trip")
		}
	})
}

// FuzzSignatureWithDifferentKeys fuzzes signature verification with different keys
func FuzzSignatureWithDifferentKeys(f *testing.F) {
	f.Add([]byte("message"))

	keyPair1, _ := GenerateKeyPair(KeyTypeEd25519)
	keyPair2, _ := GenerateKeyPair(KeyTypeEd25519)

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

	keyPair, _ := GenerateKeyPair(KeyTypeEd25519)

	f.Fuzz(func(t *testing.T, message, invalidSig []byte) {
		// Try to verify with invalid signature
		// Should not crash, should return error
		err := keyPair.Verify(message, invalidSig)

		// We expect an error for invalid signatures
		// The important thing is it doesn't panic
		_ = err
	})
}

// FuzzKeyDerivation fuzzes HPKE key derivation
func FuzzKeyDerivation(f *testing.F) {
	f.Add([]byte("context1"))
	f.Add([]byte(""))
	f.Add(make([]byte, 256))

	clientKey, _ := GenerateKeyPair(KeyTypeX25519)
	serverKey, _ := GenerateKeyPair(KeyTypeX25519)

	f.Fuzz(func(t *testing.T, context []byte) {
		// Derive keys (this should not panic)
		_, err := DeriveSessionKeys(
			clientKey,
			serverKey.PublicKey(),
			context,
		)

		if err != nil {
			// Some contexts might be invalid, that's okay
			// As long as it doesn't panic
			return
		}

		// If successful, verify we got different keys
		keys1, _ := DeriveSessionKeys(clientKey, serverKey.PublicKey(), context)
		keys2, _ := DeriveSessionKeys(clientKey, serverKey.PublicKey(), context)

		// Same input should produce same output
		if !equalBytes(keys1.EncryptKey, keys2.EncryptKey) {
			t.Fatal("Derived keys are not deterministic")
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
