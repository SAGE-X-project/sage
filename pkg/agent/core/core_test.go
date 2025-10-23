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

package core

import (
	"testing"

	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

func TestCore(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		// 명세 요구사항: Core 패키지 초기화 검증
		helpers.LogTestSection(t, "16.1.1", "Core 패키지 초기화")

		helpers.LogDetail(t, "새로운 Core 인스턴스 생성 중...")
		core := New()

		helpers.LogDetail(t, "Core 인스턴스 검증 중...")
		assert.NotNil(t, core)
		helpers.LogSuccess(t, "Core 인스턴스 생성 완료")

		helpers.LogDetail(t, "내부 매니저 초기화 검증 중...")
		assert.NotNil(t, core.cryptoManager)
		helpers.LogSuccess(t, "Crypto Manager 초기화 완료")

		assert.NotNil(t, core.didManager)
		helpers.LogSuccess(t, "DID Manager 초기화 완료")

		assert.NotNil(t, core.verificationService)
		helpers.LogSuccess(t, "Verification Service 초기화 완료")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"Core 인스턴스가 성공적으로 생성됨",
			"Crypto Manager가 초기화됨",
			"DID Manager가 초기화됨",
			"Verification Service가 초기화됨",
			"모든 내부 컴포넌트가 nil이 아님",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "16.1.1_Core_초기화",
			"initialized_components": []string{
				"cryptoManager",
				"didManager",
				"verificationService",
			},
			"validation": "초기화_검증_통과",
		}
		helpers.SaveTestData(t, "core/core_new.json", testData)
	})

	t.Run("GenerateKeyPair", func(t *testing.T) {
		// 명세 요구사항: 키 쌍 생성 기능 검증
		helpers.LogTestSection(t, "16.1.2", "Core 키 쌍 생성")

		helpers.LogDetail(t, "Core 인스턴스 생성...")
		core := New()

		// Test Ed25519
		helpers.LogDetail(t, "Ed25519 키 쌍 생성 테스트...")
		ed25519Key, err := core.GenerateKeyPair(crypto.KeyTypeEd25519)
		require.NoError(t, err)
		assert.NotNil(t, ed25519Key)
		assert.Equal(t, crypto.KeyTypeEd25519, ed25519Key.Type())
		helpers.LogSuccess(t, "Ed25519 키 쌍 생성 완료")
		helpers.LogDetail(t, "  키 타입: %s", ed25519Key.Type())

		// Test Secp256k1
		helpers.LogDetail(t, "Secp256k1 키 쌍 생성 테스트...")
		secp256k1Key, err := core.GenerateKeyPair(crypto.KeyTypeSecp256k1)
		require.NoError(t, err)
		assert.NotNil(t, secp256k1Key)
		assert.Equal(t, crypto.KeyTypeSecp256k1, secp256k1Key.Type())
		helpers.LogSuccess(t, "Secp256k1 키 쌍 생성 완료")
		helpers.LogDetail(t, "  키 타입: %s", secp256k1Key.Type())

		// Test unsupported type
		helpers.LogDetail(t, "지원되지 않는 키 타입 에러 테스트...")
		_, err = core.GenerateKeyPair(crypto.KeyType("unsupported"))
		assert.Error(t, err)
		helpers.LogSuccess(t, "예상대로 에러 발생")
		helpers.LogDetail(t, "  에러: %s", err.Error())

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"Ed25519 키 쌍 생성 성공",
			"생성된 Ed25519 키 타입 검증",
			"Secp256k1 키 쌍 생성 성공",
			"생성된 Secp256k1 키 타입 검증",
			"지원되지 않는 키 타입에 대해 에러 발생",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "16.1.2_키_쌍_생성",
			"supported_key_types": []string{
				string(crypto.KeyTypeEd25519),
				string(crypto.KeyTypeSecp256k1),
			},
			"ed25519_generated":    true,
			"secp256k1_generated":  true,
			"unsupported_rejected": true,
			"validation":           "키_생성_검증_통과",
		}
		helpers.SaveTestData(t, "core/core_generate_keypair.json", testData)
	})

	t.Run("SignMessage", func(t *testing.T) {
		// 명세 요구사항: 메시지 서명 기능 검증
		helpers.LogTestSection(t, "16.1.3", "Core 메시지 서명")

		helpers.LogDetail(t, "Core 인스턴스 생성...")
		core := New()

		helpers.LogDetail(t, "테스트용 Ed25519 키 쌍 생성...")
		keyPair, err := core.GenerateKeyPair(crypto.KeyTypeEd25519)
		require.NoError(t, err)
		helpers.LogSuccess(t, "키 쌍 생성 완료")

		message := []byte("test message")
		helpers.LogDetail(t, "메시지 서명 중...")
		helpers.LogDetail(t, "  메시지: %s", string(message))
		signature, err := core.SignMessage(keyPair, message)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)
		helpers.LogSuccess(t, "메시지 서명 완료")
		helpers.LogDetail(t, "  서명 길이: %d bytes", len(signature))

		// Verify the signature
		helpers.LogDetail(t, "서명 검증 중...")
		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
		helpers.LogSuccess(t, "서명 검증 성공")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"Ed25519 키 쌍 생성 성공",
			"메시지 서명 성공",
			"서명이 비어있지 않음",
			"생성된 서명 검증 성공",
			"서명/검증 사이클 완료",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":        "16.1.3_메시지_서명",
			"message":          string(message),
			"message_length":   len(message),
			"signature_length": len(signature),
			"key_type":         string(crypto.KeyTypeEd25519),
			"sign_success":     true,
			"verify_success":   true,
			"validation":       "서명_검증_통과",
		}
		helpers.SaveTestData(t, "core/core_sign_message.json", testData)
	})

	t.Run("CreateRFC9421Message", func(t *testing.T) {
		// 명세 요구사항: RFC9421 메시지 생성 기능 검증
		helpers.LogTestSection(t, "16.1.4", "Core RFC9421 메시지 생성")

		helpers.LogDetail(t, "Core 인스턴스 생성...")
		core := New()

		agentDID := "did:sage:ethereum:agent001"
		body := []byte("test body")
		helpers.LogDetail(t, "RFC9421 메시지 빌더 생성 중...")
		helpers.LogDetail(t, "  Agent DID: %s", agentDID)
		helpers.LogDetail(t, "  Body: %s", string(body))

		builder := core.CreateRFC9421Message(agentDID, body)
		assert.NotNil(t, builder)
		helpers.LogSuccess(t, "메시지 빌더 생성 완료")

		helpers.LogDetail(t, "메시지 빌드 중...")
		message := builder.Build()
		helpers.LogSuccess(t, "메시지 빌드 완료")

		helpers.LogDetail(t, "메시지 필드 검증 중...")
		assert.Equal(t, agentDID, message.AgentDID)
		helpers.LogSuccess(t, "Agent DID 일치 확인")

		assert.Equal(t, body, message.Body)
		helpers.LogSuccess(t, "메시지 Body 일치 확인")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"RFC9421 메시지 빌더 생성 성공",
			"빌더가 nil이 아님",
			"메시지 빌드 성공",
			"Agent DID가 올바르게 설정됨",
			"Message Body가 올바르게 설정됨",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "16.1.4_RFC9421_메시지_생성",
			"input": map[string]interface{}{
				"agent_did": agentDID,
				"body":      string(body),
			},
			"output": map[string]interface{}{
				"agent_did": message.AgentDID,
				"body":      string(message.Body),
			},
			"builder_created": true,
			"message_built":   true,
			"validation":      "메시지_생성_검증_통과",
		}
		helpers.SaveTestData(t, "core/core_create_rfc9421_message.json", testData)
	})

	t.Run("ConfigureDID", func(t *testing.T) {
		// 명세 요구사항: DID 레지스트리 설정 검증
		helpers.LogTestSection(t, "16.1.5", "Core DID 레지스트리 설정")

		helpers.LogDetail(t, "Core 인스턴스 생성...")
		core := New()

		config := &did.RegistryConfig{
			Chain:           did.ChainEthereum,
			ContractAddress: "0x1234567890abcdef",
			RPCEndpoint:     "http://localhost:8545",
		}

		helpers.LogDetail(t, "DID 레지스트리 설정 중...")
		helpers.LogDetail(t, "  Chain: %s", config.Chain)
		helpers.LogDetail(t, "  Contract Address: %s", config.ContractAddress)
		helpers.LogDetail(t, "  RPC Endpoint: %s", config.RPCEndpoint)

		// This should succeed now as we only store configuration
		err := core.ConfigureDID(did.ChainEthereum, config)
		assert.NoError(t, err)
		helpers.LogSuccess(t, "DID 레지스트리 설정 완료")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"Core 인스턴스 생성 성공",
			"DID 레지스트리 설정 객체 생성",
			"Ethereum 체인 설정 성공",
			"Contract Address 설정 성공",
			"RPC Endpoint 설정 성공",
			"ConfigureDID 호출 성공 (에러 없음)",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "16.1.5_DID_레지스트리_설정",
			"config": map[string]interface{}{
				"chain":            string(config.Chain),
				"contract_address": config.ContractAddress,
				"rpc_endpoint":     config.RPCEndpoint,
			},
			"configure_success": true,
			"validation":        "DID_설정_검증_통과",
		}
		helpers.SaveTestData(t, "core/core_configure_did.json", testData)
	})

	t.Run("GetManagers", func(t *testing.T) {
		// 명세 요구사항: 매니저 접근자 메서드 검증
		helpers.LogTestSection(t, "16.1.6", "Core 매니저 접근자 검증")

		helpers.LogDetail(t, "Core 인스턴스 생성...")
		core := New()

		helpers.LogDetail(t, "Crypto Manager 접근 테스트...")
		cryptoMgr := core.GetCryptoManager()
		assert.NotNil(t, cryptoMgr)
		helpers.LogSuccess(t, "Crypto Manager 접근 성공")

		helpers.LogDetail(t, "DID Manager 접근 테스트...")
		didMgr := core.GetDIDManager()
		assert.NotNil(t, didMgr)
		helpers.LogSuccess(t, "DID Manager 접근 성공")

		helpers.LogDetail(t, "Verification Service 접근 테스트...")
		verifyService := core.GetVerificationService()
		assert.NotNil(t, verifyService)
		helpers.LogSuccess(t, "Verification Service 접근 성공")

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"GetCryptoManager()가 nil이 아닌 매니저 반환",
			"GetDIDManager()가 nil이 아닌 매니저 반환",
			"GetVerificationService()가 nil이 아닌 서비스 반환",
			"모든 내부 컴포넌트에 접근 가능",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case": "16.1.6_매니저_접근자",
			"managers": map[string]bool{
				"crypto_manager":       cryptoMgr != nil,
				"did_manager":          didMgr != nil,
				"verification_service": verifyService != nil,
			},
			"all_accessible": true,
			"validation":     "매니저_접근_검증_통과",
		}
		helpers.SaveTestData(t, "core/core_get_managers.json", testData)
	})

	t.Run("GetSupportedChains", func(t *testing.T) {
		// 명세 요구사항: 지원 체인 목록 조회 검증
		helpers.LogTestSection(t, "16.1.7", "Core 지원 체인 목록 조회")

		helpers.LogDetail(t, "Core 인스턴스 생성...")
		core := New()

		helpers.LogDetail(t, "지원 체인 목록 조회 중...")
		chains := core.GetSupportedChains()
		assert.NotNil(t, chains)
		helpers.LogSuccess(t, "지원 체인 목록 조회 성공")

		helpers.LogDetail(t, "초기 상태 검증 (설정된 체인 없음)...")
		assert.Empty(t, chains) // No chains configured yet
		helpers.LogSuccess(t, "예상대로 빈 목록 반환")
		helpers.LogDetail(t, "  지원 체인 개수: %d", len(chains))

		// 통과 기준 체크리스트
		helpers.LogPassCriteria(t, []string{
			"GetSupportedChains()가 nil이 아닌 슬라이스 반환",
			"초기 상태에서 빈 목록 반환",
			"설정되지 않은 체인 없음 확인",
		})

		// CLI 검증용 테스트 데이터 저장
		testData := map[string]interface{}{
			"test_case":      "16.1.7_지원_체인_목록",
			"chains_count":   len(chains),
			"chains_empty":   len(chains) == 0,
			"chains_not_nil": chains != nil,
			"validation":     "체인_목록_검증_통과",
		}
		helpers.SaveTestData(t, "core/core_get_supported_chains.json", testData)
	})
}

func TestVersion(t *testing.T) {
	// 명세 요구사항: 버전 상수 검증
	helpers.LogTestSection(t, "16.2.1", "Core 패키지 버전 상수")

	expectedVersion := "0.1.0"
	helpers.LogDetail(t, "버전 상수 검증 중...")
	helpers.LogDetail(t, "  예상 버전: %s", expectedVersion)
	helpers.LogDetail(t, "  실제 버전: %s", Version)

	assert.Equal(t, expectedVersion, Version)
	helpers.LogSuccess(t, "버전 상수 일치 확인")

	// 통과 기준 체크리스트
	helpers.LogPassCriteria(t, []string{
		"Version 상수가 정의됨",
		"버전이 예상 값과 일치",
		"버전 형식이 올바름 (semantic versioning)",
	})

	// CLI 검증용 테스트 데이터 저장
	testData := map[string]interface{}{
		"test_case":        "16.2.1_버전_상수",
		"expected_version": expectedVersion,
		"actual_version":   Version,
		"version_match":    Version == expectedVersion,
		"validation":       "버전_검증_통과",
	}
	helpers.SaveTestData(t, "core/core_version.json", testData)
}
