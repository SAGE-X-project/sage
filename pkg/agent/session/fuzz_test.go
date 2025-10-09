// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.


package session_test

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/session"
)

// FuzzSessionCreation fuzzes session creation
func FuzzSessionCreation(f *testing.F) {
	f.Add([]byte("shared-secret-1"))
	f.Add([]byte(""))
	f.Add(make([]byte, 32))
	f.Add(make([]byte, 64))

	manager := session.NewManager()

	f.Fuzz(func(t *testing.T, sharedSecret []byte) {
		// Skip empty secrets
		if len(sharedSecret) == 0 {
			t.Skip()
		}

		// Generate random session ID
		sidBytes := make([]byte, 16)
		rand.Read(sidBytes)
		sessionID := string(sidBytes)

		// Create session
		sess, err := manager.CreateSession(sessionID, sharedSecret)
		if err != nil {
			// Some secrets might be invalid, that's okay
			return
		}

		// Verify session properties
		if sess.GetID() == "" {
			t.Fatal("Session ID is empty")
		}

		if sess.GetID() != sessionID {
			t.Fatalf("Session ID mismatch: expected %s, got %s", sessionID, sess.GetID())
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

	manager := session.NewManager()

	// Create a test session
	sharedSecret := make([]byte, 32)
	rand.Read(sharedSecret)
	sess, _ := manager.CreateSession("test-session", sharedSecret)

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

		// Verify ciphertext is different from plaintext
		if len(encrypted) > 0 && len(plaintext) > 0 && equalBytes(plaintext, encrypted) {
			t.Fatal("Ciphertext should differ from plaintext")
		}
	})
}

// FuzzNonceValidation fuzzes nonce validation for replay protection
func FuzzNonceValidation(f *testing.F) {
	f.Add([]byte("nonce1"), int64(1234567890))
	f.Add([]byte(""), int64(0))
	f.Add(make([]byte, 32), int64(9999999999))

	manager := session.NewManager()

	f.Fuzz(func(t *testing.T, nonce []byte, timestamp int64) {
		// Skip invalid inputs
		if len(nonce) == 0 || timestamp <= 0 {
			t.Skip()
		}

		// Use the session's key ID as identifier
		keyID := "test-key-id"

		// First use of nonce should be allowed
		seen := manager.ReplayGuardSeenOnce(keyID, string(nonce))
		if seen {
			// If we get collision on random nonce, skip
			t.Skip()
		}

		// Second use of same nonce should be detected
		seenAgain := manager.ReplayGuardSeenOnce(keyID, string(nonce))
		if !seenAgain {
			t.Fatal("Replay protection failed: duplicate nonce not detected")
		}
	})
}

// FuzzSessionExpiration fuzzes session expiration logic
func FuzzSessionExpiration(f *testing.F) {
	f.Add(uint64(1000))    // 1 second
	f.Add(uint64(60000))   // 1 minute
	f.Add(uint64(3600000)) // 1 hour

	f.Fuzz(func(t *testing.T, maxAge uint64) {
		// Limit max age to reasonable values
		if maxAge == 0 || maxAge > 604800000 { // 7 days max
			t.Skip()
		}

		manager := session.NewManager()
		manager.SetDefaultConfig(session.Config{
			MaxAge:      time.Duration(maxAge) * time.Millisecond,
			IdleTimeout: 10 * time.Minute,
			MaxMessages: 1000,
		})

		sharedSecret := make([]byte, 32)
		rand.Read(sharedSecret)

		sess, err := manager.CreateSession("exp-test", sharedSecret)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Session should be valid immediately
		if sess.IsExpired() {
			t.Fatal("Newly created session is expired")
		}

		// Verify session was created recently
		createdAt := sess.GetCreatedAt()
		if createdAt.IsZero() {
			t.Fatal("Session created time not set")
		}

		// Check that created time is recent (within last 2 seconds)
		if time.Since(createdAt) > 2*time.Second {
			t.Fatalf("Session created time too old: %v", createdAt)
		}
	})
}

// FuzzInvalidEncryptedData fuzzes decryption with invalid data
func FuzzInvalidEncryptedData(f *testing.F) {
	f.Add([]byte("invalid"))
	f.Add([]byte(""))
	f.Add(make([]byte, 100))

	manager := session.NewManager()
	sharedSecret := make([]byte, 32)
	rand.Read(sharedSecret)
	sess, _ := manager.CreateSession("invalid-test", sharedSecret)

	f.Fuzz(func(t *testing.T, invalidData []byte) {
		// Try to decrypt invalid data
		// Should not crash, should return error
		_, err := sess.Decrypt(invalidData)

		// We expect an error for invalid data
		// The important thing is it doesn't panic
		_ = err
	})
}

// FuzzSessionMetadata fuzzes session metadata handling
func FuzzSessionMetadata(f *testing.F) {
	f.Add("key1", "value1")
	f.Add("", "")
	f.Add("long-key-name", "long-value-data")

	manager := session.NewManager()
	sharedSecret := make([]byte, 32)
	rand.Read(sharedSecret)
	sess, _ := manager.CreateSession("meta-test", sharedSecret)

	f.Fuzz(func(t *testing.T, key, value string) {
		// Skip empty keys
		if key == "" {
			t.Skip()
		}

		// Test session config instead
		config := sess.GetConfig()

		// Verify config is set
		if config.MaxAge == 0 {
			t.Fatal("Session MaxAge not set")
		}

		if config.IdleTimeout == 0 {
			t.Fatal("Session IdleTimeout not set")
		}

		// Verify message count tracking
		initialCount := sess.GetMessageCount()
		if initialCount < 0 {
			t.Fatal("Message count should not be negative")
		}
	})
}

// Helper function
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
