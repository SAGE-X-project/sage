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

		// Modify the signature by changing one character in the base64
		sig := req.Header.Get("Signature")
		// Find the base64 part between colons
		start := strings.Index(sig, ":")
		end := strings.LastIndex(sig, ":")
		if start != -1 && end != -1 && start < end {
			b64Part := sig[start+1 : end]
			// Change a character in the middle
			if len(b64Part) > 10 {
				modifiedB64 := b64Part[:10] + "X" + b64Part[11:]
				sig = sig[:start+1] + modifiedB64 + sig[end:]
			}
		}
		req.Header.Set("Signature", sig)

		// Should fail verification
		err = verifier.VerifyRequest(req, publicKey, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature verification failed")
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
