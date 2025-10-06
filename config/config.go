// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Environment string             `yaml:"environment" json:"environment"`
	Blockchain  *BlockchainConfig  `yaml:"blockchain" json:"blockchain"`
	DID         *DIDConfig         `yaml:"did" json:"did"`
	KeyStore    *KeyStoreConfig    `yaml:"keystore" json:"keystore"`
	Logging     *LoggingConfig     `yaml:"logging" json:"logging"`
	Metrics     *MetricsConfig     `yaml:"metrics" json:"metrics"`
	Health      *HealthConfig      `yaml:"health" json:"health"`
}

// BlockchainConfig is already defined in blockchain.go

// DIDConfig represents DID configuration
type DIDConfig struct {
	RegistryAddress string        `yaml:"registry_address" json:"registry_address"`
	Method          string        `yaml:"method" json:"method"`
	Network         string        `yaml:"network" json:"network"`
	CacheSize       int           `yaml:"cache_size" json:"cache_size"`
	CacheTTL        time.Duration `yaml:"cache_ttl" json:"cache_ttl"`
}

// KeyStoreConfig represents key storage configuration
type KeyStoreConfig struct {
	Type          string `yaml:"type" json:"type"`
	Directory     string `yaml:"directory" json:"directory"`
	PassphraseEnv string `yaml:"passphrase_env" json:"passphrase_env"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level    string `yaml:"level" json:"level"`
	Format   string `yaml:"format" json:"format"`
	Output   string `yaml:"output" json:"output"`
	FilePath string `yaml:"file_path" json:"file_path"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Port    int    `yaml:"port" json:"port"`
	Path    string `yaml:"path" json:"path"`
}

// HealthConfig represents health check configuration
type HealthConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Port    int      `yaml:"port" json:"port"`
	Path    string   `yaml:"path" json:"path"`
	Checks  []string `yaml:"checks" json:"checks"`
}

// LoadFromFile loads configuration from a file
func LoadFromFile(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{}

	// Try to parse as YAML first
	if err := yaml.Unmarshal(data, cfg); err != nil {
		// Try JSON if YAML fails
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file (tried YAML and JSON): %w", err)
		}
	}

	// Set defaults
	setDefaults(cfg)

	return cfg, nil
}

// SaveToFile saves configuration to a file
func SaveToFile(cfg *Config, path string) error {
	// Determine format by extension
	var data []byte
	var err error

	if path[len(path)-5:] == ".json" {
		data, err = json.MarshalIndent(cfg, "", "  ")
	} else {
		data, err = yaml.Marshal(cfg)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := ioutil.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// setDefaults sets default values for configuration
func setDefaults(cfg *Config) {
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	if cfg.Blockchain != nil {
		if cfg.Blockchain.GasLimit == 0 {
			cfg.Blockchain.GasLimit = 3000000
		}
		if cfg.Blockchain.MaxRetries == 0 {
			cfg.Blockchain.MaxRetries = 3
		}
		if cfg.Blockchain.RetryDelay == 0 {
			cfg.Blockchain.RetryDelay = 1 * time.Second
		}
		if cfg.Blockchain.RequestTimeout == 0 {
			cfg.Blockchain.RequestTimeout = 30 * time.Second
		}
	}

	if cfg.DID != nil {
		if cfg.DID.Method == "" {
			cfg.DID.Method = "sage"
		}
		if cfg.DID.Network == "" {
			cfg.DID.Network = "ethereum"
		}
		if cfg.DID.CacheSize == 0 {
			cfg.DID.CacheSize = 100
		}
		if cfg.DID.CacheTTL == 0 {
			cfg.DID.CacheTTL = 5 * time.Minute
		}
	}

	if cfg.KeyStore != nil {
		if cfg.KeyStore.Type == "" {
			cfg.KeyStore.Type = "encrypted-file"
		}
		if cfg.KeyStore.Directory == "" {
			cfg.KeyStore.Directory = ".sage/keys"
		}
	}

	if cfg.Logging != nil {
		if cfg.Logging.Level == "" {
			cfg.Logging.Level = "info"
		}
		if cfg.Logging.Format == "" {
			cfg.Logging.Format = "json"
		}
		if cfg.Logging.Output == "" {
			cfg.Logging.Output = "stdout"
		}
	}
}