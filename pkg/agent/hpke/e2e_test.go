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
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	sagedid "github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// TestE2E_HPKE_Handshake_MockTransport tests complete HPKE handshake flow
// using MockTransport instead of actual gRPC/A2A transport.
// This replaces the integration test in tests/integration/tests/session/handshake/
func TestE2E_HPKE_Handshake_MockTransport(t *testing.T) {
	ctx := context.Background()

	// Setup: Generate keypairs for client and server
	clientKP, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	clientDID := sagedid.GenerateDID(sagedid.ChainEthereum, "cli-"+uuid.NewString())

	serverKP, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	serverDID := sagedid.GenerateDID(sagedid.ChainEthereum, "srv-"+uuid.NewString())

	// Generate KEM keypairs for HPKE
	clientKEMKP, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	serverKEMKP, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	// Create mock resolver for both client and server
	// Note: PublicKey must be the KeyPair, not the raw crypto.PublicKey,
	// because the server needs to call Verify() method during signature verification
	resolver := &e2eResolver{
		dids: map[sagedid.AgentDID]*sagedid.AgentMetadata{
			clientDID: {
				DID:          clientDID,
				IsActive:     true,
				PublicKey:    clientKP, // Store the KeyPair, not raw ed25519.PublicKey
				PublicKEMKey: clientKEMKP.PublicKey(),
			},
			serverDID: {
				DID:          serverDID,
				IsActive:     true,
				PublicKey:    serverKP, // Store the KeyPair, not raw ed25519.PublicKey
				PublicKEMKey: serverKEMKP.PublicKey(),
			},
		},
	}

	// Create session managers
	clientSessMgr := session.NewManager()
	serverSessMgr := session.NewManager()

	// Create server-side HPKE instance
	serverHPKE := NewServer(
		serverKP, // Server signing keypair
		serverSessMgr,
		string(serverDID),
		resolver,
		&ServerOpts{
			MaxSkew: 2 * time.Minute,
			Info:    DefaultInfoBuilder{},
			KEM:     serverKEMKP, // Server KEM keypair for HPKE
		},
	)

	// Create MockTransport with bidirectional communication
	mockTransport := &transport.MockTransport{
		SendFunc: func(ctx context.Context, msg *transport.SecureMessage) (*transport.Response, error) {
			// This simulates the server receiving the message via transport
			// The server processes HPKE messages through its HandleMessage method
			return serverHPKE.HandleMessage(ctx, msg)
		},
	}

	// Create HPKE client with MockTransport
	clientHPKE := NewClient(
		mockTransport,
		resolver,
		clientKP,
		string(clientDID),
		DefaultInfoBuilder{},
		clientSessMgr,
	)

	t.Run("Scenario_01_Valid_Handshake", func(t *testing.T) {
		// Initialize HPKE session
		ctxID := "ctx-" + uuid.NewString()
		kid, err := clientHPKE.Initialize(ctx, ctxID, string(clientDID), string(serverDID))
		require.NoError(t, err, "HPKE Initialize should succeed")
		require.NotEmpty(t, kid, "Key ID should be generated")

		// Verify client session was created
		clientSess, ok := clientSessMgr.GetByKeyID(kid)
		require.True(t, ok, "Client session should exist")
		require.NotNil(t, clientSess, "Client session should not be nil")

		// Encrypt a test message
		plaintext := []byte(`{"op":"ping","ts":1}`)
		ciphertext, err := clientSess.Encrypt(plaintext)
		require.NoError(t, err, "Encryption should succeed")
		require.NotEmpty(t, ciphertext, "Ciphertext should not be empty")

		// Verify server session exists
		serverSess, ok := serverSessMgr.GetByKeyID(kid)
		require.True(t, ok, "Server session should exist")
		require.NotNil(t, serverSess, "Server session should not be nil")

		// Decrypt on server side
		decrypted, err := serverSess.Decrypt(ciphertext)
		require.NoError(t, err, "Decryption should succeed")
		require.Equal(t, plaintext, decrypted, "Decrypted text should match original")

		t.Log("✅ Valid handshake completed successfully")
	})

	t.Run("Scenario_02_Message_Encryption_Decryption", func(t *testing.T) {
		// Use session from previous test
		ctxID := "ctx-" + uuid.NewString()
		kid, err := clientHPKE.Initialize(ctx, ctxID, string(clientDID), string(serverDID))
		require.NoError(t, err)

		clientSess, _ := clientSessMgr.GetByKeyID(kid)
		serverSess, _ := serverSessMgr.GetByKeyID(kid)

		// Test multiple messages
		messages := [][]byte{
			[]byte(`{"op":"ping","ts":1}`),
			[]byte(`{"op":"echo","data":"hello"}`),
			[]byte(`{"op":"status","code":200}`),
		}

		for i, msg := range messages {
			// Encrypt
			ciphertext, err := clientSess.Encrypt(msg)
			require.NoError(t, err, "Message %d encryption failed", i)

			// Decrypt
			decrypted, err := serverSess.Decrypt(ciphertext)
			require.NoError(t, err, "Message %d decryption failed", i)
			require.Equal(t, msg, decrypted, "Message %d content mismatch", i)
		}

		t.Log("✅ Multiple message encryption/decryption succeeded")
	})

	t.Run("Scenario_03_Invalid_Ciphertext", func(t *testing.T) {
		ctxID := "ctx-" + uuid.NewString()
		kid, err := clientHPKE.Initialize(ctx, ctxID, string(clientDID), string(serverDID))
		require.NoError(t, err)

		serverSess, _ := serverSessMgr.GetByKeyID(kid)

		// Try to decrypt invalid/tampered ciphertext
		invalidCipher := []byte("this is not valid ciphertext")
		_, err = serverSess.Decrypt(invalidCipher)
		require.Error(t, err, "Decryption of invalid ciphertext should fail")

		t.Log("✅ Invalid ciphertext correctly rejected")
	})

	t.Run("Scenario_04_Session_Isolation", func(t *testing.T) {
		// Create two independent sessions
		ctxID1 := "ctx-" + uuid.NewString()
		kid1, err := clientHPKE.Initialize(ctx, ctxID1, string(clientDID), string(serverDID))
		require.NoError(t, err)

		ctxID2 := "ctx-" + uuid.NewString()
		kid2, err := clientHPKE.Initialize(ctx, ctxID2, string(clientDID), string(serverDID))
		require.NoError(t, err)

		require.NotEqual(t, kid1, kid2, "Session key IDs should be different")

		// Get sessions
		sess1, _ := clientSessMgr.GetByKeyID(kid1)
		sess2, _ := clientSessMgr.GetByKeyID(kid2)

		// Encrypt same message with different sessions
		msg := []byte(`{"op":"test"}`)
		cipher1, _ := sess1.Encrypt(msg)
		cipher2, _ := sess2.Encrypt(msg)

		// Ciphertexts should be different (different sessions, different keys)
		require.NotEqual(t, cipher1, cipher2, "Ciphertexts from different sessions should differ")

		t.Log("✅ Session isolation verified")
	})
}

