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
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifier(t *testing.T) {
	// Generate test keypair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	
	verifier := NewVerifier()
	
	t.Run("VerifySignature with valid EdDSA signature", func(t *testing.T) {
		message := &Message{
			AgentDID:     "did:sage:ethereum:agent001",
			MessageID:    "msg-001",
			Timestamp:    time.Now(),
			Nonce:        "random-nonce",
			Body:         []byte("test message"),
			Algorithm:    string(AlgorithmEdDSA),
			SignedFields: []string{"agent_did", "message_id", "timestamp", "nonce", "body"},
		}
		
		// Sign the message
		signatureBase := verifier.ConstructSignatureBase(message)
		message.Signature = ed25519.Sign(privateKey, []byte(signatureBase))
		
		// Verify
		err := verifier.VerifySignature(publicKey, message, nil)
		assert.NoError(t, err)
	})
	
	t.Run("VerifySignature with invalid signature", func(t *testing.T) {
		message := &Message{
			AgentDID:     "did:sage:ethereum:agent001",
			MessageID:    "msg-002",
			Timestamp:    time.Now(),
			Body:         []byte("test message"),
			Algorithm:    string(AlgorithmEdDSA),
			SignedFields: []string{"body"},
			Signature:    []byte("invalid signature"),
		}
		
		err := verifier.VerifySignature(publicKey, message, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature verification failed")
	})
	
	t.Run("VerifySignature with clock skew", func(t *testing.T) {
		message := &Message{
			AgentDID:     "did:sage:ethereum:agent001",
			MessageID:    "msg-003",
			Timestamp:    time.Now().Add(10 * time.Minute), // Future timestamp
			Body:         []byte("test message"),
			Algorithm:    string(AlgorithmEdDSA),
			SignedFields: []string{"body"},
			Signature:    []byte("dummy"),
		}
		
		opts := &VerificationOptions{
			MaxClockSkew: 5 * time.Minute,
		}
		
		err := verifier.VerifySignature(publicKey, message, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timestamp outside acceptable range")
	})
	
	t.Run("VerifyWithMetadata", func(t *testing.T) {
		message := &Message{
			AgentDID:     "did:sage:ethereum:agent001",
			MessageID:    "msg-004",
			Timestamp:    time.Now(),
			Body:         []byte("test message"),
			Algorithm:    string(AlgorithmEdDSA),
			SignedFields: []string{"body"},
			Metadata: map[string]interface{}{
				"endpoint": "https://api.example.com",
				"capabilities": map[string]interface{}{
					"chat": true,
					"code": true,
				},
			},
		}
		
		// Sign the message
		signatureBase := verifier.ConstructSignatureBase(message)
		message.Signature = ed25519.Sign(privateKey, []byte(signatureBase))
		
		expectedMetadata := map[string]interface{}{
			"endpoint": "https://api.example.com",
		}
		
		requiredCapabilities := []string{"chat"}
		
		opts := &VerificationOptions{
			VerifyMetadata: true,
		}
		
		result, err := verifier.VerifyWithMetadata(publicKey, message, expectedMetadata, requiredCapabilities, opts)
		require.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Error)
	})
	
	t.Run("VerifyWithMetadata missing capability", func(t *testing.T) {
		message := &Message{
			AgentDID:     "did:sage:ethereum:agent005",
			MessageID:    "msg-005",
			Timestamp:    time.Now(),
			Body:         []byte("test message"),
			Algorithm:    string(AlgorithmEdDSA),
			SignedFields: []string{"body"},
			Metadata: map[string]interface{}{
				"capabilities": map[string]interface{}{
					"chat": true,
				},
			},
		}
		
		// Sign the message
		signatureBase := verifier.ConstructSignatureBase(message)
		message.Signature = ed25519.Sign(privateKey, []byte(signatureBase))
		
		requiredCapabilities := []string{"chat", "code"} // Missing "code"
		
		result, err := verifier.VerifyWithMetadata(publicKey, message, nil, requiredCapabilities, nil)
		require.NoError(t, err)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Error, "missing required capabilities")
	})
}

func TestConstructSignatureBase(t *testing.T) {
	verifier := &Verifier{}
	
	message := &Message{
		AgentDID:     "did:sage:ethereum:agent001",
		MessageID:    "msg-001",
		Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Nonce:        "nonce123",
		Body:         []byte("test body"),
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-Custom":     "value",
		},
		SignedFields: []string{"agent_did", "message_id", "timestamp", "nonce", "body", "header.Content-Type"},
	}
	
	expected := `agent_did: did:sage:ethereum:agent001
message_id: msg-001
timestamp: 2024-01-01T12:00:00Z
nonce: nonce123
body: test body
Content-Type: application/json`
	
	result := verifier.ConstructSignatureBase(message)
	assert.Equal(t, expected, result)
}

func TestDefaultVerificationOptions(t *testing.T) {
	opts := DefaultVerificationOptions()
	
	assert.True(t, opts.RequireActiveAgent)
	assert.Equal(t, 5*time.Minute, opts.MaxClockSkew)
	assert.True(t, opts.VerifyMetadata)
	assert.Empty(t, opts.RequiredCapabilities)
}

func TestHasRequiredCapabilities(t *testing.T) {
	capabilities := map[string]interface{}{
		"chat": true,
		"code": true,
		"voice": false,
	}
	
	tests := []struct {
		name     string
		required []string
		expected bool
	}{
		{
			name:     "All capabilities present",
			required: []string{"chat", "code"},
			expected: true,
		},
		{
			name:     "Missing capability",
			required: []string{"chat", "video"},
			expected: false,
		},
		{
			name:     "Empty required",
			required: []string{},
			expected: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasRequiredCapabilities(capabilities, tt.required)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompareValues(t *testing.T) {
	tests := []struct {
		name     string
		v1       interface{}
		v2       interface{}
		expected bool
	}{
		{
			name:     "Equal strings",
			v1:       "test",
			v2:       "test",
			expected: true,
		},
		{
			name:     "Different strings",
			v1:       "test1",
			v2:       "test2",
			expected: false,
		},
		{
			name:     "Equal maps",
			v1:       map[string]interface{}{"key": "value"},
			v2:       map[string]interface{}{"key": "value"},
			expected: true,
		},
		{
			name:     "Different maps",
			v1:       map[string]interface{}{"key": "value1"},
			v2:       map[string]interface{}{"key": "value2"},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareValues(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}
