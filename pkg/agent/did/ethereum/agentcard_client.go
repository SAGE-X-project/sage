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
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/blockchain/ethereum/contracts/agentcardregistry"
)

// AgentCardClient implements the three-phase registration flow for AgentCardRegistry
//
// THREE-PHASE REGISTRATION FLOW:
//
// Phase 1: COMMIT (anti-front-running)
//   - Generate salt and compute commitment hash
//   - Send commitment hash + 0.01 ETH stake to contract
//   - Wait 1-60 minutes before reveal
//
// Phase 2: REGISTER (reveal commitment)
//   - Send full registration params with salt
//   - Contract verifies commitment hash matches
//   - Agent registered but not active yet
//
// Phase 3: ACTIVATE (time-locked)
//   - Wait minimum 1 hour after registration
//   - Call activateAgent() to enable agent
//   - Stake is refunded after activation
//
// SECURITY FEATURES:
//   - Commit-reveal prevents front-running attacks
//   - Time delays prevent rapid spam/Sybil attacks
//   - Stake requirement adds economic security
//   - Hook system allows external verification
type AgentCardClient struct {
	client          *ethclient.Client
	contract        *agentcardregistry.AgentCardRegistry
	contractAddress common.Address
	privateKey      *ecdsa.PrivateKey
	chainID         *big.Int
	config          *did.RegistryConfig
}

// NewAgentCardClient creates a new AgentCardRegistry client
func NewAgentCardClient(config *did.RegistryConfig) (*AgentCardClient, error) {
	client, err := ethclient.Dial(config.RPCEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum node: %w", err)
	}

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get network ID: %w", err)
	}

	var privateKey *ecdsa.PrivateKey
	if config.PrivateKey != "" {
		privateKey, err = crypto.HexToECDSA(config.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid private key: %w", err)
		}
	}

	contractAddress := common.HexToAddress(config.ContractAddress)
	contract, err := agentcardregistry.NewAgentCardRegistry(contractAddress, client)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate contract: %w", err)
	}

	return &AgentCardClient{
		client:          client,
		contract:        contract,
		contractAddress: contractAddress,
		privateKey:      privateKey,
		chainID:         chainID,
		config:          config,
	}, nil
}

// CommitRegistration performs Phase 1: commit registration with hash
func (c *AgentCardClient) CommitRegistration(ctx context.Context, params *did.RegistrationParams) (*did.CommitmentStatus, error) {
	// Generate random salt
	if _, err := rand.Read(params.Salt[:]); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Compute commitment hash
	commitHash, err := c.computeCommitmentHash(params)
	if err != nil {
		return nil, fmt.Errorf("failed to compute commitment hash: %w", err)
	}

	// Get required stake amount
	stake, err := c.contract.RegistrationStake(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("failed to get stake amount: %w", err)
	}

	// Send commitment transaction with stake
	auth, err := c.getTransactor(ctx, stake)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	tx, err := c.contract.CommitRegistration(auth, commitHash)
	if err != nil {
		return nil, fmt.Errorf("failed to commit registration: %w", err)
	}

	// Wait for transaction confirmation
	receipt, err := bind.WaitMined(ctx, c.client, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for commitment: %w", err)
	}

	if receipt.Status != 1 {
		return nil, fmt.Errorf("commitment transaction failed")
	}

	// Return commitment status
	return &did.CommitmentStatus{
		Phase:           did.PhaseCommitted,
		CommitHash:      commitHash,
		CommitTimestamp: time.Now(),
		Params:          params,
	}, nil
}

