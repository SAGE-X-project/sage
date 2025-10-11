// SPDX-License-Identifier: LGPL-3.0-or-later
// security_regression_test.go
//
// This file provides regression tests and helper resolvers for HPKE handshake
// hardening: MITM/UKS, identity-binding, exporter misuse, replay protection,
// and DoS cookies/puzzles. All comments are in English as requested.

package hpke

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	sagedid "github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/pkg/agent/transport"
	"github.com/stretchr/testify/require"
	"github.com/test-go/testify/mock"
)

// Test setup helpers (transport + resolvers)

// setupHPKETestWithTransport returns a complete test rig and the MockTransport
// so tests can intercept/mutate the server response.
func setupHPKETestWithTransport(t *testing.T, srvCfg, cliCfg session.Config) (
	*Client,
	*Server,
	*session.Manager,
	*session.Manager,
	*mockResolver,
	*sagedid.MultiChainResolver,
	*transport.MockTransport,
	string, // clientDID
	string, // serverDID
) {
	t.Helper()

	// Unique DIDs for this test
	clientDID := "did:sage:test:client-" + uuid.NewString()
	serverDID := "did:sage:test:server-" + uuid.NewString()

	// Keys
	serverSignKP, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	serverKEMKP, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)
	clientSignKP, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)

	// Resolver wiring
	ethResolver := new(mockResolver)
	multiResolver := sagedid.NewMultiChainResolver()
	multiResolver.AddResolver(sagedid.ChainEthereum, ethResolver)

	aliceMeta := &sagedid.AgentMetadata{
		DID:       sagedid.AgentDID(clientDID),
		Name:      "Active Agent",
		IsActive:  true,
		PublicKey: clientSignKP, // store KeyPair so server can call Verify()
	}
	bobMeta := &sagedid.AgentMetadata{
		DID:          sagedid.AgentDID(serverDID),
		Name:         "Server Agent",
		IsActive:     true,
		PublicKEMKey: serverKEMKP.PublicKey(),
		PublicKey:    serverSignKP, // store KeyPair for convenience
	}

	// Default expectations for DID resolution
	ethResolver.On("Resolve", mock.Anything, sagedid.AgentDID(clientDID)).Return(aliceMeta, nil)
	ethResolver.On("Resolve", mock.Anything, sagedid.AgentDID(serverDID)).Return(bobMeta, nil)

	// Session managers
	srvMgr := session.NewManager()
	if srvCfg.MaxAge > 0 {
		srvMgr.SetDefaultConfig(srvCfg)
	}
	cliMgr := session.NewManager()
	if cliCfg.MaxAge > 0 {
		cliMgr.SetDefaultConfig(cliCfg)
	}
	t.Cleanup(func() {
		_ = srvMgr.Close()
		_ = cliMgr.Close()
		ethResolver.AssertExpectations(t)
	})

	// Transport + server
	mockTransport := &transport.MockTransport{}
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
	mockTransport.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		return srv.HandleMessage(ctx, msg)
	}

	// Client
	cli := NewClient(mockTransport, multiResolver, clientSignKP, clientDID, DefaultInfoBuilder{}, cliMgr)

	return cli, srv, srvMgr, cliMgr, ethResolver, multiResolver, mockTransport, clientDID, serverDID
}

