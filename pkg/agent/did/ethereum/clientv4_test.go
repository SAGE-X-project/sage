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
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	_ "github.com/sage-x-project/sage/internal/cryptoinit" // Initialize crypto wrappers
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// TestV4SingleKeyRegistration tests single-key agent registration with V4 contract
func TestV4SingleKeyRegistration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	// Configuration for local Hardhat/Anvil network
	config := &did.RegistryConfig{
		ContractAddress: "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9", // From deployment (X25519 removed)
		RPCEndpoint:     "http://localhost:8545",
		PrivateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", // Hardhat test account #0
		GasPrice:        0, // Let the client determine gas price
		MaxRetries:      10,
		ConfirmationBlocks: 0, // No need to wait for confirmations on local network
	}

	// Create V4 client
	client, err := NewEthereumClientV4(config)
	if err != nil {
		t.Fatalf("Failed to create V4 client: %v", err)
	}

	// Generate a test keypair
	keyPair, err := crypto.GenerateSecp256k1KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	// Create registration request
	testDID := did.AgentDID("did:sage:eth:test-agent-" + time.Now().Format("20060102150405"))
	req := &did.RegistrationRequest{
		DID:         testDID,
		Name:        "Test Agent V4",
		Description: "Single-key test agent for V4 contract",
		Endpoint:    "http://localhost:8080",
		Capabilities: map[string]interface{}{
			"protocols": []string{"http", "grpc"},
			"version":   "1.0.0",
		},
		KeyPair: keyPair,
	}

	// Register the agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("Registering agent with V4 contract...")
	result, err := client.Register(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	t.Logf("✓ Agent registered successfully!")
	t.Logf("  Transaction Hash: %s", result.TransactionHash)
	t.Logf("  Block Number: %d", result.BlockNumber)
	t.Logf("  Gas Used: %d", result.GasUsed)

	// Verify registration by resolving the agent
	t.Log("Resolving registered agent...")
	agent, err := client.Resolve(ctx, testDID)
	if err != nil {
		t.Fatalf("Failed to resolve agent: %v", err)
	}

	// Verify metadata
	if agent.DID != testDID {
		t.Errorf("DID mismatch: got %s, want %s", agent.DID, testDID)
	}
	if agent.Name != req.Name {
		t.Errorf("Name mismatch: got %s, want %s", agent.Name, req.Name)
	}
	if agent.Description != req.Description {
		t.Errorf("Description mismatch: got %s, want %s", agent.Description, req.Description)
	}
	if agent.Endpoint != req.Endpoint {
		t.Errorf("Endpoint mismatch: got %s, want %s", agent.Endpoint, req.Endpoint)
	}
	if !agent.IsActive {
		t.Error("Agent should be active")
	}

	t.Logf("✓ Agent resolved successfully!")
	t.Logf("  DID: %s", agent.DID)
	t.Logf("  Name: %s", agent.Name)
	t.Logf("  Owner: %s", agent.Owner)
	t.Logf("  Active: %v", agent.IsActive)
}

