package rfc9421

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanonicalizer(t *testing.T) {
	// Test 1.4.1: Basic GET request
	t.Run("basic GET request", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/foo?bar=baz", nil)
		require.NoError(t, err)
		
		components := []string{`"@method"`, `"@authority"`, `"@path"`, `"@query"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
			KeyID:             "test-key",
			Algorithm:         "ed25519",
			Created:           1719234000,
		}
		
		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)
		
		expected := `"@method": GET
"@authority": example.com
"@path": /foo
"@query": ?bar=baz
"@signature-params": ("@method" "@authority" "@path" "@query");keyid="test-key";alg="ed25519";created=1719234000`
		
		assert.Equal(t, expected, result)
	})
	
	// Test 1.4.2: POST request with Content-Digest
	t.Run("POST request with Content-Digest", func(t *testing.T) {
		body := `{"hello": "world"}`
		req, err := http.NewRequest("POST", "https://example.com/data", strings.NewReader(body))
		require.NoError(t, err)
		
		req.Header.Set("Content-Digest", "sha-256=:X48E9qOokqqrvdts8nOJRJN3OWDUoyWxBf7kbu9DBPE=:")
		req.Header.Set("Date", "Mon, 24 Jun 2024 12:00:00 GMT")
		
		components := []string{`"content-digest"`, `"date"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}
		
		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)
		
		assert.Contains(t, result, `"content-digest": sha-256=:X48E9qOokqqrvdts8nOJRJN3OWDUoyWxBf7kbu9DBPE=:`)
		assert.Contains(t, result, `"date": Mon, 24 Jun 2024 12:00:00 GMT`)
	})
	
	// Test 1.4.3: Header value whitespace handling
	t.Run("header whitespace handling", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com", nil)
		require.NoError(t, err)
		
		req.Header.Set("X-Custom", "  value with spaces  ")
		
		components := []string{`"x-custom"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}
		
		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)
		
		// Should trim outer spaces but keep internal ones
		assert.Contains(t, result, `"x-custom": value with spaces`)
	})
	
	// Test 1.4.4: Multiple headers with same name
	t.Run("multiple headers with same name", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com", nil)
		require.NoError(t, err)
		
		req.Header.Add("Via", "1.1 proxy-a")
		req.Header.Add("Via", "1.1 proxy-b")
		
		components := []string{`"via"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}
		
		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)
		
		// Should be joined with comma and space
		assert.Contains(t, result, `"via": 1.1 proxy-a, 1.1 proxy-b`)
	})
	
	// Test 1.4.5: Component not found
	t.Run("component not found", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com", nil)
		require.NoError(t, err)
		
		components := []string{`"content-digest"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}
		
		canonicalizer := NewCanonicalizer()
		_, err = canonicalizer.BuildSignatureBase(req, "sig1", params)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "component not found")
	})
	
	// Test 4.2.1: Empty path
	t.Run("empty path", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com", nil)
		require.NoError(t, err)
		
		components := []string{`"@path"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}
		
		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)
		
		// Empty path should be treated as "/"
		assert.Contains(t, result, `"@path": /`)
	})
	
	// Test 4.2.2: Special characters in path/query
	t.Run("special characters in path and query", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/users/שלום?q=a%20b+c", nil)
		require.NoError(t, err)
		
		components := []string{`"@path"`, `"@query"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}
		
		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)
		
		// Should preserve encoding
		assert.Contains(t, result, `"@path": /users/שלום`)
		assert.Contains(t, result, `"@query": ?q=a%20b+c`)
	})
	
	// Test 4.2.3: Proxy request target
	t.Run("proxy request target", func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://example.com/foo", nil)
		require.NoError(t, err)
		
		// Simulate proxy request
		req.RequestURI = "http://example.com/foo"
		
		components := []string{`"@request-target"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}
		
		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)
		
		// Should include full URL for proxy requests
		assert.Contains(t, result, `"@request-target": GET /foo`)
	})
}