// setupHPKETestWithCookiesAndTransport enables server-side cookie verifier and
// optional client cookie source for DoS mitigation tests.
func setupHPKETestWithCookiesAndTransport(
	t *testing.T,
	srvCfg, cliCfg session.Config,
	verifier CookieVerifier,
	source CookieSource,
) (
	*Client,
	*Server,
	*session.Manager,
	*session.Manager,
	*mockResolver,
	*sagedid.MultiChainResolver,
	*transport.MockTransport,
	string, // clientDID
	string, // serverDID
) {
	t.Helper()

	// Unique DIDs
	clientDID := "did:sage:test:client-" + uuid.NewString()
	serverDID := "did:sage:test:server-" + uuid.NewString()

	// Keys
	serverSignKP, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	serverKEMKP, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)
	clientSignKP, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)

	// Resolver wiring
	ethResolver := new(mockResolver)
	multiResolver := sagedid.NewMultiChainResolver()
	multiResolver.AddResolver(sagedid.ChainEthereum, ethResolver)

	aliceMeta := &sagedid.AgentMetadata{
		DID:       sagedid.AgentDID(clientDID),
		Name:      "Active Agent",
		IsActive:  true,
		PublicKey: clientSignKP,
	}
	bobMeta := &sagedid.AgentMetadata{
		DID:          sagedid.AgentDID(serverDID),
		Name:         "Server Agent",
		IsActive:     true,
		PublicKEMKey: serverKEMKP.PublicKey(),
		PublicKey:    serverSignKP,
	}

	ethResolver.On("Resolve", mock.Anything, sagedid.AgentDID(clientDID)).Return(aliceMeta, nil)
	ethResolver.On("Resolve", mock.Anything, sagedid.AgentDID(serverDID)).Return(bobMeta, nil)

	// Session managers
	srvMgr := session.NewManager()
	if srvCfg.MaxAge > 0 {
		srvMgr.SetDefaultConfig(srvCfg)
	}
	cliMgr := session.NewManager()
	if cliCfg.MaxAge > 0 {
		cliMgr.SetDefaultConfig(cliCfg)
	}
	t.Cleanup(func() {
		_ = srvMgr.Close()
		_ = cliMgr.Close()
		ethResolver.AssertExpectations(t)
	})

	// Transport + server with cookie verifier
	mt := &transport.MockTransport{}
	srv := NewServer(
		serverSignKP,
		srvMgr,
		serverDID,
		multiResolver,
		&ServerOpts{
			MaxSkew: 2 * time.Minute,
			Info:    DefaultInfoBuilder{},
			KEM:     serverKEMKP,
			Cookies: verifier, // enable DoS cookie/puzzle
		},
	)
	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		return srv.HandleMessage(ctx, msg)
	}

	// Client (optionally attach cookie source)
	cli := NewClient(mt, multiResolver, clientSignKP, clientDID, DefaultInfoBuilder{}, cliMgr)
	if source != nil {
		cli.WithCookieSource(source)
	}
	return cli, srv, srvMgr, cliMgr, ethResolver, multiResolver, mt, clientDID, serverDID
}

// CookieSourceFunc is a test adapter to fabricate cookies inline.
type CookieSourceFunc func(ctxID, initDID, respDID string) (string, bool)

func (f CookieSourceFunc) GetCookie(ctxID, initDID, respDID string) (string, bool) {
	return f(ctxID, initDID, respDID)
}

// Tests
// Happy-path smoke: server response must include Ed25519 signature and ackTag
func Test_ServerSignature_And_AckTag_HappyPath(t *testing.T) {
	ctx := context.Background()
	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	var captured []byte
	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		resp, err := srv.HandleMessage(ctx, msg)
		if err != nil {
			return nil, err
		}
		copied := make([]byte, len(resp.Data))
		copy(copied, resp.Data)
		captured = copied
		return resp, nil
	}

	ctxID := "ctx-" + uuid.NewString()
	kid, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.NoError(t, err)
	require.NotEmpty(t, kid)

	var m map[string]string
	require.NoError(t, json.Unmarshal(captured, &m))
	require.NotEmpty(t, m["sigB64"], "server response must include Ed25519 signature")
	require.NotEmpty(t, m["ackTagB64"], "server response must include ackTag")
	sig, err := base64.RawURLEncoding.DecodeString(m["sigB64"])
	require.NoError(t, err)
	require.Len(t, sig, 64)
}

