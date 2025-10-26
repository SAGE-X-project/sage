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
	"crypto/ecdh"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/pkg/blockchain/ethereum/contracts/registryv4"
)

// init registers the Ethereum V4 client creator with the factory
func init() {
	did.RegisterEthereumV4ClientCreator(func(config *did.RegistryConfig) (interface{}, error) {
		return NewEthereumClientV4(config)
	})
}

// EthereumClientV4 implements V4 DID registry operations for Ethereum with multi-key support
//
// CHAIN-SPECIFIC SIGNATURE VERIFICATION DESIGN:
//
// This client implements Ethereum-specific key verification logic. Different blockchains
// have different native cryptographic primitives and verification costs:
//
// ETHEREUM:
//   - ECDSA (secp256k1): Uses ecrecover precompile for on-chain verification (REQUIRED signature)
//   - Ed25519: No native precompile, uses off-chain verification + owner approval (NO signature)
//
// SOLANA (future implementation):
//   - Ed25519: Native ed25519_verify instruction for on-chain verification (REQUIRED signature)
//   - ECDSA: No native support, would use off-chain approval (NO signature or NOT SUPPORTED)
//
// TENDERMINT/COSMOS (future implementation):
//   - Ed25519: Native verification (REQUIRED signature)
//   - secp256k1: Supported but more expensive than Ed25519 (REQUIRED signature)
//
// This design allows each chain to use its most efficient verification method while
// supporting cross-chain agent identities through the multi-key architecture.
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

	// 3) Capabilities as JSON string
	capabilities := req.Capabilities
	if capabilities == nil {
		capabilities = map[string]any{}
	}
	capabilitiesJSON, err := json.Marshal(capabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capabilities: %w", err)
	}

	// First, populate key types and data
	for i, key := range keys {
		keyTypes[i] = uint8(key.Type) // #nosec G115 -- KeyType enum is 0-2, safe uint8 conversion
		keyData[i] = key.KeyData
	}

	// Calculate agentId (same as contract: keccak256(abi.encode(did, firstKeyData)))
	stringType, _ := abi.NewType("string", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{
		{Type: stringType},
		{Type: bytesType},
	}

	agentIdData, err := arguments.Pack(string(req.DID), keyData[0])
	if err != nil {
		return nil, fmt.Errorf("failed to encode agentId: %w", err)
	}

	agentId := crypto.Keccak256Hash(agentIdData)

	// Get owner address from private key
	ownerAddress := crypto.PubkeyToAddress(c.privateKey.PublicKey)

	// agentNonce is 0 for new registrations
	agentNonce := big.NewInt(0)

	// Generate signatures for each key based on chain-specific verification methods
	//
	// IMPORTANT: Signature requirements differ by blockchain due to on-chain verification capabilities:
	//
	// ETHEREUM (this implementation):
	//   - ECDSA/secp256k1: Signature REQUIRED
	//     → On-chain verification via ecrecover precompile (cheap and native)
	//     → Contract verifies signature matches msg.sender
	//   - Ed25519: Signature NOT REQUIRED
	//     → No native on-chain verification (would require expensive precompile)
	//     → Registered with empty signature, contract owner approves off-chain
	//
	// SOLANA/TENDERMINT (future implementations):
	//   - Ed25519: Signature REQUIRED
	//     → Native on-chain verification (ed25519_verify instruction)
	//     → Contract verifies signature proves key ownership
	//   - ECDSA: Signature NOT REQUIRED or NOT SUPPORTED
	//     → No native secp256k1 verification on Solana
	//     → Either use off-chain approval or don't support ECDSA
	//
	// This design ensures each chain uses its native cryptographic primitives efficiently
	// while maintaining cross-chain compatibility through the multi-key architecture.
	for i, key := range keys {

		// Priority 1: Use pre-computed signature if provided
		if len(key.Signature) > 0 {
			signatures[i] = key.Signature
			continue
		}

		// Priority 2: Check if signature is required based on key type and chain
		if key.Type == did.KeyTypeEd25519 {
			signatures[i] = []byte{} // empty
			continue
		}
		if key.Type == did.KeyTypeX25519 {
			if len(key.KeyData) != 32 {
				return nil, fmt.Errorf("x25519 key %d must be 32 bytes, got %d", i, len(key.KeyData))
			}
			signatures[i] = []byte{} // empty
			continue
		}

		// Priority 3: Generate ECDSA signature for Ethereum
		if key.Type == did.KeyTypeECDSA && req.KeyPair != nil && len(keys) == 1 {
			// Single-key mode: generate ECDSA signature for this key
			// Contract expects: keccak256(abi.encode(agentId, keyData, msg.sender, agentNonce))

			// Use ABI encoding to match Solidity's abi.encode
			bytes32Type, _ := abi.NewType("bytes32", "", nil)
			bytesType, _ := abi.NewType("bytes", "", nil)
			addressType, _ := abi.NewType("address", "", nil)
			uint256Type, _ := abi.NewType("uint256", "", nil)

			messageArgs := abi.Arguments{
				{Type: bytes32Type},
				{Type: bytesType},
				{Type: addressType},
				{Type: uint256Type},
			}
			messageData, err := messageArgs.Pack(agentId, keyData[i], ownerAddress, agentNonce)
			if err != nil {
				return nil, fmt.Errorf("failed to encode message for key %d: %w", i, err)
			}
			messageHash := crypto.Keccak256Hash(messageData)

			// Apply Ethereum personal sign prefix (contract does this)
			prefixedData := []byte("\x19Ethereum Signed Message:\n32")
			prefixedData = append(prefixedData, messageHash.Bytes()...)
			ethSignedHash := crypto.Keccak256Hash(prefixedData)

			// Sign the ethSignedHash with transaction signer's private key
			sig, err := crypto.Sign(ethSignedHash.Bytes(), c.privateKey)
			if err != nil {
				return nil, fmt.Errorf("failed to sign with key %d: %w", i, err)
			}

			// Adjust V value for Ethereum compatibility
			// crypto.Sign returns V as 0 or 1, but Ethereum ecrecover expects 27 or 28
			if sig[64] < 27 {
				sig[64] += 27
			}
			signatures[i] = sig
			continue
		}

		// If we reach here, key requires signature but none was provided
		return nil, fmt.Errorf("key %d (type=%d) requires signature but none provided", i, key.Type)
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

// ResolvePublicKey retrieves only the first public key for an agent (backward compatibility)
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

// ResolvePublicKey retrieves only the first public key for an agent (backward compatibility)
func (c *EthereumClientV4) ResolveKEMKey(ctx context.Context, agentDID did.AgentDID) (interface{}, error) {
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
	for _, h := range agent.KeyHashes {
		k, err := c.contract.GetKey(&bind.CallOpts{Context: ctx}, h)
		if err != nil {
			continue
		}
		if did.KeyType(k.KeyType) == did.KeyTypeX25519 {
			if len(k.KeyData) != 32 {
				return nil, fmt.Errorf("x25519 keyData must be 32 bytes, got %d", len(k.KeyData))
			}
			pub, err := ecdh.X25519().NewPublicKey(k.KeyData)
			if err != nil {
				return nil, fmt.Errorf("ecdh.X25519 NewPublicKey: %w", err)
			}
			return pub, nil
		}
	}
	return nil, fmt.Errorf("no x25519 key found for agent")
}

// ResolveAllPublicKeys retrieves all public keys for an agent
// This enables multi-key resolution for scenarios where agents have multiple keys
// for different purposes (e.g., ECDSA for Ethereum, Ed25519 for Solana)
func (c *EthereumClientV4) ResolveAllPublicKeys(
	ctx context.Context,
	agentDID did.AgentDID,
) ([]did.AgentKey, error) {
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

	keys := make([]did.AgentKey, 0, len(agent.KeyHashes))

	for _, keyHash := range agent.KeyHashes {
		keyData, err := c.contract.GetKey(&bind.CallOpts{Context: ctx}, keyHash)
		log.Println("[blockchain] Call get agent DID")
		if err != nil {
			// Skip keys that can't be retrieved
			continue
		}

		// Only include verified keys
		if !keyData.Verified {
			continue
		}

		keys = append(keys, did.AgentKey{
			Type:      did.KeyType(keyData.KeyType),
			KeyData:   keyData.KeyData,
			Signature: keyData.Signature,
			Verified:  keyData.Verified,
			CreatedAt: time.Unix(keyData.RegisteredAt.Int64(), 0),
		})
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("no verified keys found for agent")
	}

	return keys, nil
}

// ResolvePublicKeyByType retrieves a specific type of public key for an agent
// This enables protocol-specific key selection (e.g., use ECDSA for Ethereum operations,
// Ed25519 for Solana operations)
func (c *EthereumClientV4) ResolvePublicKeyByType(
	ctx context.Context,
	agentDID did.AgentDID,
	keyType did.KeyType,
) (interface{}, error) {
	keys, err := c.ResolveAllPublicKeys(ctx, agentDID)
	if err != nil {
		return nil, err
	}

	// Find first verified key of the specified type
	for _, key := range keys {
		if key.Type == keyType && key.Verified {
			// Determine key type string for unmarshaling
			keyTypeStr := "secp256k1"
			if keyType == did.KeyTypeEd25519 {
				keyTypeStr = "ed25519"
			}

			publicKey, err := did.UnmarshalPublicKey(key.KeyData, keyTypeStr)
			if err != nil {
				// Try next key if unmarshal fails
				continue
			}

			return publicKey, nil
		}
	}

	return nil, fmt.Errorf("no verified %s key found for agent", keyType)
}

// Update updates agent metadata
func (c *EthereumClientV4) Update(ctx context.Context, agentDID did.AgentDID, updates map[string]interface{}, keyPair sagecrypto.KeyPair) error {
	// Prepare transaction options first (need msg.sender)
	auth, err := c.getTransactOpts(ctx)
	if err != nil {
		return err
	}
	sender := auth.From

	// Get agent to retrieve first key data (needed to calculate correct agentId)
	agent, err := c.contract.GetAgentByDID(&bind.CallOpts{Context: ctx}, string(agentDID))
	if err != nil {
		return fmt.Errorf("failed to resolve agent: %w", err)
	}

	if agent.Did == "" {
		return did.ErrDIDNotFound
	}

	if len(agent.KeyHashes) == 0 {
		return fmt.Errorf("agent has no keys")
	}

	// Get first key data
	keyHash := agent.KeyHashes[0]
	keyData, err := c.contract.GetKey(&bind.CallOpts{Context: ctx}, keyHash)
	if err != nil {
		return fmt.Errorf("failed to get key data: %w", err)
	}

	// Calculate agentId (same as contract: keccak256(abi.encode(did, firstKeyData)))
	stringType, _ := abi.NewType("string", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{
		{Type: stringType},
		{Type: bytesType},
	}

	agentIdData, err := arguments.Pack(string(agentDID), keyData.KeyData)
	if err != nil {
		return fmt.Errorf("failed to encode agentId: %w", err)
	}

	agentId := crypto.Keccak256Hash(agentIdData)

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

	// Get current nonce from contract for replay protection
	// The contract's getNonce(bytes32 agentId) view function returns the current nonce
	// For V4.0 contracts without getNonce, we fallback to nonce=0
	var nonce *big.Int
	nonceResult, err := c.contract.GetNonce(&bind.CallOpts{Context: ctx}, agentId)
	if err == nil {
		nonce = nonceResult
	} else {
		// Fallback to nonce=0 for V4.0 contracts without getNonce support
		nonce = big.NewInt(0)
	}

	// Prepare message matching contract's signature verification
	// Contract expects: keccak256(abi.encode(agentId, name, description, endpoint, capabilities, msg.sender, agentNonce[agentId]))
	bytes32Type, _ := abi.NewType("bytes32", "", nil)
	addressType, _ := abi.NewType("address", "", nil)
	uint256Type, _ := abi.NewType("uint256", "", nil)

	sigArguments := abi.Arguments{
		{Type: bytes32Type}, // agentId
		{Type: stringType},  // name
		{Type: stringType},  // description
		{Type: stringType},  // endpoint
		{Type: stringType},  // capabilities
		{Type: addressType}, // msg.sender
		{Type: uint256Type}, // nonce
	}

	messageData, err := sigArguments.Pack(
		agentId,
		name,
		description,
		endpoint,
		capabilitiesJSON,
		sender,
		nonce,
	)
	if err != nil {
		return fmt.Errorf("failed to encode update message: %w", err)
	}

	messageHash := crypto.Keccak256Hash(messageData)

	// Apply Ethereum personal sign prefix (contract does this in _verifyEcdsaSignature)
	prefixedData := []byte("\x19Ethereum Signed Message:\n32")
	prefixedData = append(prefixedData, messageHash.Bytes()...)
	ethSignedHash := crypto.Keccak256Hash(prefixedData)

	// Sign the prefixed hash with agent's private key
	// Note: We use crypto.Sign directly here instead of keyPair.Sign because we need
	// to sign with the ECDSA private key specifically for Ethereum signature verification
	ecdsaPrivKey, ok := keyPair.PrivateKey().(*ecdsa.PrivateKey)
	if !ok {
		return fmt.Errorf("private key must be ECDSA for Update on Ethereum")
	}

	signature, err := crypto.Sign(ethSignedHash.Bytes(), ecdsaPrivKey)
	if err != nil {
		return fmt.Errorf("failed to sign update: %w", err)
	}

	// Adjust V value for Ethereum compatibility
	// go-ethereum's Sign returns V as 0 or 1, but Ethereum expects 27 or 28
	if signature[64] < 27 {
		signature[64] += 27
	}

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

	// Get agent to retrieve first key data (needed to calculate correct agentId)
	agent, err := c.contract.GetAgentByDID(&bind.CallOpts{Context: ctx}, string(agentDID))
	if err != nil {
		return fmt.Errorf("failed to resolve agent: %w", err)
	}

	if agent.Did == "" {
		return did.ErrDIDNotFound
	}

	if len(agent.KeyHashes) == 0 {
		return fmt.Errorf("agent has no keys")
	}

	// Get first key data
	keyHash := agent.KeyHashes[0]
	keyData, err := c.contract.GetKey(&bind.CallOpts{Context: ctx}, keyHash)
	if err != nil {
		return fmt.Errorf("failed to get key data: %w", err)
	}

	// Calculate agentId (same as contract: keccak256(abi.encode(did, firstKeyData)))
	stringType, _ := abi.NewType("string", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{
		{Type: stringType},
		{Type: bytesType},
	}

	agentIdData, err := arguments.Pack(string(agentDID), keyData.KeyData)
	if err != nil {
		return fmt.Errorf("failed to encode agentId: %w", err)
	}

	agentId := crypto.Keccak256Hash(agentIdData)

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
				// #nosec G115 -- ConfirmationBlocks is validated non-negative above
				if confirmations < uint64(c.config.ConfirmationBlocks) {
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

// ApproveEd25519Key approves an Ed25519 key (contract owner only)
// Implements RegistryV4 interface - takes hex-encoded key hash string
func (c *EthereumClientV4) ApproveEd25519Key(ctx context.Context, keyHashStr string) error {
	// Parse hex string to [32]byte
	keyHashHex := keyHashStr
	if len(keyHashStr) > 2 && keyHashStr[:2] == "0x" {
		keyHashHex = keyHashStr[2:]
	}

	keyHashBytes, err := hex.DecodeString(keyHashHex)
	if err != nil {
		return fmt.Errorf("invalid key hash format: %w", err)
	}

	var keyHash [32]byte
	copy(keyHash[:], keyHashBytes)

	auth, err := c.getTransactOpts(ctx)
	if err != nil {
		return err
	}

	tx, err := c.contract.ApproveEd25519Key(auth, keyHash)
	if err != nil {
		return fmt.Errorf("failed to approve Ed25519 key: %w", err)
	}

	_, err = c.waitForTransaction(ctx, tx)
	return err
}

// GetAgentKeyHash retrieves key details by calculating the key hash
func (c *EthereumClientV4) GetAgentKeyHash(ctx context.Context, agentDID did.AgentDID, keyData []byte, keyType did.KeyType) ([32]byte, error) {
	// First get agentId
	agent, err := c.contract.GetAgentByDID(&bind.CallOpts{Context: ctx}, string(agentDID))
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to get agent: %w", err)
	}

	if agent.Did == "" {
		return [32]byte{}, did.ErrDIDNotFound
	}

	// Calculate agentId (same as contract: keccak256(abi.encode(did, firstKeyData)))
	stringType, _ := abi.NewType("string", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{
		{Type: stringType},
		{Type: bytesType},
	}

	// Get first key data
	firstKeyHash := agent.KeyHashes[0]
	firstKey, err := c.contract.GetKey(&bind.CallOpts{Context: ctx}, firstKeyHash)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to get first key: %w", err)
	}

	agentIdData, err := arguments.Pack(string(agentDID), firstKey.KeyData)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to encode agentId: %w", err)
	}

	agentId := crypto.Keccak256Hash(agentIdData)

	// Calculate keyHash: keccak256(abi.encode(agentId, keyType, keyData))
	bytes32Type, _ := abi.NewType("bytes32", "", nil)
	uint8Type, _ := abi.NewType("uint8", "", nil)
	bytesType2, _ := abi.NewType("bytes", "", nil)

	keyHashArgs := abi.Arguments{
		{Type: bytes32Type},
		{Type: uint8Type},
		{Type: bytesType2},
	}

	keyHashData, err := keyHashArgs.Pack(agentId, uint8(keyType), keyData) // #nosec G115 -- KeyType enum is 0-2, safe uint8 conversion
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to encode keyHash: %w", err)
	}

	return crypto.Keccak256Hash(keyHashData), nil
}

// GetAgentKey retrieves key details by key hash
func (c *EthereumClientV4) GetAgentKey(ctx context.Context, keyHash [32]byte) (*registryv4.ISageRegistryV4AgentKey, error) {
	key, err := c.contract.GetKey(&bind.CallOpts{Context: ctx}, keyHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	if key.RegisteredAt.Int64() == 0 {
		return nil, fmt.Errorf("key not found")
	}

	return &key, nil
}

// AddKey adds a new cryptographic key to an existing agent
func (c *EthereumClientV4) AddKey(ctx context.Context, agentDID did.AgentDID, key did.AgentKey) (string, error) {
	// Get agent to calculate agentId
	agent, err := c.contract.GetAgentByDID(&bind.CallOpts{Context: ctx}, string(agentDID))
	if err != nil {
		return "", fmt.Errorf("failed to get agent: %w", err)
	}

	if agent.Did == "" {
		return "", did.ErrDIDNotFound
	}

	// Calculate agentId
	stringType, _ := abi.NewType("string", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{
		{Type: stringType},
		{Type: bytesType},
	}

	// Get first key data
	firstKeyHash := agent.KeyHashes[0]
	firstKey, err := c.contract.GetKey(&bind.CallOpts{Context: ctx}, firstKeyHash)
	if err != nil {
		return "", fmt.Errorf("failed to get first key: %w", err)
	}

	agentIdData, err := arguments.Pack(string(agentDID), firstKey.KeyData)
	if err != nil {
		return "", fmt.Errorf("failed to encode agentId: %w", err)
	}
	agentId := crypto.Keccak256Hash(agentIdData)

	// Generate signature for ECDSA keys
	signature := key.Signature
	if key.Type == did.KeyTypeECDSA && len(signature) == 0 {
		// Get owner address
		ownerAddress := crypto.PubkeyToAddress(c.privateKey.PublicKey)

		// Get current agent nonce (number of keys)
		agentNonce := big.NewInt(int64(len(agent.KeyHashes)))

		// Generate signature: keccak256(abi.encode(agentId, keyData, msg.sender, agentNonce))
		bytes32Type, _ := abi.NewType("bytes32", "", nil)
		addressType, _ := abi.NewType("address", "", nil)
		uint256Type, _ := abi.NewType("uint256", "", nil)

		messageArgs := abi.Arguments{
			{Type: bytes32Type},
			{Type: bytesType},
			{Type: addressType},
			{Type: uint256Type},
		}

		messageData, err := messageArgs.Pack(agentId, key.KeyData, ownerAddress, agentNonce)
		if err != nil {
			return "", fmt.Errorf("failed to encode message: %w", err)
		}

		messageHash := crypto.Keccak256Hash(messageData)

		// Apply Ethereum personal sign prefix
		prefixedData := []byte("\x19Ethereum Signed Message:\n32")
		prefixedData = append(prefixedData, messageHash.Bytes()...)
		ethSignedHash := crypto.Keccak256Hash(prefixedData)

		// Sign with private key
		sig, err := crypto.Sign(ethSignedHash.Bytes(), c.privateKey)
		if err != nil {
			return "", fmt.Errorf("failed to sign message: %w", err)
		}

		// Adjust V value
		if sig[64] < 27 {
			sig[64] += 27
		}

		signature = sig
	}

	// Ed25519 keys don't need signature on Ethereum
	if key.Type == did.KeyTypeEd25519 {
		signature = []byte{}
	}

	// Prepare transaction options
	auth, err := c.getTransactOpts(ctx)
	if err != nil {
		return "", err
	}

	// Call addKey on contract
	tx, err := c.contract.AddKey(auth, agentId, uint8(key.Type), key.KeyData, signature) // #nosec G115 -- KeyType enum is 0-2, safe uint8 conversion
	if err != nil {
		return "", fmt.Errorf("failed to add key: %w", err)
	}

	// Wait for transaction confirmation
	_, err = c.waitForTransaction(ctx, tx)
	if err != nil {
		return "", err
	}

	// Calculate key hash to return
	keyHash, err := c.GetAgentKeyHash(ctx, agentDID, key.KeyData, key.Type)
	if err != nil {
		return "", fmt.Errorf("failed to calculate key hash: %w", err)
	}

	return "0x" + hex.EncodeToString(keyHash[:]), nil
}

// RevokeKey revokes a cryptographic key from an agent
func (c *EthereumClientV4) RevokeKey(ctx context.Context, agentDID did.AgentDID, keyHashStr string) error {
	// Get agent to calculate agentId
	agent, err := c.contract.GetAgentByDID(&bind.CallOpts{Context: ctx}, string(agentDID))
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	if agent.Did == "" {
		return did.ErrDIDNotFound
	}

	// Calculate agentId
	stringType, _ := abi.NewType("string", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{
		{Type: stringType},
		{Type: bytesType},
	}

	// Get first key data
	firstKeyHash := agent.KeyHashes[0]
	firstKey, err := c.contract.GetKey(&bind.CallOpts{Context: ctx}, firstKeyHash)
	if err != nil {
		return fmt.Errorf("failed to get first key: %w", err)
	}

	agentIdData, err := arguments.Pack(string(agentDID), firstKey.KeyData)
	if err != nil {
		return fmt.Errorf("failed to encode agentId: %w", err)
	}
	agentId := crypto.Keccak256Hash(agentIdData)

	// Parse key hash from hex string
	keyHashBytes, err := hex.DecodeString(strings.TrimPrefix(keyHashStr, "0x"))
	if err != nil {
		return fmt.Errorf("invalid key hash format: %w", err)
	}

	var keyHash [32]byte
	copy(keyHash[:], keyHashBytes)

	// Prepare transaction options
	auth, err := c.getTransactOpts(ctx)
	if err != nil {
		return err
	}

	// Call revokeKey on contract
	tx, err := c.contract.RevokeKey(auth, agentId, keyHash)
	if err != nil {
		return fmt.Errorf("failed to revoke key: %w", err)
	}

	// Wait for transaction confirmation
	_, err = c.waitForTransaction(ctx, tx)
	return err
}

// VerifyMetadata checks whether the provided metadata matches the on-chain state (V4 client).
func (c *EthereumClientV4) VerifyMetadata(
	ctx context.Context,
	agentDID did.AgentDID,
	metadata *did.AgentMetadata,
) (*did.VerificationResult, error) {

	// Resolve authoritative metadata from chain via the existing V4 resolve path.
	onChain, err := c.Resolve(ctx, agentDID)
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

	// Field-by-field comparison (same semantics as previous client)
	valid := true
	var reason string

	if metadata.Name != onChain.Name {
		valid = false
		reason = fmt.Sprintf("name mismatch: expected %s, got %s", onChain.Name, metadata.Name)
	}
	if metadata.Description != onChain.Description {
		valid = false
		if reason != "" {
			reason += "; "
		}
		reason += "description mismatch"
	}
	if metadata.Endpoint != onChain.Endpoint {
		valid = false
		if reason != "" {
			reason += "; "
		}
		reason += fmt.Sprintf("endpoint mismatch: expected %s, got %s", onChain.Endpoint, metadata.Endpoint)
	}
	if !compareCapabilities(metadata.Capabilities, onChain.Capabilities) {
		valid = false
		if reason != "" {
			reason += "; "
		}
		reason += "capabilities mismatch"
	}

	res := &did.VerificationResult{
		Valid:      valid,
		Agent:      onChain,
		VerifiedAt: time.Now(),
	}
	if !valid {
		res.Error = reason
	}
	return res, nil
}

// ListAgentsByOwner returns all agents owned by a given EOA (V4 client).
// This uses the abigen metadata ABI to pack/call/unpack `getAgentsByOwner(address)`.
func (c *EthereumClientV4) ListAgentsByOwner(
	ctx context.Context,
	ownerAddress string,
) ([]*did.AgentMetadata, error) {

	// Basic address validation
	if !common.IsHexAddress(ownerAddress) {
		return nil, fmt.Errorf("invalid Ethereum address: %s", ownerAddress)
	}
	owner := common.HexToAddress(ownerAddress)

	// Obtain ABI from abigen metadata
	abiObj, err := registryv4.SageRegistryV4MetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("get ABI: %w", err)
	}

	// Pack call data for getAgentsByOwner(address)
	data, err := abiObj.Pack("getAgentsByOwner", owner)
	if err != nil {
		return nil, fmt.Errorf("pack getAgentsByOwner: %w", err)
	}

	// eth_call
	out, err := c.client.CallContract(ctx, ethereum.CallMsg{
		To:   &c.contractAddress,
		Data: data,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("call getAgentsByOwner: %w", err)
	}

	// Unpack DID list
	var dids []string
	if err := abiObj.UnpackIntoInterface(&dids, "getAgentsByOwner", out); err != nil {
		return nil, fmt.Errorf("unpack getAgentsByOwner: %w", err)
	}

	// Resolve each DID into AgentMetadata via existing V4 resolver
	agents := make([]*did.AgentMetadata, 0, len(dids))
	for _, d := range dids {
		meta, err := c.Resolve(ctx, did.AgentDID(d))
		if err != nil {
			// best-effort: skip entries that fail to resolve
			continue
		}
		agents = append(agents, meta)
	}
	return agents, nil
}

// Search is not supported without off-chain indexing. Return a descriptive error.
func (c *EthereumClientV4) Search(
	ctx context.Context,
	criteria did.SearchCriteria,
) ([]*did.AgentMetadata, error) {
	// Production options:
	// - Index events and build a local index
	// - Use a Graph subgraph
	// - Or an external indexing service
	return nil, fmt.Errorf("search functionality requires off-chain indexing")
}
