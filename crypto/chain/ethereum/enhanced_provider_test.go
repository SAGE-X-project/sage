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


package ethereum

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sage-x-project/sage/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEthClient is a mock implementation of ethereum client for testing
type MockEthClient struct {
	networkID      *big.Int
	blockNumber    uint64
	gasEstimate    uint64
	gasPrice       *big.Int
	nonce          uint64
	receipt        *types.Receipt
	failCount      int
	maxFails       int
	shouldFail     bool
	failPermanent  bool
}

func (m *MockEthClient) NetworkID(ctx context.Context) (*big.Int, error) {
	if m.shouldFail && (m.failPermanent || m.failCount < m.maxFails) {
		m.failCount++
		return nil, errors.New("network error")
	}
	return m.networkID, nil
}

func (m *MockEthClient) BlockNumber(ctx context.Context) (uint64, error) {
	if m.shouldFail && (m.failPermanent || m.failCount < m.maxFails) {
		m.failCount++
		return 0, errors.New("network error")
	}
	return m.blockNumber, nil
}

func (m *MockEthClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	if m.shouldFail && (m.failPermanent || m.failCount < m.maxFails) {
		m.failCount++
		return 0, errors.New("gas estimation failed")
	}
	return m.gasEstimate, nil
}

func (m *MockEthClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	if m.shouldFail && (m.failPermanent || m.failCount < m.maxFails) {
		m.failCount++
		return nil, errors.New("gas price fetch failed")
	}
	return m.gasPrice, nil
}

func (m *MockEthClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	if m.shouldFail && (m.failPermanent || m.failCount < m.maxFails) {
		m.failCount++
		return 0, errors.New("nonce fetch failed")
	}
	return m.nonce, nil
}

func (m *MockEthClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	if m.receipt == nil {
		return nil, ethereum.NotFound
	}
	return m.receipt, nil
}

func (m *MockEthClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	return &types.Header{
		Number: big.NewInt(int64(m.blockNumber)),
	}, nil
}

func (m *MockEthClient) Close() {}

