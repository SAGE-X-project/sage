package keys

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestX25519KeyPair(t *testing.T) {
	t.Run("GenerateKeyPair", func(t *testing.T) {
		keyPair, err := GenerateX25519KeyPair()
		require.NoError(t, err)
		assert.NotNil(t, keyPair)
		assert.NotNil(t, keyPair.PublicKey())
		assert.NotNil(t, keyPair.PrivateKey())
	})

	t.Run("DeriveSharedSecret", func(t *testing.T) {
		a, err := GenerateX25519KeyPair()
		require.NoError(t, err)
		b, err := GenerateX25519KeyPair()
		require.NoError(t, err)

		aKey, ok := a.(*X25519KeyPair)
		require.True(t, ok)
		bKey, ok := b.(*X25519KeyPair)
		require.True(t, ok)

		s1, err := aKey.DeriveSharedSecret(bKey.PublicBytesKey())
		require.NoError(t, err)
		s2, err := bKey.DeriveSharedSecret(aKey.PublicBytesKey())
		require.NoError(t, err)

		assert.Equal(t, s1, s2)
	})

	t.Run("EphemeralEncryptAndDecrypt", func(t *testing.T) {
		sender, err := GenerateX25519KeyPair()
		require.NoError(t, err)
		receiver, err := GenerateX25519KeyPair()
		require.NoError(t, err)

		senderKey, ok := sender.(*X25519KeyPair)
		require.True(t, ok)
		receiverKey, ok := receiver.(*X25519KeyPair)
		require.True(t, ok)

		plaintext := []byte("hello X25519 world")
		nonce, ct, err := senderKey.Encrypt(receiverKey.PublicBytesKey(), plaintext)
		require.NoError(t, err)
		require.NotEmpty(t, nonce)
		require.NotEmpty(t, ct)

		pt, err := receiverKey.DecryptWithX25519(senderKey.PublicBytesKey(), nonce, ct)
		require.NoError(t, err)
		assert.Equal(t, plaintext, pt)

		wrong, err := GenerateX25519KeyPair()
		wrongKey, ok := wrong.(*X25519KeyPair)
		require.True(t, ok)
		require.NoError(t, err)
		_, err = wrongKey.DecryptWithX25519(receiverKey.PublicBytesKey(), nonce, ct)
		assert.Error(t, err)
	})
	
	t.Run("ConvertEd25519ToX25519", func(t *testing.T) {
		keyPair, err := GenerateEd25519KeyPair()
		require.NoError(t, err)

		xPriv, err := convertEd25519PrivToX25519(keyPair.PrivateKey())
		require.NoError(t, err)
		assert.Len(t, xPriv, 32)

		xPub, err := convertEd25519PubToX25519(keyPair.PublicKey())
		require.NoError(t, err)
		assert.Len(t, xPub, 32)
	})

	t.Run("StaticEncryptAndDecryptToFromEd25519Peer", func(t *testing.T) {
		peerkeyPair, err := GenerateEd25519KeyPair()
		require.NoError(t, err)

		msg := []byte("static ed25519 messaging")
		packet, err := EncryptWithEd25519Peer(peerkeyPair.PublicKey(), msg)
		require.NoError(t, err)
		require.NotEmpty(t, packet)

		// Packet format: [32-byte ephPub || NONCE_SIZE-byte nonce || ciphertext]
		ephPub := packet[:32]
		nonce := packet[32 : 32+12]
		ct := packet[32+12:]

		assert.Len(t, ephPub, 32, "ephemeral public key must be 32 bytes")
		assert.Len(t, nonce, 12, "nonce must be 12 bytes for AES-GCM")
		assert.NotEmpty(t, ct, "ciphertext must not be empty for non-empty message")

		// correct decryption
		pt, err := DecryptWithEd25519Peer(peerkeyPair.PrivateKey(), packet)
		require.NoError(t, err)
		assert.Equal(t, msg, pt)

		// tamper ephemeral public key (first byte)
		bad := make([]byte, len(packet))
		copy(bad, packet)
		bad[0] ^= 0xFF
		_, err = DecryptWithEd25519Peer(peerkeyPair.PrivateKey(), bad)
		assert.Error(t, err, "tampered packet should fail")

		// too-short packet
		short := []byte{1,2,3}
		_, err = DecryptWithEd25519Peer(peerkeyPair.PrivateKey(), short)
		assert.Error(t, err, "short packet should error")
	})
}
