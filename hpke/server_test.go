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

type mockResolver struct{ mock.Mock }

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

func Test_HPKE_PFS(t *testing.T) {
	ctx := context.Background()

	ethResolver := new(mockResolver)
	multiResolver := sagedid.NewMultiChainResolver()
	multiResolver.AddResolver(sagedid.ChainEthereum, ethResolver)

	const clientDID = "did:sage:test:client"
	const serverDID = "did:sage:test:server"

	// Server: Ed25519 (metadata signing) + X25519 (static KEM)
	serverSignKP, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	serverKEMKP, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	// Client: Ed25519 (metadata signing)
	clientSignKP, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)

	aliceMeta := &did.AgentMetadata{
		DID:       clientDID,
		Name:      "Active Agent",
		IsActive:  true,
		PublicKey: clientSignKP.PublicKey(), // Used by the server to verify metadata signatures
	}
	bobMeta := &did.AgentMetadata{
		DID:          serverDID,
		Name:         "Active Agent",
		IsActive:     true,
		PublicKEMKey: serverKEMKP.PublicKey(),
		PublicKey:    serverSignKP.PublicKey(), // Used by the client when resolving the HPKE KEM public key
	}

	ethResolver.On("Resolve", mock.Anything, did.AgentDID(clientDID)).Return(aliceMeta, nil).Once()
	ethResolver.On("Resolve", mock.Anything, did.AgentDID(serverDID)).Return(bobMeta, nil).Twice()

	srvMgr := session.NewManager()
	cliMgr := session.NewManager()
	t.Cleanup(func() { _ = srvMgr.Close(); _ = cliMgr.Close() })

	srv := NewServer(
		serverSignKP,
		srvMgr,
		serverDID,
		multiResolver,
		&ServerOpts{
			MaxSkew: 2 * time.Minute,
			Info:    DefaultInfoBuilder{},
			KEM:     serverKEMKP,
		},
	)
	_, lis := startBufGRPC(t, srv)
	conn := dialBuf(t, lis)

	cli := NewClient(conn, multiResolver, clientSignKP, clientDID, DefaultInfoBuilder{}, cliMgr)

	ctxID := "ctx-" + uuid.NewString()
	kid, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.NoError(t, err)
	require.NotEmpty(t, kid)

	sSrv, ok := srvMgr.GetByKeyID(kid)
	require.True(t, ok)
	sCli, ok := cliMgr.GetByKeyID(kid)
	require.True(t, ok)

	// C->S
	msg1 := []byte("hello from client")
	ct1, err := sCli.Encrypt(msg1)
	require.NoError(t, err)
	pt1, err := sSrv.Decrypt(ct1)
	require.NoError(t, err)
	require.True(t, bytes.Equal(pt1, msg1))

	// S->C
	msg2 := []byte("hello from server")
	ct2, err := sSrv.Encrypt(msg2)
	require.NoError(t, err)
	pt2, err := sCli.Decrypt(ct2)
	require.NoError(t, err)
	require.True(t, bytes.Equal(pt2, msg2))

	// Covered signature path
	covered := []byte("@method:POST\n@path:/hpke/complete\nx-kid:" + kid)
	sig := sCli.SignCovered(covered)
	require.NoError(t, sSrv.VerifyCovered(covered, sig))
}

// Explicit DHKEM exporter equality test (X25519)
func Test_HPKE_DHKEM_ExporterEquality(t *testing.T) {
	// Receiver (server) static KEM keypair
	serverKEMKP, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	// Canonical HPKE "info" and "exportCtx"
	info := []byte("sage/hpke-handshake v1|ctx:ctx-abc|init:did:alice|resp:did:bob")
	exportCtx := []byte("sage/session exporter v1")

	// Sender derives (enc, exporter)
	enc, exporterA, err := keys.HPKEDeriveSharedSecretToPeer(serverKEMKP.PublicKey(), info, exportCtx, 32)
	require.NoError(t, err)
	require.Len(t, enc, 32)
	require.Len(t, exporterA, 32)

	// Receiver opens enc and derives exporterB
	exporterB, err := keys.HPKEOpenSharedSecretWithPriv(serverKEMKP.PrivateKey(), enc, info, exportCtx, 32)
	require.NoError(t, err)
	require.True(t, bytes.Equal(exporterA, exporterB), "HPKE exporter secrets must match")
}