// e2eResolver implements sagedid.Resolver for E2E testing
type e2eResolver struct {
	dids map[sagedid.AgentDID]*sagedid.AgentMetadata
}

func (r *e2eResolver) Resolve(ctx context.Context, did sagedid.AgentDID) (*sagedid.AgentMetadata, error) {
	meta, ok := r.dids[did]
	if !ok {
		return nil, sagedid.ErrDIDNotFound
	}
	return meta, nil
}

func (r *e2eResolver) ResolvePublicKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	meta, err := r.Resolve(ctx, did)
	if err != nil {
		return nil, err
	}
	return meta.PublicKey, nil
}

func (r *e2eResolver) ResolveKEMKey(ctx context.Context, did sagedid.AgentDID) (interface{}, error) {
	meta, err := r.Resolve(ctx, did)
	if err != nil {
		return nil, err
	}
	return meta.PublicKEMKey, nil
}

func (r *e2eResolver) VerifyMetadata(ctx context.Context, did sagedid.AgentDID, meta *sagedid.AgentMetadata) (*sagedid.VerificationResult, error) {
	return &sagedid.VerificationResult{
		Valid:      true,
		VerifiedAt: time.Now(),
	}, nil
}

func (r *e2eResolver) ListAgentsByOwner(ctx context.Context, owner string) ([]*sagedid.AgentMetadata, error) {
	return nil, nil
}

func (r *e2eResolver) Search(ctx context.Context, criteria sagedid.SearchCriteria) ([]*sagedid.AgentMetadata, error) {
	return nil, nil
}
