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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateA2ACard(t *testing.T) {
	now := time.Now()
	metadata := &AgentMetadataV4{
		DID:         "did:sage:ethereum:0x123",
		Name:        "Test Agent",
		Description: "A test AI agent",
		Endpoint:    "https://api.example.com",
		Keys: []AgentKey{
			{
				Type:      KeyTypeEd25519,
				KeyData:   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				Verified:  true,
				CreatedAt: now,
			},
			{
				Type:      KeyTypeECDSA,
				KeyData:   []byte{33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65},
				Verified:  true,
				CreatedAt: now,
			},
			{
				Type:      KeyTypeX25519,
				KeyData:   []byte{66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97},
				Verified:  true,
				CreatedAt: now,
			},
		},
		Capabilities: map[string]interface{}{
			"capabilities": []interface{}{"message-signing", "chat"},
		},
		Owner:     "0xabc",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	card, err := GenerateA2ACard(metadata)
	require.NoError(t, err)
	require.NotNil(t, card)

	// Verify basic fields
	assert.Equal(t, string(metadata.DID), card.ID)
	assert.Equal(t, metadata.Name, card.Name)
	assert.Equal(t, metadata.Description, card.Description)
	assert.Equal(t, metadata.CreatedAt, card.Created)
	assert.Equal(t, metadata.UpdatedAt, card.Updated)

	// Verify context
	assert.Contains(t, card.Context, "https://www.w3.org/ns/did/v1")
	assert.Contains(t, card.Context, "https://w3id.org/security/suites/ed25519-2020/v1")

	// Verify type
	assert.Contains(t, card.Type, "Agent")
	assert.Contains(t, card.Type, "AIAgent")

	// Verify public keys (all 3 keys should be included)
	assert.Len(t, card.PublicKeys, 3)

	// Verify key types
	keyTypes := make(map[string]bool)
	for _, pk := range card.PublicKeys {
		keyTypes[pk.Type] = true
		assert.NotEmpty(t, pk.ID)
		assert.Equal(t, string(metadata.DID), pk.Controller)
		assert.NotEmpty(t, pk.PublicKeyBase58)
		assert.NotEmpty(t, pk.PublicKeyHex)
	}
	assert.True(t, keyTypes["Ed25519VerificationKey2020"])
	assert.True(t, keyTypes["EcdsaSecp256k1VerificationKey2019"])
	assert.True(t, keyTypes["X25519KeyAgreementKey2019"])

	// Verify endpoints
	require.Len(t, card.Endpoints, 1)
	assert.Equal(t, "MessageService", card.Endpoints[0].Type)
	assert.Equal(t, metadata.Endpoint, card.Endpoints[0].URI)

	// Verify capabilities
	assert.Contains(t, card.Capabilities, "message-signing")
	assert.Contains(t, card.Capabilities, "chat")
}

func TestGenerateA2ACard_NilMetadata(t *testing.T) {
	card, err := GenerateA2ACard(nil)
	assert.Error(t, err)
	assert.Nil(t, card)
	assert.Contains(t, err.Error(), "metadata cannot be nil")
}

func TestGenerateA2ACard_OnlyVerifiedKeys(t *testing.T) {
	now := time.Now()
	metadata := &AgentMetadataV4{
		DID:      "did:sage:ethereum:0x123",
		Name:     "Test Agent",
		Endpoint: "https://api.example.com",
		Keys: []AgentKey{
			{
				Type:      KeyTypeEd25519,
				KeyData:   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				Verified:  true,
				CreatedAt: now,
			},
			{
				Type:      KeyTypeECDSA,
				KeyData:   []byte{33, 34, 35, 36, 37, 38, 39, 40},
				Verified:  false, // Not verified - should be excluded
				CreatedAt: now,
			},
			{
				Type:      KeyTypeX25519,
				KeyData:   []byte{66, 67, 68, 69, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97},
				Verified:  true,
				CreatedAt: now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	card, err := GenerateA2ACard(metadata)
	require.NoError(t, err)
	require.NotNil(t, card)

	// Only verified keys should be included
	assert.Len(t, card.PublicKeys, 2)

	// Verify no ECDSA key (it was not verified)
	for _, pk := range card.PublicKeys {
		assert.NotEqual(t, "EcdsaSecp256k1VerificationKey2019", pk.Type)
	}
}

func TestGenerateA2ACard_WithAdditionalEndpoints(t *testing.T) {
	now := time.Now()
	metadata := &AgentMetadataV4{
		DID:      "did:sage:ethereum:0x123",
		Name:     "Test Agent",
		Endpoint: "https://api.example.com",
		Keys: []AgentKey{
			{
				Type:      KeyTypeEd25519,
				KeyData:   []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
				Verified:  true,
				CreatedAt: now,
			},
		},
		Capabilities: map[string]interface{}{
			"endpoints": []interface{}{
				map[string]interface{}{
					"type": "grpc",
					"uri":  "grpc://grpc.example.com",
				},
				map[string]interface{}{
					"type": "websocket",
					"uri":  "wss://ws.example.com",
				},
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	card, err := GenerateA2ACard(metadata)
	require.NoError(t, err)
	require.NotNil(t, card)

	// Should have 3 endpoints: MessageService + 2 additional
	assert.Len(t, card.Endpoints, 3)

	endpointTypes := make(map[string]string)
	for _, ep := range card.Endpoints {
		endpointTypes[ep.Type] = ep.URI
	}

	assert.Equal(t, "https://api.example.com", endpointTypes["MessageService"])
	assert.Equal(t, "grpc://grpc.example.com", endpointTypes["grpc"])
	assert.Equal(t, "wss://ws.example.com", endpointTypes["websocket"])
}

func TestMapKeyTypeToA2A(t *testing.T) {
	tests := []struct {
		name     string
		keyType  KeyType
		expected string
	}{
		{
			name:     "Ed25519",
			keyType:  KeyTypeEd25519,
			expected: "Ed25519VerificationKey2020",
		},
		{
			name:     "ECDSA",
			keyType:  KeyTypeECDSA,
			expected: "EcdsaSecp256k1VerificationKey2019",
		},
		{
			name:     "X25519",
			keyType:  KeyTypeX25519,
			expected: "X25519KeyAgreementKey2019",
		},
		{
			name:     "Unknown",
			keyType:  KeyType(999),
			expected: "UnknownKeyType",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapKeyTypeToA2A(tt.keyType))
		})
	}
}

func TestExtractCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		capMap   map[string]interface{}
		expected []string
	}{
		{
			name: "Capabilities key present",
			capMap: map[string]interface{}{
				"capabilities": []interface{}{"message-signing", "chat", "code"},
			},
			expected: []string{"message-signing", "chat", "code"},
		},
		{
			name: "Functions key present",
			capMap: map[string]interface{}{
				"functions": []interface{}{"encrypt", "decrypt"},
			},
			expected: []string{"encrypt", "decrypt"},
		},
		{
			name: "Empty map",
			capMap: map[string]interface{}{},
			expected: []string{"message-signing", "message-verification"},
		},
		{
			name:     "Nil map",
			capMap:   nil,
			expected: []string{"message-signing", "message-verification"},
		},
		{
			name: "Invalid capabilities type",
			capMap: map[string]interface{}{
				"capabilities": "not-an-array",
			},
			expected: []string{"message-signing", "message-verification"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCapabilities(tt.capMap)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateA2ACard(t *testing.T) {
	now := time.Now()
	validCard := &A2AAgentCard{
		Context:     []string{"https://www.w3.org/ns/did/v1"},
		ID:          "did:sage:ethereum:0x123",
		Type:        []string{"Agent"},
		Name:        "Test Agent",
		Description: "Test Description",
		PublicKeys: []A2APublicKey{
			{
				ID:              "did:sage:ethereum:0x123#key-1",
				Type:            "Ed25519VerificationKey2020",
				Controller:      "did:sage:ethereum:0x123",
				PublicKeyBase58: "5J3mBbAH58CpQ3Y2",
			},
		},
		Endpoints: []A2AEndpoint{
			{
				Type: "MessageService",
				URI:  "https://api.example.com",
			},
		},
		Capabilities: []string{"message-signing"},
		Created:      now,
		Updated:      now,
	}

	t.Run("Valid card", func(t *testing.T) {
		err := ValidateA2ACard(validCard)
		assert.NoError(t, err)
	})

	t.Run("Nil card", func(t *testing.T) {
		err := ValidateA2ACard(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "card cannot be nil")
	})

	t.Run("Missing ID", func(t *testing.T) {
		invalidCard := *validCard
		invalidCard.ID = ""
		err := ValidateA2ACard(&invalidCard)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "card ID is required")
	})

	t.Run("Missing Name", func(t *testing.T) {
		invalidCard := *validCard
		invalidCard.Name = ""
		err := ValidateA2ACard(&invalidCard)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "card name is required")
	})

	t.Run("No public keys", func(t *testing.T) {
		invalidCard := *validCard
		invalidCard.PublicKeys = []A2APublicKey{}
		err := ValidateA2ACard(&invalidCard)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one public key is required")
	})

	t.Run("Public key missing ID", func(t *testing.T) {
		invalidCard := *validCard
		invalidCard.PublicKeys[0].ID = ""
		err := ValidateA2ACard(&invalidCard)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "public key 0: ID is required")
	})

	t.Run("Public key missing Type", func(t *testing.T) {
		invalidCard := *validCard
		invalidCard.PublicKeys = []A2APublicKey{
			{
				ID:              "did:sage:ethereum:0x123#key-1",
				Type:            "", // Missing type
				Controller:      "did:sage:ethereum:0x123",
				PublicKeyBase58: "5J3mBbAH58CpQ3Y2",
			},
		}
		err := ValidateA2ACard(&invalidCard)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "public key 0: type is required")
	})

	t.Run("Public key missing Controller", func(t *testing.T) {
		invalidCard := *validCard
		invalidCard.PublicKeys = []A2APublicKey{
			{
				ID:              "did:sage:ethereum:0x123#key-1",
				Type:            "Ed25519VerificationKey2020",
				Controller:      "", // Missing controller
				PublicKeyBase58: "5J3mBbAH58CpQ3Y2",
			},
		}
		err := ValidateA2ACard(&invalidCard)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "public key 0: controller is required")
	})

	t.Run("Public key missing both Base58 and Hex", func(t *testing.T) {
		invalidCard := *validCard
		invalidCard.PublicKeys = []A2APublicKey{
			{
				ID:              "did:sage:ethereum:0x123#key-1",
				Type:            "Ed25519VerificationKey2020",
				Controller:      "did:sage:ethereum:0x123",
				PublicKeyBase58: "", // Missing
				PublicKeyHex:    "", // Missing
			},
		}
		err := ValidateA2ACard(&invalidCard)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "public key 0: either publicKeyBase58 or publicKeyHex is required")
	})

	t.Run("No endpoints", func(t *testing.T) {
		invalidCard := *validCard
		invalidCard.PublicKeys = []A2APublicKey{
			{
				ID:              "did:sage:ethereum:0x123#key-1",
				Type:            "Ed25519VerificationKey2020",
				Controller:      "did:sage:ethereum:0x123",
				PublicKeyBase58: "5J3mBbAH58CpQ3Y2",
			},
		}
		invalidCard.Endpoints = []A2AEndpoint{}
		err := ValidateA2ACard(&invalidCard)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one endpoint is required")
	})

	t.Run("Endpoint missing Type", func(t *testing.T) {
		invalidCard := *validCard
		invalidCard.PublicKeys = []A2APublicKey{
			{
				ID:              "did:sage:ethereum:0x123#key-1",
				Type:            "Ed25519VerificationKey2020",
				Controller:      "did:sage:ethereum:0x123",
				PublicKeyBase58: "5J3mBbAH58CpQ3Y2",
			},
		}
		invalidCard.Endpoints = []A2AEndpoint{
			{
				Type: "", // Missing type
				URI:  "https://api.example.com",
			},
		}
		err := ValidateA2ACard(&invalidCard)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint 0: type is required")
	})

	t.Run("Endpoint missing URI", func(t *testing.T) {
		invalidCard := *validCard
		invalidCard.PublicKeys = []A2APublicKey{
			{
				ID:              "did:sage:ethereum:0x123#key-1",
				Type:            "Ed25519VerificationKey2020",
				Controller:      "did:sage:ethereum:0x123",
				PublicKeyBase58: "5J3mBbAH58CpQ3Y2",
			},
		}
		invalidCard.Endpoints = []A2AEndpoint{
			{
				Type: "MessageService",
				URI:  "", // Missing URI
			},
		}
		err := ValidateA2ACard(&invalidCard)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "endpoint 0: URI is required")
	})
}