// (1) Base misuse / MITM/UKS: wrong receiver KEM key must fail via ackTag
func Test_Client_ResolveKEM_WrongKey_Rejects(t *testing.T) {
	ctx := context.Background()

	cli, srv, _, _, _, baseResolver, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	attKEM, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	cliEvil := NewClient(
		mt,
		&evilResolver{base: baseResolver, serverDID: serverDID, attKEMPub: attKEM.PublicKey()},
		cli.key, cli.DID, cli.info, cli.sessMgr,
	)

	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		return srv.HandleMessage(ctx, msg)
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err = cliEvil.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "wrong KEM key must be detected by key confirmation (ackTag)")
}

// (2) Identity binding: verify server Ed25519 signature against wrong key -> fail
func Test_ServerSignature_VerifyAgainstWrongKey_Rejects(t *testing.T) {
	ctx := context.Background()

	cli, srv, _, _, _, baseResolver, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	wrongKP, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)

	cliWrong := NewClient(
		mt,
		&wrongSignResolver{base: baseResolver, serverDID: serverDID, wrongPub: wrongKP.PublicKey().(ed25519.PublicKey)},
		cli.key, cli.DID, cli.info, cli.sessMgr,
	)

	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		return srv.HandleMessage(ctx, msg)
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err = cliWrong.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "server signature must fail against a wrong Ed25519 key")
}

// (3) Exporter usage / transcript binding checks (ackTag, echoes, info hashes)
func Test_Tamper_AckTag_Fails(t *testing.T) {
	ctx := context.Background()
	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		resp, err := srv.HandleMessage(ctx, msg)
		if err != nil {
			return nil, err
		}
		var m map[string]string
		_ = json.Unmarshal(resp.Data, &m)
		m["ackTagB64"] = flipB64Byte(m["ackTagB64"]) // one-bit flip
		resp.Data, _ = json.Marshal(m)
		return resp, nil
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "tampered ackTag must be rejected")
}

func Test_Tamper_Signature_Fails(t *testing.T) {
	ctx := context.Background()
	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		resp, err := srv.HandleMessage(ctx, msg)
		if err != nil {
			return nil, err
		}
		var m map[string]string
		_ = json.Unmarshal(resp.Data, &m)
		m["sigB64"] = flipB64Byte(m["sigB64"]) // break signature
		resp.Data, _ = json.Marshal(m)
		return resp, nil
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "tampered server signature must be rejected")
}

func Test_Tamper_Enc_Echo_Fails(t *testing.T) {
	ctx := context.Background()
	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		resp, err := srv.HandleMessage(ctx, msg)
		if err != nil {
			return nil, err
		}
		var m map[string]string
		_ = json.Unmarshal(resp.Data, &m)
		if m["enc"] != "" {
			m["enc"] = flipB64Byte(m["enc"])
		}
		resp.Data, _ = json.Marshal(m)
		return resp, nil
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "enc echo mismatch must be rejected")
}

func Test_Tamper_EphC_Echo_Fails(t *testing.T) {
	ctx := context.Background()
	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		resp, err := srv.HandleMessage(ctx, msg)
		if err != nil {
			return nil, err
		}
		var m map[string]string
		_ = json.Unmarshal(resp.Data, &m)
		if m["ephC"] != "" {
			m["ephC"] = flipB64Byte(m["ephC"])
		}
		resp.Data, _ = json.Marshal(m)
		return resp, nil
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "ephC echo mismatch must be rejected")
}

func Test_Tamper_InfoHash_Fails(t *testing.T) {
	ctx := context.Background()
	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		resp, err := srv.HandleMessage(ctx, msg)
		if err != nil {
			return nil, err
		}
		var m map[string]string
		_ = json.Unmarshal(resp.Data, &m)
		bogus := bytes.Repeat([]byte{0x42}, 32) // bogus 32-byte hash
		m["infoHash"] = base64.RawURLEncoding.EncodeToString(bogus)
		resp.Data, _ = json.Marshal(m)
		return resp, nil
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "info/exportCtx hash mismatch must be rejected")
}

func Test_Server_Rejects_Info_ExportCtx_Mismatch(t *testing.T) {
	ctx := context.Background()

	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})
	cli.info = clientSkewInfo{} // inject skew on client only

	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		return srv.HandleMessage(ctx, msg)
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "server must reject when info/exportCtx differs")
}

