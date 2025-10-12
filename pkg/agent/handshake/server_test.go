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
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
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
	alice, hs, aliceKeyPair, _, _, ethResolver, _ := setupTest(t, 0)

	ctx := context.Background()
	contextId := "ctx-" + uuid.NewString()

	aliceDID := sagedid.AgentDID("did:sage:ethereum:agent001")
	aliceMeta := &sagedid.AgentMetadata{
		DID:       aliceDID,
		Name:      "Active Agent",
		IsActive:  true,
		PublicKey: aliceKeyPair.PublicKey(),
	}

	ethResolver.On("Resolve", mock.Anything, aliceDID).Return(aliceMeta, nil).Once()

	invMsg := &handshake.InvitationMessage{
		BaseMessage: message.BaseMessage{
			ContextID: contextId,
		},
	}

	// Client sends Invitation to server
	resp, err := alice.Invitation(ctx, *invMsg, string(aliceMeta.DID))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Success)

	// Verify server cached the peer
	require.True(t, handshake.HasPeer(hs, contextId))
}

func TestHandshake_Request(t *testing.T) {
	alice, hs, aliceKeyPair, bobKeyPair, _, ethResolver, _ := setupTest(t, 0)

	ctx := context.Background()
	contextId := "ctx-" + uuid.NewString()

	aliceDID := sagedid.AgentDID("did:sage:ethereum:agent001")
	aliceMeta := &sagedid.AgentMetadata{
		DID:       aliceDID,
		Name:      "Active Agent",
		IsActive:  true,
		PublicKey: aliceKeyPair.PublicKey(),
	}

	// First send invitation
	ethResolver.On("Resolve", mock.Anything, aliceDID).Return(aliceMeta, nil).Once()
	invMsg := &handshake.InvitationMessage{
		BaseMessage: message.BaseMessage{ContextID: contextId},
	}
	_, err := alice.Invitation(ctx, *invMsg, string(aliceMeta.DID))
	require.NoError(t, err)

	// Generate ephemeral key
	exporter := formats.NewJWKExporter()
	aliceEphemeralKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)
	alicePubKeyJWK, err := exporter.ExportPublic(aliceEphemeralKeyPair, sagecrypto.KeyFormatJWK)
	require.NoError(t, err)

	// Send request
	reqMsg := &handshake.RequestMessage{
		BaseMessage: message.BaseMessage{
			ContextID: contextId,
		},
		EphemeralPubKey: json.RawMessage(alicePubKeyJWK),
	}
	resp, err := alice.Request(ctx, *reqMsg, bobKeyPair.PublicKey(), string(aliceMeta.DID))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Success)

	// Verify server created pending state
	require.True(t, handshake.HasPending(hs, contextId))
}

func TestHandshake_Complete(t *testing.T) {
	alice, hs, aliceKeyPair, bobKeyPair, _, ethResolver, _ := setupTest(t, 0)

	ctx := context.Background()
	contextId := "ctx-" + uuid.NewString()

	aliceDID := sagedid.AgentDID("did:sage:ethereum:agent001")
	aliceMeta := &sagedid.AgentMetadata{
		DID:       aliceDID,
		Name:      "Active Agent",
		IsActive:  true,
		PublicKey: aliceKeyPair.PublicKey(),
	}

	// First send invitation and request
	ethResolver.On("Resolve", mock.Anything, aliceDID).Return(aliceMeta, nil).Once()
	invMsg := &handshake.InvitationMessage{
		BaseMessage: message.BaseMessage{ContextID: contextId},
	}
	_, err := alice.Invitation(ctx, *invMsg, string(aliceMeta.DID))
	require.NoError(t, err)

	exporter := formats.NewJWKExporter()
	aliceEphemeralKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)
	alicePubKeyJWK, err := exporter.ExportPublic(aliceEphemeralKeyPair, sagecrypto.KeyFormatJWK)
	require.NoError(t, err)

	reqMsg := &handshake.RequestMessage{
		BaseMessage:     message.BaseMessage{ContextID: contextId},
		EphemeralPubKey: json.RawMessage(alicePubKeyJWK),
	}
	_, err = alice.Request(ctx, *reqMsg, bobKeyPair.PublicKey(), string(aliceMeta.DID))
	require.NoError(t, err)

	// Send complete
	comMsg := &handshake.CompleteMessage{
		BaseMessage: message.BaseMessage{
			ContextID: contextId,
		},
	}
	resp, err := alice.Complete(ctx, *comMsg, string(aliceMeta.DID))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.Success)

	// Verify server created session
	// Note: Session is created via events.OnComplete which uses sessionManager
	// We can verify by checking the session manager
	time.Sleep(10 * time.Millisecond) // Give time for async processing if any

	// Check that pending state was consumed
	require.False(t, handshake.HasPending(hs, contextId), "pending state should be consumed after Complete")
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
		ethResolver.On("ResolvePublicKey", mock.Anything, aliceDID).Run(func(args mock.Arguments) {
			callCount.Add(1)
		}).Return(aliceMeta.PublicKey, nil)

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
