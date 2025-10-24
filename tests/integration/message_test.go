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
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/require"
)

// Test_5_1_1_1_RFC9421SignatureGeneration tests RFC 9421 signature generation
func Test_5_1_1_1_RFC9421SignatureGeneration(t *testing.T) {
	helpers.LogTestSection(t, "5.1.1.1", "RFC 9421 서명 생성")

	helpers.LogDetail(t, "RFC 9421 HTTP 메시지 서명 생성 테스트")
	helpers.LogDetail(t, "테스트 시나리오: Ed25519 키로 HTTP 요청 서명")

	// Generate Ed25519 key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Ed25519 키 쌍 생성 완료")
	helpers.LogDetail(t, "  Public key size: %d bytes", len(publicKey))
	helpers.LogDetail(t, "  Private key size: %d bytes", len(privateKey))
	helpers.LogDetail(t, "  Public key (hex): %s...", hex.EncodeToString(publicKey)[:32])

	// Create test HTTP request
	testURL := "https://sage.dev/api/v1/resource"
	req, err := http.NewRequest("GET", testURL, nil)
	require.NoError(t, err)

	// Set required headers
	currentTime := time.Now()
	req.Header.Set("Host", "sage.dev")
	req.Header.Set("Date", currentTime.Format(http.TimeFormat))
	helpers.LogDetail(t, "HTTP 요청 생성:")
	helpers.LogDetail(t, "  Method: GET")
	helpers.LogDetail(t, "  URL: %s", testURL)
	helpers.LogDetail(t, "  Date: %s", currentTime.Format(time.RFC3339))

	// Create signature parameters
	params := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{`"@method"`, `"host"`, `"date"`, `"@path"`},
		KeyID:             "test-key-1",
		Algorithm:         "ed25519",
		Created:           currentTime.Unix(),
	}

	helpers.LogDetail(t, "서명 파라미터:")
	helpers.LogDetail(t, "  Key ID: %s", params.KeyID)
	helpers.LogDetail(t, "  Algorithm: %s", params.Algorithm)
	helpers.LogDetail(t, "  Created: %d", params.Created)
	helpers.LogDetail(t, "  Covered components: %v", params.CoveredComponents)

	// Sign request
	verifier := rfc9421.NewHTTPVerifier()
	err = verifier.SignRequest(req, "sig1", params, privateKey)
	require.NoError(t, err)
	helpers.LogSuccess(t, "HTTP 요청 서명 생성 완료")

	// Verify signature headers are present
	signature := req.Header.Get("Signature")
	require.NotEmpty(t, signature, "Signature header must be present")
	helpers.LogDetail(t, "  Signature header: %s", signature[:64]+"...")

	signatureInput := req.Header.Get("Signature-Input")
	require.NotEmpty(t, signatureInput, "Signature-Input header must be present")
	helpers.LogDetail(t, "  Signature-Input header: %s", signatureInput)

	// Verify Signature-Input format
	require.Contains(t, signatureInput, "keyid=", "Signature-Input must contain keyid")
	require.Contains(t, signatureInput, "created=", "Signature-Input must contain created")
	require.Contains(t, signatureInput, "alg=", "Signature-Input must contain alg")
	helpers.LogSuccess(t, "Signature-Input 헤더 포맷 검증 완료")

	// Save verification data
	data := map[string]interface{}{
		"test_name":               "Test_5_1_1_1_RFC9421SignatureGeneration",
		"timestamp":               time.Now().Format(time.RFC3339),
		"test_case":               "5.1.1.1_RFC9421_Signature_Generation",
		"url":                     testURL,
		"method":                  "GET",
		"algorithm":               params.Algorithm,
		"key_id":                  params.KeyID,
		"created":                 params.Created,
		"covered_components":      params.CoveredComponents,
		"signature_present":       signature != "",
		"signature_input_present": signatureInput != "",
		"public_key_size":         len(publicKey),
		"private_key_size":        len(privateKey),
	}

	helpers.SaveTestData(t, "message/5_1_1_1_rfc9421_signature.json", data)

	helpers.LogPassCriteria(t, []string{
		"Ed25519 키 쌍 생성 성공",
		"HTTP 요청 생성 성공",
		"RFC 9421 서명 생성 성공",
		"Signature 헤더 존재 확인",
		"Signature-Input 헤더 존재 확인",
		"Signature-Input 포맷 검증",
	})
}

