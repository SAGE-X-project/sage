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

package health

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

// CheckBlockchain checks the health of blockchain connection
func CheckBlockchain(rpcURL string) *BlockchainHealth {
	health := &BlockchainHealth{
		NetworkRPC: rpcURL,
		Connected:  false,
		Status:     StatusUnhealthy,
	}

	if rpcURL == "" {
		health.Error = "RPC URL not configured"
		return health
	}

	// Measure connection latency
	start := time.Now()

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		health.Error = fmt.Sprintf("Connection failed: %v", err)
		return health
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check chain ID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		health.Error = fmt.Sprintf("Failed to get chain ID: %v", err)
		return health
	}
	health.ChainID = chainID.String()

	// Check latest block
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		health.Error = fmt.Sprintf("Failed to get block number: %v", err)
		return health
	}
	health.BlockNumber = blockNumber

	latency := time.Since(start)
	health.Latency = latency.String()
	health.Connected = true

	// Determine status based on latency
	if latency < 1*time.Second {
		health.Status = StatusHealthy
	} else if latency < 3*time.Second {
		health.Status = StatusDegraded
	} else {
		health.Status = StatusUnhealthy
		health.Error = fmt.Sprintf("High latency: %v", latency)
	}

	return health
}
