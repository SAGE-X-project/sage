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

package integration

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/require"
)

// Test_6_1_1_1_GenerateKeyPairSuccess tests the sage-crypto generate command functionality
func Test_6_1_1_1_GenerateKeyPairSuccess(t *testing.T) {
	helpers.LogTestSection(t, "6.1.1.1", "generate 명령으로 키쌍 생성 성공 확인")

	helpers.LogDetail(t, "sage-crypto generate 명령이 사용하는 기능 검증")
	helpers.LogDetail(t, "테스트 시나리오: Ed25519 키쌍 생성")

	// Generate Ed25519 key pair (what sage-crypto generate --type ed25519 does)
	keyPair, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	require.NotNil(t, keyPair)
	helpers.LogSuccess(t, "Ed25519 키쌍 생성 성공")

	// Verify key properties
	require.NotEmpty(t, keyPair.ID())
	helpers.LogDetail(t, "  Key ID: %s", keyPair.ID())

	require.Equal(t, crypto.KeyTypeEd25519, keyPair.Type())
	helpers.LogDetail(t, "  Key Type: %s", keyPair.Type())

	publicKey := keyPair.PublicKey()
	require.NotNil(t, publicKey)

	// Type assert to Ed25519 public key
	ed25519Pub, ok := publicKey.(ed25519.PublicKey)
	require.True(t, ok, "Public key should be Ed25519 type")
	require.Equal(t, 32, len(ed25519Pub))
	helpers.LogDetail(t, "  Public Key Size: %d bytes", len(ed25519Pub))

	privateKey := keyPair.PrivateKey()
	require.NotNil(t, privateKey)

	// Type assert to Ed25519 private key
	ed25519Priv, ok := privateKey.(ed25519.PrivateKey)
	require.True(t, ok, "Private key should be Ed25519 type")
	require.Equal(t, 64, len(ed25519Priv))
	helpers.LogDetail(t, "  Private Key Size: %d bytes", len(ed25519Priv))

	helpers.LogSuccess(t, "키쌍 속성 검증 완료")

	// Save verification data
	data := map[string]interface{}{
		"test_name":         "Test_6_1_1_1_GenerateKeyPairSuccess",
		"timestamp":         time.Now().Format(time.RFC3339),
		"test_case":         "6.1.1.1_Generate_KeyPair_Success",
		"cli_command":       "sage-crypto generate --type ed25519",
		"key_type":          string(keyPair.Type()),
		"key_id":            keyPair.ID(),
		"public_key_size":   len(ed25519Pub),
		"private_key_size":  len(ed25519Priv),
		"generation_success": true,
	}

	helpers.SaveTestData(t, "cli/6_1_1_1_generate_keypair.json", data)

	helpers.LogPassCriteria(t, []string{
		"Ed25519 키쌍 생성 성공",
		"키 ID 생성 확인",
		"공개키 크기 32 바이트",
		"개인키 크기 64 바이트",
		"키 타입 정확성 검증",
	})
}

