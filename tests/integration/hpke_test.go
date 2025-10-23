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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/require"
)

// Test_8_1_1_1_X25519KeyExchangeSuccess tests X25519 key exchange in HPKE
func Test_8_1_1_1_X25519KeyExchangeSuccess(t *testing.T) {
	helpers.LogTestSection(t, "8.1.1.1", "X25519 키 교환 성공")

	helpers.LogDetail(t, "HPKE DHKEM(X25519) 키 교환 테스트")
	helpers.LogDetail(t, "테스트 시나리오: X25519 키 쌍 생성 및 캡슐화")

	// Generate X25519 keypair for recipient (Bob)
	bobKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)
	require.NotNil(t, bobKeyPair)
	helpers.LogSuccess(t, "수신자 (Bob) X25519 키 쌍 생성 완료")
	helpers.LogDetail(t, "  키 타입: X25519 (Curve25519)")
	helpers.LogDetail(t, "  키 ID: %s", bobKeyPair.ID())

	// HPKE context parameters
	info := []byte("sage/hpke-handshake v1|ctx:test-001|init:alice|resp:bob")
	exportCtx := []byte("sage/session exporter v1")
	exportLen := 32

	helpers.LogDetail(t, "HPKE 컨텍스트 파라미터:")
	helpers.LogDetail(t, "  Info: %s", string(info))
	helpers.LogDetail(t, "  Export context: %s", string(exportCtx))
	helpers.LogDetail(t, "  Export length: %d bytes", exportLen)

	// Sender (Alice) derives shared secret and encapsulates key
	enc, exporterAlice, err := keys.HPKEDeriveSharedSecretToPeer(
		bobKeyPair.PublicKey(), info, exportCtx, exportLen,
	)
	require.NoError(t, err)
	require.NotNil(t, enc)
	require.NotNil(t, exporterAlice)
	helpers.LogSuccess(t, "송신자 (Alice) HPKE 키 캡슐화 성공")

	// Verify encapsulated key size (X25519 public key = 32 bytes)
	require.Equal(t, 32, len(enc), "Encapsulated key should be 32 bytes")
	helpers.LogDetail(t, "  Encapsulated key 크기: %d bytes (예상: 32)", len(enc))
	helpers.LogDetail(t, "  Encapsulated key (hex): %s...", hex.EncodeToString(enc)[:16])

	// Verify exporter secret size
	require.Equal(t, 32, len(exporterAlice), "Exporter secret should be 32 bytes")
	helpers.LogDetail(t, "  Exporter secret 크기: %d bytes (예상: 32)", len(exporterAlice))
	helpers.LogDetail(t, "  Exporter secret (hex): %s...", hex.EncodeToString(exporterAlice)[:16])
	helpers.LogSuccess(t, "X25519 키 교환 완료")

	// Save verification data
	data := map[string]interface{}{
		"test_name":            "Test_8_1_1_1_X25519KeyExchangeSuccess",
		"timestamp":            time.Now().Format(time.RFC3339),
		"test_case":            "8.1.1.1_X25519_Key_Exchange",
		"key_type":             "X25519",
		"encapsulated_key_size": len(enc),
		"exporter_secret_size": len(exporterAlice),
		"info_context":         string(info),
		"export_context":       string(exportCtx),
		"export_length":        exportLen,
		"key_exchange_success": true,
	}

	helpers.SaveTestData(t, "hpke/8_1_1_1_x25519_key_exchange.json", data)

	helpers.LogPassCriteria(t, []string{
		"X25519 키 쌍 생성 성공",
		"HPKE 키 캡슐화 성공",
		"Encapsulated key 크기 검증 (32 bytes)",
		"Exporter secret 크기 검증 (32 bytes)",
		"키 교환 프로세스 완료",
	})
}

