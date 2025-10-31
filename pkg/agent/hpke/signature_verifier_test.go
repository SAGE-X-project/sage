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

package hpke

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/asn1"
	"math/big"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECDSAVerifier_Verify_RawSignature(t *testing.T) {
	// 사양 요구사항: ECDSA (Secp256k1) 서명 검증 지원
	helpers.LogTestSection(t, "6.2.1", "HPKE ECDSA Signature Verification - Raw 64-byte Format")

	// Generate Ethereum-compatible ECDSA key
	privateKey, err := ethcrypto.GenerateKey()
	require.NoError(t, err)
	helpers.LogSuccess(t, "ECDSA Secp256k1 key pair generated")

	payload := []byte("HPKE handshake data for ECDSA verification")
	hash := ethcrypto.Keccak256(payload)

	// Sign using Ethereum crypto (produces 65-byte signature with recovery ID)
	signature, err := ethcrypto.Sign(hash, privateKey)
	require.NoError(t, err)
	require.Equal(t, 65, len(signature), "Ethereum signature should be 65 bytes")

	// Remove recovery ID for verification (last byte)
	rawSignature := signature[:64]

	verifier := NewECDSAVerifier()
	err = verifier.Verify(payload, rawSignature, &privateKey.PublicKey)
	assert.NoError(t, err, "Valid ECDSA raw signature should verify")
	helpers.LogSuccess(t, "Raw 64-byte ECDSA signature verified successfully")
}

func TestECDSAVerifier_Verify_DERSignature(t *testing.T) {
	// 사양 요구사항: ASN.1 DER 인코딩 ECDSA 서명 지원
	helpers.LogTestSection(t, "6.2.2", "HPKE ECDSA Signature Verification - DER Format")

	privateKey, err := ethcrypto.GenerateKey()
	require.NoError(t, err)

	payload := []byte("HPKE handshake with DER signature")
	hash := ethcrypto.Keccak256(payload)

	// Sign and get r, s values
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash)
	require.NoError(t, err)

	// Encode as ASN.1 DER
	type ecdsaSignature struct {
		R, S *big.Int
	}
	derSignature, err := asn1.Marshal(ecdsaSignature{R: r, S: s})
	require.NoError(t, err)

	verifier := NewECDSAVerifier()
	err = verifier.Verify(payload, derSignature, &privateKey.PublicKey)
	assert.NoError(t, err, "Valid ECDSA DER signature should verify")
	helpers.LogSuccess(t, "DER-encoded ECDSA signature verified successfully")
}

func TestECDSAVerifier_Verify_InvalidSignature(t *testing.T) {
	// 사양 요구사항: 잘못된 서명 거부
	helpers.LogTestSection(t, "6.2.3", "HPKE ECDSA Signature Verification - Invalid Signature")

	privateKey, err := ethcrypto.GenerateKey()
	require.NoError(t, err)

	payload := []byte("original payload")
	tamperedPayload := []byte("tampered payload")
	hash := ethcrypto.Keccak256(payload)

	signature, err := ethcrypto.Sign(hash, privateKey)
	require.NoError(t, err)
	rawSignature := signature[:64]

	verifier := NewECDSAVerifier()

	// Verify with tampered payload
	err = verifier.Verify(tamperedPayload, rawSignature, &privateKey.PublicKey)
	assert.Error(t, err, "Invalid signature should fail verification")
	assert.Contains(t, err.Error(), "verification failed", "Error should indicate verification failure")
	helpers.LogSuccess(t, "Invalid signature correctly rejected")
}

func TestECDSAVerifier_Supports(t *testing.T) {
	// 사양 요구사항: ECDSA 공개키 타입 지원 확인
	helpers.LogTestSection(t, "6.2.4", "HPKE ECDSA Verifier - Key Type Support")

	verifier := NewECDSAVerifier()

	tests := []struct {
		name      string
		keyGen    func() interface{}
		supported bool
	}{
		{
			name: "ECDSA Secp256k1 key",
			keyGen: func() interface{} {
				key, _ := ethcrypto.GenerateKey()
				return &key.PublicKey
			},
			supported: true,
		},
		{
			name: "Ed25519 key",
			keyGen: func() interface{} {
				pub, _, _ := ed25519.GenerateKey(rand.Reader)
				return pub
			},
			supported: false,
		},
		{
			name: "Invalid key type",
			keyGen: func() interface{} {
				return "not a key"
			},
			supported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := tt.keyGen()
			result := verifier.Supports(key)
			assert.Equal(t, tt.supported, result, "Support check should match expected")
		})
	}
	helpers.LogSuccess(t, "Key type support validation passed")
}

