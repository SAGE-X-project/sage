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

package handshake_test

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	sessioninit "github.com/sage-x-project/sage/internal"
	"github.com/sage-x-project/sage/pkg/agent/core/message"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	sagedid "github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/handshake"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/pkg/agent/transport"
)

type mockResolver struct {
	mock.Mock
}

func (m *mockResolver) Resolve(ctx context.Context, did sagedid.AgentDID) (*sagedid.AgentMetadata, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sagedid.AgentMetadata), args.Error(1)
}

func (m *mockResolver) ResolvePublicKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
}

func (m *mockResolver) ResolveKEMKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0), args.Error(1)
}

func (m *mockResolver) VerifyMetadata(ctx context.Context, did sagedid.AgentDID, metadata *sagedid.AgentMetadata) (*sagedid.VerificationResult, error) {
	args := m.Called(ctx, did, metadata)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sagedid.VerificationResult), args.Error(1)
}

func (m *mockResolver) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*sagedid.AgentMetadata, error) {
	args := m.Called(ctx, ownerAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sagedid.AgentMetadata), args.Error(1)
}

func (m *mockResolver) Search(ctx context.Context, criteria sagedid.SearchCriteria) ([]*sagedid.AgentMetadata, error) {
	args := m.Called(ctx, criteria)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*sagedid.AgentMetadata), args.Error(1)
}

// setupTest creates a Client and Server connected via MockTransport
func setupTest(t *testing.T, cleanupInterval time.Duration) (*handshake.Client, *handshake.Server, sagecrypto.KeyPair, sagecrypto.KeyPair, *session.Manager, *mockResolver, *transport.MockTransport) {
	aliceKeyPair, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	bobKeyPair, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)

	// Session manager for server
	srvSessManager := session.NewManager()
	events := sessioninit.NewCreator(srvSessManager)

	ethResolver := new(mockResolver)
	multiResolver := sagedid.NewMultiChainResolver()
	multiResolver.AddResolver(sagedid.ChainEthereum, ethResolver)

	// Create MockTransport
	mockTransport := &transport.MockTransport{}

	// Create Server
	hs := handshake.NewServer(bobKeyPair, events, multiResolver, nil, cleanupInterval, nil)
	t.Cleanup(func() {
		handshake.StopCleanupLoop(hs)
		ethResolver.AssertExpectations(t)
	})

	// Setup MockTransport to route messages to Server.HandleMessage
	mockTransport.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		return hs.HandleMessage(ctx, msg)
	}

	// Create Client with MockTransport
	alice := handshake.NewClient(mockTransport, aliceKeyPair)

	return alice, hs, aliceKeyPair, bobKeyPair, srvSessManager, ethResolver, mockTransport
}

func TestHandshake_Invitation(t *testing.T) {
	// Specification Requirement: Handshake protocol Phase 1 - Invitation
	helpers.LogTestSection(t, "10.1.1", "Handshake Server Invitation Phase")

	alice, hs, aliceKeyPair, _, _, ethResolver, _ := setupTest(t, 0)
	helpers.LogDetail(t, "Test setup complete:")
	helpers.LogDetail(t, "  Client (Alice) initialized")
	helpers.LogDetail(t, "  Server (Bob) initialized with MockTransport")

	ctx := context.Background()
	contextId := "ctx-" + uuid.NewString()
	helpers.LogDetail(t, "Context ID generated: %s", contextId)

	// Specification Requirement: Setup client DID and metadata
	aliceDID := sagedid.AgentDID("did:sage:ethereum:agent001")
	aliceMeta := &sagedid.AgentMetadata{
		DID:       aliceDID,
		Name:      "Active Agent",
		IsActive:  true,
		PublicKey: aliceKeyPair.PublicKey(),
	}
	helpers.LogDetail(t, "Alice metadata:")
	helpers.LogDetail(t, "  DID: %s", aliceDID)
	helpers.LogDetail(t, "  Name: %s", aliceMeta.Name)
	helpers.LogDetail(t, "  Active: %v", aliceMeta.IsActive)
	if pk, ok := aliceMeta.PublicKey.(ed25519.PublicKey); ok {
		helpers.LogDetail(t, "  Public key (hex): %s", hex.EncodeToString(pk))
	}

	// Specification Requirement: Setup resolver mock for DID resolution
	ethResolver.On("Resolve", mock.Anything, aliceDID).Return(aliceMeta, nil).Once()
	helpers.LogDetail(t, "Resolver configured to return Alice's metadata")

	// Specification Requirement: Create invitation message
	invMsg := &handshake.InvitationMessage{
		BaseMessage: message.BaseMessage{
			ContextID: contextId,
		},
	}
	helpers.LogDetail(t, "Invitation message created")

	// Specification Requirement: Client sends invitation to server
	helpers.LogDetail(t, "Sending invitation from Alice to Bob")
	resp, err := alice.Invitation(ctx, *invMsg, string(aliceMeta.DID))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Success)
	helpers.LogSuccess(t, "Invitation sent and acknowledged")
	helpers.LogDetail(t, "  Response success: %v", resp.Success)

	// Specification Requirement: Verify server cached the peer
	helpers.LogDetail(t, "Verifying server cached Alice as peer")
	hasPeer := handshake.HasPeer(hs, contextId)
	require.True(t, hasPeer)
	helpers.LogSuccess(t, "Server successfully cached peer")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"Client and server initialized with MockTransport",
		"Context ID generated for handshake session",
		"Alice DID and metadata configured",
		"Resolver mock configured for DID resolution",
		"Invitation message created with context ID",
		"Invitation sent successfully via MockTransport",
		"Server acknowledged invitation",
		"Server cached Alice as peer for context",
	})

	// Save test data for CLI verification
	testData := map[string]interface{}{
		"test_case": "10.1.1_Handshake_Invitation",
		"context_id": contextId,
		"alice": map[string]interface{}{
			"did": string(aliceDID),
			"name": aliceMeta.Name,
			"is_active": aliceMeta.IsActive,
		},
		"response": map[string]interface{}{
			"success": resp.Success,
		},
		"peer_cached": hasPeer,
		"phase": "invitation",
	}
	helpers.SaveTestData(t, "handshake/server_invitation.json", testData)
}