// Session lifecycle (init -> refresh by use -> idle-expire)
func Test_Session_Lifecycle_IdleExpiry(t *testing.T) {
	ctx := context.Background()

	// Mock resolver stack
	ethResolver := new(mockResolver)
	multiResolver := sagedid.NewMultiChainResolver()
	multiResolver.AddResolver(sagedid.ChainEthereum, ethResolver)

	const clientDID = "did:sage:test:client"
	const serverDID = "did:sage:test:server"

	// Keys: server Ed25519(sign) + X25519(KEM), client Ed25519(sign)
	serverSignKP, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	serverKEMKP, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)
	clientSignKP, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)

	// DID metadata: put Ed25519 into PublicKey, X25519 into PublicKEMKey
	aliceMeta := &did.AgentMetadata{
		DID:       clientDID,
		Name:      "Client",
		IsActive:  true,
		PublicKey: clientSignKP.PublicKey(), // server verifies metadata signatures
	}
	bobMeta := &did.AgentMetadata{
		DID:          serverDID,
		Name:         "Server",
		IsActive:     true,
		PublicKEMKey: serverKEMKP.PublicKey(), // client uses this for HPKE
		PublicKey:    serverSignKP.PublicKey(),
	}

	// Resolver answers
	ethResolver.On("Resolve", mock.Anything, did.AgentDID(clientDID)).Return(aliceMeta, nil).Once()
	ethResolver.On("Resolve", mock.Anything, did.AgentDID(serverDID)).Return(bobMeta, nil).Twice()

	// Tight lifetimes to exercise idle expiry
	srvMgr := session.NewManager()
	srvMgr.SetDefaultConfig(session.Config{
		MaxAge:      30 * time.Second, // generous absolute life
		IdleTimeout: 1200 * time.Millisecond,
		MaxMessages: 1000,
	})
	cliMgr := session.NewManager()
	cliMgr.SetDefaultConfig(session.Config{
		MaxAge:      30 * time.Second,
		IdleTimeout: 1200 * time.Millisecond,
		MaxMessages: 1000,
	})
	t.Cleanup(func() { _ = srvMgr.Close(); _ = cliMgr.Close() })

	// Boot server and client over in-memory gRPC
	srv := NewServer(serverSignKP, srvMgr, serverDID, multiResolver, &ServerOpts{
		MaxSkew: 2 * time.Minute,
		Info:    DefaultInfoBuilder{},
		KEM:     serverKEMKP,
	})
	_, lis := startBufGRPC(t, srv)
	conn := dialBuf(t, lis)
	cli := NewClient(conn, multiResolver, clientSignKP, clientDID, DefaultInfoBuilder{}, cliMgr)

	// Initialize one session
	ctxID := "ctx-" + uuid.NewString()
	kid, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.NoError(t, err)
	require.NotEmpty(t, kid)

	sSrv, ok := srvMgr.GetByKeyID(kid)
	require.True(t, ok)
	sCli, ok := cliMgr.GetByKeyID(kid)
	require.True(t, ok)

	// 1) Fresh session works
	ct1, err := sCli.Encrypt([]byte("m1"))
	require.NoError(t, err)
	_, err = sSrv.Decrypt(ct1)
	require.NoError(t, err)

	// 2) Use before idle timeout => still valid (also refreshes last-used)
	time.Sleep(600 * time.Millisecond)
	ct2, err := sCli.Encrypt([]byte("m2"))
	require.NoError(t, err)
	_, err = sSrv.Decrypt(ct2)
	require.NoError(t, err)

	// 3) Sleep past idle timeout => expect decrypt to fail
	time.Sleep(1400 * time.Millisecond)
	_, err = sCli.Encrypt([]byte("m3"))
	require.Error(t, err, "decrypt after idle timeout should fail")	
}

