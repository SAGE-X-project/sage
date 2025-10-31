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

package rfc9421

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// BodyIntegrityValidator validates HTTP body integrity using Content-Digest header.
// This prevents body tampering attacks where an attacker modifies the body but leaves
// the Content-Digest header unchanged.
//
// Design Pattern: Single Responsibility Principle (SRP)
// - Sole responsibility: Validate that Content-Digest header matches actual body content
// - Separated from HTTP signature verification logic
type BodyIntegrityValidator struct {
	// Future: Injectable hasher for testing and algorithm flexibility
}

// NewBodyIntegrityValidator creates a new body integrity validator.
func NewBodyIntegrityValidator() *BodyIntegrityValidator {
	return &BodyIntegrityValidator{}
}

// ValidateContentDigest validates that the Content-Digest header matches the actual request body.
//
// This implements the security requirement from PR #118:
// - Prevents attacks where body is modified but Content-Digest header remains unchanged
// - Only validates if "content-digest" is in the covered components (signature scope)
//
// Algorithm:
// 1. Check if content-digest is in covered components (case-insensitive)
// 2. If not covered, skip validation (no body integrity guarantee needed)
// 3. Read and restore the request body
// 4. Compute SHA-256 digest of body
// 5. Compare with Content-Digest header value
//
// Parameters:
//   - req: HTTP request to validate
//   - coveredComponents: List of components included in the signature
//
// Returns:
//   - error: nil if valid, error describing the validation failure otherwise
func (v *BodyIntegrityValidator) ValidateContentDigest(req *http.Request, coveredComponents []string) error {
	// Step 1: Check if content-digest is covered by the signature
	if !IsComponentCovered(coveredComponents, "content-digest") {
		// Content-Digest not in signature scope, skip validation
		return nil
	}

	// Step 2: Read body and restore it for later use
	body, err := readBodyAndRestore(req)
	if err != nil {
		return fmt.Errorf("failed to read body for content-digest validation: %w", err)
	}

	// Step 3: Compute expected Content-Digest
	expectedDigest := ComputeContentDigest(body)

	// Step 4: Get actual Content-Digest from header
	actualDigest := strings.TrimSpace(req.Header.Get("Content-Digest"))
	if actualDigest == "" {
		return fmt.Errorf("content-digest header missing while covered by signature")
	}

	// Step 5: Compare digests (supports multiple algorithms in header)
	if !equalDigestHeader(actualDigest, expectedDigest) {
		return fmt.Errorf("content-digest mismatch: actual=%q expected=%q (body tampering detected)", actualDigest, expectedDigest)
	}

	return nil
}

// IsComponentCovered checks if a component is included in the covered components list.
//
// Design: Case-insensitive matching with quote handling
// - Supports both `content-digest` and `"content-digest"` formats
// - Case-insensitive for robustness
//
// Parameters:
//   - coveredComponents: List of components from signature metadata
//   - componentName: Component to search for (e.g., "content-digest")
//
// Returns:
//   - bool: true if component is covered, false otherwise
func IsComponentCovered(coveredComponents []string, componentName string) bool {
	targetName := strings.ToLower(strings.TrimSpace(componentName))

	for _, component := range coveredComponents {
		// Normalize: lowercase, trim spaces and quotes
		normalized := strings.ToLower(strings.Trim(strings.TrimSpace(component), `"`))
		if normalized == targetName {
			return true
		}
	}

	return false
}

// ComputeContentDigest computes the RFC 9421 Content-Digest header value for a body.
//
// Format: sha-256=:<base64-encoded-hash>:
// - Uses SHA-256 algorithm (most widely supported)
// - Base64 encoding with standard alphabet
// - Wrapped with colons per RFC 9421 spec
//
// Parameters:
//   - body: Raw body bytes
//
// Returns:
//   - string: Content-Digest header value
func ComputeContentDigest(body []byte) string {
	hash := sha256.Sum256(body)
	encoded := base64.StdEncoding.EncodeToString(hash[:])
	return "sha-256=:" + encoded + ":"
}

// readBodyAndRestore reads the entire request body and restores it for later reads.
//
// Design: Non-destructive read
// - Reads body into memory
// - Creates new reader for Body field
// - Sets ContentLength for proper handling
// - Provides GetBody function for retries
//
// Note: This holds the entire body in memory. For very large bodies (>10MB),
// consider streaming validation or size limits.
//
// Parameters:
//   - req: HTTP request with body
//
// Returns:
//   - []byte: Body content
//   - error: Read error if any
func readBodyAndRestore(req *http.Request) ([]byte, error) {
	// Handle nil body (e.g., GET requests)
	if req.Body == nil {
		return []byte{}, nil
	}

	// Read entire body
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Restore body for subsequent reads
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	req.ContentLength = int64(len(bodyBytes))

	// Provide GetBody for HTTP client retries
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bodyBytes)), nil
	}

	return bodyBytes, nil
}

// equalDigestHeader checks if actual Content-Digest header matches expected value.
//
// Design: Flexible matching for multiple algorithms
// - Supports single algorithm: "sha-256=:hash:"
// - Supports multiple algorithms: "sha-512=:hash1:, sha-256=:hash2:"
// - Extracts and compares only sha-256 value
//
// Parameters:
//   - actual: Content-Digest header value from request
//   - expected: Expected sha-256 digest value
//
// Returns:
//   - bool: true if digests match, false otherwise
func equalDigestHeader(actual, expected string) bool {
	// Direct match (single algorithm case)
	if actual == expected {
		return true
	}

	// Multiple algorithms case: parse comma-separated list
	parts := strings.Split(actual, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "sha-256=") {
			return part == expected
		}
	}

	return false
}
