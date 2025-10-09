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

package integration

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sage-x-project/sage/deployments/config"
	chaineth "github.com/sage-x-project/sage/pkg/agent/crypto/chain/ethereum"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test configuration for local blockchain
func getTestConfig() *config.BlockchainConfig {
	return &config.BlockchainConfig{
		NetworkRPC:     getEnvOrDefault("SAGE_RPC_URL", "http://localhost:8545"),
		ChainID:        big.NewInt(31337),
		GasLimit:       3000000,
		MaxGasPrice:    big.NewInt(20000000000), // 20 Gwei
		MaxRetries:     3,
		RetryDelay:     time.Second,
		RequestTimeout: 30 * time.Second,
	}
}

// TestBlockchainConnection tests basic connection to local blockchain
func TestBlockchainConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	RequireBlockchain(t) // Skip if blockchain not available
	cfg := getTestConfig()

	t.Run("Connect to local blockchain", func(t *testing.T) {
		client, err := ethclient.Dial(cfg.NetworkRPC)
		require.NoError(t, err)
		defer client.Close()

		// Check chain ID
		ctx := context.Background()
		chainID, err := client.ChainID(ctx)
		require.NoError(t, err)
		assert.Equal(t, cfg.ChainID, chainID)

		// Get latest block
		blockNumber, err := client.BlockNumber(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, blockNumber, uint64(0))

		t.Logf("Connected to chain ID: %s, latest block: %d", chainID, blockNumber)
	})
}

// TestEnhancedProviderIntegration tests enhanced provider with real blockchain
func TestEnhancedProviderIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	RequireBlockchain(t) // Skip if blockchain not available
	cfg := getTestConfig()

	t.Run("Create enhanced provider", func(t *testing.T) {
		provider, err := chaineth.NewEnhancedProvider(cfg)
		require.NoError(t, err)
		defer provider.Close()

		// Test health check
		ctx := context.Background()
		err = provider.HealthCheck(ctx)
		assert.NoError(t, err)
	})

	t.Run("Gas estimation", func(t *testing.T) {
		provider, err := chaineth.NewEnhancedProvider(cfg)
		require.NoError(t, err)
		defer provider.Close()

		ctx := context.Background()

		// Create a simple transaction
		from := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
		to := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")

		msg := ethereum.CallMsg{
			From:  from,
			To:    &to,
			Value: big.NewInt(1000000000000000), // 0.001 ETH
			Data:  nil,
		}

		// Estimate gas
		gasLimit, err := provider.EstimateGas(ctx, msg)
		require.NoError(t, err)
		assert.Greater(t, gasLimit, uint64(21000)) // Should be at least 21000 for simple transfer

		t.Logf("Estimated gas: %d", gasLimit)
	})

	t.Run("Gas price suggestion", func(t *testing.T) {
		provider, err := chaineth.NewEnhancedProvider(cfg)
		require.NoError(t, err)
		defer provider.Close()

		ctx := context.Background()

		gasPrice, err := provider.SuggestGasPrice(ctx)
		require.NoError(t, err)
		assert.NotNil(t, gasPrice)
		assert.True(t, gasPrice.Sign() > 0)

		t.Logf("Suggested gas price: %s Wei", gasPrice.String())
	})

	t.Run("Retry logic on failure", func(t *testing.T) {
		// Create provider with invalid RPC first
		invalidCfg := &config.BlockchainConfig{
			NetworkRPC:     "http://localhost:9999", // Invalid port
			ChainID:        big.NewInt(31337),
			GasLimit:       3000000,
			MaxGasPrice:    big.NewInt(20000000000),
			MaxRetries:     2,
			RetryDelay:     100 * time.Millisecond,
			RequestTimeout: 1 * time.Second,
		}

		_, err := chaineth.NewEnhancedProvider(invalidCfg)
		assert.Error(t, err)
	})
}

// TestAccountBalance tests getting account balance
func TestAccountBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	RequireBlockchain(t) // Skip if blockchain not available
	cfg := getTestConfig()
	client, err := ethclient.Dial(cfg.NetworkRPC)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// Test account (first account from test mnemonic)
	testAccount := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")

	balance, err := client.BalanceAt(ctx, testAccount, nil)
	require.NoError(t, err)
	assert.NotNil(t, balance)

	// Test accounts should have some balance
	assert.True(t, balance.Sign() > 0)

	t.Logf("Test account balance: %s Wei", balance.String())
}

// BenchmarkEnhancedProvider benchmarks provider operations
func BenchmarkEnhancedProvider(b *testing.B) {
	cfg := getTestConfig()
	provider, err := chaineth.NewEnhancedProvider(cfg)
	require.NoError(b, err)
	defer provider.Close()

	ctx := context.Background()

	b.Run("HealthCheck", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = provider.HealthCheck(ctx)
		}
	})

	b.Run("GasPrice", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = provider.SuggestGasPrice(ctx)
		}
	})
}
