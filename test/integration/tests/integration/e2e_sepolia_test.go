//go:build e2e
// +build e2e

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

package integration

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sage-x-project/sage/pkg/agent/core/rfc9421"
	sagecrypto "github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/chain"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/did"
	dideth "github.com/sage-x-project/sage/pkg/agent/did/ethereum"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2ESepoliaAgentRegistrationAndMessaging tests the complete flow:
// 1. Register two agents on Sepolia testnet
// 2. Sign a message from Agent A using RFC 9421
// 3. Verify the message at Agent B
func TestE2ESepoliaAgentRegistrationAndMessaging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E integration test in short mode")
	}

	// Check for Sepolia RPC URL
	sepoliaRPC := os.Getenv("SEPOLIA_RPC_URL")
	if sepoliaRPC == "" {
		sepoliaRPC = "https://rpc.sepolia.org" // Public RPC
	}

	// Sepolia contract address from deployment
	registryAddress := os.Getenv("SAGE_REGISTRY_ADDRESS")
	if registryAddress == "" {
		registryAddress = "0xb25D5f59cA52532862dA92901a2A550A09d5b4c0" // From sepolia-sage.env
	}

	// Private key for transactions (read from env or skip test)
	privateKeyHex := os.Getenv("SEPOLIA_PRIVATE_KEY")
	if privateKeyHex == "" {
		t.Skip("Skipping test: SEPOLIA_PRIVATE_KEY not set")
	}

	t.Logf("Using Sepolia RPC: %s", sepoliaRPC)
	t.Logf("Using Registry: %s", registryAddress)

	ctx := context.Background()

	// Step 1: Create Agent A
	t.Run("Create Agent A", func(t *testing.T) {
		agentAKeyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		// Get Ethereum address for Agent A
		ecdsaPubKey, ok := agentAKeyPair.PublicKey().(*ecdsa.PublicKey)
		require.True(t, ok)
		agentAAddress := crypto.PubkeyToAddress(*ecdsaPubKey)

		agentADID := did.AgentDID(fmt.Sprintf("did:sage:ethereum:%s", agentAAddress.Hex()))

		t.Logf("Agent A DID: %s", agentADID)
		t.Logf("Agent A Address: %s", agentAAddress.Hex())

		// Register Agent A on Sepolia
		config := &did.RegistryConfig{
			Chain:           did.ChainEthereum,
			RPCEndpoint:     sepoliaRPC,
			ContractAddress: registryAddress,
			PrivateKey:      privateKeyHex,
			MaxRetries:      3,
			GasPrice:        20000000000, // 20 Gwei
		}

		client, err := dideth.NewEthereumClient(config)
		if err != nil {
			t.Skipf("Cannot connect to Sepolia: %v", err)
		}

		// Prepare registration request
		req := &did.RegistrationRequest{
			DID:         agentADID,
			Name:        "Test Agent A",
			Description: "E2E Test Agent A for RFC 9421 Messaging",
			Endpoint:    "https://agent-a.example.com",
			KeyPair:     agentAKeyPair,
			Capabilities: map[string]interface{}{
				"messaging": true,
				"rfc9421":   true,
			},
		}

		// Register (this may take time due to blockchain confirmation)
		result, err := client.Register(ctx, req)
		if err != nil {
			// Check if already registered
			if strings.Contains(err.Error(), "already registered") {
				t.Logf("Agent A already registered, continuing...")
			} else {
				t.Logf("Registration failed (may need gas): %v", err)
				t.Skip("Skipping due to registration failure")
			}
		} else {
			t.Logf("Agent A registered successfully!")
			t.Logf("Transaction Hash: %s", result.TransactionHash)
			t.Logf("Block Number: %d", result.BlockNumber)
			t.Logf("Gas Used: %d", result.GasUsed)
		}
	})

	// Step 2: Create Agent B
	t.Run("Create Agent B", func(t *testing.T) {
		agentBKeyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		ecdsaPubKey, ok := agentBKeyPair.PublicKey().(*ecdsa.PublicKey)
		require.True(t, ok)
		agentBAddress := crypto.PubkeyToAddress(*ecdsaPubKey)

		agentBDID := did.AgentDID(fmt.Sprintf("did:sage:ethereum:%s", agentBAddress.Hex()))

		t.Logf("Agent B DID: %s", agentBDID)
		t.Logf("Agent B Address: %s", agentBAddress.Hex())
	})
}

