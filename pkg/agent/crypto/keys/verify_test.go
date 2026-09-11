package keys

import (
	"crypto/ecdsa"
	"encoding/asn1"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func derEncode(t *testing.T, raw []byte) []byte {
	t.Helper()
	half := len(raw) / 2
	der, err := asn1.Marshal(struct{ R, S *big.Int }{new(big.Int).SetBytes(raw[:half]), new(big.Int).SetBytes(raw[half:])})
	require.NoError(t, err)
	return der
}

func TestVerifySignature_AllKeyTypes(t *testing.T) {
	msg := []byte("verify me")

	ed, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	sig, err := ed.Sign(msg)
	require.NoError(t, err)
	require.NoError(t, VerifySignature(ed.PublicKey(), msg, sig))
	require.Error(t, VerifySignature(ed.PublicKey(), []byte("other"), sig))

	k1, err := GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	sig, err = k1.Sign(msg) // 65 bytes r||s||v, Keccak-256
	require.NoError(t, err)
	require.NoError(t, VerifySignature(k1.PublicKey(), msg, sig))
	require.NoError(t, VerifySignature(k1.PublicKey(), msg, sig[:64]), "64-byte r||s accepted")
	require.NoError(t, VerifySignature(k1.PublicKey(), msg, derEncode(t, sig[:64])), "DER accepted")
	require.Error(t, VerifySignature(k1.PublicKey(), []byte("other"), sig))

	p256, err := GenerateP256KeyPair()
	require.NoError(t, err)
	sig, err = p256.Sign(msg) // 64 bytes, SHA-256
	require.NoError(t, err)
	require.NoError(t, VerifySignature(p256.PublicKey(), msg, sig))
	require.NoError(t, VerifySignature(p256.PublicKey(), msg, derEncode(t, sig)), "DER accepted")
	require.Error(t, VerifySignature(p256.PublicKey(), []byte("other"), sig))

	require.Error(t, VerifySignature("not a key", msg, sig))
	require.Error(t, VerifySignature((*ecdsa.PublicKey)(nil), msg, sig))
}
