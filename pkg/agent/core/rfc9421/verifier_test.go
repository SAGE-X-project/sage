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
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifier(t *testing.T) {
	// Generate test keypair
	helpers.LogDetail(t, "EdTest Ed25519 키 쌍 생성 중...")
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Ed25519 키 쌍 생성 완료")

	verifier := NewVerifier()
	helpers.LogSuccess(t, "Verifier 인스턴스 생성 완료")

	t.Run("VerifySignature with valid EdDSA signature", func(t *testing.T) {
		// 명세 요구사항: RFC9421 EdDSA 서명 검증
		helpers.LogTestSection(t, "15.1.1", "RFC9421 검증기 - 유효한 EdDSA 서명")

		message := &Message{
			AgentDID:     "did:sage:ethereum:agent001",
			MessageID:    "msg-001",
			Timestamp:    time.Now(),
			Nonce:        "random-nonce",
			Body:         []byte("test message"),
			Algorithm:    string(AlgorithmEdDSA),
			SignedFields: []string{"agent_did", "message_id", "timestamp", "nonce", "body"},
		}

		helpers.LogDetail(t, "테스트 메시지:")
		helpers.LogDetail(t, "  AgentDID: %s", message.AgentDID)
		helpers.LogDetail(t, "  MessageID: %s", message.MessageID)
		helpers.LogDetail(t, "  Nonce: %s", message.Nonce)
		helpers.LogDetail(t, "  Body: %s", string(message.Body))
		helpers.LogDetail(t, "  Algorithm: %s", message.Algorithm)
		helpers.LogDetail(t, "  SignedFields: %v", message.SignedFields)

		// Sign the message
		helpers.LogDetail(t, "서명 베이스 생성 중...")
		signatureBase := verifier.ConstructSignatureBase(message)
		helpers.LogDetail(t, "서명 베이스 길이: %d bytes", len(signatureBase))

		helpers.LogDetail(t, "Ed25519로 메시지 서명 중...")
		message.Signature = ed25519.Sign(privateKey, []byte(signatureBase))
		helpers.LogSuccess(t, "서명 생성 완료")
		helpers.LogDetail(t, "서명 길이: %d bytes", len(message.Signature))

		// Verify
		helpers.LogDetail(t, "서명 검증 시작...")
		err := verifier.VerifySignature(publicKey, message, nil)
		assert.NoError(t, err)
		helpers.LogSuccess(t, "서명 검증 성공")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"Ed25519 키 쌍 생성 성공",
			"서명 베이스 생성 성공",
			"Ed25519 서명 생성 성공",
			"서명 검증 성공 (에러 없음)",
			"RFC9421 EdDSA 서명 알고리즘 검증",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "15.1.1_RFC9421_검증기_유효한_EdDSA_서명",
			"message": map[string]interface{}{
				"agent_did":     message.AgentDID,
				"message_id":    message.MessageID,
				"nonce":         message.Nonce,
				"body":          string(message.Body),
				"algorithm":     message.Algorithm,
				"signed_fields": message.SignedFields,
			},
			"signature": map[string]interface{}{
				"algorithm":       "Ed25519",
				"signature_bytes": len(message.Signature),
			},
			"verification": map[string]interface{}{
				"success": err == nil,
				"error":   err,
			},
			"validation": "유효한_EdDSA_서명_검증_통과",
		}
		helpers.SaveTestData(t, "rfc9421/verify_valid_eddsa.json", testData)
	})

	t.Run("VerifySignature with invalid signature", func(t *testing.T) {
		// 명세 요구사항: 잘못된 서명 감지
		helpers.LogTestSection(t, "15.1.2", "RFC9421 검증기 - 잘못된 서명 거부")

		message := &Message{
			AgentDID:     "did:sage:ethereum:agent001",
			MessageID:    "msg-002",
			Timestamp:    time.Now(),
			Body:         []byte("test message"),
			Algorithm:    string(AlgorithmEdDSA),
			SignedFields: []string{"body"},
			Signature:    []byte("invalid signature"),
		}

		helpers.LogDetail(t, "잘못된 서명을 가진 메시지:")
		helpers.LogDetail(t, "  AgentDID: %s", message.AgentDID)
		helpers.LogDetail(t, "  MessageID: %s", message.MessageID)
		helpers.LogDetail(t, "  Signature: %q", string(message.Signature))

		helpers.LogDetail(t, "잘못된 서명 검증 시도...")
		err := verifier.VerifySignature(publicKey, message, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature verification failed")

		helpers.LogSuccess(t, "잘못된 서명 올바르게 거부됨")
		helpers.LogDetail(t, "  에러 메시지: %s", err.Error())

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"잘못된 서명 검증 시도",
			"검증 실패 에러 발생",
			"에러 메시지에 'signature verification failed' 포함",
			"보안 검증 기능 정상 동작",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "15.1.2_RFC9421_검증기_잘못된_서명_거부",
			"message": map[string]interface{}{
				"agent_did":   message.AgentDID,
				"message_id":  message.MessageID,
				"body":        string(message.Body),
				"algorithm":   message.Algorithm,
				"signature":   string(message.Signature),
			},
			"verification": map[string]interface{}{
				"success":       false,
				"error_present": err != nil,
				"error_message": err.Error(),
			},
			"validation": "잘못된_서명_거부_통과",
		}
		helpers.SaveTestData(t, "rfc9421/verify_invalid_signature.json", testData)
	})

	t.Run("VerifySignature with clock skew", func(t *testing.T) {
		// 명세 요구사항: 시간 동기화 오차 범위 검증
		helpers.LogTestSection(t, "15.1.3", "RFC9421 검증기 - Clock Skew 거부")

		futureTime := time.Now().Add(10 * time.Minute)
		message := &Message{
			AgentDID:     "did:sage:ethereum:agent001",
			MessageID:    "msg-003",
			Timestamp:    futureTime, // Future timestamp
			Body:         []byte("test message"),
			Algorithm:    string(AlgorithmEdDSA),
			SignedFields: []string{"body"},
			Signature:    []byte("dummy"),
		}

		helpers.LogDetail(t, "시간 동기화 오차 테스트:")
		helpers.LogDetail(t, "  현재 시간: %s", time.Now().Format(time.RFC3339))
		helpers.LogDetail(t, "  메시지 Timestamp: %s (10분 후)", futureTime.Format(time.RFC3339))

		opts := &VerificationOptions{
			MaxClockSkew: 5 * time.Minute,
		}
		helpers.LogDetail(t, "검증 옵션:")
		helpers.LogDetail(t, "  MaxClockSkew: %v", opts.MaxClockSkew)

		helpers.LogDetail(t, "Clock skew 초과 메시지 검증 시도...")
		err := verifier.VerifySignature(publicKey, message, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timestamp outside acceptable range")

		helpers.LogSuccess(t, "Clock skew 초과 메시지 올바르게 거부됨")
		helpers.LogDetail(t, "  에러 메시지: %s", err.Error())

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"미래 timestamp 메시지 생성",
			"MaxClockSkew 5분 설정",
			"10분 미래 timestamp 검증 시도",
			"Clock skew 초과 감지",
			"에러 메시지에 'timestamp outside acceptable range' 포함",
			"시간 동기화 보안 기능 정상 동작",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "15.1.3_RFC9421_검증기_Clock_Skew_거부",
			"timing": map[string]interface{}{
				"current_time":    time.Now().Format(time.RFC3339),
				"message_time":    futureTime.Format(time.RFC3339),
				"skew_minutes":    10.0,
				"max_clock_skew":  opts.MaxClockSkew.Minutes(),
			},
			"verification": map[string]interface{}{
				"success":       false,
				"error_present": err != nil,
				"error_message": err.Error(),
			},
			"validation": "Clock_Skew_거부_통과",
		}
		helpers.SaveTestData(t, "rfc9421/verify_clock_skew.json", testData)
	})

	t.Run("VerifyWithMetadata", func(t *testing.T) {
		// 명세 요구사항: 메타데이터 및 capability 검증
		helpers.LogTestSection(t, "15.1.4", "RFC9421 검증기 - 메타데이터 검증")

		message := &Message{
			AgentDID:     "did:sage:ethereum:agent001",
			MessageID:    "msg-004",
			Timestamp:    time.Now(),
			Body:         []byte("test message"),
			Algorithm:    string(AlgorithmEdDSA),
			SignedFields: []string{"body"},
			Metadata: map[string]interface{}{
				"endpoint": "https://api.example.com",
				"capabilities": map[string]interface{}{
					"chat": true,
					"code": true,
				},
			},
		}

		helpers.LogDetail(t, "메타데이터를 포함한 메시지:")
		helpers.LogDetail(t, "  Endpoint: %v", message.Metadata["endpoint"])
		helpers.LogDetail(t, "  Capabilities: %v", message.Metadata["capabilities"])

		// Sign the message
		helpers.LogDetail(t, "메시지 서명 중...")
		signatureBase := verifier.ConstructSignatureBase(message)
		message.Signature = ed25519.Sign(privateKey, []byte(signatureBase))
		helpers.LogSuccess(t, "서명 생성 완료")

		expectedMetadata := map[string]interface{}{
			"endpoint": "https://api.example.com",
		}
		requiredCapabilities := []string{"chat"}

		helpers.LogDetail(t, "검증 조건:")
		helpers.LogDetail(t, "  Expected endpoint: %v", expectedMetadata["endpoint"])
		helpers.LogDetail(t, "  Required capabilities: %v", requiredCapabilities)

		opts := &VerificationOptions{
			VerifyMetadata: true,
		}

		helpers.LogDetail(t, "메타데이터 검증 중...")
		result, err := verifier.VerifyWithMetadata(publicKey, message, expectedMetadata, requiredCapabilities, opts)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Error)
		helpers.LogSuccess(t, "메타데이터 검증 성공")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"메타데이터 포함 메시지 생성",
			"메시지 서명 생성 성공",
			"Endpoint 메타데이터 일치",
			"필수 capability 존재 확인",
			"메타데이터 검증 성공",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "15.1.4_RFC9421_검증기_메타데이터_검증",
			"metadata": message.Metadata,
			"verification_conditions": map[string]interface{}{
				"expected_metadata":      expectedMetadata,
				"required_capabilities":  requiredCapabilities,
			},
			"result": map[string]interface{}{
				"valid": result.Valid,
				"error": result.Error,
			},
			"validation": "메타데이터_검증_통과",
		}
		helpers.SaveTestData(t, "rfc9421/verify_with_metadata.json", testData)
	})

	t.Run("VerifyWithMetadata missing capability", func(t *testing.T) {
		// 명세 요구사항: 필수 capability 누락 감지
		helpers.LogTestSection(t, "15.1.5", "RFC9421 검증기 - 누락된 Capability 감지")

		message := &Message{
			AgentDID:     "did:sage:ethereum:agent005",
			MessageID:    "msg-005",
			Timestamp:    time.Now(),
			Body:         []byte("test message"),
			Algorithm:    string(AlgorithmEdDSA),
			SignedFields: []string{"body"},
			Metadata: map[string]interface{}{
				"capabilities": map[string]interface{}{
					"chat": true,
				},
			},
		}

		helpers.LogDetail(t, "메시지 capabilities:")
		helpers.LogDetail(t, "  Available: chat=true")
		helpers.LogDetail(t, "  Missing: code")

		// Sign the message
		helpers.LogDetail(t, "메시지 서명 중...")
		signatureBase := verifier.ConstructSignatureBase(message)
		message.Signature = ed25519.Sign(privateKey, []byte(signatureBase))
		helpers.LogSuccess(t, "서명 생성 완료")

		requiredCapabilities := []string{"chat", "code"} // Missing "code"
		helpers.LogDetail(t, "필수 capabilities: %v", requiredCapabilities)

		helpers.LogDetail(t, "누락된 capability로 검증 시도...")
		result, err := verifier.VerifyWithMetadata(publicKey, message, nil, requiredCapabilities, nil)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Error, "missing required capabilities")

		helpers.LogSuccess(t, "누락된 capability 올바르게 감지됨")
		helpers.LogDetail(t, "  검증 결과: Valid=%v", result.Valid)
		helpers.LogDetail(t, "  에러 메시지: %s", result.Error)

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"chat capability만 있는 메시지 생성",
			"chat + code 필수로 검증 시도",
			"code capability 누락 감지",
			"검증 결과 Valid=false",
			"에러 메시지에 'missing required capabilities' 포함",
			"Capability 검증 기능 정상 동작",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "15.1.5_RFC9421_검증기_누락된_Capability",
			"message_capabilities": message.Metadata["capabilities"],
			"required_capabilities": requiredCapabilities,
			"result": map[string]interface{}{
				"valid": result.Valid,
				"error": result.Error,
			},
			"validation": "누락된_Capability_감지_통과",
		}
		helpers.SaveTestData(t, "rfc9421/verify_missing_capability.json", testData)
	})
}