func TestQueryParamComponent(t *testing.T) {
	// Test 4.1.1: Specific parameter protection
	t.Run("specific parameter protection", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/api/v1/users?id=123&format=json&cache=false", nil)
		require.NoError(t, err)
		
		components := []string{`"@query-param";name="id"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}
		
		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)
		
		// Should include the specific query parameter
		assert.Contains(t, result, `"@query-param";name="id": 123`)
	})
	
	// Test 4.1.4: Parameter name case sensitivity
	t.Run("parameter name case sensitivity", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/api?ID=123", nil)
		require.NoError(t, err)
		
		components := []string{`"@query-param";name="id"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}
		
		canonicalizer := NewCanonicalizer()
		_, err = canonicalizer.BuildSignatureBase(req, "sig1", params)
		
		// Should fail because "id" != "ID"
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "component not found")
	})
	
	// Test 4.1.5: Non-existent parameter
	t.Run("non-existent parameter", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/api?id=123", nil)
		require.NoError(t, err)
		
		components := []string{`"@query-param";name="status"`}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}
		
		canonicalizer := NewCanonicalizer()
		_, err = canonicalizer.BuildSignatureBase(req, "sig1", params)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "component not found")
	})
	
	// Test multiple query parameters
	t.Run("multiple query parameters", func(t *testing.T) {
		req, err := http.NewRequest("GET", "https://example.com/api?id=123&name=test&format=json", nil)
		require.NoError(t, err)
		
		components := []string{
			`"@query-param";name="id"`,
			`"@query-param";name="format"`,
		}
		params := &SignatureInputParams{
			CoveredComponents: components,
		}
		
		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)
		
		assert.Contains(t, result, `"@query-param";name="id": 123`)
		assert.Contains(t, result, `"@query-param";name="format": json`)
		// Should NOT contain name parameter since it wasn't included
		assert.NotContains(t, result, `name=test`)
	})
}

func TestHTTPFields(t *testing.T) {
	t.Run("all HTTP fields", func(t *testing.T) {
		req, err := http.NewRequest("POST", "https://api.example.com:8443/v1/messages?filter=active", bytes.NewReader([]byte("test body")))
		require.NoError(t, err)
		
		req.Header.Set("Host", "api.example.com:8443")
		req.Header.Set("Date", time.Now().Format(http.TimeFormat))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Length", "9")
		
		components := []string{
			`"@method"`,
			`"@target-uri"`,
			`"@authority"`,
			`"@scheme"`,
			`"@request-target"`,
			`"@path"`,
			`"@query"`,
			`"@status"`,      // This should fail for requests
			`"host"`,
			`"date"`,
			`"content-type"`,
			`"content-length"`,
		}
		
		params := &SignatureInputParams{
			CoveredComponents: components,
		}
		
		canonicalizer := NewCanonicalizer()
		result, err := canonicalizer.BuildSignatureBase(req, "sig1", params)
		
		// Should fail because @status is only for responses
		require.Error(t, err)
		assert.Contains(t, err.Error(), "@status")
		
		// Now test without @status
		components = []string{
			`"@method"`,
			`"@target-uri"`,
			`"@authority"`,
			`"@scheme"`,
			`"@request-target"`,
			`"@path"`,
			`"@query"`,
			`"host"`,
			`"date"`,
			`"content-type"`,
			`"content-length"`,
		}
		params.CoveredComponents = components
		
		result, err = canonicalizer.BuildSignatureBase(req, "sig1", params)
		require.NoError(t, err)
		
		// Verify all components are present
		assert.Contains(t, result, `"@method": POST`)
		assert.Contains(t, result, `"@target-uri": https://api.example.com:8443/v1/messages?filter=active`)
		assert.Contains(t, result, `"@authority": api.example.com:8443`)
		assert.Contains(t, result, `"@scheme": https`)
		assert.Contains(t, result, `"@request-target": POST /v1/messages?filter=active`)
		assert.Contains(t, result, `"@path": /v1/messages`)
		assert.Contains(t, result, `"@query": ?filter=active`)
		assert.Contains(t, result, `"host": api.example.com:8443`)
		assert.Contains(t, result, `"content-type": application/json`)
		assert.Contains(t, result, `"content-length": 9`)
	})
}