// TestV4Ed25519Registration tests Ed25519 key registration with contract owner approval
func TestV4Ed25519Registration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	// Configuration for local Hardhat/Anvil network
	config := &did.RegistryConfig{
		ContractAddress: "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9", // From deployment (X25519 removed)
		RPCEndpoint:     "http://localhost:8545",
		PrivateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", // Hardhat test account #0
		GasPrice:        0, // Let the client determine gas price
		MaxRetries:      10,
		ConfirmationBlocks: 0, // No need to wait for confirmations on local network
	}

	// Create V4 client
	client, err := NewEthereumClientV4(config)
	if err != nil {
		t.Fatalf("Failed to create V4 client: %v", err)
	}

	// Generate Ed25519 keypair
	ed25519KeyPair, err := crypto.GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 keypair: %v", err)
	}

	// Marshal Ed25519 public key
	ed25519PubKey, err := did.MarshalPublicKey(ed25519KeyPair.PublicKey())
	if err != nil {
		t.Fatalf("Failed to marshal Ed25519 public key: %v", err)
	}

	t.Logf("Ed25519 public key size: %d bytes", len(ed25519PubKey))

	// Create registration request with Ed25519 key
	testDID := did.AgentDID("did:sage:sol:test-agent-" + time.Now().Format("20060102150405"))
	req := &did.RegistrationRequest{
		DID:         testDID,
		Name:        "Test Agent Ed25519",
		Description: "Ed25519 key test agent for V4 contract (Solana compatible)",
		Endpoint:    "http://localhost:8080",
		Capabilities: map[string]interface{}{
			"protocols": []string{"http", "grpc"},
			"version":   "1.0.0",
			"chain":     "solana",
		},
		Keys: []did.AgentKey{
			{
				Type:    did.KeyTypeEd25519,
				KeyData: ed25519PubKey,
			},
		},
	}

	// Register the agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("Registering agent with Ed25519 key...")
	result, err := client.Register(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	t.Logf("✓ Agent registered successfully!")
	t.Logf("  Transaction Hash: %s", result.TransactionHash)
	t.Logf("  Block Number: %d", result.BlockNumber)
	t.Logf("  Gas Used: %d", result.GasUsed)

	// Verify registration by resolving the agent
	t.Log("Resolving registered agent...")
	agent, err := client.Resolve(ctx, testDID)
	if err != nil {
		t.Fatalf("Failed to resolve agent: %v", err)
	}

	if agent.DID != testDID {
		t.Errorf("DID mismatch: got %s, want %s", agent.DID, testDID)
	}

	t.Logf("✓ Agent resolved successfully!")
	t.Logf("  DID: %s", agent.DID)
	t.Logf("  Name: %s", agent.Name)

	// Get the key hash for the Ed25519 key
	keyHash, err := client.GetAgentKeyHash(ctx, testDID, ed25519PubKey, did.KeyTypeEd25519)
	if err != nil {
		t.Fatalf("Failed to get key hash: %v", err)
	}

	t.Logf("  Key Hash: 0x%x", keyHash)

	// Check key verification status before approval
	key, err := client.GetAgentKey(ctx, keyHash)
	if err != nil {
		t.Fatalf("Failed to get key details: %v", err)
	}

	t.Logf("✓ Key retrieved before approval")
	t.Logf("  Key Type: %d (Ed25519=%d)", key.KeyType, did.KeyTypeEd25519)
	t.Logf("  Verified: %v (should be false)", key.Verified)

	if key.Verified {
		t.Error("Ed25519 key should not be verified before owner approval")
	}

	// Contract owner approves the Ed25519 key
	t.Log("Contract owner approving Ed25519 key...")
	err = client.ApproveEd25519Key(ctx, keyHash)
	if err != nil {
		t.Fatalf("Failed to approve Ed25519 key: %v", err)
	}

	t.Log("✓ Ed25519 key approved by contract owner")

	// Check key verification status after approval
	keyAfter, err := client.GetAgentKey(ctx, keyHash)
	if err != nil {
		t.Fatalf("Failed to get key details after approval: %v", err)
	}

	t.Logf("✓ Key retrieved after approval")
	t.Logf("  Verified: %v (should be true)", keyAfter.Verified)

	if !keyAfter.Verified {
		t.Error("Ed25519 key should be verified after owner approval")
	}

	t.Log("✓ Ed25519 registration and approval flow completed successfully!")
}

