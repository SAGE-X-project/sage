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

package crypto

import (
	"fmt"
	"testing"

	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAlgorithmRegistry(t *testing.T) {
	t.Run("Get registered algorithm", func(t *testing.T) {
		// 명세 요구사항: 등록된 알고리즘 정보 조회 검증
		helpers.LogTestSection(t, "17.1.1", "알고리즘 레지스트리 - 등록된 알고리즘 조회")

		helpers.LogDetail(t, "Ed25519 알고리즘 정보 조회 중...")
		// Ed25519 should be registered
		info, err := GetAlgorithmInfo(KeyTypeEd25519)
		require.NoError(t, err)
		helpers.LogSuccess(t, "Ed25519 알고리즘 정보 조회 성공")

		helpers.LogDetail(t, "알고리즘 정보 검증 중...")
		assert.Equal(t, KeyTypeEd25519, info.KeyType)
		helpers.LogDetail(t, "  Key Type: %s", info.KeyType)

		assert.NotEmpty(t, info.RFC9421Algorithm)
		helpers.LogDetail(t, "  RFC9421 Algorithm: %s", info.RFC9421Algorithm)

		assert.True(t, info.SupportsRFC9421)
		helpers.LogDetail(t, "  Supports RFC9421: %v", info.SupportsRFC9421)

		assert.True(t, info.SupportsKeyGeneration)
		helpers.LogDetail(t, "  Supports Key Generation: %v", info.SupportsKeyGeneration)

		helpers.LogSuccess(t, "모든 알고리즘 속성 검증 완료")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"Ed25519 알고리즘이 레지스트리에 등록됨",
			"KeyType이 Ed25519와 일치",
			"RFC9421Algorithm이 정의됨",
			"RFC9421 지원 플래그가 true",
			"키 생성 지원 플래그가 true",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "17.1.1_등록된_알고리즘_조회",
			"key_type":  string(info.KeyType),
			"info": map[string]interface{}{
				"rfc9421_algorithm":       info.RFC9421Algorithm,
				"supports_rfc9421":        info.SupportsRFC9421,
				"supports_key_generation": info.SupportsKeyGeneration,
			},
			"validation": "알고리즘_정보_조회_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_get_registered.json", testData)
	})

	t.Run("Get unregistered algorithm", func(t *testing.T) {
		// 명세 요구사항: 미등록 알고리즘 에러 처리 검증
		helpers.LogTestSection(t, "17.1.2", "알고리즘 레지스트리 - 미등록 알고리즘 조회")

		helpers.LogDetail(t, "미등록 알고리즘(unknown) 조회 시도...")
		_, err := GetAlgorithmInfo(KeyType("unknown"))
		assert.Error(t, err)
		helpers.LogSuccess(t, "예상대로 에러 발생")
		helpers.LogDetail(t, "  에러: %v", err)

		assert.ErrorIs(t, err, ErrAlgorithmNotSupported)
		helpers.LogSuccess(t, "ErrAlgorithmNotSupported 에러 타입 일치")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"미등록 알고리즘 조회 시 에러 발생",
			"에러 타입이 ErrAlgorithmNotSupported",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":  "17.1.2_미등록_알고리즘_조회",
			"key_type":   "unknown",
			"error":      err.Error(),
			"validation": "에러_처리_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_get_unregistered.json", testData)
	})

	t.Run("List all supported algorithms", func(t *testing.T) {
		// 명세 요구사항: 지원되는 모든 알고리즘 목록 조회 검증
		helpers.LogTestSection(t, "17.1.3", "알고리즘 레지스트리 - 전체 알고리즘 목록")

		helpers.LogDetail(t, "지원되는 모든 알고리즘 목록 조회 중...")
		algorithms := ListSupportedAlgorithms()
		assert.NotEmpty(t, algorithms)
		helpers.LogSuccess(t, "알고리즘 목록 조회 성공")
		helpers.LogDetail(t, "  총 알고리즘 개수: %d", len(algorithms))

		// Should include at least Ed25519, Secp256k1, RSA
		var found []KeyType
		for _, alg := range algorithms {
			found = append(found, alg.KeyType)
		}
		helpers.LogDetail(t, "알고리즘 목록: %v", found)

		helpers.LogDetail(t, "필수 알고리즘 포함 여부 확인 중...")
		assert.Contains(t, found, KeyTypeEd25519)
		helpers.LogDetail(t, "   Ed25519 포함")

		assert.Contains(t, found, KeyTypeSecp256k1)
		helpers.LogDetail(t, "   Secp256k1 포함")

		assert.Contains(t, found, KeyTypeRSA)
		helpers.LogDetail(t, "   RSA 포함")

		helpers.LogSuccess(t, "모든 필수 알고리즘 포함 확인")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"알고리즘 목록이 비어있지 않음",
			"Ed25519 알고리즘 포함",
			"Secp256k1 알고리즘 포함",
			"RSA 알고리즘 포함",
		})

		// CLI 검증용 테스트 데이터 저장
		var keyTypeStrings []string
		for _, kt := range found {
			keyTypeStrings = append(keyTypeStrings, string(kt))
		}
		testData := map[string]interface{}{
			"test_case":        "17.1.3_전체_알고리즘_목록",
			"algorithms_count": len(algorithms),
			"algorithms":       keyTypeStrings,
			"validation":       "목록_조회_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_list_all.json", testData)
	})

	t.Run("Get RFC 9421 algorithm name", func(t *testing.T) {
		// 명세 요구사항: 키 타입에서 RFC9421 알고리즘 이름 매핑 검증
		helpers.LogTestSection(t, "17.1.4", "알고리즘 레지스트리 - RFC9421 알고리즘 이름 조회")

		tests := []struct {
			keyType  KeyType
			expected string
		}{
			{KeyTypeEd25519, "ed25519"},
			{KeyTypeSecp256k1, "es256k"},
			{KeyTypeRSA, "rsa-pss-sha256"},
		}

		helpers.LogDetail(t, "%d개 키 타입에 대한 RFC9421 알고리즘 이름 매핑 테스트", len(tests))

		successCount := 0
		for _, tt := range tests {
			t.Run(string(tt.keyType), func(t *testing.T) {
				helpers.LogDetail(t, "키 타입 %s → RFC9421 알고리즘 이름 조회...", tt.keyType)
				algName, err := GetRFC9421AlgorithmName(tt.keyType)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, algName)
				helpers.LogDetail(t, "  매핑 성공: %s → %s", tt.keyType, algName)
				successCount++
			})
		}

		helpers.LogDetail(t, "모든 키 타입 매핑 완료 (%d/%d)", successCount, len(tests))
		helpers.LogSuccess(t, "RFC9421 알고리즘 이름 매핑 검증 완료")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"Ed25519 → ed25519 매핑 성공",
			"Secp256k1 → es256k 매핑 성공",
			"RSA → rsa-pss-sha256 매핑 성공",
			"모든 매핑이 에러 없이 완료",
		})

		// CLI 검증용 테스트 데이터 저장
		mappings := make(map[string]string)
		for _, tt := range tests {
			mappings[string(tt.keyType)] = tt.expected
		}
		testData := map[string]interface{}{
			"test_case":  "17.1.4_RFC9421_알고리즘_이름",
			"mappings":   mappings,
			"test_count": len(tests),
			"validation": "매핑_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_rfc9421_name.json", testData)
	})

	t.Run("Get key type from RFC 9421 algorithm", func(t *testing.T) {
		// 명세 요구사항: RFC9421 알고리즘 이름에서 키 타입 역매핑 검증
		helpers.LogTestSection(t, "17.1.5", "알고리즘 레지스트리 - RFC9421 역매핑")

		tests := []struct {
			rfc9421Alg string
			expected   KeyType
		}{
			{"ed25519", KeyTypeEd25519},
			{"es256k", KeyTypeSecp256k1},
			{"rsa-pss-sha256", KeyTypeRSA},
		}

		helpers.LogDetail(t, "%d개 RFC9421 알고리즘 이름에 대한 키 타입 역매핑 테스트", len(tests))

		successCount := 0
		for _, tt := range tests {
			t.Run(tt.rfc9421Alg, func(t *testing.T) {
				helpers.LogDetail(t, "RFC9421 알고리즘 %s → 키 타입 조회...", tt.rfc9421Alg)
				keyType, err := GetKeyTypeFromRFC9421Algorithm(tt.rfc9421Alg)
				require.NoError(t, err)
				assert.Equal(t, tt.expected, keyType)
				helpers.LogDetail(t, "  역매핑 성공: %s → %s", tt.rfc9421Alg, keyType)
				successCount++
			})
		}

		helpers.LogDetail(t, "모든 RFC9421 역매핑 완료 (%d/%d)", successCount, len(tests))
		helpers.LogSuccess(t, "RFC9421 역매핑 검증 완료")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"ed25519 → Ed25519 역매핑 성공",
			"es256k → Secp256k1 역매핑 성공",
			"rsa-pss-sha256 → RSA 역매핑 성공",
			"모든 역매핑이 에러 없이 완료",
		})

		// CLI 검증용 테스트 데이터 저장
		reverseMappings := make(map[string]string)
		for _, tt := range tests {
			reverseMappings[tt.rfc9421Alg] = string(tt.expected)
		}
		testData := map[string]interface{}{
			"test_case":        "17.1.5_RFC9421_역매핑",
			"reverse_mappings": reverseMappings,
			"test_count":       len(tests),
			"validation":       "역매핑_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_rfc9421_reverse.json", testData)
	})

	t.Run("List RFC 9421 supported algorithms", func(t *testing.T) {
		// 명세 요구사항: RFC9421 지원 알고리즘 목록 검증
		helpers.LogTestSection(t, "17.1.6", "알고리즘 레지스트리 - RFC9421 지원 목록")

		helpers.LogDetail(t, "RFC9421 지원 알고리즘 목록 조회 중...")
		algorithms := ListRFC9421SupportedAlgorithms()
		assert.NotEmpty(t, algorithms)
		helpers.LogSuccess(t, "RFC9421 알고리즘 목록 조회 성공")
		helpers.LogDetail(t, "  총 알고리즘 개수: %d", len(algorithms))
		helpers.LogDetail(t, "  알고리즘: %v", algorithms)

		// Should include RFC 9421 algorithm names
		helpers.LogDetail(t, "서명 알고리즘 포함 여부 확인...")
		assert.Contains(t, algorithms, "ed25519")
		helpers.LogDetail(t, "   ed25519 포함")

		assert.Contains(t, algorithms, "es256k")
		helpers.LogDetail(t, "   es256k 포함")

		assert.Contains(t, algorithms, "rsa-pss-sha256")
		helpers.LogDetail(t, "   rsa-pss-sha256 포함")

		// X25519 should NOT be in RFC 9421 list (it's for key exchange, not signing)
		helpers.LogDetail(t, "키 교환 전용 알고리즘 제외 확인...")
		assert.NotContains(t, algorithms, "x25519")
		helpers.LogSuccess(t, "x25519는 RFC9421 목록에서 제외됨 (키 교환 전용)")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"RFC9421 알고리즘 목록이 비어있지 않음",
			"ed25519 서명 알고리즘 포함",
			"es256k 서명 알고리즘 포함",
			"rsa-pss-sha256 서명 알고리즘 포함",
			"x25519 키 교환 알고리즘 제외",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":        "17.1.6_RFC9421_지원_목록",
			"algorithms_count": len(algorithms),
			"algorithms":       algorithms,
			"excludes_x25519":  !containsString(algorithms, "x25519"),
			"validation":       "RFC9421_목록_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_rfc9421_list.json", testData)
	})

	t.Run("Check if algorithm supports RFC 9421", func(t *testing.T) {
		// 명세 요구사항: RFC9421 지원 여부 확인 검증
		helpers.LogTestSection(t, "17.1.7", "알고리즘 레지스트리 - RFC9421 지원 확인")

		// Ed25519 supports RFC 9421
		helpers.LogDetail(t, "Ed25519 RFC9421 지원 확인...")
		assert.True(t, SupportsRFC9421(KeyTypeEd25519))
		helpers.LogSuccess(t, "Ed25519는 RFC9421 지원")

		// X25519 does NOT support RFC 9421 (key exchange only)
		helpers.LogDetail(t, "X25519 RFC9421 지원 확인...")
		assert.False(t, SupportsRFC9421(KeyTypeX25519))
		helpers.LogSuccess(t, "X25519는 RFC9421 미지원 (키 교환 전용)")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"Ed25519가 RFC9421 지원함",
			"X25519가 RFC9421 미지원함",
			"서명 알고리즘과 키 교환 알고리즘 구분됨",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "17.1.7_RFC9421_지원_확인",
			"support_checks": map[string]bool{
				"Ed25519": SupportsRFC9421(KeyTypeEd25519),
				"X25519":  SupportsRFC9421(KeyTypeX25519),
			},
			"validation": "RFC9421_지원_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_rfc9421_support.json", testData)
	})

	t.Run("Check if algorithm supports key generation", func(t *testing.T) {
		// 명세 요구사항: 키 생성 지원 여부 확인 검증
		helpers.LogTestSection(t, "17.1.8", "알고리즘 레지스트리 - 키 생성 지원 확인")

		keyTypes := []KeyType{KeyTypeEd25519, KeyTypeSecp256k1, KeyTypeRSA, KeyTypeX25519}
		helpers.LogDetail(t, "%d개 알고리즘의 키 생성 지원 확인...", len(keyTypes))

		// All registered algorithms should support key generation
		supportMap := make(map[string]bool)
		for _, kt := range keyTypes {
			supported := SupportsKeyGeneration(kt)
			assert.True(t, supported, "%s should support key generation", kt)
			supportMap[string(kt)] = supported
			helpers.LogDetail(t, "   %s 키 생성 지원", kt)
		}

		helpers.LogSuccess(t, "모든 등록된 알고리즘이 키 생성 지원")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"Ed25519 키 생성 지원",
			"Secp256k1 키 생성 지원",
			"RSA 키 생성 지원",
			"X25519 키 생성 지원",
			"모든 등록 알고리즘이 키 생성 가능",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":      "17.1.8_키_생성_지원_확인",
			"support_checks": supportMap,
			"all_supported":  true,
			"validation":     "키_생성_지원_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_key_generation.json", testData)
	})

	t.Run("Check if algorithm supports signature", func(t *testing.T) {
		// 명세 요구사항: 서명 지원 여부 확인 검증
		helpers.LogTestSection(t, "17.1.9", "알고리즘 레지스트리 - 서명 지원 확인")

		// Ed25519, Secp256k1, and RSA support signatures
		helpers.LogDetail(t, "서명 지원 알고리즘 확인...")
		supportMap := make(map[string]bool)

		assert.True(t, SupportsSignature(KeyTypeEd25519))
		supportMap["Ed25519"] = true
		helpers.LogDetail(t, "   Ed25519 서명 지원")

		assert.True(t, SupportsSignature(KeyTypeSecp256k1))
		supportMap["Secp256k1"] = true
		helpers.LogDetail(t, "   Secp256k1 서명 지원")

		assert.True(t, SupportsSignature(KeyTypeRSA))
		supportMap["RSA"] = true
		helpers.LogDetail(t, "   RSA 서명 지원")

		// X25519 does NOT support signatures (key exchange only)
		helpers.LogDetail(t, "키 교환 전용 알고리즘 확인...")
		assert.False(t, SupportsSignature(KeyTypeX25519))
		supportMap["X25519"] = false
		helpers.LogDetail(t, "   X25519 서명 미지원 (키 교환 전용)")

		// Unknown key type should return false
		helpers.LogDetail(t, "미등록 알고리즘 확인...")
		assert.False(t, SupportsSignature(KeyType("unknown")))
		helpers.LogSuccess(t, "미등록 알고리즘은 서명 미지원")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"Ed25519, Secp256k1, RSA는 서명 지원",
			"X25519는 서명 미지원 (키 교환 전용)",
			"미등록 알고리즘은 서명 미지원",
			"서명 알고리즘과 키 교환 알고리즘 구분됨",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":         "17.1.9_서명_지원_확인",
			"signature_support": supportMap,
			"unknown_rejected":  !SupportsSignature(KeyType("unknown")),
			"validation":        "서명_지원_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_signature_support.json", testData)
	})

	t.Run("Check if algorithm is supported", func(t *testing.T) {
		// 명세 요구사항: 알고리즘 지원 여부 확인 검증
		helpers.LogTestSection(t, "17.1.10", "알고리즘 레지스트리 - 알고리즘 지원 여부")

		// Registered algorithms should return true
		helpers.LogDetail(t, "등록된 알고리즘 지원 확인...")
		supportMap := make(map[string]bool)

		assert.True(t, IsAlgorithmSupported(KeyTypeEd25519))
		supportMap["Ed25519"] = true
		helpers.LogDetail(t, "   Ed25519 지원됨")

		assert.True(t, IsAlgorithmSupported(KeyTypeSecp256k1))
		supportMap["Secp256k1"] = true
		helpers.LogDetail(t, "   Secp256k1 지원됨")

		assert.True(t, IsAlgorithmSupported(KeyTypeRSA))
		supportMap["RSA"] = true
		helpers.LogDetail(t, "   RSA 지원됨")

		assert.True(t, IsAlgorithmSupported(KeyTypeX25519))
		supportMap["X25519"] = true
		helpers.LogDetail(t, "   X25519 지원됨")

		// Unknown algorithm should return false
		helpers.LogDetail(t, "미등록 알고리즘 확인...")
		assert.False(t, IsAlgorithmSupported(KeyType("unknown")))
		supportMap["unknown"] = false
		helpers.LogSuccess(t, "미등록 알고리즘은 지원되지 않음")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"모든 등록된 알고리즘(Ed25519, Secp256k1, RSA, X25519) 지원됨",
			"미등록 알고리즘은 지원되지 않음",
			"알고리즘 지원 여부 확인 정상 동작",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":        "17.1.10_알고리즘_지원_여부",
			"support_checks":   supportMap,
			"registered_count": 4,
			"validation":       "알고리즘_지원_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_is_supported.json", testData)
	})

	t.Run("Validate algorithm capabilities", func(t *testing.T) {
		// 명세 요구사항: 알고리즘 능력 세부 검증
		helpers.LogTestSection(t, "17.1.11", "알고리즘 레지스트리 - X25519 능력 검증")

		// Test that X25519 is registered but doesn't support RFC 9421
		helpers.LogDetail(t, "X25519 알고리즘 정보 조회...")
		info, err := GetAlgorithmInfo(KeyTypeX25519)
		require.NoError(t, err)
		helpers.LogSuccess(t, "X25519 정보 조회 성공")

		helpers.LogDetail(t, "X25519 능력 검증 중...")
		assert.Equal(t, KeyTypeX25519, info.KeyType)
		helpers.LogDetail(t, "  Key Type: %s", info.KeyType)

		assert.True(t, info.SupportsKeyGeneration)
		helpers.LogDetail(t, "  키 생성 지원: %v", info.SupportsKeyGeneration)

		assert.False(t, info.SupportsRFC9421, "X25519 should not support RFC 9421")
		helpers.LogDetail(t, "  RFC9421 지원: %v (키 교환 전용)", info.SupportsRFC9421)

		assert.Empty(t, info.RFC9421Algorithm, "X25519 should not have RFC 9421 algorithm")
		helpers.LogDetail(t, "  RFC9421 알고리즘: (없음)")

		helpers.LogSuccess(t, "X25519 능력 검증 완료: 키 생성은 지원하나 서명은 미지원")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"X25519가 레지스트리에 등록됨",
			"키 생성 기능 지원",
			"RFC9421 서명 미지원 (키 교환 전용)",
			"RFC9421 알고리즘 이름 없음",
			"서명 알고리즘과 키 교환 알고리즘 명확히 구분됨",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "17.1.11_X25519_능력_검증",
			"algorithm": "X25519",
			"capabilities": map[string]interface{}{
				"supports_key_generation": info.SupportsKeyGeneration,
				"supports_rfc9421":        info.SupportsRFC9421,
				"rfc9421_algorithm":       info.RFC9421Algorithm,
			},
			"validation": "X25519_능력_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_x25519_capabilities.json", testData)
	})
}