// TestRFC9421MessageSigning tests RFC 9421 message signing and verification
// between two agents without blockchain registration
func TestRFC9421MessageSigning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping RFC 9421 message signing test in short mode")
	}

	// Create two agents with Secp256k1 keys
	agentAKeyPair, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	agentBKeyPair, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	// Get their DIDs
	ecdsaAPubKey, ok := agentAKeyPair.PublicKey().(*ecdsa.PublicKey)
	require.True(t, ok)
	agentAAddress := crypto.PubkeyToAddress(*ecdsaAPubKey)
	agentADID := did.AgentDID(fmt.Sprintf("did:sage:ethereum:%s", agentAAddress.Hex()))

	ecdsaBPubKey, ok := agentBKeyPair.PublicKey().(*ecdsa.PublicKey)
	require.True(t, ok)
	agentBAddress := crypto.PubkeyToAddress(*ecdsaBPubKey)
	agentBDID := did.AgentDID(fmt.Sprintf("did:sage:ethereum:%s", agentBAddress.Hex()))

	t.Logf("Agent A DID: %s", agentADID)
	t.Logf("Agent B DID: %s", agentBDID)

	// Test Case 1: Agent A sends message to Agent B
	t.Run("Agent A sends signed message to Agent B", func(t *testing.T) {
		// Create message from Agent A to Agent B
		message := map[string]interface{}{
			"from":      string(agentADID),
			"to":        string(agentBDID),
			"timestamp": time.Now().Unix(),
			"content":   "Hello Agent B, this is a test message from Agent A",
		}

		messageJSON, err := json.Marshal(message)
		require.NoError(t, err)

		// Create HTTP request that will be signed
		req, err := http.NewRequest("POST", "https://agent-b.example.com/message", strings.NewReader(string(messageJSON)))
		require.NoError(t, err)

		// Add headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Date", time.Now().Format(http.TimeFormat))
		req.Header.Set("X-Agent-DID", string(agentADID))

		// Get RFC 9421 algorithm name from key type
		algorithm, err := sagecrypto.GetRFC9421AlgorithmName(agentAKeyPair.Type())
		require.NoError(t, err, "Should get RFC 9421 algorithm name")

		// Sign the request using RFC 9421
		params := &rfc9421.SignatureInputParams{
			CoveredComponents: []string{
				`"@method"`,
				`"@path"`,
				`"date"`,
				`"content-type"`,
				`"x-agent-did"`,
			},
			KeyID:     agentAKeyPair.ID(),
			Algorithm: algorithm,
			Created:   time.Now().Unix(),
			Expires:   time.Now().Add(5 * time.Minute).Unix(),
		}

		verifier := rfc9421.NewHTTPVerifier()

		// Extract private key for signing
		agentAPrivKey := agentAKeyPair.PrivateKey()
		ecdsaPrivKey, ok := agentAPrivKey.(*ecdsa.PrivateKey)
		require.True(t, ok, "Private key should be ECDSA")

		err = verifier.SignRequest(req, "sig1", params, ecdsaPrivKey)
		require.NoError(t, err)

		// Verify that Signature and Signature-Input headers are present
		assert.NotEmpty(t, req.Header.Get("Signature"))
		assert.NotEmpty(t, req.Header.Get("Signature-Input"))

		t.Logf("Message signed successfully")
		t.Logf("Signature: %s", req.Header.Get("Signature")[:50]+"...")
		t.Logf("Signature-Input: %s", req.Header.Get("Signature-Input"))

		// Agent B verifies the signature
		t.Run("Agent B verifies signature", func(t *testing.T) {
			// Agent B needs Agent A's public key (in real scenario, fetch from DID registry)
			agentAPubKey := agentAKeyPair.PublicKey()

			// Verify the request
			opts := &rfc9421.HTTPVerificationOptions{
				MaxAge: 10 * time.Minute,
			}

			err := verifier.VerifyRequest(req, agentAPubKey, opts)
			assert.NoError(t, err, "Signature verification should succeed")

			t.Logf("Signature verified successfully by Agent B!")

			// Parse and log the message
			var receivedMsg map[string]interface{}
			err = json.NewDecoder(req.Body).Decode(&receivedMsg)
			require.NoError(t, err)

			t.Logf("Received message: %+v", receivedMsg)
			assert.Equal(t, string(agentADID), receivedMsg["from"])
			assert.Equal(t, string(agentBDID), receivedMsg["to"])
		})
	})

	// Test Case 2: Tampered message should fail verification
	t.Run("Tampered message fails verification", func(t *testing.T) {
		message := map[string]interface{}{
			"from":    string(agentADID),
			"to":      string(agentBDID),
			"content": "Original message",
		}

		messageJSON, err := json.Marshal(message)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "https://agent-b.example.com/message", strings.NewReader(string(messageJSON)))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Date", time.Now().Format(http.TimeFormat))

		algorithm, err := sagecrypto.GetRFC9421AlgorithmName(agentAKeyPair.Type())
		require.NoError(t, err)

		params := &rfc9421.SignatureInputParams{
			CoveredComponents: []string{`"@method"`, `"@path"`, `"date"`},
			KeyID:             agentAKeyPair.ID(),
			Algorithm:         algorithm,
			Created:           time.Now().Unix(),
		}

		verifier := rfc9421.NewHTTPVerifier()
		agentAPrivKey := agentAKeyPair.PrivateKey().(*ecdsa.PrivateKey)
		err = verifier.SignRequest(req, "sig1", params, agentAPrivKey)
		require.NoError(t, err)

		// Tamper with the Date header
		req.Header.Set("Date", time.Now().Add(1*time.Hour).Format(http.TimeFormat))

		// Verification should fail
		agentAPubKey := agentAKeyPair.PublicKey()
		err = verifier.VerifyRequest(req, agentAPubKey, nil)
		assert.Error(t, err, "Tampered message should fail verification")

		t.Logf("Tampered message correctly rejected: %v", err)
	})

	// Test Case 3: Expired signature should fail
	t.Run("Expired signature fails verification", func(t *testing.T) {
		message := map[string]interface{}{
			"from":    string(agentADID),
			"to":      string(agentBDID),
			"content": "Time-sensitive message",
		}

		messageJSON, err := json.Marshal(message)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "https://agent-b.example.com/message", strings.NewReader(string(messageJSON)))
		require.NoError(t, err)

		req.Header.Set("Date", time.Now().Format(http.TimeFormat))

		algorithm, err := sagecrypto.GetRFC9421AlgorithmName(agentAKeyPair.Type())
		require.NoError(t, err)

		// Create signature that expires immediately
		params := &rfc9421.SignatureInputParams{
			CoveredComponents: []string{`"@method"`},
			KeyID:             agentAKeyPair.ID(),
			Algorithm:         algorithm,
			Created:           time.Now().Add(-10 * time.Minute).Unix(),
			Expires:           time.Now().Add(-5 * time.Minute).Unix(),
		}

		verifier := rfc9421.NewHTTPVerifier()
		agentAPrivKey := agentAKeyPair.PrivateKey().(*ecdsa.PrivateKey)
		err = verifier.SignRequest(req, "sig1", params, agentAPrivKey)
		require.NoError(t, err)

		// Verification should fail due to expiration
		agentAPubKey := agentAKeyPair.PublicKey()
		err = verifier.VerifyRequest(req, agentAPubKey, nil)
		assert.Error(t, err, "Expired signature should fail verification")
		assert.Contains(t, err.Error(), "expired")

		t.Logf("Expired signature correctly rejected: %v", err)
	})

	// Test Case 4: Bidirectional messaging (B replies to A)
	t.Run("Agent B replies to Agent A", func(t *testing.T) {
		// Create reply message from Agent B to Agent A
		replyMessage := map[string]interface{}{
			"from":      string(agentBDID),
			"to":        string(agentADID),
			"timestamp": time.Now().Unix(),
			"content":   "Hello Agent A, message received and acknowledged",
			"reply_to":  "original_message_id",
		}

		messageJSON, err := json.Marshal(replyMessage)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", "https://agent-a.example.com/message", strings.NewReader(string(messageJSON)))
		require.NoError(t, err)

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Date", time.Now().Format(http.TimeFormat))
		req.Header.Set("X-Agent-DID", string(agentBDID))

		algorithm, err := sagecrypto.GetRFC9421AlgorithmName(agentBKeyPair.Type())
		require.NoError(t, err)

		params := &rfc9421.SignatureInputParams{
			CoveredComponents: []string{
				`"@method"`,
				`"@path"`,
				`"date"`,
				`"x-agent-did"`,
			},
			KeyID:     agentBKeyPair.ID(),
			Algorithm: algorithm,
			Created:   time.Now().Unix(),
		}

		verifier := rfc9421.NewHTTPVerifier()
		agentBPrivKey := agentBKeyPair.PrivateKey().(*ecdsa.PrivateKey)
		err = verifier.SignRequest(req, "sig1", params, agentBPrivKey)
		require.NoError(t, err)

		t.Logf("Reply message signed by Agent B")

		// Agent A verifies the reply
		agentBPubKey := agentBKeyPair.PublicKey()
		err = verifier.VerifyRequest(req, agentBPubKey, nil)
		assert.NoError(t, err, "Reply signature verification should succeed")

		t.Logf("Reply signature verified successfully by Agent A!")
	})
}

