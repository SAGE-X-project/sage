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
	"os"
	"testing"
)

func TestSubstituteEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "simple variable substitution",
			input:    "${TEST_VAR}",
			envVars:  map[string]string{"TEST_VAR": "value123"},
			expected: "value123",
		},
		{
			name:     "variable with default - variable exists",
			input:    "${TEST_VAR:default}",
			envVars:  map[string]string{"TEST_VAR": "actual"},
			expected: "actual",
		},
		{
			name:     "variable with default - variable missing",
			input:    "${MISSING_VAR:default}",
			envVars:  map[string]string{},
			expected: "default",
		},
		{
			name:     "multiple variables in string",
			input:    "http://${HOST}:${PORT}/path",
			envVars:  map[string]string{"HOST": "localhost", "PORT": "8080"},
			expected: "http://localhost:8080/path",
		},
		{
			name:     "variable with empty default",
			input:    "${EMPTY:}",
			envVars:  map[string]string{},
			expected: "",
		},
		{
			name:     "no variables",
			input:    "plain text",
			envVars:  map[string]string{},
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := SubstituteEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("SubstituteEnvVars() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		value    string
		expected string
	}{
		{
			name:     "SAGE_ENV set",
			envVar:   "SAGE_ENV",
			value:    "production",
			expected: "production",
		},
		{
			name:     "ENVIRONMENT set",
			envVar:   "ENVIRONMENT",
			value:    "staging",
			expected: "staging",
		},
		{
			name:     "no env var - defaults to development",
			envVar:   "",
			value:    "",
			expected: "development",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear both env vars
			os.Unsetenv("SAGE_ENV")
			os.Unsetenv("ENVIRONMENT")

			if tt.envVar != "" {
				os.Setenv(tt.envVar, tt.value)
				defer os.Unsetenv(tt.envVar)
			}

			result := GetEnvironment()
			if result != tt.expected {
				t.Errorf("GetEnvironment() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsProduction(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected bool
	}{
		{"production environment", "production", true},
		{"development environment", "development", false},
		{"staging environment", "staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SAGE_ENV", tt.env)
			defer os.Unsetenv("SAGE_ENV")

			result := IsProduction()
			if result != tt.expected {
				t.Errorf("IsProduction() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsDevelopment(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected bool
	}{
		{"development environment", "development", true},
		{"local environment", "local", true},
		{"production environment", "production", false},
		{"staging environment", "staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SAGE_ENV", tt.env)
			defer os.Unsetenv("SAGE_ENV")

			result := IsDevelopment()
			if result != tt.expected {
				t.Errorf("IsDevelopment() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSubstituteEnvVarsInConfig(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_RPC", "http://test-rpc:8545")
	os.Setenv("TEST_ADDR", "0x1234567890abcdef")
	defer os.Unsetenv("TEST_RPC")
	defer os.Unsetenv("TEST_ADDR")

	cfg := &Config{
		Blockchain: &BlockchainConfig{
			NetworkRPC:   "${TEST_RPC}",
			ContractAddr: "${TEST_ADDR}",
		},
		KeyStore: &KeyStoreConfig{
			Directory: "${HOME}/.sage/keys",
		},
	}

	SubstituteEnvVarsInConfig(cfg)

	if cfg.Blockchain.NetworkRPC != "http://test-rpc:8545" {
		t.Errorf("NetworkRPC = %q, want %q", cfg.Blockchain.NetworkRPC, "http://test-rpc:8545")
	}

	if cfg.Blockchain.ContractAddr != "0x1234567890abcdef" {
		t.Errorf("ContractAddr = %q, want %q", cfg.Blockchain.ContractAddr, "0x1234567890abcdef")
	}
}
