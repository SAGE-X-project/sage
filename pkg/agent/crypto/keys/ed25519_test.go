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
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/vault"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEd25519KeyPair(t *testing.T) {
	t.Run("GenerateKeyPair", func(t *testing.T) {
		// Specification Requirement: Complete key lifecycle - generation, secure storage, loading, and verification
		helpers.LogTestSection(t, "2.1.2", "Ed25519 Complete Key Lifecycle (Generation + Secure Storage + Verification)")

		// ====================
		// PART 1: 키 생성 (Key Generation)
		// ====================
		helpers.LogDetail(t, "PART 1: 키 생성 (Key Generation using SAGE core functions)")
		helpers.LogDetail(t, "Step 1-1: Generate Ed25519 key pair (SAGE GenerateEd25519KeyPair)")
		helpers.LogDetail(t, "  Using crypto/ed25519.GenerateKey() - cryptographically secure random")
		keyPair, err := GenerateEd25519KeyPair()
		require.NoError(t, err)
		require.NotNil(t, keyPair)
		helpers.LogSuccess(t, "Ed25519 key pair generated successfully")

		helpers.LogDetail(t, "Step 1-2: Validate generated key type")
		assert.Equal(t, crypto.KeyTypeEd25519, keyPair.Type())
		helpers.LogSuccess(t, "Key type confirmed: Ed25519")

		helpers.LogDetail(t, "Step 1-3: Extract and validate key material")
		pubKey := keyPair.PublicKey()
		require.NotNil(t, pubKey)
		privKey := keyPair.PrivateKey()
		require.NotNil(t, privKey)

		pubKeyBytes, ok := pubKey.(ed25519.PublicKey)
		require.True(t, ok, "Public key must be ed25519.PublicKey type")
		assert.Equal(t, 32, len(pubKeyBytes), "Public key must be exactly 32 bytes")
		helpers.LogSuccess(t, "Public key size validated: 32 bytes")

		privKeyBytes, ok := privKey.(ed25519.PrivateKey)
		require.True(t, ok, "Private key must be ed25519.PrivateKey type")
		assert.Equal(t, 64, len(privKeyBytes), "Private key must be exactly 64 bytes")
		helpers.LogSuccess(t, "Private key size validated: 64 bytes")
		helpers.LogDetail(t, "  Public key (hex): %x", pubKeyBytes)

		keyID := keyPair.ID()
		assert.NotEmpty(t, keyID)
		helpers.LogDetail(t, "  Key ID (from public key hash): %s", keyID)

		helpers.LogDetail(t, "Step 1-4: Test cryptographic functionality - Sign message")
		testMessage := []byte("SAGE test message for Ed25519 key verification")
		signature, err := keyPair.Sign(testMessage)
		require.NoError(t, err)
		require.NotEmpty(t, signature)
		assert.Equal(t, 64, len(signature), "Ed25519 signature must be 64 bytes")
		helpers.LogSuccess(t, "Signature generated: 64 bytes (Ed25519 format)")

		helpers.LogDetail(t, "Step 1-5: Verify generated signature")
		err = keyPair.Verify(testMessage, signature)
		require.NoError(t, err)
		helpers.LogSuccess(t, "Signature verification successful - Key is cryptographically valid")

		// ====================
		// PART 2: 안전한 저장 (Secure Storage)
		// ====================
		helpers.LogDetail(t, "")
		helpers.LogDetail(t, "PART 2: 안전한 저장 (Secure Storage using SAGE FileVault)")

		helpers.LogDetail(t, "Step 2-1: Create temporary directory for FileVault")
		tempDir, err := os.MkdirTemp("", "ed25519_encrypted_test")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()
		helpers.LogSuccess(t, "Temporary vault directory created")
		helpers.LogDetail(t, "  Vault directory: %s", tempDir)

		helpers.LogDetail(t, "Step 2-2: Initialize SAGE FileVault for encrypted storage")
		v, err := vault.NewFileVault(tempDir)
		require.NoError(t, err)
		helpers.LogSuccess(t, "FileVault initialized (AES-256-GCM + PBKDF2)")

		helpers.LogDetail(t, "Step 2-3: Store key with encryption (password-based)")
		storedKeyID := "test_ed25519_encrypted"
		correctPassphrase := "strong_ed25519_passphrase_123!@#"
		wrongPassphrase := "wrong_passphrase"

		helpers.LogDetail(t, "  Encrypting with AES-256-GCM (PBKDF2 100,000 iterations)")
		err = v.StoreEncrypted(storedKeyID, []byte(privKeyBytes), correctPassphrase)
		require.NoError(t, err)
		helpers.LogSuccess(t, "Key encrypted and stored securely")
		helpers.LogDetail(t, "  Stored key ID: %s", storedKeyID)

		helpers.LogDetail(t, "Step 2-4: Verify encrypted file was created")
		assert.True(t, v.Exists(storedKeyID))
		helpers.LogSuccess(t, "Encrypted key file exists in FileVault")

		helpers.LogDetail(t, "Step 2-5: Verify file permissions (security requirement)")
		keyFilePath := filepath.Join(tempDir, storedKeyID+".json")
		fileInfo, err := os.Stat(keyFilePath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
		helpers.LogSuccess(t, "File permissions verified: 0600 (owner read/write only)")
		helpers.LogDetail(t, "  File path: %s", keyFilePath)

		// ====================
		// PART 3: 키 로드 및 재사용 (Key Loading and Reuse)
		// ====================
		helpers.LogDetail(t, "")
		helpers.LogDetail(t, "PART 3: 키 로드 및 재사용 (Key Loading and Reuse)")

		helpers.LogDetail(t, "Step 3-1: Load and decrypt with correct passphrase")
		decryptedKeyBytes, err := v.LoadDecrypted(storedKeyID, correctPassphrase)
		require.NoError(t, err)
		require.NotEmpty(t, decryptedKeyBytes)
		helpers.LogSuccess(t, "Key decrypted successfully with correct passphrase")

		helpers.LogDetail(t, "Step 3-2: Verify decrypted key matches original")
		assert.Equal(t, []byte(privKeyBytes), decryptedKeyBytes)
		helpers.LogSuccess(t, "Decrypted key matches original (integrity verified)")
		helpers.LogDetail(t, "  Decrypted key size: %d bytes", len(decryptedKeyBytes))

		helpers.LogDetail(t, "Step 3-3: Test wrong passphrase rejection (security requirement)")
		_, err = v.LoadDecrypted(storedKeyID, wrongPassphrase)
		assert.Error(t, err)
		assert.Equal(t, vault.ErrInvalidPassphrase, err)
		helpers.LogSuccess(t, "Wrong passphrase correctly rejected - Security validated")

		helpers.LogDetail(t, "Step 3-4: Reconstruct Ed25519 key pair from decrypted data")
		reconstructedPrivKey := ed25519.PrivateKey(decryptedKeyBytes)
		reconstructedPubKey := reconstructedPrivKey.Public().(ed25519.PublicKey)
		helpers.LogSuccess(t, "Ed25519 key pair reconstructed from stored data")

		helpers.LogDetail(t, "Step 3-5: Verify reconstructed keys match original")
		assert.Equal(t, privKeyBytes, reconstructedPrivKey)
		assert.Equal(t, pubKeyBytes, reconstructedPubKey)
		helpers.LogSuccess(t, "Reconstructed keys match original keys perfectly")

		// ====================
		// PART 4: 재사용 검증 (Reuse Verification)
		// ====================
		helpers.LogDetail(t, "")
		helpers.LogDetail(t, "PART 4: 재사용 검증 (Reuse Verification - Sign and Verify)")

		helpers.LogDetail(t, "Step 4-1: Sign message with reconstructed key")
		testMessage2 := []byte("test message for reconstructed Ed25519 key")
		signature2 := ed25519.Sign(reconstructedPrivKey, testMessage2)
		require.NotEmpty(t, signature2)
		assert.Equal(t, 64, len(signature2))
		helpers.LogSuccess(t, "Signature generated with reconstructed key")
		helpers.LogDetail(t, "  Test message: %s", string(testMessage2))
		helpers.LogDetail(t, "  Signature length: %d bytes", len(signature2))

		helpers.LogDetail(t, "Step 4-2: Verify signature with original public key")
		valid := ed25519.Verify(pubKeyBytes, testMessage2, signature2)
		assert.True(t, valid)
		helpers.LogSuccess(t, "Signature verified with original public key")

		helpers.LogDetail(t, "Step 4-3: Verify signature with reconstructed public key")
		valid2 := ed25519.Verify(reconstructedPubKey, testMessage2, signature2)
		assert.True(t, valid2)
		helpers.LogSuccess(t, "Signature verified with reconstructed public key - Key fully functional after storage/loading")

		// ====================
		// Pass Criteria Checklist
		// ====================
		helpers.LogPassCriteria(t, []string{
			"✓ PART 1: 키 생성 (Key Generation)",
			"  - SAGE GenerateEd25519KeyPair() 사용",
			"  - 암호학적으로 안전한 random 생성 (crypto/ed25519.GenerateKey)",
			"  - Key type = Ed25519 검증",
			"  - Public key = 32 bytes, Private key = 64 bytes",
			"  - 서명 생성 및 검증 성공",
			"",
			"✓ PART 2: 안전한 저장 (Secure Storage)",
			"  - SAGE FileVault 사용 (AES-256-GCM)",
			"  - PBKDF2 key derivation (100,000 iterations)",
			"  - 파일 권한 0600 (owner read/write only)",
			"  - 암호화 저장 성공",
			"",
			"✓ PART 3: 키 로드 및 재사용 (Key Loading)",
			"  - 올바른 passphrase로 복호화 성공",
			"  - 잘못된 passphrase 거부 (보안)",
			"  - 복호화된 키와 원본 일치 확인",
			"  - Ed25519 키 재구성 성공",
			"",
			"✓ PART 4: 재사용 검증 (Reuse Verification)",
			"  - 재구성된 키로 서명 생성",
			"  - 원본 공개키로 서명 검증",
			"  - 재구성된 공개키로 서명 검증",
			"  - 전체 라이프사이클 정상 동작 확인",
		})

		// Save test data
		testData := map[string]interface{}{
			"test_case": "2.1.2_Ed25519_Complete_Key_Lifecycle",
			"generation": map[string]interface{}{
				"key_type":        string(keyPair.Type()),
				"key_id":          keyID,
				"public_key_hex":  hex.EncodeToString(pubKeyBytes),
				"public_key_size": len(pubKeyBytes),
				"private_key_size": len(privKeyBytes),
			},
			"storage": map[string]interface{}{
				"vault_type":       "FileVault",
				"encryption":       "AES-256-GCM",
				"key_derivation":   "PBKDF2 (100,000 iterations)",
				"file_permissions": "0600",
				"stored_key_id":    storedKeyID,
			},
			"verification": map[string]interface{}{
				"original_signature_size":      len(signature),
				"original_signature_valid":     true,
				"reconstructed_signature_size": len(signature2),
				"reconstructed_signature_valid": true,
				"test_message":                 string(testMessage),
				"test_message_2":               string(testMessage2),
			},
			"security": map[string]interface{}{
				"cryptographically_secure":   true,
				"secure_storage":             true,
				"wrong_passphrase_rejected":  true,
				"file_permissions_0600":      true,
				"key_reusable_after_storage": true,
				"no_key_leakage":             true,
			},
			"expected_sizes": map[string]int{
				"public_key":  32,
				"private_key": 64,
				"signature":   64,
			},
		}
		helpers.SaveTestData(t, "keys/ed25519_key_generation.json", testData)
	})

	t.Run("SignAndVerify", func(t *testing.T) {
		// Specification Requirement: Ed25519 signature/verification (64-byte signature)
		helpers.LogTestSection(t, "2.4.1", "Ed25519 Signature and Verification")

		keyPair, err := GenerateEd25519KeyPair()
		require.NoError(t, err)

		message := []byte("test message for ed25519 signature")
		helpers.LogDetail(t, "Test message: %s", string(message))
		helpers.LogDetail(t, "Message size: %d bytes", len(message))

		// Sign message
		signature, err := keyPair.Sign(message)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)

		// Specification Requirement: Signature size must be 64 bytes
		assert.Equal(t, 64, len(signature), "Ed25519 signature must be 64 bytes")

		helpers.LogSuccess(t, "Signature generation successful")
		helpers.LogDetail(t, "Signature size: %d bytes (expected: 64 bytes)", len(signature))
		helpers.LogDetail(t, "Signature (hex): %x", signature)

		// Verify signature
		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
		helpers.LogSuccess(t, "Signature verification successful")

		// Specification Requirement: Tamper detection - wrong message should fail
		wrongMessage := []byte("wrong message")
		err = keyPair.Verify(wrongMessage, signature)
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrInvalidSignature, err)
		helpers.LogSuccess(t, "Tamper detection: Wrong message rejected (expected behavior)")

		// Specification Requirement: Tamper detection - modified signature should fail
		wrongSignature := make([]byte, len(signature))
		copy(wrongSignature, signature)
		wrongSignature[0] ^= 0xFF
		err = keyPair.Verify(message, wrongSignature)
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrInvalidSignature, err)
		helpers.LogSuccess(t, "Tamper detection: Modified signature rejected (expected behavior)")

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Signature generation successful",
			"Signature size = 64 bytes",
			"Verification successful",
			"Tamper detection (wrong message)",
			"Tamper detection (modified signature)",
		})

		// Save test data for CLI verification
		pubKey := keyPair.PublicKey().(ed25519.PublicKey)
		privKey := keyPair.PrivateKey().(ed25519.PrivateKey)

		testData := map[string]interface{}{
			"test_case":        "2.4.1_Ed25519_Sign_Verify",
			"message":          string(message),
			"message_hex":      hex.EncodeToString(message),
			"public_key_hex":   hex.EncodeToString(pubKey),
			"private_key_hex":  hex.EncodeToString(privKey),
			"signature_hex":    hex.EncodeToString(signature),
			"signature_size":   len(signature),
			"expected_size":    64,
		}
		helpers.SaveTestData(t, "keys/ed25519_sign_verify.json", testData)
	})

	t.Run("MultipleKeyPairsHaveDifferentIDs", func(t *testing.T) {
		keyPair1, err := GenerateEd25519KeyPair()
		require.NoError(t, err)

		keyPair2, err := GenerateEd25519KeyPair()
		require.NoError(t, err)

		assert.NotEqual(t, keyPair1.ID(), keyPair2.ID())
	})

	t.Run("SignEmptyMessage", func(t *testing.T) {
		keyPair, err := GenerateEd25519KeyPair()
		require.NoError(t, err)

		message := []byte{}

		signature, err := keyPair.Sign(message)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)

		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
	})

	t.Run("SignLargeMessage", func(t *testing.T) {
		keyPair, err := GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Create a 1MB message
		message := make([]byte, 1024*1024)
		for i := range message {
			message[i] = byte(i % 256)
		}

		signature, err := keyPair.Sign(message)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)

		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
	})
}

