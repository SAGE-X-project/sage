package formats

import (
	"strings"
	"testing"

	"github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPEMExporter(t *testing.T) {
	exporter := NewPEMExporter()

	t.Run("ExportEd25519KeyPair", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Export full key pair
		exported, err := exporter.Export(keyPair, crypto.KeyFormatPEM)
		require.NoError(t, err)
		assert.NotEmpty(t, exported)

		// Verify PEM format
		pemStr := string(exported)
		assert.Contains(t, pemStr, "-----BEGIN PRIVATE KEY-----")
		assert.Contains(t, pemStr, "-----END PRIVATE KEY-----")
	})

	t.Run("ExportEd25519PublicKey", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Export only public key
		exported, err := exporter.ExportPublic(keyPair, crypto.KeyFormatPEM)
		require.NoError(t, err)
		assert.NotEmpty(t, exported)

		// Verify PEM format
		pemStr := string(exported)
		assert.Contains(t, pemStr, "-----BEGIN PUBLIC KEY-----")
		assert.Contains(t, pemStr, "-----END PUBLIC KEY-----")
	})

	t.Run("ExportSecp256k1KeyPair", func(t *testing.T) {
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		// Export full key pair
		exported, err := exporter.Export(keyPair, crypto.KeyFormatPEM)
		require.NoError(t, err)
		assert.NotEmpty(t, exported)

		// Verify PEM format
		pemStr := string(exported)
		assert.Contains(t, pemStr, "-----BEGIN EC PRIVATE KEY-----")
		assert.Contains(t, pemStr, "-----END EC PRIVATE KEY-----")
	})

	t.Run("ExportSecp256k1PublicKey", func(t *testing.T) {
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		// Export only public key
		exported, err := exporter.ExportPublic(keyPair, crypto.KeyFormatPEM)
		require.NoError(t, err)
		assert.NotEmpty(t, exported)

		// Verify PEM format
		pemStr := string(exported)
		assert.Contains(t, pemStr, "-----BEGIN PUBLIC KEY-----")
		assert.Contains(t, pemStr, "-----END PUBLIC KEY-----")
	})
}

func TestPEMImporter(t *testing.T) {
	exporter := NewPEMExporter()
	importer := NewPEMImporter()

	t.Run("ImportEd25519KeyPair", func(t *testing.T) {
		// Generate and export a key pair
		originalKeyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		exported, err := exporter.Export(originalKeyPair, crypto.KeyFormatPEM)
		require.NoError(t, err)

		// Import the key pair
		importedKeyPair, err := importer.Import(exported, crypto.KeyFormatPEM)
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

		exported, err := exporter.Export(originalKeyPair, crypto.KeyFormatPEM)
		require.NoError(t, err)

		// Import the key pair
		importedKeyPair, err := importer.Import(exported, crypto.KeyFormatPEM)
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

	t.Run("ImportEd25519PublicKey", func(t *testing.T) {
		// Generate and export a public key
		originalKeyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		exported, err := exporter.ExportPublic(originalKeyPair, crypto.KeyFormatPEM)
		require.NoError(t, err)

		// Import the public key
		importedPublicKey, err := importer.ImportPublic(exported, crypto.KeyFormatPEM)
		require.NoError(t, err)
		assert.NotNil(t, importedPublicKey)
	})

	t.Run("ImportSecp256k1PublicKey", func(t *testing.T) {
		// Generate and export a public key
		originalKeyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		exported, err := exporter.ExportPublic(originalKeyPair, crypto.KeyFormatPEM)
		require.NoError(t, err)

		// Import the public key
		importedPublicKey, err := importer.ImportPublic(exported, crypto.KeyFormatPEM)
		require.NoError(t, err)
		assert.NotNil(t, importedPublicKey)
	})

	t.Run("ImportInvalidPEM", func(t *testing.T) {
		invalidData := []byte("invalid pem data")
		_, err := importer.Import(invalidData, crypto.KeyFormatPEM)
		assert.Error(t, err)
	})

	t.Run("ImportCorruptedPEM", func(t *testing.T) {
		corruptedPEM := []byte(`-----BEGIN PRIVATE KEY-----
corrupted base64 data here
-----END PRIVATE KEY-----`)
		_, err := importer.Import(corruptedPEM, crypto.KeyFormatPEM)
		assert.Error(t, err)
	})

	t.Run("MultipleKeysInPEM", func(t *testing.T) {
		// Create a PEM with multiple blocks
		keyPair1, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		keyPair2, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		pem1, err := exporter.Export(keyPair1, crypto.KeyFormatPEM)
		require.NoError(t, err)
		pem2, err := exporter.Export(keyPair2, crypto.KeyFormatPEM)
		require.NoError(t, err)

		// Combine PEMs
		combinedPEM := append(pem1, '\n')
		combinedPEM = append(combinedPEM, pem2...)

		// Should import the first key
		importedKeyPair, err := importer.Import(combinedPEM, crypto.KeyFormatPEM)
		require.NoError(t, err)
		assert.NotNil(t, importedKeyPair)
	})

	t.Run("PEMWithComments", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		exported, err := exporter.Export(keyPair, crypto.KeyFormatPEM)
		require.NoError(t, err)

		// Add comments to PEM
		lines := strings.Split(string(exported), "\n")
		lines[0] = "# This is a comment\n" + lines[0]
		pemWithComments := []byte(strings.Join(lines, "\n"))

		// Should still import successfully
		importedKeyPair, err := importer.Import(pemWithComments, crypto.KeyFormatPEM)
		require.NoError(t, err)
		assert.NotNil(t, importedKeyPair)
	})
}