// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


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
	// - Server HPKE (X25519) keypair (receiver opens ciphertexts)
	serverKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	// - Client Ed25519 signing key (used for metadata signatures)
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

	// Session managers (one per side)
	srvMgr := session.NewManager()
	cliMgr := session.NewManager()
	t.Cleanup(func() {
		_ = srvMgr.Close()
		_ = cliMgr.Close()
	})

	// HPKE server instance
	srv := NewServer(serverKeyPair, srvMgr, serverDID, multiResolver, &ServerOpts{
		MaxSkew: 2 * time.Minute,
		Info:    DefaultInfoBuilder{},
	})
	// Start gRPC bufconn server
	_, lis := startBufGRPC(t, srv)

	// Build gRPC client connection and A2A stub
	conn := dialBuf(t, lis)

	// HPKE client wrapper
	cli := NewClient(conn, multiResolver, clientKeypair, clientDID, DefaultInfoBuilder{}, cliMgr)

	// ---- Run Complete(): HPKE init + session creation + kid binding ----
	ctxID := "ctx-" + uuid.NewString()

	kid, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.NoError(t, err)
	require.NotEmpty(t, kid, "kid should be returned by server and bound on both ends")

	// Ensure both server and client resolve the session by kid
	sSrv, ok := srvMgr.GetByKeyID(kid)
	require.True(t, ok, "server should have bound kid to a session")
	sCli, ok := cliMgr.GetByKeyID(kid)
	require.True(t, ok, "client should have bound kid to a session")

	// ---- Directional key validation: confirm encrypt/decrypt for C->S and S->C ----
	// Client -> Server
	msg1 := []byte("hello from client")
	ct1, err := sCli.Encrypt(msg1) // outbound(client): c2s
	require.NoError(t, err)
	pt1, err := sSrv.Decrypt(ct1) // inbound(server): c2s
	require.NoError(t, err)
	require.True(t, bytes.Equal(pt1, msg1), "client->server plaintext mismatch")

	// Server -> Client
	msg2 := []byte("hello from server")
	ct2, err := sSrv.Encrypt(msg2) // outbound(server): s2c
	require.NoError(t, err)
	pt2, err := sCli.Decrypt(ct2) // inbound(client): s2c
	require.NoError(t, err)
	require.True(t, bytes.Equal(pt2, msg2), "server->client plaintext mismatch")

	covered := []byte("@method:POST\n@path:/hpke/complete\nx-kid:" + kid)
	sig := sCli.SignCovered(covered)                     // outbound(client)
	require.NoError(t, sSrv.VerifyCovered(covered, sig)) // inbound(server)
}
