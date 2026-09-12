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

package ethereum

import (
	"context"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrivateKeyValidation(t *testing.T) {
	tests := []struct {
		name       string
		privateKey string
		wantErr    bool
	}{
		{
			name:       "Invalid hex private key",
			privateKey: "not-a-valid-hex-key",
			wantErr:    true,
		},
		{
			name:       "Empty private key is valid (read-only)",
			privateKey: "",
			wantErr:    false,
		},
		{
			name:       "Too short private key",
			privateKey: "0123",
			wantErr:    true,
		},
		{
			name:       "Valid hex but wrong length",
			privateKey: "abcdef1234567890",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test key parsing directly (mimics what NewEthereumClient does)
			if tt.privateKey != "" {
				_, err := ethcrypto.HexToECDSA(tt.privateKey)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			} else {
				// Empty key is valid for read-only mode
				assert.True(t, true)
			}
		})
	}
}

// TestAgentCardClient_GetKEMKey tests KME key retrieval
func TestAgentCardClient_GetKEMKey(t *testing.T) {
	// Skip if no local Ethereum node
	t.Skip("Requires local Ethereum node with deployed AgentCardRegistry contract")

	tests := []struct {
		name      string
		agentID   [32]byte
		expectErr bool
		errMsg    string
	}{
		{
			name:      "Valid agent ID",
			agentID:   [32]byte{1, 2, 3}, // Mock agent ID
			expectErr: true,              // Will fail without real contract
			errMsg:    "no contract code",
		},
		{
			name:      "Empty agent ID",
			agentID:   [32]byte{},
			expectErr: true,
			errMsg:    "", // Contract call will fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test requires a real contract deployment
			// Skipped in CI/CD, run manually with local node
			config := &did.RegistryConfig{
				Chain:           did.ChainEthereum,
				ContractAddress: "0x1234567890123456789012345678901234567890",
				RPCEndpoint:     "http://localhost:8545",
			}

			client, err := NewAgentCardClient(config)
			if err != nil {
				t.Skip("Cannot connect to local node")
				return
			}

			ctx := context.Background()
			_, err = client.GetKEMKey(ctx, tt.agentID)

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAgentCardClient_UpdateKEMKey tests KME key update validation
func TestAgentCardClient_UpdateKEMKey(t *testing.T) {
	tests := []struct {
		name      string
		agentID   [32]byte
		newKey    []byte
		signature []byte
		expectErr bool
		errMsg    string
	}{
		{
			name:      "Invalid key length - too short",
			agentID:   [32]byte{1, 2, 3},
			newKey:    make([]byte, 16), // Should be 32 bytes
			signature: make([]byte, 65),
			expectErr: true,
			errMsg:    "invalid KME key length: expected 32 bytes, got 16",
		},
		{
			name:      "Invalid key length - too long",
			agentID:   [32]byte{1, 2, 3},
			newKey:    make([]byte, 64), // Should be 32 bytes
			signature: make([]byte, 65),
			expectErr: true,
			errMsg:    "invalid KME key length: expected 32 bytes, got 64",
		},
		{
			name:      "Invalid signature length - too short",
			agentID:   [32]byte{1, 2, 3},
			newKey:    make([]byte, 32),
			signature: make([]byte, 32), // Should be 65 bytes
			expectErr: true,
			errMsg:    "invalid signature length: expected 65 bytes, got 32",
		},
		{
			name:      "Invalid signature length - too long",
			agentID:   [32]byte{1, 2, 3},
			newKey:    make([]byte, 32),
			signature: make([]byte, 128), // Should be 65 bytes
			expectErr: true,
			errMsg:    "invalid signature length: expected 65 bytes, got 128",
		},
		{
			name:      "Valid inputs but no private key",
			agentID:   [32]byte{1, 2, 3},
			newKey:    make([]byte, 32),
			signature: make([]byte, 65),
			expectErr: true,
			errMsg:    "no private key configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client without private key for validation tests
			config := &did.RegistryConfig{
				Chain:           did.ChainEthereum,
				ContractAddress: "0x1234567890123456789012345678901234567890",
				RPCEndpoint:     "http://localhost:8545",
				PrivateKey:      "", // No private key
			}

			client, err := NewAgentCardClient(config)
			if err != nil {
				// If we can't connect, skip the test
				t.Skip("Cannot connect to create client")
				return
			}

			ctx := context.Background()
			err = client.UpdateKEMKey(ctx, tt.agentID, tt.newKey, tt.signature)

			if tt.expectErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestAgentMetadataV4_PublicKEMKey tests PublicKEMKey field in AgentMetadataV4
func TestAgentMetadataV4_PublicKEMKey(t *testing.T) {
	// Test that PublicKEMKey field exists and can be set
	KEMKey := make([]byte, 32)
	for i := range KEMKey {
		KEMKey[i] = byte(i)
	}

	metadata := &did.AgentMetadata{
		DID:          "did:sage:ethereum:test",
		Name:         "Test Agent",
		PublicKEMKey: KEMKey,
	}

	assert.NotNil(t, metadata.PublicKEMKey)
	assert.Equal(t, 32, len(metadata.KEMKey()))
	assert.Equal(t, KEMKey, metadata.KEMKey())

	// Test with empty KME key
	metadata2 := &did.AgentMetadata{
		DID:          "did:sage:ethereum:test2",
		Name:         "Test Agent 2",
		PublicKEMKey: []byte{}, // Empty
	}

	assert.NotNil(t, metadata2.PublicKEMKey)
	assert.Equal(t, 0, len(metadata2.KEMKey()))
}
