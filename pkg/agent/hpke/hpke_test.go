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
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
)

func Test_HPKE_Base_Exporter_To_Session(t *testing.T) {
	// 사양 요구사항: 키 파생을 통한 HPKE 기반 보안 세션 설정
	helpers.LogTestSection(t, "6.1.1", "HPKE 키 교환 및 세션 파생")

	// 사양 요구사항: KEM(키 캡슐화 메커니즘)을 위한 X25519 키 생성
	bobKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "수신자 (Bob) X25519 키 쌍 생성 성공")
	helpers.LogDetail(t, "키 타입: X25519 (Curve25519)")

	// 정규 HPKE 컨텍스트 (양쪽이 반드시 일치해야 함)
	info := []byte("sage/hpke-handshake v1|ctx:ctx-001|init:did:alice|resp:did:bob")
	exportCtx := []byte("sage/session exporter v1")
	exportLen := 32

	helpers.LogDetail(t, "HPKE info context: %s", string(info))
	helpers.LogDetail(t, "Export context: %s", string(exportCtx))
	helpers.LogDetail(t, "Export length: %d bytes", exportLen)

	// 사양 요구사항: 송신자 (Alice)가 캡슐화된 키와 공유 비밀 파생
	enc, expA, err := keys.HPKEDeriveSharedSecretToPeer(
		bobKeyPair.PublicKey(), info, exportCtx, exportLen,
	)
	require.NoError(t, err)

	// 사양 요구사항: 캡슐화된 키 크기는 32 bytes (X25519 공개키)여야 함
	require.Equal(t, 32, len(enc))
	assert.Equal(t, 32, len(expA))

	helpers.LogSuccess(t, "송신자 (Alice) HPKE 키 파생 성공")
	helpers.LogDetail(t, "Encapsulated key: %d bytes (예상값: 32)", len(enc))
	helpers.LogDetail(t, "Exporter secret: %d bytes (예상값: 32)", len(expA))
	helpers.LogDetail(t, "Encapsulated key (hex): %s", hex.EncodeToString(enc)[:32]+"...")

	// 사양 요구사항: 수신자 (Bob)가 개인키로 개봉하고 공유 비밀 파생
	expB, err := keys.HPKEOpenSharedSecretWithPriv(
		bobKeyPair.PrivateKey(), enc, info, exportCtx, exportLen,
	)
	require.NoError(t, err)

	// 사양 요구사항: 양쪽이 동일한 공유 비밀을 파생해야 함
	require.True(t, bytes.Equal(expA, expB), "exporter 불일치")
	helpers.LogSuccess(t, "수신자 (Bob) HPKE 키 개봉 성공")
	helpers.LogDetail(t, "Shared secret 일치: %v", bytes.Equal(expA, expB))
	helpers.LogDetail(t, "Exporter A (hex): %s", hex.EncodeToString(expA)[:32]+"...")
	helpers.LogDetail(t, "Exporter B (hex): %s", hex.EncodeToString(expB)[:32]+"...")

	// 사양 요구사항: 공유 비밀로부터 결정적 세션 ID 파생
	sidA, err := session.ComputeSessionIDFromSeed(expA, "sage/hpke v1")
	require.NoError(t, err)
	sidB, err := session.ComputeSessionIDFromSeed(expB, "sage/hpke v1")
	require.NoError(t, err)
	require.Equal(t, sidA, sidB, "session id 불일치")

	helpers.LogSuccess(t, "Session ID 결정적 파생")
	helpers.LogDetail(t, "Session ID (Alice): %s", sidA)
	helpers.LogDetail(t, "Session ID (Bob): %s", sidB)
	helpers.LogDetail(t, "Session ID 일치: %v", sidA == sidB)

	// 사양 요구사항: exporter로부터 보안 세션 구성
	sA, err := session.NewSecureSessionFromExporter(sidA, expA, session.Config{})
	require.NoError(t, err)
	sB, err := session.NewSecureSessionFromExporter(sidB, expB, session.Config{})
	require.NoError(t, err)

	helpers.LogSuccess(t, "HPKE exporter로부터 보안 세션 설정 완료")
	helpers.LogDetail(t, "Alice 세션 생성, ID: %s", sidA)
	helpers.LogDetail(t, "Bob 세션 생성, ID: %s", sidB)

	// 사양 요구사항: 파생된 세션 키로 AEAD 암호화/복호화
	msg := []byte("hello, secure world")
	helpers.LogDetail(t, "테스트 메시지: %s", string(msg))
	helpers.LogDetail(t, "메시지 크기: %d bytes", len(msg))

	ct, err := sA.Encrypt(msg)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Alice 메시지 암호화 성공")
	helpers.LogDetail(t, "암호문 크기: %d bytes", len(ct))
	helpers.LogDetail(t, "암호문 (hex): %s", hex.EncodeToString(ct)[:64]+"...")

	pt, err := sB.Decrypt(ct)
	require.NoError(t, err)
	require.True(t, bytes.Equal(pt, msg), "평문 불일치")
	helpers.LogSuccess(t, "Bob 메시지 복호화 성공")
	helpers.LogDetail(t, "복호화된 메시지: %s", string(pt))
	helpers.LogDetail(t, "평문 일치: %v", bytes.Equal(pt, msg))

	// 사양 요구사항: RFC 9421 스타일 covered 서명
	covered := []byte("@method:POST\n@path:/protected\nhost:example.org\ndate:Mon, 01 Jan 2024 00:00:00 GMT\ncontent-digest:sha-256=:...:\n")
	helpers.LogDetail(t, "Covered content 크기: %d bytes", len(covered))

	sig := sA.SignCovered(covered)
	helpers.LogSuccess(t, "Alice covered 서명 생성 성공")
	helpers.LogDetail(t, "서명 크기: %d bytes", len(sig))

	require.NoError(t, sB.VerifyCovered(covered, sig))
	helpers.LogSuccess(t, "Bob covered 서명 검증 성공")

	// 통과 기준 체크리스트
	helpers.LogPassCriteria(t, []string{
		"X25519 키 쌍 생성 성공",
		"HPKE 키 파생 (Alice) 성공",
		"Encapsulated key = 32 bytes",
		"Shared secret = 32 bytes",
		"HPKE 키 개봉 (Bob) 성공",
		"양쪽의 Shared secret 일치",
		"Session ID 결정적 파생",
		"양쪽의 Session ID 일치",
		"보안 세션 설정 완료",
		"AEAD 암호화 성공",
		"AEAD 복호화 성공",
		"평문이 원본과 일치",
		"Covered 서명 생성 성공",
		"Covered 서명 검증 성공",
	})

	// Save test data for CLI verification
	testData := map[string]interface{}{
		"test_case": "6.1.1_HPKE_Key_Exchange_Session",
		"key_type":  "X25519",
		"hpke": map[string]interface{}{
			"info_context":   string(info),
			"export_context": string(exportCtx),
			"export_length":  exportLen,
			"enc_size":       len(enc),
			"exporter_a_hex": hex.EncodeToString(expA),
			"exporter_b_hex": hex.EncodeToString(expB),
			"secrets_match":  bytes.Equal(expA, expB),
		},
		"session": map[string]interface{}{
			"session_id_a": sidA,
			"session_id_b": sidB,
			"ids_match":    sidA == sidB,
		},
		"encryption": map[string]interface{}{
			"message":         string(msg),
			"message_size":    len(msg),
			"ciphertext_size": len(ct),
			"decrypted":       string(pt),
			"plaintext_match": bytes.Equal(pt, msg),
		},
		"signature": map[string]interface{}{
			"covered_size":   len(covered),
			"signature_size": len(sig),
			"verification":   "successful",
		},
	}
	helpers.SaveTestData(t, "hpke/hpke_key_exchange_session.json", testData)
}