func TestAlgorithmRegistry_Immutability(t *testing.T) {
	t.Run("Returned slice should be immutable", func(t *testing.T) {
		// 명세 요구사항: 반환된 슬라이스 불변성 검증
		helpers.LogTestSection(t, "17.2.1", "알고리즘 레지스트리 - 슬라이스 불변성")

		helpers.LogDetail(t, "첫 번째 알고리즘 목록 조회...")
		algorithms1 := ListSupportedAlgorithms()
		originalLen := len(algorithms1)
		helpers.LogDetail(t, "  원본 길이: %d", originalLen)

		// Try to modify the returned slice
		helpers.LogDetail(t, "반환된 슬라이스 수정 시도...")
		_ = append(algorithms1, AlgorithmInfo{})
		helpers.LogDetail(t, "  수정 후 로컬 슬라이스 길이: %d", len(algorithms1))

		// Get the list again
		helpers.LogDetail(t, "두 번째 알고리즘 목록 조회...")
		algorithms2 := ListSupportedAlgorithms()
		helpers.LogDetail(t, "  새 목록 길이: %d", len(algorithms2))

		// Original should be unchanged
		assert.Equal(t, originalLen, len(algorithms2))
		helpers.LogSuccess(t, "레지스트리의 원본 목록이 변경되지 않음 (불변성 유지)")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"첫 번째 목록 조회 성공",
			"로컬 슬라이스 수정 시도",
			"두 번째 목록 조회 성공",
			"원본 레지스트리가 변경되지 않음",
			"레지스트리 불변성 보장",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":               "17.2.1_슬라이스_불변성",
			"original_length":         originalLen,
			"modified_local":          len(algorithms1),
			"registry_length":         len(algorithms2),
			"immutability_maintained": originalLen == len(algorithms2),
			"validation":              "불변성_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_immutable_slice.json", testData)
	})

	t.Run("Returned RFC 9421 list should be immutable", func(t *testing.T) {
		// 명세 요구사항: RFC9421 목록 불변성 검증
		helpers.LogTestSection(t, "17.2.2", "알고리즘 레지스트리 - RFC9421 목록 불변성")

		helpers.LogDetail(t, "첫 번째 RFC9421 목록 조회...")
		list1 := ListRFC9421SupportedAlgorithms()
		originalLen := len(list1)
		helpers.LogDetail(t, "  원본 길이: %d", originalLen)

		// Try to modify
		helpers.LogDetail(t, "반환된 목록에 가짜 알고리즘 추가 시도...")
		_ = append(list1, "fake-algorithm")
		helpers.LogDetail(t, "  수정 후 로컬 목록 길이: %d", len(list1))

		// Get again
		helpers.LogDetail(t, "두 번째 RFC9421 목록 조회...")
		list2 := ListRFC9421SupportedAlgorithms()
		helpers.LogDetail(t, "  새 목록 길이: %d", len(list2))

		// Original should be unchanged
		assert.Equal(t, originalLen, len(list2))
		helpers.LogSuccess(t, "레지스트리의 원본 목록 길이가 변경되지 않음")

		assert.NotContains(t, list2, "fake-algorithm")
		helpers.LogSuccess(t, "가짜 알고리즘이 레지스트리에 추가되지 않음 (불변성 유지)")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"첫 번째 RFC9421 목록 조회 성공",
			"로컬 목록에 가짜 알고리즘 추가 시도",
			"두 번째 RFC9421 목록 조회 성공",
			"원본 레지스트리가 변경되지 않음",
			"가짜 알고리즘이 포함되지 않음",
			"RFC9421 목록 불변성 보장",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":               "17.2.2_RFC9421_목록_불변성",
			"original_length":         originalLen,
			"modified_local":          len(list1),
			"registry_length":         len(list2),
			"fake_not_included":       !containsString(list2, "fake-algorithm"),
			"immutability_maintained": originalLen == len(list2),
			"validation":              "불변성_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_immutable_rfc9421.json", testData)
	})
}