// MaxMessages enforcement (absolute counter per session)
func Test_Session_MaxMessages_Enforced(t *testing.T) {
	ctx := context.Background()

	ethResolver := new(mockResolver)
	multiResolver := sagedid.NewMultiChainResolver()
	multiResolver.AddResolver(sagedid.ChainEthereum, ethResolver)

	const clientDID = "did:sage:test:client2"
	const serverDID = "did:sage:test:server2"

	serverSignKP, _ := keys.GenerateEd25519KeyPair()
	serverKEMKP, _ := keys.GenerateX25519KeyPair()
	clientSignKP, _ := keys.GenerateEd25519KeyPair()

	aliceMeta := &did.AgentMetadata{
		DID:       clientDID,
		Name:      "Client2",
		IsActive:  true,
		PublicKey: clientSignKP.PublicKey(),
	}
	bobMeta := &did.AgentMetadata{
		DID:          serverDID,
		Name:         "Server2",
		IsActive:     true,
		PublicKEMKey: serverKEMKP.PublicKey(),
		PublicKey:    serverSignKP.PublicKey(),
	}
	ethResolver.On("Resolve", mock.Anything, did.AgentDID(clientDID)).Return(aliceMeta, nil).Once()
	ethResolver.On("Resolve", mock.Anything, did.AgentDID(serverDID)).Return(bobMeta, nil).Twice()

	// Very small MaxMessages to trip the limiter quickly
	srvMgr := session.NewManager()
	srvMgr.SetDefaultConfig(session.Config{
		MaxAge:      time.Minute,
		IdleTimeout: 3 * time.Minute,
		MaxMessages: 2, // allow only two messages per direction
	})
	cliMgr := session.NewManager()
	cliMgr.SetDefaultConfig(session.Config{
		MaxAge:      time.Minute,
		IdleTimeout: 3 * time.Minute,
		MaxMessages: 2,
	})
	t.Cleanup(func() { _ = srvMgr.Close(); _ = cliMgr.Close() })

	srv := NewServer(serverSignKP, srvMgr, serverDID, multiResolver, &ServerOpts{
		MaxSkew: 2 * time.Minute,
		Info:    DefaultInfoBuilder{},
		KEM:     serverKEMKP,
	})
	_, lis := startBufGRPC(t, srv)
	conn := dialBuf(t, lis)
	cli := NewClient(conn, multiResolver, clientSignKP, clientDID, DefaultInfoBuilder{}, cliMgr)

	ctxID := "ctx-" + uuid.NewString()
	kid, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.NoError(t, err)
	require.NotEmpty(t, kid)

	sSrv, ok := srvMgr.GetByKeyID(kid)
	require.True(t, ok)
	sCli, ok := cliMgr.GetByKeyID(kid)
	require.True(t, ok)

	// 1st message OK
	ct1, err := sCli.Encrypt([]byte("one"))
	require.NoError(t, err)
	_, err = sSrv.Decrypt(ct1)
	require.NoError(t, err)

	// 2nd message OK
	ct2, err := sCli.Encrypt([]byte("two"))
	require.NoError(t, err)
	_, err = sSrv.Decrypt(ct2)
	require.NoError(t, err)

	// 3rd should exceed MaxMessages on decrypt side
	_, err = sCli.Encrypt([]byte("three"))
	require.Error(t, err, "should fail after MaxMessages is exceeded")	
}

//  AEAD integrity (tag tamper should fail)
func Test_AEAD_TagIntegrity_TamperFails(t *testing.T) {
	ctx := context.Background()

	ethResolver := new(mockResolver)
	multiResolver := sagedid.NewMultiChainResolver()
	multiResolver.AddResolver(sagedid.ChainEthereum, ethResolver)

	const clientDID = "did:sage:test:client3"
	const serverDID = "did:sage:test:server3"

	serverSignKP, _ := keys.GenerateEd25519KeyPair()
	serverKEMKP, _ := keys.GenerateX25519KeyPair()
	clientSignKP, _ := keys.GenerateEd25519KeyPair()

	aliceMeta := &did.AgentMetadata{
		DID:       clientDID,
		Name:      "Client3",
		IsActive:  true,
		PublicKey: clientSignKP.PublicKey(),
	}
	bobMeta := &did.AgentMetadata{
		DID:          serverDID,
		Name:         "Server3",
		IsActive:     true,
		PublicKEMKey: serverKEMKP.PublicKey(),
		PublicKey:    serverSignKP.PublicKey(),
	}
	ethResolver.On("Resolve", mock.Anything, did.AgentDID(clientDID)).Return(aliceMeta, nil).Once()
	ethResolver.On("Resolve", mock.Anything, did.AgentDID(serverDID)).Return(bobMeta, nil).Twice()

	srvMgr := session.NewManager()
	cliMgr := session.NewManager()
	t.Cleanup(func() { _ = srvMgr.Close(); _ = cliMgr.Close() })

	srv := NewServer(serverSignKP, srvMgr, serverDID, multiResolver, &ServerOpts{
		MaxSkew: 2 * time.Minute,
		Info:    DefaultInfoBuilder{},
		KEM:     serverKEMKP,
	})
	_, lis := startBufGRPC(t, srv)
	conn := dialBuf(t, lis)
	cli := NewClient(conn, multiResolver, clientSignKP, clientDID, DefaultInfoBuilder{}, cliMgr)

	ctxID := "ctx-" + uuid.NewString()
	kid, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.NoError(t, err)
	require.NotEmpty(t, kid)

	sSrv, ok := srvMgr.GetByKeyID(kid)
	require.True(t, ok)
	sCli, ok := cliMgr.GetByKeyID(kid)
	require.True(t, ok)

	// Encrypt a message
	ct, err := sCli.Encrypt([]byte("integrity-check"))
	require.NoError(t, err)
	require.True(t, len(ct) > 16, "nonce(12)+cipher+tag(16) expected")

	// Tamper: flip the last byte (within tag) to break Poly1305 verification
	bad := append([]byte(nil), ct...)
	bad[len(bad)-1] ^= 0x01

	// Decrypt should fail with authentication error
	_, err = sSrv.Decrypt(bad)
	require.Error(t, err, "tampered ciphertext must fail AEAD authentication")
}

