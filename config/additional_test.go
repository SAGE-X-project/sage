package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigTypes(t *testing.T) {
	t.Run("NetworkConfig", func(t *testing.T) {
		networkConfig := NetworkConfig{
			Chains: map[string]ChainConfig{
				"mainnet": {
					RPC:       "https://eth-mainnet.example.com",
					Contract:  "0x123",
					ProgramID: "",
					ChainID:   1,
				},
			},
		}

		assert.Contains(t, networkConfig.Chains, "mainnet")
		mainnet := networkConfig.Chains["mainnet"]
		assert.Equal(t, "https://eth-mainnet.example.com", mainnet.RPC)
		assert.Equal(t, "0x123", mainnet.Contract)
		assert.Equal(t, uint64(1), mainnet.ChainID)
	})

	t.Run("KeyManagementConfig", func(t *testing.T) {
		keyConfig := KeyManagementConfig{
			DefaultAlgorithm: "ed25519",
			Storage: StorageConfig{
				Type:       "file",
				Path:       "/tmp/keys",
				Encryption: true,
			},
			Algorithms: map[string]AlgorithmConfig{
				"ed25519": {
					Enabled: true,
				},
				"secp256k1": {
					Enabled:       true,
					AddressFormat: "ethereum",
				},
			},
		}

		assert.Equal(t, "ed25519", keyConfig.DefaultAlgorithm)
		assert.Equal(t, "file", keyConfig.Storage.Type)
		assert.Equal(t, "/tmp/keys", keyConfig.Storage.Path)
		assert.True(t, keyConfig.Storage.Encryption)
		
		assert.Contains(t, keyConfig.Algorithms, "ed25519")
		assert.Contains(t, keyConfig.Algorithms, "secp256k1")
		assert.True(t, keyConfig.Algorithms["ed25519"].Enabled)
		assert.Equal(t, "ethereum", keyConfig.Algorithms["secp256k1"].AddressFormat)
	})

	t.Run("ProxyConfig", func(t *testing.T) {
		proxyConfig := ProxyConfig{
			Enabled: true,
			Endpoints: []ProxyEndpoint{
				{
					URL:    "https://proxy1.example.com",
					APIKey: "key1",
				},
				{
					URL:    "https://proxy2.example.com", 
					APIKey: "key2",
				},
			},
			GasPolicy: GasPolicyConfig{
				MaxGasPrice:    "100 gwei",
				SponsorAddress: "0xsponsor",
			},
		}

		assert.True(t, proxyConfig.Enabled)
		assert.Len(t, proxyConfig.Endpoints, 2)
		assert.Equal(t, "https://proxy1.example.com", proxyConfig.Endpoints[0].URL)
		assert.Equal(t, "key1", proxyConfig.Endpoints[0].APIKey)
		assert.Equal(t, "100 gwei", proxyConfig.GasPolicy.MaxGasPrice)
		assert.Equal(t, "0xsponsor", proxyConfig.GasPolicy.SponsorAddress)
	})

	t.Run("SecurityConfig", func(t *testing.T) {
		securityConfig := SecurityConfig{
			SignatureAlgorithms: []string{"EdDSA", "ECDSA"},
			Verification: VerificationConfig{
				RequireActiveAgent: true,
				MaxClockSkew:       5,
				CacheTTL:          10,
			},
		}

		assert.Len(t, securityConfig.SignatureAlgorithms, 2)
		assert.Contains(t, securityConfig.SignatureAlgorithms, "EdDSA")
		assert.Contains(t, securityConfig.SignatureAlgorithms, "ECDSA")
		assert.True(t, securityConfig.Verification.RequireActiveAgent)
		assert.Equal(t, time.Duration(5), securityConfig.Verification.MaxClockSkew)
		assert.Equal(t, time.Duration(10), securityConfig.Verification.CacheTTL)
	})
}

