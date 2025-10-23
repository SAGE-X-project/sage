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
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// MockDIDManager is a mock implementation of DID Manager
type MockDIDManager struct {
	mock.Mock
}

func (m *MockDIDManager) Configure(chain did.Chain, config *did.RegistryConfig) error {
	args := m.Called(chain, config)
	return args.Error(0)
}

func (m *MockDIDManager) RegisterAgent(ctx context.Context, chain did.Chain, req *did.RegistrationRequest) (*did.RegistrationResult, error) {
	args := m.Called(ctx, chain, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*did.RegistrationResult), args.Error(1)
}

func (m *MockDIDManager) ResolveAgent(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error) {
	args := m.Called(ctx, agentDID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*did.AgentMetadata), args.Error(1)
}

func (m *MockDIDManager) ResolvePublicKey(ctx context.Context, did did.AgentDID) (interface{}, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
}

func (m *MockDIDManager) UpdateAgent(ctx context.Context, did did.AgentDID, updates map[string]interface{}, keyPair interface{}) error {
	args := m.Called(ctx, did, updates, keyPair)
	return args.Error(0)
}

func (m *MockDIDManager) DeactivateAgent(ctx context.Context, did did.AgentDID, keyPair interface{}) error {
	args := m.Called(ctx, did, keyPair)
	return args.Error(0)
}

func (m *MockDIDManager) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*did.AgentMetadata, error) {
	args := m.Called(ctx, ownerAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*did.AgentMetadata), args.Error(1)
}

func (m *MockDIDManager) SearchAgents(ctx context.Context, criteria did.SearchCriteria) ([]*did.AgentMetadata, error) {
	args := m.Called(ctx, criteria)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*did.AgentMetadata), args.Error(1)
}

func (m *MockDIDManager) GetRegistrationStatus(ctx context.Context, chain did.Chain, txHash string) (*did.RegistrationResult, error) {
	args := m.Called(ctx, chain, txHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*did.RegistrationResult), args.Error(1)
}

func (m *MockDIDManager) GetSupportedChains() []did.Chain {
	args := m.Called()
	return args.Get(0).([]did.Chain)
}

func (m *MockDIDManager) IsChainConfigured(chain did.Chain) bool {
	args := m.Called(chain)
	return args.Bool(0)
}