// (4) Replay protection: resubmit identical SecureMessage -> reject
func Test_Replay_Protection_Works(t *testing.T) {
	ctx := context.Background()
	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	var captured *transport.SecureMessage
	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		c := *msg
		c.Payload = append([]byte(nil), msg.Payload...)
		c.Signature = append([]byte(nil), msg.Signature...)
		captured = &c
		return srv.HandleMessage(ctx, msg)
	}

	ctxID := "ctx-" + uuid.NewString()
	kid, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.NoError(t, err)
	require.NotEmpty(t, kid)

	_, err = srv.HandleMessage(ctx, captured)
	require.Error(t, err, "replay must be detected")
}

// (5) DoS mitigation policy: optional vs required; HMAC & PoW flows

// Optional policy: if server has no verifier configured, missing cookie is allowed.
func Test_DoS_Cookie_Optional_Allows_When_NotConfigured(t *testing.T) {
	ctx := context.Background()
	cli, _, _, _, _, _, _, clientDID, serverDID :=
		setupHPKETestWithCookiesAndTransport(t, session.Config{}, session.Config{}, nil, nil)
	ctxID := "ctx-" + uuid.NewString()
	kid, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.NoError(t, err, "cookie must be optional when verifier is nil")
	require.NotEmpty(t, kid)
}

// Required policy: if server has a verifier, missing cookie is rejected.
func Test_DoS_Cookie_Missing_Rejects(t *testing.T) {
	ctx := context.Background()
	verifier := &hmacCookieVerifier{secret: []byte("server-secret")}
	cli, _, _, _, _, _, _, clientDID, serverDID :=
		setupHPKETestWithCookiesAndTransport(t, session.Config{}, session.Config{}, verifier, nil)

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "server must require cookie when verifier is configured")
}

func Test_DoS_Cookie_HMAC_Valid_Allows(t *testing.T) {
	ctx := context.Background()
	secret := []byte("shared-secret")
	verifier := &hmacCookieVerifier{secret: secret}
	source := &hmacCookieSource{secret: secret}

	cli, _, _, _, _, _, _, clientDID, serverDID :=
		setupHPKETestWithCookiesAndTransport(t, session.Config{}, session.Config{}, verifier, source)

	ctxID := "ctx-" + uuid.NewString()
	kid, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.NoError(t, err)
	require.NotEmpty(t, kid)
}

func Test_DoS_Cookie_HMAC_WrongSecret_Rejects(t *testing.T) {
	ctx := context.Background()
	verifier := &hmacCookieVerifier{secret: []byte("server-secret")}
	source := &hmacCookieSource{secret: []byte("client-wrong-secret")}

	cli, _, _, _, _, _, _, clientDID, serverDID :=
		setupHPKETestWithCookiesAndTransport(t, session.Config{}, session.Config{}, verifier, source)

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "wrong HMAC cookie secret must be rejected")
}

func Test_DoS_Puzzle_PoW_Valid_Allows(t *testing.T) {
	ctx := context.Background()
	diff := 1 // tiny difficulty to keep tests fast
	verifier := &powCookieVerifier{difficulty: diff}
	source := &powCookieSource{difficulty: diff}

	cli, _, _, _, _, _, _, clientDID, serverDID :=
		setupHPKETestWithCookiesAndTransport(t, session.Config{}, session.Config{}, verifier, source)

	ctxID := "ctx-" + uuid.NewString()
	kid, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.NoError(t, err)
	require.NotEmpty(t, kid)
}

func Test_DoS_Puzzle_PoW_Tamper_Rejects(t *testing.T) {
	ctx := context.Background()
	diff := 1
	verifier := &powCookieVerifier{difficulty: diff}
	source := &powCookieSource{difficulty: diff}

	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithCookiesAndTransport(t, session.Config{}, session.Config{}, verifier, source)

	// Tamper cookie in transit to ensure early rejection.
	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		if msg.Metadata != nil && msg.Metadata["cookie"] != "" {
			msg.Metadata["cookie"] += "X"
		}
		return srv.HandleMessage(ctx, msg)
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "tampered PoW cookie must be rejected")
}

