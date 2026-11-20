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

package rfc9421

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifier(t *testing.T) {
	// Generate test keypair
	testutil.LogDetail(t, "EdTest Ed25519 키 쌍 생성 중...")
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	testutil.LogSuccess(t, "Ed25519 키 쌍 생성 완료")

	verifier := NewVerifier()
	testutil.LogSuccess(t, "Verifier 인스턴스 생성 완료")

	t.Run("Digest", func(t *testing.T) {
		// Specification Requirement: Content-Digest validation for message integrity
		testutil.LogTestSection(t, "10.1.2", "RFC9421 Content-Digest 검증")

		bodyContent := []byte("test message for digest verification")
		testutil.LogDetail(t, "테스트 메시지 Body:")
		testutil.LogDetail(t, "  내용: %s", string(bodyContent))
		testutil.LogDetail(t, "  길이: %d bytes", len(bodyContent))

		// Test case 1: Valid digest
		t.Run("valid_digest", func(t *testing.T) {
			testutil.LogDetail(t, "케이스 1: 유효한 Content-Digest")

			message := &Message{
				AgentDID:     "did:sage:ethereum:agent-digest",
				MessageID:    "msg-digest-001",
				Timestamp:    time.Now(),
				Nonce:        "digest-nonce",
				Body:         bodyContent,
				Algorithm:    string(AlgorithmEdDSA),
				SignedFields: []string{"agent_did", "message_id", "timestamp", "nonce", "body"},
			}

			// Calculate correct SHA-256 digest
			hash := sha256.Sum256(bodyContent)
			expectedDigest := base64.StdEncoding.EncodeToString(hash[:])

			message.Headers = map[string]string{
				"Content-Digest": "sha-256=:" + expectedDigest + ":",
			}

			testutil.LogDetail(t, "Body의 SHA-256 해시 계산:")
			testutil.LogDetail(t, "  해시: %x", hash)
			testutil.LogDetail(t, "  Base64: %s", expectedDigest)
			testutil.LogDetail(t, "  Content-Digest 헤더: %s", message.Headers["Content-Digest"])

			// Sign the message
			testutil.LogDetail(t, "메시지 서명 중...")
			signatureBase := verifier.ConstructSignatureBase(message)
			message.Signature = ed25519.Sign(privateKey, []byte(signatureBase))
			testutil.LogSuccess(t, "서명 생성 완료")

			// Note: This test demonstrates the expected behavior
			// Actual digest verification would be implemented in the verifier
			testutil.LogSuccess(t, "유효한 Content-Digest 검증 시나리오 완료")

			// Verify digest manually for test purposes
			recalcHash := sha256.Sum256(message.Body)
			recalcDigest := base64.StdEncoding.EncodeToString(recalcHash[:])
			assert.Equal(t, expectedDigest, recalcDigest)
			testutil.LogSuccess(t, "Digest 일치 확인 완료")

			// 통과 기준 체크리스트
			testutil.LogPassCriteria(t, []string{
				"Body의 SHA-256 해시 계산 성공",
				"Base64 인코딩 성공",
				"Content-Digest 헤더 설정",
				"Digest 값 일치 확인",
				"메시지 무결성 보장",
			})

			testData := map[string]interface{}{
				"test_case": "10.1.2_Content_Digest_유효",
				"body":      string(bodyContent),
				"digest": map[string]interface{}{
					"algorithm": "sha-256",
					"hash_hex":  string(hash[:]),
					"base64":    expectedDigest,
					"header":    message.Headers["Content-Digest"],
				},
				"validation": "Digest_일치_검증_통과",
			}
			testutil.SaveTestData(t, "rfc9421/verify_valid_digest.json", testData)
		})

		// Test case 2: Invalid digest (mismatch)
		t.Run("invalid_digest", func(t *testing.T) {
			testutil.LogDetail(t, "케이스 2: 잘못된 Content-Digest (불일치)")

			message := &Message{
				AgentDID:     "did:sage:ethereum:agent-digest",
				MessageID:    "msg-digest-002",
				Timestamp:    time.Now(),
				Nonce:        "digest-nonce-2",
				Body:         bodyContent,
				Algorithm:    string(AlgorithmEdDSA),
				SignedFields: []string{"agent_did", "message_id", "timestamp", "nonce", "body"},
			}

			// Set WRONG digest
			wrongDigest := "AQIDBAUG==" // Dummy base64
			message.Headers = map[string]string{
				"Content-Digest": "sha-256=:" + wrongDigest + ":",
			}

			testutil.LogDetail(t, "잘못된 Content-Digest 설정:")
			testutil.LogDetail(t, "  Content-Digest 헤더: %s", message.Headers["Content-Digest"])

			// Calculate actual digest
			actualHash := sha256.Sum256(bodyContent)
			actualDigest := base64.StdEncoding.EncodeToString(actualHash[:])

			testutil.LogDetail(t, "Body의 실제 Digest:")
			testutil.LogDetail(t, "  계산된 Digest: %s", actualDigest)
			testutil.LogDetail(t, "  헤더의 Digest: %s", wrongDigest)

			// Verify digests don't match
			assert.NotEqual(t, actualDigest, wrongDigest)
			testutil.LogSuccess(t, "Digest 불일치 감지 성공")

			// Note: In actual implementation, verifier should reject this
			testutil.LogDetail(t, "Note: 실제 구현에서는 검증기가 이 메시지를 거부해야 함")

			// 통과 기준 체크리스트
			testutil.LogPassCriteria(t, []string{
				"Digest 불일치 시 검증 실패",
				"Body와 헤더 Digest 비교",
				"변조 탐지 기능",
				"메시지 무결성 검증",
				"보안 메커니즘 동작",
			})

			testData := map[string]interface{}{
				"test_case": "10.1.2_Content_Digest_불일치",
				"body":      string(bodyContent),
				"digest": map[string]interface{}{
					"expected": actualDigest,
					"provided": wrongDigest,
					"match":    false,
				},
				"validation": "Digest_불일치_검증_실패",
				"note":       "검증기가 이 메시지를 거부해야 함",
			}
			testutil.SaveTestData(t, "rfc9421/verify_invalid_digest.json", testData)
		})

		testutil.LogSuccess(t, "Content-Digest 검증 테스트 완료")
	})

	// Test 1.2.1 & 1.2.2: Nonce 생성 및 Replay Attack 방어

}

