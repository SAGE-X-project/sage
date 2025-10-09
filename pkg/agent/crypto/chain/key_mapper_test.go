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


package chain

import (
	"testing"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChainKeyTypeMapper(t *testing.T) {
	mapper := NewChainKeyTypeMapper()
	assert.NotNil(t, mapper)
}

func TestGetRecommendedKeyType(t *testing.T) {
	mapper := NewChainKeyTypeMapper()

	tests := []struct {
		name      string
		chainType ChainType
		expected  sagecrypto.KeyType
		wantErr   bool
	}{
		{
			name:      "Ethereum uses Secp256k1",
			chainType: ChainTypeEthereum,
			expected:  sagecrypto.KeyTypeSecp256k1,
			wantErr:   false,
		},
		{
			name:      "Solana uses Ed25519",
			chainType: ChainTypeSolana,
			expected:  sagecrypto.KeyTypeEd25519,
			wantErr:   false,
		},
		{
			name:      "Bitcoin uses Secp256k1",
			chainType: ChainTypeBitcoin,
			expected:  sagecrypto.KeyTypeSecp256k1,
			wantErr:   false,
		},
		{
			name:      "Cosmos uses Secp256k1",
			chainType: ChainTypeCosmos,
			expected:  sagecrypto.KeyTypeSecp256k1,
			wantErr:   false,
		},
		{
			name:      "Unsupported chain returns error",
			chainType: ChainType("unknown"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyType, err := mapper.GetRecommendedKeyType(tt.chainType)

			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrChainNotSupported)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, keyType)
			}
		})
	}
}

func TestGetSupportedKeyTypes(t *testing.T) {
	mapper := NewChainKeyTypeMapper()

	tests := []struct {
		name      string
		chainType ChainType
		expected  []sagecrypto.KeyType
		wantErr   bool
	}{
		{
			name:      "Ethereum supports Secp256k1",
			chainType: ChainTypeEthereum,
			expected:  []sagecrypto.KeyType{sagecrypto.KeyTypeSecp256k1},
			wantErr:   false,
		},
		{
			name:      "Solana supports Ed25519",
			chainType: ChainTypeSolana,
			expected:  []sagecrypto.KeyType{sagecrypto.KeyTypeEd25519},
			wantErr:   false,
		},
		{
			name:      "Bitcoin supports Secp256k1",
			chainType: ChainTypeBitcoin,
			expected:  []sagecrypto.KeyType{sagecrypto.KeyTypeSecp256k1},
			wantErr:   false,
		},
		{
			name:      "Cosmos supports multiple key types",
			chainType: ChainTypeCosmos,
			expected:  []sagecrypto.KeyType{sagecrypto.KeyTypeSecp256k1, sagecrypto.KeyTypeEd25519},
			wantErr:   false,
		},
		{
			name:      "Unsupported chain returns error",
			chainType: ChainType("unknown"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyTypes, err := mapper.GetSupportedKeyTypes(tt.chainType)

			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrChainNotSupported)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, keyTypes)
			}
		})
	}
}

func TestGetSupportedKeyTypes_ImmutableResult(t *testing.T) {
	mapper := NewChainKeyTypeMapper()

	// Get the supported key types
	keyTypes1, err := mapper.GetSupportedKeyTypes(ChainTypeEthereum)
	require.NoError(t, err)

	// Modify the returned slice
	originalLen := len(keyTypes1)
	keyTypes1 = append(keyTypes1, sagecrypto.KeyTypeRSA)

	// Get the supported key types again
	keyTypes2, err := mapper.GetSupportedKeyTypes(ChainTypeEthereum)
	require.NoError(t, err)

	// Verify the original wasn't modified
	assert.Equal(t, originalLen, len(keyTypes2), "returned slice should be immutable")
}

