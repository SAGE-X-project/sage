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

package keys

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateP256KeyPair(t *testing.T) {
	t.Run("성공적인 P-256 키 쌍 생성", func(t *testing.T) {
		keyPair, err := GenerateP256KeyPair()
		require.NoError(t, err)
		require.NotNil(t, keyPair)

		// Type 검증
		assert.Equal(t, sagecrypto.KeyTypeP256, keyPair.Type())

		// ID 검증
		assert.NotEmpty(t, keyPair.ID())
		assert.Len(t, keyPair.ID(), 16) // 8 bytes hex-encoded = 16 chars

		// Public key 검증
		pubKey := keyPair.PublicKey()
		require.NotNil(t, pubKey)

		ecdsaPubKey, ok := pubKey.(*ecdsa.PublicKey)
		require.True(t, ok, "Public key should be *ecdsa.PublicKey")
		assert.Equal(t, elliptic.P256(), ecdsaPubKey.Curve)

		// Private key 검증
		privKey := keyPair.PrivateKey()
		require.NotNil(t, privKey)

		ecdsaPrivKey, ok := privKey.(*ecdsa.PrivateKey)
		require.True(t, ok, "Private key should be *ecdsa.PrivateKey")
		assert.Equal(t, elliptic.P256(), ecdsaPrivKey.Curve)

		t.Logf("✅ P-256 키 쌍 생성 성공")
		t.Logf("   KeyPair ID: %s", keyPair.ID())
		t.Logf("   Type: %s", keyPair.Type())
		t.Logf("   Curve: P-256")
	})

	t.Run("고유한 키 ID 생성", func(t *testing.T) {
		keyPair1, err := GenerateP256KeyPair()
		require.NoError(t, err)

		keyPair2, err := GenerateP256KeyPair()
		require.NoError(t, err)

		// 서로 다른 키 쌍은 서로 다른 ID를 가져야 함
		assert.NotEqual(t, keyPair1.ID(), keyPair2.ID())

		t.Logf("✅ 고유한 키 ID 확인")
		t.Logf("   KeyPair 1 ID: %s", keyPair1.ID())
		t.Logf("   KeyPair 2 ID: %s", keyPair2.ID())
	})
}

func TestP256KeyPair_Sign(t *testing.T) {
	keyPair, err := GenerateP256KeyPair()
	require.NoError(t, err)

	t.Run("메시지 서명 성공", func(t *testing.T) {
		message := []byte("Hello, SAGE P-256!")

		signature, err := keyPair.Sign(message)
		require.NoError(t, err)
		require.NotNil(t, signature)

		// P-256 서명은 64바이트 (32 bytes R + 32 bytes S)
		assert.Len(t, signature, 64)

		t.Logf("✅ P-256 서명 생성 성공")
		t.Logf("   Message: %s", string(message))
		t.Logf("   Signature length: %d bytes", len(signature))
	})

	t.Run("같은 메시지의 서명은 다름 (nonce 사용)", func(t *testing.T) {
		message := []byte("Test message")

		sig1, err := keyPair.Sign(message)
		require.NoError(t, err)

		sig2, err := keyPair.Sign(message)
		require.NoError(t, err)

		// ECDSA는 nonce를 사용하므로 같은 메시지라도 서명이 다름
		assert.NotEqual(t, sig1, sig2)

		t.Logf("✅ Nonce 기반 서명 확인")
	})
}

