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
	"net/http"
	"strings"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	// Test 2.1.1: Ed25519 signature/verification
	t.Run("Ed25519 end-to-end", func(t *testing.T) {
		// Generate Ed25519 key pair
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		// Create request
		req, err := http.NewRequest("GET", "https://sage.dev/resource/123?user=alice", nil)
		require.NoError(t, err)

		req.Header.Set("Host", "sage.dev")
		req.Header.Set("Date", time.Now().Format(http.TimeFormat))

		// Sign request
		params := &SignatureInputParams{
			CoveredComponents: []string{`"@method"`, `"host"`, `"date"`, `"@path"`, `"@query"`},
			KeyID:             "test-key-ed25519",
			Algorithm:         "ed25519",
			Created:           time.Now().Unix(),
		}

		verifier := NewHTTPVerifier()
		err = verifier.SignRequest(req, "sig1", params, privateKey)
		require.NoError(t, err)

		// Verify request
		err = verifier.VerifyRequest(req, publicKey, nil)
		assert.NoError(t, err)
	})

	// Test 2.1.2: ECDSA P-256 signature/verification
	t.Run("ECDSA P-256 end-to-end", func(t *testing.T) {
		// Generate ECDSA P-256 key pair
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		publicKey := &privateKey.PublicKey

		// Create POST request with body
		body := `{"a":1}`
		req, err := http.NewRequest("POST", "https://sage.dev/data", strings.NewReader(body))
		require.NoError(t, err)

		req.Header.Set("Date", time.Now().Format(http.TimeFormat))
		req.Header.Set("Content-Digest", "sha-256=:RBsLjMq4VvLtwL6W0heDElJPTe2WbHL7gWRYYhHbAw0=:")
		req.Header.Set("Content-Type", "application/json")

		// Sign request
		params := &SignatureInputParams{
			CoveredComponents: []string{`"date"`, `"content-digest"`},
			KeyID:             "test-key-ecdsa",
			Algorithm:         "", // Empty algorithm - will be inferred from key type
			Created:           time.Now().Unix(),
		}

		verifier := NewHTTPVerifier()
		err = verifier.SignRequest(req, "sig1", params, privateKey)
		require.NoError(t, err)

		// Verify request
		err = verifier.VerifyRequest(req, publicKey, nil)
		assert.NoError(t, err)
	})

	// Test 2.1.3: ECDSA Secp256k1 signature/verification (Ethereum compatible)
	t.Run("ECDSA Secp256k1 end-to-end", func(t *testing.T) {
		// Generate ECDSA Secp256k1 key pair (Ethereum compatible)
		privateKeyEth, err := ethcrypto.GenerateKey()
		require.NoError(t, err)

		// Convert to standard ecdsa.PrivateKey for RFC 9421
		privateKey := privateKeyEth
		publicKey := &privateKey.PublicKey

		// Get Ethereum address from public key
		ethAddress := ethcrypto.PubkeyToAddress(*publicKey).Hex()

		// Create POST request with body (Ethereum transaction format)
		body := `{"action":"transfer","amount":100,"to":"0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"}`
		req, err := http.NewRequest("POST", "https://ethereum.sage.dev/transaction", strings.NewReader(body))
		require.NoError(t, err)

		req.Header.Set("Date", time.Now().Format(http.TimeFormat))
		req.Header.Set("Content-Digest", "sha-256=:k8H1234567890abcdefghijklmnopqrstuvwxyz+/=:")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Ethereum-Address", ethAddress)

		// Sign request with Secp256k1 (Ethereum curve)
		params := &SignatureInputParams{
			CoveredComponents: []string{`"@method"`, `"@path"`, `"date"`, `"content-digest"`, `"x-ethereum-address"`},
			KeyID:             "ethereum-key-secp256k1",
			Algorithm:         "es256k", // RFC 9421 algorithm for Secp256k1 (Ethereum-compatible)
			Created:           time.Now().Unix(),
		}

		verifier := NewHTTPVerifier()
		err = verifier.SignRequest(req, "sig1", params, privateKey)
		require.NoError(t, err)

		// Verify request
		err = verifier.VerifyRequest(req, publicKey, nil)
		assert.NoError(t, err)

		// Verify that signature-input contains the ethereum address header
		sigInput := req.Header.Get("Signature-Input")
		assert.Contains(t, sigInput, "x-ethereum-address", "Signature should cover Ethereum address")

		// Verify signature header exists
		signature := req.Header.Get("Signature")
		assert.NotEmpty(t, signature, "Signature header must be present")

		// Verify Ethereum address format
		assert.True(t, strings.HasPrefix(ethAddress, "0x"), "Ethereum address must start with 0x")
		assert.Len(t, ethAddress, 42, "Ethereum address must be 42 characters (0x + 40 hex chars)")
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

		// Change a character in the middle (flip first bit of 11th character)
		modifiedB64 := b64Part[:10] + "X" + b64Part[11:]
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
