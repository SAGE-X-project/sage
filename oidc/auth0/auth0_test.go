// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


package auth0

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/sage-x-project/sage/crypto/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgent_RequestToken tests the JWT Bearer grant with RFC-8707 resource and DID.
func TestAgent_RequestToken(t *testing.T) {
	keyPair, err := keys.GenerateRSAKeyPair()
	require.NoError(t, err)

	exporter := formats.NewJWKExporter()
	exportedPub, err := exporter.ExportPublic(keyPair, sagecrypto.KeyFormatJWK)
	require.NoError(t, err)
	assert.NotEmpty(t, exportedPub)

	var jwk formats.JWK
	require.NoError(t, json.Unmarshal(exportedPub, &jwk))
	require.NotEmpty(t, jwk.Kid)

	rsaPriv, ok := keyPair.PrivateKey().(*rsa.PrivateKey)
	require.True(t, ok)

	var jwksHits int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/jwks.json" {
			atomic.AddInt32(&jwksHits, 1)
			_ = json.NewEncoder(w).Encode(struct {
				Keys []formats.JWK `json:"keys"`
			}{Keys: []formats.JWK{jwk}})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(ts.Close)

	issuer := ts.URL + "/" 
	audience := "https://api.example.test"
	sub := "client-123@clients"

	v := NewVerifier(VerifierConfig{
		Identifier:  audience,
		CacheTTL:    10 * time.Minute,
		HTTPTimeout: 2 * time.Second,
	})

	ctx := context.Background()

	t.Run("valid token", func(t *testing.T) {
		token := signJWT(t, rsaPriv, jwk.Kid, issuer, audience, sub, time.Minute, map[string]any{
			"did": "did:example:123",
		})

		claims, err := v.Verify(ctx, token, issuer)
		require.NoError(t, err)

		assert.Equal(t, "did:example:123", claims["did"])
		assert.Equal(t, sub, claims["sub"])
		assert.Equal(t, audience, claims["aud"])
		assert.Equal(t, issuer, claims["iss"])
	})

	t.Run("invalid signature", func(t *testing.T) {
		token := signJWT(t, rsaPriv, jwk.Kid, issuer, audience, sub, time.Minute, nil)
		bad, err := TamperSignatureRS256(token)
		assert.NoError(t, err)

		_, err = v.Verify(ctx, bad, issuer)
		require.Error(t, err)
		assert.True(t,
			strings.Contains(err.Error(), "verification") ||
			strings.Contains(err.Error(), "invalid") ||
			strings.Contains(err.Error(), "signature"),
			"unexpected error: %v", err)
	})

	t.Run("invalid audience", func(t *testing.T) {
		token := signJWT(t, rsaPriv, jwk.Kid, issuer, audience, sub, time.Minute, nil)

		vBadAud := NewVerifier(VerifierConfig{
			Identifier:  "https://wrong.example/api",
			CacheTTL:    time.Minute,
			HTTPTimeout: 2 * time.Second,
		})

		_, err := vBadAud.Verify(ctx, token, issuer)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid audience")
	})

	t.Run("expired token", func(t *testing.T) {
		token := signJWT(t, rsaPriv, jwk.Kid, issuer, audience, sub, -1*time.Minute, nil) // already expired
		_, err := v.Verify(ctx, token, issuer)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("issuer mismatch", func(t *testing.T) {
		token := signJWT(t, rsaPriv, jwk.Kid, issuer, audience, sub, time.Minute, nil)
		_, err := v.Verify(ctx, token, issuer+"different")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid issuer")
	})

	t.Run("jwks cache is used on subsequent verifications", func(t *testing.T) {
		start := atomic.LoadInt32(&jwksHits)

		token := signJWT(t, rsaPriv, jwk.Kid, issuer, audience, sub, time.Minute, nil)
		_, err := v.Verify(ctx, token, issuer)
		require.NoError(t, err)

		_, err = v.Verify(ctx, token, issuer)
		require.NoError(t, err)

		end := atomic.LoadInt32(&jwksHits)
		assert.LessOrEqual(t, end-start, int32(1), "jwks should be fetched at most once due to caching")
	})

}


func TamperSignatureRS256(tok string) (string, error) {
    parts := strings.Split(tok, ".")
    if len(parts) != 3 {
        return "", fmt.Errorf("not a JWT: segments=%d", len(parts))
    }
    sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
    if err != nil {
        return "", fmt.Errorf("decode sig: %w", err)
    }
    if len(sigBytes) == 0 {
        return "", fmt.Errorf("empty sig")
    }
    sigBytes[0] ^= 0x01 // flip first byte's bit (ensure verification failure)
    parts[2] = base64.RawURLEncoding.EncodeToString(sigBytes)
    return strings.Join(parts, "."), nil
}

func signJWT(t *testing.T, priv *rsa.PrivateKey, kid, iss, aud, sub string, lifetime time.Duration, extra map[string]any) string {
	t.Helper()
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"iss": iss,
		"sub": sub,
		"aud": aud,
		"iat": now,
		"exp": now + int64(lifetime.Seconds()),
	}
	for k, v := range extra {
		claims[k] = v
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(priv)
	require.NoError(t, err)
	return s
}