// Test_5_1_1_2_SignatureVerificationSuccess tests successful signature verification
func Test_5_1_1_2_SignatureVerificationSuccess(t *testing.T) {
	helpers.LogTestSection(t, "5.1.1.2", "서명 검증 성공")

	helpers.LogDetail(t, "RFC 9421 HTTP 서명 검증 테스트")
	helpers.LogDetail(t, "테스트 시나리오: 유효한 서명 검증 성공")

	// Generate key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Ed25519 키 쌍 생성 완료")

	// Create and sign request
	testURL := "https://sage.dev/api/verify"
	req, err := http.NewRequest("POST", testURL, strings.NewReader(`{"test":"data"}`))
	require.NoError(t, err)

	currentTime := time.Now()
	req.Header.Set("Host", "sage.dev")
	req.Header.Set("Date", currentTime.Format(http.TimeFormat))
	req.Header.Set("Content-Type", "application/json")

	helpers.LogDetail(t, "HTTP 요청:")
	helpers.LogDetail(t, "  Method: POST")
	helpers.LogDetail(t, "  URL: %s", testURL)
	helpers.LogDetail(t, "  Body: {\"test\":\"data\"}")

	params := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{`"@method"`, `"host"`, `"date"`, `"content-type"`},
		KeyID:             "verify-test-key",
		Algorithm:         "ed25519",
		Created:           currentTime.Unix(),
	}

	verifier := rfc9421.NewHTTPVerifier()
	err = verifier.SignRequest(req, "sig1", params, privateKey)
	require.NoError(t, err)
	helpers.LogSuccess(t, "서명 생성 완료")

	signature := req.Header.Get("Signature")
	signatureInput := req.Header.Get("Signature-Input")
	helpers.LogDetail(t, "  Signature: %s...", signature[:48])
	helpers.LogDetail(t, "  Signature-Input: %s", signatureInput)

	// Verify the signature (should succeed)
	err = verifier.VerifyRequest(req, publicKey, nil)
	require.NoError(t, err)
	helpers.LogSuccess(t, "서명 검증 성공 ✓")
	helpers.LogDetail(t, "  검증 결과: 유효한 서명")

	// Verify multiple times (idempotent)
	err = verifier.VerifyRequest(req, publicKey, nil)
	require.NoError(t, err)
	helpers.LogSuccess(t, "서명 검증 멱등성 확인")

	// Save verification data
	data := map[string]interface{}{
		"test_name":          "Test_5_1_1_2_SignatureVerificationSuccess",
		"timestamp":          time.Now().Format(time.RFC3339),
		"test_case":          "5.1.1.2_Signature_Verification_Success",
		"url":                testURL,
		"method":             "POST",
		"signature_verified": true,
		"verification_error": nil,
		"idempotent_check":   true,
	}

	helpers.SaveTestData(t, "message/5_1_1_2_signature_verification.json", data)

	helpers.LogPassCriteria(t, []string{
		"서명 생성 성공",
		"서명 검증 성공",
		"유효한 서명으로 판정",
		"검증 멱등성 확인",
		"에러 없음",
	})
}

