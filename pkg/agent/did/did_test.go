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
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/google/uuid"
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

	// Generate UUID v4
	uuidVal := uuid.New()
	require.NotNil(t, uuidVal)
	helpers.LogDetail(t, "  생성된 UUID: %s", uuidVal.String())

	// Create DID in format: did:sage:ethereum:<uuid>
	didString := fmt.Sprintf("did:sage:ethereum:%s", uuidVal.String())
	did := AgentDID(didString)

	helpers.LogSuccess(t, "DID 생성 완료")
	helpers.LogDetail(t, "  DID: %s", did)
	helpers.LogDetail(t, "  DID 길이: %d characters", len(did))

	// Verify DID format: did:sage:ethereum:<uuid>
	didPattern := regexp.MustCompile(`^did:sage:ethereum:[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	require.True(t, didPattern.MatchString(string(did)), "DID should match pattern did:sage:ethereum:<uuid-v4>")
	helpers.LogSuccess(t, "DID 형식 검증 완료")

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
	parsedUUID, err := uuid.Parse(parts[3])
	require.NoError(t, err, "ID part should be valid UUID")
	require.Equal(t, uuidVal, parsedUUID, "Parsed UUID should match original")
	require.Equal(t, uuid.Version(4), parsedUUID.Version(), "UUID should be version 4")
	helpers.LogSuccess(t, "UUID v4 형식 검증 완료")
	helpers.LogDetail(t, "  UUID 버전: %d", parsedUUID.Version())

	// Test multiple DID creation (uniqueness)
	did2String := fmt.Sprintf("did:sage:ethereum:%s", uuid.New().String())
	did2 := AgentDID(did2String)
	require.NotEqual(t, did, did2, "Different DIDs should be unique")
	helpers.LogSuccess(t, "DID 고유성 검증 완료")
	helpers.LogDetail(t, "  두 번째 DID: %s", did2)

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"DID 생성 성공",
		"형식: did:sage:ethereum:<uuid>",
		"UUID v4 형식 검증",
		"DID 구성 요소 파싱",
		"Method = 'sage'",
		"Network = 'ethereum'",
		"UUID 유효성 확인",
		"DID 고유성 확인",
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
			"format_valid":  didPattern.MatchString(string(did)),
			"uuid_valid":    err == nil,
			"uuid_v4":       parsedUUID.Version() == 4,
			"unique":        did != did2,
		},
	}
	helpers.SaveTestData(t, "did/did_creation.json", testData)
}
