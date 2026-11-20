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

// TestVerificationServiceRefusesMessages: every entry of the service
// verifies the SAGE Message format, which is not part of SAGE
// 0.9.0-draft.1.1 (E03-19) and was forgeable (C1); QuickVerify also chose
// the algorithm from the DID's chain (E01-11). They refuse every input,
// before resolving anything, until removal (W-010, W-079).
func TestVerificationServiceRefusesMessages(t *testing.T) {
	ctx := context.Background()
	resolver := new(MockDIDManager) // no expectations: nothing may be resolved
	service := NewVerificationService(resolver)

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	msg := &rfc9421.Message{
		AgentDID: "did:sage:ethereum:0xa11ce", MessageID: "m-1", Timestamp: time.Now(), Nonce: "nonce-0123456789",
		Body: []byte("hello"), Algorithm: string(rfc9421.AlgorithmEdDSA), KeyID: "k1",
		SignedFields: []string{"agent_did", "message_id", "timestamp", "nonce", "body"},
	}
	msg.Signature = ed25519.Sign(priv, []byte(rfc9421.NewVerifier().ConstructSignatureBase(msg)))

	result, err := service.VerifyAgentMessage(ctx, msg, rfc9421.DefaultVerificationOptions())
	assert.ErrorIs(t, err, rfc9421.ErrMessageFormatRemoved)
	require.NotNil(t, result)
	assert.False(t, result.Valid)

	result, err = service.VerifyMessageFromHeaders(ctx, map[string]string{"X-Agent-DID": msg.AgentDID}, msg.Body, msg.Signature)
	assert.ErrorIs(t, err, rfc9421.ErrMessageFormatRemoved)
	require.NotNil(t, result)
	assert.False(t, result.Valid)

	assert.ErrorIs(t, service.QuickVerify(ctx, msg.AgentDID, msg.Body, ed25519.Sign(priv, msg.Body)), rfc9421.ErrMessageFormatRemoved)

	result, err = service.VerifyAgentMessage(ctx, nil, nil)
	assert.ErrorIs(t, err, rfc9421.ErrMessageFormatRemoved)
	assert.False(t, result.Valid)

	resolver.AssertNotCalled(t, "ResolveAgent", mock.Anything, mock.Anything)
	resolver.AssertNotCalled(t, "ResolvePublicKey", mock.Anything, mock.Anything)
}