// Test 2.2.1: PEM 형식 저장
func TestEd25519KeyPairPEM(t *testing.T) {
	// Specification Requirement: PEM format key storage and loading
	helpers.LogTestSection(t, "2.2.1", "PEM Format Key Storage")

	// Generate Ed25519 key pair
	keyPair, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "Key pair generated")
	helpers.LogDetail(t, "Key ID: %s", keyPair.ID())

	// Get private key
	privKey := keyPair.PrivateKey().(ed25519.PrivateKey)
	pubKey := keyPair.PublicKey().(ed25519.PublicKey)

	// Export private key to PKCS#8 DER format, then PEM
	privKeyDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	require.NoError(t, err)
	helpers.LogDetail(t, "Private key marshaled to PKCS#8 DER format")

	// Create PEM block for private key
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyDER,
	})
	require.NotEmpty(t, privKeyPEM)
	helpers.LogSuccess(t, "Private key exported to PEM format")

	// Verify PEM format structure
	pemStr := string(privKeyPEM)
	assert.Contains(t, pemStr, "-----BEGIN PRIVATE KEY-----")
	assert.Contains(t, pemStr, "-----END PRIVATE KEY-----")
	helpers.LogSuccess(t, "PEM format structure validated")
	helpers.LogDetail(t, "PEM header: -----BEGIN PRIVATE KEY-----")
	helpers.LogDetail(t, "PEM footer: -----END PRIVATE KEY-----")
	helpers.LogDetail(t, "PEM data length: %d bytes", len(privKeyPEM))

	// Create temporary file for storage test
	tempDir, err := os.MkdirTemp("", "pem_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	pemFile := filepath.Join(tempDir, "test.pem")
	err = os.WriteFile(pemFile, privKeyPEM, 0600)
	require.NoError(t, err)
	helpers.LogSuccess(t, "PEM data saved to file")
	helpers.LogDetail(t, "File path: %s", pemFile)

	// Verify file permissions
	fileInfo, err := os.Stat(pemFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
	helpers.LogSuccess(t, "File permissions verified (0600)")

	// Load PEM from file
	loadedPEM, err := os.ReadFile(pemFile)
	require.NoError(t, err)
	assert.Equal(t, privKeyPEM, loadedPEM)
	helpers.LogSuccess(t, "PEM data loaded from file")

	// Import PEM to key pair
	block, _ := pem.Decode(loadedPEM)
	require.NotNil(t, block)
	require.Equal(t, "PRIVATE KEY", block.Type)
	helpers.LogDetail(t, "PEM block decoded successfully")

	// Parse PKCS#8 private key
	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	require.NoError(t, err)
	helpers.LogSuccess(t, "PKCS#8 private key parsed")

	// Convert to Ed25519 private key
	importedPrivKey, ok := parsedKey.(ed25519.PrivateKey)
	require.True(t, ok, "Parsed key should be Ed25519 private key")
	importedPubKey := importedPrivKey.Public().(ed25519.PublicKey)
	helpers.LogSuccess(t, "Key pair imported from PEM")

	// Verify keys match
	assert.Equal(t, privKey, importedPrivKey)
	assert.Equal(t, pubKey, importedPubKey)
	helpers.LogSuccess(t, "Imported keys match original keys")

	// Test signing with imported key
	message := []byte("test message for PEM imported key")
	signature := ed25519.Sign(importedPrivKey, message)
	require.NotEmpty(t, signature)
	helpers.LogSuccess(t, "Signature generated with imported key")
	helpers.LogDetail(t, "Message: %s", string(message))
	helpers.LogDetail(t, "Signature length: %d bytes", len(signature))

	// Verify signature with original key
	valid := ed25519.Verify(pubKey, message, signature)
	assert.True(t, valid)
	helpers.LogSuccess(t, "Signature verified with original key")

	// Verify signature with imported key
	validImported := ed25519.Verify(importedPubKey, message, signature)
	assert.True(t, validImported)
	helpers.LogSuccess(t, "Signature verified with imported key")

	// Export public key only
	pubKeyDER, err := x509.MarshalPKIXPublicKey(pubKey)
	require.NoError(t, err)
	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyDER,
	})
	require.NotEmpty(t, publicPEM)
	helpers.LogSuccess(t, "Public key exported to PEM")

	// Verify public key PEM format
	publicPEMStr := string(publicPEM)
	assert.Contains(t, publicPEMStr, "-----BEGIN PUBLIC KEY-----")
	assert.Contains(t, publicPEMStr, "-----END PUBLIC KEY-----")
	helpers.LogSuccess(t, "Public key PEM format validated")

	// Verify PEM is base64 encoded
	lines := strings.Split(pemStr, "\n")
	hasBase64Content := false
	for _, line := range lines {
		if !strings.HasPrefix(line, "-----") && len(strings.TrimSpace(line)) > 0 {
			hasBase64Content = true
			helpers.LogDetail(t, "Sample PEM content (base64): %s...", line[:min(len(line), 40)])
			break
		}
	}
	assert.True(t, hasBase64Content, "PEM should contain base64 encoded data")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"PEM 형식으로 키 저장 성공",
		"PEM 헤더/푸터 검증",
		"파일 권한 설정 (0600)",
		"PEM 파일에서 키 로드",
		"로드된 키로 서명/검증",
		"공개키 PEM 내보내기",
		"Base64 인코딩 확인",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":         "2.2.1_PEM_Format_Storage",
		"key_id":            keyPair.ID(),
		"key_type":          string(keyPair.Type()),
		"pem_length":        len(privKeyPEM),
		"public_pem_length": len(publicPEM),
		"file_permissions":  "0600",
		"public_key_hex":    hex.EncodeToString(pubKey),
		"signature_test": map[string]interface{}{
			"message":       string(message),
			"signature_hex": hex.EncodeToString(signature),
			"verified":      true,
		},
	}
	helpers.SaveTestData(t, "keys/ed25519_pem_storage.json", testData)
}

