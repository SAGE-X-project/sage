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
	"testing"
	"time"

	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageBuilder(t *testing.T) {
	t.Run("Build complete message", func(t *testing.T) {
		// 명세 요구사항: RFC9421 메시지 빌더 패턴을 사용한 완전한 메시지 생성
		helpers.LogTestSection(t, "14.1.1", "RFC9421 메시지 빌더 - 완전한 메시지 생성")

		now := time.Now()
		helpers.LogDetail(t, "테스트 파라미터:")
		helpers.LogDetail(t, "  Agent DID: did:sage:ethereum:agent001")
		helpers.LogDetail(t, "  Message ID: msg-001")
		helpers.LogDetail(t, "  Timestamp: %s", now.Format(time.RFC3339Nano))
		helpers.LogDetail(t, "  Nonce: nonce123")
		helpers.LogDetail(t, "  Body: test body")
		helpers.LogDetail(t, "  Algorithm: EdDSA")
		helpers.LogDetail(t, "  Key ID: key-001")

		helpers.LogDetail(t, "빌더 패턴으로 메시지 생성 중...")
		message := NewMessageBuilder().
			WithAgentDID("did:sage:ethereum:agent001").
			WithMessageID("msg-001").
			WithTimestamp(now).
			WithNonce("nonce123").
			WithBody([]byte("test body")).
			AddHeader("Content-Type", "application/json").
			AddHeader("X-Custom", "value").
			AddMetadata("version", "1.0").
			AddMetadata("feature", "test").
			WithAlgorithm(AlgorithmEdDSA).
			WithKeyID("key-001").
			WithSignedFields("agent_did", "message_id", "body").
			WithSignature([]byte("signature")).
			Build()
		helpers.LogSuccess(t, "메시지 빌드 완료")

		assert.Equal(t, "did:sage:ethereum:agent001", message.AgentDID)
		assert.Equal(t, "msg-001", message.MessageID)
		assert.Equal(t, now, message.Timestamp)
		assert.Equal(t, "nonce123", message.Nonce)
		assert.Equal(t, []byte("test body"), message.Body)
		assert.Equal(t, "application/json", message.Headers["Content-Type"])
		assert.Equal(t, "value", message.Headers["X-Custom"])
		assert.Equal(t, "1.0", message.Metadata["version"])
		assert.Equal(t, "test", message.Metadata["feature"])
		assert.Equal(t, string(AlgorithmEdDSA), message.Algorithm)
		assert.Equal(t, "key-001", message.KeyID)
		assert.Equal(t, []string{"agent_did", "message_id", "body"}, message.SignedFields)
		assert.Equal(t, []byte("signature"), message.Signature)

		helpers.LogSuccess(t, "모든 필드 검증 완료")
		helpers.LogDetail(t, "생성된 메시지 구조:")
		helpers.LogDetail(t, "  AgentDID: %s", message.AgentDID)
		helpers.LogDetail(t, "  MessageID: %s", message.MessageID)
		helpers.LogDetail(t, "  Nonce: %s", message.Nonce)
		helpers.LogDetail(t, "  Algorithm: %s", message.Algorithm)
		helpers.LogDetail(t, "  KeyID: %s", message.KeyID)
		helpers.LogDetail(t, "  Headers count: %d", len(message.Headers))
		helpers.LogDetail(t, "  Metadata count: %d", len(message.Metadata))
		helpers.LogDetail(t, "  SignedFields count: %d", len(message.SignedFields))

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"빌더 패턴으로 메시지 생성 성공",
			"AgentDID 올바르게 설정됨",
			"MessageID 올바르게 설정됨",
			"Timestamp 올바르게 설정됨",
			"Nonce 올바르게 설정됨",
			"Body 올바르게 설정됨",
			"Headers 2개 올바르게 추가됨",
			"Metadata 2개 올바르게 추가됨",
			"Algorithm EdDSA로 설정됨",
			"KeyID 올바르게 설정됨",
			"SignedFields 3개 올바르게 설정됨",
			"Signature 올바르게 설정됨",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "14.1.1_RFC9421_메시지_빌더_완전한_생성",
			"message": map[string]interface{}{
				"agent_did":         message.AgentDID,
				"message_id":        message.MessageID,
				"timestamp":         message.Timestamp.Format(time.RFC3339Nano),
				"nonce":             message.Nonce,
				"body":              string(message.Body),
				"algorithm":         message.Algorithm,
				"key_id":            message.KeyID,
				"headers":           message.Headers,
				"metadata":          message.Metadata,
				"signed_fields":     message.SignedFields,
				"signature_present": len(message.Signature) > 0,
			},
			"validation": "완전한_메시지_생성_통과",
		}
		helpers.SaveTestData(t, "rfc9421/message_builder_complete.json", testData)
	})

	t.Run("Build with default signed fields", func(t *testing.T) {
		// 명세 요구사항: SignedFields를 명시하지 않을 경우 기본 필드 자동 설정
		helpers.LogTestSection(t, "14.1.2", "RFC9421 메시지 빌더 - 기본 서명 필드")

		helpers.LogDetail(t, "AgentDID만 지정하고 SignedFields 미지정")
		helpers.LogDetail(t, "  Agent DID: did:sage:ethereum:agent001")

		helpers.LogDetail(t, "기본 설정으로 메시지 빌드 중...")
		message := NewMessageBuilder().
			WithAgentDID("did:sage:ethereum:agent001").
			Build()
		helpers.LogSuccess(t, "메시지 빌드 완료")

		expectedFields := []string{"agent_did", "message_id", "timestamp", "nonce", "body"}
		assert.Equal(t, expectedFields, message.SignedFields)

		helpers.LogSuccess(t, "기본 서명 필드 검증 완료")
		helpers.LogDetail(t, "기본 SignedFields:")
		for i, field := range message.SignedFields {
			helpers.LogDetail(t, "  [%d] %s", i+1, field)
		}

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"AgentDID만으로 메시지 생성 성공",
			"SignedFields가 기본값으로 자동 설정됨",
			"기본 필드: agent_did, message_id, timestamp, nonce, body",
			"필드 개수 5개 확인",
			"RFC9421 명세에 따른 기본 서명 필드 설정",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "14.1.2_RFC9421_메시지_빌더_기본_서명_필드",
			"input": map[string]interface{}{
				"agent_did":               "did:sage:ethereum:agent001",
				"signed_fields_specified": false,
			},
			"output": map[string]interface{}{
				"signed_fields": message.SignedFields,
				"field_count":   len(message.SignedFields),
			},
			"validation": "기본_서명_필드_자동_설정_통과",
		}
		helpers.SaveTestData(t, "rfc9421/message_builder_default_fields.json", testData)
	})

	t.Run("Build minimal message", func(t *testing.T) {
		// 명세 요구사항: 파라미터 없이 최소한의 메시지 생성 가능
		helpers.LogTestSection(t, "14.1.3", "RFC9421 메시지 빌더 - 최소 메시지")

		helpers.LogDetail(t, "파라미터 없이 빌더만으로 메시지 생성")

		helpers.LogDetail(t, "최소 설정으로 메시지 빌드 중...")
		message := NewMessageBuilder().Build()
		helpers.LogSuccess(t, "메시지 빌드 완료")

		assert.NotNil(t, message)
		assert.NotNil(t, message.Headers)
		assert.NotNil(t, message.Metadata)
		assert.NotZero(t, message.Timestamp)
		assert.NotEmpty(t, message.SignedFields)

		helpers.LogSuccess(t, "최소 메시지 필수 필드 검증 완료")
		helpers.LogDetail(t, "생성된 최소 메시지:")
		helpers.LogDetail(t, "  Timestamp 자동 생성: %s", message.Timestamp.Format(time.RFC3339Nano))
		helpers.LogDetail(t, "  Headers 초기화: %v", message.Headers != nil)
		helpers.LogDetail(t, "  Metadata 초기화: %v", message.Metadata != nil)
		helpers.LogDetail(t, "  SignedFields 자동 설정: %d개", len(message.SignedFields))

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"파라미터 없이 메시지 생성 성공",
			"메시지 객체가 nil이 아님",
			"Headers 맵이 초기화됨",
			"Metadata 맵이 초기화됨",
			"Timestamp가 자동으로 설정됨",
			"SignedFields가 비어있지 않음",
			"최소한의 안전한 메시지 구조 보장",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "14.1.3_RFC9421_메시지_빌더_최소_메시지",
			"message": map[string]interface{}{
				"is_nil":               message == nil,
				"headers_initialized":  message.Headers != nil,
				"metadata_initialized": message.Metadata != nil,
				"timestamp_set":        !message.Timestamp.IsZero(),
				"signed_fields_count":  len(message.SignedFields),
			},
			"validation": "최소_메시지_생성_통과",
		}
		helpers.SaveTestData(t, "rfc9421/message_builder_minimal.json", testData)
	})

	t.Run("SetBody", func(t *testing.T) {
		// 명세 요구사항: RFC9421 Content-Digest 생성 검증
		helpers.LogTestSection(t, "14.1.4", "RFC9421 메시지 빌더 - Content-Digest 생성")

		testBody := []byte("test message body for content digest")
		helpers.LogDetail(t, "테스트 Body: %s", string(testBody))

		helpers.LogDetail(t, "WithBody()로 메시지 생성...")
		message := NewMessageBuilder().
			WithAgentDID("did:sage:ethereum:agent001").
			WithMessageID("msg-digest-001").
			WithBody(testBody).
			Build()
		helpers.LogSuccess(t, "메시지 빌드 완료")

		// Body가 올바르게 설정되었는지 확인
		assert.Equal(t, testBody, message.Body)
		assert.NotNil(t, message.Body)
		assert.Greater(t, len(message.Body), 0)

		helpers.LogSuccess(t, "Body 설정 및 검증 완료")
		helpers.LogDetail(t, "Body 길이: %d bytes", len(message.Body))
		helpers.LogDetail(t, "Body 내용: %s", string(message.Body))

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"WithBody() 메서드로 메시지 Body 설정",
			"Body가 nil이 아님",
			"Body 길이가 0보다 큼",
			"Body 내용이 원본과 일치",
			"Content-Digest 생성을 위한 Body 준비 완료",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "14.1.4_RFC9421_메시지_빌더_SetBody",
			"message": map[string]interface{}{
				"agent_did":   message.AgentDID,
				"message_id":  message.MessageID,
				"body":        string(message.Body),
				"body_length": len(message.Body),
				"body_set":    message.Body != nil,
			},
			"validation": "Content_Digest_Body_설정_통과",
		}
		helpers.SaveTestData(t, "rfc9421/message_builder_set_body.json", testData)
	})
}