func TestMergeA2ACard(t *testing.T) {
	now := time.Now()
	metadata := &AgentMetadataV4{
		DID:          "did:sage:ethereum:0x123",
		Name:         "Old Name",
		Description:  "Old Description",
		Endpoint:     "https://old.example.com",
		Keys:         []AgentKey{},
		Capabilities: map[string]interface{}{},
		CreatedAt:    now.Add(-time.Hour),
		UpdatedAt:    now.Add(-time.Hour),
	}

	card := &A2AAgentCard{
		Context:     []string{"https://www.w3.org/ns/did/v1"},
		ID:          "did:sage:ethereum:0x123",
		Type:        []string{"Agent"},
		Name:        "New Name",
		Description: "New Description",
		PublicKeys: []A2APublicKey{
			{
				ID:              "did:sage:ethereum:0x123#key-1",
				Type:            "Ed25519VerificationKey2020",
				Controller:      "did:sage:ethereum:0x123",
				PublicKeyBase58: "5J3mBbAH58CpQ3Y2",
			},
		},
		Endpoints: []A2AEndpoint{
			{
				Type: "MessageService",
				URI:  "https://new.example.com",
			},
			{
				Type: "grpc",
				URI:  "grpc://grpc.example.com",
			},
		},
		Capabilities: []string{"message-signing", "chat"},
		Created:      now,
		Updated:      now,
	}

	err := MergeA2ACard(metadata, card)
	require.NoError(t, err)

	// Verify metadata updated
	assert.Equal(t, "New Name", metadata.Name)
	assert.Equal(t, "New Description", metadata.Description)
	assert.Equal(t, "https://new.example.com", metadata.Endpoint)

	// Verify capabilities merged
	caps, ok := metadata.Capabilities["capabilities"].([]string)
	require.True(t, ok)
	assert.Contains(t, caps, "message-signing")
	assert.Contains(t, caps, "chat")

	// Verify endpoints merged
	endpoints, ok := metadata.Capabilities["endpoints"].([]map[string]interface{})
	require.True(t, ok)
	assert.Len(t, endpoints, 2)

	// Verify timestamps updated
	assert.Equal(t, now, metadata.CreatedAt)
	assert.Equal(t, now, metadata.UpdatedAt)
}