func TestHandshake_Request(t *testing.T) {
	// Specification Requirement: Handshake protocol Phase 2 - Request with ephemeral key
	helpers.LogTestSection(t, "10.1.2", "Handshake Server Request Phase")

	alice, hs, aliceKeyPair, bobKeyPair, _, ethResolver, _ := setupTest(t, 0)
	helpers.LogDetail(t, "Test setup complete with client and server")

	ctx := context.Background()
	contextId := "ctx-" + uuid.NewString()
	helpers.LogDetail(t, "Context ID generated: %s", contextId)

	aliceDID := sagedid.AgentDID("did:sage:ethereum:agent001")
	aliceMeta := &sagedid.AgentMetadata{
		DID:       aliceDID,
		Name:      "Active Agent",
		IsActive:  true,
		PublicKey: aliceKeyPair.PublicKey(),
	}
	helpers.LogDetail(t, "Alice DID: %s", aliceDID)

	// Specification Requirement: First send invitation to establish peer cache
	ethResolver.On("Resolve", mock.Anything, aliceDID).Return(aliceMeta, nil).Once()
	helpers.LogDetail(t, "Sending invitation (prerequisite for request)")
	invMsg := &handshake.InvitationMessage{
		BaseMessage: message.BaseMessage{ContextID: contextId},
	}
	_, err := alice.Invitation(ctx, *invMsg, string(aliceMeta.DID))
	require.NoError(t, err)
	helpers.LogSuccess(t, "Invitation phase completed")

	// Specification Requirement: Generate ephemeral X25519 key for ECDH
	helpers.LogDetail(t, "Generating Alice's ephemeral X25519 key pair")
	exporter := formats.NewJWKExporter()
	aliceEphemeralKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)
	alicePubKeyJWK, err := exporter.ExportPublic(aliceEphemeralKeyPair, sagecrypto.KeyFormatJWK)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Ephemeral key pair generated")
	helpers.LogDetail(t, "  Ephemeral public key (JWK): %d bytes", len(alicePubKeyJWK))

	// Specification Requirement: Send request with ephemeral public key
	helpers.LogDetail(t, "Creating request message with ephemeral key")
	reqMsg := &handshake.RequestMessage{
		BaseMessage: message.BaseMessage{
			ContextID: contextId,
		},
		EphemeralPubKey: json.RawMessage(alicePubKeyJWK),
	}
	helpers.LogDetail(t, "Sending request from Alice to Bob")
	resp, err := alice.Request(ctx, *reqMsg, bobKeyPair.PublicKey(), string(aliceMeta.DID))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Success)
	helpers.LogSuccess(t, "Request sent and acknowledged")
	helpers.LogDetail(t, "  Response success: %v", resp.Success)

	// Specification Requirement: Verify server created pending state for completion
	helpers.LogDetail(t, "Verifying server created pending state")
	hasPending := handshake.HasPending(hs, contextId)
	require.True(t, hasPending)
	helpers.LogSuccess(t, "Server created pending state for context")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"Client and server initialized",
		"Invitation phase completed successfully",
		"Ephemeral X25519 key pair generated",
		"Ephemeral public key exported to JWK format",
		"Request message created with ephemeral key",
		"Request sent successfully via MockTransport",
		"Server acknowledged request",
		"Server created pending state for context",
	})

	// Save test data for CLI verification
	testData := map[string]interface{}{
		"test_case": "10.1.2_Handshake_Request",
		"context_id": contextId,
		"alice_did": string(aliceDID),
		"ephemeral_key_size": len(alicePubKeyJWK),
		"response": map[string]interface{}{
			"success": resp.Success,
		},
		"pending_created": hasPending,
		"phase": "request",
	}
	helpers.SaveTestData(t, "handshake/server_request.json", testData)
}

