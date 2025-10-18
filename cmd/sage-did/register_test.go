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

package main

import (
	"strings"
	"testing"

	_ "github.com/sage-x-project/sage/internal/cryptoinit"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

func TestGenerateAgentDIDWithAddress(t *testing.T) {
	tests := []struct {
		name          string
		chain         did.Chain
		ownerAddress  string
		expectedDID   string
		expectContain string
	}{
		{
			name:          "Ethereum with 0x prefix",
			chain:         did.ChainEthereum,
			ownerAddress:  "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
			expectedDID:   "did:sage:ethereum:0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
			expectContain: "0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
		},
		{
			name:          "Ethereum without 0x prefix",
			chain:         did.ChainEthereum,
			ownerAddress:  "742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
			expectedDID:   "did:sage:ethereum:0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
			expectContain: "0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
		},
		{
			name:          "Solana chain",
			chain:         did.ChainSolana,
			ownerAddress:  "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
			expectedDID:   "did:sage:solana:0x742d35cc6634c0532925a3b844bc9e7595f0beb0",
			expectContain: "solana",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateAgentDIDWithAddress(tt.chain, tt.ownerAddress)

			if string(result) != tt.expectedDID {
				t.Errorf("generateAgentDIDWithAddress() = %v, want %v", result, tt.expectedDID)
			}

			if !strings.Contains(string(result), tt.expectContain) {
				t.Errorf("DID should contain %s, got %v", tt.expectContain, result)
			}

			// Verify lowercase
			if strings.ToLower(string(result)) != string(result) {
				t.Errorf("DID should be lowercase, got %v", result)
			}
		})
	}
}

func TestGenerateAgentDIDWithNonce(t *testing.T) {
	tests := []struct {
		name         string
		chain        did.Chain
		ownerAddress string
		nonce        uint64
		expectedDID  string
	}{
		{
			name:         "Ethereum with nonce 0",
			chain:        did.ChainEthereum,
			ownerAddress: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
			nonce:        0,
			expectedDID:  "did:sage:ethereum:0x742d35cc6634c0532925a3b844bc9e7595f0beb0:0",
		},
		{
			name:         "Ethereum with nonce 1",
			chain:        did.ChainEthereum,
			ownerAddress: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
			nonce:        1,
			expectedDID:  "did:sage:ethereum:0x742d35cc6634c0532925a3b844bc9e7595f0beb0:1",
		},
		{
			name:         "Ethereum with large nonce",
			chain:        did.ChainEthereum,
			ownerAddress: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
			nonce:        999999,
			expectedDID:  "did:sage:ethereum:0x742d35cc6634c0532925a3b844bc9e7595f0beb0:999999",
		},
		{
			name:         "Without 0x prefix",
			chain:        did.ChainEthereum,
			ownerAddress: "742d35Cc6634C0532925a3b844Bc9e7595f0bEb0",
			nonce:        5,
			expectedDID:  "did:sage:ethereum:0x742d35cc6634c0532925a3b844bc9e7595f0beb0:5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateAgentDIDWithNonce(tt.chain, tt.ownerAddress, tt.nonce)

			if string(result) != tt.expectedDID {
				t.Errorf("generateAgentDIDWithNonce() = %v, want %v", result, tt.expectedDID)
			}

			// Verify format
			parts := strings.Split(string(result), ":")
			if len(parts) != 5 {
				t.Errorf("DID should have 5 parts separated by ':', got %d parts", len(parts))
			}

			// Verify lowercase address
			if strings.ToLower(parts[3]) != parts[3] {
				t.Errorf("Address should be lowercase, got %v", parts[3])
			}
		})
	}
}

func TestDeriveEthereumAddress(t *testing.T) {
	t.Run("Should derive address from secp256k1 keypair", func(t *testing.T) {
		// Generate a secp256k1 keypair
		keyPair, err := crypto.GenerateSecp256k1KeyPair()
		if err != nil {
			t.Fatalf("Failed to generate keypair: %v", err)
		}

		// Derive address
		address, err := deriveEthereumAddress(keyPair)
		if err != nil {
			t.Fatalf("Failed to derive address: %v", err)
		}

		// Verify format
		if !strings.HasPrefix(address, "0x") {
			t.Errorf("Address should start with 0x, got %s", address)
		}

		// Verify length (0x + 40 hex chars = 42 chars)
		if len(address) != 42 {
			t.Errorf("Address should be 42 chars long, got %d", len(address))
		}

		// Verify lowercase
		if strings.ToLower(address) != address {
			t.Errorf("Address should be lowercase, got %s", address)
		}

		// Verify all hex chars
		addressWithoutPrefix := strings.TrimPrefix(address, "0x")
		for i, c := range addressWithoutPrefix {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("Address contains non-hex character at position %d: %c", i, c)
			}
		}
	})

	t.Run("Should fail for non-secp256k1 keypair", func(t *testing.T) {
		// Generate an Ed25519 keypair
		keyPair, err := crypto.GenerateEd25519KeyPair()
		if err != nil {
			t.Fatalf("Failed to generate Ed25519 keypair: %v", err)
		}

		// Try to derive address (should fail)
		_, err = deriveEthereumAddress(keyPair)
		if err == nil {
			t.Error("Expected error for Ed25519 keypair, got nil")
		}

		if !strings.Contains(err.Error(), "secp256k1") {
			t.Errorf("Error should mention secp256k1, got: %v", err)
		}
	})

	t.Run("Should generate consistent address for same keypair", func(t *testing.T) {
		// Generate a keypair
		keyPair, err := crypto.GenerateSecp256k1KeyPair()
		if err != nil {
			t.Fatalf("Failed to generate keypair: %v", err)
		}

		// Derive address twice
		address1, err := deriveEthereumAddress(keyPair)
		if err != nil {
			t.Fatalf("Failed to derive address (first): %v", err)
		}

		address2, err := deriveEthereumAddress(keyPair)
		if err != nil {
			t.Fatalf("Failed to derive address (second): %v", err)
		}

		// Should be identical
		if address1 != address2 {
			t.Errorf("Address derivation should be deterministic, got %s and %s", address1, address2)
		}
	})
}

func TestDIDGenerationIntegration(t *testing.T) {
	t.Run("Complete DID generation flow with derived address", func(t *testing.T) {
		// Generate keypair
		keyPair, err := crypto.GenerateSecp256k1KeyPair()
		if err != nil {
			t.Fatalf("Failed to generate keypair: %v", err)
		}

		// Derive address
		address, err := deriveEthereumAddress(keyPair)
		if err != nil {
			t.Fatalf("Failed to derive address: %v", err)
		}

		// Generate DID with address
		didWithAddr := generateAgentDIDWithAddress(did.ChainEthereum, address)

		// Verify DID contains the address
		if !strings.Contains(string(didWithAddr), address) {
			t.Errorf("DID should contain address %s, got %s", address, didWithAddr)
		}

		// Generate DID with nonce
		didWithNonce := generateAgentDIDWithNonce(did.ChainEthereum, address, 1)

		// Verify DID format
		expectedPrefix := "did:sage:ethereum:" + address + ":"
		if !strings.HasPrefix(string(didWithNonce), expectedPrefix) {
			t.Errorf("DID should start with %s, got %s", expectedPrefix, didWithNonce)
		}

		t.Logf("Generated DID with address: %s", didWithAddr)
		t.Logf("Generated DID with nonce: %s", didWithNonce)
	})
}