func Test_DoS_Cookie_EarlyReject_Path(t *testing.T) {
	ctx := context.Background()
	verifier := &hmacCookieVerifier{secret: []byte("server-secret")}
	source := CookieSourceFunc(func(ctxID, initDID, respDID string) (string, bool) {
		// Return syntactically invalid cookie to trigger early fail.
		return "invalid-prefix:abcdef", true
	})

	cli, _, _, _, _, _, _, clientDID, serverDID :=
		setupHPKETestWithCookiesAndTransport(t, session.Config{}, session.Config{}, verifier, source)

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "invalid cookie prefix must be rejected")
}

// (6) Zeroization helper: sanity check
func Test_ZeroBytes_Wipes_Buffer(t *testing.T) {
	b := []byte{1, 2, 3, 4}
	zeroBytes(b) // helper provided by the implementation under test
	for i, v := range b {
		if v != 0 {
			t.Fatalf("expected zeroized buffer at %d, found %d", i, v)
		}
	}
}

// Client rejects a response carrying an all-zero ECDH secret.
func Test_Tamper_EphS_AllZero_Fails(t *testing.T) {
	ctx := context.Background()
	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	// Tamper: server returns ephS = 32 zero bytes
	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		resp, err := srv.HandleMessage(ctx, msg)
		if err != nil {
			return nil, err
		}
		var m map[string]string
		_ = json.Unmarshal(resp.Data, &m)
		m["ephS"] = base64.RawURLEncoding.EncodeToString(make([]byte, 32))
		resp.Data, _ = json.Marshal(m)
		return resp, nil
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "ECDH all-zero must be rejected")
}

// Reject responses that attempt a protocol version downgrade.
func Test_Downgrade_Version_Rejects(t *testing.T) {
	ctx := context.Background()
	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		resp, err := srv.HandleMessage(ctx, msg)
		if err != nil {
			return nil, err
		}
		var m map[string]string
		_ = json.Unmarshal(resp.Data, &m)
		m["v"] = "v0" // downgrade
		resp.Data, _ = json.Marshal(m)
		return resp, nil
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "version downgrade must be rejected")
}

// Reject responses that advertise an unexpected task identifier.
func Test_Downgrade_Task_Rejects(t *testing.T) {
	ctx := context.Background()
	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		resp, err := srv.HandleMessage(ctx, msg)
		if err != nil {
			return nil, err
		}
		var m map[string]string
		_ = json.Unmarshal(resp.Data, &m)
		m["task"] = "hpke/other@v1" // wrong task
		resp.Data, _ = json.Marshal(m)
		return resp, nil
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "unexpected task must be rejected")
}

// Client pinning: reject when the returned key differs from the stored pin.
func Test_Client_Pinning_Rejects_KeyChange(t *testing.T) {
	ctx := context.Background()
	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	// Preload WRONG pin for serverDID
	wrongKP, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	// Client.pins is exported within the same package.
	if cli.pins == nil {
		cli.pins = make(map[string][]byte)
	}
	cli.pins[serverDID] = wrongKP.PublicKey().(ed25519.PublicKey)

	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		return srv.HandleMessage(ctx, msg)
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err = cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "pin mismatch must be rejected")
}

// Server suite whitelist: reject suites that are not explicitly allowed.
func Test_SuiteWhitelist_Rejects_Unsupported(t *testing.T) {
	ctx := context.Background()

	srvCfg, cliCfg := session.Config{}, session.Config{}
	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, srvCfg, cliCfg)

	// Configure allowedSuites with a value different from the active suite
	//    => the server should reject with "suite not allowed".
	srv.allowedSuites = []string{"UNSUPPORTED-SUITE"}

	// Route the message straight to the server.
	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		return srv.HandleMessage(ctx, msg)
	}

	ctxID := "ctx-" + uuid.NewString()
	_, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.Error(t, err, "suite not allowed must be rejected")
}