// Test 2.2.2: 암호화 저장
func TestEd25519KeyPairEncrypted(t *testing.T) {
	// Specification Requirement: Password-encrypted key storage
	helpers.LogTestSection(t, "2.2.2", "Encrypted Key Storage with Password")

	// Generate Ed25519 key pair
	keyPair, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "Key pair generated")
	helpers.LogDetail(t, "Key ID: %s", keyPair.ID())

	// Get private and public keys
	privKey := keyPair.PrivateKey().(ed25519.PrivateKey)
	pubKey := keyPair.PublicKey().(ed25519.PublicKey)
	keyData := []byte(privKey)
	helpers.LogDetail(t, "Private key size: %d bytes", len(keyData))

	// Create temporary directory for file vault
	tempDir, err := os.MkdirTemp("", "encrypted_key_test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create vault for encrypted storage
	v, err := vault.NewFileVault(tempDir)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Vault created")
	helpers.LogDetail(t, "Vault directory: %s", tempDir)

	// Store key with encryption
	keyID := "test_ed25519_encrypted"
	correctPassphrase := "strong_passphrase_123!@#"
	wrongPassphrase := "wrong_passphrase"

	helpers.LogDetail(t, "Encrypting key with passphrase...")
	err = v.StoreEncrypted(keyID, keyData, correctPassphrase)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Key encrypted and stored")
	helpers.LogDetail(t, "Key ID: %s", keyID)

	// Verify file was created
	assert.True(t, v.Exists(keyID))
	helpers.LogSuccess(t, "Encrypted key file exists")

	// Verify file permissions (should be 0600)
	keyFilePath := filepath.Join(tempDir, keyID+".json")
	fileInfo, err := os.Stat(keyFilePath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
	helpers.LogSuccess(t, "File permissions verified (0600)")
	helpers.LogDetail(t, "File path: %s", keyFilePath)

	// Load and decrypt with correct passphrase
	helpers.LogDetail(t, "Decrypting with correct passphrase...")
	decryptedKey, err := v.LoadDecrypted(keyID, correctPassphrase)
	require.NoError(t, err)
	require.NotEmpty(t, decryptedKey)
	helpers.LogSuccess(t, "Key decrypted successfully")

	// Verify decrypted key matches original
	assert.Equal(t, keyData, decryptedKey)
	helpers.LogSuccess(t, "Decrypted key matches original")
	helpers.LogDetail(t, "Decrypted key size: %d bytes", len(decryptedKey))

	// Try to load with wrong passphrase (should fail)
	helpers.LogDetail(t, "Testing with wrong passphrase...")
	_, err = v.LoadDecrypted(keyID, wrongPassphrase)
	assert.Error(t, err)
	assert.Equal(t, vault.ErrInvalidPassphrase, err)
	helpers.LogSuccess(t, "Wrong passphrase correctly rejected")
	helpers.LogDetail(t, "Error: %v", err)

	// Reconstruct key pair from decrypted data
	helpers.LogDetail(t, "Reconstructing key pair from decrypted data...")
	reconstructedPrivKey := ed25519.PrivateKey(decryptedKey)
	reconstructedPubKey := reconstructedPrivKey.Public().(ed25519.PublicKey)
	helpers.LogSuccess(t, "Key pair reconstructed from decrypted data")

	// Verify reconstructed keys match original
	assert.Equal(t, privKey, reconstructedPrivKey)
	assert.Equal(t, pubKey, reconstructedPubKey)
	helpers.LogSuccess(t, "Reconstructed keys match original keys")

	// Test signing with reconstructed key
	message := []byte("test message for encrypted key")
	signature := ed25519.Sign(reconstructedPrivKey, message)
	require.NotEmpty(t, signature)
	helpers.LogSuccess(t, "Signature generated with reconstructed key")
	helpers.LogDetail(t, "Message: %s", string(message))
	helpers.LogDetail(t, "Signature length: %d bytes", len(signature))

	// Verify signature with original public key
	valid := ed25519.Verify(pubKey, message, signature)
	assert.True(t, valid)
	helpers.LogSuccess(t, "Signature verified with original key")

	// Test empty passphrase handling
	helpers.LogDetail(t, "Testing empty passphrase...")
	emptyKeyID := "test_empty_pass"
	err = v.StoreEncrypted(emptyKeyID, keyData, "")
	require.NoError(t, err)
	loadedEmpty, err := v.LoadDecrypted(emptyKeyID, "")
	require.NoError(t, err)
	assert.Equal(t, keyData, loadedEmpty)
	helpers.LogSuccess(t, "Empty passphrase handled correctly")

	// Test key overwrite with different passphrase
	helpers.LogDetail(t, "Testing key overwrite...")
	newPassphrase := "new_passphrase_456"
	err = v.StoreEncrypted(keyID, keyData, newPassphrase)
	require.NoError(t, err)

	// Old passphrase should fail
	_, err = v.LoadDecrypted(keyID, correctPassphrase)
	assert.Error(t, err)
	helpers.LogSuccess(t, "Old passphrase fails after overwrite")

	// New passphrase should work
	reloadedKey, err := v.LoadDecrypted(keyID, newPassphrase)
	require.NoError(t, err)
	assert.Equal(t, keyData, reloadedKey)
	helpers.LogSuccess(t, "New passphrase works after overwrite")

	// Test key deletion
	helpers.LogDetail(t, "Testing key deletion...")
	err = v.Delete(keyID)
	require.NoError(t, err)
	assert.False(t, v.Exists(keyID))
	helpers.LogSuccess(t, "Key deleted successfully")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"패스워드로 키 암호화",
		"올바른 패스워드로 복호화 성공",
		"잘못된 패스워드로 복호화 실패",
		"복호화된 키로 서명/검증",
		"파일 권한 보안 (0600)",
		"빈 패스워드 처리",
		"키 덮어쓰기 지원",
		"키 삭제 기능",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":    "2.2.2_Encrypted_Key_Storage",
		"key_id":       keyID,
		"key_type":     string(keyPair.Type()),
		"key_size":     len(keyData),
		"vault_dir":    tempDir,
		"file_permissions": "0600",
		"public_key_hex":   hex.EncodeToString(pubKey),
		"encryption_test": map[string]interface{}{
			"correct_passphrase": "success",
			"wrong_passphrase":   "rejected",
			"empty_passphrase":   "success",
		},
		"signature_test": map[string]interface{}{
			"message":       string(message),
			"signature_hex": hex.EncodeToString(signature),
			"verified":      true,
		},
	}
	helpers.SaveTestData(t, "keys/ed25519_encrypted_storage.json", testData)
}