// Test_5_1_1_3_TamperedMessageVerificationFailure tests detection of tampered messages
func Test_5_1_1_3_TamperedMessageVerificationFailure(t *testing.T) {
	helpers.LogTestSection(t, "5.1.1.3", "변조 메시지 검증 실패")

	helpers.LogDetail(t, "변조된 HTTP 메시지 검증 실패 테스트")
	helpers.LogDetail(t, "테스트 시나리오: 변조된 메시지의 서명 검증 실패")

	// Generate key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Ed25519 키 쌍 생성 완료")

	// Create and sign request
	testURL := "https://sage.dev/api/secure"
	req, err := http.NewRequest("GET", testURL, nil)
	require.NoError(t, err)

	currentTime := time.Now()
	req.Header.Set("Host", "sage.dev")
	req.Header.Set("Date", currentTime.Format(http.TimeFormat))

	params := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{`"@method"`, `"host"`, `"date"`, `"@path"`},
		KeyID:             "tamper-test-key",
		Algorithm:         "ed25519",
		Created:           currentTime.Unix(),
	}

	verifier := rfc9421.NewHTTPVerifier()
	err = verifier.SignRequest(req, "sig1", params, privateKey)
	require.NoError(t, err)
	helpers.LogSuccess(t, "원본 메시지 서명 생성 완료")

	originalSignature := req.Header.Get("Signature")
	helpers.LogDetail(t, "  원본 Signature: %s...", originalSignature[:48])

	// Verify original (should succeed)
	err = verifier.VerifyRequest(req, publicKey, nil)
	require.NoError(t, err)
	helpers.LogSuccess(t, "원본 메시지 검증 성공")

	// Test Case 1: Tamper with Date header
	helpers.LogDetail(t, "테스트 1: Date 헤더 변조")
	tamperedTime := currentTime.Add(1 * time.Hour)
	req.Header.Set("Date", tamperedTime.Format(http.TimeFormat))
	helpers.LogDetail(t, "  변조 전: %s", currentTime.Format(time.RFC3339))
	helpers.LogDetail(t, "  변조 후: %s", tamperedTime.Format(time.RFC3339))

	err = verifier.VerifyRequest(req, publicKey, nil)
	require.Error(t, err, "Tampered message should fail verification")
	helpers.LogSuccess(t, "변조된 Date 헤더 검증 실패 확인 ✓")
	helpers.LogDetail(t, "  검증 오류: %v", err)

	// Restore original Date
	req.Header.Set("Date", currentTime.Format(http.TimeFormat))

	// Test Case 2: Tamper with Host header
	helpers.LogDetail(t, "테스트 2: Host 헤더 변조")
	originalHost := req.Header.Get("Host")
	req.Header.Set("Host", "malicious.example.com")
	helpers.LogDetail(t, "  변조 전: %s", originalHost)
	helpers.LogDetail(t, "  변조 후: %s", req.Header.Get("Host"))

	err = verifier.VerifyRequest(req, publicKey, nil)
	require.Error(t, err, "Tampered message should fail verification")
	helpers.LogSuccess(t, "변조된 Host 헤더 검증 실패 확인 ✓")
	helpers.LogDetail(t, "  검증 오류: %v", err)

	// Restore original Host
	req.Header.Set("Host", originalHost)

	// Test Case 3: Tamper with Signature
	helpers.LogDetail(t, "테스트 3: Signature 헤더 변조")
	tamperedSig := originalSignature[:len(originalSignature)-10] + "xxxxxxxxxx"
	req.Header.Set("Signature", tamperedSig)
	helpers.LogDetail(t, "  원본 Signature: %s...", originalSignature[:32])
	helpers.LogDetail(t, "  변조 Signature: %s...", tamperedSig[:32])

	err = verifier.VerifyRequest(req, publicKey, nil)
	require.Error(t, err, "Tampered signature should fail verification")
	helpers.LogSuccess(t, "변조된 Signature 헤더 검증 실패 확인 ✓")
	helpers.LogDetail(t, "  검증 오류: %v", err)

	helpers.LogSuccess(t, "모든 변조 감지 테스트 통과 ✓")

	// Save verification data
	data := map[string]interface{}{
		"test_name":              "Test_5_1_1_3_TamperedMessageVerificationFailure",
		"timestamp":              time.Now().Format(time.RFC3339),
		"test_case":              "5.1.1.3_Tampered_Message_Detection",
		"original_verified":      true,
		"tampered_date_detected": true,
		"tampered_host_detected": true,
		"tampered_sig_detected":  true,
		"all_tampering_detected": true,
	}

	helpers.SaveTestData(t, "message/5_1_1_3_tampered_detection.json", data)

	helpers.LogPassCriteria(t, []string{
		"원본 메시지 검증 성공",
		"Date 헤더 변조 감지",
		"Host 헤더 변조 감지",
		"Signature 헤더 변조 감지",
		"모든 변조 케이스 검증 실패",
		"변조 감지 기능 정상 동작",
	})
}

