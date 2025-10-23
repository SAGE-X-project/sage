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

package did

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	assert.Equal(t, "0.1.0", Version)
}

func TestGetDefaultManager(t *testing.T) {
	manager := GetDefaultManager()
	assert.NotNil(t, manager)
	assert.Equal(t, defaultManager, manager)
}

func TestPackageLevelFunctions(t *testing.T) {
	ctx := context.Background()

	// Save original default manager
	originalManager := defaultManager
	defer func() {
		defaultManager = originalManager
	}()

	// Create a new manager with mocks
	defaultManager = NewManager()
	mockRegistry := new(MockRegistry)
	mockResolver := new(MockResolver)

	// Add mocks to the default manager
	defaultManager.registry.registries[ChainEthereum] = mockRegistry
	defaultManager.resolver.resolvers[ChainEthereum] = mockResolver

	t.Run("Configure", func(t *testing.T) {
		config := &RegistryConfig{
			Chain:           ChainEthereum,
			ContractAddress: "0x1234567890abcdef",
			RPCEndpoint:     "http://localhost:8545",
		}

		// This should succeed now as we only store configuration
		err := Configure(ChainEthereum, config)
		assert.NoError(t, err)
	})

	t.Run("RegisterAgent", func(t *testing.T) {
		mockKeyPair := new(MockKeyPair)
		req := &RegistrationRequest{
			DID:      "agent001",
			Name:     "Test Agent",
			Endpoint: "https://api.example.com",
			KeyPair:  mockKeyPair,
		}

		expectedResult := &RegistrationResult{
			TransactionHash: "0xabc123",
			BlockNumber:     12345,
		}

		mockRegistry.On("Register", ctx, mock.MatchedBy(func(r *RegistrationRequest) bool {
			return true
		})).Return(expectedResult, nil).Once()

		result, err := RegisterAgent(ctx, ChainEthereum, req)
		require.NoError(t, err)
		assert.Equal(t, expectedResult, result)

		mockRegistry.AssertExpectations(t)
	})

	t.Run("ResolveAgent", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent001")
		expectedMetadata := &AgentMetadata{
			DID:      did,
			Name:     "Test Agent",
			IsActive: true,
		}

		mockResolver.On("Resolve", ctx, did).Return(expectedMetadata, nil).Once()

		metadata, err := ResolveAgent(ctx, did)
		require.NoError(t, err)
		assert.Equal(t, expectedMetadata, metadata)

		mockResolver.AssertExpectations(t)
	})

	t.Run("ValidateAgent", func(t *testing.T) {
		// Generate test keypair
		publicKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		did := AgentDID("did:sage:ethereum:agent001")

		agent := &AgentMetadata{
			DID:       did,
			Name:      "Test Agent",
			IsActive:  true,
			PublicKey: publicKey,
			Capabilities: map[string]interface{}{
				"messaging": true,
			},
		}

		mockResolver.On("Resolve", ctx, did).Return(agent, nil).Once()

		result, err := ValidateAgent(ctx, did, nil)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, agent.Name, result.Name)

		mockResolver.AssertExpectations(t)
	})
}