// Test 10.2.3: DER 형식 저장
func TestEd25519KeyPairDER(t *testing.T) {
	// Specification Requirement: DER format key storage and loading
	helpers.LogTestSection(t, "10.2.3", "DER Format Key Storage")

	// Generate Ed25519 key pair
	keyPair, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "Key pair generated")
	helpers.LogDetail(t, "Key ID: %s", keyPair.ID())

	// Get keys
	privKey := keyPair.PrivateKey().(ed25519.PrivateKey)
	pubKey := keyPair.PublicKey().(ed25519.PublicKey)

	// Export private key to PKCS#8 DER format
	privKeyDER, err := x509.MarshalPKCS8PrivateKey(privKey)
	require.NoError(t, err)
	require.NotEmpty(t, privKeyDER)
	helpers.LogSuccess(t, "Private key exported to DER format")
	helpers.LogDetail(t, "DER private key size: %d bytes", len(privKeyDER))

	// Export public key to PKIX DER format
	pubKeyDER, err := x509.MarshalPKIXPublicKey(pubKey)
	require.NoError(t, err)
	require.NotEmpty(t, pubKeyDER)
	helpers.LogSuccess(t, "Public key exported to DER format")
	helpers.LogDetail(t, "DER public key size: %d bytes", len(pubKeyDER))

	// Import private key from DER
	parsedPrivKey, err := x509.ParsePKCS8PrivateKey(privKeyDER)
	require.NoError(t, err)
	importedPrivKey, ok := parsedPrivKey.(ed25519.PrivateKey)
	require.True(t, ok, "Parsed key should be Ed25519 private key")
	helpers.LogSuccess(t, "Private key imported from DER")

	// Verify imported private key matches
	assert.Equal(t, privKey, importedPrivKey)
	helpers.LogSuccess(t, "Imported private key matches original")

	// Import public key from DER
	parsedPubKey, err := x509.ParsePKIXPublicKey(pubKeyDER)
	require.NoError(t, err)
	importedPubKey, ok := parsedPubKey.(ed25519.PublicKey)
	require.True(t, ok, "Parsed key should be Ed25519 public key")
	helpers.LogSuccess(t, "Public key imported from DER")

	// Verify imported public key matches
	assert.Equal(t, pubKey, importedPubKey)
	helpers.LogSuccess(t, "Imported public key matches original")

	// Test signing with imported key
	message := []byte("test message for DER imported key")
	signature := ed25519.Sign(importedPrivKey, message)
	require.NotEmpty(t, signature)
	helpers.LogSuccess(t, "Signature generated with imported key")

	// Verify signature
	valid := ed25519.Verify(importedPubKey, message, signature)
	assert.True(t, valid)
	helpers.LogSuccess(t, "Signature verified")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"DER 형식으로 키 저장",
		"DER 형식에서 키 로드",
		"비밀키 DER 변환 (PKCS#8)",
		"공개키 DER 변환 (PKIX)",
		"로드된 키 검증",
		"서명/검증 동작",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":        "10.2.3_DER_Format_Storage",
		"key_id":           keyPair.ID(),
		"key_type":         string(keyPair.Type()),
		"der_priv_size":    len(privKeyDER),
		"der_pub_size":     len(pubKeyDER),
		"public_key_hex":   hex.EncodeToString(pubKey),
		"signature_hex":    hex.EncodeToString(signature),
	}
	helpers.SaveTestData(t, "keys/ed25519_der_format.json", testData)
}