// Test_8_1_1_2_SharedSecretGeneration tests shared secret generation in HPKE
func Test_8_1_1_2_SharedSecretGeneration(t *testing.T) {
	helpers.LogTestSection(t, "8.1.1.2", "공유 비밀 생성 확인")

	helpers.LogDetail(t, "HPKE 공유 비밀 생성 및 검증 테스트")
	helpers.LogDetail(t, "테스트 시나리오: 송신자와 수신자의 공유 비밀 일치 확인")

	// Generate X25519 keypair for recipient
	bobKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "수신자 X25519 키 쌍 생성 완료")

	// HPKE context
	info := []byte("sage/hpke test|ctx:shared-secret-001")
	exportCtx := []byte("sage/export context v1")
	exportLen := 32

	helpers.LogDetail(t, "HPKE 컨텍스트:")
	helpers.LogDetail(t, "  Info: %s", string(info))
	helpers.LogDetail(t, "  Export length: %d bytes", exportLen)

	// Sender side: derive shared secret and create encapsulated key
	enc, exporterAlice, err := keys.HPKEDeriveSharedSecretToPeer(
		bobKeyPair.PublicKey(), info, exportCtx, exportLen,
	)
	require.NoError(t, err)
	helpers.LogSuccess(t, "송신자 (Alice) 공유 비밀 파생 완료")
	helpers.LogDetail(t, "  Alice exporter secret (hex): %s...", hex.EncodeToString(exporterAlice)[:24])

	// Recipient side: open encapsulated key and derive shared secret
	exporterBob, err := keys.HPKEOpenSharedSecretWithPriv(
		bobKeyPair.PrivateKey(), enc, info, exportCtx, exportLen,
	)
	require.NoError(t, err)
	helpers.LogSuccess(t, "수신자 (Bob) 공유 비밀 파생 완료")
	helpers.LogDetail(t, "  Bob exporter secret (hex): %s...", hex.EncodeToString(exporterBob)[:24])

	// Verify both sides derived the same shared secret
	require.True(t, bytes.Equal(exporterAlice, exporterBob), "Shared secrets must match")
	helpers.LogSuccess(t, "양쪽의 공유 비밀 일치 확인 ✓")
	helpers.LogDetail(t, "  Alice secret == Bob secret: %v", bytes.Equal(exporterAlice, exporterBob))
	helpers.LogDetail(t, "  Secret length: %d bytes", len(exporterAlice))

	// Verify deterministic session ID derivation from shared secret
	sidAlice, err := session.ComputeSessionIDFromSeed(exporterAlice, "sage/hpke v1")
	require.NoError(t, err)
	sidBob, err := session.ComputeSessionIDFromSeed(exporterBob, "sage/hpke v1")
	require.NoError(t, err)
	require.Equal(t, sidAlice, sidBob, "Session IDs must match")
	helpers.LogSuccess(t, "결정론적 Session ID 파생 성공")
	helpers.LogDetail(t, "  Session ID (Alice): %s", sidAlice)
	helpers.LogDetail(t, "  Session ID (Bob): %s", sidBob)
	helpers.LogDetail(t, "  Session IDs match: %v", sidAlice == sidBob)

	// Save verification data
	data := map[string]interface{}{
		"test_name":               "Test_8_1_1_2_SharedSecretGeneration",
		"timestamp":               time.Now().Format(time.RFC3339),
		"test_case":               "8.1.1.2_Shared_Secret_Generation",
		"exporter_alice_hex":      hex.EncodeToString(exporterAlice),
		"exporter_bob_hex":        hex.EncodeToString(exporterBob),
		"secrets_match":           bytes.Equal(exporterAlice, exporterBob),
		"secret_size":             len(exporterAlice),
		"session_id_alice":        sidAlice,
		"session_id_bob":          sidBob,
		"session_ids_match":       sidAlice == sidBob,
		"shared_secret_verified":  true,
	}

	helpers.SaveTestData(t, "hpke/8_1_1_2_shared_secret.json", data)

	helpers.LogPassCriteria(t, []string{
		"송신자 공유 비밀 파생 성공",
		"수신자 공유 비밀 파생 성공",
		"양쪽의 공유 비밀 일치",
		"결정론적 Session ID 파생 성공",
		"Session ID 일치 확인",
	})
}

