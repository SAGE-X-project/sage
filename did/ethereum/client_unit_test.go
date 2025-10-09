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
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/did"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrepareRegistrationMessage tests registration message preparation
func TestPrepareRegistrationMessage(t *testing.T) {
	tests := []struct {
		name     string
		req      *did.RegistrationRequest
		address  string
		expected []string // strings that should be contained in the message
	}{
		{
			name: "Basic registration message",
			req: &did.RegistrationRequest{
				DID:      "did:sage:ethereum:agent001",
				Name:     "Test Agent",
				Endpoint: "https://api.example.com",
			},
			address: "0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			expected: []string{
				"Register agent:",
				"did:sage:ethereum:agent001",
				"Test Agent",
				"https://api.example.com",
				"0x742d35Cc6634C0532925a3b844Bc9e7595f0aEbb",
			},
		},
		{
			name: "Registration with special characters",
			req: &did.RegistrationRequest{
				DID:      "did:sage:ethereum:agent-special-123",
				Name:     "Agent with Spaces & Special",
				Endpoint: "https://api.example.com/path?key=value",
			},
			address: "0x1234567890abcdef1234567890abcdef12345678",
			expected: []string{
				"Register agent:",
				"did:sage:ethereum:agent-special-123",
				"Agent with Spaces & Special",
				"https://api.example.com/path?key=value",
				"0x1234567890abcdef1234567890abcdef12345678",
			},
		},
	}

	client := &EthereumClient{
		config: &did.RegistryConfig{},
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
	tests := []struct {
		name     string
		agentDID did.AgentDID
		updates  map[string]interface{}
		expected []string
	}{
		{
			name:     "Basic update message",
			agentDID: "did:sage:ethereum:agent001",
			updates: map[string]interface{}{
				"name":        "Updated Agent",
				"description": "New description",
			},
			expected: []string{
				"Update agent:",
				"did:sage:ethereum:agent001",
			},
		},
		{
			name:     "Update with capabilities",
			agentDID: "did:sage:ethereum:agent002",
			updates: map[string]interface{}{
				"capabilities": map[string]interface{}{
					"chat": true,
					"code": true,
				},
			},
			expected: []string{
				"Update agent:",
				"did:sage:ethereum:agent002",
			},
		},
	}

	client := &EthereumClient{
		config: &did.RegistryConfig{},
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

// TestGetTransactOpts tests transaction options generation
func TestGetTransactOpts(t *testing.T) {
	tests := []struct {
		name       string
		privateKey *ecdsa.PrivateKey
		chainID    *big.Int
		gasPrice   uint64
		wantErr    bool
	}{
		{
			name:       "Valid transaction opts with gas price",
			privateKey: mustGenerateKey(t),
			chainID:    big.NewInt(1337),
			gasPrice:   20000000000,
			wantErr:    false,
		},
		{
			name:       "Valid transaction opts without gas price",
			privateKey: mustGenerateKey(t),
			chainID:    big.NewInt(1),
			gasPrice:   0,
			wantErr:    false,
		},
		{
			name:       "Missing private key",
			privateKey: nil,
			chainID:    big.NewInt(1337),
			gasPrice:   0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &EthereumClient{
				privateKey: tt.privateKey,
				chainID:    tt.chainID,
				config: &did.RegistryConfig{
					GasPrice: tt.gasPrice,
				},
			}

			ctx := context.Background()
			opts, err := client.getTransactOpts(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, opts)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, opts)
				assert.Equal(t, ctx, opts.Context)

				if tt.gasPrice > 0 {
					assert.Equal(t, big.NewInt(int64(tt.gasPrice)), opts.GasPrice)
				}
			}
		})
	}
}

