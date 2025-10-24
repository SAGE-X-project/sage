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

// Package helpers provides common test helper functions for SAGE test enhancement
package helpers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

// findProjectRoot finds the project root by looking for go.mod file
func findProjectRoot() (string, error) {
	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Walk up the directory tree looking for go.mod
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

// SaveTestData saves test data to JSON file for CLI verification
// This function creates test data files that can be used to verify
// CLI tools produce the same results as the code-level tests.
func SaveTestData(t *testing.T, filename string, data interface{}) {
	t.Helper()

	// Find project root by looking for go.mod
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Logf("Warning: Failed to find project root: %v", err)
		return
	}

	// Create testdata directory at project root
	testDataDir := filepath.Join(projectRoot, "testdata", "verification")
	if err := os.MkdirAll(testDataDir, 0750); err != nil {
		t.Logf("Warning: Failed to create testdata directory: %v", err)
		return
	}

	// Add metadata
	fullData := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"test_name": t.Name(),
		"data":      data,
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(fullData, "", "  ")
	if err != nil {
		t.Logf("Warning: Failed to marshal test data: %v", err)
		return
	}

	// Write to file
	filePath := filepath.Join(testDataDir, filename)

	// Create parent directory for the file (handles nested paths like "keys/file.json")
	fileDir := filepath.Dir(filePath)
	if err := os.MkdirAll(fileDir, 0750); err != nil {
		t.Logf("Warning: Failed to create file directory: %v", err)
		return
	}

	if err := os.WriteFile(filePath, jsonData, 0600); err != nil {
		t.Logf("Warning: Failed to write test data file: %v", err)
		return
	}

	t.Logf("  Test data saved: %s", filePath)
}

// ValidateUUID validates UUID v4 format
// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
// where y is one of [89ab]
func ValidateUUID(t *testing.T, uuidStr string) bool {
	t.Helper()

	// UUID v4 format pattern
	pattern := `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`
	matched, err := regexp.MatchString(pattern, uuidStr)
	if err != nil {
		t.Logf("  UUID validation error: %v", err)
		return false
	}

	if !matched {
		t.Logf("  UUID format invalid: %s", uuidStr)
		return false
	}

	t.Logf("[PASS] UUID v4 format confirmed")
	t.Logf("  UUID: %s", uuidStr)
	t.Logf("  Version: 4")

	return true
}

// LogKeyInfo logs cryptographic key information
// This function provides consistent logging for key generation tests
func LogKeyInfo(t *testing.T, keyType string, publicKey, privateKey []byte) {
	t.Helper()

	t.Logf("[PASS] %s key generation successful", keyType)
	t.Logf("  Public key size: %d bytes", len(publicKey))
	t.Logf("  Public key (hex): %x", publicKey)

	if privateKey != nil {
		t.Logf("  Private key size: %d bytes", len(privateKey))
		// Private key는 hex 출력하지 않음 (보안)
	}
}

// LogTestSection prints a test section header
func LogTestSection(t *testing.T, sectionID, description string) {
	t.Helper()
	t.Logf("===== %s %s =====", sectionID, description)
}

// LogSuccess logs a success message
func LogSuccess(t *testing.T, message string) {
	t.Helper()
	t.Logf("[PASS] %s", message)
}

// LogDetail logs a detail message with indentation
func LogDetail(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	message := format
	if len(args) > 0 {
		t.Logf("  "+message, args...)
	} else {
		t.Logf("  " + message)
	}
}

// LogPassCriteria prints pass criteria checklist
func LogPassCriteria(t *testing.T, criteria []string) {
	t.Helper()
	t.Log("===== Pass Criteria Checklist =====")
	for _, criterion := range criteria {
		t.Logf("  [PASS] %s", criterion)
	}
}

// ValidateKeySize validates key size and logs the result
func ValidateKeySize(t *testing.T, keyName string, actualSize, expectedSize int) bool {
	t.Helper()

	if actualSize != expectedSize {
		t.Logf("  [FAIL] %s size mismatch: actual=%d bytes, expected=%d bytes", keyName, actualSize, expectedSize)
		return false
	}

	t.Logf("[PASS] %s size: %d bytes (expected: %d bytes)", keyName, actualSize, expectedSize)
	return true
}

// ValidateSignatureSize validates signature size and logs the result
func ValidateSignatureSize(t *testing.T, actualSize, expectedSize int) bool {
	t.Helper()

	if actualSize != expectedSize {
		t.Logf("  [FAIL] Signature size mismatch: actual=%d bytes, expected=%d bytes", actualSize, expectedSize)
		return false
	}

	t.Logf("[PASS] Signature size: %d bytes (expected: %d bytes)", actualSize, expectedSize)
	return true
}