// Test retry logic with exponential backoff
func TestRetryWithBackoff(t *testing.T) {
	t.Run("Success on first try", func(t *testing.T) {
		callCount := 0
		err := retryWithBackoff(3, 10*time.Millisecond, func() error {
			callCount++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, callCount, "Function should be called only once on success")
	})

	t.Run("Success after retries", func(t *testing.T) {
		callCount := 0
		maxFails := 2

		err := retryWithBackoff(3, 10*time.Millisecond, func() error {
			callCount++
			if callCount <= maxFails {
				return errors.New("temporary error")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, maxFails+1, callCount, "Function should succeed after retries")
	})

	t.Run("Permanent failure", func(t *testing.T) {
		callCount := 0
		maxRetries := 3

		err := retryWithBackoff(maxRetries, 10*time.Millisecond, func() error {
			callCount++
			return errors.New("permanent error")
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "operation failed after 3 retries")
		assert.Equal(t, maxRetries+1, callCount, "Function should be called maxRetries+1 times")
	})

	t.Run("Exponential backoff timing", func(t *testing.T) {
		callCount := 0
		startTime := time.Now()
		baseDelay := 10 * time.Millisecond

		err := retryWithBackoff(2, baseDelay, func() error {
			callCount++
			return errors.New("error")
		})

		elapsed := time.Since(startTime)
		// First retry: 10ms, Second retry: 20ms = 30ms total minimum
		expectedMinDelay := 30 * time.Millisecond

		assert.Error(t, err)
		assert.GreaterOrEqual(t, elapsed, expectedMinDelay, "Should respect exponential backoff delays")
	})
}

// Test enhanced provider creation and configuration
func TestNewEnhancedProvider(t *testing.T) {
	t.Run("Valid configuration with mock", func(t *testing.T) {
		cfg := &config.BlockchainConfig{
			NetworkRPC:     "http://localhost:8545",
			ChainID:        big.NewInt(31337),
			GasLimit:       3000000,
			MaxGasPrice:    big.NewInt(20000000000),
			MaxRetries:     3,
			RetryDelay:     time.Millisecond * 10,
			RequestTimeout: 30 * time.Second,
		}

		mockClient := &MockEthClient{
			networkID: big.NewInt(31337),
		}

		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		assert.NoError(t, err)
		assert.NotNil(t, provider)
		assert.Equal(t, cfg, provider.GetConfig())
	})

	t.Run("Invalid configuration", func(t *testing.T) {
		testCases := []struct {
			name string
			cfg  *config.BlockchainConfig
		}{
			{
				name: "Missing RPC URL",
				cfg: &config.BlockchainConfig{
					ChainID:     big.NewInt(31337),
					GasLimit:    3000000,
					MaxGasPrice: big.NewInt(20000000000),
				},
			},
			{
				name: "Invalid Chain ID",
				cfg: &config.BlockchainConfig{
					NetworkRPC:  "http://localhost:8545",
					GasLimit:    3000000,
					MaxGasPrice: big.NewInt(20000000000),
				},
			},
			{
				name: "Zero Gas Limit",
				cfg: &config.BlockchainConfig{
					NetworkRPC:  "http://localhost:8545",
					ChainID:     big.NewInt(31337),
					MaxGasPrice: big.NewInt(20000000000),
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				_, err := NewEnhancedProviderWithClient(nil, tc.cfg)
				assert.Error(t, err, "Should fail with invalid configuration")
			})
		}
	})
}

// Test gas estimation with buffer and capping
func TestEstimateGas(t *testing.T) {
	cfg := &config.BlockchainConfig{
		NetworkRPC:     "http://localhost:8545",
		ChainID:        big.NewInt(31337),
		GasLimit:       3000000,
		MaxGasPrice:    big.NewInt(20000000000),
		MaxRetries:     3,
		RetryDelay:     time.Millisecond * 10,
		RequestTimeout: 30 * time.Second,
	}

	t.Run("Successful gas estimation with buffer", func(t *testing.T) {
		mockClient := &MockEthClient{
				gasEstimate: 100000,
		}
		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		require.NoError(t, err)

		ctx := context.Background()
		msg := ethereum.CallMsg{
			From: common.HexToAddress("0x0"),
			To:   &common.Address{},
		}

		gas, err := provider.EstimateGas(ctx, msg)

		assert.NoError(t, err)
		assert.Equal(t, uint64(120000), gas, "Should add 20% buffer to gas estimate")
	})

	t.Run("Gas limit capping", func(t *testing.T) {
		mockClient := &MockEthClient{
				gasEstimate: 3000000, // Will exceed limit with buffer
		}
		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		require.NoError(t, err)

		ctx := context.Background()
		msg := ethereum.CallMsg{}

		gas, err := provider.EstimateGas(ctx, msg)

		assert.NoError(t, err)
		assert.Equal(t, cfg.GasLimit, gas, "Should cap at configured gas limit")
	})

	t.Run("Retry on failure", func(t *testing.T) {
		mockClient := &MockEthClient{
			gasEstimate: 100000,
			shouldFail:  true,
			maxFails:    2,
		}

		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		require.NoError(t, err)

		ctx := context.Background()
		msg := ethereum.CallMsg{}

		gas, err := provider.EstimateGas(ctx, msg)

		assert.NoError(t, err)
		assert.Equal(t, uint64(120000), gas)
		assert.Equal(t, 2, mockClient.failCount, "Should retry on failure")
	})
}

// Test gas price suggestion with capping
func TestSuggestGasPrice(t *testing.T) {
	cfg := &config.BlockchainConfig{
		NetworkRPC:     "http://localhost:8545",
		ChainID:        big.NewInt(31337),
		GasLimit:       3000000,
		MaxGasPrice:    big.NewInt(50000000000), // 50 Gwei
		MaxRetries:     3,
		RetryDelay:     time.Millisecond * 10,
		RequestTimeout: 30 * time.Second,
	}

	t.Run("Normal gas price", func(t *testing.T) {
		suggestedPrice := big.NewInt(30000000000) // 30 Gwei
		mockClient := &MockEthClient{
				gasPrice: suggestedPrice,
		}
		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		require.NoError(t, err)

		ctx := context.Background()
		price, err := provider.SuggestGasPrice(ctx)

		assert.NoError(t, err)
		assert.Equal(t, suggestedPrice, price, "Should return suggested price when below max")
	})

	t.Run("Gas price capping", func(t *testing.T) {
		suggestedPrice := big.NewInt(100000000000) // 100 Gwei - exceeds max
		mockClient := &MockEthClient{
				gasPrice: suggestedPrice,
		}
		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		require.NoError(t, err)

		ctx := context.Background()
		price, err := provider.SuggestGasPrice(ctx)

		assert.NoError(t, err)
		assert.Equal(t, cfg.MaxGasPrice, price, "Should cap at max configured price")
	})
}

// Test health check functionality
func TestHealthCheck(t *testing.T) {
	cfg := &config.BlockchainConfig{
		NetworkRPC:     "http://localhost:8545",
		ChainID:        big.NewInt(31337),
		GasLimit:       3000000,
		MaxGasPrice:    big.NewInt(20000000000),
		MaxRetries:     3,
		RetryDelay:     time.Millisecond * 10,
		RequestTimeout: 30 * time.Second,
	}

	t.Run("Successful health check", func(t *testing.T) {
		mockClient := &MockEthClient{
				blockNumber: 12345,
		}
		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = provider.HealthCheck(ctx)

		assert.NoError(t, err, "Health check should succeed")
	})

	t.Run("Health check with retry", func(t *testing.T) {
		mockClient := &MockEthClient{
			blockNumber: 12345,
			shouldFail:  true,
			maxFails:    2,
		}

		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = provider.HealthCheck(ctx)

		assert.NoError(t, err, "Health check should succeed after retries")
		assert.Equal(t, 2, mockClient.failCount, "Should retry on failure")
	})

	t.Run("Health check failure", func(t *testing.T) {
		mockClient := &MockEthClient{
				shouldFail:    true,
				failPermanent: true,
		}
		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = provider.HealthCheck(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "health check failed")
	})
}

// Test transaction waiting with confirmations
func TestWaitForTransaction(t *testing.T) {
	cfg := &config.BlockchainConfig{
		NetworkRPC:     "http://localhost:8545",
		ChainID:        big.NewInt(31337),
		GasLimit:       3000000,
		MaxGasPrice:    big.NewInt(20000000000),
		MaxRetries:     3,
		RetryDelay:     time.Millisecond * 10,
		RequestTimeout: 30 * time.Second,
	}

	t.Run("Transaction found immediately", func(t *testing.T) {
		receipt := &types.Receipt{
			Status:      types.ReceiptStatusSuccessful,
			BlockNumber: big.NewInt(100),
		}

		mockClient := &MockEthClient{
				receipt:     receipt,
				blockNumber: 105,
		}
		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		require.NoError(t, err)

		ctx := context.Background()
		txHash := common.HexToHash("0x123")

		result, err := provider.WaitForTransaction(ctx, txHash, 0)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, receipt.Status, result.Status)
	})

	t.Run("Wait for confirmations", func(t *testing.T) {
		receipt := &types.Receipt{
			Status:      types.ReceiptStatusSuccessful,
			BlockNumber: big.NewInt(100),
		}

		mockClient := &MockEthClient{
			receipt:     receipt,
			blockNumber: 103, // 3 confirmations
		}

		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		require.NoError(t, err)

		ctx := context.Background()
		txHash := common.HexToHash("0x123")

		result, err := provider.WaitForTransaction(ctx, txHash, 3)

		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("Transaction not found timeout", func(t *testing.T) {
		mockClient := &MockEthClient{
				receipt: nil, // Transaction not mined
		}
		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		require.NoError(t, err)

		// Use a very short timeout for testing
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		txHash := common.HexToHash("0x123")

		result, err := provider.WaitForTransaction(ctx, txHash, 0)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "transaction wait timeout")
		assert.Nil(t, result)
	})
}

// Test ExecuteWithRetry wrapper
func TestExecuteWithRetry(t *testing.T) {
	cfg := &config.BlockchainConfig{
		NetworkRPC:     "http://localhost:8545",
		ChainID:        big.NewInt(31337),
		GasLimit:       3000000,
		MaxGasPrice:    big.NewInt(20000000000),
		MaxRetries:     3,
		RetryDelay:     time.Millisecond * 10,
		RequestTimeout: 30 * time.Second,
	}

	provider := &EnhancedProvider{
		client: &MockEthClient{},
		config: cfg,
	}

	t.Run("Successful execution", func(t *testing.T) {
		ctx := context.Background()
		callCount := 0

		err := provider.ExecuteWithRetry(ctx, func() error {
			callCount++
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, callCount)
	})

	t.Run("Retry and succeed", func(t *testing.T) {
		ctx := context.Background()
		callCount := 0

		err := provider.ExecuteWithRetry(ctx, func() error {
			callCount++
			if callCount < 3 {
				return errors.New("temporary error")
			}
			return nil
		})

		assert.NoError(t, err)
		assert.Equal(t, 3, callCount)
	})

	t.Run("Max retries exceeded", func(t *testing.T) {
		ctx := context.Background()
		callCount := 0

		err := provider.ExecuteWithRetry(ctx, func() error {
			callCount++
			return errors.New("permanent error")
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "operation failed after")
		assert.Equal(t, cfg.MaxRetries+1, callCount)
	})
}

// Test GetTransactionOpts
func TestGetTransactionOpts(t *testing.T) {
	cfg := &config.BlockchainConfig{
		NetworkRPC:     "http://localhost:8545",
		ChainID:        big.NewInt(31337),
		GasLimit:       3000000,
		MaxGasPrice:    big.NewInt(20000000000),
		MaxRetries:     3,
		RetryDelay:     time.Millisecond * 10,
		RequestTimeout: 30 * time.Second,
	}

	t.Run("Create transaction options", func(t *testing.T) {
		mockClient := &MockEthClient{
			nonce:    5,
			gasPrice: big.NewInt(15000000000),
		}

		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		require.NoError(t, err)

		ctx := context.Background()
		from := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80")

		opts, err := provider.GetTransactionOpts(ctx, from, nil)

		assert.NoError(t, err)
		assert.NotNil(t, opts)
		assert.Equal(t, from, opts.From)
		assert.Equal(t, big.NewInt(5), opts.Nonce)
		assert.Equal(t, cfg.GasLimit, opts.GasLimit)
		assert.Equal(t, mockClient.gasPrice, opts.GasPrice)
	})

	t.Run("Handle nonce fetch failure with retry", func(t *testing.T) {
		mockClient := &MockEthClient{
			nonce:      5,
			gasPrice:   big.NewInt(15000000000),
			shouldFail: true,
			maxFails:   2,
		}

		provider, err := NewEnhancedProviderWithClient(mockClient, cfg)
		require.NoError(t, err)

		ctx := context.Background()
		from := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f2bd80")

		opts, err := provider.GetTransactionOpts(ctx, from, nil)

		assert.NoError(t, err)
		assert.NotNil(t, opts)
		assert.Equal(t, big.NewInt(5), opts.Nonce)
	})
}
