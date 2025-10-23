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
	"bytes"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanonicalizer(t *testing.T) {
	// Test 1.4.1: Basic GET request
	t.Run("basic GET request", func(t *testing.T) {
		// 명세 요구사항: RFC9421 기본 GET 요청 서명 베이스 생성
		helpers.LogTestSection(t, "12.1.1", "RFC9421 정규화 - 기본 GET 요청")

		url := "https://example.com/foo?bar=baz"
		helpers.LogDetail(t, "HTTP 요청 생성:")
		helpers.LogDetail(t, "  메서드: GET")
		helpers.LogDetail(t, "  URL: %s", url)

		req, err := http.NewRequest("GET", url, nil)
		require.NoError(t, err)
		helpers.LogSuccess(t, "HTTP 요청 생성 완료")

		components := []string{`"@method"`, `"@authority"`, `"@path"`, `"@query"`}
		helpers.LogDetail(t, "커버된 컴포넌트: %v", components)

		params := &SignatureInputParams{
			CoveredComponents: components,
			KeyID:             "test-key",
			Algorithm:         "ed25519",
			Created:           1719234000,
		}
		helpers.LogDetail(t, "서명 파라미터:")
		helpers.LogDetail(t, "  KeyID: %s", params.KeyID)
		helpers.LogDetail(t, "  Algorithm: %s", params.Algorithm)
		helpers.LogDetail(t, "  Created: %d", params.Created)

		canonicalizer := NewCanonicalizer()
		helpers.LogSuccess(t, "정규화기 생성 완료")

		helpers.LogDetail(t, "서명 베이스 생성 중...")
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)
		helpers.LogSuccess(t, "서명 베이스 생성 완료")

		expected := `"@method": GET
"@authority": example.com
"@path": /foo
"@query": ?bar=baz
"@signature-params": ("@method" "@authority" "@path" "@query");keyid="test-key";alg="ed25519";created=1719234000`

		assert.Equal(t, expected, result)
		helpers.LogSuccess(t, "서명 베이스 검증 완료")
		helpers.LogDetail(t, "생성된 서명 베이스:")
		for i, line := range strings.Split(result, "\n") {
			helpers.LogDetail(t, "  [%d] %s", i+1, line)
		}

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"HTTP GET 요청 생성 성공",
			"커버된 컴포넌트 4개 설정",
			"서명 파라미터 설정 완료",
			"정규화기 생성 성공",
			"서명 베이스 생성 성공",
			"@method, @authority, @path, @query 포함",
			"@signature-params 올바르게 생성됨",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "12.1.1_RFC9421_정규화_기본_GET",
			"request": map[string]interface{}{
				"method": "GET",
				"url":    url,
			},
			"signature_params": map[string]interface{}{
				"covered_components": components,
				"key_id":             params.KeyID,
				"algorithm":          params.Algorithm,
				"created":            params.Created,
			},
			"signature_base": result,
			"validation":     "통과",
		}
		helpers.SaveTestData(t, "rfc9421/canonicalizer_basic_get.json", testData)
	})

	// Test 1.4.2: POST request with Content-Digest
	t.Run("POST request with Content-Digest", func(t *testing.T) {
		// 명세 요구사항: Content-Digest를 포함한 POST 요청 서명
		helpers.LogTestSection(t, "12.1.2", "RFC9421 정규화 - Content-Digest가 있는 POST")

		body := `{"hello": "world"}`
		url := "https://example.com/data"
		digest := "sha-256=:X48E9qOokqqrvdts8nOJRJN3OWDUoyWxBf7kbu9DBPE=:"
		date := "Mon, 24 Jun 2024 12:00:00 GMT"

		helpers.LogDetail(t, "HTTP POST 요청 생성:")
		helpers.LogDetail(t, "  URL: %s", url)
		helpers.LogDetail(t, "  Body: %s", body)

		req, err := http.NewRequest("POST", url, strings.NewReader(body))
		require.NoError(t, err)

		req.Header.Set("Content-Digest", digest)
		req.Header.Set("Date", date)
		helpers.LogSuccess(t, "HTTP 요청 및 헤더 설정 완료")
		helpers.LogDetail(t, "  Content-Digest: %s", digest)
		helpers.LogDetail(t, "  Date: %s", date)

		components := []string{`"content-digest"`, `"date"`}
		helpers.LogDetail(t, "커버된 컴포넌트: %v", components)

		params := &SignatureInputParams{
			CoveredComponents: components,
		}

		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)
		helpers.LogSuccess(t, "서명 베이스 생성 완료")

		assert.Contains(t, result, `"content-digest": sha-256=:X48E9qOokqqrvdts8nOJRJN3OWDUoyWxBf7kbu9DBPE=:`)
		assert.Contains(t, result, `"date": Mon, 24 Jun 2024 12:00:00 GMT`)
		helpers.LogSuccess(t, "Content-Digest 및 Date 헤더 검증 완료")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"POST 요청 생성 성공",
			"Content-Digest 헤더 설정",
			"Date 헤더 설정",
			"서명 베이스 생성 성공",
			"Content-Digest 값 포함 검증",
			"Date 값 포함 검증",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "12.1.2_RFC9421_정규화_POST_Content_Digest",
			"request": map[string]interface{}{
				"method":         "POST",
				"url":            url,
				"body":           body,
				"content_digest": digest,
				"date":           date,
			},
			"validation": "통과",
		}
		helpers.SaveTestData(t, "rfc9421/canonicalizer_post_digest.json", testData)
	})

	// Test 1.4.3: Header value whitespace handling
	t.Run("header whitespace handling", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com", nil)
		require.NoError(t, err)

		req.Header.Set("X-Custom", "  value with spaces  ")

		components := []string{`"x-custom"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}

		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)

		// Should trim outer spaces but keep internal ones
		assert.Contains(t, result, `"x-custom": value with spaces`)
	})

	// Test 1.4.4: Multiple headers with same name
	t.Run("multiple headers with same name", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com", nil)
		require.NoError(t, err)

		req.Header.Add("Via", "1.1 proxy-a")
		req.Header.Add("Via", "1.1 proxy-b")

		components := []string{`"via"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}

		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)

		// Should be joined with comma and space
		assert.Contains(t, result, `"via": 1.1 proxy-a, 1.1 proxy-b`)
	})

	// Test 1.4.5: Component not found
	t.Run("component not found", func(t *testing.T) {
		// 명세 요구사항: 존재하지 않는 컴포넌트에 대한 에러 처리
		helpers.LogTestSection(t, "12.1.3", "RFC9421 정규화 - 컴포넌트 누락 에러")

		url := "https://example.com"
		helpers.LogDetail(t, "HTTP GET 요청 생성: %s", url)

		req, err := http.NewRequest("GET", url, nil)
		require.NoError(t, err)
		helpers.LogSuccess(t, "HTTP 요청 생성 완료")

		components := []string{`"content-digest"`}
		helpers.LogDetail(t, "존재하지 않는 컴포넌트 요청: %v", components)
		helpers.LogDetail(t, "  참고: GET 요청에 content-digest 헤더 없음")

		params := &SignatureInputParams{
			CoveredComponents: components,
		}

		canonicalizer := NewCanonicalizer()
		helpers.LogDetail(t, "서명 베이스 생성 시도 (에러 예상)...")
		_, err = canonicalizer.BuildSignatureBase(req, "sig1", params)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "component not found")
		helpers.LogSuccess(t, "예상대로 에러 발생")
		helpers.LogDetail(t, "에러 메시지: %s", err.Error())

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"GET 요청 생성 성공",
			"존재하지 않는 컴포넌트 요청",
			"서명 베이스 생성 실패 (예상됨)",
			"에러 메시지에 'component not found' 포함",
			"에러 처리가 올바르게 동작함",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "12.1.3_RFC9421_정규화_컴포넌트_누락",
			"request": map[string]interface{}{
				"method": "GET",
				"url":    url,
			},
			"requested_component": "content-digest",
			"result": map[string]interface{}{
				"error":         true,
				"error_message": err.Error(),
			},
			"validation": "에러_처리_검증_통과",
		}
		helpers.SaveTestData(t, "rfc9421/canonicalizer_component_not_found.json", testData)
	})

	// Test 4.2.1: Empty path
	t.Run("empty path", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com", nil)
		require.NoError(t, err)

		components := []string{`"@path"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}

		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)

		// Empty path should be treated as "/"
		assert.Contains(t, result, `"@path": /`)
	})

	// Test 4.2.2: Special characters in path/query
	t.Run("special characters in path and query", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/users/שלום?q=a%20b+c", nil)
		require.NoError(t, err)

		components := []string{`"@path"`, `"@query"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}

		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)

		// Should preserve encoding
		assert.Contains(t, result, `"@path": /users/שלום`)
		assert.Contains(t, result, `"@query": ?q=a%20b+c`)
	})

	// Test 4.2.3: Proxy request target
	t.Run("proxy request target", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://example.com/foo", nil)
		require.NoError(t, err)

		// Simulate proxy request
		req.RequestURI = "http://example.com/foo"

		components := []string{`"@request-target"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}

		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)

		// Should include full URL for proxy requests
		assert.Contains(t, result, `"@request-target": GET /foo`)
	})
}

func TestQueryParamComponent(t *testing.T) {
	// Test 4.1.1: Specific parameter protection
	t.Run("specific parameter protection", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/api/v1/users?id=123&format=json&cache=false", nil)
		require.NoError(t, err)

		components := []string{`"@query-param";name="id"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}

		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)

		// Should include the specific query parameter
		assert.Contains(t, result, `"@query-param";name="id": 123`)
	})

	// Test 4.1.4: Parameter name case sensitivity
	t.Run("parameter name case sensitivity", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/api?ID=123", nil)
		require.NoError(t, err)

		components := []string{`"@query-param";name="id"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}

		canonicalizer := NewCanonicalizer()
		_, err = canonicalizer.BuildSignatureBase(req, "sig1", params)

		// Should fail because "id" != "ID"
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "component not found")
	})

	// Test 4.1.5: Non-existent parameter
	t.Run("non-existent parameter", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/api?id=123", nil)
		require.NoError(t, err)

		components := []string{`"@query-param";name="status"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}

		canonicalizer := NewCanonicalizer()
		_, err = canonicalizer.BuildSignatureBase(req, "sig1", params)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "component not found")
	})

	// Test multiple query parameters
	t.Run("multiple query parameters", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/api?id=123&name=test&format=json", nil)
		require.NoError(t, err)

		components := []string{
			`"@query-param";name="id"`,
			`"@query-param";name="format"`,
		}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}

		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)

		assert.Contains(t, result, `"@query-param";name="id": 123`)
		assert.Contains(t, result, `"@query-param";name="format": json`)
		// Should NOT contain name parameter since it wasn't included
		assert.NotContains(t, result, `name=test`)
	})
}