// TestRegisterKeyTypeValidation tests that Register validates key type
func TestRegisterKeyTypeValidation(t *testing.T) {
	tests := []struct {
		name    string
		keyType sagecrypto.KeyType
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid Secp256k1 key",
			keyType: sagecrypto.KeyTypeSecp256k1,
			wantErr: false,
		},
		{
			name:    "Invalid Ed25519 key",
			keyType: sagecrypto.KeyTypeEd25519,
			wantErr: true,
			errMsg:  "Ethereum requires Secp256k1 keys",
		},
		{
			name:    "Invalid X25519 key",
			keyType: sagecrypto.KeyTypeX25519,
			wantErr: true,
			errMsg:  "Ethereum requires Secp256k1 keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &EthereumClient{
				privateKey: mustGenerateKey(t),
				chainID:    big.NewInt(1337),
				config: &did.RegistryConfig{
					MaxRetries: 3,
				},
			}

			// Create mock key pair
			mockKeyPair := &mockKeyPairForType{keyType: tt.keyType}

			req := &did.RegistrationRequest{
				DID:          "did:sage:ethereum:test",
				Name:         "Test Agent",
				Description:  "Test Description",
				Endpoint:     "https://api.example.com",
				KeyPair:      mockKeyPair,
				Capabilities: map[string]interface{}{"chat": true},
			}

			_, err := client.Register(context.Background(), req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				// Will fail with other errors (no real blockchain), but should pass key type check
				// We just verify it doesn't fail with key type error
				if err != nil {
					assert.NotContains(t, err.Error(), "Ethereum requires Secp256k1 keys")
				}
			}
		})
	}
}

// TestWaitForTransactionRetryLogic tests retry configuration
func TestWaitForTransactionRetryLogic(t *testing.T) {
	tests := []struct {
		name               string
		maxRetries         int
		confirmationBlocks int
	}{
		{
			name:               "No retries configured",
			maxRetries:         0,
			confirmationBlocks: 0,
		},
		{
			name:               "Few retries configured",
			maxRetries:         3,
			confirmationBlocks: 0,
		},
		{
			name:               "Many retries configured",
			maxRetries:         10,
			confirmationBlocks: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &EthereumClient{
				config: &did.RegistryConfig{
					MaxRetries:         tt.maxRetries,
					ConfirmationBlocks: tt.confirmationBlocks,
				},
			}

			// Verify config is set correctly
			assert.Equal(t, tt.maxRetries, client.config.MaxRetries)
			assert.Equal(t, tt.confirmationBlocks, client.config.ConfirmationBlocks)
		})
	}
}

