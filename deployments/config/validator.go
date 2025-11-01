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
	"context"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
	Level   string // "error", "warning", "info"
}

// ValidateConfiguration validates the entire configuration
func ValidateConfiguration(cfg *Config) []ValidationError {
	var errors []ValidationError

	// Validate blockchain config
	if cfg.Blockchain != nil {
		errors = append(errors, validateBlockchainConfig(cfg.Blockchain)...)
	}

	// Validate DID config
	if cfg.DID != nil {
		errors = append(errors, validateDIDConfig(cfg.DID)...)
	}

	// Validate environment
	errors = append(errors, validateEnvironment(cfg.Environment)...)

	return errors
}

// validateBlockchainConfig validates blockchain configuration
func validateBlockchainConfig(cfg *BlockchainConfig) []ValidationError {
	var errors []ValidationError

	// Check RPC URL
	if cfg.NetworkRPC == "" {
		errors = append(errors, ValidationError{
			Field:   "Blockchain.NetworkRPC",
			Message: "RPC URL is required",
			Level:   "error",
		})
	} else {
		// Validate URL format
		if _, err := url.Parse(cfg.NetworkRPC); err != nil {
			errors = append(errors, ValidationError{
				Field:   "Blockchain.NetworkRPC",
				Message: fmt.Sprintf("Invalid RPC URL: %v", err),
				Level:   "error",
			})
		}

		// Try to connect
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client, err := ethclient.DialContext(ctx, cfg.NetworkRPC)
		if err != nil {
			errors = append(errors, ValidationError{
				Field:   "Blockchain.NetworkRPC",
				Message: fmt.Sprintf("Cannot connect to RPC: %v", err),
				Level:   "warning",
			})
		} else {
			defer client.Close()

			// Check chain ID matches
			chainID, err := client.ChainID(ctx)
			if err != nil {
				errors = append(errors, ValidationError{
					Field:   "Blockchain.ChainID",
					Message: fmt.Sprintf("Cannot get chain ID: %v", err),
					Level:   "warning",
				})
			} else if cfg.ChainID != nil && chainID.Cmp(cfg.ChainID) != 0 {
				errors = append(errors, ValidationError{
					Field:   "Blockchain.ChainID",
					Message: fmt.Sprintf("Chain ID mismatch: configured=%s, actual=%s", cfg.ChainID, chainID),
					Level:   "error",
				})
			}
		}
	}

	// Check gas settings
	if cfg.GasLimit == 0 {
		errors = append(errors, ValidationError{
			Field:   "Blockchain.GasLimit",
			Message: "Gas limit should be set (recommended: 3000000)",
			Level:   "warning",
		})
	}

	if cfg.MaxGasPrice == nil || cfg.MaxGasPrice.Cmp(big.NewInt(0)) == 0 {
		errors = append(errors, ValidationError{
			Field:   "Blockchain.MaxGasPrice",
			Message: "Max gas price should be set to prevent excessive fees",
			Level:   "warning",
		})
	}

	// Check retry settings
	if cfg.MaxRetries < 0 {
		errors = append(errors, ValidationError{
			Field:   "Blockchain.MaxRetries",
			Message: "Max retries cannot be negative",
			Level:   "error",
		})
	}

	if cfg.RetryDelay < 0 {
		errors = append(errors, ValidationError{
			Field:   "Blockchain.RetryDelay",
			Message: "Retry delay cannot be negative",
			Level:   "error",
		})
	}

	return errors
}

// validateDIDConfig validates DID configuration
func validateDIDConfig(cfg *DIDConfig) []ValidationError {
	var errors []ValidationError

	// Check registry address
	if cfg.RegistryAddress == "" {
		errors = append(errors, ValidationError{
			Field:   "DID.RegistryAddress",
			Message: "DID registry address is required",
			Level:   "error",
		})
	}

	// Check method
	if cfg.Method == "" {
		cfg.Method = "sage" // Set default
	}

	// Check network
	if cfg.Network == "" {
		cfg.Network = "ethereum" // Set default
	}

	// Check cache settings
	if cfg.CacheSize < 0 {
		errors = append(errors, ValidationError{
			Field:   "DID.CacheSize",
			Message: "Cache size cannot be negative",
			Level:   "error",
		})
	}

	if cfg.CacheTTL < 0 {
		errors = append(errors, ValidationError{
			Field:   "DID.CacheTTL",
			Message: "Cache TTL cannot be negative",
			Level:   "error",
		})
	}

	return errors
}

// validateEnvironment validates environment settings
func validateEnvironment(env string) []ValidationError {
	var errors []ValidationError

	validEnvs := []string{"local", "development", "staging", "production"}
	env = strings.ToLower(env)

	valid := false
	for _, v := range validEnvs {
		if env == v {
			valid = true
			break
		}
	}

	if !valid {
		errors = append(errors, ValidationError{
			Field:   "Environment",
			Message: fmt.Sprintf("Invalid environment: %s (valid: %v)", env, validEnvs),
			Level:   "error",
		})
	}

	// Warn about production environment
	if env == "production" {
		errors = append(errors, ValidationError{
			Field:   "Environment",
			Message: "Running in production mode - ensure all security settings are configured",
			Level:   "info",
		})
	}

	return errors
}

// ValidateFile validates a configuration file
func ValidateFile(path string) ([]ValidationError, error) {
	// Check file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", path)
	}

	// Load configuration
	cfg, err := LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate
	return ValidateConfiguration(cfg), nil
}

// PrintValidationErrors prints validation errors in a formatted way
func PrintValidationErrors(errors []ValidationError) {
	if len(errors) == 0 {
		fmt.Println(" Configuration is valid")
		return
	}

	// Group by level
	var errorCount, warningCount, infoCount int
	for _, e := range errors {
		switch e.Level {
		case "error":
			errorCount++
		case "warning":
			warningCount++
		case "info":
			infoCount++
		}
	}

	fmt.Printf("Configuration validation found %d errors, %d warnings, %d info messages\n\n",
		errorCount, warningCount, infoCount)

	// Print errors first
	for _, e := range errors {
		if e.Level == "error" {
			fmt.Printf(" ERROR: %s - %s\n", e.Field, e.Message)
		}
	}

	// Then warnings
	for _, e := range errors {
		if e.Level == "warning" {
			fmt.Printf("  WARNING: %s - %s\n", e.Field, e.Message)
		}
	}

	// Finally info
	for _, e := range errors {
		if e.Level == "info" {
			fmt.Printf("ℹ  INFO: %s - %s\n", e.Field, e.Message)
		}
	}
}
