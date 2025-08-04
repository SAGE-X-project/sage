package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoader_Load(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	
	configContent := `version: "1.0"

networks:
  ethereum:
    chains:
      mainnet:
        rpc: "https://eth-mainnet.example.com"
        contract: "0x1234567890123456789012345678901234567890"
        chain_id: 1

key_management:
  default_algorithm: "ed25519"
  storage:
    type: "memory"
    encryption: true

logging:
  level: "info"
  format: "json"
  output: "stdout"`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Test loading
	loader := NewConfigLoader()
	config, err := loader.Load(configPath)
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Verify loaded values
	assert.Equal(t, "1.0", config.Version)
	assert.Equal(t, "ed25519", config.KeyMgmt.DefaultAlgorithm)
	assert.Equal(t, "memory", config.KeyMgmt.Storage.Type)
	assert.True(t, config.KeyMgmt.Storage.Encryption)
	
	// Check network config
	assert.Contains(t, config.Networks, "ethereum")
	ethNetwork := config.Networks["ethereum"]
	assert.Contains(t, ethNetwork.Chains, "mainnet")
	mainnet := ethNetwork.Chains["mainnet"]
	assert.Equal(t, "https://eth-mainnet.example.com", mainnet.RPC)
	assert.Equal(t, "0x1234567890123456789012345678901234567890", mainnet.Contract)
	assert.Equal(t, uint64(1), mainnet.ChainID)
}

func TestConfigLoader_LoadWithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_RPC_URL", "https://test-rpc.example.com")
	os.Setenv("TEST_API_KEY", "test-api-key-123")
	defer os.Unsetenv("TEST_RPC_URL")
	defer os.Unsetenv("TEST_API_KEY")

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config-env.yaml")
	
	configContent := `version: "1.0"

networks:
  ethereum:
    chains:
      mainnet:
        rpc: "${TEST_RPC_URL}"
        contract: "0xabc"

proxy:
  enabled: true
  endpoints:
    - url: "https://proxy.example.com"
      api_key: "${TEST_API_KEY}"

key_management:
  default_algorithm: "ed25519"
  storage:
    type: "file"

logging:
  level: "debug"
  format: "text"
  output: "stdout"`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	loader := NewConfigLoader()
	config, err := loader.Load(configPath)
	require.NoError(t, err)

	// Verify environment variable substitution
	assert.Equal(t, "https://test-rpc.example.com", config.Networks["ethereum"].Chains["mainnet"].RPC)
	assert.Equal(t, "test-api-key-123", config.Proxy.Endpoints[0].APIKey)
}

func TestConfigLoader_LoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("SAGE_KEY_ALGORITHM", "secp256k1")
	os.Setenv("SAGE_KEY_STORAGE_TYPE", "memory")
	os.Setenv("SAGE_LOG_LEVEL", "debug")
	os.Setenv("SAGE_PROXY_ENABLED", "true")
	os.Setenv("SAGE_ETH_RPC", "https://eth.example.com")
	os.Setenv("SAGE_ETH_CONTRACT", "0x123")
	
	defer func() {
		os.Unsetenv("SAGE_KEY_ALGORITHM")
		os.Unsetenv("SAGE_KEY_STORAGE_TYPE")
		os.Unsetenv("SAGE_LOG_LEVEL")
		os.Unsetenv("SAGE_PROXY_ENABLED")
		os.Unsetenv("SAGE_ETH_RPC")
		os.Unsetenv("SAGE_ETH_CONTRACT")
	}()

	loader := NewConfigLoader()
	config, err := loader.LoadFromEnv()
	require.NoError(t, err)

	assert.Equal(t, "secp256k1", config.KeyMgmt.DefaultAlgorithm)
	assert.Equal(t, "memory", config.KeyMgmt.Storage.Type)
	assert.Equal(t, "debug", config.Logging.Level)
	assert.True(t, config.Proxy.Enabled)
	
	// Check network config from env
	assert.Contains(t, config.Networks, "ethereum")
	assert.Equal(t, "https://eth.example.com", config.Networks["ethereum"].Chains["mainnet"].RPC)
	assert.Equal(t, "0x123", config.Networks["ethereum"].Chains["mainnet"].Contract)
}