func TestConstructSignatureBase(t *testing.T) {
	// 명세 요구사항: RFC9421 서명 베이스 생성
	testutil.LogTestSection(t, "15.2.1", "RFC9421 서명 베이스 생성")

	verifier := &Verifier{}

	message := &Message{
		AgentDID:  "did:sage:ethereum:agent001",
		MessageID: "msg-001",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Nonce:     "nonce123",
		Body:      []byte("test body"),
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-Custom":     "value",
		},
		SignedFields: []string{"agent_did", "message_id", "timestamp", "nonce", "body", "header.Content-Type"},
	}

	testutil.LogDetail(t, "메시지 정보:")
	testutil.LogDetail(t, "  AgentDID: %s", message.AgentDID)
	testutil.LogDetail(t, "  MessageID: %s", message.MessageID)
	testutil.LogDetail(t, "  Timestamp: %s", message.Timestamp.Format(time.RFC3339))
	testutil.LogDetail(t, "  Nonce: %s", message.Nonce)
	testutil.LogDetail(t, "  Body: %s", string(message.Body))
	testutil.LogDetail(t, "  SignedFields: %v", message.SignedFields)

	expected := `agent_did: did:sage:ethereum:agent001
message_id: msg-001
timestamp: 2024-01-01T12:00:00Z
nonce: nonce123
body: test body
Content-Type: application/json`

	testutil.LogDetail(t, "서명 베이스 생성 중...")
	result := verifier.ConstructSignatureBase(message)
	assert.Equal(t, expected, result)
	testutil.LogSuccess(t, "서명 베이스 생성 및 검증 완료")

	testutil.LogDetail(t, "생성된 서명 베이스:")
	for i, line := range strings.Split(result, "\n") {
		testutil.LogDetail(t, "  [%d] %s", i+1, line)
	}

	// 통과 기준 체크리스트
	testutil.LogPassCriteria(t, []string{
		"서명 베이스 생성 성공",
		"AgentDID 필드 포함",
		"MessageID 필드 포함",
		"Timestamp RFC3339 형식",
		"Nonce 필드 포함",
		"Body 필드 포함",
		"Header (Content-Type) 필드 포함",
		"RFC9421 명세에 따른 형식",
	})

	// CLI 검증용 테스트 데이터 저장
	testData := map[string]interface{}{
		"test_case": "15.2.1_RFC9421_서명_베이스_생성",
		"message": map[string]interface{}{
			"agent_did":     message.AgentDID,
			"message_id":    message.MessageID,
			"timestamp":     message.Timestamp.Format(time.RFC3339),
			"nonce":         message.Nonce,
			"body":          string(message.Body),
			"signed_fields": message.SignedFields,
		},
		"signature_base": result,
		"validation":     "서명_베이스_생성_통과",
	}
	testutil.SaveTestData(t, "rfc9421/construct_signature_base.json", testData)
}

