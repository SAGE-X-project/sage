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
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// ResolvePublicKey retrieves only the public key for an agent
func (c *EthereumClient) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
	metadata, err := c.Resolve(ctx, agentDID)
	if err != nil {
		return nil, err
	}

	if !metadata.IsActive {
		return nil, did.ErrInactiveAgent
	}
	if metadata.PublicKey == nil {
		return nil, did.ErrNoSigningKey
	}

	return metadata.PublicKey, nil
}

// ResolvePublicKey retrieves only the public key for an agent
func (c *EthereumClient) ResolveKEMKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
	metadata, err := c.Resolve(ctx, agentDID)
	if err != nil {
		return nil, err
	}

	if !metadata.IsActive {
		return nil, did.ErrInactiveAgent
	}

	return metadata.PublicKEMKey, nil
}

// VerifyMetadata checks if the provided metadata matches the on-chain data
func (c *EthereumClient) VerifyMetadata(ctx context.Context, agentDID did.AgentDID, metadata *did.AgentMetadata) (*did.VerificationResult, error) {
	// Fetch on-chain data
	onChainData, err := c.Resolve(ctx, agentDID)
	if err != nil {
		if errors.Is(err, did.ErrDIDNotFound) {
			return &did.VerificationResult{
				Valid:      false,
				Error:      "DID not found on chain",
				VerifiedAt: time.Now(),
			}, nil
		}
		return nil, err
	}

	// Compare metadata
	valid := true
	var errorMsg string

	if metadata.Name != onChainData.Name {
		valid = false
		errorMsg = fmt.Sprintf("name mismatch: expected %s, got %s", onChainData.Name, metadata.Name)
	}

	if metadata.Description != onChainData.Description {
		valid = false
		if errorMsg != "" {
			errorMsg += "; "
		}
		errorMsg += "description mismatch"
	}

	if metadata.Endpoint != onChainData.Endpoint {
		valid = false
		if errorMsg != "" {
			errorMsg += "; "
		}
		errorMsg += fmt.Sprintf("endpoint mismatch: expected %s, got %s", onChainData.Endpoint, metadata.Endpoint)
	}

	// Compare capabilities (deep comparison)
	if !compareCapabilities(metadata.Capabilities, onChainData.Capabilities) {
		valid = false
		if errorMsg != "" {
			errorMsg += "; "
		}
		errorMsg += "capabilities mismatch"
	}

	result := &did.VerificationResult{
		Valid:      valid,
		Agent:      onChainData,
		VerifiedAt: time.Now(),
	}

	if !valid {
		result.Error = errorMsg
	}

	return result, nil
}

// ListAgentsByOwner retrieves all agents owned by a specific address
func (c *EthereumClient) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*did.AgentMetadata, error) {
	// Validate address
	if !common.IsHexAddress(ownerAddress) {
		return nil, fmt.Errorf("invalid Ethereum address: %s", ownerAddress)
	}

	owner := common.HexToAddress(ownerAddress)

	// Prepare call data
	callData, err := c.contractABI.Pack("getAgentsByOwner", owner)
	if err != nil {
		return nil, fmt.Errorf("failed to pack call data: %w", err)
	}

	// Make the call
	output, err := c.client.CallContract(ctx, ethereum.CallMsg{
		To:   &c.contractAddress,
		Data: callData,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	// Unpack the result
	var dids []string
	err = c.contractABI.UnpackIntoInterface(&dids, "getAgentsByOwner", output)
	if err != nil {
		return nil, fmt.Errorf("failed to get agents by owner: %w", err)
	}

	// Fetch metadata for each DID
	agents := make([]*did.AgentMetadata, 0, len(dids))
	for _, agentDID := range dids {
		metadata, err := c.Resolve(ctx, did.AgentDID(agentDID))
		if err != nil {
			// Skip failed resolutions
			continue
		}
		agents = append(agents, metadata)
	}

	return agents, nil
}

// Search finds agents matching the given criteria
func (c *EthereumClient) Search(ctx context.Context, criteria did.SearchCriteria) ([]*did.AgentMetadata, error) {
	// Note: This is a simplified implementation. In production, you would:
	// 1. Use events to build an index
	// 2. Query a graph protocol subgraph
	// 3. Use a separate indexing service

	// For now, we'll return an error indicating this needs off-chain indexing
	return nil, fmt.Errorf("search functionality requires off-chain indexing")
}

// GetRegistrationStatus checks the status of a registration transaction
func (c *EthereumClient) GetRegistrationStatus(ctx context.Context, txHash string) (*did.RegistrationResult, error) {
	hash := common.HexToHash(txHash)

	// Get transaction receipt
	receipt, err := c.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// Check if transaction failed
	if receipt.Status == 0 {
		return nil, fmt.Errorf("transaction failed")
	}

	// Get block for timestamp
	block, err := c.client.BlockByNumber(ctx, receipt.BlockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get block: %w", err)
	}

	blockTime := block.Time()
	const maxInt64 = 1<<63 - 1
	if blockTime > maxInt64 {
		return nil, fmt.Errorf("block timestamp overflow: %d exceeds maximum int64 value", blockTime)
	}

	return &did.RegistrationResult{
		TransactionHash: txHash,
		BlockNumber:     receipt.BlockNumber.Uint64(),
		Timestamp:       time.Unix(int64(blockTime), 0), // #nosec G115 - overflow checked above
		GasUsed:         receipt.GasUsed,
	}, nil
}

// Helper function to compare capabilities
func compareCapabilities(cap1, cap2 map[string]interface{}) bool {
	if len(cap1) != len(cap2) {
		return false
	}

	// Marshal both to JSON for deep comparison
	json1, err1 := json.Marshal(cap1)
	json2, err2 := json.Marshal(cap2)

	if err1 != nil || err2 != nil {
		return false
	}

	return strings.EqualFold(string(json1), string(json2))
}
