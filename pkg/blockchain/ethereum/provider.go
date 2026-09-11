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

// Package ethereum holds the EVM RPC provider used by the DID and contract
// layers: connection with retries, gas estimation and capping, transaction
// options and receipt polling. It is configured with an Endpoint and knows
// nothing about deployment files or network presets.
package ethereum

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// EthClient defines the interface we need from ethereum client
type EthClient interface {
	NetworkID(ctx context.Context) (*big.Int, error)
	BlockNumber(ctx context.Context) (uint64, error)
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	Close()
}

// Endpoint describes how to reach one EVM network and the limits applied to
// transactions sent through it. It replaces the deployment-level
// BlockchainConfig at this layer so the provider depends only on what it uses.
type Endpoint struct {
	RPCURL         string
	ChainID        *big.Int
	GasLimit       uint64
	MaxGasPrice    *big.Int
	MaxRetries     int
	RetryDelay     time.Duration
	RequestTimeout time.Duration
}

// Validate reports the first missing or invalid field.
func (e Endpoint) Validate() error {
	if e.RPCURL == "" {
		return fmt.Errorf("network RPC URL is required")
	}
	if e.ChainID == nil || e.ChainID.Sign() <= 0 {
		return fmt.Errorf("valid chain ID is required")
	}
	if e.GasLimit == 0 {
		return fmt.Errorf("gas limit must be greater than 0")
	}
	if e.MaxGasPrice == nil || e.MaxGasPrice.Sign() <= 0 {
		return fmt.Errorf("valid max gas price is required")
	}
	if e.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	return nil
}

// Provider wraps an Ethereum RPC client with retry, gas estimation with a
// safety margin, gas-price capping and receipt polling.
type Provider struct {
	client   EthClient
	endpoint Endpoint
}

// NewProvider dials the endpoint (with retries), verifies the chain id and
// returns a Provider.
func NewProvider(cfg Endpoint) (*Provider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Connect with retry
	var client *ethclient.Client
	err := retryWithBackoff(cfg.MaxRetries, cfg.RetryDelay, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.RequestTimeout)
		defer cancel()

		c, err := ethclient.DialContext(ctx, cfg.RPCURL)
		if err != nil {
			return fmt.Errorf("failed to connect to network: %w", err)
		}

		// Verify chain ID
		chainID, err := c.NetworkID(ctx)
		if err != nil {
			c.Close()
			return fmt.Errorf("failed to get network ID: %w", err)
		}

		if chainID.Cmp(cfg.ChainID) != 0 {
			c.Close()
			return fmt.Errorf("chain ID mismatch: expected %s, got %s", cfg.ChainID, chainID)
		}

		client = c
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &Provider{client: client, endpoint: cfg}, nil
}

// NewProviderWithClient wraps an existing client (tests, custom transports).
func NewProviderWithClient(client EthClient, cfg Endpoint) (*Provider, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return &Provider{client: client, endpoint: cfg}, nil
}

// ExecuteWithRetry executes a function with retry logic
func (p *Provider) ExecuteWithRetry(
	ctx context.Context,
	fn func() error,
) error {
	return retryWithBackoff(p.endpoint.MaxRetries, p.endpoint.RetryDelay, fn)
}

// EstimateGas estimates gas for a transaction with safety margin
func (p *Provider) EstimateGas(
	ctx context.Context,
	msg ethereum.CallMsg,
) (uint64, error) {
	var gasEstimate uint64

	err := p.ExecuteWithRetry(ctx, func() error {
		estimate, err := p.client.EstimateGas(ctx, msg)
		if err != nil {
			return fmt.Errorf("gas estimation failed: %w", err)
		}

		// Add 20% buffer for safety
		gasEstimate = estimate + (estimate / 5)

		// Cap at configured limit
		if gasEstimate > p.endpoint.GasLimit {
			gasEstimate = p.endpoint.GasLimit
		}

		return nil
	})

	return gasEstimate, err
}

