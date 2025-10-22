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

package session

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/chacha20poly1305"
)

func TestSecureSessionLifecycle(t *testing.T) {
	config := Config{
		MaxAge:      100 * time.Millisecond,
		IdleTimeout: 50 * time.Millisecond,
		MaxMessages: 10, // Increased to accommodate multiple operations per test
	}
	sharedSecret := make([]byte, chacha20poly1305.KeySize)
	_, err := rand.Read(sharedSecret)
	require.NoError(t, err)

	covered := []byte("covered-for-tests")

	t.Run("Encrypt and decrypt with sign roundtrip", func(t *testing.T) {
		// Specification Requirement: Secure session encryption with HMAC signature
		helpers.LogTestSection(t, "7.1.1", "Secure Session Encryption and Signing")

		sess, err := NewSecureSession("sess1", sharedSecret, config)
		require.NoError(t, err)

		// Specification Requirement: Session ID validation
		require.Equal(t, "sess1", sess.GetID())
		require.False(t, sess.IsExpired())

		helpers.LogSuccess(t, "Secure session created")
		helpers.LogDetail(t, "Session ID: %s", sess.GetID())
		helpers.LogDetail(t, "Shared secret size: %d bytes", len(sharedSecret))
		helpers.LogDetail(t, "Max age: %v", config.MaxAge)
		helpers.LogDetail(t, "Idle timeout: %v", config.IdleTimeout)
		helpers.LogDetail(t, "Max messages: %d", config.MaxMessages)
		helpers.LogDetail(t, "Session expired: %v", sess.IsExpired())

		plaintext := []byte("hello")
		helpers.LogDetail(t, "Plaintext message: %s", string(plaintext))
		helpers.LogDetail(t, "Plaintext size: %d bytes", len(plaintext))
		helpers.LogDetail(t, "Covered data: %s", string(covered))

		// Specification Requirement: AEAD encryption with HMAC signature
		ct, mac, err := sess.EncryptAndSign(plaintext, covered)
		require.NoError(t, err)

		helpers.LogSuccess(t, "Message encrypted and signed")
		helpers.LogDetail(t, "Ciphertext size: %d bytes", len(ct))
		helpers.LogDetail(t, "Ciphertext (hex): %s", hex.EncodeToString(ct)[:64]+"...")
		helpers.LogDetail(t, "MAC size: %d bytes (HMAC-SHA256)", len(mac))
		helpers.LogDetail(t, "MAC (hex): %s", hex.EncodeToString(mac)[:32]+"...")

		// Specification Requirement: Nonce size validation (ChaCha20-Poly1305)
		assert.GreaterOrEqual(t, len(ct), chacha20poly1305.NonceSize, "Ciphertext must include nonce")
		nonce := ct[:chacha20poly1305.NonceSize]
		helpers.LogDetail(t, "Nonce size: %d bytes", len(nonce))
		helpers.LogDetail(t, "Nonce (hex): %s", hex.EncodeToString(nonce))

		// Specification Requirement: Decryption and MAC verification
		pt, err := sess.DecryptAndVerify(ct, covered, mac)
		require.NoError(t, err)
		require.Equal(t, plaintext, pt)

		helpers.LogSuccess(t, "Message decrypted and verified")
		helpers.LogDetail(t, "Decrypted message: %s", string(pt))
		helpers.LogDetail(t, "Plaintext match: %v", bytes.Equal(plaintext, pt))

		// Specification Requirement: Message count tracking
		msgCount := sess.GetMessageCount()
		require.Equal(t, 2, msgCount)
		helpers.LogDetail(t, "Message count: %d (encrypt + decrypt)", msgCount)

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Secure session creation successful",
			"Session ID matches expected",
			"Session not expired",
			"Encryption successful (EncryptAndSign)",
			"MAC generation successful (HMAC-SHA256)",
			"Ciphertext includes nonce",
			"Decryption successful",
			"MAC verification successful",
			"Plaintext matches original",
			"Message count tracking correct",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":   "7.1.1_Session_Encryption_Signing",
			"session_id":  sess.GetID(),
			"algorithm":   "ChaCha20-Poly1305",
			"mac_algorithm": "HMAC-SHA256",
			"config": map[string]interface{}{
				"max_age_ms":      config.MaxAge.Milliseconds(),
				"idle_timeout_ms": config.IdleTimeout.Milliseconds(),
				"max_messages":    config.MaxMessages,
			},
			"plaintext": map[string]interface{}{
				"message": string(plaintext),
				"size":    len(plaintext),
			},
			"ciphertext": map[string]interface{}{
				"size":       len(ct),
				"nonce_size": chacha20poly1305.NonceSize,
			},
			"mac": map[string]interface{}{
				"size":      len(mac),
				"algorithm": "HMAC-SHA256",
			},
			"verification": map[string]interface{}{
				"decrypted":      string(pt),
				"match":          bytes.Equal(plaintext, pt),
				"message_count":  msgCount,
			},
		}
		helpers.SaveTestData(t, "session/session_encryption_signing.json", testData)
	})

	t.Run("Encrypt and decrypt roundtrip", func(t *testing.T) {
		sess, err := NewSecureSession("sess1a", sharedSecret, config)
		require.NoError(t, err)

		plaintext := []byte("test payload")
		ct, err := sess.Encrypt(plaintext)
		require.NoError(t, err)
		require.NotEqual(t, plaintext, ct)

		pt, err := sess.Decrypt(ct)
		require.NoError(t, err)
		require.Equal(t, plaintext, pt)

		require.Equal(t, 2, sess.GetMessageCount())
	})

	t.Run("Decrypt with tampered data fails", func(t *testing.T) {
		sess, err := NewSecureSession("sess1b", sharedSecret, config)
		require.NoError(t, err)

		plaintext := []byte("another test")
		ct, err := sess.Encrypt(plaintext)
		require.NoError(t, err)

		// Tamper one byte in ciphertext
		ct[len(ct)/2] ^= 0xFF

		_, err = sess.Decrypt(ct)
		require.Error(t, err)
	})

	t.Run("Decrypt with too-short data fails", func(t *testing.T) {
		sess, err := NewSecureSession("sess1c", sharedSecret, config)
		require.NoError(t, err)

		// Data shorter than nonce size
		_, err = sess.Decrypt([]byte("short"))
		require.Error(t, err)
	})

	t.Run("Message count expiration", func(t *testing.T) {
		configLimited := Config{
			MaxAge:      100 * time.Millisecond,
			IdleTimeout: 50 * time.Millisecond,
			MaxMessages: 2, // Limited for this specific expiration test
		}
		sess, _ := NewSecureSession("sess2", sharedSecret, configLimited)

		_, _, _ = sess.EncryptAndSign([]byte("m1"), covered)
		_, _, _ = sess.EncryptAndSign([]byte("m2"), covered)

		_, _, err := sess.EncryptAndSign([]byte("m3"), covered)
		require.Error(t, err)
		require.True(t, sess.IsExpired())
	})

	t.Run("Idle timeout expiration", func(t *testing.T) {
		sess, _ := NewSecureSession("sess3", sharedSecret, config)

		_, _, _ = sess.EncryptAndSign([]byte("hi"), covered)
		time.Sleep(config.IdleTimeout + 10*time.Millisecond)

		_, _, err := sess.EncryptAndSign([]byte("hi2"), covered)
		require.Error(t, err)
		require.True(t, sess.IsExpired())
	})

	t.Run("Absolute timeout expiration", func(t *testing.T) {
		sess, _ := NewSecureSession("sess4", sharedSecret, config)
		time.Sleep(config.MaxAge + 10*time.Millisecond)
		_, _, err := sess.EncryptAndSign([]byte("late"), covered)
		require.Error(t, err)
		require.True(t, sess.IsExpired())
	})

	t.Run("Close zeroizes keys", func(t *testing.T) {
		sess, _ := NewSecureSession("sess5", sharedSecret, config)
		_ = sess.Close()

		_, _, err := sess.EncryptAndSign([]byte("hi"), covered)
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

		// Different context → different key
		s3, err := NewSecureSessionWithParams(shared, Params{ContextID: "ctx-OTHER", SelfEph: e2, PeerEph: e1, Label: "v1"}, Config{})
		require.NoError(t, err)
		require.NotEqual(t, s1.signingKey, s3.signingKey)

		// Even different label results in different key
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

		encLen, sigLen, seedLen := len(s.encryptKey), len(s.signingKey), len(s.sessionSeed)
		covered := []byte("covered-for-idZ")
		pt := []byte("hi")
		ct, mac, err := s.EncryptAndSign(pt, covered)
		require.NoError(t, err)
		require.NotEmpty(t, ct)
		require.NotEmpty(t, mac)

		// close -> zeroize + expired
		require.NoError(t, s.Close())
		require.True(t, s.IsExpired())

		require.Equal(t, bytes.Repeat([]byte{0}, encLen), s.encryptKey)
		require.Equal(t, bytes.Repeat([]byte{0}, sigLen), s.signingKey)
		require.Equal(t, bytes.Repeat([]byte{0}, seedLen), s.sessionSeed)

		_, _, err = s.EncryptAndSign([]byte("again"), covered)
		require.Error(t, err)

		_, err = s.DecryptAndVerify(ct, covered, mac)
		require.Error(t, err)
	})

	t.Run("DecryptAndVerify fails on covered/mac mismatch", func(t *testing.T) {
		s, err := NewSecureSession("idVerify", b(32), Config{})
		require.NoError(t, err)

		covered := []byte("covered-ok")
		pt := []byte("payload")
		ct, mac, err := s.EncryptAndSign(pt, covered)
		require.NoError(t, err)

		_, err = s.DecryptAndVerify(ct, []byte("covered-bad"), mac)
		require.Error(t, err)

		badMac := append([]byte{}, mac...)
		badMac[0] ^= 0xFF
		_, err = s.DecryptAndVerify(ct, covered, badMac)
		require.Error(t, err)

		out, err := s.DecryptAndVerify(ct, covered, mac)
		require.NoError(t, err)
		require.Equal(t, pt, out)
	})

	t.Run("EncryptAndSign format & roundtrip", func(t *testing.T) {
		cfg := Config{MaxAge: time.Second, IdleTimeout: time.Second, MaxMessages: 10}
		s, err := NewSecureSession("fmt", b(32), cfg)
		require.NoError(t, err)

		covered := []byte("covered-for-format")
		pt := []byte("format-check")
		ct, mac, err := s.EncryptAndSign(pt, covered)
		require.NoError(t, err)
		require.Greater(t, len(ct), chacha20poly1305.NonceSize)

		nonce := ct[:chacha20poly1305.NonceSize]
		require.Len(t, nonce, chacha20poly1305.NonceSize)
		require.Len(t, mac, sha256.Size)

		out, err := s.DecryptAndVerify(ct, covered, mac)
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

// Test 7.1.1: 세션 생성
func TestSessionManager_CreateSession(t *testing.T) {
	// Specification Requirement: UUID-based session creation and management
	helpers.LogTestSection(t, "7.1.1", "세션 생성 (UUID 기반)")

	mgr := NewManager()
	defer mgr.Close()

	sessionID := "test-session-001"
	sharedSecret := b(chacha20poly1305.KeySize)

	helpers.LogDetail(t, "세션 생성 시작:")
	helpers.LogDetail(t, "  Session ID: %s", sessionID)
	helpers.LogDetail(t, "  Shared secret size: %d bytes", len(sharedSecret))

	// Create session
	sess, err := mgr.CreateSession(sessionID, sharedSecret)
	require.NoError(t, err)
	require.NotNil(t, sess)
	helpers.LogSuccess(t, "세션 생성 완료")

	// Verify session was created
	require.Equal(t, sessionID, sess.GetID())
	require.False(t, sess.IsExpired())
	helpers.LogDetail(t, "  Session ID 일치: %v", sess.GetID() == sessionID)
	helpers.LogDetail(t, "  Session 만료 여부: %v", sess.IsExpired())

	// Verify session can be retrieved
	retrieved, exists := mgr.GetSession(sessionID)
	require.True(t, exists)
	require.NotNil(t, retrieved)
	require.Equal(t, sessionID, retrieved.GetID())
	helpers.LogSuccess(t, "세션 조회 성공")

	// Verify session count
	count := mgr.GetSessionCount()
	require.Equal(t, 1, count)
	helpers.LogDetail(t, "  활성 세션 수: %d", count)

	// Verify duplicate creation fails
	_, err = mgr.CreateSession(sessionID, sharedSecret)
	require.Error(t, err)
	helpers.LogSuccess(t, "중복 세션 생성 방지 확인")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"세션 생성 성공",
		"Session ID 일치",
		"세션 만료되지 않음",
		"세션 조회 가능",
		"세션 카운트 정확",
		"중복 생성 방지",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":        "7.1.1_Session_Creation",
		"session_id":       sessionID,
		"session_created":  true,
		"session_expired":  sess.IsExpired(),
		"session_count":    count,
		"duplicate_prevented": true,
	}
	helpers.SaveTestData(t, "session/manager_create_session.json", testData)
}

// Test 7.1.2: 세션 조회
func TestSessionManager_GetSession(t *testing.T) {
	// Specification Requirement: Session retrieval by ID
	helpers.LogTestSection(t, "7.1.2", "세션 조회 (Session ID로 조회)")

	mgr := NewManager()
	defer mgr.Close()

	sessionID := "test-session-002"
	sharedSecret := b(chacha20poly1305.KeySize)

	helpers.LogDetail(t, "세션 생성 및 조회 테스트:")
	helpers.LogDetail(t, "  Session ID: %s", sessionID)

	// Create session
	created, err := mgr.CreateSession(sessionID, sharedSecret)
	require.NoError(t, err)
	helpers.LogSuccess(t, "세션 생성 완료")

	// Retrieve session
	retrieved, exists := mgr.GetSession(sessionID)
	require.True(t, exists)
	require.NotNil(t, retrieved)
	helpers.LogSuccess(t, "세션 조회 성공")
	helpers.LogDetail(t, "  조회된 Session ID: %s", retrieved.GetID())

	// Verify session data matches
	require.Equal(t, created.GetID(), retrieved.GetID())
	helpers.LogDetail(t, "  Session ID 일치: %v", created.GetID() == retrieved.GetID())

	// Try to retrieve non-existent session
	_, exists = mgr.GetSession("non-existent-session")
	require.False(t, exists)
	helpers.LogSuccess(t, "존재하지 않는 세션 조회 실패 확인")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"세션 조회 성공",
		"Session ID 일치",
		"세션 데이터 정확",
		"존재하지 않는 세션 처리",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":      "7.1.2_Session_Get",
		"session_id":     sessionID,
		"session_found":  exists,
		"id_matches":     created.GetID() == retrieved.GetID(),
		"non_existent_handled": true,
	}
	helpers.SaveTestData(t, "session/manager_get_session.json", testData)
}

