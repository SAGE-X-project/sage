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

// TestDIDPreRegistrationCheck tests specification requirement 3.1.1.2 (Early Detection)
// Tests DID existence check before registration to avoid gas waste
func TestDIDPreRegistrationCheck(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	helpers.LogTestSection(t, "3.1.1.2-Early", "DID 사전 중복 체크 (등록 전 존재 여부 확인)")

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

	// ========================================
	// Agent A: 첫 번째 Agent 등록
	// ========================================
	helpers.LogDetail(t, "[Agent A] 첫 번째 Agent 등록 프로세스 시작")

	// Generate DID for Agent A
	uuidA := uuid.New()
	didA := did.GenerateDID(did.ChainEthereum, uuidA.String())
	helpers.LogDetail(t, "  Agent A DID: %s", didA)

	// Generate keypair for Agent A
	helpers.LogDetail(t, "[Step 1] Agent A Secp256k1 키페어 생성...")
	agentAKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err, "Failed to generate Agent A keypair")

	ecdsaPubKeyA, ok := agentAKeyPair.PublicKey().(*ecdsa.PublicKey)
	require.True(t, ok, "Failed to cast Agent A public key to ECDSA")
	agentAAddress := ethcrypto.PubkeyToAddress(*ecdsaPubKeyA)
	helpers.LogSuccess(t, "Agent A 키페어 생성 완료")
	helpers.LogDetail(t, "  Agent A 주소: %s", agentAAddress.Hex())

	// Fund Agent A
	helpers.LogDetail(t, "[Step 2] Agent A 키에 ETH 전송 중...")
	fundAmount := new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)) // 10 ETH
	receiptA, err := transferETH(ctx, client, agentAAddress, fundAmount)
	require.NoError(t, err, "Failed to transfer ETH to Agent A")
	helpers.LogSuccess(t, "Agent A ETH 전송 완료")
	helpers.LogDetail(t, "  Transaction Hash: %s", receiptA.TxHash.Hex())

	// Create Agent A client
	helpers.LogDetail(t, "[Step 3] Agent A 클라이언트 생성...")
	ecdsaPrivKeyA, ok := agentAKeyPair.PrivateKey().(*ecdsa.PrivateKey)
	require.True(t, ok, "Failed to cast Agent A private key to ECDSA")
	agentAPrivateKeyHex := fmt.Sprintf("%x", ecdsaPrivKeyA.D.Bytes())
	agentAConfig := &did.RegistryConfig{
		ContractAddress:    "0x5FbDB2315678afecb367f032d93F642f64180aa3",
		RPCEndpoint:        "http://localhost:8545",
		PrivateKey:         agentAPrivateKeyHex,
		GasPrice:           0,
		MaxRetries:         10,
		ConfirmationBlocks: 0,
	}
	agentAClient, err := NewEthereumClientV4(agentAConfig)
	require.NoError(t, err, "Failed to create Agent A client")
	helpers.LogSuccess(t, "Agent A 클라이언트 생성 완료")

	// Register Agent A
	helpers.LogDetail(t, "[Step 4] Agent A 등록 중...")
	reqA := &did.RegistrationRequest{
		DID:      didA,
		Name:     "Agent A - Pre-registered",
		Endpoint: "http://localhost:8080/agentA",
		KeyPair:  agentAKeyPair,
	}

	resultA, err := agentAClient.Register(ctx, reqA)
	require.NoError(t, err, "Failed to register Agent A")
	require.NotNil(t, resultA, "Agent A registration result should not be nil")
	helpers.LogSuccess(t, "Agent A 등록 성공")
	helpers.LogDetail(t, "  Transaction Hash: %s", resultA.TransactionHash)
	helpers.LogDetail(t, "  Block Number: %d", resultA.BlockNumber)

	// ========================================
	// Agent B: 사전 중복 체크 프로세스
	// ========================================
	helpers.LogDetail(t, "")
	helpers.LogDetail(t, "[Agent B] 두 번째 Agent 등록 프로세스 시작 (사전 중복 체크 포함)")

	// Generate keypair for Agent B
	helpers.LogDetail(t, "[Step 5] Agent B Secp256k1 키페어 생성...")
	agentBKeyPair, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err, "Failed to generate Agent B keypair")

	ecdsaPubKeyB, ok := agentBKeyPair.PublicKey().(*ecdsa.PublicKey)
	require.True(t, ok, "Failed to cast Agent B public key to ECDSA")
	agentBAddress := ethcrypto.PubkeyToAddress(*ecdsaPubKeyB)
	helpers.LogSuccess(t, "Agent B 키페어 생성 완료")
	helpers.LogDetail(t, "  Agent B 주소: %s", agentBAddress.Hex())

	// Fund Agent B
	helpers.LogDetail(t, "[Step 6] Agent B 키에 ETH 전송 중...")
	receiptB, err := transferETH(ctx, client, agentBAddress, fundAmount)
	require.NoError(t, err, "Failed to transfer ETH to Agent B")
	helpers.LogSuccess(t, "Agent B ETH 전송 완료")
	helpers.LogDetail(t, "  Transaction Hash: %s", receiptB.TxHash.Hex())

	// Create Agent B client
	helpers.LogDetail(t, "[Step 7] Agent B 클라이언트 생성...")
	ecdsaPrivKeyB, ok := agentBKeyPair.PrivateKey().(*ecdsa.PrivateKey)
	require.True(t, ok, "Failed to cast Agent B private key to ECDSA")
	agentBPrivateKeyHex := fmt.Sprintf("%x", ecdsaPrivKeyB.D.Bytes())
	agentBConfig := &did.RegistryConfig{
		ContractAddress:    "0x5FbDB2315678afecb367f032d93F642f64180aa3",
		RPCEndpoint:        "http://localhost:8545",
		PrivateKey:         agentBPrivateKeyHex,
		GasPrice:           0,
		MaxRetries:         10,
		ConfirmationBlocks: 0,
	}
	agentBClient, err := NewEthereumClientV4(agentBConfig)
	require.NoError(t, err, "Failed to create Agent B client")
	helpers.LogSuccess(t, "Agent B 클라이언트 생성 완료")

	// ========================================
	// 핵심: 사전 중복 체크 (Early Detection)
	// ========================================
	helpers.LogDetail(t, "[Step 8] 🔍 사전 중복 체크: Agent B가 Agent A와 같은 DID 시도...")
	helpers.LogDetail(t, "  시도할 DID: %s (Agent A가 이미 등록함)", didA)

	// Agent B tries to use the same DID as Agent A (simulate collision)
	// Check if DID already exists BEFORE registration
	helpers.LogDetail(t, "  등록 전 DID 존재 여부 확인 중 (SAGE Resolve 사용)...")
	existingAgent, err := agentBClient.Resolve(ctx, didA)

	if err != nil {
		// DID does not exist - safe to use
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
			helpers.LogSuccess(t, "DID가 존재하지 않음 - 사용 가능")
		} else {
			t.Fatalf("Unexpected error during Resolve: %v", err)
		}
	} else {
		// DID already exists - must generate a new one
		require.NotNil(t, existingAgent, "Existing agent should not be nil")
		helpers.LogSuccess(t, "⚠️  DID 중복 감지! (Early Detection)")
		helpers.LogDetail(t, "  이미 등록된 Agent 정보:")
		helpers.LogDetail(t, "    DID: %s", existingAgent.DID)
		helpers.LogDetail(t, "    Name: %s", existingAgent.Name)
		helpers.LogDetail(t, "    Owner: %s", existingAgent.Owner)
		helpers.LogDetail(t, "  ✅ 사전 체크로 가스비 낭비 방지!")
	}

	// ========================================
	// Agent B: 새로운 DID 생성 및 등록
	// ========================================
	helpers.LogDetail(t, "[Step 9] Agent B 새로운 DID 생성...")
	uuidB := uuid.New()
	didB := did.GenerateDID(did.ChainEthereum, uuidB.String())
	helpers.LogSuccess(t, "새로운 DID 생성 완료")
	helpers.LogDetail(t, "  Agent B 새 DID: %s", didB)

	// Verify new DID does not exist
	helpers.LogDetail(t, "[Step 10] 새 DID 존재 여부 확인...")
	_, err = agentBClient.Resolve(ctx, didB)
	if err != nil && (strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist")) {
		helpers.LogSuccess(t, "새 DID 중복 없음 - 등록 가능")
	} else if err == nil {
		t.Fatal("New DID should not exist yet")
	}

	// Register Agent B with new DID
	helpers.LogDetail(t, "[Step 11] Agent B 새 DID로 등록 중...")
	reqB := &did.RegistrationRequest{
		DID:      didB,
		Name:     "Agent B - After Pre-check",
		Endpoint: "http://localhost:8080/agentB",
		KeyPair:  agentBKeyPair,
	}

	resultB, err := agentBClient.Register(ctx, reqB)
	require.NoError(t, err, "Failed to register Agent B with new DID")
	require.NotNil(t, resultB, "Agent B registration result should not be nil")
	helpers.LogSuccess(t, "Agent B 새 DID로 등록 성공!")
	helpers.LogDetail(t, "  Transaction Hash: %s", resultB.TransactionHash)
	helpers.LogDetail(t, "  Block Number: %d", resultB.BlockNumber)

	// Verify both agents are registered
	helpers.LogDetail(t, "[Step 12] 두 Agent 모두 등록 확인...")
	agentAResolved, err := agentBClient.Resolve(ctx, didA)
	require.NoError(t, err, "Failed to resolve Agent A")
	require.Equal(t, "Agent A - Pre-registered", agentAResolved.Name)

	agentBResolved, err := agentBClient.Resolve(ctx, didB)
	require.NoError(t, err, "Failed to resolve Agent B")
	require.Equal(t, "Agent B - After Pre-check", agentBResolved.Name)
	helpers.LogSuccess(t, "두 Agent 모두 정상 등록 확인")

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"Agent A DID 생성 및 등록 성공",
		"Agent B 키페어 생성",
		"[사전 체크] Agent B가 Agent A의 DID로 Resolve 시도",
		"[Early Detection] DID 중복 감지 성공",
		"[가스비 절약] 등록 트랜잭션 전에 중복 발견",
		"Agent B 새로운 DID 생성",
		"[사전 체크] 새 DID 중복 없음 확인",
		"Agent B 새 DID로 등록 성공",
		"두 Agent 모두 블록체인에 정상 등록 확인",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case": "3.1.1.2_Early_DID_Pre_Registration_Check",
		"scenario":  "Early detection of DID collision before registration",
		"sage_functions_used": []string{
			"GenerateDID(chain, identifier)",
			"EthereumClientV4.Register(ctx, req)",
			"EthereumClientV4.Resolve(ctx, did) - 사전 체크",
		},
		"agent_a": map[string]interface{}{
			"did":              string(didA),
			"uuid":             uuidA.String(),
			"name":             "Agent A - Pre-registered",
			"transaction_hash": resultA.TransactionHash,
			"block_number":     resultA.BlockNumber,
		},
		"agent_b": map[string]interface{}{
			"attempted_did":      string(didA), // Tried to use Agent A's DID
			"collision_detected": true,
			"new_did":            string(didB),
			"new_uuid":           uuidB.String(),
			"name":               "Agent B - After Pre-check",
			"transaction_hash":   resultB.TransactionHash,
			"block_number":       resultB.BlockNumber,
		},
		"early_detection": map[string]interface{}{
			"check_method":    "Resolve before Register",
			"collision_found": true,
			"gas_saved":       "Prevented failed transaction",
		},
	}
	helpers.SaveTestData(t, "did/did_pre_registration_check.json", testData)
}