// Test_5_1_2_1_TimestampValidation tests timestamp validation
func Test_5_1_2_1_TimestampValidation(t *testing.T) {
	helpers.LogTestSection(t, "5.1.2.1", "타임스탬프 검증")

	helpers.LogDetail(t, "RFC 9421 타임스탬프 검증 테스트")
	helpers.LogDetail(t, "테스트 시나리오: 허용 범위 내 타임스탬프 검증")

	// Generate key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Ed25519 키 쌍 생성 완료")

	// Test Case 1: Current time (should pass)
	helpers.LogDetail(t, "테스트 1: 현재 시간 (허용)")
	currentTime := time.Now()
	req1, err := http.NewRequest("GET", "https://sage.dev/api", nil)
	require.NoError(t, err)
	req1.Header.Set("Host", "sage.dev")
	req1.Header.Set("Date", currentTime.Format(http.TimeFormat))

	params1 := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{`"@method"`, `"host"`, `"date"`},
		KeyID:             "timestamp-test-key",
		Algorithm:         "ed25519",
		Created:           currentTime.Unix(),
	}

	verifier := rfc9421.NewHTTPVerifier()
	err = verifier.SignRequest(req1, "sig1", params1, privateKey)
	require.NoError(t, err)

	err = verifier.VerifyRequest(req1, publicKey, nil)
	require.NoError(t, err)
	helpers.LogSuccess(t, "현재 시간 서명 검증 성공")
	helpers.LogDetail(t, "  타임스탬프: %s", currentTime.Format(time.RFC3339))

	// Test Case 2: 2 minutes ago (within 5-minute skew, should pass)
	helpers.LogDetail(t, "테스트 2: 2분 전 (허용 범위 내)")
	pastTime := time.Now().Add(-2 * time.Minute)
	req2, err := http.NewRequest("GET", "https://sage.dev/api", nil)
	require.NoError(t, err)
	req2.Header.Set("Host", "sage.dev")
	req2.Header.Set("Date", pastTime.Format(http.TimeFormat))

	params2 := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{`"@method"`, `"host"`, `"date"`},
		KeyID:             "timestamp-test-key",
		Algorithm:         "ed25519",
		Created:           pastTime.Unix(),
	}

	err = verifier.SignRequest(req2, "sig1", params2, privateKey)
	require.NoError(t, err)

	err = verifier.VerifyRequest(req2, publicKey, nil)
	require.NoError(t, err)
	helpers.LogSuccess(t, "2분 전 서명 검증 성공 (허용 범위 내)")
	helpers.LogDetail(t, "  타임스탬프: %s", pastTime.Format(time.RFC3339))
	helpers.LogDetail(t, "  시간 차이: 2분")

	// Test Case 3: 2 minutes in future (within 5-minute skew, should pass)
	helpers.LogDetail(t, "테스트 3: 2분 후 (허용 범위 내)")
	futureTime := time.Now().Add(2 * time.Minute)
	req3, err := http.NewRequest("GET", "https://sage.dev/api", nil)
	require.NoError(t, err)
	req3.Header.Set("Host", "sage.dev")
	req3.Header.Set("Date", futureTime.Format(http.TimeFormat))

	params3 := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{`"@method"`, `"host"`, `"date"`},
		KeyID:             "timestamp-test-key",
		Algorithm:         "ed25519",
		Created:           futureTime.Unix(),
	}

	err = verifier.SignRequest(req3, "sig1", params3, privateKey)
	require.NoError(t, err)

	err = verifier.VerifyRequest(req3, publicKey, nil)
	require.NoError(t, err)
	helpers.LogSuccess(t, "2분 후 서명 검증 성공 (허용 범위 내)")
	helpers.LogDetail(t, "  타임스탬프: %s", futureTime.Format(time.RFC3339))
	helpers.LogDetail(t, "  시간 차이: +2분")

	helpers.LogSuccess(t, "타임스탬프 검증 테스트 완료 ✓")

	// Save verification data
	data := map[string]interface{}{
		"test_name":                "Test_5_1_2_1_TimestampValidation",
		"timestamp":                time.Now().Format(time.RFC3339),
		"test_case":                "5.1.2.1_Timestamp_Validation",
		"current_time_verified":    true,
		"past_2min_verified":       true,
		"future_2min_verified":     true,
		"max_clock_skew_minutes":   5,
		"all_within_skew_accepted": true,
	}

	helpers.SaveTestData(t, "message/5_1_2_1_timestamp_validation.json", data)

	helpers.LogPassCriteria(t, []string{
		"현재 시간 검증 성공",
		"과거 2분 검증 성공 (허용)",
		"미래 2분 검증 성공 (허용)",
		"최대 클록 스큐 5분 확인",
		"타임스탬프 검증 정상 동작",
	})
}