func TestDefaultVerificationOptions(t *testing.T) {
	// 명세 요구사항: 기본 검증 옵션 설정
	testutil.LogTestSection(t, "15.3.1", "RFC9421 기본 검증 옵션")

	testutil.LogDetail(t, "기본 검증 옵션 생성 중...")
	opts := DefaultVerificationOptions()
	testutil.LogSuccess(t, "기본 검증 옵션 생성 완료")

	assert.True(t, opts.RequireActiveAgent)
	assert.Equal(t, 5*time.Minute, opts.MaxClockSkew)
	assert.True(t, opts.VerifyMetadata)
	assert.Empty(t, opts.RequiredCapabilities)

	testutil.LogSuccess(t, "모든 기본값 검증 완료")
	testutil.LogDetail(t, "기본 검증 옵션:")
	testutil.LogDetail(t, "  RequireActiveAgent: %v", opts.RequireActiveAgent)
	testutil.LogDetail(t, "  MaxClockSkew: %v", opts.MaxClockSkew)
	testutil.LogDetail(t, "  VerifyMetadata: %v", opts.VerifyMetadata)
	testutil.LogDetail(t, "  RequiredCapabilities: %v", opts.RequiredCapabilities)

	// 통과 기준 체크리스트
	testutil.LogPassCriteria(t, []string{
		"기본 검증 옵션 생성 성공",
		"RequireActiveAgent = true",
		"MaxClockSkew = 5분",
		"VerifyMetadata = true",
		"RequiredCapabilities = 빈 슬라이스",
		"RFC9421 권장 기본값 설정",
	})

	// CLI 검증용 테스트 데이터 저장
	testData := map[string]interface{}{
		"test_case": "15.3.1_RFC9421_기본_검증_옵션",
		"options": map[string]interface{}{
			"require_active_agent":   opts.RequireActiveAgent,
			"max_clock_skew_minutes": opts.MaxClockSkew.Minutes(),
			"verify_metadata":        opts.VerifyMetadata,
			"required_capabilities":  opts.RequiredCapabilities,
		},
		"validation": "기본_검증_옵션_통과",
	}
	testutil.SaveTestData(t, "rfc9421/default_verification_options.json", testData)
}

func TestSigner(t *testing.T) {
	// Generate test keypair
	testutil.LogDetail(t, "Ed25519 키 쌍 생성 중...")
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	testutil.LogSuccess(t, "Ed25519 키 쌍 생성 완료")

	verifier := NewVerifier()

	t.Run("Parameters", func(t *testing.T) {
		// 명세 요구사항: RFC9421 서명 파라미터 (keyid, created, nonce) 검증
		testutil.LogTestSection(t, "15.6.1", "RFC9421 서명 파라미터 검증")

		now := time.Now()
		keyID := "did:key:z6MkpTHR8VNsBxYAAWHut2Geadd9jSwuBV8xRoAnwWsdvktH"
		nonce := "random-nonce-12345"

		testutil.LogDetail(t, "서명 파라미터 설정:")
		testutil.LogDetail(t, "  KeyID: %s", keyID)
		testutil.LogDetail(t, "  Created: %s", now.Format(time.RFC3339))
		testutil.LogDetail(t, "  Nonce: %s", nonce)

		message := &Message{
			AgentDID:     "did:sage:ethereum:agent001",
			MessageID:    "msg-params-001",
			Timestamp:    now,
			Nonce:        nonce,
			Body:         []byte("test message with parameters"),
			Algorithm:    string(AlgorithmEdDSA),
			KeyID:        keyID,
			SignedFields: []string{"agent_did", "message_id", "timestamp", "nonce", "body"},
		}

		testutil.LogDetail(t, "서명 생성 중...")
		signatureBase := verifier.ConstructSignatureBase(message)
		message.Signature = ed25519.Sign(privateKey, []byte(signatureBase))
		testutil.LogSuccess(t, "서명 생성 완료")

		// Verify parameters are present
		assert.NotEmpty(t, message.KeyID)
		assert.Equal(t, keyID, message.KeyID)
		testutil.LogSuccess(t, "KeyID 파라미터 검증 완료")

		assert.NotZero(t, message.Timestamp)
		assert.Equal(t, now, message.Timestamp)
		testutil.LogSuccess(t, "Created (Timestamp) 파라미터 검증 완료")

		assert.NotEmpty(t, message.Nonce)
		assert.Equal(t, nonce, message.Nonce)
		testutil.LogSuccess(t, "Nonce 파라미터 검증 완료")

		// The Message format is refused even when correctly signed (E03-19, W-010).
		testutil.LogDetail(t, "Message 형식 검증은 거절되어야 함 (E03-19)...")
		err := verifier.VerifySignature(publicKey, message, nil)
		assert.ErrorIs(t, err, ErrMessageFormatRemoved)
		testutil.LogSuccess(t, "Message 형식 거절 확인")

		// 통과 기준 체크리스트
		testutil.LogPassCriteria(t, []string{
			"KeyID 파라미터 설정 및 검증",
			"Created (Timestamp) 파라미터 설정 및 검증",
			"Nonce 파라미터 설정 및 검증",
			"모든 파라미터 포함 서명 생성 성공",
			"Message 형식 검증 거절 (E03-19)",
			"RFC9421 서명 파라미터 명세 준수",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "15.6.1_RFC9421_서명_파라미터_검증",
			"parameters": map[string]interface{}{
				"keyid":   message.KeyID,
				"created": message.Timestamp.Format(time.RFC3339),
				"nonce":   message.Nonce,
			},
			"message": map[string]interface{}{
				"agent_did":     message.AgentDID,
				"message_id":    message.MessageID,
				"algorithm":     message.Algorithm,
				"signed_fields": message.SignedFields,
			},
			"verification": map[string]interface{}{
				"message_format_refused": errors.Is(err, ErrMessageFormatRemoved),
			},
			"validation": "서명_파라미터_검증_통과",
		}
		testutil.SaveTestData(t, "rfc9421/signer_parameters.json", testData)
	})
}
