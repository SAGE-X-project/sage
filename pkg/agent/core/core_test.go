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


package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

func TestCore(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		core := New()
		assert.NotNil(t, core)
		assert.NotNil(t, core.cryptoManager)
		assert.NotNil(t, core.didManager)
		assert.NotNil(t, core.verificationService)
	})
	
	t.Run("GenerateKeyPair", func(t *testing.T) {
		core := New()
		
		// Test Ed25519
		ed25519Key, err := core.GenerateKeyPair(crypto.KeyTypeEd25519)
		require.NoError(t, err)
		assert.NotNil(t, ed25519Key)
		assert.Equal(t, crypto.KeyTypeEd25519, ed25519Key.Type())
		
		// Test Secp256k1
		secp256k1Key, err := core.GenerateKeyPair(crypto.KeyTypeSecp256k1)
		require.NoError(t, err)
		assert.NotNil(t, secp256k1Key)
		assert.Equal(t, crypto.KeyTypeSecp256k1, secp256k1Key.Type())
		
		// Test unsupported type
		_, err = core.GenerateKeyPair(crypto.KeyType("unsupported"))
		assert.Error(t, err)
	})
	
	t.Run("SignMessage", func(t *testing.T) {
		core := New()
		
		keyPair, err := core.GenerateKeyPair(crypto.KeyTypeEd25519)
		require.NoError(t, err)
		
		message := []byte("test message")
		signature, err := core.SignMessage(keyPair, message)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)
		
		// Verify the signature
		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)
	})
	
	t.Run("CreateRFC9421Message", func(t *testing.T) {
		core := New()
		
		builder := core.CreateRFC9421Message("did:sage:ethereum:agent001", []byte("test body"))
		assert.NotNil(t, builder)
		
		message := builder.Build()
		assert.Equal(t, "did:sage:ethereum:agent001", message.AgentDID)
		assert.Equal(t, []byte("test body"), message.Body)
	})
	
	t.Run("ConfigureDID", func(t *testing.T) {
		core := New()
		
		config := &did.RegistryConfig{
			Chain:           did.ChainEthereum,
			ContractAddress: "0x1234567890abcdef",
			RPCEndpoint:     "http://localhost:8545",
		}
		
		// This should succeed now as we only store configuration
		err := core.ConfigureDID(did.ChainEthereum, config)
		assert.NoError(t, err)
	})
	
	t.Run("GetManagers", func(t *testing.T) {
		core := New()
		
		assert.NotNil(t, core.GetCryptoManager())
		assert.NotNil(t, core.GetDIDManager())
		assert.NotNil(t, core.GetVerificationService())
	})
	
	t.Run("GetSupportedChains", func(t *testing.T) {
		core := New()
		
		chains := core.GetSupportedChains()
		assert.NotNil(t, chains)
		assert.Empty(t, chains) // No chains configured yet
	})
}

func TestVersion(t *testing.T) {
	assert.Equal(t, "0.1.0", Version)
}
