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

package main

import "C"

import (
	"github.com/sage-x-project/sage/pkg/agent/core"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// Version returns the library version
//
//export SageVersion
func SageVersion() *C.char {
	return C.CString("1.3.1")
}

// Initialize initializes the Sage library
//
//export SageInit
func SageInit() C.int {
	// Initialize core components
	return 0
}

// Cleanup cleans up library resources
//
//export SageCleanup
func SageCleanup() {
	// Cleanup resources
}

// These are placeholder references to ensure packages are included in the library
var (
	_ = core.NewVerificationService
	_ = crypto.NewManager
	_ = did.NewManager
)

func main() {
	// Required for buildmode=c-shared/c-archive
}
