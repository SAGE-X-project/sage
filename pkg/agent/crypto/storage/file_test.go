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
	"os"
	"path/filepath"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileKeyStorage(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := os.MkdirTemp("", "sage-key-storage-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	storage, err := NewFileKeyStorage(tempDir)
	require.NoError(t, err)

	t.Run("StoreAndLoadKeyPair", func(t *testing.T) {
		// 명세 요구사항: 파일 스토리지에 키 저장 및 로드
		helpers.LogTestSection(t, "20.1.1", "File Storage - 키 쌍 저장 및 로드")

		helpers.LogDetail(t, "Ed25519 키 쌍 생성 중...")
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		helpers.LogDetail(t, "키 쌍 생성 완료: %s", keyPair.ID())

		// Store the key pair
		helpers.LogDetail(t, "키 쌍 파일로 저장 중... (key: test-key)")
		err = storage.Store("test-key", keyPair)
		require.NoError(t, err)
		helpers.LogSuccess(t, "키 쌍 파일 저장 완료")

		// Verify file was created
		keyFile := filepath.Join(tempDir, "test-key.key")
		helpers.LogDetail(t, "파일 존재 여부 확인: %s", keyFile)
		assert.FileExists(t, keyFile)
		helpers.LogSuccess(t, "키 파일 생성 확인")

		// Load the key pair
		helpers.LogDetail(t, "파일에서 키 쌍 로드 중...")
		loadedKeyPair, err := storage.Load("test-key")
		require.NoError(t, err)
		assert.NotNil(t, loadedKeyPair)
		assert.Equal(t, keyPair.Type(), loadedKeyPair.Type())
		helpers.LogSuccess(t, "키 쌍 로드 완료 - 타입 일치")
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
			"키 쌍 파일로 저장 성공",
			"키 파일 생성 확인",
			"파일에서 키 쌍 로드 성공",
			"로드된 키의 타입이 원본과 일치",
			"로드된 키로 서명 생성 성공",
			"원본 키로 서명 검증 성공",
		})

		testData := map[string]interface{}{
			"test_case":   "20.1.1_파일_키_쌍_저장_및_로드",
			"key_id":      keyPair.ID(),
			"key_type":    string(keyPair.Type()),
			"file_path":   keyFile,
			"sign_verify": "성공",
			"validation":  "파일_저장_로드_검증_통과",
		}
		helpers.SaveTestData(t, "storage/file_store_and_load.json", testData)
	})

	t.Run("StoreSecp256k1KeyPair", func(t *testing.T) {
		// 명세 요구사항: Secp256k1 키 타입 파일 저장 및 로드
		helpers.LogTestSection(t, "20.1.2", "File Storage - Secp256k1 키 저장 및 로드")

		helpers.LogDetail(t, "Secp256k1 키 쌍 생성 중...")
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)
		helpers.LogDetail(t, "키 쌍 생성 완료: %s", keyPair.ID())

		// Store the key pair
		helpers.LogDetail(t, "Secp256k1 키 쌍 파일로 저장 중... (key: secp256k1-key)")
		err = storage.Store("secp256k1-key", keyPair)
		require.NoError(t, err)
		helpers.LogSuccess(t, "Secp256k1 키 쌍 파일 저장 완료")

		// Load the key pair
		helpers.LogDetail(t, "파일에서 Secp256k1 키 쌍 로드 중...")
		loadedKeyPair, err := storage.Load("secp256k1-key")
		require.NoError(t, err)
		assert.NotNil(t, loadedKeyPair)
		assert.Equal(t, crypto.KeyTypeSecp256k1, loadedKeyPair.Type())
		helpers.LogSuccess(t, "Secp256k1 키 쌍 로드 완료 - 타입 확인")
		helpers.LogDetail(t, "로드된 키 타입: %s", loadedKeyPair.Type())

		helpers.LogPassCriteria(t, []string{
			"Secp256k1 키 쌍 생성 성공",
			"Secp256k1 키 쌍 파일로 저장 성공",
			"파일에서 키 쌍 로드 성공",
			"로드된 키 타입이 Secp256k1",
		})

		testData := map[string]interface{}{
			"test_case":  "20.1.2_Secp256k1_키_저장_및_로드",
			"key_id":     keyPair.ID(),
			"key_type":   string(loadedKeyPair.Type()),
			"validation": "Secp256k1_저장_로드_검증_통과",
		}
		helpers.SaveTestData(t, "storage/file_store_secp256k1.json", testData)
	})

	t.Run("LoadNonExistentKey", func(t *testing.T) {
		// 명세 요구사항: 존재하지 않는 파일 로드 시 에러 처리
		helpers.LogTestSection(t, "20.1.3", "File Storage - 존재하지 않는 키 로드")

		helpers.LogDetail(t, "존재하지 않는 키 파일 로드 시도... (key: non-existent)")
		_, err := storage.Load("non-existent")
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrKeyNotFound, err)
		helpers.LogSuccess(t, "예상대로 ErrKeyNotFound 에러 발생")
		helpers.LogDetail(t, "에러: %v", err)

		helpers.LogPassCriteria(t, []string{
			"존재하지 않는 키 파일 로드 시도 시 에러 발생",
			"에러가 ErrKeyNotFound",
			"파일 부재 에러 처리가 올바르게 동작",
		})

		testData := map[string]interface{}{
			"test_case":    "20.1.3_존재하지_않는_키_로드",
			"key":          "non-existent",
			"error":        "key not found",
			"error_raised": true,
			"validation":   "에러_처리_검증_통과",
		}
		helpers.SaveTestData(t, "storage/file_load_nonexistent.json", testData)
	})

	t.Run("DeleteKey", func(t *testing.T) {
		// 명세 요구사항: 키 파일 삭제 동작 확인
		helpers.LogTestSection(t, "20.1.4", "File Storage - 키 파일 삭제")

		helpers.LogDetail(t, "Ed25519 키 쌍 생성...")
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		helpers.LogDetail(t, "키 쌍 생성 완료: %s", keyPair.ID())

		// Store the key
		helpers.LogDetail(t, "키 파일 저장 중... (key: delete-test)")
		err = storage.Store("delete-test", keyPair)
		require.NoError(t, err)
		helpers.LogSuccess(t, "키 파일 저장 완료")

		// Verify file exists
		keyFile := filepath.Join(tempDir, "delete-test.key")
		helpers.LogDetail(t, "파일 존재 확인: %s", keyFile)
		assert.FileExists(t, keyFile)
		helpers.LogSuccess(t, "키 파일 존재 확인")

		// Delete the key
		helpers.LogDetail(t, "키 파일 삭제 중...")
		err = storage.Delete("delete-test")
		require.NoError(t, err)
		helpers.LogSuccess(t, "키 파일 삭제 완료")

		// Verify file is gone
		helpers.LogDetail(t, "파일 삭제 확인 중...")
		assert.NoFileExists(t, keyFile)
		helpers.LogSuccess(t, "키 파일이 존재하지 않음 확인")

		// Try to load deleted key
		helpers.LogDetail(t, "삭제된 키 로드 시도...")
		_, err = storage.Load("delete-test")
		assert.Error(t, err)
		assert.Equal(t, crypto.ErrKeyNotFound, err)
		helpers.LogSuccess(t, "예상대로 ErrKeyNotFound 에러 발생")

		helpers.LogPassCriteria(t, []string{
			"키 쌍 생성 및 저장 성공",
			"키 파일 존재 확인 성공",
			"키 파일 삭제 성공",
			"파일 시스템에서 파일 삭제 확인",
			"삭제된 키 로드 시 ErrKeyNotFound 발생",
		})

		testData := map[string]interface{}{
			"test_case":  "20.1.4_키_파일_삭제",
			"key_id":     keyPair.ID(),
			"file_path":  keyFile,
			"deleted":    true,
			"validation": "파일_삭제_검증_통과",
		}
		helpers.SaveTestData(t, "storage/file_delete.json", testData)
	})

	t.Run("ListKeys", func(t *testing.T) {
		// 명세 요구사항: 파일 스토리지의 키 목록 조회
		helpers.LogTestSection(t, "20.1.5", "File Storage - 키 목록 조회")

		// Create new storage in clean directory
		helpers.LogDetail(t, "새로운 임시 디렉터리 생성...")
		listDir, err := os.MkdirTemp("", "sage-list-test-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(listDir) }()
		helpers.LogDetail(t, "임시 디렉터리: %s", listDir)

		listStorage, err := NewFileKeyStorage(listDir)
		require.NoError(t, err)
		helpers.LogSuccess(t, "파일 스토리지 생성 완료")

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

		err = listStorage.Store("key1", keyPair1)
		require.NoError(t, err)
		err = listStorage.Store("key2", keyPair2)
		require.NoError(t, err)
		err = listStorage.Store("key3", keyPair3)
		require.NoError(t, err)
		helpers.LogSuccess(t, "3개의 키 파일 저장 완료")

		// List all keys
		helpers.LogDetail(t, "전체 키 목록 조회 중...")
		ids, err := listStorage.List()
		require.NoError(t, err)
		assert.Len(t, ids, 3)
		assert.Contains(t, ids, "key1")
		assert.Contains(t, ids, "key2")
		assert.Contains(t, ids, "key3")
		helpers.LogSuccess(t, "3개의 키 목록 조회 성공")
		helpers.LogDetail(t, "조회된 키: %v", ids)

		helpers.LogPassCriteria(t, []string{
			"임시 디렉터리 생성 성공",
			"파일 스토리지 생성 성공",
			"Ed25519 키 2개, Secp256k1 키 1개 생성 및 저장 성공",
			"List로 3개의 키 조회 성공",
			"조회된 키 목록에 key1, key2, key3 모두 포함",
		})

		testData := map[string]interface{}{
			"test_case":   "20.1.5_파일_키_목록_조회",
			"directory":   listDir,
			"keys_count":  len(ids),
			"key_ids":     ids,
			"key_types": []string{
				string(keyPair1.Type()),
				string(keyPair2.Type()),
				string(keyPair3.Type()),
			},
			"validation": "목록_조회_검증_통과",
		}
		helpers.SaveTestData(t, "storage/file_list.json", testData)
	})

	t.Run("InvalidKeyID", func(t *testing.T) {
		// 명세 요구사항: 잘못된 키 ID에 대한 에러 처리
		helpers.LogTestSection(t, "20.1.6", "File Storage - 잘못된 키 ID 처리")

		helpers.LogDetail(t, "Ed25519 키 쌍 생성...")
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		helpers.LogDetail(t, "키 쌍 생성 완료: %s", keyPair.ID())

		// Try to store with invalid ID containing path separator
		helpers.LogDetail(t, "경로 구분자 포함 키 ID로 저장 시도: ../invalid/key")
		err = storage.Store("../invalid/key", keyPair)
		assert.Error(t, err)
		helpers.LogSuccess(t, "예상대로 에러 발생")
		helpers.LogDetail(t, "에러: %v", err)

		helpers.LogDetail(t, "백슬래시 포함 키 ID로 저장 시도: invalid\\key")
		err = storage.Store("invalid\\key", keyPair)
		assert.Error(t, err)
		helpers.LogSuccess(t, "예상대로 에러 발생")
		helpers.LogDetail(t, "에러: %v", err)

		helpers.LogPassCriteria(t, []string{
			"키 쌍 생성 성공",
			"경로 구분자 포함 키 ID 저장 시도 시 에러 발생",
			"백슬래시 포함 키 ID 저장 시도 시 에러 발생",
			"경로 탐색 공격 방지 확인",
		})

		testData := map[string]interface{}{
			"test_case": "20.1.6_잘못된_키_ID_처리",
			"invalid_ids": []string{
				"../invalid/key",
				"invalid\\key",
			},
			"security":  "경로_탐색_공격_방지",
			"validation": "보안_검증_통과",
		}
		helpers.SaveTestData(t, "storage/file_invalid_key_id.json", testData)
	})

	t.Run("CorruptedKeyFile", func(t *testing.T) {
		// 명세 요구사항: 손상된 키 파일 로드 시 에러 처리
		helpers.LogTestSection(t, "20.1.7", "File Storage - 손상된 키 파일 처리")

		// Create a corrupted key file
		corruptedFile := filepath.Join(tempDir, "corrupted.key")
		helpers.LogDetail(t, "손상된 키 파일 생성: %s", corruptedFile)
		err := os.WriteFile(corruptedFile, []byte("corrupted data"), 0600)
		require.NoError(t, err)
		helpers.LogSuccess(t, "손상된 키 파일 생성 완료")

		// Try to load corrupted key
		helpers.LogDetail(t, "손상된 키 파일 로드 시도...")
		_, err = storage.Load("corrupted")
		assert.Error(t, err)
		helpers.LogSuccess(t, "예상대로 에러 발생")
		helpers.LogDetail(t, "에러: %v", err)

		helpers.LogPassCriteria(t, []string{
			"손상된 키 파일 생성 성공",
			"손상된 키 파일 로드 시도 시 에러 발생",
			"잘못된 형식 파일에 대한 에러 처리가 올바르게 동작",
		})

		testData := map[string]interface{}{
			"test_case":    "20.1.7_손상된_키_파일_처리",
			"file_path":    corruptedFile,
			"file_content": "corrupted data",
			"error_raised": true,
			"validation":   "손상된_파일_검증_통과",
		}
		helpers.SaveTestData(t, "storage/file_corrupted.json", testData)
	})

	t.Run("FilePermissions", func(t *testing.T) {
		// 명세 요구사항: 키 파일 권한 검증 (0600)
		helpers.LogTestSection(t, "20.1.8", "File Storage - 파일 권한 검증")

		helpers.LogDetail(t, "Ed25519 키 쌍 생성...")
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)
		helpers.LogDetail(t, "키 쌍 생성 완료: %s", keyPair.ID())

		// Store the key
		helpers.LogDetail(t, "키 파일 저장 중... (key: perm-test)")
		err = storage.Store("perm-test", keyPair)
		require.NoError(t, err)
		helpers.LogSuccess(t, "키 파일 저장 완료")

		// Check file permissions
		keyFile := filepath.Join(tempDir, "perm-test.key")
		helpers.LogDetail(t, "파일 권한 확인: %s", keyFile)
		info, err := os.Stat(keyFile)
		require.NoError(t, err)

		// Should be readable/writable by owner only (0600)
		actualPerm := info.Mode().Perm()
		expectedPerm := os.FileMode(0600)
		helpers.LogDetail(t, "예상 권한: %o", expectedPerm)
		helpers.LogDetail(t, "실제 권한: %o", actualPerm)
		assert.Equal(t, expectedPerm, actualPerm)
		helpers.LogSuccess(t, "파일 권한 확인 완료 - 소유자만 읽기/쓰기 가능 (0600)")

		helpers.LogPassCriteria(t, []string{
			"키 쌍 생성 성공",
			"키 파일 저장 성공",
			"파일 권한 정보 조회 성공",
			"파일 권한이 0600 (소유자 읽기/쓰기만 허용)",
			"보안 요구사항 충족",
		})

		testData := map[string]interface{}{
			"test_case":         "20.1.8_파일_권한_검증",
			"file_path":         keyFile,
			"expected_perm":     "0600",
			"actual_perm":       fmt.Sprintf("%o", actualPerm),
			"security":          "소유자_전용_권한",
			"validation":        "권한_검증_통과",
		}
		helpers.SaveTestData(t, "storage/file_permissions.json", testData)
	})
}
