package auth0

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

// helper to extract form values from request body
func parseFormBody(t *testing.T, r *http.Request) url.Values {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        t.Fatalf("failed to read body: %v", err)
    }
    vals, err := url.ParseQuery(string(body))
    if err != nil {
        t.Fatalf("failed to parse query: %v", err)
    }
    return vals
}

func TestAuth0Provider(t *testing.T) {
    ctx := context.Background()

    t.Run("RFC-8707 Token Exchange - missing params", func(t *testing.T) {
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            vals := parseFormBody(t, r)
            assert.Equal(t, "urn:ietf:params:oauth:grant-type:token-exchange", vals.Get("grant_type"))
            assert.Equal(t, "subject-value", vals.Get("subject_token"))
            assert.Equal(t, "urn:ietf:params:oauth:token-type:access_token", vals.Get("subject_token_type"))
            assert.Equal(t, "audience-value", vals.Get("audience"))
            assert.Equal(t, "client-id", vals.Get("client_id"))
            assert.Equal(t, "client-secret", vals.Get("client_secret"))
            http.Error(w, "bad request", http.StatusBadRequest)
        }))
        defer server.Close()

        provider := NewProvider(server.URL, "client-id", "client-secret")
        _, err := provider.ExchangeToken(ctx, "subject-value", "urn:ietf:params:oauth:token-type:access_token", "audience-value")
        assert.Error(t, err)
    })

    t.Run("RFC-8707 Token Exchange - success", func(t *testing.T) {
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            vals := parseFormBody(t, r)
            assert.Equal(t, "urn:ietf:params:oauth:grant-type:token-exchange", vals.Get("grant_type"))
            assert.Equal(t, "abc123", vals.Get("subject_token"))
            assert.Equal(t, "urn:ietf:params:oauth:token-type:refresh_token", vals.Get("subject_token_type"))
            assert.Equal(t, "api://default", vals.Get("audience"))
            assert.Equal(t, "client-id", vals.Get("client_id"))
            assert.Equal(t, "client-secret", vals.Get("client_secret"))

            resp := oauth2.Token{
                AccessToken:  "new-access-token",
                TokenType:    "Bearer",
                RefreshToken: "new-refresh-token",
                Expiry:       time.Now().Add(time.Hour),
            }
            w.Header().Set("Content-Type", "application/json")
            json.NewEncoder(w).Encode(resp)
        }))
        defer server.Close()

        provider := NewProvider(server.URL, "client-id", "client-secret")
        token, err := provider.ExchangeToken(ctx, "abc123", "urn:ietf:params:oauth:token-type:refresh_token", "api://default")
        assert.NoError(t, err)
        assert.Equal(t, "new-access-token", token.AccessToken)
        assert.Equal(t, "new-refresh-token", token.RefreshToken)
        assert.Equal(t, "Bearer", token.TokenType)
    })

    t.Run("VerifyIDToken - invalid header", func(t *testing.T) {
        provider := NewProvider("https://example.com", "id", "secret")
        _, err := provider.VerifyIDToken(ctx, "not-a-jwt")
        assert.Error(t, err)
    })

    t.Run("VerifyIDToken - no matching JWK", func(t *testing.T) {
        jwks := map[string][]formats.JWK{"keys": {{
            Kty: "OKP", Crv: "Ed25519", X: "AAA", Kid: "other-kid", Alg: "EdDSA",
        }}}
        jwksJSON, _ := json.Marshal(jwks)

        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Content-Type", "application/json")
            w.Write(jwksJSON)
        }))
        defer server.Close()

        provider := NewProvider(server.URL, "id", "secret")
        token := jwt.New(jwt.SigningMethodEdDSA)
        token.Header["kid"] = "wanted-kid"
        raw, _ := token.SigningString()

        _, err := provider.VerifyIDToken(ctx, raw+".")
        assert.Error(t, err)
    })

    t.Run("VerifyIDToken - expired token", func(t *testing.T) {
        pub, priv, _ := ed25519.GenerateKey(rand.Reader)
        jwk := formats.JWK{
            Kty: "OKP", Crv: "Ed25519", X: base64.RawURLEncoding.EncodeToString(pub),
            Kid: "exp-kid", Alg: "EdDSA",
        }
        jwks := map[string][]formats.JWK{"keys": {jwk}}
        jwksJSON, _ := json.Marshal(jwks)
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Content-Type", "application/json")
            w.Write(jwksJSON)
        }))
        defer server.Close()

        provider := NewProvider(server.URL, "id", "secret")

        // Create expired token
        claims := jwt.MapClaims{
            "sub": "u",
            "exp": time.Now().Add(-time.Hour).Unix(),
            "aud": "id",
            "iss": server.URL,
        }
        token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
        token.Header["kid"] = "exp-kid"
        raw, _ := token.SignedString(priv)

        _, err := provider.VerifyIDToken(ctx, raw)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "expired")
    })


    t.Run("VerifyIDToken - audience mismatch", func(t *testing.T) {
        pub, priv, _ := ed25519.GenerateKey(rand.Reader)
        jwk := formats.JWK{
            Kty: "OKP", Crv: "Ed25519", X: base64.RawURLEncoding.EncodeToString(pub),
            Kid: "aud-kid", Alg: "EdDSA",
        }
        jwks := map[string][]formats.JWK{"keys": {jwk}}
        jwksJSON, _ := json.Marshal(jwks)
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Content-Type", "application/json")
            w.Write(jwksJSON)
        }))
        defer server.Close()

        provider := NewProvider(server.URL, "correct-client", "secret")
        claims := jwt.MapClaims{"sub": "u", "exp": time.Now().Add(time.Hour).Unix(), "aud": "other-client"}
        token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
        token.Header["kid"] = "aud-kid"
        raw, _ := token.SignedString(priv)

        _, err := provider.VerifyIDToken(ctx, raw)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "audience mismatch")
    })

    t.Run("VerifyIDToken - issuer trailing slash", func(t *testing.T) {
        pub, priv, _ := ed25519.GenerateKey(rand.Reader)
        jwk := formats.JWK{
            Kty: "OKP", Crv: "Ed25519", X: base64.RawURLEncoding.EncodeToString(pub),
            Kid: "iss-kid", Alg: "EdDSA",
        }
        jwks := map[string][]formats.JWK{"keys": {jwk}}
        jwksJSON, _ := json.Marshal(jwks)
        server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Content-Type", "application/json")
            w.Write(jwksJSON)
        }))
        defer server.Close()

        provider := NewProvider(server.URL, "id", "secret")
        // issuer with trailing slash
        iss := server.URL + "/"
        claims := jwt.MapClaims{"sub": "u", "exp": time.Now().Add(time.Hour).Unix(), "aud": "id", "iss": iss}
        token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
        token.Header["kid"] = "iss-kid"
        raw, _ := token.SignedString(priv)

        parsed, err := provider.VerifyIDToken(ctx, raw)
        assert.NoError(t, err)
        assert.Equal(t, "u", parsed["sub"])
    })
}
