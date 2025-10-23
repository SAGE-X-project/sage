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
	"strings"
	"testing"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/did"
	"github.com/sage-x-project/sage/tests/helpers"
	"github.com/stretchr/testify/require"
)

// TestDIDDuplicateDetection tests specification requirement 3.1.1.2
// Tests duplicate DID detection by attempting to register the same DID twice
func TestDIDDuplicateDetection(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	helpers.LogTestSection(t, "3.1.1.2", "중복 DID 생성 시 오류 반환 (중복 등록 시도)")

	// Step 1: Create client with Hardhat default account (has initial ETH balance)
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

	// Step 2: Generate a new DID
	uuidVal := uuid.New()
	testDID := did.GenerateDID(did.ChainEthereum, uuidVal.String())
	helpers.LogDetail(t, "생성된 테스트 DID: %s", testDID)

	// Step 3: Generate Secp256k1 keypair for the agent
	helpers.LogDetail(t, "[Step 1] Secp256k1 키페어 생성...")
	agentKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err, "Failed to generate agent keypair")

	// Derive Ethereum address from the public key
	ecdsaPubKey, ok := agentKeyPair.PublicKey().(*ecdsa.PublicKey)
	require.True(t, ok, "Failed to cast public key to ECDSA")
	agentAddress := ethcrypto.PubkeyToAddress(*ecdsaPubKey)
	helpers.LogSuccess(t, "키페어 생성 완료")
	helpers.LogDetail(t, "  Agent 주소: %s", agentAddress.Hex())

	// Step 4: Fund the agent key with ETH from Hardhat default account
	helpers.LogDetail(t, "[Step 2] Agent 키에 ETH 전송 중...")
	fundAmount := new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)) // 10 ETH
	receipt, err := transferETH(ctx, client, agentAddress, fundAmount)
	require.NoError(t, err, "Failed to transfer ETH")
	helpers.LogSuccess(t, "ETH 전송 완료")
	helpers.LogDetail(t, "  Transaction Hash: %s", receipt.TxHash.Hex())
	helpers.LogDetail(t, "  Gas Used: %d", receipt.GasUsed)

	// Verify balance
	balance, err := client.client.BalanceAt(ctx, agentAddress, nil)
	require.NoError(t, err, "Failed to check balance")
	helpers.LogDetail(t, "  Agent 잔액: %s wei", balance.String())

	// Step 5: Create NEW client using agent's keypair for signing
	// This ensures msg.sender matches the public key being registered
	helpers.LogDetail(t, "[Step 3] Agent 키로 새 클라이언트 생성...")
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

	// Step 6: Create registration request
	req := &did.RegistrationRequest{
		DID:      testDID,
		Name:     "Test Agent for Duplicate Detection",
		Endpoint: "http://localhost:8080",
		KeyPair:  agentKeyPair,
	}

	// Step 7: First registration (should succeed)
	helpers.LogDetail(t, "[Step 4] 첫 번째 Agent 등록 시도...")
	result1, err := agentClient.Register(ctx, req)
	require.NoError(t, err, "First registration should succeed")
	require.NotNil(t, result1, "Registration result should not be nil")
	helpers.LogSuccess(t, "첫 번째 Agent 등록 성공")
	helpers.LogDetail(t, "  Transaction Hash: %s", result1.TransactionHash)
	helpers.LogDetail(t, "  Block Number: %d", result1.BlockNumber)
	helpers.LogDetail(t, "  Gas Used: ~%d", result1.BlockNumber) // Approximate

	// Step 8: Verify DID can be resolved
	helpers.LogDetail(t, "[Step 5] 등록된 DID 조회...")
	agent, err := agentClient.Resolve(ctx, testDID)
	require.NoError(t, err, "Should resolve registered DID")
	require.NotNil(t, agent, "Agent should be found")
	require.Equal(t, testDID, agent.DID, "DID should match")
	helpers.LogSuccess(t, "DID 조회 성공")
	helpers.LogDetail(t, "  Agent 이름: %s", agent.Name)
	helpers.LogDetail(t, "  Agent 활성 상태: %t", agent.IsActive)

	// Step 9: Try to register the same DID again (should fail)
	helpers.LogDetail(t, "[Step 6] 동일한 DID로 재등록 시도...")
	result2, err := agentClient.Register(ctx, req)

	// Verify that duplicate registration fails
	if err != nil {
		// Expected: Error should occur
		helpers.LogSuccess(t, "중복 등록 시 오류 발생 (예상된 동작)")
		helpers.LogDetail(t, "  에러 메시지: %v", err)

		// Check if error contains expected keywords
		errorMsg := err.Error()
		if strings.Contains(strings.ToLower(errorMsg), "already") ||
			strings.Contains(strings.ToLower(errorMsg), "exists") ||
			strings.Contains(strings.ToLower(errorMsg), "duplicate") ||
			strings.Contains(strings.ToLower(errorMsg), "revert") {
			helpers.LogSuccess(t, "중복 DID 에러 확인 (블록체인 revert 또는 중복 감지)")
		} else {
			t.Logf("Warning: Error occurred but may not be explicit duplicate error: %v", err)
		}

		require.Nil(t, result2, "Second registration result should be nil")
	} else {
		// Unexpected: No error occurred
		t.Errorf("Expected error for duplicate registration, but succeeded")
		if result2 != nil {
			t.Errorf("  Transaction Hash: %s", result2.TransactionHash)
			t.Errorf("  This indicates the blockchain allowed duplicate registration!")
		}
		t.FailNow()
	}

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"DID 생성 (SAGE GenerateDID 사용)",
		"Secp256k1 키페어 생성",
		"Hardhat 계정 → Agent 키로 ETH 전송 (gas 비용용)",
		"첫 번째 Agent 등록 성공",
		"등록된 DID 조회 성공 (SAGE Resolve)",
		"동일 DID 재등록 시도 → 에러 발생",
		"중복 등록 방지 확인",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case": "3.1.1.2_DID_Duplicate_Detection",
		"did":       string(testDID),
		"uuid":      uuidVal.String(),
		"sage_functions_used": []string{
			"GenerateDID(chain, identifier)",
			"EthereumClientV4.Register(ctx, req) - 첫번째",
			"EthereumClientV4.Resolve(ctx, did)",
			"EthereumClientV4.Register(ctx, req) - 두번째 (중복)",
		},
		"first_registration": map[string]interface{}{
			"success":          result1 != nil,
			"transaction_hash": result1.TransactionHash,
			"block_number":     result1.BlockNumber,
		},
		"duplicate_registration": map[string]interface{}{
			"error_occurred": err != nil,
			"error_message":  fmt.Sprintf("%v", err),
		},
		"eth_transfer": map[string]interface{}{
			"from":     "Hardhat Account #0",
			"to":       agentAddress.Hex(),
			"amount":   "10 ETH",
			"gas_used": receipt.GasUsed,
		},
	}
	helpers.SaveTestData(t, "did/did_duplicate_detection.json", testData)
}
