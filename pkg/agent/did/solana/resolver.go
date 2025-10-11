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

package solana

import (
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// ResolvePublicKey retrieves only the public key for an agent
func (c *SolanaClient) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID) (crypto.PublicKey, error) {
	metadata, err := c.Resolve(ctx, agentDID)
	if err != nil {
		return nil, err
	}

	if !metadata.IsActive {
		return nil, did.ErrInactiveAgent
	}

	return metadata.PublicKey, nil
}

// VerifyMetadata checks if the provided metadata matches the on-chain data
func (c *SolanaClient) VerifyMetadata(ctx context.Context, agentDID did.AgentDID, metadata *did.AgentMetadata) (*did.VerificationResult, error) {
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

	// Compare capabilities
	capJSON1, _ := json.Marshal(metadata.Capabilities)
	capJSON2, _ := json.Marshal(onChainData.Capabilities)
	if string(capJSON1) != string(capJSON2) {
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
func (c *SolanaClient) ListAgentsByOwner(ctx context.Context, ownerAddress string) ([]*did.AgentMetadata, error) {
	// Validate address
	ownerPubkey, err := solana.PublicKeyFromBase58(ownerAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid Solana address: %s", ownerAddress)
	}

	// Get program accounts filtered by owner
	// This is a simplified approach - in production, you would:
	// 1. Use getProgramAccounts with proper filters
	// 2. Implement pagination for large datasets
	// 3. Use an indexer for better performance

	accounts, err := c.client.GetProgramAccountsWithOpts(
		ctx,
		c.programID,
		&rpc.GetProgramAccountsOpts{
			Commitment: rpc.CommitmentFinalized,
			Filters: []rpc.RPCFilter{
				{
					Memcmp: &rpc.RPCFilterMemcmp{
						Offset: 8, // Skip discriminator
						Bytes:  ownerPubkey[:],
					},
				},
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get program accounts: %w", err)
	}

	agents := make([]*did.AgentMetadata, 0, len(accounts))
	for _, account := range accounts {
		var agentAccount AgentAccount
		err := deserializeAccount(account.Account.Data.GetBinary(), &agentAccount)
		if err != nil {
			continue
		}

		agents = append(agents, &did.AgentMetadata{
			DID:          did.AgentDID(agentAccount.DID),
			Name:         agentAccount.Name,
			Description:  agentAccount.Description,
			Endpoint:     agentAccount.Endpoint,
			PublicKey:    agentAccount.PublicKey[:],
			Capabilities: agentAccount.Capabilities,
			Owner:        agentAccount.Owner.String(),
			IsActive:     agentAccount.IsActive,
			CreatedAt:    time.Unix(agentAccount.CreatedAt, 0),
			UpdatedAt:    time.Unix(agentAccount.UpdatedAt, 0),
		})
	}

	return agents, nil
}

// Search finds agents matching the given criteria
func (c *SolanaClient) Search(ctx context.Context, criteria did.SearchCriteria) ([]*did.AgentMetadata, error) {
	// Similar to Ethereum, this requires off-chain indexing for efficiency
	// Options include:
	// 1. Using a GraphQL API (Solana has several indexing services)
	// 2. Building a custom indexer using Geyser plugins
	// 3. Maintaining an off-chain database synced with on-chain data

	return nil, fmt.Errorf("search functionality requires off-chain indexing")
}

// GetRegistrationStatus checks the status of a registration transaction
func (c *SolanaClient) GetRegistrationStatus(ctx context.Context, txHash string) (*did.RegistrationResult, error) {
	sig, err := solana.SignatureFromBase58(txHash)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction hash: %w", err)
	}

	// Get transaction status
	status, err := c.client.GetSignatureStatuses(ctx, false, sig)
	if err != nil {
		return nil, fmt.Errorf("failed to get signature status: %w", err)
	}

	if status == nil || status.Value == nil || len(status.Value) == 0 {
		return nil, fmt.Errorf("transaction not found")
	}

	txStatus := status.Value[0]
	if txStatus.Err != nil {
		return nil, fmt.Errorf("transaction failed: %v", txStatus.Err)
	}

	// Get transaction details for more info
	tx, err := c.client.GetTransaction(
		ctx,
		sig,
		&rpc.GetTransactionOpts{
			Commitment: rpc.CommitmentFinalized,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	var timestamp time.Time
	if tx.BlockTime != nil {
		timestamp = tx.BlockTime.Time()
	} else {
		timestamp = time.Now()
	}

	return &did.RegistrationResult{
		TransactionHash: txHash,
		Slot:            txStatus.Slot,
		Timestamp:       timestamp,
	}, nil
}