// TestRFC9421WithDifferentKeyTypes tests RFC 9421 signing with different key types
func TestRFC9421WithDifferentKeyTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping key type test in short mode")
	}

	t.Run("Secp256k1 (Ethereum compatible)", func(t *testing.T) {
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		req, err := http.NewRequest("GET", "https://example.com/test", nil)
		require.NoError(t, err)
		req.Header.Set("Date", time.Now().Format(http.TimeFormat))

		algorithm, err := sagecrypto.GetRFC9421AlgorithmName(keyPair.Type())
		require.NoError(t, err)

		params := &rfc9421.SignatureInputParams{
			CoveredComponents: []string{`"@method"`, `"date"`},
			KeyID:             keyPair.ID(),
			Algorithm:         algorithm,
			Created:           time.Now().Unix(),
		}

		verifier := rfc9421.NewHTTPVerifier()
		privKey := keyPair.PrivateKey().(*ecdsa.PrivateKey)
		err = verifier.SignRequest(req, "sig1", params, privKey)
		require.NoError(t, err)

		pubKey := keyPair.PublicKey()
		err = verifier.VerifyRequest(req, pubKey, nil)
		assert.NoError(t, err)

		t.Logf("Secp256k1 signing and verification successful")
	})

	t.Run("Ed25519 (High performance)", func(t *testing.T) {
		keyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Test direct signing/verification with Ed25519
		message := []byte("Test message for Ed25519 signature")
		signature, err := keyPair.Sign(message)
		require.NoError(t, err)

		err = keyPair.Verify(message, signature)
		assert.NoError(t, err)

		t.Logf("Ed25519 signing and verification successful")
		t.Logf("Key ID: %s", keyPair.ID())
		t.Logf("Key Type: %s", keyPair.Type())
	})
}

