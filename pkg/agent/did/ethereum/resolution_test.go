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
	"encoding/hex"
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

// TestDIDResolution tests specification requirement 3.3.1
// Tests DID resolution from blockchain, DID document parsing, and public key extraction
func TestDIDResolution(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("SAGE_INTEGRATION_TEST") != "1" {
		t.Skip("Skipping integration test. Set SAGE_INTEGRATION_TEST=1 to run")
	}

	helpers.LogTestSection(t, "3.3.1", "DID 조회 (블록체인에서 조회, DID 문서 파싱, 공개키 추출)")

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

	// Store public key bytes for later verification
	pubKeyBytes, err := did.MarshalPublicKey(ecdsaPubKey)
	require.NoError(t, err, "Failed to marshal public key")
	helpers.LogDetail(t, "  공개키 크기: %d bytes", len(pubKeyBytes))
	helpers.LogDetail(t, "  공개키 (hex, 처음 32 bytes): %s...", hex.EncodeToString(pubKeyBytes[:min(32, len(pubKeyBytes))]))

	// Step 4: Fund the agent key
	helpers.LogDetail(t, "[Step 3] Agent 키에 ETH 전송 중...")
	fundAmount := new(big.Int).Mul(big.NewInt(10), big.NewInt(1e18)) // 10 ETH
	receipt, err := transferETH(ctx, client, agentAddress, fundAmount)
	require.NoError(t, err, "Failed to transfer ETH")
	helpers.LogSuccess(t, "ETH 전송 완료")
	helpers.LogDetail(t, "  Transaction Hash: %s", receipt.TxHash.Hex())
	helpers.LogDetail(t, "  Gas Used: %d", receipt.GasUsed)

	// Step 5: Create new client using agent's keypair
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

	// Step 6: Register DID
	helpers.LogDetail(t, "[Step 5] DID 등록 중...")
	req := &did.RegistrationRequest{
		DID:      testDID,
		Name:     "DID Resolution Test Agent",
		Endpoint: "http://localhost:8080/agent",
		KeyPair:  agentKeyPair,
	}

	regResult, err := agentClient.Register(ctx, req)
	require.NoError(t, err, "Failed to register DID")
	require.NotNil(t, regResult, "Registration result should not be nil")
	helpers.LogSuccess(t, "DID 등록 성공")
	helpers.LogDetail(t, "  Transaction Hash: %s", regResult.TransactionHash)
	helpers.LogDetail(t, "  Block Number: %d", regResult.BlockNumber)

	// ========================================
	// 3.3.1.1 블록체인에서 조회
	// ========================================
	helpers.LogDetail(t, "[Step 6] 3.3.1.1 블록체인에서 DID 조회 중...")

	agent, err := agentClient.Resolve(ctx, testDID)
	require.NoError(t, err, "블록체인에서 DID 조회 실패")
	require.NotNil(t, agent, "Agent 정보가 nil")
	helpers.LogSuccess(t, "블록체인에서 DID 조회 성공")
	helpers.LogDetail(t, "  DID: %s", agent.DID)
	helpers.LogDetail(t, "  이름: %s", agent.Name)
	helpers.LogDetail(t, "  활성 상태: %t", agent.IsActive)
	helpers.LogDetail(t, "  엔드포인트: %s", agent.Endpoint)

	// ========================================
	// 3.3.1.2 DID 문서 파싱
	// ========================================
	helpers.LogDetail(t, "[Step 7] 3.3.1.2 DID 문서 파싱 및 검증...")

	// Verify all AgentMetadata fields are properly parsed
	require.Equal(t, testDID, agent.DID, "DID가 일치하지 않음")
	require.Equal(t, "DID Resolution Test Agent", agent.Name, "Agent 이름이 일치하지 않음")
	require.True(t, agent.IsActive, "Agent가 활성 상태가 아님")
	require.Equal(t, "http://localhost:8080/agent", agent.Endpoint, "Endpoint가 일치하지 않음")
	require.NotEmpty(t, agent.Owner, "Owner 주소가 비어있음")

	helpers.LogSuccess(t, "DID 문서 파싱 완료")
	helpers.LogDetail(t, "  파싱된 필드:")
	helpers.LogDetail(t, "    ✓ DID: %s", agent.DID)
	helpers.LogDetail(t, "    ✓ Name: %s", agent.Name)
	helpers.LogDetail(t, "    ✓ IsActive: %t", agent.IsActive)
	helpers.LogDetail(t, "    ✓ Endpoint: %s", agent.Endpoint)
	helpers.LogDetail(t, "    ✓ Owner: %s", agent.Owner)
	helpers.LogDetail(t, "    ✓ CreatedAt: %s", agent.CreatedAt.Format(time.RFC3339))

	// ========================================
	// 3.3.1.3 공개키 추출
	// ========================================
	helpers.LogDetail(t, "[Step 8] 3.3.1.3 공개키 추출 및 검증...")

	require.NotNil(t, agent.PublicKey, "공개키가 nil")

	// Extract public key bytes from resolved agent
	resolvedPubKeyBytes, err := did.MarshalPublicKey(agent.PublicKey)
	require.NoError(t, err, "공개키 직렬화 실패")

	helpers.LogSuccess(t, "공개키 추출 성공")
	helpers.LogDetail(t, "  공개키 타입: %T", agent.PublicKey)
	helpers.LogDetail(t, "  공개키 크기: %d bytes", len(resolvedPubKeyBytes))
	helpers.LogDetail(t, "  공개키 (hex, 처음 32 bytes): %s...", hex.EncodeToString(resolvedPubKeyBytes[:min(32, len(resolvedPubKeyBytes))]))

	// Verify extracted public key matches the original
	helpers.LogDetail(t, "[Step 9] 공개키 일치 여부 검증...")
	require.Equal(t, len(pubKeyBytes), len(resolvedPubKeyBytes), "공개키 크기가 일치하지 않음")
	require.Equal(t, pubKeyBytes, resolvedPubKeyBytes, "공개키가 원본과 일치하지 않음")
	helpers.LogSuccess(t, "공개키 일치 확인 완료")

	// Additional verification: Verify we can use the extracted public key
	helpers.LogDetail(t, "[Step 10] 추출된 공개키로 ECDSA 복원 테스트...")

	// Unmarshal the public key back to verify it's valid
	recoveredPubKey, err := did.UnmarshalPublicKey(resolvedPubKeyBytes, "secp256k1")
	require.NoError(t, err, "공개키 역직렬화 실패")
	require.NotNil(t, recoveredPubKey, "복원된 공개키가 nil")

	recoveredECDSAPubKey, ok := recoveredPubKey.(*ecdsa.PublicKey)
	require.True(t, ok, "복원된 공개키를 ECDSA로 변환 실패")

	// Verify the recovered public key produces the same Ethereum address
	recoveredAddress := ethcrypto.PubkeyToAddress(*recoveredECDSAPubKey)
	require.Equal(t, agentAddress.Hex(), recoveredAddress.Hex(), "복원된 공개키의 Ethereum 주소가 일치하지 않음")

	helpers.LogSuccess(t, "공개키 복원 및 검증 완료")
	helpers.LogDetail(t, "  원본 주소: %s", agentAddress.Hex())
	helpers.LogDetail(t, "  복원 주소: %s", recoveredAddress.Hex())

	// Pass criteria checklist
	helpers.LogPassCriteria(t, []string{
		"DID 생성 (SAGE GenerateDID 사용)",
		"Secp256k1 키페어 생성",
		"Hardhat 계정 → Agent 키로 ETH 전송",
		"Agent 등록 성공",
		"[3.3.1.1] 블록체인에서 DID 조회 성공 (SAGE Resolve)",
		"[3.3.1.2] DID 문서 파싱 성공 (모든 필드 검증)",
		"[3.3.1.2] DID 메타데이터 검증 (DID, Name, IsActive, Endpoint, Owner)",
		"[3.3.1.3] 공개키 추출 성공",
		"[3.3.1.3] 추출된 공개키가 원본과 일치",
		"[3.3.1.3] 공개키 복원 및 Ethereum 주소 검증 완료",
	})

	// Save test data
	testData := map[string]interface{}{
		"test_case": "3.3.1_DID_Resolution",
		"did":       string(testDID),
		"uuid":      uuidVal.String(),
		"sage_functions_used": []string{
			"GenerateDID(chain, identifier)",
			"EthereumClientV4.Register(ctx, req)",
			"EthereumClientV4.Resolve(ctx, did)",
			"MarshalPublicKey(publicKey)",
			"UnmarshalPublicKey(data, keyType)",
		},
		"resolution": map[string]interface{}{
			"success":     true,
			"did":         string(agent.DID),
			"name":        agent.Name,
			"is_active":   agent.IsActive,
			"endpoint":    agent.Endpoint,
			"owner":       agent.Owner,
			"created_at": agent.CreatedAt.Format(time.RFC3339),
		},
		"public_key": map[string]interface{}{
			"type":          fmt.Sprintf("%T", agent.PublicKey),
			"size_bytes":    len(resolvedPubKeyBytes),
			"matches_original": len(pubKeyBytes) == len(resolvedPubKeyBytes) && string(pubKeyBytes) == string(resolvedPubKeyBytes),
			"ethereum_address": agentAddress.Hex(),
			"recovered_address": recoveredAddress.Hex(),
		},
		"eth_transfer": map[string]interface{}{
			"from":     "Hardhat Account #0",
			"to":       agentAddress.Hex(),
			"amount":   "10 ETH",
			"gas_used": receipt.GasUsed,
		},
	}
	helpers.SaveTestData(t, "did/did_resolution.json", testData)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
