package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/formats"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/crypto/storage"
)

func TestLoadKeyPair_JWKPlainAndWrapper(t *testing.T) {
	kp, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	jwk, err := formats.NewJWKExporter().Export(kp, sagecrypto.KeyFormatJWK)
	require.NoError(t, err)

	dir := t.TempDir()
	plain := filepath.Join(dir, "plain.jwk")
	require.NoError(t, os.WriteFile(plain, jwk, 0o600))

	wrapped := filepath.Join(dir, "wrapped.json")
	w, err := json.Marshal(map[string]any{"private_key": json.RawMessage(jwk), "key_id": kp.ID(), "key_type": string(kp.Type())})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(wrapped, w, 0o600))

	for _, p := range []string{plain, wrapped} {
		got, err := LoadKeyPair(p, "jwk", "", "")
		require.NoError(t, err, p)
		require.Equal(t, kp.PublicKey(), got.PublicKey(), "loaded key must be the stored key, not a fresh one")
	}
}

func TestLoadKeyPair_PEM(t *testing.T) {
	kp, err := keys.GenerateEd25519KeyPair()
	require.NoError(t, err)
	pem, err := formats.NewPEMExporter().Export(kp, sagecrypto.KeyFormatPEM)
	require.NoError(t, err)
	p := filepath.Join(t.TempDir(), "key.pem")
	require.NoError(t, os.WriteFile(p, pem, 0o600))

	got, err := LoadKeyPair(p, "pem", "", "")
	require.NoError(t, err)
	require.Equal(t, kp.PublicKey(), got.PublicKey())
}

func TestLoadKeyPair_Storage(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewFileKeyStorage(dir)
	require.NoError(t, err)
	kp, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	require.NoError(t, store.Store("k1", kp))

	got, err := LoadKeyPair("", "", dir, "k1")
	require.NoError(t, err)
	require.Equal(t, kp.ID(), got.ID())
}

func TestLoadKeyPair_Errors(t *testing.T) {
	_, err := LoadKeyPair("", "jwk", "", "")
	require.Error(t, err)
	_, err = LoadKeyPair("/nonexistent", "xml", "", "")
	require.Error(t, err)
}
