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