func Test_SuiteWhitelist_Allows_CurrentSuite(t *testing.T) {
	ctx := context.Background()
	cli, srv, _, _, _, _, mt, clientDID, serverDID :=
		setupHPKETestWithTransport(t, session.Config{}, session.Config{})

	// Allow the suite used by the current implementation.
	srv.allowedSuites = []string{hpkeSuiteID}

	mt.SendFunc = func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
		return srv.HandleMessage(ctx, msg)
	}

	ctxID := "ctx-" + uuid.NewString()
	kid, err := cli.Initialize(ctx, ctxID, clientDID, serverDID)
	require.NoError(t, err)
	require.NotEmpty(t, kid)
}

// Local helper functions

// flipB64Byte decodes a base64url string, flips one bit, and re-encodes.
// Keeps length/encoding intact so JSON parsing still succeeds.
func flipB64Byte(in string) string {
	b, err := base64.RawURLEncoding.DecodeString(in)
	if err != nil || len(b) == 0 {
		return in
	}
	b[0] ^= 0x01
	return base64.RawURLEncoding.EncodeToString(b)
}

// Evil resolvers for negative-path tests (package scope)
// evilResolver lies about the server's KEM pubkey (to simulate MITM/UKS).
type evilResolver struct {
	base      *sagedid.MultiChainResolver
	serverDID string
	attKEMPub interface{}
}

var _ sagedid.Resolver = (*evilResolver)(nil)

func (e *evilResolver) Resolve(ctx context.Context, did sagedid.AgentDID) (*sagedid.AgentMetadata, error) {
	return e.base.Resolve(ctx, did)
}
func (e *evilResolver) ResolvePublicKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	return e.base.ResolvePublicKey(ctx, did)
}
func (e *evilResolver) ResolveKEMKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	if string(did) == e.serverDID {
		if _, err := e.base.Resolve(ctx, did); err != nil {
			return nil, err
		}
		return e.attKEMPub, nil // wrong KEM key for the server DID
	}
	return e.base.ResolveKEMKey(ctx, did)
}
func (e *evilResolver) VerifyMetadata(ctx context.Context, did sagedid.AgentDID, md *sagedid.AgentMetadata) (*sagedid.VerificationResult, error) {
	return e.base.VerifyMetadata(ctx, did, md)
}
func (e *evilResolver) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*sagedid.AgentMetadata, error) {
	return e.base.ListAgentsByOwner(ctx, ownerAddress)
}
func (e *evilResolver) Search(ctx context.Context, c sagedid.SearchCriteria) ([]*sagedid.AgentMetadata, error) {
	return e.base.Search(ctx, c)
}

// wrongSignResolver lies about the server's Ed25519 pubkey for signature verify.
type wrongSignResolver struct {
	base      *sagedid.MultiChainResolver
	serverDID string
	wrongPub  ed25519.PublicKey
}

var _ sagedid.Resolver = (*wrongSignResolver)(nil)

func (r *wrongSignResolver) Resolve(ctx context.Context, did sagedid.AgentDID) (*sagedid.AgentMetadata, error) {
	return r.base.Resolve(ctx, did)
}
func (r *wrongSignResolver) ResolvePublicKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	if string(did) == r.serverDID {
		if _, err := r.base.Resolve(ctx, did); err != nil {
			return nil, err
		}
		return r.wrongPub, nil // wrong Ed25519 public key
	}
	return r.base.ResolvePublicKey(ctx, did)
}
func (r *wrongSignResolver) ResolveKEMKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	return r.base.ResolveKEMKey(ctx, did)
}
func (r *wrongSignResolver) VerifyMetadata(ctx context.Context, did sagedid.AgentDID, md *sagedid.AgentMetadata) (*sagedid.VerificationResult, error) {
	return r.base.VerifyMetadata(ctx, did, md)
}
func (r *wrongSignResolver) ListAgentsByOwner(ctx context.Context, owner string) ([]*sagedid.AgentMetadata, error) {
	return r.base.ListAgentsByOwner(ctx, owner)
}
func (r *wrongSignResolver) Search(ctx context.Context, c sagedid.SearchCriteria) ([]*sagedid.AgentMetadata, error) {
	return r.base.Search(ctx, c)
}

