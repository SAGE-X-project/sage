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
	"testing"
	"time"

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
	ctx := context.Background()
	
	// Generate test keypair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	
	mockDIDManager := new(MockDIDManager)
	service := NewVerificationService(mockDIDManager)
	
	t.Run("VerifyAgentMessage with active agent", func(t *testing.T) {
		agentDID := "did:sage:ethereum:agent001"
		
		// Create message
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
		
		// Sign the message
		verifier := rfc9421.NewVerifier()
		signatureBase := verifier.ConstructSignatureBase(message)
		message.Signature = ed25519.Sign(privateKey, []byte(signatureBase))
		
		// Mock agent metadata
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
		
		mockDIDManager.On("ResolveAgent", ctx, did.AgentDID(agentDID)).Return(agentMetadata, nil).Once()
		
		opts := rfc9421.DefaultVerificationOptions()
		result, err := service.VerifyAgentMessage(ctx, message, opts)
		
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Error)
		assert.Equal(t, agentDID, result.AgentID)
		assert.Equal(t, "Test Agent", result.AgentName)
		assert.Equal(t, "0x1234567890abcdef", result.AgentOwner)
		assert.NotNil(t, result.Capabilities)
		
		mockDIDManager.AssertExpectations(t)
	})
	
	t.Run("VerifyAgentMessage with inactive agent", func(t *testing.T) {
		agentDID := "did:sage:ethereum:agent002"
		
		message := &rfc9421.Message{
			AgentDID:  agentDID,
			MessageID: "msg-002",
			Timestamp: time.Now(),
		}
		
		// Mock inactive agent
		agentMetadata := &did.AgentMetadata{
			DID:       did.AgentDID(agentDID),
			Name:      "Inactive Agent",
			IsActive:  false,
			PublicKey: publicKey,
		}
		
		mockDIDManager.On("ResolveAgent", ctx, did.AgentDID(agentDID)).Return(agentMetadata, nil).Once()
		
		opts := &rfc9421.VerificationOptions{
			RequireActiveAgent: true,
		}
		
		result, err := service.VerifyAgentMessage(ctx, message, opts)
		
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Equal(t, "agent is deactivated", result.Error)
		
		mockDIDManager.AssertExpectations(t)
	})
	
	t.Run("VerifyMessageFromHeaders", func(t *testing.T) {
		agentDID := "did:sage:ethereum:agent003"
		
		headers := map[string]string{
			"X-Agent-DID":           agentDID,
			"X-Message-ID":          "msg-003",
			"X-Timestamp":           time.Now().Format(time.RFC3339),
			"X-Nonce":               "nonce123",
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
