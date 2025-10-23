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

package storage

import (
	"fmt"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryKeyStorage(t *testing.T) {
	storage := NewMemoryKeyStorage()

	t.Run("StoreAndLoadKeyPair", func(t *testing.T) {
		// 명세 요구사항: 메모리 스토리지에 키 저장 및 로드
		helpers.LogTestSection(t, "19.1.1", "Memory Storage - 키 쌍 저장 및 로드")

		helpers.LogDetail(t, "Ed25519 키 쌍 생성 중...")
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		helpers.LogDetail(t, "키 쌍 생성 완료: %s", keyPair.ID())

		// Store the key pair
		helpers.LogDetail(t, "키 쌍 저장 중... (key: test-key)")
		err = storage.Store("test-key", keyPair)
		require.NoError(t, err)
		helpers.LogSuccess(t, "키 쌍 저장 완료")

		// Load the key pair
		helpers.LogDetail(t, "키 쌍 로드 중... (key: test-key)")
		loadedKeyPair, err := storage.Load("test-key")
		require.NoError(t, err)
		assert.NotNil(t, loadedKeyPair)
		assert.Equal(t, keyPair.ID(), loadedKeyPair.ID())
		assert.Equal(t, keyPair.Type(), loadedKeyPair.Type())
		helpers.LogSuccess(t, "키 쌍 로드 완료 - 원본과 일치")
		helpers.LogDetail(t, "로드된 키 ID: %s", loadedKeyPair.ID())
		helpers.LogDetail(t, "로드된 키 타입: %s", loadedKeyPair.Type())

		// Test signing with loaded key
		helpers.LogDetail(t, "로드된 키로 서명 테스트 중...")
		message := []byte("test message")
		signature, err := loadedKeyPair.Sign(message)
		require.NoError(t, err)
		helpers.LogDetail(t, "서명 생성 완료 (%d bytes)", len(signature))

		// Verify with original key
		helpers.LogDetail(t, "원본 키로 서명 검증 중...")
		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
		helpers.LogSuccess(t, "서명 검증 성공 - 키가 올바르게 저장/로드됨")

		helpers.LogPassCriteria(t, []string{
			"Ed25519 키 쌍 생성 성공",
			"키 쌍 메모리 스토리지에 저장 성공",
			"저장된 키 쌍 로드 성공",
			"로드된 키의 ID와 타입이 원본과 일치",
			"로드된 키로 서명 생성 성공",
			"원본 키로 서명 검증 성공",
		})

		testData := map[string]interface{}{
			"test_case":   "19.1.1_키_쌍_저장_및_로드",
			"key_id":      keyPair.ID(),
			"key_type":    string(keyPair.Type()),
			"storage_key": "test-key",
			"sign_verify": "성공",
			"validation":  "저장_로드_검증_통과",
		}
		helpers.SaveTestData(t, "storage/memory_store_and_load.json", testData)
	})

	t.Run("LoadNonExistentKey", func(t *testing.T) {
		// 명세 요구사항: 존재하지 않는 키 로드 시 에러 처리
		helpers.LogTestSection(t, "19.1.2", "Memory Storage - 존재하지 않는 키 로드")

		helpers.LogDetail(t, "존재하지 않는 키 로드 시도... (key: non-existent)")
		_, err := storage.Load("non-existent")
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrKeyNotFound, err)
		helpers.LogSuccess(t, "예상대로 ErrKeyNotFound 에러 발생")
		helpers.LogDetail(t, "에러: %v", err)

		helpers.LogPassCriteria(t, []string{
			"존재하지 않는 키 로드 시도 시 에러 발생",
			"에러가 ErrKeyNotFound",
			"에러 처리가 올바르게 동작",
		})

		testData := map[string]interface{}{
			"test_case":    "19.1.2_존재하지_않는_키_로드",
			"key":          "non-existent",
			"error":        "key not found",
			"error_raised": true,
			"validation":   "에러_처리_검증_통과",
		}
		helpers.SaveTestData(t, "storage/memory_load_nonexistent.json", testData)
	})

	t.Run("OverwriteExistingKey", func(t *testing.T) {
		// 명세 요구사항: 기존 키 덮어쓰기 동작 확인
		helpers.LogTestSection(t, "19.1.3", "Memory Storage - 기존 키 덮어쓰기")

		helpers.LogDetail(t, "첫 번째 Ed25519 키 쌍 생성...")
		keyPair1, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		helpers.LogDetail(t, "  키 1 ID: %s", keyPair1.ID())

		helpers.LogDetail(t, "두 번째 Ed25519 키 쌍 생성...")
		keyPair2, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		helpers.LogDetail(t, "  키 2 ID: %s", keyPair2.ID())

		// Store first key
		helpers.LogDetail(t, "첫 번째 키 저장... (key: overwrite-test)")
		err = storage.Store("overwrite-test", keyPair1)
		require.NoError(t, err)
		helpers.LogSuccess(t, "첫 번째 키 저장 완료")

		// Overwrite with second key
		helpers.LogDetail(t, "두 번째 키로 덮어쓰기... (key: overwrite-test)")
		err = storage.Store("overwrite-test", keyPair2)
		require.NoError(t, err)
		helpers.LogSuccess(t, "두 번째 키로 덮어쓰기 완료")

		// Load should return the second key
		helpers.LogDetail(t, "덮어쓴 키 로드 중...")
		loadedKeyPair, err := storage.Load("overwrite-test")
		require.NoError(t, err)
		assert.Equal(t, keyPair2.ID(), loadedKeyPair.ID())
		helpers.LogSuccess(t, "두 번째 키가 로드됨 - 덮어쓰기 성공")
		helpers.LogDetail(t, "로드된 키 ID: %s (키 2와 일치)", loadedKeyPair.ID())

		helpers.LogPassCriteria(t, []string{
			"두 개의 Ed25519 키 쌍 생성 성공",
			"첫 번째 키 저장 성공",
			"같은 키 이름으로 두 번째 키 저장 성공",
			"로드된 키가 두 번째 키와 일치",
			"덮어쓰기 동작이 올바르게 작동",
		})

		testData := map[string]interface{}{
			"test_case":   "19.1.3_기존_키_덮어쓰기",
			"storage_key": "overwrite-test",
			"key1_id":     keyPair1.ID(),
			"key2_id":     keyPair2.ID(),
			"loaded_id":   loadedKeyPair.ID(),
			"overwrite":   "성공",
			"validation":  "덮어쓰기_검증_통과",
		}
		helpers.SaveTestData(t, "storage/memory_overwrite.json", testData)
	})

	t.Run("DeleteKey", func(t *testing.T) {
		// 명세 요구사항: 키 삭제 동작 확인
		helpers.LogTestSection(t, "19.1.4", "Memory Storage - 키 삭제")

		helpers.LogDetail(t, "Ed25519 키 쌍 생성...")
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		helpers.LogDetail(t, "키 쌍 생성 완료: %s", keyPair.ID())

		// Store the key
		helpers.LogDetail(t, "키 저장 중... (key: delete-test)")
		err = storage.Store("delete-test", keyPair)
		require.NoError(t, err)
		helpers.LogSuccess(t, "키 저장 완료")

		// Verify it exists
		helpers.LogDetail(t, "키 존재 여부 확인...")
		assert.True(t, storage.Exists("delete-test"))
		helpers.LogSuccess(t, "키 존재 확인")

		// Delete the key
		helpers.LogDetail(t, "키 삭제 중... (key: delete-test)")
		err = storage.Delete("delete-test")
		require.NoError(t, err)
		helpers.LogSuccess(t, "키 삭제 완료")

		// Verify it's gone
		helpers.LogDetail(t, "키 삭제 확인 중...")
		assert.False(t, storage.Exists("delete-test"))
		helpers.LogSuccess(t, "키가 존재하지 않음 확인")

		// Try to load deleted key
		helpers.LogDetail(t, "삭제된 키 로드 시도...")
		_, err = storage.Load("delete-test")
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrKeyNotFound, err)
		helpers.LogSuccess(t, "예상대로 ErrKeyNotFound 에러 발생")

		helpers.LogPassCriteria(t, []string{
			"키 쌍 생성 및 저장 성공",
			"Exists로 키 존재 확인 성공",
			"키 삭제 성공",
			"Exists로 키 부재 확인 성공",
			"삭제된 키 로드 시 ErrKeyNotFound 발생",
		})

		testData := map[string]interface{}{
			"test_case":   "19.1.4_키_삭제",
			"storage_key": "delete-test",
			"key_id":      keyPair.ID(),
			"deleted":     true,
			"validation":  "삭제_검증_통과",
		}
		helpers.SaveTestData(t, "storage/memory_delete.json", testData)
	})

	t.Run("DeleteNonExistentKey", func(t *testing.T) {
		// 명세 요구사항: 존재하지 않는 키 삭제 시 에러 처리
		helpers.LogTestSection(t, "19.1.5", "Memory Storage - 존재하지 않는 키 삭제")

		helpers.LogDetail(t, "존재하지 않는 키 삭제 시도... (key: non-existent)")
		err := storage.Delete("non-existent")
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrKeyNotFound, err)
		helpers.LogSuccess(t, "예상대로 ErrKeyNotFound 에러 발생")
		helpers.LogDetail(t, "에러: %v", err)

		helpers.LogPassCriteria(t, []string{
			"존재하지 않는 키 삭제 시도 시 에러 발생",
			"에러가 ErrKeyNotFound",
			"에러 처리가 올바르게 동작",
		})

		testData := map[string]interface{}{
			"test_case":    "19.1.5_존재하지_않는_키_삭제",
			"key":          "non-existent",
			"error":        "key not found",
			"error_raised": true,
			"validation":   "에러_처리_검증_통과",
		}
		helpers.SaveTestData(t, "storage/memory_delete_nonexistent.json", testData)
	})

	t.Run("ListKeys", func(t *testing.T) {
		// 명세 요구사항: 저장된 모든 키 목록 조회
		helpers.LogTestSection(t, "19.1.6", "Memory Storage - 키 목록 조회")

		// Clear storage first
		helpers.LogDetail(t, "새로운 스토리지 생성 (초기화)")
		storage = NewMemoryKeyStorage()

		// Add multiple keys
		helpers.LogDetail(t, "3개의 키 쌍 생성 및 저장 중...")
		keyPair1, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		helpers.LogDetail(t, "  키 1 (Ed25519): %s", keyPair1.ID())

		keyPair2, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)
		helpers.LogDetail(t, "  키 2 (Secp256k1): %s", keyPair2.ID())

		keyPair3, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		helpers.LogDetail(t, "  키 3 (Ed25519): %s", keyPair3.ID())

		err = storage.Store("key1", keyPair1)
		require.NoError(t, err)
		err = storage.Store("key2", keyPair2)
		require.NoError(t, err)
		err = storage.Store("key3", keyPair3)
		require.NoError(t, err)
		helpers.LogSuccess(t, "3개의 키 쌍 저장 완료")

		// List all keys
		helpers.LogDetail(t, "전체 키 목록 조회 중...")
		ids, err := storage.List()
		require.NoError(t, err)
		assert.Len(t, ids, 3)
		assert.Contains(t, ids, "key1")
		assert.Contains(t, ids, "key2")
		assert.Contains(t, ids, "key3")
		helpers.LogSuccess(t, "3개의 키 목록 조회 성공")
		helpers.LogDetail(t, "조회된 키: %v", ids)

		helpers.LogPassCriteria(t, []string{
			"새로운 스토리지 생성 성공",
			"Ed25519 키 2개, Secp256k1 키 1개 생성 및 저장 성공",
			"List로 3개의 키 조회 성공",
			"조회된 키 목록에 key1, key2, key3 모두 포함",
		})

		testData := map[string]interface{}{
			"test_case":  "19.1.6_키_목록_조회",
			"keys_count": len(ids),
			"key_ids":    ids,
			"key_types": []string{
				string(keyPair1.Type()),
				string(keyPair2.Type()),
				string(keyPair3.Type()),
			},
			"validation": "목록_조회_검증_통과",
		}
		helpers.SaveTestData(t, "storage/memory_list.json", testData)
	})

	t.Run("EmptyStorageList", func(t *testing.T) {
		// 명세 요구사항: 빈 스토리지의 키 목록 조회
		helpers.LogTestSection(t, "19.1.7", "Memory Storage - 빈 스토리지 목록 조회")

		helpers.LogDetail(t, "빈 스토리지 생성...")
		emptyStorage := NewMemoryKeyStorage()

		helpers.LogDetail(t, "빈 스토리지의 키 목록 조회 중...")
		ids, err := emptyStorage.List()
		require.NoError(t, err)
		assert.Empty(t, ids)
		helpers.LogSuccess(t, "빈 목록 반환 성공")
		helpers.LogDetail(t, "반환된 키 개수: %d", len(ids))

		helpers.LogPassCriteria(t, []string{
			"빈 스토리지 생성 성공",
			"빈 스토리지에서 목록 조회 성공",
			"반환된 키 목록이 비어있음",
			"에러가 발생하지 않음",
		})

		testData := map[string]interface{}{
			"test_case":  "19.1.7_빈_스토리지_목록_조회",
			"keys_count": len(ids),
			"validation": "빈_목록_검증_통과",
		}
		helpers.SaveTestData(t, "storage/memory_list_empty.json", testData)
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		// 명세 요구사항: 동시 접근 시 스레드 안전성 확인
		helpers.LogTestSection(t, "19.1.8", "Memory Storage - 동시 접근 테스트")

		helpers.LogDetail(t, "새로운 스토리지 생성...")
		storage := NewMemoryKeyStorage()
		done := make(chan bool)

		// Multiple goroutines storing keys
		helpers.LogDetail(t, "10개의 고루틴으로 동시 키 저장 시작...")
		for i := 0; i < 10; i++ {
			go func(id int) {
				keyPair, _ := keys.GenerateEd25519KeyPair()
				_ = storage.Store(fmt.Sprintf("concurrent-%d", id), keyPair)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		helpers.LogDetail(t, "모든 고루틴 완료 대기 중...")
		for i := 0; i < 10; i++ {
			<-done
		}
		helpers.LogSuccess(t, "10개의 고루틴 모두 완료")

		// Verify all keys were stored
		helpers.LogDetail(t, "저장된 키 목록 검증 중...")
		ids, err := storage.List()
		require.NoError(t, err)
		assert.Len(t, ids, 10)
		helpers.LogSuccess(t, "10개의 키가 모두 저장됨")
		helpers.LogDetail(t, "저장된 키 개수: %d", len(ids))

		helpers.LogPassCriteria(t, []string{
			"스토리지 생성 성공",
			"10개의 고루틴이 동시에 키 저장 시작",
			"모든 고루틴이 정상적으로 완료",
			"10개의 키가 모두 저장됨",
			"동시 접근 시 데이터 손실 없음",
			"스레드 안전성 검증 통과",
		})

		testData := map[string]interface{}{
			"test_case":       "19.1.8_동시_접근_테스트",
			"goroutines":      10,
			"keys_stored":     len(ids),
			"concurrent_safe": true,
			"validation":      "동시_접근_검증_통과",
		}
		helpers.SaveTestData(t, "storage/memory_concurrent.json", testData)
	})
}