// TestV4MultiKeyRegistration tests multi-key agent registration with pre-computed signatures
func TestV4MultiKeyRegistration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	// Configuration for local Hardhat/Anvil network
	config := &did.RegistryConfig{
		ContractAddress: "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9",
		RPCEndpoint:     "http://localhost:8545",
		PrivateKey:      "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
		GasPrice:        0,
		MaxRetries:      10,
		ConfirmationBlocks: 0,
	}

	// Create V4 client
	client, err := NewEthereumClientV4(config)
	if err != nil {
		t.Fatalf("Failed to create V4 client: %v", err)
	}

	// Generate keypairs for multi-key test (2 ECDSA keys + 1 Ed25519 key)
	ecdsaKeyPair1, err := crypto.GenerateSecp256k1KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate ECDSA keypair 1: %v", err)
	}

	ecdsaKeyPair2, err := crypto.GenerateSecp256k1KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate ECDSA keypair 2: %v", err)
	}

	ed25519KeyPair, err := crypto.GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 keypair: %v", err)
	}

	// Marshal public keys
	ecdsaPubKey1, err := did.MarshalPublicKey(ecdsaKeyPair1.PublicKey())
	if err != nil {
		t.Fatalf("Failed to marshal ECDSA public key 1: %v", err)
	}

	ecdsaPubKey2, err := did.MarshalPublicKey(ecdsaKeyPair2.PublicKey())
	if err != nil {
		t.Fatalf("Failed to marshal ECDSA public key 2: %v", err)
	}

	ed25519PubKey, err := did.MarshalPublicKey(ed25519KeyPair.PublicKey())
	if err != nil {
		t.Fatalf("Failed to marshal Ed25519 public key: %v", err)
	}

	// Prepare registration message
	testDID := did.AgentDID("did:sage:eth:multikey-" + time.Now().Format("20060102150405"))

	// Pre-compute signatures for ECDSA keys
	// Note: Each key owner signs with their own private key to prove ownership
	keyData := [][]byte{ecdsaPubKey1, ecdsaPubKey2, ed25519PubKey}
	keyTypes := []did.KeyType{did.KeyTypeECDSA, did.KeyTypeECDSA, did.KeyTypeEd25519}
	keyPairs := []crypto.KeyPair{ecdsaKeyPair1, ecdsaKeyPair2, ed25519KeyPair}

	// Generate signatures
	signatures, err := generateMultiKeySignatures(
		client,
		testDID,
		keyData,
		keyTypes,
		keyPairs,
	)
	if err != nil {
		t.Fatalf("Failed to generate signatures: %v", err)
	}

	// Create registration request with pre-signed keys
	req := &did.RegistrationRequest{
		DID:         testDID,
		Name:        "Multi-Key Test Agent V4",
		Description: "Multi-key test agent with ECDSA and Ed25519 keys",
		Endpoint:    "http://localhost:8080",
		Capabilities: map[string]interface{}{
			"protocols": []string{"http", "grpc"},
			"version":   "2.0.0",
			"multikey":  true,
			"chains":    []string{"ethereum", "solana"},
		},
		Keys: []did.AgentKey{
			{
				Type:      did.KeyTypeECDSA,
				KeyData:   ecdsaPubKey1,
				Signature: signatures[0],
			},
			{
				Type:      did.KeyTypeECDSA,
				KeyData:   ecdsaPubKey2,
				Signature: signatures[1],
			},
			{
				Type:      did.KeyTypeEd25519,
				KeyData:   ed25519PubKey,
				Signature: signatures[2], // Empty for Ed25519 on Ethereum
			},
		},
	}

	// Register the agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("Registering multi-key agent with pre-signed keys...")
	t.Logf("  Keys: 2x ECDSA (signed) + 1x Ed25519 (unsigned)")
	result, err := client.Register(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register multi-key agent: %v", err)
	}

	t.Logf("✓ Multi-key agent registered successfully!")
	t.Logf("  Transaction Hash: %s", result.TransactionHash)
	t.Logf("  Block Number: %d", result.BlockNumber)
	t.Logf("  Gas Used: %d", result.GasUsed)

	// Verify registration
	t.Log("Resolving multi-key agent...")
	agent, err := client.Resolve(ctx, testDID)
	if err != nil {
		t.Fatalf("Failed to resolve multi-key agent: %v", err)
	}

	t.Logf("✓ Multi-key agent resolved successfully!")
	t.Logf("  DID: %s", agent.DID)
	t.Logf("  Name: %s", agent.Name)
	t.Logf("  Keys: %d registered", len(req.Keys))

	// Verify each key
	for i, key := range req.Keys {
		keyHash, err := client.GetAgentKeyHash(ctx, testDID, key.KeyData, key.Type)
		if err != nil {
			t.Errorf("Failed to get key hash for key %d: %v", i, err)
			continue
		}

		keyInfo, err := client.GetAgentKey(ctx, keyHash)
		if err != nil {
			t.Errorf("Failed to get key info for key %d: %v", i, err)
			continue
		}

		t.Logf("  Key %d: Type=%d, Verified=%v", i, keyInfo.KeyType, keyInfo.Verified)

		// ECDSA keys should be verified immediately
		if key.Type == did.KeyTypeECDSA && !keyInfo.Verified {
			t.Errorf("ECDSA key %d should be verified", i)
		}

		// Ed25519 keys should NOT be verified until owner approves
		if key.Type == did.KeyTypeEd25519 && keyInfo.Verified {
			t.Errorf("Ed25519 key %d should NOT be verified before owner approval", i)
		}
	}

	t.Log("✓ Multi-key registration with pre-signed signatures completed successfully!")
}

