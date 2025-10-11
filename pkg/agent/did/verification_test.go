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

package did

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataVerifier(t *testing.T) {
	ctx := context.Background()

	// Generate test keypair
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	// Create mock resolver
	mockResolver := new(MockResolver)
	verifier := NewMetadataVerifier(mockResolver)

	t.Run("ValidateAgent with active agent", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent001")

		// Mock agent metadata
		agent := &AgentMetadata{
			DID:         did,
			Name:        "Test Agent",
			Description: "A test agent",
			Endpoint:    "https://agent.example.com",
			PublicKey:   publicKey,
			Capabilities: map[string]interface{}{
				"messaging": true,
				"compute":   true,
			},
			Owner:     "0x1234567890abcdef",
			IsActive:  true,
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now(),
		}

		mockResolver.On("Resolve", ctx, did).Return(agent, nil)

		opts := &ValidationOptions{
			RequireActiveAgent: true,
		}

		result, err := verifier.ValidateAgent(ctx, did, opts)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, agent.Name, result.Name)
		assert.True(t, result.IsActive)
	})

	t.Run("ValidateAgent with inactive agent", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent002")

		agent := &AgentMetadata{
			DID:      did,
			IsActive: false,
		}

		mockResolver.On("Resolve", ctx, did).Return(agent, nil)

		opts := &ValidationOptions{
			RequireActiveAgent: true,
		}

		_, err := verifier.ValidateAgent(ctx, did, opts)
		assert.Error(t, err)
		assert.Equal(t, ErrInactiveAgent, err)
	})

	t.Run("CheckCapabilities", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent003")

		agent := &AgentMetadata{
			DID:      did,
			IsActive: true,
			Capabilities: map[string]interface{}{
				"messaging": true,
				"compute":   true,
				"storage":   false,
			},
		}

		mockResolver.On("Resolve", ctx, did).Return(agent, nil)

		// Test with capabilities the agent has
		hasCapability, err := verifier.CheckCapabilities(ctx, did, []string{"messaging", "compute"})
		assert.NoError(t, err)
		assert.True(t, hasCapability)

		// Test with capabilities the agent doesn't have
		hasCapability, err = verifier.CheckCapabilities(ctx, did, []string{"messaging", "ai-training"})
		assert.NoError(t, err)
		assert.False(t, hasCapability)
	})

	t.Run("MatchMetadata", func(t *testing.T) {
		agent := &AgentMetadata{
			Name:     "Test Agent",
			Endpoint: "https://agent.example.com",
			Capabilities: map[string]interface{}{
				"messaging": true,
				"compute": map[string]interface{}{
					"max_memory": "16GB",
					"gpu":        true,
				},
			},
		}

		// Test exact match
		expectedValues := map[string]interface{}{
			"name":     "Test Agent",
			"endpoint": "https://agent.example.com",
			"capabilities": map[string]interface{}{
				"messaging": true,
			},
		}

		err := verifier.MatchMetadata(agent, expectedValues)
		assert.NoError(t, err)

		// Test mismatch
		expectedValues["endpoint"] = "https://different.com"
		err = verifier.MatchMetadata(agent, expectedValues)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint mismatch")
	})

	t.Run("ValidateAgentForOperation", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent004")

		agent := &AgentMetadata{
			DID:      did,
			Name:     "AI Agent",
			IsActive: true,
			Capabilities: map[string]interface{}{
				"messaging": true,
				"compute":   true,
				// ai-training capability is missing, not false
			},
		}

		mockResolver.On("Resolve", ctx, did).Return(agent, nil).Twice()

		// Test valid operation
		result, err := verifier.ValidateAgentForOperation(ctx, did, "send-message", []string{"messaging"})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid)
		assert.Equal(t, "send-message", result.OperationType)

		// Test operation with missing capabilities
		result, err = verifier.ValidateAgentForOperation(ctx, did, "train-model", []string{"ai-training", "compute"})
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Valid)
		assert.Contains(t, result.Error, "missing required capabilities")
		assert.Contains(t, result.MissingCapabilities, "ai-training")
	})

	t.Run("VerifyMetadataConsistency", func(t *testing.T) {
		did := AgentDID("did:sage:ethereum:agent005")

		providedMetadata := &AgentMetadata{
			DID:      did,
			Name:     "Consistent Agent",
			Endpoint: "https://agent.example.com",
		}

		expectedResult := &VerificationResult{
			Valid:      true,
			Agent:      providedMetadata,
			VerifiedAt: time.Now(),
		}

		mockResolver.On("VerifyMetadata", ctx, did, providedMetadata).Return(expectedResult, nil)

		result, err := verifier.VerifyMetadataConsistency(ctx, did, providedMetadata)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid)
	})
}

func TestValidationOptions(t *testing.T) {
	opts := DefaultValidationOptions()
	assert.NotNil(t, opts)
	assert.True(t, opts.RequireActiveAgent)
	assert.False(t, opts.ValidateEndpoint)
}

func TestHelperFunctions(t *testing.T) {
	t.Run("hasRequiredCapabilities", func(t *testing.T) {
		agentCaps := map[string]interface{}{
			"messaging": true,
			"compute":   true,
			"storage":   false,
		}

		// All required capabilities present
		assert.True(t, hasRequiredCapabilities(agentCaps, []string{"messaging", "compute"}))

		// Missing capability
		assert.False(t, hasRequiredCapabilities(agentCaps, []string{"messaging", "ai-training"}))

		// Empty required capabilities
		assert.True(t, hasRequiredCapabilities(agentCaps, []string{}))
	})

	t.Run("findMissingCapabilities", func(t *testing.T) {
		agentCaps := map[string]interface{}{
			"messaging": true,
			"compute":   true,
		}

		missing := findMissingCapabilities(agentCaps, []string{"messaging", "storage", "ai-training"})
		assert.Len(t, missing, 2)
		assert.Contains(t, missing, "storage")
		assert.Contains(t, missing, "ai-training")
	})

	t.Run("compareValues", func(t *testing.T) {
		// Simple values
		assert.True(t, compareValues("test", "test"))
		assert.False(t, compareValues("test", "different"))

		// Complex values
		v1 := map[string]interface{}{"key": "value", "num": 42}
		v2 := map[string]interface{}{"key": "value", "num": 42}
		v3 := map[string]interface{}{"key": "value", "num": 43}

		assert.True(t, compareValues(v1, v2))
		assert.False(t, compareValues(v1, v3))
	})
}