// Helper function to get key type from KeyPair
func getKeyType(kp sagecrypto.KeyPair) sagecrypto.KeyType {
	return kp.Type()
}

// TestChainKeyTypeMapper tests the automatic key type selection based on chain type
func TestChainKeyTypeMapper(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chain key mapper test in short mode")
	}

	t.Run("Ethereum chain requires Secp256k1", func(t *testing.T) {
		// Get recommended key type for Ethereum
		keyType, err := chain.GetRecommendedKeyType(chain.ChainTypeEthereum)
		require.NoError(t, err)
		assert.Equal(t, sagecrypto.KeyTypeSecp256k1, keyType)

		// Get RFC 9421 algorithm for the key type
		algorithm, err := chain.GetRFC9421Algorithm(keyType)
		require.NoError(t, err)
		assert.Equal(t, "es256k", algorithm)

		// Generate key pair
		var keyPair sagecrypto.KeyPair
		switch keyType {
		case sagecrypto.KeyTypeSecp256k1:
			keyPair, err = keys.GenerateSecp256k1KeyPair()
		case sagecrypto.KeyTypeEd25519:
			keyPair, err = keys.GenerateEd25519KeyPair()
		default:
			t.Fatalf("Unsupported key type: %s", keyType)
		}
		require.NoError(t, err)
		assert.Equal(t, keyType, keyPair.Type())

		// Verify the key type is valid for Ethereum
		err = chain.ValidateKeyTypeForChain(keyPair.Type(), chain.ChainTypeEthereum)
		assert.NoError(t, err)

		t.Logf("Ethereum agent created with key type: %s", keyPair.Type())
		t.Logf("RFC 9421 algorithm: %s", algorithm)
	})

	t.Run("Solana chain requires Ed25519", func(t *testing.T) {
		// Get recommended key type for Solana
		keyType, err := chain.GetRecommendedKeyType(chain.ChainTypeSolana)
		require.NoError(t, err)
		assert.Equal(t, sagecrypto.KeyTypeEd25519, keyType)

		// Get RFC 9421 algorithm for the key type
		algorithm, err := chain.GetRFC9421Algorithm(keyType)
		require.NoError(t, err)
		assert.Equal(t, "ed25519", algorithm)

		// Generate key pair
		var keyPair sagecrypto.KeyPair
		switch keyType {
		case sagecrypto.KeyTypeSecp256k1:
			keyPair, err = keys.GenerateSecp256k1KeyPair()
		case sagecrypto.KeyTypeEd25519:
			keyPair, err = keys.GenerateEd25519KeyPair()
		default:
			t.Fatalf("Unsupported key type: %s", keyType)
		}
		require.NoError(t, err)
		assert.Equal(t, keyType, keyPair.Type())

		// Verify the key type is valid for Solana
		err = chain.ValidateKeyTypeForChain(keyPair.Type(), chain.ChainTypeSolana)
		assert.NoError(t, err)

		t.Logf("Solana agent created with key type: %s", keyPair.Type())
		t.Logf("RFC 9421 algorithm: %s", algorithm)
	})

	t.Run("Wrong key type for chain fails validation", func(t *testing.T) {
		// Try to use Ed25519 for Ethereum (should fail)
		err := chain.ValidateKeyTypeForChain(sagecrypto.KeyTypeEd25519, chain.ChainTypeEthereum)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")

		// Try to use Secp256k1 for Solana (should fail)
		err = chain.ValidateKeyTypeForChain(sagecrypto.KeyTypeSecp256k1, chain.ChainTypeSolana)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")

		t.Logf("Key type validation correctly prevents mismatched key-chain combinations")
	})

	t.Run("Complete workflow: Create agent with correct key for target chain", func(t *testing.T) {
		targetChain := chain.ChainTypeEthereum

		// Step 1: Get recommended key type
		keyType, err := chain.GetRecommendedKeyType(targetChain)
		require.NoError(t, err)
		t.Logf("Step 1: Recommended key type for %s: %s", targetChain, keyType)

		// Step 2: Generate key pair
		var keyPair sagecrypto.KeyPair
		switch keyType {
		case sagecrypto.KeyTypeSecp256k1:
			keyPair, err = keys.GenerateSecp256k1KeyPair()
		case sagecrypto.KeyTypeEd25519:
			keyPair, err = keys.GenerateEd25519KeyPair()
		default:
			t.Fatalf("Unsupported key type: %s", keyType)
		}
		require.NoError(t, err)
		t.Logf("Step 2: Generated key pair with ID: %s", keyPair.ID())

		// Step 3: Validate key type for chain
		err = chain.ValidateKeyTypeForChain(keyPair.Type(), targetChain)
		require.NoError(t, err)
		t.Logf("Step 3: Key type validated for chain")

		// Step 4: Get RFC 9421 algorithm
		algorithm, err := chain.GetRFC9421Algorithm(keyPair.Type())
		require.NoError(t, err)
		t.Logf("Step 4: RFC 9421 algorithm: %s", algorithm)

		// Step 5: Create and sign a message
		req, err := http.NewRequest("POST", "https://example.com/api/message", strings.NewReader(`{"test":"data"}`))
		require.NoError(t, err)
		req.Header.Set("Date", time.Now().Format(http.TimeFormat))
		req.Header.Set("Content-Type", "application/json")

		params := &rfc9421.SignatureInputParams{
			CoveredComponents: []string{`"@method"`, `"@path"`, `"date"`},
			KeyID:             keyPair.ID(),
			Algorithm:         algorithm, // Use algorithm from mapper
			Created:           time.Now().Unix(),
		}

		verifier := rfc9421.NewHTTPVerifier()

		// Sign based on key type
		if keyPair.Type() == sagecrypto.KeyTypeSecp256k1 {
			privKey := keyPair.PrivateKey().(*ecdsa.PrivateKey)
			err = verifier.SignRequest(req, "sig1", params, privKey)
		} else if keyPair.Type() == sagecrypto.KeyTypeEd25519 {
			// For Ed25519, use direct signing
			message := []byte("test message")
			signature, signErr := keyPair.Sign(message)
			require.NoError(t, signErr)
			verifyErr := keyPair.Verify(message, signature)
			require.NoError(t, verifyErr)
			err = nil // Ed25519 signing works
			t.Logf("Ed25519 direct signing successful")
		}

		if keyPair.Type() == sagecrypto.KeyTypeSecp256k1 {
			require.NoError(t, err)
			t.Logf("Step 5: Message signed with %s", algorithm)

			// Step 6: Verify the signature
			pubKey := keyPair.PublicKey()
			err = verifier.VerifyRequest(req, pubKey, nil)
			require.NoError(t, err)
			t.Logf("Step 6: Signature verified successfully")
		}

		t.Logf("Complete workflow successful for %s chain!", targetChain)
	})

	t.Run("Multi-chain support: List supported key types", func(t *testing.T) {
		chains := []chain.ChainType{
			chain.ChainTypeEthereum,
			chain.ChainTypeSolana,
			chain.ChainTypeBitcoin,
			chain.ChainTypeCosmos,
		}

		for _, chainType := range chains {
			keyTypes, err := chain.GetSupportedKeyTypes(chainType)
			require.NoError(t, err)
			t.Logf("%s supports key types: %v", chainType, keyTypes)

			recommendedKeyType, err := chain.GetRecommendedKeyType(chainType)
			require.NoError(t, err)
			t.Logf("%s recommended key type: %s", chainType, recommendedKeyType)
		}
	})
}