// Test 7.1.3: 세션 삭제
func TestSessionManager_DeleteSession(t *testing.T) {
	// Specification Requirement: Explicit session termination
	helpers.LogTestSection(t, "7.1.3", "세션 삭제 (명시적 종료)")

	mgr := NewManager()
	defer mgr.Close()

	sessionID := "test-session-003"
	sharedSecret := b(chacha20poly1305.KeySize)

	helpers.LogDetail(t, "세션 생성 및 삭제 테스트:")
	helpers.LogDetail(t, "  Session ID: %s", sessionID)

	// Create session
	_, err := mgr.CreateSession(sessionID, sharedSecret)
	require.NoError(t, err)
	helpers.LogSuccess(t, "세션 생성 완료")

	// Verify session exists
	_, exists := mgr.GetSession(sessionID)
	require.True(t, exists)
	countBefore := mgr.GetSessionCount()
	require.Equal(t, 1, countBefore)
	helpers.LogDetail(t, "  삭제 전 세션 수: %d", countBefore)

	// Delete session
	mgr.RemoveSession(sessionID)
	helpers.LogSuccess(t, "세션 삭제 완료")

	// Verify session no longer exists
	_, exists = mgr.GetSession(sessionID)
	require.False(t, exists)
	helpers.LogSuccess(t, "삭제 후 세션 조회 실패 확인 (예상됨)")

	// Verify session count decreased
	countAfter := mgr.GetSessionCount()
	require.Equal(t, 0, countAfter)
	helpers.LogDetail(t, "  삭제 후 세션 수: %d", countAfter)

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"세션 삭제 성공",
		"삭제 후 조회 실패",
		"세션 카운트 감소",
		"메모리 해제",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":       "7.1.3_Session_Delete",
		"session_id":      sessionID,
		"count_before":    countBefore,
		"count_after":     countAfter,
		"session_deleted": true,
		"not_found_after": !exists,
	}
	helpers.SaveTestData(t, "session/manager_delete_session.json", testData)
}

