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
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		envVars map[string]string
		verify  func(*testing.T, *BlockchainConfig)
	}{
		{
			name: "Load local preset",
			env:  "local",
			verify: func(t *testing.T, cfg *BlockchainConfig) {
				assert.Contains(t, cfg.NetworkRPC, "localhost")
				assert.NotNil(t, cfg.ChainID)
				assert.Greater(t, cfg.GasLimit, uint64(0))
			},
		},
		{
			name: "Load mainnet preset",
			env:  "mainnet",
			verify: func(t *testing.T, cfg *BlockchainConfig) {
				assert.NotEmpty(t, cfg.NetworkRPC)
				assert.NotNil(t, cfg.ChainID)
			},
		},
		{
			name: "Load sepolia preset",
			env:  "sepolia",
			verify: func(t *testing.T, cfg *BlockchainConfig) {
				// Sepolia may not be in presets, or may have different chainID
				// Just verify config is valid
				assert.NotNil(t, cfg.ChainID)
				assert.NotEmpty(t, cfg.NetworkRPC)
			},
		},
		{
			name: "Override with environment variables",
			env:  "local",
			envVars: map[string]string{
				"SAGE_RPC_URL":     "http://custom-rpc:8545",
				"SAGE_CHAIN_ID":    "999",
				"SAGE_GAS_LIMIT":   "5000000",
				"SAGE_MAX_RETRIES": "5",
			},
			verify: func(t *testing.T, cfg *BlockchainConfig) {
				assert.Equal(t, "http://custom-rpc:8545", cfg.NetworkRPC)
				assert.Equal(t, int64(999), cfg.ChainID.Int64())
				assert.Equal(t, uint64(5000000), cfg.GasLimit)
				assert.Equal(t, 5, cfg.MaxRetries)
			},
		},
		{
			name: "Registry address priority",
			env:  "local",
			envVars: map[string]string{
				"SAGE_REGISTRY_ADDRESS": "0x1111111111111111111111111111111111111111",
				"SAGE_CONTRACT_ADDRESS": "0x2222222222222222222222222222222222222222",
			},
			verify: func(t *testing.T, cfg *BlockchainConfig) {
				// Should use SAGE_REGISTRY_ADDRESS over SAGE_CONTRACT_ADDRESS
				assert.Equal(t, "0x1111111111111111111111111111111111111111", cfg.ContractAddr)
			},
		},
		{
			name: "Fallback to contract address",
			env:  "local",
			envVars: map[string]string{
				"SAGE_CONTRACT_ADDRESS": "0x2222222222222222222222222222222222222222",
			},
			verify: func(t *testing.T, cfg *BlockchainConfig) {
				assert.Equal(t, "0x2222222222222222222222222222222222222222", cfg.ContractAddr)
			},
		},
		{
			name: "Max gas price override",
			env:  "local",
			envVars: map[string]string{
				"SAGE_MAX_GAS_PRICE": "100000000000",
			},
			verify: func(t *testing.T, cfg *BlockchainConfig) {
				expected := new(big.Int)
				expected.SetString("100000000000", 10)
				assert.Equal(t, 0, cfg.MaxGasPrice.Cmp(expected))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}
			defer func() {
				// Clean up
				for k := range tt.envVars {
					_ = os.Unsetenv(k)
				}
			}()

			cfg, err := LoadConfig(tt.env)
			require.NoError(t, err)
			require.NotNil(t, cfg)

			tt.verify(t, cfg)
		})
	}
}

