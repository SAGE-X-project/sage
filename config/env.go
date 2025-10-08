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
	"os"
	"regexp"
	"strings"
)

// envVarPattern matches ${VAR} or ${VAR:default}
var envVarPattern = regexp.MustCompile(`\$\{([^}:]+)(?::([^}]*))?\}`)

// SubstituteEnvVars replaces ${VAR} or ${VAR:default} with environment variable values
func SubstituteEnvVars(input string) string {
	return envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name and default value
		parts := envVarPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		varName := parts[1]
		defaultValue := ""
		if len(parts) > 2 {
			defaultValue = parts[2]
		}

		// Get environment variable
		value := os.Getenv(varName)
		if value == "" {
			return defaultValue
		}
		return value
	})
}

// SubstituteEnvVarsInConfig recursively substitutes environment variables in config
func SubstituteEnvVarsInConfig(cfg *Config) {
	if cfg == nil {
		return
	}

	// Substitute in Blockchain config
	if cfg.Blockchain != nil {
		cfg.Blockchain.NetworkRPC = SubstituteEnvVars(cfg.Blockchain.NetworkRPC)
		cfg.Blockchain.ContractAddr = SubstituteEnvVars(cfg.Blockchain.ContractAddr)
	}

	// Substitute in DID config
	if cfg.DID != nil {
		cfg.DID.RegistryAddress = SubstituteEnvVars(cfg.DID.RegistryAddress)
		cfg.DID.Method = SubstituteEnvVars(cfg.DID.Method)
		cfg.DID.Network = SubstituteEnvVars(cfg.DID.Network)
	}

	// Substitute in KeyStore config
	if cfg.KeyStore != nil {
		cfg.KeyStore.Type = SubstituteEnvVars(cfg.KeyStore.Type)
		cfg.KeyStore.Directory = SubstituteEnvVars(cfg.KeyStore.Directory)
		cfg.KeyStore.PassphraseEnv = SubstituteEnvVars(cfg.KeyStore.PassphraseEnv)
	}

	// Substitute in Logging config
	if cfg.Logging != nil {
		cfg.Logging.Level = SubstituteEnvVars(cfg.Logging.Level)
		cfg.Logging.Format = SubstituteEnvVars(cfg.Logging.Format)
		cfg.Logging.Output = SubstituteEnvVars(cfg.Logging.Output)
		cfg.Logging.FilePath = SubstituteEnvVars(cfg.Logging.FilePath)
	}

	// Substitute in Health config
	if cfg.Health != nil {
		cfg.Health.Path = SubstituteEnvVars(cfg.Health.Path)
	}

	// Substitute in Metrics config
	if cfg.Metrics != nil {
		cfg.Metrics.Path = SubstituteEnvVars(cfg.Metrics.Path)
	}
}

// GetEnvironment returns the current environment from SAGE_ENV or defaults to development
func GetEnvironment() string {
	env := os.Getenv("SAGE_ENV")
	if env == "" {
		env = os.Getenv("ENVIRONMENT")
	}
	if env == "" {
		env = "development"
	}
	return strings.ToLower(env)
}

// IsProduction returns true if running in production environment
func IsProduction() bool {
	return GetEnvironment() == "production"
}

// IsDevelopment returns true if running in development or local environment
func IsDevelopment() bool {
	env := GetEnvironment()
	return env == "development" || env == "local"
}