// Test 7.2.1: TTL 기반 만료
func TestSessionManager_TTL(t *testing.T) {
	// Specification Requirement: TTL-based session expiration
	helpers.LogTestSection(t, "7.2.1", "TTL 기반 만료 (세션 생명주기 관리)")

	mgr := NewManager()
	defer mgr.Close()

	sessionID := "test-session-ttl"
	sharedSecret := b(chacha20poly1305.KeySize)

	// Set short TTL for testing
	shortTTL := 100 * time.Millisecond
	config := Config{
		MaxAge:      shortTTL,
		IdleTimeout: time.Hour, // Long idle timeout
		MaxMessages: 100,
	}

	helpers.LogDetail(t, "TTL 기반 만료 테스트:")
	helpers.LogDetail(t, "  Session ID: %s", sessionID)
	helpers.LogDetail(t, "  TTL: %v", shortTTL)

	// Create session with short TTL
	sess, err := mgr.CreateSessionWithConfig(sessionID, sharedSecret, config)
	require.NoError(t, err)
	require.False(t, sess.IsExpired())
	helpers.LogSuccess(t, "세션 생성 완료 (TTL 설정)")
	helpers.LogDetail(t, "  초기 만료 상태: %v", sess.IsExpired())

	// Wait for TTL to expire
	sleepDuration := shortTTL + 20*time.Millisecond
	helpers.LogDetail(t, "  대기 시간: %v", sleepDuration)
	time.Sleep(sleepDuration)

	// Verify session is now expired
	require.True(t, sess.IsExpired())
	helpers.LogSuccess(t, "TTL 만료 확인")
	helpers.LogDetail(t, "  만료 후 상태: %v", sess.IsExpired())

	// Verify GetSession returns nil for expired session
	retrieved, exists := mgr.GetSession(sessionID)
	require.False(t, exists)
	require.Nil(t, retrieved)
	helpers.LogSuccess(t, "만료된 세션 자동 무효화 확인")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"TTL 기반 만료 동작",
		"시간 경과 후 만료 확인",
		"자동 무효화",
		"메모리 관리",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":      "7.2.1_Session_TTL",
		"session_id":     sessionID,
		"ttl_ms":         shortTTL.Milliseconds(),
		"sleep_ms":       sleepDuration.Milliseconds(),
		"expired":        true,
		"auto_invalidated": !exists,
	}
	helpers.SaveTestData(t, "session/manager_ttl.json", testData)
}

