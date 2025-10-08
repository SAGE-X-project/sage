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


package auth0

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/sage-x-project/sage/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAgentFromEnvSuffix(suffix string) (*Agent, error) {
    env := func(key string) string {
        return os.Getenv(fmt.Sprintf("%s_%s", key, suffix))
    }

    domain       := env("AUTH0_DOMAIN")
    clientID     := env("AUTH0_CLIENT_ID")
    clientSecret := env("AUTH0_CLIENT_SECRET")
    did          := env("TEST_DID")
    resource     := env("IDENTIFIER")
    keyId        := env("AUTH0_KEY_ID")

    _, privPEM, _, err := LoadOrCreateKeyPair(suffix)
    if err != nil {
        return nil, fmt.Errorf("load/create keypair: %w", err)
    }

    cfg := Config{
        KeyId:         keyId,
        Domain:        "https://" + domain ,
        ClientID:      clientID,
        ClientSecret:  clientSecret,
        PrivateKeyPEM: string(privPEM),
        DID:           did,
        Resource:      resource,
        HTTPTimeout:   10 * time.Second,
    }

    return NewAgent(cfg), nil
}

func TestIntegration_Auth0(t *testing.T) {
	os.Clearenv()
	
	// Try to load .env file, skip test if not available
	err := godotenv.Overload("../../.env")
	if err != nil {
		t.Skip("Skipping Auth0 integration test: .env file not found. Copy .env.example to .env and configure Auth0 credentials.")
		return
	}

    agentA, err := newAgentFromEnvSuffix("1")
    require.NoError(t, err)

    agentB, err := newAgentFromEnvSuffix("2")
    require.NoError(t, err)
    
    
    
    agentATokenurl := agentA.cfg.Domain + "/oauth/token"
    // agentBurl := "https://" + agentA.cfg.Domain + "/oauth/token"

    agentBverifierCfg := VerifierConfig{
        Identifier:   agentB.cfg.Resource,
        CacheTTL:    5 * time.Minute,
        HTTPTimeout: 5 * time.Second,
    }
    agentBverifier := NewVerifier(agentBverifierCfg)

    ctx := context.Background()
    t.Run("agent A get JWT contains did and agent B verify JWT", func(t *testing.T) {
        token, err := agentA.RequestToken(ctx, agentATokenurl, agentB.cfg.Resource)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(token, "eyJ"), "expected JWT format")

		parts := strings.Split(token, ".")
		require.Len(t, parts, 3, "token should have 3 parts")

		// payload
		payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
		require.NoError(t, err)

		var claims map[string]interface{}
		require.NoError(t, json.Unmarshal(payloadBytes, &claims), "failed to unmarshal payload")
		assert.Equal(t, agentA.cfg.DID, claims["did"], "did claim mismatch")

        verified, err := agentBverifier.Verify(ctx, token, agentA.cfg.Domain)
        require.NoError(t, err)
        assert.Equal(t, agentA.cfg.DID, verified["did"])
        assert.Equal(t, agentBverifier.cfg.Identifier, verified["aud"])
        assert.Equal(t, agentA.cfg.ClientID+"@clients", verified["sub"])
    })

    t.Run("error: invalid signature", func(t *testing.T) {
        token, err := agentA.RequestToken(ctx, agentATokenurl, agentB.cfg.Resource)
        require.NoError(t, err)

        bad, err := TamperSignatureRS256(token)
        require.NoError(t, err)
        _, err = agentBverifier.Verify(ctx, bad, agentA.cfg.Domain)
        require.Error(t, err)
        assert.Contains(t, err.Error(), "token verification failed") 
    })

    t.Run("error: invalid audience (resource)", func(t *testing.T) {
        token, err := agentA.RequestToken(ctx, agentATokenurl, agentB.cfg.Resource)
        require.NoError(t, err)

        wrong := NewVerifier(VerifierConfig{
            Identifier:  "https://example.invalid/api", 
            CacheTTL:    5 * time.Minute,
            HTTPTimeout: 5 * time.Second,
        })

        _, err = wrong.Verify(ctx, token, agentA.cfg.Domain)
        require.Error(t, err)
        require.ErrorIs(t, err, oidc.ErrInvalidAudience)
    })

    t.Run("error: token expired", func(t *testing.T) {
        ttlStr := os.Getenv("TEST_API_TOKEN_TTL_SECONDS") 
        if ttlStr == "" {
            t.Skip("set TEST_API_TOKEN_TTL_SECONDS (e.g. 60) to run this test")
        }
        ttl, err := strconv.Atoi(ttlStr)
        require.NoError(t, err)

        token, err := agentA.RequestToken(ctx, agentATokenurl, agentB.cfg.Resource)
        require.NoError(t, err)

        // TTL + interval
        time.Sleep(time.Duration(ttl+5) * time.Second)

        _, err = agentBverifier.Verify(ctx, token, agentA.cfg.Domain)
        require.Error(t, err)
        require.ErrorIs(t, err, oidc.ErrTokenExpired)
    })
}

