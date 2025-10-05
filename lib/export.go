// Package main provides C-compatible library exports for Sage
package main

import "C"

import (
	"github.com/sage-x-project/sage/core"
	"github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/did"
)

// Version returns the library version
//
//export SageVersion
func SageVersion() *C.char {
	return C.CString("1.0.0")
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
