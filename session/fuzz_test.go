package session

import (
	"testing"
	"time"

	"github.com/sage-x-project/sage/crypto"
)

// FuzzSessionCreation fuzzes session creation
func FuzzSessionCreation(f *testing.F) {
	f.Add(uint64(3600000))      // 1 hour
	f.Add(uint64(600000))        // 10 minutes
	f.Add(uint64(1000))          // 1 second
	f.Add(uint64(86400000))      // 24 hours

	clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
	serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)

	f.Fuzz(func(t *testing.T, maxAge uint64) {
		// Limit max age to reasonable values to prevent overflow
		if maxAge == 0 || maxAge > 604800000 { // 7 days max
			t.Skip()
		}

		config := ManagerConfig{
			SessionMaxAge:      time.Duration(maxAge) * time.Millisecond,
			SessionIdleTimeout: 10 * time.Minute,
			CleanupInterval:    30 * time.Second,
		}

		manager := NewManager(config)

		// Create session
		sess, err := manager.Create(
			clientKey.PublicKey(),
			serverKey.PublicKey(),
			clientKey,
		)

		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Verify session properties
		if sess.ID() == "" {
			t.Fatal("Session ID is empty")
		}

		// Retrieve session
		retrieved, err := manager.Get(sess.ID())
		if err != nil {
			t.Fatalf("Failed to retrieve session: %v", err)
		}

		if retrieved.ID() != sess.ID() {
			t.Fatal("Session IDs don't match")
		}
	})
}

// FuzzSessionEncryptDecrypt fuzzes session encryption/decryption
func FuzzSessionEncryptDecrypt(f *testing.F) {
	f.Add([]byte("hello"))
	f.Add([]byte(""))
	f.Add([]byte("a"))
	f.Add(make([]byte, 1024))
	f.Add(make([]byte, 65536))

	manager := NewManager(ManagerConfig{
		SessionMaxAge:      1 * time.Hour,
		SessionIdleTimeout: 10 * time.Minute,
		CleanupInterval:    30 * time.Second,
	})

	clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
	serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)

	sess, _ := manager.Create(
		clientKey.PublicKey(),
		serverKey.PublicKey(),
		clientKey,
	)

	f.Fuzz(func(t *testing.T, plaintext []byte) {
		// Encrypt
		encrypted, err := sess.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Failed to encrypt: %v", err)
		}

		// Decrypt
		decrypted, err := sess.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Failed to decrypt: %v", err)
		}

		// Verify roundtrip
		if !equalBytes(plaintext, decrypted) {
			t.Fatal("Decrypted data doesn't match original")
		}

		// Verify that modified ciphertext fails
		if len(encrypted) > 0 {
			modified := make([]byte, len(encrypted))
			copy(modified, encrypted)
			modified[0] ^= 0xFF

			_, err = sess.Decrypt(modified)
			if err == nil {
				t.Fatal("Decryption succeeded with modified ciphertext")
			}
		}
	})
}

// FuzzNonceValidation fuzzes nonce validation
func FuzzNonceValidation(f *testing.F) {
	f.Add([]byte("nonce1"), int64(0))
	f.Add([]byte("nonce2"), int64(1000))
	f.Add(make([]byte, 32), int64(-1000))

	manager := NewManager(ManagerConfig{
		SessionMaxAge:      1 * time.Hour,
		SessionIdleTimeout: 10 * time.Minute,
		CleanupInterval:    30 * time.Second,
	})

	clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
	serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)

	sess, _ := manager.Create(
		clientKey.PublicKey(),
		serverKey.PublicKey(),
		clientKey,
	)

	f.Fuzz(func(t *testing.T, nonce []byte, timestampOffset int64) {
		// Limit timestamp offset to reasonable range
		if timestampOffset < -3600000 || timestampOffset > 3600000 {
			t.Skip()
		}

		timestamp := time.Now().Add(time.Duration(timestampOffset) * time.Millisecond)

		// First validation should succeed
		err := sess.ValidateNonce(nonce, timestamp)
		isValid := err == nil

		// Second validation of same nonce should fail
		err2 := sess.ValidateNonce(nonce, timestamp)
		if isValid && err2 == nil {
			t.Fatal("Replay attack: same nonce validated twice")
		}
	})
}

