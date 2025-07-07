package ethereum

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
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
	
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/chain"
	"github.com/sage-x-project/sage/did"
)

// EthereumClient implements DID registry operations for Ethereum
type EthereumClient struct {
	client          *ethclient.Client
	contract        *bind.BoundContract
	contractABI     abi.ABI
	contractAddress common.Address
	privateKey      *ecdsa.PrivateKey
	chainID         *big.Int
	config          *did.RegistryConfig
}

// NewEthereumClient creates a new Ethereum DID client
func NewEthereumClient(config *did.RegistryConfig) (*EthereumClient, error) {
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
	
	// Parse the contract ABI
	contractABI, err := abi.JSON(strings.NewReader(SageRegistryABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %w", err)
	}
	
	contract := bind.NewBoundContract(contractAddress, contractABI, client, client, client)
	
	return &EthereumClient{
		client:          client,
		contract:        contract,
		contractABI:     contractABI,
		contractAddress: contractAddress,
		privateKey:      privateKey,
		chainID:         chainID,
		config:          config,
	}, nil
}

// Register registers a new agent on Ethereum
func (c *EthereumClient) Register(ctx context.Context, req *did.RegistrationRequest) (*did.RegistrationResult, error) {
	// Convert the key pair to Ethereum format
	if req.KeyPair.Type() != sagecrypto.KeyTypeSecp256k1 {
		return nil, fmt.Errorf("Ethereum requires Secp256k1 keys")
	}
	
	// Get the Ethereum address for the public key
	provider, err := chain.GetProvider(chain.ChainTypeEthereum)
	if err != nil {
		return nil, err
	}
	
	address, err := provider.GenerateAddress(req.KeyPair.PublicKey(), chain.NetworkEthereumMainnet)
	if err != nil {
		return nil, err
	}
	
	// Prepare the message to sign
	message := c.prepareRegistrationMessage(req, address.Value)
	messageHash := crypto.Keccak256([]byte(message))
	
	// Sign the message
	signature, err := req.KeyPair.Sign(messageHash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign registration: %w", err)
	}
	
	// Prepare capabilities as JSON string
	capabilitiesJSON, err := json.Marshal(req.Capabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capabilities: %w", err)
	}
	
	// Get public key bytes
	publicKeyBytes, err := did.MarshalPublicKey(req.KeyPair.PublicKey())
	if err != nil {
		return nil, err
	}
	
	// Prepare transaction options
	auth, err := c.getTransactOpts(ctx)
	if err != nil {
		return nil, err
	}
	
	// Call the contract
	tx, err := c.contract.Transact(auth, "registerAgent",
		string(req.DID),
		req.Name,
		req.Description,
		req.Endpoint,
		publicKeyBytes,
		string(capabilitiesJSON),
		signature,
	)
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

// Resolve retrieves agent metadata from Ethereum
func (c *EthereumClient) Resolve(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error) {
	var result struct {
		Exists       bool
		Name         string
		Description  string
		Endpoint     string
		PublicKey    []byte
		Capabilities string
		Owner        common.Address
		IsActive     bool
		CreatedAt    *big.Int
		UpdatedAt    *big.Int
	}
	
	// Prepare call data
	callData, err := c.contractABI.Pack("getAgent", string(agentDID))
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
	err = c.contractABI.UnpackIntoInterface(&result, "getAgent", output)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	
	if !result.Exists {
		return nil, did.ErrDIDNotFound
	}
	
	// Parse public key
	// TODO: Determine key type from contract or metadata
	publicKey, err := did.UnmarshalPublicKey(result.PublicKey, "ed25519")
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal public key: %w", err)
	}
	
	// Parse capabilities
	var capabilities map[string]interface{}
	if result.Capabilities != "" {
		err = json.Unmarshal([]byte(result.Capabilities), &capabilities)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal capabilities: %w", err)
		}
	}
	
	return &did.AgentMetadata{
		DID:          agentDID,
		Name:         result.Name,
		Description:  result.Description,
		Endpoint:     result.Endpoint,
		PublicKey:    publicKey,
		Capabilities: capabilities,
		Owner:        result.Owner.Hex(),
		IsActive:     result.IsActive,
		CreatedAt:    time.Unix(result.CreatedAt.Int64(), 0),
		UpdatedAt:    time.Unix(result.UpdatedAt.Int64(), 0),
	}, nil
}

// Update updates agent metadata on Ethereum
func (c *EthereumClient) Update(ctx context.Context, agentDID did.AgentDID, updates map[string]interface{}, keyPair sagecrypto.KeyPair) error {
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
	
	// Call the contract
	tx, err := c.contract.Transact(auth, "updateAgent",
		string(agentDID),
		name,
		description,
		endpoint,
		capabilitiesJSON,
		signature,
	)
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}
	
	// Wait for confirmation
	_, err = c.waitForTransaction(ctx, tx)
	return err
}

// Deactivate deactivates an agent on Ethereum
func (c *EthereumClient) Deactivate(ctx context.Context, agentDID did.AgentDID, keyPair sagecrypto.KeyPair) error {
	// Prepare deactivation message
	message := fmt.Sprintf("Deactivate agent: %s", agentDID)
	messageHash := crypto.Keccak256([]byte(message))
	
	// Sign the message
	signature, err := keyPair.Sign(messageHash)
	if err != nil {
		return fmt.Errorf("failed to sign deactivation: %w", err)
	}
	
	// Prepare transaction options
	auth, err := c.getTransactOpts(ctx)
	if err != nil {
		return err
	}
	
	// Call the contract
	tx, err := c.contract.Transact(auth, "deactivateAgent", string(agentDID), signature)
	if err != nil {
		return fmt.Errorf("failed to deactivate agent: %w", err)
	}
	
	// Wait for confirmation
	_, err = c.waitForTransaction(ctx, tx)
	return err
}

// Helper methods

func (c *EthereumClient) getTransactOpts(ctx context.Context) (*bind.TransactOpts, error) {
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
		auth.GasPrice = big.NewInt(int64(c.config.GasPrice))
	}
	
	return auth, nil
}

func (c *EthereumClient) waitForTransaction(ctx context.Context, tx *types.Transaction) (*types.Receipt, error) {
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
				
				confirmations := currentBlock - receipt.BlockNumber.Uint64()
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

func (c *EthereumClient) prepareRegistrationMessage(req *did.RegistrationRequest, address string) string {
	return fmt.Sprintf("Register agent:\nDID: %s\nName: %s\nEndpoint: %s\nAddress: %s",
		req.DID, req.Name, req.Endpoint, address)
}

func (c *EthereumClient) prepareUpdateMessage(agentDID did.AgentDID, updates map[string]interface{}) string {
	return fmt.Sprintf("Update agent: %s\nUpdates: %v", agentDID, updates)
}