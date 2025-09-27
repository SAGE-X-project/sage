// filename: hpke_shared_secret_session_test.go
package hpke

import (
	"bytes"
	"testing"

	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/sage-x-project/sage/session"
	"github.com/test-go/testify/require"
)

func Test_HPKE(t *testing.T) {
	bobKeyPair, err := keys.GenerateX25519KeyPair()
	require.NoError(t, err)

	// HPKE context that both sides must derive identically.
	info := []byte("sage/hpke-handshake v1|ctx:ctx-001|init:did:alice|resp:did:bob")
	exportCtx := []byte("sage/session exporter v1")
	exportLen := 32

	// Alice
	enc, expA, err := keys.HPKEDeriveSharedSecretToPeer(bobKeyPair.PublicKey(), info, exportCtx, exportLen)
	if err != nil {
		t.Fatalf("HPKE sender derive: %v", err)
	}
	if len(enc) != 32 || len(expA) != 32 {
		t.Fatalf("unexpected sizes: enc=%d, exporter=%d", len(enc), len(expA))
	}

	// Receiver (server): open HPKE using enc and derive exporterSecretB, which must match Alice's.
	expB, err := keys.HPKEOpenSharedSecretWithPriv(bobKeyPair.PrivateKey(), enc, info, exportCtx, exportLen)
	if err != nil {
		t.Fatalf("HPKE receiver open: %v", err)
	}
	if !bytes.Equal(expA, expB) {
		t.Fatalf("exporter mismatch")
	}

	sidA, err := session.ComputeSessionIDFromSeed(expA, "sage/hpke v1")
	if err != nil {
		t.Fatalf("sidA: %v", err)
	}
	sidB, err := session.ComputeSessionIDFromSeed(expB, "sage/hpke v1")
	if err != nil {
		t.Fatalf("sidB: %v", err)
	}

	if sidA != sidB {
		t.Fatalf("session id mismatch: %s vs %s", sidA, sidB)
	}

	// session create
	sA, err := session.NewSecureSessionFromExporter(sidA, expA, session.Config{})
	if err != nil {
		t.Fatalf("sessA: %v", err)
	}
	sB, err := session.NewSecureSessionFromExporter(sidB, expB, session.Config{})
	if err != nil {
		t.Fatalf("sessB: %v", err)
	}

	// session encrypt/decrypt
	msg := []byte("hello, secure world")
	ct, err := sA.Encrypt(msg)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	pt, err := sB.Decrypt(ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(pt, msg) {
		t.Fatalf("plaintext mismatch")
	}

	covered := []byte("@method:POST\n@path:/protected\nhost:example.org\ndate:Mon, 01 Jan 2024 00:00:00 GMT\ncontent-digest:sha-256=:...:\n")
	sig := sA.SignCovered(covered)
	if err := sB.VerifyCovered(covered, sig); err != nil {
		t.Fatalf("verify covered failed: %v", err)
	}
}