func TestHandshake_Complete(t *testing.T) {
	// Specification Requirement: Handshake protocol Phase 3 - Complete and session establishment
	helpers.LogTestSection(t, "10.1.3", "Handshake Server Complete Phase")

	alice, hs, aliceKeyPair, bobKeyPair, _, ethResolver, _ := setupTest(t, 0)
	helpers.LogDetail(t, "Test setup complete with client and server")

	ctx := context.Background()
	contextId := "ctx-" + uuid.NewString()
	helpers.LogDetail(t, "Context ID generated: %s", contextId)

	aliceDID := sagedid.AgentDID("did:sage:ethereum:agent001")
	aliceMeta := &sagedid.AgentMetadata{
		DID:       aliceDID,
		Name:      "Active Agent",
		IsActive:  true,
		PublicKey: aliceKeyPair.PublicKey(),
	}
	helpers.LogDetail(t, "Alice DID: %s", aliceDID)

	// Specification Requirement: First send invitation and request (prerequisites)
	ethResolver.On("Resolve", mock.Anything, aliceDID).Return(aliceMeta, nil).Once()
	helpers.LogDetail(t, "Phase 1: Sending invitation")
	invMsg := &handshake.InvitationMessage{
		BaseMessage: message.BaseMessage{ContextID: contextId},
	}
	_, err := alice.Invitation(ctx, *invMsg, string(aliceMeta.DID))
	require.NoError(t, err)
	helpers.LogSuccess(t, "Invitation phase completed")

	helpers.LogDetail(t, "Phase 2: Generating ephemeral key and sending request")
	exporter := formats.NewJWKExporter()
	aliceEphemeralKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)
	alicePubKeyJWK, err := exporter.ExportPublic(aliceEphemeralKeyPair, sagecrypto.KeyFormatJWK)
	require.NoError(t, err)
	helpers.LogDetail(t, "  Ephemeral key generated and exported to JWK")

	reqMsg := &handshake.RequestMessage{
		BaseMessage:     message.BaseMessage{ContextID: contextId},
		EphemeralPubKey: json.RawMessage(alicePubKeyJWK),
	}
	_, err = alice.Request(ctx, *reqMsg, bobKeyPair.PublicKey(), string(aliceMeta.DID))
	require.NoError(t, err)
	helpers.LogSuccess(t, "Request phase completed")

	// Specification Requirement: Send complete message to finalize handshake
	helpers.LogDetail(t, "Phase 3: Sending complete message")
	comMsg := &handshake.CompleteMessage{
		BaseMessage: message.BaseMessage{
			ContextID: contextId,
		},
	}
	resp, err := alice.Complete(ctx, *comMsg, string(aliceMeta.DID))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Success)
	helpers.LogSuccess(t, "Complete message sent and acknowledged")
	helpers.LogDetail(t, "  Response success: %v", resp.Success)

	// Specification Requirement: Verify session establishment
	// Note: Session is created via events.OnComplete which uses sessionManager
	// Give time for async processing if any
	helpers.LogDetail(t, "Waiting for session creation (async)")
	time.Sleep(10 * time.Millisecond)

	// Specification Requirement: Check that pending state was consumed after completion
	helpers.LogDetail(t, "Verifying pending state consumed")
	hasPending := handshake.HasPending(hs, contextId)
	require.False(t, hasPending, "pending state should be consumed after Complete")
	helpers.LogSuccess(t, "Pending state consumed successfully")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"Client and server initialized",
		"Phase 1 (Invitation) completed",
		"Phase 2 (Request) completed with ephemeral key",
		"Phase 3 (Complete) message sent",
		"Server acknowledged complete message",
		"Pending state consumed after completion",
		"Three-phase handshake completed successfully",
	})

	// Save test data for CLI verification
	testData := map[string]interface{}{
		"test_case": "10.1.3_Handshake_Complete",
		"context_id": contextId,
		"alice_did": string(aliceDID),
		"phases_completed": []string{"invitation", "request", "complete"},
		"response": map[string]interface{}{
			"success": resp.Success,
		},
		"pending_consumed": !hasPending,
		"phase": "complete",
		"handshake_status": "finalized",
	}
	helpers.SaveTestData(t, "handshake/server_complete.json", testData)
}

