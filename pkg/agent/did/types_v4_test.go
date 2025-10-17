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

func TestKeyType_String(t *testing.T) {
	tests := []struct {
		name     string
		keyType  KeyType
		expected string
	}{
		{
			name:     "Ed25519",
			keyType:  KeyTypeEd25519,
			expected: "Ed25519",
		},
		{
			name:     "ECDSA",
			keyType:  KeyTypeECDSA,
			expected: "ECDSA",
		},
		{
			name:     "X25519",
			keyType:  KeyTypeX25519,
			expected: "X25519",
		},
		{
			name:     "Unknown",
			keyType:  KeyType(999),
			expected: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.keyType.String())
		})
	}
}

func TestAgentMetadataV4_GetKeyByType(t *testing.T) {
	now := time.Now()
	ed25519Key := AgentKey{
		Type:      KeyTypeEd25519,
		KeyData:   []byte("ed25519-key-32-bytes-test-data!!"),
		Verified:  true,
		CreatedAt: now,
	}
	ecdsaKey := AgentKey{
		Type:      KeyTypeECDSA,
		KeyData:   []byte("ecdsa-key-65-bytes"),
		Verified:  true,
		CreatedAt: now,
	}
	x25519Key := AgentKey{
		Type:      KeyTypeX25519,
		KeyData:   []byte("x25519-key-32-bytes-test-data!!!"),
		Verified:  true,
		CreatedAt: now,
	}

	metadata := &AgentMetadataV4{
		DID:  "did:sage:ethereum:0x123",
		Keys: []AgentKey{ed25519Key, ecdsaKey, x25519Key},
	}

	tests := []struct {
		name     string
		keyType  KeyType
		expected *AgentKey
	}{
		{
			name:     "Get Ed25519 key",
			keyType:  KeyTypeEd25519,
			expected: &ed25519Key,
		},
		{
			name:     "Get ECDSA key",
			keyType:  KeyTypeECDSA,
			expected: &ecdsaKey,
		},
		{
			name:     "Get X25519 key",
			keyType:  KeyTypeX25519,
			expected: &x25519Key,
		},
		{
			name:     "Key not found",
			keyType:  KeyType(999),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := metadata.GetKeyByType(tt.keyType)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Type, result.Type)
				assert.Equal(t, tt.expected.KeyData, result.KeyData)
			}
		})
	}
}

func TestAgentMetadataV4_GetVerifiedKeys(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name          string
		keys          []AgentKey
		expectedCount int
	}{
		{
			name: "All keys verified",
			keys: []AgentKey{
				{Type: KeyTypeEd25519, KeyData: []byte("key1"), Verified: true, CreatedAt: now},
				{Type: KeyTypeECDSA, KeyData: []byte("key2"), Verified: true, CreatedAt: now},
				{Type: KeyTypeX25519, KeyData: []byte("key3"), Verified: true, CreatedAt: now},
			},
			expectedCount: 3,
		},
		{
			name: "Some keys not verified",
			keys: []AgentKey{
				{Type: KeyTypeEd25519, KeyData: []byte("key1"), Verified: true, CreatedAt: now},
				{Type: KeyTypeECDSA, KeyData: []byte("key2"), Verified: false, CreatedAt: now},
				{Type: KeyTypeX25519, KeyData: []byte("key3"), Verified: true, CreatedAt: now},
			},
			expectedCount: 2,
		},
		{
			name: "No keys verified",
			keys: []AgentKey{
				{Type: KeyTypeEd25519, KeyData: []byte("key1"), Verified: false, CreatedAt: now},
				{Type: KeyTypeECDSA, KeyData: []byte("key2"), Verified: false, CreatedAt: now},
			},
			expectedCount: 0,
		},
		{
			name:          "Empty keys",
			keys:          []AgentKey{},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := &AgentMetadataV4{
				DID:  "did:sage:ethereum:0x123",
				Keys: tt.keys,
			}
			verified := metadata.GetVerifiedKeys()
			assert.Equal(t, tt.expectedCount, len(verified))

			// Verify all returned keys are verified
			for _, key := range verified {
				assert.True(t, key.Verified)
			}
		})
	}
}

