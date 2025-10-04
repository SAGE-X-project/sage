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
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"github.com/sage-x-project/sage/oidc"
)

// Config holds the settings required by the Agent.
type Config struct {
	Domain         string 
	KeyId		   string
    ClientID       string // Auth0 Application Client ID
    ClientSecret   string // Auth0 Application Client Secret
    PrivateKeyPEM  string // PEM-encoded RSA private key for signing assertions
    DID            string // Decentralized Identifier to include in the token
    Resource       string // RFC-8707 resource indicator (e.g. "api://orders")
    HTTPTimeout    time.Duration // HTTP client timeout
}

// Agent performs JWT Bearer grant requests to Auth0.
type Agent struct {
    cfg    Config
    http   *http.Client
    importer  sagecrypto.KeyImporter
}

// NewAgent creates a new Agent with the given configuration.
func NewAgent(cfg Config) *Agent {
    return &Agent{
        cfg: cfg,
        http: &http.Client{Timeout: cfg.HTTPTimeout},
        importer: formats.NewPEMImporter(),
    }
}

// RequestToken performs a JWT Bearer grant with RFC-8707 resource and DID.
// authorizationURL is provided for completeness but not used by JWT Bearer grant.
// tokenURL is the Auth0 /oauth/token endpoint.
// Returns the raw access token (JWT).
func (a *Agent) RequestToken(ctx context.Context, tokenURL string, audience string) (string, error) {
    keyPair, err := a.importer.Import([]byte(a.cfg.PrivateKeyPEM), sagecrypto.KeyFormatPEM)
    if err != nil {
        return "", fmt.Errorf("import private key: %w", err)
    }
    signer := keyPair.PrivateKey().(crypto.Signer)

    now := time.Now().Unix()
    claims := jwt.MapClaims{
        "iss": a.cfg.ClientID,
		"sub": a.cfg.ClientID,
        "aud": tokenURL,
        "iat": now,
        "exp": now + 60,
		"jti": uuid.NewString(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
    token.Header["kid"] = a.cfg.KeyId

    assertion, err := token.SignedString(signer)
    if err != nil {
        return "", fmt.Errorf("sign assertion: %w", err)
    }

    form := url.Values{
        "grant_type":            {"client_credentials"},
		"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
		"client_assertion":      {assertion},
		"audience":              {audience}, 
		"did":                   {a.cfg.DID},  
        // "scope":         {"openid"},
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
    if err != nil {
        return "", fmt.Errorf("new request: %w", err)
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := a.http.Do(req)
    if err != nil {
        return "", fmt.Errorf("do request: %w", err)
    }
    defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}
	// fmt.Println(string(bodyBytes))

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("token endpoint returned status %d", resp.StatusCode)
    }

    var respData struct {
    	AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}

	if err := json.Unmarshal(bodyBytes, &respData); err != nil {
		return "", fmt.Errorf("unmarshal token response: %w", err)
	}

	return respData.AccessToken, nil
}



// VerifierConfig holds settings for token verification.
type VerifierConfig struct {
	Identifier  string        // expected 'aud' (API Identifier, e.g. https://<domain>/api/v2/)
	CacheTTL    time.Duration // e.g. 10 * time.Minute
	HTTPTimeout time.Duration // e.g. 10 * time.Second
}

// verifier caches JWKS and verifies JWTs.
type verifier struct {
	cfg       VerifierConfig
	http      *http.Client
	cache     []formats.JWK
	expiresAt time.Time
	mu        sync.RWMutex
}

// NewVerifier creates a new JWT verifier.
func NewVerifier(cfg VerifierConfig) *verifier {
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 10 * time.Minute
	}
	httpClient := &http.Client{Timeout: cfg.HTTPTimeout}
	if cfg.HTTPTimeout == 0 {
		httpClient.Timeout = 10 * time.Second
	}
	return &verifier{
		cfg:  cfg,
		http: httpClient,
	}
}

// Verify parses and validates the JWT, returning claims map on success.
func (v *verifier) Verify(ctx context.Context, tokenString string, issuer string) (map[string]interface{}, error) {
	parser := jwt.NewParser()
	unverified, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("parse token header: %w", err)
	}
	kid, _ := unverified.Header["kid"].(string)
	if kid == "" {
		return nil, oidc.ErrMissingKid
	}
	if typ, _ := unverified.Header["typ"].(string); typ != "" && typ != "JWT" {
		return nil, oidc.ErrUnexpectedTyp
	}

	pubKey := v.lookupKeyFromCache(kid)
	token, err := v.parseAndVerifyWithKey(tokenString, pubKey)
	if err != nil || token == nil || !token.Valid {
		keys, ferr := v.getJWKS(ctx, issuer)
		if ferr != nil {
			return nil, fmt.Errorf("get jwks: %w", ferr)
		}
		pubKey = findKeyByKID(keys, kid)
		if pubKey == nil {
			return nil, oidc.ErrNoMatchingJWK
		}
		token, err = v.parseAndVerifyWithKey(tokenString, pubKey)
		if err != nil {
			return nil, fmt.Errorf("token verification failed: %w", err)
		}
		if !token.Valid {
			return nil, oidc.ErrInvalidToken
		}
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	if !has(claims, v.cfg.Identifier) {
		return nil, fmt.Errorf("%w: expected %s, got %v", oidc.ErrInvalidAudience, v.cfg.Identifier, claims["aud"])
	}

	now := time.Now().Unix()
	const leeway = int64(60)

	exp, ok := toInt64(claims["exp"])
	if !ok || exp <= now-leeway {
		return nil, oidc.ErrTokenExpired
	}
	if nbf, ok := toInt64(claims["nbf"]); ok && nbf > now+leeway {
		return nil, oidc.ErrTokenNotYetValid
	}
	if iat, ok := toInt64(claims["iat"]); ok && iat > now+leeway {
		return nil, oidc.ErrTokenIssuedInFuture
	}

	gotIss, _ := claims["iss"].(string)
	if normalizeIssuer(gotIss) != normalizeIssuer(issuer) {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", issuer, gotIss)
	}

	if sub, _ := claims["sub"].(string); strings.TrimSpace(sub) == "" {
		return nil, oidc.ErrMissingSub
	}

	out := make(map[string]interface{}, len(claims))
	for k, val := range claims {
		out[k] = val
	}
	return out, nil
}


func (v *verifier) parseAndVerifyWithKey(tokenString string, pubKey crypto.PublicKey) (*jwt.Token, error) {
	if pubKey == nil {
		return nil, errors.New("no public key provided")
	}
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		alg := t.Method.Alg()
		// Auth0 default is RS256. Allow PS256 if needed.
		if alg != "RS256" && alg != "PS256" {
			return nil, fmt.Errorf("unexpected signing method: %s", alg)
		}
		return pubKey, nil
	})
}

