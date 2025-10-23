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

package crypto_test

import (
	"testing"

	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/sage-x-project/sage/internal/cryptoinit" // Initialize wrappers
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
)

func TestNewManager(t *testing.T) {
	// 명세 요구사항: Manager 생성 검증
	helpers.LogTestSection(t, "18.1.1", "Crypto Manager - 생성")

	helpers.LogDetail(t, "새로운 Crypto Manager 생성...")
	manager := sagecrypto.NewManager()
	assert.NotNil(t, manager)
	helpers.LogSuccess(t, "Manager 생성 완료")

	// 통과 기준 체크리스트
	helpers.LogPassCriteria(t, []string{
		"Manager 인스턴스가 성공적으로 생성됨",
		"Manager가 nil이 아님",
	})

	// CLI 검증용 테스트 데이터 저장
	testData := map[string]interface{}{
		"test_case":       "18.1.1_Manager_생성",
		"manager_created": manager != nil,
		"validation":      "Manager_생성_검증_통과",
	}
	helpers.SaveTestData(t, "crypto/manager_new.json", testData)
}

func TestManager_SetStorage(t *testing.T) {
	// 명세 요구사항: 커스텀 스토리지 설정 검증
	helpers.LogTestSection(t, "18.1.2", "Crypto Manager - 스토리지 설정")

	helpers.LogDetail(t, "Manager 생성...")
	manager := sagecrypto.NewManager()
	helpers.LogDetail(t, "Memory Key Storage 생성...")
	customStorage := sagecrypto.NewMemoryKeyStorage()

	// Should not panic
	helpers.LogDetail(t, "커스텀 스토리지 설정 중...")
	manager.SetStorage(customStorage)
	helpers.LogSuccess(t, "커스텀 스토리지 설정 완료 (패닉 없음)")

	// 통과 기준 체크리스트
	helpers.LogPassCriteria(t, []string{
		"Manager 생성 성공",
		"Memory Storage 생성 성공",
		"SetStorage 호출 성공 (패닉 없음)",
	})

	// CLI 검증용 테스트 데이터 저장
	testData := map[string]interface{}{
		"test_case":   "18.1.2_스토리지_설정",
		"storage_set": true,
		"no_panic":    true,
		"validation":  "스토리지_설정_검증_통과",
	}
	helpers.SaveTestData(t, "crypto/manager_set_storage.json", testData)
}

