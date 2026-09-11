package keys

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyID_MatchesKeyPairIDs(t *testing.T) {
	ed, err := GenerateEd25519KeyPair()
	require.NoError(t, err)
	pub := ed.PublicKey().(ed25519.PublicKey)
	sum := sha256.Sum256(pub)
	assert.Equal(t, hex.EncodeToString(sum[:8]), KeyID(pub))
	assert.Equal(t, KeyID(pub), ed.ID(), "key pairs report KeyID of their public key bytes")
	assert.Len(t, KeyID(pub), 16)
}

func TestEthereumAddress(t *testing.T) {
	kp, err := GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	pub := kp.PublicKey().(*ecdsa.PublicKey)
	addr, err := EthereumAddress(pub)
	require.NoError(t, err)
	assert.Equal(t, strings.ToLower(ethcrypto.PubkeyToAddress(*pub).Hex()), addr)
	assert.True(t, strings.HasPrefix(addr, "0x"))
	assert.Len(t, addr, 42)

	p256, err := GenerateP256KeyPair()
	require.NoError(t, err)
	_, err = EthereumAddress(p256.PublicKey().(*ecdsa.PublicKey))
	require.Error(t, err, "only secp256k1 keys have Ethereum addresses")
	_, err = EthereumAddress(nil)
	require.Error(t, err)
}