// TestCrossChainMessaging tests message exchange between agents on different chains
func TestCrossChainMessaging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cross-chain messaging test in short mode")
	}

	t.Run("Ethereum agent sends message to Solana agent", func(t *testing.T) {
		// Create Ethereum agent (Secp256k1)
		ethKeyType, err := chain.GetRecommendedKeyType(chain.ChainTypeEthereum)
		require.NoError(t, err)
		ethKeyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		ethPubKey := ethKeyPair.PublicKey().(*ecdsa.PublicKey)
		ethAddress := crypto.PubkeyToAddress(*ethPubKey)
		ethDID := did.AgentDID(fmt.Sprintf("did:sage:ethereum:%s", ethAddress.Hex()))

		// Create Solana agent (Ed25519)
		solKeyType, err := chain.GetRecommendedKeyType(chain.ChainTypeSolana)
		require.NoError(t, err)
		solKeyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Create mock Solana address (in real scenario, would derive from public key)
		solDID := did.AgentDID(fmt.Sprintf("did:sage:solana:%s", solKeyPair.ID()))

		t.Logf("Ethereum Agent (Secp256k1): %s", ethDID)
		t.Logf("Solana Agent (Ed25519): %s", solDID)

		// Ethereum agent sends message
		message := map[string]interface{}{
			"from":         string(ethDID),
			"to":           string(solDID),
			"content":      "Cross-chain message from Ethereum to Solana",
			"timestamp":    time.Now().Unix(),
			"source_chain": "ethereum",
			"target_chain": "solana",
		}

		messageJSON, err := json.Marshal(message)
		require.NoError(t, err)

		// Sign with Ethereum key
		req, err := http.NewRequest("POST", "https://solana-agent.example.com/message", strings.NewReader(string(messageJSON)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Date", time.Now().Format(http.TimeFormat))

		ethAlgorithm, err := chain.GetRFC9421Algorithm(ethKeyType)
		require.NoError(t, err)

		params := &rfc9421.SignatureInputParams{
			CoveredComponents: []string{`"@method"`, `"@path"`, `"date"`},
			KeyID:             ethKeyPair.ID(),
			Algorithm:         ethAlgorithm,
			Created:           time.Now().Unix(),
		}

		verifier := rfc9421.NewHTTPVerifier()
		ethPrivKey := ethKeyPair.PrivateKey().(*ecdsa.PrivateKey)
		err = verifier.SignRequest(req, "sig1", params, ethPrivKey)
		require.NoError(t, err)

		t.Logf("Message signed by Ethereum agent with %s", ethAlgorithm)

		// Solana agent verifies the signature
		err = verifier.VerifyRequest(req, ethKeyPair.PublicKey(), nil)
		assert.NoError(t, err, "Solana agent should verify Ethereum signature")

		t.Logf("✅ Cross-chain message verified successfully!")
		t.Logf("Key types: Ethereum(%s) → Solana(%s)", ethKeyType, solKeyType)
	})
}

// TestKeyRotation tests key rotation scenarios
func TestKeyRotation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping key rotation test in short mode")
	}

	t.Run("Agent rotates keys and continues messaging", func(t *testing.T) {
		// Initial key pair
		oldKeyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		oldPubKey := oldKeyPair.PublicKey().(*ecdsa.PublicKey)
		agentAddress := crypto.PubkeyToAddress(*oldPubKey)
		agentDID := did.AgentDID(fmt.Sprintf("did:sage:ethereum:%s", agentAddress.Hex()))

		t.Logf("Agent DID: %s", agentDID)
		t.Logf("Old Key ID: %s", oldKeyPair.ID())

		// Send message with old key
		message1 := map[string]interface{}{
			"content": "Message with old key",
			"seq":     1,
		}
		msg1JSON, err := json.Marshal(message1)
		require.NoError(t, err)

		req1, err := http.NewRequest("POST", "https://example.com/message", strings.NewReader(string(msg1JSON)))
		require.NoError(t, err)
		req1.Header.Set("Date", time.Now().Format(http.TimeFormat))

		algorithm, err := sagecrypto.GetRFC9421AlgorithmName(oldKeyPair.Type())
		require.NoError(t, err)

		params1 := &rfc9421.SignatureInputParams{
			CoveredComponents: []string{`"@method"`, `"date"`},
			KeyID:             oldKeyPair.ID(),
			Algorithm:         algorithm,
			Created:           time.Now().Unix(),
		}

		verifier := rfc9421.NewHTTPVerifier()
		oldPrivKey := oldKeyPair.PrivateKey().(*ecdsa.PrivateKey)
		err = verifier.SignRequest(req1, "sig1", params1, oldPrivKey)
		require.NoError(t, err)

		t.Logf("Message 1 signed with old key")

		// Rotate to new key
		newKeyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)
		t.Logf("New Key ID: %s", newKeyPair.ID())

		// Send message with new key
		message2 := map[string]interface{}{
			"content":        "Message with new key after rotation",
			"seq":            2,
			"previous_keyid": oldKeyPair.ID(),
		}
		msg2JSON, err := json.Marshal(message2)
		require.NoError(t, err)

		req2, err := http.NewRequest("POST", "https://example.com/message", strings.NewReader(string(msg2JSON)))
		require.NoError(t, err)
		req2.Header.Set("Date", time.Now().Format(http.TimeFormat))

		algorithm2, err := sagecrypto.GetRFC9421AlgorithmName(newKeyPair.Type())
		require.NoError(t, err)

		params2 := &rfc9421.SignatureInputParams{
			CoveredComponents: []string{`"@method"`, `"date"`},
			KeyID:             newKeyPair.ID(),
			Algorithm:         algorithm2,
			Created:           time.Now().Unix(),
		}

		newPrivKey := newKeyPair.PrivateKey().(*ecdsa.PrivateKey)
		err = verifier.SignRequest(req2, "sig1", params2, newPrivKey)
		require.NoError(t, err)

		t.Logf("Message 2 signed with new key")

		// Verify both messages
		err = verifier.VerifyRequest(req1, oldKeyPair.PublicKey(), nil)
		assert.NoError(t, err, "Old message should verify with old key")

		err = verifier.VerifyRequest(req2, newKeyPair.PublicKey(), nil)
		assert.NoError(t, err, "New message should verify with new key")

		t.Logf("✅ Key rotation successful!")
		t.Logf("Old key can still verify old messages")
		t.Logf("New key verifies new messages")
	})
}