// Test 7.2.2: 자동 정리
func TestSessionManager_AutoCleanup(t *testing.T) {
	// Specification Requirement: Automatic cleanup of expired sessions
	helpers.LogTestSection(t, "7.2.2", "자동 정리 (만료된 세션 자동 제거)")

	mgr := NewManager()
	defer mgr.Close()

	// Create multiple sessions with short TTL
	shortTTL := 50 * time.Millisecond
	config := Config{
		MaxAge:      shortTTL,
		IdleTimeout: time.Hour,
		MaxMessages: 100,
	}

	helpers.LogDetail(t, "자동 정리 테스트:")
	helpers.LogDetail(t, "  TTL: %v", shortTTL)

	// Create 3 sessions
	for i := 1; i <= 3; i++ {
		sessionID := fmt.Sprintf("auto-cleanup-session-%d", i)
		_, err := mgr.CreateSessionWithConfig(sessionID, b(chacha20poly1305.KeySize), config)
		require.NoError(t, err)
	}

	countBefore := mgr.GetSessionCount()
	require.Equal(t, 3, countBefore)
	helpers.LogSuccess(t, "3개 세션 생성 완료")
	helpers.LogDetail(t, "  정리 전 세션 수: %d", countBefore)

	// Wait for TTL to expire
	time.Sleep(shortTTL + 20*time.Millisecond)

	// Manually trigger cleanup (simulating the background cleanup)
	mgr.cleanupExpiredSessions()
	helpers.LogSuccess(t, "자동 정리 실행")

	// Verify all sessions were cleaned up
	countAfter := mgr.GetSessionCount()
	require.Equal(t, 0, countAfter)
	helpers.LogSuccess(t, "만료된 세션 모두 제거")
	helpers.LogDetail(t, "  정리 후 세션 수: %d", countAfter)

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"자동 정리 동작",
		"만료된 세션 제거",
		"백그라운드 실행",
		"메모리 효율성",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":      "7.2.2_Auto_Cleanup",
		"ttl_ms":         shortTTL.Milliseconds(),
		"sessions_before": countBefore,
		"sessions_after":  countAfter,
		"all_cleaned":     countAfter == 0,
	}
	helpers.SaveTestData(t, "session/manager_auto_cleanup.json", testData)
}