// SuggestGasPrice suggests an appropriate gas price with retry
func (p *Provider) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	var gasPrice *big.Int

	err := p.ExecuteWithRetry(ctx, func() error {
		suggested, err := p.client.SuggestGasPrice(ctx)
		if err != nil {
			return fmt.Errorf("failed to suggest gas price: %w", err)
		}

		// Cap at max configured price
		if suggested.Cmp(p.endpoint.MaxGasPrice) > 0 {
			gasPrice = new(big.Int).Set(p.endpoint.MaxGasPrice)
		} else {
			gasPrice = suggested
		}

		return nil
	})

	return gasPrice, err
}

// GetTransactionOpts creates transaction options with optimal gas settings
func (p *Provider) GetTransactionOpts(
	ctx context.Context,
	from common.Address,
	privateKey interface{}, // Can be *ecdsa.PrivateKey or other signer
) (*bind.TransactOpts, error) {
	// Get nonce with retry
	var nonce uint64
	err := p.ExecuteWithRetry(ctx, func() error {
		n, err := p.client.PendingNonceAt(ctx, from)
		if err != nil {
			return fmt.Errorf("failed to get nonce: %w", err)
		}
		nonce = n
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Get gas price
	gasPrice, err := p.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	const maxInt64 = 1<<63 - 1
	if nonce > maxInt64 {
		return nil, fmt.Errorf("nonce overflow: %d exceeds maximum int64 value", nonce)
	}

	opts := &bind.TransactOpts{
		From:     from,
		Nonce:    big.NewInt(int64(nonce)), // #nosec G115 - overflow checked above
		GasLimit: p.endpoint.GasLimit,
		GasPrice: gasPrice,
		Context:  ctx,
	}

	// Note: Signer must be set by the caller based on private key type
	// The privateKey parameter is reserved for future implementation

	return opts, nil
}

// WaitForTransaction waits for a transaction to be mined with timeout
func (p *Provider) WaitForTransaction(
	ctx context.Context,
	txHash common.Hash,
	confirmations uint64,
) (*types.Receipt, error) {
	// Create a context with timeout
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			return nil, fmt.Errorf("transaction wait timeout: %w", waitCtx.Err())
		case <-ticker.C:
			var receipt *types.Receipt
			err := p.ExecuteWithRetry(ctx, func() error {
				r, err := p.client.TransactionReceipt(ctx, txHash)
				if err != nil {
					if err == ethereum.NotFound {
						return nil // Transaction not yet mined
					}
					return err
				}
				receipt = r
				return nil
			})

			if err != nil {
				return nil, err
			}

			if receipt != nil {
				// Check if we have enough confirmations
				if confirmations > 0 {
					var currentBlock uint64
					err := p.ExecuteWithRetry(ctx, func() error {
						header, err := p.client.HeaderByNumber(ctx, nil)
						if err != nil {
							return err
						}
						currentBlock = header.Number.Uint64()
						return nil
					})

					if err != nil {
						return nil, err
					}

					if currentBlock-receipt.BlockNumber.Uint64() >= confirmations {
						return receipt, nil
					}
				} else {
					return receipt, nil
				}
			}
		}
	}
}

// GetClient returns the underlying ethereum client for direct access
func (p *Provider) GetClient() EthClient {
	return p.client
}

// Endpoint returns the endpoint this provider was built for.
func (p *Provider) Endpoint() Endpoint {
	return p.endpoint
}

// Close closes the underlying client connection
func (p *Provider) Close() {
	if p.client != nil {
		p.client.Close()
	}
}

// retryWithBackoff implements exponential backoff retry logic
func retryWithBackoff(maxRetries int, baseDelay time.Duration, fn func() error) error {
	var lastErr error
	delay := baseDelay

	for i := 0; i <= maxRetries; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if i < maxRetries {
				time.Sleep(delay)
				delay *= 2 // Exponential backoff
				if delay > 30*time.Second {
					delay = 30 * time.Second // Cap at 30 seconds
				}
			}
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

// HealthCheck performs a health check on the blockchain connection
func (p *Provider) HealthCheck(ctx context.Context) error {
	return p.ExecuteWithRetry(ctx, func() error {
		// Try to get the latest block number
		_, err := p.client.BlockNumber(ctx)
		if err != nil {
			return fmt.Errorf("health check failed: %w", err)
		}
		return nil
	})
}
