package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigLoader handles configuration loading and management
type ConfigLoader struct {
	config *Config
	mu     sync.RWMutex
}

// NewConfigLoader creates a new configuration loader
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{}
}

// Load loads configuration from a file
func (c *ConfigLoader) Load(path string) (*Config, error) {
	// Expand home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Read file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	// Parse YAML
	config, err := c.parseConfig(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate
	if err := c.Validate(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Apply environment variable substitutions
	if err := c.applyEnvVars(config); err != nil {
		return nil, fmt.Errorf("failed to apply environment variables: %w", err)
	}

	c.mu.Lock()
	c.config = config
	c.mu.Unlock()

	return config, nil
}

// LoadFromEnv loads configuration from environment variables
func (c *ConfigLoader) LoadFromEnv() (*Config, error) {
	config := &Config{
		Version: "1.0",
		Networks: make(map[string]NetworkConfig),
		KeyMgmt: KeyManagementConfig{
			DefaultAlgorithm: getEnvOrDefault("SAGE_KEY_ALGORITHM", "ed25519"),
			Storage: StorageConfig{
				Type:       getEnvOrDefault("SAGE_KEY_STORAGE_TYPE", "file"),
				Path:       getEnvOrDefault("SAGE_KEY_STORAGE_PATH", "~/.sage/keys"),
				Encryption: getEnvBool("SAGE_KEY_ENCRYPTION", true),
			},
		},
		Proxy: ProxyConfig{
			Enabled: getEnvBool("SAGE_PROXY_ENABLED", false),
		},
		Registry: RegistrationConfig{
			AutoRegister:  getEnvBool("SAGE_AUTO_REGISTER", true),
			CheckInterval: getEnvDuration("SAGE_CHECK_INTERVAL", 5*time.Minute),
		},
		Security: SecurityConfig{
			Verification: VerificationConfig{
				RequireActiveAgent: getEnvBool("SAGE_REQUIRE_ACTIVE_AGENT", true),
				MaxClockSkew:       getEnvDuration("SAGE_MAX_CLOCK_SKEW", 5*time.Minute),
				CacheTTL:           getEnvDuration("SAGE_CACHE_TTL", 10*time.Minute),
			},
		},
		Logging: LoggingConfig{
			Level:  getEnvOrDefault("SAGE_LOG_LEVEL", "info"),
			Format: getEnvOrDefault("SAGE_LOG_FORMAT", "json"),
			Output: getEnvOrDefault("SAGE_LOG_OUTPUT", "stdout"),
		},
	}

	// Load network configurations
	if ethRPC := os.Getenv("SAGE_ETH_RPC"); ethRPC != "" {
		config.Networks["ethereum"] = NetworkConfig{
			Chains: map[string]ChainConfig{
				"mainnet": {
					RPC:      ethRPC,
					Contract: os.Getenv("SAGE_ETH_CONTRACT"),
					ChainID:  1,
				},
			},
		}
	}

	if solRPC := os.Getenv("SAGE_SOL_RPC"); solRPC != "" {
		config.Networks["solana"] = NetworkConfig{
			Chains: map[string]ChainConfig{
				"mainnet": {
					RPC:       solRPC,
					ProgramID: os.Getenv("SAGE_SOL_PROGRAM"),
				},
			},
		}
	}

	if err := c.Validate(config); err != nil {
		return nil, fmt.Errorf("invalid configuration from environment: %w", err)
	}

	c.mu.Lock()
	c.config = config
	c.mu.Unlock()

	return config, nil
}

// Validate validates a configuration
func (c *ConfigLoader) Validate(config *Config) error {
	if config.Version == "" {
		return fmt.Errorf("version is required")
	}

	// Validate key management
	if config.KeyMgmt.DefaultAlgorithm == "" {
		return fmt.Errorf("default algorithm is required")
	}

	validAlgorithms := []string{"ed25519", "secp256k1", "rsa"}
	if !contains(validAlgorithms, config.KeyMgmt.DefaultAlgorithm) {
		return fmt.Errorf("invalid default algorithm: %s", config.KeyMgmt.DefaultAlgorithm)
	}

	// Validate storage type
	validStorageTypes := []string{"file", "memory", "hsm"}
	if !contains(validStorageTypes, config.KeyMgmt.Storage.Type) {
		return fmt.Errorf("invalid storage type: %s", config.KeyMgmt.Storage.Type)
	}

	// Validate logging
	validLogLevels := []string{"debug", "info", "warn", "error"}
	if !contains(validLogLevels, config.Logging.Level) {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	validLogFormats := []string{"json", "text"}
	if !contains(validLogFormats, config.Logging.Format) {
		return fmt.Errorf("invalid log format: %s", config.Logging.Format)
	}

	return nil
}

// Watch watches a configuration file for changes
func (c *ConfigLoader) Watch(path string, callback func(*Config)) error {
	// This would implement file watching using fsnotify
	// For now, returning not implemented
	return fmt.Errorf("watch not implemented")
}

// GetConfig returns the current configuration
func (c *ConfigLoader) GetConfig() *Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// parseConfig parses configuration from a reader
func (c *ConfigLoader) parseConfig(r io.Reader) (*Config, error) {
	var config Config
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

// applyEnvVars applies environment variable substitutions
func (c *ConfigLoader) applyEnvVars(config *Config) error {
	// Replace ${VAR} patterns with environment variable values
	// This would walk through all string fields and replace patterns
	// For brevity, showing just a few examples

	// Networks
	for _, network := range config.Networks {
		for name, chain := range network.Chains {
			chain.RPC = expandEnvVars(chain.RPC)
			chain.Contract = expandEnvVars(chain.Contract)
			chain.ProgramID = expandEnvVars(chain.ProgramID)
			network.Chains[name] = chain
		}
	}

	// Proxy
	for i, endpoint := range config.Proxy.Endpoints {
		endpoint.URL = expandEnvVars(endpoint.URL)
		endpoint.APIKey = expandEnvVars(endpoint.APIKey)
		config.Proxy.Endpoints[i] = endpoint
	}

	return nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.ToLower(value) == "true" || value == "1"
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

func expandEnvVars(s string) string {
	return os.ExpandEnv(s)
}