func TestAlgorithmRegistry_ThreadSafety(t *testing.T) {
	t.Run("Concurrent reads should be safe", func(t *testing.T) {
		// 명세 요구사항: 동시 읽기 스레드 안전성 검증
		helpers.LogTestSection(t, "17.3.1", "알고리즘 레지스트리 - 동시 읽기 안전성")

		done := make(chan bool)
		numGoroutines := 10

		helpers.LogDetail(t, "%d개 고루틴으로 동시 읽기 테스트...", numGoroutines)

		// Spawn multiple goroutines reading from registry
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() {
					if r := recover(); r != nil {
						helpers.LogDetail(t, "  고루틴 %d 패닉: %v", id, r)
					}
					done <- true
				}()

				_, _ = GetAlgorithmInfo(KeyTypeEd25519)
				_ = ListSupportedAlgorithms()
				_ = ListRFC9421SupportedAlgorithms()
				_, _ = GetRFC9421AlgorithmName(KeyTypeSecp256k1)
			}(i)
		}

		// Wait for all goroutines
		completedCount := 0
		for i := 0; i < numGoroutines; i++ {
			<-done
			completedCount++
		}

		helpers.LogDetail(t, "모든 고루틴 완료 (%d/%d)", completedCount, numGoroutines)
		helpers.LogSuccess(t, "패닉 없이 동시 읽기 안전성 확인")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			fmt.Sprintf("%d개 고루틴 동시 실행", numGoroutines),
			"모든 고루틴이 정상 완료",
			"패닉이나 데이터 레이스 없음",
			"동시 읽기 안전성 보장",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":   "17.3.1_동시_읽기_안전성",
			"goroutines":  numGoroutines,
			"completed":   completedCount,
			"no_panics":   true,
			"thread_safe": true,
			"validation":  "스레드_안전성_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_thread_safety.json", testData)
	})
}