func TestConstructSignatureBase(t *testing.T) {
	// 명세 요구사항: RFC9421 서명 베이스 생성
	helpers.LogTestSection(t, "15.2.1", "RFC9421 서명 베이스 생성")

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

	helpers.LogDetail(t, "메시지 정보:")
	helpers.LogDetail(t, "  AgentDID: %s", message.AgentDID)
	helpers.LogDetail(t, "  MessageID: %s", message.MessageID)
	helpers.LogDetail(t, "  Timestamp: %s", message.Timestamp.Format(time.RFC3339))
	helpers.LogDetail(t, "  Nonce: %s", message.Nonce)
	helpers.LogDetail(t, "  Body: %s", string(message.Body))
	helpers.LogDetail(t, "  SignedFields: %v", message.SignedFields)

	expected := `agent_did: did:sage:ethereum:agent001
message_id: msg-001
timestamp: 2024-01-01T12:00:00Z
nonce: nonce123
body: test body
Content-Type: application/json`

	helpers.LogDetail(t, "서명 베이스 생성 중...")
	result := verifier.ConstructSignatureBase(message)
	assert.Equal(t, expected, result)
	helpers.LogSuccess(t, "서명 베이스 생성 및 검증 완료")

	helpers.LogDetail(t, "생성된 서명 베이스:")
	for i, line := range strings.Split(result, "\n") {
		helpers.LogDetail(t, "  [%d] %s", i+1, line)
	}

	// 통과 기준 체크리스트
	helpers.LogPassCriteria(t, []string{
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
	helpers.SaveTestData(t, "rfc9421/construct_signature_base.json", testData)
}

func TestDefaultVerificationOptions(t *testing.T) {
	// 명세 요구사항: 기본 검증 옵션 설정
	helpers.LogTestSection(t, "15.3.1", "RFC9421 기본 검증 옵션")

	helpers.LogDetail(t, "기본 검증 옵션 생성 중...")
	opts := DefaultVerificationOptions()
	helpers.LogSuccess(t, "기본 검증 옵션 생성 완료")

	assert.True(t, opts.RequireActiveAgent)
	assert.Equal(t, 5*time.Minute, opts.MaxClockSkew)
	assert.True(t, opts.VerifyMetadata)
	assert.Empty(t, opts.RequiredCapabilities)

	helpers.LogSuccess(t, "모든 기본값 검증 완료")
	helpers.LogDetail(t, "기본 검증 옵션:")
	helpers.LogDetail(t, "  RequireActiveAgent: %v", opts.RequireActiveAgent)
	helpers.LogDetail(t, "  MaxClockSkew: %v", opts.MaxClockSkew)
	helpers.LogDetail(t, "  VerifyMetadata: %v", opts.VerifyMetadata)
	helpers.LogDetail(t, "  RequiredCapabilities: %v", opts.RequiredCapabilities)

	// 통과 기준 체크리스트
	helpers.LogPassCriteria(t, []string{
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
			"require_active_agent":  opts.RequireActiveAgent,
			"max_clock_skew_minutes": opts.MaxClockSkew.Minutes(),
			"verify_metadata":       opts.VerifyMetadata,
			"required_capabilities": opts.RequiredCapabilities,
		},
		"validation": "기본_검증_옵션_통과",
	}
	helpers.SaveTestData(t, "rfc9421/default_verification_options.json", testData)
}

func TestHasRequiredCapabilities(t *testing.T) {
	// 명세 요구사항: Capability 검증 로직 테스트
	helpers.LogTestSection(t, "15.4.1", "RFC9421 Capability 검증")

	capabilities := map[string]interface{}{
		"chat":  true,
		"code":  true,
		"voice": false,
	}

	helpers.LogDetail(t, "테스트용 Capabilities:")
	helpers.LogDetail(t, "  chat: true")
	helpers.LogDetail(t, "  code: true")
	helpers.LogDetail(t, "  voice: false")

	tests := []struct {
		name     string
		required []string
		expected bool
	}{
		{
			name:     "All capabilities present",
			required: []string{"chat", "code"},
			expected: true,
		},
		{
			name:     "Missing capability",
			required: []string{"chat", "video"},
			expected: false,
		},
		{
			name:     "Empty required",
			required: []string{},
			expected: true,
		},
	}

	passedCount := 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.LogDetail(t, "테스트 케이스: %s", tt.name)
			helpers.LogDetail(t, "  필수 capabilities: %v", tt.required)
			helpers.LogDetail(t, "  예상 결과: %v", tt.expected)

			result := hasRequiredCapabilities(capabilities, tt.required)
			assert.Equal(t, tt.expected, result)

			if result == tt.expected {
				passedCount++
				helpers.LogSuccess(t, "테스트 케이스 통과")
			}
			helpers.LogDetail(t, "  실제 결과: %v", result)
		})
	}

	helpers.LogSuccess(t, "모든 Capability 검증 테스트 완료")
	helpers.LogDetail(t, "통과한 테스트: %d/%d", passedCount, len(tests))

	// 통과 기준 체크리스트
	helpers.LogPassCriteria(t, []string{
		"모든 capability 존재 시 true 반환",
		"누락된 capability 존재 시 false 반환",
		"필수 capability 없을 시 true 반환",
		"Capability 검증 로직 정상 동작",
	})

	// CLI 검증용 테스트 데이터 저장
	testData := map[string]interface{}{
		"test_case":      "15.4.1_RFC9421_Capability_검증",
		"capabilities":   capabilities,
		"test_cases":     len(tests),
		"passed_count":   passedCount,
		"validation":     "Capability_검증_통과",
	}
	helpers.SaveTestData(t, "rfc9421/has_required_capabilities.json", testData)
}