func TestCompositeVerifier_SelectsCorrectVerifier(t *testing.T) {
	// 사양 요구사항: 키 타입에 따라 올바른 검증기 선택
	helpers.LogTestSection(t, "6.2.5", "HPKE Composite Verifier - Strategy Pattern")

	composite := NewCompositeVerifier()

	t.Run("Ed25519 key uses Ed25519 verifier", func(t *testing.T) {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		payload := []byte("Ed25519 test message")
		signature := ed25519.Sign(privateKey, payload)

		err = composite.Verify(payload, signature, publicKey)
		assert.NoError(t, err, "Ed25519 signature should verify through composite")
		helpers.LogSuccess(t, "Ed25519 verifier selected correctly")
	})

	t.Run("ECDSA key uses ECDSA verifier", func(t *testing.T) {
		privateKey, err := ethcrypto.GenerateKey()
		require.NoError(t, err)

		payload := []byte("ECDSA test message")
		hash := ethcrypto.Keccak256(payload)
		signature, err := ethcrypto.Sign(hash, privateKey)
		require.NoError(t, err)
		rawSignature := signature[:64]

		err = composite.Verify(payload, rawSignature, &privateKey.PublicKey)
		assert.NoError(t, err, "ECDSA signature should verify through composite")
		helpers.LogSuccess(t, "ECDSA verifier selected correctly")
	})

	t.Run("Unsupported key type returns error", func(t *testing.T) {
		err := composite.Verify([]byte("test"), []byte("sig"), "invalid-key")
		assert.Error(t, err, "Unsupported key type should return error")
		assert.Contains(t, err.Error(), "unsupported", "Error should indicate unsupported key")
		helpers.LogSuccess(t, "Unsupported key type handled correctly")
	})
}

func TestVerifySignature_WithECDSA(t *testing.T) {
	// 사양 요구사항: verifySignature 함수가 ECDSA 지원
	helpers.LogTestSection(t, "6.2.6", "HPKE verifySignature Integration - ECDSA Support")

	privateKey, err := ethcrypto.GenerateKey()
	require.NoError(t, err)

	payload := []byte("Integration test with verifySignature")
	hash := ethcrypto.Keccak256(payload)

	signature, err := ethcrypto.Sign(hash, privateKey)
	require.NoError(t, err)
	rawSignature := signature[:64]

	// Test the global verifySignature function
	err = verifySignature(payload, rawSignature, &privateKey.PublicKey)
	assert.NoError(t, err, "verifySignature should support ECDSA keys")
	helpers.LogSuccess(t, "verifySignature function supports ECDSA")
}

func TestECDSAVerifier_Verify_With65ByteSignature(t *testing.T) {
	// 사양 요구사항: 65바이트 Ethereum 서명 처리 (recovery ID 포함)
	helpers.LogTestSection(t, "6.2.7", "HPKE ECDSA Signature - 65-byte Ethereum Format")

	privateKey, err := ethcrypto.GenerateKey()
	require.NoError(t, err)

	payload := []byte("Ethereum signature with recovery ID")
	hash := ethcrypto.Keccak256(payload)

	// 65-byte signature (includes recovery ID)
	signature65, err := ethcrypto.Sign(hash, privateKey)
	require.NoError(t, err)
	require.Equal(t, 65, len(signature65))

	verifier := NewECDSAVerifier()

	// Should handle 65-byte format by stripping last byte
	err = verifier.Verify(payload, signature65, &privateKey.PublicKey)
	assert.NoError(t, err, "65-byte signature should be handled automatically")
	helpers.LogSuccess(t, "65-byte Ethereum signature format handled correctly")
}

func BenchmarkECDSAVerifier_Verify(b *testing.B) {
	privateKey, _ := ethcrypto.GenerateKey()
	payload := []byte("Benchmark ECDSA verification")
	hash := ethcrypto.Keccak256(payload)
	signature, _ := ethcrypto.Sign(hash, privateKey)
	rawSignature := signature[:64]

	verifier := NewECDSAVerifier()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = verifier.Verify(payload, rawSignature, &privateKey.PublicKey)
	}
}

func BenchmarkCompositeVerifier_Verify_ECDSA(b *testing.B) {
	privateKey, _ := ethcrypto.GenerateKey()
	payload := []byte("Benchmark composite verifier with ECDSA")
	hash := ethcrypto.Keccak256(payload)
	signature, _ := ethcrypto.Sign(hash, privateKey)
	rawSignature := signature[:64]

	composite := NewCompositeVerifier()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = composite.Verify(payload, rawSignature, &privateKey.PublicKey)
	}
}