func TestVerificationService(t *testing.T) {
	// Specification Requirement: Agent message verification with RFC9421
	helpers.LogTestSection(t, "11.1.1", "Verification Service - Active Agent")

	ctx := context.Background()

	// Specification Requirement: Generate Ed25519 key pair for signing and verification
	helpers.LogDetail(t, "Generating Ed25519 key pair for test")
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Key pair generated")
	helpers.LogDetail(t, "  Public key (hex): %s", hex.EncodeToString(publicKey))
	helpers.LogDetail(t, "  Private key size: %d bytes", len(privateKey))

	mockDIDManager := new(MockDIDManager)
	service := NewVerificationService(mockDIDManager)
	helpers.LogDetail(t, "Verification service initialized with mock DID manager")

	t.Run("VerifyAgentMessage_with_active_agent", func(t *testing.T) {
		// Specification Requirement: Verify message from active agent with valid signature
		agentDID := "did:sage:ethereum:agent001"
		helpers.LogDetail(t, "Agent DID: %s", agentDID)

		// Specification Requirement: Create RFC9421 message with metadata
		helpers.LogDetail(t, "Creating RFC9421 message")
		message := &rfc9421.Message{
			AgentDID:     agentDID,
			MessageID:    "msg-001",
			Timestamp:    time.Now(),
			Nonce:        "nonce123",
			Body:         []byte("test message"),
			Algorithm:    string(rfc9421.AlgorithmEdDSA),
			SignedFields: []string{"body"},
			Metadata: map[string]interface{}{
				"endpoint": "https://api.example.com",
				"name":     "Test Agent",
			},
		}
		helpers.LogDetail(t, "  Message ID: %s", message.MessageID)
		helpers.LogDetail(t, "  Nonce: %s", message.Nonce)
		helpers.LogDetail(t, "  Body: %s", string(message.Body))
		helpers.LogDetail(t, "  Algorithm: %s", message.Algorithm)

		// Specification Requirement: Sign message using Ed25519 private key
		helpers.LogDetail(t, "Signing message with Ed25519")
		verifier := rfc9421.NewVerifier()
		signatureBase := verifier.ConstructSignatureBase(message)
		message.Signature = ed25519.Sign(privateKey, []byte(signatureBase))
		helpers.LogSuccess(t, "Message signed")
		helpers.LogDetail(t, "  Signature size: %d bytes", len(message.Signature))
		helpers.LogDetail(t, "  Signature (hex): %s", hex.EncodeToString(message.Signature))

		// Specification Requirement: Mock agent metadata from DID registry
		agentMetadata := &did.AgentMetadata{
			DID:       did.AgentDID(agentDID),
			Name:      "Test Agent",
			IsActive:  true,
			PublicKey: publicKey,
			Endpoint:  "https://api.example.com",
			Owner:     "0x1234567890abcdef",
			Capabilities: map[string]interface{}{
				"chat": true,
				"code": true,
			},
		}
		helpers.LogDetail(t, "Agent metadata configured:")
		helpers.LogDetail(t, "  Name: %s", agentMetadata.Name)
		helpers.LogDetail(t, "  Active: %v", agentMetadata.IsActive)
		helpers.LogDetail(t, "  Owner: %s", agentMetadata.Owner)
		helpers.LogDetail(t, "  Endpoint: %s", agentMetadata.Endpoint)

		mockDIDManager.On("ResolveAgent", ctx, did.AgentDID(agentDID)).Return(agentMetadata, nil).Once()

		// Specification Requirement: Verify agent message with default options
		helpers.LogDetail(t, "Verifying agent message")
		opts := rfc9421.DefaultVerificationOptions()
		result, err := service.VerifyAgentMessage(ctx, message, opts)

		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Error)
		assert.Equal(t, agentDID, result.AgentID)
		assert.Equal(t, "Test Agent", result.AgentName)
		assert.Equal(t, "0x1234567890abcdef", result.AgentOwner)
		assert.NotNil(t, result.Capabilities)
		helpers.LogSuccess(t, "Message verification succeeded")
		helpers.LogDetail(t, "  Valid: %v", result.Valid)
		helpers.LogDetail(t, "  Agent ID: %s", result.AgentID)
		helpers.LogDetail(t, "  Agent name: %s", result.AgentName)
		helpers.LogDetail(t, "  Agent owner: %s", result.AgentOwner)

		mockDIDManager.AssertExpectations(t)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Ed25519 key pair generated successfully",
			"Verification service initialized",
			"RFC9421 message created with metadata",
			"Message signed with Ed25519 private key",
			"Agent metadata configured in mock",
			"DID resolver called for agent lookup",
			"Message verification succeeded",
			"Agent ID, name, and owner verified",
			"Capabilities returned correctly",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case": "11.1.1_Verification_Active_Agent",
			"agent_did": agentDID,
			"message": map[string]interface{}{
				"id":             message.MessageID,
				"nonce":          message.Nonce,
				"body":           string(message.Body),
				"algorithm":      message.Algorithm,
				"signature_size": len(message.Signature),
			},
			"agent_metadata": map[string]interface{}{
				"name":     agentMetadata.Name,
				"is_active": agentMetadata.IsActive,
				"owner":    agentMetadata.Owner,
				"endpoint": agentMetadata.Endpoint,
			},
			"verification_result": map[string]interface{}{
				"valid":      result.Valid,
				"error":      result.Error,
				"agent_id":   result.AgentID,
				"agent_name": result.AgentName,
				"agent_owner": result.AgentOwner,
			},
		}
		helpers.SaveTestData(t, "verification/active_agent.json", testData)
	})

	t.Run("VerifyAgentMessage_with_inactive_agent", func(t *testing.T) {
		// Specification Requirement: Reject messages from inactive/deactivated agents
		helpers.LogTestSection(t, "11.1.2", "Verification Service - Inactive Agent")

		agentDID := "did:sage:ethereum:agent002"
		helpers.LogDetail(t, "Testing inactive agent: %s", agentDID)

		// Specification Requirement: Create message from inactive agent
		message := &rfc9421.Message{
			AgentDID:  agentDID,
			MessageID: "msg-002",
			Timestamp: time.Now(),
		}
		helpers.LogDetail(t, "Message ID: %s", message.MessageID)

		// Specification Requirement: Mock inactive agent metadata
		agentMetadata := &did.AgentMetadata{
			DID:       did.AgentDID(agentDID),
			Name:      "Inactive Agent",
			IsActive:  false,
			PublicKey: publicKey,
		}
		helpers.LogDetail(t, "Agent metadata:")
		helpers.LogDetail(t, "  Name: %s", agentMetadata.Name)
		helpers.LogDetail(t, "  Active: %v", agentMetadata.IsActive)

		mockDIDManager.On("ResolveAgent", ctx, did.AgentDID(agentDID)).Return(agentMetadata, nil).Once()

		// Specification Requirement: Verification options requiring active agent
		opts := &rfc9421.VerificationOptions{
			RequireActiveAgent: true,
		}
		helpers.LogDetail(t, "Verification options: RequireActiveAgent=true")

		// Specification Requirement: Verification should fail for inactive agent
		helpers.LogDetail(t, "Attempting to verify message from inactive agent")
		result, err := service.VerifyAgentMessage(ctx, message, opts)

		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "agent is deactivated", result.Error)
		helpers.LogSuccess(t, "Verification correctly failed for inactive agent")
		helpers.LogDetail(t, "  Valid: %v", result.Valid)
		helpers.LogDetail(t, "  Error: %s", result.Error)

		mockDIDManager.AssertExpectations(t)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Message created from inactive agent",
			"Inactive agent metadata configured",
			"Verification options set to require active agent",
			"Verification failed as expected",
			"Error message: agent is deactivated",
			"Inactive agent protection working correctly",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case": "11.1.2_Verification_Inactive_Agent",
			"agent_did": agentDID,
			"message": map[string]interface{}{
				"id": message.MessageID,
			},
			"agent_metadata": map[string]interface{}{
				"name":     agentMetadata.Name,
				"is_active": agentMetadata.IsActive,
			},
			"verification_result": map[string]interface{}{
				"valid": result.Valid,
				"error": result.Error,
			},
		}
		helpers.SaveTestData(t, "verification/inactive_agent.json", testData)
	})

	t.Run("VerifyMessageFromHeaders", func(t *testing.T) {
		agentDID := "did:sage:ethereum:agent003"

		headers := map[string]string{
			"X-Agent-DID":           agentDID,
			"X-Message-ID":          "msg-003",
			"X-Timestamp":           time.Now().Format(time.RFC3339),
			"X-Nonce":               "nonce456", // Use unique nonce for this test
			"X-Signature-Algorithm": "EdDSA",
			"X-Signed-Fields":       "body",
			"X-Metadata-Endpoint":   "https://api.example.com",
			"X-Metadata-Name":       "Test Agent",
		}

		body := []byte("test message")

		// Create expected message and sign it
		message, _ := rfc9421.ParseMessageFromHeaders(headers, body)
		message.Metadata = map[string]interface{}{
			"endpoint": "https://api.example.com",
			"name":     "Test Agent",
		}
		verifier := rfc9421.NewVerifier()
		signatureBase := verifier.ConstructSignatureBase(message)
		signature := ed25519.Sign(privateKey, []byte(signatureBase))

		// Mock agent metadata
		agentMetadata := &did.AgentMetadata{
			DID:       did.AgentDID(agentDID),
			Name:      "Test Agent",
			IsActive:  true,
			PublicKey: publicKey,
			Endpoint:  "https://api.example.com",
		}

		mockDIDManager.On("ResolveAgent", ctx, did.AgentDID(agentDID)).Return(agentMetadata, nil).Once()

		result, err := service.VerifyMessageFromHeaders(ctx, headers, body, signature)

		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Error)

		mockDIDManager.AssertExpectations(t)
	})

	t.Run("QuickVerify", func(t *testing.T) {
		agentDID := "did:sage:solana:agent004"
		message := []byte("test message")

		// Create minimal RFC9421 message for signing
		msg := &rfc9421.Message{
			Body:         message,
			Algorithm:    string(rfc9421.AlgorithmEdDSA),
			SignedFields: []string{"body"},
		}

		verifier := rfc9421.NewVerifier()
		signatureBase := verifier.ConstructSignatureBase(msg)
		signature := ed25519.Sign(privateKey, []byte(signatureBase))

		mockDIDManager.On("ResolvePublicKey", ctx, did.AgentDID(agentDID)).Return(publicKey, nil).Once()

		err := service.QuickVerify(ctx, agentDID, message, signature)
		assert.NoError(t, err)

		mockDIDManager.AssertExpectations(t)
	})
}