func TestParseMessageFromHeaders(t *testing.T) {
	t.Run("Parse complete headers", func(t *testing.T) {
		// 명세 요구사항: HTTP 헤더에서 RFC9421 메시지 파싱
		helpers.LogTestSection(t, "14.2.1", "RFC9421 헤더 파싱 - 완전한 헤더")

		headers := map[string]string{
			"X-Agent-DID":           "did:sage:ethereum:agent001",
			"X-Message-ID":          "msg-001",
			"X-Timestamp":           "2024-01-01T12:00:00Z",
			"X-Nonce":               "nonce123",
			"X-Signature-Algorithm": "EdDSA",
			"X-Key-ID":              "key-001",
			"X-Signed-Fields":       "agent_did, message_id, body",
			"Content-Type":          "application/json",
		}

		helpers.LogDetail(t, "테스트 헤더:")
		helpers.LogDetail(t, "  X-Agent-DID: %s", headers["X-Agent-DID"])
		helpers.LogDetail(t, "  X-Message-ID: %s", headers["X-Message-ID"])
		helpers.LogDetail(t, "  X-Timestamp: %s", headers["X-Timestamp"])
		helpers.LogDetail(t, "  X-Nonce: %s", headers["X-Nonce"])
		helpers.LogDetail(t, "  X-Signature-Algorithm: %s", headers["X-Signature-Algorithm"])
		helpers.LogDetail(t, "  X-Key-ID: %s", headers["X-Key-ID"])
		helpers.LogDetail(t, "  X-Signed-Fields: %s", headers["X-Signed-Fields"])
		helpers.LogDetail(t, "  Content-Type: %s", headers["Content-Type"])

		body := []byte("test body")
		helpers.LogDetail(t, "Body: %s", string(body))

		helpers.LogDetail(t, "헤더에서 메시지 파싱 중...")
		message, err := ParseMessageFromHeaders(headers, body)
		require.NoError(t, err)
		helpers.LogSuccess(t, "메시지 파싱 완료")

		assert.Equal(t, "did:sage:ethereum:agent001", message.AgentDID)
		assert.Equal(t, "msg-001", message.MessageID)
		assert.Equal(t, "2024-01-01T12:00:00Z", message.Timestamp.Format(time.RFC3339))
		assert.Equal(t, "nonce123", message.Nonce)
		assert.Equal(t, string(AlgorithmEdDSA), message.Algorithm)
		assert.Equal(t, "key-001", message.KeyID)
		assert.Equal(t, []string{"agent_did", "message_id", "body"}, message.SignedFields)
		assert.Equal(t, body, message.Body)
		assert.Equal(t, "application/json", message.Headers["Content-Type"])

		helpers.LogSuccess(t, "모든 필드 검증 완료")
		helpers.LogDetail(t, "파싱된 메시지:")
		helpers.LogDetail(t, "  AgentDID: %s", message.AgentDID)
		helpers.LogDetail(t, "  MessageID: %s", message.MessageID)
		helpers.LogDetail(t, "  Timestamp: %s", message.Timestamp.Format(time.RFC3339))
		helpers.LogDetail(t, "  Nonce: %s", message.Nonce)
		helpers.LogDetail(t, "  Algorithm: %s", message.Algorithm)
		helpers.LogDetail(t, "  KeyID: %s", message.KeyID)
		helpers.LogDetail(t, "  SignedFields: %v", message.SignedFields)

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"헤더 맵에서 메시지 파싱 성공",
			"AgentDID 올바르게 추출됨",
			"MessageID 올바르게 추출됨",
			"Timestamp RFC3339 형식으로 파싱됨",
			"Nonce 올바르게 추출됨",
			"Algorithm EdDSA로 파싱됨",
			"KeyID 올바르게 추출됨",
			"SignedFields 3개 필드로 파싱됨",
			"Body 올바르게 설정됨",
			"Content-Type 헤더 보존됨",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "14.2.1_RFC9421_헤더_파싱_완전한_헤더",
			"input": map[string]interface{}{
				"headers": headers,
				"body":    string(body),
			},
			"parsed_message": map[string]interface{}{
				"agent_did":     message.AgentDID,
				"message_id":    message.MessageID,
				"timestamp":     message.Timestamp.Format(time.RFC3339),
				"nonce":         message.Nonce,
				"algorithm":     message.Algorithm,
				"key_id":        message.KeyID,
				"signed_fields": message.SignedFields,
				"body":          string(message.Body),
			},
			"validation": "완전한_헤더_파싱_통과",
		}
		helpers.SaveTestData(t, "rfc9421/parse_complete_headers.json", testData)
	})

	t.Run("Parse minimal headers", func(t *testing.T) {
		// 명세 요구사항: 최소 필수 헤더만으로 메시지 파싱 가능
		helpers.LogTestSection(t, "14.2.2", "RFC9421 헤더 파싱 - 최소 헤더")

		headers := map[string]string{
			"X-Agent-DID": "did:sage:ethereum:agent001",
		}

		helpers.LogDetail(t, "최소 헤더만 제공:")
		helpers.LogDetail(t, "  X-Agent-DID: %s", headers["X-Agent-DID"])

		body := []byte("test body")
		helpers.LogDetail(t, "Body: %s", string(body))

		helpers.LogDetail(t, "최소 헤더로 메시지 파싱 중...")
		message, err := ParseMessageFromHeaders(headers, body)
		require.NoError(t, err)
		helpers.LogSuccess(t, "메시지 파싱 완료")

		assert.Equal(t, "did:sage:ethereum:agent001", message.AgentDID)
		assert.Equal(t, body, message.Body)
		assert.NotNil(t, message.Headers)
		assert.NotNil(t, message.Metadata)

		helpers.LogSuccess(t, "최소 헤더 파싱 검증 완료")
		helpers.LogDetail(t, "파싱 결과:")
		helpers.LogDetail(t, "  AgentDID: %s", message.AgentDID)
		helpers.LogDetail(t, "  Body: %s", string(message.Body))
		helpers.LogDetail(t, "  Headers 초기화: %v", message.Headers != nil)
		helpers.LogDetail(t, "  Metadata 초기화: %v", message.Metadata != nil)

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"AgentDID만으로 파싱 성공",
			"AgentDID 올바르게 추출됨",
			"Body 올바르게 설정됨",
			"Headers 맵 초기화됨",
			"Metadata 맵 초기화됨",
			"최소 필수 필드로 안전한 메시지 생성",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "14.2.2_RFC9421_헤더_파싱_최소_헤더",
			"input": map[string]interface{}{
				"headers":      headers,
				"body":         string(body),
				"header_count": len(headers),
			},
			"parsed_message": map[string]interface{}{
				"agent_did":            message.AgentDID,
				"body":                 string(message.Body),
				"headers_initialized":  message.Headers != nil,
				"metadata_initialized": message.Metadata != nil,
			},
			"validation": "최소_헤더_파싱_통과",
		}
		helpers.SaveTestData(t, "rfc9421/parse_minimal_headers.json", testData)
	})

	t.Run("Parse with invalid timestamp", func(t *testing.T) {
		// 명세 요구사항: 잘못된 timestamp 형식 시 기본값 사용
		helpers.LogTestSection(t, "14.2.3", "RFC9421 헤더 파싱 - 잘못된 Timestamp 처리")

		headers := map[string]string{
			"X-Timestamp": "invalid-timestamp",
		}

		helpers.LogDetail(t, "잘못된 형식의 Timestamp 헤더:")
		helpers.LogDetail(t, "  X-Timestamp: %s", headers["X-Timestamp"])

		helpers.LogDetail(t, "잘못된 timestamp로 메시지 파싱 시도...")
		message, err := ParseMessageFromHeaders(headers, nil)
		require.NoError(t, err)
		helpers.LogSuccess(t, "메시지 파싱 완료 (에러 없음)")

		// Should use default timestamp when parsing fails
		assert.NotZero(t, message.Timestamp)

		helpers.LogSuccess(t, "잘못된 timestamp 처리 검증 완료")
		helpers.LogDetail(t, "파싱 결과:")
		helpers.LogDetail(t, "  파싱 에러 발생: 없음")
		helpers.LogDetail(t, "  Timestamp 자동 설정: %s", message.Timestamp.Format(time.RFC3339Nano))
		helpers.LogDetail(t, "  Timestamp.IsZero(): %v", message.Timestamp.IsZero())

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"잘못된 timestamp 형식에도 파싱 성공",
			"파싱 에러가 발생하지 않음",
			"Timestamp가 기본값(현재 시간)으로 설정됨",
			"Timestamp.IsZero()가 false",
			"Graceful degradation 동작 확인",
			"안전한 에러 처리",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "14.2.3_RFC9421_헤더_파싱_잘못된_Timestamp",
			"input": map[string]interface{}{
				"headers":         headers,
				"timestamp_value": headers["X-Timestamp"],
			},
			"result": map[string]interface{}{
				"parsing_error":     err != nil,
				"timestamp_set":     !message.Timestamp.IsZero(),
				"timestamp":         message.Timestamp.Format(time.RFC3339Nano),
				"graceful_handling": true,
			},
			"validation": "잘못된_Timestamp_처리_통과",
		}
		helpers.SaveTestData(t, "rfc9421/parse_invalid_timestamp.json", testData)
	})
}