func (v *verifier) lookupKeyFromCache(kid string) crypto.PublicKey {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if len(v.cache) == 0 {
		return nil
	}
	for _, jwk := range v.cache {
		if jwk.Kid != kid {
			continue
		}
		data, _ := json.Marshal(jwk)
		key, err := formats.NewJWKImporter().ImportPublic(data, sagecrypto.KeyFormatJWK)
		if err == nil {
			return key
		}
	}
	return nil
}

func findKeyByKID(keys []formats.JWK, kid string) crypto.PublicKey {
	for _, jwk := range keys {
		if jwk.Kid != kid {
			continue
		}
		data, _ := json.Marshal(jwk)
		key, err := formats.NewJWKImporter().ImportPublic(data, sagecrypto.KeyFormatJWK)
		if err == nil {
			return key
		}
	}
	return nil
}

// getJWKS returns cached JWKS or fetches fresh set 
func (v *verifier) getJWKS(ctx context.Context, issuer string) ([]formats.JWK, error) {
	v.mu.RLock()
	if time.Now().Before(v.expiresAt) && len(v.cache) > 0 {
		keys := v.cache
		v.mu.RUnlock()
		return keys, nil
	}
	v.mu.RUnlock()

	jwksURL := strings.TrimRight(issuer, "/") + "/.well-known/jwks.json"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create JWKS request: %w", err)
	}
	resp, err := v.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var doc struct{ Keys []formats.JWK `json:"keys"` }
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode JWKS response: %w", err)
	}
	if len(doc.Keys) == 0 {
		return nil, errors.New("no keys found in JWKS")
	}

	v.mu.Lock()
	v.cache = doc.Keys
	v.expiresAt = time.Now().Add(v.cfg.CacheTTL)
	v.mu.Unlock()

	return doc.Keys, nil
}



func has(claims map[string]interface{}, expected string) bool {
	v, ok := claims["aud"]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case string:
		return t == expected
	case []interface{}:
		for _, x := range t {
			if s, ok := x.(string); ok && s == expected {
				return true
			}
		}
	}
	return false
}

func toInt64(v interface{}) (int64, bool) {
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case int64:
		return t, true
	case int:
		return int64(t), true
	case json.Number:
		i, err := t.Int64()
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func normalizeIssuer(s string) string {
	if s == "" {
		return s
	}
	if strings.HasSuffix(s, "/") {
		return s[:len(s)-1]
	}
	return s
}

 
func containsScope(claims map[string]interface{}, want string) bool {
	raw, ok := claims["scope"].(string)
	if !ok || raw == "" {
		return false
	}
	for _, sc := range strings.Split(raw, " ") {
		if sc == want {
			return true
		}
	}
	return false
}