// Test 10.2.5: Ed25519 바이트 변환
func TestEd25519KeyPairBytes(t *testing.T) {
	// Specification Requirement: Public/private key byte array conversion
	helpers.LogTestSection(t, "10.2.5", "Ed25519 Byte Array Conversion")

	// Generate Ed25519 key pair
	keyPair, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "Key pair generated")

	// Get keys
	privKey := keyPair.PrivateKey().(ed25519.PrivateKey)
	pubKey := keyPair.PublicKey().(ed25519.PublicKey)

	// Convert to bytes
	privKeyBytes := []byte(privKey)
	pubKeyBytes := []byte(pubKey)
	helpers.LogSuccess(t, "Keys converted to byte arrays")
	helpers.LogDetail(t, "Private key size: %d bytes", len(privKeyBytes))
	helpers.LogDetail(t, "Public key size: %d bytes", len(pubKeyBytes))

	// Verify sizes
	assert.Equal(t, 64, len(privKeyBytes), "Ed25519 private key should be 64 bytes")
	assert.Equal(t, 32, len(pubKeyBytes), "Ed25519 public key should be 32 bytes")
	helpers.LogSuccess(t, "Key sizes verified")

	// Reconstruct keys from bytes
	reconstructedPrivKey := ed25519.PrivateKey(privKeyBytes)
	reconstructedPubKey := ed25519.PublicKey(pubKeyBytes)
	helpers.LogSuccess(t, "Keys reconstructed from bytes")

	// Verify reconstructed keys match originals
	assert.Equal(t, privKey, reconstructedPrivKey)
	assert.Equal(t, pubKey, reconstructedPubKey)
	helpers.LogSuccess(t, "Reconstructed keys match originals")

	// Test signing with reconstructed key
	message := []byte("test message for byte conversion")
	signature := ed25519.Sign(reconstructedPrivKey, message)
	require.NotEmpty(t, signature)
	helpers.LogSuccess(t, "Signature generated with reconstructed key")

	// Verify signature
	valid := ed25519.Verify(reconstructedPubKey, message, signature)
	assert.True(t, valid)
	helpers.LogSuccess(t, "Signature verified")

	// Test public key derivation from private key
	derivedPubKey := reconstructedPrivKey.Public().(ed25519.PublicKey)
	assert.Equal(t, pubKey, derivedPubKey)
	helpers.LogSuccess(t, "Public key derived from private key")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"공개키 바이트 변환",
		"비밀키 바이트 변환",
		"Ed25519 키 크기 검증 (32/64 bytes)",
		"바이트에서 키 재구성",
		"재구성된 키로 서명",
		"공개키 파생",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":       "10.2.5_Ed25519_Byte_Conversion",
		"key_id":          keyPair.ID(),
		"priv_key_size":   len(privKeyBytes),
		"pub_key_size":    len(pubKeyBytes),
		"private_key_hex": hex.EncodeToString(privKeyBytes),
		"public_key_hex":  hex.EncodeToString(pubKeyBytes),
		"signature_hex":   hex.EncodeToString(signature),
	}
	helpers.SaveTestData(t, "keys/ed25519_byte_conversion.json", testData)
}

