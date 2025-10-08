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
	"crypto/ecdh"
	"encoding/base64"
	"encoding/json"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	a2a "github.com/a2aproject/a2a/grpc"
	"github.com/google/uuid"

	// "github.com/sage-x-project/sage/core/adapter"
	"github.com/sage-x-project/sage/core/message"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/did"
	sagedid "github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/handshake"
	sessioninit "github.com/sage-x-project/sage/internal"
	"github.com/sage-x-project/sage/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func init() {
    resolver.SetDefaultScheme("passthrough")
}

// mockOutboundClient captures SendMessage calls that the server pushes to the peer.
type mockOutboundClient struct {
    a2a.A2AServiceClient
    ch chan *a2a.SendMessageRequest
}

func (m *mockOutboundClient) SendMessage(ctx context.Context, in *a2a.SendMessageRequest, _ ...grpc.CallOption) (*a2a.SendMessageResponse, error) {
    m.ch <- in
    return &a2a.SendMessageResponse{}, nil
}

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

// start a bufconn grpc server and register our handshake.Server (system under test)
func startSUT(t *testing.T, hs a2a.A2AServiceServer) (*grpc.Server, *bufconn.Listener) {
    lis := bufconn.Listen(bufSize)
    srv := grpc.NewServer()
    a2a.RegisterA2AServiceServer(srv, hs)
    go func() { _ = srv.Serve(lis) }()
    t.Cleanup(func() {
        srv.Stop()
        lis.Close()
    })
    return srv, lis
}

// dial bufconn as a client
func dialBuf(t *testing.T, lis *bufconn.Listener) *grpc.ClientConn {
    t.Helper()
    conn, err := grpc.NewClient(
        "bufnet",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
            return lis.Dial()
        }),
    )
    require.NoError(t, err)
    return conn
}


func setupTest(t *testing.T, cleanupInterval time.Duration) (*handshake.Client, *handshake.Server, sagecrypto.KeyPair, sagecrypto.KeyPair, *session.Manager, *mockResolver) {
    aliceKeyPair, err := keys.GenerateEd25519KeyPair()
    require.NoError(t, err)
    bobKeyPair, err := keys.GenerateEd25519KeyPair()
    require.NoError(t, err)

    // Events (upper-layer) – no-op for this test
    srvSessManager := session.NewManager()
    events := sessioninit.NewCreator(srvSessManager)

    ethResolver := new(mockResolver)
    multiResolver := sagedid.NewMultiChainResolver()
    multiResolver.AddResolver(sagedid.ChainEthereum, ethResolver)

    // Pass nil to use default session config (1h, 10m, 10k msgs)
    hs := handshake.NewServer(bobKeyPair, events, multiResolver, nil, cleanupInterval)
    t.Cleanup(func() {
        handshake.StopCleanupLoop(hs)
        ethResolver.AssertExpectations(t)
    })

    // Start SUT gRPC server
    _, lis := startSUT(t, hs)

    // Dial SUT as a2a client (Alice → Bob)
    conn := dialBuf(t, lis)
    alice := handshake.NewClient(conn, aliceKeyPair)

    return alice, hs, aliceKeyPair, bobKeyPair, srvSessManager, ethResolver
}

