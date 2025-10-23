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

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/require"
)

// TestDIDDeactivation tests specification requirement 3.4.2
// Tests DID deactivation and inactive state verification
func TestDIDDeactivation(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	helpers.LogTestSection(t, "3.4.2", "DID 비활성화 및 inactive 상태 확인")

	// Step 1: Create client with Hardhat default account
	config := &did.RegistryConfig{
		ContractAddress:    "0x5FbDB2315678afecb367f032d93F642f64180aa3",
		RPCEndpoint:        "http://localhost:8545",
		PrivateKey:         "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", // Hardhat account #0
		GasPrice:           0,
		MaxRetries:         10,
		ConfirmationBlocks: 0,
	}

	client, err := NewEthereumClientV4(config)
	require.NoError(t, err, "Failed to create V4 client")
	helpers.LogSuccess(t, "V4 Client 생성 완료")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Step 2: Generate test DID
	uuidVal := uuid.New()
	testDID := did.GenerateDID(did.ChainEthereum, uuidVal.String())
	helpers.LogDetail(t, "[Step 1] 생성된 테스트 DID: %s", testDID)

	// Step 3: Generate Secp256k1 keypair for the agent
	helpers.LogDetail(t, "[Step 2] Secp256k1 키페어 생성...")
	agentKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err, "Failed to generate agent keypair")

	// Derive Ethereum address from the public key
	ecdsaPubKey, ok := agentKeyPair.PublicKey().(*ecdsa.PublicKey)
	require.True(t, ok, "Failed to cast public key to ECDSA")
	agentAddress := ethcrypto.PubkeyToAddress(*ecdsaPubKey)
	helpers.LogSuccess(t, "키페어 생성 완료")
	helpers.LogDetail(t, "  Agent 주소: %s", agentAddress.Hex())

	// Step 4: Fund the agent key
	helpers.LogDetail(t, "[Step 3] Agent 키에 ETH 전송 중...")
	fundAmount := new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)) // 10 ETH
	receipt, err := transferETH(ctx, client, agentAddress, fundAmount)
	require.NoError(t, err, "Failed to transfer ETH")
	helpers.LogSuccess(t, "ETH 전송 완료")
	helpers.LogDetail(t, "  Transaction Hash: %s", receipt.TxHash.Hex())
	helpers.LogDetail(t, "  Gas Used: %d", receipt.GasUsed)

	// Step 5: Create NEW client using agent's keypair for signing
	// This ensures msg.sender matches the public key being registered
	helpers.LogDetail(t, "[Step 4] Agent 키로 새 클라이언트 생성...")
	ecdsaPrivKey, ok := agentKeyPair.PrivateKey().(*ecdsa.PrivateKey)
	require.True(t, ok, "Failed to cast private key to ECDSA")
	agentPrivateKeyHex := fmt.Sprintf("%x", ecdsaPrivKey.D.Bytes())
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
	helpers.LogSuccess(t, "Agent 클라이언트 생성 완료")

	// Step 6: Register DID using agent's client (agent will be owner)
	helpers.LogDetail(t, "[Step 5] Agent 키로 DID 등록 중...")
	req := &did.RegistrationRequest{
		DID:      testDID,
		Name:     "Deactivation Test Agent",
		Endpoint: "http://localhost:8080/deactivation-test",
		KeyPair:  agentKeyPair,
	}

	regResult, err := agentClient.Register(ctx, req)
	require.NoError(t, err, "Failed to register DID")
	require.NotNil(t, regResult, "Registration result should not be nil")
	helpers.LogSuccess(t, "DID 등록 성공")
	helpers.LogDetail(t, "  Transaction Hash: %s", regResult.TransactionHash)
	helpers.LogDetail(t, "  Block Number: %d", regResult.BlockNumber)

	// Step 7: Verify DID is initially active
	helpers.LogDetail(t, "[Step 6] 등록된 DID 활성 상태 확인...")
	agent, err := agentClient.Resolve(ctx, testDID)
	require.NoError(t, err, "Failed to resolve DID")
	require.NotNil(t, agent, "Agent should not be nil")
	require.True(t, agent.IsActive, "Agent should be active initially")
	helpers.LogSuccess(t, "DID 초기 활성 상태 확인 완료")
	helpers.LogDetail(t, "  DID: %s", agent.DID)
	helpers.LogDetail(t, "  Name: %s", agent.Name)
	helpers.LogDetail(t, "  IsActive: %t", agent.IsActive)

	// Step 8: Deactivate the DID (using agent client - agent is owner)
	helpers.LogDetail(t, "[Step 7] DID 비활성화 실행 중...")
	err = agentClient.Deactivate(ctx, testDID, agentKeyPair)
	require.NoError(t, err, "Failed to deactivate DID")
	helpers.LogSuccess(t, "DID 비활성화 트랜잭션 성공")

	// Step 9: Verify DID is now inactive
	helpers.LogDetail(t, "[Step 8] 비활성화된 DID 상태 확인...")
	agentAfterDeactivation, err := agentClient.Resolve(ctx, testDID)
	require.NoError(t, err, "Failed to resolve DID after deactivation")
	require.NotNil(t, agentAfterDeactivation, "Agent should not be nil after deactivation")
	require.False(t, agentAfterDeactivation.IsActive, "Agent should be inactive after deactivation")
	helpers.LogSuccess(t, "DID 비활성 상태 확인 완료")
	helpers.LogDetail(t, "  DID: %s", agentAfterDeactivation.DID)
	helpers.LogDetail(t, "  IsActive: %t (비활성화 전: %t)", agentAfterDeactivation.IsActive, agent.IsActive)

	// Step 10: Verify state change
	helpers.LogDetail(t, "[Step 9] 상태 변경 검증...")
	require.NotEqual(t, agent.IsActive, agentAfterDeactivation.IsActive, "Active 상태가 변경되어야 함")
	helpers.LogSuccess(t, "상태 변경 확인 완료")
	helpers.LogDetail(t, "  활성화 전: IsActive = %t", agent.IsActive)
	helpers.LogDetail(t, "  비활성화 후: IsActive = %t", agentAfterDeactivation.IsActive)

	// Step 11: Verify metadata is still accessible
	helpers.LogDetail(t, "[Step 10] 비활성화된 DID 메타데이터 접근 확인...")
	require.Equal(t, testDID, agentAfterDeactivation.DID, "DID should be the same")
	require.Equal(t, agent.Name, agentAfterDeactivation.Name, "Name should be preserved")
	require.Equal(t, agent.Endpoint, agentAfterDeactivation.Endpoint, "Endpoint should be preserved")
	helpers.LogSuccess(t, "비활성화된 DID 메타데이터 접근 가능 확인")
	helpers.LogDetail(t, "  DID: %s", agentAfterDeactivation.DID)
	helpers.LogDetail(t, "  Name: %s", agentAfterDeactivation.Name)
	helpers.LogDetail(t, "  Endpoint: %s", agentAfterDeactivation.Endpoint)

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"DID 생성 (SAGE GenerateDID 사용)",
		"Secp256k1 키페어 생성",
		"Hardhat 계정 → Agent 키로 ETH 전송",
		"DID 등록 성공",
		"DID 초기 활성 상태 확인 (IsActive = true)",
		"[3.4.2] DID 비활성화 트랜잭션 성공 (SAGE Deactivate)",
		"[3.4.2] 비활성화 후 상태 확인 (IsActive = false)",
		"[3.4.2] Active 상태 변경 확인 (true → false)",
		"[3.4.2] 비활성화된 DID 메타데이터 접근 가능",
		"[3.4.2] DID 상태 일관성 유지",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case": "3.4.2_DID_Deactivation",
		"did":       string(testDID),
		"uuid":      uuidVal.String(),
		"sage_functions_used": []string{
			"GenerateDID(chain, identifier)",
			"EthereumClientV4.Register(ctx, req)",
			"EthereumClientV4.Resolve(ctx, did)",
			"EthereumClientV4.Deactivate(ctx, did, keyPair)",
		},
		"registration": map[string]interface{}{
			"success":          regResult != nil,
			"transaction_hash": regResult.TransactionHash,
			"block_number":     regResult.BlockNumber,
		},
		"before_deactivation": map[string]interface{}{
			"did":       string(agent.DID),
			"name":      agent.Name,
			"is_active": agent.IsActive,
			"endpoint":  agent.Endpoint,
		},
		"after_deactivation": map[string]interface{}{
			"did":       string(agentAfterDeactivation.DID),
			"name":      agentAfterDeactivation.Name,
			"is_active": agentAfterDeactivation.IsActive,
			"endpoint":  agentAfterDeactivation.Endpoint,
		},
		"state_change": map[string]interface{}{
			"from":    agent.IsActive,
			"to":      agentAfterDeactivation.IsActive,
			"changed": agent.IsActive != agentAfterDeactivation.IsActive,
		},
		"eth_transfer": map[string]interface{}{
			"from":     "Hardhat Account #0",
			"to":       agentAddress.Hex(),
			"amount":   "10 ETH",
			"gas_used": receipt.GasUsed,
		},
	}
	helpers.SaveTestData(t, "did/did_deactivation.json", testData)
}