func TestP256KeyPair_Verify(t *testing.T) {
	keyPair, err := GenerateP256KeyPair()
	require.NoError(t, err)

	t.Run("유효한 서명 검증 성공", func(t *testing.T) {
		message := []byte("Valid message for P-256 signature")

		// 서명 생성
		signature, err := keyPair.Sign(message)
		require.NoError(t, err)

		// 서명 검증
		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)

		t.Logf("✅ P-256 서명 검증 성공")
	})

	t.Run("변조된 메시지 검증 실패", func(t *testing.T) {
		originalMessage := []byte("Original message")
		tamperedMessage := []byte("Tampered message")

		// 원본 메시지로 서명 생성
		signature, err := keyPair.Sign(originalMessage)
		require.NoError(t, err)

		// 변조된 메시지로 검증 시도 → 실패해야 함
		err = keyPair.Verify(tamperedMessage, signature)
		assert.Error(t, err)
		assert.Equal(t, sagecrypto.ErrInvalidSignature, err)

		t.Logf("✅ 변조된 메시지 탐지 성공")
	})

	t.Run("잘못된 서명 검증 실패", func(t *testing.T) {
		message := []byte("Test message")

		// 올바른 서명 생성
		signature, err := keyPair.Sign(message)
		require.NoError(t, err)

		// 서명 변조
		corruptedSignature := make([]byte, len(signature))
		copy(corruptedSignature, signature)
		corruptedSignature[0] ^= 0xFF // 첫 바이트 반전

		// 변조된 서명으로 검증 시도 → 실패해야 함
		err = keyPair.Verify(message, corruptedSignature)
		assert.Error(t, err)
		assert.Equal(t, sagecrypto.ErrInvalidSignature, err)

		t.Logf("✅ 잘못된 서명 탐지 성공")
	})

	t.Run("잘못된 서명 길이 검증 실패", func(t *testing.T) {
		message := []byte("Test message")

		// 잘못된 길이의 서명
		invalidSignature := []byte{0x01, 0x02, 0x03}

		err = keyPair.Verify(message, invalidSignature)
		assert.Error(t, err)
		assert.Equal(t, sagecrypto.ErrInvalidSignature, err)

		t.Logf("✅ 잘못된 서명 길이 탐지 성공")
	})
}

func TestNewP256KeyPair(t *testing.T) {
	t.Run("기존 ECDSA 개인키로 KeyPair 생성", func(t *testing.T) {
		// ECDSA P-256 키 생성
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		// KeyPair로 래핑
		keyPair, err := NewP256KeyPair(privKey, "")
		require.NoError(t, err)
		require.NotNil(t, keyPair)

		// Type 검증
		assert.Equal(t, sagecrypto.KeyTypeP256, keyPair.Type())

		// ID 자동 생성 확인
		assert.NotEmpty(t, keyPair.ID())

		// 키가 올바르게 설정되었는지 확인 (서명/검증 테스트)
		message := []byte("Test message")
		signature, err := keyPair.Sign(message)
		require.NoError(t, err)

		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)

		t.Logf("✅ 기존 ECDSA 키로 KeyPair 생성 성공")
	})

	t.Run("사용자 지정 ID로 KeyPair 생성", func(t *testing.T) {
		privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		customID := "custom-p256-key-001"
		keyPair, err := NewP256KeyPair(privKey, customID)
		require.NoError(t, err)

		assert.Equal(t, customID, keyPair.ID())

		t.Logf("✅ 사용자 지정 ID 확인: %s", keyPair.ID())
	})

	t.Run("잘못된 곡선의 키 거부", func(t *testing.T) {
		// P-384 키 생성 (P-256이 아님)
		privKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		require.NoError(t, err)

		// P-256 KeyPair로 생성 시도 → 실패해야 함
		_, err = NewP256KeyPair(privKey, "")
		assert.Error(t, err)
		assert.Equal(t, sagecrypto.ErrInvalidKeyType, err)

		t.Logf("✅ 잘못된 곡선 거부 확인")
	})
}

func TestP256KeyPair_RFC9421Compatibility(t *testing.T) {
	t.Run("RFC 9421 서명 형식 호환성", func(t *testing.T) {
		keyPair, err := GenerateP256KeyPair()
		require.NoError(t, err)

		message := []byte("RFC 9421 test message")

		// 서명 생성
		signature, err := keyPair.Sign(message)
		require.NoError(t, err)

		// RFC 9421은 raw format (64 bytes) 서명 사용
		assert.Len(t, signature, 64, "RFC 9421 requires 64-byte raw signatures")

		// R과 S 부분 확인 (각 32 bytes)
		r := signature[:32]
		s := signature[32:64]

		assert.NotEqual(t, make([]byte, 32), r, "R should not be all zeros")
		assert.NotEqual(t, make([]byte, 32), s, "S should not be all zeros")

		t.Logf("✅ RFC 9421 호환 서명 형식 확인")
		t.Logf("   Total length: %d bytes", len(signature))
		t.Logf("   R length: %d bytes", len(r))
		t.Logf("   S length: %d bytes", len(s))
	})
}