// Client-only skew of exportCtx (used by mismatch test)
type clientSkewInfo struct{ DefaultInfoBuilder }

func (clientSkewInfo) BuildExportContext(ctxID string) []byte {
	return []byte("SKEW-" + string(DefaultInfoBuilder{}.BuildExportContext(ctxID)))
}

// DoS cookie/puzzle helpers (HMAC and PoW) used by tests
// HMAC cookie format: "hmac:<b64url(hmac(v1|ctx|init|resp))>"
type hmacCookieVerifier struct{ secret []byte }

func (v *hmacCookieVerifier) Verify(cookie, ctxID, initDID, respDID string) bool {
	const prefix = "hmac:"
	if len(cookie) <= len(prefix) || cookie[:len(prefix)] != prefix {
		return false
	}
	gotRaw, err := base64.RawURLEncoding.DecodeString(cookie[len(prefix):])
	if err != nil {
		return false
	}
	m := hmac.New(sha256.New, v.secret)
	m.Write([]byte("SAGE-Cookie|v1|"))
	m.Write([]byte(ctxID))
	m.Write([]byte("|"))
	m.Write([]byte(initDID))
	m.Write([]byte("|"))
	m.Write([]byte(respDID))
	exp := m.Sum(nil)
	return hmac.Equal(gotRaw, exp)
}

type hmacCookieSource struct{ secret []byte }

func (s *hmacCookieSource) GetCookie(ctxID, initDID, respDID string) (string, bool) {
	m := hmac.New(sha256.New, s.secret)
	m.Write([]byte("SAGE-Cookie|v1|"))
	m.Write([]byte(ctxID))
	m.Write([]byte("|"))
	m.Write([]byte(initDID))
	m.Write([]byte("|"))
	m.Write([]byte(respDID))
	out := base64.RawURLEncoding.EncodeToString(m.Sum(nil))
	return "hmac:" + out, true
}

// PoW cookie format: "pow:<nonce>:<hex(sha256(ctx|init|resp|nonce))>"
// Difficulty = # of leading zero nibbles in the SHA-256 digest.
type powCookieVerifier struct{ difficulty int }

func leadingZeroNibbles(sum []byte) int {
	n := 0
	for _, b := range sum {
		hi := b >> 4
		lo := b & 0x0F
		if hi == 0 {
			n++
		} else {
			return n
		}
		if lo == 0 {
			n++
		} else {
			return n
		}
	}
	return n
}

func (v *powCookieVerifier) Verify(cookie, ctxID, initDID, respDID string) bool {
	parts := strings.SplitN(cookie, ":", 3)
	if len(parts) != 3 || parts[0] != "pow" || parts[1] == "" || parts[2] == "" {
		return false
	}
	nonce := parts[1]
	hexHash := parts[2]
	sum := sha256.Sum256([]byte("SAGE-PoW|" + ctxID + "|" + initDID + "|" + respDID + "|" + nonce))
	if hex.EncodeToString(sum[:]) != hexHash {
		return false
	}
	return leadingZeroNibbles(sum[:]) >= v.difficulty
}

type powCookieSource struct{ difficulty int }

func (s *powCookieSource) GetCookie(ctxID, initDID, respDID string) (string, bool) {
	for nonce := 0; nonce < 1<<24; nonce++ {
		ns := fmt.Sprintf("%x", nonce)
		sum := sha256.Sum256([]byte("SAGE-PoW|" + ctxID + "|" + initDID + "|" + respDID + "|" + ns))
		if leadingZeroNibbles(sum[:]) >= s.difficulty {
			return fmt.Sprintf("pow:%s:%s", ns, hex.EncodeToString(sum[:])), true
		}
	}
	return "", false
}