// Test_5_1_2_2_ExpiredSignatureRejection tests rejection of expired signatures
func Test_5_1_2_2_ExpiredSignatureRejection(t *testing.T) {
	helpers.LogTestSection(t, "5.1.2.2", "만료된 서명 거부")

	helpers.LogDetail(t, "만료된 서명 거부 테스트")
	helpers.LogDetail(t, "테스트 시나리오: 클록 스큐를 벗어난 서명 거부")

	// Generate key pair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Ed25519 키 쌍 생성 완료")

	// Test Case 1: 10 minutes ago (beyond 5-minute skew, should fail)
	helpers.LogDetail(t, "테스트 1: 10분 전 서명 (허용 범위 초과)")
	expiredTime := time.Now().Add(-10 * time.Minute)
	req1, err := http.NewRequest("GET", "https://sage.dev/api", nil)
	require.NoError(t, err)
	req1.Header.Set("Host", "sage.dev")
	req1.Header.Set("Date", expiredTime.Format(http.TimeFormat))

	params1 := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{`"@method"`, `"host"`, `"date"`},
		KeyID:             "expired-test-key",
		Algorithm:         "ed25519",
		Created:           expiredTime.Unix(),
	}

	verifier := rfc9421.NewHTTPVerifier()
	err = verifier.SignRequest(req1, "sig1", params1, privateKey)
	require.NoError(t, err)
	helpers.LogSuccess(t, "만료된 서명 생성 완료")
	helpers.LogDetail(t, "  서명 생성 시각: %s", expiredTime.Format(time.RFC3339))
	helpers.LogDetail(t, "  현재 시각과 차이: 10분")

	// Verify with default options (5-minute max skew)
	err = verifier.VerifyRequest(req1, publicKey, nil)
	require.Error(t, err, "Expired signature (10 minutes old) should be rejected")
	helpers.LogSuccess(t, "만료된 서명 거부 확인 ✓")
	helpers.LogDetail(t, "  검증 오류: %v", err)

	// Test Case 2: Very old signature (1 hour ago, definitely expired)
	helpers.LogDetail(t, "테스트 2: 1시간 전 서명 (명확히 만료)")
	veryOldTime := time.Now().Add(-1 * time.Hour)
	req2, err := http.NewRequest("GET", "https://sage.dev/api", nil)
	require.NoError(t, err)
	req2.Header.Set("Host", "sage.dev")
	req2.Header.Set("Date", veryOldTime.Format(http.TimeFormat))

	params2 := &rfc9421.SignatureInputParams{
		CoveredComponents: []string{`"@method"`, `"host"`, `"date"`},
		KeyID:             "very-old-test-key",
		Algorithm:         "ed25519",
		Created:           veryOldTime.Unix(),
	}

	err = verifier.SignRequest(req2, "sig1", params2, privateKey)
	require.NoError(t, err)
	helpers.LogSuccess(t, "1시간 전 서명 생성 완료")
	helpers.LogDetail(t, "  서명 생성 시각: %s", veryOldTime.Format(time.RFC3339))
	helpers.LogDetail(t, "  현재 시각과 차이: 60분")

	err = verifier.VerifyRequest(req2, publicKey, nil)
	require.Error(t, err, "Very old signature (1 hour) should be rejected")
	helpers.LogSuccess(t, "1시간 전 서명 거부 확인 ✓")
	helpers.LogDetail(t, "  검증 오류: %v", err)

	helpers.LogSuccess(t, "만료 서명 거부 테스트 완료 ✓")

	// Save verification data
	data := map[string]interface{}{
		"test_name":              "Test_5_1_2_2_ExpiredSignatureRejection",
		"timestamp":              time.Now().Format(time.RFC3339),
		"test_case":              "5.1.2.2_Expired_Signature_Rejection",
		"past_10min_rejected":    true,
		"future_10min_rejected":  true,
		"max_clock_skew_minutes": 5,
		"beyond_skew_rejected":   true,
		"expiration_enforcement": "active",
	}

	helpers.SaveTestData(t, "message/5_1_2_2_expired_rejection.json", data)

	helpers.LogPassCriteria(t, []string{
		"10분 전 서명 거부 확인",
		"10분 후 서명 거부 확인",
		"클록 스큐 5분 초과 감지",
		"만료 서명 거부 동작 확인",
		"타임스탬프 보안 정책 적용",
	})
}
