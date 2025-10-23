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

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/hpke"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/pkg/agent/transport"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/require"
	"github.com/test-go/testify/mock"
)

// TestPackageCompiles verifies that the integration test package compiles
func TestPackageCompiles(t *testing.T) {
	t.Log("Integration test package compiled successfully")
}

// mockResolver implements did.Resolver interface
type mockResolver struct{ mock.Mock }

func (m *mockResolver) Resolve(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error) {
	args := m.Called(ctx, agentDID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*did.AgentMetadata), args.Error(1)
}
func (m *mockResolver) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
	args := m.Called(ctx, agentDID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
}
func (m *mockResolver) ResolveKEMKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
	args := m.Called(ctx, agentDID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
}
func (m *mockResolver) VerifyMetadata(ctx context.Context, agentDID did.AgentDID, metadata *did.AgentMetadata) (*did.VerificationResult, error) {
	args := m.Called(ctx, agentDID, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*did.VerificationResult), args.Error(1)
}
func (m *mockResolver) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*did.AgentMetadata, error) {
	args := m.Called(ctx, ownerAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*did.AgentMetadata), args.Error(1)
}
func (m *mockResolver) Search(ctx context.Context, criteria did.SearchCriteria) ([]*did.AgentMetadata, error) {
	args := m.Called(ctx, criteria)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*did.AgentMetadata), args.Error(1)
}