// Session ID / KeyID uniqueness smoke-check
// We don't introspect internal session IDs; we use KeyIDs bound on both ends.
// Creating two sessions should yield different kids (unique binding).
func Test_Session_KeyID_Uniqueness(t *testing.T) {
	ctx := context.Background()

	ethResolver := new(mockResolver)
	multiResolver := sagedid.NewMultiChainResolver()
	multiResolver.AddResolver(sagedid.ChainEthereum, ethResolver)

	const clientDID = "did:sage:test:client4"
	const serverDID = "did:sage:test:server4"

	serverSignKP, _ := keys.GenerateEd25519KeyPair()
	serverKEMKP, _ := keys.GenerateX25519KeyPair()
	clientSignKP, _ := keys.GenerateEd25519KeyPair()

	aliceMeta := &did.AgentMetadata{DID: clientDID, Name: "Client4", IsActive: true, PublicKey: clientSignKP.PublicKey()}
	bobMeta := &did.AgentMetadata{DID: serverDID, Name: "Server4", IsActive: true, PublicKEMKey: serverKEMKP.PublicKey(), PublicKey: serverSignKP.PublicKey()}

	ethResolver.On("Resolve", mock.Anything, did.AgentDID(clientDID)).Return(aliceMeta, nil).Twice()
	ethResolver.On("Resolve", mock.Anything, did.AgentDID(serverDID)).Return(bobMeta, nil).Times(4)

	srvMgr := session.NewManager()
	cliMgr := session.NewManager()
	t.Cleanup(func() { _ = srvMgr.Close(); _ = cliMgr.Close() })

	srv := NewServer(serverSignKP, srvMgr, serverDID, multiResolver, &ServerOpts{
		MaxSkew: 2 * time.Minute,
		Info:    DefaultInfoBuilder{},
		KEM:     serverKEMKP,
	})
	_, lis := startBufGRPC(t, srv)
	conn := dialBuf(t, lis)
	cli := NewClient(conn, multiResolver, clientSignKP, clientDID, DefaultInfoBuilder{}, cliMgr)

	// Create two distinct sessions (contexts) and ensure different kids
	ctxID1 := "ctx-" + uuid.NewString()
	kid1, err := cli.Initialize(ctx, ctxID1, clientDID, serverDID)
	require.NoError(t, err)
	require.NotEmpty(t, kid1)

	ctxID2 := "ctx-" + uuid.NewString()
	kid2, err := cli.Initialize(ctx, ctxID2, clientDID, serverDID)
	require.NoError(t, err)
	require.NotEmpty(t, kid2)


	require.NotEqual(t, kid1, kid2, "KeyIDs should be unique per session")

	srvSession, _ := srvMgr.GetByKeyID(kid1)
	cliSession, _ := cliMgr.GetByKeyID(kid1)

	// If the shared secret is the same, the session ID will also be the same.
	require.Equal(t, srvSession.GetID(), cliSession.GetID(), "shared secrets should match across phases")
	srvSession, _ = srvMgr.GetByKeyID(kid2)
	cliSession, _ = cliMgr.GetByKeyID(kid2)
	require.Equal(t, srvSession.GetID(), cliSession.GetID(), "shared secrets should match across phases")
	
}