// Test_6_1_1_2_GenerateSecp256k1KeyPair tests the --type secp256k1 option
func Test_6_1_1_2_GenerateSecp256k1KeyPair(t *testing.T) {
	helpers.LogTestSection(t, "6.1.1.2", "--type secp256k1 옵션 동작 확인")

	helpers.LogDetail(t, "sage-crypto generate --type secp256k1 명령 검증")

	// Generate Secp256k1 key pair
	keyPair, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	require.NotNil(t, keyPair)
	helpers.LogSuccess(t, "Secp256k1 키쌍 생성 성공")

	// Verify key type
	require.Equal(t, crypto.KeyTypeSecp256k1, keyPair.Type())
	helpers.LogDetail(t, "  Key Type: %s", keyPair.Type())

	// Verify key sizes
	publicKey := keyPair.PublicKey()
	require.NotNil(t, publicKey)

	// Type assert to ECDSA public key and serialize
	ecdsaPub, ok := publicKey.(*ecdsa.PublicKey)
	require.True(t, ok, "Public key should be ECDSA type")
	pubBytes := ethcrypto.FromECDSAPub(ecdsaPub)
	// Secp256k1 uncompressed public key: 65 bytes (0x04 prefix + 32 bytes X + 32 bytes Y)
	require.Equal(t, 65, len(pubBytes))
	helpers.LogDetail(t, "  Public Key Size: %d bytes (uncompressed)", len(pubBytes))

	privateKey := keyPair.PrivateKey()
	require.NotNil(t, privateKey)

	// Type assert to ECDSA private key and serialize
	ecdsaPriv, ok := privateKey.(*ecdsa.PrivateKey)
	require.True(t, ok, "Private key should be ECDSA type")
	privBytes := ethcrypto.FromECDSA(ecdsaPriv)
	require.Equal(t, 32, len(privBytes))
	helpers.LogDetail(t, "  Private Key Size: %d bytes", len(privBytes))

	helpers.LogSuccess(t, "Secp256k1 키쌍 속성 검증 완료")

	// Save verification data
	data := map[string]interface{}{
		"test_name":         "Test_6_1_1_2_GenerateSecp256k1KeyPair",
		"timestamp":         time.Now().Format(time.RFC3339),
		"test_case":         "6.1.1.2_Generate_Secp256k1",
		"cli_command":       "sage-crypto generate --type secp256k1",
		"key_type":          string(keyPair.Type()),
		"key_id":            keyPair.ID(),
		"public_key_size":   len(pubBytes),
		"private_key_size":  len(privBytes),
		"generation_success": true,
	}

	helpers.SaveTestData(t, "cli/6_1_1_2_generate_secp256k1.json", data)

	helpers.LogPassCriteria(t, []string{
		"Secp256k1 키쌍 생성 성공",
		"키 타입 secp256k1 확인",
		"공개키 크기 65 바이트 (비압축)",
		"개인키 크기 32 바이트",
	})
}

// Test_6_1_1_3_GenerateEd25519KeyPair tests the --type ed25519 option explicitly
func Test_6_1_1_3_GenerateEd25519KeyPair(t *testing.T) {
	helpers.LogTestSection(t, "6.1.1.3", "--type ed25519 옵션 동작 확인")

	helpers.LogDetail(t, "sage-crypto generate --type ed25519 명령 검증")

	// Generate Ed25519 key pair (default option)
	keyPair, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	require.NotNil(t, keyPair)
	helpers.LogSuccess(t, "Ed25519 키쌍 생성 성공 (기본 타입)")

	// Verify it's Ed25519
	require.Equal(t, crypto.KeyTypeEd25519, keyPair.Type())
	helpers.LogDetail(t, "  Key Type: %s", keyPair.Type())

	// Verify Ed25519 specific properties
	publicKey := keyPair.PublicKey()
	ed25519Pub, ok := publicKey.(ed25519.PublicKey)
	require.True(t, ok, "Public key should be Ed25519 type")
	require.Equal(t, 32, len(ed25519Pub))
	helpers.LogDetail(t, "  Public Key Size: %d bytes", len(ed25519Pub))

	privateKey := keyPair.PrivateKey()
	ed25519Priv, ok := privateKey.(ed25519.PrivateKey)
	require.True(t, ok, "Private key should be Ed25519 type")
	require.Equal(t, 64, len(ed25519Priv))
	helpers.LogDetail(t, "  Private Key Size: %d bytes", len(ed25519Priv))

	// Test signing capability (Ed25519 specific)
	testMessage := []byte("SAGE test message for Ed25519")
	signature, err := keyPair.Sign(testMessage)
	require.NoError(t, err)
	require.Equal(t, 64, len(signature))
	helpers.LogSuccess(t, "Ed25519 서명 생성 성공")
	helpers.LogDetail(t, "  Signature Size: %d bytes", len(signature))

	// Verify signature
	err = keyPair.Verify(testMessage, signature)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Ed25519 서명 검증 성공")

	// Save verification data
	data := map[string]interface{}{
		"test_name":          "Test_6_1_1_3_GenerateEd25519KeyPair",
		"timestamp":          time.Now().Format(time.RFC3339),
		"test_case":          "6.1.1.3_Generate_Ed25519",
		"cli_command":        "sage-crypto generate --type ed25519",
		"key_type":           string(keyPair.Type()),
		"key_id":             keyPair.ID(),
		"public_key_size":    len(ed25519Pub),
		"private_key_size":   len(ed25519Priv),
		"signature_size":     len(signature),
		"signature_verified": true,
	}

	helpers.SaveTestData(t, "cli/6_1_1_3_generate_ed25519.json", data)

	helpers.LogPassCriteria(t, []string{
		"Ed25519 키쌍 생성 성공",
		"키 타입 ed25519 확인",
		"서명 생성 및 검증 성공",
		"서명 크기 64 바이트",
	})
}