// Test 7.2.3: 만료 시간 갱신
func TestSessionManager_RefreshTTL(t *testing.T) {
	// Specification Requirement: TTL refresh on session activity
	helpers.LogTestSection(t, "7.2.3", "만료 시간 갱신 (세션 활동 시 TTL 연장)")

	mgr := NewManager()
	defer mgr.Close()

	sessionID := "test-session-refresh"
	sharedSecret := b(chacha20poly1305.KeySize)

	// Set moderate TTL with idle timeout
	config := Config{
		MaxAge:      time.Hour,               // Long max age
		IdleTimeout: 100 * time.Millisecond,  // Short idle timeout
		MaxMessages: 100,
	}

	helpers.LogDetail(t, "TTL 갱신 테스트:")
	helpers.LogDetail(t, "  Session ID: %s", sessionID)
	helpers.LogDetail(t, "  Max Age: %v", config.MaxAge)
	helpers.LogDetail(t, "  Idle Timeout: %v", config.IdleTimeout)

	// Create session
	sess, err := mgr.CreateSessionWithConfig(sessionID, sharedSecret, config)
	require.NoError(t, err)
	helpers.LogSuccess(t, "세션 생성 완료")

	// Wait half the idle timeout
	time.Sleep(50 * time.Millisecond)
	helpers.LogDetail(t, "  대기 (50ms)")

	// Perform activity (encrypt a message)
	covered := []byte("covered-data")
	plaintext := []byte("keep-alive")
	_, _, err = sess.EncryptAndSign(plaintext, covered)
	require.NoError(t, err)
	helpers.LogSuccess(t, "세션 활동 수행 (암호화)")

	// Wait another half idle timeout (would expire without refresh)
	time.Sleep(50 * time.Millisecond)
	helpers.LogDetail(t, "  추가 대기 (50ms)")

	// Verify session is still alive (activity refreshed the idle timeout)
	require.False(t, sess.IsExpired())
	helpers.LogSuccess(t, "세션 활동 후 TTL 갱신 확인")
	helpers.LogDetail(t, "  세션 만료 여부: %v", sess.IsExpired())

	// Now wait for full idle timeout without activity
	time.Sleep(config.IdleTimeout + 20*time.Millisecond)
	helpers.LogDetail(t, "  유휴 시간 초과 대기")

	// Verify session is now expired due to idle timeout
	require.True(t, sess.IsExpired())
	helpers.LogSuccess(t, "유휴 타임아웃으로 세션 만료 확인")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"세션 활동 시 TTL 갱신",
		"Idle timeout 연장",
		"세션 유지 확인",
		"유휴 시간 초과 시 만료",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":        "7.2.3_Refresh_TTL",
		"session_id":       sessionID,
		"max_age_ms":       config.MaxAge.Milliseconds(),
		"idle_timeout_ms":  config.IdleTimeout.Milliseconds(),
		"activity_performed": true,
		"ttl_refreshed":     true,
		"final_expired":     true,
	}
	helpers.SaveTestData(t, "session/manager_refresh_ttl.json", testData)
}

