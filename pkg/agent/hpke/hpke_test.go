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
	"encoding/hex"
	"testing"

	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/session"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
)

func Test_HPKE_Base_Exporter_To_Session(t *testing.T) {
	// Specification Requirement: HPKE-based secure session establishment with key derivation
	helpers.LogTestSection(t, "6.1.1", "HPKE Key Exchange and Session Derivation")

	// Specification Requirement: X25519 key generation for KEM (Key Encapsulation Mechanism)
	bobKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)
	helpers.LogSuccess(t, "Receiver (Bob) X25519 key pair generated")
	helpers.LogDetail(t, "Key type: X25519 (Curve25519)")

	// Canonical HPKE context (MUST match on both sides)
	info := []byte("sage/hpke-handshake v1|ctx:ctx-001|init:did:alice|resp:did:bob")
	exportCtx := []byte("sage/session exporter v1")
	exportLen := 32

	helpers.LogDetail(t, "HPKE info context: %s", string(info))
	helpers.LogDetail(t, "Export context: %s", string(exportCtx))
	helpers.LogDetail(t, "Export length: %d bytes", exportLen)

	// Specification Requirement: Sender (Alice) derives encapsulated key and shared secret
	enc, expA, err := keys.HPKEDeriveSharedSecretToPeer(
		bobKeyPair.PublicKey(), info, exportCtx, exportLen,
	)
	require.NoError(t, err)

	// Specification Requirement: Encapsulated key size must be 32 bytes (X25519 public key)
	require.Equal(t, 32, len(enc))
	assert.Equal(t, 32, len(expA))

	helpers.LogSuccess(t, "Sender (Alice) HPKE key derivation successful")
	helpers.LogDetail(t, "Encapsulated key size: %d bytes (expected: 32)", len(enc))
	helpers.LogDetail(t, "Exporter secret size: %d bytes (expected: 32)", len(expA))
	helpers.LogDetail(t, "Encapsulated key (hex): %s", hex.EncodeToString(enc)[:32]+"...")

	// Specification Requirement: Receiver (Bob) opens with private key and derives shared secret
	expB, err := keys.HPKEOpenSharedSecretWithPriv(
		bobKeyPair.PrivateKey(), enc, info, exportCtx, exportLen,
	)
	require.NoError(t, err)

	// Specification Requirement: Both parties must derive identical shared secret
	require.True(t, bytes.Equal(expA, expB), "exporter mismatch")
	helpers.LogSuccess(t, "Receiver (Bob) HPKE key opening successful")
	helpers.LogDetail(t, "Shared secret match: %v", bytes.Equal(expA, expB))
	helpers.LogDetail(t, "Exporter A (hex): %s", hex.EncodeToString(expA)[:32]+"...")
	helpers.LogDetail(t, "Exporter B (hex): %s", hex.EncodeToString(expB)[:32]+"...")

	// Specification Requirement: Deterministic session ID derivation from shared secret
	sidA, err := session.ComputeSessionIDFromSeed(expA, "sage/hpke v1")
	require.NoError(t, err)
	sidB, err := session.ComputeSessionIDFromSeed(expB, "sage/hpke v1")
	require.NoError(t, err)
	require.Equal(t, sidA, sidB, "session id mismatch")

	helpers.LogSuccess(t, "Session IDs derived deterministically")
	helpers.LogDetail(t, "Session ID (Alice): %s", sidA)
	helpers.LogDetail(t, "Session ID (Bob): %s", sidB)
	helpers.LogDetail(t, "Session IDs match: %v", sidA == sidB)

	// Specification Requirement: Secure session construction from exporter
	sA, err := session.NewSecureSessionFromExporter(sidA, expA, session.Config{})
	require.NoError(t, err)
	sB, err := session.NewSecureSessionFromExporter(sidB, expB, session.Config{})
	require.NoError(t, err)

	helpers.LogSuccess(t, "Secure sessions established from HPKE exporter")
	helpers.LogDetail(t, "Alice session created with ID: %s", sidA)
	helpers.LogDetail(t, "Bob session created with ID: %s", sidB)

	// Specification Requirement: AEAD encryption/decryption with derived session keys
	msg := []byte("hello, secure world")
	helpers.LogDetail(t, "Test message: %s", string(msg))
	helpers.LogDetail(t, "Message size: %d bytes", len(msg))

	ct, err := sA.Encrypt(msg)
	require.NoError(t, err)
	helpers.LogSuccess(t, "Alice encrypted message")
	helpers.LogDetail(t, "Ciphertext size: %d bytes", len(ct))
	helpers.LogDetail(t, "Ciphertext (hex): %s", hex.EncodeToString(ct)[:64]+"...")

	pt, err := sB.Decrypt(ct)
	require.NoError(t, err)
	require.True(t, bytes.Equal(pt, msg), "plaintext mismatch")
	helpers.LogSuccess(t, "Bob decrypted message successfully")
	helpers.LogDetail(t, "Decrypted message: %s", string(pt))
	helpers.LogDetail(t, "Plaintext matches original: %v", bytes.Equal(pt, msg))

	// Specification Requirement: RFC 9421 style covered signature
	covered := []byte("@method:POST\n@path:/protected\nhost:example.org\ndate:Mon, 01 Jan 2024 00:00:00 GMT\ncontent-digest:sha-256=:...:\n")
	helpers.LogDetail(t, "Covered content size: %d bytes", len(covered))

	sig := sA.SignCovered(covered)
	helpers.LogSuccess(t, "Alice signed covered content")
	helpers.LogDetail(t, "Signature size: %d bytes", len(sig))

	require.NoError(t, sB.VerifyCovered(covered, sig))
	helpers.LogSuccess(t, "Bob verified covered signature")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"X25519 key pair generation successful",
		"HPKE key derivation (Alice) successful",
		"Encapsulated key = 32 bytes",
		"Shared secret = 32 bytes",
		"HPKE key opening (Bob) successful",
		"Shared secrets match between parties",
		"Session IDs derived deterministically",
		"Session IDs match between parties",
		"Secure sessions established",
		"AEAD encryption successful",
		"AEAD decryption successful",
		"Plaintext matches original",
		"Covered signature generation successful",
		"Covered signature verification successful",
	})

	// Save test data for CLI verification
	testData := map[string]interface{}{
		"test_case": "6.1.1_HPKE_Key_Exchange_Session",
		"key_type": "X25519",
		"hpke": map[string]interface{}{
			"info_context":     string(info),
			"export_context":   string(exportCtx),
			"export_length":    exportLen,
			"enc_size":         len(enc),
			"exporter_a_hex":   hex.EncodeToString(expA),
			"exporter_b_hex":   hex.EncodeToString(expB),
			"secrets_match":    bytes.Equal(expA, expB),
		},
		"session": map[string]interface{}{
			"session_id_a":     sidA,
			"session_id_b":     sidB,
			"ids_match":        sidA == sidB,
		},
		"encryption": map[string]interface{}{
			"message":          string(msg),
			"message_size":     len(msg),
			"ciphertext_size":  len(ct),
			"decrypted":        string(pt),
			"plaintext_match":  bytes.Equal(pt, msg),
		},
		"signature": map[string]interface{}{
			"covered_size":     len(covered),
			"signature_size":   len(sig),
			"verification":     "successful",
		},
	}
	helpers.SaveTestData(t, "hpke/hpke_key_exchange_session.json", testData)
}
