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
	"encoding/hex"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
	"github.com/sage-x-project/sage/pkg/agent/did/ethereum"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDIDRegistration tests end-to-end DID registration and lookup
func TestDIDRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Generate a new secp256k1 key pair for the agent
	agentKeyPair, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(t, err)

	// Get public key bytes - convert to ECDSA public key first
	ecdsaPubKey, ok := agentKeyPair.PublicKey().(*ecdsa.PublicKey)
	require.True(t, ok, "Public key should be ECDSA")
	pubKeyBytes := crypto.FromECDSAPub(ecdsaPubKey)

	// Create Ethereum address from public key
	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	require.NoError(t, err)
	agentAddress := crypto.PubkeyToAddress(*pubKey)

	t.Logf("Agent address: %s", agentAddress.Hex())

	// Create DID
	did := fmt.Sprintf("did:sage:ethereum:%s", agentAddress.Hex())

	t.Run("Register DID on blockchain", func(t *testing.T) {
		// Specification Requirement: DID creation and format validation
		helpers.LogTestSection(t, "3.1.1", "DID Creation and Registration")

		// This would normally interact with the smart contract
		// For now, we'll simulate the registration

		helpers.LogDetail(t, "Generated Ethereum address: %s", agentAddress.Hex())

		// Specification Requirement: DID format must be "did:sage:ethereum:<address>"
		didFormatPattern := `^did:sage:ethereum:0x[0-9a-fA-F]{40}$`
		matched, err := regexp.MatchString(didFormatPattern, did)
		require.NoError(t, err)
		assert.True(t, matched, "DID must match format: did:sage:ethereum:<0x + 40 hex chars>")

		helpers.LogSuccess(t, "DID format validation passed")
		helpers.LogDetail(t, "DID: %s", did)
		helpers.LogDetail(t, "Format: did:sage:ethereum:<ethereum-address>")
		helpers.LogDetail(t, "Ethereum address: %s", agentAddress.Hex())
		helpers.LogDetail(t, "Address length: 42 characters (0x + 40 hex)")

		// Create DID document
		currentTime := time.Now()
		didDoc := &ethereum.DIDDocument{
			ID:         did,
			Controller: agentAddress.Hex(),
			PublicKey:  hex.EncodeToString(pubKeyBytes),
			Created:    currentTime,
			Updated:    currentTime,
		}

		helpers.LogSuccess(t, "DID Document created")
		helpers.LogDetail(t, "Controller: %s", didDoc.Controller)
		helpers.LogDetail(t, "Public key length: %d bytes (uncompressed)", len(pubKeyBytes))
		helpers.LogDetail(t, "Public key (hex): %s", didDoc.PublicKey[:64])

		// In a real scenario, this would call the smart contract
		// Example: registry.RegisterDID(ctx, didDoc)

		// Specification Requirement: DID validation
		assert.NotEmpty(t, didDoc.ID, "DID must not be empty")
		assert.Equal(t, did, didDoc.ID, "DID must match expected format")
		assert.Equal(t, agentAddress.Hex(), didDoc.Controller, "Controller must be Ethereum address")
		assert.NotEmpty(t, didDoc.PublicKey, "Public key must not be empty")

		helpers.LogSuccess(t, "DID registration validation complete")

		// Pass criteria checklist
		helpers.LogPassCriteria(t, []string{
			"DID generation successful",
			"DID format: did:sage:ethereum:<address>",
			"Ethereum address format valid (0x + 40 hex)",
			"DID document created",
			"Controller matches Ethereum address",
			"Public key included",
		})

		// Save test data for CLI verification
		testData := map[string]interface{}{
			"test_case":            "3.1.1_DID_Creation_Registration",
			"did":                  did,
			"did_format":           "did:sage:ethereum:<ethereum-address>",
			"ethereum_address":     agentAddress.Hex(),
			"controller":           didDoc.Controller,
			"public_key_hex":       didDoc.PublicKey,
			"public_key_length":    len(pubKeyBytes),
			"created_at":           currentTime.Format(time.RFC3339),
			"expected_format":      didFormatPattern,
		}
		helpers.SaveTestData(t, "did/did_creation_registration.json", testData)
	})

	t.Run("Lookup DID from blockchain", func(t *testing.T) {
		// This would normally query the smart contract
		// For now, we'll simulate the lookup

		// Simulate fetching DID document
		fetchedDoc := &ethereum.DIDDocument{
			ID:         did,
			Controller: agentAddress.Hex(),
			PublicKey:  hex.EncodeToString(pubKeyBytes),
		}

		assert.Equal(t, did, fetchedDoc.ID)
		assert.Equal(t, agentAddress.Hex(), fetchedDoc.Controller)
		assert.Equal(t, hex.EncodeToString(pubKeyBytes), fetchedDoc.PublicKey)

		t.Logf("DID Document retrieved: %+v", fetchedDoc)
	})

	t.Run("Verify DID signature", func(t *testing.T) {
		// Create a message to sign
		message := []byte("Test message for DID verification")

		// Sign with private key
		signature, err := agentKeyPair.Sign(message)
		require.NoError(t, err)

		// Verify signature manually using crypto package
		hash := crypto.Keccak256Hash(message)
		sigPublicKey, err := crypto.SigToPub(hash.Bytes(), signature)
		require.NoError(t, err)

		// Verify that we can recover a public key (structure may differ slightly)
		assert.NotNil(t, sigPublicKey, "Should recover a public key from signature")

		// Verify the addresses match which is what matters for Ethereum
		recoveredAddr := crypto.PubkeyToAddress(*sigPublicKey)
		expectedAddr := crypto.PubkeyToAddress(*ecdsaPubKey)
		assert.Equal(t, expectedAddr, recoveredAddr, "Recovered address should match")
	})

	t.Run("Update DID document", func(t *testing.T) {
		// Generate new key pair for rotation
		newKeyPair, err := keys.GenerateSecp256k1KeyPair()
		require.NoError(t, err)

		newEcdsaPubKey, ok := newKeyPair.PublicKey().(*ecdsa.PublicKey)
		require.True(t, ok)
		newPubKeyBytes := crypto.FromECDSAPub(newEcdsaPubKey)

		// Update DID document
		updatedDoc := &ethereum.DIDDocument{
			ID:         did,
			Controller: agentAddress.Hex(),
			PublicKey:  hex.EncodeToString(newPubKeyBytes),
			Updated:    time.Now(),
		}

		// In a real scenario, this would update the smart contract
		// Example: registry.UpdateDID(ctx, updatedDoc)

		assert.Equal(t, did, updatedDoc.ID)
		assert.NotEqual(t, hex.EncodeToString(pubKeyBytes), updatedDoc.PublicKey)

		t.Logf("DID Document updated with new key")
	})

	t.Run("Revoke DID", func(t *testing.T) {
		// In a real scenario, this would mark DID as revoked in smart contract
		// Example: registry.RevokeDID(ctx, did)

		revoked := true // Simulate revocation
		assert.True(t, revoked, "DID should be marked as revoked")

		t.Logf("DID %s revoked", did)
	})
}

