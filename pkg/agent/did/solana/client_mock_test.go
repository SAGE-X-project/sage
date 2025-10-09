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

package solana

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/stretchr/testify/assert"
)

// TestRegisterKeyTypeValidation tests that Register validates key type
func TestRegisterKeyTypeValidation(t *testing.T) {
	client := &SolanaClient{
		config: &did.RegistryConfig{
			MaxRetries: 3,
		},
	}

	tests := []struct {
		name    string
		keyType sagecrypto.KeyType
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid Ed25519 key",
			keyType: sagecrypto.KeyTypeEd25519,
			wantErr: true, // Will still error due to missing blockchain, but should pass key type check
		},
		{
			name:    "Invalid Secp256k1 key",
			keyType: sagecrypto.KeyTypeSecp256k1,
			wantErr: true,
			errMsg:  "Solana requires Ed25519 keys",
		},
		{
			name:    "Invalid X25519 key",
			keyType: sagecrypto.KeyTypeX25519,
			wantErr: true,
			errMsg:  "Solana requires Ed25519 keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockKeyPair := &mockKeyPairForType{keyType: tt.keyType}

			req := &did.RegistrationRequest{
				DID:          "did:sage:solana:test",
				Name:         "Test Agent",
				Description:  "Test Description",
				Endpoint:     "https://api.example.com",
				KeyPair:      mockKeyPair,
				Capabilities: map[string]interface{}{"chat": true},
			}

			_, err := client.Register(context.Background(), req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestPrepareRegistrationMessage tests message preparation
func TestPrepareRegistrationMessage(t *testing.T) {
	client := &SolanaClient{
		config: &did.RegistryConfig{},
	}

	tests := []struct {
		name     string
		req      *did.RegistrationRequest
		address  string
		expected []string
	}{
		{
			name: "Basic registration message",
			req: &did.RegistrationRequest{
				DID:      "did:sage:solana:agent001",
				Name:     "Test Agent",
				Endpoint: "https://api.example.com",
			},
			address: "11111111111111111111111111111111",
			expected: []string{
				"Register agent:",
				"did:sage:solana:agent001",
				"Test Agent",
				"https://api.example.com",
				"11111111111111111111111111111111",
			},
		},
		{
			name: "Registration with special characters",
			req: &did.RegistrationRequest{
				DID:      "did:sage:solana:agent-special-123",
				Name:     "Agent with Spaces & Special",
				Endpoint: "https://api.example.com/path?key=value",
			},
			address: "FooBarBaz1234567890123456789012",
			expected: []string{
				"Register agent:",
				"did:sage:solana:agent-special-123",
				"Agent with Spaces & Special",
				"https://api.example.com/path?key=value",
				"FooBarBaz1234567890123456789012",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := client.prepareRegistrationMessage(tt.req, tt.address)

			for _, expected := range tt.expected {
				assert.Contains(t, message, expected)
			}
		})
	}
}

// TestPrepareUpdateMessage tests update message preparation
func TestPrepareUpdateMessage(t *testing.T) {
	client := &SolanaClient{
		config: &did.RegistryConfig{},
	}

	tests := []struct {
		name     string
		agentDID did.AgentDID
		updates  map[string]interface{}
		expected []string
	}{
		{
			name:     "Basic update message",
			agentDID: "did:sage:solana:agent001",
			updates: map[string]interface{}{
				"name":        "Updated Agent",
				"description": "New description",
			},
			expected: []string{
				"Update agent:",
				"did:sage:solana:agent001",
			},
		},
		{
			name:     "Update with capabilities",
			agentDID: "did:sage:solana:agent002",
			updates: map[string]interface{}{
				"capabilities": map[string]interface{}{
					"chat": true,
					"code": true,
				},
			},
			expected: []string{
				"Update agent:",
				"did:sage:solana:agent002",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := client.prepareUpdateMessage(tt.agentDID, tt.updates)

			for _, expected := range tt.expected {
				assert.Contains(t, message, expected)
			}
		})
	}
}

// TestAgentAccountSerialization tests AgentAccount data structure
func TestAgentAccountSerialization(t *testing.T) {
	account := AgentAccount{
		DID:         "did:sage:solana:test",
		Name:        "Test Agent",
		Description: "Test Description",
		Endpoint:    "https://api.example.com",
		PublicKey:   [32]byte{1, 2, 3, 4, 5},
		Capabilities: map[string]interface{}{
			"chat": true,
			"code": true,
		},
		IsActive:  true,
		CreatedAt: 1234567890,
		UpdatedAt: 1234567890,
	}

	assert.Equal(t, "did:sage:solana:test", account.DID)
	assert.Equal(t, "Test Agent", account.Name)
	assert.Equal(t, "Test Description", account.Description)
	assert.True(t, account.IsActive)
	assert.NotNil(t, account.Capabilities)
	assert.Equal(t, int64(1234567890), account.CreatedAt)
	assert.Equal(t, int64(1234567890), account.UpdatedAt)
}

// Helper types

type mockKeyPairForType struct {
	keyType sagecrypto.KeyType
}

func (m *mockKeyPairForType) Type() sagecrypto.KeyType {
	return m.keyType
}

func (m *mockKeyPairForType) PublicKey() crypto.PublicKey {
	// Return a mock Ed25519 public key for Solana
	if m.keyType == sagecrypto.KeyTypeEd25519 {
		_, pub, _ := ed25519.GenerateKey(rand.Reader)
		return pub
	}
	// For other types, return a basic ed25519 public key anyway
	_, pub, _ := ed25519.GenerateKey(rand.Reader)
	return pub
}

func (m *mockKeyPairForType) PrivateKey() crypto.PrivateKey {
	priv, _, _ := ed25519.GenerateKey(rand.Reader)
	return priv
}

func (m *mockKeyPairForType) Sign(data []byte) ([]byte, error) {
	return make([]byte, ed25519.SignatureSize), nil
}

func (m *mockKeyPairForType) Verify(message, signature []byte) error {
	return nil
}

func (m *mockKeyPairForType) ID() string {
	return "mock-key-id"
}