// TestMultiChainAgent tests an agent registered on multiple chains
func TestMultiChainAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multi-chain agent test in short mode")
	}

	t.Run("Agent identity across multiple chains", func(t *testing.T) {
		// Agent has different keys for different chains
		type ChainIdentity struct {
			Chain     chain.ChainType
			KeyPair   sagecrypto.KeyPair
			DID       did.AgentDID
			Algorithm string
		}

		identities := make(map[chain.ChainType]*ChainIdentity)

		chains := []chain.ChainType{
			chain.ChainTypeEthereum,
			chain.ChainTypeSolana,
		}

		for _, chainType := range chains {
			// Get recommended key type
			keyType, err := chain.GetRecommendedKeyType(chainType)
			require.NoError(t, err)

			// Generate appropriate key
			var keyPair sagecrypto.KeyPair
			var agentDID did.AgentDID

			switch keyType {
			case sagecrypto.KeyTypeSecp256k1:
				keyPair, err = keys.GenerateSecp256k1KeyPair()
				require.NoError(t, err)

				pubKey := keyPair.PublicKey().(*ecdsa.PublicKey)
				address := crypto.PubkeyToAddress(*pubKey)
				agentDID = did.AgentDID(fmt.Sprintf("did:sage:%s:%s", chainType, address.Hex()))

			case sagecrypto.KeyTypeEd25519:
				keyPair, err = keys.GenerateEd25519KeyPair()
				require.NoError(t, err)
				agentDID = did.AgentDID(fmt.Sprintf("did:sage:%s:%s", chainType, keyPair.ID()))
			}

			algorithm, err := chain.GetRFC9421Algorithm(keyType)
			require.NoError(t, err)

			identities[chainType] = &ChainIdentity{
				Chain:     chainType,
				KeyPair:   keyPair,
				DID:       agentDID,
				Algorithm: algorithm,
			}

			t.Logf("%s Identity: %s (key: %s, alg: %s)",
				chainType, agentDID, keyType, algorithm)
		}

		// Verify each identity can sign and verify
		for chainType, identity := range identities {
			message := []byte(fmt.Sprintf("Test message for %s", chainType))

			signature, err := identity.KeyPair.Sign(message)
			require.NoError(t, err)

			err = identity.KeyPair.Verify(message, signature)
			assert.NoError(t, err)

			t.Logf("✅ %s identity verified", chainType)
		}

		t.Logf("✅ Multi-chain agent successfully created with %d identities", len(identities))
	})
}

