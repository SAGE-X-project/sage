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
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/blockchain/ethereum/contracts/registryv4"
)

// EthereumClientV4 implements V4 DID registry operations for Ethereum with multi-key support
type EthereumClientV4 struct {
	client          *ethclient.Client
	contract        *registryv4.SageRegistryV4
	contractAddress common.Address
	privateKey      *ecdsa.PrivateKey
	chainID         *big.Int
	config          *did.RegistryConfig
}

// NewEthereumClientV4 creates a new Ethereum V4 DID client with multi-key support
func NewEthereumClientV4(config *did.RegistryConfig) (*EthereumClientV4, error) {
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

	// Create V4 contract instance
	contract, err := registryv4.NewSageRegistryV4(contractAddress, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract instance: %w", err)
	}

	return &EthereumClientV4{
		client:          client,
		contract:        contract,
		contractAddress: contractAddress,
		privateKey:      privateKey,
		chainID:         chainID,
		config:          config,
	}, nil
}

// Register registers an agent (supports both single and multi-key)
func (c *EthereumClientV4) Register(ctx context.Context, req *did.RegistrationRequest) (*did.RegistrationResult, error) {
	// Determine keys to register
	var keys []did.AgentKey
	if len(req.Keys) > 0 {
		// Multi-key registration
		keys = req.Keys
	} else if req.KeyPair != nil {
		// Single-key registration (backward compatibility)
		pubKeyBytes, err := did.MarshalPublicKey(req.KeyPair.PublicKey())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal public key: %w", err)
		}
		keys = []did.AgentKey{
			{
				Type:    did.KeyTypeECDSA, // Default to ECDSA for Ethereum
				KeyData: pubKeyBytes,
			},
		}
	} else {
		return nil, fmt.Errorf("no keys provided for registration")
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("at least one key required")
	}
	if len(keys) > 10 {
		return nil, fmt.Errorf("maximum 10 keys allowed")
	}

	// Prepare arrays for V4 registration
	keyTypes := make([]uint8, len(keys))
	keyData := make([][]byte, len(keys))
	signatures := make([][]byte, len(keys))

	// Prepare capabilities as JSON string
	capabilitiesJSON, err := json.Marshal(req.Capabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	// Prepare the message to sign for each key
	message := c.prepareRegistrationMessage(req, keys)
	messageHash := crypto.Keccak256([]byte(message))

	// Process each key
	for i, key := range keys {
		// Set key type
		keyTypes[i] = uint8(key.Type)
		keyData[i] = key.KeyData

		// Sign message with each key
		if len(key.Signature) > 0 {
			// Use pre-computed signature
			signatures[i] = key.Signature
		} else if req.KeyPair != nil && len(keys) == 1 {
			// Single-key mode: sign with the provided keypair
			sig, err := req.KeyPair.Sign(messageHash)
			if err != nil {
				return nil, fmt.Errorf("failed to sign with key %d: %w", i, err)
			}
			signatures[i] = sig
		} else {
			return nil, fmt.Errorf("key %d has no signature", i)
		}
	}

	// Prepare transaction options
	auth, err := c.getTransactOpts(ctx)
	if err != nil {
		return nil, err
	}

	// Build V4 registration params
	params := registryv4.ISageRegistryV4RegistrationParams{
		Did:          string(req.DID),
		Name:         req.Name,
		Description:  req.Description,
		Endpoint:     req.Endpoint,
		KeyTypes:     keyTypes,
		KeyData:      keyData,
		Signatures:   signatures,
		Capabilities: string(capabilitiesJSON),
	}

	// Call the contract
	tx, err := c.contract.RegisterAgent(auth, params)
	if err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	// Wait for transaction confirmation
	receipt, err := c.waitForTransaction(ctx, tx)
	if err != nil {
		return nil, err
	}

	return &did.RegistrationResult{
		TransactionHash: tx.Hash().Hex(),
		BlockNumber:     receipt.BlockNumber.Uint64(),
		Timestamp:       time.Now(),
		GasUsed:         receipt.GasUsed,
	}, nil
}

// Resolve retrieves agent metadata from V4 contract
func (c *EthereumClientV4) Resolve(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error) {
	// Call getAgentByDID
	agent, err := c.contract.GetAgentByDID(&bind.CallOpts{Context: ctx}, string(agentDID))
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Check if agent exists (empty DID means not found)
	if agent.Did == "" {
		return nil, did.ErrDIDNotFound
	}

	// Parse first public key for backward compatibility
	var publicKey interface{}
	if len(agent.KeyHashes) > 0 {
		// Get the first key
		keyHash := agent.KeyHashes[0]
		keyData, err := c.contract.GetKey(&bind.CallOpts{Context: ctx}, keyHash)
		if err == nil && len(keyData.KeyData) > 0 {
			// Try to unmarshal as secp256k1 first (most common on Ethereum)
			publicKey, _ = did.UnmarshalPublicKey(keyData.KeyData, "secp256k1")
		}
	}

	// Parse capabilities
	var capabilities map[string]interface{}
	if agent.Capabilities != "" {
		err = json.Unmarshal([]byte(agent.Capabilities), &capabilities)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
		}
	}

	return &did.AgentMetadata{
		DID:          agentDID,
		Name:         agent.Name,
		Description:  agent.Description,
		Endpoint:     agent.Endpoint,
		PublicKey:    publicKey,
		Capabilities: capabilities,
		Owner:        agent.Owner.Hex(),
		IsActive:     agent.Active,
		CreatedAt:    time.Unix(agent.RegisteredAt.Int64(), 0),
		UpdatedAt:    time.Unix(agent.UpdatedAt.Int64(), 0),
	}, nil
}