func TestAgentMetadataV4_HasKeyType(t *testing.T) {
	now := time.Now()
	metadata := &AgentMetadataV4{
		DID: "did:sage:ethereum:0x123",
		Keys: []AgentKey{
			{Type: KeyTypeEd25519, KeyData: []byte("key1"), Verified: true, CreatedAt: now},
			{Type: KeyTypeECDSA, KeyData: []byte("key2"), Verified: true, CreatedAt: now},
		},
	}

	tests := []struct {
		name     string
		keyType  KeyType
		expected bool
	}{
		{
			name:     "Has Ed25519",
			keyType:  KeyTypeEd25519,
			expected: true,
		},
		{
			name:     "Has ECDSA",
			keyType:  KeyTypeECDSA,
			expected: true,
		},
		{
			name:     "Does not have X25519",
			keyType:  KeyTypeX25519,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, metadata.HasKeyType(tt.keyType))
		})
	}
}

func TestAgentMetadataV4_ToAgentMetadata(t *testing.T) {
	now := time.Now()
	ecdsaKeyData := []byte("ecdsa-public-key-data")
	x25519KeyData := []byte("x25519-kem-key-data")

	v4Metadata := &AgentMetadataV4{
		DID:         "did:sage:ethereum:0x123",
		Name:        "Test Agent",
		Description: "Test Description",
		Endpoint:    "https://test.example.com",
		Keys: []AgentKey{
			{Type: KeyTypeEd25519, KeyData: []byte("ed25519-key"), Verified: true, CreatedAt: now},
			{Type: KeyTypeECDSA, KeyData: ecdsaKeyData, Verified: true, CreatedAt: now},
			{Type: KeyTypeX25519, KeyData: x25519KeyData, Verified: true, CreatedAt: now},
		},
		Capabilities: map[string]interface{}{
			"chat": true,
		},
		Owner:     "0xabc",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	legacy := v4Metadata.ToAgentMetadata()

	assert.Equal(t, v4Metadata.DID, legacy.DID)
	assert.Equal(t, v4Metadata.Name, legacy.Name)
	assert.Equal(t, v4Metadata.Description, legacy.Description)
	assert.Equal(t, v4Metadata.Endpoint, legacy.Endpoint)
	assert.Equal(t, ecdsaKeyData, legacy.PublicKey.([]byte))
	assert.Equal(t, x25519KeyData, legacy.PublicKEMKey.([]byte))
	assert.Equal(t, v4Metadata.Capabilities, legacy.Capabilities)
	assert.Equal(t, v4Metadata.Owner, legacy.Owner)
	assert.Equal(t, v4Metadata.IsActive, legacy.IsActive)
	assert.Equal(t, v4Metadata.CreatedAt, legacy.CreatedAt)
	assert.Equal(t, v4Metadata.UpdatedAt, legacy.UpdatedAt)
}

func TestAgentMetadataV4_ToAgentMetadata_MissingKeys(t *testing.T) {
	now := time.Now()

	// Test with no ECDSA key
	v4NoEcdsa := &AgentMetadataV4{
		DID:  "did:sage:ethereum:0x123",
		Keys: []AgentKey{{Type: KeyTypeEd25519, KeyData: []byte("ed25519-key"), Verified: true, CreatedAt: now}},
	}
	legacyNoEcdsa := v4NoEcdsa.ToAgentMetadata()
	assert.Nil(t, legacyNoEcdsa.PublicKey)

	// Test with no X25519 key
	v4NoX25519 := &AgentMetadataV4{
		DID:  "did:sage:ethereum:0x456",
		Keys: []AgentKey{{Type: KeyTypeECDSA, KeyData: []byte("ecdsa-key"), Verified: true, CreatedAt: now}},
	}
	legacyNoX25519 := v4NoX25519.ToAgentMetadata()
	assert.Nil(t, legacyNoX25519.PublicKEMKey)
}

func TestFromAgentMetadata(t *testing.T) {
	now := time.Now()
	ecdsaKeyData := []byte("ecdsa-public-key-data")
	x25519KeyData := []byte("x25519-kem-key-data")

	legacy := &AgentMetadata{
		DID:          "did:sage:ethereum:0x123",
		Name:         "Test Agent",
		Description:  "Test Description",
		Endpoint:     "https://test.example.com",
		PublicKey:    ecdsaKeyData,
		PublicKEMKey: x25519KeyData,
		Capabilities: map[string]interface{}{
			"chat": true,
		},
		Owner:     "0xabc",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	v4 := FromAgentMetadata(legacy)

	assert.Equal(t, legacy.DID, v4.DID)
	assert.Equal(t, legacy.Name, v4.Name)
	assert.Equal(t, legacy.Description, v4.Description)
	assert.Equal(t, legacy.Endpoint, v4.Endpoint)
	assert.Equal(t, legacy.Capabilities, v4.Capabilities)
	assert.Equal(t, legacy.Owner, v4.Owner)
	assert.Equal(t, legacy.IsActive, v4.IsActive)
	assert.Equal(t, legacy.CreatedAt, v4.CreatedAt)
	assert.Equal(t, legacy.UpdatedAt, v4.UpdatedAt)

	// Verify keys conversion
	require.Len(t, v4.Keys, 2)

	// Check ECDSA key
	ecdsaKey := v4.GetKeyByType(KeyTypeECDSA)
	require.NotNil(t, ecdsaKey)
	assert.Equal(t, ecdsaKeyData, ecdsaKey.KeyData)
	assert.True(t, ecdsaKey.Verified)

	// Check X25519 key
	x25519Key := v4.GetKeyByType(KeyTypeX25519)
	require.NotNil(t, x25519Key)
	assert.Equal(t, x25519KeyData, x25519Key.KeyData)
	assert.True(t, x25519Key.Verified)
}

func TestFromAgentMetadata_MissingKeys(t *testing.T) {
	now := time.Now()

	// Test with no PublicKey
	legacyNoPublic := &AgentMetadata{
		DID:          "did:sage:ethereum:0x123",
		PublicKEMKey: []byte("kem-key"),
		CreatedAt:    now,
	}
	v4NoPublic := FromAgentMetadata(legacyNoPublic)
	assert.Len(t, v4NoPublic.Keys, 1)
	assert.NotNil(t, v4NoPublic.GetKeyByType(KeyTypeX25519))
	assert.Nil(t, v4NoPublic.GetKeyByType(KeyTypeECDSA))

	// Test with no PublicKEMKey
	legacyNoKEM := &AgentMetadata{
		DID:       "did:sage:ethereum:0x456",
		PublicKey: []byte("public-key"),
		CreatedAt: now,
	}
	v4NoKEM := FromAgentMetadata(legacyNoKEM)
	assert.Len(t, v4NoKEM.Keys, 1)
	assert.NotNil(t, v4NoKEM.GetKeyByType(KeyTypeECDSA))
	assert.Nil(t, v4NoKEM.GetKeyByType(KeyTypeX25519))

	// Test with non-byte slice keys
	legacyInvalidKeys := &AgentMetadata{
		DID:          "did:sage:ethereum:0x789",
		PublicKey:    "not-a-byte-slice",
		PublicKEMKey: 12345,
		CreatedAt:    now,
	}
	v4InvalidKeys := FromAgentMetadata(legacyInvalidKeys)
	assert.Len(t, v4InvalidKeys.Keys, 0)
}

func TestAgentMetadataV4_RoundTrip(t *testing.T) {
	now := time.Now()
	original := &AgentMetadataV4{
		DID:         "did:sage:ethereum:0x123",
		Name:        "Test Agent",
		Description: "Test Description",
		Endpoint:    "https://test.example.com",
		Keys: []AgentKey{
			{Type: KeyTypeECDSA, KeyData: []byte("ecdsa-key"), Verified: true, CreatedAt: now},
			{Type: KeyTypeX25519, KeyData: []byte("x25519-key"), Verified: true, CreatedAt: now},
		},
		Capabilities: map[string]interface{}{
			"chat": true,
		},
		Owner:     "0xabc",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Convert V4 -> Legacy -> V4
	legacy := original.ToAgentMetadata()
	converted := FromAgentMetadata(legacy)

	// Verify round-trip preserves data
	assert.Equal(t, original.DID, converted.DID)
	assert.Equal(t, original.Name, converted.Name)
	assert.Equal(t, original.Description, converted.Description)
	assert.Equal(t, original.Endpoint, converted.Endpoint)
	assert.Equal(t, original.Owner, converted.Owner)
	assert.Equal(t, original.IsActive, converted.IsActive)

	// Verify keys preserved
	assert.Len(t, converted.Keys, 2)
	assert.NotNil(t, converted.GetKeyByType(KeyTypeECDSA))
	assert.NotNil(t, converted.GetKeyByType(KeyTypeX25519))
}
