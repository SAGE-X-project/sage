// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


package solana

import (
	"testing"

	"github.com/stretchr/testify/assert"
	
	"github.com/sage-x-project/sage/did"
)

func TestNewSolanaClient(t *testing.T) {
	config := &did.RegistryConfig{
		Chain:           did.ChainSolana,
		ContractAddress: "11111111111111111111111111111111",
		RPCEndpoint:     "http://localhost:8899",
		PrivateKey:      "", // No private key for read-only
	}
	
	// This will succeed but won't connect to actual RPC
	client, err := NewSolanaClient(config)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, config.RPCEndpoint, client.config.RPCEndpoint)
	
	// Test with invalid program ID
	invalidConfig := &did.RegistryConfig{
		Chain:           did.ChainSolana,
		ContractAddress: "invalid-address",
		RPCEndpoint:     "http://localhost:8899",
	}
	
	_, err = NewSolanaClient(invalidConfig)
	assert.Error(t, err)
}

func TestSolanaHelperMethods(t *testing.T) {
	client := &SolanaClient{
		config: &did.RegistryConfig{
			MaxRetries: 30,
		},
	}
	
	// Test prepareRegistrationMessage
	req := &did.RegistrationRequest{
		DID:      "did:sage:solana:agent001",
		Name:     "Test Agent",
		Endpoint: "https://api.example.com",
	}
	
	message := client.prepareRegistrationMessage(req, "11111111111111111111111111111111")
	assert.Contains(t, message, "Register agent:")
	assert.Contains(t, message, string(req.DID))
	assert.Contains(t, message, req.Name)
	assert.Contains(t, message, req.Endpoint)
	assert.Contains(t, message, "11111111111111111111111111111111")
	
	// Test prepareUpdateMessage
	agentDID := did.AgentDID("did:sage:solana:agent001")
	updates := map[string]interface{}{
		"name":        "Updated Agent",
		"description": "New description",
	}
	
	updateMessage := client.prepareUpdateMessage(agentDID, updates)
	assert.Contains(t, updateMessage, "Update agent:")
	assert.Contains(t, updateMessage, string(agentDID))
}

func TestSerializeDeserialize(t *testing.T) {
	// Test simple serialization (in production, use borsh)
	data := struct {
		Name  string
		Value int
	}{
		Name:  "test",
		Value: 42,
	}
	
	serialized := serializeInstruction(data)
	assert.NotEmpty(t, serialized)
	
	// Test deserialization
	var result struct {
		Name  string
		Value int
	}
	
	err := deserializeAccount(serialized, &result)
	assert.NoError(t, err)
	assert.Equal(t, data.Name, result.Name)
	assert.Equal(t, data.Value, result.Value)
}

func TestAgentAccount(t *testing.T) {
	account := AgentAccount{
		DID:         "did:sage:solana:agent001",
		Name:        "Test Agent",
		Description: "A test agent",
		Endpoint:    "https://api.example.com",
		PublicKey:   [32]byte{1, 2, 3, 4}, // Sample public key
		Capabilities: map[string]interface{}{
			"chat": true,
			"code": true,
		},
		IsActive:  true,
		CreatedAt: 1234567890,
		UpdatedAt: 1234567890,
	}
	
	assert.Equal(t, "did:sage:solana:agent001", account.DID)
	assert.Equal(t, "Test Agent", account.Name)
	assert.True(t, account.IsActive)
	assert.Equal(t, int64(1234567890), account.CreatedAt)
}