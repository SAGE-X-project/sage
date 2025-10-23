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
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/require"
)

// TestV4Update tests agent metadata update functionality (Spec 3.4.1)
func TestV4Update(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	helpers.LogTestSection(t, "3.4.1", "메타데이터 업데이트")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// ========================================
	// Setup: Create and register test agent
	// ========================================
	t.Log("[Setup] Generating keypair and creating client...")

	// Generate agent keypair
	agentKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err, "Failed to generate keypair")

	// Get agent's Ethereum address
	ecdsaPrivKey, ok := agentKeyPair.PrivateKey().(*ecdsa.PrivateKey)
	require.True(t, ok, "Failed to cast private key to ECDSA")
	agentPrivateKeyHex := fmt.Sprintf("%x", ecdsaPrivKey.D.Bytes())

	// Create client using agent's keypair
	agentConfig := &did.RegistryConfig{
		ContractAddress:    "0x5FbDB2315678afecb367f032d93F642f64180aa3",
		RPCEndpoint:        "http://localhost:8545",
		PrivateKey:         agentPrivateKeyHex,
		GasPrice:           0,
		MaxRetries:         10,
		ConfirmationBlocks: 0,
	}

	agentClient, err := NewEthereumClientV4(agentConfig)
	require.NoError(t, err, "Failed to create agent client")

	// Register test agent
	testDID := did.AgentDID("did:sage:ethereum:" + uuid.New().String())
	req := &did.RegistrationRequest{
		DID:         testDID,
		Name:        "Update Test Agent",
		Description: "Initial description for update test",
		Endpoint:    "http://localhost:8080",
		Capabilities: map[string]interface{}{
			"version":  "1.0.0",
			"features": []string{"chat", "search"},
		},
		KeyPair: agentKeyPair,
	}

	helpers.LogDetail(t, "[Setup] Registering test agent: %s", testDID)
	regResult, err := agentClient.Register(ctx, req)
	require.NoError(t, err, "Failed to register agent")
	helpers.LogDetail(t, "    ✓ Agent registered (gas: %d)", regResult.GasUsed)

	// Verify initial state
	initialAgent, err := agentClient.Resolve(ctx, testDID)
	require.NoError(t, err, "Failed to resolve agent")
	helpers.LogDetail(t, "    ✓ Initial state verified:")
	helpers.LogDetail(t, "      - Name: %s", initialAgent.Name)
	helpers.LogDetail(t, "      - Description: %s", initialAgent.Description)
	helpers.LogDetail(t, "      - Endpoint: %s", initialAgent.Endpoint)

	// ========================================
	// 3.4.1.1 메타데이터 업데이트 (Metadata Update)
	// ========================================
	t.Log("[Step 1] 3.4.1.1 메타데이터 업데이트 테스트...")

	updates := map[string]interface{}{
		"name":        "Updated Test Agent",
		"description": "Updated description after metadata change",
		"endpoint":    "http://localhost:9090",
		"capabilities": map[string]interface{}{
			"version":  "2.0.0",
			"features": []string{"chat", "search", "analytics"},
		},
	}

	helpers.LogDetail(t, "    → Updating agent metadata...")
	err = agentClient.Update(ctx, testDID, updates, agentKeyPair)
	require.NoError(t, err, "Failed to update agent")
	helpers.LogDetail(t, "    ✓ Update transaction successful")

	// Verify update
	updatedAgent, err := agentClient.Resolve(ctx, testDID)
	require.NoError(t, err, "Failed to resolve after update")

	helpers.LogDetail(t, "    ✓ Verifying updated metadata:")
	require.Equal(t, "Updated Test Agent", updatedAgent.Name, "Name should be updated")
	helpers.LogDetail(t, "      ✓ Name: %s", updatedAgent.Name)

	require.Equal(t, "Updated description after metadata change", updatedAgent.Description, "Description should be updated")
	helpers.LogDetail(t, "      ✓ Description: %s", updatedAgent.Description)

	require.Equal(t, "http://localhost:9090", updatedAgent.Endpoint, "Endpoint should be updated")
	helpers.LogDetail(t, "      ✓ Endpoint: %s", updatedAgent.Endpoint)

	// Verify capabilities
	capabilities, ok := updatedAgent.Capabilities["version"].(string)
	require.True(t, ok, "Capabilities should contain version")
	require.Equal(t, "2.0.0", capabilities, "Version should be updated")
	helpers.LogDetail(t, "      ✓ Capabilities version: %s", capabilities)

	// ========================================
	// 3.4.1.2 엔드포인트 변경 (Endpoint Change)
	// ========================================
	t.Log("[Step 2] 3.4.1.2 엔드포인트 변경 테스트...")

	newEndpoint := "https://api.example.com/agent"
	endpointUpdate := map[string]interface{}{
		"name":        updatedAgent.Name,        // Keep same
		"description": updatedAgent.Description, // Keep same
		"endpoint":    newEndpoint,              // Only change this
		"capabilities": updatedAgent.Capabilities,
	}

	helpers.LogDetail(t, "    → Changing endpoint to: %s", newEndpoint)
	err = agentClient.Update(ctx, testDID, endpointUpdate, agentKeyPair)
	require.NoError(t, err, "Failed to update endpoint")
	helpers.LogDetail(t, "    ✓ Endpoint update successful")

	// Verify endpoint change
	finalAgent, err := agentClient.Resolve(ctx, testDID)
	require.NoError(t, err, "Failed to resolve after endpoint update")

	require.Equal(t, newEndpoint, finalAgent.Endpoint, "Endpoint should be changed")
	helpers.LogDetail(t, "    ✓ Endpoint verified: %s", finalAgent.Endpoint)

	// Verify other fields remain unchanged
	require.Equal(t, updatedAgent.Name, finalAgent.Name, "Name should remain unchanged")
	require.Equal(t, updatedAgent.Description, finalAgent.Description, "Description should remain unchanged")
	helpers.LogDetail(t, "    ✓ Other fields remain unchanged")

	// ========================================
	// 3.4.1.3 UpdatedAt 타임스탬프 검증
	// ========================================
	t.Log("[Step 3] 3.4.1.3 UpdatedAt 타임스탬프 검증...")

	helpers.LogDetail(t, "    → Initial CreatedAt: %s", initialAgent.CreatedAt.Format(time.RFC3339))
	helpers.LogDetail(t, "    → Initial UpdatedAt: %s", initialAgent.UpdatedAt.Format(time.RFC3339))
	helpers.LogDetail(t, "    → Final UpdatedAt: %s", finalAgent.UpdatedAt.Format(time.RFC3339))

	require.True(t, finalAgent.UpdatedAt.After(initialAgent.UpdatedAt),
		"UpdatedAt should be later than initial UpdatedAt")
	helpers.LogDetail(t, "    ✓ UpdatedAt timestamp correctly updated")

	require.Equal(t, initialAgent.CreatedAt, finalAgent.CreatedAt,
		"CreatedAt should remain unchanged")
	helpers.LogDetail(t, "    ✓ CreatedAt timestamp unchanged")

	// ========================================
	// 3.4.1.4 소유권 검증 (Ownership Verification)
	// ========================================
	t.Log("[Step 4] 3.4.1.4 소유권 검증...")

	require.Equal(t, initialAgent.Owner, finalAgent.Owner, "Owner should remain unchanged")
	require.True(t, finalAgent.IsActive, "Agent should remain active")
	helpers.LogDetail(t, "    ✓ Owner remains: %s", finalAgent.Owner)
	helpers.LogDetail(t, "    ✓ Agent remains active")

	// ========================================
	// 3.4.1.5 여러 번 업데이트 (Multiple Updates with Nonce)
	// ========================================
	t.Log("[Step 5] 3.4.1.5 여러 번 업데이트 테스트 (Nonce 검증)...")

	// Third update
	thirdUpdate := map[string]interface{}{
		"name":        "Final Test Agent Name",
		"description": finalAgent.Description,
		"endpoint":    finalAgent.Endpoint,
		"capabilities": map[string]interface{}{
			"version":  "3.0.0",
			"features": []string{"chat", "search", "analytics", "reporting"},
		},
	}

	helpers.LogDetail(t, "    → Performing third update...")
	err = agentClient.Update(ctx, testDID, thirdUpdate, agentKeyPair)
	require.NoError(t, err, "Failed to perform third update")
	helpers.LogDetail(t, "    ✓ Third update successful")

	// Verify third update
	thirdAgent, err := agentClient.Resolve(ctx, testDID)
	require.NoError(t, err, "Failed to resolve after third update")

	require.Equal(t, "Final Test Agent Name", thirdAgent.Name, "Name should be updated in third update")
	helpers.LogDetail(t, "    ✓ Name updated to: %s", thirdAgent.Name)

	thirdCap, ok := thirdAgent.Capabilities["version"].(string)
	require.True(t, ok, "Capabilities should contain version")
	require.Equal(t, "3.0.0", thirdCap, "Version should be 3.0.0")
	helpers.LogDetail(t, "    ✓ Capabilities version: %s", thirdCap)

	// Verify UpdatedAt is later than previous update
	require.True(t, thirdAgent.UpdatedAt.After(finalAgent.UpdatedAt),
		"UpdatedAt should be later than second update")
	helpers.LogDetail(t, "    ✓ UpdatedAt correctly incremented")

	// Fourth update
	fourthUpdate := map[string]interface{}{
		"name":        thirdAgent.Name,
		"description": "Fourth update - testing nonce management",
		"endpoint":    "https://final.api.example.com/v4",
		"capabilities": thirdAgent.Capabilities,
	}

	helpers.LogDetail(t, "    → Performing fourth update...")
	err = agentClient.Update(ctx, testDID, fourthUpdate, agentKeyPair)
	require.NoError(t, err, "Failed to perform fourth update")
	helpers.LogDetail(t, "    ✓ Fourth update successful")

	// Verify fourth update
	fourthAgent, err := agentClient.Resolve(ctx, testDID)
	require.NoError(t, err, "Failed to resolve after fourth update")

	require.Equal(t, "Fourth update - testing nonce management", fourthAgent.Description,
		"Description should be updated in fourth update")
	require.Equal(t, "https://final.api.example.com/v4", fourthAgent.Endpoint,
		"Endpoint should be updated in fourth update")
	helpers.LogDetail(t, "    ✓ Description: %s", fourthAgent.Description)
	helpers.LogDetail(t, "    ✓ Endpoint: %s", fourthAgent.Endpoint)

	// ========================================
	// Test Summary
	// ========================================
	helpers.LogSuccess(t, "✅ All update tests passed!")
	helpers.LogDetail(t, "")
	helpers.LogDetail(t, "Verified Specifications:")
	helpers.LogDetail(t, "  ✓ 3.4.1.1 메타데이터 업데이트")
	helpers.LogDetail(t, "  ✓ 3.4.1.2 엔드포인트 변경")
	helpers.LogDetail(t, "  ✓ 3.4.1.3 UpdatedAt 타임스탬프")
	helpers.LogDetail(t, "  ✓ 3.4.1.4 소유권 유지")
	helpers.LogDetail(t, "  ✓ 3.4.1.5 여러 번 업데이트 (Nonce 관리)")
	helpers.LogDetail(t, "")
	helpers.LogDetail(t, "Total Updates Performed: 4")
	helpers.LogDetail(t, "  - Update 1: Full metadata change")
	helpers.LogDetail(t, "  - Update 2: Endpoint only")
	helpers.LogDetail(t, "  - Update 3: Name and capabilities")
	helpers.LogDetail(t, "  - Update 4: Description and endpoint")
}