func TestHandshake_cache(t *testing.T) {
	alice, hs, aliceKeyPair, bobKeyPair, _, ethResolver, _ := setupTest(t, 10*time.Millisecond)

	ctx := context.Background()
	exporter := formats.NewJWKExporter()
	aliceEphemeralKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)
	alicePubKeyJWK, err := exporter.ExportPublic(aliceEphemeralKeyPair, sagecrypto.KeyFormatJWK)
	require.NoError(t, err)
	require.NotEmpty(t, alicePubKeyJWK)

	t.Run("cache clean", func(t *testing.T) {
		cacheCtxID := "ctx-" + uuid.NewString()
		aliceDID := sagedid.AgentDID("did:sage:ethereum:agent-cache-clean")
		aliceMeta := &sagedid.AgentMetadata{
			DID:       aliceDID,
			Name:      "Cache Clean Agent",
			IsActive:  true,
			PublicKey: aliceKeyPair.PublicKey(),
		}

		handshake.OverridePendingTTL(hs, 50*time.Millisecond)

		ethResolver.On("Resolve", mock.Anything, aliceDID).Return(aliceMeta, nil).Once()
		invMsg := &handshake.InvitationMessage{
			BaseMessage: message.BaseMessage{ContextID: cacheCtxID},
		}
		_, err := alice.Invitation(ctx, *invMsg, string(aliceMeta.DID))
		require.NoError(t, err)
		require.True(t, handshake.HasPeer(hs, cacheCtxID))

		require.Eventually(t, func() bool {
			return !handshake.HasPeer(hs, cacheCtxID)
		}, time.Second, 10*time.Millisecond)

		reqMsg := &handshake.RequestMessage{
			BaseMessage:     message.BaseMessage{ContextID: cacheCtxID},
			EphemeralPubKey: json.RawMessage(alicePubKeyJWK),
		}
		_, err = alice.Request(ctx, *reqMsg, bobKeyPair.PublicKey(), string(aliceMeta.DID))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no cached peer")
	})

	t.Run("cache clean retains active peers", func(t *testing.T) {
		expiredCtxID := "ctx-" + uuid.NewString()
		activeCtxID := "ctx-" + uuid.NewString()

		handshake.OverridePendingTTL(hs, time.Hour)

		expiredDID := sagedid.AgentDID("did:sage:ethereum:agent-expired")
		expiredMeta := &sagedid.AgentMetadata{
			DID:       expiredDID,
			Name:      "Expired Agent",
			IsActive:  true,
			PublicKey: aliceKeyPair.PublicKey(),
		}
		ethResolver.On("Resolve", mock.Anything, expiredDID).Return(expiredMeta, nil).Once()
		expiredInv := &handshake.InvitationMessage{
			BaseMessage: message.BaseMessage{ContextID: expiredCtxID},
		}
		_, err := alice.Invitation(ctx, *expiredInv, string(expiredMeta.DID))
		require.NoError(t, err)

		activeDID := sagedid.AgentDID("did:sage:ethereum:agent-active")
		activeMeta := &sagedid.AgentMetadata{
			DID:       activeDID,
			Name:      "Active Agent",
			IsActive:  true,
			PublicKey: aliceKeyPair.PublicKey(),
		}

		ethResolver.On("Resolve", mock.Anything, activeDID).Return(activeMeta, nil).Once()
		activeInv := &handshake.InvitationMessage{
			BaseMessage: message.BaseMessage{ContextID: activeCtxID},
		}
		_, err = alice.Invitation(ctx, *activeInv, string(activeMeta.DID))
		require.NoError(t, err)

		require.True(t, handshake.HasPeer(hs, expiredCtxID))
		require.True(t, handshake.HasPeer(hs, activeCtxID))

		handshake.SetPeerExpiry(hs, expiredCtxID, time.Now().Add(-time.Minute))
		handshake.SetPeerExpiry(hs, activeCtxID, time.Now().Add(time.Hour))

		require.Eventually(t, func() bool {
			return !handshake.HasPeer(hs, expiredCtxID)
		}, time.Second, 10*time.Millisecond)
		time.Sleep(100 * time.Millisecond)
		assert.True(t, handshake.HasPeer(hs, activeCtxID))
	})
}

