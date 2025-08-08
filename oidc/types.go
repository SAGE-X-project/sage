package oidc

import (
	"errors"
)


var (
    ErrMissingKid          = errors.New("missing kid in token header")
    ErrUnexpectedTyp       = errors.New("unexpected typ header")
    ErrNoMatchingJWK       = errors.New("no matching JWK found")
    ErrNoPublicKey         = errors.New("no public key provided")

    ErrInvalidSigningAlg   = errors.New("unexpected signing method")
    ErrInvalidToken        = errors.New("token is invalid")

    ErrInvalidAudience     = errors.New("invalid audience")
    ErrTokenExpired        = errors.New("token expired")
    ErrTokenNotYetValid    = errors.New("token not yet valid")
    ErrTokenIssuedInFuture = errors.New("token issued in the future")
    ErrInvalidIssuer       = errors.New("invalid issuer")
    ErrMissingSub          = errors.New("missing sub")
)


