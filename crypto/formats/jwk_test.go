package formats

import (
	"encoding/json"
	"testing"

	"github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWKExporter(t *testing.T) {
	exporter := NewJWKExporter()

	t.Run("ExportEd25519KeyPair", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Export full key pair
		exported, err := exporter.Export(keyPair, crypto.KeyFormatJWK)
		require.NoError(t, err)
		assert.NotEmpty(t, exported)

		// Verify it's valid JSON
		var jwk map[string]interface{}
		err = json.Unmarshal(exported, &jwk)
		require.NoError(t, err)

		// Check required fields
		assert.Equal(t, "OKP", jwk["kty"])
		assert.Equal(t, "Ed25519", jwk["crv"])
		assert.NotEmpty(t, jwk["x"])
		assert.NotEmpty(t, jwk["d"]) // Private key component
		assert.NotEmpty(t, jwk["kid"])
	})

	t.Run("ExportEd25519PublicKey", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Export only public key
		exported, err := exporter.ExportPublic(keyPair, crypto.KeyFormatJWK)
		require.NoError(t, err)
		assert.NotEmpty(t, exported)

		// Verify it's valid JSON
		var jwk map[string]interface{}
		err = json.Unmarshal(exported, &jwk)
		require.NoError(t, err)

		// Check required fields
		assert.Equal(t, "OKP", jwk["kty"])
		assert.Equal(t, "Ed25519", jwk["crv"])
		assert.NotEmpty(t, jwk["x"])
		assert.Empty(t, jwk["d"]) // No private key component
		assert.NotEmpty(t, jwk["kid"])
	})

	t.Run("ExportSecp256k1KeyPair", func(t *testing.T) {
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		// Export full key pair
		exported, err := exporter.Export(keyPair, crypto.KeyFormatJWK)
		require.NoError(t, err)
		assert.NotEmpty(t, exported)

		// Verify it's valid JSON
		var jwk map[string]interface{}
		err = json.Unmarshal(exported, &jwk)
		require.NoError(t, err)

		// Check required fields
		assert.Equal(t, "EC", jwk["kty"])
		assert.Equal(t, "secp256k1", jwk["crv"])
		assert.NotEmpty(t, jwk["x"])
		assert.NotEmpty(t, jwk["y"])
		assert.NotEmpty(t, jwk["d"]) // Private key component
		assert.NotEmpty(t, jwk["kid"])
	})

	t.Run("ExportSecp256k1PublicKey", func(t *testing.T) {
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		// Export only public key
		exported, err := exporter.ExportPublic(keyPair, crypto.KeyFormatJWK)
		require.NoError(t, err)
		assert.NotEmpty(t, exported)

		// Verify it's valid JSON
		var jwk map[string]interface{}
		err = json.Unmarshal(exported, &jwk)
		require.NoError(t, err)

		// Check required fields
		assert.Equal(t, "EC", jwk["kty"])
		assert.Equal(t, "secp256k1", jwk["crv"])
		assert.NotEmpty(t, jwk["x"])
		assert.NotEmpty(t, jwk["y"])
		assert.Empty(t, jwk["d"]) // No private key component
		assert.NotEmpty(t, jwk["kid"])
	})

	t.Run("ExportRSAKeyPair", func(t *testing.T) {
        keyPair, err := keys.GenerateRSAKeyPair()
        require.NoError(t, err)

        // Export full RSA key pair
        exported, err := exporter.Export(keyPair, crypto.KeyFormatJWK)
        require.NoError(t, err)
        assert.NotEmpty(t, exported)

        // Verify JSON structure
        var jwk map[string]interface{}
        err = json.Unmarshal(exported, &jwk)
        require.NoError(t, err)

        // Check required fields
        assert.Equal(t, "RSA", jwk["kty"])
        assert.Equal(t, "RS256", jwk["alg"])
        assert.NotEmpty(t, jwk["n"])   // modulus
        assert.NotEmpty(t, jwk["e"])   // exponent
        assert.NotEmpty(t, jwk["d"])   // private exponent
        assert.NotEmpty(t, jwk["kid"])
    })

	t.Run("ExportRSAKeyPair", func(t *testing.T) {
        keyPair, err := keys.GenerateRSAKeyPair()
        require.NoError(t, err)

        // Export full RSA key pair
        exported, err := exporter.Export(keyPair, crypto.KeyFormatJWK)
        require.NoError(t, err)
        assert.NotEmpty(t, exported)

        // Verify JSON structure
        var jwk map[string]interface{}
        err = json.Unmarshal(exported, &jwk)
        require.NoError(t, err)

        // Check required fields
        assert.Equal(t, "RSA", jwk["kty"])
        assert.Equal(t, "RS256", jwk["alg"])
        assert.NotEmpty(t, jwk["n"])   // modulus
        assert.NotEmpty(t, jwk["e"])   // exponent
        assert.NotEmpty(t, jwk["d"])   // private exponent
        assert.NotEmpty(t, jwk["kid"])
    })

	t.Run("ExportRSAPublicKey", func(t *testing.T) {
        keyPair, err := keys.GenerateRSAKeyPair()
        require.NoError(t, err)

        // Export only RSA public key
        exported, err := exporter.ExportPublic(keyPair, crypto.KeyFormatJWK)
        require.NoError(t, err)
        assert.NotEmpty(t, exported)

        // Verify JSON structure
        var jwk map[string]interface{}
        err = json.Unmarshal(exported, &jwk)
        require.NoError(t, err)

        // Check required fields
        assert.Equal(t, "RSA", jwk["kty"])
        assert.Equal(t, "RS256", jwk["alg"])
        assert.NotEmpty(t, jwk["n"])
        assert.NotEmpty(t, jwk["e"])
        assert.Empty(t, jwk["d"])   // no private exponent
        assert.NotEmpty(t, jwk["kid"])
    })
}

