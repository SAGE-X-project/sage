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
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	agentID, regTs, err := c.extractRegisteredIDAndTs(receipt)

	// Get activation delay
	delay, err := c.contract.ActivationDelay(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, fmt.Errorf("failed to get activation delay: %w", err)
	}

	// Update status
	status.Phase = did.PhaseRegistered
	status.AgentID = agentID
	// Safe conversion: delay is typically small (e.g., 3600 seconds = 1 hour)
	// Max int64 = 9,223,372,036,854,775,807 seconds ≈ 292 billion years
	if delay.Cmp(big.NewInt(math.MaxInt64)) > 0 {
		return nil, fmt.Errorf("activation delay too large: %s seconds", delay.String())
	}
	status.CanActivateAt = time.Unix(regTs.Int64()+delay.Int64(), 0)
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

func (c *AgentCardClient) GetActivationDelay(ctx context.Context) (time.Duration, error) {
	d, err := c.contract.ActivationDelay(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, fmt.Errorf("ActivationDelay(): %w", err)
	}
	if d.Cmp(big.NewInt(math.MaxInt64)) > 0 {
		return 0, fmt.Errorf("activation delay too large: %s", d.String())
	}
	return time.Duration(d.Int64()) * time.Second, nil
}

func (c *AgentCardClient) SetActivationDelay(ctx context.Context, secs uint64) error {
	auth, err := c.getTransactor(ctx, nil)
	if err != nil {
		return fmt.Errorf("tx signer: %w", err)
	}
	tx, err := c.contract.SetActivationDelay(auth, new(big.Int).SetUint64(secs))
	if err != nil {
		return fmt.Errorf("SetActivationDelay call: %w", err)
	}
	if _, err := bind.WaitMined(ctx, c.client, tx); err != nil {
		return fmt.Errorf("wait mined: %w", err)
	}
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
	// Call contract to get agent metadata
	metadata, err := c.contract.GetAgent(&bind.CallOpts{Context: ctx}, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Check if agent exists (DID not empty)
	if metadata.Did == "" {
		return nil, fmt.Errorf("agent not found")
	}

	// Convert contract metadata to AgentMetadataV4
	agent := &did.AgentMetadataV4{
		DID:          did.AgentDID(metadata.Did),
		Name:         metadata.Name,
		Description:  metadata.Description,
		Endpoint:     metadata.Endpoint,
		Keys:         make([]did.AgentKey, 0, len(metadata.KeyHashes)),
		Capabilities: make(map[string]interface{}),
		Owner:        metadata.Owner.Hex(),
		IsActive:     metadata.Active,
		CreatedAt:    time.Unix(metadata.RegisteredAt.Int64(), 0),
		UpdatedAt:    time.Unix(metadata.UpdatedAt.Int64(), 0),
		PublicKEMKey: metadata.KemPublicKey, // ✅ Populate KME public key
	}

	// Parse capabilities JSON
	if metadata.Capabilities != "" {
		var caps map[string]interface{}
		if err := json.Unmarshal([]byte(metadata.Capabilities), &caps); err == nil {
			agent.Capabilities = caps
		}
	}

	// Fetch keys for each key hash
	for _, keyHash := range metadata.KeyHashes {
		keyData, err := c.contract.GetKey(&bind.CallOpts{Context: ctx}, keyHash)
		if err != nil {
			continue // Skip keys that can't be fetched
		}

		agent.Keys = append(agent.Keys, did.AgentKey{
			Type:      did.KeyType(keyData.KeyType),
			KeyData:   keyData.KeyData,
			Signature: keyData.Signature,
			Verified:  keyData.Verified,
			CreatedAt: time.Unix(keyData.RegisteredAt.Int64(), 0),
		})
	}

	return agent, nil
}

// GetAgentByDID retrieves agent metadata by DID string
func (c *AgentCardClient) GetAgentByDID(ctx context.Context, didStr string) (*did.AgentMetadataV4, error) {
	// 1) Call on-chain view
	md, err := c.contract.GetAgentByDID(&bind.CallOpts{Context: ctx}, didStr)
	if err != nil {
		return nil, fmt.Errorf("getAgentByDID: %w", err)
	}
	// Check existence
	if md.Owner == (common.Address{}) || md.Did == "" {
		return nil, fmt.Errorf("agent not found")
	}

	// 2) Convert to local struct
	agent := &did.AgentMetadataV4{
		DID:          did.AgentDID(md.Did),
		Name:         md.Name,
		Description:  md.Description,
		Endpoint:     md.Endpoint,
		Keys:         make([]did.AgentKey, 0, len(md.KeyHashes)),
		Capabilities: map[string]interface{}{},
		Owner:        md.Owner.Hex(),
		IsActive:     md.Active,
		CreatedAt:    time.Unix(md.RegisteredAt.Int64(), 0),
		UpdatedAt:    time.Unix(md.UpdatedAt.Int64(), 0),
		PublicKEMKey: md.KemPublicKey, // raw 32-byte X25519 key
	}

	// 3) Parse capabilities JSON if present
	if s := strings.TrimSpace(md.Capabilities); s != "" {
		var caps map[string]interface{}
		if err := json.Unmarshal([]byte(s), &caps); err == nil {
			agent.Capabilities = caps
		}
	}

	// 4) Resolve each keyHash with getKey and append to result
	for _, kh := range md.KeyHashes {
		k, err := c.contract.GetKey(&bind.CallOpts{Context: ctx}, kh)
		if err != nil {
			// Skip keys that cannot be fetched
			continue
		}
		agent.Keys = append(agent.Keys, did.AgentKey{
			Type:      did.KeyType(k.KeyType),
			KeyData:   k.KeyData,
			Signature: k.Signature,
			Verified:  k.Verified,
			CreatedAt: time.Unix(k.RegisteredAt.Int64(), 0),
		})
	}

	return agent, nil
}

// SetApprovalForAgent approves or revokes an operator for the agent
func (c *AgentCardClient) SetApprovalForAgent(ctx context.Context, agentID [32]byte, operator common.Address, approved bool) error {
	auth, err := c.getTransactor(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}

	tx, err := c.contract.SetApprovalForAgent(auth, agentID, operator, approved)
	if err != nil {
		return fmt.Errorf("failed to set approval: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, c.client, tx)
	if err != nil {
		return fmt.Errorf("failed to wait for approval: %w", err)
	}

	if receipt.Status != 1 {
		return fmt.Errorf("approval transaction failed")
	}

	return nil
}

// IsApprovedOperator checks if an address is an approved operator for the agent
func (c *AgentCardClient) IsApprovedOperator(ctx context.Context, agentID [32]byte, operator common.Address) (bool, error) {
	return c.contract.IsApprovedOperator(&bind.CallOpts{Context: ctx}, agentID, operator)
}

// UpdateAgent updates agent endpoint and capabilities
func (c *AgentCardClient) UpdateAgent(ctx context.Context, agentID [32]byte, endpoint, capabilities string) error {
	auth, err := c.getTransactor(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}

	tx, err := c.contract.UpdateAgent(auth, agentID, endpoint, capabilities)
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, c.client, tx)
	if err != nil {
		return fmt.Errorf("failed to wait for update: %w", err)
	}

	if receipt.Status != 1 {
		return fmt.Errorf("update transaction failed")
	}

	return nil
}

// DeactivateAgent deactivates an agent
func (c *AgentCardClient) DeactivateAgent(ctx context.Context, agentID [32]byte) error {
	auth, err := c.getTransactor(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}

	tx, err := c.contract.DeactivateAgentByHash(auth, agentID)
	if err != nil {
		return fmt.Errorf("failed to deactivate agent: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, c.client, tx)
	if err != nil {
		return fmt.Errorf("failed to wait for deactivation: %w", err)
	}

	if receipt.Status != 1 {
		return fmt.Errorf("deactivation transaction failed")
	}

	return nil
}

// GetKEMKey retrieves the KME (Key Management Encryption) public key for an agent
// Returns the X25519 public key used for HPKE (RFC 9180) encryption
func (c *AgentCardClient) GetKEMKey(ctx context.Context, agentID [32]byte) ([]byte, error) {
	KEMKey, err := c.contract.GetKEMKey(&bind.CallOpts{Context: ctx}, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get KME key: %w", err)
	}

	// Empty key means agent doesn't have X25519 key registered
	if len(KEMKey) == 0 {
		return nil, fmt.Errorf("agent does not have KME key registered")
	}

	return KEMKey, nil
}

// UpdateKEMKey updates the KME public key for an agent
// The new key must be 32 bytes (X25519 public key)
// Requires ECDSA signature for ownership proof
func (c *AgentCardClient) UpdateKEMKey(ctx context.Context, agentID [32]byte, newKEMKey []byte, signature []byte) error {
	if len(newKEMKey) != 32 {
		return fmt.Errorf("invalid KME key length: expected 32 bytes, got %d", len(newKEMKey))
	}

	if len(signature) != 65 {
		return fmt.Errorf("invalid signature length: expected 65 bytes, got %d", len(signature))
	}

	auth, err := c.getTransactor(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create transactor: %w", err)
	}

	tx, err := c.contract.UpdateKEMKey(auth, agentID, newKEMKey, signature)
	if err != nil {
		return fmt.Errorf("failed to update KME key: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, c.client, tx)
	if err != nil {
		return fmt.Errorf("failed to wait for KME key update: %w", err)
	}

	if receipt.Status != 1 {
		return fmt.Errorf("KME key update transaction failed")
	}

	return nil
}

// Helper functions

func (c *AgentCardClient) computeCommitmentHash(params *did.RegistrationParams) ([32]byte, error) {
	// Must match Solidity: keccak256(abi.encode(did, keys, owner, salt, chainId))

	owner := crypto.PubkeyToAddress(c.privateKey.PublicKey)

	// Encode parameters using Ethereum ABI encoding
	// This must exactly match the Solidity abi.encode() output

	// Create ABI types
	stringTy, _ := abi.NewType("string", "", nil)
	bytesTy, _ := abi.NewType("bytes[]", "", nil)
	addressTy, _ := abi.NewType("address", "", nil)
	bytes32Ty, _ := abi.NewType("bytes32", "", nil)
	uint256Ty, _ := abi.NewType("uint256", "", nil)

	arguments := abi.Arguments{
		{Type: stringTy},  // did
		{Type: bytesTy},   // keys
		{Type: addressTy}, // owner
		{Type: bytes32Ty}, // salt
		{Type: uint256Ty}, // chainId
	}

	// Encode
	encoded, err := arguments.Pack(
		params.DID,
		params.Keys,
		owner,
		params.Salt,
		c.chainID,
	)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to encode commitment: %w", err)
	}

	// Hash
	hash := crypto.Keccak256Hash(encoded)
	return hash, nil
}

func (c *AgentCardClient) computeAgentID(didStr string) [32]byte {
	return crypto.Keccak256Hash([]byte(didStr))
}

func (c *AgentCardClient) toContractParams(params *did.RegistrationParams) (*agentcardregistry.AgentCardStorageRegistrationParams, error) {
	// Convert KeyType slice to uint8 slice
	keyTypes := make([]uint8, len(params.KeyTypes))
	for i, kt := range params.KeyTypes {
		// Validate KeyType is within valid uint8 range
		if kt < 0 || kt > 255 {
			return nil, fmt.Errorf("invalid KeyType value: %d (must be 0-255)", kt)
		}
		keyTypes[i] = uint8(kt) // #nosec G115 - validated above
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

	// Safe conversion: nonce is always < math.MaxInt64 in practice
	if nonce > math.MaxInt64 {
		return nil, fmt.Errorf("nonce too large: %d", nonce)
	}
	auth.Nonce = big.NewInt(int64(nonce)) // #nosec G115 - validated above
	auth.Value = value
	auth.GasLimit = uint64(3000000) // TODO: Estimate gas
	auth.GasPrice = gasPrice
	auth.Context = ctx

	return auth, nil
}

func (c *AgentCardClient) extractRegisteredIDAndTs(receipt *types.Receipt) ([32]byte, *big.Int, error) {
	for _, lg := range receipt.Logs {
		if lg.Address != c.contractAddress {
			continue
		}
		ev, err := c.contract.ParseAgentRegistered(*lg) // event AgentRegistered(bytes32 agentId, string did, address owner, uint256 timestamp)
		if err == nil {
			return ev.AgentId, ev.Timestamp, nil
		}
	}
	return [32]byte{}, nil, fmt.Errorf("AgentRegistered event not found")
}