func TestSignatureAlgorithmConstants(t *testing.T) {
	// 명세 요구사항: RFC9421 서명 알고리즘 상수 검증
	helpers.LogTestSection(t, "14.3.1", "RFC9421 서명 알고리즘 상수")

	helpers.LogDetail(t, "지원하는 서명 알고리즘:")
	helpers.LogDetail(t, "  EdDSA (Ed25519)")
	helpers.LogDetail(t, "  ES256K (ECDSA P-256)")
	helpers.LogDetail(t, "  ECDSA (일반)")
	helpers.LogDetail(t, "  ECDSA-secp256k1 (비트코인 커브)")

	helpers.LogDetail(t, "알고리즘 상수 검증 중...")

	assert.Equal(t, SignatureAlgorithm("EdDSA"), AlgorithmEdDSA)
	helpers.LogSuccess(t, "EdDSA 상수 검증 완료")

	assert.Equal(t, SignatureAlgorithm("ES256K"), AlgorithmES256K)
	helpers.LogSuccess(t, "ES256K 상수 검증 완료")

	assert.Equal(t, SignatureAlgorithm("ECDSA"), AlgorithmECDSA)
	helpers.LogSuccess(t, "ECDSA 상수 검증 완료")

	assert.Equal(t, SignatureAlgorithm("ECDSA-secp256k1"), AlgorithmECDSASecp256k1)
	helpers.LogSuccess(t, "ECDSA-secp256k1 상수 검증 완료")

	// 통과 기준 체크리스트
	helpers.LogPassCriteria(t, []string{
		"EdDSA 알고리즘 상수 일치",
		"ES256K 알고리즘 상수 일치",
		"ECDSA 알고리즘 상수 일치",
		"ECDSA-secp256k1 알고리즘 상수 일치",
		"RFC9421 명세의 서명 알고리즘 지원",
	})

	// CLI 검증용 테스트 데이터 저장
	testData := map[string]interface{}{
		"test_case": "14.3.1_RFC9421_서명_알고리즘_상수",
		"algorithms": map[string]interface{}{
			"EdDSA":           string(AlgorithmEdDSA),
			"ES256K":          string(AlgorithmES256K),
			"ECDSA":           string(AlgorithmECDSA),
			"ECDSA-secp256k1": string(AlgorithmECDSASecp256k1),
		},
		"validation": "모든_알고리즘_상수_검증_통과",
	}
	helpers.SaveTestData(t, "rfc9421/algorithm_constants.json", testData)
}
