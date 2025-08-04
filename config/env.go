package config

import (
	"os"
	"strings"
)

// Environment variable names
const (
	// General
	EnvConfigPath = "SAGE_CONFIG_PATH"
	
	// Network
	EnvEthereumRPC      = "SAGE_ETH_RPC"
	EnvEthereumContract = "SAGE_ETH_CONTRACT"
	EnvSolanaRPC        = "SAGE_SOL_RPC"
	EnvSolanaProgram    = "SAGE_SOL_PROGRAM"
	
	// Key Management
	EnvKeyAlgorithm    = "SAGE_KEY_ALGORITHM"
	EnvKeyStorageType  = "SAGE_KEY_STORAGE_TYPE"
	EnvKeyStoragePath  = "SAGE_KEY_STORAGE_PATH"
	EnvKeyEncryption   = "SAGE_KEY_ENCRYPTION"
	
	// Proxy
	EnvProxyEnabled = "SAGE_PROXY_ENABLED"
	EnvProxyURL     = "SAGE_PROXY_URL"
	EnvProxyAPIKey  = "SAGE_PROXY_API_KEY"
	
	// Registration
	EnvAutoRegister  = "SAGE_AUTO_REGISTER"
	EnvCheckInterval = "SAGE_CHECK_INTERVAL"
	
	// Security
	EnvRequireActiveAgent = "SAGE_REQUIRE_ACTIVE_AGENT"
	EnvMaxClockSkew       = "SAGE_MAX_CLOCK_SKEW"
	EnvCacheTTL           = "SAGE_CACHE_TTL"
	
	// Logging
	EnvLogLevel  = "SAGE_LOG_LEVEL"
	EnvLogFormat = "SAGE_LOG_FORMAT"
	EnvLogOutput = "SAGE_LOG_OUTPUT"
)

// EnvConfig provides environment variable configuration helpers
type EnvConfig struct{}

// NewEnvConfig creates a new environment configuration helper
func NewEnvConfig() *EnvConfig {
	return &EnvConfig{}
}

// GetConfigPath returns the configuration file path from environment
func (e *EnvConfig) GetConfigPath() string {
	if path := os.Getenv(EnvConfigPath); path != "" {
		return path
	}
	
	// Default paths
	home, err := os.UserHomeDir()
	if err != nil {
		return "sage-config.yaml"
	}
	
	// Check common locations
	locations := []string{
		"sage-config.yaml",
		".sage-config.yaml",
		home + "/.sage/config.yaml",
		home + "/.config/sage/config.yaml",
		"/etc/sage/config.yaml",
	}
	
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}
	
	return "sage-config.yaml"
}

// LoadFromEnvironment creates a config from environment variables only
func (e *EnvConfig) LoadFromEnvironment() map[string]string {
	env := make(map[string]string)
	
	// Get all SAGE_ prefixed environment variables
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "SAGE_") {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}
	}
	
	return env
}

// SetDefaults sets default environment variables if not already set
func (e *EnvConfig) SetDefaults() {
	defaults := map[string]string{
		EnvKeyAlgorithm:       "ed25519",
		EnvKeyStorageType:     "file",
		EnvKeyStoragePath:     "~/.sage/keys",
		EnvKeyEncryption:      "true",
		EnvAutoRegister:       "true",
		EnvCheckInterval:      "5m",
		EnvRequireActiveAgent: "true",
		EnvMaxClockSkew:       "5m",
		EnvCacheTTL:           "10m",
		EnvLogLevel:           "info",
		EnvLogFormat:          "json",
		EnvLogOutput:          "stdout",
	}
	
	for key, value := range defaults {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}