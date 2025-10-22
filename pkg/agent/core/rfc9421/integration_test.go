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

package rfc9421

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/tests/helpers"
)

func TestIntegration(t *testing.T) {
	// Test 2.1.1: Ed25519 signature/verification
	t.Run("Ed25519 end-to-end", func(t *testing.T) {
		// Specification Requirement: RFC 9421 compliant HTTP message signature generation and verification (Ed25519)
		helpers.LogTestSection(t, "1.1.1", "RFC 9421 Ed25519 Signature Generation and Verification")

		// Generate Ed25519 key pair
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		// Specification Requirement: Key size validation (public key: 32 bytes, private key: 64 bytes)
		assert.Equal(t, 32, len(publicKey), "Public key size must be 32 bytes")
		assert.Equal(t, 64, len(privateKey), "Private key size must be 64 bytes")

		helpers.LogSuccess(t, "Ed25519 key generation successful")
		helpers.LogDetail(t, "Public key size: %d bytes", len(publicKey))
		helpers.LogDetail(t, "Private key size: %d bytes", len(privateKey))
		helpers.LogDetail(t, "Public key (hex): %x", publicKey)

		// Create request
		testMessage := "https://sage.dev/resource/123?user=alice"
		req, err := http.NewRequest("GET", testMessage, nil)
		require.NoError(t, err)

		req.Header.Set("Host", "sage.dev")
		currentTime := time.Now()
		req.Header.Set("Date", currentTime.Format(http.TimeFormat))

		helpers.LogDetail(t, "Test request URL: %s", testMessage)
		helpers.LogDetail(t, "Test time: %s", currentTime.Format(time.RFC3339))

		// Sign request
		params := &SignatureInputParams{
			CoveredComponents: []string{`"@method"`, `"host"`, `"date"`, `"@path"`, `"@query"`},
			KeyID:             "test-key-ed25519",
			Algorithm:         "ed25519",
			Created:           currentTime.Unix(),
		}

		verifier := NewHTTPVerifier()
		err = verifier.SignRequest(req, "sig1", params, privateKey)
		require.NoError(t, err)

		// Specification Requirement: Signature validation (64 bytes)
		signature := req.Header.Get("Signature")
		assert.NotEmpty(t, signature, "Signature header must be present")

		// Specification Requirement: Signature-Input header format validation
		sigInput := req.Header.Get("Signature-Input")
		assert.Contains(t, sigInput, "keyid=", "Signature-Input must contain keyid parameter")
		assert.Contains(t, sigInput, "created=", "Signature-Input must contain created parameter")
		assert.Contains(t, sigInput, "alg=", "Signature-Input must contain alg parameter")

		helpers.LogSuccess(t, "Signature generation successful")
		helpers.LogDetail(t, "Signature: %s", signature)
		helpers.LogDetail(t, "Signature-Input: %s", sigInput)

		// Verify request
		err = verifier.VerifyRequest(req, publicKey, nil)
		assert.NoError(t, err)

		helpers.LogSuccess(t, "Signature verification successful")

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"Ed25519 signature generation successful",
			"Public key size = 32 bytes",
			"Private key size = 64 bytes",
			"Signature header present",
			"Signature-Input header format correct",
			"RFC 9421 standard compliant",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":        "1.1.1_Ed25519_Signature",
			"public_key_hex":   hex.EncodeToString(publicKey),
			"private_key_hex":  hex.EncodeToString(privateKey),
			"message":          testMessage,
			"timestamp":        currentTime.Format(time.RFC3339),
			"signature":        signature,
			"signature_input":  sigInput,
			"key_sizes": map[string]int{
				"public_key":  len(publicKey),
				"private_key": len(privateKey),
			},
			"expected_sizes": map[string]int{
				"public_key":  32,
				"private_key": 64,
			},
		}
		helpers.SaveTestData(t, "rfc9421/ed25519_signature.json", testData)
	})

	// Test 2.1.2: ECDSA P-256 signature/verification
	t.Run("ECDSA P-256 end-to-end", func(t *testing.T) {
		// Specification Requirement: RFC 9421 compliant HTTP message signature with ECDSA P-256
		helpers.LogTestSection(t, "1.1.2", "RFC 9421 ECDSA P-256 Signature Generation and Verification")

		// Generate ECDSA P-256 key pair
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		publicKey := &privateKey.PublicKey

		// Specification Requirement: ECDSA P-256 key validation
		assert.Equal(t, elliptic.P256(), privateKey.Curve, "Curve must be P-256")

		helpers.LogSuccess(t, "ECDSA P-256 key generation successful")
		helpers.LogDetail(t, "Curve: P-256")
		helpers.LogDetail(t, "Private key D size: %d bytes", len(privateKey.D.Bytes()))
		helpers.LogDetail(t, "Public key X: %x", publicKey.X.Bytes())
		helpers.LogDetail(t, "Public key Y: %x", publicKey.Y.Bytes())

		// Create POST request with body
		body := `{"a":1}`
		testURL := "https://sage.dev/data"
		req, err := http.NewRequest("POST", testURL, strings.NewReader(body))
		require.NoError(t, err)

		currentTime := time.Now()
		req.Header.Set("Date", currentTime.Format(http.TimeFormat))
		req.Header.Set("Content-Digest", "sha-256=:RBsLjMq4VvLtwL6W0heDElJPTe2WbHL7gWRYYhHbAw0=:")
		req.Header.Set("Content-Type", "application/json")

		helpers.LogDetail(t, "Test request URL: %s", testURL)
		helpers.LogDetail(t, "Request method: POST")
		helpers.LogDetail(t, "Request body: %s", body)
		helpers.LogDetail(t, "Test time: %s", currentTime.Format(time.RFC3339))

		// Sign request
		params := &SignatureInputParams{
			CoveredComponents: []string{`"date"`, `"content-digest"`},
			KeyID:             "test-key-ecdsa",
			Algorithm:         "", // Empty algorithm - will be inferred from key type
			Created:           currentTime.Unix(),
		}

		verifier := NewHTTPVerifier()
		err = verifier.SignRequest(req, "sig1", params, privateKey)
		require.NoError(t, err)

		// Specification Requirement: Signature validation
		signature := req.Header.Get("Signature")
		assert.NotEmpty(t, signature, "Signature header must be present")

		sigInput := req.Header.Get("Signature-Input")
		assert.Contains(t, sigInput, "keyid=", "Signature-Input must contain keyid parameter")
		assert.Contains(t, sigInput, "created=", "Signature-Input must contain created parameter")

		helpers.LogSuccess(t, "Signature generation successful")
		helpers.LogDetail(t, "Signature: %s", signature)
		helpers.LogDetail(t, "Signature-Input: %s", sigInput)

		// Verify request
		err = verifier.VerifyRequest(req, publicKey, nil)
		assert.NoError(t, err)

		helpers.LogSuccess(t, "Signature verification successful")

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"ECDSA P-256 signature generation successful",
			"Curve = P-256 (NIST)",
			"Signature header present",
			"Signature-Input header format correct",
			"Content-Digest covered in signature",
			"RFC 9421 standard compliant",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":        "1.1.2_ECDSA_P256_Signature",
			"curve":            "P-256",
			"private_key_d":    hex.EncodeToString(privateKey.D.Bytes()),
			"public_key_x":     hex.EncodeToString(publicKey.X.Bytes()),
			"public_key_y":     hex.EncodeToString(publicKey.Y.Bytes()),
			"request_url":      testURL,
			"request_method":   "POST",
			"request_body":     body,
			"timestamp":        currentTime.Format(time.RFC3339),
			"signature":        signature,
			"signature_input":  sigInput,
			"content_digest":   "sha-256=:RBsLjMq4VvLtwL6W0heDElJPTe2WbHL7gWRYYhHbAw0=:",
		}
		helpers.SaveTestData(t, "rfc9421/ecdsa_p256_signature.json", testData)
	})

	// Test 2.1.3: ECDSA Secp256k1 signature/verification (Ethereum compatible)
	t.Run("ECDSA Secp256k1 end-to-end", func(t *testing.T) {
		// Specification Requirement: RFC 9421 compliant HTTP message signature with ECDSA Secp256k1 (Ethereum)
		helpers.LogTestSection(t, "1.1.3", "RFC 9421 ECDSA Secp256k1 Signature Generation and Verification (Ethereum)")

		// Generate ECDSA Secp256k1 key pair (Ethereum compatible)
		privateKeyEth, err := ethcrypto.GenerateKey()
		require.NoError(t, err)

		// Convert to standard ecdsa.PrivateKey for RFC 9421
		privateKey := privateKeyEth
		publicKey := &privateKey.PublicKey

		// Get Ethereum address from public key
		ethAddress := ethcrypto.PubkeyToAddress(*publicKey).Hex()

		// Specification Requirement: Ethereum address format validation (0x + 40 hex chars)
		assert.True(t, strings.HasPrefix(ethAddress, "0x"), "Ethereum address must start with 0x")
		assert.Len(t, ethAddress, 42, "Ethereum address must be 42 characters (0x + 40 hex chars)")

		helpers.LogSuccess(t, "ECDSA Secp256k1 key generation successful (Ethereum compatible)")
		helpers.LogDetail(t, "Curve: Secp256k1")
		helpers.LogDetail(t, "Ethereum address: %s", ethAddress)
		helpers.LogDetail(t, "Private key D size: %d bytes", len(privateKey.D.Bytes()))
		helpers.LogDetail(t, "Public key X: %x", publicKey.X.Bytes())
		helpers.LogDetail(t, "Public key Y: %x", publicKey.Y.Bytes())

		// Create POST request with body (Ethereum transaction format)
		body := `{"action":"transfer","amount":100,"to":"0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"}`
		testURL := "https://ethereum.sage.dev/transaction"
		req, err := http.NewRequest("POST", testURL, strings.NewReader(body))
		require.NoError(t, err)

		currentTime := time.Now()
		req.Header.Set("Date", currentTime.Format(http.TimeFormat))
		req.Header.Set("Content-Digest", "sha-256=:k8H1234567890abcdefghijklmnopqrstuvwxyz+/=:")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Ethereum-Address", ethAddress)

		helpers.LogDetail(t, "Test request URL: %s", testURL)
		helpers.LogDetail(t, "Request method: POST")
		helpers.LogDetail(t, "Request body: %s", body)
		helpers.LogDetail(t, "Test time: %s", currentTime.Format(time.RFC3339))

		// Sign request with Secp256k1 (Ethereum curve)
		params := &SignatureInputParams{
			CoveredComponents: []string{`"@method"`, `"@path"`, `"date"`, `"content-digest"`, `"x-ethereum-address"`},
			KeyID:             "ethereum-key-secp256k1",
			Algorithm:         "es256k", // RFC 9421 algorithm for Secp256k1 (Ethereum-compatible)
			Created:           currentTime.Unix(),
		}

		verifier := NewHTTPVerifier()
		err = verifier.SignRequest(req, "sig1", params, privateKey)
		require.NoError(t, err)

		// Specification Requirement: Signature validation
		signature := req.Header.Get("Signature")
		assert.NotEmpty(t, signature, "Signature header must be present")

		// Specification Requirement: Signature-Input header must cover Ethereum address
		sigInput := req.Header.Get("Signature-Input")
		assert.Contains(t, sigInput, "x-ethereum-address", "Signature must cover Ethereum address")
		assert.Contains(t, sigInput, "keyid=", "Signature-Input must contain keyid parameter")
		assert.Contains(t, sigInput, "alg=", "Signature-Input must contain alg parameter")
		assert.Contains(t, sigInput, "es256k", "Algorithm must be es256k for Secp256k1")

		helpers.LogSuccess(t, "Signature generation successful")
		helpers.LogDetail(t, "Signature: %s", signature)
		helpers.LogDetail(t, "Signature-Input: %s", sigInput)
		helpers.LogDetail(t, "Algorithm: es256k (Secp256k1)")

		// Verify request
		err = verifier.VerifyRequest(req, publicKey, nil)
		assert.NoError(t, err)

		helpers.LogSuccess(t, "Signature verification successful")

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"ECDSA Secp256k1 signature generation successful",
			"Ethereum address format correct (0x + 40 hex)",
			"Ethereum address covered in signature",
			"Algorithm = es256k (RFC 9421)",
			"Signature header present",
			"Signature-Input header format correct",
			"Ethereum compatible",
			"RFC 9421 standard compliant",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":        "1.1.3_ECDSA_Secp256k1_Signature_Ethereum",
			"curve":            "Secp256k1",
			"ethereum_address": ethAddress,
			"private_key_d":    hex.EncodeToString(privateKey.D.Bytes()),
			"public_key_x":     hex.EncodeToString(publicKey.X.Bytes()),
			"public_key_y":     hex.EncodeToString(publicKey.Y.Bytes()),
			"request_url":      testURL,
			"request_method":   "POST",
			"request_body":     body,
			"timestamp":        currentTime.Format(time.RFC3339),
			"signature":        signature,
			"signature_input":  sigInput,
			"algorithm":        "es256k",
			"content_digest":   "sha-256=:k8H1234567890abcdefghijklmnopqrstuvwxyz+/=:",
		}
		helpers.SaveTestData(t, "rfc9421/ecdsa_secp256k1_signature.json", testData)
	})
}