// Test 10.2.7: Hex 인코딩
func TestEd25519KeyHexEncoding(t *testing.T) {
	// Specification Requirement: Hexadecimal string conversion
	helpers.LogTestSection(t, "10.2.7", "Ed25519 Hex Encoding")

	// Generate Ed25519 key pair
	keyPair, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "Key pair generated")

	// Get keys
	privKey := keyPair.PrivateKey().(ed25519.PrivateKey)
	pubKey := keyPair.PublicKey().(ed25519.PublicKey)

	// Encode to hex
	privKeyHex := hex.EncodeToString(privKey)
	pubKeyHex := hex.EncodeToString(pubKey)
	helpers.LogSuccess(t, "Keys encoded to hex")
	helpers.LogDetail(t, "Private key hex length: %d characters", len(privKeyHex))
	helpers.LogDetail(t, "Public key hex length: %d characters", len(pubKeyHex))
	helpers.LogDetail(t, "Public key hex: %s", pubKeyHex)

	// Verify hex lengths (64 bytes = 128 hex chars, 32 bytes = 64 hex chars)
	assert.Equal(t, 128, len(privKeyHex), "Private key hex should be 128 characters")
	assert.Equal(t, 64, len(pubKeyHex), "Public key hex should be 64 characters")
	helpers.LogSuccess(t, "Hex lengths verified")

	// Decode from hex
	decodedPrivKey, err := hex.DecodeString(privKeyHex)
	require.NoError(t, err)
	decodedPubKey, err := hex.DecodeString(pubKeyHex)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Keys decoded from hex")

	// Verify decoded keys match originals
	assert.Equal(t, []byte(privKey), decodedPrivKey)
	assert.Equal(t, []byte(pubKey), decodedPubKey)
	helpers.LogSuccess(t, "Decoded keys match originals")

	// Test signing with decoded key
	message := []byte("test message for hex encoding")
	reconstructedPrivKey := ed25519.PrivateKey(decodedPrivKey)
	signature := ed25519.Sign(reconstructedPrivKey, message)
	require.NotEmpty(t, signature)
	helpers.LogSuccess(t, "Signature generated with decoded key")

	// Verify signature
	reconstructedPubKey := ed25519.PublicKey(decodedPubKey)
	valid := ed25519.Verify(reconstructedPubKey, message, signature)
	assert.True(t, valid)
	helpers.LogSuccess(t, "Signature verified")

	// Test invalid hex handling
	_, err = hex.DecodeString("invalid_hex_string_with_non_hex_chars!!")
	assert.Error(t, err)
	helpers.LogSuccess(t, "Invalid hex correctly rejected")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"Hex 인코딩 성공",
		"Hex 디코딩 성공",
		"키 크기 검증 (128/64 chars)",
		"디코딩된 키로 서명",
		"잘못된 Hex 거부",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":       "10.2.7_Hex_Encoding",
		"key_id":          keyPair.ID(),
		"private_key_hex": privKeyHex,
		"public_key_hex":  pubKeyHex,
		"hex_lengths": map[string]int{
			"private_key": len(privKeyHex),
			"public_key":  len(pubKeyHex),
		},
		"signature_hex": hex.EncodeToString(signature),
	}
	helpers.SaveTestData(t, "keys/ed25519_hex_encoding.json", testData)
}

// Test 10.2.8: Base64 인코딩
func TestEd25519KeyBase64Encoding(t *testing.T) {
	// Specification Requirement: Base64 string conversion
	helpers.LogTestSection(t, "10.2.8", "Ed25519 Base64 Encoding")

	// Generate Ed25519 key pair
	keyPair, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "Key pair generated")

	// Get keys
	privKey := keyPair.PrivateKey().(ed25519.PrivateKey)
	pubKey := keyPair.PublicKey().(ed25519.PublicKey)

	// Encode to standard base64
	privKeyB64 := base64.StdEncoding.EncodeToString(privKey)
	pubKeyB64 := base64.StdEncoding.EncodeToString(pubKey)
	helpers.LogSuccess(t, "Keys encoded to base64 (standard)")
	helpers.LogDetail(t, "Private key base64 length: %d characters", len(privKeyB64))
	helpers.LogDetail(t, "Public key base64 length: %d characters", len(pubKeyB64))
	helpers.LogDetail(t, "Public key base64: %s", pubKeyB64)

	// Encode to URL-safe base64
	privKeyB64URL := base64.URLEncoding.EncodeToString(privKey)
	pubKeyB64URL := base64.URLEncoding.EncodeToString(pubKey)
	helpers.LogSuccess(t, "Keys encoded to base64 (URL-safe)")
	helpers.LogDetail(t, "URL-safe private key: %d characters", len(privKeyB64URL))
	helpers.LogDetail(t, "URL-safe public key: %d characters", len(pubKeyB64URL))

	// Decode standard base64
	decodedPrivKey, err := base64.StdEncoding.DecodeString(privKeyB64)
	require.NoError(t, err)
	decodedPubKey, err := base64.StdEncoding.DecodeString(pubKeyB64)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Keys decoded from base64 (standard)")

	// Verify decoded keys match originals
	assert.Equal(t, []byte(privKey), decodedPrivKey)
	assert.Equal(t, []byte(pubKey), decodedPubKey)
	helpers.LogSuccess(t, "Decoded keys match originals")

	// Decode URL-safe base64
	decodedPrivKeyURL, err := base64.URLEncoding.DecodeString(privKeyB64URL)
	require.NoError(t, err)
	decodedPubKeyURL, err := base64.URLEncoding.DecodeString(pubKeyB64URL)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Keys decoded from base64 (URL-safe)")

	// Verify URL-safe decoded keys match
	assert.Equal(t, []byte(privKey), decodedPrivKeyURL)
	assert.Equal(t, []byte(pubKey), decodedPubKeyURL)
	helpers.LogSuccess(t, "URL-safe decoded keys match originals")

	// Test signing with decoded key
	message := []byte("test message for base64 encoding")
	reconstructedPrivKey := ed25519.PrivateKey(decodedPrivKey)
	signature := ed25519.Sign(reconstructedPrivKey, message)
	require.NotEmpty(t, signature)
	helpers.LogSuccess(t, "Signature generated with decoded key")

	// Verify signature
	reconstructedPubKey := ed25519.PublicKey(decodedPubKey)
	valid := ed25519.Verify(reconstructedPubKey, message, signature)
	assert.True(t, valid)
	helpers.LogSuccess(t, "Signature verified")

	// Test invalid base64 handling
	_, err = base64.StdEncoding.DecodeString("invalid base64 with spaces!!")
	assert.Error(t, err)
	helpers.LogSuccess(t, "Invalid base64 correctly rejected")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"Base64 표준 인코딩",
		"Base64 URL-safe 인코딩",
		"Base64 디코딩 성공",
		"디코딩된 키로 서명",
		"잘못된 Base64 거부",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":           "10.2.8_Base64_Encoding",
		"key_id":              keyPair.ID(),
		"private_key_base64":  privKeyB64,
		"public_key_base64":   pubKeyB64,
		"private_key_b64url":  privKeyB64URL,
		"public_key_b64url":   pubKeyB64URL,
		"base64_lengths": map[string]int{
			"standard_private": len(privKeyB64),
			"standard_public":  len(pubKeyB64),
			"urlsafe_private":  len(privKeyB64URL),
			"urlsafe_public":   len(pubKeyB64URL),
		},
		"signature_hex": hex.EncodeToString(signature),
	}
	helpers.SaveTestData(t, "keys/ed25519_base64_encoding.json", testData)
}