func TestConfigLoader_EdgeCases(t *testing.T) {
	loader := NewConfigLoader()

	t.Run("Load non-existent file", func(t *testing.T) {
		_, err := loader.Load("/non/existent/file.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open config file")
	})

	t.Run("Load invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid.yaml")
		
		invalidYAML := `
version: "1.0"
invalid: [unclosed array
`
		err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
		require.NoError(t, err)

		_, err = loader.Load(configPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse config")
	})

	t.Run("Load with missing required fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "missing-fields.yaml")
		
		configContent := `# Missing version field
key_management:
  default_algorithm: "ed25519"
  storage:
    type: "memory"
`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		_, err = loader.Load(configPath)
		// Load itself should fail validation
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version is required")
	})
}

func TestEnvironmentVariableHelpers(t *testing.T) {
	t.Run("getEnvOrDefault", func(t *testing.T) {
		os.Setenv("TEST_EXISTING", "existing_value")
		defer os.Unsetenv("TEST_EXISTING")

		// Test existing variable
		assert.Equal(t, "existing_value", getEnvOrDefault("TEST_EXISTING", "default"))
		
		// Test non-existing variable
		assert.Equal(t, "default", getEnvOrDefault("TEST_NON_EXISTING", "default"))
	})

	t.Run("getEnvBool", func(t *testing.T) {
		testCases := []struct {
			envValue    string
			defaultVal  bool
			expected    bool
		}{
			{"true", false, true},
			{"false", true, false}, // "false" != "true" and != "1", so returns false
			{"1", false, true},
			{"0", true, false}, // "0" != "true" and != "1", so returns false
			{"yes", false, false}, // "yes" != "true" and != "1", so returns false
			{"no", true, false}, // "no" != "true" and != "1", so returns false
			{"invalid", true, false}, // != "true" and != "1", so returns false
		}

		for _, tc := range testCases {
			os.Setenv("TEST_BOOL", tc.envValue)
			result := getEnvBool("TEST_BOOL", tc.defaultVal)
			assert.Equal(t, tc.expected, result, "envValue: %s, default: %v", tc.envValue, tc.defaultVal)
		}
		
		os.Unsetenv("TEST_BOOL")
		
		// Test non-existing variable
		assert.True(t, getEnvBool("TEST_NON_EXISTING_BOOL", true))
		assert.False(t, getEnvBool("TEST_NON_EXISTING_BOOL", false))
	})

	t.Run("getEnvDuration", func(t *testing.T) {
		testCases := []struct {
			envValue    string
			defaultVal  time.Duration
			expected    time.Duration
		}{
			{"5s", 10 * time.Second, 5 * time.Second},
			{"2m", 10 * time.Second, 2 * time.Minute},
			{"1h", 10 * time.Second, 1 * time.Hour},
			{"invalid", 10 * time.Second, 10 * time.Second}, // Should return default
		}

		for _, tc := range testCases {
			os.Setenv("TEST_DURATION", tc.envValue)
			result := getEnvDuration("TEST_DURATION", tc.defaultVal)
			assert.Equal(t, tc.expected, result, "envValue: %s, default: %v", tc.envValue, tc.defaultVal)
		}
		
		os.Unsetenv("TEST_DURATION")
		
		// Test non-existing variable
		defaultDuration := 15 * time.Minute
		assert.Equal(t, defaultDuration, getEnvDuration("TEST_NON_EXISTING_DURATION", defaultDuration))
	})
}

func TestConfigLoader_Concurrency(t *testing.T) {
	loader := NewConfigLoader()
	
	// Set initial config
	testConfig := &Config{
		Version: "1.0",
		KeyMgmt: KeyManagementConfig{
			DefaultAlgorithm: "ed25519",
		},
	}
	
	loader.mu.Lock()
	loader.config = testConfig
	loader.mu.Unlock()

	// Test concurrent access
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			config := loader.GetConfig()
			assert.NotNil(t, config)
			assert.Equal(t, "1.0", config.Version)
			done <- true
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for concurrent access")
		}
	}
}

func TestValidationEdgeCases(t *testing.T) {
	loader := NewConfigLoader()

	t.Run("empty config", func(t *testing.T) {
		config := &Config{}
		err := loader.Validate(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version is required")
	})

	t.Run("config with only version", func(t *testing.T) {
		config := &Config{
			Version: "1.0",
		}
		err := loader.Validate(config)
		// Validation requires default algorithm to be set
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "default algorithm is required")
	})
}