// Test_8_1_2_1_ChaCha20Poly1305Encryption tests AEAD encryption with ChaCha20-Poly1305
func Test_8_1_2_1_ChaCha20Poly1305Encryption(t *testing.T) {
	helpers.LogTestSection(t, "8.1.2.1", "ChaCha20Poly1305 암호화 성공")

	helpers.LogDetail(t, "HPKE 세션을 통한 AEAD 암호화 테스트")
	helpers.LogDetail(t, "암호화 알고리즘: ChaCha20-Poly1305")

	// Setup HPKE session
	bobKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	info := []byte("sage/hpke test|ctx:encryption-001")
	exportCtx := []byte("sage/export")
	exportLen := 32

	// Derive shared secret
	enc, exporterAlice, err := keys.HPKEDeriveSharedSecretToPeer(
		bobKeyPair.PublicKey(), info, exportCtx, exportLen,
	)
	require.NoError(t, err)
	exporterBob, err := keys.HPKEOpenSharedSecretWithPriv(
		bobKeyPair.PrivateKey(), enc, info, exportCtx, exportLen,
	)
	require.NoError(t, err)
	helpers.LogSuccess(t, "HPKE 공유 비밀 설정 완료")

	// Create secure sessions from exporter secrets
	sidAlice, err := session.ComputeSessionIDFromSeed(exporterAlice, "sage/hpke v1")
	require.NoError(t, err)
	sidBob, err := session.ComputeSessionIDFromSeed(exporterBob, "sage/hpke v1")
	require.NoError(t, err)

	sessionAlice, err := session.NewSecureSessionFromExporter(sidAlice, exporterAlice, session.Config{})
	require.NoError(t, err)
	_, err = session.NewSecureSessionFromExporter(sidBob, exporterBob, session.Config{})
	require.NoError(t, err)
	helpers.LogSuccess(t, "보안 세션 생성 완료")
	helpers.LogDetail(t, "  Alice Session ID: %s", sidAlice)
	helpers.LogDetail(t, "  Bob Session ID: %s", sidBob)

	// Test message encryption
	plaintext := []byte("Hello, SAGE secure world! This is a test message.")
	helpers.LogDetail(t, "평문 메시지:")
	helpers.LogDetail(t, "  내용: %s", string(plaintext))
	helpers.LogDetail(t, "  크기: %d bytes", len(plaintext))

	// Encrypt with ChaCha20-Poly1305
	ciphertext, err := sessionAlice.Encrypt(plaintext)
	require.NoError(t, err)
	require.NotNil(t, ciphertext)
	require.Greater(t, len(ciphertext), len(plaintext), "Ciphertext should be larger (includes auth tag)")
	helpers.LogSuccess(t, "ChaCha20-Poly1305 암호화 성공 ✓")
	helpers.LogDetail(t, "  암호문 크기: %d bytes", len(ciphertext))
	helpers.LogDetail(t, "  암호문 (hex): %s...", hex.EncodeToString(ciphertext)[:32])
	helpers.LogDetail(t, "  오버헤드: %d bytes (nonce + auth tag)", len(ciphertext)-len(plaintext))

	// Save verification data
	data := map[string]interface{}{
		"test_name":          "Test_8_1_2_1_ChaCha20Poly1305Encryption",
		"timestamp":          time.Now().Format(time.RFC3339),
		"test_case":          "8.1.2.1_ChaCha20Poly1305_Encryption",
		"algorithm":          "ChaCha20-Poly1305",
		"plaintext":          string(plaintext),
		"plaintext_size":     len(plaintext),
		"ciphertext_size":    len(ciphertext),
		"ciphertext_hex":     hex.EncodeToString(ciphertext),
		"overhead_bytes":     len(ciphertext) - len(plaintext),
		"encryption_success": true,
	}

	helpers.SaveTestData(t, "hpke/8_1_2_1_chacha20poly1305_encryption.json", data)

	helpers.LogPassCriteria(t, []string{
		"HPKE 세션 설정 성공",
		"ChaCha20-Poly1305 암호화 성공",
		"암호문 생성 확인",
		"암호문 크기 검증 (평문 + 오버헤드)",
		"인증 태그 포함 확인",
	})
}