func TestLoadConfigInvalidInputs(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr string
	}{
		{
			name: "Invalid chain ID",
			envVars: map[string]string{
				"SAGE_CHAIN_ID": "not-a-number",
			},
			wantErr: "invalid chain ID",
		},
		{
			name: "Invalid gas limit",
			envVars: map[string]string{
				"SAGE_GAS_LIMIT": "not-a-number",
			},
			wantErr: "invalid gas limit",
		},
		{
			name: "Invalid max gas price",
			envVars: map[string]string{
				"SAGE_MAX_GAS_PRICE": "not-a-number",
			},
			wantErr: "invalid max gas price",
		},
		{
			name: "Invalid max retries",
			envVars: map[string]string{
				"SAGE_MAX_RETRIES": "not-a-number",
			},
			wantErr: "invalid max retries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}
			defer func() {
				// Clean up
				for k := range tt.envVars {
					_ = os.Unsetenv(k)
				}
			}()

			_, err := LoadConfig("local")
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *BlockchainConfig
		wantErr bool
	}{
		{
			name: "Valid configuration",
			config: &BlockchainConfig{
				NetworkRPC:     "http://localhost:8545",
				ChainID:        big.NewInt(1337),
				GasLimit:       3000000,
				MaxGasPrice:    big.NewInt(100000000000),
				MaxRetries:     3,
				RetryDelay:     time.Second,
				RequestTimeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "Missing RPC endpoint",
			config: &BlockchainConfig{
				NetworkRPC: "",
				ChainID:    big.NewInt(1337),
				GasLimit:   3000000,
			},
			wantErr: true,
		},
		{
			name: "Invalid chain ID",
			config: &BlockchainConfig{
				NetworkRPC: "http://localhost:8545",
				ChainID:    big.NewInt(0),
				GasLimit:   3000000,
			},
			wantErr: true,
		},
		{
			name: "Invalid gas limit",
			config: &BlockchainConfig{
				NetworkRPC: "http://localhost:8545",
				ChainID:    big.NewInt(1337),
				GasLimit:   0,
			},
			wantErr: true,
		},
		{
			name: "Invalid max gas price",
			config: &BlockchainConfig{
				NetworkRPC:  "http://localhost:8545",
				ChainID:     big.NewInt(1337),
				GasLimit:    3000000,
				MaxGasPrice: big.NewInt(0),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetRetryConfig(t *testing.T) {
	config := &BlockchainConfig{
		MaxRetries: 5,
		RetryDelay: 2 * time.Second,
	}

	maxRetries, retryDelay := config.GetRetryConfig()
	assert.Equal(t, 5, maxRetries)
	assert.Equal(t, 2*time.Second, retryDelay)
}

func TestIsLocal(t *testing.T) {
	tests := []struct {
		name     string
		config   *BlockchainConfig
		expected bool
	}{
		{
			name: "Localhost RPC",
			config: &BlockchainConfig{
				NetworkRPC: "http://localhost:8545",
				ChainID:    big.NewInt(1337), // Initialize ChainID
			},
			expected: true,
		},
		{
			name: "127.0.0.1 RPC",
			config: &BlockchainConfig{
				NetworkRPC: "http://127.0.0.1:8545",
				ChainID:    big.NewInt(1337), // Initialize ChainID
			},
			expected: true,
		},
		{
			name: "Remote RPC",
			config: &BlockchainConfig{
				NetworkRPC: "https://mainnet.infura.io/v3/...",
				ChainID:    big.NewInt(1), // Initialize ChainID
			},
			expected: false,
		},
		{
			name: "Empty RPC",
			config: &BlockchainConfig{
				NetworkRPC: "",
				ChainID:    big.NewInt(1), // Initialize ChainID
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsLocal()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNetworkPresets(t *testing.T) {
	tests := []struct {
		name      string
		presetKey string
		verify    func(*testing.T, *BlockchainConfig)
	}{
		{
			name:      "Local preset exists",
			presetKey: "local",
			verify: func(t *testing.T, cfg *BlockchainConfig) {
				assert.NotNil(t, cfg)
				assert.NotNil(t, cfg.ChainID)
			},
		},
		{
			name:      "Sepolia preset may exist",
			presetKey: "sepolia",
			verify: func(t *testing.T, cfg *BlockchainConfig) {
				// Sepolia might not exist, skip if not present
				if cfg == nil {
					t.Skip("Sepolia preset not defined")
				}
				assert.NotNil(t, cfg.ChainID)
			},
		},
		{
			name:      "Mainnet preset exists",
			presetKey: "mainnet",
			verify: func(t *testing.T, cfg *BlockchainConfig) {
				assert.NotNil(t, cfg)
				assert.NotNil(t, cfg.ChainID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, exists := NetworkPresets[tt.presetKey]
			if !exists {
				t.Skipf("Preset %s not defined", tt.presetKey)
				return
			}
			tt.verify(t, cfg)
		})
	}
}

func TestLoadConfigDefaultsToLocal(t *testing.T) {
	// Use a non-existent environment
	cfg, err := LoadConfig("non-existent-env")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should default to local preset
	assert.Contains(t, cfg.NetworkRPC, "localhost")
	assert.NotNil(t, cfg.ChainID)
}

func TestConfigCopyDoesNotModifyPreset(t *testing.T) {
	// Get original preset
	originalPreset := NetworkPresets["local"]
	originalChainID := new(big.Int).Set(originalPreset.ChainID)

	// Set environment variable to override chain ID
	_ = os.Setenv("SAGE_CHAIN_ID", "999")
	defer func() { _ = os.Unsetenv("SAGE_CHAIN_ID") }()

	// Load config
	cfg, err := LoadConfig("local")
	require.NoError(t, err)

	// Verify the returned config has the override
	assert.Equal(t, int64(999), cfg.ChainID.Int64())

	// Verify the original preset was not modified
	assert.Equal(t, 0, originalPreset.ChainID.Cmp(originalChainID))
}
