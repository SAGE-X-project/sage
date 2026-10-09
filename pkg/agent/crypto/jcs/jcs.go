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

// Package jcs implements the JSON Canonicalization Scheme (RFC 8785).
//
// Every signed JSON structure in SAGE (A2A card proofs, the HPKE handshake
// response) is canonicalized with this package before hashing or signing, so
// that any implementation, in any language, produces and verifies exactly the
// same bytes from the same JSON value. The rules are those of RFC 8785:
// object members sorted by the UTF-16 code units of their names, no
// insignificant whitespace, strings escaped as ECMAScript JSON.stringify does
// (only `"`, `\` and control characters below U+0020), and numbers formatted
// as ECMAScript Number::toString.
//
// Canonicalize keeps the last of duplicate member names and has no size,
// member or depth limits. It is not a SAGE 0.10.0 validation entry point; use
// guard010.Canonicalize for 0.10.0 JSON.
package jcs

import "github.com/sage-x-project/sage/pkg/agent/internal/rfc8785"

// Marshal encodes v with encoding/json and returns its canonical form.
func Marshal(v interface{}) ([]byte, error) { return rfc8785.Marshal(v) }

// Canonicalize returns the RFC 8785 canonical form of a JSON document.
func Canonicalize(raw []byte) ([]byte, error) { return rfc8785.Canonicalize(raw) }