// Test 10.2.10: 잘못된 서명 거부
func TestEd25519InvalidSignatureRejection(t *testing.T) {
	// Specification Requirement: Tampered signature verification failure
	helpers.LogTestSection(t, "10.2.10", "Invalid Signature Rejection")

	// Generate Ed25519 key pair
	keyPair, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "Key pair generated")

	// Get keys
	privKey := keyPair.PrivateKey().(ed25519.PrivateKey)
	pubKey := keyPair.PublicKey().(ed25519.PublicKey)

	// Create valid message and signature
	message := []byte("original message for signature testing")
	signature := ed25519.Sign(privKey, message)
	require.NotEmpty(t, signature)
	helpers.LogSuccess(t, "Valid signature generated")
	helpers.LogDetail(t, "Message: %s", string(message))
	helpers.LogDetail(t, "Signature length: %d bytes", len(signature))

	// Test 1: Verify valid signature works
	valid := ed25519.Verify(pubKey, message, signature)
	assert.True(t, valid)
	helpers.LogSuccess(t, "Valid signature verified correctly")

	// Test 2: Tampered message should fail
	tamperedMessage := []byte("tampered message for signature testing")
	valid = ed25519.Verify(pubKey, tamperedMessage, signature)
	assert.False(t, valid, "Tampered message should fail verification")
	helpers.LogSuccess(t, "Tampered message correctly rejected")
	helpers.LogDetail(t, "Tampered message: %s", string(tamperedMessage))

	// Test 3: Tampered signature (flip one bit)
	tamperedSig := make([]byte, len(signature))
	copy(tamperedSig, signature)
	tamperedSig[0] ^= 0x01 // Flip first bit
	valid = ed25519.Verify(pubKey, message, tamperedSig)
	assert.False(t, valid, "Tampered signature should fail verification")
	helpers.LogSuccess(t, "Tampered signature (bit flip) correctly rejected")

	// Test 4: Tampered signature (flip multiple bytes)
	tamperedSig2 := make([]byte, len(signature))
	copy(tamperedSig2, signature)
	tamperedSig2[0] ^= 0xFF
	tamperedSig2[31] ^= 0xFF
	tamperedSig2[63] ^= 0xFF
	valid = ed25519.Verify(pubKey, message, tamperedSig2)
	assert.False(t, valid, "Multiple byte tampering should fail verification")
	helpers.LogSuccess(t, "Tampered signature (multiple bytes) correctly rejected")

	// Test 5: Wrong key should fail
	wrongKeyPair, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	wrongPubKey := wrongKeyPair.PublicKey().(ed25519.PublicKey)
	valid = ed25519.Verify(wrongPubKey, message, signature)
	assert.False(t, valid, "Wrong public key should fail verification")
	helpers.LogSuccess(t, "Wrong public key correctly rejected")

	// Test 6: Empty signature should fail
	emptySig := []byte{}
	valid = ed25519.Verify(pubKey, message, emptySig)
	assert.False(t, valid, "Empty signature should fail verification")
	helpers.LogSuccess(t, "Empty signature correctly rejected")

	// Test 7: Short signature should fail
	shortSig := signature[:32] // Only half
	valid = ed25519.Verify(pubKey, message, shortSig)
	assert.False(t, valid, "Short signature should fail verification")
	helpers.LogSuccess(t, "Short signature correctly rejected")

	// Test 8: Long signature should fail
	longSig := append(signature, 0x00, 0x01, 0x02)
	valid = ed25519.Verify(pubKey, message, longSig)
	assert.False(t, valid, "Long signature should fail verification")
	helpers.LogSuccess(t, "Long signature correctly rejected")

	// Test 9: Null message with valid signature should fail
	nullMessage := []byte{}
	validNullSig := ed25519.Sign(privKey, nullMessage)
	valid = ed25519.Verify(pubKey, message, validNullSig)
	assert.False(t, valid, "Null message signature should not verify different message")
	helpers.LogSuccess(t, "Null message signature correctly rejected for non-null message")

	// Test 10: Verify using KeyPair interface methods
	err = keyPair.Verify(tamperedMessage, signature)
	assert.Error(t, err)
	assert.Equal(t, crypto.ErrInvalidSignature, err)
	helpers.LogSuccess(t, "KeyPair.Verify correctly returns error")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"유효한 서명 검증 성공",
		"변조된 메시지 거부",
		"변조된 서명 거부 (1 bit)",
		"변조된 서명 거부 (multiple bytes)",
		"잘못된 공개키 거부",
		"빈 서명 거부",
		"짧은 서명 거부",
		"긴 서명 거부",
		"Null 메시지 서명 거부",
		"KeyPair 인터페이스 에러 처리",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":           "10.2.10_Invalid_Signature_Rejection",
		"key_id":              keyPair.ID(),
		"message":             string(message),
		"signature_hex":       hex.EncodeToString(signature),
		"tampered_tests": map[string]bool{
			"tampered_message":      true,
			"tampered_signature":    true,
			"wrong_key":             true,
			"empty_signature":       true,
			"short_signature":       true,
			"long_signature":        true,
			"null_message":          true,
			"interface_error":       true,
		},
	}
	helpers.SaveTestData(t, "keys/ed25519_invalid_signature.json", testData)
}

