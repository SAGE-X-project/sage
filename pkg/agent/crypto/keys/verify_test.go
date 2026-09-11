package keys

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/asn1"
	"math/big"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
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

// Signatures made with crypto/ecdsa (random nonce, no low-S normalisation) or
// re-encoded with s' = N - s must still verify: (r, s) and (r, N-s) are both
// valid ECDSA signatures and go-ethereum only accepts the low form.
func TestVerifySignature_Secp256k1AcceptsHighS(t *testing.T) {
	kp, err := GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	priv := kp.PrivateKey().(*ecdsa.PrivateKey)
	msg := []byte("high-s tolerance")
	digest := ethcrypto.Keccak256(msg)

	for i := 0; i < 32; i++ {
		r, s, err := ecdsa.Sign(rand.Reader, priv, digest)
		require.NoError(t, err)
		// Force the high form explicitly as well.
		high := new(big.Int).Sub(priv.Curve.Params().N, LowS(priv.Curve, s))
		for _, sv := range []*big.Int{s, high} {
			raw := make([]byte, 64)
			r.FillBytes(raw[:32])
			sv.FillBytes(raw[32:])
			require.NoError(t, VerifySignature(kp.PublicKey(), msg, raw), "raw r||s")
			require.NoError(t, VerifySignature(kp.PublicKey(), msg, derEncode(t, raw)), "DER")
			require.NoError(t, VerifySecp256k1Keccak(priv.Public().(*ecdsa.PublicKey), msg, raw))
		}
	}
}
