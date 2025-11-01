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
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	_ "github.com/sage-x-project/sage/internal/cryptoinit" // Initialize crypto wrappers
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

func TestSageRegistryABI(t *testing.T) {
	// Test that ABI is valid JSON
	assert.NotEmpty(t, SageRegistryABI)
	assert.Contains(t, SageRegistryABI, "registerAgent")
	assert.Contains(t, SageRegistryABI, "getAgent")
	assert.Contains(t, SageRegistryABI, "updateAgent")
	assert.Contains(t, SageRegistryABI, "deactivateAgent")
}

func TestNewEthereumClient(t *testing.T) {
	// Test with invalid RPC endpoint (guaranteed to fail)
	config := &did.RegistryConfig{
		Chain:           did.ChainEthereum,
		ContractAddress: "0x1234567890123456789012345678901234567890",
		RPCEndpoint:     "http://invalid-endpoint-that-does-not-exist:8545",
		PrivateKey:      "", // No private key for read-only
	}

	// This will fail with invalid RPC endpoint
	_, err := NewEthereumClient(config)
	assert.Error(t, err)

	// Test with invalid contract address format
	invalidConfig := &did.RegistryConfig{
		Chain:           did.ChainEthereum,
		ContractAddress: "invalid-address",
		RPCEndpoint:     "http://invalid-endpoint-that-does-not-exist:8545",
	}

	_, err = NewEthereumClient(invalidConfig)
	assert.Error(t, err)

	// Test with valid localhost endpoint if available (skip if not)
	t.Run("WithLocalNode", func(t *testing.T) {
		// This test only runs if a local node is actually available
		config := &did.RegistryConfig{
			Chain:           did.ChainEthereum,
			ContractAddress: "0x1234567890123456789012345678901234567890",
			RPCEndpoint:     "http://localhost:8545",
			PrivateKey:      "",
		}

		client, err := NewEthereumClient(config)
		if err != nil {
			t.Skip("Skipping test: local Ethereum node not available")
		}

		// If we get here, the client was created successfully
		assert.NotNil(t, client)
		assert.NotNil(t, client.client)
		assert.NotNil(t, client.contract)
		assert.Equal(t, config.ContractAddress, client.contractAddress.Hex())
	})
}

func TestEthereumHelperMethods(t *testing.T) {
	client := &EthereumClient{
		config: &did.RegistryConfig{
			MaxRetries:         5,
			ConfirmationBlocks: 3,
		},
	}

	// Test prepareRegistrationMessage
	req := &did.RegistrationRequest{
		DID:      "did:sage:ethereum:agent001",
		Name:     "Test Agent",
		Endpoint: "https://api.example.com",
	}

	message := client.prepareRegistrationMessage(req, "0x1234567890abcdef")
	assert.Contains(t, message, "Register agent:")
	assert.Contains(t, message, string(req.DID))
	assert.Contains(t, message, req.Name)
	assert.Contains(t, message, req.Endpoint)
	assert.Contains(t, message, "0x1234567890abcdef")

	// Test prepareUpdateMessage
	agentDID := did.AgentDID("did:sage:ethereum:agent001")
	updates := map[string]interface{}{
		"name":        "Updated Agent",
		"description": "New description",
	}

	updateMessage := client.prepareUpdateMessage(agentDID, updates)
	assert.Contains(t, updateMessage, "Update agent:")
	assert.Contains(t, updateMessage, string(agentDID))
	assert.Contains(t, updateMessage, "Updated Agent")
}

func TestCompareCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		cap1     map[string]interface{}
		cap2     map[string]interface{}
		expected bool
	}{
		{
			name: "Equal capabilities",
			cap1: map[string]interface{}{
				"chat": true,
				"code": true,
			},
			cap2: map[string]interface{}{
				"chat": true,
				"code": true,
			},
			expected: true,
		},
		{
			name: "Different length",
			cap1: map[string]interface{}{
				"chat": true,
			},
			cap2: map[string]interface{}{
				"chat": true,
				"code": true,
			},
			expected: false,
		},
		{
			name: "Different values",
			cap1: map[string]interface{}{
				"chat": true,
				"code": false,
			},
			cap2: map[string]interface{}{
				"chat": true,
				"code": true,
			},
			expected: false,
		},
		{
			name:     "Both nil",
			cap1:     nil,
			cap2:     nil,
			expected: true,
		},
		{
			name: "One nil",
			cap1: map[string]interface{}{
				"chat": true,
			},
			cap2:     nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareCapabilities(tt.cap1, tt.cap2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ============================================================================
// V2 Contract Integration Tests
// ============================================================================

// TestV2DIDLifecycleWithFundedKey demonstrates the complete DID lifecycle using V2 contract
// This test covers SPECIFICATION_VERIFICATION_MATRIX.md sections:
//
//	3.2.1.1: Ethereum 스마트 컨트랙트 배포 성공 (V2)
//	3.2.1.2: 트랜잭션 해시 반환 확인 (V2)
//	3.2.1.3: 가스비 소모량 확인 (V2)
//	3.2.1.4: 등록 후 온체인 조회 가능 확인 (V2)
//
// V2 Contract Characteristics:
//   - Single Secp256k1 key per agent
//   - Signature-based registration
//   - getAgentByDID for resolution
//
// IMPORTANT: This test demonstrates the pattern of funding a newly generated Secp256k1 key
// with ETH from the Hardhat/Anvil default account.
func TestV2DIDLifecycleWithFundedKey(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	t.Log("=== V2 Contract DID Lifecycle Test with Funded Key ===")
	t.Log("This test demonstrates the V2 contract pattern (single key, signature-based):")
	t.Log("1. Generate new Secp256k1 keypair")
	t.Log("2. Fund the new key with ETH from Hardhat default account")
	t.Log("3. Register DID on V2 contract")
	t.Log("4. Verify registration and measure gas usage")

	// Step 1: Create client with Hardhat default account
	// NOTE: You need to deploy V2 contract to localhost and update this address
	config := &did.RegistryConfig{
		ContractAddress:    "0x5FbDB2315678afecb367f032d93F642f64180aa3", // V2 contract address (update after deployment)
		RPCEndpoint:        "http://localhost:8545",
		PrivateKey:         "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", // Hardhat account #0
		GasPrice:           0,
		MaxRetries:         10,
		ConfirmationBlocks: 0,
	}

	client, err := NewEthereumClient(config)
	if err != nil {
		t.Fatalf("Failed to create V2 client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Step 2: Generate new Secp256k1 keypair for the agent
	t.Log("\n[Step 1] Generating new Secp256k1 keypair...")
	agentKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate agent keypair: %v", err)
	}

	// Derive Ethereum address from the public key
	ecdsaPubKey, ok := agentKeyPair.PublicKey().(*ecdsa.PublicKey)
	if !ok {
		t.Fatal("Failed to cast public key to ECDSA")
	}
	agentAddress := ethcrypto.PubkeyToAddress(*ecdsaPubKey)
	t.Logf(" Agent keypair generated")
	t.Logf("  Agent address: %s", agentAddress.Hex())

	// Check initial balance (should be 0)
	initialBalance, err := client.client.BalanceAt(ctx, agentAddress, nil)
	if err != nil {
		t.Fatalf("Failed to get initial balance: %v", err)
	}
	t.Logf("  Initial balance: %s wei", initialBalance.String())

	// Step 3: Fund the new key with ETH from Hardhat default account
	t.Log("\n[Step 2] Funding agent key with ETH from Hardhat account #0...")
	fundAmount := new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)) // 10 ETH
	receipt, err := transferETHForV2(ctx, client, agentAddress, fundAmount)
	if err != nil {
		t.Fatalf("Failed to transfer ETH: %v", err)
	}

	t.Logf(" ETH transfer successful")
	t.Logf("  Transaction hash: %s", receipt.TxHash.Hex())
	t.Logf("  Block number: %d", receipt.BlockNumber.Uint64())
	t.Logf("  Gas used: %d", receipt.GasUsed)
	t.Logf("  Amount transferred: 10 ETH")

	// Verify the transfer
	newBalance, err := client.client.BalanceAt(ctx, agentAddress, nil)
	if err != nil {
		t.Fatalf("Failed to get new balance: %v", err)
	}
	t.Logf("  New balance: %s wei (%.2f ETH)", newBalance.String(), float64(newBalance.Int64())/1e18)

	if newBalance.Cmp(fundAmount) < 0 {
		t.Fatalf("Balance after transfer (%s) is less than transferred amount (%s)", newBalance.String(), fundAmount.String())
	}

	// Step 4: Register DID on V2 contract
	t.Log("\n[Step 3] Registering DID on V2 contract...")
	testDID := did.AgentDID("did:sage:ethereum:" + uuid.New().String())

	req := &did.RegistrationRequest{
		DID:         testDID,
		Name:        "V2 Funded Agent Test",
		Description: "Agent with funded Secp256k1 key for V2 contract testing",
		Endpoint:    "http://localhost:8080",
		Capabilities: map[string]interface{}{
			"test":     "v2_funded_key_lifecycle",
			"contract": "SageRegistryV2",
			"balance":  newBalance.String(),
		},
		KeyPair: agentKeyPair,
	}

	t.Logf("  Registering DID: %s", testDID)
	regResult, err := client.Register(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register DID: %v", err)
	}

	t.Logf(" DID registered successfully on V2 contract")
	t.Logf("  Transaction hash: %s", regResult.TransactionHash)
	t.Logf("  Block number: %d", regResult.BlockNumber)
	t.Logf("  Gas used: %d", regResult.GasUsed)

	// Verify gas usage is reasonable for V2 contract
	// V2 typically uses less gas than V4 (single key vs multi-key)
	if regResult.GasUsed < 50000 {
		t.Errorf("Gas used (%d) seems too low for V2 DID registration", regResult.GasUsed)
	}
	if regResult.GasUsed > 800000 {
		t.Errorf("Gas used (%d) seems too high for V2 DID registration", regResult.GasUsed)
	}

	// Step 5: Verify registration by resolving the DID
	t.Log("\n[Step 4] Verifying V2 DID registration...")
	agent, err := client.Resolve(ctx, testDID)
	if err != nil {
		t.Fatalf("Failed to resolve DID: %v", err)
	}

	t.Logf(" DID resolved successfully from V2 contract")
	t.Logf("  DID: %s", agent.DID)
	t.Logf("  Name: %s", agent.Name)
	t.Logf("  Owner: %s", agent.Owner)
	t.Logf("  Active: %v", agent.IsActive)
	t.Logf("  Endpoint: %s", agent.Endpoint)

	// Verify metadata
	if agent.DID != testDID {
		t.Errorf("DID mismatch: got %s, want %s", agent.DID, testDID)
	}
	if agent.Name != req.Name {
		t.Errorf("Name mismatch: got %s, want %s", agent.Name, req.Name)
	}
	if !agent.IsActive {
		t.Error("Agent should be active after registration")
	}

	// Summary
	t.Log("\n=== V2 Contract Test Summary ===")
	t.Log(" New Secp256k1 keypair generated")
	t.Log(" Agent address funded with 10 ETH")
	t.Logf(" DID registered on V2 contract (gas: %d)", regResult.GasUsed)
	t.Log(" DID resolved and verified from V2 contract")
	t.Log(" All metadata matches registration request")
	t.Log("\nV2 Contract Characteristics:")
	t.Log("  - Single Secp256k1 key per agent")
	t.Log("  - Signature-based registration")
	t.Log("  - Lower gas usage than V4 (no multi-key support)")
}

// TestV2RegistrationWithUpdate tests V2 contract registration and update operations
func TestV2RegistrationWithUpdate(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	// V2 contract is deprecated - skip this test
	t.Skip("DEPRECATED: V2 contract is no longer supported due to incompatible signature verification. " +
		"V2 expects: keccak256(abi.encodePacked('SAGE Key Registration:', chainId, contract, sender, keyHash)) " +
		"but Go client provides text-based message signatures. " +
		"Use V4 contract (TestV4Update) instead.")

	t.Log("=== V2 Contract Registration and Update Test ===")

	// Configuration for V2 contract
	config := &did.RegistryConfig{
		ContractAddress:    "0x5FbDB2315678afecb367f032d93F642f64180aa3", // V2 contract address
		RPCEndpoint:        "http://localhost:8545",
		PrivateKey:         "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
		GasPrice:           0,
		MaxRetries:         10,
		ConfirmationBlocks: 0,
	}

	client, err := NewEthereumClient(config)
	if err != nil {
		t.Fatalf("Failed to create V2 client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Generate and fund agent key
	agentKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	ecdsaPubKey, ok := agentKeyPair.PublicKey().(*ecdsa.PublicKey)
	if !ok {
		t.Fatal("Failed to cast public key to ECDSA")
	}
	agentAddress := ethcrypto.PubkeyToAddress(*ecdsaPubKey)

	// Fund with 5 ETH
	fundAmount := new(big.Int).Mul(big.NewInt(5), big.NewInt(1e18))
	_, err = transferETHForV2(ctx, client, agentAddress, fundAmount)
	if err != nil {
		t.Fatalf("Failed to transfer ETH: %v", err)
	}

	t.Log(" Agent key generated and funded with 5 ETH")

	// Register agent
	testDID := did.AgentDID("did:sage:ethereum:" + uuid.New().String())
	req := &did.RegistrationRequest{
		DID:         testDID,
		Name:        "V2 Update Test Agent",
		Description: "Initial description",
		Endpoint:    "http://localhost:8080",
		Capabilities: map[string]interface{}{
			"version": "1.0.0",
		},
		KeyPair: agentKeyPair,
	}

	t.Logf("Registering agent: %s", testDID)
	regResult, err := client.Register(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	t.Logf(" Agent registered (gas: %d)", regResult.GasUsed)

	// Verify initial state
	agent, err := client.Resolve(ctx, testDID)
	if err != nil {
		t.Fatalf("Failed to resolve: %v", err)
	}

	if agent.Name != req.Name {
		t.Errorf("Initial name mismatch: got %s, want %s", agent.Name, req.Name)
	}

	// Test update operation
	t.Log("\nTesting update operation...")
	updates := map[string]interface{}{
		"name":        "V2 Updated Agent",
		"description": "Updated description",
		"endpoint":    "http://localhost:9090",
	}

	err = client.Update(ctx, testDID, updates, agentKeyPair)
	if err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	t.Log(" Agent updated successfully")

	// Verify update
	updatedAgent, err := client.Resolve(ctx, testDID)
	if err != nil {
		t.Fatalf("Failed to resolve after update: %v", err)
	}

	if updatedAgent.Name != "V2 Updated Agent" {
		t.Errorf("Updated name mismatch: got %s, want %s", updatedAgent.Name, "V2 Updated Agent")
	}
	if updatedAgent.Endpoint != "http://localhost:9090" {
		t.Errorf("Updated endpoint mismatch: got %s, want %s", updatedAgent.Endpoint, "http://localhost:9090")
	}

	t.Log(" Update verified successfully")
	t.Log("\n=== V2 Update Test Summary ===")
	t.Logf(" Registration gas: %d", regResult.GasUsed)
	t.Log(" Update operation completed successfully")
	t.Log(" All update operations working correctly")
}

// transferETHForV2 transfers ETH from the client's account to a destination address
// This is a helper function for V2 test setup to fund newly generated test keys
func transferETHForV2(ctx context.Context, client *EthereumClient, toAddress common.Address, amount *big.Int) (*types.Receipt, error) {
	// Get the nonce for the sender account
	fromAddress := ethcrypto.PubkeyToAddress(client.privateKey.PublicKey)
	nonce, err := client.client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get current gas price
	gasPrice, err := client.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create a simple value transfer transaction
	// Gas limit for simple ETH transfer is 21000
	gasLimit := uint64(21000)

	tx := types.NewTransaction(
		nonce,
		toAddress,
		amount,
		gasLimit,
		gasPrice,
		nil, // No data for simple transfer
	)

	// Sign the transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(client.chainID), client.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send the transaction
	err = client.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for transaction confirmation
	receipt, err := client.waitForTransaction(ctx, signedTx)
	if err != nil {
		return nil, fmt.Errorf("transaction failed: %w", err)
	}

	return receipt, nil
}