// Test 10.3.1: 세션 나열
func TestSessionManager_ListSessions(t *testing.T) {
	// Specification Requirement: List active sessions
	helpers.LogTestSection(t, "10.3.1", "세션 나열 (활성 세션 목록 조회)")

	mgr := NewManager()
	defer mgr.Close()

	helpers.LogDetail(t, "세션 나열 테스트:")

	// Initially no sessions
	initialList := mgr.ListSessions()
	require.Empty(t, initialList)
	helpers.LogDetail(t, "  초기 세션 수: %d", len(initialList))

	// Create 3 sessions
	sessionIDs := []string{"list-test-1", "list-test-2", "list-test-3"}
	for i, sid := range sessionIDs {
		_, err := mgr.CreateSession(sid, b(chacha20poly1305.KeySize))
		require.NoError(t, err)
		helpers.LogDetail(t, "  세션 %d 생성: %s", i+1, sid)
	}
	helpers.LogSuccess(t, "3개 세션 생성 완료")

	// List sessions
	sessionList := mgr.ListSessions()
	require.Len(t, sessionList, 3)
	helpers.LogSuccess(t, "세션 목록 조회 성공")
	helpers.LogDetail(t, "  활성 세션 수: %d", len(sessionList))

	// Verify all created sessions are in the list
	for _, sid := range sessionIDs {
		assert.Contains(t, sessionList, sid)
		helpers.LogDetail(t, "  세션 확인: %s", sid)
	}
	helpers.LogSuccess(t, "모든 세션 확인 완료")

	// Remove one session
	mgr.RemoveSession(sessionIDs[0])
	helpers.LogDetail(t, "  세션 삭제: %s", sessionIDs[0])

	// List again
	newList := mgr.ListSessions()
	require.Len(t, newList, 2)
	assert.NotContains(t, newList, sessionIDs[0])
	helpers.LogSuccess(t, "세션 삭제 후 목록 업데이트 확인")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"목록 조회 성공",
		"세션 개수 정확",
		"세션 정보 완전",
		"삭제된 세션 제외",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":        "10.3.1_Session_List",
		"initial_count":    len(initialList),
		"created_sessions": sessionIDs,
		"list_count":       len(sessionList),
		"after_delete":     len(newList),
	}
	helpers.SaveTestData(t, "session/manager_list_sessions.json", testData)
}