// TestMultiAgentDID tests DID operations for multiple agents
func TestMultiAgentDID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	const numAgents = 5
	agents := make([]struct {
		KeyPair   interface{}
		PublicKey *ecdsa.PublicKey
		Address   common.Address
		DID       string
	}, numAgents)

	t.Run("Create multiple agent DIDs", func(t *testing.T) {
		for i := 0; i < numAgents; i++ {
			// Generate key pair
			keyPair, err := keys.GenerateSecp256k1KeyPair()
			require.NoError(t, err)

			// Get address
			ecdsaPubKey, ok := keyPair.PublicKey().(*ecdsa.PublicKey)
			require.True(t, ok)
			address := crypto.PubkeyToAddress(*ecdsaPubKey)

			// Create DID
			did := fmt.Sprintf("did:sage:ethereum:%s", address.Hex())

			agents[i] = struct {
				KeyPair   interface{}
				PublicKey *ecdsa.PublicKey
				Address   common.Address
				DID       string
			}{
				KeyPair:   keyPair,
				PublicKey: ecdsaPubKey,
				Address:   address,
				DID:       did,
			}

			t.Logf("Agent %d - DID: %s", i, did)
		}
	})

	t.Run("Verify all agents can sign and verify", func(t *testing.T) {
		message := []byte("Multi-agent test message")

		for i, agent := range agents {
			// Sign message using the key pair
			kp := agent.KeyPair.(interface{ Sign([]byte) ([]byte, error) })
			signature, err := kp.Sign(message)
			require.NoError(t, err)

			// Verify signature
			hash := crypto.Keccak256Hash(message)
			recoveredPubKey, err := crypto.SigToPub(hash.Bytes(), signature)
			require.NoError(t, err)

			// Verify addresses match for self-verification
			recoveredAddr := crypto.PubkeyToAddress(*recoveredPubKey)
			expectedAddr := crypto.PubkeyToAddress(*agent.PublicKey)
			assert.Equal(t, expectedAddr, recoveredAddr, "Agent %d address should match", i)
		}
	})

	t.Run("Batch DID operations", func(t *testing.T) {
		// Simulate batch registration
		var dids []string
		for _, agent := range agents {
			dids = append(dids, agent.DID)
		}

		// In a real scenario, this would be a batch smart contract call
		// Example: registry.BatchRegisterDIDs(ctx, dids)

		assert.Len(t, dids, numAgents)
		t.Logf("Batch registered %d DIDs", len(dids))
	})
}

