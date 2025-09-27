// filename: hpke_grpc_integration_test.go
package hpke

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	a2a "github.com/a2aproject/a2a/grpc"
	"github.com/google/uuid"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/did"
	sagedid "github.com/sage-x-project/sage/did"
	"github.com/sage-x-project/sage/session"
	"github.com/stretchr/testify/require"
	"github.com/test-go/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

// ----- bufconn helpers -----

const bufSize = 1 << 20

func startBufGRPC(t *testing.T, s a2a.A2AServiceServer) (*grpc.Server, *bufconn.Listener) {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	a2a.RegisterA2AServiceServer(srv, s)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() {
		srv.Stop()
		lis.Close()
	})
	return srv, lis
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

// dial bufconn as a client
func dialBuf(t *testing.T, lis *bufconn.Listener) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.Dial()
		}),
	)
	require.NoError(t, err)
	return conn
}
// ----- test -----

func Test_HPKE_Grpc_EndToEnd(t *testing.T) {
	ctx := context.Background()

	ethResolver := new(mockResolver)
	multiResolver := sagedid.NewMultiChainResolver()
	multiResolver.AddResolver(sagedid.ChainEthereum, ethResolver)

	// DIDs
	const clientDID = "did:sage:test:client"
	const serverDID = "did:sage:test:server"

	// Keys:
	// - Server HPKE(X25519) keypair (수신자: 개인키로 Open)
	serverKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	// - Client 서명키(Ed25519) (메타 서명용)
	clientKeypair, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)

	aliceMeta := &did.AgentMetadata{
		DID:       clientDID,
		Name:      "Active Agent",
		IsActive:  true,
		PublicKey: clientKeypair.PublicKey(),
	}

	bobMeta := &did.AgentMetadata{
		DID:       serverDID,
		Name:      "Active Agent",
		IsActive:  true,
		PublicKey: serverKeyPair.PublicKey(),
	}

	ethResolver.On("Resolve", mock.Anything, did.AgentDID(clientDID)).Return(aliceMeta, nil).Once()
	ethResolver.On("Resolve", mock.Anything, did.AgentDID(serverDID)).Return(bobMeta, nil).Once()

	// Session managers (서버/클라이언트 각각)
	srvMgr := session.NewManager()
	cliMgr := session.NewManager()
	t.Cleanup(func() {
		_ = srvMgr.Close()
		_ = cliMgr.Close()
	})

	// HPKE 서버 인스턴스
	srv := NewServer(serverKeyPair, srvMgr, serverDID, multiResolver, &ServerOpts{
		MaxSkew: 2 * time.Minute,
		Info:    DefaultInfoBuilder{},
	})
	// gRPC bufconn 서버 시작
	_, lis := startBufGRPC(t, srv)

	// gRPC 클라이언트 연결 + A2A stub
	conn := dialBuf(t, lis)

	// HPKE 클라이언트 래퍼
	cli := NewClient(conn, multiResolver, clientKeypair, clientDID, DefaultInfoBuilder{}, cliMgr)

	// ---- Run Complete(): HPKE Init + 세션 생성 + kid 바인딩 ----
	ctxID := "ctx-" + uuid.NewString()

	kid, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.NoError(t, err)
	require.NotEmpty(t, kid, "kid should be returned by server and bound on both ends")

	// 서버/클라 모두 kid → session 찾기
	sSrv, ok := srvMgr.GetByKeyID(kid)
	require.True(t, ok, "server should have bound kid to a session")
	sCli, ok := cliMgr.GetByKeyID(kid)
	require.True(t, ok, "client should have bound kid to a session")

	// ---- 방향 분리 키 검증: C->S, S->C 암복호화 모두 동작 확인 ----
	// C → S
	msg1 := []byte("hello from client")
	ct1, err := sCli.Encrypt(msg1) // outbound(client): c2s
	require.NoError(t, err)
	pt1, err := sSrv.Decrypt(ct1) // inbound(server): c2s
	require.NoError(t, err)
	require.True(t, bytes.Equal(pt1, msg1), "client→server plaintext mismatch")

	// S → C
	msg2 := []byte("hello from server")
	ct2, err := sSrv.Encrypt(msg2) // outbound(server): s2c
	require.NoError(t, err)
	pt2, err := sCli.Decrypt(ct2) // inbound(client): s2c
	require.NoError(t, err)
	require.True(t, bytes.Equal(pt2, msg2), "server→client plaintext mismatch")

	// ---- 선택: RFC-9421 스타일 covered 서명/검증 확인(세션 HMAC) ----
	covered := []byte("@method:POST\n@path:/hpke/complete\nx-kid:" + kid)
	sig := sCli.SignCovered(covered)             // outbound(client)
	require.NoError(t, sSrv.VerifyCovered(covered, sig)) // inbound(server)

	// 추가 확인: 서버 Task 응답 메타에 포함되던 ackTag는
	// Client.Complete 내부에서 이미 검증을 끝냈음(ack mismatch 시 에러로 리턴)
}

// 보조: proto struct에서 b64 읽기 예(필요시)
// func b64Field(st *structpb.Struct, key string) ([]byte, error) {
// 	v, ok := st.Fields[key]
// 	if !ok {
// 		return nil, fmt.Errorf("missing %q", key)
// 	}
// 	return base64.RawURLEncoding.DecodeString(v.GetStringValue())
// }