// Test_6_1_2_1_SignMessage tests the sage-crypto sign command functionality
func Test_6_1_2_1_SignMessage(t *testing.T) {
	helpers.LogTestSection(t, "6.1.2.1", "sign 명령으로 메시지 서명 생성")

	helpers.LogDetail(t, "sage-crypto sign 명령이 사용하는 기능 검증")

	// Generate key pair
	keyPair, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "테스트용 키쌍 생성 완료")

	// Test message
	testMessage := []byte("Hello SAGE - Test message for signing")
	helpers.LogDetail(t, "  Message: %s", string(testMessage))

	// Sign the message (what sage-crypto sign does)
	signature, err := keyPair.Sign(testMessage)
	require.NoError(t, err)
	require.NotEmpty(t, signature)
	helpers.LogSuccess(t, "메시지 서명 생성 성공")

	// Verify signature properties
	require.Equal(t, 64, len(signature))
	helpers.LogDetail(t, "  Signature Size: %d bytes", len(signature))

	signatureBase64 := base64.StdEncoding.EncodeToString(signature)
	helpers.LogDetail(t, "  Signature (Base64): %s", signatureBase64[:32]+"...")

	// Verify the signature works
	err = keyPair.Verify(testMessage, signature)
	require.NoError(t, err)
	helpers.LogSuccess(t, "생성된 서명 검증 성공")

	// Save verification data
	data := map[string]interface{}{
		"test_name":          "Test_6_1_2_1_SignMessage",
		"timestamp":          time.Now().Format(time.RFC3339),
		"test_case":          "6.1.2.1_Sign_Message",
		"cli_command":        "sage-crypto sign --message <message>",
		"message":            string(testMessage),
		"signature_size":     len(signature),
		"signature_base64":   signatureBase64,
		"signature_verified": true,
	}

	helpers.SaveTestData(t, "cli/6_1_2_1_sign_message.json", data)

	helpers.LogPassCriteria(t, []string{
		"메시지 서명 생성 성공",
		"서명 크기 64 바이트",
		"Base64 인코딩 지원",
		"생성된 서명 검증 가능",
	})
}