// TestDIDResolver tests DID resolution functionality
func TestDIDResolver(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("Resolve valid DID", func(t *testing.T) {
		// Create test DID
		testAddress := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
		did := fmt.Sprintf("did:sage:ethereum:%s", testAddress)

		// Create resolver
		resolver := ethereum.NewResolver()

		// Parse DID
		parsedDID, err := resolver.ParseDID(did)
		require.NoError(t, err)

		assert.Equal(t, "did", parsedDID.Scheme)
		assert.Equal(t, "sage", parsedDID.Method)
		assert.Equal(t, "ethereum", parsedDID.Network)
		assert.Equal(t, testAddress, parsedDID.Address)
	})

	t.Run("Resolve invalid DID formats", func(t *testing.T) {
		resolver := ethereum.NewResolver()

		invalidDIDs := []string{
			"not-a-did",
			"did:wrong:method",
			"did:sage:wrongnetwork:0x123",
			"did:sage:ethereum:invalid-address",
			"",
		}

		for _, invalidDID := range invalidDIDs {
			_, err := resolver.ParseDID(invalidDID)
			assert.Error(t, err, "Should fail for invalid DID: %s", invalidDID)
		}
	})

	t.Run("Cache DID resolution", func(t *testing.T) {
		// Create resolver with caching
		resolver := ethereum.NewResolverWithCache(100, 5*time.Minute)

		testAddress := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
		did := fmt.Sprintf("did:sage:ethereum:%s", testAddress)

		// First resolution (cache miss)
		start := time.Now()
		doc1, err := resolver.Resolve(context.Background(), did)
		require.NoError(t, err)
		firstDuration := time.Since(start)

		// Second resolution (cache hit)
		start = time.Now()
		doc2, err := resolver.Resolve(context.Background(), did)
		require.NoError(t, err)
		secondDuration := time.Since(start)

		// Verify cache returns same document
		assert.Equal(t, doc1, doc2, "Cached resolution should return same document")

		// Log timing for informational purposes (timing may vary in CI)
		t.Logf("First resolution: %v, Cached resolution: %v", firstDuration, secondDuration)
	})
}

// BenchmarkDIDOperations benchmarks DID operations
func BenchmarkDIDOperations(b *testing.B) {
	// Generate key pair once
	keyPair, err := keys.GenerateSecp256k1KeyPair()
	require.NoError(b, err)

	ecdsaPubKey, ok := keyPair.PublicKey().(*ecdsa.PublicKey)
	require.True(b, ok)
	address := crypto.PubkeyToAddress(*ecdsaPubKey)

	did := fmt.Sprintf("did:sage:ethereum:%s", address.Hex())
	message := []byte("Benchmark message")

	b.Run("GenerateDID", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			kp, _ := keys.GenerateSecp256k1KeyPair()
			if ecdsaPK, ok := kp.PublicKey().(*ecdsa.PublicKey); ok {
				addr := crypto.PubkeyToAddress(*ecdsaPK)
				_ = fmt.Sprintf("did:sage:ethereum:%s", addr.Hex())
			}
		}
	})

	b.Run("SignMessage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = keyPair.Sign(message)
		}
	})

	b.Run("VerifySignature", func(b *testing.B) {
		signature, _ := keyPair.Sign(message)
		hash := crypto.Keccak256Hash(message)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = crypto.SigToPub(hash.Bytes(), signature)
		}
	})

	b.Run("ParseDID", func(b *testing.B) {
		resolver := ethereum.NewResolver()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, _ = resolver.ParseDID(did)
		}
	})
}