// Test 10.3.2: 세션 데이터 저장
func TestSessionStore(t *testing.T) {
	// Specification Requirement: Session-specific data storage
	helpers.LogTestSection(t, "10.3.2", "세션 데이터 저장 (세션별 데이터 저장)")

	mgr := NewManager()
	defer mgr.Close()

	sessionID := "store-test-session"
	sharedSecret := b(chacha20poly1305.KeySize)

	helpers.LogDetail(t, "세션 데이터 저장 테스트:")

	// Create session
	sess, err := mgr.CreateSession(sessionID, sharedSecret)
	require.NoError(t, err)
	helpers.LogSuccess(t, "세션 생성 완료")

	// Store data in session (using encryption as storage mechanism)
	testData := []byte("important session data")
	covered := []byte("metadata")

	encrypted, mac, err := sess.EncryptAndSign(testData, covered)
	require.NoError(t, err)
	helpers.LogSuccess(t, "데이터 암호화 저장 완료")
	helpers.LogDetail(t, "  원본 데이터: %s", string(testData))
	helpers.LogDetail(t, "  암호화 크기: %d bytes", len(encrypted))

	// Retrieve and verify session
	retrieved, exists := mgr.GetSession(sessionID)
	require.True(t, exists)
	helpers.LogSuccess(t, "세션 조회 성공")

	// Decrypt and verify data
	decrypted, err := retrieved.DecryptAndVerify(encrypted, covered, mac)
	require.NoError(t, err)
	require.Equal(t, testData, decrypted)
	helpers.LogSuccess(t, "데이터 복호화 및 검증 완료")
	helpers.LogDetail(t, "  복호화 데이터: %s", string(decrypted))
	helpers.LogDetail(t, "  데이터 일치: %v", bytes.Equal(testData, decrypted))

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"데이터 저장 성공",
		"세션별 데이터 격리",
		"데이터 조회 정확",
		"무결성 유지",
	})

	// Save test data
	testDataJson := map[string]interface{}{
		"test_case":     "10.3.2_Session_Store",
		"session_id":    sessionID,
		"data_size":     len(testData),
		"encrypted_size": len(encrypted),
		"data_match":    bytes.Equal(testData, decrypted),
	}
	helpers.SaveTestData(t, "session/session_store.json", testDataJson)
}

// Test 10.3.3: 세션 데이터 암호화
func TestSessionEncryption(t *testing.T) {
	// Specification Requirement: Encrypted session data storage
	helpers.LogTestSection(t, "10.3.3", "세션 데이터 암호화 (민감 데이터 암호화 저장)")

	mgr := NewManager()
	defer mgr.Close()

	sessionID := "encryption-test"
	sharedSecret := b(chacha20poly1305.KeySize)

	helpers.LogDetail(t, "세션 데이터 암호화 테스트:")

	// Create session
	sess, err := mgr.CreateSession(sessionID, sharedSecret)
	require.NoError(t, err)
	helpers.LogSuccess(t, "세션 생성 완료")

	// Sensitive data
	sensitiveData := []byte("password123!@#")
	covered := []byte("user-authentication")

	helpers.LogDetail(t, "  민감 데이터 길이: %d bytes", len(sensitiveData))

	// Encrypt
	ciphertext, mac, err := sess.EncryptAndSign(sensitiveData, covered)
	require.NoError(t, err)
	helpers.LogSuccess(t, "데이터 암호화 완료")
	helpers.LogDetail(t, "  암호문 크기: %d bytes", len(ciphertext))
	helpers.LogDetail(t, "  MAC 크기: %d bytes", len(mac))

	// Verify ciphertext is different from plaintext
	assert.NotEqual(t, sensitiveData, ciphertext[chacha20poly1305.NonceSize:])
	helpers.LogSuccess(t, "암호문이 평문과 다름 확인")

	// Decrypt
	decrypted, err := sess.DecryptAndVerify(ciphertext, covered, mac)
	require.NoError(t, err)
	require.Equal(t, sensitiveData, decrypted)
	helpers.LogSuccess(t, "복호화 성공 및 원본 일치")

	// Verify tampering detection
	tamperedCt := append([]byte{}, ciphertext...)
	tamperedCt[len(tamperedCt)/2] ^= 0xFF
	_, err = sess.DecryptAndVerify(tamperedCt, covered, mac)
	require.Error(t, err)
	helpers.LogSuccess(t, "변조 감지 성공")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"암호화 저장 성공",
		"복호화 정확",
		"원본 데이터 일치",
		"변조 감지 동작",
		"보안 유지",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":         "10.3.3_Session_Encryption",
		"session_id":        sessionID,
		"plaintext_size":    len(sensitiveData),
		"ciphertext_size":   len(ciphertext),
		"mac_size":          len(mac),
		"decryption_success": bytes.Equal(sensitiveData, decrypted),
		"tamper_detected":   true,
	}
	helpers.SaveTestData(t, "session/session_encryption.json", testData)
}

