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

package hpke

const TaskHPKEComplete = "hpke/complete@v1"

type InfoBuilder interface {
	BuildInfo(ctxID, initDID, respDID string) []byte
	BuildExportContext(ctxID string) []byte
}

// KeyIDBinder optionally lets the server issue custom key IDs.
type KeyIDBinder interface {
	IssueKeyID(ctxID string) (keyid string, ok bool)
}

// CookieVerifier optionally enables cheap pre-validation (anti-DoS).
type CookieVerifier interface {
	// Verify should be cheap and stateless if possible.
	Verify(cookie, ctxID, initDID, respDID string) bool
}

// CookieSource optionally provides DoS cookie to attach.
type CookieSource interface {
	GetCookie(ctxID, initDID, respDID string) (string, bool)
}

const (
	hpkeSuiteID    = "hpke-base+x25519+hkdf-sha256"
	combinerID     = "e2e-x25519-hkdf-v1"  // Combines HPKE exporter output with (ephC, ephS) ECDH secret
	infoLabel      = "sage/hpke-info|v1"   // Domain label used for the HPKE info transcript
	exportCtxLabel = "sage/hpke-export|v1" // Domain label used for the HPKE export context

	ackKeyLabel = "SAGE-ack-key-v1"
	cbLabel     = "SAGE-cb-v1"
	c2sKeyLabel = "SAGE-c2s:key"
	c2sIVLabel  = "SAGE-c2s:iv"
	s2cKeyLabel = "SAGE-s2c:key"
	s2cIVLabel  = "SAGE-s2c:iv"
)

type DefaultInfoBuilder struct{}