func TestCheckCapabilities(t *testing.T) {
	ctx := context.Background()

	// Save original default manager
	originalManager := defaultManager
	defer func() {
		defaultManager = originalManager
	}()

	// Create a new manager with mocks
	defaultManager = NewManager()
	mockResolver := new(MockResolver)
	defaultManager.resolver.resolvers[ChainEthereum] = mockResolver

	t.Run("CheckCapabilities with all capabilities present", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent001")

		agent := &AgentMetadata{
			DID:      did,
			IsActive: true,
			Capabilities: map[string]interface{}{
				"messaging": true,
				"compute":   true,
				"storage":   true,
			},
		}

		mockResolver.On("Resolve", ctx, did).Return(agent, nil).Once()

		hasCapabilities, err := CheckCapabilities(ctx, did, []string{"messaging", "compute"})
		assert.NoError(t, err)
		assert.True(t, hasCapabilities)

		mockResolver.AssertExpectations(t)
	})

	t.Run("CheckCapabilities with missing capabilities", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent002")

		agent := &AgentMetadata{
			DID:      did,
			IsActive: true,
			Capabilities: map[string]interface{}{
				"messaging": true,
			},
		}

		mockResolver.On("Resolve", ctx, did).Return(agent, nil).Once()

		hasCapabilities, err := CheckCapabilities(ctx, did, []string{"messaging", "compute"})
		assert.NoError(t, err)
		assert.False(t, hasCapabilities)

		mockResolver.AssertExpectations(t)
	})

	t.Run("CheckCapabilities with inactive agent", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent003")

		agent := &AgentMetadata{
			DID:      did,
			IsActive: false,
			Capabilities: map[string]interface{}{
				"messaging": true,
			},
		}

		mockResolver.On("Resolve", ctx, did).Return(agent, nil).Once()

		hasCapabilities, err := CheckCapabilities(ctx, did, []string{"messaging"})
		assert.Error(t, err)
		assert.Equal(t, ErrInactiveAgent, err)
		assert.False(t, hasCapabilities)

		mockResolver.AssertExpectations(t)
	})

	t.Run("CheckCapabilities with resolve error", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent004")

		mockResolver.On("Resolve", ctx, did).
			Return(nil, ErrDIDNotFound).Once()

		hasCapabilities, err := CheckCapabilities(ctx, did, []string{"messaging"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve agent DID")
		assert.False(t, hasCapabilities)

		mockResolver.AssertExpectations(t)
	})
}

