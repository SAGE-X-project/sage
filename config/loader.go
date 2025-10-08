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


package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// LoaderOptions configures the configuration loader
type LoaderOptions struct {
	// ConfigDir is the directory containing config files (default: ./config)
	ConfigDir string
	// Environment overrides automatic environment detection
	Environment string
	// SkipEnvSubstitution disables environment variable substitution
	SkipEnvSubstitution bool
	// SkipValidation disables configuration validation
	SkipValidation bool
}

// DefaultLoaderOptions returns default loader options
func DefaultLoaderOptions() LoaderOptions {
	return LoaderOptions{
		ConfigDir:           "config",
		Environment:         "",
		SkipEnvSubstitution: false,
		SkipValidation:      false,
	}
}

// Load loads configuration with automatic environment detection
func Load(opts ...LoaderOptions) (*Config, error) {
	options := DefaultLoaderOptions()
	if len(opts) > 0 {
		options = opts[0]
	}

	// Determine environment
	env := options.Environment
	if env == "" {
		env = GetEnvironment()
	}

	// Try to load environment-specific config file
	envConfigPath := filepath.Join(options.ConfigDir, fmt.Sprintf("%s.yaml", env))
	cfg, err := loadConfigFile(envConfigPath)
	if err != nil {
		// Fall back to default config file
		defaultConfigPath := filepath.Join(options.ConfigDir, "default.yaml")
		cfg, err = loadConfigFile(defaultConfigPath)
		if err != nil {
			// Fall back to config.yaml
			configPath := filepath.Join(options.ConfigDir, "config.yaml")
			cfg, err = loadConfigFile(configPath)
			if err != nil {
				// Return empty config with defaults
				cfg = &Config{}
			}
		}
	}

	// Set environment
	if cfg.Environment == "" {
		cfg.Environment = env
	}

	// Apply defaults
	setDefaults(cfg)

	// Substitute environment variables
	if !options.SkipEnvSubstitution {
		SubstituteEnvVarsInConfig(cfg)
	}

	// Override with environment variables (highest priority)
	applyEnvironmentOverrides(cfg)

	// Validate configuration
	if !options.SkipValidation {
		errors := ValidateConfiguration(cfg)
		// Only fail on error-level validation issues
		for _, e := range errors {
			if e.Level == "error" {
				return nil, fmt.Errorf("configuration validation failed: %s - %s", e.Field, e.Message)
			}
		}
	}

	return cfg, nil
}

// loadConfigFile loads a single config file
func loadConfigFile(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}
	return LoadFromFile(path)
}

// applyEnvironmentOverrides overrides config with environment variables
func applyEnvironmentOverrides(cfg *Config) {
	// Blockchain overrides
	if rpc := os.Getenv("SAGE_BLOCKCHAIN_RPC"); rpc != "" && cfg.Blockchain != nil {
		cfg.Blockchain.NetworkRPC = rpc
	}
	if addr := os.Getenv("SAGE_CONTRACT_ADDRESS"); addr != "" && cfg.Blockchain != nil {
		cfg.Blockchain.ContractAddr = addr
	}

	// DID overrides
	if registryAddr := os.Getenv("SAGE_DID_REGISTRY"); registryAddr != "" && cfg.DID != nil {
		cfg.DID.RegistryAddress = registryAddr
	}

	// KeyStore overrides
	if ksDir := os.Getenv("SAGE_KEYSTORE_DIR"); ksDir != "" && cfg.KeyStore != nil {
		cfg.KeyStore.Directory = ksDir
	}

	// Logging overrides
	if logLevel := os.Getenv("SAGE_LOG_LEVEL"); logLevel != "" && cfg.Logging != nil {
		cfg.Logging.Level = logLevel
	}
	if logFormat := os.Getenv("SAGE_LOG_FORMAT"); logFormat != "" && cfg.Logging != nil {
		cfg.Logging.Format = logFormat
	}

	// Metrics overrides
	if os.Getenv("SAGE_METRICS_ENABLED") == "true" && cfg.Metrics != nil {
		cfg.Metrics.Enabled = true
	}
	if os.Getenv("SAGE_METRICS_ENABLED") == "false" && cfg.Metrics != nil {
		cfg.Metrics.Enabled = false
	}
}

// LoadForEnvironment loads configuration for a specific environment
func LoadForEnvironment(environment string) (*Config, error) {
	return Load(LoaderOptions{
		ConfigDir:   "config",
		Environment: environment,
	})
}

// MustLoad loads configuration or panics on error
func MustLoad(opts ...LoaderOptions) *Config {
	cfg, err := Load(opts...)
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}
	return cfg
}