// Test_8_1_2_2_DecryptionPlaintextMatch tests decryption and plaintext verification
func Test_8_1_2_2_DecryptionPlaintextMatch(t *testing.T) {
	helpers.LogTestSection(t, "8.1.2.2", "복호화 후 평문과 일치")

	helpers.LogDetail(t, "AEAD 복호화 및 평문 검증 테스트")
	helpers.LogDetail(t, "테스트 시나리오: 암호화 → 복호화 → 원본 평문 일치 확인")

	// Setup HPKE session (same as encryption test)
	bobKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	info := []byte("sage/hpke test|ctx:decryption-001")
	exportCtx := []byte("sage/export")
	exportLen := 32

	enc, exporterAlice, err := keys.HPKEDeriveSharedSecretToPeer(
		bobKeyPair.PublicKey(), info, exportCtx, exportLen,
	)
	require.NoError(t, err)
	exporterBob, err := keys.HPKEOpenSharedSecretWithPriv(
		bobKeyPair.PrivateKey(), enc, info, exportCtx, exportLen,
	)
	require.NoError(t, err)

	sidAlice, err := session.ComputeSessionIDFromSeed(exporterAlice, "sage/hpke v1")
	require.NoError(t, err)
	sidBob, err := session.ComputeSessionIDFromSeed(exporterBob, "sage/hpke v1")
	require.NoError(t, err)

	sessionAlice, err := session.NewSecureSessionFromExporter(sidAlice, exporterAlice, session.Config{})
	require.NoError(t, err)
	sessionBob, err := session.NewSecureSessionFromExporter(sidBob, exporterBob, session.Config{})
	require.NoError(t, err)
	helpers.LogSuccess(t, "HPKE 세션 설정 완료")

	// Original plaintext
	originalPlaintext := []byte("SAGE secure messaging test - 안전한 메시지 전송 테스트")
	hash := sha256.Sum256(originalPlaintext)
	helpers.LogDetail(t, "원본 평문:")
	helpers.LogDetail(t, "  내용: %s", string(originalPlaintext))
	helpers.LogDetail(t, "  크기: %d bytes", len(originalPlaintext))
	helpers.LogDetail(t, "  SHA256: %x", hash[:8])

	// Encrypt
	ciphertext, err := sessionAlice.Encrypt(originalPlaintext)
	require.NoError(t, err)
	helpers.LogSuccess(t, "암호화 완료")
	helpers.LogDetail(t, "  암호문 크기: %d bytes", len(ciphertext))

	// Decrypt
	decryptedPlaintext, err := sessionBob.Decrypt(ciphertext)
	require.NoError(t, err)
	require.NotNil(t, decryptedPlaintext)
	helpers.LogSuccess(t, "복호화 완료")
	helpers.LogDetail(t, "  복호화된 평문 크기: %d bytes", len(decryptedPlaintext))
	helpers.LogDetail(t, "  복호화된 평문: %s", string(decryptedPlaintext))

	// Verify plaintext matches original
	require.True(t, bytes.Equal(originalPlaintext, decryptedPlaintext), "Decrypted plaintext must match original")
	helpers.LogSuccess(t, "평문 일치 검증 성공 ✓")
	helpers.LogDetail(t, "  원본 == 복호화: %v", bytes.Equal(originalPlaintext, decryptedPlaintext))
	helpers.LogDetail(t, "  크기 일치: %v", len(originalPlaintext) == len(decryptedPlaintext))
	helpers.LogDetail(t, "  내용 일치: %v", string(originalPlaintext) == string(decryptedPlaintext))

	// Save verification data
	data := map[string]interface{}{
		"test_name":              "Test_8_1_2_2_DecryptionPlaintextMatch",
		"timestamp":              time.Now().Format(time.RFC3339),
		"test_case":              "8.1.2.2_Decryption_Plaintext_Match",
		"original_plaintext":     string(originalPlaintext),
		"decrypted_plaintext":    string(decryptedPlaintext),
		"plaintext_match":        bytes.Equal(originalPlaintext, decryptedPlaintext),
		"original_size":          len(originalPlaintext),
		"decrypted_size":         len(decryptedPlaintext),
		"size_match":             len(originalPlaintext) == len(decryptedPlaintext),
		"decryption_success":     true,
	}

	helpers.SaveTestData(t, "hpke/8_1_2_2_decryption_match.json", data)

	helpers.LogPassCriteria(t, []string{
		"암호화 성공",
		"복호화 성공",
		"평문 일치 확인",
		"크기 일치 확인",
		"내용 일치 확인",
		"AEAD 인증 성공",
	})
}