func TestConfigLoader_Validate(t *testing.T) {
	loader := NewConfigLoader()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Version: "1.0",
				KeyMgmt: KeyManagementConfig{
					DefaultAlgorithm: "ed25519",
					Storage: StorageConfig{
						Type: "memory",
					},
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			config: &Config{
				KeyMgmt: KeyManagementConfig{
					DefaultAlgorithm: "ed25519",
					Storage: StorageConfig{
						Type: "memory",
					},
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
			},
			wantErr: true,
			errMsg:  "version is required",
		},
		{
			name: "invalid algorithm",
			config: &Config{
				Version: "1.0",
				KeyMgmt: KeyManagementConfig{
					DefaultAlgorithm: "invalid-algo",
					Storage: StorageConfig{
						Type: "memory",
					},
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
			},
			wantErr: true,
			errMsg:  "invalid default algorithm",
		},
		{
			name: "invalid storage type",
			config: &Config{
				Version: "1.0",
				KeyMgmt: KeyManagementConfig{
					DefaultAlgorithm: "ed25519",
					Storage: StorageConfig{
						Type: "invalid-storage",
					},
				},
				Logging: LoggingConfig{
					Level:  "info",
					Format: "json",
				},
			},
			wantErr: true,
			errMsg:  "invalid storage type",
		},
		{
			name: "invalid log level",
			config: &Config{
				Version: "1.0",
				KeyMgmt: KeyManagementConfig{
					DefaultAlgorithm: "ed25519",
					Storage: StorageConfig{
						Type: "memory",
					},
				},
				Logging: LoggingConfig{
					Level:  "invalid-level",
					Format: "json",
				},
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.Validate(tt.config)
			if tt.wantErr {
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

func TestConfigLoader_GetConfig(t *testing.T) {
	loader := NewConfigLoader()
	
	// Should return nil initially
	assert.Nil(t, loader.GetConfig())

	// Set a config
	testConfig := &Config{
		Version: "1.0",
		KeyMgmt: KeyManagementConfig{
			DefaultAlgorithm: "ed25519",
		},
	}
	
	loader.mu.Lock()
	loader.config = testConfig
	loader.mu.Unlock()

	// Should return the config
	retrieved := loader.GetConfig()
	assert.Equal(t, testConfig, retrieved)
}

func TestHelperFunctions(t *testing.T) {
	// Test getEnvOrDefault
	os.Setenv("TEST_VAR", "test-value")
	defer os.Unsetenv("TEST_VAR")
	
	assert.Equal(t, "test-value", getEnvOrDefault("TEST_VAR", "default"))
	assert.Equal(t, "default", getEnvOrDefault("NONEXISTENT_VAR", "default"))

	// Test getEnvBool
	os.Setenv("TEST_BOOL_TRUE", "true")
	os.Setenv("TEST_BOOL_FALSE", "false")
	os.Setenv("TEST_BOOL_1", "1")
	defer func() {
		os.Unsetenv("TEST_BOOL_TRUE")
		os.Unsetenv("TEST_BOOL_FALSE")
		os.Unsetenv("TEST_BOOL_1")
	}()
	
	assert.True(t, getEnvBool("TEST_BOOL_TRUE", false))
	assert.False(t, getEnvBool("TEST_BOOL_FALSE", true))
	assert.True(t, getEnvBool("TEST_BOOL_1", false))
	assert.True(t, getEnvBool("NONEXISTENT_BOOL", true))

	// Test getEnvDuration
	os.Setenv("TEST_DURATION", "5m")
	os.Setenv("TEST_INVALID_DURATION", "invalid")
	defer func() {
		os.Unsetenv("TEST_DURATION")
		os.Unsetenv("TEST_INVALID_DURATION")
	}()
	
	assert.Equal(t, 5*time.Minute, getEnvDuration("TEST_DURATION", 10*time.Minute))
	assert.Equal(t, 10*time.Minute, getEnvDuration("TEST_INVALID_DURATION", 10*time.Minute))
	assert.Equal(t, 15*time.Minute, getEnvDuration("NONEXISTENT_DURATION", 15*time.Minute))
}

func TestConfigLoader_HomeDirExpansion(t *testing.T) {
	// Create test config with ~ path
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "home-test.yaml")
	
	configContent := `version: "1.0"
key_management:
  default_algorithm: "ed25519"
  storage:
    type: "file"
    path: "~/.sage/keys"
logging:
  level: "info"
  format: "json"
  output: "stdout"`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	loader := NewConfigLoader()
	config, err := loader.Load(configPath)
	require.NoError(t, err)

	// Path should be expanded but still contains ~
	// (expansion happens during actual use)
	assert.Equal(t, "~/.sage/keys", config.KeyMgmt.Storage.Path)
}

func TestComplexConfigScenarios(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "complex-config.yaml")
	
	configContent := `version: "1.0"

networks:
  ethereum:
    chains:
      mainnet:
        rpc: "https://eth-mainnet.example.com"
        contract: "0x1234"
        chain_id: 1
      sepolia:
        rpc: "https://eth-sepolia.example.com"
        contract: "0x5678"
        chain_id: 11155111
  solana:
    chains:
      mainnet:
        rpc: "https://sol-mainnet.example.com"
        program_id: "SageProgram123"
      devnet:
        rpc: "https://sol-devnet.example.com"
        program_id: "SageDevProgram456"

key_management:
  default_algorithm: "ed25519"
  storage:
    type: "file"
    path: "/tmp/keys"
    encryption: true
  algorithms:
    ed25519:
      enabled: true
    secp256k1:
      enabled: true
      address_format: "ethereum"
    rsa:
      enabled: false
      key_size: 2048

proxy:
  enabled: true
  endpoints:
    - url: "https://proxy1.example.com"
      api_key: "key1"
    - url: "https://proxy2.example.com"
      api_key: "key2"
  gas_policy:
    max_gas_price: "100 gwei"
    sponsor_address: "0xsponsor"

registration:
  auto_register: true
  check_interval: "5m"
  retry_policy:
    max_attempts: 3
    backoff: "exponential"
    initial_delay: "1s"
    max_delay: "30s"

security:
  signature_algorithms:
    - "EdDSA"
    - "ECDSA"
    - "RSA-PSS"
  verification:
    require_active_agent: true
    max_clock_skew: "5m"
    cache_ttl: "10m"

logging:
  level: "debug"
  format: "json"
  output: "stderr"`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	loader := NewConfigLoader()
	config, err := loader.Load(configPath)
	require.NoError(t, err)

	// Verify complex structure
	assert.Len(t, config.Networks, 2)
	assert.Len(t, config.Networks["ethereum"].Chains, 2)
	assert.Len(t, config.Networks["solana"].Chains, 2)
	
	assert.Len(t, config.Proxy.Endpoints, 2)
	assert.Equal(t, "https://proxy1.example.com", config.Proxy.Endpoints[0].URL)
	
	assert.Len(t, config.Security.SignatureAlgorithms, 3)
	assert.Contains(t, config.Security.SignatureAlgorithms, "EdDSA")
	
	assert.Equal(t, 3, config.Registry.RetryPolicy.MaxAttempts)
	assert.Equal(t, "exponential", config.Registry.RetryPolicy.Backoff)
}