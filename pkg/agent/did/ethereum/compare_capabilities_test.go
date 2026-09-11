package ethereum

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestCompareCapabilitiesEdgeCases tests additional edge cases for capability comparison
func TestCompareCapabilitiesEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		cap1     map[string]interface{}
		cap2     map[string]interface{}
		expected bool
	}{
		{
			name: "Nested capabilities - complex structure",
			cap1: map[string]interface{}{
				"chat": true,
				"advanced": map[string]interface{}{
					"translation": true,
					"summarization": map[string]interface{}{
						"enabled":   true,
						"maxLength": 1000,
					},
				},
			},
			cap2: map[string]interface{}{
				"chat": true,
				"advanced": map[string]interface{}{
					"translation": true,
					"summarization": map[string]interface{}{
						"enabled":   true,
						"maxLength": 1000,
					},
				},
			},
			expected: true,
		},
		{
			name: "Deep nested capabilities - different",
			cap1: map[string]interface{}{
				"advanced": map[string]interface{}{
					"summarization": map[string]interface{}{
						"maxLength": 1000,
					},
				},
			},
			cap2: map[string]interface{}{
				"advanced": map[string]interface{}{
					"summarization": map[string]interface{}{
						"maxLength": 2000,
					},
				},
			},
			expected: false,
		},
		{
			name:     "Both nil capabilities",
			cap1:     nil,
			cap2:     nil,
			expected: true,
		},
		{
			name:     "Empty vs nil capabilities",
			cap1:     map[string]interface{}{},
			cap2:     nil,
			expected: false, // JSON marshaling treats {} and null differently
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareCapabilities(tt.cap1, tt.cap2)
			assert.Equal(t, tt.expected, result)
		})
	}
}
