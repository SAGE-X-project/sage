package core

import (
	"context"
	"testing"

	"github.com/sage-x-project/sage/config"
	"github.com/sage-x-project/sage/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWithConfig(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Version: "1.0",
		Networks: map[string]config.NetworkConfig{
			"ethereum": {
				Chains: map[string]config.ChainConfig{
					"mainnet": {
						RPC:      "https://eth-test.example.com",
						Contract: "0x1234567890123456789012345678901234567890",
						ChainID:  1,
					},
				},
			},
		},
		KeyMgmt: config.KeyManagementConfig{
			DefaultAlgorithm: "ed25519",
			Storage: config.StorageConfig{
				Type: "memory",
			},
		},
	}

	// Create core with config
	core, err := NewWithConfig(cfg)
	require.NoError(t, err)
	assert.NotNil(t, core)
	assert.Equal(t, cfg, core.config)
}

func TestApplyConfig(t *testing.T) {
	core := New()

	t.Run("memory storage", func(t *testing.T) {
		cfg := &config.Config{
			KeyMgmt: config.KeyManagementConfig{
				Storage: config.StorageConfig{
					Type: "memory",
				},
			},
			Networks: make(map[string]config.NetworkConfig),
		}

		err := core.ApplyConfig(cfg)
		assert.NoError(t, err)
		
		// Verify storage is set (note: storage is private, can't directly test)
		// Test that the config was applied without error
		assert.NotNil(t, core.cryptoManager)
	})

	t.Run("file storage", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{
			KeyMgmt: config.KeyManagementConfig{
				Storage: config.StorageConfig{
					Type: "file",
					Path: tmpDir,
				},
			},
			Networks: make(map[string]config.NetworkConfig),
		}

		err := core.ApplyConfig(cfg)
		assert.NoError(t, err)
	})

	t.Run("unsupported storage type", func(t *testing.T) {
		cfg := &config.Config{
			KeyMgmt: config.KeyManagementConfig{
				Storage: config.StorageConfig{
					Type: "unsupported",
				},
			},
		}

		err := core.ApplyConfig(cfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported storage type")
	})

	t.Run("network configuration", func(t *testing.T) {
		cfg := &config.Config{
			KeyMgmt: config.KeyManagementConfig{
				Storage: config.StorageConfig{
					Type: "memory",
				},
			},
			Networks: map[string]config.NetworkConfig{
				"ethereum": {
					Chains: map[string]config.ChainConfig{
						"mainnet": {
							RPC:      "https://eth-test.example.com",
							Contract: "0xabc",
						},
					},
				},
				"solana": {
					Chains: map[string]config.ChainConfig{
						"mainnet": {
							RPC:       "https://sol-test.example.com",
							ProgramID: "SageProgram123",
						},
					},
				},
			},
		}

		// Note: This would fail in real implementation because DID manager
		// needs actual blockchain clients. For testing, we'd need to mock
		// the Configure method.
		err := core.ApplyConfig(cfg)
		// The error is expected because we're not mocking the blockchain clients
		assert.Error(t, err)
	})
}

func TestCore_IsAgentRegistered(t *testing.T) {
	core := New()
	ctx := context.Background()

	// This test would need a mock DID manager to work properly
	// For now, testing the method exists and returns expected error
	registered, err := core.IsAgentRegistered(ctx, "did:sage:ethereum:test001")
	assert.Error(t, err) // Expected because no chain is configured
	assert.False(t, registered)
}

func TestCore_GetAgentRegistrationStatus(t *testing.T) {
	core := New()
	ctx := context.Background()

	// This test would need a mock DID manager to work properly
	// For now, testing the method exists and returns expected error
	status, err := core.GetAgentRegistrationStatus(ctx, "did:sage:ethereum:test002")
	assert.Error(t, err) // Expected because no chain is configured
	assert.Nil(t, status)
}

func TestCompleteConfigurationFlow(t *testing.T) {
	// Create a complete configuration
	cfg := &config.Config{
		Version: "1.0",
		Networks: map[string]config.NetworkConfig{
			"ethereum": {
				Chains: map[string]config.ChainConfig{
					"mainnet": {
						RPC:      "https://eth-mainnet.example.com",
						Contract: "0x1234567890123456789012345678901234567890",
						ChainID:  1,
					},
					"sepolia": {
						RPC:      "https://eth-sepolia.example.com",
						Contract: "0x0987654321098765432109876543210987654321",
						ChainID:  11155111,
					},
				},
			},
		},
		KeyMgmt: config.KeyManagementConfig{
			DefaultAlgorithm: "ed25519",
			Storage: config.StorageConfig{
				Type:       "memory",
				Encryption: true,
			},
			Algorithms: map[string]config.AlgorithmConfig{
				"ed25519": {
					Enabled: true,
				},
				"secp256k1": {
					Enabled:       true,
					AddressFormat: "ethereum",
				},
			},
		},
		Proxy: config.ProxyConfig{
			Enabled: false,
		},
		Registry: config.RegistrationConfig{
			AutoRegister:  true,
			CheckInterval: 5,
			RetryPolicy: config.RetryPolicyConfig{
				MaxAttempts:  3,
				Backoff:      "exponential",
				InitialDelay: 1,
				MaxDelay:     30,
			},
		},
		Security: config.SecurityConfig{
			SignatureAlgorithms: []string{"EdDSA", "ECDSA"},
			Verification: config.VerificationConfig{
				RequireActiveAgent: true,
				MaxClockSkew:       5,
				CacheTTL:           10,
			},
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}

	// Test creating core with this config
	core, err := NewWithConfig(cfg)
	// Note: Configuration is successful, but blockchain clients aren't initialized
	// This is expected behavior for the configuration system
	require.NoError(t, err)
	assert.NotNil(t, core)
	assert.Equal(t, cfg, core.config)
}

func TestCore_GenerateKeyPairWithConfig(t *testing.T) {
	cfg := &config.Config{
		KeyMgmt: config.KeyManagementConfig{
			DefaultAlgorithm: "ed25519",
			Storage: config.StorageConfig{
				Type: "memory",
			},
		},
		Networks: make(map[string]config.NetworkConfig),
	}

	core, err := NewWithConfig(cfg)
	require.NoError(t, err)

	// Generate key pair using default algorithm from config
	keyPair, err := core.GenerateKeyPair(crypto.KeyTypeEd25519)
	assert.NoError(t, err)
	assert.NotNil(t, keyPair)
	assert.Equal(t, crypto.KeyTypeEd25519, keyPair.Type())
}