func TestCompareValues(t *testing.T) {
	// 명세 요구사항: 값 비교 로직 테스트
	helpers.LogTestSection(t, "15.5.1", "RFC9421 값 비교 로직")

	tests := []struct {
		name     string
		v1       interface{}
		v2       interface{}
		expected bool
	}{
		{
			name:     "Equal strings",
			v1:       "test",
			v2:       "test",
			expected: true,
		},
		{
			name:     "Different strings",
			v1:       "test1",
			v2:       "test2",
			expected: false,
		},
		{
			name:     "Equal maps",
			v1:       map[string]interface{}{"key": "value"},
			v2:       map[string]interface{}{"key": "value"},
			expected: true,
		},
		{
			name:     "Different maps",
			v1:       map[string]interface{}{"key": "value1"},
			v2:       map[string]interface{}{"key": "value2"},
			expected: false,
		},
	}

	helpers.LogDetail(t, "값 비교 테스트 케이스 %d개", len(tests))

	passedCount := 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers.LogDetail(t, "테스트 케이스: %s", tt.name)
			helpers.LogDetail(t, "  값1: %v", tt.v1)
			helpers.LogDetail(t, "  값2: %v", tt.v2)
			helpers.LogDetail(t, "  예상 결과: %v", tt.expected)

			result := compareValues(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result)

			if result == tt.expected {
				passedCount++
				helpers.LogSuccess(t, "테스트 케이스 통과")
			}
			helpers.LogDetail(t, "  실제 결과: %v", result)
		})
	}

	helpers.LogSuccess(t, "모든 값 비교 테스트 완료")
	helpers.LogDetail(t, "통과한 테스트: %d/%d", passedCount, len(tests))

	// 통과 기준 체크리스트
	helpers.LogPassCriteria(t, []string{
		"동일한 문자열 비교 성공",
		"다른 문자열 비교 성공",
		"동일한 맵 비교 성공",
		"다른 맵 비교 성공",
		"값 비교 로직 정상 동작",
	})

	// CLI 검증용 테스트 데이터 저장
	testData := map[string]interface{}{
		"test_case":    "15.5.1_RFC9421_값_비교_로직",
		"test_cases":   len(tests),
		"passed_count": passedCount,
		"validation":   "값_비교_로직_통과",
	}
	helpers.SaveTestData(t, "rfc9421/compare_values.json", testData)
}
