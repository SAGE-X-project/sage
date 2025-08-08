package handshake

import (
	"context"
	"encoding/base64"
	"net"
	"testing"

	a2a "github.com/a2aproject/a2a/grpc"
	"github.com/google/uuid"
	"github.com/sage-x-project/sage/core/message"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/did"
	"github.com/stretchr/testify/require"
	"github.com/test-go/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
)

const bufSize = 1024 * 1024

func init() {
    resolver.SetDefaultScheme("passthrough")
}

type mockRecvService struct {
    a2a.UnimplementedA2AServiceServer
    recvCh chan *a2a.SendMessageRequest
}

func (s *mockRecvService) SendMessage(ctx context.Context, req *a2a.SendMessageRequest) (*a2a.SendMessageResponse, error) {
    s.recvCh <- req
    return &a2a.SendMessageResponse{}, nil
}

func setupTest(t *testing.T) (*Client, *Server, chan *a2a.SendMessageRequest, sagecrypto.KeyPair, sagecrypto.KeyPair) {
	aliceKeyPair, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	bobKeyPair, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)

	lis := bufconn.Listen(bufSize)
    srv := grpc.NewServer()
    recvCh := make(chan *a2a.SendMessageRequest, 1)
	bob := NewServer(&mockRecvService{recvCh: recvCh}, bobKeyPair)
	a2a.RegisterA2AServiceServer(srv, bob)
	
	go srv.Serve(lis)
	t.Cleanup(func() {
        srv.Stop()
        lis.Close()
    })

	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
        return lis.Dial()
    }
	conn, err := grpc.NewClient(
        lis.Addr().String(),
        grpc.WithContextDialer(dialer),
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	alice := NewClient(conn, aliceKeyPair)
	return alice, bob, recvCh, aliceKeyPair, bobKeyPair
}