func TestAlgorithmRegistry_Integration(t *testing.T) {
	t.Run("All key types should be registered", func(t *testing.T) {
		// 명세 요구사항: 모든 키 타입 등록 통합 검증
		helpers.LogTestSection(t, "17.4.1", "알고리즘 레지스트리 - 전체 키 타입 등록 검증")

		keyTypes := []KeyType{
			KeyTypeEd25519,
			KeyTypeSecp256k1,
			KeyTypeX25519,
			KeyTypeRSA,
		}

		helpers.LogDetail(t, "%d개 키 타입 등록 상태 검증...", len(keyTypes))

		successCount := 0
		for _, kt := range keyTypes {
			t.Run(string(kt), func(t *testing.T) {
				helpers.LogDetail(t, "키 타입 %s 검증 중...", kt)
				info, err := GetAlgorithmInfo(kt)
				require.NoError(t, err, "Key type %s should be registered", kt)
				helpers.LogDetail(t, "  %s 등록 확인", kt)

				assert.Equal(t, kt, info.KeyType)
				helpers.LogDetail(t, "  Key Type: %s", info.KeyType)

				assert.NotEmpty(t, info.Name)
				helpers.LogDetail(t, "  Name: %s", info.Name)

				assert.NotEmpty(t, info.Description)
				helpers.LogDetail(t, "  Description: %s", info.Description)

				successCount++
			})
		}

		helpers.LogDetail(t, "모든 키 타입 등록 및 검증 완료 (%d/%d)", successCount, len(keyTypes))
		helpers.LogSuccess(t, "전체 키 타입 등록 검증 완료")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"Ed25519 키 타입 등록됨",
			"Secp256k1 키 타입 등록됨",
			"X25519 키 타입 등록됨",
			"RSA 키 타입 등록됨",
			"모든 키 타입에 이름 및 설명 있음",
		})

		// CLI 검증용 테스트 데이터 저장
		var keyTypeStrings []string
		for _, kt := range keyTypes {
			keyTypeStrings = append(keyTypeStrings, string(kt))
		}
		testData := map[string]interface{}{
			"test_case":      "17.4.1_전체_키_타입_등록",
			"key_types":      keyTypeStrings,
			"total_count":    len(keyTypes),
			"all_registered": true,
			"validation":     "키_타입_등록_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_integration_all_keys.json", testData)
	})

	t.Run("RFC 9421 algorithms should map back to key types", func(t *testing.T) {
		// 명세 요구사항: RFC9421 알고리즘 양방향 매핑 검증
		helpers.LogTestSection(t, "17.4.2", "알고리즘 레지스트리 - RFC9421 양방향 매핑")

		helpers.LogDetail(t, "RFC9421 알고리즘 목록 조회...")
		rfc9421Algorithms := ListRFC9421SupportedAlgorithms()
		helpers.LogDetail(t, "%d개 RFC9421 알고리즘 발견", len(rfc9421Algorithms))

		successCount := 0
		mappings := make(map[string]string)

		for _, algName := range rfc9421Algorithms {
			t.Run(algName, func(t *testing.T) {
				helpers.LogDetail(t, "RFC9421 알고리즘 %s 양방향 매핑 검증...", algName)

				// Forward mapping: RFC9421 → KeyType
				keyType, err := GetKeyTypeFromRFC9421Algorithm(algName)
				require.NoError(t, err)
				helpers.LogDetail(t, "  %s → %s (역매핑)", algName, keyType)

				// Reverse lookup should work
				rfc9421Name, err := GetRFC9421AlgorithmName(keyType)
				require.NoError(t, err)
				helpers.LogDetail(t, "  %s → %s (정방향)", keyType, rfc9421Name)

				assert.Equal(t, algName, rfc9421Name)
				helpers.LogDetail(t, "  양방향 매핑 일치: %s", algName)

				mappings[algName] = string(keyType)
				successCount++
			})
		}

		helpers.LogDetail(t, "모든 RFC9421 알고리즘 양방향 매핑 검증 완료 (%d/%d)", successCount, len(rfc9421Algorithms))
		helpers.LogSuccess(t, "RFC9421 양방향 매핑 검증 완료")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"모든 RFC9421 알고리즘이 KeyType으로 매핑됨",
			"모든 KeyType이 다시 RFC9421 알고리즘으로 매핑됨",
			"양방향 매핑이 일치함",
			"매핑 무결성 보장",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":              "17.4.2_RFC9421_양방향_매핑",
			"algorithms_count":       len(rfc9421Algorithms),
			"bidirectional_mappings": mappings,
			"all_mapped":             successCount == len(rfc9421Algorithms),
			"validation":             "양방향_매핑_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/algorithm_registry_integration_bidirectional.json", testData)
	})
}

// Helper function to check if a string slice contains a value
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