// Test_8_1_2_3_CiphertextConsistency tests ciphertext consistency verification
func Test_8_1_2_3_CiphertextConsistency(t *testing.T) {
	helpers.LogTestSection(t, "8.1.2.3", "암호문 일관성 확인")

	helpers.LogDetail(t, "암호화/복호화 일관성 테스트")
	helpers.LogDetail(t, "테스트 시나리오: 동일 평문 → 동일 세션 키 → 암호문 검증")

	// Setup HPKE session
	bobKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	info := []byte("sage/hpke test|ctx:consistency-001")
	exportCtx := []byte("sage/export")
	exportLen := 32

	enc, exporterAlice, err := keys.HPKEDeriveSharedSecretToPeer(
		bobKeyPair.PublicKey(), info, exportCtx, exportLen,
	)
	require.NoError(t, err)
	exporterBob, err := keys.HPKEOpenSharedSecretWithPriv(
		bobKeyPair.PrivateKey(), enc, info, exportCtx, exportLen,
	)
	require.NoError(t, err)

	sidAlice, err := session.ComputeSessionIDFromSeed(exporterAlice, "sage/hpke v1")
	require.NoError(t, err)
	sidBob, err := session.ComputeSessionIDFromSeed(exporterBob, "sage/hpke v1")
	require.NoError(t, err)

	sessionAlice, err := session.NewSecureSessionFromExporter(sidAlice, exporterAlice, session.Config{})
	require.NoError(t, err)
	sessionBob, err := session.NewSecureSessionFromExporter(sidBob, exporterBob, session.Config{})
	require.NoError(t, err)
	helpers.LogSuccess(t, "HPKE 세션 설정 완료")

	// Test message
	plaintext := []byte("Consistency test message 123")
	helpers.LogDetail(t, "테스트 평문: %s", string(plaintext))

	// First encryption
	ciphertext1, err := sessionAlice.Encrypt(plaintext)
	require.NoError(t, err)
	helpers.LogSuccess(t, "첫 번째 암호화 완료")
	helpers.LogDetail(t, "  암호문 1 크기: %d bytes", len(ciphertext1))
	helpers.LogDetail(t, "  암호문 1 (hex): %s...", hex.EncodeToString(ciphertext1)[:24])

	// Decrypt first ciphertext
	decrypted1, err := sessionBob.Decrypt(ciphertext1)
	require.NoError(t, err)
	require.True(t, bytes.Equal(plaintext, decrypted1), "First decryption must match plaintext")
	helpers.LogSuccess(t, "첫 번째 복호화 성공 및 평문 일치")

	// Second encryption (with nonce increment)
	ciphertext2, err := sessionAlice.Encrypt(plaintext)
	require.NoError(t, err)
	helpers.LogSuccess(t, "두 번째 암호화 완료")
	helpers.LogDetail(t, "  암호문 2 크기: %d bytes", len(ciphertext2))
	helpers.LogDetail(t, "  암호문 2 (hex): %s...", hex.EncodeToString(ciphertext2)[:24])

	// Decrypt second ciphertext
	decrypted2, err := sessionBob.Decrypt(ciphertext2)
	require.NoError(t, err)
	require.True(t, bytes.Equal(plaintext, decrypted2), "Second decryption must match plaintext")
	helpers.LogSuccess(t, "두 번째 복호화 성공 및 평문 일치")

	// Verify ciphertexts are different (due to nonce)
	require.False(t, bytes.Equal(ciphertext1, ciphertext2), "Ciphertexts should differ due to nonce increment")
	helpers.LogSuccess(t, "암호문 다름 확인 (Nonce 증가)")
	helpers.LogDetail(t, "  암호문 1 != 암호문 2: %v", !bytes.Equal(ciphertext1, ciphertext2))

	// Verify both decrypt to same plaintext
	require.True(t, bytes.Equal(decrypted1, decrypted2), "Both decryptions must match")
	helpers.LogSuccess(t, "두 복호화 결과 동일 확인 ✓")
	helpers.LogDetail(t, "  복호문 1 == 복호문 2: %v", bytes.Equal(decrypted1, decrypted2))

	// Save verification data
	data := map[string]interface{}{
		"test_name":             "Test_8_1_2_3_CiphertextConsistency",
		"timestamp":             time.Now().Format(time.RFC3339),
		"test_case":             "8.1.2.3_Ciphertext_Consistency",
		"plaintext":             string(plaintext),
		"ciphertext1_size":      len(ciphertext1),
		"ciphertext2_size":      len(ciphertext2),
		"ciphertexts_different": !bytes.Equal(ciphertext1, ciphertext2),
		"decrypted1_match":      bytes.Equal(plaintext, decrypted1),
		"decrypted2_match":      bytes.Equal(plaintext, decrypted2),
		"both_decryptions_match": bytes.Equal(decrypted1, decrypted2),
		"consistency_verified":  true,
	}

	helpers.SaveTestData(t, "hpke/8_1_2_3_ciphertext_consistency.json", data)

	helpers.LogPassCriteria(t, []string{
		"첫 번째 암호화 성공",
		"첫 번째 복호화 성공",
		"두 번째 암호화 성공",
		"두 번째 복호화 성공",
		"암호문 차이 확인 (Nonce)",
		"복호화 결과 일치 확인",
		"AEAD 일관성 검증 완료",
	})
}