func TestManager_GenerateKeyPair(t *testing.T) {
	manager := sagecrypto.NewManager()

	t.Run("Generate Ed25519 key pair", func(t *testing.T) {
		// 명세 요구사항: Ed25519 키 쌍 생성 검증
		helpers.LogTestSection(t, "18.2.1", "Manager - Ed25519 키 쌍 생성")

		helpers.LogDetail(t, "Ed25519 키 쌍 생성 중...")
		keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
		require.NoError(t, err)
		assert.NotNil(t, keyPair)
		helpers.LogSuccess(t, "Ed25519 키 쌍 생성 완료")

		helpers.LogDetail(t, "키 속성 검증 중...")
		assert.Equal(t, sagecrypto.KeyTypeEd25519, keyPair.Type())
		helpers.LogDetail(t, "  Key Type: %s", keyPair.Type())

		assert.NotEmpty(t, keyPair.ID())
		helpers.LogDetail(t, "  Key ID: %s", keyPair.ID())

		assert.NotNil(t, keyPair.PublicKey())
		assert.NotNil(t, keyPair.PrivateKey())
		helpers.LogSuccess(t, "모든 키 속성 검증 완료")

		helpers.LogPassCriteria(t, []string{
			"Ed25519 키 쌍 생성 성공",
			"Key Type이 Ed25519",
			"Key ID가 비어있지 않음",
			"Public Key 존재",
			"Private Key 존재",
		})

		testData := map[string]interface{}{
			"test_case":  "18.2.1_Ed25519_키_생성",
			"key_type":   string(keyPair.Type()),
			"has_id":     keyPair.ID() != "",
			"has_keys":   true,
			"validation": "Ed25519_키_생성_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_generate_ed25519.json", testData)
	})

	t.Run("Generate Secp256k1 key pair", func(t *testing.T) {
		// 명세 요구사항: Secp256k1 키 쌍 생성 검증
		helpers.LogTestSection(t, "18.2.2", "Manager - Secp256k1 키 쌍 생성")

		helpers.LogDetail(t, "Secp256k1 키 쌍 생성 중...")
		keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeSecp256k1)
		require.NoError(t, err)
		assert.NotNil(t, keyPair)
		helpers.LogSuccess(t, "Secp256k1 키 쌍 생성 완료")

		helpers.LogDetail(t, "키 속성 검증 중...")
		assert.Equal(t, sagecrypto.KeyTypeSecp256k1, keyPair.Type())
		helpers.LogDetail(t, "  Key Type: %s", keyPair.Type())

		assert.NotEmpty(t, keyPair.ID())
		helpers.LogDetail(t, "  Key ID: %s", keyPair.ID())

		assert.NotNil(t, keyPair.PublicKey())
		assert.NotNil(t, keyPair.PrivateKey())
		helpers.LogSuccess(t, "모든 키 속성 검증 완료")

		helpers.LogPassCriteria(t, []string{
			"Secp256k1 키 쌍 생성 성공",
			"Key Type이 Secp256k1",
			"Key ID가 비어있지 않음",
			"Public Key 존재",
			"Private Key 존재",
		})

		testData := map[string]interface{}{
			"test_case":  "18.2.2_Secp256k1_키_생성",
			"key_type":   string(keyPair.Type()),
			"has_id":     keyPair.ID() != "",
			"has_keys":   true,
			"validation": "Secp256k1_키_생성_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_generate_secp256k1.json", testData)
	})

	t.Run("Unsupported key type", func(t *testing.T) {
		// 명세 요구사항: 미지원 키 타입 에러 처리 검증
		helpers.LogTestSection(t, "18.2.3", "Manager - 미지원 키 타입 에러")

		helpers.LogDetail(t, "미지원 키 타입으로 생성 시도...")
		_, err := manager.GenerateKeyPair(sagecrypto.KeyType("unsupported"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported key type")
		helpers.LogSuccess(t, "예상대로 에러 발생")
		helpers.LogDetail(t, "  에러: %s", err.Error())

		helpers.LogPassCriteria(t, []string{
			"미지원 키 타입 생성 시도 시 에러 발생",
			"에러 메시지에 'unsupported key type' 포함",
		})

		testData := map[string]interface{}{
			"test_case":  "18.2.3_미지원_키_타입",
			"error":      err.Error(),
			"validation": "에러_처리_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_generate_unsupported.json", testData)
	})

	t.Run("X25519 key type not supported by Manager", func(t *testing.T) {
		// 명세 요구사항: X25519 Manager 미지원 검증
		helpers.LogTestSection(t, "18.2.4", "Manager - X25519 미지원")

		helpers.LogDetail(t, "X25519 키 타입으로 생성 시도...")
		_, err := manager.GenerateKeyPair(sagecrypto.KeyTypeX25519)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported key type")
		helpers.LogSuccess(t, "Manager에서 X25519 미지원 확인")
		helpers.LogDetail(t, "  에러: %s", err.Error())

		helpers.LogPassCriteria(t, []string{
			"X25519 키 생성 시도 시 에러 발생",
			"Manager는 X25519를 지원하지 않음",
		})

		testData := map[string]interface{}{
			"test_case":  "18.2.4_X25519_미지원",
			"key_type":   "X25519",
			"error":      err.Error(),
			"validation": "미지원_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_generate_x25519_unsupported.json", testData)
	})

	t.Run("RSA key type not supported by Manager", func(t *testing.T) {
		// 명세 요구사항: RSA Manager 미지원 검증
		helpers.LogTestSection(t, "18.2.5", "Manager - RSA 미지원")

		helpers.LogDetail(t, "RSA 키 타입으로 생성 시도...")
		_, err := manager.GenerateKeyPair(sagecrypto.KeyTypeRSA)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported key type")
		helpers.LogSuccess(t, "Manager에서 RSA 미지원 확인")
		helpers.LogDetail(t, "  에러: %s", err.Error())

		helpers.LogPassCriteria(t, []string{
			"RSA 키 생성 시도 시 에러 발생",
			"Manager는 RSA를 지원하지 않음",
		})

		testData := map[string]interface{}{
			"test_case":  "18.2.5_RSA_미지원",
			"key_type":   "RSA",
			"error":      err.Error(),
			"validation": "미지원_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_generate_rsa_unsupported.json", testData)
	})
}

func TestManager_StoreKeyPair(t *testing.T) {
	// 명세 요구사항: 키 쌍 저장 검증
	helpers.LogTestSection(t, "18.3.1", "Manager - 키 쌍 저장")

	helpers.LogDetail(t, "Manager 생성...")
	manager := sagecrypto.NewManager()

	helpers.LogDetail(t, "Ed25519 키 쌍 생성...")
	keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
	require.NoError(t, err)

	helpers.LogDetail(t, "키 쌍 저장 중...")
	err = manager.StoreKeyPair(keyPair)
	assert.NoError(t, err)
	helpers.LogSuccess(t, "키 쌍 저장 완료")

	helpers.LogPassCriteria(t, []string{
		"키 쌍 생성 성공",
		"키 쌍 저장 성공",
	})

	testData := map[string]interface{}{
		"test_case":  "18.3.1_키_쌍_저장",
		"key_stored": true,
		"validation": "저장_검증_통과",
	}
	helpers.SaveTestData(t, "crypto/manager_store.json", testData)
}

func TestManager_LoadKeyPair(t *testing.T) {
	manager := sagecrypto.NewManager()

	// Generate and store a key pair
	keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
	require.NoError(t, err)

	err = manager.StoreKeyPair(keyPair)
	require.NoError(t, err)

	t.Run("Load existing key pair", func(t *testing.T) {
		// 명세 요구사항: 저장된 키 쌍 로드 검증
		helpers.LogTestSection(t, "18.4.1", "Manager - 키 쌍 로드")

		helpers.LogDetail(t, "키 쌍 로드 중: %s", keyPair.ID())
		loadedKeyPair, err := manager.LoadKeyPair(keyPair.ID())
		assert.NoError(t, err)
		assert.NotNil(t, loadedKeyPair)
		helpers.LogSuccess(t, "키 쌍 로드 완료")

		helpers.LogDetail(t, "키 쌍 일치 여부 검증...")
		assert.Equal(t, keyPair.ID(), loadedKeyPair.ID())
		assert.Equal(t, keyPair.Type(), loadedKeyPair.Type())
		helpers.LogSuccess(t, "로드된 키 쌍이 원본과 일치")

		helpers.LogPassCriteria(t, []string{
			"저장된 키 쌍 로드 성공",
			"로드된 키 ID가 원본과 일치",
			"로드된 키 타입이 원본과 일치",
		})

		testData := map[string]interface{}{
			"test_case":  "18.4.1_키_쌍_로드",
			"key_loaded": true,
			"ids_match":  keyPair.ID() == loadedKeyPair.ID(),
			"validation": "로드_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_load_existing.json", testData)
	})

	t.Run("Load non-existent key pair", func(t *testing.T) {
		// 명세 요구사항: 존재하지 않는 키 로드 에러 처리 검증
		helpers.LogTestSection(t, "18.4.2", "Manager - 존재하지 않는 키 로드")

		helpers.LogDetail(t, "존재하지 않는 키 ID로 로드 시도...")
		_, err := manager.LoadKeyPair("non-existent-id")
		assert.Error(t, err)
		assert.Equal(t, sagecrypto.ErrKeyNotFound, err)
		helpers.LogSuccess(t, "예상대로 ErrKeyNotFound 에러 발생")

		helpers.LogPassCriteria(t, []string{
			"존재하지 않는 키 로드 시 에러 발생",
			"에러가 ErrKeyNotFound",
		})

		testData := map[string]interface{}{
			"test_case":  "18.4.2_존재하지_않는_키_로드",
			"error":      "key not found",
			"validation": "에러_처리_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_load_nonexistent.json", testData)
	})
}

func TestManager_DeleteKeyPair(t *testing.T) {
	manager := sagecrypto.NewManager()

	// Generate and store a key pair
	keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
	require.NoError(t, err)

	err = manager.StoreKeyPair(keyPair)
	require.NoError(t, err)

	t.Run("Delete existing key pair", func(t *testing.T) {
		// 명세 요구사항: 키 쌍 삭제 검증
		helpers.LogTestSection(t, "18.5.1", "Manager - 키 쌍 삭제")

		helpers.LogDetail(t, "키 쌍 삭제 중: %s", keyPair.ID())
		err := manager.DeleteKeyPair(keyPair.ID())
		assert.NoError(t, err)
		helpers.LogSuccess(t, "키 쌍 삭제 완료")

		// Verify it's deleted
		helpers.LogDetail(t, "삭제 검증 중...")
		_, err = manager.LoadKeyPair(keyPair.ID())
		assert.Error(t, err)
		assert.Equal(t, sagecrypto.ErrKeyNotFound, err)
		helpers.LogSuccess(t, "삭제된 키 로드 시 ErrKeyNotFound 발생 확인")

		helpers.LogPassCriteria(t, []string{
			"키 쌍 삭제 성공",
			"삭제된 키 로드 시 ErrKeyNotFound 발생",
		})

		testData := map[string]interface{}{
			"test_case":   "18.5.1_키_쌍_삭제",
			"key_deleted": true,
			"load_failed": true,
			"validation":  "삭제_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_delete_existing.json", testData)
	})

	t.Run("Delete non-existent key pair", func(t *testing.T) {
		// 명세 요구사항: 존재하지 않는 키 삭제 에러 처리 검증
		helpers.LogTestSection(t, "18.5.2", "Manager - 존재하지 않는 키 삭제")

		helpers.LogDetail(t, "존재하지 않는 키 삭제 시도...")
		err := manager.DeleteKeyPair("non-existent-id")
		assert.Error(t, err)
		assert.Equal(t, sagecrypto.ErrKeyNotFound, err)
		helpers.LogSuccess(t, "예상대로 ErrKeyNotFound 에러 발생")

		helpers.LogPassCriteria(t, []string{
			"존재하지 않는 키 삭제 시 에러 발생",
			"에러가 ErrKeyNotFound",
		})

		testData := map[string]interface{}{
			"test_case":  "18.5.2_존재하지_않는_키_삭제",
			"error":      "key not found",
			"validation": "에러_처리_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_delete_nonexistent.json", testData)
	})
}

func TestManager_ListKeyPairs(t *testing.T) {
	manager := sagecrypto.NewManager()

	t.Run("List empty storage", func(t *testing.T) {
		// 명세 요구사항: 빈 스토리지에서 키 쌍 목록 조회
		helpers.LogTestSection(t, "18.6.1", "Manager - 빈 스토리지 키 목록 조회")

		helpers.LogDetail(t, "빈 스토리지에서 키 쌍 목록 조회 중...")
		ids, err := manager.ListKeyPairs()
		assert.NoError(t, err)
		assert.Empty(t, ids)
		helpers.LogSuccess(t, "빈 스토리지에서 빈 목록 반환 성공")
		helpers.LogDetail(t, "반환된 키 개수: %d", len(ids))

		helpers.LogPassCriteria(t, []string{
			"빈 스토리지에서 목록 조회 성공",
			"반환된 키 목록이 비어있음",
			"에러가 발생하지 않음",
		})

		testData := map[string]interface{}{
			"test_case":  "18.6.1_빈_스토리지_키_목록_조회",
			"keys_count": len(ids),
			"validation": "빈_목록_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_list_empty.json", testData)
	})

	t.Run("List with stored keys", func(t *testing.T) {
		// 명세 요구사항: 저장된 키 쌍들의 목록 조회
		helpers.LogTestSection(t, "18.6.2", "Manager - 저장된 키 쌍 목록 조회")

		// Generate and store multiple key pairs
		helpers.LogDetail(t, "Ed25519 키 쌍 생성 및 저장...")
		keyPair1, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
		require.NoError(t, err)
		err = manager.StoreKeyPair(keyPair1)
		require.NoError(t, err)
		helpers.LogDetail(t, "  키 1 저장 완료: %s", keyPair1.ID())

		helpers.LogDetail(t, "Secp256k1 키 쌍 생성 및 저장...")
		keyPair2, err := manager.GenerateKeyPair(sagecrypto.KeyTypeSecp256k1)
		require.NoError(t, err)
		err = manager.StoreKeyPair(keyPair2)
		require.NoError(t, err)
		helpers.LogDetail(t, "  키 2 저장 완료: %s", keyPair2.ID())

		helpers.LogDetail(t, "저장된 키 쌍 목록 조회 중...")
		ids, err := manager.ListKeyPairs()
		assert.NoError(t, err)
		assert.Len(t, ids, 2)
		assert.Contains(t, ids, keyPair1.ID())
		assert.Contains(t, ids, keyPair2.ID())
		helpers.LogSuccess(t, "2개의 키 쌍 목록 조회 성공")
		helpers.LogDetail(t, "반환된 키 ID: %v", ids)

		helpers.LogPassCriteria(t, []string{
			"Ed25519 키 쌍 생성 및 저장 성공",
			"Secp256k1 키 쌍 생성 및 저장 성공",
			"목록 조회에서 2개의 키 반환",
			"두 키 ID가 모두 목록에 포함됨",
		})

		testData := map[string]interface{}{
			"test_case":  "18.6.2_저장된_키_목록_조회",
			"keys_count": len(ids),
			"key_ids":    ids,
			"key_types": []string{
				string(keyPair1.Type()),
				string(keyPair2.Type()),
			},
			"validation": "목록_조회_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_list_stored.json", testData)
	})
}

func TestManager_ExportKeyPair(t *testing.T) {
	manager := sagecrypto.NewManager()

	helpers.LogDetail(t, "테스트용 Ed25519 키 쌍 생성 중...")
	keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
	require.NoError(t, err)
	helpers.LogDetail(t, "키 쌍 생성 완료: %s", keyPair.ID())

	t.Run("Export as JWK", func(t *testing.T) {
		// 명세 요구사항: 키 쌍을 JWK 형식으로 내보내기
		helpers.LogTestSection(t, "18.7.1", "Manager - JWK 형식으로 키 내보내기")

		helpers.LogDetail(t, "키 쌍을 JWK 형식으로 내보내는 중...")
		data, err := manager.ExportKeyPair(keyPair, sagecrypto.KeyFormatJWK)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
		helpers.LogSuccess(t, "JWK 형식으로 키 내보내기 성공")
		helpers.LogDetail(t, "내보낸 데이터 크기: %d bytes", len(data))

		// Should be valid JSON
		helpers.LogDetail(t, "JWK JSON 형식 검증 중...")
		assert.Contains(t, string(data), "kty")
		helpers.LogSuccess(t, "JWK 필수 필드 'kty' 존재 확인")

		helpers.LogPassCriteria(t, []string{
			"JWK 형식으로 내보내기 성공",
			"내보낸 데이터가 비어있지 않음",
			"JSON 형식이 올바름 (kty 필드 존재)",
		})

		testData := map[string]interface{}{
			"test_case":  "18.7.1_JWK_형식_내보내기",
			"format":     "JWK",
			"data_size":  len(data),
			"has_kty":    true,
			"validation": "JWK_내보내기_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_export_jwk.json", testData)
	})

	t.Run("Export as PEM", func(t *testing.T) {
		// 명세 요구사항: 키 쌍을 PEM 형식으로 내보내기
		helpers.LogTestSection(t, "18.7.2", "Manager - PEM 형식으로 키 내보내기")

		helpers.LogDetail(t, "키 쌍을 PEM 형식으로 내보내는 중...")
		data, err := manager.ExportKeyPair(keyPair, sagecrypto.KeyFormatPEM)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
		helpers.LogSuccess(t, "PEM 형식으로 키 내보내기 성공")
		helpers.LogDetail(t, "내보낸 데이터 크기: %d bytes", len(data))

		// Should contain PEM markers
		helpers.LogDetail(t, "PEM 형식 마커 검증 중...")
		assert.Contains(t, string(data), "BEGIN")
		assert.Contains(t, string(data), "END")
		helpers.LogSuccess(t, "PEM 형식 마커 (BEGIN/END) 존재 확인")

		helpers.LogPassCriteria(t, []string{
			"PEM 형식으로 내보내기 성공",
			"내보낸 데이터가 비어있지 않음",
			"PEM 형식 마커 (BEGIN) 존재",
			"PEM 형식 마커 (END) 존재",
		})

		testData := map[string]interface{}{
			"test_case":  "18.7.2_PEM_형식_내보내기",
			"format":     "PEM",
			"data_size":  len(data),
			"has_begin":  true,
			"has_end":    true,
			"validation": "PEM_내보내기_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_export_pem.json", testData)
	})

	t.Run("Export with unsupported format", func(t *testing.T) {
		// 명세 요구사항: 지원하지 않는 형식으로 내보내기 시 에러 처리
		helpers.LogTestSection(t, "18.7.3", "Manager - 지원하지 않는 형식으로 키 내보내기")

		helpers.LogDetail(t, "지원하지 않는 형식으로 내보내기 시도...")
		_, err := manager.ExportKeyPair(keyPair, sagecrypto.KeyFormat("unsupported"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported key format")
		helpers.LogSuccess(t, "예상대로 unsupported key format 에러 발생")
		helpers.LogDetail(t, "에러 메시지: %s", err.Error())

		helpers.LogPassCriteria(t, []string{
			"지원하지 않는 형식으로 내보내기 시도 시 에러 발생",
			"에러 메시지에 'unsupported key format' 포함",
			"에러 처리가 올바르게 동작함",
		})

		testData := map[string]interface{}{
			"test_case":    "18.7.3_지원하지_않는_형식_내보내기",
			"format":       "unsupported",
			"error":        "unsupported key format",
			"error_raised": true,
			"validation":   "에러_처리_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_export_unsupported.json", testData)
	})
}

func TestManager_ImportKeyPair(t *testing.T) {
	manager := sagecrypto.NewManager()

	// Generate a key pair and export it
	helpers.LogDetail(t, "테스트용 Ed25519 키 쌍 생성 중...")
	originalKeyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
	require.NoError(t, err)
	helpers.LogDetail(t, "원본 키 쌍 생성 완료: %s", originalKeyPair.ID())

	t.Run("Import from JWK", func(t *testing.T) {
		// 명세 요구사항: JWK 형식에서 키 쌍 가져오기
		helpers.LogTestSection(t, "18.8.1", "Manager - JWK 형식에서 키 가져오기")

		// Export as JWK
		helpers.LogDetail(t, "원본 키를 JWK 형식으로 내보내는 중...")
		jwkData, err := manager.ExportKeyPair(originalKeyPair, sagecrypto.KeyFormatJWK)
		require.NoError(t, err)
		helpers.LogDetail(t, "JWK 데이터 크기: %d bytes", len(jwkData))

		// Import from JWK
		helpers.LogDetail(t, "JWK 데이터에서 키 쌍 가져오는 중...")
		importedKeyPair, err := manager.ImportKeyPair(jwkData, sagecrypto.KeyFormatJWK)
		assert.NoError(t, err)
		assert.NotNil(t, importedKeyPair)
		assert.Equal(t, originalKeyPair.Type(), importedKeyPair.Type())
		helpers.LogSuccess(t, "JWK에서 키 쌍 가져오기 성공")
		helpers.LogDetail(t, "가져온 키 쌍 타입: %s", importedKeyPair.Type())

		// Verify the keys are functionally equivalent by signing and verifying
		helpers.LogDetail(t, "키 기능 동등성 검증 중...")
		message := []byte("test message")
		signature, err := originalKeyPair.Sign(message)
		require.NoError(t, err)

		err = importedKeyPair.Verify(message, signature)
		assert.NoError(t, err)
		helpers.LogSuccess(t, "서명 검증 성공 - 키가 기능적으로 동등함")

		helpers.LogPassCriteria(t, []string{
			"JWK 형식으로 내보내기 성공",
			"JWK에서 키 쌍 가져오기 성공",
			"가져온 키 쌍의 타입이 원본과 동일",
			"원본 키로 생성한 서명을 가져온 키로 검증 성공",
		})

		testData := map[string]interface{}{
			"test_case":     "18.8.1_JWK_형식_가져오기",
			"format":        "JWK",
			"key_type":      string(importedKeyPair.Type()),
			"jwk_data_size": len(jwkData),
			"sign_verify":   "성공",
			"validation":    "JWK_가져오기_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_import_jwk.json", testData)
	})

	t.Run("Import from PEM", func(t *testing.T) {
		// 명세 요구사항: PEM 형식에서 키 쌍 가져오기
		helpers.LogTestSection(t, "18.8.2", "Manager - PEM 형식에서 키 가져오기")

		// Export as PEM
		helpers.LogDetail(t, "원본 키를 PEM 형식으로 내보내는 중...")
		pemData, err := manager.ExportKeyPair(originalKeyPair, sagecrypto.KeyFormatPEM)
		require.NoError(t, err)
		helpers.LogDetail(t, "PEM 데이터 크기: %d bytes", len(pemData))

		// Import from PEM
		helpers.LogDetail(t, "PEM 데이터에서 키 쌍 가져오는 중...")
		importedKeyPair, err := manager.ImportKeyPair(pemData, sagecrypto.KeyFormatPEM)
		assert.NoError(t, err)
		assert.NotNil(t, importedKeyPair)
		assert.Equal(t, originalKeyPair.Type(), importedKeyPair.Type())
		helpers.LogSuccess(t, "PEM에서 키 쌍 가져오기 성공")
		helpers.LogDetail(t, "가져온 키 쌍 타입: %s", importedKeyPair.Type())

		// Verify the keys are functionally equivalent
		helpers.LogDetail(t, "키 기능 동등성 검증 중...")
		message := []byte("test message")
		signature, err := originalKeyPair.Sign(message)
		require.NoError(t, err)

		err = importedKeyPair.Verify(message, signature)
		assert.NoError(t, err)
		helpers.LogSuccess(t, "서명 검증 성공 - 키가 기능적으로 동등함")

		helpers.LogPassCriteria(t, []string{
			"PEM 형식으로 내보내기 성공",
			"PEM에서 키 쌍 가져오기 성공",
			"가져온 키 쌍의 타입이 원본과 동일",
			"원본 키로 생성한 서명을 가져온 키로 검증 성공",
		})

		testData := map[string]interface{}{
			"test_case":     "18.8.2_PEM_형식_가져오기",
			"format":        "PEM",
			"key_type":      string(importedKeyPair.Type()),
			"pem_data_size": len(pemData),
			"sign_verify":   "성공",
			"validation":    "PEM_가져오기_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_import_pem.json", testData)
	})

	t.Run("Import with unsupported format", func(t *testing.T) {
		// 명세 요구사항: 지원하지 않는 형식에서 가져오기 시 에러 처리
		helpers.LogTestSection(t, "18.8.3", "Manager - 지원하지 않는 형식에서 키 가져오기")

		helpers.LogDetail(t, "지원하지 않는 형식에서 가져오기 시도...")
		_, err := manager.ImportKeyPair([]byte("data"), sagecrypto.KeyFormat("unsupported"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported key format")
		helpers.LogSuccess(t, "예상대로 unsupported key format 에러 발생")
		helpers.LogDetail(t, "에러 메시지: %s", err.Error())

		helpers.LogPassCriteria(t, []string{
			"지원하지 않는 형식에서 가져오기 시도 시 에러 발생",
			"에러 메시지에 'unsupported key format' 포함",
			"에러 처리가 올바르게 동작함",
		})

		testData := map[string]interface{}{
			"test_case":    "18.8.3_지원하지_않는_형식_가져오기",
			"format":       "unsupported",
			"error":        "unsupported key format",
			"error_raised": true,
			"validation":   "에러_처리_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_import_unsupported.json", testData)
	})

	t.Run("Import with invalid data", func(t *testing.T) {
		// 명세 요구사항: 잘못된 데이터로 가져오기 시 에러 처리
		helpers.LogTestSection(t, "18.8.4", "Manager - 잘못된 데이터에서 키 가져오기")

		helpers.LogDetail(t, "잘못된 JWK 데이터로 가져오기 시도...")
		_, err := manager.ImportKeyPair([]byte("invalid data"), sagecrypto.KeyFormatJWK)
		assert.Error(t, err)
		helpers.LogSuccess(t, "예상대로 에러 발생")
		helpers.LogDetail(t, "에러 메시지: %s", err.Error())

		helpers.LogPassCriteria(t, []string{
			"잘못된 JWK 데이터로 가져오기 시도 시 에러 발생",
			"에러 처리가 올바르게 동작함",
			"파싱 실패 시 적절한 에러 반환",
		})

		testData := map[string]interface{}{
			"test_case":    "18.8.4_잘못된_데이터_가져오기",
			"format":       "JWK",
			"data":         "invalid data",
			"error_raised": true,
			"validation":   "에러_처리_검증_통과",
		}
		helpers.SaveTestData(t, "crypto/manager_import_invalid.json", testData)
	})
}

func TestManager_Integration(t *testing.T) {
	// 명세 요구사항: Manager 전체 라이프사이클 통합 테스트
	helpers.LogTestSection(t, "18.9.1", "Manager - 전체 라이프사이클 통합 테스트")

	manager := sagecrypto.NewManager()

	// Generate key pair
	helpers.LogDetail(t, "1단계: 키 쌍 생성")
	keyPair, err := manager.GenerateKeyPair(sagecrypto.KeyTypeEd25519)
	require.NoError(t, err)
	helpers.LogDetail(t, "  생성된 키 ID: %s", keyPair.ID())
	helpers.LogSuccess(t, "키 쌍 생성 완료")

	// Store it
	helpers.LogDetail(t, "2단계: 키 쌍 저장")
	err = manager.StoreKeyPair(keyPair)
	require.NoError(t, err)
	helpers.LogSuccess(t, "키 쌍 저장 완료")

	// Load it
	helpers.LogDetail(t, "3단계: 저장된 키 쌍 로드")
	loadedKeyPair, err := manager.LoadKeyPair(keyPair.ID())
	require.NoError(t, err)
	helpers.LogSuccess(t, "키 쌍 로드 완료")

	// Export it
	helpers.LogDetail(t, "4단계: 키 쌍 JWK 형식으로 내보내기")
	jwkData, err := manager.ExportKeyPair(loadedKeyPair, sagecrypto.KeyFormatJWK)
	require.NoError(t, err)
	helpers.LogDetail(t, "  내보낸 데이터 크기: %d bytes", len(jwkData))
	helpers.LogSuccess(t, "키 쌍 내보내기 완료")

	// Delete it
	helpers.LogDetail(t, "5단계: 키 쌍 삭제")
	err = manager.DeleteKeyPair(keyPair.ID())
	require.NoError(t, err)
	helpers.LogSuccess(t, "키 쌍 삭제 완료")

	// Import it back
	helpers.LogDetail(t, "6단계: JWK 데이터에서 키 쌍 다시 가져오기")
	importedKeyPair, err := manager.ImportKeyPair(jwkData, sagecrypto.KeyFormatJWK)
	require.NoError(t, err)
	helpers.LogSuccess(t, "키 쌍 가져오기 완료")

	// Store it again with its ID
	helpers.LogDetail(t, "7단계: 가져온 키 쌍 다시 저장")
	err = manager.StoreKeyPair(importedKeyPair)
	require.NoError(t, err)
	helpers.LogSuccess(t, "키 쌍 재저장 완료")

	// Verify we can list it
	helpers.LogDetail(t, "8단계: 저장된 키 쌍 목록 확인")
	ids, err := manager.ListKeyPairs()
	require.NoError(t, err)
	assert.Contains(t, ids, importedKeyPair.ID())
	helpers.LogDetail(t, "  목록에서 발견된 키 개수: %d", len(ids))
	helpers.LogSuccess(t, "목록 조회 완료 - 키 쌍 존재 확인")

	helpers.LogPassCriteria(t, []string{
		"1. 키 쌍 생성 성공",
		"2. 키 쌍 저장 성공",
		"3. 저장된 키 쌍 로드 성공",
		"4. 키 쌍 JWK 형식 내보내기 성공",
		"5. 키 쌍 삭제 성공",
		"6. JWK에서 키 쌍 가져오기 성공",
		"7. 가져온 키 쌍 재저장 성공",
		"8. 목록에서 키 쌍 확인 성공",
		"전체 라이프사이클 정상 동작 확인",
	})

	testData := map[string]interface{}{
		"test_case": "18.9.1_전체_라이프사이클_통합",
		"lifecycle_steps": []string{
			"생성",
			"저장",
			"로드",
			"내보내기",
			"삭제",
			"가져오기",
			"재저장",
			"목록확인",
		},
		"key_id":          importedKeyPair.ID(),
		"key_type":        string(importedKeyPair.Type()),
		"jwk_data_size":   len(jwkData),
		"final_key_count": len(ids),
		"validation":      "통합_테스트_검증_통과",
	}
	helpers.SaveTestData(t, "crypto/manager_integration.json", testData)
}
