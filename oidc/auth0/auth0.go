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
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/formats"
	"golang.org/x/oauth2"
)

var (
    ErrInvalidToken          = errors.New("invalid token")
    ErrMissingKID            = errors.New("missing kid in token header")
    ErrNoMatchingJWK         = errors.New("no matching JWK found")
    ErrInvalidClaimsFormat   = errors.New("invalid claims format")
    ErrFetchJWKSFailed       = errors.New("failed to fetch JWKS")
    ErrParseJWKSFailed       = errors.New("failed to parse JWKS")
    ErrImportPublicKeyFailed = errors.New("failed to import public key")
    ErrTokenVerification     = errors.New("token verification failed")
    ErrTokenExchangeFailed   = errors.New("token exchange failed")
    ErrParseTokenResponse    = errors.New("failed to parse token response")
)

// jwksCache holds cached JWKS entries with expiration.
type jwksCache struct {
    sync.RWMutex
    keys      []formats.JWK
    expiresAt time.Time
}

func (c *jwksCache) get() ([]formats.JWK, bool) {
    c.RLock()
    defer c.RUnlock()
    if time.Now().Before(c.expiresAt) {
        return c.keys, true
    }
    return nil, false
}

func (c *jwksCache) set(keys []formats.JWK, ttl time.Duration) {
    c.Lock()
    defer c.Unlock()
    c.keys = keys
    c.expiresAt = time.Now().Add(ttl)
}

// Provider implements RFC-8707 token exchange and JWKS-based ID token verification.
type Provider struct {
    BaseURL      string
    ClientID     string
    ClientSecret string
    HTTPClient   *http.Client
    jwksCache    jwksCache
    jwksTTL      time.Duration
}

// NewProvider initializes a new Auth0 OIDC provider with a default HTTP timeout.
func NewProvider(baseURL, clientID, clientSecret string) *Provider {
    return &Provider{
        BaseURL:      strings.TrimRight(baseURL, "/"),
        ClientID:     clientID,
        ClientSecret: clientSecret,
        HTTPClient:   &http.Client{Timeout: 5 * time.Second},
        jwksTTL:      10 * time.Minute,
    }
}

// ExchangeToken implements RFC-8707 Token Exchange.
func (p *Provider) ExchangeToken(ctx context.Context, subjectToken, subjectTokenType, audience string) (*oauth2.Token, error) {
    form := url.Values{
        "grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
        "subject_token":      {subjectToken},
        "subject_token_type": {subjectTokenType},
        "audience":           {audience},
        "client_id":          {p.ClientID},
        "client_secret":      {p.ClientSecret},
    }

    endpoint := fmt.Sprintf("%s/oauth/token", p.BaseURL)
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := p.HTTPClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("%w: %s", ErrTokenExchangeFailed, string(body))
    }

    var token oauth2.Token
    if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
        return nil, fmt.Errorf("%w: %v", ErrParseTokenResponse, err)
    }
    return &token, nil
}

// VerifyIDToken validates an ID token using JWKS, with caching and standard claims checks.
func (p *Provider) VerifyIDToken(ctx context.Context, rawToken string) (map[string]interface{}, error) {
    parser := new(jwt.Parser)
    unverified, _, err := parser.ParseUnverified(rawToken, jwt.MapClaims{})
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
    }

    kid, ok := unverified.Header["kid"].(string)
    if !ok || kid == "" {
        return nil, ErrMissingKID
    }

    // Load JWKS (from cache or fetch)
    var jwks []formats.JWK
    if cached, ok := p.jwksCache.get(); ok {
        jwks = cached
    } else {
        url := fmt.Sprintf("%s/.well-known/jwks.json", p.BaseURL)
        resp, err := p.HTTPClient.Get(url)
        if err != nil {
            return nil, fmt.Errorf("%w: %v", ErrFetchJWKSFailed, err)
        }
        defer resp.Body.Close()

        body, err := io.ReadAll(resp.Body)
        if err != nil {
            return nil, fmt.Errorf("%w: %v", ErrFetchJWKSFailed, err)
        }
        var doc struct{ Keys []formats.JWK `json:"keys"` }
        if err := json.Unmarshal(body, &doc); err != nil {
            return nil, fmt.Errorf("%w: %v", ErrParseJWKSFailed, err)
        }
        jwks = doc.Keys
        p.jwksCache.set(jwks, p.jwksTTL)
    }

    // Find matching JWK and import public key
    var publicKey crypto.PublicKey
    importer := formats.NewJWKImporter()
    for _, key := range jwks {
        if key.Kid == kid {
            data, _ := json.Marshal(key)
            pub, err := importer.ImportPublic(data, sagecrypto.KeyFormatJWK)
            if err != nil {
                return nil, fmt.Errorf("%w: %v", ErrImportPublicKeyFailed, err)
            }
            publicKey = pub
            break
        }
    }
    if publicKey == nil {
        return nil, ErrNoMatchingJWK
    }

    // Verify signature and parse claims
    token, err := jwt.Parse(rawToken, func(t *jwt.Token) (interface{}, error) {
        return publicKey, nil
    })
    if err != nil || !token.Valid {
        return nil, fmt.Errorf("%w: %v", ErrTokenVerification, err)
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return nil, ErrInvalidClaimsFormat
    }

    // Validate exp
    expVal, ok := claims["exp"]
    if !ok {
        return nil, fmt.Errorf("missing exp claim")
    }
    var exp int64
    switch v := expVal.(type) {
    case float64:
        exp = int64(v)
    case json.Number:
        exp, _ = v.Int64()
    default:
        return nil, fmt.Errorf("invalid exp claim type: %T", expVal)
    }
    if time.Now().Unix() > exp {
        return nil, fmt.Errorf("token expired at %d", exp)
    }

    // Validate aud
    audVal, ok := claims["aud"]
    if !ok {
        return nil, fmt.Errorf("missing aud claim")
    }
    validAud := false
    switch v := audVal.(type) {
    case string:
        validAud = (v == p.ClientID)
    case []interface{}:
        for _, entry := range v {
            if s, ok := entry.(string); ok && s == p.ClientID {
                validAud = true
                break
            }
        }
    default:
        return nil, fmt.Errorf("invalid aud claim type: %T", audVal)
    }
    if !validAud {
        return nil, fmt.Errorf("audience mismatch: %v", audVal)
    }

    // Validate iss (allow with/without trailing slash)
    iss, ok := claims["iss"].(string)
    if !ok {
        return nil, fmt.Errorf("missing iss claim")
    }
    expectedIss := p.BaseURL
    if !strings.EqualFold(strings.TrimRight(iss, "/"), strings.TrimRight(expectedIss, "/")) {
        return nil, fmt.Errorf("issuer mismatch: %s", iss)
    }

    return claims, nil
}
