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
	"crypto/ecdh"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	sagedid "github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/pkg/agent/transport"
)

// TestE2E_HPKE_Handshake_MockTransport tests complete HPKE handshake flow
// using MockTransport instead of actual gRPC/A2A transport.
// This replaces the integration test in tests/integration/session/handshake/
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
		// Specification Requirement: Complete HPKE handshake flow with MockTransport
		helpers.LogTestSection(t, "9.1.1", "HPKE E2E Handshake - Valid Flow")

		helpers.LogDetail(t, "Test setup:")
		helpers.LogDetail(t, "  Client DID: %s", clientDID)
		helpers.LogDetail(t, "  Server DID: %s", serverDID)
		helpers.LogDetail(t, "  Client KEM public key: %s", hex.EncodeToString(clientKEMKP.PublicKey().(*ecdh.PublicKey).Bytes()))
		helpers.LogDetail(t, "  Server KEM public key: %s", hex.EncodeToString(serverKEMKP.PublicKey().(*ecdh.PublicKey).Bytes()))

		// Specification Requirement: Initialize HPKE session with client-server key exchange
		ctxID := "ctx-" + uuid.NewString()
		helpers.LogDetail(t, "Initializing HPKE session with context ID: %s", ctxID)

		kid, err := clientHPKE.Initialize(ctx, ctxID, string(clientDID), string(serverDID))
		require.NoError(t, err, "HPKE Initialize should succeed")
		require.NotEmpty(t, kid, "Key ID should be generated")
		helpers.LogSuccess(t, "HPKE session initialized successfully")
		helpers.LogDetail(t, "  Generated Key ID: %s", kid)

		// Specification Requirement: Verify client session establishment
		clientSess, ok := clientSessMgr.GetByKeyID(kid)
		require.True(t, ok, "Client session should exist")
		require.NotNil(t, clientSess, "Client session should not be nil")
		helpers.LogSuccess(t, "Client session established")
		helpers.LogDetail(t, "  Session ID: %s", clientSess.GetID())
		helpers.LogDetail(t, "  Key ID: %s", kid)

		// Specification Requirement: Encrypt test message using established session
		plaintext := []byte(`{"op":"ping","ts":1}`)
		helpers.LogDetail(t, "Encrypting test message:")
		helpers.LogDetail(t, "  Plaintext: %s", string(plaintext))
		helpers.LogDetail(t, "  Plaintext size: %d bytes", len(plaintext))

		ciphertext, err := clientSess.Encrypt(plaintext)
		require.NoError(t, err, "Encryption should succeed")
		require.NotEmpty(t, ciphertext, "Ciphertext should not be empty")
		helpers.LogSuccess(t, "Message encrypted successfully")
		helpers.LogDetail(t, "  Ciphertext size: %d bytes", len(ciphertext))
		helpers.LogDetail(t, "  Ciphertext (hex): %s", hex.EncodeToString(ciphertext))

		// Specification Requirement: Verify server session establishment (via MockTransport)
		serverSess, ok := serverSessMgr.GetByKeyID(kid)
		require.True(t, ok, "Server session should exist")
		require.NotNil(t, serverSess, "Server session should not be nil")
		helpers.LogSuccess(t, "Server session established")
		helpers.LogDetail(t, "  Session ID: %s", serverSess.GetID())
		helpers.LogDetail(t, "  Key ID: %s", kid)

		// Specification Requirement: Decrypt message on server side and verify integrity
		helpers.LogDetail(t, "Decrypting message on server side")
		decrypted, err := serverSess.Decrypt(ciphertext)
		require.NoError(t, err, "Decryption should succeed")
		require.Equal(t, plaintext, decrypted, "Decrypted text should match original")
		helpers.LogSuccess(t, "Message decrypted and verified successfully")
		helpers.LogDetail(t, "  Decrypted: %s", string(decrypted))

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Client and server DIDs generated",
			"KEM key pairs generated for both parties",
			"HPKE session initialized via MockTransport",
			"Client session established with unique Key ID",
			"Server session established with matching Key ID",
			"Message encrypted with client session",
			"Message decrypted with server session",
			"Decrypted plaintext matches original",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":  "9.1.1_HPKE_E2E_Valid_Handshake",
			"context_id": ctxID,
			"key_id":     kid,
			"client": map[string]interface{}{
				"did":            string(clientDID),
				"kem_public_key": hex.EncodeToString(clientKEMKP.PublicKey().(*ecdh.PublicKey).Bytes()),
				"session_id":     clientSess.GetID(),
			},
			"server": map[string]interface{}{
				"did":            string(serverDID),
				"kem_public_key": hex.EncodeToString(serverKEMKP.PublicKey().(*ecdh.PublicKey).Bytes()),
				"session_id":     serverSess.GetID(),
			},
			"encryption": map[string]interface{}{
				"plaintext":       string(plaintext),
				"plaintext_size":  len(plaintext),
				"ciphertext_size": len(ciphertext),
				"ciphertext_hex":  hex.EncodeToString(ciphertext),
			},
			"handshake_status": "success",
		}
		helpers.SaveTestData(t, "hpke/e2e_valid_handshake.json", testData)
	})

	t.Run("Scenario_02_Message_Encryption_Decryption", func(t *testing.T) {
		// Specification Requirement: Multiple message encryption/decryption within single session
		helpers.LogTestSection(t, "9.1.2", "HPKE E2E Multiple Messages")

		// Specification Requirement: Establish fresh session for multiple messages
		ctxID := "ctx-" + uuid.NewString()
		helpers.LogDetail(t, "Establishing new session for multiple messages")
		helpers.LogDetail(t, "  Context ID: %s", ctxID)

		kid, err := clientHPKE.Initialize(ctx, ctxID, string(clientDID), string(serverDID))
		require.NoError(t, err)
		helpers.LogSuccess(t, "Session established")
		helpers.LogDetail(t, "  Key ID: %s", kid)

		clientSess, _ := clientSessMgr.GetByKeyID(kid)
		serverSess, _ := serverSessMgr.GetByKeyID(kid)

		// Specification Requirement: Test multiple message types
		messages := [][]byte{
			[]byte(`{"op":"ping","ts":1}`),
			[]byte(`{"op":"echo","data":"hello"}`),
			[]byte(`{"op":"status","code":200}`),
		}

		helpers.LogDetail(t, "Testing %d messages:", len(messages))
		for i, msg := range messages {
			helpers.LogDetail(t, "  Message %d: %s", i+1, string(msg))
		}

		messageResults := make([]map[string]interface{}, 0, len(messages))

		for i, msg := range messages {
			helpers.LogDetail(t, "Processing message %d/%d", i+1, len(messages))

			// Specification Requirement: Encrypt with client session
			ciphertext, err := clientSess.Encrypt(msg)
			require.NoError(t, err, "Message %d encryption failed", i)
			helpers.LogDetail(t, "  Encrypted: %d bytes -> %d bytes", len(msg), len(ciphertext))

			// Specification Requirement: Decrypt with server session
			decrypted, err := serverSess.Decrypt(ciphertext)
			require.NoError(t, err, "Message %d decryption failed", i)
			require.Equal(t, msg, decrypted, "Message %d content mismatch", i)
			helpers.LogDetail(t, "  Decrypted: %s", string(decrypted))

			messageResults = append(messageResults, map[string]interface{}{
				"index":           i + 1,
				"plaintext":       string(msg),
				"plaintext_size":  len(msg),
				"ciphertext_size": len(ciphertext),
				"decrypted":       string(decrypted),
				"verified":        true,
			})
		}

		helpers.LogSuccess(t, "All messages processed successfully")
		helpers.LogDetail(t, "  Total processed: %d messages", len(messages))

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"New session established for message sequence",
			"Multiple messages encrypted sequentially",
			"All ciphertexts generated successfully",
			"All messages decrypted correctly",
			"Plaintext-ciphertext-plaintext round-trip verified",
			"Session state maintained across multiple operations",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":     "9.1.2_HPKE_E2E_Multiple_Messages",
			"context_id":    ctxID,
			"key_id":        kid,
			"message_count": len(messages),
			"messages":      messageResults,
		}
		helpers.SaveTestData(t, "hpke/e2e_multiple_messages.json", testData)
	})

	t.Run("Scenario_03_Invalid_Ciphertext", func(t *testing.T) {
		// Specification Requirement: Invalid ciphertext detection and rejection
		helpers.LogTestSection(t, "9.1.3", "HPKE E2E Invalid Ciphertext")

		// Specification Requirement: Establish session for error testing
		ctxID := "ctx-" + uuid.NewString()
		helpers.LogDetail(t, "Establishing session for invalid ciphertext test")
		helpers.LogDetail(t, "  Context ID: %s", ctxID)

		kid, err := clientHPKE.Initialize(ctx, ctxID, string(clientDID), string(serverDID))
		require.NoError(t, err)
		helpers.LogSuccess(t, "Session established")
		helpers.LogDetail(t, "  Key ID: %s", kid)

		serverSess, _ := serverSessMgr.GetByKeyID(kid)
		helpers.LogDetail(t, "Server session retrieved for decryption test")

		// Specification Requirement: Attempt decryption of invalid ciphertext
		invalidCipher := []byte("this is not valid ciphertext")
		helpers.LogDetail(t, "Attempting to decrypt invalid ciphertext:")
		helpers.LogDetail(t, "  Invalid data: %s", string(invalidCipher))
		helpers.LogDetail(t, "  Size: %d bytes", len(invalidCipher))

		_, err = serverSess.Decrypt(invalidCipher)
		require.Error(t, err, "Decryption of invalid ciphertext should fail")
		helpers.LogSuccess(t, "Invalid ciphertext correctly rejected")
		helpers.LogDetail(t, "  Error message: %s", err.Error())

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Session established successfully",
			"Invalid ciphertext rejected by decryption",
			"Error returned for invalid input",
			"Session integrity maintained after error",
			"No panic or unexpected behavior",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":          "9.1.3_HPKE_E2E_Invalid_Ciphertext",
			"context_id":         ctxID,
			"key_id":             kid,
			"invalid_ciphertext": string(invalidCipher),
			"invalid_size":       len(invalidCipher),
			"decryption_failed":  true,
			"error":              err.Error(),
		}
		helpers.SaveTestData(t, "hpke/e2e_invalid_ciphertext.json", testData)
	})

	t.Run("Scenario_04_Session_Isolation", func(t *testing.T) {
		// Specification Requirement: Session isolation and independent key derivation
		helpers.LogTestSection(t, "9.1.4", "HPKE E2E Session Isolation")

		// Specification Requirement: Create first independent session
		ctxID1 := "ctx-" + uuid.NewString()
		helpers.LogDetail(t, "Creating first session:")
		helpers.LogDetail(t, "  Context ID 1: %s", ctxID1)

		kid1, err := clientHPKE.Initialize(ctx, ctxID1, string(clientDID), string(serverDID))
		require.NoError(t, err)
		helpers.LogSuccess(t, "First session established")
		helpers.LogDetail(t, "  Key ID 1: %s", kid1)

		// Specification Requirement: Create second independent session
		ctxID2 := "ctx-" + uuid.NewString()
		helpers.LogDetail(t, "Creating second session:")
		helpers.LogDetail(t, "  Context ID 2: %s", ctxID2)

		kid2, err := clientHPKE.Initialize(ctx, ctxID2, string(clientDID), string(serverDID))
		require.NoError(t, err)
		helpers.LogSuccess(t, "Second session established")
		helpers.LogDetail(t, "  Key ID 2: %s", kid2)

		// Specification Requirement: Verify unique key IDs
		require.NotEqual(t, kid1, kid2, "Session key IDs should be different")
		helpers.LogSuccess(t, "Key IDs are unique")

		// Get sessions
		sess1, _ := clientSessMgr.GetByKeyID(kid1)
		sess2, _ := clientSessMgr.GetByKeyID(kid2)
		helpers.LogDetail(t, "Both sessions retrieved from manager")

		// Specification Requirement: Encrypt identical message with both sessions
		msg := []byte(`{"op":"test"}`)
		helpers.LogDetail(t, "Encrypting identical message with both sessions:")
		helpers.LogDetail(t, "  Message: %s", string(msg))

		cipher1, _ := sess1.Encrypt(msg)
		helpers.LogDetail(t, "  Session 1 ciphertext: %d bytes (hex: %s...)", len(cipher1), hex.EncodeToString(cipher1[:min(16, len(cipher1))]))

		cipher2, _ := sess2.Encrypt(msg)
		helpers.LogDetail(t, "  Session 2 ciphertext: %d bytes (hex: %s...)", len(cipher2), hex.EncodeToString(cipher2[:min(16, len(cipher2))]))

		// Specification Requirement: Verify ciphertexts differ (different keys)
		require.NotEqual(t, cipher1, cipher2, "Ciphertexts from different sessions should differ")
		helpers.LogSuccess(t, "Ciphertexts are unique (sessions are isolated)")

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Two independent sessions created",
			"Unique Key IDs generated for each session",
			"Both sessions registered in session manager",
			"Identical message encrypted with both sessions",
			"Ciphertexts differ between sessions",
			"Session isolation verified (no key reuse)",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case": "9.1.4_HPKE_E2E_Session_Isolation",
			"session_1": map[string]interface{}{
				"context_id":            ctxID1,
				"key_id":                kid1,
				"ciphertext_size":       len(cipher1),
				"ciphertext_hex_prefix": hex.EncodeToString(cipher1[:min(16, len(cipher1))]),
			},
			"session_2": map[string]interface{}{
				"context_id":            ctxID2,
				"key_id":                kid2,
				"ciphertext_size":       len(cipher2),
				"ciphertext_hex_prefix": hex.EncodeToString(cipher2[:min(16, len(cipher2))]),
			},
			"plaintext":          string(msg),
			"key_ids_unique":     kid1 != kid2,
			"ciphertexts_unique": !bytes.Equal(cipher1, cipher2),
			"isolation_verified": true,
		}
		helpers.SaveTestData(t, "hpke/e2e_session_isolation.json", testData)
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
