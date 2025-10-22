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
	"strings"
	"testing"

	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSignatureInput(t *testing.T) {
	// Test 1.1.1: Basic parsing
	t.Run("basic parsing", func(t *testing.T) {
		// 명세 요구사항: RFC9421 서명 입력 기본 파싱
		helpers.LogTestSection(t, "13.1.1", "RFC9421 파서 - 기본 서명 입력 파싱")

		input := `sig1=("@method" "host");keyid="did:key:z6Mk...";alg="ed25519";created=1719234000`
		helpers.LogDetail(t, "서명 입력 문자열:")
		helpers.LogDetail(t, "  %s", input)

		helpers.LogDetail(t, "서명 입력 파싱 중...")
		result, err := ParseSignatureInput(input)
		require.NoError(t, err)
		require.NotNil(t, result)
		helpers.LogSuccess(t, "서명 입력 파싱 완료")

		sig1, exists := result["sig1"]
		require.True(t, exists)
		helpers.LogSuccess(t, "sig1 서명 발견")

		assert.Equal(t, []string{`"@method"`, `"host"`}, sig1.CoveredComponents)
		assert.Equal(t, "did:key:z6Mk...", sig1.KeyID)
		assert.Equal(t, "ed25519", sig1.Algorithm)
		assert.Equal(t, int64(1719234000), sig1.Created)

		helpers.LogSuccess(t, "서명 파라미터 검증 완료")
		helpers.LogDetail(t, "파싱된 sig1 파라미터:")
		helpers.LogDetail(t, "  커버된 컴포넌트: %v", sig1.CoveredComponents)
		helpers.LogDetail(t, "  KeyID: %s", sig1.KeyID)
		helpers.LogDetail(t, "  Algorithm: %s", sig1.Algorithm)
		helpers.LogDetail(t, "  Created: %d", sig1.Created)

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"서명 입력 문자열 파싱 성공",
			"sig1 서명 존재 확인",
			"커버된 컴포넌트 2개 검증",
			"KeyID DID 형식 검증",
			"Algorithm ed25519 검증",
			"Created 타임스탬프 검증",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":     "13.1.1_RFC9421_파서_기본_파싱",
			"signature_input": input,
			"parsed_signature": map[string]interface{}{
				"name":               "sig1",
				"covered_components": sig1.CoveredComponents,
				"key_id":             sig1.KeyID,
				"algorithm":          sig1.Algorithm,
				"created":            sig1.Created,
			},
			"validation": "통과",
		}
		helpers.SaveTestData(t, "rfc9421/parser_basic.json", testData)
	})

	// Test 1.1.2: Multiple signatures and parameters
	t.Run("multiple signatures with parameters", func(t *testing.T) {
		// 명세 요구사항: 다중 서명 및 파라미터 파싱
		helpers.LogTestSection(t, "13.1.2", "RFC9421 파서 - 다중 서명 파싱")

		input := `sig-a=("@method");expires=1719237600, sig-b=("host" "date");keyid="test-key-2";nonce="abcdef"`
		helpers.LogDetail(t, "다중 서명 입력 문자열:")
		helpers.LogDetail(t, "  %s", input)

		helpers.LogDetail(t, "다중 서명 파싱 중...")
		result, err := ParseSignatureInput(input)
		require.NoError(t, err)
		require.NotNil(t, result)
		helpers.LogSuccess(t, "다중 서명 파싱 완료")
		helpers.LogDetail(t, "파싱된 서명 개수: %d", len(result))

		// Check sig-a
		helpers.LogDetail(t, "sig-a 서명 검증 중...")
		sigA, exists := result["sig-a"]
		require.True(t, exists)
		assert.Equal(t, []string{`"@method"`}, sigA.CoveredComponents)
		assert.Equal(t, int64(1719237600), sigA.Expires)
		helpers.LogSuccess(t, "sig-a 검증 완료")
		helpers.LogDetail(t, "  커버된 컴포넌트: %v", sigA.CoveredComponents)
		helpers.LogDetail(t, "  Expires: %d", sigA.Expires)

		// Check sig-b
		helpers.LogDetail(t, "sig-b 서명 검증 중...")
		sigB, exists := result["sig-b"]
		require.True(t, exists)
		assert.Equal(t, []string{`"host"`, `"date"`}, sigB.CoveredComponents)
		assert.Equal(t, "test-key-2", sigB.KeyID)
		assert.Equal(t, "abcdef", sigB.Nonce)
		helpers.LogSuccess(t, "sig-b 검증 완료")
		helpers.LogDetail(t, "  커버된 컴포넌트: %v", sigB.CoveredComponents)
		helpers.LogDetail(t, "  KeyID: %s", sigB.KeyID)
		helpers.LogDetail(t, "  Nonce: %s", sigB.Nonce)

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"다중 서명 입력 파싱 성공",
			"sig-a 서명 존재 및 검증",
			"sig-a expires 파라미터 검증",
			"sig-b 서명 존재 및 검증",
			"sig-b keyid 파라미터 검증",
			"sig-b nonce 파라미터 검증",
			"다중 서명 처리 정상 동작",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":     "13.1.2_RFC9421_파서_다중_서명",
			"signature_input": input,
			"parsed_signatures": map[string]interface{}{
				"sig-a": map[string]interface{}{
					"covered_components": sigA.CoveredComponents,
					"expires":            sigA.Expires,
				},
				"sig-b": map[string]interface{}{
					"covered_components": sigB.CoveredComponents,
					"key_id":             sigB.KeyID,
					"nonce":              sigB.Nonce,
				},
			},
			"signature_count": len(result),
			"validation":      "통과",
		}
		helpers.SaveTestData(t, "rfc9421/parser_multiple.json", testData)
	})

	// Test 1.1.3: Whitespace and case variations
	t.Run("whitespace and case handling", func(t *testing.T) {
		input := `sig1 = ( "@path"  "@query" ); KeyId = "test-key" ; Alg = "ecdsa-p256"`

		result, err := ParseSignatureInput(input)
		require.NoError(t, err)
		require.NotNil(t, result)

		sig1, exists := result["sig1"]
		require.True(t, exists)

		assert.Equal(t, []string{`"@path"`, `"@query"`}, sig1.CoveredComponents)
		assert.Equal(t, "test-key", sig1.KeyID)
		assert.Equal(t, "ecdsa-p256", sig1.Algorithm)
	})

	// Test 1.3.1: Malformed input
	t.Run("malformed input", func(t *testing.T) {
		// 명세 요구사항: 잘못된 형식의 입력에 대한 에러 처리
		helpers.LogTestSection(t, "13.1.3", "RFC9421 파서 - 잘못된 입력 처리")

		inputs := []struct {
			value       string
			description string
		}{
			{`sig1=("@method"`, "닫는 괄호 누락"},
			{`sig1="key=val"`, "RFC 8941 형식 아님"},
			{`sig1=(method)`, "따옴표 누락"},
			{`sig1=("@method";keyid="x"`, "잘못된 파라미터 형식"},
		}

		helpers.LogDetail(t, "잘못된 입력 %d개 테스트", len(inputs))

		errorCount := 0
		for i, input := range inputs {
			helpers.LogDetail(t, "테스트 케이스 %d: %s", i+1, input.description)
			helpers.LogDetail(t, "  입력: %s", input.value)

			_, err := ParseSignatureInput(input.value)
			assert.Error(t, err, "입력이 실패해야 함: %s", input.value)

			if err != nil {
				errorCount++
				helpers.LogSuccess(t, "예상대로 에러 발생")
				helpers.LogDetail(t, "  에러: %s", err.Error())
			}
		}

		helpers.LogSuccess(t, "모든 잘못된 입력에 대해 에러 처리 완료")
		helpers.LogDetail(t, "총 에러 발생 개수: %d/%d", errorCount, len(inputs))

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"닫는 괄호 누락 감지",
			"RFC 8941 형식 위반 감지",
			"따옴표 누락 감지",
			"잘못된 파라미터 형식 감지",
			"모든 잘못된 입력에 대해 에러 발생",
			"에러 처리가 올바르게 동작",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":     "13.1.3_RFC9421_파서_잘못된_입력",
			"test_cases":    len(inputs),
			"errors_caught": errorCount,
			"validation":    "에러_처리_검증_통과",
		}
		helpers.SaveTestData(t, "rfc9421/parser_malformed.json", testData)
	})
}