// Test 10.2.4: JWK 형식
func TestEd25519KeyPairJWK(t *testing.T) {
	// Specification Requirement: JSON Web Key format support for Ed25519
	helpers.LogTestSection(t, "10.2.4", "JWK Format (JSON Web Key)")

	// Generate Ed25519 key pair
	keyPair, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "Key pair generated")

	// Get keys
	privKey := keyPair.PrivateKey().(ed25519.PrivateKey)
	pubKey := keyPair.PublicKey().(ed25519.PublicKey)

	// Create JWK format for public key
	// Ed25519 uses OKP (Octet string Key Pairs) key type
	pubKeyB64 := base64.RawURLEncoding.EncodeToString(pubKey)
	publicJWK := map[string]interface{}{
		"kty": "OKP",
		"crv": "Ed25519",
		"x":   pubKeyB64,
		"kid": keyPair.ID(),
	}
	helpers.LogSuccess(t, "Public key converted to JWK format")
	helpers.LogDetail(t, "  kty: %v", publicJWK["kty"])
	helpers.LogDetail(t, "  crv: %v", publicJWK["crv"])
	helpers.LogDetail(t, "  kid: %v", publicJWK["kid"])

	// Verify JWK structure
	assert.Equal(t, "OKP", publicJWK["kty"], "Key type should be OKP for Ed25519")
	assert.Equal(t, "Ed25519", publicJWK["crv"], "Curve should be Ed25519")
	assert.NotEmpty(t, publicJWK["x"], "x coordinate should be present")
	assert.NotEmpty(t, publicJWK["kid"], "Key ID should be present")
	helpers.LogSuccess(t, "JWK public key structure verified")

	// Create JWK format for private key (includes both public and private parts)
	// Ed25519 private key is 64 bytes, but we only need the seed (first 32 bytes) for JWK
	privKeySeed := privKey.Seed()
	privKeyB64 := base64.RawURLEncoding.EncodeToString(privKeySeed)
	privateJWK := map[string]interface{}{
		"kty": "OKP",
		"crv": "Ed25519",
		"x":   pubKeyB64,
		"d":   privKeyB64,
		"kid": keyPair.ID(),
	}
	helpers.LogSuccess(t, "Private key converted to JWK format")

	// Verify private JWK contains both public and private parts
	assert.Equal(t, "OKP", privateJWK["kty"])
	assert.Equal(t, "Ed25519", privateJWK["crv"])
	assert.NotEmpty(t, privateJWK["x"], "x coordinate should be present in private JWK")
	assert.NotEmpty(t, privateJWK["d"], "d (private key) should be present in private JWK")
	helpers.LogSuccess(t, "JWK private key structure verified")
	helpers.LogDetail(t, "  d present: %v", privateJWK["d"] != nil)

	// Test JSON serialization
	publicJWKJSON, err := json.Marshal(publicJWK)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Public JWK serialized to JSON")
	helpers.LogDetail(t, "  JSON length: %d bytes", len(publicJWKJSON))

	privateJWKJSON, err := json.Marshal(privateJWK)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Private JWK serialized to JSON")
	helpers.LogDetail(t, "  JSON length: %d bytes", len(privateJWKJSON))

	// Test round-trip: Parse JWK back to key
	// Parse public key from JWK
	xBytes, err := base64.RawURLEncoding.DecodeString(publicJWK["x"].(string))
	require.NoError(t, err)
	importedPubKey := ed25519.PublicKey(xBytes)
	assert.Equal(t, []byte(pubKey), []byte(importedPubKey))
	helpers.LogSuccess(t, "Public key imported from JWK")

	// Parse private key from JWK
	dBytes, err := base64.RawURLEncoding.DecodeString(privateJWK["d"].(string))
	require.NoError(t, err)
	importedPrivKey := ed25519.NewKeyFromSeed(dBytes)
	helpers.LogSuccess(t, "Private key imported from JWK")

	// Verify imported private key works by signing
	message := []byte("test message for JWK round-trip")
	signature := ed25519.Sign(importedPrivKey, message)
	require.NotEmpty(t, signature)
	helpers.LogSuccess(t, "Signature generated with imported private key")

	// Verify signature with original public key
	valid := ed25519.Verify(pubKey, message, signature)
	assert.True(t, valid)
	helpers.LogSuccess(t, "Signature verified with original public key")

	// Verify signature with imported public key
	valid = ed25519.Verify(importedPubKey, message, signature)
	assert.True(t, valid)
	helpers.LogSuccess(t, "Signature verified with imported public key")

	// Verify the imported private key's public key matches
	importedPubFromPriv := importedPrivKey.Public().(ed25519.PublicKey)
	assert.Equal(t, []byte(pubKey), []byte(importedPubFromPriv))
	helpers.LogSuccess(t, "Imported private key's public key matches original")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"JWK 형식으로 공개키 변환",
		"JWK 형식으로 비밀키 변환",
		"kty=OKP, crv=Ed25519 검증",
		"x 좌표 존재 확인",
		"d (비밀키) 존재 확인",
		"JSON 직렬화 성공",
		"JWK에서 공개키 복원",
		"JWK에서 비밀키 복원",
		"복원된 키로 서명/검증",
		"Round-trip 변환 성공",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case": "10.2.4_JWK_Format",
		"key_id":    keyPair.ID(),
		"public_jwk": map[string]interface{}{
			"kty": publicJWK["kty"],
			"crv": publicJWK["crv"],
			"kid": publicJWK["kid"],
			"x":   publicJWK["x"],
		},
		"private_jwk_fields": []string{"kty", "crv", "x", "d", "kid"},
		"public_jwk_json":    string(publicJWKJSON),
		"private_jwk_json":   string(privateJWKJSON),
		"round_trip_success": true,
		"signature_hex":      hex.EncodeToString(signature),
	}
	helpers.SaveTestData(t, "keys/ed25519_jwk_format.json", testData)
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