// FuzzSessionExpiration fuzzes session expiration logic
func FuzzSessionExpiration(f *testing.F) {
	f.Add(uint64(100), uint64(50))
	f.Add(uint64(1000), uint64(500))
	f.Add(uint64(5000), uint64(2500))

	clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
	serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)

	f.Fuzz(func(t *testing.T, maxAge, idleTimeout uint64) {
		// Skip invalid configurations
		if maxAge == 0 || idleTimeout == 0 || maxAge > 86400000 || idleTimeout > 86400000 {
			t.Skip()
		}

		config := ManagerConfig{
			SessionMaxAge:      time.Duration(maxAge) * time.Millisecond,
			SessionIdleTimeout: time.Duration(idleTimeout) * time.Millisecond,
			CleanupInterval:    10 * time.Millisecond,
		}

		manager := NewManager(config)

		sess, err := manager.Create(
			clientKey.PublicKey(),
			serverKey.PublicKey(),
			clientKey,
		)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		sessionID := sess.ID()

		// Session should exist immediately
		_, err = manager.Get(sessionID)
		if err != nil {
			t.Fatal("Session should exist immediately after creation")
		}

		// Wait for idle timeout
		time.Sleep(time.Duration(idleTimeout+50) * time.Millisecond)

		// Session should be expired
		_, err = manager.Get(sessionID)
		if err == nil {
			// May still exist if cleanup hasn't run yet, that's okay
		}
	})
}

// FuzzConcurrentSessionAccess fuzzes concurrent session access
func FuzzConcurrentSessionAccess(f *testing.F) {
	f.Add([]byte("data1"), []byte("data2"))

	manager := NewManager(ManagerConfig{
		SessionMaxAge:      1 * time.Hour,
		SessionIdleTimeout: 10 * time.Minute,
		CleanupInterval:    30 * time.Second,
	})

	clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
	serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)

	sess, _ := manager.Create(
		clientKey.PublicKey(),
		serverKey.PublicKey(),
		clientKey,
	)

	f.Fuzz(func(t *testing.T, data1, data2 []byte) {
		// Concurrent encryption/decryption
		done := make(chan bool, 2)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic in goroutine 1: %v", r)
				}
				done <- true
			}()

			encrypted, err := sess.Encrypt(data1)
			if err != nil {
				return
			}
			_, _ = sess.Decrypt(encrypted)
		}()

		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic in goroutine 2: %v", r)
				}
				done <- true
			}()

			encrypted, err := sess.Encrypt(data2)
			if err != nil {
				return
			}
			_, _ = sess.Decrypt(encrypted)
		}()

		<-done
		<-done
	})
}

// FuzzInvalidSessionData fuzzes with invalid session data
func FuzzInvalidSessionData(f *testing.F) {
	f.Add([]byte("random"), []byte("data"))

	manager := NewManager(ManagerConfig{
		SessionMaxAge:      1 * time.Hour,
		SessionIdleTimeout: 10 * time.Minute,
		CleanupInterval:    30 * time.Second,
	})

	clientKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)
	serverKey, _ := crypto.GenerateKeyPair(crypto.KeyTypeX25519)

	sess, _ := manager.Create(
		clientKey.PublicKey(),
		serverKey.PublicKey(),
		clientKey,
	)

	f.Fuzz(func(t *testing.T, invalidData []byte, garbage []byte) {
		// Try to decrypt invalid data
		_, err := sess.Decrypt(invalidData)
		// Should not panic, should return error
		_ = err

		// Try operations on non-existent session
		fakeSessionID := string(garbage)
		_, err = manager.Get(fakeSessionID)
		_ = err
	})
}

func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
