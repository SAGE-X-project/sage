package rfc9421

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSignatureInput(t *testing.T) {
	// Test 1.1.1: Basic parsing
	t.Run("basic parsing", func(t *testing.T) {
		input := `sig1=("@method" "host");keyid="did:key:z6Mk...";alg="ed25519";created=1719234000`
		
		result, err := ParseSignatureInput(input)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		sig1, exists := result["sig1"]
		require.True(t, exists)
		
		assert.Equal(t, []string{`"@method"`, `"host"`}, sig1.CoveredComponents)
		assert.Equal(t, "did:key:z6Mk...", sig1.KeyID)
		assert.Equal(t, "ed25519", sig1.Algorithm)
		assert.Equal(t, int64(1719234000), sig1.Created)
	})
	
	// Test 1.1.2: Multiple signatures and parameters
	t.Run("multiple signatures with parameters", func(t *testing.T) {
		input := `sig-a=("@method");expires=1719237600, sig-b=("host" "date");keyid="test-key-2";nonce="abcdef"`
		
		result, err := ParseSignatureInput(input)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		// Check sig-a
		sigA, exists := result["sig-a"]
		require.True(t, exists)
		assert.Equal(t, []string{`"@method"`}, sigA.CoveredComponents)
		assert.Equal(t, int64(1719237600), sigA.Expires)
		
		// Check sig-b
		sigB, exists := result["sig-b"]
		require.True(t, exists)
		assert.Equal(t, []string{`"host"`, `"date"`}, sigB.CoveredComponents)
		assert.Equal(t, "test-key-2", sigB.KeyID)
		assert.Equal(t, "abcdef", sigB.Nonce)
	})
	
	// Test 1.1.3: Whitespace and case variations
	t.Run("whitespace and case handling", func(t *testing.T) {
		input := `sig1 = ( "@path"  "@query" ); KeyId = "test-key" ; Alg = "ecdsa-p256"`
		
		result, err := ParseSignatureInput(input)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		sig1, exists := result["sig1"]
		require.True(t, exists)
		
		assert.Equal(t, []string{`"@path"`, `"@query"`}, sig1.CoveredComponents)
		assert.Equal(t, "test-key", sig1.KeyID)
		assert.Equal(t, "ecdsa-p256", sig1.Algorithm)
	})
	
	// Test 1.3.1: Malformed input
	t.Run("malformed input", func(t *testing.T) {
		inputs := []string{
			`sig1=("@method"`,           // Missing closing parenthesis
			`sig1="key=val"`,            // Not RFC 8941 format
			`sig1=(method)`,             // Missing quotes
			`sig1=("@method";keyid="x"`, // Malformed parameters
		}
		
		for _, input := range inputs {
			_, err := ParseSignatureInput(input)
			assert.Error(t, err, "Input should fail: %s", input)
		}
	})
}

func TestParseSignature(t *testing.T) {
	// Test 1.2.1: Basic parsing
	t.Run("basic parsing", func(t *testing.T) {
		input := `sig1=:MEUCIQDkjN/g30k+A5U9F+a9ZcR6s5wzO8Y8Z8Y8Z8Y8Z8Y8ZwIgIiRBBBBCR4o/1eXgZQRGJwZBRxNf9Z6Hm3AmjZoU4w8=:`
		
		result, err := ParseSignature(input)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		sig1, exists := result["sig1"]
		require.True(t, exists)
		assert.NotEmpty(t, sig1)
		
		// Verify it's valid base64
		assert.Greater(t, len(sig1), 0)
	})
	
	// Test 1.3.2: Invalid base64
	t.Run("invalid base64", func(t *testing.T) {
		input := `sig1=:invalid-base64!@#$%^&*():`
		
		_, err := ParseSignature(input)
		assert.Error(t, err)
		// Should contain either "base64" or "byte sequence"
		assert.True(t, strings.Contains(err.Error(), "base64") || strings.Contains(err.Error(), "byte sequence"))
	})
	
	// Test multiple signatures
	t.Run("multiple signatures", func(t *testing.T) {
		input := `sig1=:MEUCIQDkjN/g30k+A5U9F+a9ZcR6s5wzO8Y8Z8Y8Z8Y8Z8Y8ZwIgIiRBBBBCR4o/1eXgZQRGJwZBRxNf9Z6Hm3AmjZoU4w8=:, sig2=:AQIDBAU=:`
		
		result, err := ParseSignature(input)
		require.NoError(t, err)
		require.NotNil(t, result)
		
		assert.Len(t, result, 2)
		assert.NotEmpty(t, result["sig1"])
		assert.NotEmpty(t, result["sig2"])
	})
}

func TestParseQueryParam(t *testing.T) {
	tests := []struct {
		name      string
		component string
		expected  string
		hasError  bool
	}{
		{
			name:      "valid query param",
			component: `"@query-param";name="id"`,
			expected:  "id",
			hasError:  false,
		},
		{
			name:      "with extra spaces",
			component: `"@query-param" ; name = "format"`,
			expected:  "format",
			hasError:  false,
		},
		{
			name:      "missing name parameter",
			component: `"@query-param"`,
			expected:  "",
			hasError:  true,
		},
		{
			name:      "invalid format",
			component: `@query-param;name=id`,
			expected:  "",
			hasError:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseQueryParam(tt.component)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}