package handshake_test

import (
	"context"
	"crypto/ecdh"
	"encoding/base64"
	"encoding/json"
	"net"
	"testing"

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
	"github.com/sage-x-project/sage/handshake/session"
	sessioninit "github.com/sage-x-project/sage/internal"
	"github.com/stretchr/testify/require"
	"github.com/test-go/testify/assert"
	"github.com/test-go/testify/mock"
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

func setupTest(t *testing.T) (*handshake.Client, *handshake.Server, chan *a2a.SendMessageRequest, sagecrypto.KeyPair, sagecrypto.KeyPair, *session.Manager, *mockResolver ) {
	aliceKeyPair, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	bobKeyPair, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)

	// Outbound capture channel (server pushes Response here)
	outboundCh := make(chan *a2a.SendMessageRequest, 8)
	outboundMock := &mockOutboundClient{ch: outboundCh}

	// Events (upper-layer) – no-op for this test
	srvSessManager := session.NewManager()
	events := sessioninit.NewCreator(srvSessManager)
	

	// Resolver returns the sender's pubkey (Alice's)
	// resolve := func(ctx context.Context, msg *a2a.Message, meta *structpb.Struct) (crypto.PublicKey, error) {
	// 	return aliceKeyPair.PublicKey(), nil
	// }

	ethResolver := new(mockResolver)
	multiResolver := sagedid.NewMultiChainResolver()
	multiResolver.AddResolver(sagedid.ChainEthereum, ethResolver)
	
	// Pass nil to use default session config (1h, 10m, 10k msgs)
	hs := handshake.NewServer(bobKeyPair, events, outboundMock, multiResolver, nil)

	// Start SUT gRPC server
	_, lis := startSUT(t, hs)

	// Dial SUT as a2a client (Alice → Bob)
	conn := dialBuf(t, lis)
	alice := handshake.NewClient(conn, aliceKeyPair)

	return alice, hs, outboundCh, aliceKeyPair, bobKeyPair, srvSessManager, ethResolver
}

func TestHandshake(t *testing.T) {
	alice, _, outboundCh, aliceKeyPair, bobKeyPair, srvSessManager, ethResolver := setupTest(t)

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
	t.Run("Alice → Bob: Invitation", func(t *testing.T) {
		aliceDID := did.AgentDID("did:sage:ethereum:agent001")
		aliceMeta := &did.AgentMetadata{
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

		// Client sends Invitation to server (SUT)
		resp, err := alice.Invitation(ctx, *invMsg, string(aliceMeta.DID))
		require.NoError(t, err)

		// Server ack payload checks
		payload := resp.GetTask()
		require.NotNil(t, payload)
		assert.Equal(t, contextId, payload.GetContextId())
		assert.NotEmpty(t, payload.GetId())

	})

	t.Run("Alice → Bob: Request (encrypted with Bob pubkey)", func(t *testing.T) {
		aliceDID := did.AgentDID("did:sage:ethereum:agent001")
		aliceMeta := &did.AgentMetadata{
			DID:       aliceDID,
			Name:      "Active Agent",
			IsActive:  true,
			PublicKey: aliceKeyPair.PublicKey(),
		}
		ethResolver.On("Resolve", mock.Anything, aliceDID).Return(aliceMeta, nil).Once()
		reqMsg := &handshake.RequestMessage{
			BaseMessage: message.BaseMessage{
				ContextID: 	contextId,    
			},
			EphemeralPubKey: json.RawMessage(alicePubKeyJWK),
		}
		_, err := alice.Request(ctx, *reqMsg, bobKeyPair.PublicKey(), string(aliceMeta.DID))
		require.NoError(t, err)

		out := <-outboundCh
		require.NotNil(t, out)
		assert.Equal(t, handshake.GenerateTaskID(handshake.Response), out.Request.GetTaskId())

		// Decrypt Response envelope (b64) with Alice private key
		sb := out.Request.Content[0].GetData().GetData()
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

		exported, err := importer.ImportPublic([]byte(res.EphemeralPubKey), sagecrypto.KeyFormatJWK)
		peerEph, _ := exported.(*ecdh.PublicKey)

		// Initiator (Alice) derives shared secret using its X25519 private and peer (server) ephemeral pub from Response
		s2, err := aliceEphemeralKeyPair.(*keys.X25519KeyPair).DeriveSharedSecret(peerEph.Bytes())
		require.NoError(t, err)
		require.NotEmpty(t, s2)
		aliceRaw := aliceEphemeralKeyPair.(*keys.X25519KeyPair).PublicBytesKey()
		_, sid, ok, err := session.NewManager().EnsureSessionWithParams(
						session.Params{
							ContextID: contextId,
							SelfEph: aliceRaw,
							PeerEph: peerEph.Bytes(),
							Label: "a2a/handshake v1",
							SharedSecret:s2,
						}, nil)
		require.NoError(t, err)
		assert.False(t, ok)
		sessionID = sid
	})

	t.Run("Alice → Bob: Complete", func(t *testing.T) {
		aliceDID := did.AgentDID("did:sage:ethereum:agent001")
		aliceMeta := &did.AgentMetadata{
			DID:       aliceDID,
			Name:      "Active Agent",
			IsActive:  true,
			PublicKey: aliceKeyPair.PublicKey(),
		}

		ethResolver.On("Resolve", mock.Anything, aliceDID).Return(aliceMeta, nil).Once()
		comMsg := &handshake.CompleteMessage{
			BaseMessage: message.BaseMessage{
				ContextID: contextId,
			},
		}
		// Client calls Complete RPC (server will ack)
		_, err := alice.Complete(ctx, *comMsg, string(aliceMeta.DID))
		require.NoError(t, err)

		out := <-outboundCh
		require.NotNil(t, out)
		assert.Equal(t, handshake.GenerateTaskID(handshake.Response), out.Request.GetTaskId())

		// Decrypt Response envelope (b64) with Alice private key
		sb := out.Request.Content[0].GetData().GetData()
		require.NotNil(t, sb)
		b64 := sb.Fields["b64"].GetStringValue()
		packet, err := base64.RawURLEncoding.DecodeString(b64)
		require.NoError(t, err)

		decBytes, err := keys.DecryptWithEd25519Peer(aliceKeyPair.PrivateKey(), packet)
		require.NoError(t, err)

		var res handshake.ResponseMessage
		require.NoError(t, fromBytes(decBytes, &res))
		assert.True(t, res.Ack)
		
		
		session, ok := srvSessManager.GetByKeyID(res.KeyID)
		assert.True(t, ok)
		
		// If the shared secret is the same, the session ID will also be the same.
		assert.Equal(t, sessionID, session.GetID(), "shared secrets should match across phases")
	})
}

func fromBytes(data []byte, v any) error {
    return json.Unmarshal(data, v)
}