// generateMultiKeySignatures generates signatures for multiple keys
// Each key is signed by its corresponding private key to prove ownership
//
// ETHEREUM SIGNATURE LOGIC:
//   - ECDSA keys: Sign with the corresponding secp256k1 private key
//   - Ed25519 keys: Return empty signature (no on-chain verification on Ethereum)
func generateMultiKeySignatures(
	client *EthereumClientV4,
	agentDID did.AgentDID,
	keyData [][]byte,
	keyTypes []did.KeyType,
	keyPairs []crypto.KeyPair,
) ([][]byte, error) {
	if len(keyData) != len(keyTypes) || len(keyData) != len(keyPairs) {
		return nil, crypto.ErrInvalidKeyType
	}

	signatures := make([][]byte, len(keyData))

	// Calculate agentId (same as contract: keccak256(abi.encode(did, firstKeyData)))
	stringType, _ := abi.NewType("string", "", nil)
	bytesType, _ := abi.NewType("bytes", "", nil)
	arguments := abi.Arguments{
		{Type: stringType},
		{Type: bytesType},
	}

	agentIdData, err := arguments.Pack(string(agentDID), keyData[0])
	if err != nil {
		return nil, err
	}

	agentId := ethcrypto.Keccak256Hash(agentIdData)

	// Get owner address from client's private key
	ownerAddress := ethcrypto.PubkeyToAddress(client.privateKey.PublicKey)

	// agentNonce is 0 for new registrations
	agentNonce := big.NewInt(0)

	// Generate signature for each key
	for i, keyType := range keyTypes {
		if keyType == did.KeyTypeEd25519 {
			// Ed25519 keys on Ethereum don't need signatures
			signatures[i] = []byte{}
			continue
		}

		if keyType == did.KeyTypeECDSA {
			// IMPORTANT: Contract verifies that msg.sender signed the message
			// NOT that the key owner signed it. All signatures must use client.privateKey
			// This proves that the transaction sender (msg.sender) authorizes adding this key

			// Contract expects: keccak256(abi.encode(agentId, keyData, msg.sender, agentNonce))
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
				return nil, err
			}

			messageHash := ethcrypto.Keccak256Hash(messageData)

			// Apply Ethereum personal sign prefix
			prefixedData := []byte("\x19Ethereum Signed Message:\n32")
			prefixedData = append(prefixedData, messageHash.Bytes()...)
			ethSignedHash := ethcrypto.Keccak256Hash(prefixedData)

			// Sign with client's private key (msg.sender)
			// This proves the transaction sender authorizes adding this key
			sig, err := ethcrypto.Sign(ethSignedHash.Bytes(), client.privateKey)
			if err != nil {
				return nil, err
			}

			// Adjust V value for Ethereum compatibility
			if sig[64] < 27 {
				sig[64] += 27
			}

			signatures[i] = sig
		}
	}

	return signatures, nil
}

// extractECDSAPrivateKey extracts the ECDSA private key from a KeyPair
func extractECDSAPrivateKey(keyPair crypto.KeyPair) (*ecdsa.PrivateKey, error) {
	// Get the private key interface
	privKey := keyPair.PrivateKey()

	// Type assert to *ecdsa.PrivateKey
	ecdsaPrivKey, ok := privKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, crypto.ErrInvalidKeyType
	}

	return ecdsaPrivKey, nil
}
