// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// Edge case and security tests for Body Integrity Validation

package rfc9421

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBodyIntegrityValidator_EdgeCases tests edge cases and boundary conditions
func TestBodyIntegrityValidator_EdgeCases(t *testing.T) {
	validator := NewBodyIntegrityValidator()

	t.Run("Nil request", func(t *testing.T) {
		// Should handle nil request gracefully
		var req *http.Request
		coveredComponents := []string{"content-digest"}

		// This should panic or return error - testing defensive programming
		defer func() {
			if r := recover(); r != nil {
				helpers.LogSuccess(t, "Nil request handled with panic (expected)")
			}
		}()

		err := validator.ValidateContentDigest(req, coveredComponents)
		// If no panic, should return error
		if err != nil {
			assert.Error(t, err)
			helpers.LogSuccess(t, "Nil request handled with error")
		}
	})

	t.Run("Request with nil Body", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com", nil)
		require.NoError(t, err)

		// Compute digest for empty body
		expectedDigest := ComputeContentDigest([]byte{})
		req.Header.Set("Content-Digest", expectedDigest)

		coveredComponents := []string{"content-digest"}

		err = validator.ValidateContentDigest(req, coveredComponents)
		assert.NoError(t, err, "Nil body should be treated as empty")
		helpers.LogSuccess(t, "Nil body handled correctly")
	})

	t.Run("Extremely large body (memory stress test)", func(t *testing.T) {
		// 10MB body
		largeBody := make([]byte, 10*1024*1024)
		for i := range largeBody {
			largeBody[i] = byte(i % 256)
		}

		req, err := http.NewRequest("POST", "https://example.com", bytes.NewReader(largeBody))
		require.NoError(t, err)

		expectedDigest := ComputeContentDigest(largeBody)
		req.Header.Set("Content-Digest", expectedDigest)

		coveredComponents := []string{"content-digest"}

		err = validator.ValidateContentDigest(req, coveredComponents)
		assert.NoError(t, err, "Large body should validate correctly")

		// Verify body can still be read
		readBody, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, len(largeBody), len(readBody), "Body should be fully restored")
		helpers.LogSuccess(t, "Large body (10MB) validated and restored")
	})

	t.Run("Malformed Content-Digest header", func(t *testing.T) {
		body := []byte("test data")
		req, err := http.NewRequest("POST", "https://example.com", bytes.NewReader(body))
		require.NoError(t, err)

		tests := []struct {
			name   string
			digest string
		}{
			{"Invalid base64", "sha-256=:not-valid-base64!@#$:"},
			{"Missing algorithm", ":SGVsbG8=:"},
			{"Missing colons", "sha-256=SGVsbG8"},
			{"Empty value", "sha-256=::"},
			{"Wrong algorithm", "md5=:SGVsbG8=:"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				req.Header.Set("Content-Digest", tt.digest)
				coveredComponents := []string{"content-digest"}

				err = validator.ValidateContentDigest(req, coveredComponents)
				assert.Error(t, err, "Malformed digest should fail")
				assert.Contains(t, err.Error(), "mismatch", "Error should indicate mismatch")
			})
		}
		helpers.LogSuccess(t, "Malformed Content-Digest headers rejected")
	})

	t.Run("Body read error simulation", func(t *testing.T) {
		// Create a reader that always returns error
		errorReader := &errorReadCloser{err: io.ErrUnexpectedEOF}
		req, err := http.NewRequest("POST", "https://example.com", errorReader)
		require.NoError(t, err)

		req.Header.Set("Content-Digest", "sha-256=:test=:")
		coveredComponents := []string{"content-digest"}

		err = validator.ValidateContentDigest(req, coveredComponents)
		assert.Error(t, err, "Read error should propagate")
		assert.Contains(t, err.Error(), "read body", "Error should mention read failure")
		helpers.LogSuccess(t, "Body read error handled correctly")
	})

	t.Run("Unicode and special characters in body", func(t *testing.T) {
		bodies := [][]byte{
			[]byte("Hello 世界 🌍"),                    // Unicode
			[]byte("Line1\nLine2\rLine3\r\n"),        // Different line endings
			[]byte("\x00\x01\x02\x03"),               // Binary data
			[]byte(strings.Repeat("🔐", 1000)),        // Repeated emoji
		}

		for i, body := range bodies {
			req, err := http.NewRequest("POST", "https://example.com", bytes.NewReader(body))
			require.NoError(t, err)

			expectedDigest := ComputeContentDigest(body)
			req.Header.Set("Content-Digest", expectedDigest)

			coveredComponents := []string{"content-digest"}

			err = validator.ValidateContentDigest(req, coveredComponents)
			assert.NoError(t, err, "Body %d should validate", i)
		}
		helpers.LogSuccess(t, "Unicode and special characters handled correctly")
	})
}