func TestMergeA2ACard_NilInputs(t *testing.T) {
	now := time.Now()
	metadata := &AgentMetadataV4{
		DID:       "did:sage:ethereum:0x123",
		CreatedAt: now,
	}
	card := &A2AAgentCard{
		ID:   "did:sage:ethereum:0x123",
		Name: "Test",
		PublicKeys: []A2APublicKey{
			{
				ID:              "key-1",
				Type:            "Ed25519VerificationKey2020",
				Controller:      "did:sage:ethereum:0x123",
				PublicKeyBase58: "test",
			},
		},
		Endpoints: []A2AEndpoint{{Type: "test", URI: "https://test"}},
	}

	t.Run("Nil metadata", func(t *testing.T) {
		err := MergeA2ACard(nil, card)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metadata cannot be nil")
	})

	t.Run("Nil card", func(t *testing.T) {
		err := MergeA2ACard(metadata, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "card cannot be nil")
	})
}

func TestMergeA2ACard_InvalidCard(t *testing.T) {
	now := time.Now()
	metadata := &AgentMetadataV4{
		DID:       "did:sage:ethereum:0x123",
		CreatedAt: now,
	}

	invalidCard := &A2AAgentCard{
		ID: "", // Invalid - missing ID
	}

	err := MergeA2ACard(metadata, invalidCard)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid A2A card")
}
