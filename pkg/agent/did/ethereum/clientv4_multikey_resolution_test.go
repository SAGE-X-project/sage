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
	"os"
	"testing"
	"time"

	_ "github.com/sage-x-project/sage/internal/cryptoinit" // Initialize crypto wrappers
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
)

// TestV4MultiKeyResolution tests the new multi-key resolution functions
// This verifies Task 2.2 implementation: ResolveAllPublicKeys and ResolvePublicKeyByType
func TestV4MultiKeyResolution(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	// Configuration for local Hardhat/Anvil network
	config := &did.RegistryConfig{
		ContractAddress:    "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9",
		RPCEndpoint:        "http://localhost:8545",
		PrivateKey:         "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
		GasPrice:           0,
		MaxRetries:         10,
		ConfirmationBlocks: 0,
	}

	// Create V4 client
	client, err := NewEthereumClientV4(config)
	if err != nil {
		t.Fatalf("Failed to create V4 client: %v", err)
	}

	// Generate multiple keypairs (2 ECDSA + 1 Ed25519)
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

	// Prepare registration
	testDID := did.AgentDID("did:sage:eth:multikey-resolution-" + time.Now().Format("20060102150405"))

	keyData := [][]byte{ecdsaPubKey1, ecdsaPubKey2, ed25519PubKey}
	keyTypes := []did.KeyType{did.KeyTypeECDSA, did.KeyTypeECDSA, did.KeyTypeEd25519}
	keyPairs := []crypto.KeyPair{ecdsaKeyPair1, ecdsaKeyPair2, ed25519KeyPair}

	// Generate signatures
	signatures, err := generateMultiKeySignatures(client, testDID, keyData, keyTypes, keyPairs)
	if err != nil {
		t.Fatalf("Failed to generate signatures: %v", err)
	}

	// Create registration request
	req := &did.RegistrationRequest{
		DID:         testDID,
		Name:        "Multi-Key Resolution Test Agent",
		Description: "Testing ResolveAllPublicKeys and ResolvePublicKeyByType",
		Endpoint:    "http://localhost:8080",
		Capabilities: map[string]interface{}{
			"test":   "multi_key_resolution",
			"chains": []string{"ethereum", "solana"},
		},
		Keys: []did.AgentKey{
			{Type: did.KeyTypeECDSA, KeyData: ecdsaPubKey1, Signature: signatures[0]},
			{Type: did.KeyTypeECDSA, KeyData: ecdsaPubKey2, Signature: signatures[1]},
			{Type: did.KeyTypeEd25519, KeyData: ed25519PubKey, Signature: signatures[2]},
		},
	}

	// Register the agent
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("Registering multi-key agent...")
	t.Logf("  Keys: 2x ECDSA + 1x Ed25519")
	result, err := client.Register(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	t.Logf("Agent registered successfully!")
	t.Logf("  Transaction Hash: %s", result.TransactionHash)
	t.Logf("  Gas Used: %d", result.GasUsed)

	// Approve Ed25519 key so it shows up in resolution
	ed25519KeyHash, err := client.GetAgentKeyHash(ctx, testDID, ed25519PubKey, did.KeyTypeEd25519)
	if err != nil {
		t.Fatalf("Failed to get Ed25519 key hash: %v", err)
	}

	t.Log("Approving Ed25519 key...")
	err = client.ApproveEd25519Key(ctx, ed25519KeyHash)
	if err != nil {
		t.Fatalf("Failed to approve Ed25519 key: %v", err)
	}
	t.Log("Ed25519 key approved")

	// TEST 1: ResolveAllPublicKeys()
	t.Log("")
	t.Log("TEST 1: ResolveAllPublicKeys() - retrieve all verified keys")
	allKeys, err := client.ResolveAllPublicKeys(ctx, testDID)
	if err != nil {
		t.Fatalf("Failed to resolve all public keys: %v", err)
	}

	t.Logf("Resolved %d verified keys (expected 3)", len(allKeys))

	if len(allKeys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(allKeys))
	}

	// Verify key types
	ecdsaCount := 0
	ed25519Count := 0

	for i, key := range allKeys {
		t.Logf("  Key %d: Type=%d, Verified=%v, CreatedAt=%v",
			i, key.Type, key.Verified, key.CreatedAt)

		if !key.Verified {
			t.Errorf("Key %d should be verified", i)
		}

		if key.Type == did.KeyTypeECDSA {
			ecdsaCount++
		} else if key.Type == did.KeyTypeEd25519 {
			ed25519Count++
		}
	}

	if ecdsaCount != 2 {
		t.Errorf("Expected 2 ECDSA keys, got %d", ecdsaCount)
	}

	if ed25519Count != 1 {
		t.Errorf("Expected 1 Ed25519 key, got %d", ed25519Count)
	}

	t.Log("ResolveAllPublicKeys() PASSED")

	// TEST 2: ResolvePublicKeyByType() - ECDSA
	t.Log("")
	t.Log("TEST 2: ResolvePublicKeyByType(ECDSA) - get ECDSA key")
	ecdsaKey, err := client.ResolvePublicKeyByType(ctx, testDID, did.KeyTypeECDSA)
	if err != nil {
		t.Fatalf("Failed to resolve ECDSA key by type: %v", err)
	}

	if ecdsaKey == nil {
		t.Fatal("Expected ECDSA key, got nil")
	}

	t.Logf("Successfully resolved ECDSA key: %T", ecdsaKey)
	t.Log("ResolvePublicKeyByType(ECDSA) PASSED")

	// TEST 3: ResolvePublicKeyByType() - Ed25519
	t.Log("")
	t.Log("TEST 3: ResolvePublicKeyByType(Ed25519) - get Ed25519 key")
	ed25519Key, err := client.ResolvePublicKeyByType(ctx, testDID, did.KeyTypeEd25519)
	if err != nil {
		t.Fatalf("Failed to resolve Ed25519 key by type: %v", err)
	}

	if ed25519Key == nil {
		t.Fatal("Expected Ed25519 key, got nil")
	}

	t.Logf("Successfully resolved Ed25519 key: %T", ed25519Key)
	t.Log("ResolvePublicKeyByType(Ed25519) PASSED")

	// TEST 4: Backward compatibility - ResolvePublicKey() should still work
	t.Log("")
	t.Log("TEST 4: ResolvePublicKey() - backward compatibility (returns first key)")
	firstKey, err := client.ResolvePublicKey(ctx, testDID)
	if err != nil {
		t.Fatalf("Failed to resolve first public key: %v", err)
	}

	if firstKey == nil {
		t.Fatal("Expected public key, got nil")
	}

	t.Logf("Successfully resolved first key: %T", firstKey)
	t.Log("ResolvePublicKey() backward compatibility PASSED")

	// Summary
	t.Log("")
	t.Log("MULTI-KEY RESOLUTION TEST PASSED!")
	t.Log("  - ResolveAllPublicKeys(): VERIFIED (returns all 3 verified keys)")
	t.Log("  - ResolvePublicKeyByType(ECDSA): VERIFIED")
	t.Log("  - ResolvePublicKeyByType(Ed25519): VERIFIED")
	t.Log("  - ResolvePublicKey() backward compatibility: VERIFIED")
	t.Log("")
	t.Log("Task 2.2: Multi-key Resolution - COMPLETED")
}

