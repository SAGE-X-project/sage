// Copyright (C) 2025 sage-x-project
// SPDX-License-Identifier: LGPL-3.0-or-later

// filename: hpke_shared_secret_session_test.go
package hpke

import (
	"bytes"
	"testing"

	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/session"
	"github.com/test-go/testify/require"
)

func Test_HPKE_Base_Exporter_To_Session(t *testing.T) {
	// Receiver's static KEM key (X25519)
	bobKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	// Canonical HPKE context (MUST match on both sides)
	info := []byte("sage/hpke-handshake v1|ctx:ctx-001|init:did:alice|resp:did:bob")
	exportCtx := []byte("sage/session exporter v1")
	exportLen := 32

	// ---- Sender (Alice): derive enc + exporter
	enc, expA, err := keys.HPKEDeriveSharedSecretToPeer(
		bobKeyPair.PublicKey(), info, exportCtx, exportLen,
	)
	require.NoError(t, err)
	require.Equal(t, 32, len(enc))
	require.Equal(t, 32, len(expA))

	// ---- Receiver (Bob): open with skR and derive exporter
	expB, err := keys.HPKEOpenSharedSecretWithPriv(
		bobKeyPair.PrivateKey(), enc, info, exportCtx, exportLen,
	)
	require.NoError(t, err)
	require.True(t, bytes.Equal(expA, expB), "exporter mismatch")

	// ---- Derive Session IDs deterministically from exporter
	sidA, err := session.ComputeSessionIDFromSeed(expA, "sage/hpke v1")
	require.NoError(t, err)
	sidB, err := session.ComputeSessionIDFromSeed(expB, "sage/hpke v1")
	require.NoError(t, err)
	require.Equal(t, sidA, sidB, "session id mismatch")

	// ---- Construct sessions from exporter (same policy on both ends)
	sA, err := session.NewSecureSessionFromExporter(sidA, expA, session.Config{})
	require.NoError(t, err)
	sB, err := session.NewSecureSessionFromExporter(sidB, expB, session.Config{})
	require.NoError(t, err)

	// ---- AEAD encrypt/decrypt
	msg := []byte("hello, secure world")
	ct, err := sA.Encrypt(msg)
	require.NoError(t, err)
	pt, err := sB.Decrypt(ct)
	require.NoError(t, err)
	require.True(t, bytes.Equal(pt, msg), "plaintext mismatch")

	// ---- Covered-signature (RFC 9421 style header binding)
	covered := []byte("@method:POST\n@path:/protected\nhost:example.org\ndate:Mon, 01 Jan 2024 00:00:00 GMT\ncontent-digest:sha-256=:...:\n")
	sig := sA.SignCovered(covered)
	require.NoError(t, sB.VerifyCovered(covered, sig))
}
