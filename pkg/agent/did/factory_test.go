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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
)

func TestRegisterClientCreator(t *testing.T) {
	prev := ClientCreatorFor(ChainEthereum)
	t.Cleanup(func() { RegisterClientCreator(ChainEthereum, prev) })

	called := false
	RegisterClientCreator(ChainEthereum, func(config *RegistryConfig) (ChainClient, error) {
		called = true
		return &fakeChainClient{meta: &AgentMetadata{DID: "did:sage:ethereum:0x1"}}, nil
	})
	creator := ClientCreatorFor(ChainEthereum)
	require.NotNil(t, creator)
	client, err := creator(&RegistryConfig{Chain: ChainEthereum})
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.True(t, called)

	RegisterClientCreator(ChainEthereum, nil)
	assert.Nil(t, ClientCreatorFor(ChainEthereum))
	assert.Nil(t, ClientCreatorFor("bitcoin"))
}

func TestGetRecommendedKeyType(t *testing.T) {
	tests := []struct {
		chainType Chain
		want      sagecrypto.KeyType
		wantErr   bool
	}{
		{ChainEthereum, sagecrypto.KeyTypeSecp256k1, false},
		{ChainSolana, sagecrypto.KeyTypeEd25519, false},
		{"bitcoin", "", true},
	}
	for _, tt := range tests {
		got, err := GetRecommendedKeyType(tt.chainType)
		if tt.wantErr {
			assert.Error(t, err, string(tt.chainType))
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, tt.want, got)
	}
}

func TestValidateKeyTypeForChain(t *testing.T) {
	assert.NoError(t, ValidateKeyTypeForChain(sagecrypto.KeyTypeSecp256k1, ChainEthereum))
	assert.NoError(t, ValidateKeyTypeForChain(sagecrypto.KeyTypeEd25519, ChainSolana))
	assert.Error(t, ValidateKeyTypeForChain(sagecrypto.KeyTypeSecp256k1, ChainSolana))
	assert.Error(t, ValidateKeyTypeForChain(sagecrypto.KeyTypeEd25519, "bitcoin"))
}

func TestGetRFC9421Algorithm(t *testing.T) {
	alg, err := GetRFC9421Algorithm(sagecrypto.KeyTypeEd25519)
	require.NoError(t, err)
	assert.Equal(t, "ed25519", alg)
	alg, err = GetRFC9421Algorithm(sagecrypto.KeyTypeSecp256k1)
	require.NoError(t, err)
	assert.Equal(t, "es256k", alg)
	_, err = GetRFC9421Algorithm("rsa")
	assert.Error(t, err)
}
