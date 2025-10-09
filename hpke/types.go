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

const (
	hpkeSuiteID    = "hpke-base+x25519+hkdf-sha256"
	combinerID     = "e2e-x25519-hkdf-v1"  // Combines HPKE exporter output with (ephC, ephS) ECDH secret
	infoLabel      = "sage/hpke-info|v1"   // Domain label used for the HPKE info transcript
	exportCtxLabel = "sage/hpke-export|v1" // Domain label used for the HPKE export context
)

type DefaultInfoBuilder struct{}