func TestInvitation_ResolverSingleflight(t *testing.T) {
	alice, hs, aliceKeyPair, bobKeyPair, _, ethResolver, _ := setupTest(t, 0)

	t.Run("dedups concurrent resolve", func(t *testing.T) {
		ctx := context.Background()
		contextId := "ctx-" + uuid.NewString()

		aliceDID := sagedid.AgentDID("did:sage:ethereum:agent-concurrent")
		aliceMeta := &sagedid.AgentMetadata{
			DID:       aliceDID,
			Name:      "Concurrent Agent",
			IsActive:  true,
			PublicKey: aliceKeyPair.PublicKey(),
		}
		var callCount atomic.Int32
		ethResolver.On("Resolve", mock.Anything, aliceDID).Run(func(args mock.Arguments) {
			callCount.Add(1)
		}).Return(aliceMeta, nil)

		invMsg := &handshake.InvitationMessage{
			BaseMessage: message.BaseMessage{ContextID: contextId},
		}

		const N = 10
		var wg sync.WaitGroup
		wg.Add(N)

		errs := make(chan error, N)
		for i := 0; i < N; i++ {
			go func() {
				defer wg.Done()
				_, err := alice.Invitation(ctx, *invMsg, string(aliceDID))
				errs <- err
			}()
		}
		wg.Wait()
		close(errs)

		for err := range errs {
			require.NoError(t, err)
		}

		require.True(t, handshake.HasPeer(hs, contextId), "peer should be cached after invitation(s)")
		require.Equal(t, int32(1), callCount.Load(), "resolver should be called exactly once despite 10 concurrent invitations")
	})

	t.Run("avoids second resolve", func(t *testing.T) {
		ctx := context.Background()
		contextId := "ctx-" + uuid.NewString()
		aliceDID := sagedid.AgentDID("did:sage:ethereum:agent-cache-fast")
		aliceMeta := &sagedid.AgentMetadata{
			DID:       aliceDID,
			Name:      "Cache Fast Agent",
			IsActive:  true,
			PublicKey: aliceKeyPair.PublicKey(),
		}
		ethResolver.On("Resolve", mock.Anything, aliceDID).Return(aliceMeta, nil).Once()

		inv := &handshake.InvitationMessage{
			BaseMessage: message.BaseMessage{ContextID: contextId},
		}

		_, err := alice.Invitation(ctx, *inv, string(aliceDID))
		require.NoError(t, err)

		_, err = alice.Invitation(ctx, *inv, string(aliceDID))
		require.NoError(t, err)

		ethResolver.AssertExpectations(t)
	})

	t.Run("full handshake uses cached peer", func(t *testing.T) {
		ctx := context.Background()
		contextId := "ctx-" + uuid.NewString()

		aliceDID := sagedid.AgentDID("did:sage:ethereum:agent-rcache")
		aliceMeta := &sagedid.AgentMetadata{
			DID:       aliceDID,
			Name:      "Cache Agent",
			IsActive:  true,
			PublicKey: aliceKeyPair, // Store the KeyPair, not the raw PublicKey
		}
		ethResolver.On("Resolve", mock.Anything, aliceDID).Return(aliceMeta, nil).Once()

		inv := &handshake.InvitationMessage{
			BaseMessage: message.BaseMessage{ContextID: contextId},
		}
		_, err := alice.Invitation(ctx, *inv, string(aliceDID))
		require.NoError(t, err)

		exporter := formats.NewJWKExporter()
		eph, err := keys.GenerateX25519KeyPair()
		require.NoError(t, err)
		jwk, err := exporter.ExportPublic(eph, sagecrypto.KeyFormatJWK)
		require.NoError(t, err)

		req := &handshake.RequestMessage{
			BaseMessage:     message.BaseMessage{ContextID: contextId},
			EphemeralPubKey: json.RawMessage(jwk),
		}
		_, err = alice.Request(ctx, *req, bobKeyPair.PublicKey(), string(aliceDID))
		require.NoError(t, err)

		comp := &handshake.CompleteMessage{
			BaseMessage: message.BaseMessage{ContextID: contextId},
		}
		_, err = alice.Complete(ctx, *comp, string(aliceDID))
		require.NoError(t, err)

		ethResolver.AssertExpectations(t)
	})
}