func TestParseSignature(t *testing.T) {
	// Test 1.2.1: Basic parsing
	t.Run("basic parsing", func(t *testing.T) {
		input := `sig1=:MEUCIQDkjN/g30k+A5U9F+a9ZcR6s5wzO8Y8Z8Y8Z8Y8Z8Y8ZwIgIiRBBBBCR4o/1eXgZQRGJwZBRxNf9Z6Hm3AmjZoU4w8=:`

		result, err := ParseSignature(input)
		require.NoError(t, err)
		require.NotNil(t, result)

		sig1, exists := result["sig1"]
		require.True(t, exists)
		assert.NotEmpty(t, sig1)

		// Verify it's valid base64
		assert.Greater(t, len(sig1), 0)
	})

	// Test 1.3.2: Invalid base64
	t.Run("invalid base64", func(t *testing.T) {
		input := `sig1=:invalid-base64!@#$%^&*():`

		_, err := ParseSignature(input)
		assert.Error(t, err)
		// Should contain either "base64" or "byte sequence"
		assert.True(t, strings.Contains(err.Error(), "base64") || strings.Contains(err.Error(), "byte sequence"))
	})

	// Test multiple signatures
	t.Run("multiple signatures", func(t *testing.T) {
		input := `sig1=:MEUCIQDkjN/g30k+A5U9F+a9ZcR6s5wzO8Y8Z8Y8Z8Y8Z8Y8ZwIgIiRBBBBCR4o/1eXgZQRGJwZBRxNf9Z6Hm3AmjZoU4w8=:, sig2=:AQIDBAU=:`

		result, err := ParseSignature(input)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Len(t, result, 2)
		assert.NotEmpty(t, result["sig1"])
		assert.NotEmpty(t, result["sig2"])
	})
}

func TestParseQueryParam(t *testing.T) {
	tests := []struct {
		name      string
		component string
		expected  string
		hasError  bool
	}{
		{
			name:      "valid query param",
			component: `"@query-param";name="id"`,
			expected:  "id",
			hasError:  false,
		},
		{
			name:      "with extra spaces",
			component: `"@query-param" ; name = "format"`,
			expected:  "format",
			hasError:  false,
		},
		{
			name:      "missing name parameter",
			component: `"@query-param"`,
			expected:  "",
			hasError:  true,
		},
		{
			name:      "invalid format",
			component: `@query-param;name=id`,
			expected:  "",
			hasError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseQueryParam(tt.component)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