func TestJWKImporter(t *testing.T) {
	exporter := NewJWKExporter()
	importer := NewJWKImporter()

	t.Run("ImportEd25519KeyPair", func(t *testing.T) {
		// Generate and export a key pair
		originalKeyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		exported, err := exporter.Export(originalKeyPair, crypto.KeyFormatJWK)
		require.NoError(t, err)

		// Import the key pair
		importedKeyPair, err := importer.Import(exported, crypto.KeyFormatJWK)
		require.NoError(t, err)
		assert.NotNil(t, importedKeyPair)
		assert.Equal(t, crypto.KeyTypeEd25519, importedKeyPair.Type())

		// Test signing with imported key
		message := []byte("test message")
		signature, err := importedKeyPair.Sign(message)
		require.NoError(t, err)

		// Verify with original public key
		err = originalKeyPair.Verify(message, signature)
		assert.NoError(t, err)
	})

	t.Run("ImportSecp256k1KeyPair", func(t *testing.T) {
		// Generate and export a key pair
		originalKeyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		exported, err := exporter.Export(originalKeyPair, crypto.KeyFormatJWK)
		require.NoError(t, err)

		// Import the key pair
		importedKeyPair, err := importer.Import(exported, crypto.KeyFormatJWK)
		require.NoError(t, err)
		assert.NotNil(t, importedKeyPair)
		assert.Equal(t, crypto.KeyTypeSecp256k1, importedKeyPair.Type())

		// Test signing with imported key
		message := []byte("test message")
		signature, err := importedKeyPair.Sign(message)
		require.NoError(t, err)

		// Verify with original public key
		err = originalKeyPair.Verify(message, signature)
		assert.NoError(t, err)
	})

	t.Run("ImportRSAKeyPair", func(t *testing.T) {
        originalKeyPair, err := keys.GenerateRSAKeyPair()
        require.NoError(t, err)

        exported, err := exporter.Export(originalKeyPair, crypto.KeyFormatJWK)
        require.NoError(t, err)

        importedKeyPair, err := importer.Import(exported, crypto.KeyFormatJWK)
        require.NoError(t, err)
        assert.NotNil(t, importedKeyPair)
        assert.Equal(t, crypto.KeyTypeRSA, importedKeyPair.Type())

        // Test signing and verifying
        message := []byte("test message")
        signature, err := importedKeyPair.Sign(message)
        require.NoError(t, err)

        err = originalKeyPair.Verify(message, signature)
        assert.NoError(t, err)
    })

	t.Run("ImportEd25519PublicKey", func(t *testing.T) {
		// Generate and export a public key
		originalKeyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		exported, err := exporter.ExportPublic(originalKeyPair, crypto.KeyFormatJWK)
		require.NoError(t, err)

		// Import the public key
		importedPublicKey, err := importer.ImportPublic(exported, crypto.KeyFormatJWK)
		require.NoError(t, err)
		assert.NotNil(t, importedPublicKey)
	})

	t.Run("ImportRSAPublicKey", func(t *testing.T) {
        originalKeyPair, err := keys.GenerateRSAKeyPair()
        require.NoError(t, err)

        exported, err := exporter.ExportPublic(originalKeyPair, crypto.KeyFormatJWK)
        require.NoError(t, err)

        importedPublicKey, err := importer.ImportPublic(exported, crypto.KeyFormatJWK)
        require.NoError(t, err)
		assert.NotNil(t, importedPublicKey)
    })

	t.Run("ImportInvalidJSON", func(t *testing.T) {
		invalidData := []byte("invalid json")
		_, err := importer.Import(invalidData, crypto.KeyFormatJWK)
		assert.Error(t, err)
	})

	t.Run("ImportMissingKeyType", func(t *testing.T) {
		invalidJWK := []byte(`{"x": "test"}`)
		_, err := importer.Import(invalidJWK, crypto.KeyFormatJWK)
		assert.Error(t, err)
	})
}