func TestHandshake(t *testing.T) {
	alice, bob, recvCh, aliceKeyPair, bobKeyPair := setupTest(t)
	
	contextId := contextIDPrefix + uuid.NewString()
	sessionId := sessionIDPrefix + uuid.NewString()

	aliceEphemeralKeyPair, err := keys.GenerateX25519KeyPair()
	bobEphemeralKeyPair, err := keys.GenerateX25519KeyPair()

	var s1, s2 []byte

	t.Run("Alice sends invitation to Bob", func(t *testing.T) {
		
		// ------------ client ------------
		ctx := context.Background()
		aliceDID :=  did.AgentDID("did:sage:ethereum:agent001")

		invMsg := &InvitationMessage{
			BaseMessage: message.BaseMessage{
				ContextID: contextId,
				SessionID: sessionId,
				DID: string(aliceDID),
			},
		}

		// handshake Invitation
		_, err := alice.Invitation(ctx, *invMsg)
		require.NoError(t, err)

		// ------------- A2A Server ------------
		// bob receive messabe
		msg := <-recvCh
		assert.Equal(t, invMsg.ContextID, msg.Request.ContextId)
		sb := msg.Request.Content[0].GetData().GetData()
		
		var recvMsg InvitationMessage
		require.NoError(t, fromStructPB(sb, &recvMsg))
		assert.Equal(t, invMsg.SessionID, recvMsg.SessionID)
		assert.Equal(t, invMsg.DID, recvMsg.DID)

		// verify sigatrue
		sigField, ok := msg.Metadata.Fields["signature"]
		require.True(t, ok)
		sigB64 := sigField.GetStringValue()
		sigBytes, err := base64.RawURLEncoding.DecodeString(sigB64)
		require.NoError(t, err)

		bytes, err := proto.MarshalOptions{Deterministic: true,}.Marshal(msg.Request)
		require.NoError(t, err)

		// verify against the raw proto bytes
		err = aliceKeyPair.Verify(bytes, sigBytes)
		require.NoError(t, err)
	})
	
	t.Run("Alice sends encrypted request to Bob", func(t *testing.T) {
		ctx := context.Background()

		// ------------ client ------------
		aliceDID :=  did.AgentDID("did:sage:ethereum:agent001")
		
		require.NoError(t, err)

		reqMsg := &RequestMessage{
			BaseMessage: message.BaseMessage{
				ContextID: contextId,
				SessionID: sessionId,
				EphemeralPubKey: aliceEphemeralKeyPair.(*keys.X25519KeyPair).PublicBytesKey(),
				DID: string(aliceDID),
			},
			Session: Session{
				ID: sessionId,
			},
		}

		// handshake Request
		// Encrypt the request with Bob’s key and send it
		_, err = alice.Request(ctx, *reqMsg, bobKeyPair.PublicKey())
		require.NoError(t, err)

		// ------------- A2A Server ------------
		// bob receive messabe
		msg := <-recvCh
		assert.Equal(t, reqMsg.ContextID, msg.Request.ContextId)
		sb := msg.Request.Content[0].GetData().GetData()

		// decrypt the message
		b64 := sb.Fields["packet"].GetStringValue()
		packet, err := base64.RawURLEncoding.DecodeString(b64)
		require.NoError(t, err)

		decMsg, err := keys.DecryptWithEd25519Peer(bobKeyPair.PrivateKey(), packet)
		require.NoError(t, err)

		var recvMsg RequestMessage
		require.NoError(t, fromBytes(decMsg, &recvMsg))
		assert.Equal(t, reqMsg.SessionID, recvMsg.Session.ID)
		assert.Equal(t, reqMsg.EphemeralPubKey, recvMsg.EphemeralPubKey)
		assert.Equal(t, reqMsg.DID, recvMsg.DID)

		// verify sigatrue
		sigField, ok := msg.Metadata.Fields["signature"]
		require.True(t, ok)
		sigB64 := sigField.GetStringValue()
		sigBytes, err := base64.RawURLEncoding.DecodeString(sigB64)
		require.NoError(t, err)

		bytes, err := proto.MarshalOptions{Deterministic: true,}.Marshal(msg.Request)
		require.NoError(t, err)

		// verify against the raw proto bytes
		err = aliceKeyPair.Verify(bytes, sigBytes)
		require.NoError(t, err)

		s1, err = bobEphemeralKeyPair.(*keys.X25519KeyPair).DeriveSharedSecret(recvMsg.EphemeralPubKey)
		require.NoError(t, err)
	})

	t.Run("Alice sends encrypted response to alice", func(t *testing.T) {
		ctx := context.Background()

		// ------------ client ------------
		bobID :=  did.AgentDID("did:sage:ethereum:agent001")
		require.NoError(t, err)

		resMsg := &ResponseMessage{
			BaseMessage: message.BaseMessage{
				ContextID: contextId,
				SessionID: sessionId,
				EphemeralPubKey: bobEphemeralKeyPair.(*keys.X25519KeyPair).PublicBytesKey(),
				DID: string(bobID),
			},
			Session: Session{
				ID: sessionId,
			},
		}

		// handshake Response
		// Encrypt the reResponsequest with Alice's key and send it
		_, err = bob.Response(ctx, *resMsg, aliceKeyPair.PublicKey())
		require.NoError(t, err)
		
		// ------------- A2A Server ------------
		// alice receive messabe
		msg := <-recvCh
		assert.Equal(t, resMsg.ContextID, msg.Request.ContextId)
		sb := msg.Request.Content[0].GetData().GetData()

		// decrypt the message
		b64 := sb.Fields["packet"].GetStringValue()
		packet, err := base64.RawURLEncoding.DecodeString(b64)
		require.NoError(t, err)

		decMsg, err := keys.DecryptWithEd25519Peer(aliceKeyPair.PrivateKey(), packet)
		require.NoError(t, err)

		var recvMsg RequestMessage
		require.NoError(t, fromBytes(decMsg, &recvMsg))
		assert.Equal(t, resMsg.SessionID, recvMsg.Session.ID)
		assert.Equal(t, resMsg.EphemeralPubKey, recvMsg.EphemeralPubKey)
		assert.Equal(t, resMsg.DID, recvMsg.DID)

		// verify sigatrue
		sigField, ok := msg.Metadata.Fields["signature"]
		require.True(t, ok)
		sigB64 := sigField.GetStringValue()
		sigBytes, err := base64.RawURLEncoding.DecodeString(sigB64)
		require.NoError(t, err)

		bytes, err := proto.MarshalOptions{Deterministic: true,}.Marshal(msg.Request)
		require.NoError(t, err)

		// verify against the raw proto bytes
		err = bobKeyPair.Verify(bytes, sigBytes)
		require.NoError(t, err)

		s2, err = aliceEphemeralKeyPair.(*keys.X25519KeyPair).DeriveSharedSecret(recvMsg.EphemeralPubKey)
		require.NoError(t, err)
	})

	t.Run("Alice complete handshake", func(t *testing.T) {
		// shared secret
		assert.Equal(t, s1, s2)
		
		// ------------ client ------------
		ctx := context.Background()
		aliceDID :=  did.AgentDID("did:sage:ethereum:agent001")

		comMsg := &CompleteMessage{
			BaseMessage: message.BaseMessage{
				ContextID: contextId,
				SessionID: sessionId,
				DID: string(aliceDID),
			},
		}

		// handshake Complete
		_, err := alice.Complete(ctx, *comMsg)
		require.NoError(t, err)

		// ------------- A2A Server ------------
		// bob receive messabe
		msg := <-recvCh
		assert.Equal(t, comMsg.ContextID, msg.Request.ContextId)
		sb := msg.Request.Content[0].GetData().GetData()

		var recvMsg CompleteMessage
		require.NoError(t, fromStructPB(sb, &recvMsg))
		assert.Equal(t, comMsg.SessionID, recvMsg.SessionID)
		assert.Equal(t, comMsg.DID, recvMsg.DID)

		// verify sigatrue
		sigField, ok := msg.Metadata.Fields["signature"]
		require.True(t, ok)
		sigB64 := sigField.GetStringValue()
		sigBytes, err := base64.RawURLEncoding.DecodeString(sigB64)
		require.NoError(t, err)

		bytes, err := proto.MarshalOptions{Deterministic: true,}.Marshal(msg.Request)
		require.NoError(t, err)

		// verify against the raw proto bytes
		err = aliceKeyPair.Verify(bytes, sigBytes)
		require.NoError(t, err)
	})
	
}