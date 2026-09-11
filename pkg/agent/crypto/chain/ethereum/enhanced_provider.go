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
	"github.com/sage-x-project/sage/deployments/config"
	blockchain "github.com/sage-x-project/sage/pkg/blockchain/ethereum"
)

// EthClient is the RPC client surface used by the provider.
//
// Deprecated: use github.com/sage-x-project/sage/pkg/blockchain/ethereum.EthClient.
type EthClient = blockchain.EthClient

// EnhancedProvider is the retrying EVM provider.
//
// Deprecated: use github.com/sage-x-project/sage/pkg/blockchain/ethereum.Provider.
type EnhancedProvider = blockchain.Provider

// NewEnhancedProvider dials the network described by cfg.
//
// Deprecated: use blockchain.NewProvider(cfg.Endpoint()).
func NewEnhancedProvider(cfg *config.BlockchainConfig) (*EnhancedProvider, error) {
	if cfg == nil {
		return nil, blockchain.Endpoint{}.Validate()
	}
	return blockchain.NewProvider(cfg.Endpoint())
}

// NewEnhancedProviderWithClient wraps an existing client.
//
// Deprecated: use blockchain.NewProviderWithClient(client, cfg.Endpoint()).
func NewEnhancedProviderWithClient(client EthClient, cfg *config.BlockchainConfig) (*EnhancedProvider, error) {
	if cfg == nil {
		return nil, blockchain.Endpoint{}.Validate()
	}
	return blockchain.NewProviderWithClient(client, cfg.Endpoint())
}