// ResolvePublicKey retrieves only the public key for an agent
func (c *EthereumClientV4) ResolvePublicKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
	agent, err := c.contract.GetAgentByDID(&bind.CallOpts{Context: ctx}, string(agentDID))
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if agent.Did == "" {
		return nil, did.ErrDIDNotFound
	}

	if len(agent.KeyHashes) == 0 {
		return nil, fmt.Errorf("agent has no keys")
	}

	// Get the first key
	keyHash := agent.KeyHashes[0]
	keyData, err := c.contract.GetKey(&bind.CallOpts{Context: ctx}, keyHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get key data: %w", err)
	}

	// Try to unmarshal based on key type
	keyTypeStr := "secp256k1" // Default
	if keyData.KeyType == uint8(did.KeyTypeEd25519) {
		keyTypeStr = "ed25519"
	}

	publicKey, err := did.UnmarshalPublicKey(keyData.KeyData, keyTypeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	return publicKey, nil
}

// Update updates agent metadata
func (c *EthereumClientV4) Update(ctx context.Context, agentDID did.AgentDID, updates map[string]interface{}, keyPair sagecrypto.KeyPair) error {
	// Prepare update message
	message := c.prepareUpdateMessage(agentDID, updates)
	messageHash := crypto.Keccak256([]byte(message))

	// Sign the message
	signature, err := keyPair.Sign(messageHash)
	if err != nil {
		return fmt.Errorf("failed to sign update: %w", err)
	}

	// Prepare transaction options
	auth, err := c.getTransactOpts(ctx)
	if err != nil {
		return err
	}

	// Extract update fields
	name, _ := updates["name"].(string)
	description, _ := updates["description"].(string)
	endpoint, _ := updates["endpoint"].(string)

	capabilitiesJSON := ""
	if capabilities, ok := updates["capabilities"]; ok {
		capBytes, err := json.Marshal(capabilities)
		if err != nil {
			return fmt.Errorf("failed to marshal capabilities: %w", err)
		}
		capabilitiesJSON = string(capBytes)
	}

	// Generate agentId from DID (keccak256 hash)
	agentId := crypto.Keccak256Hash([]byte(string(agentDID)))

	// Call the contract
	tx, err := c.contract.UpdateAgent(auth, agentId, name, description, endpoint, capabilitiesJSON, signature)
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	// Wait for confirmation
	_, err = c.waitForTransaction(ctx, tx)
	return err
}

// Deactivate deactivates an agent
func (c *EthereumClientV4) Deactivate(ctx context.Context, agentDID did.AgentDID, keyPair sagecrypto.KeyPair) error {
	// Prepare transaction options
	auth, err := c.getTransactOpts(ctx)
	if err != nil {
		return err
	}

	// Generate agentId from DID (keccak256 hash)
	agentId := crypto.Keccak256Hash([]byte(string(agentDID)))

	// Call the contract
	tx, err := c.contract.DeactivateAgent(auth, agentId)
	if err != nil {
		return fmt.Errorf("failed to deactivate agent: %w", err)
	}

	// Wait for confirmation
	_, err = c.waitForTransaction(ctx, tx)
	return err
}

// Helper methods

func (c *EthereumClientV4) getTransactOpts(ctx context.Context) (*bind.TransactOpts, error) {
	if c.privateKey == nil {
		return nil, fmt.Errorf("private key required for transactions")
	}

	auth, err := bind.NewKeyedTransactorWithChainID(c.privateKey, c.chainID)
	if err != nil {
		return nil, err
	}

	auth.Context = ctx

	// Set gas price if configured
	if c.config.GasPrice > 0 {
		const maxInt64 = 1<<63 - 1
		if c.config.GasPrice > maxInt64 {
			return nil, fmt.Errorf("gas price overflow: %d exceeds maximum int64 value", c.config.GasPrice)
		}
		auth.GasPrice = big.NewInt(int64(c.config.GasPrice)) // #nosec G115 - overflow checked above
	}

	return auth, nil
}

func (c *EthereumClientV4) waitForTransaction(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
	// Wait for transaction to be mined
	for i := 0; i < c.config.MaxRetries; i++ {
		receipt, err := c.client.TransactionReceipt(ctx, tx.Hash())
		if err == nil {
			if receipt.Status == types.ReceiptStatusFailed {
				return nil, fmt.Errorf("transaction failed")
			}

			// Wait for confirmations
			if c.config.ConfirmationBlocks > 0 {
				currentBlock, err := c.client.BlockNumber(ctx)
				if err != nil {
					return nil, err
				}

				if c.config.ConfirmationBlocks < 0 {
					return nil, fmt.Errorf("confirmation blocks must be non-negative: %d", c.config.ConfirmationBlocks)
				}
				confirmations := currentBlock - receipt.BlockNumber.Uint64()
				if confirmations < uint64(c.config.ConfirmationBlocks) { // #nosec G115 - negative values checked above
					time.Sleep(5 * time.Second)
					continue
				}
			}

			return receipt, nil
		}

		time.Sleep(5 * time.Second)
	}

	return nil, fmt.Errorf("transaction timeout")
}

func (c *EthereumClientV4) prepareRegistrationMessage(req *did.RegistrationRequest, keys []did.AgentKey) string {
	return fmt.Sprintf("Register agent:\nDID: %s\nName: %s\nEndpoint: %s\nKeys: %d",
		req.DID, req.Name, req.Endpoint, len(keys))
}

func (c *EthereumClientV4) prepareUpdateMessage(agentDID did.AgentDID, updates map[string]interface{}) string {
	return fmt.Sprintf("Update agent: %s\nUpdates: %v", agentDID, updates)
}