func TestValidateKeyTypeForChain(t *testing.T) {
	mapper := NewChainKeyTypeMapper()

	tests := []struct {
		name      string
		keyType   sagecrypto.KeyType
		chainType ChainType
		wantErr   bool
	}{
		{
			name:      "Secp256k1 is valid for Ethereum",
			keyType:   sagecrypto.KeyTypeSecp256k1,
			chainType: ChainTypeEthereum,
			wantErr:   false,
		},
		{
			name:      "Ed25519 is valid for Solana",
			keyType:   sagecrypto.KeyTypeEd25519,
			chainType: ChainTypeSolana,
			wantErr:   false,
		},
		{
			name:      "Ed25519 is invalid for Ethereum",
			keyType:   sagecrypto.KeyTypeEd25519,
			chainType: ChainTypeEthereum,
			wantErr:   true,
		},
		{
			name:      "Secp256k1 is invalid for Solana",
			keyType:   sagecrypto.KeyTypeSecp256k1,
			chainType: ChainTypeSolana,
			wantErr:   true,
		},
		{
			name:      "Secp256k1 is valid for Bitcoin",
			keyType:   sagecrypto.KeyTypeSecp256k1,
			chainType: ChainTypeBitcoin,
			wantErr:   false,
		},
		{
			name:      "Both Secp256k1 and Ed25519 valid for Cosmos",
			keyType:   sagecrypto.KeyTypeSecp256k1,
			chainType: ChainTypeCosmos,
			wantErr:   false,
		},
		{
			name:      "Ed25519 also valid for Cosmos",
			keyType:   sagecrypto.KeyTypeEd25519,
			chainType: ChainTypeCosmos,
			wantErr:   false,
		},
		{
			name:      "RSA is invalid for Ethereum",
			keyType:   sagecrypto.KeyTypeRSA,
			chainType: ChainTypeEthereum,
			wantErr:   true,
		},
		{
			name:      "Unknown chain returns error",
			keyType:   sagecrypto.KeyTypeSecp256k1,
			chainType: ChainType("unknown"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapper.ValidateKeyTypeForChain(tt.keyType, tt.chainType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetRFC9421Algorithm(t *testing.T) {
	mapper := NewChainKeyTypeMapper()

	tests := []struct {
		name      string
		keyType   sagecrypto.KeyType
		expected  string
		wantErr   bool
	}{
		{
			name:     "Secp256k1 maps to es256k",
			keyType:  sagecrypto.KeyTypeSecp256k1,
			expected: "es256k",
			wantErr:  false,
		},
		{
			name:     "Ed25519 maps to ed25519",
			keyType:  sagecrypto.KeyTypeEd25519,
			expected: "ed25519",
			wantErr:  false,
		},
		{
			name:     "RSA maps to rsa-pss-sha256",
			keyType:  sagecrypto.KeyTypeRSA,
			expected: "rsa-pss-sha256",
			wantErr:  false,
		},
		{
			name:     "X25519 has no RFC 9421 mapping",
			keyType:  sagecrypto.KeyTypeX25519,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			algorithm, err := mapper.GetRFC9421Algorithm(tt.keyType)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, algorithm)
			}
		})
	}
}

func TestConvenienceFunctions(t *testing.T) {
	t.Run("GetRecommendedKeyType convenience function", func(t *testing.T) {
		keyType, err := GetRecommendedKeyType(ChainTypeEthereum)
		require.NoError(t, err)
		assert.Equal(t, sagecrypto.KeyTypeSecp256k1, keyType)
	})

	t.Run("GetSupportedKeyTypes convenience function", func(t *testing.T) {
		keyTypes, err := GetSupportedKeyTypes(ChainTypeSolana)
		require.NoError(t, err)
		assert.Equal(t, []sagecrypto.KeyType{sagecrypto.KeyTypeEd25519}, keyTypes)
	})

	t.Run("ValidateKeyTypeForChain convenience function", func(t *testing.T) {
		err := ValidateKeyTypeForChain(sagecrypto.KeyTypeSecp256k1, ChainTypeEthereum)
		assert.NoError(t, err)
	})

	t.Run("GetRFC9421Algorithm convenience function", func(t *testing.T) {
		algorithm, err := GetRFC9421Algorithm(sagecrypto.KeyTypeEd25519)
		require.NoError(t, err)
		assert.Equal(t, "ed25519", algorithm)
	})
}

func TestIntegrationScenario(t *testing.T) {
	mapper := NewChainKeyTypeMapper()

	t.Run("Complete workflow: Ethereum agent with correct key type", func(t *testing.T) {
		// Step 1: Get recommended key type for Ethereum
		keyType, err := mapper.GetRecommendedKeyType(ChainTypeEthereum)
		require.NoError(t, err)
		assert.Equal(t, sagecrypto.KeyTypeSecp256k1, keyType)

		// Step 2: Validate the key type is supported
		err = mapper.ValidateKeyTypeForChain(keyType, ChainTypeEthereum)
		assert.NoError(t, err)

		// Step 3: Get RFC 9421 algorithm for message signing
		algorithm, err := mapper.GetRFC9421Algorithm(keyType)
		require.NoError(t, err)
		assert.Equal(t, "es256k", algorithm)
	})

	t.Run("Complete workflow: Solana agent with correct key type", func(t *testing.T) {
		// Step 1: Get recommended key type for Solana
		keyType, err := mapper.GetRecommendedKeyType(ChainTypeSolana)
		require.NoError(t, err)
		assert.Equal(t, sagecrypto.KeyTypeEd25519, keyType)

		// Step 2: Validate the key type is supported
		err = mapper.ValidateKeyTypeForChain(keyType, ChainTypeSolana)
		assert.NoError(t, err)

		// Step 3: Get RFC 9421 algorithm for message signing
		algorithm, err := mapper.GetRFC9421Algorithm(keyType)
		require.NoError(t, err)
		assert.Equal(t, "ed25519", algorithm)
	})

	t.Run("Error scenario: Wrong key type for chain", func(t *testing.T) {
		// Try to use Ed25519 for Ethereum (should fail)
		err := mapper.ValidateKeyTypeForChain(sagecrypto.KeyTypeEd25519, ChainTypeEthereum)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
	})

	t.Run("Multi-chain support: Cosmos accepts multiple key types", func(t *testing.T) {
		supportedTypes, err := mapper.GetSupportedKeyTypes(ChainTypeCosmos)
		require.NoError(t, err)
		assert.Contains(t, supportedTypes, sagecrypto.KeyTypeSecp256k1)
		assert.Contains(t, supportedTypes, sagecrypto.KeyTypeEd25519)

		// Both should validate successfully
		err = mapper.ValidateKeyTypeForChain(sagecrypto.KeyTypeSecp256k1, ChainTypeCosmos)
		assert.NoError(t, err)

		err = mapper.ValidateKeyTypeForChain(sagecrypto.KeyTypeEd25519, ChainTypeCosmos)
		assert.NoError(t, err)
	})
}
