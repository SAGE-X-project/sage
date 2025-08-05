package oidc

import (
	"context"

	"golang.org/x/oauth2"
)

// OIDCProvider defines the common interface for OIDC providers.
type OIDCProvider interface {
	// ExchangeToken implements RFC-8707 Token Exchange.
	ExchangeToken(ctx context.Context, subjectToken, subjectTokenType, audience string) (*oauth2.Token, error)

	// VerifyIDToken validates ID Token using JWKS and returns claims.
	VerifyIDToken(ctx context.Context, rawToken string) (map[string]interface{}, error)
}
