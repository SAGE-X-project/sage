package ethereum

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

func secpKey(t *testing.T) []byte {
	t.Helper()
	k, err := ethcrypto.GenerateKey()
	require.NoError(t, err)
	return ethcrypto.FromECDSAPub(&k.PublicKey)
}

func edKey(t *testing.T) []byte {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	return pub
}

func x25519Key(t *testing.T) []byte {
	t.Helper()
	b := make([]byte, 32)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return b
}

// The on-chain registry marks revoked keys with verified=false and keeps their
// hashes, so the resolver must ignore any key that is not verified.
func TestSelectAgentKeys_SkipsUnverifiedAndRevokedKeys(t *testing.T) {
	revoked, good := secpKey(t), secpKey(t)
	pub, _, err := selectAgentKeys([]onChainKey{
		{KeyType: uint8(did.KeyTypeECDSA), KeyData: revoked, Verified: false},
		{KeyType: uint8(did.KeyTypeECDSA), KeyData: good, Verified: true},
	})
	require.NoError(t, err)
	require.Equal(t, good, ethcrypto.FromECDSAPub(pub.(*ecdsa.PublicKey)))
}

func TestSelectAgentKeys_NoVerifiedSigningKey(t *testing.T) {
	pub, kem, err := selectAgentKeys([]onChainKey{
		{KeyType: uint8(did.KeyTypeECDSA), KeyData: secpKey(t), Verified: false},
	})
	require.NoError(t, err)
	require.Nil(t, pub, "an unverified key must not be returned as the signing key")
	require.Nil(t, kem)
}

// Ed25519-only agents used to resolve to a nil public key; they must resolve
// to their Ed25519 key.
func TestSelectAgentKeys_Ed25519Fallback(t *testing.T) {
	ed := edKey(t)
	pub, _, err := selectAgentKeys([]onChainKey{
		{KeyType: uint8(did.KeyTypeEd25519), KeyData: ed, Verified: true},
	})
	require.NoError(t, err)
	require.Equal(t, ed25519.PublicKey(ed), pub)
}

// A verified ECDSA key takes precedence over a verified Ed25519 key on Ethereum.
func TestSelectAgentKeys_PrefersECDSA(t *testing.T) {
	ed, secp := edKey(t), secpKey(t)
	pub, _, err := selectAgentKeys([]onChainKey{
		{KeyType: uint8(did.KeyTypeEd25519), KeyData: ed, Verified: true},
		{KeyType: uint8(did.KeyTypeECDSA), KeyData: secp, Verified: true},
	})
	require.NoError(t, err)
	require.Equal(t, secp, ethcrypto.FromECDSAPub(pub.(*ecdsa.PublicKey)))
}

// A verified X25519 key registered through addKey is returned as the KEM key.
func TestSelectAgentKeys_KEMFromKeyList(t *testing.T) {
	kem := x25519Key(t)
	_, gotKEM, err := selectAgentKeys([]onChainKey{
		{KeyType: uint8(did.KeyTypeX25519), KeyData: kem, Verified: true},
		{KeyType: uint8(did.KeyTypeX25519), KeyData: x25519Key(t), Verified: false},
	})
	require.NoError(t, err)
	require.Equal(t, kem, gotKEM)
}

func TestSelectAgentKeys_MalformedVerifiedKeyIsAnError(t *testing.T) {
	_, _, err := selectAgentKeys([]onChainKey{
		{KeyType: uint8(did.KeyTypeECDSA), KeyData: []byte{1, 2, 3}, Verified: true},
	})
	require.Error(t, err)
}