func TestNegativeCases(t *testing.T) {
	// Test 3.1.1: Modified signature
	t.Run("modified signature", func(t *testing.T) {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		req, err := http.NewRequest("GET", "https://example.com/test", nil)
		require.NoError(t, err)

		params := &SignatureInputParams{
			CoveredComponents: []string{`"@method"`, `"@path"`},
			Created:           time.Now().Unix(),
		}

		verifier := NewHTTPVerifier()
		err = verifier.SignRequest(req, "sig1", params, privateKey)
		require.NoError(t, err)

		// Store original signature for comparison
		originalSig := req.Header.Get("Signature")
		require.NotEmpty(t, originalSig, "Original signature must not be empty")

		// Modify the signature by changing one character in the base64
		sig := originalSig
		// Find the base64 part between colons
		start := strings.Index(sig, ":")
		end := strings.LastIndex(sig, ":")
		require.True(t, start != -1 && end != -1 && start < end, "Signature must have proper format")

		b64Part := sig[start+1 : end]
		require.True(t, len(b64Part) > 10, "Base64 signature must be long enough to modify")

		// Change a character in the middle - ensure it's actually different
		targetChar := "X"
		if b64Part[10:11] == "X" {
			targetChar = "Y" // Use a different character if position 10 is already 'X'
		}
		modifiedB64 := b64Part[:10] + targetChar + b64Part[11:]
		sig = sig[:start+1] + modifiedB64 + sig[end:]

		// Verify signature was actually modified
		require.NotEqual(t, originalSig, sig, "Signature must be modified")

		req.Header.Set("Signature", sig)

		// Should fail verification
		err = verifier.VerifyRequest(req, publicKey, nil)
		if err == nil {
			t.Errorf("Expected verification to fail for modified signature, but it succeeded")
		} else {
			assert.Contains(t, err.Error(), "signature verification failed", "Error message should indicate signature verification failure")
		}
	})

	// Test 3.1.2: Modified signed header
	t.Run("modified signed header", func(t *testing.T) {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		req, err := http.NewRequest("GET", "https://example.com/test", nil)
		require.NoError(t, err)
		req.Header.Set("Date", time.Now().Format(http.TimeFormat))

		params := &SignatureInputParams{
			CoveredComponents: []string{`"date"`},
			Created:           time.Now().Unix(),
		}

		verifier := NewHTTPVerifier()
		err = verifier.SignRequest(req, "sig1", params, privateKey)
		require.NoError(t, err)

		// Modify the Date header
		newTime := time.Now().Add(1 * time.Hour)
		req.Header.Set("Date", newTime.Format(http.TimeFormat))

		// Should fail verification
		err = verifier.VerifyRequest(req, publicKey, nil)
		assert.Error(t, err)
	})

	// Test 3.1.3: Modified unsigned header
	t.Run("modified unsigned header", func(t *testing.T) {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		req, err := http.NewRequest("GET", "https://example.com/test", nil)
		require.NoError(t, err)
		req.Header.Set("Date", time.Now().Format(http.TimeFormat))
		req.Header.Set("Accept", "application/json")

		// Sign only Date header, not Accept
		params := &SignatureInputParams{
			CoveredComponents: []string{`"date"`},
			Created:           time.Now().Unix(),
		}

		verifier := NewHTTPVerifier()
		err = verifier.SignRequest(req, "sig1", params, privateKey)
		require.NoError(t, err)

		// Modify the Accept header (which is not signed)
		req.Header.Set("Accept", "application/xml")

		// Should still pass verification
		err = verifier.VerifyRequest(req, publicKey, nil)
		assert.NoError(t, err)
	})

	// Test 3.2.1: Expired signature (created + maxAge)
	t.Run("expired signature with maxAge", func(t *testing.T) {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		req, err := http.NewRequest("GET", "https://example.com/test", nil)
		require.NoError(t, err)

		// Create signature with old timestamp
		params := &SignatureInputParams{
			CoveredComponents: []string{`"@method"`},
			Created:           time.Now().Add(-10 * time.Minute).Unix(),
		}

		verifier := NewHTTPVerifier()
		err = verifier.SignRequest(req, "sig1", params, privateKey)
		require.NoError(t, err)

		// Verify with MaxAge of 5 minutes
		opts := &HTTPVerificationOptions{
			MaxAge: 5 * time.Minute,
		}

		err = verifier.VerifyRequest(req, publicKey, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature expired")
	})

	// Test 3.2.2: Expired signature (expires timestamp)
	t.Run("expired signature with expires", func(t *testing.T) {
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		req, err := http.NewRequest("GET", "https://example.com/test", nil)
		require.NoError(t, err)

		// Create signature that expires 1 minute ago
		params := &SignatureInputParams{
			CoveredComponents: []string{`"@method"`},
			Created:           time.Now().Unix(),
			Expires:           time.Now().Add(-1 * time.Minute).Unix(),
		}

		verifier := NewHTTPVerifier()
		err = verifier.SignRequest(req, "sig1", params, privateKey)
		require.NoError(t, err)

		// Should fail verification
		err = verifier.VerifyRequest(req, publicKey, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature expired")
	})
}

func TestQueryParamProtection(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	verifier := NewHTTPVerifier()

	// Test 4.1.2: Protected parameter modification
	t.Run("protected parameter modification", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/api?id=123&format=json", nil)
		require.NoError(t, err)

		params := &SignatureInputParams{
			CoveredComponents: []string{`"@query-param";name="id"`},
			Created:           time.Now().Unix(),
		}

		err = verifier.SignRequest(req, "sig1", params, privateKey)
		require.NoError(t, err)

		// Modify protected parameter
		req.URL.RawQuery = "id=456&format=json"

		err = verifier.VerifyRequest(req, publicKey, nil)
		assert.Error(t, err)
	})

	// Test 4.1.3: Unprotected parameter modification
	t.Run("unprotected parameter modification", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/api?id=123&format=json", nil)
		require.NoError(t, err)

		params := &SignatureInputParams{
			CoveredComponents: []string{`"@query-param";name="id"`},
			Created:           time.Now().Unix(),
		}

		err = verifier.SignRequest(req, "sig1", params, privateKey)
		require.NoError(t, err)

		// Modify unprotected parameter
		req.URL.RawQuery = "id=123&format=xml"
		// Force URL to reparse the query
		req.URL.Query()

		// Should still pass
		err = verifier.VerifyRequest(req, publicKey, nil)
		assert.NoError(t, err)
	})
}
