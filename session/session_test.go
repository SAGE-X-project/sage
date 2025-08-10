package session

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
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

func TestSecureSession_WithParamsSuite(t *testing.T) {
	t.Run("Deterministic seed/id/keys & cross-encrypt", func(t *testing.T) {
		sharedSecret := b(chacha20poly1305.KeySize)
		selfA, selfB := b(32), b(32)
		ctxID := "ctx-1234"
		label := "a2a/handshake v1"

		pA := Params{ContextID: ctxID, SelfEph: selfA, PeerEph: selfB, Label: label, SharedSecret: sharedSecret}
		pB := Params{ContextID: ctxID, SelfEph: selfB, PeerEph: selfA, Label: label, SharedSecret: sharedSecret}

		seedA, err := DeriveSessionSeed(sharedSecret, pA)
		require.NoError(t, err)
		seedB, err := DeriveSessionSeed(sharedSecret, pB)
		require.NoError(t, err)
		require.Equal(t, seedA, seedB)

		idA, err := ComputeSessionIDFromSeed(seedA, label)
		require.NoError(t, err)
		idB, err := ComputeSessionIDFromSeed(seedB, label)
		require.NoError(t, err)
		require.Equal(t, idA, idB)

		cfg := Config{MaxAge: time.Second, IdleTimeout: time.Second, MaxMessages: 100}
		sessA, err := NewSecureSession(idA, seedA, cfg)
		require.NoError(t, err)
		sessB, err := NewSecureSession(idB, seedB, cfg)
		require.NoError(t, err)

		require.Equal(t, sessA.encryptKey, sessB.encryptKey)
		require.Equal(t, sessA.signingKey, sessB.signingKey)

		// A → B
		msg1 := []byte("hello from A")
		ct1, err := sessA.Encrypt(msg1)
		require.NoError(t, err)
		pt1, err := sessB.Decrypt(ct1)
		require.NoError(t, err)
		require.Equal(t, msg1, pt1)

		// B → A
		msg2 := []byte("hello from B")
		ct2, err := sessB.Encrypt(msg2)
		require.NoError(t, err)
		pt2, err := sessA.Decrypt(ct2)
		require.NoError(t, err)
		require.Equal(t, msg2, pt2)
	})

	t.Run("Signing key HMAC verify (ok/tamper/different context or label)", func(t *testing.T) {
		shared := b(32)
		e1, e2 := b(32), b(32)

		s1, err := NewSecureSessionWithParams(shared, Params{ContextID: "ctx", SelfEph: e1, PeerEph: e2, Label: "v1"}, Config{})
		require.NoError(t, err)
		s2, err := NewSecureSessionWithParams(shared, Params{ContextID: "ctx", SelfEph: e2, PeerEph: e1, Label: "v1"}, Config{})
		require.NoError(t, err)

		msg := []byte("sign me")
		sig1 := hmacSHA256(s1.signingKey, msg)
		sig2 := hmacSHA256(s2.signingKey, msg)
		require.Equal(t, sig1, sig2)

		tampered := append([]byte{}, msg...)
		tampered[0] ^= 0xFF
		require.NotEqual(t, sig1, hmacSHA256(s2.signingKey, tampered))

		// 다른 컨텍스트 → 다른 키
		s3, err := NewSecureSessionWithParams(shared, Params{ContextID: "ctx-OTHER", SelfEph: e2, PeerEph: e1, Label: "v1"}, Config{})
		require.NoError(t, err)
		require.NotEqual(t, s1.signingKey, s3.signingKey)

		// 라벨만 달라도 다른 키
		s4, err := NewSecureSessionWithParams(shared, Params{ContextID: "ctx", SelfEph: e2, PeerEph: e1, Label: "v2"}, Config{})
		require.NoError(t, err)
		require.NotEqual(t, s1.signingKey, s4.signingKey)
	})

	t.Run("NewSecureSessionWithParams determinism & error cases", func(t *testing.T) {
		shared := b(32)
		eA, eB := b(32), b(32)

		// determinism
		sA, err := NewSecureSessionWithParams(shared, Params{ContextID: "C", SelfEph: eA, PeerEph: eB, Label: "L"}, Config{})
		require.NoError(t, err)
		sB, err := NewSecureSessionWithParams(shared, Params{ContextID: "C", SelfEph: eB, PeerEph: eA, Label: "L"}, Config{})
		require.NoError(t, err)
		require.Equal(t, sA.id, sB.id)
		require.Equal(t, sA.encryptKey, sB.encryptKey)
		require.Equal(t, sA.signingKey, sB.signingKey)

		// error paths
		_, err = DeriveSessionSeed(nil, Params{ContextID: "C", SelfEph: eA, PeerEph: eB})
		require.Error(t, err)
		_, err = DeriveSessionSeed(shared, Params{ContextID: "", SelfEph: eA, PeerEph: eB})
		require.Error(t, err)
		_, err = ComputeSessionIDFromSeed(nil, "L")
		require.Error(t, err)
	})

	t.Run("Decrypt fails when params differ", func(t *testing.T) {
		shared := b(32)
		e1, e2, e3 := b(32), b(32), b(32)

		sA, _ := NewSecureSessionWithParams(shared, Params{ContextID: "X", SelfEph: e1, PeerEph: e2, Label: "v1"}, Config{})
		sB, _ := NewSecureSessionWithParams(shared, Params{ContextID: "X", SelfEph: e2, PeerEph: e1, Label: "v1"}, Config{})
		sC, _ := NewSecureSessionWithParams(shared, Params{ContextID: "X", SelfEph: e1, PeerEph: e3, Label: "v1"}, Config{})

		ct, err := sA.Encrypt([]byte("secret"))
		require.NoError(t, err)

		_, err = sB.Decrypt(ct)
		require.NoError(t, err)

		_, err = sC.Decrypt(ct)
		require.Error(t, err)
	})

	t.Run("Nonce randomness (same plaintext → different ciphertexts)", func(t *testing.T) {
		cfg := Config{MaxAge: time.Second, IdleTimeout: time.Second, MaxMessages: 100}
		seed := b(32)
		s, err := NewSecureSession("id", seed, cfg)
		require.NoError(t, err)

		pt := []byte("same-plaintext")
		ct1, err := s.Encrypt(pt)
		require.NoError(t, err)
		ct2, err := s.Encrypt(pt)
		require.NoError(t, err)

		require.NotEqual(t, ct1, ct2)
		require.True(t, len(ct1) > chacha20poly1305.NonceSize)
		require.True(t, len(ct2) > chacha20poly1305.NonceSize)

		nonce1 := ct1[:chacha20poly1305.NonceSize]
		nonce2 := ct2[:chacha20poly1305.NonceSize]
		require.NotEqual(t, nonce1, nonce2)
	})

	t.Run("Close() zeroizes key material & forbids further use", func(t *testing.T) {
		seed := b(32)
		s, err := NewSecureSession("idZ", seed, Config{})
		require.NoError(t, err)

		// 길이 기록
		encLen, sigLen, seedLen := len(s.encryptKey), len(s.signingKey), len(s.sessionSeed)

		require.NoError(t, s.Close())
		require.True(t, s.IsExpired())

		// zeroized
		require.Equal(t, bytes.Repeat([]byte{0}, encLen), s.encryptKey)
		require.Equal(t, bytes.Repeat([]byte{0}, sigLen), s.signingKey)
		require.Equal(t, bytes.Repeat([]byte{0}, seedLen), s.sessionSeed)

		_, err = s.EncryptAndSign([]byte("hi"))
		require.Error(t, err)
	})

	t.Run("EncryptAndSign format & roundtrip", func(t *testing.T) {
		cfg := Config{MaxAge: time.Second, IdleTimeout: time.Second, MaxMessages: 10}
		s, err := NewSecureSession("fmt", b(32), cfg)
		require.NoError(t, err)

		pt := []byte("format-check")
		ct, err := s.EncryptAndSign(pt)
		require.NoError(t, err)
		require.Greater(t, len(ct), chacha20poly1305.NonceSize)

		nonce := ct[:chacha20poly1305.NonceSize]
		require.Len(t, nonce, chacha20poly1305.NonceSize)

		out, err := s.DecryptAndVerify(ct)
		require.NoError(t, err)
		require.Equal(t, pt, out)
	})

	t.Run("canonicalOrder sorts lexicographically", func(t *testing.T) {
		a := []byte{0x01, 0xFF}
		bb := []byte{0x02, 0x00}
		lo, hi := canonicalOrder(a, bb)
		require.True(t, bytes.Compare(lo, hi) < 0)
		require.Equal(t, a, lo)
		require.Equal(t, bb, hi)

		lo2, hi2 := canonicalOrder(bb, a)
		require.Equal(t, lo, lo2)
		require.Equal(t, hi, hi2)
	})
}


func b(n int) []byte {
	out := make([]byte, n)
	_, _ = rand.Read(out)
	return out
}

func hmacSHA256(k, msg []byte) []byte {
	m := hmac.New(sha256.New, k)
	m.Write(msg)
	return m.Sum(nil)
}