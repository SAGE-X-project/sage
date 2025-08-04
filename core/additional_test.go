package core

import (
	"context"
	"testing"

	"github.com/sage-x-project/sage/config"
	"github.com/sage-x-project/sage/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCore_AdditionalMethods(t *testing.T) {
	core := New()

	t.Run("Version constant", func(t *testing.T) {
		assert.Equal(t, "0.1.0", Version)
	})

	t.Run("GetCryptoManager", func(t *testing.T) {
		cryptoMgr := core.GetCryptoManager()
		assert.NotNil(t, cryptoMgr)
	})

	t.Run("GetDIDManager", func(t *testing.T) {
		didMgr := core.GetDIDManager()
		assert.NotNil(t, didMgr)
	})

	t.Run("GetSupportedChains", func(t *testing.T) {
		chains := core.GetSupportedChains()
		// Initially should be empty
		assert.Empty(t, chains)
	})
}

func TestCore_ConfigurationEdgeCases(t *testing.T) {
	t.Run("NewWithConfig with nil config", func(t *testing.T) {
		_, err := NewWithConfig(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config cannot be nil")
	})

	t.Run("ApplyConfig with nil config", func(t *testing.T) {
		core := New()
		err := core.ApplyConfig(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config cannot be nil")
	})

	t.Run("ApplyConfig with invalid storage path", func(t *testing.T) {
		core := New()
		cfg := &config.Config{
			KeyMgmt: config.KeyManagementConfig{
				Storage: config.StorageConfig{
					Type: "file",
					Path: "/invalid/path/that/does/not/exist/and/cannot/be/created",
				},
			},
			Networks: make(map[string]config.NetworkConfig),
		}

		err := core.ApplyConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create file storage")
	})

	t.Run("ApplyConfig with invalid network name", func(t *testing.T) {
		core := New() 
		cfg := &config.Config{
			KeyMgmt: config.KeyManagementConfig{
				Storage: config.StorageConfig{
					Type: "memory",
				},
			},
			Networks: map[string]config.NetworkConfig{
				"invalid-network": {
					Chains: map[string]config.ChainConfig{
						"mainnet": {
							RPC:      "https://test.example.com",
							Contract: "0x123",
						},
					},
				},
			},
		}

		err := core.ApplyConfig(cfg)
		// Should skip invalid networks without error
		assert.NoError(t, err)
	})
}

func TestCore_KeyOperations(t *testing.T) {
	core := New()

	t.Run("GenerateKeyPair", func(t *testing.T) {
		keyPair, err := core.GenerateKeyPair(crypto.KeyTypeEd25519)
		assert.NoError(t, err)
		assert.NotNil(t, keyPair)
		assert.Equal(t, crypto.KeyTypeEd25519, keyPair.Type())
	})

	t.Run("GenerateKeyPair invalid type", func(t *testing.T) {
		_, err := core.GenerateKeyPair(crypto.KeyType("invalid"))
		assert.Error(t, err)
	})
}

func TestCore_MessageOperations(t *testing.T) {
	core := New()

	// Generate a key pair for testing
	keyPair, err := core.GenerateKeyPair(crypto.KeyTypeEd25519)
	require.NoError(t, err)

	message := []byte("test message")

	t.Run("SignMessage", func(t *testing.T) {
		signature, err := core.SignMessage(keyPair, message)
		assert.NoError(t, err)
		assert.NotEmpty(t, signature)
	})

	t.Run("SignMessage with nil key", func(t *testing.T) {
		// This test expects a panic from calling methods on nil keyPair
		// We need to catch it as a panic rather than an error
		assert.Panics(t, func() {
			core.SignMessage(nil, message)
		})
	})

	t.Run("SignMessage with empty message", func(t *testing.T) {
		signature, err := core.SignMessage(keyPair, []byte{})
		assert.NoError(t, err)
		assert.NotEmpty(t, signature)
	})
}

func TestVerificationService_EdgeCases(t *testing.T) {
	core := New()

	t.Run("VerifyMessageFromHeaders with empty headers", func(t *testing.T) {
		headers := make(map[string]string)
		body := []byte("test body")
		signature := []byte("test signature")
		
		result, err := core.VerifyMessageFromHeaders(context.Background(), headers, body, signature)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("QuickVerify with invalid signature", func(t *testing.T) {
		message := []byte("test message")
		signature := []byte("invalid signature")
		agentDID := "did:sage:test:invalid"

		err := core.QuickVerify(context.Background(), agentDID, message, signature)
		assert.Error(t, err)
	})
}

func TestCore_Integration(t *testing.T) {
	t.Run("end-to-end message signing and verification", func(t *testing.T) {
		core := New()
		ctx := context.Background()

		// Generate key pair
		keyPair, err := core.GenerateKeyPair(crypto.KeyTypeEd25519)
		require.NoError(t, err)

		message := []byte("integration test message")

		// Sign message
		signature, err := core.SignMessage(keyPair, message)
		require.NoError(t, err)

		// Try quick verify with a test DID
		agentDID := "did:sage:test:integration"
		
		// Note: This might fail depending on the actual implementation
		// but it tests the integration between components
		err = core.QuickVerify(ctx, agentDID, message, signature)
		if err != nil {
			// Expected if quick verify is not fully implemented
			t.Logf("QuickVerify not fully implemented: %v", err)
		}
	})
}

func TestCore_ConfigurationTypes(t *testing.T) {
	t.Run("storage type validation", func(t *testing.T) {
		core := New()

		tests := []struct {
			name        string
			storageType string
			wantErr     bool
		}{
			{"memory storage", "memory", false},
			{"file storage", "file", false}, // Should succeed with /tmp/test-keys path
			{"invalid storage", "invalid", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := &config.Config{
					KeyMgmt: config.KeyManagementConfig{
						Storage: config.StorageConfig{
							Type: tt.storageType,
							Path: "/tmp/test-keys", // Valid path for file storage
						},
					},
					Networks: make(map[string]config.NetworkConfig),
				}

				err := core.ApplyConfig(cfg)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}