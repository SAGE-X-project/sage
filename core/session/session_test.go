package session

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/chacha20poly1305"
)

func TestSecureSessionLifecycle(t *testing.T) {
    config := Config{
        MaxAge:      100 * time.Millisecond,
        IdleTimeout: 50 * time.Millisecond,
        MaxMessages: 2,
    }
    sharedSecret := make([]byte, chacha20poly1305.KeySize)
    _, err := rand.Read(sharedSecret)
    require.NoError(t, err)

	sess, err := NewSecureSession("sess1", sharedSecret, config)
	require.NoError(t, err)
	t.Run("Encrypt and decrypt with sign roundtrip", func(t *testing.T) {
        require.NoError(t, err)
        require.Equal(t, "sess1", sess.GetID())
        require.False(t, sess.IsExpired())

        plaintext := []byte("hello")
        ct, err := sess.EncryptAndSign(plaintext)
        require.NoError(t, err)
        pt, err := sess.DecryptAndVerify(ct)
        require.NoError(t, err)
        require.Equal(t, plaintext, pt)

        require.Equal(t, 2, sess.GetMessageCount())
    })

    t.Run("Encrypt and decrypt roundtrip", func(t *testing.T) {
        plaintext := []byte("test payload")
        ct, err := sess.Encrypt(plaintext)
        require.NoError(t, err)
        require.NotEqual(t, plaintext, ct)

        pt, err := sess.Decrypt(ct)
        require.NoError(t, err)
        require.Equal(t, plaintext, pt)

		require.Equal(t, 4, sess.GetMessageCount())
    })

    t.Run("Decrypt with tampered data fails", func(t *testing.T) {
        plaintext := []byte("another test")
        ct, err := sess.Encrypt(plaintext)
        require.NoError(t, err)

        // Tamper one byte in ciphertext
        ct[ len(ct)/2 ] ^= 0xFF

        _, err = sess.Decrypt(ct)
        require.Error(t, err)
    })

    t.Run("Decrypt with too-short data fails", func(t *testing.T) {
        // Data shorter than nonce size
        _, err := sess.Decrypt([]byte("short"))
        require.Error(t, err)
    })

    t.Run("Message count expiration", func(t *testing.T) {
        sess, _ := NewSecureSession("sess2", sharedSecret, config)

        _, _ = sess.EncryptAndSign([]byte("m1"))
        _, _ = sess.EncryptAndSign([]byte("m2"))

        _, err := sess.EncryptAndSign([]byte("m3"))
        require.Error(t, err)
        require.True(t, sess.IsExpired())
    })

    t.Run("Idle timeout expiration", func(t *testing.T) {
        sess, _ := NewSecureSession("sess3", sharedSecret, config)

        _, _ = sess.EncryptAndSign([]byte("hi"))
        time.Sleep(config.IdleTimeout + 10*time.Millisecond)

        _, err := sess.EncryptAndSign([]byte("hi2"))
        require.Error(t, err)
        require.True(t, sess.IsExpired())
    })

    t.Run("Absolute timeout expiration", func(t *testing.T) {
        sess, _ := NewSecureSession("sess4", sharedSecret, config)
        time.Sleep(config.MaxAge + 10*time.Millisecond)
        _, err := sess.EncryptAndSign([]byte("late"))
        require.Error(t, err)
        require.True(t, sess.IsExpired())
    })

    t.Run("Close zeroizes keys", func(t *testing.T) {
        sess, _ := NewSecureSession("sess5", sharedSecret, config)
        _ = sess.Close()

        _, err := sess.EncryptAndSign([]byte("hi"))
        require.Error(t, err)
    })

	
}