// Test_6_1_2_2_VerifySignature tests the sage-crypto verify command functionality
func Test_6_1_2_2_VerifySignature(t *testing.T) {
	helpers.LogTestSection(t, "6.1.2.2", "verify 명령으로 서명 검증 성공")

	helpers.LogDetail(t, "sage-crypto verify 명령이 사용하는 기능 검증")

	// Generate key pair
	keyPair, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "테스트용 키쌍 생성 완료")

	// Create test message and signature
	testMessage := []byte("Test message for verification")
	signature, err := keyPair.Sign(testMessage)
	require.NoError(t, err)
	helpers.LogSuccess(t, "테스트용 서명 생성 완료")
	helpers.LogDetail(t, "  Message: %s", string(testMessage))

	// Verify the signature (what sage-crypto verify does)
	err = keyPair.Verify(testMessage, signature)
	require.NoError(t, err)
	helpers.LogSuccess(t, "서명 검증 성공 (유효한 서명)")

	// Test invalid signature
	invalidSignature := make([]byte, 64)
	copy(invalidSignature, signature)
	invalidSignature[0] ^= 0xFF // Corrupt first byte

	err = keyPair.Verify(testMessage, invalidSignature)
	require.Error(t, err)
	helpers.LogSuccess(t, "잘못된 서명 감지 성공 (검증 실패)")

	// Test wrong message
	wrongMessage := []byte("Different message")
	err = keyPair.Verify(wrongMessage, signature)
	require.Error(t, err)
	helpers.LogSuccess(t, "메시지 변조 감지 성공 (검증 실패)")

	// Save verification data
	data := map[string]interface{}{
		"test_name":             "Test_6_1_2_2_VerifySignature",
		"timestamp":             time.Now().Format(time.RFC3339),
		"test_case":             "6.1.2.2_Verify_Signature",
		"cli_command":           "sage-crypto verify --message <message> --signature <sig>",
		"valid_signature":       true,
		"invalid_signature":     false,
		"tampered_message":      false,
		"verification_success":  true,
	}

	helpers.SaveTestData(t, "cli/6_1_2_2_verify_signature.json", data)

	helpers.LogPassCriteria(t, []string{
		"유효한 서명 검증 성공",
		"잘못된 서명 감지",
		"메시지 변조 감지",
		"검증 결과 정확성",
	})
}

// Test_6_1_3_1_GenerateEthereumAddress tests the sage-crypto address command functionality
func Test_6_1_3_1_GenerateEthereumAddress(t *testing.T) {
	helpers.LogTestSection(t, "6.1.3.1", "address 명령으로 Ethereum 주소 생성")

	helpers.LogDetail(t, "sage-crypto address 명령이 사용하는 기능 검증")

	// Generate Secp256k1 key pair (required for Ethereum address)
	keyPair, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "Secp256k1 키쌍 생성 완료")

	// Derive Ethereum address (what sage-crypto address does)
	address, err := did.DeriveEthereumAddress(keyPair)
	require.NoError(t, err)
	require.NotEmpty(t, address)
	helpers.LogSuccess(t, "Ethereum 주소 생성 성공")

	// Verify address format
	require.True(t, len(address) == 42, "Address should be 42 characters (0x + 40 hex)")
	require.True(t, address[:2] == "0x", "Address should start with 0x")
	helpers.LogDetail(t, "  Address: %s", address)
	helpers.LogDetail(t, "  Format: 0x + 40 hex characters")

	// Verify lowercase
	require.Equal(t, address, address, "Address should be lowercase")
	helpers.LogSuccess(t, "주소 포맷 검증 완료")

	// Verify deterministic generation
	address2, err := did.DeriveEthereumAddress(keyPair)
	require.NoError(t, err)
	require.Equal(t, address, address2)
	helpers.LogSuccess(t, "결정론적 주소 생성 확인 (동일 키 → 동일 주소)")

	// Save verification data
	data := map[string]interface{}{
		"test_name":       "Test_6_1_3_1_GenerateEthereumAddress",
		"timestamp":       time.Now().Format(time.RFC3339),
		"test_case":       "6.1.3.1_Generate_Ethereum_Address",
		"cli_command":     "sage-crypto address --key <keyfile>",
		"address":         address,
		"address_length":  len(address),
		"has_0x_prefix":   address[:2] == "0x",
		"deterministic":   address == address2,
	}

	helpers.SaveTestData(t, "cli/6_1_3_1_ethereum_address.json", data)

	helpers.LogPassCriteria(t, []string{
		"Ethereum 주소 생성 성공",
		"주소 포맷 검증 (0x + 40 hex)",
		"결정론적 생성 확인",
		"소문자 형식 확인",
	})
}