func TestHTTPFields(t *testing.T) {
	t.Run("all HTTP fields", func(t *testing.T) {
		req, err := http.NewRequest("POST", "https://api.example.com:8443/v1/messages?filter=active", bytes.NewReader([]byte("test body")))
		require.NoError(t, err)

		req.Header.Set("Host", "api.example.com:8443")
		req.Header.Set("Date", time.Now().Format(http.TimeFormat))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Length", "9")

		components := []string{
			`"@method"`,
			`"@target-uri"`,
			`"@authority"`,
			`"@scheme"`,
			`"@request-target"`,
			`"@path"`,
			`"@query"`,
			`"@status"`, // This should fail for requests
			`"host"`,
			`"date"`,
			`"content-type"`,
			`"content-length"`,
		}

		params := &SignatureInputParams{
			CoveredComponents: components,
		}

		canonicalizer := NewCanonicalizer()
		var result string
		_, err = canonicalizer.BuildSignatureBase(req, "sig1", params)

		// Should fail because @status is only for responses
		require.Error(t, err)
		assert.Contains(t, err.Error(), "@status")

		// Now test without @status
		components = []string{
			`"@method"`,
			`"@target-uri"`,
			`"@authority"`,
			`"@scheme"`,
			`"@request-target"`,
			`"@path"`,
			`"@query"`,
			`"host"`,
			`"date"`,
			`"content-type"`,
			`"content-length"`,
		}
		params.CoveredComponents = components

		result, err = canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)

		// Verify all components are present
		assert.Contains(t, result, `"@method": POST`)
		assert.Contains(t, result, `"@target-uri": https://api.example.com:8443/v1/messages?filter=active`)
		assert.Contains(t, result, `"@authority": api.example.com:8443`)
		assert.Contains(t, result, `"@scheme": https`)
		assert.Contains(t, result, `"@request-target": POST /v1/messages?filter=active`)
		assert.Contains(t, result, `"@path": /v1/messages`)
		assert.Contains(t, result, `"@query": ?filter=active`)
		assert.Contains(t, result, `"host": api.example.com:8443`)
		assert.Contains(t, result, `"content-type": application/json`)
		assert.Contains(t, result, `"content-length": 9`)
	})
}