// TestV4MultiKeyResolutionFiltering tests that only verified keys are returned
func TestV4MultiKeyResolutionFiltering(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	// Configuration for local Hardhat/Anvil network
	config := &did.RegistryConfig{
		ContractAddress:    "0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9",
		RPCEndpoint:        "http://localhost:8545",
		PrivateKey:         "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
		GasPrice:           0,
		MaxRetries:         10,
		ConfirmationBlocks: 0,
	}

	client, err := NewEthereumClientV4(config)
	if err != nil {
		t.Fatalf("Failed to create V4 client: %v", err)
	}

	// Generate keypairs (1 ECDSA + 1 Ed25519)
	ecdsaKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate ECDSA keypair: %v", err)
	}

	ed25519KeyPair, err := crypto.GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 keypair: %v", err)
	}

	// Marshal public keys
	ecdsaPubKey, err := did.MarshalPublicKey(ecdsaKeyPair.PublicKey())
	if err != nil {
		t.Fatalf("Failed to marshal ECDSA public key: %v", err)
	}

	ed25519PubKey, err := did.MarshalPublicKey(ed25519KeyPair.PublicKey())
	if err != nil {
		t.Fatalf("Failed to marshal Ed25519 public key: %v", err)
	}

	// Register agent
	testDID := did.AgentDID("did:sage:eth:filter-test-" + time.Now().Format("20060102150405"))

	keyData := [][]byte{ecdsaPubKey, ed25519PubKey}
	keyTypes := []did.KeyType{did.KeyTypeECDSA, did.KeyTypeEd25519}
	keyPairs := []crypto.KeyPair{ecdsaKeyPair, ed25519KeyPair}

	signatures, err := generateMultiKeySignatures(client, testDID, keyData, keyTypes, keyPairs)
	if err != nil {
		t.Fatalf("Failed to generate signatures: %v", err)
	}

	req := &did.RegistrationRequest{
		DID:         testDID,
		Name:        "Filtering Test Agent",
		Description: "Testing that only verified keys are returned",
		Endpoint:    "http://localhost:8080",
		Capabilities: map[string]interface{}{
			"test": "filtering",
		},
		Keys: []did.AgentKey{
			{Type: did.KeyTypeECDSA, KeyData: ecdsaPubKey, Signature: signatures[0]},
			{Type: did.KeyTypeEd25519, KeyData: ed25519PubKey, Signature: signatures[1]},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Log("Registering agent with 1 ECDSA (verified) + 1 Ed25519 (unverified)...")
	_, err = client.Register(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	// ResolveAllPublicKeys should only return the verified ECDSA key
	t.Log("Resolving all public keys (should only return verified keys)...")
	allKeys, err := client.ResolveAllPublicKeys(ctx, testDID)
	if err != nil {
		t.Fatalf("Failed to resolve all public keys: %v", err)
	}

	t.Logf("Resolved %d verified keys (expected 1 - only ECDSA)", len(allKeys))

	if len(allKeys) != 1 {
		t.Errorf("Expected 1 verified key, got %d", len(allKeys))
	}

	if len(allKeys) > 0 && allKeys[0].Type != did.KeyTypeECDSA {
		t.Errorf("Expected ECDSA key, got type %d", allKeys[0].Type)
	}

	// ResolvePublicKeyByType for Ed25519 should fail since it's not verified
	t.Log("Attempting to resolve unverified Ed25519 key (should fail)...")
	_, err = client.ResolvePublicKeyByType(ctx, testDID, did.KeyTypeEd25519)
	if err == nil {
		t.Error("Expected error when resolving unverified Ed25519 key")
	} else {
		t.Logf("Correctly failed with: %v", err)
	}

	// Now approve the Ed25519 key and verify it appears
	ed25519KeyHash, err := client.GetAgentKeyHash(ctx, testDID, ed25519PubKey, did.KeyTypeEd25519)
	if err != nil {
		t.Fatalf("Failed to get Ed25519 key hash: %v", err)
	}

	t.Log("Approving Ed25519 key...")
	err = client.ApproveEd25519Key(ctx, ed25519KeyHash)
	if err != nil {
		t.Fatalf("Failed to approve Ed25519 key: %v", err)
	}

	// Now resolve should return 2 keys
	t.Log("Resolving all public keys after approval (should return 2 keys)...")
	allKeysAfter, err := client.ResolveAllPublicKeys(ctx, testDID)
	if err != nil {
		t.Fatalf("Failed to resolve all public keys after approval: %v", err)
	}

	t.Logf("Resolved %d verified keys after approval (expected 2)", len(allKeysAfter))

	if len(allKeysAfter) != 2 {
		t.Errorf("Expected 2 verified keys after approval, got %d", len(allKeysAfter))
	}

	// Now Ed25519 resolution should work
	t.Log("Resolving verified Ed25519 key (should succeed)...")
	_, err = client.ResolvePublicKeyByType(ctx, testDID, did.KeyTypeEd25519)
	if err != nil {
		t.Errorf("Failed to resolve verified Ed25519 key: %v", err)
	} else {
		t.Log("Successfully resolved verified Ed25519 key")
	}

	t.Log("")
	t.Log("FILTERING TEST PASSED!")
	t.Log("  - Only verified keys are returned: VERIFIED")
	t.Log("  - Ed25519 key filtered out when unverified: VERIFIED")
	t.Log("  - Ed25519 key appears after approval: VERIFIED")
}