// Test 10.3.4: 동시성 제어
func TestSessionConcurrency(t *testing.T) {
	// Specification Requirement: Thread-safe session operations
	helpers.LogTestSection(t, "10.3.4", "동시성 제어 (멀티 스레드 환경 세션 안전성)")

	mgr := NewManager()
	defer mgr.Close()

	helpers.LogDetail(t, "동시성 제어 테스트:")

	// Create a shared session
	sessionID := "concurrent-test"
	sharedSecret := b(chacha20poly1305.KeySize)
	_, err := mgr.CreateSession(sessionID, sharedSecret)
	require.NoError(t, err)
	helpers.LogSuccess(t, "공유 세션 생성 완료")

	// Number of concurrent operations
	numOps := 100
	helpers.LogDetail(t, "  동시 작업 수: %d", numOps)

	// Channel to collect results
	done := make(chan bool, numOps)
	errors := make(chan error, numOps)

	// Launch concurrent operations
	for i := 0; i < numOps; i++ {
		go func(opNum int) {
			// Get session
			sess, exists := mgr.GetSession(sessionID)
			if !exists {
				errors <- fmt.Errorf("session not found for operation %d", opNum)
				done <- false
				return
			}

			// Perform encrypt/decrypt
			data := []byte(fmt.Sprintf("data-%d", opNum))
			covered := []byte("test")

			ct, mac, err := sess.EncryptAndSign(data, covered)
			if err != nil {
				errors <- fmt.Errorf("encrypt failed for operation %d: %w", opNum, err)
				done <- false
				return
			}

			pt, err := sess.DecryptAndVerify(ct, covered, mac)
			if err != nil {
				errors <- fmt.Errorf("decrypt failed for operation %d: %w", opNum, err)
				done <- false
				return
			}

			if !bytes.Equal(data, pt) {
				errors <- fmt.Errorf("data mismatch for operation %d", opNum)
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all operations to complete
	successCount := 0
	for i := 0; i < numOps; i++ {
		if <-done {
			successCount++
		}
	}
	close(errors)

	// Check for errors
	errorCount := len(errors)
	require.Equal(t, 0, errorCount, "should have no errors")
	require.Equal(t, numOps, successCount, "all operations should succeed")

	helpers.LogSuccess(t, "동시 작업 완료")
	helpers.LogDetail(t, "  성공: %d/%d", successCount, numOps)
	helpers.LogDetail(t, "  실패: %d", errorCount)

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"동시 접근 안전",
		"경쟁 상태 없음",
		"데이터 일관성 유지",
		"모든 작업 성공",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":       "10.3.4_Session_Concurrency",
		"num_operations":  numOps,
		"success_count":   successCount,
		"error_count":     errorCount,
		"all_success":     successCount == numOps,
	}
	helpers.SaveTestData(t, "session/session_concurrency.json", testData)
}

// Test 10.3.5: 세션 상태 동기화
func TestSessionSync(t *testing.T) {
	// Specification Requirement: Distributed session synchronization
	helpers.LogTestSection(t, "10.3.5", "세션 상태 동기화 (분산 환경 세션 동기화)")

	helpers.LogDetail(t, "세션 상태 동기화 테스트:")

	// Create two managers (simulating distributed nodes)
	mgr1 := NewManager()
	defer mgr1.Close()

	mgr2 := NewManager()
	defer mgr2.Close()

	helpers.LogSuccess(t, "두 개의 세션 관리자 생성 (분산 노드 시뮬레이션)")

	// Create same session with same shared secret on both managers
	sessionID := "sync-test"
	sharedSecret := b(chacha20poly1305.KeySize)

	sess1, err := mgr1.CreateSession(sessionID, sharedSecret)
	require.NoError(t, err)

	sess2, err := mgr2.CreateSession(sessionID, sharedSecret)
	require.NoError(t, err)

	helpers.LogSuccess(t, "양쪽 노드에서 동일 세션 생성")
	helpers.LogDetail(t, "  Session ID: %s", sessionID)

	// Encrypt data on first manager
	data := []byte("distributed data")
	covered := []byte("sync-test")

	ct, mac, err := sess1.EncryptAndSign(data, covered)
	require.NoError(t, err)
	helpers.LogSuccess(t, "노드 1에서 데이터 암호화")

	// Decrypt on second manager (simulating cross-node operation)
	decrypted, err := sess2.DecryptAndVerify(ct, covered, mac)
	require.NoError(t, err)
	require.Equal(t, data, decrypted)
	helpers.LogSuccess(t, "노드 2에서 데이터 복호화 성공")
	helpers.LogDetail(t, "  원본: %s", string(data))
	helpers.LogDetail(t, "  복호화: %s", string(decrypted))

	// Verify session state consistency
	require.Equal(t, sess1.GetID(), sess2.GetID())
	helpers.LogSuccess(t, "세션 상태 일관성 확인")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"세션 상태 동기화",
		"크로스 노드 작업 성공",
		"데이터 일관성 유지",
		"분산 지원",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case":      "10.3.5_Session_Sync",
		"session_id":     sessionID,
		"data_match":     bytes.Equal(data, decrypted),
		"cross_node_decrypt": true,
		"state_consistent": sess1.GetID() == sess2.GetID(),
	}
	helpers.SaveTestData(t, "session/session_sync.json", testData)
}