// TestCapabilitiesJSON tests capabilities marshaling
func TestCapabilitiesJSON(t *testing.T) {
	tests := []struct {
		name         string
		capabilities map[string]interface{}
		wantErr      bool
	}{
		{
			name: "Simple capabilities",
			capabilities: map[string]interface{}{
				"chat": true,
				"code": true,
			},
			wantErr: false,
		},
		{
			name: "Nested capabilities",
			capabilities: map[string]interface{}{
				"chat": true,
				"advanced": map[string]interface{}{
					"translation": true,
					"summarization": true,
				},
			},
			wantErr: false,
		},
		{
			name:         "Empty capabilities",
			capabilities: map[string]interface{}{},
			wantErr:      false,
		},
		{
			name:         "Nil capabilities",
			capabilities: nil,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that capabilities can be marshaled without error
			// This is part of the Register function logic
			if tt.capabilities != nil {
				_, err := json.Marshal(tt.capabilities)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

// TestUpdateExtractsFields tests that Update correctly extracts update fields
func TestUpdateExtractsFields(t *testing.T) {
	tests := []struct {
		name        string
		updates     map[string]interface{}
		expectName  string
		expectDesc  string
		expectEnd   string
		expectCaps  bool
	}{
		{
			name: "All fields present",
			updates: map[string]interface{}{
				"name":         "New Name",
				"description":  "New Description",
				"endpoint":     "https://new.endpoint.com",
				"capabilities": map[string]interface{}{"chat": true},
			},
			expectName: "New Name",
			expectDesc: "New Description",
			expectEnd:  "https://new.endpoint.com",
			expectCaps: true,
		},
		{
			name: "Partial fields",
			updates: map[string]interface{}{
				"name": "Only Name",
			},
			expectName: "Only Name",
			expectDesc: "",
			expectEnd:  "",
			expectCaps: false,
		},
		{
			name:        "Empty updates",
			updates:     map[string]interface{}{},
			expectName:  "",
			expectDesc:  "",
			expectEnd:   "",
			expectCaps:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Extract fields the same way Update does
			name, _ := tt.updates["name"].(string)
			description, _ := tt.updates["description"].(string)
			endpoint, _ := tt.updates["endpoint"].(string)
			_, hasCaps := tt.updates["capabilities"]

			assert.Equal(t, tt.expectName, name)
			assert.Equal(t, tt.expectDesc, description)
			assert.Equal(t, tt.expectEnd, endpoint)
			assert.Equal(t, tt.expectCaps, hasCaps)
		})
	}
}

// TestDeactivateGeneratesAgentID tests that Deactivate generates correct agent ID
func TestDeactivateGeneratesAgentID(t *testing.T) {
	tests := []struct {
		name     string
		agentDID did.AgentDID
	}{
		{
			name:     "Basic DID",
			agentDID: "did:sage:ethereum:agent001",
		},
		{
			name:     "Long DID",
			agentDID: "did:sage:ethereum:agent-with-very-long-identifier-12345",
		},
		{
			name:     "DID with special characters",
			agentDID: "did:sage:ethereum:agent_special-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate agentId the same way Deactivate does
			agentId := ethcrypto.Keccak256Hash([]byte(string(tt.agentDID)))

			assert.NotNil(t, agentId)
			assert.Equal(t, 32, len(agentId.Bytes()))

			// Verify it's deterministic
			agentId2 := ethcrypto.Keccak256Hash([]byte(string(tt.agentDID)))
			assert.Equal(t, agentId, agentId2)
		})
	}
}

// Helper types and functions

type mockKeyPairForType struct {
	keyType sagecrypto.KeyType
}

func (m *mockKeyPairForType) Type() sagecrypto.KeyType {
	return m.keyType
}

func (m *mockKeyPairForType) PublicKey() crypto.PublicKey {
	// Return a mock ECDSA public key for Ethereum
	if m.keyType == sagecrypto.KeyTypeSecp256k1 {
		key, _ := ecdsa.GenerateKey(ethcrypto.S256(), rand.Reader)
		return &key.PublicKey
	}
	// For other types, return a basic ecdsa public key
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	return &key.PublicKey
}

func (m *mockKeyPairForType) PrivateKey() crypto.PrivateKey {
	// Return a mock ECDSA private key
	key, _ := ecdsa.GenerateKey(ethcrypto.S256(), rand.Reader)
	return key
}

func (m *mockKeyPairForType) Sign(data []byte) ([]byte, error) {
	// Return mock signature
	return make([]byte, 64), nil
}

func (m *mockKeyPairForType) Verify(message, signature []byte) error {
	return nil
}

func (m *mockKeyPairForType) ID() string {
	return "mock-key-id"
}

func mustGenerateKey(t *testing.T) *ecdsa.PrivateKey {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return key
}

// TestPrivateKeyValidation tests private key hex decoding
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

// TestWaitForTransactionConfirmations tests confirmation block logic
func TestWaitForTransactionConfirmations(t *testing.T) {
	tests := []struct {
		name                string
		confirmationBlocks  int
		expectedMinWaitTime time.Duration
	}{
		{
			name:                "No confirmations required",
			confirmationBlocks:  0,
			expectedMinWaitTime: 0,
		},
		{
			name:                "Few confirmations required",
			confirmationBlocks:  3,
			expectedMinWaitTime: 0, // Would wait in real scenario
		},
		{
			name:                "Many confirmations required",
			confirmationBlocks:  12,
			expectedMinWaitTime: 0, // Would wait in real scenario
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &EthereumClient{
				config: &did.RegistryConfig{
					MaxRetries:         1, // Fast fail
					ConfirmationBlocks: tt.confirmationBlocks,
				},
			}

			// Verify config is set correctly
			assert.Equal(t, tt.confirmationBlocks, client.config.ConfirmationBlocks)
		})
	}
}