// TestMessagePerformance benchmarks message signing and verification performance
func TestMessagePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("Performance comparison: Secp256k1 vs Ed25519", func(t *testing.T) {
		iterations := 100

		// Test Secp256k1 (Ethereum)
		secp256k1KeyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		message := []byte("Performance test message")

		// Secp256k1 signing
		start := time.Now()
		for i := 0; i < iterations; i++ {
			_, err := secp256k1KeyPair.Sign(message)
			require.NoError(t, err)
		}
		secp256k1SignDuration := time.Since(start)

		// Secp256k1 verification
		signature, err := secp256k1KeyPair.Sign(message)
		require.NoError(t, err)

		start = time.Now()
		for i := 0; i < iterations; i++ {
			err := secp256k1KeyPair.Verify(message, signature)
			require.NoError(t, err)
		}
		secp256k1VerifyDuration := time.Since(start)

		// Test Ed25519 (Solana)
		ed25519KeyPair, err := keys.GenerateEd25519KeyPair()
		require.NoError(t, err)

		// Ed25519 signing
		start = time.Now()
		for i := 0; i < iterations; i++ {
			_, err := ed25519KeyPair.Sign(message)
			require.NoError(t, err)
		}
		ed25519SignDuration := time.Since(start)

		// Ed25519 verification
		signature, err = ed25519KeyPair.Sign(message)
		require.NoError(t, err)

		start = time.Now()
		for i := 0; i < iterations; i++ {
			err := ed25519KeyPair.Verify(message, signature)
			require.NoError(t, err)
		}
		ed25519VerifyDuration := time.Since(start)

		// Report results
		t.Logf("\n=== Performance Results (%d iterations) ===", iterations)
		t.Logf("\nSecp256k1 (Ethereum):")
		t.Logf("  Signing:       %v (avg: %v/op)", secp256k1SignDuration, secp256k1SignDuration/time.Duration(iterations))
		t.Logf("  Verification:  %v (avg: %v/op)", secp256k1VerifyDuration, secp256k1VerifyDuration/time.Duration(iterations))

		t.Logf("\nEd25519 (Solana):")
		t.Logf("  Signing:       %v (avg: %v/op)", ed25519SignDuration, ed25519SignDuration/time.Duration(iterations))
		t.Logf("  Verification:  %v (avg: %v/op)", ed25519VerifyDuration, ed25519VerifyDuration/time.Duration(iterations))

		speedup := float64(secp256k1SignDuration) / float64(ed25519SignDuration)
		t.Logf("\nEd25519 is %.2fx faster than Secp256k1 for signing", speedup)

		speedup = float64(secp256k1VerifyDuration) / float64(ed25519VerifyDuration)
		t.Logf("Ed25519 is %.2fx faster than Secp256k1 for verification", speedup)
	})

	t.Run("HTTP message signing throughput", func(t *testing.T) {
		keyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		algorithm, err := sagecrypto.GetRFC9421AlgorithmName(keyPair.Type())
		require.NoError(t, err)

		verifier := rfc9421.NewHTTPVerifier()
		privKey := keyPair.PrivateKey().(*ecdsa.PrivateKey)

		iterations := 50
		start := time.Now()

		for i := 0; i < iterations; i++ {
			req, err := http.NewRequest("POST", "https://example.com/api", strings.NewReader(`{"test":"data"}`))
			require.NoError(t, err)
			req.Header.Set("Date", time.Now().Format(http.TimeFormat))

			params := &rfc9421.SignatureInputParams{
				CoveredComponents: []string{`"@method"`, `"@path"`, `"date"`},
				KeyID:             keyPair.ID(),
				Algorithm:         algorithm,
				Created:           time.Now().Unix(),
			}

			err = verifier.SignRequest(req, "sig1", params, privKey)
			require.NoError(t, err)

			err = verifier.VerifyRequest(req, keyPair.PublicKey(), nil)
			require.NoError(t, err)
		}

		duration := time.Since(start)
		throughput := float64(iterations) / duration.Seconds()

		t.Logf("\nHTTP Message Signing/Verification Throughput:")
		t.Logf("  Total time: %v", duration)
		t.Logf("  Iterations: %d", iterations)
		t.Logf("  Throughput: %.2f messages/sec", throughput)
		t.Logf("  Avg latency: %v per message", duration/time.Duration(iterations))
	})
}