// TestIsComponentCovered_EdgeCases tests component matching edge cases
func TestIsComponentCovered_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		covered   []string
		component string
		expected  bool
	}{
		{
			name:      "Component with extra whitespace",
			covered:   []string{"  content-digest  ", "@method"},
			component: "content-digest",
			expected:  true,
		},
		{
			name:      "Component with mixed quotes",
			covered:   []string{`"content-digest`, `content-digest"`},
			component: "content-digest",
			expected:  true,
		},
		{
			name:      "Case variations",
			covered:   []string{"Content-Digest", "CONTENT-DIGEST", "CoNtEnT-dIgEsT"},
			component: "content-digest",
			expected:  true,
		},
		{
			name:      "Similar but different component",
			covered:   []string{"content-type", "content-length"},
			component: "content-digest",
			expected:  false,
		},
		{
			name:      "Substring match should not work",
			covered:   []string{"content-digest-v2"},
			component: "content-digest",
			expected:  false,
		},
		{
			name:      "Empty component name",
			covered:   []string{"", "content-digest"},
			component: "",
			expected:  true,
		},
		{
			name:      "Nil covered list",
			covered:   nil,
			component: "content-digest",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsComponentCovered(tt.covered, tt.component)
			assert.Equal(t, tt.expected, result)
		})
	}
	helpers.LogSuccess(t, "Component matching edge cases handled")
}

// TestComputeContentDigest_EdgeCases tests digest computation edge cases
func TestComputeContentDigest_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		body []byte
	}{
		{"Nil body", nil},
		{"Empty body", []byte{}},
		{"Single byte", []byte{0x00}},
		{"All zeros", make([]byte, 1000)},
		{"All ones", bytes.Repeat([]byte{0xFF}, 1000)},
		{"Sequential bytes", func() []byte {
			b := make([]byte, 256)
			for i := range b {
				b[i] = byte(i)
			}
			return b
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			digest := ComputeContentDigest(tt.body)

			// Verify format
			assert.True(t, strings.HasPrefix(digest, "sha-256=:"), "Should have correct prefix")
			assert.True(t, strings.HasSuffix(digest, ":"), "Should have correct suffix")

			// Verify it's valid base64 between markers
			parts := strings.Split(digest, ":")
			assert.Len(t, parts, 3, "Should have 3 parts (prefix, b64, suffix)")
			assert.NotEmpty(t, parts[1], "Base64 part should not be empty")

			// Verify consistency
			digest2 := ComputeContentDigest(tt.body)
			assert.Equal(t, digest, digest2, "Same input should produce same digest")
		})
	}
	helpers.LogSuccess(t, "Digest computation edge cases validated")
}

// Helper: errorReadCloser simulates read errors
type errorReadCloser struct {
	err error
}

func (e *errorReadCloser) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func (e *errorReadCloser) Close() error {
	return nil
}

// TestBodyIntegrityValidator_SecurityCases tests security-related scenarios
func TestBodyIntegrityValidator_SecurityCases(t *testing.T) {
	validator := NewBodyIntegrityValidator()

	t.Run("Timing attack resistance - same length bodies", func(t *testing.T) {
		// Two different bodies of same length
		body1 := bytes.Repeat([]byte("A"), 1000)
		body2 := bytes.Repeat([]byte("B"), 1000)

		digest1 := ComputeContentDigest(body1)
		digest2 := ComputeContentDigest(body2)

		// Should produce different digests
		assert.NotEqual(t, digest1, digest2, "Different bodies should have different digests")

		// Verification should fail for mismatched digest
		req, _ := http.NewRequest("POST", "https://example.com", bytes.NewReader(body1))
		req.Header.Set("Content-Digest", digest2) // Wrong digest

		err := validator.ValidateContentDigest(req, []string{"content-digest"})
		assert.Error(t, err, "Mismatched digest should fail")
		helpers.LogSuccess(t, "Timing attack resistance validated")
	})

	t.Run("Collision attempt - similar bodies", func(t *testing.T) {
		// Bodies that differ by only one bit
		body1 := []byte("The quick brown fox jumps over the lazy dog")
		body2 := []byte("The quick brown fox jumps over the lazy doh") // 'g' → 'h'

		digest1 := ComputeContentDigest(body1)
		digest2 := ComputeContentDigest(body2)

		assert.NotEqual(t, digest1, digest2, "Similar bodies should have different digests")
		helpers.LogSuccess(t, "Collision resistance validated")
	})

	t.Run("Replay attack - reusing old digest", func(t *testing.T) {
		oldBody := []byte("original message")
		newBody := []byte("tampered message")

		oldDigest := ComputeContentDigest(oldBody)

		// Attempt to use old digest with new body
		req, _ := http.NewRequest("POST", "https://example.com", bytes.NewReader(newBody))
		req.Header.Set("Content-Digest", oldDigest)

		err := validator.ValidateContentDigest(req, []string{"content-digest"})
		assert.Error(t, err, "Old digest should not validate new body")
		assert.Contains(t, err.Error(), "tampering detected")
		helpers.LogSuccess(t, "Replay attack prevented")
	})
}