func TestValidateDID(t *testing.T) {
	tests := []struct {
		name        string
		did         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid Ethereum DID",
			did:         "did:sage:ethereum:agent001",
			expectError: false,
		},
		{
			name:        "Valid Solana DID",
			did:         "did:sage:solana:agent002",
			expectError: false,
		},
		{
			name:        "Valid short chain prefix",
			did:         "did:sage:eth:agent001",
			expectError: false,
		},
		{
			name:        "Too short",
			did:         "did:sage",
			expectError: true,
			errorMsg:    "DID too short",
		},
		{
			name:        "Wrong prefix",
			did:         "invalid:sage:ethereum:agent001",
			expectError: true,
			errorMsg:    "DID must start with 'did:'",
		},
		{
			name:        "Invalid format",
			did:         "did:invalid",
			expectError: true,
			errorMsg:    "invalid DID format",
		},
		{
			name:        "Unknown chain",
			did:         "did:sage:unknown:agent001",
			expectError: true,
			errorMsg:    "unknown chain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDID(tt.did)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test 3.1.1: DID 생성
func TestCreateDID(t *testing.T) {
	// Specification Requirement: SAGE DID creation with format did:sage:ethereum:<uuid>
	helpers.LogTestSection(t, "3.1.1", "DID 생성 (did:sage:ethereum:<uuid> 형식)")

	helpers.LogDetail(t, "DID 생성 테스트:")

	// Generate UUID v4 as identifier
	uuidVal := uuid.New()
	require.NotNil(t, uuidVal)
	helpers.LogDetail(t, "  생성된 UUID: %s", uuidVal.String())

	// Use SAGE's GenerateDID function (NOT manual string concatenation)
	did := GenerateDID(ChainEthereum, uuidVal.String())

	helpers.LogSuccess(t, "DID 생성 완료 (SAGE GenerateDID 사용)")
	helpers.LogDetail(t, "  DID: %s", did)
	helpers.LogDetail(t, "  DID 길이: %d characters", len(did))

	// Verify DID format using SAGE's ValidateDID function
	err := ValidateDID(string(did))
	require.NoError(t, err, "DID should be valid according to SAGE ValidateDID")
	helpers.LogSuccess(t, "DID 형식 검증 완료 (SAGE ValidateDID 사용)")

	// Parse DID components
	parts := strings.Split(string(did), ":")
	require.Len(t, parts, 4, "DID should have 4 parts")
	helpers.LogDetail(t, "  DID 구성 요소:")
	helpers.LogDetail(t, "    Method: %s", parts[1])
	helpers.LogDetail(t, "    Network: %s", parts[2])
	helpers.LogDetail(t, "    ID: %s", parts[3])

	// Verify components
	assert.Equal(t, "did", parts[0], "First part should be 'did'")
	assert.Equal(t, "sage", parts[1], "Method should be 'sage'")
	assert.Equal(t, "ethereum", parts[2], "Network should be 'ethereum'")
	helpers.LogSuccess(t, "DID 구성 요소 검증 완료")

	// Verify UUID v4 format
	parsedUUID, err2 := uuid.Parse(parts[3])
	require.NoError(t, err2, "ID part should be valid UUID")
	require.Equal(t, uuidVal, parsedUUID, "Parsed UUID should match original")
	require.Equal(t, uuid.Version(4), parsedUUID.Version(), "UUID should be version 4")
	helpers.LogSuccess(t, "UUID v4 형식 검증 완료")
	helpers.LogDetail(t, "  UUID 버전: %d", parsedUUID.Version())

	// Test duplicate DID creation with same UUID
	didDuplicate := GenerateDID(ChainEthereum, uuidVal.String())
	require.Equal(t, did, didDuplicate, "Same UUID should generate same DID")
	helpers.LogSuccess(t, "중복 DID 생성 검증 완료 (같은 UUID → 같은 DID)")
	helpers.LogDetail(t, "  원본 DID: %s", did)
	helpers.LogDetail(t, "  중복 DID: %s", didDuplicate)

	// Test multiple DID creation (uniqueness) - using SAGE's GenerateDID
	did2 := GenerateDID(ChainEthereum, uuid.New().String())
	require.NotEqual(t, did, did2, "Different UUIDs should generate different DIDs")
	helpers.LogSuccess(t, "DID 고유성 검증 완료 (다른 UUID → 다른 DID)")
	helpers.LogDetail(t, "  두 번째 DID: %s", did2)

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"DID 생성 성공 (SAGE GenerateDID 사용)",
		"형식 검증 (SAGE ValidateDID 사용)",
		"형식: did:sage:ethereum:<uuid>",
		"UUID v4 형식 검증",
		"DID 구성 요소 파싱",
		"Method = 'sage'",
		"Network = 'ethereum'",
		"UUID 유효성 확인",
		"중복 DID 검증 (같은 UUID → 같은 DID)",
		"DID 고유성 확인 (다른 UUID → 다른 DID)",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case": "3.1.1_DID_Creation",
		"did":       string(did),
		"format":    "did:sage:ethereum:<uuid>",
		"uuid":      uuidVal.String(),
		"uuid_version": parsedUUID.Version(),
		"components": map[string]string{
			"prefix":  parts[0],
			"method":  parts[1],
			"network": parts[2],
			"id":      parts[3],
		},
		"validation": map[string]bool{
			"sage_validate_did": err == nil,
			"uuid_valid":        err2 == nil,
			"uuid_v4":           parsedUUID.Version() == 4,
			"duplicate_check":   did == didDuplicate,
			"unique":            did != did2,
		},
		"sage_functions_used": []string{
			"GenerateDID(chain, identifier)",
			"ValidateDID(did)",
		},
	}
	helpers.SaveTestData(t, "did/did_creation.json", testData)
}

// Test 3.1.1.2: 중복 DID 생성 시 오류 반환
func TestDIDDuplicateDetection(t *testing.T) {
	// Specification Requirement: Detect duplicate DID by attempting double registration
	helpers.LogTestSection(t, "3.1.1.2", "중복 DID 생성 시 오류 반환 (중복 등록 시도)")

	// Skip if not integration test
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	ctx := context.Background()

	// Save original default manager
	originalManager := defaultManager
	defer func() {
		defaultManager = originalManager
	}()

	// Create a new manager
	defaultManager = NewManager()

	// Configure for Ethereum V4
	config := &RegistryConfig{
		Chain:              ChainEthereum,
		ContractAddress:    "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9", // V4 contract
		RPCEndpoint:        "http://localhost:8545",
		PrivateKey:         "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
		ConfirmationBlocks: 1,
	}

	err := Configure(ChainEthereum, config)
	require.NoError(t, err, "Failed to configure DID manager")
	helpers.LogSuccess(t, "DID Manager 설정 완료")

	// Step 1: Generate a new DID
	uuidVal := uuid.New()
	testDID := GenerateDID(ChainEthereum, uuidVal.String())
	helpers.LogDetail(t, "생성된 테스트 DID: %s", testDID)

	// Step 2: Generate Secp256k1 keypair for Agent
	helpers.LogDetail(t, "[Step 1] Secp256k1 키페어 생성...")
	agentKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err, "Failed to generate keypair")
	helpers.LogSuccess(t, "키페어 생성 완료")

	// Step 3: Fund the agent key with ETH from Hardhat default account
	// Note: This step requires access to ethereum client internals
	// For now, we'll use the manager's RegisterAgent which handles this internally
	helpers.LogDetail(t, "[Step 2] 첫 번째 Agent 등록 시도...")

	// Create registration request
	req := &RegistrationRequest{
		DID:      testDID,
		Name:     "Test Agent for Duplicate Detection",
		Endpoint: "http://localhost:8080",
		KeyPair:  agentKeyPair,
	}

	// First registration should succeed
	result1, err := RegisterAgent(ctx, ChainEthereum, req)
	require.NoError(t, err, "First registration should succeed")
	require.NotNil(t, result1, "Registration result should not be nil")
	helpers.LogSuccess(t, "첫 번째 Agent 등록 성공")
	helpers.LogDetail(t, "  Transaction Hash: %s", result1.TransactionHash)
	helpers.LogDetail(t, "  Block Number: %d", result1.BlockNumber)

	// Step 4: Verify DID can be resolved
	helpers.LogDetail(t, "[Step 3] 등록된 DID 조회...")
	agent, err := ResolveAgent(ctx, testDID)
	require.NoError(t, err, "Should resolve registered DID")
	require.NotNil(t, agent, "Agent should be found")
	assert.Equal(t, testDID, agent.DID, "DID should match")
	helpers.LogSuccess(t, "DID 조회 성공")
	helpers.LogDetail(t, "  Agent 이름: %s", agent.Name)

	// Step 5: Try to register the same DID again (should fail)
	helpers.LogDetail(t, "[Step 4] 동일한 DID로 재등록 시도...")

	// Second registration with the same DID should fail
	result2, err := RegisterAgent(ctx, ChainEthereum, req)

	// Verify that duplicate registration fails
	if err != nil {
		// Expected: Error should occur
		helpers.LogSuccess(t, "중복 등록 시 오류 발생 (예상된 동작)")
		helpers.LogDetail(t, "  에러 메시지: %v", err)

		// Check if it's the expected error type
		if errors.Is(err, ErrDIDAlreadyExists) || strings.Contains(err.Error(), "already") {
			helpers.LogSuccess(t, "중복 DID 에러 확인 (ErrDIDAlreadyExists 또는 유사)")
		} else {
			t.Logf("Warning: Error occurred but may not be the expected type: %v", err)
		}

		assert.Nil(t, result2, "Second registration result should be nil")
	} else {
		// Unexpected: No error occurred
		t.Errorf("Expected error for duplicate registration, but succeeded")
		if result2 != nil {
			t.Errorf("  Transaction Hash: %s", result2.TransactionHash)
		}
	}

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"DID 생성 (SAGE GenerateDID 사용)",
		"Secp256k1 키페어 생성",
		"첫 번째 Agent 등록 성공",
		"등록된 DID 조회 성공 (SAGE ResolveAgent)",
		"동일 DID 재등록 시도 → 에러 발생",
		"중복 등록 방지 확인",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case": "3.1.1.2_DID_Duplicate_Detection",
		"did":       string(testDID),
		"uuid":      uuidVal.String(),
		"sage_functions_used": []string{
			"GenerateDID(chain, identifier)",
			"RegisterAgent(ctx, chain, req) - 첫번째",
			"ResolveAgent(ctx, did)",
			"RegisterAgent(ctx, chain, req) - 두번째 (중복)",
		},
		"first_registration": map[string]interface{}{
			"success":          result1 != nil,
			"transaction_hash": result1.TransactionHash,
			"block_number":     result1.BlockNumber,
		},
		"duplicate_registration": map[string]interface{}{
			"error_occurred": err != nil,
			"error_message":  fmt.Sprintf("%v", err),
		},
	}
	helpers.SaveTestData(t, "did/did_duplicate_detection.json", testData)
}