func TestHandshake(t *testing.T) {
    alice, _, aliceKeyPair, bobKeyPair, srvSessManager, ethResolver := setupTest(t, 0)

    ctx := context.Background()
    contextId := "ctx-" + uuid.NewString()
    exporter := formats.NewJWKExporter()
    importer := formats.NewJWKImporter()

    aliceEphemeralKeyPair, err := keys.GenerateX25519KeyPair()
    require.NoError(t, err)

    // alice's session key
    alicePubKeyJWK, err := exporter.ExportPublic(aliceEphemeralKeyPair, sagecrypto.KeyFormatJWK)
    require.NoError(t, err)
    require.NotEmpty(t, alicePubKeyJWK)

    var sessionID string
    // Alice → Bob: Invitation
    aliceDID := did.AgentDID("did:sage:ethereum:agent001")
    aliceMeta := &did.AgentMetadata{
        DID:       aliceDID,
        Name:      "Active Agent",
        IsActive:  true,
        PublicKey: aliceKeyPair.PublicKey(),
    }
    
    // blockchain call once
    ethResolver.On("Resolve", mock.Anything, aliceDID).Return(aliceMeta, nil).Once()
    invMsg := &handshake.InvitationMessage{
        BaseMessage: message.BaseMessage{
            ContextID: contextId,
        },
    }

    // Client sends Invitation to server (SUT)
    resp, err := alice.Invitation(ctx, *invMsg, string(aliceMeta.DID))
    require.NoError(t, err)

    // Server ack payload checks
    payload := resp.GetTask()
    require.NotNil(t, payload)
    assert.Equal(t, contextId, payload.GetContextId())
    assert.NotEmpty(t, payload.GetId())

    // Alice → Bob: Request
    reqMsg := &handshake.RequestMessage{
        BaseMessage: message.BaseMessage{
            ContextID: contextId,
        },
        EphemeralPubKey: json.RawMessage(alicePubKeyJWK),
    }
    resp, err = alice.Request(ctx, *reqMsg, bobKeyPair.PublicKey(), string(aliceMeta.DID))
    require.NoError(t, err)
    require.NotNil(t, resp)

    msg := resp.GetMsg()
    require.NotNil(t, msg)

    assert.Equal(t, handshake.GenerateTaskID(handshake.Response), msg.GetTaskId())

    // Decrypt Response envelope (b64) with Alice private key
    parts := msg.GetContent()
    require.GreaterOrEqual(t, len(parts), 1)
    dp := parts[0].GetData()
    require.NotNil(t, dp)
    sb := dp.GetData()
    require.NotNil(t, sb)

    b64 := sb.Fields["b64"].GetStringValue()
    packet, err := base64.RawURLEncoding.DecodeString(b64)
    require.NoError(t, err)

    decBytes, err := keys.DecryptWithEd25519Peer(aliceKeyPair.PrivateKey(), packet)
    require.NoError(t, err)

    var res handshake.ResponseMessage
    require.NoError(t, fromBytes(decBytes, &res))
    assert.True(t, res.Ack)
    require.NotNil(t, res.EphemeralPubKey)

    exported, _ := importer.ImportPublic([]byte(res.EphemeralPubKey), sagecrypto.KeyFormatJWK)
    peerEph, _ := exported.(*ecdh.PublicKey)

    // Initiator (Alice) derives shared secret using its X25519 private and peer (server) ephemeral pub from Response
    s2, err := aliceEphemeralKeyPair.(*keys.X25519KeyPair).DeriveSharedSecret(peerEph.Bytes())
    require.NoError(t, err)
    require.NotEmpty(t, s2)
    aliceRaw := aliceEphemeralKeyPair.(*keys.X25519KeyPair).PublicBytesKey()
    _, sid, ok, err := session.NewManager().EnsureSessionWithParams(
        session.Params{
            ContextID:    contextId,
            SelfEph:      aliceRaw,
            PeerEph:      peerEph.Bytes(),
            Label:        "a2a/handshake v1",
            SharedSecret: s2,
        }, nil)
    require.NoError(t, err)
    assert.False(t, ok)
    sessionID = sid

    // Alice → Bob: Complete
    comMsg := &handshake.CompleteMessage{
        BaseMessage: message.BaseMessage{
            ContextID: contextId,
        },
        }
    // Client calls Complete RPC (server will ack)
    resp, err = alice.Complete(ctx, *comMsg, string(aliceMeta.DID))
    require.NoError(t, err)
    require.NotNil(t, resp)

    msg = resp.GetMsg()
    require.NotNil(t, msg)

    assert.Equal(t, handshake.GenerateTaskID(handshake.Response), msg.GetTaskId())

    // Decrypt Response envelope (b64) with Alice private key
    parts = msg.GetContent()
    require.GreaterOrEqual(t, len(parts), 1)
    dp = parts[0].GetData()
    require.NotNil(t, dp)
    sb = dp.GetData()
    require.NotNil(t, sb)

    b64 = sb.Fields["b64"].GetStringValue()
    packet, err = base64.RawURLEncoding.DecodeString(b64)
    require.NoError(t, err)

    decBytes, err = keys.DecryptWithEd25519Peer(aliceKeyPair.PrivateKey(), packet)
    require.NoError(t, err)

    require.NoError(t, fromBytes(decBytes, &res))
    assert.True(t, res.Ack)

    session, ok := srvSessManager.GetByKeyID(res.KeyID)
    assert.True(t, ok)

    // If the shared secret is the same, the session ID will also be the same.
    assert.Equal(t, sessionID, session.GetID(), "shared secrets should match across phases")

}

func TestHandshake_cache(t *testing.T) {
    alice, hs, aliceKeyPair, bobKeyPair, _, ethResolver := setupTest(t, 10*time.Millisecond)

    ctx := context.Background()
    exporter := formats.NewJWKExporter()
    aliceEphemeralKeyPair, err := keys.GenerateX25519KeyPair()
    require.NoError(t, err)
    alicePubKeyJWK, err := exporter.ExportPublic(aliceEphemeralKeyPair, sagecrypto.KeyFormatJWK)
    require.NoError(t, err)
    require.NotEmpty(t, alicePubKeyJWK)

    t.Run("cache clean", func(t *testing.T) {
        cacheCtxID := "ctx-" + uuid.NewString()
        aliceDID := did.AgentDID("did:sage:ethereum:agent-cache-clean")
        aliceMeta := &did.AgentMetadata{
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

        expiredDID := did.AgentDID("did:sage:ethereum:agent-expired")
        expiredMeta := &did.AgentMetadata{
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

        activeDID := did.AgentDID("did:sage:ethereum:agent-active")
        activeMeta := &did.AgentMetadata{
            DID:       activeDID,
            Name:      "Active Agent",
            IsActive:  true,
            PublicKey: aliceKeyPair.PublicKey(),
        }

        ethResolver.On("Resolve", mock.Anything, activeDID).Return(expiredMeta, nil).Once()
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

func fromBytes(data []byte, v any) error {
    return json.Unmarshal(data, v)
}


func TestInvitation_ResolverSingleflight(t *testing.T) {
    alice, hs, aliceKeyPair, bobKeyPair, _, ethResolver := setupTest(t, 0)

    t.Run("dedups concurrent resolve", func(t *testing.T) {
        ctx := context.Background()
        contextId := "ctx-" + uuid.NewString()

        aliceDID := did.AgentDID("did:sage:ethereum:agent-concurrent")
        aliceMeta := &did.AgentMetadata{
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
        aliceDID := did.AgentDID("did:sage:ethereum:agent-cache-fast")
        aliceMeta := &did.AgentMetadata{
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

        aliceDID := did.AgentDID("did:sage:ethereum:agent-rcache")
        aliceMeta := &did.AgentMetadata{
            DID:       aliceDID,
            Name:      "Cache Agent",
            IsActive:  true,
            PublicKey: aliceKeyPair.PublicKey(),
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