// RegisterAgent performs Phase 2: reveal commitment and register
func (c *AgentCardClient) RegisterAgent(ctx context.Context, status *did.CommitmentStatus) (*did.CommitmentStatus, error) {
	if status.Phase != did.PhaseCommitted {
		return nil, fmt.Errorf("invalid phase: must be PhaseCommitted, got %v", status.Phase)
	}

	// Check minimum wait time (1 minute)
	minWait := time.Minute
	elapsed := time.Since(status.CommitTimestamp)
	if elapsed < minWait {
		return nil, fmt.Errorf("must wait at least 1 minute after commitment (waited %v)", elapsed)
	}

	// Check maximum wait time (60 minutes)
	maxWait := 60 * time.Minute
	if elapsed > maxWait {
		return nil, fmt.Errorf("commitment expired after 60 minutes (elapsed %v)", elapsed)
	}

	// Convert to contract params
	contractParams, err := c.toContractParams(status.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to convert params: %w", err)
	}

	// Send registration transaction
	auth, err := c.getTransactor(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	tx, err := c.contract.RegisterAgentWithParams(auth, *contractParams)
	if err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	// Wait for transaction confirmation
	receipt, err := bind.WaitMined(ctx, c.client, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for registration: %w", err)
	}

	if receipt.Status != 1 {
		return nil, fmt.Errorf("registration transaction failed")
	}

	// Extract agent ID from logs (TODO: parse AgentRegistered event)
	agentID := c.computeAgentID(status.Params.DID)

	// Get activation delay
	delay, err := c.contract.ActivationDelay(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("failed to get activation delay: %w", err)
	}

	// Update status
	status.Phase = did.PhaseRegistered
	status.AgentID = agentID
	status.CanActivateAt = time.Now().Add(time.Duration(delay.Uint64()) * time.Second)
	status.Params = nil // Clear params for security

	return status, nil
}

// ActivateAgent performs Phase 3: activate agent after time delay
func (c *AgentCardClient) ActivateAgent(ctx context.Context, status *did.CommitmentStatus) error {
	if status.Phase != did.PhaseRegistered {
		return fmt.Errorf("invalid phase: must be PhaseRegistered, got %v", status.Phase)
	}

	// Check if activation delay has passed
	if time.Now().Before(status.CanActivateAt) {
		waitTime := time.Until(status.CanActivateAt)
		return fmt.Errorf("must wait %v before activation (can activate at %v)", waitTime, status.CanActivateAt)
	}

	// Send activation transaction
	auth, err := c.getTransactor(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}

	tx, err := c.contract.ActivateAgent(auth, status.AgentID)
	if err != nil {
		return fmt.Errorf("failed to activate agent: %w", err)
	}

	// Wait for transaction confirmation
	receipt, err := bind.WaitMined(ctx, c.client, tx)
	if err != nil {
		return fmt.Errorf("failed to wait for activation: %w", err)
	}

	if receipt.Status != 1 {
		return fmt.Errorf("activation transaction failed")
	}

	// Update status
	status.Phase = did.PhaseActivated

	return nil
}

// GetCommitmentState queries on-chain commitment state
func (c *AgentCardClient) GetCommitmentState(ctx context.Context, owner common.Address) (*did.CommitmentState, error) {
	result, err := c.contract.RegistrationCommitments(&bind.CallOpts{Context: ctx}, owner)
	if err != nil {
		return nil, fmt.Errorf("failed to get commitment: %w", err)
	}

	return &did.CommitmentState{
		CommitHash: result.CommitHash,
		Timestamp:  time.Unix(result.Timestamp.Int64(), 0),
		Revealed:   result.Revealed,
	}, nil
}

// GetAgent retrieves agent metadata by agent ID
func (c *AgentCardClient) GetAgent(ctx context.Context, agentID [32]byte) (*did.AgentMetadataV4, error) {
	// TODO: Implement agent metadata retrieval
	return nil, fmt.Errorf("not implemented")
}

// GetAgentByDID retrieves agent metadata by DID string
func (c *AgentCardClient) GetAgentByDID(ctx context.Context, didStr string) (*did.AgentMetadataV4, error) {
	agentID := c.computeAgentID(didStr)
	return c.GetAgent(ctx, agentID)
}

// Helper functions

func (c *AgentCardClient) computeCommitmentHash(params *did.RegistrationParams) ([32]byte, error) {
	// TODO: Implement commitment hash calculation matching Solidity
	// keccak256(abi.encode(params, salt))
	return [32]byte{}, fmt.Errorf("not implemented")
}

func (c *AgentCardClient) computeAgentID(didStr string) [32]byte {
	return crypto.Keccak256Hash([]byte(didStr))
}

func (c *AgentCardClient) toContractParams(params *did.RegistrationParams) (*agentcardregistry.AgentCardStorageRegistrationParams, error) {
	// Convert KeyType slice to uint8 slice
	keyTypes := make([]uint8, len(params.KeyTypes))
	for i, kt := range params.KeyTypes {
		keyTypes[i] = uint8(kt)
	}

	return &agentcardregistry.AgentCardStorageRegistrationParams{
		Did:          params.DID,
		Name:         params.Name,
		Description:  params.Description,
		Endpoint:     params.Endpoint,
		Capabilities: params.Capabilities,
		Keys:         params.Keys,
		KeyTypes:     keyTypes,
		Signatures:   params.Signatures,
		Salt:         params.Salt,
	}, nil
}

func (c *AgentCardClient) getTransactor(ctx context.Context, value *big.Int) (*bind.TransactOpts, error) {
	if c.privateKey == nil {
		return nil, fmt.Errorf("no private key configured")
	}

	nonce, err := c.client.PendingNonceAt(ctx, crypto.PubkeyToAddress(c.privateKey.PublicKey))
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	gasPrice, err := c.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(c.privateKey, c.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = value
	auth.GasLimit = uint64(3000000) // TODO: Estimate gas
	auth.GasPrice = gasPrice
	auth.Context = ctx

	return auth, nil
}
