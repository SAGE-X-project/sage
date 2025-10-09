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
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	
	sagecrypto "github.com/sage-x-project/sage/crypto"
	"github.com/sage-x-project/sage/crypto/chain"
	"github.com/sage-x-project/sage/did"
)

// SolanaClient implements DID registry operations for Solana
type SolanaClient struct {
	client          *rpc.Client
	programID       solana.PublicKey
	registryPDA     solana.PublicKey
	feePayer        solana.PrivateKey
	config          *did.RegistryConfig
}

// AgentAccount represents the on-chain agent data structure
type AgentAccount struct {
	DID          string                 `json:"did"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Endpoint     string                 `json:"endpoint"`
	PublicKey    [32]byte               `json:"public_key"`
	Capabilities map[string]interface{} `json:"capabilities"`
	Owner        solana.PublicKey       `json:"owner"`
	IsActive     bool                   `json:"is_active"`
	CreatedAt    int64                  `json:"created_at"`
	UpdatedAt    int64                  `json:"updated_at"`
}

// init registers the Solana client creator with the factory
func init() {
	did.RegisterSolanaClientCreator(func(config *did.RegistryConfig) (did.Client, error) {
		return NewSolanaClient(config)
	})
}

// NewSolanaClient creates a new Solana DID client
func NewSolanaClient(config *did.RegistryConfig) (*SolanaClient, error) {
	client := rpc.New(config.RPCEndpoint)
	
	programID, err := solana.PublicKeyFromBase58(config.ContractAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid program ID: %w", err)
	}
	
	// Derive registry PDA
	registryPDA, _, err := solana.FindProgramAddress(
		[][]byte{[]byte("registry")},
		programID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to derive registry PDA: %w", err)
	}
	
	var feePayer solana.PrivateKey
	if config.PrivateKey != "" {
		feePayer, err = solana.PrivateKeyFromBase58(config.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid fee payer private key: %w", err)
		}
	}
	
	return &SolanaClient{
		client:      client,
		programID:   programID,
		registryPDA: registryPDA,
		feePayer:    feePayer,
		config:      config,
	}, nil
}

// Register registers a new agent on Solana
func (c *SolanaClient) Register(ctx context.Context, req *did.RegistrationRequest) (*did.RegistrationResult, error) {
	// Solana requires Ed25519 keys
	if req.KeyPair.Type() != sagecrypto.KeyTypeEd25519 {
		return nil, fmt.Errorf("Solana requires Ed25519 keys")
	}
	
	// Get the Solana address for the public key
	provider, err := chain.GetProvider(chain.ChainTypeSolana)
	if err != nil {
		return nil, err
	}
	
	address, err := provider.GenerateAddress(req.KeyPair.PublicKey(), chain.NetworkSolanaMainnet)
	if err != nil {
		return nil, err
	}
	
	ownerPubkey, err := solana.PublicKeyFromBase58(address.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}
	
	// Derive agent PDA
	agentPDA, _, err := solana.FindProgramAddress(
		[][]byte{
			[]byte("agent"),
			[]byte(req.DID),
		},
		c.programID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to derive agent PDA: %w", err)
	}
	
	// Prepare the message to sign
	message := c.prepareRegistrationMessage(req, address.Value)
	
	// Sign the message
	signature, err := req.KeyPair.Sign([]byte(message))
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
	
	// Build the transaction
	recentBlockhash, err := c.client.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent blockhash: %w", err)
	}
	
	// Create instruction data
	instructionData := struct {
		Instruction  uint8
		DID          string
		Name         string
		Description  string
		Endpoint     string
		PublicKey    []byte
		Capabilities string
		Signature    [64]byte
	}{
		Instruction:  0, // RegisterAgent instruction
		DID:          string(req.DID),
		Name:         req.Name,
		Description:  req.Description,
		Endpoint:     req.Endpoint,
		PublicKey:    publicKeyBytes,
		Capabilities: string(capabilitiesJSON),
	}
	copy(instructionData.Signature[:], signature)
	
	// Create the instruction
	instruction := &solana.GenericInstruction{
		ProgID: c.programID,
		AccountValues: solana.AccountMetaSlice{
			{PublicKey: agentPDA, IsWritable: true, IsSigner: false},
			{PublicKey: c.registryPDA, IsWritable: true, IsSigner: false},
			{PublicKey: ownerPubkey, IsWritable: true, IsSigner: true},
			{PublicKey: solana.SystemProgramID, IsWritable: false, IsSigner: false},
			{PublicKey: solana.SysVarInstructionsPubkey, IsWritable: false, IsSigner: false},
		},
		DataBytes: serializeInstruction(instructionData),
	}
	
	// Build and send transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recentBlockhash.Value.Blockhash,
		solana.TransactionPayer(c.feePayer.PublicKey()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}
	
	// Sign the transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(c.feePayer.PublicKey()) {
			return &c.feePayer
		}
		// Additional logic for owner signing would go here
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}
	
	// Send transaction
	sig, err := c.client.SendTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}
	
	// Wait for confirmation
	result, err := c.waitForConfirmation(ctx, sig)
	if err != nil {
		return nil, err
	}
	
	return result, nil
}

// Resolve retrieves agent metadata from Solana
func (c *SolanaClient) Resolve(ctx context.Context, agentDID did.AgentDID) (*did.AgentMetadata, error) {
	// Derive agent PDA
	agentPDA, _, err := solana.FindProgramAddress(
		[][]byte{
			[]byte("agent"),
			[]byte(agentDID),
		},
		c.programID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to derive agent PDA: %w", err)
	}
	
	// Fetch account data
	accountInfo, err := c.client.GetAccountInfo(ctx, agentPDA)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}
	
	if accountInfo == nil || accountInfo.Value == nil {
		return nil, did.ErrDIDNotFound
	}
	
	// Deserialize account data
	var agentAccount AgentAccount
	err = deserializeAccount(accountInfo.Value.Data.GetBinary(), &agentAccount)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize account: %w", err)
	}
	
	// Convert public key
	publicKey := ed25519.PublicKey(agentAccount.PublicKey[:])
	
	return &did.AgentMetadata{
		DID:          agentDID,
		Name:         agentAccount.Name,
		Description:  agentAccount.Description,
		Endpoint:     agentAccount.Endpoint,
		PublicKey:    publicKey,
		Capabilities: agentAccount.Capabilities,
		Owner:        agentAccount.Owner.String(),
		IsActive:     agentAccount.IsActive,
		CreatedAt:    time.Unix(agentAccount.CreatedAt, 0),
		UpdatedAt:    time.Unix(agentAccount.UpdatedAt, 0),
	}, nil
}

// Update updates agent metadata on Solana
func (c *SolanaClient) Update(ctx context.Context, agentDID did.AgentDID, updates map[string]interface{}, keyPair sagecrypto.KeyPair) error {
	// Prepare update message
	message := c.prepareUpdateMessage(agentDID, updates)
	
	// Sign the message
	signature, err := keyPair.Sign([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to sign update: %w", err)
	}
	
	// Get owner address
	provider, err := chain.GetProvider(chain.ChainTypeSolana)
	if err != nil {
		return err
	}
	
	address, err := provider.GenerateAddress(keyPair.PublicKey(), chain.NetworkSolanaMainnet)
	if err != nil {
		return err
	}
	
	ownerPubkey, err := solana.PublicKeyFromBase58(address.Value)
	if err != nil {
		return fmt.Errorf("invalid owner address: %w", err)
	}
	
	// Derive agent PDA
	agentPDA, _, err := solana.FindProgramAddress(
		[][]byte{
			[]byte("agent"),
			[]byte(agentDID),
		},
		c.programID,
	)
	if err != nil {
		return fmt.Errorf("failed to derive agent PDA: %w", err)
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
	
	// Create instruction data
	instructionData := struct {
		Instruction  uint8
		Name         string
		Description  string
		Endpoint     string
		Capabilities string
		Signature    [64]byte
	}{
		Instruction:  1, // UpdateAgent instruction
		Name:         name,
		Description:  description,
		Endpoint:     endpoint,
		Capabilities: capabilitiesJSON,
	}
	copy(instructionData.Signature[:], signature)

	// Get recent blockhash
	recentBlockhash, err := c.client.GetLatestBlockhash(ctx, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Create instruction
	instruction := solana.NewInstruction(
		c.programID,
		solana.AccountMetaSlice{
			{PublicKey: agentPDA, IsWritable: true, IsSigner: false},
			{PublicKey: c.registryPDA, IsWritable: true, IsSigner: false},
			{PublicKey: ownerPubkey, IsWritable: true, IsSigner: true},
			{PublicKey: solana.SystemProgramID, IsWritable: false, IsSigner: false},
			{PublicKey: solana.SysVarInstructionsPubkey, IsWritable: false, IsSigner: false},
		},
		serializeInstruction(instructionData),
	)

	// Build transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recentBlockhash.Value.Blockhash,
		solana.TransactionPayer(c.feePayer.PublicKey()),
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign the transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(c.feePayer.PublicKey()) {
			return &c.feePayer
		}
		// Owner signature would be handled by the keyPair parameter
		// In a real implementation, you would need to pass the owner's private key
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	sig, err := c.client.SendTransaction(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for confirmation
	_, err = c.waitForConfirmation(ctx, sig)
	if err != nil {
		return fmt.Errorf("failed to confirm transaction: %w", err)
	}

	return nil
}

// Deactivate deactivates an agent on Solana
func (c *SolanaClient) Deactivate(ctx context.Context, agentDID did.AgentDID, keyPair sagecrypto.KeyPair) error {
	// Extract owner public key from keyPair
	publicKey := keyPair.PublicKey()

	// Convert to bytes (assuming Ed25519)
	var publicKeyBytes []byte
	switch pk := publicKey.(type) {
	case ed25519.PublicKey:
		publicKeyBytes = pk
	default:
		return fmt.Errorf("unsupported public key type for Solana")
	}

	if len(publicKeyBytes) != 32 {
		return fmt.Errorf("invalid public key length: expected 32, got %d", len(publicKeyBytes))
	}

	var ownerPubkeyBytes [32]byte
	copy(ownerPubkeyBytes[:], publicKeyBytes)
	ownerPubkey := solana.PublicKeyFromBytes(ownerPubkeyBytes[:])

	// Derive agent PDA
	agentPDA, _, err := solana.FindProgramAddress(
		[][]byte{
			[]byte("agent"),
			[]byte(agentDID),
		},
		c.programID,
	)
	if err != nil {
		return fmt.Errorf("failed to derive agent PDA: %w", err)
	}

	// Create signature for deactivation
	message := fmt.Sprintf("deactivate:%s", agentDID)
	signature, err := keyPair.Sign([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to sign deactivation message: %w", err)
	}

	// Create instruction data
	instructionData := struct {
		Instruction uint8
		Signature   [64]byte
	}{
		Instruction: 2, // DeactivateAgent instruction
	}
	copy(instructionData.Signature[:], signature)

	// Get recent blockhash
	recentBlockhash, err := c.client.GetLatestBlockhash(ctx, rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	// Create instruction
	instruction := solana.NewInstruction(
		c.programID,
		solana.AccountMetaSlice{
			{PublicKey: agentPDA, IsWritable: true, IsSigner: false},
			{PublicKey: c.registryPDA, IsWritable: true, IsSigner: false},
			{PublicKey: ownerPubkey, IsWritable: true, IsSigner: true},
			{PublicKey: solana.SystemProgramID, IsWritable: false, IsSigner: false},
		},
		serializeInstruction(instructionData),
	)

	// Build transaction
	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		recentBlockhash.Value.Blockhash,
		solana.TransactionPayer(c.feePayer.PublicKey()),
	)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// Sign the transaction
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(c.feePayer.PublicKey()) {
			return &c.feePayer
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	sig, err := c.client.SendTransaction(ctx, tx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for confirmation
	_, err = c.waitForConfirmation(ctx, sig)
	if err != nil {
		return fmt.Errorf("failed to confirm transaction: %w", err)
	}

	return nil
}

// Helper methods

func (c *SolanaClient) waitForConfirmation(ctx context.Context, sig solana.Signature) (*did.RegistrationResult, error) {
	// Wait for transaction confirmation
	maxRetries := c.config.MaxRetries
	if maxRetries == 0 {
		maxRetries = 30
	}
	
	for i := 0; i < maxRetries; i++ {
		status, err := c.client.GetSignatureStatuses(ctx, false, sig)
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		
		if status != nil && status.Value != nil && len(status.Value) > 0 {
			txStatus := status.Value[0]
			if txStatus.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
				return &did.RegistrationResult{
					TransactionHash: sig.String(),
					Slot:            txStatus.Slot,
					Timestamp:       time.Now(),
				}, nil
			}
		}
		
		time.Sleep(2 * time.Second)
	}
	
	return nil, fmt.Errorf("transaction confirmation timeout")
}

func (c *SolanaClient) prepareRegistrationMessage(req *did.RegistrationRequest, address string) string {
	return fmt.Sprintf("Register agent:\nDID: %s\nName: %s\nEndpoint: %s\nAddress: %s",
		req.DID, req.Name, req.Endpoint, address)
}

func (c *SolanaClient) prepareUpdateMessage(agentDID did.AgentDID, updates map[string]interface{}) string {
	return fmt.Sprintf("Update agent: %s\nUpdates: %v", agentDID, updates)
}

// serializeInstruction serializes instruction data (simplified)
func serializeInstruction(data interface{}) []byte {
	// In production, use proper borsh serialization
	bytes, _ := json.Marshal(data)
	return bytes
}

// deserializeAccount deserializes account data (simplified)
func deserializeAccount(data []byte, v interface{}) error {
	// In production, use proper borsh deserialization
	return json.Unmarshal(data, v)
}