// Test 10.6.7: 멀티 에이전트 시나리오
func TestMultiAgentCommunication(t *testing.T) {
	// Specification Requirement: Multi-agent message exchange
	helpers.LogTestSection(t, "10.6.7", "멀티 에이전트 시나리오 (여러 에이전트 간 메시지 교환)")

	ctx := context.Background()

	// Create three agents: A, B, and C
	agentADID := "did:sage:test:agentA-" + uuid.NewString()
	agentBDID := "did:sage:test:agentB-" + uuid.NewString()
	agentCDID := "did:sage:test:agentC-" + uuid.NewString()

	helpers.LogDetail(t, "Agent A DID: %s", agentADID)
	helpers.LogDetail(t, "Agent B DID: %s", agentBDID)
	helpers.LogDetail(t, "Agent C DID: %s", agentCDID)

	// Generate keys for each agent
	agentASignKey, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	agentAKEMKey, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	agentBSignKey, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	agentBKEMKey, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	agentCSignKey, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	agentCKEMKey, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	helpers.LogSuccess(t, "Generated keys for 3 agents")

	// Setup resolver with all agent metadata
	resolver := new(mockResolver)
	multiResolver := did.NewMultiChainResolver()
	multiResolver.AddResolver(did.ChainEthereum, resolver)

	// Agent metadata
	agentAMeta := &did.AgentMetadata{
		DID:          did.AgentDID(agentADID),
		Name:         "Agent A",
		IsActive:     true,
		PublicKey:    agentASignKey,
		PublicKEMKey: agentAKEMKey.PublicKey(),
	}
	agentBMeta := &did.AgentMetadata{
		DID:          did.AgentDID(agentBDID),
		Name:         "Agent B",
		IsActive:     true,
		PublicKey:    agentBSignKey,
		PublicKEMKey: agentBKEMKey.PublicKey(),
	}
	agentCMeta := &did.AgentMetadata{
		DID:          did.AgentDID(agentCDID),
		Name:         "Agent C",
		IsActive:     true,
		PublicKey:    agentCSignKey,
		PublicKEMKey: agentCKEMKey.PublicKey(),
	}

	resolver.On("Resolve", mock.Anything, did.AgentDID(agentADID)).Return(agentAMeta, nil)
	resolver.On("Resolve", mock.Anything, did.AgentDID(agentBDID)).Return(agentBMeta, nil)
	resolver.On("Resolve", mock.Anything, did.AgentDID(agentCDID)).Return(agentCMeta, nil)

	helpers.LogSuccess(t, "Configured DID resolver for 3 agents")

	// Create session managers for each agent
	sessionA := session.NewManager()
	sessionB := session.NewManager()
	sessionC := session.NewManager()
	defer func() {
		_ = sessionA.Close()
		_ = sessionB.Close()
		_ = sessionC.Close()
	}()

	// Create transport network (agents can send messages to each other)
	transportAB := &transport.MockTransport{}
	transportBC := &transport.MockTransport{}
	transportCA := &transport.MockTransport{}

	// Create HPKE servers for each agent
	serverA := hpke.NewServer(agentASignKey, sessionA, agentADID, multiResolver,
		&hpke.ServerOpts{MaxSkew: 2 * time.Minute, Info: hpke.DefaultInfoBuilder{}, KEM: agentAKEMKey})
	serverB := hpke.NewServer(agentBSignKey, sessionB, agentBDID, multiResolver,
		&hpke.ServerOpts{MaxSkew: 2 * time.Minute, Info: hpke.DefaultInfoBuilder{}, KEM: agentBKEMKey})
	serverC := hpke.NewServer(agentCSignKey, sessionC, agentCDID, multiResolver,
		&hpke.ServerOpts{MaxSkew: 2 * time.Minute, Info: hpke.DefaultInfoBuilder{}, KEM: agentCKEMKey})

	// Setup transport routing
	transportAB.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		return serverB.HandleMessage(ctx, msg)
	}
	transportBC.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		return serverC.HandleMessage(ctx, msg)
	}
	transportCA.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		return serverA.HandleMessage(ctx, msg)
	}

	// Create HPKE clients
	clientAtoB := hpke.NewClient(transportAB, multiResolver, agentASignKey, agentADID, hpke.DefaultInfoBuilder{}, sessionA)
	clientBtoC := hpke.NewClient(transportBC, multiResolver, agentBSignKey, agentBDID, hpke.DefaultInfoBuilder{}, sessionB)
	clientCtoA := hpke.NewClient(transportCA, multiResolver, agentCSignKey, agentCDID, hpke.DefaultInfoBuilder{}, sessionC)

	helpers.LogSuccess(t, "Created HPKE servers and clients for 3 agents")

	// Test 1: Agent A → Agent B
	helpers.LogDetail(t, "Testing Agent A → Agent B communication")
	ctxAB := "ctx-AB-" + uuid.NewString()
	kidAB, err := clientAtoB.Initialize(ctx, ctxAB, agentADID, agentBDID)
	require.NoError(t, err)
	require.NotEmpty(t, kidAB)
	helpers.LogSuccess(t, "Agent A → Agent B: Session initialized")

	sessionAB_A, ok := sessionA.GetByKeyID(kidAB)
	require.True(t, ok)
	sessionAB_B, ok := sessionB.GetByKeyID(kidAB)
	require.True(t, ok)

	msgAtoB := []byte("Hello from Agent A to Agent B")
	encAB, err := sessionAB_A.Encrypt(msgAtoB)
	require.NoError(t, err)
	decAB, err := sessionAB_B.Decrypt(encAB)
	require.NoError(t, err)
	require.Equal(t, msgAtoB, decAB)
	helpers.LogSuccess(t, "Agent A → Agent B: Message delivered")
	helpers.LogDetail(t, "  Message: %s", string(decAB))

	// Test 2: Agent B → Agent C
	helpers.LogDetail(t, "Testing Agent B → Agent C communication")
	ctxBC := "ctx-BC-" + uuid.NewString()
	kidBC, err := clientBtoC.Initialize(ctx, ctxBC, agentBDID, agentCDID)
	require.NoError(t, err)
	require.NotEmpty(t, kidBC)
	helpers.LogSuccess(t, "Agent B → Agent C: Session initialized")

	sessionBC_B, ok := sessionB.GetByKeyID(kidBC)
	require.True(t, ok)
	sessionBC_C, ok := sessionC.GetByKeyID(kidBC)
	require.True(t, ok)

	msgBtoC := []byte("Hello from Agent B to Agent C")
	encBC, err := sessionBC_B.Encrypt(msgBtoC)
	require.NoError(t, err)
	decBC, err := sessionBC_C.Decrypt(encBC)
	require.NoError(t, err)
	require.Equal(t, msgBtoC, decBC)
	helpers.LogSuccess(t, "Agent B → Agent C: Message delivered")
	helpers.LogDetail(t, "  Message: %s", string(decBC))

	// Test 3: Agent C → Agent A
	helpers.LogDetail(t, "Testing Agent C → Agent A communication")
	ctxCA := "ctx-CA-" + uuid.NewString()
	kidCA, err := clientCtoA.Initialize(ctx, ctxCA, agentCDID, agentADID)
	require.NoError(t, err)
	require.NotEmpty(t, kidCA)
	helpers.LogSuccess(t, "Agent C → Agent A: Session initialized")

	sessionCA_C, ok := sessionC.GetByKeyID(kidCA)
	require.True(t, ok)
	sessionCA_A, ok := sessionA.GetByKeyID(kidCA)
	require.True(t, ok)

	msgCtoA := []byte("Hello from Agent C to Agent A")
	encCA, err := sessionCA_C.Encrypt(msgCtoA)
	require.NoError(t, err)
	decCA, err := sessionCA_A.Decrypt(encCA)
	require.NoError(t, err)
	require.Equal(t, msgCtoA, decCA)
	helpers.LogSuccess(t, "Agent C → Agent A: Message delivered")
	helpers.LogDetail(t, "  Message: %s", string(decCA))

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"멀티 에이전트 생성 (3개)",
		"에이전트 A → B 메시지 교환",
		"에이전트 B → C 메시지 교환",
		"에이전트 C → A 메시지 교환",
		"HPKE 암호화 통신",
		"서명 검증 (내장)",
		"세션 관리",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case": "10.6.7_Multi_Agent_Communication",
		"agents": map[string]string{
			"agent_A": agentADID,
			"agent_B": agentBDID,
			"agent_C": agentCDID,
		},
		"messages_exchanged": 3,
		"sessions_created":   3,
		"encryption":         "HPKE + ChaCha20-Poly1305",
		"signature":          "Ed25519",
		"communication_flow": []string{
			"A → B",
			"B → C",
			"C → A",
		},
	}
	helpers.SaveTestData(t, "integration/multi_agent_communication.json", testData)
}
